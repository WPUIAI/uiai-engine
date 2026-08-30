package evidencederivative

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"reflect"
	"strings"
)

var (
	ErrDerivativeContractInvalid      = errors.New("evidence derivative contract invalid")
	ErrDerivativeScopeMismatch        = errors.New("evidence derivative scope mismatch")
	ErrDerivativeIdentityMismatch     = errors.New("evidence derivative identity mismatch")
	ErrDerivativeSelectionIncomplete  = errors.New("evidence derivative selection incomplete")
	ErrDerivativeUnsafeArchive        = errors.New("evidence derivative archive unsafe")
	ErrDerivativeLicenseMissing       = errors.New("evidence derivative license missing")
	ErrDeliveryOutcomeUnknown         = errors.New("evidence derivative delivery outcome unknown")
	ErrDeliveryReconciliationRequired = errors.New("evidence derivative delivery reconciliation required")
)

func DigestRequest(value DerivativeRequest) (string, error)   { return digest(value) }
func DigestManifest(value DerivativeManifest) (string, error) { return digest(value) }
func DigestDelivery(value DeliveryReceipt) (string, error)    { return digest(value) }

func DerivativeID(request DerivativeRequest, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation) (string, error) {
	value := struct {
		Request  DerivativeRequest    `json:"request"`
		Renderer RendererIdentity     `json:"renderer"`
		Matrix   ViewerMatrix         `json:"viewer_matrix"`
		Licenses []LicenseAttestation `json:"licenses"`
	}{request, renderer, matrix, licenses}
	digest, err := digest(value)
	if err != nil {
		return "", err
	}
	return "derivative:" + digest, nil
}

func ValidateRequest(request DerivativeRequest) error {
	if request.Schema != RequestSchema || blank(request.RequestID) || !validScope(request.Scope) || blank(request.ArtifactRef) ||
		!validSHA256(request.ArtifactSHA256) || request.ArtifactRevision == 0 || blank(request.ProjectionRef) ||
		!validSHA256(request.ProjectionSHA256) || !validDerivativeType(request.DerivativeType) ||
		len(request.RequiredEvidenceRefs) == 0 || len(request.RequiredEvidenceRefs) > MaxRefs ||
		!validLists(request.ClaimRefs, request.AssetRefs, request.CitationRefs, request.OmissionRefs, request.RequiredEvidenceRefs) ||
		blank(request.Rendering.ProfileRef) || !validSHA256(request.Rendering.ProfileSHA256) ||
		len(request.Rendering.FontRefs) == 0 || hasBlankOrDuplicate(request.Rendering.FontRefs) ||
		blank(request.Rendering.ColorProfileRef) || hasBlankOrDuplicate(request.Rendering.DependencyRefs) ||
		blank(request.Locale) || !validDirection(request.Direction) || !validAccessibility(request.AccessibilityTarget) ||
		blank(request.LicensePolicyRef) || !validSHA256(request.LicensePolicySHA256) || blank(request.IdempotencyKey) {
		return ErrDerivativeContractInvalid
	}
	selected := append(append(append([]string{}, request.ClaimRefs...), request.AssetRefs...), request.CitationRefs...)
	selected = append(selected, request.OmissionRefs...)
	if !subset(request.RequiredEvidenceRefs, selected) {
		return ErrDerivativeSelectionIncomplete
	}
	for _, ref := range request.Rendering.DependencyRefs {
		if !relativeRef(ref) {
			return ErrDerivativeContractInvalid
		}
	}
	return sizeOK(request)
}

