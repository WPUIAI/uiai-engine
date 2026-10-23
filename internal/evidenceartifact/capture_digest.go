package evidenceartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CaptureDigestSchema is the schema constant for the 30-run capture assembly
// digest report (CG-05 evidence atom).
const CaptureDigestSchema = "uiai.capture_digest_report.v1"

// CaptureDigestReport is the deterministic evidence artifact binding a fixed
// multi-modal capture assembly scenario to N assembly runs and their
// byte-identical sealed manifests. It carries no timestamps so it can be
// committed and verified byte-for-byte by an independent reviewer.
type CaptureDigestReport struct {
	Schema string `json:"schema"`

	CodeRef string `json:"code_ref"`
	Runs    int    `json:"runs"`

	// Scenario summary of the assembled manifest (deterministic labels only).
	AssetClasses     []string `json:"asset_classes"`
	RequirementKinds []string `json:"requirement_kinds"`
	ViewportRefs     []string `json:"viewport_refs"`
	OmissionCount    int      `json:"omission_count"`
	AnnotationCount  int      `json:"annotation_count"`
	ObservationCount int      `json:"observation_count"`

	// DigestSHA256 is the sealed manifest hash shared by every run.
	DigestSHA256 string `json:"digest_sha256"`
	AllIdentical bool   `json:"all_identical"`

	// PerRun hashes (all identical by construction; kept so an independent
	// reviewer can verify run-by-run).
	PerRun []string `json:"per_run"`
}

