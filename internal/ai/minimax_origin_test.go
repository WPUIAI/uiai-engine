package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestMiniMaxVisionUsesConfiguredInitialOrigin(t *testing.T) {
	cases := []struct{ name, base, host, path string }{
		{"default", "", "api.minimax.io", "/v1/coding_plan/vlm"},
		{"official-bare", "https://api.minimax.io", "api.minimax.io", "/v1/coding_plan/vlm"},
		{"official-versioned", "https://api.minimax.io/v1", "api.minimax.io", "/v1/coding_plan/vlm"},
		{"official-trailing", "https://api.minimax.io/v1/", "api.minimax.io", "/v1/coding_plan/vlm"},
		{"custom-bare", "https://approved.invalid", "approved.invalid", "/v1/coding_plan/vlm"},
		{"custom-prefix", "https://approved.invalid/proxy", "approved.invalid", "/proxy/v1/coding_plan/vlm"},
		{"custom-trailing", "https://approved.invalid/proxy/", "approved.invalid", "/proxy/v1/coding_plan/vlm"},
		{"custom-versioned", "https://approved.invalid/proxy/v1", "approved.invalid", "/proxy/v1/coding_plan/vlm"},
		{"custom-versioned-trailing", "https://approved.invalid/proxy/v1/", "approved.invalid", "/proxy/v1/coding_plan/vlm"},
		{"custom-loopback", "http://127.0.0.1:1234/proxy", "127.0.0.1:1234", "/proxy/v1/coding_plan/vlm"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"minimax": {APIURL: tc.base}}}}, keys: aiKeys{MiniMax: redactedString("synthetic-origin")}}
			p.client = &http.Client{Transport: credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != tc.host || r.URL.Path != tc.path {
					t.Errorf("configured origin/path ignored: got host=%s path=%s", r.URL.Host, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer synthetic-origin" {
					t.Error("fixture authentication changed")
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(b), "fixture-prompt") || !strings.Contains(string(b), "fixture-image") {
					t.Error("vision payload changed")
				}
				return cacheFixtureResponse(200, `{"content":"fixture"}`), nil
			})}
			out, err := p.completeOnce(context.Background(), Request{Provider: "minimax", Model: "fixture-model", Prompt: "fixture-prompt", ImageBase64: "fixture-image"})
			if err != nil || out == nil || out.Content != "fixture" || calls != 1 {
				t.Fatal("vision fixture failed or unexpected fallback")
			}
		})
	}
}

func TestMiniMaxVisionFailureTextFallbackKeepsConfiguredOrigin(t *testing.T) {
	for _, image := range []bool{false, true} {
		t.Run(map[bool]string{false: "text-only", true: "vision-to-text"}[image], func(t *testing.T) {
			calls := 0
			p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"minimax": {APIURL: "https://approved.invalid/proxy"}}}}, keys: aiKeys{MiniMax: redactedString("synthetic-origin")}}
			p.client = &http.Client{Transport: credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "approved.invalid" {
					t.Error("configured origin ignored during fallback")
				}
				if r.URL.Path == "/proxy/v1/coding_plan/vlm" {
					return cacheFixtureResponse(503, `{}`), nil
				}
				if r.URL.Path != "/proxy/text/chatcompletion_v2" {
					t.Error("text fallback path changed")
				}
				return cacheFixtureResponse(200, `{"choices":[{"message":{"content":"fixture"}}]}`), nil
			})}
			req := Request{Provider: "minimax", Model: "fixture-model", Prompt: "fixture-prompt"}
			want := 1
			if image {
				req.ImageBase64 = "fixture-image"
				want = 2
			}
			out, err := p.completeOnce(context.Background(), req)
			if err != nil || out == nil || out.Content != "fixture" || calls != want {
				t.Fatal("existing text/vision fallback behavior changed")
			}
		})
	}
}
