package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestLegacyShareViewerInertDOMBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser-rendered share regression")
	}
	router := chi.NewRouter()
	router.Get("/v/{token}", HandleShareViewer(&config.Config{}))
	router.Get("/target", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!doctype html><p>synthetic shared page</p>"))
	})
	server := httptest.NewServer(router)
	defer server.Close()
	const token = "inert-viewer-fixture"
	const title = `</title><script data-xss-fixture>window.__xss=true</script><h2 data-xss-fixture>injected</h2>`
	shareStore.Store(token, &shareEntry{URL: server.URL + "/target", Title: title, ExpiresAt: time.Now().Add(time.Hour)})
	defer shareStore.Delete(token)
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	sess, _, err := manager.Open(server.URL+"/v/"+token, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	result, _, err := sess.Eval(`return JSON.stringify({injected:Boolean(window.__xss)||Boolean(document.querySelector('[data-xss-fixture]')),title:document.title,sandbox:document.querySelector('iframe').getAttribute('sandbox')});`)
	if err != nil {
		t.Fatal(err)
	}
	var observed struct {
		Injected bool   `json:"injected"`
		Title    string `json:"title"`
		Sandbox  string `json:"sandbox"`
	}
	if err = json.Unmarshal([]byte(result), &observed); err != nil {
		t.Fatal(err)
	}
	if observed.Injected || observed.Title != title || observed.Sandbox != "" {
		t.Fatal("stored title was not inert or frame was not sandboxed in browser DOM")
	}
}
