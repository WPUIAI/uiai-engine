package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func clearShareViewerTestStore(t *testing.T) {
	t.Helper()
	clear := func() {
		shareStore.Range(func(key, _ any) bool {
			shareStore.Delete(key)
			return true
		})
	}
	clear()
	t.Cleanup(clear)
}

func requestShareViewer(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Get("/v/{token}", HandleShareViewer(nil))
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v/"+token, nil)
	router.ServeHTTP(recorder, req)
	return recorder
}

func assertShareViewerSecurityHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	want := map[string]string{
		"Content-Type":            "text/html; charset=utf-8",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"X-Frame-Options":         "DENY",
		"X-Robots-Tag":            "noindex, nofollow, noarchive",
		"Content-Security-Policy": "default-src 'none'; frame-src http: https:; base-uri 'none'; form-action 'none'; object-src 'none'; script-src 'none'; style-src 'none'; frame-ancestors 'none'",
	}
	for name, expected := range want {
		if actual := recorder.Header().Get(name); actual != expected {
			t.Fatalf("%s = %q, want %q", name, actual, expected)
		}
	}
}

func TestShareViewerEscapesStoredValuesAndPreservesValidLinks(t *testing.T) {
	clearShareViewerTestStore(t)
	entry := &shareEntry{
		ID:        "safe-token",
		Title:     `<script>alert("title")</script>`,
		URL:       `https://example.com/design?a=1&b=2`,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	shareStore.Store(entry.ID, entry)

	recorder := requestShareViewer(t, entry.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	assertShareViewerSecurityHeaders(t, recorder)
	body := recorder.Body.String()
	if strings.Contains(body, `<script>alert("title")</script>`) {
		t.Fatal("stored title rendered as active markup")
	}
	if !strings.Contains(body, `&lt;script&gt;alert(&#34;title&#34;)&lt;/script&gt;`) {
		t.Fatalf("escaped title missing: %s", body)
	}
	if !strings.Contains(body, `href="https://example.com/design?a=1&amp;b=2"`) ||
		!strings.Contains(body, `src="https://example.com/design?a=1&amp;b=2"`) {
		t.Fatalf("valid URL not rendered safely in link and iframe: %s", body)
	}
	if entry.Views != 1 {
		t.Fatalf("views = %d, want 1", entry.Views)
	}
}

func TestShareViewerRejectsUnsafeAndMalformedURLs(t *testing.T) {
	for _, raw := range []string{
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"mailto:test@example.com",
		"//example.com/missing-scheme",
		"https://",
		" https://example.com/leading-space",
		"https://example.com/header\r\ninjection",
	} {
		t.Run(raw, func(t *testing.T) {
			clearShareViewerTestStore(t)
			entry := &shareEntry{ID: "unsafe-token", Title: "unsafe", URL: raw, ExpiresAt: time.Now().Add(time.Hour)}
			shareStore.Store(entry.ID, entry)

			recorder := requestShareViewer(t, entry.ID)
			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body=%s", recorder.Code, recorder.Body.String())
			}
			assertShareViewerSecurityHeaders(t, recorder)
			if strings.Contains(recorder.Body.String(), raw) {
				t.Fatal("unsafe stored URL leaked into rendered response")
			}
			if entry.Views != 0 {
				t.Fatalf("invalid share incremented views to %d", entry.Views)
			}
		})
	}
}

func TestShareViewerPreservesNotFoundAndExpiredResponses(t *testing.T) {
	clearShareViewerTestStore(t)

	notFound := requestShareViewer(t, "missing-token")
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), "Share Not Found") {
		t.Fatalf("not found response = %d %q", notFound.Code, notFound.Body.String())
	}
	assertShareViewerSecurityHeaders(t, notFound)

	expired := &shareEntry{ID: "expired-token", URL: "https://example.com", ExpiresAt: time.Now().Add(-time.Minute)}
	shareStore.Store(expired.ID, expired)
	recorder := requestShareViewer(t, expired.ID)
	if recorder.Code != http.StatusGone || !strings.Contains(recorder.Body.String(), "Share Expired") {
		t.Fatalf("expired response = %d %q", recorder.Code, recorder.Body.String())
	}
	assertShareViewerSecurityHeaders(t, recorder)
	if _, ok := shareStore.Load(expired.ID); ok {
		t.Fatal("expired share remained in store")
	}
}
