package evidencederivative

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func TestRenderProjectionMarkdownIncludesOnlySelectedEvidence(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeMarkdown
	if len(projection.Claims) > 1 {
		request.ClaimRefs = request.ClaimRefs[:1]
	}
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	rendered, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:markdown", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rendered.Output)
	if !strings.Contains(body, projection.Title) || !strings.Contains(body, projection.Claims[0].Statement) || !strings.Contains(body, "## Evidence posture") {
		t.Fatalf("missing selected evidence in markdown:\n%s", body)
	}
	if len(projection.Claims) > 1 && strings.Contains(body, projection.Claims[1].Statement) {
		t.Fatalf("unselected claim leaked into markdown:\n%s", body)
	}
	if rendered.Manifest.OutputMIME != "text/markdown; charset=utf-8" || !strings.HasSuffix(rendered.Manifest.OutputRef, ".md") {
		t.Fatalf("markdown manifest = %#v", rendered.Manifest)
	}
	if err := ValidateManifest(rendered.Manifest, request); err != nil {
		t.Fatal(err)
	}
}

func TestRenderProjectionCSVIsDeterministicAndParseable(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeCSV
	first, err := RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:csv", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:csv", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("CSV render is not deterministic")
	}
	rows, err := csv.NewReader(strings.NewReader(string(first.Output))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	wantRows := 1 + len(request.ClaimRefs) + len(request.AssetRefs) + len(request.CitationRefs) + len(manifest.Licenses) + len(request.OmissionRefs)
	if len(rows) != wantRows || rows[0][0] != "record_type" {
		t.Fatalf("CSV rows = %#v, want %d rows", rows, wantRows)
	}
	if first.Manifest.OutputMIME != "text/csv; charset=utf-8" || !strings.HasSuffix(first.Manifest.OutputRef, ".csv") {
		t.Fatalf("CSV manifest = %#v", first.Manifest)
	}
}

func TestPortableOutputsIncludeRequiredAttribution(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	for _, derivativeType := range []DerivativeType{DerivativeMarkdown, DerivativeEmailText, DerivativeCSV} {
		request.DerivativeType = derivativeType
		var output RenderedDerivative
		var err error
		if derivativeType == DerivativeCSV {
			output, err = RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:license", manifest.CreatedAt)
		} else {
			output, err = RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:license", manifest.CreatedAt)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{manifest.Licenses[0].LicenseRef, manifest.Licenses[0].LicenseSHA256, manifest.Licenses[0].AttributionRef, manifest.Licenses[0].EvidenceRef} {
			if !strings.Contains(string(output.Output), value) {
				t.Fatalf("%s omitted licensing value %s", derivativeType, value)
			}
		}
		if !strings.Contains(strings.ToLower(string(output.Output)), "derivative permitted: true") {
			t.Fatalf("%s omitted derivative permission", derivativeType)
		}
	}
}

func TestRenderProjectionEmailTextUsesPlainTextDeliveryShape(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeEmailText
	rendered, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:email-text", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if rendered.Manifest.OutputMIME != "text/plain; charset=utf-8" || !strings.HasSuffix(rendered.Manifest.OutputRef, ".txt") {
		t.Fatalf("email text manifest = %#v", rendered.Manifest)
	}
	body := string(rendered.Output)
	if strings.HasPrefix(body, "# ") || strings.Contains(body, "**") || strings.Contains(body, "`"+projection.Assets[0].Ref+"`") {
		t.Fatalf("email text contains Markdown presentation syntax:\n%s", body)
	}
}

func TestPortableRenderersRequireCompleteCanonicalOmissions(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeMarkdown
	if len(request.OmissionRefs) < 2 {
		t.Fatal("fixture lacks omitted evidence")
	}
	missing := request
	missing.OmissionRefs = append([]string(nil), request.OmissionRefs[1:]...)
	missing.RequiredEvidenceRefs = selectedEvidenceRefs(missing)
	if _, err := RenderProjectionMarkdown(missing, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:missing-omission", manifest.CreatedAt); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("missing omission error = %v", err)
	}
	reordered := request
	reordered.OmissionRefs = append([]string(nil), request.OmissionRefs...)
	reordered.OmissionRefs[0], reordered.OmissionRefs[1] = reordered.OmissionRefs[1], reordered.OmissionRefs[0]
	reordered.RequiredEvidenceRefs = selectedEvidenceRefs(reordered)
	if _, err := RenderProjectionMarkdown(reordered, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:reordered-omission", manifest.CreatedAt); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("noncanonical omission order error = %v", err)
	}
	rendered, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:complete-omission", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rendered.Output)
	for _, ref := range []string{"event:capture", "inspection:redaction", "security:1", "custody:1", "attestation:1", "trust:1", "receipt:1", "warning:bounded_projection"} {
		if !strings.Contains(body, markdownText(ref)) {
			t.Fatalf("omitted evidence %s missing from output", ref)
		}
	}
}

