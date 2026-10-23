package evidencederivative

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func TestHTMLSystemUIProfileDigestIsFrozen(t *testing.T) {
	digest := sha256.Sum256([]byte("uiai.evidence.html-viewer-system-ui-srgb-v2\n"))
	if got := fmt.Sprintf("%x", digest); got != HTMLSystemUIProfileSHA256 {
		t.Fatalf("profile digest = %s", got)
	}
}

func TestRenderProjectionHTMLIncludesOnlySelectedEvidence(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeHTML
	request.Rendering = HTMLSystemUIRenderingProfile()
	if len(projection.Claims) > 1 {
		request.ClaimRefs = request.ClaimRefs[:1]
	}
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	rendered, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rendered.Output)
	if !strings.HasPrefix(body, "<!doctype html>\n") || !strings.Contains(body, htmlText(projection.Title)) || !strings.Contains(body, htmlText(projection.Claims[0].Statement)) {
		t.Fatalf("selected evidence missing from HTML:\n%s", body)
	}
	if len(projection.Claims) > 1 && strings.Contains(body, htmlText(projection.Claims[1].Statement)) {
		t.Fatalf("unselected claim leaked into HTML:\n%s", body)
	}
	for _, value := range []string{manifest.Licenses[0].LicenseRef, manifest.Licenses[0].LicenseSHA256, manifest.Licenses[0].AttributionRef, manifest.Licenses[0].EvidenceRef, "Derivative permitted: true"} {
		if !strings.Contains(body, htmlText(value)) {
			t.Fatalf("license or attribution %s missing from HTML", value)
		}
	}
	if rendered.Manifest.OutputMIME != "text/html; charset=utf-8" || !strings.HasSuffix(rendered.Manifest.OutputRef, ".html") {
		t.Fatalf("HTML manifest = %#v", rendered.Manifest)
	}
	if err := ValidateManifest(rendered.Manifest, request); err != nil {
		t.Fatal(err)
	}
}

func TestRenderProjectionHTMLEscapesHostileContentAndNeverFetches(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeHTML
	request.Rendering = HTMLSystemUIRenderingProfile()
	projection.Title = `<script>alert("title")</script>`
	projection.Claims[0].Statement = `<img src="https://attacker.invalid/x" onerror="alert(1)">`
	bindHTMLProjection(t, &request, projection)
	rendered, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:hostile-html", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rendered.Output)
	for _, activeTag := range []string{"<script", "<img ", "<a ", "<link ", "<iframe", "<form"} {
		if strings.Contains(body, activeTag) {
			t.Fatalf("active HTML tag %q escaped containment:\n%s", activeTag, body)
		}
	}
	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&lt;img src=&#34;https://attacker.invalid/x&#34;") {
		t.Fatalf("hostile text was not visibly escaped:\n%s", body)
	}
	if !strings.Contains(body, "Content-Security-Policy") || !strings.Contains(body, "default-src 'none'") {
		t.Fatalf("fail-closed content policy missing:\n%s", body)
	}
}

func TestRenderProjectionHTMLIsDeterministicForWebAndEmail(t *testing.T) {
	for _, derivativeType := range []DerivativeType{DerivativeHTML, DerivativeEmailHTML} {
		t.Run(string(derivativeType), func(t *testing.T) {
			request, projection, manifest := portableFixture(t)
			request.DerivativeType = derivativeType
			request.Rendering = HTMLSystemUIRenderingProfile()
			first, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html-deterministic", manifest.CreatedAt)
			if err != nil {
				t.Fatal(err)
			}
			second, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html-deterministic", manifest.CreatedAt)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("%s render is not deterministic", derivativeType)
			}
		})
	}
}

func TestRenderProjectionHTMLRejectsFalseFontProfile(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeHTML
	if _, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unimplemented requested font profile accepted: %v", err)
	}
}

func TestRenderProjectionHTMLFailsClosedOnUnsupportedTypeAndBindingDrift(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativePDF
	if _, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unsupported type error = %v", err)
	}
	request.DerivativeType = DerivativeHTML
	request.Rendering = HTMLSystemUIRenderingProfile()
	request.ArtifactRevision++
	if _, err := RenderProjectionHTML(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:html", manifest.CreatedAt); !errors.Is(err, ErrProjectionMismatch) {
		t.Fatalf("binding drift error = %v", err)
	}
}

func bindHTMLProjection(t *testing.T, request *DerivativeRequest, projection evidencepwa.Projection) {
	t.Helper()
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
}
