package evidencederivative

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func RenderProjectionMarkdown(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if (request.DerivativeType != DerivativeMarkdown && request.DerivativeType != DerivativeEmailText) ||
		!renderingProfileMatches(request.Rendering, PortableDataRenderingProfile()) {
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
	if request.DerivativeType == DerivativeEmailText {
		return buildRenderedDerivative(request, renderProjectionPlainText(projection, selection, request.OmissionRefs, licenseSet), "txt", "text/plain; charset=utf-8", renderer, matrix, licenses, receiptRef, createdAt)
	}
	var body strings.Builder
	body.WriteString("# ")
	body.WriteString(markdownText(projection.Title))
	body.WriteString("\n\n")
	body.WriteString(markdownText(projection.Summary))
	body.WriteString("\n\n")
	body.WriteString("## Evidence posture\n\n")
	fmt.Fprintf(&body, "- Availability: %s\n- Access: %s\n- Redaction: %s\n- Freshness observed: %s\n", projection.Availability, projection.Access, projection.Redaction.State, projection.FreshnessObservedAt.UTC().Format(time.RFC3339Nano))
	if len(selection.claims) > 0 {
		body.WriteString("\n## Claims\n")
		for _, claim := range selection.claims {
			fmt.Fprintf(&body, "\n### %s\n\n%s\n\n- Posture: %s\n", markdownText(claim.ClaimID), markdownText(claim.Statement), markdownText(claim.Posture))
			citations := intersection(claim.CitationIDs, selection.citationIDs)
			if len(citations) > 0 {
				fmt.Fprintf(&body, "- Citations: %s\n", markdownText(strings.Join(citations, ", ")))
			}
		}
	}
	if len(selection.assets) > 0 {
		body.WriteString("\n## Assets\n")
		for _, asset := range selection.assets {
			fmt.Fprintf(&body, "\n- **%s** — `%s`; %s; SHA-256 `%s`\n", markdownText(asset.AssetID), markdownText(asset.Ref), markdownText(asset.MIME), asset.SHA256)
		}
	}
	if len(selection.citations) > 0 {
		body.WriteString("\n## Citations\n")
		for _, citation := range selection.citations {
			fmt.Fprintf(&body, "\n- **%s** — `%s`; locator `%s`; SHA-256 `%s`\n", markdownText(citation.CitationID), markdownText(citation.SourceRef), markdownText(citation.Locator), citation.SHA256)
		}
	}
	if len(licenseSet) > 0 {
		body.WriteString("\n## Licenses and attribution\n")
		for _, license := range licenseSet {
			fmt.Fprintf(&body, "\n- **%s** — license `%s`; evidence `%s`", markdownText(license.AssetRef), markdownText(license.LicenseRef), markdownText(license.EvidenceRef))
			if license.AttributionRequired {
				fmt.Fprintf(&body, "; attribution `%s`", markdownText(license.AttributionRef))
			}
			body.WriteString("\n")
		}
	}
	if len(request.OmissionRefs) > 0 {
		body.WriteString("\n## Explicit omissions\n")
		for _, ref := range request.OmissionRefs {
			fmt.Fprintf(&body, "\n- %s\n", markdownText(ref))
		}
	}
	return buildRenderedDerivative(request, []byte(body.String()), "md", "text/markdown; charset=utf-8", renderer, matrix, licenses, receiptRef, createdAt)
}

func renderProjectionPlainText(projection evidencepwa.Projection, selection projectionSelection, omissionRefs []string, licenses []LicenseAttestation) []byte {
	var body strings.Builder
	body.WriteString(plainText(projection.Title))
	body.WriteString("\n\n")
	body.WriteString(plainText(projection.Summary))
	body.WriteString("\n\nEVIDENCE POSTURE\n")
	fmt.Fprintf(&body, "Availability: %s\nAccess: %s\nRedaction: %s\nFreshness observed: %s\n", projection.Availability, projection.Access, projection.Redaction.State, projection.FreshnessObservedAt.UTC().Format(time.RFC3339Nano))
	if len(selection.claims) > 0 {
		body.WriteString("\nCLAIMS\n")
		for _, claim := range selection.claims {
			fmt.Fprintf(&body, "\n%s\n%s\nPosture: %s\n", plainText(claim.ClaimID), plainText(claim.Statement), plainText(claim.Posture))
			citations := intersection(claim.CitationIDs, selection.citationIDs)
			if len(citations) > 0 {
				fmt.Fprintf(&body, "Citations: %s\n", plainText(strings.Join(citations, ", ")))
			}
		}
	}
	if len(selection.assets) > 0 {
		body.WriteString("\nASSETS\n")
		for _, asset := range selection.assets {
			fmt.Fprintf(&body, "\n%s\nReference: %s\nMIME: %s\nSHA-256: %s\n", plainText(asset.AssetID), plainText(asset.Ref), plainText(asset.MIME), asset.SHA256)
		}
	}
	if len(selection.citations) > 0 {
		body.WriteString("\nCITATIONS\n")
		for _, citation := range selection.citations {
			fmt.Fprintf(&body, "\n%s\nSource: %s\nLocator: %s\nSHA-256: %s\n", plainText(citation.CitationID), plainText(citation.SourceRef), plainText(citation.Locator), citation.SHA256)
		}
	}
	if len(licenses) > 0 {
		body.WriteString("\nLICENSES AND ATTRIBUTION\n")
		for _, license := range licenses {
			fmt.Fprintf(&body, "%s\nLicense: %s\nEvidence: %s\n", plainText(license.AssetRef), plainText(license.LicenseRef), plainText(license.EvidenceRef))
			if license.AttributionRequired {
				fmt.Fprintf(&body, "Attribution: %s\n", plainText(license.AttributionRef))
			}
		}
	}
	if len(omissionRefs) > 0 {
		body.WriteString("\nEXPLICIT OMISSIONS\n")
		for _, ref := range omissionRefs {
			fmt.Fprintf(&body, "%s\n", plainText(ref))
		}
	}
	return []byte(body.String())
}

func RenderProjectionCSV(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeCSV || !renderingProfileMatches(request.Rendering, PortableDataRenderingProfile()) {
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
	var body bytes.Buffer
	writer := csv.NewWriter(&body)
	_ = writer.Write([]string{"record_type", "id", "content", "posture_or_mime", "citation_or_locator", "sha256"})
	for _, claim := range selection.claims {
		_ = writer.Write([]string{"claim", claim.ClaimID, claim.Statement, claim.Posture, strings.Join(intersection(claim.CitationIDs, selection.citationIDs), " "), ""})
	}
	for _, asset := range selection.assets {
		_ = writer.Write([]string{"asset", asset.AssetID, asset.Ref, asset.MIME, "", asset.SHA256})
	}
	for _, citation := range selection.citations {
		_ = writer.Write([]string{"citation", citation.CitationID, citation.SourceRef, "", citation.Locator, citation.SHA256})
	}
	for _, license := range licenseSet {
		attribution := ""
		if license.AttributionRequired {
			attribution = license.AttributionRef
		}
		_ = writer.Write([]string{"license", license.AssetRef, license.LicenseRef, attribution, license.EvidenceRef, license.LicenseSHA256})
	}
	for _, ref := range request.OmissionRefs {
		_ = writer.Write([]string{"omission", ref, "explicitly omitted", "", "", ""})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	return buildRenderedDerivative(request, body.Bytes(), "csv", "text/csv; charset=utf-8", renderer, matrix, licenses, receiptRef, createdAt)
}

type projectionSelection struct {
	claims      []evidencepwa.Claim
	assets      []evidencepwa.Asset
	citations   []evidencepwa.Citation
	citationIDs map[string]struct{}
}

func selectProjection(request DerivativeRequest, projection evidencepwa.Projection) (projectionSelection, error) {
	if err := ValidateRequest(request); err != nil {
		return projectionSelection{}, err
	}
	if err := evidencepwa.ValidateProjection(projection); err != nil {
		return projectionSelection{}, ErrProjectionMismatch
	}
	projectionDigest, err := evidencepwa.DigestProjection(projection)
	if err != nil || request.ProjectionRef != projection.ProjectionID || request.ProjectionSHA256 != projectionDigest ||
		request.ArtifactRef != projection.Artifact.ArtifactRef || request.ArtifactSHA256 != projection.Artifact.ManifestSHA256 ||
		request.ArtifactRevision != projection.Artifact.Revision || request.Scope != projection.Artifact.Scope {
		return projectionSelection{}, ErrProjectionMismatch
	}
	if expected := DerivativeOmissionRefs(request, projection); !slices.Equal(request.OmissionRefs, expected) {
		return projectionSelection{}, ErrDerivativeSelectionIncomplete
	}
	claims := make(map[string]evidencepwa.Claim, len(projection.Claims))
	assets := make(map[string]evidencepwa.Asset, len(projection.Assets))
	citations := make(map[string]evidencepwa.Citation, len(projection.Citations))
	for _, claim := range projection.Claims {
		claims[claim.ClaimID] = claim
	}
	for _, asset := range projection.Assets {
		assets[asset.AssetID] = asset
	}
	for _, citation := range projection.Citations {
		citations[citation.CitationID] = citation
	}
	selection := projectionSelection{citationIDs: make(map[string]struct{}, len(request.CitationRefs))}
	for _, ref := range request.ClaimRefs {
		claim, found := claims[ref]
		if !found {
			return projectionSelection{}, ErrDerivativeSelectionIncomplete
		}
		selection.claims = append(selection.claims, claim)
	}
	for _, ref := range request.AssetRefs {
		asset, found := assets[ref]
		if !found {
			return projectionSelection{}, ErrDerivativeSelectionIncomplete
		}
		selection.assets = append(selection.assets, asset)
	}
	for _, ref := range request.CitationRefs {
		citation, found := citations[ref]
		if !found {
			return projectionSelection{}, ErrDerivativeSelectionIncomplete
		}
		selection.citations = append(selection.citations, citation)
		selection.citationIDs[ref] = struct{}{}
	}
	return selection, nil
}

func DerivativeOmissionRefs(request DerivativeRequest, projection evidencepwa.Projection) []string {
	selected := make(map[string]struct{}, len(request.ClaimRefs)+len(request.AssetRefs)+len(request.CitationRefs))
	for _, group := range [][]string{request.ClaimRefs, request.AssetRefs, request.CitationRefs} {
		for _, ref := range group {
			selected[ref] = struct{}{}
		}
	}
	all := make(map[string]struct{})
	for _, claim := range projection.Claims {
		all[claim.ClaimID] = struct{}{}
	}
	for _, asset := range projection.Assets {
		all[asset.AssetID] = struct{}{}
	}
	for _, citation := range projection.Citations {
		all[citation.CitationID] = struct{}{}
	}
	for _, entry := range projection.Timeline {
		all[entry.EntryID] = struct{}{}
	}
	for _, group := range [][]string{projection.InspectionRefs, projection.SecurityRefs, projection.CustodyRefs,
		projection.AttestationRefs, projection.TrustRefs, projection.OmissionRefs, projection.RelatedArtifactRefs, projection.ReceiptRefs} {
		for _, ref := range group {
			all[ref] = struct{}{}
		}
	}
	for _, warning := range projection.Warnings {
		all["warning:"+warning.Code] = struct{}{}
	}
	omissions := make([]string, 0, len(all))
	for ref := range all {
		if _, included := selected[ref]; !included {
			omissions = append(omissions, ref)
		}
	}
	sort.Strings(omissions)
	return omissions
}

func intersection(values []string, allowed map[string]struct{}) []string {
	selected := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			selected = append(selected, value)
		}
	}
	sort.Strings(selected)
	return selected
}

func plainText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func markdownText(value string) string {
	value = plainText(value)
	replacer := strings.NewReplacer("\\", "\\\\", "#", "\\#", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "\\<", ">", "\\>", "|", "\\|")
	return replacer.Replace(value)
}
