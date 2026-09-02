package evidencederivative

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func RenderProjectionRichText(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeRichText {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	plain := renderProjectionPlainText(projection, selection, request.OmissionRefs)
	var out strings.Builder
	out.WriteString("{\\rtf1\\ansi\\deff0{\\fonttbl{\\f0 Arial;}}\\f0\\fs22 ")
	for len(plain) > 0 {
		r, size := utf8.DecodeRune(plain)
		plain = plain[size:]
		switch r {
		case '\\', '{', '}':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\n':
			out.WriteString("\\par\n")
		default:
			if r >= 32 && r <= 126 {
				out.WriteRune(r)
			} else {
				n := int32(r)
				if n > 32767 {
					n -= 65536
				}
				fmt.Fprintf(&out, "\\u%d?", n)
			}
		}
	}
	out.WriteString("}")
	return buildRenderedDerivative(request, []byte(out.String()), "rtf", "application/rtf", renderer, matrix, licenses, receiptRef, createdAt)
}
