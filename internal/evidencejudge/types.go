package evidencejudge

import (
	"strings"
	"time"
)

const (
	JudgeViewSchema     = "uiai.evidence_judge_view.v1"
	JudgeRequestSchema  = "uiai.evidence_judge_request.v1"
	JudgeResultSchema   = "uiai.evidence_judge_result.v1"
	JudgeCitationSchema = "uiai.evidence_judge_citation.v1"
	JudgeModalitySchema = "uiai.evidence_judge_modality.v1"

	MaxContractBytes        = 1 << 20
	MaxRefs                 = 256
	MaxCitations            = 512
	MaxModalities           = 32
	MaxAcceptanceAtoms      = 256
	MaxRationales           = 256
	MaxRationaleRunes       = 2000
	MaxForbiddenAssumptions = 64
)

type Verdict string

const (
	VerdictSupported            Verdict = "supported"
	VerdictRebutted             Verdict = "rebutted"
	VerdictInsufficientEvidence Verdict = "insufficient_evidence"
	VerdictBlocked              Verdict = "blocked"
	VerdictDisputed             Verdict = "disputed"
)

type JudgeOutcome string

const (
	OutcomeVerified             JudgeOutcome = "verified"
	OutcomeRejected             JudgeOutcome = "rejected"
	OutcomeInsufficientEvidence JudgeOutcome = "insufficient_evidence"
	OutcomeBlocked              JudgeOutcome = "blocked"
	OutcomeDisputed             JudgeOutcome = "disputed"
)

type JudgeErrorCode string

const (
	ErrorCapabilityMismatch       JudgeErrorCode = "capability_mismatch"
	ErrorModalityMissing          JudgeErrorCode = "modality_missing"
	ErrorEvidenceMissing          JudgeErrorCode = "evidence_missing"
	ErrorStaleRevision            JudgeErrorCode = "stale_revision"
	ErrorHashMismatch             JudgeErrorCode = "hash_mismatch"
	ErrorScopeMismatch            JudgeErrorCode = "scope_mismatch"
	ErrorInformationSetDrift      JudgeErrorCode = "information_set_drift"
	ErrorPromptInjectionSuspected JudgeErrorCode = "prompt_injection_suspected"
	ErrorBudgetExhausted          JudgeErrorCode = "budget_exhausted"
	ErrorJudgeUnavailable         JudgeErrorCode = "judge_unavailable"
)

type Modality string

const (
	ModalityStaticImage       Modality = "static_image"
	ModalitySynchronizedVideo Modality = "synchronized_video"
	ModalityStructuredData    Modality = "structured_data"
	ModalityBoundedText       Modality = "bounded_text"
	ModalityActionReceipt     Modality = "action_receipt"
	ModalityCustodyEvent      Modality = "custody_event"
	ModalitySourceSnapshot    Modality = "source_snapshot"
)

type ModalityStatus string

const (
	ModalitySatisfied     ModalityStatus = "satisfied"
	ModalityInsufficient  ModalityStatus = "insufficient"
	ModalityBlocked       ModalityStatus = "blocked"
	ModalityNotApplicable ModalityStatus = "not_applicable"
)

type LocatorType string

const (
	LocatorClaimID        LocatorType = "claim_id"
	LocatorAssetID        LocatorType = "asset_id"
	LocatorCustodyOrdinal LocatorType = "custody_ordinal"
	LocatorReceiptRef     LocatorType = "receipt_ref"
	LocatorJSONPointer    LocatorType = "json_pointer"
	LocatorTextLines      LocatorType = "text_lines"
	LocatorImageRegion    LocatorType = "image_region"
	LocatorMediaTime      LocatorType = "media_time"
)

type ScopeBinding struct {
	ProjectRef    string `json:"project_ref"`
	WorkstreamRef string `json:"workstream_ref"`
	WorksetRef    string `json:"workset_ref"`
	CallGraphRef  string `json:"callgraph_ref"`
	WorkpointRef  string `json:"workpoint_ref"`
	WorkItemRef   string `json:"work_item_ref"`
}

type ArtifactBinding struct {
	ArtifactRef     string       `json:"artifact_ref"`
	Revision        uint64       `json:"revision"`
	BundleSHA256    string       `json:"bundle_sha256"`
	ManifestSHA256  string       `json:"manifest_sha256"`
	Scope           ScopeBinding `json:"scope"`
	AttestationRefs []string     `json:"attestation_refs,omitempty"`
	TrustRefs       []string     `json:"trust_refs,omitempty"`
	SecurityRefs    []string     `json:"security_refs,omitempty"`
}

type AcceptanceAtom struct {
	AtomRef            string     `json:"atom_ref"`
	Revision           uint64     `json:"revision"`
	Question           string     `json:"question"`
	RequiredModalities []Modality `json:"required_modalities,omitempty"`
}

