package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// The session scope accepts two spellings for the same two bindings. Reading only
// workpoint_id/continuity_id blocked delivery for a caller that sent the
// Focusa-native refs, so both spellings must produce the same complete scope.
func TestEvidenceScopeFromFocusaAcceptsBothSpellings(t *testing.T) {
	byID := evidenceScopeFromFocusa(&vision.FocusaScope{
		WorkpointID: "workpoint:1", ContinuityID: "continuity:2",
		ProjectRef: "project:p", WorkstreamRef: "workstream:w", WorksetRef: "workset:s",
		CallGraphRef: "callgraph:c", WorkItemRef: "work-item:i",
	})
	byRef := evidenceScopeFromFocusa(&vision.FocusaScope{
		WorkpointRef: "workpoint:1", ContinuityRef: "continuity:2",
		ProjectRef: "project:p", WorkstreamRef: "workstream:w", WorksetRef: "workset:s",
		CallGraphRef: "callgraph:c", WorkItemRef: "work-item:i",
	})

	for name, scope := range map[string]struct {
		WorkpointRef  string
		ContinuityRef string
	}{
		"by id":  {byID.WorkpointRef, byID.ContinuityRef},
		"by ref": {byRef.WorkpointRef, byRef.ContinuityRef},
	} {
		if scope.WorkpointRef != "workpoint:1" {
			t.Fatalf("%s: workpoint ref = %q, want workpoint:1", name, scope.WorkpointRef)
		}
		if scope.ContinuityRef != "continuity:2" {
			t.Fatalf("%s: continuity ref = %q, want continuity:2", name, scope.ContinuityRef)
		}
	}
}
