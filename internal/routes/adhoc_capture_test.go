package routes

import (
	"testing"
	"time"
)

func TestAdhocCaptureRegistryServesOnlyFreshTokens(t *testing.T) {
	token := registerAdhocCapture([]byte("pixels"), "png")
	if len(token) != adhocTokenLen {
		t.Fatalf("token length=%d want=%d", len(token), adhocTokenLen)
	}
	adhocMu.Lock()
	entry, ok := adhocCaptures[token]
	adhocMu.Unlock()
	if !ok || string(entry.data) != "pixels" || entry.format != "png" {
		t.Fatalf("registered capture missing: %#v", entry)
	}
	if time.Until(entry.expiresAt) <= 0 || time.Until(entry.expiresAt) > adhocTokenTTL {
		t.Fatalf("expiry window out of bounds: %v", entry.expiresAt)
	}

	// Expired entries must never serve.
	adhocMu.Lock()
	entry.expiresAt = time.Now().Add(-time.Second)
	adhocMu.Unlock()
	stale := registerAdhocCapture([]byte("other"), "png")
	if stale == "" || stale == token {
		t.Fatalf("fresh registration failed: %q vs %q", stale, token)
	}
	adhocMu.Lock()
	_, stillThere := adhocCaptures[token]
	adhocMu.Unlock()
	if stillThere {
		t.Fatal("expired entry survived the sweep")
	}
}

func TestAdhocCaptureRegistryEvictsOldestWhenBounded(t *testing.T) {
	adhocMu.Lock()
	adhocCaptures = make(map[string]*adhocCapture)
	adhocMu.Unlock()
	first := registerAdhocCapture([]byte("first"), "png")
	// Fill to capacity with newer entries.
	for i := 0; len(first) > 0 && i < adhocMaxCaptures; i++ {
		registerAdhocCapture([]byte("bulk"), "png")
	}
	adhocMu.Lock()
	_, firstSurvives := adhocCaptures[first]
	size := len(adhocCaptures)
	adhocMu.Unlock()
	if firstSurvives {
		t.Fatal("oldest entry survived capacity eviction")
	}
	if size > adhocMaxCaptures {
		t.Fatalf("registry exceeded bound: %d", size)
	}
}