type EvidenceSource struct {
	SourceRef string   `json:"source_ref"`
	SHA256    string   `json:"sha256"`
	Modality  Modality `json:"modality"`
}

type CitationLocator struct {
	Type   LocatorType `json:"type"`
	Value  string      `json:"value,omitempty"`
	Start  int64       `json:"start,omitempty"`
	End    int64       `json:"end,omitempty"`
	X      float64     `json:"x,omitempty"`
	Y      float64     `json:"y,omitempty"`
	Width  float64     `json:"width,omitempty"`
	Height float64     `json:"height,omitempty"`
}

type Citation struct {
	Schema        string          `json:"schema"`
	CitationID    string          `json:"citation_id"`
	SourceRef     string          `json:"source_ref"`
	SourceSHA256  string          `json:"source_sha256"`
	Modality      Modality        `json:"modality"`
	Locator       CitationLocator `json:"locator"`
	SupportsAtoms []string        `json:"supports_atoms,omitempty"`
	RebutsAtoms   []string        `json:"rebuts_atoms,omitempty"`
}

type ModalityRequirement struct {
	Schema        string         `json:"schema"`
	RequirementID string         `json:"requirement_id"`
	AtomRef       string         `json:"atom_ref"`
	Modality      Modality       `json:"modality"`
	Required      bool           `json:"required"`
	Status        ModalityStatus `json:"status"`
	CitationIDs   []string       `json:"citation_ids,omitempty"`
	ReasonCode    string         `json:"reason_code,omitempty"`
}

type Omission struct {
	Ref        string `json:"ref"`
	ReasonCode string `json:"reason_code"`
}

type JudgePolicy struct {
	PolicyRef              string   `json:"policy_ref"`
	PolicyRevision         string   `json:"policy_revision"`
	RubricRef              string   `json:"rubric_ref"`
	IndependenceRequired   bool     `json:"independence_required"`
	BlindingProfileRef     string   `json:"blinding_profile_ref,omitempty"`
	ContradictionPolicyRef string   `json:"contradiction_policy_ref"`
	ForbiddenAssumptions   []string `json:"forbidden_assumptions,omitempty"`
	RequiredCitations      bool     `json:"required_citations"`
}

type JudgeView struct {
	Schema                     string                `json:"schema"`
	ViewID                     string                `json:"view_id"`
	Artifact                   ArtifactBinding       `json:"artifact"`
	CompletionContractRef      string                `json:"completion_contract_ref"`
	CompletionContractRevision string                `json:"completion_contract_revision"`
	AcceptanceAtoms            []AcceptanceAtom      `json:"acceptance_atoms"`
	AllowedVerdicts            []Verdict             `json:"allowed_verdicts"`
	InformationSetRef          string                `json:"information_set_ref"`
	InformationSetSHA256       string                `json:"information_set_sha256"`
	Sources                    []EvidenceSource      `json:"sources"`
	Citations                  []Citation            `json:"citations"`
	Modalities                 []ModalityRequirement `json:"modalities"`
	Omissions                  []Omission            `json:"omissions,omitempty"`
	Policy                     JudgePolicy           `json:"policy"`
	CreatedAt                  time.Time             `json:"created_at"`
	ExpiresAt                  time.Time             `json:"expires_at"`
}

type JudgeBudget struct {
	MaxTokens      uint64 `json:"max_tokens"`
	MaxMediaBytes  uint64 `json:"max_media_bytes"`
	MaxSpendMicros uint64 `json:"max_spend_micros"`
	MaxDurationMS  uint64 `json:"max_duration_ms"`
}

type JudgeRequest struct {
	Schema               string                `json:"schema"`
	RequestID            string                `json:"request_id"`
	IdempotencyRef       string                `json:"idempotency_ref"`
	ViewRef              string                `json:"view_ref"`
	ViewSHA256           string                `json:"view_sha256"`
	InformationSetRef    string                `json:"information_set_ref"`
	InformationSetSHA256 string                `json:"information_set_sha256"`
	AssignmentRef        string                `json:"assignment_ref"`
	ExecutorIdentityRef  string                `json:"executor_identity_ref"`
	VerifierIdentityRef  string                `json:"verifier_identity_ref"`
	PolicyRefs           []string              `json:"policy_refs"`
	AcceptanceAtomRefs   []string              `json:"acceptance_atom_refs"`
	RequiredModalities   []ModalityRequirement `json:"required_modalities"`
	Budget               JudgeBudget           `json:"budget"`
	ExpiresAt            time.Time             `json:"expires_at"`
	ResultDetail         string                `json:"result_detail"`
}

type AtomDecision struct {
	AtomRef     string   `json:"atom_ref"`
	Verdict     Verdict  `json:"verdict"`
	CitationIDs []string `json:"citation_ids,omitempty"`
	ReasonCode  string   `json:"reason_code"`
}