// CaptureDigestScenario builds the deterministic multi-modal assembly request
// used for the digest: static visual + temporal media + diagnostic + structured
// claims, two viewports, one typed omission policy, one annotation with bound
// geometry, contiguous observation ordinals, and a complete capture window.
func CaptureDigestScenario() (CaptureAssembly, error) {
	base := ReferenceEvidenceManifest()
	base.Schema = SchemaManifestV1
	base.ArtifactID = "artifact:cg05-digest"
	base.Title = "CG05 digest capture"
	base.Claims = []Claim{
		{ClaimID: "claim:static-visual", Summary: "Responsive page rendering at 390x844 and 1440x900", Status: ClaimActual, AcceptanceAtomRefs: []string{"atom:static-visual"}, EvidenceRefs: []string{"asset:proof-mobile", "asset:proof-wide"}, ReviewRequirementRefs: []string{"review-requirement:llm"}},
		{ClaimID: "claim:temporal-media", Summary: "Recorded interaction video with transcript", Status: ClaimActual, AcceptanceAtomRefs: []string{"atom:temporal-media"}, EvidenceRefs: []string{"asset:proof-video"}, ReviewRequirementRefs: []string{"review-requirement:llm"}},
		{ClaimID: "claim:diagnostics", Summary: "Diagnostics console log captured", Status: ClaimActual, AcceptanceAtomRefs: []string{"atom:diagnostics"}, EvidenceRefs: []string{"asset:proof-diagnostics"}, ReviewRequirementRefs: []string{"review-requirement:llm"}},
		{ClaimID: "claim:structured", Summary: "Network summary exported", Status: ClaimActual, AcceptanceAtomRefs: []string{"atom:structured"}, EvidenceRefs: []string{"asset:proof-network"}, ReviewRequirementRefs: []string{"review-requirement:llm"}},
	}
	base.Assets = []Asset{
		{AssetID: "asset:proof-mobile", Kind: "screenshot", MediaType: "image/png", Path: "assets/proof-mobile.png", SHA256: "1111111111111111111111111111111111111111111111111111111111111111", ByteSize: 2048, CapturedAt: "2026-09-10T00:00:00Z", SourceRef: "observation:0", ClaimRefs: []string{"claim:static-visual"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe, Width: 390, Height: 844, AltText: "Responsive page mobile"},
		{AssetID: "asset:proof-wide", Kind: "screenshot", MediaType: "image/png", Path: "assets/proof-wide.png", SHA256: "2222222222222222222222222222222222222222222222222222222222222222", ByteSize: 4096, CapturedAt: "2026-09-10T00:00:00Z", SourceRef: "observation:1", ClaimRefs: []string{"claim:static-visual"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe, Width: 1440, Height: 900, AltText: "Responsive page desktop"},
		{AssetID: "asset:proof-video", Kind: "video", MediaType: "video/mp4", Path: "assets/proof-video.mp4", SHA256: "3333333333333333333333333333333333333333333333333333333333333333", ByteSize: 8192, CapturedAt: "2026-09-10T00:00:00Z", SourceRef: "observation:2", ClaimRefs: []string{"claim:temporal-media"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe, DurationMS: 1200, TranscriptRef: "transcript:video"},
		{AssetID: "asset:proof-diagnostics", Kind: "diagnostic", MediaType: "application/json", Path: "assets/proof-diagnostics.json", SHA256: "4444444444444444444444444444444444444444444444444444444444444444", ByteSize: 512, CapturedAt: "2026-09-10T00:00:00Z", SourceRef: "observation:3", ClaimRefs: []string{"claim:diagnostics"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe},
		{AssetID: "asset:proof-network", Kind: "structured_data", MediaType: "text/csv", Path: "assets/proof-network.csv", SHA256: "5555555555555555555555555555555555555555555555555555555555555555", ByteSize: 256, CapturedAt: "2026-09-10T00:00:00Z", SourceRef: "observation:4", ClaimRefs: []string{"claim:structured"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe},
	}
	sealed, err := Seal(base)
	if err != nil {
		return CaptureAssembly{}, fmt.Errorf("seal digest base manifest: %w", err)
	}
	viewportRefs := []string{"viewport:390x844", "viewport:1440x900"}
	return CaptureAssembly{
		Base: sealed,
		CaptureMetadata: CaptureMetadata{
			RunRef:         "run:cg05-digest",
			RuntimeRef:     "runtime:uiai",
			EnvironmentRef: "environment:reference",
			TargetRef:      "target:page",
			AccountRef:     "account:reference",
			WindowComplete: true,
			Viewports: []CaptureViewport{
				{ViewportRef: "viewport:390x844", Width: 390, Height: 844, DPR: 3},
				{ViewportRef: "viewport:1440x900", Width: 1440, Height: 900, DPR: 1},
			},
			Observations: []CaptureObservation{
				{ObservationRef: "observation:0", Ordinal: 0, OccurredAt: "2026-09-10T00:00:00Z", ActionRef: "action:screenshot-mobile", ReceiptRef: "receipt:mobile", ViewportRef: "viewport:390x844", AssetID: "asset:proof-mobile"},
				{ObservationRef: "observation:1", Ordinal: 1, OccurredAt: "2026-09-10T00:00:01Z", ActionRef: "action:screenshot-wide", ReceiptRef: "receipt:wide", ViewportRef: "viewport:1440x900", AssetID: "asset:proof-wide"},
				{ObservationRef: "observation:2", Ordinal: 2, OccurredAt: "2026-09-10T00:00:02Z", ActionRef: "action:record-interaction", ReceiptRef: "receipt:video", AssetID: "asset:proof-video"},
				{ObservationRef: "observation:3", Ordinal: 3, OccurredAt: "2026-09-10T00:00:03Z", ActionRef: "action:capture-diagnostics", ReceiptRef: "receipt:diagnostics", AssetID: "asset:proof-diagnostics"},
				{ObservationRef: "observation:4", Ordinal: 4, OccurredAt: "2026-09-10T00:00:04Z", ActionRef: "action:export-network", ReceiptRef: "receipt:network", AssetID: "asset:proof-network"},
				{ObservationRef: "observation:5", Ordinal: 5, OccurredAt: "2026-09-10T00:00:05Z", ActionRef: "action:record-audio", ReceiptRef: "receipt:audio"},
			},
			Requirements: []ClaimRequirement{
				{ClaimID: "claim:static-visual", Kind: RequirementStaticVisual, ActualRequired: true, ViewportRefs: viewportRefs},
				{ClaimID: "claim:temporal-media", Kind: RequirementTemporalInteraction, ActualRequired: true},
				{ClaimID: "claim:diagnostics", Kind: RequirementDiagnostic, ActualRequired: true},
				{ClaimID: "claim:structured", Kind: RequirementStructured, ActualRequired: true},
			},
			Omissions: []CaptureOmission{
				{ObservationRef: "observation:5", Reason: "audio track unavailable in headless runner", PolicyRef: "uiai.capture_omission.media_unavailable.v1"},
			},
			Annotations: []CaptureAnnotation{
				{
					AnnotationRef: "annotation:mobile-cta", ObservationRef: "observation:0",
					SourceAssetID: "asset:proof-mobile", SourceAssetSHA256: "1111111111111111111111111111111111111111111111111111111111111111",
					SourceRevision: 1, AuthorRef: "author:reviewer", CreatedAt: "2026-09-10T00:00:05Z",
					OverlaySHA256: "6666666666666666666666666666666666666666666666666666666666666666",
					Label:         "Primary call-to-action", Geometry: AnnotationGeometry{CoordinateSpace: "source_pixels", X: 24, Y: 700, Width: 342, Height: 48, SourceWidth: 390, SourceHeight: 844},
				},
			},
		},
	}, nil
}

// RunCaptureDigest executes runs sequential AssembleCapture runs over the
// scenario, hashes each sealed manifest canonically, and emits the digest
// report. It is deterministic: identical inputs always produce an identical
// report.
func RunCaptureDigest(runs int, codeRef string) (CaptureDigestReport, error) {
	if runs < 1 || runs > 1000 {
		return CaptureDigestReport{}, fmt.Errorf("runs must be 1..1000: %w", ErrAssemblyInvalid)
	}
	scenario, err := CaptureDigestScenario()
	if err != nil {
		return CaptureDigestReport{}, err
	}
	perRun := make([]string, 0, runs)
	for i := 0; i < runs; i++ {
		request := scenario
		manifest, err := AssembleCapture(request)
		if err != nil {
			return CaptureDigestReport{}, fmt.Errorf("assembly run %d: %w", i+1, err)
		}
		if err := VerifyManifestSHA256(manifest); err != nil {
			return CaptureDigestReport{}, fmt.Errorf("integrity run %d: %w", i+1, err)
		}
		canonical, err := CanonicalBytes(manifest)
		if err != nil {
			return CaptureDigestReport{}, fmt.Errorf("canonical run %d: %w", i+1, err)
		}
		sum := sha256.Sum256(canonical)
		perRun = append(perRun, hex.EncodeToString(sum[:]))
	}
	for _, h := range perRun {
		if h != perRun[0] {
			return CaptureDigestReport{}, fmt.Errorf("run digest divergence: %w", ErrAssemblyInvalid)
		}
	}
	// Bind the digest to the sealed manifest hash of the scenario run.
	final, err := AssembleCapture(scenario)
	if err != nil {
		return CaptureDigestReport{}, err
	}
	if final.Integrity.ManifestSHA256 == "" {
		return CaptureDigestReport{}, fmt.Errorf("final manifest unsealed: %w", ErrInvalidIntegrity)
	}
	return CaptureDigestReport{
		Schema:           CaptureDigestSchema,
		CodeRef:          codeRef,
		Runs:             runs,
		AssetClasses:     []string{"image/png", "image/png", "video/mp4", "application/json", "text/csv"},
		RequirementKinds: []string{string(RequirementStaticVisual), string(RequirementTemporalInteraction), string(RequirementDiagnostic), string(RequirementStructured)},
		ViewportRefs:     []string{"viewport:390x844", "viewport:1440x900"},
		OmissionCount:    1,
		AnnotationCount:  1,
		ObservationCount: 6,
		DigestSHA256:     final.Integrity.ManifestSHA256,
		AllIdentical:     true,
		PerRun:           perRun,
	}, nil
}

// Validate checks report internal consistency for independent verification.
func (r CaptureDigestReport) Validate() error {
	if r.Schema != CaptureDigestSchema {
		return fmt.Errorf("schema: %w", ErrInvalidIntegrity)
	}
	if r.Runs < 1 || len(r.PerRun) != r.Runs {
		return fmt.Errorf("runs/per_run mismatch: %w", ErrInvalidIntegrity)
	}
	if !r.AllIdentical {
		return fmt.Errorf("all_identical false: %w", ErrAssemblyInvalid)
	}
	for _, h := range r.PerRun {
		if h != r.PerRun[0] {
			return fmt.Errorf("divergent per-run hash: %w", ErrAssemblyInvalid)
		}
	}
	if r.PerRun[0] != r.DigestSHA256 {
		return fmt.Errorf("digest binding: %w", ErrInvalidIntegrity)
	}
	return nil
}

// VerifyCaptureDigestReport re-runs the digest and compares the digest hash
// against a prior report (used by the CLI --verify-against and drift guard).
func VerifyCaptureDigestReport(prior CaptureDigestReport, runs int) error {
	if err := prior.Validate(); err != nil {
		return err
	}
	if runs == 0 {
		runs = prior.Runs
	}
	fresh, err := RunCaptureDigest(runs, prior.CodeRef)
	if err != nil {
		return err
	}
	if fresh.DigestSHA256 != prior.DigestSHA256 {
		return fmt.Errorf("digest drift: %w", ErrInvalidIntegrity)
	}
	return nil
}
