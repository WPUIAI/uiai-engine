package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestResolveFocusaScopeFromFlatFields(t *testing.T) {
	scope := resolveFocusaScope(nil, "wp1", "cont1", "/tmp/project", "ev1")
	if scope == nil {
		t.Fatal("scope is nil")
	}
	if scope.WorkpointID != "wp1" || scope.ContinuityID != "cont1" || scope.ProjectRoot != "/tmp/project" || scope.EvidenceRef != "ev1" {
		t.Fatalf("unexpected scope: %+v", scope)
	}
}

func TestSessionInfoPayloadIncludesFocusaScope(t *testing.T) {
	sess := &vision.Session{ID: "sid", URL: "https://example.test", FocusaScope: &vision.FocusaScope{WorkpointID: "wp1"}}
	payload := sessionInfoPayload(sess)
	scope, ok := payload["focusa_scope"].(*vision.FocusaScope)
	if !ok || scope.WorkpointID != "wp1" {
		t.Fatalf("missing focusa scope: %#v", payload["focusa_scope"])
	}
}
