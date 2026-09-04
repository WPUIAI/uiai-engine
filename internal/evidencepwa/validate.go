package evidencepwa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"
)

var (
	ErrProjectionInvalid           = errors.New("evidence PWA projection invalid")
	ErrProjectionTooLarge          = errors.New("evidence PWA projection too large")
	ErrAvailabilityInvalid         = errors.New("evidence PWA availability invalid")
	ErrAccessPostureInvalid        = errors.New("evidence PWA access posture invalid")
	ErrRedactionInvalid            = errors.New("evidence PWA redaction invalid")
	ErrRelativeRefInvalid          = errors.New("evidence PWA relative reference invalid")
	ErrProjectionBindingMismatch   = errors.New("evidence PWA projection binding mismatch")
	ErrProjectionReferenceDangling = errors.New("evidence PWA projection reference dangling")
)

func DigestProjection(projection Projection) (string, error) {
	body, err := json.Marshal(projection)
	if err != nil {
		return "", ErrProjectionInvalid
	}
	if len(body) > MaxProjectionBytes {
		return "", ErrProjectionTooLarge
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func ValidateProjection(projection Projection) error {
	body, err := json.Marshal(projection)
	if err != nil {
		return ErrProjectionInvalid
	}
	if len(body) > MaxProjectionBytes {
		return ErrProjectionTooLarge
	}
	if projection.Schema != ProjectionSchema || blank(projection.ProjectionID) ||
		utf8.RuneCountInString(projection.Title) == 0 || utf8.RuneCountInString(projection.Title) > MaxTitleRunes ||
		utf8.RuneCountInString(projection.Summary) == 0 || utf8.RuneCountInString(projection.Summary) > MaxSummaryRunes ||
		projection.FreshnessObservedAt.IsZero() || blank(projection.FederationPosture) {
		return ErrProjectionInvalid
	}
	if err := validateArtifact(projection.Artifact); err != nil {
		return err
	}
	if err := validateWorkItemScope(projection.Artifact.Scope, projection.WorkItems); err != nil {
		return err
	}
	if !validAvailability(projection.Availability) {
		return ErrAvailabilityInvalid
	}
	if !validAccess(projection.Access) || !validInteraction(projection.Interaction) {
		return ErrAccessPostureInvalid
	}
	if err := validateSections(projection.Sections); err != nil {
		return err
	}
	known, err := validateEvidence(projection)
	if err != nil {
		return err
	}
	if err := validateRefs(projection, known); err != nil {
		return err
	}
	if err := validateRedaction(projection); err != nil {
		return err
	}
	if err := validatePage(projection); err != nil {
		return err
	}
	if err := validateLinks(projection); err != nil {
		return err
	}
	if hasPrivatePath(string(body)) {
		return ErrRelativeRefInvalid
	}
	return nil
}

func validateArtifact(binding ArtifactBinding) error {
	if blank(binding.ArtifactRef) || binding.Revision == 0 || !validSHA256(binding.ManifestSHA256) ||
		!validSHA256(binding.BundleSHA256) || blank(binding.Scope.ProjectRef) ||
		blank(binding.Scope.WorkstreamRef) || blank(binding.Scope.WorksetRef) ||
		blank(binding.Scope.CallGraphRef) || blank(binding.Scope.WorkpointRef) ||
		blank(binding.Scope.WorkItemRef) {
		return ErrProjectionBindingMismatch
	}
	return nil
}

func validateSections(sections []Section) error {
	expected := []SectionID{SectionOverview, SectionEvidence, SectionTimeline, SectionInspect, SectionDeveloper}
	if len(sections) != len(expected) || len(sections) > MaxSections {
		return ErrProjectionInvalid
	}
	for index, section := range sections {
		if section.ID != expected[index] || blank(section.Heading) {
			return ErrProjectionInvalid
		}
	}
	return nil
}

func validateEvidence(projection Projection) (map[string]struct{}, error) {
	if len(projection.Claims) > MaxClaims || len(projection.Assets) > MaxAssets ||
		len(projection.Citations) > MaxCitations || len(projection.Timeline) > MaxTimelineEntries ||
		len(projection.Warnings) > MaxWarnings {
		return nil, ErrProjectionTooLarge
	}
	known := map[string]struct{}{projection.Artifact.ArtifactRef: {}}
	assets := make(map[string]Asset, len(projection.Assets))
	for _, asset := range projection.Assets {
		if blank(asset.AssetID) || !relativeRef(asset.Ref) || !validSHA256(asset.SHA256) ||
			blank(asset.MIME) || blank(asset.Modality) || asset.Bytes == 0 || !addUnique(known, asset.AssetID) {
			return nil, ErrProjectionInvalid
		}
		assets[asset.AssetID] = asset
	}
	citations := make(map[string]Citation, len(projection.Citations))
	for _, citation := range projection.Citations {
		asset, ok := assets[citation.SourceRef]
		if blank(citation.CitationID) || !ok || citation.SHA256 != asset.SHA256 || blank(citation.Locator) ||
			!addUnique(known, citation.CitationID) {
			return nil, ErrProjectionReferenceDangling
		}
		citations[citation.CitationID] = citation
	}
	claims := make(map[string]struct{}, len(projection.Claims))
	for _, claim := range projection.Claims {
		if blank(claim.ClaimID) || blank(claim.Statement) || blank(claim.Posture) ||
			hasBlankOrDuplicate(claim.CitationIDs) || hasBlankOrDuplicate(claim.AssetIDs) ||
			!allExist(claim.CitationIDs, citations) || !allExist(claim.AssetIDs, assets) || !addUnique(known, claim.ClaimID) {
			return nil, ErrProjectionReferenceDangling
		}
		claims[claim.ClaimID] = struct{}{}
	}
	for _, entry := range projection.Timeline {
		if blank(entry.EntryID) || entry.OccurredAt.IsZero() || blank(entry.EventType) ||
			hasBlankOrDuplicate(entry.Refs) || !allExist(entry.Refs, known) || !addUnique(known, entry.EntryID) {
			return nil, ErrProjectionReferenceDangling
		}
	}
	for _, warning := range projection.Warnings {
		if blank(warning.Code) || blank(warning.Message) || hasBlankOrDuplicate(warning.EvidenceRefs) ||
			!allExist(warning.EvidenceRefs, known) {
			return nil, ErrProjectionReferenceDangling
		}
	}
	return known, nil
}

func validateRefs(projection Projection, known map[string]struct{}) error {
	groups := [][]string{
		projection.InspectionRefs, projection.SecurityRefs, projection.CustodyRefs,
		projection.AttestationRefs, projection.TrustRefs, projection.OmissionRefs,
		projection.RelatedArtifactRefs, projection.ReceiptRefs,
	}
	for _, group := range groups {
		if len(group) > MaxRefs || hasBlankOrDuplicate(group) {
			return ErrProjectionInvalid
		}
		for _, ref := range group {
			known[ref] = struct{}{}
		}
	}
	return nil
}

func validateRedaction(projection Projection) error {
	redaction := projection.Redaction
	if !validRedaction(redaction.State) {
		return ErrRedactionInvalid
	}
	if (redaction.State == RedactionApplied || redaction.State == RedactionPartiallyApplied) && blank(redaction.EvidenceRef) {
		return ErrRedactionInvalid
	}
	if !blank(redaction.EvidenceRef) && !contains(projection.InspectionRefs, redaction.EvidenceRef) && !contains(projection.SecurityRefs, redaction.EvidenceRef) {
		return ErrRedactionInvalid
	}
	if projection.Access == AccessPublicSafe {
		if redaction.State != RedactionApplied || blank(redaction.EvidenceRef) ||
			redaction.PrivateRefCount != 0 || redaction.SecretFindingCount != 0 || redaction.PIIFindingCount != 0 ||
			projection.Interaction != InteractionReadOnly || !blank(projection.HandoffRef) {
			return ErrRedactionInvalid
		}
	}
	if projection.Access == AccessOfflineSnapshot && (projection.Interaction != InteractionReadOnly || !blank(projection.HandoffRef)) {
		return ErrAccessPostureInvalid
	}
	if projection.Interaction == InteractionAuthenticatedHandoff && blank(projection.HandoffRef) {
		return ErrAccessPostureInvalid
	}
	if !blank(projection.HandoffRef) && !relativeRef(projection.HandoffRef) {
		return ErrRelativeRefInvalid
	}
	return nil
}

func validatePage(projection Projection) error {
	if projection.Page.PageSize == 0 || projection.Page.PageSize > MaxPageSize ||
		projection.Page.TotalCount < uint64(len(projection.Claims)) ||
		(!blank(projection.Page.NextCursor) && projection.Page.NextCursor == projection.Page.Cursor) {
		return ErrProjectionInvalid
	}
	return nil
}

func validateLinks(projection Projection) error {
	seen := make(map[string]struct{}, len(projection.Links))
	if len(projection.Links) > MaxRefs {
		return ErrProjectionTooLarge
	}
	for _, link := range projection.Links {
		if blank(link.Rel) || !relativeRef(link.Href) {
			return ErrRelativeRefInvalid
		}
		key := link.Rel + "\x00" + link.Href
		if _, exists := seen[key]; exists {
			return ErrProjectionInvalid
		}
		seen[key] = struct{}{}
	}
	return nil
}

func relativeRef(value string) bool {
	if blank(value) || strings.ContainsAny(value, "\\\r\n\x00") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Scheme != "" {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return len(value) > 1
	}
	cleaned := path.Clean(parsed.Path)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

func hasPrivatePath(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"/home/", "/root/", "file://", `c:\\users\\`, "authorization: bearer", "cookie:"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validAvailability(value AvailabilityState) bool {
	return value == AvailabilityLoading || value == AvailabilityReady || value == AvailabilityUnavailable ||
		value == AvailabilityBlocked || value == AvailabilityCorrupt || value == AvailabilityStale ||
		value == AvailabilityRedacted || value == AvailabilityDegraded
}
func validAccess(value AccessPosture) bool {
	return value == AccessLocalhost || value == AccessLAN || value == AccessTailnet || value == AccessPrivate ||
		value == AccessUnlisted || value == AccessPublicSafe || value == AccessOfflineSnapshot
}
func validRedaction(value RedactionState) bool {
	return value == RedactionNotRequired || value == RedactionApplied || value == RedactionPartiallyApplied ||
		value == RedactionBlocked || value == RedactionUnknown
}
func validInteraction(value InteractionMode) bool {
	return value == InteractionReadOnly || value == InteractionAuthenticatedHandoff
}
func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}
func blank(value string) bool { return strings.TrimSpace(value) == "" }
func addUnique(values map[string]struct{}, value string) bool {
	if _, exists := values[value]; exists {
		return false
	}
	values[value] = struct{}{}
	return true
}
func hasBlankOrDuplicate(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if blank(value) || !addUnique(seen, value) {
			return true
		}
	}
	return false
}
func allExist[T any](values []string, known map[string]T) bool {
	for _, value := range values {
		if _, exists := known[value]; !exists {
			return false
		}
	}
	return true
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
