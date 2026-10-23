package routes

import (
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

func legacyViewerResponse(t *testing.T, entry *shareEntry) *httptest.ResponseRecorder {
	t.Helper()
	token := "viewer-fixture-" + strings.ReplaceAll(t.Name(), "/", "-")
	if entry != nil {
		shareStore.Store(token, entry)
		t.Cleanup(func() { shareStore.Delete(token) })
	}
	router := chi.NewRouter()
	router.Get("/v/{token}", HandleShareViewer(&config.Config{}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v/"+token, nil))
	return response
}

func TestLegacyShareViewerEscapesStoredMetadata(t *testing.T) {
	title := `</title><script id="synthetic-injection">window.injected=true</script><img src=x onerror=alert(1)>`
	entry := &shareEntry{Title: title, URL: `https://example.test/path?x=%22%3E%3Cscript%3E&code=SYNTHETIC_VIEWER_CANARY`, ExpiresAt: time.Now().Add(time.Hour)}
	response := legacyViewerResponse(t, entry)
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("valid HTTPS target failed: %d", response.Code)
	}
	if strings.Contains(body, `<script id=`) || strings.Contains(body, `<img src=x`) {
		t.Fatal("stored title became executable markup")
	}
	if !strings.Contains(body, html.EscapeString(title)) {
		t.Fatal("title was not rendered as inert text")
	}
	if strings.Contains(body, "SYNTHETIC_VIEWER_CANARY") {
		t.Fatal("callback value leaked through shared URL")
	}
	if !strings.Contains(body, " sandbox ") || !strings.Contains(body, `referrerpolicy="no-referrer"`) {
		t.Fatal("embedded target lacks sandbox/referrer policy")
	}
}

func TestLegacyShareViewerRejectsUnsafeURLClasses(t *testing.T) {
	for _, target := range []string{"javascript:alert(1)", "data:text/html,synthetic", "//example.test/path", "https:///missing", "https://user:password@example.test/", "https://bad host/", "https://example.test:invalid/", "https://example.test:70000/", "https://example.test:/", "https://-bad..test/", "https://example.test/\nsynthetic", "https://example.test/%0d%0aevil"} {
		t.Run(strings.ReplaceAll(target, "/", "_"), func(t *testing.T) {
			response := legacyViewerResponse(t, &shareEntry{URL: target, ExpiresAt: time.Now().Add(time.Hour)})
			if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "<iframe") {
				t.Fatalf("unsafe target was rendered: status=%d", response.Code)
			}
		})
	}
}

func TestLegacyShareViewerSecurityHeadersAndExpiry(t *testing.T) {
	for _, test := range []struct {
		name   string
		entry  *shareEntry
		status int
	}{
		{"missing", nil, http.StatusNotFound},
		{"expired", &shareEntry{URL: "https://example.test", ExpiresAt: time.Now().Add(-time.Hour)}, http.StatusGone},
		{"valid", &shareEntry{URL: "https://example.test", ExpiresAt: time.Now().Add(time.Hour)}, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := legacyViewerResponse(t, test.entry)
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d", response.Code, test.status)
			}
			for key, want := range map[string]string{"Content-Security-Policy": legacyShareCSP, "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "X-Frame-Options": "DENY", "X-Robots-Tag": "noindex, noarchive", "Cache-Control": "no-store"} {
				if response.Header().Get(key) != want {
					t.Errorf("header %s is not hardened", key)
				}
			}
		})
	}
}
