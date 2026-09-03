package evidencederivative

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
)

func TestRenderCanonicalJSONBindsProjectionAndTruthInputs(t *testing.T) {
	request, manifest, _ := contracts()
	projection := []byte(" {\"b\":2,\"a\":1} \n")
	request.DerivativeType = DerivativeJSON
	request.Rendering = PortableDataRenderingProfile()
	request.AccessibilityTarget = AccessibilityNotApplicable
	request.ProjectionSHA256 = hashJSONTest(projection)
	matrixBefore := cloneViewerMatrix(manifest.ViewerMatrix)
	licensesBefore := append([]LicenseAttestation(nil), manifest.Licenses...)
	rendered, err := RenderCanonicalJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json-render", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(rendered.Output), "{\"a\":1,\"b\":2}\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if rendered.Manifest.OutputSHA256 != hashJSONTest(rendered.Output) || rendered.Manifest.OutputBytes != uint64(len(rendered.Output)) || rendered.Manifest.OutputMIME != "application/json" {
		t.Fatalf("output manifest = %#v", rendered.Manifest)
	}
	if rendered.Manifest.ProjectionSHA256 != request.ProjectionSHA256 || rendered.Manifest.RequestRef != request.RequestID || rendered.Manifest.ReceiptRef != "receipt:json-render" {
		t.Fatalf("truth bindings = %#v", rendered.Manifest)
	}
	if err := ValidateManifest(rendered.Manifest, request); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.ViewerMatrix, matrixBefore) || !reflect.DeepEqual(manifest.Licenses, licensesBefore) {
		t.Fatal("renderer mutated caller-owned truth inputs")
	}
}

func TestRenderCanonicalJSONIsDeterministic(t *testing.T) {
	request, manifest, _ := contracts()
	projection := []byte("{\"n\":9007199254740993,\"nested\":{\"z\":0,\"a\":true}}")
	request.DerivativeType = DerivativeJSON
	request.Rendering = PortableDataRenderingProfile()
	request.AccessibilityTarget = AccessibilityNotApplicable
	request.ProjectionSHA256 = hashJSONTest(projection)
	first, err := RenderCanonicalJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderCanonicalJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || string(first.Output) != "{\"n\":9007199254740993,\"nested\":{\"a\":true,\"z\":0}}\n" {
		t.Fatalf("nondeterministic render: first=%#v second=%#v", first, second)
	}
}

func TestRenderCanonicalJSONFailsClosed(t *testing.T) {
	request, manifest, _ := contracts()
	projection := []byte("{\"ok\":true}")
	request.DerivativeType = DerivativeJSON
	request.Rendering = PortableDataRenderingProfile()
	request.AccessibilityTarget = AccessibilityNotApplicable
	request.ProjectionSHA256 = hashJSONTest(projection)
	request.ProjectionSHA256 = hashJSONTest([]byte("different"))
	if _, err := RenderCanonicalJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrProjectionMismatch) {
		t.Fatalf("projection mismatch error = %v", err)
	}
	request.ProjectionSHA256 = hashJSONTest([]byte("{} {}"))
	if _, err := RenderCanonicalJSON(request, []byte("{} {}"), manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("multiple JSON values error = %v", err)
	}
	request.DerivativeType = DerivativePDF
	request.ProjectionSHA256 = hashJSONTest(projection)
	if _, err := RenderCanonicalJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("wrong derivative type error = %v", err)
	}
}

func hashJSONTest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
