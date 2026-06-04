package routes

import (
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestScreenshotEvidenceRefStable(t *testing.T) {
	ref1 := screenshotEvidenceRef([]byte("image-bytes"))
	ref2 := screenshotEvidenceRef([]byte("image-bytes"))
	if ref1 != ref2 {
		t.Fatalf("refs differ: %s %s", ref1, ref2)
	}
	if !strings.HasPrefix(ref1, "uiai-screenshot:sha256:") {
		t.Fatalf("unexpected ref: %s", ref1)
	}
}

func TestShareEvidence(t *testing.T) {
	ev := shareEvidence("abc", "https://example.test", "Example", nil)
	if ev["evidence_ref"] != "uiai-share:abc" {
		t.Fatalf("unexpected evidence ref: %#v", ev)
	}
	if ev["artifact_ref"] != "/api/share/abc" {
		t.Fatalf("unexpected artifact ref: %#v", ev)
	}
}

func TestShareEvidenceIncludesFocusaScope(t *testing.T) {
	scope := &vision.FocusaScope{WorkpointID: "wp1", ContinuityID: "cont1", ProjectRoot: "/tmp/project", EvidenceRef: "ev1"}
	ev := shareEvidence("abc", "https://example.test", "Example", scope)
	got, ok := ev["focusa_scope"].(*vision.FocusaScope)
	if !ok || got.WorkpointID != "wp1" || got.ContinuityID != "cont1" || got.ProjectRoot != "/tmp/project" || got.EvidenceRef != "ev1" {
		t.Fatalf("missing focusa scope in share evidence: %#v", ev)
	}
}
