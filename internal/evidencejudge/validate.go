package evidencejudge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
	"unicode/utf8"
)

var (
	ErrJudgeContractInvalid   = errors.New("evidence judge contract invalid")
	ErrJudgeViewInvalid       = errors.New("evidence judge view invalid")
	ErrJudgeRequestInvalid    = errors.New("evidence judge request invalid")
	ErrJudgeResultInvalid     = errors.New("evidence judge result invalid")
	ErrCitationInvalid        = errors.New("evidence judge citation invalid")
	ErrModalityUnsatisfied    = errors.New("evidence judge modality unsatisfied")
	ErrInformationSetMismatch = errors.New("evidence judge information set mismatch")
	ErrJudgeAssignmentInvalid = errors.New("evidence judge assignment invalid")
	ErrJudgeBudgetExceeded    = errors.New("evidence judge budget exceeded")
	ErrJudgeExpired           = errors.New("evidence judge request expired")
)

func DigestJudgeView(view JudgeView) (string, error)       { return digestContract(view) }
func DigestJudgeRequest(req JudgeRequest) (string, error)  { return digestContract(req) }
func DigestJudgeResult(result JudgeResult) (string, error) { return digestContract(result) }

func digestContract(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: encoding", ErrJudgeContractInvalid)
	}
	if len(body) > MaxContractBytes {
		return "", ErrJudgeBudgetExceeded
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func ValidateJudgeView(view JudgeView) error {
	if view.Schema != JudgeViewSchema || blank(view.ViewID) {
		return ErrJudgeViewInvalid
	}
	if err := errSize(view); err != nil {
		return err
	}
	if err := validateArtifact(view.Artifact); err != nil {
		return err
	}
	if blank(view.CompletionContractRef) || blank(view.CompletionContractRevision) ||
		blank(view.InformationSetRef) || !validSHA256(view.InformationSetSHA256) {
		return ErrJudgeViewInvalid
	}
	if view.CreatedAt.IsZero() || !view.ExpiresAt.After(view.CreatedAt) {
		return ErrJudgeViewInvalid
	}
	atoms, err := validateAtoms(view.AcceptanceAtoms)
	if err != nil {
		return err
	}
	allowed := make(map[Verdict]struct{}, len(view.AllowedVerdicts))
	if len(view.AllowedVerdicts) == 0 || len(view.AllowedVerdicts) > 5 {
		return ErrJudgeViewInvalid
	}
	for _, verdict := range view.AllowedVerdicts {
		if !validVerdict(verdict) || hasVerdict(allowed, verdict) {
			return ErrJudgeViewInvalid
		}
		allowed[verdict] = struct{}{}
	}
	sources, err := validateSources(view.Sources)
	if err != nil {
		return err
	}
	citations, err := validateCitations(view.Citations, sources, atoms)
	if err != nil {
		return err
	}
	if err := validateRequirements(view.Modalities, atoms, citations); err != nil {
		return err
	}
	if !requirementsCoverAtoms(view.AcceptanceAtoms, view.Modalities) {
		return ErrModalityUnsatisfied
	}
	if len(view.Omissions) > MaxRefs || hasBlankOrDuplicateOmissions(view.Omissions) {
		return ErrJudgeViewInvalid
	}
	if err := validatePolicy(view.Policy); err != nil {
		return err
	}
	return nil
}

func ValidateJudgeViewAt(view JudgeView, now time.Time) error {
	if err := ValidateJudgeView(view); err != nil {
		return err
	}
	if !now.Before(view.ExpiresAt) {
		return ErrJudgeExpired
	}
	return nil
}

func ValidateJudgeRequest(req JudgeRequest) error {
	if req.Schema != JudgeRequestSchema || blank(req.RequestID) || blank(req.IdempotencyRef) ||
		blank(req.ViewRef) || !validSHA256(req.ViewSHA256) || blank(req.InformationSetRef) ||
		!validSHA256(req.InformationSetSHA256) || blank(req.AssignmentRef) ||
		blank(req.ExecutorIdentityRef) || blank(req.VerifierIdentityRef) || req.ExpiresAt.IsZero() {
		return ErrJudgeRequestInvalid
	}
	if err := errSize(req); err != nil {
		return err
	}
	if req.ExecutorIdentityRef == req.VerifierIdentityRef {
		return ErrJudgeAssignmentInvalid
	}
	if len(req.PolicyRefs) == 0 || len(req.PolicyRefs) > MaxRefs || hasBlankOrDuplicate(req.PolicyRefs) ||
		len(req.AcceptanceAtomRefs) == 0 || len(req.AcceptanceAtomRefs) > MaxAcceptanceAtoms ||
		hasBlankOrDuplicate(req.AcceptanceAtomRefs) {
		return ErrJudgeRequestInvalid
	}
	if len(req.RequiredModalities) == 0 || len(req.RequiredModalities) > MaxModalities {
		return ErrJudgeRequestInvalid
	}
	seenRequirements := make(map[string]struct{}, len(req.RequiredModalities))
	for _, requirement := range req.RequiredModalities {
		if !validRequirementShape(requirement) {
			return ErrJudgeRequestInvalid
		}
		if _, exists := seenRequirements[requirement.RequirementID]; exists {
			return ErrJudgeRequestInvalid
		}
		seenRequirements[requirement.RequirementID] = struct{}{}
	}
	if req.Budget.MaxTokens == 0 || req.Budget.MaxDurationMS == 0 || blank(req.ResultDetail) {
		return ErrJudgeRequestInvalid
	}
	return nil
}

func ValidateJudgeRequestAt(req JudgeRequest, now time.Time) error {
	if err := ValidateJudgeRequest(req); err != nil {
		return err
	}
	if !now.Before(req.ExpiresAt) {
		return ErrJudgeExpired
	}
	return nil
}

func ValidateJudgeRequestAgainst(req JudgeRequest, view JudgeView) error {
	if err := ValidateJudgeRequest(req); err != nil {
		return err
	}
	if err := ValidateJudgeView(view); err != nil {
		return err
	}
	viewDigest, err := DigestJudgeView(view)
	if err != nil {
		return err
	}
	if req.ViewRef != view.ViewID || req.ViewSHA256 != viewDigest {
		return ErrJudgeRequestInvalid
	}
	if req.InformationSetRef != view.InformationSetRef || req.InformationSetSHA256 != view.InformationSetSHA256 {
		return ErrInformationSetMismatch
	}
	atoms := stringSet(req.AcceptanceAtomRefs)
	if len(atoms) != len(view.AcceptanceAtoms) {
		return ErrJudgeRequestInvalid
	}
	for _, atom := range view.AcceptanceAtoms {
		if _, ok := atoms[atom.AtomRef]; !ok {
			return ErrJudgeRequestInvalid
		}
	}
	if !contains(req.PolicyRefs, view.Policy.PolicyRef) || !reflect.DeepEqual(req.RequiredModalities, view.Modalities) ||
		req.ExpiresAt.After(view.ExpiresAt) || !req.ExpiresAt.After(view.CreatedAt) {
		return ErrJudgeRequestInvalid
	}
	if view.Policy.IndependenceRequired && req.ExecutorIdentityRef == req.VerifierIdentityRef {
		return ErrJudgeAssignmentInvalid
	}
	return nil
}

func ValidateJudgeResult(result JudgeResult) error {
	if result.Schema != JudgeResultSchema || blank(result.ResultID) || blank(result.RequestID) ||
		!validSHA256(result.RequestSHA256) || blank(result.ViewRef) || !validSHA256(result.ViewSHA256) ||
		!validSHA256(result.InformationSetSHA256) || blank(result.JudgeIdentityRef) ||
		blank(result.ModelProvider) || blank(result.ModelVersion) || !validSHA256(result.CapabilityDigest) ||
		blank(result.PolicyRevision) || result.EvaluatedAt.IsZero() || !validOutcome(result.Outcome) ||
		result.ConfidencePPM > 1_000_000 {
		return ErrJudgeResultInvalid
	}
	if err := errSize(result); err != nil {
		return err
	}
	if len(result.AtomDecisions) == 0 || len(result.AtomDecisions) > MaxAcceptanceAtoms ||
		len(result.CitationIDs) > MaxCitations || hasBlankOrDuplicate(result.CitationIDs) ||
		len(result.Rationales) > MaxRationales || len(result.ContradictionRefs) > MaxRefs ||
		len(result.OmissionRefs) > MaxRefs || len(result.DisagreementRefs) > MaxRefs ||
		len(result.AppealRefs) > MaxRefs || len(result.ErrorCodes) > MaxRefs ||
		hasBlankOrDuplicate(result.ContradictionRefs) || hasBlankOrDuplicate(result.OmissionRefs) ||
		hasBlankOrDuplicate(result.DisagreementRefs) || hasBlankOrDuplicate(result.AppealRefs) {
		return ErrJudgeResultInvalid
	}
	decisions := make(map[string]Verdict, len(result.AtomDecisions))
	for _, decision := range result.AtomDecisions {
		if blank(decision.AtomRef) || !validVerdict(decision.Verdict) || blank(decision.ReasonCode) ||
			hasBlankOrDuplicate(decision.CitationIDs) {
			return ErrJudgeResultInvalid
		}
		if _, exists := decisions[decision.AtomRef]; exists {
			return ErrJudgeResultInvalid
		}
		decisions[decision.AtomRef] = decision.Verdict
	}
	if len(result.Rationales) != len(decisions) {
		return ErrJudgeResultInvalid
	}
	rationaleAtoms := make(map[string]struct{}, len(result.Rationales))
	for _, rationale := range result.Rationales {
		if _, ok := decisions[rationale.AtomRef]; !ok || blank(rationale.Summary) ||
			utf8.RuneCountInString(rationale.Summary) > MaxRationaleRunes ||
			len(rationale.CitationIDs) == 0 || hasBlankOrDuplicate(rationale.CitationIDs) {
			return ErrJudgeResultInvalid
		}
		if _, duplicate := rationaleAtoms[rationale.AtomRef]; duplicate {
			return ErrJudgeResultInvalid
		}
		rationaleAtoms[rationale.AtomRef] = struct{}{}
	}
	seenErrors := make(map[JudgeErrorCode]struct{}, len(result.ErrorCodes))
	for _, code := range result.ErrorCodes {
		if !validJudgeErrorCode(code) {
			return ErrJudgeResultInvalid
		}
		if _, duplicate := seenErrors[code]; duplicate {
			return ErrJudgeResultInvalid
		}
		seenErrors[code] = struct{}{}
	}
	if result.Outcome == OutcomeVerified && (len(result.ErrorCodes) > 0 || len(result.ContradictionRefs) > 0 || len(result.DisagreementRefs) > 0) {
		return ErrJudgeResultInvalid
	}
	if !outcomeMatches(result.Outcome, decisions) {
		return ErrJudgeResultInvalid
	}
	return nil
}

func ValidateJudgeResultAgainst(result JudgeResult, req JudgeRequest, view JudgeView) error {
	if err := ValidateJudgeResult(result); err != nil {
		return err
	}
	if err := ValidateJudgeRequestAgainst(req, view); err != nil {
		return err
	}
	reqDigest, err := DigestJudgeRequest(req)
	if err != nil {
		return err
	}
	if result.RequestID != req.RequestID || result.RequestSHA256 != reqDigest ||
		result.ViewRef != req.ViewRef || result.ViewSHA256 != req.ViewSHA256 {
		return ErrJudgeResultInvalid
	}
	if result.InformationSetSHA256 != req.InformationSetSHA256 {
		return ErrInformationSetMismatch
	}
	if result.JudgeIdentityRef != req.VerifierIdentityRef || result.JudgeIdentityRef == req.ExecutorIdentityRef {
		return ErrJudgeAssignmentInvalid
	}
	if result.EvaluatedAt.After(req.ExpiresAt) || result.EvaluatedAt.Before(view.CreatedAt) {
		return ErrJudgeExpired
	}
	if result.PolicyRevision != view.Policy.PolicyRevision {
		return ErrJudgeResultInvalid
	}
	atoms := make(map[string]struct{}, len(view.AcceptanceAtoms))
	for _, atom := range view.AcceptanceAtoms {
		atoms[atom.AtomRef] = struct{}{}
	}
	citations := make(map[string]struct{}, len(view.Citations))
	for _, citation := range view.Citations {
		citations[citation.CitationID] = struct{}{}
	}
	resultCitations := stringSet(result.CitationIDs)
	if len(result.AtomDecisions) != len(atoms) || !allExist(result.CitationIDs, citations) {
		return ErrJudgeResultInvalid
	}
	for _, decision := range result.AtomDecisions {
		if _, ok := atoms[decision.AtomRef]; !ok || !allExist(decision.CitationIDs, citations) || !allExist(decision.CitationIDs, resultCitations) {
			return ErrJudgeResultInvalid
		}
		if view.Policy.RequiredCitations && len(decision.CitationIDs) == 0 {
			return ErrCitationInvalid
		}
	}
	for _, rationale := range result.Rationales {
		if !allExist(rationale.CitationIDs, citations) || !allExist(rationale.CitationIDs, resultCitations) {
			return ErrCitationInvalid
		}
	}
	if result.Outcome == OutcomeVerified {
		for _, requirement := range view.Modalities {
			if requirement.Required && requirement.Status != ModalitySatisfied {
				return ErrModalityUnsatisfied
			}
		}
	}
	return nil
}

func validateArtifact(binding ArtifactBinding) error {
	if blank(binding.ArtifactRef) || binding.Revision == 0 || !validSHA256(binding.BundleSHA256) ||
		!validSHA256(binding.ManifestSHA256) || blank(binding.Scope.ProjectRef) ||
		blank(binding.Scope.WorkstreamRef) || blank(binding.Scope.WorksetRef) ||
		blank(binding.Scope.CallGraphRef) || blank(binding.Scope.WorkpointRef) ||
		blank(binding.Scope.WorkItemRef) || hasBlankOrDuplicate(binding.AttestationRefs) ||
		hasBlankOrDuplicate(binding.TrustRefs) || hasBlankOrDuplicate(binding.SecurityRefs) {
		return ErrJudgeViewInvalid
	}
	return nil
}

func validateAtoms(in []AcceptanceAtom) (map[string]struct{}, error) {
	if len(in) == 0 || len(in) > MaxAcceptanceAtoms {
		return nil, ErrJudgeViewInvalid
	}
	out := make(map[string]struct{}, len(in))
	for _, atom := range in {
		if blank(atom.AtomRef) || atom.Revision == 0 || blank(atom.Question) || len(atom.RequiredModalities) > MaxModalities {
			return nil, ErrJudgeViewInvalid
		}
		if _, exists := out[atom.AtomRef]; exists {
			return nil, ErrJudgeViewInvalid
		}
		for _, modality := range atom.RequiredModalities {
			if !validModality(modality) {
				return nil, ErrJudgeViewInvalid
			}
		}
		out[atom.AtomRef] = struct{}{}
	}
	return out, nil
}

func validateSources(in []EvidenceSource) (map[string]EvidenceSource, error) {
	if len(in) == 0 || len(in) > MaxRefs {
		return nil, ErrJudgeViewInvalid
	}
	out := make(map[string]EvidenceSource, len(in))
	for _, source := range in {
		if blank(source.SourceRef) || !validSHA256(source.SHA256) || !validModality(source.Modality) {
			return nil, ErrJudgeViewInvalid
		}
		if _, exists := out[source.SourceRef]; exists {
			return nil, ErrJudgeViewInvalid
		}
		out[source.SourceRef] = source
	}
	return out, nil
}

func validateCitations(in []Citation, sources map[string]EvidenceSource, atoms map[string]struct{}) (map[string]Citation, error) {
	if len(in) == 0 || len(in) > MaxCitations {
		return nil, ErrCitationInvalid
	}
	out := make(map[string]Citation, len(in))
	for _, citation := range in {
		source, ok := sources[citation.SourceRef]
		if citation.Schema != JudgeCitationSchema || blank(citation.CitationID) || !ok ||
			citation.SourceSHA256 != source.SHA256 || citation.Modality != source.Modality ||
			!validLocator(citation.Locator) || (len(citation.SupportsAtoms) == 0 && len(citation.RebutsAtoms) == 0) ||
			hasBlankOrDuplicate(citation.SupportsAtoms) || hasBlankOrDuplicate(citation.RebutsAtoms) {
			return nil, ErrCitationInvalid
		}
		if _, exists := out[citation.CitationID]; exists || !allExist(citation.SupportsAtoms, atoms) ||
			!allExist(citation.RebutsAtoms, atoms) || intersects(citation.SupportsAtoms, citation.RebutsAtoms) {
			return nil, ErrCitationInvalid
		}
		out[citation.CitationID] = citation
	}
	return out, nil
}

func validateRequirements(in []ModalityRequirement, atoms map[string]struct{}, citations map[string]Citation) error {
	if len(in) == 0 || len(in) > MaxModalities {
		return ErrJudgeViewInvalid
	}
	seen := make(map[string]struct{}, len(in))
	for _, requirement := range in {
		if requirement.Schema != JudgeModalitySchema || blank(requirement.RequirementID) ||
			!validModality(requirement.Modality) || !validModalityStatus(requirement.Status) ||
			hasBlankOrDuplicate(requirement.CitationIDs) {
			return ErrJudgeViewInvalid
		}
		if _, ok := atoms[requirement.AtomRef]; !ok {
			return ErrJudgeViewInvalid
		}
		if _, exists := seen[requirement.RequirementID]; exists || (requirement.Required && requirement.Status == ModalityNotApplicable) {
			return ErrJudgeViewInvalid
		}
		seen[requirement.RequirementID] = struct{}{}
		if requirement.Status == ModalitySatisfied && len(requirement.CitationIDs) == 0 {
			return ErrModalityUnsatisfied
		}
		for _, citationID := range requirement.CitationIDs {
			citation, ok := citations[citationID]
			if !ok || citation.Modality != requirement.Modality || !contains(citation.SupportsAtoms, requirement.AtomRef) {
				return ErrModalityUnsatisfied
			}
		}
	}
	return nil
}

func requirementsCoverAtoms(atoms []AcceptanceAtom, requirements []ModalityRequirement) bool {
	covered := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		if requirement.Required {
			covered[requirement.AtomRef+"\x00"+string(requirement.Modality)] = struct{}{}
		}
	}
	for _, atom := range atoms {
		for _, modality := range atom.RequiredModalities {
			if _, ok := covered[atom.AtomRef+"\x00"+string(modality)]; !ok {
				return false
			}
		}
	}
	return true
}

func validatePolicy(policy JudgePolicy) error {
	if blank(policy.PolicyRef) || blank(policy.PolicyRevision) || blank(policy.RubricRef) ||
		blank(policy.ContradictionPolicyRef) || len(policy.ForbiddenAssumptions) > MaxForbiddenAssumptions ||
		hasBlankOrDuplicate(policy.ForbiddenAssumptions) {
		return ErrJudgeViewInvalid
	}
	return nil
}

func outcomeMatches(outcome JudgeOutcome, decisions map[string]Verdict) bool {
	wanted := map[JudgeOutcome]Verdict{
		OutcomeRejected: VerdictRebutted, OutcomeInsufficientEvidence: VerdictInsufficientEvidence,
		OutcomeBlocked: VerdictBlocked, OutcomeDisputed: VerdictDisputed,
	}
	if outcome == OutcomeVerified {
		for _, verdict := range decisions {
			if verdict != VerdictSupported {
				return false
			}
		}
		return true
	}
	needle := wanted[outcome]
	for _, verdict := range decisions {
		if verdict == needle {
			return true
		}
	}
	return false
}

func errSize(value any) error {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxContractBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}
