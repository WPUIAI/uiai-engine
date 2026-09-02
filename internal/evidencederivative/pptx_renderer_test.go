package evidencederivative

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestRenderProjectionPPTXIsDeterministicAndSafe(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePPTX
	r.AccessibilityTarget = AccessibilityNotApplicable
	a, e := RenderProjectionPPTX(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pptx", m.CreatedAt)
	if e != nil {
		t.Fatal(e)
	}
	b, e := RenderProjectionPPTX(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pptx", m.CreatedAt)
	if e != nil || !bytes.Equal(a.Output, b.Output) {
		t.Fatal("nondeterministic PPTX")
	}
	z, e := zip.NewReader(bytes.NewReader(a.Output), int64(len(a.Output)))
	if e != nil {
		t.Fatal(e)
	}
	required := map[string]bool{"[Content_Types].xml": false, "_rels/.rels": false, "ppt/presentation.xml": false, "ppt/_rels/presentation.xml.rels": false}
	for _, f := range z.File {
		if strings.Contains(f.Name, "..") || strings.HasPrefix(f.Name, "/") {
			t.Fatalf("unsafe %q", f.Name)
		}
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
		rc, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(rc)
		rc.Close()
		if e != nil {
			t.Fatal(e)
		}
		if bytes.Contains(body, []byte("TargetMode=\"External\"")) || bytes.Contains(body, []byte("vbaProject")) {
			t.Fatalf("active content in %s", f.Name)
		}
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			var value any
			if err := xml.Unmarshal(body, &value); err != nil {
				t.Fatalf("invalid XML %s: %v", f.Name, err)
			}
		}
	}
	for name, ok := range required {
		if !ok {
			t.Fatalf("missing %s", name)
		}
	}
	if a.Manifest.ArchivePosture != ArchiveSafe || len(a.Manifest.ArchiveEntries) != len(z.File) {
		t.Fatal("archive manifest mismatch")
	}
	if e := ValidateManifest(a.Manifest, r); e != nil {
		t.Fatal(e)
	}
}