type Rationale struct {
	AtomRef     string   `json:"atom_ref"`
	Summary     string   `json:"summary"`
	CitationIDs []string `json:"citation_ids"`
}

type JudgeResult struct {
	Schema               string           `json:"schema"`
	ResultID             string           `json:"result_id"`
	RequestID            string           `json:"request_id"`
	RequestSHA256        string           `json:"request_sha256"`
	ViewRef              string           `json:"view_ref"`
	ViewSHA256           string           `json:"view_sha256"`
	InformationSetSHA256 string           `json:"information_set_sha256"`
	JudgeIdentityRef     string           `json:"judge_identity_ref"`
	ModelProvider        string           `json:"model_provider"`
	ModelVersion         string           `json:"model_version"`
	CapabilityDigest     string           `json:"capability_digest"`
	PolicyRevision       string           `json:"policy_revision"`
	EvaluatedAt          time.Time        `json:"evaluated_at"`
	Outcome              JudgeOutcome     `json:"outcome"`
	AtomDecisions        []AtomDecision   `json:"atom_decisions"`
	CitationIDs          []string         `json:"citation_ids"`
	Rationales           []Rationale      `json:"rationales"`
	ConfidencePPM        uint32           `json:"confidence_ppm"`
	Uncertainty          string           `json:"uncertainty,omitempty"`
	ContradictionRefs    []string         `json:"contradiction_refs,omitempty"`
	OmissionRefs         []string         `json:"omission_refs,omitempty"`
	DisagreementRefs     []string         `json:"disagreement_refs,omitempty"`
	AppealRefs           []string         `json:"appeal_refs,omitempty"`
	ErrorCodes           []JudgeErrorCode `json:"error_codes,omitempty"`
}

func blank(value string) bool { return strings.TrimSpace(value) == "" }
func validVerdict(value Verdict) bool {
	return value == VerdictSupported || value == VerdictRebutted || value == VerdictInsufficientEvidence || value == VerdictBlocked || value == VerdictDisputed
}
func validOutcome(value JudgeOutcome) bool {
	return value == OutcomeVerified || value == OutcomeRejected || value == OutcomeInsufficientEvidence || value == OutcomeBlocked || value == OutcomeDisputed
}
func validJudgeErrorCode(value JudgeErrorCode) bool {
	return value == ErrorCapabilityMismatch || value == ErrorModalityMissing || value == ErrorEvidenceMissing ||
		value == ErrorStaleRevision || value == ErrorHashMismatch || value == ErrorScopeMismatch ||
		value == ErrorInformationSetDrift || value == ErrorPromptInjectionSuspected ||
		value == ErrorBudgetExhausted || value == ErrorJudgeUnavailable
}
func validModality(value Modality) bool {
	return value == ModalityStaticImage || value == ModalitySynchronizedVideo || value == ModalityStructuredData || value == ModalityBoundedText || value == ModalityActionReceipt || value == ModalityCustodyEvent || value == ModalitySourceSnapshot
}
func validModalityStatus(value ModalityStatus) bool {
	return value == ModalitySatisfied || value == ModalityInsufficient || value == ModalityBlocked || value == ModalityNotApplicable
}
func validRequirementShape(value ModalityRequirement) bool {
	return value.Schema == JudgeModalitySchema && !blank(value.RequirementID) && !blank(value.AtomRef) &&
		validModality(value.Modality) && validModalityStatus(value.Status) && !hasBlankOrDuplicate(value.CitationIDs) &&
		!(value.Required && value.Status == ModalityNotApplicable) &&
		(value.Status != ModalitySatisfied || len(value.CitationIDs) > 0)
}
func validLocator(value CitationLocator) bool {
	if value.Type == LocatorTextLines || value.Type == LocatorMediaTime {
		return value.Start >= 0 && value.End > value.Start
	}
	if value.Type == LocatorImageRegion {
		return value.X >= 0 && value.Y >= 0 && value.Width > 0 && value.Height > 0 && value.X+value.Width <= 1 && value.Y+value.Height <= 1
	}
	return (value.Type == LocatorClaimID || value.Type == LocatorAssetID || value.Type == LocatorReceiptRef || value.Type == LocatorJSONPointer) && !blank(value.Value) || value.Type == LocatorCustodyOrdinal && value.Start >= 0
}
func hasVerdict(seen map[Verdict]struct{}, value Verdict) bool { _, ok := seen[value]; return ok }
func hasBlankOrDuplicate(values []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if blank(value) {
			return true
		}
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
func hasBlankOrDuplicateOmissions(values []Omission) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if blank(value.Ref) || blank(value.ReasonCode) {
			return true
		}
		if _, ok := seen[value.Ref]; ok {
			return true
		}
		seen[value.Ref] = struct{}{}
	}
	return false
}
func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
func allExist[T any](values []string, set map[string]T) bool {
	for _, value := range values {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}
func intersects(left, right []string) bool {
	set := stringSet(left)
	for _, value := range right {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
