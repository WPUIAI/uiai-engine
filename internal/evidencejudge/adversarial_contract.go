package evidencejudge

import (
	"errors"
	"time"
)

const (
	AdversarialCorpusSchema    = "uiai.evidence_judge_adversarial_corpus.v1"
	AdversarialCaseSchema      = "uiai.evidence_judge_adversarial_case.v1"
	CalibrationResultSchema    = "uiai.evidence_judge_calibration_result.v1"
	DriftReportSchema          = "uiai.evidence_judge_drift_report.v1"
	MaxAdversarialCases        = 256
	MaxAdversarialPayloadRunes = 4096
	MaxCalibrationValidityMS   = 90 * 24 * 60 * 60 * 1000
)

var (
	ErrAdversarialCorpusInvalid       = errors.New("evidence judge adversarial corpus invalid")
	ErrAdversarialCaseInvalid         = errors.New("evidence judge adversarial case invalid")
	ErrAdversarialExpectationMismatch = errors.New("evidence judge adversarial expectation mismatch")
	ErrCalibrationResultInvalid       = errors.New("evidence judge calibration result invalid")
	ErrCalibrationThresholdUnmet      = errors.New("evidence judge calibration threshold unmet")
	ErrDriftReportInvalid             = errors.New("evidence judge drift report invalid")
)

type MutationClass string

const (
	MutationPromptInjection          MutationClass = "prompt_injection"
	MutationPersuasiveSummary        MutationClass = "persuasive_summary"
	MutationSourceOrder              MutationClass = "source_order"
	MutationCitationOrder            MutationClass = "citation_order"
	MutationResultOrder              MutationClass = "result_order"
	MutationMissingModality          MutationClass = "missing_modality"
	MutationModalitySubstitution     MutationClass = "modality_substitution"
	MutationIdentityMismatch         MutationClass = "identity_mismatch"
	MutationDigestMismatch           MutationClass = "digest_mismatch"
	MutationStaleEvidence            MutationClass = "stale_evidence"
	MutationCitationEscape           MutationClass = "citation_escape"
	MutationCorrelatedVerifier       MutationClass = "correlated_verifier"
	MutationBudgetPressure           MutationClass = "budget_pressure"
	MutationProviderFailure          MutationClass = "provider_failure"
	MutationModelRevision            MutationClass = "model_revision"
	MutationPolicyRevision           MutationClass = "policy_revision"
	MutationHighConsequenceThreshold MutationClass = "high_consequence_threshold"
)

type DriftStatus string

const (
	DriftWithinThreshold DriftStatus = "within_threshold"
	DriftWarning         DriftStatus = "warning"
	DriftBlocked         DriftStatus = "blocked"
)

type RationalThreshold struct {
	Numerator   uint32 `json:"numerator"`
	Denominator uint32 `json:"denominator"`
}

type ProtectedJudgeInvariants struct {
	ScopeSHA256          string    `json:"scope_sha256"`
	RubricRef            string    `json:"rubric_ref"`
	PolicyRef            string    `json:"policy_ref"`
	PolicyRevision       string    `json:"policy_revision"`
	InformationSetSHA256 string    `json:"information_set_sha256"`
	AssignmentRef        string    `json:"assignment_ref"`
	BudgetRef            string    `json:"budget_ref"`
	AuthorityRef         string    `json:"authority_ref"`
	AllowedVerdicts      []Verdict `json:"allowed_verdicts"`
}

type AdversarialAtomExpectation struct {
	AtomRef string  `json:"atom_ref"`
	Verdict Verdict `json:"verdict"`
}

type AdversarialModalityExpectation struct {
	AtomRef  string         `json:"atom_ref"`
	Modality Modality       `json:"modality"`
	Status   ModalityStatus `json:"status"`
}

type AdversarialExpectation struct {
	Outcome                JudgeOutcome                     `json:"outcome"`
	Verdict                Verdict                          `json:"verdict"`
	ErrorCode              JudgeErrorCode                   `json:"error_code,omitempty"`
	OutcomeKnown           bool                             `json:"outcome_known"`
	Atoms                  []AdversarialAtomExpectation     `json:"atoms"`
	Modalities             []AdversarialModalityExpectation `json:"modalities"`
	CitationRefs           []string                         `json:"citation_refs,omitempty"`
	MinimumQuorum          uint32                           `json:"minimum_quorum"`
	MinimumCapabilityClass string                           `json:"minimum_capability_class"`
	MinimumCapabilityRank  uint32                           `json:"minimum_capability_rank"`
}

