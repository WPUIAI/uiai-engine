package evidencejudge

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"
)

const (
	JudgeExecutionPlanSchema    = "uiai.evidence_judge_execution_plan.v1"
	JudgeExecutionAttemptSchema = "uiai.evidence_judge_execution_attempt.v1"
	JudgeQuorumResultSchema     = "uiai.evidence_judge_quorum_result.v1"
	JudgeAppealSchema           = "uiai.evidence_judge_appeal.v1"
	MaxJudgeRetryCooldownMS     = 24 * 60 * 60 * 1000
)

var (
	ErrJudgeExecutionPlanInvalid = errors.New("evidence judge execution plan invalid")
	ErrJudgeExecutorUnavailable  = errors.New("evidence judge executor unavailable")
	ErrJudgeExecutionMalformed   = errors.New("evidence judge execution malformed")
	ErrJudgeOutcomeUnknown       = errors.New("evidence judge outcome unknown")
	ErrJudgeQuorumUnmet          = errors.New("evidence judge quorum unmet")
	ErrJudgeDisagreement         = errors.New("evidence judge disagreement")
	ErrJudgeCircuitOpen          = errors.New("evidence judge circuit open")
	ErrJudgeDriftDetected        = errors.New("evidence judge drift detected")
	ErrJudgeAppealInvalid        = errors.New("evidence judge appeal invalid")
)

type QuorumMode string

const (
	QuorumAll       QuorumMode = "all"
	QuorumUnanimous QuorumMode = "unanimous"
	QuorumThreshold QuorumMode = "threshold"
)

type AttemptStatus string

const (
	AttemptSucceeded      AttemptStatus = "succeeded"
	AttemptFailed         AttemptStatus = "failed"
	AttemptOutcomeUnknown AttemptStatus = "outcome_unknown"
	AttemptCanceled       AttemptStatus = "canceled"
)

type QuorumStatus string

const (
	QuorumMet      QuorumStatus = "met"
	QuorumUnmet    QuorumStatus = "unmet"
	QuorumDisputed QuorumStatus = "disputed"
	QuorumBlocked  QuorumStatus = "blocked"
)

type JudgeUsage struct {
	Tokens      uint64 `json:"tokens"`
	MediaBytes  uint64 `json:"media_bytes"`
	SpendMicros uint64 `json:"spend_micros"`
	DurationMS  uint64 `json:"duration_ms"`
}

type JudgeExecutionPlan struct {
	Schema                    string          `json:"schema"`
	PlanID                    string          `json:"plan_id"`
	Artifact                  ArtifactBinding `json:"artifact"`
	ViewRef                   string          `json:"view_ref"`
	ViewSHA256                string          `json:"view_sha256"`
	InformationSetRef         string          `json:"information_set_ref"`
	InformationSetSHA256      string          `json:"information_set_sha256"`
	PolicyRef                 string          `json:"policy_ref"`
	PolicyRevision            string          `json:"policy_revision"`
	QuorumMode                QuorumMode      `json:"quorum_mode"`
	Threshold                 uint32          `json:"threshold"`
	JudgeOrder                []string        `json:"judge_order"`
	MinimumDistinctProviders  uint32          `json:"minimum_distinct_providers"`
	RequireDistinctPrincipals bool            `json:"require_distinct_principals"`
	DisallowedCorrelationRefs []string        `json:"disallowed_correlation_refs,omitempty"`
	MaxAttemptsPerJudge       uint32          `json:"max_attempts_per_judge"`
	RetryCooldownMS           uint64          `json:"retry_cooldown_ms"`
	TotalBudget               JudgeBudget     `json:"total_budget"`
	EvaluationAt              time.Time       `json:"evaluation_at"`
	Deadline                  time.Time       `json:"deadline"`
	MaxConsecutiveFailures    uint32          `json:"max_consecutive_failures"`
	IdempotencyRef            string          `json:"idempotency_ref"`
	AppealPolicyRef           string          `json:"appeal_policy_ref"`
	DriftPolicyRef            string          `json:"drift_policy_ref"`
}

type JudgeExecutionInput struct {
	ExecutorRef       string                 `json:"executor_ref"`
	View              JudgeView              `json:"view"`
	Request           JudgeRequest           `json:"request"`
	AssignmentRequest JudgeAssignmentRequest `json:"assignment_request"`
	Assignment        JudgeAssignment        `json:"assignment"`
	Capability        JudgeCapability        `json:"capability"`
}

