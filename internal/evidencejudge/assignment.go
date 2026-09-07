package evidencejudge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

const (
	JudgeCapabilitySchema        = "uiai.evidence_judge_capability.v1"
	JudgeAssignmentRequestSchema = "uiai.evidence_judge_assignment_request.v1"
	JudgeAssignmentSchema        = "uiai.evidence_judge_assignment.v1"

	MaxJudgeCandidates       = 64
	MaxAssignmentRefs        = 128
	MaxAssignmentBytes       = 1 << 20
	MaxAssignmentLeaseMillis = 24 * 60 * 60 * 1000
)

var (
	ErrJudgeCapabilityInvalid     = errors.New("evidence judge capability invalid")
	ErrJudgeCapabilityMismatch    = errors.New("evidence judge capability mismatch")
	ErrJudgeIndependenceViolation = errors.New("evidence judge independence violation")
	ErrJudgeCalibrationInvalid    = errors.New("evidence judge calibration invalid")
	ErrJudgeAuthorityMissing      = errors.New("evidence judge assignment authority missing")
	ErrJudgeAutonomousIneligible  = errors.New("evidence judge autonomous execution ineligible")
	ErrJudgeCandidateUnavailable  = errors.New("evidence judge candidate unavailable")
	ErrJudgeAssignmentMismatch    = errors.New("evidence judge assignment mismatch")
)

type CapabilityStatus string

const (
	CapabilityEligible CapabilityStatus = "eligible"
	CapabilityBlocked  CapabilityStatus = "blocked"
	CapabilityRevoked  CapabilityStatus = "revoked"
)

type CalibrationStatus string

const (
	CalibrationPassed  CalibrationStatus = "passed"
	CalibrationFailed  CalibrationStatus = "failed"
	CalibrationExpired CalibrationStatus = "expired"
)

type AssignmentStatus string

const (
	AssignmentAppointed            AssignmentStatus = "appointed"
	AssignmentBlocked              AssignmentStatus = "blocked"
	AssignmentAutonomousIneligible AssignmentStatus = "autonomous_ineligible"
)

type JudgeCalibration struct {
	CorpusRef  string            `json:"corpus_ref"`
	ResultRef  string            `json:"result_ref"`
	Revision   string            `json:"revision"`
	Status     CalibrationStatus `json:"status"`
	ValidUntil time.Time         `json:"valid_until"`
}

type JudgeCapability struct {
	Schema                string           `json:"schema"`
	CapabilityDigest      string           `json:"capability_digest,omitempty"`
	JudgeIdentityRef      string           `json:"judge_identity_ref"`
	PrincipalRef          string           `json:"principal_ref"`
	InstanceRef           string           `json:"instance_ref"`
	HarnessRef            string           `json:"harness_ref"`
	ProviderRef           string           `json:"provider_ref"`
	ModelRef              string           `json:"model_ref"`
	SupportedModalities   []Modality       `json:"supported_modalities"`
	SupportedResultDetail []string         `json:"supported_result_detail"`
	Calibration           JudgeCalibration `json:"calibration"`
	IndependenceRefs      []string         `json:"independence_refs"`
	CorrelationRefs       []string         `json:"correlation_refs,omitempty"`
	PolicyRefs            []string         `json:"policy_refs"`
	Status                CapabilityStatus `json:"status"`
	MaxBudget             JudgeBudget      `json:"max_budget"`
	ValidFrom             time.Time        `json:"valid_from"`
	ValidUntil            time.Time        `json:"valid_until"`
}

