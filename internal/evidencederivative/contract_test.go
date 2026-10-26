package evidencederivative

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
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
func TestTranscriptAndKeyframeBindingsAreTypedAndIdentityBound(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	request.TranscriptRefs = []string{request.AssetRefs[0]}
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	if _, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:wrong-transcript", manifest.CreatedAt); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("static asset accepted as transcript: %v", err)
	}
	projection.Assets[0].Modality = AssetModalityTranscript
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	rendered, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:transcript", manifest.CreatedAt)
	if err != nil || !reflect.DeepEqual(rendered.Manifest.TranscriptRefs, request.TranscriptRefs) {
		t.Fatalf("typed transcript asset rejected or unbound: %#v %v", rendered.Manifest, err)
	}
	request.KeyframeRefs = append([]string(nil), request.TranscriptRefs...)
	if err := ValidateRequest(request); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("overlapping transcript/keyframe accepted: %v", err)
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

// T09-A independent conformance vectors: source-only, exact issue124 allowlist.
func TestT09AReceiptTimesAndDeliveredStateAreTruthful(t *testing.T) {
	for name, mutate := range map[string]func(*DeliveryReceipt){
		"future_acceptance": func(d *DeliveryReceipt) {
			d.State = DeliveryAccepted
			d.DeliveredAt = nil
			v := d.ObservedAt.Add(time.Second)
			d.AcceptedAt = &v
		},
		"zero_acceptance": func(d *DeliveryReceipt) {
			d.State = DeliveryAccepted
			d.DeliveredAt = nil
			v := time.Time{}
			d.AcceptedAt = &v
		},
		"queued_with_delivery":  func(d *DeliveryReceipt) { d.State = DeliveryQueued },
		"unknown_with_delivery": func(d *DeliveryReceipt) { d.State = DeliveryOutcomeUnknown },
	} {
		t.Run(name, func(t *testing.T) {
			_, _, d := contracts()
			mutate(&d)
			if !errors.Is(ValidateDelivery(d), ErrDerivativeContractInvalid) {
				t.Fatal("contradictory delivery receipt accepted")
			}
		})
	}
}

func TestT09AArchiveAliasesAndExecutableMIMEFailClosed(t *testing.T) {
	for name, entries := range map[string][]ArchiveEntry{
		"normalized_duplicate":        {{Path: "a/b.txt", SHA256: da, MIME: "text/plain", Bytes: 1}, {Path: "a/./b.txt", SHA256: da, MIME: "text/plain", Bytes: 1}},
		"encoded_duplicate":           {{Path: "a/b.txt", SHA256: da, MIME: "text/plain", Bytes: 1}, {Path: "a%2fb.txt", SHA256: da, MIME: "text/plain", Bytes: 1}},
		"case_insensitive_executable": {{Path: "payload", SHA256: da, MIME: "Application/X-Executable", Bytes: 1}},
		"windows_executable":          {{Path: "payload", SHA256: da, MIME: "application/x-msdownload", Bytes: 1}},
		"path_query":                  {{Path: "proof.txt?other=1", SHA256: da, MIME: "text/plain", Bytes: 1}},
		"path_fragment":               {{Path: "proof.txt#other", SHA256: da, MIME: "text/plain", Bytes: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			if !errors.Is(validateArchive(ArchiveSafe, entries), ErrDerivativeUnsafeArchive) {
				t.Fatal("unsafe archive metadata accepted")
			}
		})
	}
}

func TestT09ARequiredManifestOmissionsAndMIMEAreBound(t *testing.T) {
	r, m, _ := contracts()
	r.OmissionRefs = []string{"omission:explicit"}
	r.RequiredEvidenceRefs = append(r.RequiredEvidenceRefs, r.OmissionRefs...)
	m.RequestSHA256, _ = DigestRequest(r)
	m.DerivativeID, _ = DerivativeID(r, m.Renderer, m.ViewerMatrix, m.Licenses)
	if !errors.Is(ValidateManifest(m, r), ErrDerivativeScopeMismatch) {
		t.Fatal("required request omission disappeared from manifest")
	}
	r, m, _ = contracts()
	m.OutputMIME = "application/octet-stream"
	if !errors.Is(ValidateManifest(m, r), ErrDerivativeContractInvalid) {
		t.Fatal("PDF manifest accepted unrelated output MIME")
	}
}

