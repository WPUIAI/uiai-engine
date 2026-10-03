package routes

import (
	"encoding/json"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestScreenshotScopeUsesCompleteTypedEvidenceReferences(t *testing.T) {
	expected := completeEvidenceScope()
	payload, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var scope vision.FocusaScope
	if err := json.Unmarshal(payload, &scope); err != nil {
		t.Fatal(err)
	}
	if status := routeFocusaScopeStatus(&scope); status != "present" {
		t.Fatalf("complete typed scope marked %q: %#v", status, scope)
	}
	if project := scopeProject(&scope); project != expected.ProjectRef {
		t.Fatalf("settings project ref=%q, want %q", project, expected.ProjectRef)
	}
	if workstream := scopeWorkstream(&scope); workstream != expected.WorkstreamRef {
		t.Fatalf("settings workstream ref=%q, want %q", workstream, expected.WorkstreamRef)
	}
	metadata := screenshotFocusaMetadata("https://example.test", "artifact:test", "png", 1, &scope)
	if metadata["focusa_scope_status"] != "present" {
		t.Fatalf("screenshot metadata lost complete scope: %#v", metadata)
	}
}
