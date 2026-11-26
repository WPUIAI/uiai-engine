package routes

import (
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

// #147: public FPV sharing defaults to deny for health-origin sessions
// (patient-portal/medical-record contexts) and payment SCA sessions, while
// ordinary public pages keep the normal share path.
func TestPublicFPVShareOriginCategoriesBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	t.Chdir(t.TempDir())

	newRouter := func() (*chi.Mux, *vision.SessionManager, *vision.SessionManager) {
		pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
		if err != nil {
			t.Fatal(err)
		}
		manager := vision.NewSessionManager(pool)
		cfg := &config.Config{}
		isolateEvidencePublicationStorage(t, cfg)
		router := chi.NewRouter()
		MountFPVRoutes(router, cfg, manager)
		t.Cleanup(func() { manager.CloseAll(); pool.Close() })
		return router, manager, nil
	}

	share := func(router *chi.Mux, sessionID string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/share", strings.NewReader(`{"session_id":"`+sessionID+`"}`)))
		return response
	}

	// Patient-portal context: multiple health markers default-deny.
	healthSite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Patient Portal</h1>
		<p>Welcome back. Your medical record and prescription history are below, along with your latest lab results.</p>
		</body></html>`)
	}))
	defer healthSite.Close()
	router, manager, _ := newRouter()
	session, _, err := manager.Open(healthSite.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	response := share(router, session.ID)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "fpv_health_origin") {
		t.Fatalf("patient-portal share not denied: status=%d body=%.200s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"token"`) {
		t.Fatal("health denial exposed a capability")
	}

	// Payment SCA context: denied through the auth-challenge guard.
	paymentSite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Confirm your payment</h1>
		<p>3-D Secure verification required. Complete Strong Customer Authentication to continue.</p>
		</body></html>`)
	}))
	defer paymentSite.Close()
	router2, manager2, _ := newRouter()
	session2, _, err := manager2.Open(paymentSite.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	response2 := share(router2, session2.ID)
	if response2.Code != http.StatusForbidden ||
		(!strings.Contains(response2.Body.String(), "fpv_private_session") && !strings.Contains(response2.Body.String(), "fpv_health_origin")) {
		t.Fatalf("payment share not denied: status=%d body=%.200s", response2.Code, response2.Body.String())
	}

	// Ordinary public page: the health guard must not over-block. A single
	// casual mention ("lab results were published in a news article") is not
	// a health context and must not trigger the typed denial.
	ordinarySite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Local News</h1>
		<p>A study about lab results reporting was covered today. No patient data involved.</p>
		</body></html>`)
	}))
	defer ordinarySite.Close()
	router3, manager3, _ := newRouter()
	session3, _, err := manager3.Open(ordinarySite.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	response3 := share(router3, session3.ID)
	if strings.Contains(response3.Body.String(), "fpv_health_origin") {
		t.Fatalf("ordinary page wrongly denied as health origin: status=%d body=%.200s", response3.Code, response3.Body.String())
	}
}
