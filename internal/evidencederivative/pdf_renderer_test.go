package evidencederivative

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
)

func TestPDFCore14ProfileDigestIsFrozen(t *testing.T) {
	digest := sha256.Sum256([]byte("uiai.evidence.pdf-core14-ascii.v1\n"))
	if got := fmt.Sprintf("%x", digest); got != PDFCore14ProfileSHA256 {
		t.Fatalf("profile digest = %s", got)
	}
}

func TestRenderProjectionPDFIsDeterministicAndStructurallyBound(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePDF
	r.AccessibilityTarget = AccessibilityNotApplicable
	r.Rendering = PDFCore14RenderingProfile()
	a, e := RenderProjectionPDF(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pdf", m.CreatedAt)
	if e != nil {
		t.Fatal(e)
	}
	b, e := RenderProjectionPDF(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pdf", m.CreatedAt)
	if e != nil || !bytes.Equal(a.Output, b.Output) {
		t.Fatal("PDF is not deterministic")
	}
	for _, marker := range [][]byte{[]byte("%PDF-1.7"), []byte("xref\n"), []byte("startxref\n"), []byte("%%EOF\n")} {
		if !bytes.Contains(a.Output, marker) {
			t.Fatalf("missing %q", marker)
		}
	}
	for _, value := range []string{m.Licenses[0].LicenseRef, m.Licenses[0].AttributionRef, m.Licenses[0].EvidenceRef} {
		if !bytes.Contains(a.Output, []byte(value)) {
			t.Fatalf("license or attribution %s missing from PDF", value)
		}
	}
	if a.Manifest.OutputMIME != "application/pdf" || a.Manifest.AccessibilityPosture != ConformanceNotClaimed {
		t.Fatalf("manifest=%#v", a.Manifest)
	}
	if e := ValidateManifest(a.Manifest, r); e != nil {
		t.Fatal(e)
	}
}
func TestRenderProjectionPDFRejectsUnprovenPDFUA(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePDF
	r.Rendering = PDFCore14RenderingProfile()
	r.AccessibilityTarget = AccessibilityPDFUA1
	_, e := RenderProjectionPDF(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pdf", m.CreatedAt)
	if !errors.Is(e, ErrDerivativeContractInvalid) {
		t.Fatalf("error=%v", e)
	}
}

func TestRenderProjectionPDFRejectsFalseFontProfile(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePDF
	r.AccessibilityTarget = AccessibilityNotApplicable
	if _, err := RenderProjectionPDF(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pdf", m.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unimplemented requested font profile accepted: %v", err)
	}
}
