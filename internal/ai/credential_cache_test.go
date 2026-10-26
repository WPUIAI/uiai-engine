package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCredentialCacheDoesNotReuseMinuteOldKeys(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	p.keysTTL = 300 * time.Second
	p.keys = aiKeys{OpenAI: redactedString("synthetic-stale-key"), fetchedAt: time.Now().Add(-61 * time.Second)}
	var calls atomic.Int32
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return cacheFixtureResponse(200, `{"openai":{"key":"synthetic-new-key"}}`), nil
	})
	if err := p.ensureKeysContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || p.keys.OpenAI.value() != "synthetic-new-key" {
		t.Fatal("credential cache reused a key older than the 60-second local bound")
	}
}

func cacheFixtureResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestCredentialCacheClockAndConfiguredBounds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ttl, age time.Duration
		refresh  bool
	}{
		{"below hard bound", 300 * time.Second, 60*time.Second - time.Nanosecond, false},
		{"at hard bound", 300 * time.Second, 60 * time.Second, true},
		{"shorter configured TTL", 10 * time.Second, 10 * time.Second, true},
		{"below shorter TTL", 10 * time.Second, 9 * time.Second, false},
		{"zero disables cache", 0, 0, true},
		{"negative disables cache", -time.Second, 0, true},
		{"clock rollback", 300 * time.Second, -time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 10, 26, 5, 31, 0, 0, time.UTC)
			p := boundaryProvider("https://example.invalid")
			p.now = func() time.Time { return now }
			p.keysTTL = tc.ttl
			p.keys = aiKeys{OpenAI: redactedString("synthetic-old"), fetchedAt: now.Add(-tc.age)}
			calls := 0
			p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return cacheFixtureResponse(200, `{"openai":{"key":"synthetic-new"}}`), nil
			})
			if err := p.ensureKeys(); err != nil {
				t.Fatal(err)
			}
			if (calls == 1) != tc.refresh {
				t.Fatal("cache expiry did not respect local/configured/clock boundary")
			}
			if tc.refresh && p.keys.fetchedAt != now {
				t.Fatal("refresh did not use the same testable clock")
			}
		})
	}
}

func TestCredentialCacheInvalidationClearsLocalSnapshot(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	p.keys = aiKeys{OpenAI: redactedString("synthetic-old"), Anthropic: redactedString("synthetic-old-anthropic"), fetchedAt: time.Now()}
	p.models = []ProviderModel{{ID: "old-model", Provider: "openai"}}
	p.InvalidateCredentialCache()
	if p.keys.OpenAI.value() != "" || p.keys.Anthropic.value() != "" || !p.keys.fetchedAt.IsZero() || len(p.AvailableModels()) != 0 {
		t.Fatal("local cache snapshot survived invalidation")
	}
	epoch := p.keysEpoch
	p.InvalidateCredentialCache()
	if p.keysEpoch != epoch+1 {
		t.Fatal("repeated invalidation failed to fence pending refreshes")
	}
}

func TestCredentialCacheInvalidationFencesOldResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"settings", 200, `{"default_provider":"openai","default_model":"old-model","openai":{"key":"synthetic-old"}}`},
		{"HTTP env fallback", 503, `upstream failure`},
		{"JSON env fallback", 200, `not json`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_KEY", "synthetic-env-old")
			t.Setenv("UIAI_DEFAULT_PROVIDER", "")
			t.Setenv("UIAI_DEFAULT_MODEL", "")
			p := boundaryProvider("https://example.invalid")
			p.keysTTL = 300 * time.Second
			started, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(release) })
			var calls atomic.Int32
			p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					close(started)
					<-release
					return cacheFixtureResponse(tc.status, tc.body), nil
				}
				return cacheFixtureResponse(200, `{"default_provider":"openai","default_model":"new-model","openai":{"key":"synthetic-new","model":"new-model"}}`), nil
			})
			done := make(chan error, 1)
			go func() { done <- p.ensureKeys() }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("fixture refresh did not begin")
			}
			p.InvalidateCredentialCache()
			if err := p.ensureKeys(); err != nil {
				t.Fatal(err)
			}
			releaseOnce.Do(func() { close(release) })
			select {
			case err := <-done:
				if !errors.Is(err, errCredentialCacheInvalidated) {
					t.Fatal("stale response was not rejected after invalidation")
				}
			case <-time.After(time.Second):
				t.Fatal("invalidation deadlocked refresh")
			}
			if p.keys.OpenAI.value() != "synthetic-new" || p.cfg.AI.DefaultModel != "new-model" {
				t.Fatal("old settings/fallback resurrected a credential or defaults")
			}
			models := p.AvailableModels()
			if len(models) == 0 || models[0].ID != "new-model" {
				t.Fatal("old response restored stale model metadata")
			}
		})
	}
}

type credentialCancelBody struct {
	io.Reader
	cancel context.CancelFunc
}

func (b credentialCancelBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.cancel()
	return n, err
}
func (credentialCancelBody) Close() error { return nil }

func TestCredentialCacheCanceledResponseDoesNotPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := boundaryProvider("https://example.invalid")
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		r := cacheFixtureResponse(200, "")
		r.Body = credentialCancelBody{Reader: strings.NewReader(`{"default_model":"cancelled-model","openai":{"key":"synthetic-cancelled"}}`), cancel: cancel}
		return r, nil
	})
	if !errors.Is(p.ensureKeysContext(ctx), context.Canceled) {
		t.Fatal("late caller cancellation was ignored")
	}
	if !p.keys.fetchedAt.IsZero() || p.keys.OpenAI.value() != "" || p.cfg.AI.DefaultModel != "" || len(p.AvailableModels()) != 0 {
		t.Fatal("cancelled settings populated cache/defaults/models")
	}
}

func TestCredentialCacheConcurrentRefreshInvalidateAndModelReads(t *testing.T) {
	t.Setenv("UIAI_DEFAULT_PROVIDER", "")
	t.Setenv("UIAI_DEFAULT_MODEL", "")
	p := boundaryProvider("https://example.invalid")
	p.keysTTL = 300 * time.Second
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return cacheFixtureResponse(200, `{"default_provider":"invalid-fixture","default_model":"fixture-model","openai":{"key":"synthetic-concurrent","model":"fixture-model"}}`), nil
	})
	var workers sync.WaitGroup
	for lane := 0; lane < 8; lane++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 40; i++ {
				p.InvalidateCredentialCache()
				if err := p.ensureKeys(); err != nil && !errors.Is(err, errCredentialCacheInvalidated) {
					t.Errorf("unexpected cache error: %v", err)
				}
				_ = p.AvailableModels()
				if _, err := p.Complete(context.Background(), Request{}); err == nil {
					t.Error("unsupported fixture provider unexpectedly completed")
				}
			}
		}()
	}
	workers.Wait()
	p.InvalidateCredentialCache()
	if !p.keys.fetchedAt.IsZero() || len(p.AvailableModels()) != 0 {
		t.Fatal("final invalidation left a published snapshot")
	}
}
