package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestAuthPendingResponseOffersPrivateModeWithoutResuming(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for private synthetic auth browser")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input type="password"></body></html>`))
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
		t.Fatal("explicit auth route fixture unavailable")
	}
	if _, err := session.DetectAuthChallenge(); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{}, manager)
	for _, action := range []string{"click", "auth/resolve"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/"+session.ID+"/"+action, strings.NewReader(`{"resume_token":"synthetic-invalid"}`)))
		var result struct {
			Code     string `json:"code"`
			NextMode string `json:"next_mode"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusConflict || result.Code != "auth_challenge_pending" || result.NextMode != browserModePrivateAuth {
			t.Fatal("pending auth lacks bounded mode recommendation")
		}
		if session.LeaseState() != vision.LeaseAwaitingAuth {
			t.Fatal("discovery hint resumed pending authentication")
		}
	}
}