func TestPortableRenderersRejectFalseRenderingProfile(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.Rendering.ProfileRef = "rendering:unimplemented"
	for _, derivativeType := range []DerivativeType{DerivativeMarkdown, DerivativeEmailText} {
		request.DerivativeType = derivativeType
		if _, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:false-profile", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
			t.Fatalf("%s false profile accepted: %v", derivativeType, err)
		}
	}
	request.DerivativeType = DerivativeCSV
	if _, err := RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:false-profile", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("CSV false profile accepted: %v", err)
	}
}

func TestPortableRenderersFailClosedOnBindingAndSelectionDrift(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeMarkdown
	request.ProjectionSHA256 = strings.Repeat("f", 64)
	if _, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:markdown", manifest.CreatedAt); !errors.Is(err, ErrProjectionMismatch) {
		t.Fatalf("projection drift error = %v", err)
	}
	request, projection, manifest = portableFixture(t)
	request.DerivativeType = DerivativeCSV
	request.ClaimRefs[0] = "claim:not-in-projection"
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	if _, err := RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:csv", manifest.CreatedAt); !errors.Is(err, ErrDerivativeSelectionIncomplete) {
		t.Fatalf("selection drift error = %v", err)
	}
}

func TestCSVNeutralizesFormulasAndDisplayControls(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeCSV
	projection.Claims[0].Statement = "=HYPERLINK(\"https://attacker.invalid\")\x1b\u202e"
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
	rendered, err := RenderProjectionCSV(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:hostile-csv", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(rendered.Output))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		for _, cell := range row {
			if strings.HasPrefix(cell, "=") || strings.HasPrefix(cell, "+") || strings.HasPrefix(cell, "-") || strings.HasPrefix(cell, "@") || strings.ContainsAny(cell, "\x1b\u202e") {
				t.Fatalf("unsafe CSV cell %q", cell)
			}
		}
	}
	if len(rows) < 2 || rows[1][2] != "'=HYPERLINK(\"https://attacker.invalid\")" {
		t.Fatalf("formula was not visibly neutralized: %#v", rows)
	}
}

func TestMarkdownEscapesInjectedStructureAndUnselectedCitations(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeMarkdown
	projection.Claims[0].Statement = "trusted line\n# injected heading *emphasis*"
	request.CitationRefs = nil
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
	rendered, err := RenderProjectionMarkdown(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:markdown", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rendered.Output)
	if strings.Contains(body, "\n# injected") || strings.Contains(body, "- Citations:") || !strings.Contains(body, "\\# injected heading \\*emphasis\\*") {
		t.Fatalf("unsafe or leaky markdown:\n%s", body)
	}
}

func portableFixture(t *testing.T) (DerivativeRequest, evidencepwa.Projection, DerivativeManifest) {
	t.Helper()
	body, err := os.ReadFile("../evidencepwa/testdata/projection.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var projection evidencepwa.Projection
	if err := json.Unmarshal(body, &projection); err != nil {
		t.Fatal(err)
	}
	if err := evidencepwa.ValidateProjection(projection); err != nil {
		t.Fatal(err)
	}
	request, manifest, _ := contracts()
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.Scope = projection.Artifact.Scope
	request.ArtifactRef = projection.Artifact.ArtifactRef
	request.ArtifactSHA256 = projection.Artifact.ManifestSHA256
	request.ArtifactRevision = projection.Artifact.Revision
	request.ProjectionRef = projection.ProjectionID
	request.ProjectionSHA256 = digest
	request.AccessibilityTarget = AccessibilityPlainText
	request.ClaimRefs = projectionClaimRefs(projection)
	request.AssetRefs = projectionAssetRefs(projection)
	request.CitationRefs = projectionCitationRefs(projection)
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	request.Rendering = PortableDataRenderingProfile()
	if len(request.ClaimRefs) == 0 || len(request.AssetRefs) == 0 || len(request.CitationRefs) == 0 {
		t.Fatal("projection fixture lacks portable derivative evidence")
	}
	manifest.Licenses = manifest.Licenses[:1]
	manifest.Licenses[0].AssetRef = request.AssetRefs[0]
	if err := ValidateRequest(request); err != nil {
		t.Fatal(err)
	}
	return request, projection, manifest
}

func projectionClaimRefs(projection evidencepwa.Projection) []string {
	refs := make([]string, len(projection.Claims))
	for index, claim := range projection.Claims {
		refs[index] = claim.ClaimID
	}
	return refs
}

func projectionAssetRefs(projection evidencepwa.Projection) []string {
	refs := make([]string, len(projection.Assets))
	for index, asset := range projection.Assets {
		refs[index] = asset.AssetID
	}
	return refs
}

func projectionCitationRefs(projection evidencepwa.Projection) []string {
	refs := make([]string, len(projection.Citations))
	for index, citation := range projection.Citations {
		refs[index] = citation.CitationID
	}
	return refs
}

func selectedEvidenceRefs(request DerivativeRequest) []string {
	refs := append([]string(nil), request.ClaimRefs...)
	refs = append(refs, request.AssetRefs...)
	refs = append(refs, request.CitationRefs...)
	refs = append(refs, request.OmissionRefs...)
	return refs
}
