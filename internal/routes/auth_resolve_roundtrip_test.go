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

// #82 acceptance: a detected login wall enters the pending lease exactly
// once, the pending envelope blocks ordinary ops, and the valid in-process
// resume token resolves the challenge through the route so ops resume —
// the human/broker round trip the escalation protocol names.
func TestAuthChallengeResolveRoundTripBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for auth resolve round-trip browser")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		t.Fatalf("open: %v", err)
	}
	challenge, err := session.DetectAuthChallenge()
	if err != nil || challenge == nil {
		t.Fatalf("login wall not detected: %v %v", err, challenge)
	}
	if session.LeaseState() != vision.LeaseAwaitingAuth {
		t.Fatalf("challenge did not enter the pending lease: %s", session.LeaseState())
	}

	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{}, manager)

	// While pending: ordinary ops return the typed pending envelope.
	pending := httptest.NewRecorder()
	router.ServeHTTP(pending, httptest.NewRequest("POST", "/"+session.ID+"/eval", strings.NewReader(`{"js":"return 1;"}`)))
	if pending.Code != http.StatusConflict || !strings.Contains(pending.Body.String(), "auth_challenge_pending") {
		t.Fatalf("pending op missing the typed envelope: status=%d body=%.200s", pending.Code, pending.Body.String())
	}
	if strings.Contains(pending.Body.String(), challenge.ResumeToken) {
		t.Fatal("pending envelope leaked the private resume token")
	}

	// The human/broker resolution: the valid token clears the lease.
	resolved := httptest.NewRecorder()
	router.ServeHTTP(resolved, httptest.NewRequest("POST", "/"+session.ID+"/auth/resolve",
		strings.NewReader(`{"resume_token":"`+challenge.ResumeToken+`"}`)))
	if resolved.Code != http.StatusOK || !strings.Contains(resolved.Body.String(), `"resolved":true`) {
		t.Fatalf("valid resume token did not resolve: status=%d body=%.200s", resolved.Code, resolved.Body.String())
	}
	if session.LeaseState() != vision.LeaseActive || session.CurrentAuthChallenge() != nil {
		t.Fatalf("lease not restored after resolve: %s %v", session.LeaseState(), session.CurrentAuthChallenge())
	}

	// Ops resume immediately after resolution.
	after := httptest.NewRecorder()
	router.ServeHTTP(after, httptest.NewRequest("POST", "/"+session.ID+"/eval", strings.NewReader(`{"js":"return 1;"}`)))
	if after.Code == http.StatusConflict || strings.Contains(after.Body.String(), "auth_challenge_pending") {
		t.Fatalf("ops still blocked after resolution: status=%d body=%.200s", after.Code, after.Body.String())
	}
}
