package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFillHTTPTimeoutIsObservableAndSessionClosable(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for the browser-backed timeout regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><input id="slow"><script>
   document.querySelector('input').addEventListener('focus',()=>{
    const until=performance.now()+6500;while(performance.now()<until){}
   },{once:true});
  </script>`)
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	pid := pool.Stats()["browser_pid"]
	router := chi.NewRouter()
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, &config.Config{}, manager) })
	start := time.Now()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/session/"+session.ID+"/fill", strings.NewReader(`{"selector":"#slow","text":"SYNTHETIC_FILL_CANARY"}`)))
	if time.Since(start) > 8*time.Second {
		t.Fatal("HTTP fill exceeded bounded recovery window")
	}
	var failure struct {
		ErrorID string         `json:"error_id"`
		Class   string         `json:"error_class"`
		Details map[string]any `json:"details"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if response.Code < 400 || failure.ErrorID == "" || failure.Class != "timeout" {
		t.Fatalf("unstructured timeout response: status=%d class=%s id=%s", response.Code, failure.Class, failure.ErrorID)
	}
	if failure.Details["session_id"] != session.ID || failure.Details["action"] != "fill" {
		t.Fatal("timeout lost session/action correlation")
	}
	if strings.Contains(response.Body.String(), "SYNTHETIC_FILL_CANARY") {
		t.Fatal("timeout telemetry exposed input text")
	}
	found := false
	for _, event := range observability.Recent(100, "browser_session", "") {
		if event.ID == failure.ErrorID && event.SessionID == session.ID && event.Class == "timeout" {
			found = true
		}
	}
	if !found {
		t.Fatal("fill timeout was not recorded in error telemetry")
	}
	for _, request := range []struct{ method, path string }{{"GET", "/diagnostics"}, {"DELETE", "/"}} {
		done := make(chan *httptest.ResponseRecorder, 1)
		go func(method, path string) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(method, "/api/session/"+session.ID+path, nil))
			done <- w
		}(request.method, request.path)
		select {
		case result := <-done:
			if request.method == "DELETE" {
				if result.Code != 200 {
					t.Fatalf("close status=%d", result.Code)
				}
			} else {
				if result.Code != 200 && result.Code != 202 {
					t.Fatalf("diagnostics status=%d", result.Code)
				}
				var diagnostics map[string]any
				if err := json.Unmarshal(result.Body.Bytes(), &diagnostics); err != nil {
					t.Fatal(err)
				}
				if diagnostics["session_id"] != session.ID || diagnostics["summary"] == nil {
					t.Fatal("diagnostics did not return the session observations")
				}
				if result.Code == 202 && (diagnostics["artifact_payload_posture"] != "unsealed_no_epwa_scope" || diagnostics["delivery_state"] == "ready") {
					t.Fatal("fixture diagnostics misrepresented evidence delivery")
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("recovery route %s remained wedged", request.path)
		}
	}
	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("closed session remained active")
	}
	if pool.Stats()["browser_pid"] != pid {
		t.Fatal("timeout recovery required a browser restart")
	}
}
