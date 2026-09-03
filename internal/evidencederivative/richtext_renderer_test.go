package evidencederivative

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
)

func TestRTFArialProfileDigestIsFrozen(t *testing.T) {
	digest := sha256.Sum256([]byte("uiai.evidence.rtf-arial-v1\n"))
	if got := fmt.Sprintf("%x", digest); got != RTFArialProfileSHA256 {
		t.Fatalf("profile digest = %s", got)
	}
}

func TestRenderProjectionRichTextDeterministicSafe(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativeRichText
	r.Rendering = RTFArialRenderingProfile()
	a, e := RenderProjectionRichText(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:rtf", m.CreatedAt)
	if e != nil {
		t.Fatal(e)
	}
	b, e := RenderProjectionRichText(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:rtf", m.CreatedAt)
	if e != nil || !bytes.Equal(a.Output, b.Output) {
		t.Fatal("nondeterministic rich text")
	}
	if !bytes.HasPrefix(a.Output, []byte(`{\rtf1`)) {
		t.Fatalf("unsafe RTF %q", a.Output)
	}
	for _, value := range []string{m.Licenses[0].LicenseRef, m.Licenses[0].AttributionRef, m.Licenses[0].EvidenceRef} {
		if !bytes.Contains(a.Output, []byte(value)) {
			t.Fatalf("license or attribution %s missing from RTF", value)
		}
	}
	if e := ValidateManifest(a.Manifest, r); e != nil {
		t.Fatal(e)
	}
}

func TestRenderProjectionRichTextRejectsFalseFontProfile(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativeRichText
	if _, err := RenderProjectionRichText(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:rtf", m.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unimplemented requested font profile accepted: %v", err)
	}
}