type AdversarialCase struct {
	Schema                  string                   `json:"schema"`
	CaseID                  string                   `json:"case_id"`
	CorpusRevision          string                   `json:"corpus_revision"`
	PolicyRef               string                   `json:"policy_ref"`
	PolicyRevision          string                   `json:"policy_revision"`
	RubricRef               string                   `json:"rubric_ref"`
	RequiredModelRef        string                   `json:"required_model_ref"`
	RequiredCapabilityClass string                   `json:"required_capability_class"`
	BaseViewSHA256          string                   `json:"base_view_sha256"`
	BaseRequestSHA256       string                   `json:"base_request_sha256"`
	BaseResultSHA256        string                   `json:"base_result_sha256"`
	MutationClass           MutationClass            `json:"mutation_class"`
	Expected                AdversarialExpectation   `json:"expected"`
	Protected               ProtectedJudgeInvariants `json:"protected_invariants"`
	DeterministicSeed       uint64                   `json:"deterministic_seed"`
	HighConsequence         bool                     `json:"high_consequence"`
	FixtureLicenseRef       string                   `json:"fixture_license_ref"`
	ProvenanceRefs          []string                 `json:"provenance_refs"`
	Synthetic               bool                     `json:"synthetic"`
	UntrustedEvidenceData   string                   `json:"untrusted_evidence_data"`
}

type AdversarialCorpus struct {
	Schema                  string            `json:"schema"`
	CorpusRef               string            `json:"corpus_ref"`
	CorpusRevision          string            `json:"corpus_revision"`
	CorpusSHA256            string            `json:"corpus_sha256"`
	PolicyRef               string            `json:"policy_ref"`
	PolicyRevision          string            `json:"policy_revision"`
	RubricRef               string            `json:"rubric_ref"`
	HarnessRef              string            `json:"harness_ref"`
	RequiredModelRef        string            `json:"required_model_ref"`
	RequiredCapabilityClass string            `json:"required_capability_class"`
	PassThreshold           RationalThreshold `json:"pass_threshold"`
	CalibrationValidityMS   uint64            `json:"calibration_validity_ms"`
	FixtureLicenseRef       string            `json:"fixture_license_ref"`
	ProvenanceRefs          []string          `json:"provenance_refs"`
	Cases                   []AdversarialCase `json:"cases"`
}

type ObservedAdversarialOutcome struct {
	CaseID          string                           `json:"case_id"`
	CorpusRevision  string                           `json:"corpus_revision"`
	MutationClass   MutationClass                    `json:"mutation_class"`
	Outcome         JudgeOutcome                     `json:"outcome"`
	Verdict         Verdict                          `json:"verdict"`
	ErrorCode       JudgeErrorCode                   `json:"error_code,omitempty"`
	OutcomeKnown    bool                             `json:"outcome_known"`
	Atoms           []AdversarialAtomExpectation     `json:"atoms"`
	Modalities      []AdversarialModalityExpectation `json:"modalities"`
	CitationRefs    []string                         `json:"citation_refs,omitempty"`
	Protected       ProtectedJudgeInvariants         `json:"protected_invariants"`
	Quorum          uint32                           `json:"quorum"`
	CapabilityClass string                           `json:"capability_class"`
	CapabilityRank  uint32                           `json:"capability_rank"`
}

type CaseEvaluation struct {
	CaseID          string        `json:"case_id"`
	CorpusRevision  string        `json:"corpus_revision"`
	MutationClass   MutationClass `json:"mutation_class"`
	Passed          bool          `json:"passed"`
	HighConsequence bool          `json:"high_consequence"`
	AtomRefs        []string      `json:"atom_refs"`
	Modalities      []Modality    `json:"modalities"`
	CitationRefs    []string      `json:"citation_refs,omitempty"`
	FailureCode     string        `json:"failure_code,omitempty"`
}

type CalibrationCapability struct {
	CapabilityDigest string    `json:"capability_digest"`
	CapabilityClass  string    `json:"capability_class"`
	CapabilityRank   uint32    `json:"capability_rank"`
	JudgeIdentityRef string    `json:"judge_identity_ref"`
	ProviderRef      string    `json:"provider_ref"`
	ModelRef         string    `json:"model_ref"`
	HarnessRef       string    `json:"harness_ref"`
	PolicyRefs       []string  `json:"policy_refs"`
	ValidFrom        time.Time `json:"valid_from"`
	ValidUntil       time.Time `json:"valid_until"`
}

