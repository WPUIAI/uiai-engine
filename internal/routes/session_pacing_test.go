package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestPacingHTTPProfileSequenceAndAuthGuard(t *testing.T) {
	if os.Getenv("UIAI_PACING_TESTS") == "" {
		t.Skip("set UIAI_PACING_TESTS=1 for owned route fixture")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input id="a"><input id="b"><input id="code"><button id="submit" onclick="window.fixtureSubmitted=true">Submit</button></body></html>`))
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
		t.Fatal("explicit pacing HTTP fixture unavailable")
	}
	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{}, manager)
	call := func(action, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/"+session.ID+"/"+action, bytes.NewBufferString(body)))
		return response
	}
	for _, body := range []string{"null", `{"mode":"unknown"}`, `{"mode":"human","unknown":1}`, `{"mode":"human"} {}`} {
		if call("pace", body).Code != 400 {
			t.Fatal("invalid profile request accepted")
		}
	}
	if response := call("pace", `{"mode":"custom","median_ms":2,"min_click_ms":1,"cps_min":200,"cps_max":400,"seed":7}`); response.Code != 200 {
		t.Fatal("valid profile rejected")
	}
	response := call("sequence", `{"total_budget_ms":4000,"operations":[{"action":"type","selector":"#a","text":"alpha"},{"action":"fill","selector":"#b","text":"beta"},{"action":"type","selector":"#code","text":"synthetic-only"},{"action":"click","selector":"#submit"}]}`)
	// Artifact delivery can remain blocked; action completion is verified separately.
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if result, ok := body["sequence"].(map[string]any); !ok || result["completed"] != float64(4) {
		t.Fatalf("HTTP sequence completion not retained; status=%d", response.Code)
	}
	if session.Pacing().ActionWaits != 4 || session.Pacing().KeyWaits < 3 {
		t.Fatal("configured HTTP actions did not use pacing")
	}
	if response := call("sequence", `{"total_budget_ms":0,"operations":[]}`); response.Code != 400 {
		t.Fatal("invalid budget accepted")
	}
	if response := call("pace", `{"mode":"instant"}`); response.Code != 200 || session.Pacing() != nil {
		t.Fatal("instant reset changed default contract")
	}
	// Changing the page to a synthetic password form may pause, never bypass, auth.
	if _, _, err := session.Eval(`document.body.innerHTML='<input type="password">'; return true;`); err != nil {
		t.Fatal(err)
	}
	if _, err := session.DetectAuthChallenge(); err != nil {
		t.Fatal(err)
	}
	if response := call("pace", `{"mode":"human"}`); response.Code != 409 {
		t.Fatal("profile route bypassed auth guard")
	}
	if response := call("sequence", `{"total_budget_ms":1000,"operations":[{"action":"click","selector":"button"}]}`); response.Code != 409 {
		t.Fatal("sequence bypassed auth guard")
	}
}
