package vision

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStructuredTitleCredentialRedactionBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	const canary = "UIAI_SYNTHETIC_TITLE_0123456789"
	titles := []string{
		`{"password":"` + canary + `","echo":"` + canary + `"}`,
		`Status {"password":["` + canary + `"],"echo":"` + canary + `"} suffix`,
		`{"echo":"` + canary + `","padding":"` + strings.Repeat("x", 1400) + `","password":"` + canary + `"}`,
	}
	for _, title := range titles {
		func() {
			site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = fmt.Fprintf(w, "<html><title>%s</title><body>ordinary fixture</body></html>", html.EscapeString(title))
			}))
			defer site.Close()
			pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			manager := NewSessionManager(pool)
			sess, initial, err := manager.Open(site.URL, 800, 600)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(sess.ID)
			assertClean := func(value any) {
				t.Helper()
				encoded, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(encoded), canary) {
					t.Fatal("synthetic credential escaped page-title metadata")
				}
			}
			assertClean(initial)
			assertClean(sess.Title)
			value, snapshot, err := sess.Eval(`return "` + canary + `"`)
			if err != nil {
				t.Fatal(err)
			}
			assertClean(value)
			assertClean(snapshot)
			if !strings.Contains(value, "REDACTED_SECRET") {
				t.Fatal("title-labelled output alias was not redacted")
			}
			read, err := sess.ReadPage(ReadOptions{MaxChars: 8000})
			if err != nil {
				t.Fatal(err)
			}
			assertClean(read)
			assertClean(sess.Diagnostics(20, "all", false))
			if read.Text != "ordinary fixture" {
				t.Fatal("ordinary page content changed")
			}
			wrapped := WrapPage(sess.page, site.URL, 800, 600)
			if wrapped.diagnosticsCancel != nil {
				defer wrapped.diagnosticsCancel()
			}
			assertClean(wrapped.Title)
		}()
	}
}
