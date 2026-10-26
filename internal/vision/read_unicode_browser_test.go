package vision

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"unicode/utf8"
)

func TestReadPageUnicodeCharacterBudgetBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser regression")
	}
	const content = "é😀文Z"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><meta charset="utf-8"></head><body><main><p>` + content + `</p></main></body></html>`))
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
		for budget := 1; budget <= 5; budget++ {
			t.Run(format+string(rune('0'+budget)), func(t *testing.T) {
				result, err := sess.ReadPage(ReadOptions{Format: format, MaxChars: budget})
				if err != nil {
					t.Fatal(err)
				}
				count := budget
				if count > 4 {
					count = 4
				}
				expected := string([]rune(content)[:count])
				if !utf8.ValidString(result.Text) || result.Text != expected {
					t.Fatalf("character boundary damaged: format=%s budget=%d got=%q expected=%q", format, budget, result.Text, expected)
				}
				if result.Chars != count || result.Truncated != (budget < 4) {
					t.Fatalf("wrong Unicode metadata: chars=%d truncated=%v", result.Chars, result.Truncated)
				}
			})
		}
	}
}
