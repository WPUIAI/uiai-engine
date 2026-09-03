package evidencederivative

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const da = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const db = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestGoldenDeterminismAndValidation(t *testing.T) {
	r, m, d := contracts()
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	if err := ValidateManifest(m, r); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDelivery(d); err != nil {
		t.Fatal(err)
	}
	gold(t, "derivative-manifest.golden.json", m)
	gold(t, "delivery-receipt.golden.json", d)
	a, _ := DigestManifest(m)
	for i := 0; i < 30; i++ {
		b, _ := DigestManifest(m)
		if a != b {
			t.Fatal("digest drift")
		}
	}
}
func TestIdentityBindsTruthInputs(t *testing.T) {
	for name, mutate := range map[string]func(*DerivativeRequest, *DerivativeManifest){
		"source":    func(r *DerivativeRequest, _ *DerivativeManifest) { r.ArtifactSHA256 = db },
		"selection": func(r *DerivativeRequest, _ *DerivativeManifest) { r.ClaimRefs = append(r.ClaimRefs, "claim:2") },
		"profile":   func(r *DerivativeRequest, _ *DerivativeManifest) { r.Rendering.ProfileSHA256 = da },
		"renderer":  func(_ *DerivativeRequest, m *DerivativeManifest) { m.Renderer.Version = "2.0.0" },
		"license":   func(_ *DerivativeRequest, m *DerivativeManifest) { m.Licenses[0].LicenseSHA256 = db },
	} {
		r, m, _ := contracts()
		base := m.DerivativeID
		mutate(&r, &m)
		changed, _ := DerivativeID(r, m.Renderer, m.ViewerMatrix, m.Licenses)
		if changed == base {
			t.Fatalf("%s did not change identity", name)
		}
	}
}
func TestSelectionArchiveLicenseAndDeliveryFailClosed(t *testing.T) {
	r, m, d := contracts()
	r.RequiredEvidenceRefs = append(r.RequiredEvidenceRefs, "asset:missing")
	if !errors.Is(ValidateRequest(r), ErrDerivativeSelectionIncomplete) {
		t.Fatal("missing evidence accepted")
	}
	r, m, _ = contracts()
	m.ArchivePosture = ArchiveSafe
	m.ArchiveEntries = []ArchiveEntry{{Path: "../escape", SHA256: da, MIME: "text/plain", Bytes: 1}}
	if !errors.Is(ValidateManifest(m, r), ErrDerivativeUnsafeArchive) {
		t.Fatal("unsafe archive accepted")
	}
	r, m, _ = contracts()
	m.Licenses = nil
	if !errors.Is(ValidateManifest(m, r), ErrDerivativeLicenseMissing) {
		t.Fatal("missing license accepted")
	}
	d.State = DeliveryAccepted
	d.DeliveredAt = d.AcceptedAt
	if !errors.Is(ValidateDelivery(d), ErrDerivativeContractInvalid) {
		t.Fatal("acceptance serialized as delivery")
	}
	_, _, d = contracts()
	d.State = DeliveryOutcomeUnknown
	d.DeliveredAt = nil
	d.AcceptedAt = nil
	d.RetryPermitted = true
	if !errors.Is(ValidateDelivery(d), ErrDeliveryReconciliationRequired) {
		t.Fatal("ambiguous retry accepted")
	}
}
func TestPDFConformanceVocabularyAndTargetMatch(t *testing.T) {
	for name, mutate := range map[string]func(*DerivativeRequest, *DerivativeManifest){
		"unknown PDF/UA profile":   func(_ *DerivativeRequest, manifest *DerivativeManifest) { manifest.PDFUAProfile = "PDF/UA-future" },
		"mismatched PDF/UA target": func(_ *DerivativeRequest, manifest *DerivativeManifest) { manifest.PDFUAProfile = PDFUAProfile2 },
		"unknown PDF/A profile":    func(_ *DerivativeRequest, manifest *DerivativeManifest) { manifest.PDFAProfile = "PDF/A-9z" },
		"unclaimed profile": func(_ *DerivativeRequest, manifest *DerivativeManifest) {
			manifest.AccessibilityPosture = ConformanceNotClaimed
		},
	} {
		t.Run(name, func(t *testing.T) {
			request, manifest, _ := contracts()
			mutate(&request, &manifest)
			if !errors.Is(ValidateManifest(manifest, request), ErrDerivativeContractInvalid) {
				t.Fatalf("invalid conformance accepted: %#v", manifest)
			}
		})
	}
	request, manifest, _ := contracts()
	request.AccessibilityTarget = AccessibilityPDFUA2
	manifest.AccessibilityTarget = AccessibilityPDFUA2
	manifest.PDFUAProfile = PDFUAProfile2
	requestDigest, _ := DigestRequest(request)
	manifest.RequestSHA256 = requestDigest
	manifest.DerivativeID, _ = DerivativeID(request, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses)
	if err := ValidateManifest(manifest, request); err != nil {
		t.Fatalf("matching PDF/UA-2 rejected: %v", err)
	}
}

