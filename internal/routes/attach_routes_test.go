package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// #143 acceptance 3: attach fails safe. Disabled deployments answer with a
// typed refusal; enabled deployments accept only allow-listed endpoint names
// and never a caller-supplied debugger URL.
func TestAttachRoutesFailSafe(t *testing.T) {
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()

	disabled := &config.Config{}
	router := chi.NewRouter()
	MountAttachRoutes(router, disabled, manager, vision.NewAttachRegistry())

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest("POST", "/attach", strings.NewReader(`{"endpoint":"owner-chrome","url":"https://example.test/"}`)))
	if resp.Code != http.StatusNotFound || !strings.Contains(resp.Body.String(), "attach_disabled") {
		t.Fatalf("disabled deployment not typed-refused: status=%d body=%.150s", resp.Code, resp.Body.String())
	}

	listing := httptest.NewRecorder()
	router.ServeHTTP(listing, httptest.NewRequest("GET", "/attach", nil))
	if listing.Code != http.StatusOK || !strings.Contains(listing.Body.String(), `"enabled":false`) {
		t.Fatalf("disabled listing not bounded: status=%d body=%.150s", listing.Code, listing.Body.String())
	}

	enabled := &config.Config{}
	enabled.Attach.Enabled = true
	enabled.Attach.Endpoints = []config.AttachEndpoint{{Name: "owner-chrome", URL: "http://127.0.0.1:19998", AllowPrivate: false}}
	router2 := chi.NewRouter()
	MountAttachRoutes(router2, enabled, manager, vision.NewAttachRegistry())

	unknown := httptest.NewRecorder()
	router2.ServeHTTP(unknown, httptest.NewRequest("POST", "/attach", strings.NewReader(`{"endpoint":"attacker-named","url":"https://example.test/"}`)))
	if unknown.Code != http.StatusBadRequest || !strings.Contains(unknown.Body.String(), "attach_endpoint_unknown") {
		t.Fatalf("unknown endpoint not typed-refused: status=%d body=%.150s", unknown.Code, unknown.Body.String())
	}

	// A caller cannot smuggle a debugger URL: the request shape has no such
	// field, and the configured-but-dead endpoint surfaces as a session error,
	// never a silent connect to a caller address.
	dead := httptest.NewRecorder()
	router2.ServeHTTP(dead, httptest.NewRequest("POST", "/attach", strings.NewReader(`{"endpoint":"owner-chrome","url":"https://example.test/"}`)))
	if dead.Code != http.StatusInternalServerError || strings.Contains(dead.Body.String(), "webSocketDebugger") {
		t.Fatalf("dead endpoint not a typed session error: status=%d body=%.150s", dead.Code, dead.Body.String())
	}

	list2 := httptest.NewRecorder()
	router2.ServeHTTP(list2, httptest.NewRequest("GET", "/attach", nil))
	if !strings.Contains(list2.Body.String(), `"name":"owner-chrome"`) {
		t.Fatalf("allow-listed endpoint missing from bounded listing: %.150s", list2.Body.String())
	}
}
