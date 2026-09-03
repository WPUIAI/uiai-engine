package routes

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func isolatedFPVRegistry(t *testing.T) {
	t.Helper()
	t.Setenv("UIAI_FPV_REGISTRY_PATH", filepath.Join(t.TempDir(), "fpv-shares.json"))
}

func TestFPVShareOriginDeniesSensitiveSurfaces(t *testing.T) {
	for _, raw := range []string{
		"https://accounts.example.com/", "https://privacyportal.example.com/", "https://dashboard.stripe.com/", "https://example.com/login", "https://example.com/privacy",
		"https://example.com/checkout", "https://example.com/health", "https://example.com/page?token=secret",
		"https://user@example.com/page", "https://example.com/page#secret", "file:///tmp/private",
	} {
		if _, err := fpvShareOrigin(raw); !errors.Is(err, errFPVSensitiveOrigin) {
			t.Fatalf("sensitive origin %q accepted: %v", raw, err)
		}
	}
	origin, err := fpvShareOrigin("HTTPS://Example.COM:443/public/docs")
	if err != nil || origin != "https://example.com:443" {
		t.Fatalf("ordinary origin = %q, %v", origin, err)
	}
}

func TestFPVEntryRejectsLegacyAndUnboundedCapabilities(t *testing.T) {
	isolatedFPVRegistry(t)
	now := time.Now().UTC()
	for _, entry := range []*fpvShare{
		{Token: "legacy", SessionID: "session:1", Origin: "https://example.com", MaxViews: 1, ExpiresAt: now.Add(time.Minute)},
		{PolicyVersion: fpvSharePolicyVersion, Token: "unbounded", SessionID: "session:1", Origin: "https://example.com", ConsentRef: "consent:test", MaxViews: 0, ExpiresAt: now.Add(time.Minute)},
		{PolicyVersion: fpvSharePolicyVersion, Token: "too-many", SessionID: "session:1", Origin: "https://example.com", ConsentRef: "consent:test", MaxViews: fpvMaximumMaxViews + 1, ExpiresAt: now.Add(time.Minute)},
		{PolicyVersion: fpvSharePolicyVersion, Token: "control", SessionID: "session:1", Origin: "https://example.com", ConsentRef: "consent:test", Controls: true, MaxViews: 1, ExpiresAt: now.Add(time.Minute)},
	} {
		fpvShares.Store(entry.Token, entry)
		if _, ok := fpvEntry(entry.Token); ok {
			t.Fatalf("unsafe share %q remained usable", entry.Token)
		}
	}
}

func TestFPVRegistryPersistenceFailuresFailClosed(t *testing.T) {
	registryDirectory := t.TempDir()
	t.Setenv("UIAI_FPV_REGISTRY_PATH", registryDirectory)
	if share, err := fpvCreateShare("session:persist", "https://example.com", "consent:test", 1, false, false, 1); !errors.Is(err, errFPVRegistryPersistence) || share != nil {
		t.Fatalf("creation persistence failure = %#v, %v", share, err)
	}
	entry := &fpvShare{PolicyVersion: fpvSharePolicyVersion, Token: "revoke-persistence", SessionID: "session:persist", Origin: "https://example.com", ConsentRef: "consent:test", MaxViews: 1, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	fpvShares.Store(entry.Token, entry)
	_, found, err := fpvRevokeToken(entry.Token)
	if !found || !errors.Is(err, errFPVRegistryPersistence) {
		t.Fatalf("revocation persistence failure found=%t error=%v", found, err)
	}
	if _, usable := fpvEntry(entry.Token); usable {
		t.Fatal("persistence failure left revoked share usable")
	}
}

func TestFPVViewLimitIsAtomicUnderConcurrency(t *testing.T) {
	isolatedFPVRegistry(t)
	entry := &fpvShare{PolicyVersion: fpvSharePolicyVersion, Token: "one-view", SessionID: "session:view", Origin: "https://example.com", ConsentRef: "consent:test", OneTime: true, MaxViews: 1, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	fpvShares.Store(entry.Token, entry)
	var wait sync.WaitGroup
	results := make(chan error, 64)
	for range 64 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := fpvConsumeViewIf(entry.Token, func(*fpvShare) bool { return true })
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	limited := 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, errFPVViewLimit) {
			limited++
		} else {
			t.Fatalf("unexpected consume error: %v", err)
		}
	}
	if successes != 1 || limited != 63 {
		t.Fatalf("successes=%d limited=%d", successes, limited)
	}
}

func TestFPVSessionRevocationIsImmediateAndIdempotent(t *testing.T) {
	isolatedFPVRegistry(t)
	entry := &fpvShare{PolicyVersion: fpvSharePolicyVersion, Token: "revoke-me", SessionID: "session:revoke", Origin: "https://example.com", ConsentRef: "consent:test", MaxViews: 1, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	fpvShares.Store(entry.Token, entry)
	if got, err := fpvRevokeSessionShares(entry.SessionID); err != nil || got != 1 {
		t.Fatalf("revoked = %d, error = %v", got, err)
	}
	if _, ok := fpvEntry(entry.Token); ok {
		t.Fatal("revoked session capability remained usable")
	}
	if got, err := fpvRevokeSessionShares(entry.SessionID); err != nil || got != 0 {
		t.Fatalf("idempotent revoke count = %d, error = %v", got, err)
	}
}
