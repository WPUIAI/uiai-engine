package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func assertCredentialBufferCleared(t *testing.T, s redactedString) {
	t.Helper()
	for _, b := range []byte(s.value()) {
		if b != 0 {
			t.Fatal("evicted credential backing storage retained nonzero bytes")
		}
	}
}

func TestCredentialInvalidationClearsOwnedBackingStorage(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	p.keys = aiKeys{Anthropic: redactedString("synthetic-a"), OpenAI: redactedString("synthetic-o"), OpenRouter: redactedString("synthetic-r"), Fireworks: redactedString("synthetic-f"), Kimi: redactedString("synthetic-k"), MiniMax: redactedString("synthetic-m"), Qwen: redactedString("synthetic-q"), fetchedAt: time.Now()}
	cached := []redactedString{p.keys.Anthropic, p.keys.OpenAI, p.keys.OpenRouter, p.keys.Fireworks, p.keys.Kimi, p.keys.MiniMax, p.keys.Qwen}
	p.InvalidateCredentialCache()
	for _, s := range cached {
		assertCredentialBufferCleared(t, s)
	}
}

func TestExpiredCredentialBackingStorageClearedOnRefreshFailure(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	p.keysTTL = 300 * time.Second
	p.keys = aiKeys{OpenAI: redactedString("synthetic-expired-value"), fetchedAt: time.Now().Add(-61 * time.Second)}
	cached := p.keys.OpenAI
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("fixture transport failure") })
	if err := p.ensureKeys(); err == nil {
		t.Fatal("fixture transport failure was ignored")
	}
	assertCredentialBufferCleared(t, cached)
	if !p.keys.fetchedAt.IsZero() || p.keys.OpenAI.value() != "" {
		t.Fatal("expired snapshot remained cached after refresh failure")
	}
}

func TestCredentialDetachedCopyAndFormatting(t *testing.T) {
	input := strings.Repeat("synthetic-input", 2)
	secret := redactedString(input)
	detached := secret.clone()
	secret.wipe()
	assertCredentialBufferCleared(t, secret)
	if detached.value() != input || input != strings.Repeat("synthetic-input", 2) {
		t.Fatal("wiping cache corrupted detached transport copy or immutable input")
	}
	for _, format := range []string{"%s", "%q", "%v", "%+v", "%#v", "%d", "%x", "%X", "%c", "%b", "%f", "%10000s", "%.2s"} {
		if out := fmt.Sprintf(format, detached); out != "[secret]" {
			t.Fatalf("credential formatting did not mask verb %s", format)
		}
	}
	text, err := detached.MarshalText()
	if err != nil || string(text) != "[secret]" {
		t.Fatal("text serialization did not mask credential")
	}
	detached.wipe()
	assertCredentialBufferCleared(t, detached)
}

func TestOverlappingCredentialRefreshWipesReplacedBuffers(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var lock sync.Mutex
	calls := 0
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		lock.Lock()
		calls++
		which := calls
		lock.Unlock()
		if which == 1 {
			close(started)
			<-release
		}
		return cacheFixtureResponse(200, fmt.Sprintf(`{"openai":{"key":"synthetic-response-%d"}}`, which)), nil
	})
	done := make(chan error, 1)
	go func() { done <- p.ensureKeys() }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("fixture refresh did not begin")
	}
	if err := p.ensureKeys(); err != nil {
		t.Fatal(err)
	}
	p.keysMu.RLock()
	old := p.keys.OpenAI
	p.keysMu.RUnlock()
	once.Do(func() { close(release) })
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("overlapping refresh hung")
	}
	assertCredentialBufferCleared(t, old)
}

func TestCredentialInferenceSnapshotsRaceSafelyWithCacheWipe(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	p.client.Transport = credentialBoundaryTransport(func(r *http.Request) (*http.Response, error) {
		key := r.Header.Get("x-api-key")
		if key == "" {
			key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if key != "synthetic-current" {
			t.Error("inference credential changed during cache eviction")
		}
		if strings.HasSuffix(r.URL.Path, "/messages") {
			return cacheFixtureResponse(200, `{"content":[{"type":"text","text":"fixture"}]}`), nil
		}
		return cacheFixtureResponse(200, `{"choices":[{"message":{"content":"fixture"}}]}`), nil
	})
	publish := func() {
		p.keysMu.Lock()
		p.keys.wipe()
		p.keys = aiKeys{OpenAI: redactedString("synthetic-current"), Anthropic: redactedString("synthetic-current")}
		p.keysMu.Unlock()
	}
	publish()
	var workers sync.WaitGroup
	workers.Add(3)
	go func() {
		defer workers.Done()
		for i := 0; i < 150; i++ {
			publish()
		}
	}()
	for _, provider := range []string{"openai", "anthropic"} {
		go func(provider string) {
			defer workers.Done()
			for i := 0; i < 100; i++ {
				req := Request{Provider: provider, Model: "fixture-model", Prompt: strings.Repeat("fixture ", 200)}
				if _, err := p.completeOnce(context.Background(), req); err != nil {
					t.Errorf("fixture inference failed: %v", err)
				}
			}
		}(provider)
	}
	workers.Wait()
	p.InvalidateCredentialCache()
}
