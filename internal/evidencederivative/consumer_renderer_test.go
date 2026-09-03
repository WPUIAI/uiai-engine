package evidencederivative

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFInfoConsumerAcceptsRenderedPDF(t *testing.T) {
	binary, err := exec.LookPath("pdfinfo")
	if err != nil {
		t.Skip("pdfinfo consumer unavailable")
	}
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativePDF
	request.AccessibilityTarget = AccessibilityNotApplicable
	rendered, err := RenderProjectionPDF(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:pdf-consumer", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "evidence.pdf")
	if err := os.WriteFile(path, rendered.Output, 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(binary, path).CombinedOutput()
	if err != nil {
		t.Fatalf("pdfinfo rejected derivative: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Pages:") || !strings.Contains(string(output), "PDF version:") {
		t.Fatalf("pdfinfo omitted expected structure:\n%s", output)
	}
}

func TestUnzipConsumerAcceptsRenderedPPTX(t *testing.T) {
	binary, err := exec.LookPath("unzip")
	if err != nil {
		t.Skip("unzip consumer unavailable")
	}
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativePPTX
	request.AccessibilityTarget = AccessibilityNotApplicable
	rendered, err := RenderProjectionPPTX(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:pptx-consumer", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "evidence.pptx")
	if err := os.WriteFile(path, rendered.Output, 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(binary, "-t", path).CombinedOutput()
	if err != nil {
		t.Fatalf("unzip rejected derivative: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "No errors detected") {
		t.Fatalf("unzip did not confirm archive integrity:\n%s", output)
	}
}