type JudgeAssignmentRequest struct {
	Schema                    string          `json:"schema"`
	RequestID                 string          `json:"request_id"`
	IdempotencyRef            string          `json:"idempotency_ref"`
	Artifact                  ArtifactBinding `json:"artifact"`
	ViewRef                   string          `json:"view_ref"`
	ViewSHA256                string          `json:"view_sha256"`
	InformationSetRef         string          `json:"information_set_ref"`
	InformationSetSHA256      string          `json:"information_set_sha256"`
	ExecutorIdentityRef       string          `json:"executor_identity_ref"`
	ExecutorPrincipalRef      string          `json:"executor_principal_ref"`
	ExecutorInstanceRef       string          `json:"executor_instance_ref"`
	ExecutorCorrelationRefs   []string        `json:"executor_correlation_refs,omitempty"`
	VerifierPolicyRef         string          `json:"verifier_policy_ref"`
	VerifierPolicyRevision    string          `json:"verifier_policy_revision"`
	RequiredModalities        []Modality      `json:"required_modalities"`
	ResultDetail              string          `json:"result_detail"`
	Budget                    JudgeBudget     `json:"budget"`
	AssignmentAuthorityRef    string          `json:"assignment_authority_ref"`
	DisallowedCorrelationRefs []string        `json:"disallowed_correlation_refs,omitempty"`
	RequestedAt               time.Time       `json:"requested_at"`
	ExpiresAt                 time.Time       `json:"expires_at"`
	LeaseDurationMS           uint64          `json:"lease_duration_ms"`
	MaxCandidates             uint32          `json:"max_candidates"`
	IndependenceRequired      bool            `json:"independence_required"`
	AutonomousEligible        bool            `json:"autonomous_eligible"`
	HumanAuthorityRequired    bool            `json:"human_authority_required"`
}

