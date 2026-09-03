package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"testing"
)

func TestPPTXMinimalProfileDigestIsFrozen(t *testing.T) {
	digest := sha256.Sum256([]byte("uiai.evidence.pptx-minimal-arial-v1\n"))
	if got := fmt.Sprintf("%x", digest); got != PPTXMinimalProfileSHA256 {
		t.Fatalf("profile digest = %s", got)
	}
}

func TestRenderProjectionPPTXIsDeterministicAndSafe(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePPTX
	r.AccessibilityTarget = AccessibilityNotApplicable
	r.Rendering = PPTXMinimalRenderingProfile()
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
	required := map[string]bool{
		"[Content_Types].xml": false, "_rels/.rels": false, "ppt/presentation.xml": false,
		"ppt/_rels/presentation.xml.rels": false, "ppt/slideMasters/slideMaster1.xml": false,
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": false, "ppt/slideLayouts/slideLayout1.xml": false,
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": false, "ppt/theme/theme1.xml": false,
	}
	names := make(map[string]struct{}, len(z.File))
	relationshipBodies := map[string][]byte{}
	var xmlPayload bytes.Buffer
	for _, f := range z.File {
		names[f.Name] = struct{}{}
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
			xmlPayload.Write(body)
			if strings.HasSuffix(f.Name, ".rels") {
				relationshipBodies[f.Name] = body
			}
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
	for relationshipPath, body := range relationshipBodies {
		assertPPTXRelationshipsResolve(t, relationshipPath, body, names)
	}
	for _, ref := range []string{r.AssetRefs[0], r.CitationRefs[0], r.OmissionRefs[0]} {
		if !bytes.Contains(xmlPayload.Bytes(), []byte(ref)) {
			t.Fatalf("selected or omitted evidence %s missing from slides", ref)
		}
	}
	if a.Manifest.ArchivePosture != ArchiveSafe || len(a.Manifest.ArchiveEntries) != len(z.File) {
		t.Fatal("archive manifest mismatch")
	}
	if err := VerifyArchive(a.Output, a.Manifest); err != nil {
		t.Fatal(err)
	}
	if e := ValidateManifest(a.Manifest, r); e != nil {
		t.Fatal(e)
	}
}

func assertPPTXRelationshipsResolve(t *testing.T, relationshipPath string, body []byte, names map[string]struct{}) {
	t.Helper()
	var relationships struct {
		Items []struct {
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(body, &relationships); err != nil {
		t.Fatal(err)
	}
	base := ""
	if relationshipPath != "_rels/.rels" {
		prefix, file, found := strings.Cut(relationshipPath, "/_rels/")
		if !found || !strings.HasSuffix(file, ".rels") {
			t.Fatalf("invalid relationship path %s", relationshipPath)
		}
		base = path.Dir(prefix + "/" + strings.TrimSuffix(file, ".rels"))
	}
	for _, relationship := range relationships.Items {
		resolved := path.Clean(path.Join(base, relationship.Target))
		if _, found := names[resolved]; !found {
			t.Fatalf("%s has dangling target %s (%s)", relationshipPath, relationship.Target, resolved)
		}
	}
}

func TestRenderProjectionPPTXRejectsFalseFontProfile(t *testing.T) {
	r, p, m := portableFixture(t)
	r.DerivativeType = DerivativePPTX
	r.AccessibilityTarget = AccessibilityNotApplicable
	if _, err := RenderProjectionPPTX(r, p, m.Renderer, m.ViewerMatrix, m.Licenses, "receipt:pptx", m.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unimplemented requested font profile accepted: %v", err)
	}
}
