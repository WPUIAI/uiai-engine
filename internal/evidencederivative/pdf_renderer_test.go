package evidencederivative

import (
	"bytes"
	"errors"
	"testing"
)

func TestRenderProjectionPDFIsDeterministicAndStructurallyBound(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePDF
	r.AccessibilityTarget = AccessibilityNotApplicable
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
	r.AccessibilityTarget = AccessibilityPDFUA1
	_, e := RenderProjectionPDF(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pdf", m.CreatedAt)
	if !errors.Is(e, ErrDerivativeContractInvalid) {
		t.Fatalf("error=%v", e)
	}
}
