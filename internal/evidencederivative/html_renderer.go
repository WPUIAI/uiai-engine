package evidencederivative

import (
	"html"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const HTMLSystemUIProfileRef = "rendering:html-system-ui-srgb-v1"
const HTMLSystemUIProfileSHA256 = "3d7990731013c334923fd6ac668735af471bfd9c8550149ec5a8bc62c8ec5e4b"

func HTMLSystemUIRenderingProfile() RenderingProfile {
	return RenderingProfile{
		ProfileRef:      HTMLSystemUIProfileRef,
		ProfileSHA256:   HTMLSystemUIProfileSHA256,
		FontRefs:        []string{"builtin:system-ui"},
		ColorProfileRef: "css-srgb",
	}
}

func RenderProjectionHTML(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeHTML && request.DerivativeType != DerivativeEmailHTML && request.DerivativeType != DerivativePrint && request.DerivativeType != DerivativeHTMLSlides {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	if request.Rendering.ProfileRef != HTMLSystemUIProfileRef || request.Rendering.ProfileSHA256 != HTMLSystemUIProfileSHA256 ||
		len(request.Rendering.FontRefs) != 1 || request.Rendering.FontRefs[0] != "builtin:system-ui" ||
		request.Rendering.ColorProfileRef != "css-srgb" || len(request.Rendering.DependencyRefs) != 0 {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	var body strings.Builder
	body.WriteString("<!doctype html>\n<html lang=\"")
	body.WriteString(htmlText(request.Locale))
	body.WriteString("\" dir=\"")
	body.WriteString(htmlText(string(request.Direction)))
	body.WriteString("\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">\n<meta name=\"referrer\" content=\"no-referrer\">\n")
	body.WriteString("<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; base-uri 'none'; form-action 'none'\">\n")
	if request.DerivativeType == DerivativePrint {
		body.WriteString("<style>@page{size:auto;margin:18mm}*{print-color-adjust:exact}section,article{break-inside:avoid}nav,button{display:none!important}</style>\n")
	} else if request.DerivativeType == DerivativeHTMLSlides {
		body.WriteString("<style>main>header,main>section{box-sizing:border-box;min-height:100vh;padding:6vh 7vw;break-after:page}main{max-width:none;padding:0}main>section{margin:0;border:0}</style>\n")
	}
	body.WriteString("<title>")
	body.WriteString(htmlText(projection.Title))
	body.WriteString("</title>\n<style>body{font-family:system-ui,sans-serif;line-height:1.5;margin:0;color:#171717;background:#fff}main{max-width:72rem;margin:auto;padding:2rem}h1,h2,h3{line-height:1.2}section{border-top:1px solid #d4d4d4;margin-top:1.5rem;padding-top:1rem}.posture{display:grid;grid-template-columns:max-content 1fr;gap:.25rem 1rem}.record{margin:1rem 0;padding:.75rem;border:1px solid #e5e5e5;border-radius:.4rem}code{overflow-wrap:anywhere}ul{padding-left:1.25rem}</style>\n</head>\n<body>\n<main>\n<header><h1>")
	body.WriteString(htmlText(projection.Title))
	body.WriteString("</h1><p>")
	body.WriteString(htmlText(projection.Summary))
	body.WriteString("</p></header>\n<section aria-labelledby=\"posture-heading\"><h2 id=\"posture-heading\">Evidence posture</h2><dl class=\"posture\">")
	htmlDefinition(&body, "Availability", string(projection.Availability))
	htmlDefinition(&body, "Access", string(projection.Access))
	htmlDefinition(&body, "Redaction", string(projection.Redaction.State))
	htmlDefinition(&body, "Freshness observed", projection.FreshnessObservedAt.UTC().Format(time.RFC3339Nano))
	body.WriteString("</dl></section>\n")
	if len(selection.claims) > 0 {
		body.WriteString("<section aria-labelledby=\"claims-heading\"><h2 id=\"claims-heading\">Claims</h2>\n")
		for _, claim := range selection.claims {
			body.WriteString("<article class=\"record\"><h3>")
			body.WriteString(htmlText(claim.ClaimID))
			body.WriteString("</h3><p>")
			body.WriteString(htmlText(claim.Statement))
			body.WriteString("</p><p><strong>Posture:</strong> ")
			body.WriteString(htmlText(claim.Posture))
			body.WriteString("</p>")
			citations := intersection(claim.CitationIDs, selection.citationIDs)
			if len(citations) > 0 {
				body.WriteString("<p><strong>Citations:</strong> ")
				body.WriteString(htmlText(strings.Join(citations, ", ")))
				body.WriteString("</p>")
			}
			body.WriteString("</article>\n")
		}
		body.WriteString("</section>\n")
	}
	if len(selection.assets) > 0 {
		body.WriteString("<section aria-labelledby=\"assets-heading\"><h2 id=\"assets-heading\">Assets</h2><ul>\n")
		for _, asset := range selection.assets {
			body.WriteString("<li class=\"record\"><strong>")
			body.WriteString(htmlText(asset.AssetID))
			body.WriteString("</strong><br>Reference: <code>")
			body.WriteString(htmlText(asset.Ref))
			body.WriteString("</code><br>MIME: ")
			body.WriteString(htmlText(asset.MIME))
			body.WriteString("<br>SHA-256: <code>")
			body.WriteString(htmlText(asset.SHA256))
			body.WriteString("</code></li>\n")
		}
		body.WriteString("</ul></section>\n")
	}
	if len(selection.citations) > 0 {
		body.WriteString("<section aria-labelledby=\"citations-heading\"><h2 id=\"citations-heading\">Citations</h2><ul>\n")
		for _, citation := range selection.citations {
			body.WriteString("<li class=\"record\"><strong>")
			body.WriteString(htmlText(citation.CitationID))
			body.WriteString("</strong><br>Source: <code>")
			body.WriteString(htmlText(citation.SourceRef))
			body.WriteString("</code><br>Locator: <code>")
			body.WriteString(htmlText(citation.Locator))
			body.WriteString("</code><br>SHA-256: <code>")
			body.WriteString(htmlText(citation.SHA256))
			body.WriteString("</code></li>\n")
		}
		body.WriteString("</ul></section>\n")
	}
	if len(request.OmissionRefs) > 0 {
		body.WriteString("<section aria-labelledby=\"omissions-heading\"><h2 id=\"omissions-heading\">Explicit omissions</h2><ul>")
		for _, ref := range request.OmissionRefs {
			body.WriteString("<li>")
			body.WriteString(htmlText(ref))
			body.WriteString("</li>")
		}
		body.WriteString("</ul></section>\n")
	}
	body.WriteString("</main>\n</body>\n</html>\n")
	return buildRenderedDerivative(request, []byte(body.String()), "html", "text/html; charset=utf-8", renderer, matrix, licenses, receiptRef, createdAt)
}

func htmlDefinition(body *strings.Builder, term, value string) {
	body.WriteString("<dt>")
	body.WriteString(htmlText(term))
	body.WriteString("</dt><dd>")
	body.WriteString(htmlText(value))
	body.WriteString("</dd>")
}

func htmlText(value string) string {
	return html.EscapeString(plainText(value))
}
