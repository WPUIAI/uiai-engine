package vision

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/events"
)

func TestAuthChallengeMetadataUsesCanonicalURLRedaction(t *testing.T) {
	const secret = "synthetic-private"
	sess := &Session{ID: "fixture-auth-redaction"}
	before := events.LatestSeq()
	challenge := sess.raiseAuthChallenge("credential_form", "https://user:"+secret+"@example.test/login?token="+secret+"#"+secret)
	for _, value := range []any{challenge, sess.CurrentAuthChallenge(), events.Since(before)} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "user:") {
			t.Fatal("auth challenge metadata/event disclosed private URL material")
		}
	}
	if challenge.URLHint != "https://example.test/login" {
		t.Fatal("auth hint lost safe route context")
	}
}

func TestAuthChallengeReturnedMetadataCannotMutatePendingState(t *testing.T) {
	sess := &Session{ID: "fixture-auth-copy"}
	first := sess.raiseAuthChallenge("credential_form", "https://example.test/login")
	original := first.ResumeToken
	first.ResumeToken = "forged"
	first.Class = "modified"
	first.URLHint = "https://changed.invalid"
	stored := sess.CurrentAuthChallenge()
	if stored.ResumeToken != original || stored.Class != "credential_form" || stored.URLHint != "https://example.test/login" {
		t.Fatal("returned metadata alias mutated pending authentication state")
	}
	if err := sess.ResolveAuthChallenge("forged"); err == nil {
		t.Fatal("mutated copy authorized challenge resolution")
	}
	if err := sess.ResolveAuthChallenge(original); err != nil {
		t.Fatal("original token lost after independent metadata mutation")
	}
}
