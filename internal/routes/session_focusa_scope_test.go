package routes

import (
	"encoding/json"
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

func TestFocusaScopeNormalizesCompleteEvidenceRefAliases(t *testing.T) {
	expected := completeEvidenceScope()
	payload, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var scope vision.FocusaScope
	if err := json.Unmarshal(payload, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.WorkpointID != expected.WorkpointRef || scope.ContinuityID != expected.ContinuityRef {
		t.Fatalf("evidence aliases were not normalized: %#v", scope)
	}
	if !scope.HasEvidenceBinding() || scope.ProjectReference() != expected.ProjectRef || scope.WorkstreamReference() != expected.WorkstreamRef {
		t.Fatalf("typed evidence references were not retained: %#v", scope)
	}
	delivered := evidenceScopeFromFocusa(&scope)
	if delivered.ProjectRef != expected.ProjectRef || delivered.WorkstreamRef != expected.WorkstreamRef || delivered.WorksetRef != expected.WorksetRef || delivered.CallGraphRef != expected.CallGraphRef || delivered.WorkpointRef != expected.WorkpointRef || delivered.WorkItemRef != expected.WorkItemRef || delivered.ContinuityRef != expected.ContinuityRef || len(delivered.WorkItems) != len(expected.WorkItems) {
		t.Fatalf("complete scope lost after normalization: %#v", delivered)
	}
}

func TestMergeFocusaScopePreservesCompleteEvidenceBindings(t *testing.T) {
	expected := completeEvidenceScope()
	current := &vision.FocusaScope{
		ProjectRoot: "/workspace/wirebot", ContinuityID: "legacy-continuity",
		WorkstreamKey: "/workspace/wirebot::legacy-continuity", EvidenceRef: "existing-evidence",
	}
	incoming := &vision.FocusaScope{
		ProjectRef: expected.ProjectRef, WorkstreamRef: expected.WorkstreamRef, WorksetRef: expected.WorksetRef,
		CallGraphRef: expected.CallGraphRef, WorkpointID: expected.WorkpointRef, WorkItemRef: expected.WorkItemRef,
		ContinuityID: expected.ContinuityRef, WorkItems: expected.WorkItems,
	}

	merged := mergeFocusaScope(current, incoming)
	if merged == nil {
		t.Fatal("merged scope is nil")
	}
	if merged.ProjectRoot != current.ProjectRoot || merged.EvidenceRef != current.EvidenceRef {
		t.Fatalf("existing session binding lost: %#v", merged)
	}
	if merged.ProjectRef != expected.ProjectRef || merged.WorkstreamRef != expected.WorkstreamRef || merged.WorksetRef != expected.WorksetRef || merged.CallGraphRef != expected.CallGraphRef || merged.WorkpointID != expected.WorkpointRef || merged.WorkItemRef != expected.WorkItemRef || merged.ContinuityID != expected.ContinuityRef {
		t.Fatalf("complete capture binding lost: %#v", merged)
	}
	if len(merged.WorkItems) != len(expected.WorkItems) || len(merged.WorkItems) == 0 {
		t.Fatalf("work-items binding lost: %#v", merged.WorkItems)
	}
	incoming.WorkItems[0].Title = "mutated after merge"
	if merged.WorkItems[0].Title == incoming.WorkItems[0].Title {
		t.Fatalf("work-items binding aliases caller input: %#v", merged.WorkItems)
	}

	sess := &vision.Session{}
	sess.SetFocusaScope(merged)
	if sess.FocusaScope.WorkstreamKey != "/workspace/wirebot::"+expected.ContinuityRef {
		t.Fatalf("workstream key was not recalculated: %#v", sess.FocusaScope)
	}
	delivered := evidenceScopeFromFocusa(sess.FocusaScope)
	if delivered.ProjectRef != expected.ProjectRef || delivered.WorkstreamRef != expected.WorkstreamRef || delivered.WorksetRef != expected.WorksetRef || delivered.CallGraphRef != expected.CallGraphRef || delivered.WorkpointRef != expected.WorkpointRef || delivered.WorkItemRef != expected.WorkItemRef || delivered.ContinuityRef != expected.ContinuityRef || len(delivered.WorkItems) != len(expected.WorkItems) {
		t.Fatalf("EPWA scope parity lost: %#v", delivered)
	}
}
