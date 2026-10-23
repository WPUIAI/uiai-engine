package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadAuthRestorationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for real browser regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>storage fixture</body></html>`))
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
	state, _ := json.Marshal(map[string]any{"url": site.URL, "localStorage": map[string]any{site.URL + "/": map[string]string{"probe": "restored"}, "https://unrelated.test": map[string]string{"foreign": "never-copy"}}, "sessionStorage": map[string]string{"session_probe": "restored"}})
	report, err := sess.LoadAuth(state)
	if err != nil {
		t.Fatal(err)
	}
	if report["localstorage_keys"] != 1 || report["sessionstorage_keys"] != 1 {
		t.Fatalf("wrong restoration counts: %#v", report)
	}
	result, err := sess.page.Eval(`() => JSON.stringify([localStorage.getItem('probe'),localStorage.getItem('foreign'),sessionStorage.getItem('session_probe')])`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Str() != `["restored",null,"restored"]` {
		t.Fatalf("incorrect restored values: %s", result.Value.Str())
	}
	if _, err = sess.page.Eval(`() => { Storage.prototype.setItem = function() { throw new Error('synthetic write failure'); }; }`); err != nil {
		t.Fatal(err)
	}
	if _, err = sess.LoadAuth(json.RawMessage(`{"localStorage":{"probe":"new"}}`)); err == nil {
		t.Fatal("storage write failure reported success")
	}
}
