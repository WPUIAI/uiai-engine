package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestProviderFailurePreservesLegacyRetrySignals(t *testing.T) {
	for _, tc := range []struct {
		code    int
		message string
		retry   bool
	}{
		{1001, "authentication denied", false}, {1002, "Rate limit reached", true}, {1003, "connection refused", true}, {500, "", true}, {501, "unknown application code", false},
	} {
		err := providerApplicationFailure("fixture", tc.code, tc.message)
		if isRetryableError(err) != tc.retry || isRetryableError(fmt.Errorf("wrapped: %w", err)) != tc.retry {
			t.Fatal("application retry signal changed")
		}
	}
	for _, tc := range []struct {
		message string
		retry   bool
	}{
		{"context deadline exceeded", true}, {"context canceled", true}, {"no such host", true}, {"unexpected EOF", true}, {"broken pipe", true}, {"unrelated failure", false},
	} {
		if isRetryableError(errors.New(tc.message)) != tc.retry {
			t.Fatal("legacy transport retry signal changed")
		}
	}
	if isRetryableError(nil) {
		t.Fatal("nil failure is retryable")
	}
}

func TestProviderApplicationErrorsDoNotExposeUpstreamMessages(t *testing.T) {
	const secret = "synthetic-application-error-value"
	for _, tc := range []struct {
		name  string
		image bool
	}{{"text", false}, {"vision fallback log", true}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"base_resp":{"status_code":1002,"status_msg":"rate limit %s"}}`, secret)
			}))
			defer server.Close()
			p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"minimax": {APIURL: server.URL + "/v1"}}}}, client: server.Client(), keys: aiKeys{MiniMax: redactedString(secret)}}
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			defer log.SetOutput(previous)
			req := Request{Provider: "minimax", Model: "fixture-model", Prompt: "fixture"}
			if tc.image {
				req.ImageBase64 = "synthetic-image"
			}
			_, err := p.openaiComplete(context.Background(), req)
			if err == nil {
				t.Fatal("application failure became success")
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(output.String(), secret) || strings.Contains(fmt.Sprintf("%#v", err), secret) {
				t.Fatal("upstream application message exposed credential material in error or fallback log")
			}
			if !isRetryableError(err) || !isRetryableError(fmt.Errorf("wrapped: %w", err)) {
				t.Fatal("safe application error lost existing rate-limit retry classification")
			}
			if !strings.Contains(err.Error(), "1002") {
				t.Fatal("safe error lost numeric provider code")
			}
		})
	}
}

func TestProviderHTTPFailuresKeepRetryClassification(t *testing.T) {
	for _, status := range []int{401, 403, 418, 429, 500, 501, 503, 599} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("synthetic-private-body"))
			}))
			defer server.Close()
			p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"openai": {APIURL: server.URL}, "anthropic": {APIURL: server.URL}}}}, client: server.Client(), keys: aiKeys{OpenAI: redactedString("synthetic-http-key"), Anthropic: redactedString("synthetic-http-key")}}
			for _, provider := range []string{"openai", "anthropic"} {
				_, err := p.completeOnce(context.Background(), Request{Provider: provider, Model: "fixture-model"})
				want := status == 429 || status >= 500 && status <= 599
				if err == nil || isRetryableError(err) != want || isRetryableError(fmt.Errorf("wrapped: %w", err)) != want {
					t.Fatal("redacted HTTP error lost structured retry classification")
				}
				if strings.Contains(err.Error(), "synthetic-private-body") {
					t.Fatal("HTTP error exposed provider response body")
				}
			}
			_, visionErr := p.minimaxCodingPlanVisionComplete(context.Background(), "synthetic-http-key", server.URL+"/v1", Request{ImageBase64: "fixture-image"})
			want := status == 429 || status >= 500 && status <= 599
			if visionErr == nil || isRetryableError(visionErr) != want || strings.Contains(visionErr.Error(), "synthetic-private-body") {
				t.Fatal("vision HTTP failure lost safe retry classification")
			}
		})
	}
}

func TestCompleteFallsBackAfterSecretSafeHTTPFailure(t *testing.T) {
	for _, status := range []int{429, 503, 401, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var primary, fallback atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/openai") {
					primary.Add(1)
					w.WriteHeader(status)
					_, _ = w.Write([]byte("synthetic-private-body"))
					return
				}
				fallback.Add(1)
				if r.Header.Get("x-api-key") != "synthetic-fallback" {
					t.Error("fixture fallback lost credential boundary")
				}
				_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"fixture fallback"}]}`))
			}))
			defer server.Close()
			p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"openai": {APIURL: server.URL + "/openai"}, "anthropic": {APIURL: server.URL + "/anthropic"}}}}, client: server.Client(), keysTTL: 300 * time.Second, keys: aiKeys{OpenAI: redactedString("synthetic-primary"), Anthropic: redactedString("synthetic-fallback"), fetchedAt: time.Now()}}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			out, err := p.Complete(ctx, Request{Provider: "openai", Model: "fixture-model", Prompt: "fixture"})
			if status == 401 || status == 403 {
				if err == nil || out != nil || primary.Load() != 1 || fallback.Load() != 0 {
					t.Fatal("authentication denial incorrectly triggered fallback")
				}
				return
			}
			if err != nil || out == nil || out.Content != "fixture fallback" || primary.Load() != 1 || fallback.Load() != 1 {
				t.Fatal("completion failed instead of using configured fixture fallback after HTTP failure")
			}
		})
	}
}
