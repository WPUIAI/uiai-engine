package vision

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/events"
)

func TestAuthChallengeLeasePausesUntilResolution(t *testing.T) {
	sess := &Session{ID: "sess-auth", ActorRef: "actor-1"}
	before := events.LatestSeq()
	challenge := sess.raiseAuthChallenge("credential_form", "https://example.test/login")
	if challenge.ResumeToken == "" || sess.LeaseState() != LeaseAwaitingAuth {
		t.Fatalf("challenge did not enter awaiting-auth state: %#v", challenge)
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