type JudgeLease struct {
	LeaseID    string    `json:"lease_id"`
	Generation uint64    `json:"generation"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type JudgeAssignment struct {
	Schema                   string           `json:"schema"`
	AssignmentID             string           `json:"assignment_id"`
	AssignmentSHA256         string           `json:"assignment_sha256,omitempty"`
	RequestID                string           `json:"request_id"`
	RequestSHA256            string           `json:"request_sha256"`
	Artifact                 ArtifactBinding  `json:"artifact"`
	ViewRef                  string           `json:"view_ref"`
	ViewSHA256               string           `json:"view_sha256"`
	InformationSetRef        string           `json:"information_set_ref"`
	InformationSetSHA256     string           `json:"information_set_sha256"`
	JudgeIdentityRef         string           `json:"judge_identity_ref,omitempty"`
	JudgePrincipalRef        string           `json:"judge_principal_ref,omitempty"`
	JudgeInstanceRef         string           `json:"judge_instance_ref,omitempty"`
	CapabilityDigest         string           `json:"capability_digest,omitempty"`
	VerifierPolicyRef        string           `json:"verifier_policy_ref"`
	VerifierPolicyRevision   string           `json:"verifier_policy_revision"`
	AssignmentAuthorityRef   string           `json:"assignment_authority_ref"`
	Lease                    JudgeLease       `json:"lease"`
	Budget                   JudgeBudget      `json:"budget"`
	IndependenceEvidenceRefs []string         `json:"independence_evidence_refs,omitempty"`
	CalibrationRefs          []string         `json:"calibration_refs,omitempty"`
	SelectionReason          string           `json:"selection_reason"`
	Status                   AssignmentStatus `json:"status"`
}

func AssignJudge(request JudgeAssignmentRequest, candidates []JudgeCapability) (JudgeAssignment, error) {
	if err := ValidateJudgeAssignmentRequest(request); err != nil {
		return JudgeAssignment{}, err
	}
	requestDigest, err := ComputeJudgeAssignmentRequestDigest(request)
	if err != nil {
		return JudgeAssignment{}, err
	}
	if !request.AutonomousEligible || request.HumanAuthorityRequired {
		blocked := blockedAssignment(request, requestDigest, AssignmentAutonomousIneligible, "external_human_authority_required")
		return blocked, ErrJudgeAutonomousIneligible
	}
	if len(candidates) == 0 || len(candidates) > int(request.MaxCandidates) {
		return blockedAssignment(request, requestDigest, AssignmentBlocked, "candidate_unavailable"), ErrJudgeCandidateUnavailable
	}
	ordered := append([]JudgeCapability(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].JudgeIdentityRef == ordered[j].JudgeIdentityRef {
			return ordered[i].CapabilityDigest < ordered[j].CapabilityDigest
		}
		return ordered[i].JudgeIdentityRef < ordered[j].JudgeIdentityRef
	})
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].JudgeIdentityRef == ordered[i].JudgeIdentityRef {
			return blockedAssignment(request, requestDigest, AssignmentBlocked, "ambiguous_candidate_identity"), ErrJudgeAssignmentInvalid
		}
	}
	var sawCapability, sawIndependence, sawCalibration, sawExpired, sawBudget bool
	leaseExpires := request.RequestedAt.Add(time.Duration(request.LeaseDurationMS) * time.Millisecond)
	for _, capability := range ordered {
		admission := admitJudgeCandidate(request, capability, leaseExpires)
		switch {
		case admission == nil:
			return appointedAssignment(request, requestDigest, capability, leaseExpires)
		case errors.Is(admission, ErrJudgeIndependenceViolation):
			sawIndependence = true
		case errors.Is(admission, ErrJudgeCalibrationInvalid):
			sawCalibration = true
		case errors.Is(admission, ErrJudgeExpired):
			sawExpired = true
		case errors.Is(admission, ErrJudgeCapabilityMismatch):
			sawCapability = true
		case errors.Is(admission, ErrJudgeBudgetExceeded):
			sawBudget = true
		}
	}
	blocked := blockedAssignment(request, requestDigest, AssignmentBlocked, "no_eligible_candidate")
	switch {
	case sawIndependence:
		return blocked, ErrJudgeIndependenceViolation
	case sawCalibration:
		return blocked, ErrJudgeCalibrationInvalid
	case sawExpired:
		return blocked, ErrJudgeExpired
	case sawBudget:
		return blocked, ErrJudgeBudgetExceeded
	case sawCapability:
		return blocked, ErrJudgeCapabilityMismatch
	default:
		return blocked, ErrJudgeCandidateUnavailable
	}
}

func appointedAssignment(request JudgeAssignmentRequest, requestDigest string, capability JudgeCapability, leaseExpires time.Time) (JudgeAssignment, error) {
	seed := assignmentDigestString(requestDigest + "\x00" + capability.CapabilityDigest)
	assignment := JudgeAssignment{
		Schema:                   JudgeAssignmentSchema,
		AssignmentID:             "judge-assignment:sha256:" + seed,
		RequestID:                request.RequestID,
		RequestSHA256:            requestDigest,
		Artifact:                 request.Artifact,
		ViewRef:                  request.ViewRef,
		ViewSHA256:               request.ViewSHA256,
		InformationSetRef:        request.InformationSetRef,
		InformationSetSHA256:     request.InformationSetSHA256,
		JudgeIdentityRef:         capability.JudgeIdentityRef,
		JudgePrincipalRef:        capability.PrincipalRef,
		JudgeInstanceRef:         capability.InstanceRef,
		CapabilityDigest:         capability.CapabilityDigest,
		VerifierPolicyRef:        request.VerifierPolicyRef,
		VerifierPolicyRevision:   request.VerifierPolicyRevision,
		AssignmentAuthorityRef:   request.AssignmentAuthorityRef,
		Lease:                    JudgeLease{LeaseID: "judge-lease:sha256:" + seed, Generation: 1, IssuedAt: request.RequestedAt.UTC(), ExpiresAt: leaseExpires.UTC()},
		Budget:                   request.Budget,
		IndependenceEvidenceRefs: sortedAssignmentStrings(capability.IndependenceRefs),
		CalibrationRefs:          []string{capability.Calibration.CorpusRef, capability.Calibration.ResultRef},
		SelectionReason:          "stable_first_eligible",
		Status:                   AssignmentAppointed,
	}
	digest, err := ComputeJudgeAssignmentDigest(assignment)
	if err != nil {
		return JudgeAssignment{}, err
	}
	assignment.AssignmentSHA256 = digest
	return assignment, nil
}

func blockedAssignment(request JudgeAssignmentRequest, requestDigest string, status AssignmentStatus, reason string) JudgeAssignment {
	return JudgeAssignment{
		Schema: JudgeAssignmentSchema, AssignmentID: "judge-assignment:blocked:" + assignmentDigestString(requestDigest+"\x00"+reason),
		RequestID: request.RequestID, RequestSHA256: requestDigest, Artifact: request.Artifact,
		ViewRef: request.ViewRef, ViewSHA256: request.ViewSHA256, InformationSetRef: request.InformationSetRef,
		InformationSetSHA256: request.InformationSetSHA256, VerifierPolicyRef: request.VerifierPolicyRef,
		VerifierPolicyRevision: request.VerifierPolicyRevision, AssignmentAuthorityRef: request.AssignmentAuthorityRef,
		SelectionReason: reason, Status: status,
	}
}

func admitJudgeCandidate(request JudgeAssignmentRequest, capability JudgeCapability, leaseExpires time.Time) error {
	if err := ValidateJudgeCapabilityAt(capability, request.RequestedAt); err != nil {
		return err
	}
	if capability.ValidUntil.Before(leaseExpires) || capability.Calibration.ValidUntil.Before(leaseExpires) {
		return ErrJudgeExpired
	}
	if request.IndependenceRequired && (capability.JudgeIdentityRef == request.ExecutorIdentityRef ||
		capability.PrincipalRef == request.ExecutorPrincipalRef || capability.InstanceRef == request.ExecutorInstanceRef) {
		return ErrJudgeIndependenceViolation
	}
	if intersectsAssignment(capability.CorrelationRefs, request.ExecutorCorrelationRefs) ||
		intersectsAssignment(capability.CorrelationRefs, request.DisallowedCorrelationRefs) {
		return ErrJudgeIndependenceViolation
	}
	if !containsAssignment(capability.PolicyRefs, request.VerifierPolicyRef) ||
		!containsAssignment(capability.SupportedResultDetail, request.ResultDetail) ||
		!modalitiesSupported(capability.SupportedModalities, request.RequiredModalities) {
		return ErrJudgeCapabilityMismatch
	}
	if !budgetWithin(request.Budget, capability.MaxBudget) {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func ComputeJudgeCapabilityDigest(capability JudgeCapability) (string, error) {
	if err := validateJudgeCapabilityShape(capability, false); err != nil {
		return "", err
	}
	capability.CapabilityDigest = ""
	return digestAssignmentContract(normalizeJudgeCapability(capability))
}

func VerifyJudgeCapabilityDigest(capability JudgeCapability) error {
	if !validAssignmentSHA256(capability.CapabilityDigest) {
		return ErrJudgeCapabilityInvalid
	}
	digest, err := ComputeJudgeCapabilityDigest(capability)
	if err != nil || digest != capability.CapabilityDigest {
		return ErrJudgeCapabilityInvalid
	}
	return nil
}

func ComputeJudgeAssignmentRequestDigest(request JudgeAssignmentRequest) (string, error) {
	if err := ValidateJudgeAssignmentRequest(request); err != nil {
		return "", err
	}
	return digestAssignmentContract(normalizeJudgeAssignmentRequest(request))
}

func ComputeJudgeAssignmentDigest(assignment JudgeAssignment) (string, error) {
	if err := validateJudgeAssignmentShape(assignment, false); err != nil {
		return "", err
	}
	assignment.AssignmentSHA256 = ""
	return digestAssignmentContract(normalizeJudgeAssignment(assignment))
}

func VerifyJudgeAssignmentDigest(assignment JudgeAssignment) error {
	if !validAssignmentSHA256(assignment.AssignmentSHA256) {
		return ErrJudgeAssignmentMismatch
	}
	digest, err := ComputeJudgeAssignmentDigest(assignment)
	if err != nil || digest != assignment.AssignmentSHA256 {
		return ErrJudgeAssignmentMismatch
	}
	return nil
}

func CanonicalJudgeCapabilityBytes(capability JudgeCapability) ([]byte, error) {
	if err := validateJudgeCapabilityShape(capability, true); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJudgeCapability(capability))
}

func CanonicalJudgeAssignmentRequestBytes(request JudgeAssignmentRequest) ([]byte, error) {
	if err := ValidateJudgeAssignmentRequest(request); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJudgeAssignmentRequest(request))
}

func CanonicalJudgeAssignmentBytes(assignment JudgeAssignment) ([]byte, error) {
	if err := validateJudgeAssignmentShape(assignment, true); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJudgeAssignment(assignment))
}

func digestAssignmentContract(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxAssignmentBytes {
		return "", ErrJudgeBudgetExceeded
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func sortedAssignmentStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func containsAssignment(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intersectsAssignment(left, right []string) bool {
	for _, value := range left {
		if containsAssignment(right, value) {
			return true
		}
	}
	return false
}

func modalitiesSupported(supported, required []Modality) bool {
	for _, modality := range required {
		found := false
		for _, candidate := range supported {
			found = found || candidate == modality
		}
		if !found {
			return false
		}
	}
	return true
}

func budgetWithin(want, limit JudgeBudget) bool {
	return want.MaxTokens <= limit.MaxTokens && want.MaxMediaBytes <= limit.MaxMediaBytes &&
		want.MaxSpendMicros <= limit.MaxSpendMicros && want.MaxDurationMS <= limit.MaxDurationMS
}
