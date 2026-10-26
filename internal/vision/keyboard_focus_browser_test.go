package vision

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPressReactivatesBackgroundSessionBrowser(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser input regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body tabindex="0">synthetic keyboard fixture</body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	first, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(first.ID)
	if _, err = first.page.Eval(`() => { window.__keys=[]; window.__keyFocus=[]; document.addEventListener('keydown',e=>{window.__keys.push(e.key);window.__keyFocus.push(document.hasFocus());}); document.body.focus(); }`); err != nil {
		t.Fatal(err)
	}
	second, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(second.ID)
	if _, err = second.page.Activate(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"a", "Enter"} {
		if _, err = first.Press(key); err != nil {
			t.Fatal(err)
		}
	}
	result, err := first.page.Eval(`() => JSON.stringify({keys:window.__keys,keydown_focus:window.__keyFocus})`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Str() != `{"keys":["a","Enter"],"keydown_focus":[true,true]}` {
		t.Fatalf("key delivery to background session failed: %s", result.Value.Str())
	}
}
