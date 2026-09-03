package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

type pptxFile struct{ path, mime, body string }

const PPTXMinimalProfileRef = "rendering:pptx-minimal-arial-v1"
const PPTXMinimalProfileSHA256 = "0ca5e6ee3a268b838e077a474bbdd68756199ed5f75ce65b73b8a0a99cc51924"

func PPTXMinimalRenderingProfile() RenderingProfile {
	return RenderingProfile{
		ProfileRef:      PPTXMinimalProfileRef,
		ProfileSHA256:   PPTXMinimalProfileSHA256,
		FontRefs:        []string{"builtin:arial"},
		ColorProfileRef: "ooxml-default",
	}
}

func RenderProjectionPPTX(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativePPTX || request.AccessibilityTarget != AccessibilityNotApplicable ||
		request.Direction != DirectionLTR || request.Rendering.ProfileRef != PPTXMinimalProfileRef ||
		request.Rendering.ProfileSHA256 != PPTXMinimalProfileSHA256 || len(request.Rendering.FontRefs) != 1 ||
		request.Rendering.FontRefs[0] != "builtin:arial" || request.Rendering.ColorProfileRef != "ooxml-default" ||
		len(request.Rendering.DependencyRefs) != 0 {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	slides := []string{projection.Title + "\n" + projection.Summary}
	for _, c := range selection.claims {
		slides = append(slides, c.ClaimID+"\n"+c.Statement+"\nPosture: "+c.Posture)
	}
	for _, asset := range selection.assets {
		slides = append(slides, asset.AssetID+"\nReference: "+asset.Ref+"\nMIME: "+asset.MIME+"\nSHA-256: "+asset.SHA256)
	}
	for _, citation := range selection.citations {
		slides = append(slides, citation.CitationID+"\nSource: "+citation.SourceRef+"\nLocator: "+citation.Locator+"\nSHA-256: "+citation.SHA256)
	}
	if len(request.OmissionRefs) > 0 {
		slides = append(slides, "Explicit omissions\n"+strings.Join(request.OmissionRefs, "\n"))
	}
	files := pptxFiles(slides, request.Locale)
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	entries := make([]ArchiveEntry, 0, len(files))
	for _, f := range files {
		h := &zip.FileHeader{Name: f.path, Method: zip.Store}
		h.SetMode(0644)
		h.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		w, e := zw.CreateHeader(h)
		if e != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		if _, e = w.Write([]byte(f.body)); e != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		sum := sha256.Sum256([]byte(f.body))
		entries = append(entries, ArchiveEntry{Path: f.path, SHA256: hex.EncodeToString(sum[:]), MIME: f.mime, Bytes: uint64(len(f.body))})
	}
	if err := zw.Close(); err != nil {
		return RenderedDerivative{}, ErrDerivativeUnsafeArchive
	}
	return buildRenderedDerivativeWithArchive(request, out.Bytes(), "pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation", renderer, matrix, licenses, receiptRef, createdAt, ArchiveSafe, entries)
}

func pptxFiles(slides []string, locale string) []pptxFile {
	var overrides strings.Builder
	var ids strings.Builder
	var rels strings.Builder
	for i := range slides {
		n := i + 1
		fmt.Fprintf(&overrides, `<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, n)
		fmt.Fprintf(&ids, `<p:sldId id="%d" r:id="rId%d"/>`, 255+n, n)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, n, n)
	}
	files := []pptxFile{{"[Content_Types].xml", "application/xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>` + overrides.String() + `</Types>`}, {"_rels/.rels", "application/vnd.openxmlformats-package.relationships+xml", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`}, {"ppt/presentation.xml", "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml", `<?xml version="1.0" encoding="UTF-8"?><p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldIdLst>` + ids.String() + `</p:sldIdLst><p:sldSz cx="12192000" cy="6858000"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`}, {"ppt/_rels/presentation.xml.rels", "application/vnd.openxmlformats-package.relationships+xml", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rels.String() + `</Relationships>`}}
	for i, text := range slides {
		parts := strings.SplitN(text, "\n", 2)
		title := parts[0]
		body := ""
		if len(parts) > 1 {
			body = parts[1]
		}
		xml := `<?xml version="1.0" encoding="UTF-8"?><p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/><p:sp><p:nvSpPr><p:cNvPr id="2" name="Evidence"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr/><p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr lang="` + html.EscapeString(locale) + `" sz="2800" b="1"><a:latin typeface="Arial"/></a:rPr><a:t>` + html.EscapeString(title) + `</a:t></a:r></a:p><a:p><a:r><a:rPr lang="` + html.EscapeString(locale) + `" sz="1800"><a:latin typeface="Arial"/></a:rPr><a:t>` + html.EscapeString(body) + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`
		files = append(files, pptxFile{fmt.Sprintf("ppt/slides/slide%d.xml", i+1), "application/vnd.openxmlformats-officedocument.presentationml.slide+xml", xml})
	}
	return files
}
