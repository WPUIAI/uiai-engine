package evidencejudge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func artifactBinding(manifest evidenceartifact.Manifest, workItemRef, manifestSHA string) (ArtifactBinding, error) {
	var workItemFound bool
	for _, item := range manifest.Scope.WorkItems {
		if item.WorkItemRef == workItemRef {
			workItemFound = true
			break
		}
	}
	if !workItemFound {
		return ArtifactBinding{}, ErrInformationScopeMismatch
	}
	securityRefs := append([]string{manifest.Security.PolicyRef}, manifest.Security.InspectionReceiptRefs...)
	securityRefs = append(securityRefs, manifest.Security.SanitizationRefs...)
	securityRefs = append(securityRefs, manifest.Security.RedactionRefs...)
	trustRefs := []string{manifest.Authority.SourceAuthorityRef, manifest.Authority.EvidenceAuthorityRef, manifest.Authority.ReviewerPolicyRef}
	return ArtifactBinding{
		ArtifactRef:    manifest.ArtifactID,
		Revision:       manifest.Revision,
		BundleSHA256:   manifest.Integrity.BundleSHA256,
		ManifestSHA256: manifestSHA,
		Scope: ScopeBinding{
			ProjectRef:    manifest.Scope.Project.ProjectRef,
			WorkstreamRef: manifest.Scope.Workstream.WorkstreamRef,
			WorksetRef:    manifest.Scope.Workset.WorksetRef,
			CallGraphRef:  manifest.Scope.CallGraph.DefinitionRef,
			WorkpointRef:  manifest.Scope.Workpoint.WorkpointRef,
			WorkItemRef:   workItemRef,
		},
		TrustRefs:    sortedCopy(trustRefs),
		SecurityRefs: sortedCopy(securityRefs),
	}, nil
}

func buildSourceInventory(manifest evidenceartifact.Manifest) (map[string]inventorySource, error) {
	inventory := map[string]inventorySource{}
	add := func(item inventorySource) error {
		if !validInformationRef(item.source.SourceRef) || !validInformationSHA256(item.source.SHA256) {
			return ErrInformationSetInvalid
		}
		if _, exists := inventory[item.source.SourceRef]; exists {
			return ErrInformationSetInvalid
		}
		inventory[item.source.SourceRef] = item
		return nil
	}
	claimAtoms := map[string]map[string]struct{}{}
	for _, claim := range manifest.Claims {
		digest, err := informationDigest(claim)
		if err != nil {
			return nil, err
		}
		atoms := informationStringSet(claim.AcceptanceAtomRefs)
		claimAtoms[claim.ClaimID] = atoms
		reason := OmissionReason("")
		if claim.Status == evidenceartifact.ClaimBlocked || claim.Status == evidenceartifact.ClaimMissing {
			reason = OmissionUnavailable
		}
		if err := add(inventorySource{source: EvidenceSource{SourceRef: claim.ClaimID, SHA256: digest, Modality: ModalityBoundedText}, locator: CitationLocator{Type: LocatorClaimID, Value: claim.ClaimID}, allowedAtoms: atoms, unavailable: reason}); err != nil {
			return nil, err
		}
	}
	for _, asset := range manifest.Assets {
		modality, supported := modalityForAsset(asset)
		reason := OmissionReason("")
		if !supported {
			reason = OmissionUnsupported
		} else if manifest.Authority.Posture == evidenceartifact.PostureStale {
			reason = OmissionStale
		} else if asset.RedactionState == evidenceartifact.RedactionBlocked {
			reason = OmissionBlocked
		} else if asset.RedactionState == evidenceartifact.RedactionRedacted {
			reason = OmissionRedacted
		}
		atoms := map[string]struct{}{}
		for _, claimRef := range asset.ClaimRefs {
			for atomRef := range claimAtoms[claimRef] {
				atoms[atomRef] = struct{}{}
			}
		}
		if err := add(inventorySource{source: EvidenceSource{SourceRef: asset.AssetID, SHA256: asset.SHA256, Modality: modality}, locator: CitationLocator{Type: LocatorAssetID, Value: asset.AssetID}, allowedAtoms: atoms, unavailable: reason}); err != nil {
			return nil, err
		}
	}
	for ordinal, event := range manifest.Provenance.Custody {
		digest, err := informationDigest(event)
		if err != nil {
			return nil, err
		}
		if err := add(inventorySource{source: EvidenceSource{SourceRef: event.EventID, SHA256: digest, Modality: ModalityCustodyEvent}, locator: CitationLocator{Type: LocatorCustodyOrdinal, Start: int64(ordinal)}, allowedAtoms: map[string]struct{}{}}); err != nil {
			return nil, err
		}
	}
	for _, receiptRef := range manifest.ReceiptRefs {
		if err := add(inventorySource{source: EvidenceSource{SourceRef: receiptRef, SHA256: informationDigestString(receiptRef), Modality: ModalityActionReceipt}, locator: CitationLocator{Type: LocatorReceiptRef, Value: receiptRef}, allowedAtoms: map[string]struct{}{}}); err != nil {
			return nil, err
		}
	}
	for _, sourceRef := range manifest.Provenance.SourceRefs {
		if _, exists := inventory[sourceRef]; exists {
			continue
		}
		if err := add(inventorySource{source: EvidenceSource{SourceRef: sourceRef, SHA256: informationDigestString(sourceRef), Modality: ModalitySourceSnapshot}, locator: CitationLocator{Type: LocatorJSONPointer, Value: "/provenance/source_refs"}, allowedAtoms: map[string]struct{}{}}); err != nil {
			return nil, err
		}
	}
	return inventory, nil
}

