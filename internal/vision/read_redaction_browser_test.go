package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestReadPageRedactsBeforeTruncationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser regression")
	}
	const dummy = "DUMMY_BEARER_VALUE_abcdefghijklmnopqrstuvwxyz"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Bearer ` + dummy + `</title></head><body><main><h1>Bearer ` + dummy + `</h1><p>Bearer ` + dummy + `</p><p>` + strings.Repeat("ordinary prose ", 150) + `</p><a href="https://example.test/?token=` + dummy + `">reference</a></main></body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	for _, format := range []string{"text", "markdown"} {
		result, err := sess.ReadPage(ReadOptions{Format: format, IncludeLinks: true, MaxChars: 8000})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), dummy) {
			t.Fatalf("%s leaked dummy bearer value", format)
		}
		if len(result.Text) <= 1000 {
			t.Fatal("page read was truncated to diagnostic-preview budget")
		}
	}
	result, err := sess.ReadPage(ReadOptions{MaxChars: 14})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Text, "DUMMY") {
		t.Fatal("truncation leaked credential prefix")
	}
}

func TestReadPageKeepsTokenTypeDocsAndWithholdsRealCredentialsBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser regression")
	}
	const canary = "UIAI_SYNTHETIC_DOCS_KEY_0123456789"
	const code = "switch tt {\ncase html.ErrorToken:\nreturn z.Err()\ncase html.TextToken:\nif depth > 0 { emit(z.Text()) }\n}\nspecification"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><main><h1>Tokenizer documentation</h1><pre><code>` + code + `</code></pre><dl><dt>API key</dt><dd><code>` + canary + `</code></dd></dl></main></body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	for _, format := range []string{"text", "markdown"} {
		result, err := sess.ReadPage(ReadOptions{Format: format, MaxChars: 8000})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("actual labelled credential escaped browser read")
		}
		for _, word := range []string{"return z.Err()", "if depth", "specification"} {
			if !strings.Contains(result.Text, word) {
				t.Fatal("ordinary tokenizer docs corrupted by secret aliases")
			}
		}
	}
}