func TestUnknownVocabularyAndScope(t *testing.T) {
	r, m, d := contracts()
	r.DerivativeType = "video"
	if !errors.Is(ValidateRequest(r), ErrDerivativeContractInvalid) {
		t.Fatal("unknown type accepted")
	}
	r, m, _ = contracts()
	m.ArtifactRef = "artifact:other"
	if !errors.Is(ValidateManifest(m, r), ErrDerivativeScopeMismatch) {
		t.Fatal("scope mismatch accepted")
	}
	r, _, _ = contracts()
	r.Rendering.DependencyRefs = []string{"https://cdn.example/font.woff2"}
	if !errors.Is(ValidateRequest(r), ErrDerivativeContractInvalid) {
		t.Fatal("external dependency accepted")
	}
	r, _, _ = contracts()
	r.LicensePolicyRef = "/home/operator/license.json"
	if !errors.Is(ValidateRequest(r), ErrDerivativeContractInvalid) {
		t.Fatal("private path accepted")
	}
	_, _, d = contracts()
	d.State = "sent"
	if !errors.Is(ValidateDelivery(d), ErrDerivativeContractInvalid) {
		t.Fatal("unknown delivery state accepted")
	}
}

func contracts() (DerivativeRequest, DerivativeManifest, DeliveryReceipt) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	s := ScopeBinding{ProjectRef: "project:1", WorkstreamRef: "workstream:1", WorksetRef: "workset:1", CallGraphRef: "callgraph:1", WorkpointRef: "workpoint:1", WorkItemRef: "github:WPUIAI/uiai-engine#124"}
	r := DerivativeRequest{Schema: RequestSchema, RequestID: "request:1", Scope: s, ArtifactRef: "artifact:1", ArtifactSHA256: da, ArtifactRevision: 5, ProjectionRef: "projection:1", ProjectionSHA256: db, DerivativeType: DerivativePDF, ClaimRefs: []string{"claim:1"}, AssetRefs: []string{"asset:1"}, CitationRefs: []string{"citation:1"}, RequiredEvidenceRefs: []string{"claim:1", "asset:1", "citation:1"}, Rendering: RenderingProfile{ProfileRef: "profile:1", ProfileSHA256: db, FontRefs: []string{"font:1"}, ColorProfileRef: "srgb", DependencyRefs: []string{"./fonts/font.woff2"}}, Locale: "en-US", Direction: DirectionLTR, AccessibilityTarget: AccessibilityPDFUA1, LicensePolicyRef: "license-policy:1", LicensePolicySHA256: da, IdempotencyKey: "idempotency:1"}
	ren := RendererIdentity{RendererRef: "renderer:1", Version: "1.0.0", BinarySHA256: da}
	v := ViewerMatrix{Schema: ViewerMatrixSchema, MatrixRef: "viewers:1", Entries: []ViewerEntry{{Client: "Acrobat", Version: "2026", Status: ViewerSupported, EvidenceRefs: []string{"evidence:viewer"}}}}
	l := []LicenseAttestation{{Schema: LicenseSchema, AssetRef: "asset:1", LicenseRef: "license:1", LicenseSHA256: da, AttributionRequired: true, AttributionRef: "attribution:1", DerivativePermitted: true, EvidenceRef: "evidence:license"}}
	id, _ := DerivativeID(r, ren, v, l)
	rd, _ := DigestRequest(r)
	m := DerivativeManifest{Schema: ManifestSchema, DerivativeID: id, RequestRef: r.RequestID, RequestSHA256: rd, ArtifactRef: r.ArtifactRef, ArtifactSHA256: r.ArtifactSHA256, ProjectionRef: r.ProjectionRef, ProjectionSHA256: r.ProjectionSHA256, OutputRef: "./exports/proof.pdf", OutputSHA256: db, OutputBytes: 4096, OutputMIME: "application/pdf", Renderer: ren, Rendering: r.Rendering, AccessibilityTarget: r.AccessibilityTarget, AccessibilityPosture: ConformanceVerified, PDFUAProfile: "PDF/UA-1", AccessibilityEvidenceRefs: []string{"evidence:a11y"}, ArchivePosture: ArchiveNotApplicable, ViewerMatrix: v, Licenses: l, ReceiptRef: "receipt:render", CreatedAt: now}
	md, _ := DigestManifest(m)
	accepted := now.Add(time.Minute)
	delivered := now.Add(2 * time.Minute)
	d := DeliveryReceipt{Schema: DeliverySchema, DeliveryID: "delivery:1", DerivativeRef: m.DerivativeID, DerivativeSHA256: md, DestinationRef: "destination:1", IdempotencyKey: "delivery-key:1", State: DeliveryDelivered, ProviderReceiptRef: "provider:1", AcceptedAt: &accepted, DeliveredAt: &delivered, EvidenceRefs: []string{"evidence:delivery"}, ObservedAt: delivered}
	return r, m, d
}
func gold(t *testing.T, name string, v any) {
	t.Helper()
	b, _ := json.MarshalIndent(v, "", "  ")
	b = append(b, '\n')
	p := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	w, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, w) {
		t.Fatal("golden drift")
	}
}
