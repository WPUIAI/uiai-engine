package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCanceledInferenceDoesNotEnterTransport(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic", "minimax-vlm"} {
		t.Run(provider, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			calls := 0
			p := inferenceRedirectProvider("https://approved.invalid", &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, context.Canceled })})
			req := Request{Provider: provider, Model: "fixture"}
			var err error
			if provider == "minimax-vlm" {
				_, err = p.minimaxCodingPlanVisionComplete(ctx, "synthetic", "https://approved.invalid/v1", req)
			} else {
				_, err = p.completeOnce(ctx, req)
			}
			if !errors.Is(err, context.Canceled) || calls != 0 {
				t.Fatal("already canceled inference entered transport or lost context identity")
			}
		})
	}
}

func TestCanceledMiniMaxVisionDoesNotSendTextFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	calls := 0
	p := inferenceRedirectProvider("https://approved.invalid/v1", nil)
	p.cfg.AI.Providers["minimax"] = p.cfg.AI.Providers["openai"]
	p.keys.MiniMax = redactedString("synthetic")
	p.keys.fetchedAt = time.Now()
	p.keysTTL = 300 * time.Second
	p.client = &http.Client{Transport: credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		return nil, fmt.Errorf("synthetic-private: %w", context.Canceled)
	})}
	out, err := p.Complete(ctx, Request{Provider: "minimax", Model: "fixture", ImageBase64: "fixture-image"})
	if out != nil || !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("canceled vision initiated a text/failover request or lost context identity")
	}
}
