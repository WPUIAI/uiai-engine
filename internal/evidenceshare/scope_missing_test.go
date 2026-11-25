package evidenceshare

import (
	"reflect"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

// #236: blocked deliveries must name exactly which scope bindings are
// missing so callers can repair the envelope without guessing.
func TestMissingRequirements(t *testing.T) {
	workItem := func(ref string) evidencepwa.WorkItemProjection {
		return evidencepwa.WorkItemProjection{WorkItemRef: ref}
	}

	cases := []struct {
		name string
		sc   Scope
		want []string
	}{
		{
			name: "empty scope names all bindings",
			sc:   Scope{},
			want: []string{
				"project_ref", "workstream_ref", "workset_ref", "callgraph_ref",
				"workpoint_ref", "work_item_ref", "continuity_ref", "work_items",
			},
		},
		{
			name: "partial scope names only the gaps",
			sc: Scope{
				ProjectRef: "proj-1", WorkpointRef: "wp-1",
				WorkItems: []evidencepwa.WorkItemProjection{workItem("wi-1")},
			},
			want: []string{
				"workstream_ref", "workset_ref", "callgraph_ref",
				"work_item_ref", "continuity_ref", "work_items_matching_work_item_ref",
			},
		},
		{
			name: "work items present but no match for work item ref",
			sc: Scope{
				ProjectRef: "p", WorkstreamRef: "w", WorksetRef: "ws", CallGraphRef: "cg",
				WorkpointRef: "wp", WorkItemRef: "wi-1", ContinuityRef: "c",
				WorkItems: []evidencepwa.WorkItemProjection{workItem("wi-other")},
			},
			want: []string{"work_items_matching_work_item_ref"},
		},
		{
			name: "complete scope names nothing",
			sc: Scope{
				ProjectRef: "p", WorkstreamRef: "w", WorksetRef: "ws", CallGraphRef: "cg",
				WorkpointRef: "wp", WorkItemRef: "wi-1", ContinuityRef: "c",
				WorkItems: []evidencepwa.WorkItemProjection{workItem("wi-1")},
			},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sc.MissingRequirements(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