type FrozenJudgeEnvelope struct {
	ViewRef               string           `json:"view_ref"`
	ViewSHA256            string           `json:"view_sha256"`
	InformationSetRef     string           `json:"information_set_ref"`
	InformationSetSHA256  string           `json:"information_set_sha256"`
	Sources               []EvidenceSource `json:"sources"`
	Citations             []Citation       `json:"citations"`
	Omissions             []Omission       `json:"omissions,omitempty"`
	Request               JudgeRequest     `json:"request"`
	Assignment            JudgeAssignment  `json:"assignment"`
	CapabilityDigest      string           `json:"capability_digest"`
	TrustedPolicyRef      string           `json:"trusted_policy_ref"`
	TrustedPolicyRevision string           `json:"trusted_policy_revision"`
	EvidenceClass         string           `json:"evidence_class"`
	EvaluationAt          time.Time        `json:"evaluation_at"`
}

type JudgeExecutionResponse struct {
	Result             JudgeResult `json:"result"`
	Usage              JudgeUsage  `json:"usage"`
	ProviderReceiptRef string      `json:"provider_receipt_ref"`
	OutcomeKnown       bool        `json:"outcome_known"`
}

type JudgeExecutor interface {
	ExecuteJudge(context.Context, FrozenJudgeEnvelope) (JudgeExecutionResponse, error)
}

type JudgeExecutionAttempt struct {
	Schema             string        `json:"schema"`
	AttemptID          string        `json:"attempt_id"`
	AssignmentID       string        `json:"assignment_id"`
	ExecutorRef        string        `json:"executor_ref"`
	Ordinal            uint32        `json:"ordinal"`
	Status             AttemptStatus `json:"status"`
	FailureCode        string        `json:"failure_code,omitempty"`
	ResultRef          string        `json:"result_ref,omitempty"`
	ResultSHA256       string        `json:"result_sha256,omitempty"`
	ProviderReceiptRef string        `json:"provider_receipt_ref,omitempty"`
	Usage              JudgeUsage    `json:"usage"`
}

type CountedJudgeResult struct {
	ResultRef    string `json:"result_ref"`
	ResultSHA256 string `json:"result_sha256"`
	AssignmentID string `json:"assignment_id"`
	JudgeRef     string `json:"judge_ref"`
	PrincipalRef string `json:"principal_ref"`
	ProviderRef  string `json:"provider_ref"`
	ModelRef     string `json:"model_ref"`
}

type acceptedRuntimeResult struct {
	result       JudgeResult
	assignmentID string
	principalRef string
	providerRef  string
	modelRef     string
}

type QuorumAtomDecision struct {
	AtomRef    string   `json:"atom_ref"`
	Verdict    Verdict  `json:"verdict"`
	SupportPPM uint32   `json:"support_ppm"`
	ResultRefs []string `json:"result_refs"`
}

type JudgeQuorumResult struct {
	Schema             string                  `json:"schema"`
	QuorumResultID     string                  `json:"quorum_result_id"`
	QuorumResultSHA256 string                  `json:"quorum_result_sha256,omitempty"`
	PlanID             string                  `json:"plan_id"`
	PlanSHA256         string                  `json:"plan_sha256"`
	Status             QuorumStatus            `json:"status"`
	Outcome            JudgeOutcome            `json:"outcome"`
	Attempts           []JudgeExecutionAttempt `json:"attempts"`
	CountedResults     []CountedJudgeResult    `json:"counted_results"`
	AtomDecisions      []QuorumAtomDecision    `json:"atom_decisions"`
	Usage              JudgeUsage              `json:"usage"`
	EvaluatedAt        time.Time               `json:"evaluated_at"`
	FailureCodes       []string                `json:"failure_codes,omitempty"`
}

type AppealPolicy struct {
	PolicyRef           string    `json:"policy_ref"`
	PolicyRevision      string    `json:"policy_revision"`
	ArbitratorRefs      []string  `json:"arbitrator_refs"`
	MaxAppealGeneration uint64    `json:"max_appeal_generation"`
	Deadline            time.Time `json:"deadline"`
}

