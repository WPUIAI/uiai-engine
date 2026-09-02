package evidencederivative

import (
	"bytes"
	"testing"
)

func TestRenderProjectionRichTextDeterministicSafe(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativeRichText
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
	if e := ValidateManifest(a.Manifest, r); e != nil {
		t.Fatal(e)
	}
}