func ValidateManifest(manifest DerivativeManifest, request DerivativeRequest) error {
	if err := ValidateRequest(request); err != nil {
		return err
	}
	if manifest.Schema != ManifestSchema || blank(manifest.DerivativeID) || blank(manifest.RequestRef) ||
		!validSHA256(manifest.RequestSHA256) || blank(manifest.OutputRef) || !relativeRef(manifest.OutputRef) ||
		!validSHA256(manifest.OutputSHA256) || manifest.OutputBytes == 0 || blank(manifest.OutputMIME) ||
		!validRenderer(manifest.Renderer) || !validAccessibility(manifest.AccessibilityTarget) ||
		!validConformance(manifest.AccessibilityPosture) || !validArchive(manifest.ArchivePosture) ||
		blank(manifest.ReceiptRef) || manifest.CreatedAt.IsZero() || !validLists(manifest.TranscriptRefs, manifest.KeyframeRefs,
		manifest.AccessibilityEvidenceRefs, manifest.OmissionRefs, manifest.WarningRefs) {
		return ErrDerivativeContractInvalid
	}
	requestDigest, _ := DigestRequest(request)
	if manifest.RequestRef != request.RequestID || manifest.RequestSHA256 != requestDigest || manifest.ArtifactRef != request.ArtifactRef ||
		manifest.ArtifactSHA256 != request.ArtifactSHA256 || manifest.ProjectionRef != request.ProjectionRef ||
		manifest.ProjectionSHA256 != request.ProjectionSHA256 || !reflect.DeepEqual(manifest.Rendering, request.Rendering) ||
		manifest.AccessibilityTarget != request.AccessibilityTarget {
		return ErrDerivativeScopeMismatch
	}
	if err := validateViewerMatrix(manifest.ViewerMatrix); err != nil {
		return err
	}
	if err := validateLicenses(manifest.Licenses, request.AssetRefs); err != nil {
		return err
	}
	if err := validateArchive(manifest.ArchivePosture, manifest.ArchiveEntries); err != nil {
		return err
	}
	expectedID, err := DerivativeID(request, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses)
	if err != nil {
		return err
	}
	if manifest.DerivativeID != expectedID {
		return ErrDerivativeIdentityMismatch
	}
	if manifest.AccessibilityPosture == ConformanceVerified && len(manifest.AccessibilityEvidenceRefs) == 0 {
		return ErrDerivativeContractInvalid
	}
	if request.DerivativeType == DerivativePDF || request.DerivativeType == DerivativePresentationPDF {
		if blank(manifest.PDFAProfile) && blank(manifest.PDFUAProfile) && manifest.AccessibilityPosture == ConformanceVerified {
			return ErrDerivativeContractInvalid
		}
		if (request.AccessibilityTarget == AccessibilityPDFUA1 || request.AccessibilityTarget == AccessibilityPDFUA2) && blank(manifest.PDFUAProfile) {
			return ErrDerivativeContractInvalid
		}
	} else if !blank(manifest.PDFAProfile) || !blank(manifest.PDFUAProfile) {
		return ErrDerivativeContractInvalid
	}
	if (request.DerivativeType == DerivativeArchive || request.DerivativeType == DerivativePPTX) && manifest.ArchivePosture != ArchiveSafe {
		return ErrDerivativeUnsafeArchive
	}
	if request.DerivativeType != DerivativeArchive && request.DerivativeType != DerivativePPTX && manifest.ArchivePosture != ArchiveNotApplicable {
		return ErrDerivativeContractInvalid
	}
	return sizeOK(manifest)
}

func ValidateDelivery(receipt DeliveryReceipt) error {
	if receipt.Schema != DeliverySchema || blank(receipt.DeliveryID) || blank(receipt.DerivativeRef) ||
		!validSHA256(receipt.DerivativeSHA256) || blank(receipt.DestinationRef) || blank(receipt.IdempotencyKey) ||
		!validDelivery(receipt.State) || len(receipt.EvidenceRefs) > MaxRefs || hasBlankOrDuplicate(receipt.EvidenceRefs) ||
		len(receipt.ReconciliationRefs) > MaxRefs || hasBlankOrDuplicate(receipt.ReconciliationRefs) || receipt.ObservedAt.IsZero() {
		return ErrDerivativeContractInvalid
	}
	if receipt.State == DeliveryAccepted && (receipt.AcceptedAt == nil || receipt.DeliveredAt != nil || blank(receipt.ProviderReceiptRef)) {
		return ErrDerivativeContractInvalid
	}
	if receipt.State == DeliveryDelivered && (receipt.AcceptedAt == nil || receipt.DeliveredAt == nil || blank(receipt.ProviderReceiptRef) ||
		len(receipt.EvidenceRefs) == 0 || receipt.DeliveredAt.Before(*receipt.AcceptedAt) || receipt.ObservedAt.Before(*receipt.DeliveredAt)) {
		return ErrDerivativeContractInvalid
	}
	if receipt.State == DeliveryOutcomeUnknown {
		if receipt.RetryPermitted {
			return ErrDeliveryReconciliationRequired
		}
		return ErrDeliveryOutcomeUnknown
	}
	if receipt.RetryPermitted && len(receipt.ReconciliationRefs) == 0 {
		return ErrDeliveryReconciliationRequired
	}
	return sizeOK(receipt)
}

