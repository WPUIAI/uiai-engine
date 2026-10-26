package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
)

func TestMalformedInferenceEndpointReturnsSafeErrorWithoutPanic(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		t.Run(provider, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { calls++; return cacheFixtureResponse(200, `{}`), nil })}
			p := inferenceRedirectProvider("https://synthetic-private%zz", client)
			defer func() {
				if recover() != nil {
					t.Error("malformed configured inference URL caused panic instead of safe failure")
				}
			}()
			out, err := inferenceRedirectCall(p, provider, "https://synthetic-private%zz")
			if err == nil || out != nil || calls != 0 {
				t.Fatal("invalid endpoint became success or contacted transport")
			}
			if strings.Contains(err.Error(), "synthetic-private") || strings.Contains(fmt.Sprintf("%#v", err), "synthetic-private") {
				t.Fatal("request construction error exposed configured endpoint material")
			}
			if isRetryableError(err) {
				t.Fatal("malformed endpoint became retryable")
			}
		})
	}
}

func TestInferenceTransportFailuresAreSafeAndPreserveSemantics(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		for _, raw := range []error{
			errors.New("connection refused synthetic-private"),
			errors.New("unexpected EOF synthetic-private"),
			errors.New("unrelated synthetic-private"),
			fmt.Errorf("synthetic-private: %w", context.Canceled),
			fmt.Errorf("synthetic-private: %w", context.DeadlineExceeded),
			fmt.Errorf("synthetic-private: %w", errors.Join(context.Canceled, context.DeadlineExceeded)),
		} {
			t.Run(provider+raw.Error(), func(t *testing.T) {
				client := &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, raw })}
				p := inferenceRedirectProvider("https://synthetic-private@approved.invalid", client)
				out, err := inferenceRedirectCall(p, provider, "https://synthetic-private@approved.invalid")
				if err == nil || out != nil {
					t.Fatal("transport failure became inference success")
				}
				for e := err; e != nil; e = errors.Unwrap(e) {
					if strings.Contains(e.Error(), "synthetic-private") || strings.Contains(fmt.Sprintf("%#v", e), "synthetic-private") {
						t.Error("transport error chain exposed private endpoint/cause material")
					}
				}
				if isRetryableError(err) != isRetryableError(raw) || isRetryableError(fmt.Errorf("wrapped: %w", err)) != isRetryableError(raw) {
					t.Error("safe transport error changed legacy retry classification")
				}
				for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
					if errors.Is(err, sentinel) != errors.Is(raw, sentinel) || errors.Is(fmt.Errorf("wrapped: %w", err), sentinel) != errors.Is(raw, sentinel) {
						t.Error("safe transport failure lost cancellation/deadline identity")
					}
				}
			})
		}
	}
}

func TestMiniMaxTransportFallbackLogDoesNotEchoPrivateCause(t *testing.T) {
	const secret = "synthetic-private"
	client := &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection refused " + secret) })}
	p := inferenceRedirectProvider("https://approved.invalid/v1", client)
	p.cfg.AI.Providers["minimax"] = p.cfg.AI.Providers["openai"]
	p.keys.MiniMax = redactedString(secret)
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	out, err := p.completeOnce(context.Background(), Request{Provider: "minimax", Model: "fixture", ImageBase64: "fixture"})
	if out != nil || err == nil {
		t.Fatal("transport failure became success")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(output.String(), secret) {
		t.Fatal("transport error/fallback log exposed private cause")
	}
	if !isRetryableError(err) {
		t.Fatal("safe transport error lost connection-refused retry signal")
	}
}
