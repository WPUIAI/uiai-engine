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
	challenge, err := session.DetectAuthChallenge()
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{}, manager)
	info := httptest.NewRecorder()
	router.ServeHTTP(info, httptest.NewRequest("GET", "/"+session.ID+"/", nil))
	if info.Code != http.StatusOK || strings.Contains(info.Body.String(), challenge.ResumeToken) || strings.Contains(info.Body.String(), "\"resume_token\"") {
		t.Fatalf("ordinary session payload exposed private auth material: status=%d body=%s", info.Code, info.Body.String())
	}
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
		if strings.Contains(response.Body.String(), challenge.ResumeToken) {
			t.Fatal("ordinary pending auth response exposed the private resume token")
		}
	}
	cancelled := httptest.NewRecorder()
	router.ServeHTTP(cancelled, httptest.NewRequest("POST", "/"+session.ID+"/auth/cancel", strings.NewReader("{}")))
	if cancelled.Code != http.StatusOK || !strings.Contains(cancelled.Body.String(), "\"cancelled\":true") {
		t.Fatalf("auth cancel did not report closure: status=%d body=%s", cancelled.Code, cancelled.Body.String())
	}
	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("auth cancel did not close and evict the session")
	}
}
