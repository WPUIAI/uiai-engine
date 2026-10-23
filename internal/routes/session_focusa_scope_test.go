package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
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

func TestCompleteEvidenceScopeIsRetained(t *testing.T) {
	scope := &vision.FocusaScope{
		ProjectRef:    "wirebot-app",
		WorkstreamRef: "wirebot-app-execution-v1",
		WorksetRef:    "wirebot-app-acceptance-v2",
		CallGraphRef:  "wirebot-app-execution-v1",
		WorkpointRef:  "wp-1",
		WorkItemRef:   "wirebot-app-zr7",
		ContinuityRef: "focusa-cont-wirebot-app-user0",
		WorkItems: []evidencepwa.WorkItemProjection{{
			WorkItemRef: "wirebot-app-zr7", ProviderSurface: "beads",
		}},
	}

	got := resolveFocusaScope(scope, "", "", "", "")
	if got == nil || got.WorksetRef != scope.WorksetRef || got.CallGraphRef != scope.CallGraphRef || got.WorkItemRef != scope.WorkItemRef {
		t.Fatalf("complete scope lost: %#v", got)
	}
	if len(got.WorkItems) != 1 || got.WorkItems[0].WorkItemRef != "wirebot-app-zr7" {
		t.Fatalf("work-item projection lost: %#v", got.WorkItems)
	}
}

func TestMergeFocusaScopeKeepsExistingAndCaptureBindings(t *testing.T) {
	current := &vision.FocusaScope{ProjectRoot: "/home/wirebot/Wirebot-App", ContinuityID: "cont-1", WorksetRef: "workset-1"}
	incoming := &vision.FocusaScope{
		WorkpointRef: "wp-1", WorkItemRef: "wirebot-app-zr7",
		WorkItems: []evidencepwa.WorkItemProjection{{WorkItemRef: "wirebot-app-zr7"}},
	}

	got := mergeFocusaScope(current, incoming)
	if got.ProjectRoot != current.ProjectRoot || got.ContinuityID != current.ContinuityID || got.WorksetRef != current.WorksetRef {
		t.Fatalf("session binding lost: %#v", got)
	}
	if got.WorkpointRef != incoming.WorkpointRef || got.WorkItemRef != incoming.WorkItemRef || len(got.WorkItems) != 1 {
		t.Fatalf("capture binding lost: %#v", got)
	}
}

func TestRouteFocusaScopeStatusAcceptsCompleteEvidenceRefs(t *testing.T) {
	scope := &vision.FocusaScope{ProjectRef: "wirebot-app", ContinuityRef: "focusa-cont-wirebot-app-user0"}
	if got := routeFocusaScopeStatus(scope); got != "present" {
		t.Fatalf("expected complete evidence refs to be present, got %q", got)
	}
}
