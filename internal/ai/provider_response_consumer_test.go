package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type providerFixtureEOF struct{}

func (providerFixtureEOF) Read([]byte) (int, error) {
	return 0, fmt.Errorf("synthetic-private: %w", io.ErrUnexpectedEOF)
}

func TestCompleteResponseProtocolAndReadFailurePolicy(t *testing.T) {
	for _, readFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(readFailure), func(t *testing.T) {
			primary, fallback := 0, 0
			p := inferenceRedirectProvider("https://approved.invalid", nil)
			p.keys.fetchedAt = time.Now()
			p.keysTTL = 300 * time.Second
			p.client = &http.Client{Transport: credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Authorization") != "" {
					primary++
					if !readFailure {
						return cacheFixtureResponse(200, "synthetic-private invalid json"), nil
					}
					resp := cacheFixtureResponse(200, "")
					resp.Body = io.NopCloser(io.MultiReader(strings.NewReader(inferenceRedirectResult("openai")), providerFixtureEOF{}))
					return resp, nil
				}
				fallback++
				return cacheFixtureResponse(200, inferenceRedirectResult("anthropic")), nil
			})}
			out, err := p.Complete(context.Background(), Request{Provider: "openai", Model: "fixture"})
			if readFailure {
				if err != nil || out == nil || out.Content != "fixture" || primary != 1 || fallback != 1 {
					t.Fatal("transient response-read failure lost configured fallback")
				}
			} else {
				if err == nil || out != nil || primary != 1 || fallback != 0 || isRetryableError(err) {
					t.Fatal("malformed protocol response became success or dispatched another provider")
				}
				if strings.Contains(err.Error(), "synthetic-private") {
					t.Fatal("protocol failure exposed body")
				}
			}
		})
	}
}

func TestProviderResponseTextCompatibility(t *testing.T) {
	for _, tc := range []struct{ provider, body, want string }{
		{"anthropic", `{"content":[{"type":"thinking","thinking":"private"},{"type":"text","text":"  a"},{"type":"text","text":"b  "}]}`, "  ab  "},
		{"anthropic", `{"content":[{"text":"  legacy  "}]}`, "  legacy  "},
		{"openai", `{"choices":[{"message":{"content":"  fixture  "}}]}`, "  fixture  "},
		{"minimax-vlm", `{"content":"  fixture  "}`, "fixture"},
	} {
		t.Run(tc.provider+tc.want, func(t *testing.T) {
			p := inferenceRedirectProvider("https://approved.invalid", &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return cacheFixtureResponse(200, tc.body), nil })})
			out, err := inferenceRedirectCall(p, tc.provider, "https://approved.invalid")
			if err != nil || out == nil || out.Content != tc.want {
				t.Fatal("valid text/legacy block/whitespace behavior changed")
			}
		})
	}
}

func TestResponseReadFailureKeepsContextIdentity(t *testing.T) {
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		err := decodeProviderResponse("fixture", providerContextReader{sentinel}, new(map[string]any))
		if err == nil || !errors.Is(err, sentinel) || strings.Contains(err.Error(), "synthetic-private") {
			t.Fatal("reader failure lost safe context identity")
		}
	}
}

type providerContextReader struct{ err error }

func (r providerContextReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("synthetic-private: %w", r.err)
}
