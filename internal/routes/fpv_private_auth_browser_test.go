package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestPublicFPVShareRejectsAuthenticationSessionsBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	t.Chdir(t.TempDir())
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
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
	cfg := &config.Config{}
	isolateEvidencePublicationStorage(t, cfg)
	router := chi.NewRouter()
	MountFPVRoutes(router, cfg, manager)
	for _, state := range []string{"fresh", "pending", "pending_markers_removed"} {
		session, _, err := manager.Open(site.URL, 640, 480)
		if err != nil {
			t.Fatal(err)
		}
		if state != "fresh" {
			if _, err = session.DetectAuthChallenge(); err != nil {
				t.Fatal(err)
			}
		}
		if state == "pending_markers_removed" {
			if _, _, err = session.Eval(`document.body.replaceChildren(); return true`); err != nil {
				t.Fatal(err)
			}
		}
		before := 0
		fpvShares.Range(func(_, _ any) bool { before++; return true })
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/share", strings.NewReader(`{"session_id":"`+session.ID+`"}`)))
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "fpv_private_session") {
			t.Fatalf("public auth share was not denied (state=%s status=%d)", state, response.Code)
		}
		if strings.Contains(response.Body.String(), `"token"`) || strings.Contains(response.Body.String(), "resume_") {
			t.Fatal("denial exposed a capability")
		}
		after := 0
		fpvShares.Range(func(_, _ any) bool { after++; return true })
		if after != before {
			t.Fatal("denied request minted a public share")
		}
		if session.CurrentAuthChallenge() == nil {
			t.Fatal("private auth state was lost")
		}
		manager.Close(session.ID)
	}
}