func validateViewerMatrix(matrix ViewerMatrix) error {
	if matrix.Schema != ViewerMatrixSchema || blank(matrix.MatrixRef) || len(matrix.Entries) == 0 || len(matrix.Entries) > MaxEntries {
		return ErrDerivativeContractInvalid
	}
	seen := map[string]struct{}{}
	for _, entry := range matrix.Entries {
		key := entry.Client + "\x00" + entry.Version
		if blank(entry.Client) || blank(entry.Version) || !validViewer(entry.Status) || hasBlankOrDuplicate(entry.EvidenceRefs) || !addUnique(seen, key) ||
			((entry.Status == ViewerSupported || entry.Status == ViewerDegraded) && len(entry.EvidenceRefs) == 0) {
			return ErrDerivativeContractInvalid
		}
	}
	return nil
}
func validateLicenses(licenses []LicenseAttestation, assets []string) error {
	byAsset := map[string]struct{}{}
	for _, license := range licenses {
		if license.Schema != LicenseSchema || blank(license.AssetRef) || blank(license.LicenseRef) || !validSHA256(license.LicenseSHA256) ||
			!license.DerivativePermitted || blank(license.EvidenceRef) || (license.AttributionRequired && blank(license.AttributionRef)) || !addUnique(byAsset, license.AssetRef) {
			return ErrDerivativeLicenseMissing
		}
	}
	if !subset(assets, keys(byAsset)) {
		return ErrDerivativeLicenseMissing
	}
	return nil
}
func validateArchive(posture ArchivePosture, entries []ArchiveEntry) error {
	if posture == ArchiveNotApplicable && len(entries) != 0 {
		return ErrDerivativeUnsafeArchive
	}
	if posture == ArchiveSafe && len(entries) == 0 {
		return ErrDerivativeUnsafeArchive
	}
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if !relativeRef(entry.Path) || entry.Link || !validSHA256(entry.SHA256) || blank(entry.MIME) || entry.Bytes == 0 || !addUnique(seen, entry.Path) {
			return ErrDerivativeUnsafeArchive
		}
		if strings.Contains(entry.MIME, "x-executable") || strings.Contains(entry.MIME, "x-sh") {
			return ErrDerivativeUnsafeArchive
		}
	}
	return nil
}
func validRenderer(value RendererIdentity) bool {
	return !blank(value.RendererRef) && !blank(value.Version) && validSHA256(value.BinarySHA256)
}
func relativeRef(value string) bool {
	if blank(value) || strings.ContainsAny(value, "\\\r\n\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	clean := path.Clean(parsed.Path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}
func validLists(groups ...[]string) bool {
	for _, values := range groups {
		if len(values) > MaxRefs || hasBlankOrDuplicate(values) {
			return false
		}
	}
	return true
}
func validScope(value ScopeBinding) bool {
	return !blank(value.ProjectRef) && !blank(value.WorkstreamRef) && !blank(value.WorksetRef) && !blank(value.CallGraphRef) && !blank(value.WorkpointRef) && !blank(value.WorkItemRef)
}
func validDerivativeType(v DerivativeType) bool {
	return v == DerivativePrint || v == DerivativePDF || v == DerivativeEmailText || v == DerivativeEmailHTML || v == DerivativeMarkdown || v == DerivativeRichText || v == DerivativeHTML || v == DerivativeJSON || v == DerivativeCSV || v == DerivativeArchive || v == DerivativePPTX || v == DerivativeHTMLSlides || v == DerivativePresentationPDF
}
func validAccessibility(v AccessibilityTarget) bool {
	return v == AccessibilityPlainText || v == AccessibilityWCAG22AA || v == AccessibilityPDFUA1 || v == AccessibilityPDFUA2 || v == AccessibilityNotApplicable
}
func validConformance(v ConformancePosture) bool {
	return v == ConformanceNotClaimed || v == ConformanceTargeted || v == ConformanceVerified || v == ConformanceFailed
}
func validArchive(v ArchivePosture) bool {
	return v == ArchiveNotApplicable || v == ArchiveSafe || v == ArchiveBlocked
}
func validDirection(v Direction) bool { return v == DirectionLTR || v == DirectionRTL }
func validViewer(v ViewerStatus) bool {
	return v == ViewerSupported || v == ViewerDegraded || v == ViewerUnsupported || v == ViewerUntested
}
func validDelivery(v DeliveryState) bool {
	return v == DeliveryNotAttempted || v == DeliveryQueued || v == DeliveryAccepted || v == DeliveryDelivered || v == DeliveryRejected || v == DeliveryBounced || v == DeliveryBlocked || v == DeliveryOutcomeUnknown
}
func digest(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxContractBytes {
		return "", ErrDerivativeContractInvalid
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
func sizeOK(value any) error {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxContractBytes {
		return ErrDerivativeContractInvalid
	}
	lower := strings.ToLower(string(body))
	for _, marker := range []string{"/home/", "/root/", "file://", `c:\\users\\`, "authorization: bearer", "cookie:"} {
		if strings.Contains(lower, marker) {
			return ErrDerivativeContractInvalid
		}
	}
	return nil
}
func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}
func blank(value string) bool { return strings.TrimSpace(value) == "" }
func addUnique(set map[string]struct{}, value string) bool {
	if _, ok := set[value]; ok {
		return false
	}
	set[value] = struct{}{}
	return true
}
func hasBlankOrDuplicate(values []string) bool {
	set := map[string]struct{}{}
	for _, value := range values {
		if blank(value) || !addUnique(set, value) {
			return true
		}
	}
	return false
}
func subset(values, allowed []string) bool {
	set := map[string]struct{}{}
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}
func keys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	return out
}
