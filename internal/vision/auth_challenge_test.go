package vision

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/events"
)

func TestAuthChallengeLeasePausesUntilResolution(t *testing.T) {
	expired := make(chan struct{}, 1)
	sess := &Session{ID: "sess-auth", ActorRef: "actor-1", timer: time.AfterFunc(10*time.Millisecond, func() { expired <- struct{}{} })}
	before := events.LatestSeq()
	challenge := sess.raiseAuthChallenge("credential_form", "https://example.test/login")
	if challenge.ResumeToken == "" || sess.LeaseState() != LeaseAwaitingAuth {
		t.Fatalf("challenge did not enter awaiting-auth state: %#v", challenge)
	}
	select {
	case <-expired:
		t.Fatal("session timer expired while authentication challenge was pending")
	case <-time.After(25 * time.Millisecond):
	}
	if err := sess.ResolveAuthChallenge("wrong"); err == nil {
		t.Fatal("invalid resume token unexpectedly resolved challenge")
	}
	if err := sess.ResolveAuthChallenge(challenge.ResumeToken); err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if sess.LeaseState() != LeaseActive || sess.CurrentAuthChallenge() != nil {
		t.Fatalf("challenge remained active after resolution")
	}
	eventsSeen := events.Since(before)
	if len(eventsSeen) != 2 || eventsSeen[0].EventType != "auth_challenge_required" || eventsSeen[1].EventType != "auth_challenge_resolved" {
		t.Fatalf("unexpected challenge event sequence: %#v", eventsSeen)
	}
}

func TestAuthChallengeDetectorLoginWall(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("browser auth-challenge test requires UIAI_AUTH_TESTS=1")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><form action="/login"><input type="password"></form></body></html>`))
	}))
	defer server.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	sess, _, err := sm.Open(server.URL, 800, 600)
	if err != nil {
		t.Skipf("browser unavailable: %v", err)
	}
	defer sm.Close(sess.ID)
	before := events.LatestSeq()
	first, err := sess.DetectAuthChallenge()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	second, err := sess.DetectAuthChallenge()
	if err != nil {
		t.Fatalf("repeat detect: %v", err)
	}
	if first == nil || first.Class != "credential_form" || second.ResumeToken != first.ResumeToken {
		t.Fatalf("unexpected detection: first=%#v second=%#v", first, second)
	}
	if got := len(events.Since(before)); got != 1 {
		t.Fatalf("login wall emitted %d events, want exactly one", got)
	}
}

func TestAuthChallengeDuplicateDetectionIsIdempotent(t *testing.T) {
	sess := &Session{ID: "sess-duplicate"}
	before := events.LatestSeq()
	first := sess.raiseAuthChallenge("captcha", "https://example.test")
	second := sess.raiseAuthChallenge("captcha", "https://example.test")
	if first.ResumeToken != second.ResumeToken {
		t.Fatal("duplicate challenge replaced the active resume token")
	}
	if got := len(events.Since(before)); got != 1 {
		t.Fatalf("duplicate challenge emitted %d events, want 1", got)
	}
}
