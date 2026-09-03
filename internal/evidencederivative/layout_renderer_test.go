package evidencederivative

import (
	"bytes"
	"testing"
)

func TestRenderProjectionHTMLLayoutModesAreSelfContained(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.Rendering = HTMLSystemUIRenderingProfile()
	for _, kind := range []DerivativeType{DerivativePrint, DerivativeHTMLSlides} {
		request.DerivativeType = kind
		rendered, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:layout", manifest.CreatedAt)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if bytes.Contains(rendered.Output, []byte("<script")) || bytes.Contains(rendered.Output, []byte("http://")) || bytes.Contains(rendered.Output, []byte("https://")) {
			t.Fatalf("%s is not self-contained", kind)
		}
		if err := ValidateManifest(rendered.Manifest, request); err != nil {
			t.Fatalf("%s manifest: %v", kind, err)
		}
	}
}
