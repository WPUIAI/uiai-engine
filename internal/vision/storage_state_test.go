package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestStorageStateExportImportRoundTrip proves the #81 reuse path: a
// brokered login's authenticated state (cookies + web storage) exports to
// an opaque blob and imports into a fresh session on the same origin -
// reuse without re-injecting secrets.
func TestStorageStateExportImportRoundTrip(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><span id="who"></span>
		<script>document.getElementById('who').textContent = localStorage.getItem('session-user') || 'anonymous';</script>
		</body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()

	// First session: an authenticated marker lands in cookie + storage.
	first, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := first.Eval(`document.cookie = 'authed=1; path=/'; localStorage.setItem('session-user', 'brokered-login'); return 'ok';`); err != nil {
		t.Fatalf("state setup failed: %v", err)
	}
	state, err := first.ExportStorageState()
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if state.Schema != StorageStateSchema || state.Origin == "" {
		t.Fatalf("blob schema invalid: %+v", state.Schema)
	}
	foundCookie := false
	for _, cookie := range state.Cookies {
		if cookie.Name == "authed" {
			foundCookie = true
		}
	}
	if !foundCookie {
		t.Fatalf("exported blob missing the authed cookie: %+v", state.Cookies)
	}
	if state.LocalStorage["session-user"] != "brokered-login" {
		t.Fatalf("exported blob missing localStorage: %+v", state.LocalStorage)
	}
	if err := manager.Close(first.ID); err != nil {
		t.Fatal(err)
	}

	// Fresh session on the same origin: import restores the state.
	second, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.ImportStorageState(state); err != nil {
		t.Fatalf("import failed: %v", err)
	}
	value, _, err := second.Eval(`return localStorage.getItem('session-user');`)
	if err != nil || value != "brokered-login" {
		t.Fatalf("imported localStorage missing: %v %q", err, value)
	}
	cookieRead, _, err := second.Eval(`return document.cookie.includes('authed=1');`)
	if err != nil || cookieRead != "true" {
		t.Fatalf("imported cookie missing: %v %q", err, cookieRead)
	}

	// The blob is opaque JSON with the declared schema.
	encoded, err := json.Marshal(state)
	if err != nil || len(encoded) == 0 {
		t.Fatalf("blob not serializable: %v", err)
	}

	// Schema mismatches are typed refusals, never silent application.
	if err := second.ImportStorageState(&StorageState{Schema: "other"}); err == nil {
		t.Fatal("mismatched schema accepted")
	}
}
