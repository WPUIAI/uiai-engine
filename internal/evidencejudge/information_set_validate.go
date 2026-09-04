package evidencejudge

import "encoding/json"

func validateSelectionRequest(request InformationSelectionRequest) error {
	if !validInformationRef(request.InformationSetID) || !validInformationRef(request.ArtifactRef) ||
		!validInformationRef(request.WorkItemRef) || !validInformationRef(request.PolicyRef) ||
		!validInformationRef(request.RubricRef) || !validOptionalInformationRef(request.BlindingProfileRef) ||
		!validInformationRef(request.ContradictionPolicyRef) || request.CreatedAt.IsZero() ||
		!request.ExpiresAt.After(request.CreatedAt) || len(request.AcceptanceAtomRefs) == 0 ||
		len(request.AcceptanceAtomRefs) > MaxAcceptanceAtoms || informationHasDuplicate(request.AcceptanceAtomRefs) ||
		len(request.Sources) == 0 || len(request.Sources) > MaxInformationSelectionRefs {
		return ErrInformationSelectionInvalid
	}
	atoms := informationStringSet(request.AcceptanceAtomRefs)
	seenSources := map[string]struct{}{}
	for _, source := range request.Sources {
		if !validInformationRef(source.SourceRef) || len(source.AtomRefs) == 0 ||
			len(source.AtomRefs) > MaxAcceptanceAtoms || informationHasDuplicate(source.AtomRefs) {
			return ErrInformationSelectionInvalid
		}
		if _, exists := seenSources[source.SourceRef]; exists {
			return ErrInformationSelectionInvalid
		}
		seenSources[source.SourceRef] = struct{}{}
		for _, atomRef := range source.AtomRefs {
			if !validInformationRef(atomRef) {
				return ErrInformationSelectionInvalid
			}
			if _, exists := atoms[atomRef]; !exists {
				return ErrInformationSelectionInvalid
			}
		}
	}
	return nil
}

func validateInformationSetShape(set InformationSet) error {
	if set.Schema != InformationSetSchema || !validInformationRef(set.InformationSetID) ||
		(set.InformationSetSHA256 != "" && !validInformationSHA256(set.InformationSetSHA256)) ||
		!validInformationRef(set.Artifact.ArtifactRef) || set.Artifact.Revision == 0 ||
		!validInformationSHA256(set.Artifact.ManifestSHA256) || !validInformationSHA256(set.Artifact.BundleSHA256) ||
		!validScopeBinding(set.Artifact.Scope) || len(set.AcceptanceAtomRefs) == 0 ||
		len(set.AcceptanceAtomRefs) > MaxAcceptanceAtoms || informationHasDuplicate(set.AcceptanceAtomRefs) ||
		len(set.Sources) == 0 || len(set.Sources) > MaxInformationSelectionRefs ||
		len(set.Citations) == 0 || len(set.Citations) > MaxCitations ||
		len(set.Omissions) > MaxInformationOmissions || !validInformationRef(set.PolicyRef) ||
		!validInformationRef(set.RubricRef) || !validOptionalInformationRef(set.BlindingProfileRef) ||
		!validInformationRef(set.ContradictionPolicyRef) || set.EvidenceInstructionClass != "untrusted_evidence_data" ||
		set.CreatedAt.IsZero() || !set.ExpiresAt.After(set.CreatedAt) {
		return ErrInformationSetInvalid
	}
	atoms := informationStringSet(set.AcceptanceAtomRefs)
	sources := map[string]EvidenceSource{}
	for _, source := range set.Sources {
		if !validInformationRef(source.SourceRef) || !validInformationSHA256(source.SHA256) || !validModality(source.Modality) {
			return ErrInformationSetInvalid
		}
		if _, exists := sources[source.SourceRef]; exists {
			return ErrInformationSetInvalid
		}
		sources[source.SourceRef] = source
	}
	citationIDs := map[string]struct{}{}
	for _, citation := range set.Citations {
		source, exists := sources[citation.SourceRef]
		if citation.Schema != JudgeCitationSchema || !validInformationRef(citation.CitationID) || !exists ||
			citation.SourceSHA256 != source.SHA256 || citation.Modality != source.Modality || !validLocator(citation.Locator) ||
			len(citation.SupportsAtoms) == 0 || informationHasDuplicate(citation.SupportsAtoms) || len(citation.RebutsAtoms) != 0 {
			return ErrInformationSetInvalid
		}
		if _, exists := citationIDs[citation.CitationID]; exists {
			return ErrInformationSetInvalid
		}
		citationIDs[citation.CitationID] = struct{}{}
		for _, atomRef := range citation.SupportsAtoms {
			if _, exists := atoms[atomRef]; !exists {
				return ErrInformationSetInvalid
			}
		}
	}
	omitted := map[string]struct{}{}
	for _, omission := range set.Omissions {
		if !validInformationRef(omission.SourceRef) || omission.Required || !validOmissionReason(omission.ReasonCode) {
			return ErrInformationOmissionInvalid
		}
		if _, exists := sources[omission.SourceRef]; exists {
			return ErrInformationOmissionInvalid
		}
		if _, exists := omitted[omission.SourceRef]; exists {
			return ErrInformationOmissionInvalid
		}
		omitted[omission.SourceRef] = struct{}{}
	}
	body, err := json.Marshal(set)
	if err != nil || len(body) > MaxInformationSetBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}
