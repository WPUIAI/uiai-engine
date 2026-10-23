package routes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestSessionInfoRedactsOAuthCallback(t *testing.T) {
	const raw = "https://example.test/callback?code=SYNTHETIC_ROUTE_CANARY&state=SYNTHETIC_ROUTE_CANARY#SYNTHETIC_ROUTE_CANARY"
	sess := &vision.Session{ID: "session-fixture", URL: raw}
	body, err := json.Marshal(sessionInfoPayload(sess))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "SYNTHETIC_ROUTE_CANARY") {
		t.Fatal("session info exposed callback values")
	}
	if sess.URL != raw {
		t.Fatal("session info modified browser navigation URL")
	}
}
