package evidenceartifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCaptureDigestThirtyRunsIdentical proves the committed canonical report
// regenerates byte-for-byte from the current code (drift guard, CG-05 atom).
func TestCaptureDigestThirtyRunsIdentical(t *testing.T) {
	report, err := RunCaptureDigest(30, "cg05-canonical")
	if err != nil {
		t.Fatalf("run digest: %v", err)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	path := filepath.Join("testdata", "capture-digest", "digest-report.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read canonical report: %v", err)
	}
	var canonical CaptureDigestReport
	if err := json.Unmarshal(body, &canonical); err != nil {
		t.Fatalf("decode canonical report: %v", err)
	}
	if err := canonical.Validate(); err != nil {
		t.Fatalf("canonical invalid: %v", err)
	}
	if report.DigestSHA256 != canonical.DigestSHA256 {
		t.Fatalf("digest drift: fresh=%s canonical=%s", report.DigestSHA256, canonical.DigestSHA256)
	}
	if canonical.Runs != 30 || !canonical.AllIdentical {
		t.Fatalf("canonical report shape: runs=%d all_identical=%v", canonical.Runs, canonical.AllIdentical)
	}
}

// TestCaptureDigestVerifiesAgainstCanonical exercises the verify-against path
// with a different run count (the sealed manifest is run-count independent).
func TestCaptureDigestVerifiesAgainstCanonical(t *testing.T) {
	path := filepath.Join("testdata", "capture-digest", "digest-report.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read canonical report: %v", err)
	}
	var canonical CaptureDigestReport
	if err := json.Unmarshal(body, &canonical); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := VerifyCaptureDigestReport(canonical, 7); err != nil {
		t.Fatalf("verify with alternate runs: %v", err)
	}
}

// TestCaptureDigestScenarioCoversJoinDimensions pins the scenario shape so the
// digest cannot silently narrow: static visual + temporal media + diagnostics +
// structured requirements, two viewports, typed omission, bound annotation,
// contiguous ordinals, complete window.
func TestCaptureDigestScenarioCoversJoinDimensions(t *testing.T) {
	scenario, err := CaptureDigestScenario()
	if err != nil {
		t.Fatalf("scenario: %v", err)
	}
	if len(scenario.Requirements) != 4 {
		t.Fatalf("requirements: %d", len(scenario.Requirements))
	}
	if scenario.WindowComplete != true {
		t.Fatal("window must be complete")
	}
	if len(scenario.Omissions) != 1 || scenario.Omissions[0].ObservationRef != "observation:5" {
		t.Fatalf("omissions: %+v", scenario.Omissions)
	}
	if len(scenario.Annotations) != 1 {
		t.Fatalf("annotations: %d", len(scenario.Annotations))
	}
	manifest, err := AssembleCapture(scenario)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if err := VerifyManifestSHA256(manifest); err != nil {
		t.Fatalf("integrity: %v", err)
	}
}
