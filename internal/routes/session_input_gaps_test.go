package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// #303: empty fill text clears a field; coordinate clicks reach canvas
// targets; wait accepts text. One fixture exercises all three contracts.
func TestSessionInputGapsFillClearClickXYAndWaitText(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for the browser-backed input-gap regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><html><body>
<input id="field" value="initial">
<canvas id="board" width="640" height="480" style="width:640px;height:480px"></canvas>
<div id="later" hidden>loaded-marker</div>
<script>
document.getElementById('board').addEventListener('click', function(e) {
  const c = this.getContext('2d');
  c.fillStyle = '#00ff00';
  c.fillRect(e.offsetX - 2, e.offsetY - 2, 4, 4);
  this.dataset.hit = e.offsetX + ',' + e.offsetY;
  setTimeout(() => { document.getElementById('later').hidden = false; }, 150);
});
</script></body></html>`)
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
	router := chi.NewRouter()
	cfg := &config.Config{}
	isolateEvidencePublicationStorage(t, cfg)
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, cfg, manager) })

	// 1. Empty fill text clears the field (no 400, value becomes empty).
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/session/"+session.ID+"/fill", strings.NewReader(`{"selector":"#field","text":""}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("empty fill rejected: %d %s", response.Code, response.Body.String())
	}
	valueRaw, _, err := session.Eval(`return document.getElementById('field').value`)
	if err != nil {
		t.Fatal(err)
	}
	if valueRaw != "" {
		t.Fatalf("fill with empty text did not clear the field: %q", valueRaw)
	}

	// 2. Click at coordinates reaches the canvas and registers a hit.
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/session/"+session.ID+"/click", strings.NewReader(`{"x":120,"y":90}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("coordinate click rejected: %d %s", response.Code, response.Body.String())
	}
	hit, _, err := session.Eval(`return document.getElementById('board').dataset.hit || ''`)
	if err != nil {
		t.Fatal(err)
	}
	if hit == "" {
		t.Fatal("coordinate click never reached the canvas")
	}

	// 3. wait accepts a text condition, not only selectors.
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/session/"+session.ID+"/wait", strings.NewReader(`{"text":"loaded-marker","timeout_ms":3000}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("wait by text failed: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Title string `json:"title"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &payload)

	// 4. Policy exposure: the session list reports effective limits (#303).
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/session", nil))
	var list struct {
		Max           int `json:"max"`
		MaxGlobal     int `json:"max_global"`
		IdleTTLMinutes int `json:"idle_ttl_minutes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Max < 1 || list.MaxGlobal < 1 || list.IdleTTLMinutes < 1 {
		t.Fatalf("session list does not report effective limits: %+v", list)
	}
}
