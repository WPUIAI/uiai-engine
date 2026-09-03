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
	return RenderingProfile{ProfileRef: PPTXMinimalProfileRef, ProfileSHA256: PPTXMinimalProfileSHA256, FontRefs: []string{"builtin:arial"}, ColorProfileRef: "ooxml-default"}
}

func RenderProjectionPPTX(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativePPTX || request.AccessibilityTarget != AccessibilityNotApplicable ||
		request.Direction != DirectionLTR || request.Rendering.ProfileRef != PPTXMinimalProfileRef ||
		request.Rendering.ProfileSHA256 != PPTXMinimalProfileSHA256 || len(request.Rendering.FontRefs) != 1 ||
		request.Rendering.FontRefs[0] != "builtin:arial" || request.Rendering.ColorProfileRef != "ooxml-default" || len(request.Rendering.DependencyRefs) != 0 {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	licenseSet, err := normalizedLicenses(licenses, request.AssetRefs)
	if err != nil {
		return RenderedDerivative{}, err
	}
	slides := []string{projection.Title + "\n" + projection.Summary}
	for _, claim := range selection.claims {
		slides = append(slides, claim.ClaimID+"\n"+claim.Statement+"\nPosture: "+claim.Posture)
	}
	for _, asset := range selection.assets {
		slides = append(slides, asset.AssetID+"\nReference: "+asset.Ref+"\nMIME: "+asset.MIME+"\nSHA-256: "+asset.SHA256)
	}
	for _, citation := range selection.citations {
		slides = append(slides, citation.CitationID+"\nSource: "+citation.SourceRef+"\nLocator: "+citation.Locator+"\nSHA-256: "+citation.SHA256)
	}
	for _, license := range licenseSet {
		body := "License: " + license.LicenseRef + "\nEvidence: " + license.EvidenceRef
		if license.AttributionRequired {
			body += "\nAttribution: " + license.AttributionRef
		}
		slides = append(slides, license.AssetRef+" license and attribution\n"+body)
	}
	if len(request.OmissionRefs) > 0 {
		slides = append(slides, "Explicit omissions\n"+strings.Join(request.OmissionRefs, "\n"))
	}
	files := pptxFiles(slides, request.Locale)
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	entries := make([]ArchiveEntry, 0, len(files))
	for _, file := range files {
		header := &zip.FileHeader{Name: file.path, Method: zip.Store}
		header.SetMode(0o644)
		header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		destination, err := writer.CreateHeader(header)
		if err != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		if _, err = destination.Write([]byte(file.body)); err != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		digest := sha256.Sum256([]byte(file.body))
		entries = append(entries, ArchiveEntry{Path: file.path, SHA256: hex.EncodeToString(digest[:]), MIME: file.mime, Bytes: uint64(len(file.body))})
	}
	if err := writer.Close(); err != nil {
		return RenderedDerivative{}, ErrDerivativeUnsafeArchive
	}
	return buildRenderedDerivativeWithArchive(request, output.Bytes(), "pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation", renderer, matrix, licenseSet, receiptRef, createdAt, ArchiveSafe, entries)
}

func pptxFiles(slides []string, locale string) []pptxFile {
	files := []pptxFile{
		{"[Content_Types].xml", "application/xml", pptxContentTypes(len(slides))},
		{"_rels/.rels", "application/vnd.openxmlformats-package.relationships+xml", pptxRootRelationships},
		{"ppt/presentation.xml", "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml", pptxPresentation(len(slides))},
		{"ppt/_rels/presentation.xml.rels", "application/vnd.openxmlformats-package.relationships+xml", pptxPresentationRelationships(len(slides))},
		{"ppt/slideMasters/slideMaster1.xml", "application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml", pptxSlideMaster},
		{"ppt/slideMasters/_rels/slideMaster1.xml.rels", "application/vnd.openxmlformats-package.relationships+xml", pptxMasterRelationships},
		{"ppt/slideLayouts/slideLayout1.xml", "application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml", pptxSlideLayout},
		{"ppt/slideLayouts/_rels/slideLayout1.xml.rels", "application/vnd.openxmlformats-package.relationships+xml", pptxLayoutRelationships},
		{"ppt/theme/theme1.xml", "application/vnd.openxmlformats-officedocument.theme+xml", pptxTheme},
	}
	for index, text := range slides {
		number := index + 1
		files = append(files,
			pptxFile{fmt.Sprintf("ppt/slides/slide%d.xml", number), "application/vnd.openxmlformats-officedocument.presentationml.slide+xml", pptxSlide(text, locale)},
			pptxFile{fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", number), "application/vnd.openxmlformats-package.relationships+xml", pptxSlideRelationships},
		)
	}
	return files
}

func pptxContentTypes(slideCount int) string {
	var slides strings.Builder
	for number := 1; number <= slideCount; number++ {
		fmt.Fprintf(&slides, `<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, number)
	}
	return `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/><Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/><Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>` + slides.String() + `</Types>`
}

func pptxPresentation(slideCount int) string {
	var ids strings.Builder
	for number := 1; number <= slideCount; number++ {
		fmt.Fprintf(&ids, `<p:sldId id="%d" r:id="rId%d"/>`, 255+number, number)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId%d"/></p:sldMasterIdLst><p:sldIdLst>%s</p:sldIdLst><p:sldSz cx="12192000" cy="6858000" type="screen16x9"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`, slideCount+1, ids.String())
}

func pptxPresentationRelationships(slideCount int) string {
	var relationships strings.Builder
	for number := 1; number <= slideCount; number++ {
		fmt.Fprintf(&relationships, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, number, number)
	}
	fmt.Fprintf(&relationships, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`, slideCount+1)
	return `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + relationships.String() + `</Relationships>`
}

func pptxSlide(text, locale string) string {
	parts := strings.SplitN(text, "\n", 2)
	title, body := parts[0], ""
	if len(parts) > 1 {
		body = parts[1]
	}
	language := html.EscapeString(plainText(locale))
	return `<?xml version="1.0" encoding="UTF-8"?><p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/><p:sp><p:nvSpPr><p:cNvPr id="2" name="Evidence"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr/><p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr lang="` + language + `" sz="2800" b="1"><a:latin typeface="Arial"/></a:rPr><a:t>` + pptxText(title) + `</a:t></a:r></a:p><a:p><a:r><a:rPr lang="` + language + `" sz="1800"><a:latin typeface="Arial"/></a:rPr><a:t>` + pptxText(body) + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`
}

func pptxText(value string) string {
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = plainText(line)
	}
	return html.EscapeString(strings.Join(lines, "\n"))
}

const pptxRootRelationships = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`
const pptxSlideRelationships = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/></Relationships>`
const pptxMasterRelationships = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/></Relationships>`
const pptxLayoutRelationships = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/></Relationships>`
const pptxSlideLayout = `<?xml version="1.0" encoding="UTF-8"?><p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="blank" preserve="1"><p:cSld name="Evidence Blank"><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/></p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sldLayout>`
const pptxSlideMaster = `<?xml version="1.0" encoding="UTF-8"?><p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld name="Evidence Master"><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/></p:spTree></p:cSld><p:clrMap accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" bg1="lt1" bg2="lt2" folHlink="folHlink" hlink="hlink" tx1="dk1" tx2="dk2"/><p:sldLayoutIdLst><p:sldLayoutId id="1" r:id="rId1"/></p:sldLayoutIdLst><p:txStyles><p:titleStyle/><p:bodyStyle/><p:otherStyle/></p:txStyles></p:sldMaster>`
const pptxTheme = `<?xml version="1.0" encoding="UTF-8"?><a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Evidence"><a:themeElements><a:clrScheme name="Evidence"><a:dk1><a:srgbClr val="000000"/></a:dk1><a:lt1><a:srgbClr val="FFFFFF"/></a:lt1><a:dk2><a:srgbClr val="1F2937"/></a:dk2><a:lt2><a:srgbClr val="F3F4F6"/></a:lt2><a:accent1><a:srgbClr val="2563EB"/></a:accent1><a:accent2><a:srgbClr val="7C3AED"/></a:accent2><a:accent3><a:srgbClr val="059669"/></a:accent3><a:accent4><a:srgbClr val="D97706"/></a:accent4><a:accent5><a:srgbClr val="DC2626"/></a:accent5><a:accent6><a:srgbClr val="0891B2"/></a:accent6><a:hlink><a:srgbClr val="0000FF"/></a:hlink><a:folHlink><a:srgbClr val="800080"/></a:folHlink></a:clrScheme><a:fontScheme name="Arial"><a:majorFont><a:latin typeface="Arial"/></a:majorFont><a:minorFont><a:latin typeface="Arial"/></a:minorFont></a:fontScheme><a:fmtScheme name="Evidence"><a:fillStyleLst/><a:lnStyleLst/><a:effectStyleLst/><a:bgFillStyleLst/></a:fmtScheme></a:themeElements></a:theme>`
