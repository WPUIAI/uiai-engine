package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
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

type fakeFPVSessionStore map[string]*vision.Session

func (store fakeFPVSessionStore) Get(id string) (*vision.Session, bool) {
	session, found := store[id]
	return session, found
}

func TestFPVShareRouteDeniesUnsafeRequestsBeforeMinting(t *testing.T) {
	isolatedFPVRegistry(t)
	cases := []struct {
		name, sessionURL, body string
	}{
		{"missing consent", "https://example.com/public", `{"session_id":"session:route","expected_origin":"https://example.com"}`},
		{"origin mismatch", "https://example.com/public", `{"session_id":"session:route","expected_origin":"https://other.example","explicit_consent_ref":"consent:test"}`},
		{"sensitive origin", "https://accounts.example.com/", `{"session_id":"session:route","expected_origin":"https://accounts.example.com","explicit_consent_ref":"consent:test"}`},
		{"control", "https://example.com/public", `{"session_id":"session:route","expected_origin":"https://example.com","explicit_consent_ref":"consent:test","controls":true}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountFPVRoutes(router, fakeFPVSessionStore{"session:route": {ID: "session:route", URL: testCase.sessionURL}})
			request := httptest.NewRequest(http.MethodPost, "/share", bytes.NewBufferString(testCase.body))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden || bytes.Contains(response.Body.Bytes(), []byte(`"token"`)) || fpvSessionShareCount("session:route") != 0 {
				t.Fatalf("unsafe request response=%d body=%s shares=%d", response.Code, response.Body.String(), fpvSessionShareCount("session:route"))
			}
		})
	}
}

func TestFPVShareRouteMintsOnlyExplicitBoundedReadOnlyCapability(t *testing.T) {
	isolatedFPVRegistry(t)
	before := fpvSessionShareCount("session:allowed")
	defer func() { _, _ = fpvRevokeSessionShares("session:allowed") }()
	router := chi.NewRouter()
	MountFPVRoutes(router, fakeFPVSessionStore{"session:allowed": {ID: "session:allowed", URL: "https://example.com/public"}})
	request := httptest.NewRequest(http.MethodPost, "/share", bytes.NewBufferString(`{"session_id":"session:allowed","expected_origin":"https://example.com","explicit_consent_ref":"consent:test","expires_minutes":5,"max_views":1,"one_time":true}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || payload["controls"] != false || payload["mode"] != "read_only" || payload["max_views"] != float64(1) || fpvSessionShareCount("session:allowed") != before+1 {
		t.Fatalf("allowed response=%d payload=%#v shares=%d", response.Code, payload, fpvSessionShareCount("session:allowed"))
	}
	if token, ok := payload["token"].(string); !ok || len(token) != 32 {
		t.Fatalf("missing bounded capability: %#v", payload)
	}
}

func fpvSessionShareCount(sessionID string) int {
	count := 0
	fpvShares.Range(func(_, value any) bool {
		entry, ok := value.(*fpvShare)
		if ok && entry.SessionID == sessionID && !entry.Revoked {
			count++
		}
		return true
	})
	return count
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