func TestT09AValidatorsDoNotMutateAndEveryLineageFieldBinds(t *testing.T) {
	r, m, d := contracts()
	beforeR, _ := json.Marshal(r)
	beforeM, _ := json.Marshal(m)
	beforeD, _ := json.Marshal(d)
	_ = ValidateRequest(r)
	_ = ValidateManifest(m, r)
	_ = ValidateDelivery(d)
	afterR, _ := json.Marshal(r)
	afterM, _ := json.Marshal(m)
	afterD, _ := json.Marshal(d)
	if !bytes.Equal(beforeR, afterR) || !bytes.Equal(beforeM, afterM) || !bytes.Equal(beforeD, afterD) {
		t.Fatal("validator mutated contract")
	}
	for _, mutate := range []func(*ScopeBinding){func(s *ScopeBinding) { s.ProjectRef = "project:other" }, func(s *ScopeBinding) { s.WorkstreamRef = "workstream:other" }, func(s *ScopeBinding) { s.WorksetRef = "workset:other" }, func(s *ScopeBinding) { s.CallGraphRef = "callgraph:other" }, func(s *ScopeBinding) { s.WorkpointRef = "workpoint:other" }, func(s *ScopeBinding) { s.WorkItemRef = "item:other" }} {
		r, m, _ := contracts()
		mutate(&r.Scope)
		if !errors.Is(ValidateManifest(m, r), ErrDerivativeScopeMismatch) {
			t.Fatal("lineage change retained old manifest binding")
		}
	}
}

func TestT09AAdditionalIdentityInputs(t *testing.T) {
	for name, mutate := range map[string]func(*DerivativeRequest, *DerivativeManifest){
		"font":            func(r *DerivativeRequest, _ *DerivativeManifest) { r.Rendering.FontRefs = []string{"font:other"} },
		"color":           func(r *DerivativeRequest, _ *DerivativeManifest) { r.Rendering.ColorProfileRef = "display-p3" },
		"accessibility":   func(r *DerivativeRequest, _ *DerivativeManifest) { r.AccessibilityTarget = AccessibilityPDFUA2 },
		"locale":          func(r *DerivativeRequest, _ *DerivativeManifest) { r.Locale = "ar" },
		"direction":       func(r *DerivativeRequest, _ *DerivativeManifest) { r.Direction = DirectionRTL },
		"revision":        func(r *DerivativeRequest, _ *DerivativeManifest) { r.ArtifactRevision++ },
		"projection":      func(r *DerivativeRequest, _ *DerivativeManifest) { r.ProjectionSHA256 = da },
		"renderer_digest": func(_ *DerivativeRequest, m *DerivativeManifest) { m.Renderer.BinarySHA256 = db },
		"viewer":          func(_ *DerivativeRequest, m *DerivativeManifest) { m.ViewerMatrix.Entries[0].Version = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			r, m, _ := contracts()
			old := m.DerivativeID
			mutate(&r, &m)
			next, err := DerivativeID(r, m.Renderer, m.ViewerMatrix, m.Licenses)
			if err != nil || next == old {
				t.Fatal("truth input did not bind identity")
			}
		})
	}
}

