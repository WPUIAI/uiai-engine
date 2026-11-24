package vision

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// #320/#152: engine sessions must carry the page's real Origin on outbound
// cross-origin requests. The old pool launched Chromium with web security
// disabled, which stamps `Origin: null` on every cross-origin request and is
// exactly what makes domain-allowlisted provider SDKs (Stytch, OIDC, Google
// OAuth) reject engine sessions as opaque origins.
func TestEngineBrowserSendsRealOriginOnCrossOriginRequests(t *testing.T) {
	var lastOrigin atomic.Value
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodOptions {
			// CORS preflight: permit the page origin so the actual POST fires.
			w.Header().Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		lastOrigin.Store(req.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
		writeJSONTest(w, map[string]string{"ok": "true"})
	}))
	defer echo.Close()

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body><h1>origin probe</h1></body></html>`)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}

	js := fmt.Sprintf(`() => fetch(%q, {method:"POST", headers:{"Content-Type":"application/json"}, body:"{}"}).then(r => r.status).catch(() => -1)`, echo.URL)
	result, err := session.page.Eval(js)
	if err != nil {
		t.Fatal(err)
	}
	if status := result.Value.Str(); status != "200" {
		t.Fatalf("cross-origin probe fetch blocked: status=%s", status)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if stored, ok := lastOrigin.Load().(string); ok {
			if stored == "" || stored == "null" {
				t.Fatalf("engine browser sent a null/absent Origin: %q", stored)
			}
			if stored != site.URL {
				t.Fatalf("engine browser Origin %q does not match the page origin %q", stored, site.URL)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("cross-origin probe request never reached the echo server")
}

func writeJSONTest(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