type JudgeAppeal struct {
	Schema             string    `json:"schema"`
	AppealID           string    `json:"appeal_id"`
	AppealSHA256       string    `json:"appeal_sha256,omitempty"`
	QuorumResultRef    string    `json:"quorum_result_ref"`
	QuorumResultSHA256 string    `json:"quorum_result_sha256"`
	PriorResultRefs    []string  `json:"prior_result_refs"`
	PriorResultSHA256s []string  `json:"prior_result_sha256s"`
	PolicyRef          string    `json:"policy_ref"`
	PolicyRevision     string    `json:"policy_revision"`
	ArbitratorRefs     []string  `json:"arbitrator_refs"`
	Generation         uint64    `json:"generation"`
	ReasonCode         string    `json:"reason_code"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

func RunJudgeQuorum(ctx context.Context, plan JudgeExecutionPlan, inputs []JudgeExecutionInput, executors map[string]JudgeExecutor) (JudgeQuorumResult, error) {
	if err := ValidateExecutionPlanAt(plan, plan.EvaluationAt); err != nil {
		return JudgeQuorumResult{}, err
	}
	planDigest, err := ComputeExecutionPlanSHA256(plan)
	if err != nil {
		return JudgeQuorumResult{}, err
	}
	byAssignment := make(map[string]JudgeExecutionInput, len(inputs))
	for _, input := range inputs {
		if _, duplicate := byAssignment[input.Assignment.AssignmentID]; duplicate {
			return JudgeQuorumResult{}, ErrJudgeExecutionPlanInvalid
		}
		byAssignment[input.Assignment.AssignmentID] = input
	}
	if len(byAssignment) != len(plan.JudgeOrder) || validateExecutionCohort(plan, byAssignment) != nil {
		return JudgeQuorumResult{}, ErrJudgeExecutionPlanInvalid
	}
	result := JudgeQuorumResult{Schema: JudgeQuorumResultSchema, PlanID: plan.PlanID, PlanSHA256: planDigest, EvaluatedAt: plan.EvaluationAt.UTC()}
	var accepted []acceptedRuntimeResult
	var consecutiveFailures uint32
	for _, assignmentID := range plan.JudgeOrder {
		input, ok := byAssignment[assignmentID]
		if !ok {
			return JudgeQuorumResult{}, ErrJudgeExecutionPlanInvalid
		}
		if err := validateExecutionInput(input, plan); err != nil {
			result.FailureCodes = append(result.FailureCodes, runtimeFailureCode(err))
			consecutiveFailures++
			if consecutiveFailures >= plan.MaxConsecutiveFailures {
				return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeCircuitOpen)
			}
			continue
		}
		executor, ok := executors[input.ExecutorRef]
		if !ok || executor == nil {
			result.Attempts = append(result.Attempts, newRuntimeAttempt(input, 1, AttemptFailed, "executor_unavailable"))
			result.FailureCodes = append(result.FailureCodes, "executor_unavailable")
			consecutiveFailures++
			if consecutiveFailures >= plan.MaxConsecutiveFailures {
				return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeCircuitOpen)
			}
			continue
		}
		for ordinal := uint32(1); ordinal <= plan.MaxAttemptsPerJudge; ordinal++ {
			if !usageFitsTotal(result.Usage, budgetAsUsage(input.Request.Budget), plan.TotalBudget) {
				result.Attempts = append(result.Attempts, newRuntimeAttempt(input, ordinal, AttemptFailed, "budget_exhausted"))
				result.FailureCodes = append(result.FailureCodes, "budget_exhausted")
				consecutiveFailures++
				break
			}
			if err := ctx.Err(); err != nil {
				result.Attempts = append(result.Attempts, newRuntimeAttempt(input, ordinal, AttemptCanceled, "canceled"))
				return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeExecutorUnavailable)
			}
			envelope := buildFrozenEnvelope(input, plan)
			response, callErr := executor.ExecuteJudge(ctx, envelope)
			attempt := newRuntimeAttempt(input, ordinal, AttemptFailed, "executor_unavailable")
			if callErr != nil {
				attempt.FailureCode = runtimeFailureCode(callErr)
				if errors.Is(callErr, ErrJudgeOutcomeUnknown) {
					attempt.Status = AttemptOutcomeUnknown
				}
				result.Attempts = append(result.Attempts, attempt)
				result.FailureCodes = append(result.FailureCodes, attempt.FailureCode)
				consecutiveFailures++
				if errors.Is(callErr, ErrJudgeOutcomeUnknown) {
					return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeOutcomeUnknown)
				}
				if consecutiveFailures >= plan.MaxConsecutiveFailures ||
					!errors.Is(callErr, ErrJudgeExecutorUnavailable) || ordinal == plan.MaxAttemptsPerJudge {
					break
				}
				continue
			}
			if !response.OutcomeKnown {
				attempt.Status, attempt.FailureCode = AttemptOutcomeUnknown, "outcome_unknown"
				result.Attempts = append(result.Attempts, attempt)
				result.FailureCodes = append(result.FailureCodes, "outcome_unknown")
				return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeOutcomeUnknown)
			}
			if err := validateExecutionResponse(response, input, plan, result.Usage); err != nil {
				attempt.FailureCode = runtimeFailureCode(err)
				result.Attempts = append(result.Attempts, attempt)
				result.FailureCodes = append(result.FailureCodes, attempt.FailureCode)
				consecutiveFailures++
				break
			}
			resultDigest, _ := DigestJudgeResult(response.Result)
			attempt.Status, attempt.FailureCode = AttemptSucceeded, ""
			attempt.ResultRef, attempt.ResultSHA256 = response.Result.ResultID, resultDigest
			attempt.ProviderReceiptRef, attempt.Usage = response.ProviderReceiptRef, response.Usage
			result.Attempts = append(result.Attempts, attempt)
			result.Usage = addJudgeUsage(result.Usage, response.Usage)
			accepted = append(accepted, acceptedRuntimeResult{result: response.Result, assignmentID: input.Assignment.AssignmentID,
				principalRef: input.Capability.PrincipalRef, providerRef: input.Capability.ProviderRef, modelRef: input.Capability.ModelRef})
			consecutiveFailures = 0
			break
		}
		if consecutiveFailures >= plan.MaxConsecutiveFailures {
			return finalizeQuorumResult(result, accepted, plan, QuorumBlocked, ErrJudgeCircuitOpen)
		}
	}
	return finalizeQuorumResult(result, accepted, plan, "", nil)
}

func BuildJudgeAppeal(result JudgeQuorumResult, policy AppealPolicy, now time.Time) (JudgeAppeal, error) {
	if err := ValidateQuorumResult(result); err != nil || (result.Status != QuorumDisputed && result.Status != QuorumUnmet) ||
		!validAssignmentRef(policy.PolicyRef) || !validAssignmentRef(policy.PolicyRevision) ||
		assignmentHasBadRefs(policy.ArbitratorRefs, MaxAssignmentRefs, false) || policy.MaxAppealGeneration == 0 ||
		!policy.Deadline.After(now) {
		return JudgeAppeal{}, ErrJudgeAppealInvalid
	}
	appeal := JudgeAppeal{Schema: JudgeAppealSchema, QuorumResultRef: result.QuorumResultID,
		QuorumResultSHA256: result.QuorumResultSHA256, PolicyRef: policy.PolicyRef, PolicyRevision: policy.PolicyRevision,
		ArbitratorRefs: sortedAssignmentStrings(policy.ArbitratorRefs), Generation: 1, ReasonCode: "quorum_disputed",
		CreatedAt: now.UTC(), ExpiresAt: policy.Deadline.UTC()}
	for _, counted := range result.CountedResults {
		appeal.PriorResultRefs = append(appeal.PriorResultRefs, counted.ResultRef)
		appeal.PriorResultSHA256s = append(appeal.PriorResultSHA256s, counted.ResultSHA256)
	}
	appeal.AppealID = "judge-appeal:sha256:" + assignmentDigestString(result.QuorumResultSHA256+"\x00"+policy.PolicyRef)
	digest, err := ComputeJudgeAppealSHA256(appeal)
	if err != nil {
		return JudgeAppeal{}, err
	}
	appeal.AppealSHA256 = digest
	return appeal, nil
}

func buildFrozenEnvelope(input JudgeExecutionInput, plan JudgeExecutionPlan) FrozenJudgeEnvelope {
	return FrozenJudgeEnvelope{ViewRef: input.View.ViewID, ViewSHA256: input.Request.ViewSHA256,
		InformationSetRef: input.View.InformationSetRef, InformationSetSHA256: input.View.InformationSetSHA256,
		Sources: append([]EvidenceSource(nil), input.View.Sources...), Citations: append([]Citation(nil), input.View.Citations...),
		Omissions: append([]Omission(nil), input.View.Omissions...), Request: input.Request, Assignment: input.Assignment,
		CapabilityDigest: input.Capability.CapabilityDigest, TrustedPolicyRef: plan.PolicyRef,
		TrustedPolicyRevision: plan.PolicyRevision, EvidenceClass: "untrusted_evidence_data", EvaluationAt: plan.EvaluationAt.UTC()}
}

func newRuntimeAttempt(input JudgeExecutionInput, ordinal uint32, status AttemptStatus, code string) JudgeExecutionAttempt {
	seed := input.Assignment.AssignmentID + "\x00" + input.ExecutorRef + "\x00" + strconv.FormatUint(uint64(ordinal), 10)
	return JudgeExecutionAttempt{Schema: JudgeExecutionAttemptSchema, AttemptID: "judge-attempt:sha256:" + assignmentDigestString(seed),
		AssignmentID: input.Assignment.AssignmentID, ExecutorRef: input.ExecutorRef, Ordinal: ordinal, Status: status, FailureCode: code}
}

func addJudgeUsage(left, right JudgeUsage) JudgeUsage {
	return JudgeUsage{Tokens: left.Tokens + right.Tokens, MediaBytes: left.MediaBytes + right.MediaBytes,
		SpendMicros: left.SpendMicros + right.SpendMicros, DurationMS: left.DurationMS + right.DurationMS}
}

func runtimeFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrJudgeOutcomeUnknown):
		return "outcome_unknown"
	case errors.Is(err, ErrJudgeBudgetExceeded):
		return "budget_exhausted"
	case errors.Is(err, ErrJudgeDriftDetected):
		return "drift_detected"
	case errors.Is(err, ErrJudgeExpired):
		return "expired"
	case errors.Is(err, ErrJudgeResultInvalid):
		return "result_invalid"
	case errors.Is(err, ErrInformationSetMismatch):
		return "information_set_mismatch"
	case errors.Is(err, ErrJudgeAssignmentInvalid), errors.Is(err, ErrJudgeAssignmentMismatch):
		return "assignment_invalid"
	case errors.Is(err, ErrJudgeExecutorUnavailable):
		return "executor_unavailable"
	default:
		return "execution_malformed"
	}
}

func sortCountedResults(values []CountedJudgeResult) {
	sort.Slice(values, func(i, j int) bool { return values[i].JudgeRef < values[j].JudgeRef })
}

func budgetAsUsage(budget JudgeBudget) JudgeUsage {
	return JudgeUsage{Tokens: budget.MaxTokens, MediaBytes: budget.MaxMediaBytes,
		SpendMicros: budget.MaxSpendMicros, DurationMS: budget.MaxDurationMS}
}

func validateExecutionCohort(plan JudgeExecutionPlan, inputs map[string]JudgeExecutionInput) error {
	providers, principals, executors := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	judges, instances, correlations := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, assignmentID := range plan.JudgeOrder {
		input, ok := inputs[assignmentID]
		if !ok || input.Assignment.AssignmentID != assignmentID {
			return ErrJudgeExecutionPlanInvalid
		}
		if _, duplicate := executors[input.ExecutorRef]; duplicate {
			return ErrJudgeExecutionPlanInvalid
		}
		executors[input.ExecutorRef], providers[input.Capability.ProviderRef] = struct{}{}, struct{}{}
		if _, duplicate := judges[input.Capability.JudgeIdentityRef]; duplicate {
			return ErrJudgeExecutionPlanInvalid
		}
		judges[input.Capability.JudgeIdentityRef] = struct{}{}
		_, duplicatePrincipal := principals[input.Capability.PrincipalRef]
		_, duplicateInstance := instances[input.Capability.InstanceRef]
		if plan.RequireDistinctPrincipals && (duplicatePrincipal || duplicateInstance) {
			return ErrJudgeExecutionPlanInvalid
		}
		principals[input.Capability.PrincipalRef], instances[input.Capability.InstanceRef] = struct{}{}, struct{}{}
		for _, ref := range input.Capability.CorrelationRefs {
			if containsAssignment(plan.DisallowedCorrelationRefs, ref) {
				return ErrJudgeExecutionPlanInvalid
			}
			if _, duplicate := correlations[ref]; duplicate && plan.RequireDistinctPrincipals {
				return ErrJudgeExecutionPlanInvalid
			}
			correlations[ref] = struct{}{}
		}
	}
	if uint32(len(providers)) < plan.MinimumDistinctProviders {
		return ErrJudgeExecutionPlanInvalid
	}
	return nil
}
