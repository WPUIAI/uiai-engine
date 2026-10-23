package evidenceartifact

import (
	"errors"
	"reflect"
	"testing"
)

func TestAssembleCaptureStaticVisual(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	manifest, err := AssembleCapture(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifestSHA256(manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Capture == nil || manifest.Capture.Viewports[0].DPR != 3 {
		t.Fatalf("capture metadata=%#v", manifest.Capture)
	}
	if len(manifest.Provenance.Custody) != 1 || manifest.Provenance.Custody[0].EventID != "observation:0" {
		t.Fatalf("custody=%#v", manifest.Provenance.Custody)
	}
	for _, ref := range []string{request.RunRef, request.RuntimeRef, request.TargetRef, request.AccountRef} {
		if _, ok := stringSet(manifest.Links.RelatedRefs)[ref]; !ok {
			t.Fatalf("missing related ref %s", ref)
		}
	}
}

func TestAssembleCaptureOrderingIsDeterministicAndInputImmutable(t *testing.T) {
	first := twoViewportFixture(t)
	second := twoViewportFixture(t)
	second.Observations[0], second.Observations[1] = second.Observations[1], second.Observations[0]
	second.Viewports[0], second.Viewports[1] = second.Viewports[1], second.Viewports[0]
	original := append([]CaptureObservation(nil), second.Observations...)
	left, err := AssembleCapture(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := AssembleCapture(second)
	if err != nil {
		t.Fatal(err)
	}
	if left.Integrity.ManifestSHA256 != right.Integrity.ManifestSHA256 {
		t.Fatalf("hashes differ: %s %s", left.Integrity.ManifestSHA256, right.Integrity.ManifestSHA256)
	}
	if !reflect.DeepEqual(second.Observations, original) {
		t.Fatal("assembler mutated input")
	}
	changed := twoViewportFixture(t)
	changed.Viewports[0].DPR = 2
	other, err := AssembleCapture(changed)
	if err != nil {
		t.Fatal(err)
	}
	if other.Integrity.ManifestSHA256 == left.Integrity.ManifestSHA256 {
		t.Fatal("viewport DPR was not integrity-bound")
	}
}

func TestAssembleCaptureSequenceAndRuntimeFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CaptureAssembly)
		want   error
	}{
		{"gap", func(r *CaptureAssembly) { r.Observations[0].Ordinal = 2 }, ErrObservationGap},
		{"duplicate", func(r *CaptureAssembly) { r.Observations = append(r.Observations, r.Observations[0]) }, ErrObservationGap},
		{"bad_time", func(r *CaptureAssembly) { r.Observations[0].OccurredAt = "yesterday" }, ErrObservationGap},
		{"runtime", func(r *CaptureAssembly) { r.RuntimeRef = "" }, ErrRuntimeAmbiguous},
		{"target", func(r *CaptureAssembly) { r.TargetRef = "" }, ErrRuntimeAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := assemblyFixture(t, RequirementStaticVisual)
			tt.mutate(&request)
			if _, err := AssembleCapture(request); !errors.Is(err, tt.want) {
				t.Fatalf("err=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestAssembleCaptureCoverageAndClassFailures(t *testing.T) {
	t.Run("missing_viewport", func(t *testing.T) {
		request := assemblyFixture(t, RequirementStaticVisual)
		request.Requirements[0].ViewportRefs = append(request.Requirements[0].ViewportRefs, "viewport:missing")
		if _, err := AssembleCapture(request); !errors.Is(err, ErrCoverageIncomplete) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("surrogate", func(t *testing.T) {
		request := assemblyFixture(t, RequirementStaticVisual)
		request.Base.Assets[0].VerificationClass = VerificationSurrogate
		request.Base = reseal(t, request.Base)
		if _, err := AssembleCapture(request); !errors.Is(err, ErrEvidenceClassMismatch) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("missing_required_claim", func(t *testing.T) {
		request := assemblyFixture(t, RequirementStaticVisual)
		request.Requirements[0].ClaimID = "claim:missing"
		if _, err := AssembleCapture(request); !errors.Is(err, ErrCoverageIncomplete) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestAssembleCaptureTemporalAndProofOfAbsence(t *testing.T) {
	temporal := assemblyFixture(t, RequirementTemporalInteraction)
	if _, err := AssembleCapture(temporal); err != nil {
		t.Fatal(err)
	}
	absence := assemblyFixture(t, RequirementProofOfAbsence)
	absence.WindowComplete = false
	if _, err := AssembleCapture(absence); !errors.Is(err, ErrCoverageIncomplete) {
		t.Fatalf("err=%v", err)
	}
	absence.WindowComplete = true
	if _, err := AssembleCapture(absence); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleCaptureOmissionPartition(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	request.Omissions = []CaptureOmission{{ObservationRef: request.Observations[0].ObservationRef, Reason: "operator excluded", PolicyRef: "policy:omission"}}
	if _, err := AssembleCapture(request); !errors.Is(err, ErrOmissionConflict) {
		t.Fatalf("selected and omitted err=%v", err)
	}
	request.Observations[0].AssetID = ""
	if _, err := AssembleCapture(request); !errors.Is(err, ErrCoverageIncomplete) {
		t.Fatalf("required omission err=%v", err)
	}
}

func assemblyFixture(t *testing.T, kind RequirementKind) CaptureAssembly {
	t.Helper()
	base := testManifest()
	asset := &base.Assets[0]
	asset.SourceRef = "observation:0"
	asset.ClaimRefs = []string{base.Claims[0].ClaimID}
	base.Claims[0].EvidenceRefs = []string{asset.AssetID}
	viewportRef := ""
	switch kind {
	case RequirementStaticVisual:
		asset.MediaType = "image/png"
		asset.Width, asset.Height, asset.AltText = 390, 844, "Responsive page"
		viewportRef = "viewport:390x844"
	case RequirementTemporalInteraction:
		asset.MediaType = "video/mp4"
		asset.DurationMS, asset.TranscriptRef = 1200, "transcript:video"
	case RequirementDiagnostic, RequirementProofOfAbsence:
		asset.MediaType = "application/json"
	case RequirementStructured:
		asset.MediaType = "text/csv"
	}
	base = reseal(t, base)
	viewports := []CaptureViewport{}
	viewportRefs := []string{}
	if viewportRef != "" {
		viewports = append(viewports, CaptureViewport{ViewportRef: viewportRef, Width: 390, Height: 844, DPR: 3})
		viewportRefs = append(viewportRefs, viewportRef)
	}
	return CaptureAssembly{
		Base: base,
		CaptureMetadata: CaptureMetadata{
			RunRef: "run:capture", RuntimeRef: "runtime:uiai", EnvironmentRef: "environment:ovh",
			TargetRef: "target:page", AccountRef: "account:test", WindowComplete: true,
			Viewports:    viewports,
			Observations: []CaptureObservation{{ObservationRef: "observation:0", Ordinal: 0, OccurredAt: "2026-08-29T12:00:00Z", ActionRef: "action:capture", ReceiptRef: "receipt:capture", ViewportRef: viewportRef, AssetID: asset.AssetID}},
			Requirements: []ClaimRequirement{{ClaimID: base.Claims[0].ClaimID, Kind: kind, ActualRequired: true, ViewportRefs: viewportRefs}},
			Omissions:    []CaptureOmission{},
		},
	}
}

func twoViewportFixture(t *testing.T) CaptureAssembly {
	t.Helper()
	request := assemblyFixture(t, RequirementStaticVisual)
	second := request.Base.Assets[0]
	second.AssetID = "asset:proof-wide"
	second.Path = "assets/proof-wide.png"
	second.SourceRef = "observation:1"
	second.Width, second.Height = 1440, 900
	request.Base.Assets = append(request.Base.Assets, second)
	request.Base.Claims[0].EvidenceRefs = append(request.Base.Claims[0].EvidenceRefs, second.AssetID)
	request.Base = reseal(t, request.Base)
	request.Viewports = append(request.Viewports, CaptureViewport{ViewportRef: "viewport:1440x900", Width: 1440, Height: 900, DPR: 1})
	request.Observations = append(request.Observations, CaptureObservation{ObservationRef: "observation:1", Ordinal: 1, OccurredAt: "2026-08-29T12:00:01Z", ActionRef: "action:capture", ReceiptRef: "receipt:capture-wide", ViewportRef: "viewport:1440x900", AssetID: second.AssetID})
	request.Requirements[0].ViewportRefs = append(request.Requirements[0].ViewportRefs, "viewport:1440x900")
	return request
}

func reseal(t *testing.T, manifest Manifest) Manifest {
	t.Helper()
	manifest.Integrity.ManifestSHA256 = ""
	sealed, err := Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}
