package evidencederivative

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const pdfLinesPerPage = 48

func RenderProjectionPDF(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativePDF && request.DerivativeType != DerivativePresentationPDF {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	if request.AccessibilityTarget == AccessibilityPDFUA1 || request.AccessibilityTarget == AccessibilityPDFUA2 {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	lines, err := pdfTextLines(renderProjectionPlainText(projection, selection, request.OmissionRefs))
	if err != nil {
		return RenderedDerivative{}, err
	}
	if request.DerivativeType == DerivativePresentationPDF {
		lines = presentationLines(lines)
	}
	output, err := buildPDF(lines)
	if err != nil {
		return RenderedDerivative{}, err
	}
	return buildRenderedDerivative(request, output, "pdf", "application/pdf", renderer, matrix, licenses, receiptRef, createdAt)
}

func pdfTextLines(input []byte) ([]string, error) {
	if !utf8.Valid(input) {
		return nil, ErrDerivativeContractInvalid
	}
	var lines []string
	for _, raw := range strings.Split(strings.ReplaceAll(string(input), "\r\n", "\n"), "\n") {
		var current strings.Builder
		for _, r := range raw {
			if r < 32 || r > 126 {
				return nil, ErrDerivativeContractInvalid
			}
			current.WriteRune(r)
			if current.Len() == 86 {
				lines = append(lines, current.String())
				current.Reset()
			}
		}
		lines = append(lines, current.String())
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines, nil
}

func presentationLines(lines []string) []string {
	out := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		out = append(out, line)
		if strings.TrimSpace(line) == "" && len(out)%pdfLinesPerPage != 0 {
			for len(out)%pdfLinesPerPage != 0 {
				out = append(out, "")
			}
		}
	}
	return out
}

func buildPDF(lines []string) ([]byte, error) {
	pageCount := (len(lines) + pdfLinesPerPage - 1) / pdfLinesPerPage
	objectCount := 3 + pageCount*2
	objects := make([][]byte, objectCount+1)
	objects[1] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, pageCount)
	for page := 0; page < pageCount; page++ {
		pageID := 4 + page*2
		contentID := pageID + 1
		kids[page] = fmt.Sprintf("%d 0 R", pageID)
		objects[pageID] = []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentID))
		start := page * pdfLinesPerPage
		end := start + pdfLinesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		var stream strings.Builder
		stream.WriteString("BT\n/F1 10 Tf\n50 742 Td\n14 TL\n")
		for _, line := range lines[start:end] {
			stream.WriteByte('(')
			stream.WriteString(pdfEscape(line))
			stream.WriteString(") Tj\nT*\n")
		}
		stream.WriteString("ET\n")
		body := stream.String()
		objects[contentID] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(body), body))
	}
	objects[2] = []byte(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", pageCount, strings.Join(kids, " ")))
	objects[3] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")

	var output bytes.Buffer
	output.WriteString("%PDF-1.7\n% deterministic evidence derivative\n")
	offsets := make([]int, objectCount+1)
	for id := 1; id <= objectCount; id++ {
		offsets[id] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", id)
		output.Write(objects[id])
		output.WriteString("\nendobj\n")
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", objectCount+1)
	for id := 1; id <= objectCount; id++ {
		output.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[id]))
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", objectCount+1, xref)
	return output.Bytes(), nil
}

func pdfEscape(value string) string {
	var out strings.Builder
	for _, r := range value {
		switch r {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteRune(r)
		default:
			if r < 32 || r > 126 {
				out.WriteString("?")
			} else {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}
