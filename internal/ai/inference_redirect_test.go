package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func inferenceRedirectProvider(endpoint string, client *http.Client) *Provider {
	return &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"openai": {APIURL: endpoint}, "anthropic": {APIURL: endpoint}}}}, client: client, keys: aiKeys{OpenAI: redactedString("synthetic-inference"), Anthropic: redactedString("synthetic-inference")}}
}
func inferenceRedirectCall(p *Provider, provider, base string) (*Response, error) {
	req := Request{Provider: provider, Model: "fixture-model", Prompt: "fixture"}
	if provider == "minimax-vlm" {
		req.ImageBase64 = "fixture-image"
		return p.minimaxCodingPlanVisionComplete(context.Background(), "synthetic-inference", base+"/v1", req)
	}
	return p.completeOnce(context.Background(), req)
}
func inferenceRedirectResult(provider string) string {
	if provider == "anthropic" {
		return `{"content":[{"type":"text","text":"fixture"}]}`
	}
	if provider == "minimax-vlm" {
		return `{"content":"fixture"}`
	}
	return `{"choices":[{"message":{"content":"fixture"}}]}`
}

func TestInferenceRedirectCannotForwardCredentialOrPromptAcrossOrigin(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		t.Run(provider, func(t *testing.T) {
			var received atomic.Int32
			var headerReceived, payloadReceived atomic.Bool
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received.Add(1)
				if r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
					headerReceived.Store(true)
				}
				body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil {
					t.Error("fixture payload read failed")
				}
				if strings.Contains(string(body), "fixture") {
					payloadReceived.Store(true)
				}
				_, _ = w.Write([]byte(inferenceRedirectResult(provider)))
			}))
			defer target.Close()
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL+"/capture", http.StatusTemporaryRedirect)
			}))
			defer source.Close()
			p := inferenceRedirectProvider(source.URL, source.Client())
			_, err := inferenceRedirectCall(p, provider, source.URL)
			if received.Load() != 0 || headerReceived.Load() || payloadReceived.Load() {
				t.Fatalf("foreign fixture requests=%d credential_header=%t prompt_or_image=%t", received.Load(), headerReceived.Load(), payloadReceived.Load())
			}
			if err == nil {
				t.Fatal("foreign-origin redirect was accepted as inference success")
			}
		})
	}
}

func TestInferenceSameOriginRedirectPreservesPolicyAndClient(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		t.Run(provider, func(t *testing.T) {
			var received, policies atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/redirected" {
					http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
					return
				}
				received.Add(1)
				key := r.Header.Get("x-api-key")
				if key == "" {
					key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				}
				if key != "synthetic-inference" {
					t.Error("same-origin credential transport changed")
				}
				_, _ = w.Write([]byte(inferenceRedirectResult(provider)))
			}))
			defer server.Close()
			p := inferenceRedirectProvider(server.URL, server.Client())
			p.client.CheckRedirect = func(*http.Request, []*http.Request) error { policies.Add(1); return nil }
			original := reflect.ValueOf(p.client.CheckRedirect).Pointer()
			out, err := inferenceRedirectCall(p, provider, server.URL)
			if err != nil || out == nil || out.Content != "fixture" || received.Load() != 1 || policies.Load() != 1 {
				t.Fatal("same-origin redirect lost existing policy or inference")
			}
			if reflect.ValueOf(p.client.CheckRedirect).Pointer() != original {
				t.Fatal("inference modified the shared client policy")
			}
		})
	}
}

func TestInferenceRedirectHonorsDefaultBoundAndRejectPolicy(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		for _, reject := range []bool{false, true} {
			t.Run(provider+fmt.Sprint(reject), func(t *testing.T) {
				var calls, policies atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					http.Redirect(w, r, "/loop", http.StatusTemporaryRedirect)
				}))
				defer server.Close()
				p := inferenceRedirectProvider(server.URL, server.Client())
				if reject {
					p.client.CheckRedirect = func(*http.Request, []*http.Request) error { policies.Add(1); return http.ErrUseLastResponse }
				}
				_, err := inferenceRedirectCall(p, provider, server.URL)
				want := int32(10)
				if reject {
					want = 1
				}
				if err == nil || calls.Load() != want {
					t.Fatal("inference redirect bypassed existing rejection/default loop bound")
				}
				if reject && policies.Load() != 1 {
					t.Fatal("custom rejection policy was not called")
				}
			})
		}
	}
}

func TestInferenceRedirectRejectsSchemeAndUserinfoExpansion(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		for _, location := range []string{"http://approved.invalid/capture", "https://injected:synthetic@approved.invalid/capture"} {
			t.Run(provider+fmt.Sprint(len(location)), func(t *testing.T) {
				var calls atomic.Int32
				client := &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						r := cacheFixtureResponse(307, `{}`)
						r.Header.Set("Location", location)
						return r, nil
					}
					return cacheFixtureResponse(200, inferenceRedirectResult(provider)), nil
				})}
				p := inferenceRedirectProvider("https://approved.invalid", client)
				_, err := inferenceRedirectCall(p, provider, "https://approved.invalid")
				if err == nil || calls.Load() != 1 {
					t.Fatal("scheme/userinfo redirect contacted an unauthorized target")
				}
			})
		}
	}
}
