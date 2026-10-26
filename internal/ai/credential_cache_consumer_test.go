package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCompleteRefreshesExpiredCredentialBeforeInference(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	now := time.Date(2026, 10, 26, 5, 31, 0, 0, time.UTC)
	p.now = func() time.Time { return now }
	p.keysTTL = 300 * time.Second
	p.keys = aiKeys{OpenAI: redactedString("synthetic-stale"), fetchedAt: now.Add(-60 * time.Second)}
	var settings, inference atomic.Int32
	p.client.Transport = credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/ai-settings") {
			settings.Add(1)
			return cacheFixtureResponse(200, `{"openai":{"key":"synthetic-current"}}`), nil
		}
		inference.Add(1)
		if r.Header.Get("Authorization") != "Bearer synthetic-current" {
			t.Error("inference reused expired credential")
		}
		return cacheFixtureResponse(200, `{"choices":[{"message":{"content":"fixture response"}}]}`), nil
	})
	out, err := p.Complete(context.Background(), Request{Provider: "openai", Model: "fixture-model", Prompt: "fixture"})
	if err != nil || out == nil || out.Content != "fixture response" || settings.Load() != 1 || inference.Load() != 1 {
		t.Fatal("completion did not refresh before its fixture-only inference")
	}
}

func TestCompleteInvalidatedRefreshNeverStartsInference(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var inference atomic.Int32
	p.client.Transport = credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/ai-settings") {
			close(started)
			<-release
			return cacheFixtureResponse(200, `{"openai":{"key":"synthetic-invalidated"}}`), nil
		}
		inference.Add(1)
		return cacheFixtureResponse(200, `{"choices":[{"message":{"content":"unexpected"}}]}`), nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := p.Complete(context.Background(), Request{Provider: "openai", Model: "fixture-model"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("fixture settings request did not start")
	}
	p.InvalidateCredentialCache()
	once.Do(func() { close(release) })
	select {
	case err := <-done:
		if !errors.Is(err, errCredentialCacheInvalidated) {
			t.Fatal("completion ignored invalidation")
		}
	case <-time.After(time.Second):
		t.Fatal("completion did not terminate")
	}
	if inference.Load() != 0 {
		t.Fatal("invalidated settings still initiated inference")
	}
}
