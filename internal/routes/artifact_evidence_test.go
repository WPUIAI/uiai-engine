package routes

import (
	"strings"
	"testing"
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
	ev := shareEvidence("abc", "https://example.test", "Example")
	if ev["evidence_ref"] != "uiai-share:abc" {
		t.Fatalf("unexpected evidence ref: %#v", ev)
	}
	if ev["artifact_ref"] != "/api/share/abc" {
		t.Fatalf("unexpected artifact ref: %#v", ev)
	}
}
