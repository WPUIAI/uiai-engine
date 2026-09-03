package evidencederivative

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const RTFArialProfileRef = "rendering:rtf-arial-v1"
const RTFArialProfileSHA256 = "26c4fe4dd749ccaf47d5d50e77ac207f340d705c34272c13abd9402af96bc12d"

func RTFArialRenderingProfile() RenderingProfile {
	return RenderingProfile{
		ProfileRef:      RTFArialProfileRef,
		ProfileSHA256:   RTFArialProfileSHA256,
		FontRefs:        []string{"builtin:arial"},
		ColorProfileRef: "rtf-default",
	}
}

func RenderProjectionRichText(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeRichText || request.Direction != DirectionLTR ||
		request.Rendering.ProfileRef != RTFArialProfileRef || request.Rendering.ProfileSHA256 != RTFArialProfileSHA256 ||
		len(request.Rendering.FontRefs) != 1 || request.Rendering.FontRefs[0] != "builtin:arial" ||
		request.Rendering.ColorProfileRef != "rtf-default" || len(request.Rendering.DependencyRefs) != 0 {
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
	plain := renderProjectionPlainText(projection, selection, request.OmissionRefs, licenseSet)
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
	return buildRenderedDerivative(request, []byte(out.String()), "rtf", "application/rtf", renderer, matrix, licenseSet, receiptRef, createdAt)
}
