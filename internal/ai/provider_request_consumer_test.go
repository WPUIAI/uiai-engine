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
	"time"
)

func TestCompletePreservesSafeTransportFailoverAndCancellation(t *testing.T) {
	for _, cancelPrimary := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelPrimary), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			primary, fallback := 0, 0
			p := inferenceRedirectProvider("https://approved.invalid", nil)
			p.keys.fetchedAt = time.Now()
			p.keysTTL = 300 * time.Second
			p.client = &http.Client{Transport: credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Authorization") != "" {
					primary++
					if cancelPrimary {
						cancel()
						return nil, fmt.Errorf("synthetic-private: %w", context.Canceled)
					}
					return nil, errors.New("connection refused synthetic-private")
				}
				fallback++
				return cacheFixtureResponse(200, `{"content":[{"type":"text","text":"fixture fallback"}]}`), nil
			})}
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			defer log.SetOutput(previous)
			out, err := p.Complete(ctx, Request{Provider: "openai", Model: "fixture-model", Prompt: "fixture"})
			if cancelPrimary {
				if out != nil || !errors.Is(err, context.Canceled) || primary != 1 || fallback != 0 {
					t.Fatal("canceled completion sent another provider request or lost cancellation")
				}
			} else if err != nil || out == nil || out.Content != "fixture fallback" || primary != 1 || fallback != 1 {
				t.Fatal("safe transport failure lost configured completion fallback")
			}
			if strings.Contains(output.String(), "synthetic-private") {
				t.Fatal("Complete failover log exposed private transport details")
			}
		})
	}
}
