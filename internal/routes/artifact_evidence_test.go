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
	ev := shareEvidence("abc", "https://example.test?token=secret#frag", "Example", nil)
	if ev["evidence_ref"] != "uiai-share:abc" {
		t.Fatalf("unexpected evidence ref: %#v", ev)
	}
	if ev["artifact_ref"] != nil || ev["operational_share_ref"] != "/api/share/abc" || ev["delivery_posture"] != "ephemeral_non_evidence" {
		t.Fatalf("operational share was mislabeled as artifact delivery: %#v", ev)
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

func TestScreenshotFocusaMetadata(t *testing.T) {
	ev := screenshotFocusaMetadata("https://example.test/page?token=secret#frag", "uiai-screenshot:sha256:abc", "png", 123, nil)
	if ev["evidence_ref"] != "uiai-screenshot:sha256:abc" || ev["preferred_tool"] != "focusa_evidence_capture" {
		t.Fatalf("unexpected screenshot metadata: %#v", ev)
	}
	if strings.Contains(ev["target_ref"].(string), "secret") || strings.Contains(ev["target_ref"].(string), "#frag") {
		t.Fatalf("target_ref leaked secret/fragment: %#v", ev)
	}
	if _, ok := ev["next_tools"].([]string); !ok {
		t.Fatalf("missing next_tools: %#v", ev)
	}
}
