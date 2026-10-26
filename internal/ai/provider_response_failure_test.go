package ai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type providerFixtureReadFailure struct{}

func (providerFixtureReadFailure) Read([]byte) (int, error) {
	return 0, errors.New("synthetic-private body read failure")
}

func TestInferenceRejectsInvalidOrMissingResponseText(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		invalidType := `{"choices":[{"message":{"content":"fixture"}}],"usage":{"prompt_tokens":"synthetic-private"}}`
		if provider == "anthropic" {
			invalidType = `{"content":[{"type":"text","text":"fixture"}],"usage":{"input_tokens":"synthetic-private"}}`
		}
		if provider == "minimax-vlm" {
			invalidType = `{"content":"fixture","base_resp":{"status_code":"synthetic-private"}}`
		}
		for _, body := range []string{"synthetic-private invalid json", `null`, `{}`, invalidType, strings.ReplaceAll(inferenceRedirectResult(provider), "fixture", "   ")} {
			t.Run(provider+fmt.Sprint(len(body)), func(t *testing.T) {
				p := inferenceRedirectProvider("https://approved.invalid/v1", &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return cacheFixtureResponse(200, body), nil })})
				out, err := inferenceRedirectCall(p, provider, "https://approved.invalid")
				if out != nil || err == nil {
					t.Fatal("invalid/missing inference response was reported as success")
				}
				if strings.Contains(err.Error(), "synthetic-private") || strings.Contains(fmt.Sprintf("%#v", err), "synthetic-private") {
					t.Fatal("response failure exposed private body material")
				}
			})
		}
	}
}

func TestInferenceRejectsIncompleteBodyEvenWithValidJSON(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		t.Run(provider, func(t *testing.T) {
			p := inferenceRedirectProvider("https://approved.invalid/v1", &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
				r := cacheFixtureResponse(200, "")
				r.Body = io.NopCloser(io.MultiReader(strings.NewReader(inferenceRedirectResult(provider)), providerFixtureReadFailure{}))
				return r, nil
			})})
			out, err := inferenceRedirectCall(p, provider, "https://approved.invalid")
			if out != nil || err == nil {
				t.Fatal("incomplete response body was published as inference success")
			}
			if strings.Contains(err.Error(), "synthetic-private") {
				t.Fatal("body reader error exposed private cause")
			}
		})
	}
}