func modalityForAsset(asset evidenceartifact.Asset) (Modality, bool) {
	switch {
	case strings.HasPrefix(asset.MediaType, "image/"):
		return ModalityStaticImage, true
	case strings.HasPrefix(asset.MediaType, "video/"):
		return ModalitySynchronizedVideo, true
	case asset.MediaType == "application/json" || asset.Kind == "structured_data":
		return ModalityStructuredData, true
	case strings.HasPrefix(asset.MediaType, "text/"):
		return ModalityBoundedText, true
	default:
		return ModalityBoundedText, false
	}
}

func normalizeInformationSet(set InformationSet) InformationSet {
	body, _ := json.Marshal(set)
	var out InformationSet
	_ = json.Unmarshal(body, &out)
	out.AcceptanceAtomRefs = sortedCopy(out.AcceptanceAtomRefs)
	out.Artifact.AttestationRefs = sortedCopy(out.Artifact.AttestationRefs)
	out.Artifact.TrustRefs = sortedCopy(out.Artifact.TrustRefs)
	out.Artifact.SecurityRefs = sortedCopy(out.Artifact.SecurityRefs)
	for i := range out.Citations {
		out.Citations[i].SupportsAtoms = sortedCopy(out.Citations[i].SupportsAtoms)
		out.Citations[i].RebutsAtoms = sortedCopy(out.Citations[i].RebutsAtoms)
	}
	sort.Slice(out.Sources, func(i, j int) bool { return out.Sources[i].SourceRef < out.Sources[j].SourceRef })
	sort.Slice(out.Citations, func(i, j int) bool { return out.Citations[i].CitationID < out.Citations[j].CitationID })
	sort.Slice(out.Omissions, func(i, j int) bool { return out.Omissions[i].SourceRef < out.Omissions[j].SourceRef })
	return out
}

func normalizeAcceptanceAtoms(atoms []AcceptanceAtom) []AcceptanceAtom {
	body, _ := json.Marshal(atoms)
	var out []AcceptanceAtom
	_ = json.Unmarshal(body, &out)
	for i := range out {
		sort.Slice(out[i].RequiredModalities, func(a, b int) bool { return out[i].RequiredModalities[a] < out[i].RequiredModalities[b] })
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AtomRef < out[j].AtomRef })
	return out
}

func normalizeModalityRequirements(requirements []ModalityRequirement) []ModalityRequirement {
	body, _ := json.Marshal(requirements)
	var out []ModalityRequirement
	_ = json.Unmarshal(body, &out)
	for i := range out {
		out[i].CitationIDs = sortedCopy(out[i].CitationIDs)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RequirementID < out[j].RequirementID })
	return out
}

func sameAtomRefs(atoms []AcceptanceAtom, refs []string) bool {
	if len(atoms) != len(refs) {
		return false
	}
	sortedRefs := sortedCopy(refs)
	for i := range atoms {
		if atoms[i].AtomRef != sortedRefs[i] {
			return false
		}
	}
	return true
}

func computeInformationSetSHA256Unchecked(set InformationSet) (string, error) {
	set.InformationSetSHA256 = ""
	body, err := json.Marshal(normalizeInformationSet(set))
	if err != nil || len(body) > MaxInformationSetBytes {
		return "", ErrJudgeBudgetExceeded
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func informationCitationID(source EvidenceSource, atomRefs []string) (string, error) {
	digest, err := informationDigest(struct {
		Source   EvidenceSource
		AtomRefs []string
	}{source, atomRefs})
	if err != nil {
		return "", err
	}
	return "citation:sha256:" + digest, nil
}

func informationDigest(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxInformationSetBytes {
		return "", ErrJudgeBudgetExceeded
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func informationDigestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func informationStringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func informationHasDuplicate(values []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if !validInformationRef(value) {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func validScopeBinding(scope ScopeBinding) bool {
	return validInformationRef(scope.ProjectRef) && validInformationRef(scope.WorkstreamRef) &&
		validInformationRef(scope.WorksetRef) && validInformationRef(scope.CallGraphRef) &&
		validInformationRef(scope.WorkpointRef) && validInformationRef(scope.WorkItemRef)
}

func validInformationRef(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) ||
		utf8.RuneCountInString(value) > MaxInformationRefRunes || strings.HasPrefix(value, "/") ||
		strings.HasPrefix(strings.ToLower(value), "file:") || strings.Contains(value, "\\") ||
		strings.Contains(value, "://") || strings.ContainsAny(value, "?#") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validOptionalInformationRef(value string) bool { return value == "" || validInformationRef(value) }

func validInformationSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validOmissionReason(value OmissionReason) bool {
	return value == OmissionMissing || value == OmissionRedacted || value == OmissionStale ||
		value == OmissionUnsupported || value == OmissionUnavailable || value == OmissionBlocked
}