type CalibrationCount struct {
	Key    string `json:"key"`
	Total  uint32 `json:"total"`
	Passed uint32 `json:"passed"`
	Failed uint32 `json:"failed"`
}

type CalibrationResult struct {
	Schema                string             `json:"schema"`
	CalibrationID         string             `json:"calibration_id"`
	CalibrationSHA256     string             `json:"calibration_sha256"`
	CorpusRef             string             `json:"corpus_ref"`
	CorpusRevision        string             `json:"corpus_revision"`
	CorpusSHA256          string             `json:"corpus_sha256"`
	PolicyRef             string             `json:"policy_ref"`
	PolicyRevision        string             `json:"policy_revision"`
	RubricRef             string             `json:"rubric_ref"`
	HarnessRef            string             `json:"harness_ref"`
	CapabilityDigest      string             `json:"capability_digest"`
	CapabilityClass       string             `json:"capability_class"`
	CapabilityRank        uint32             `json:"capability_rank"`
	JudgeIdentityRef      string             `json:"judge_identity_ref"`
	ProviderRef           string             `json:"provider_ref"`
	ModelRef              string             `json:"model_ref"`
	Status                CalibrationStatus  `json:"status"`
	Threshold             RationalThreshold  `json:"threshold"`
	Total                 uint32             `json:"total"`
	Passed                uint32             `json:"passed"`
	Failed                uint32             `json:"failed"`
	HighConsequenceCases  uint32             `json:"high_consequence_cases"`
	HighConsequencePassed uint32             `json:"high_consequence_passed"`
	MutationCounts        []CalibrationCount `json:"mutation_counts"`
	AtomCounts            []CalibrationCount `json:"atom_counts"`
	ModalityCounts        []CalibrationCount `json:"modality_counts"`
	CitationRefs          []string           `json:"citation_refs"`
	OmissionRefs          []string           `json:"omission_refs,omitempty"`
	Uncertainty           string             `json:"uncertainty,omitempty"`
	EvaluatedAt           time.Time          `json:"evaluated_at"`
	ExpiresAt             time.Time          `json:"expires_at"`
}

type CalibrationDriftPolicy struct {
	PolicyRef           string    `json:"policy_ref"`
	PolicyRevision      string    `json:"policy_revision"`
	WarningDropPPM      uint32    `json:"warning_drop_ppm"`
	BlockDropPPM        uint32    `json:"block_drop_ppm"`
	AllowModelRevision  bool      `json:"allow_model_revision"`
	AllowProviderChange bool      `json:"allow_provider_change"`
	AllowPolicyRevision bool      `json:"allow_policy_revision"`
	EvaluatedAt         time.Time `json:"evaluated_at"`
}

type DriftReport struct {
	Schema                string      `json:"schema"`
	DriftReportID         string      `json:"drift_report_id"`
	DriftReportSHA256     string      `json:"drift_report_sha256"`
	BaselineRef           string      `json:"baseline_ref"`
	BaselineSHA256        string      `json:"baseline_sha256"`
	CandidateRef          string      `json:"candidate_ref"`
	CandidateSHA256       string      `json:"candidate_sha256"`
	CorpusRef             string      `json:"corpus_ref"`
	CorpusRevision        string      `json:"corpus_revision"`
	PolicyRef             string      `json:"policy_ref"`
	PolicyRevision        string      `json:"policy_revision"`
	Status                DriftStatus `json:"status"`
	Comparable            bool        `json:"comparable"`
	BaselinePassPPM       uint32      `json:"baseline_pass_ppm"`
	CandidatePassPPM      uint32      `json:"candidate_pass_ppm"`
	DropPPM               uint32      `json:"drop_ppm"`
	ModelRevisionChanged  bool        `json:"model_revision_changed"`
	ProviderChanged       bool        `json:"provider_changed"`
	PolicyRevisionChanged bool        `json:"policy_revision_changed"`
	CapabilityDowngraded  bool        `json:"capability_downgraded"`
	ClassDropPPM          uint32      `json:"class_drop_ppm"`
	ReasonCodes           []string    `json:"reason_codes,omitempty"`
	EvaluatedAt           time.Time   `json:"evaluated_at"`
}
