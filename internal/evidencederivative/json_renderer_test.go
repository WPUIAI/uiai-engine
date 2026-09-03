package evidencederivative

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func TestRenderProjectionJSONIncludesOnlySelectedEvidence(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	if len(request.ClaimRefs) > 1 {
		request.ClaimRefs = request.ClaimRefs[:1]
	}
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	matrixBefore := cloneViewerMatrix(manifest.ViewerMatrix)
	licensesBefore := append([]LicenseAttestation(nil), manifest.Licenses...)
	rendered, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Schema       string              `json:"schema"`
		Claims       []evidencepwa.Claim `json:"claims"`
		OmissionRefs []string            `json:"omission_refs"`
	}
	if err := json.Unmarshal(rendered.Output, &output); err != nil {
		t.Fatal(err)
	}
	if output.Schema != "uiai.evidence_derivative_json.v1" || len(output.Claims) != len(request.ClaimRefs) || !reflect.DeepEqual(output.OmissionRefs, request.OmissionRefs) {
		t.Fatalf("selected output = %#v", output)
	}
	if len(projection.Claims) > len(request.ClaimRefs) && bytes.Contains(rendered.Output, []byte(projection.Claims[1].Statement)) {
		t.Fatal("unselected claim leaked into JSON")
	}
	if err := ValidateManifest(rendered.Manifest, request); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.ViewerMatrix, matrixBefore) || !reflect.DeepEqual(manifest.Licenses, licensesBefore) {
		t.Fatal("renderer mutated caller-owned truth inputs")
	}
}

func TestRenderCanonicalJSONStrictlyDecodesPWAProjection(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	body, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := RenderCanonicalJSON(request, body, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	typed, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil || !reflect.DeepEqual(wrapped, typed) {
		t.Fatalf("wrapper mismatch: %v", err)
	}
	unknown := append(append([]byte(nil), body[:len(body)-1]...), []byte(`,"unknown":true}`)...)
	if _, err := RenderCanonicalJSON(request, unknown, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unknown field error = %v", err)
	}
	multiple := append(append([]byte(nil), body...), body...)
	if _, err := RenderCanonicalJSON(request, multiple, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("multiple values error = %v", err)
	}
}

func TestRenderProjectionJSONPreservesLargeIntegersDeterministically(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	projection.Assets[0].Bytes = 9007199254740993
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
	first, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt)
	if err != nil || !reflect.DeepEqual(first, second) || !strings.Contains(string(first.Output), "9007199254740993") {
		t.Fatalf("large integer render mismatch: %v", err)
	}
}

func TestRenderProjectionJSONFailsClosedOnBindingTypeAndOmissions(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	request.ProjectionSHA256 = strings.Repeat("f", 64)
	if _, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrProjectionMismatch) {
		t.Fatalf("projection mismatch error = %v", err)
	}
	request, projection, manifest = portableFixture(t)
	request.DerivativeType = DerivativeJSON
	request.OmissionRefs = request.OmissionRefs[1:]
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	if _, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("omission error = %v", err)
	}
	request, projection, manifest = portableFixture(t)
	request.DerivativeType = DerivativePDF
	if _, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:json", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("wrong type error = %v", err)
	}
}