func TestT09ABoundsVocabularyLicenseAndPrivatePaths(t *testing.T) {
	r, m, _ := contracts()
	r.AssetRefs = append(r.AssetRefs, r.ClaimRefs[0])
	if ValidateRequest(r) == nil {
		t.Fatal("cross-class duplicate selected ref accepted")
	}
	for _, value := range []string{"/tmp/private", "/etc/private", `Z:\\private\\proof`, "file:///private/proof"} {
		r, _, _ := contracts()
		r.LicensePolicyRef = value
		if !errors.Is(ValidateRequest(r), ErrDerivativeContractInvalid) {
			t.Fatal("private path accepted")
		}
	}
	for _, mutate := range []func(*DerivativeRequest, *DerivativeManifest){
		func(r *DerivativeRequest, _ *DerivativeManifest) { r.Direction = "future" },
		func(r *DerivativeRequest, _ *DerivativeManifest) { r.AccessibilityTarget = "future" },
		func(_ *DerivativeRequest, m *DerivativeManifest) { m.ArchivePosture = "future" },
		func(_ *DerivativeRequest, m *DerivativeManifest) { m.AccessibilityPosture = "future" },
		func(_ *DerivativeRequest, m *DerivativeManifest) { m.ViewerMatrix.Entries[0].Status = "future" },
		func(_ *DerivativeRequest, m *DerivativeManifest) {
			m.ViewerMatrix.Entries[0].EvidenceRefs = make([]string, MaxRefs+1)
			for i := range m.ViewerMatrix.Entries[0].EvidenceRefs {
				m.ViewerMatrix.Entries[0].EvidenceRefs[i] = fmt.Sprintf("evidence:%d", i)
			}
		},
	} {
		r, m, _ := contracts()
		mutate(&r, &m)
		if ValidateManifest(m, r) == nil {
			t.Fatal("unknown or unbounded contract accepted")
		}
	}
	entries := make([]ArchiveEntry, MaxEntries+1)
	for i := range entries {
		entries[i] = ArchiveEntry{Path: fmt.Sprintf("file-%d", i), SHA256: da, MIME: "text/plain", Bytes: 1}
	}
	if !errors.Is(validateArchive(ArchiveSafe, entries), ErrDerivativeUnsafeArchive) {
		t.Fatal("unbounded archive entries accepted")
	}
	for _, mutate := range []func(*DerivativeManifest){func(m *DerivativeManifest) { m.Licenses[0].AttributionRef = "" }, func(m *DerivativeManifest) { m.Licenses[0].EvidenceRef = "" }, func(m *DerivativeManifest) { m.Licenses[0].DerivativePermitted = false }} {
		r, m, _ := contracts()
		mutate(&m)
		if !errors.Is(ValidateManifest(m, r), ErrDerivativeLicenseMissing) {
			t.Fatal("unproven licensed use accepted")
		}
	}
	_ = r
	_ = m
}

func TestT09AGoldenRoundTrips(t *testing.T) {
	request, _, _ := contracts()
	for _, name := range []string{"derivative-manifest.golden.json", "delivery-receipt.golden.json"} {
		raw, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if name == "derivative-manifest.golden.json" {
			var v DerivativeManifest
			if err = json.Unmarshal(raw, &v); err != nil {
				t.Fatal(err)
			}
			if err = ValidateManifest(v, request); err != nil {
				t.Fatal(err)
			}
			value = v
		} else {
			var v DeliveryReceipt
			if err = json.Unmarshal(raw, &v); err != nil {
				t.Fatal(err)
			}
			if err = ValidateDelivery(v); err != nil {
				t.Fatal(err)
			}
			value = v
		}
		next, err := json.MarshalIndent(value, "", "  ")
		if err != nil || !bytes.Equal(append(next, '\n'), raw) {
			t.Fatal("golden roundtrip changed contract")
		}
	}
}

func TestT09AEncodedUnsafePathsFailClosed(t *testing.T) {
	for _, value := range []string{"%2fetc/passwd", "a%00b", "a%5cb", "a%0ab"} {
		if relativeRef(value) {
			t.Fatal("encoded unsafe path accepted")
		}
	}
	if validateArchive(ArchiveSafe, []ArchiveEntry{{Path: "payload", SHA256: da, MIME: "application/x-pie-executable", Bytes: 1}}) == nil {
		t.Fatal("PIE executable MIME accepted")
	}
}
