package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFPVExistingShareRejectsAuthenticationTransitionBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("actual Chromium privacy transition")
	}
	t.Chdir(t.TempDir())
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>Owned fixture</title><body>Public observation</body>`)
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	router := chi.NewRouter()
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	for _, path := range []string{"", "/status", "/events", "/screenshot.jpg", "/stream.cdp.mjpg", "/stream.mjpg"} {
		t.Run(path, func(t *testing.T) {
			session, _, err := manager.Open(site.URL, 320, 240)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			token := fmt.Sprintf("synthetic-auth-transition-%d", time.Now().UnixNano())
			fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
			defer fpvShares.Delete(token)
			before := httptest.NewRecorder()
			router.ServeHTTP(before, httptest.NewRequest("GET", "/m/"+token+"/status", nil))
			if before.Code != http.StatusOK {
				t.Fatalf("ordinary public session status=%d", before.Code)
			}
			if _, _, err := session.Eval(`document.body.innerHTML='<input type="password">'; return true;`); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", "/m/"+token+path, nil).WithContext(ctx))
			if response.Code != http.StatusForbidden && response.Code != http.StatusGone {
				t.Fatalf("existing public share survived authentication transition: status=%d", response.Code)
			}
			if strings.Contains(response.Body.String(), token) || strings.Contains(response.Body.String(), "resume_") {
				t.Fatal("privacy denial exposed a capability")
			}
			stored, ok := fpvShares.Load(token)
			if !ok || !stored.(*fpvShare).Revoked {
				t.Fatal("authentication transition did not permanently revoke the existing share")
			}
			challenge := session.CurrentAuthChallenge()
			if challenge == nil {
				t.Fatal("fresh authentication transition was not detected")
			}
			if err := session.ResolveAuthChallenge(challenge.ResumeToken); err != nil {
				t.Fatal(err)
			}
			if _, available := fpvEntry(token); available {
				t.Fatal("resolved authentication resurrected the public capability")
			}
		})
	}
}
