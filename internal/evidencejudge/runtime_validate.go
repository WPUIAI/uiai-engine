package evidencejudge

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
)

func ValidateExecutionPlanAt(plan JudgeExecutionPlan, now time.Time) error {
	if plan.Schema != JudgeExecutionPlanSchema || !validAssignmentRef(plan.PlanID) || !validAssignmentArtifact(plan.Artifact) ||
		!validAssignmentRef(plan.ViewRef) || !validAssignmentSHA256(plan.ViewSHA256) ||
		!validAssignmentRef(plan.InformationSetRef) || !validAssignmentSHA256(plan.InformationSetSHA256) ||
		!validAssignmentRef(plan.PolicyRef) || !validAssignmentRef(plan.PolicyRevision) ||
		!validQuorumMode(plan.QuorumMode) || len(plan.JudgeOrder) == 0 || len(plan.JudgeOrder) > MaxJudgeCandidates ||
		assignmentHasBadRefs(plan.JudgeOrder, MaxJudgeCandidates, false) || !sort.StringsAreSorted(plan.JudgeOrder) || plan.Threshold == 0 ||
		plan.Threshold > uint32(len(plan.JudgeOrder)) || plan.MinimumDistinctProviders == 0 ||
		plan.MinimumDistinctProviders > uint32(len(plan.JudgeOrder)) || assignmentHasBadRefs(plan.DisallowedCorrelationRefs, MaxAssignmentRefs, true) ||
		plan.MaxAttemptsPerJudge == 0 || plan.MaxAttemptsPerJudge > 3 || plan.RetryCooldownMS > MaxJudgeRetryCooldownMS ||
		!validAssignmentBudget(plan.TotalBudget) || plan.EvaluationAt.IsZero() || !plan.Deadline.After(plan.EvaluationAt) ||
		plan.MaxConsecutiveFailures == 0 || plan.MaxConsecutiveFailures > uint32(len(plan.JudgeOrder))*plan.MaxAttemptsPerJudge ||
		!validAssignmentRef(plan.IdempotencyRef) || !validAssignmentRef(plan.AppealPolicyRef) || !validAssignmentRef(plan.DriftPolicyRef) {
		return ErrJudgeExecutionPlanInvalid
	}
	if (plan.QuorumMode == QuorumAll || plan.QuorumMode == QuorumUnanimous) && plan.Threshold != uint32(len(plan.JudgeOrder)) {
		return ErrJudgeExecutionPlanInvalid
	}
	if plan.QuorumMode == QuorumThreshold && plan.Threshold*2 <= uint32(len(plan.JudgeOrder)) {
		return ErrJudgeExecutionPlanInvalid
	}
	if now.Before(plan.EvaluationAt) || !now.Before(plan.Deadline) {
		return ErrJudgeExpired
	}
	body, err := json.Marshal(plan)
	if err != nil || len(body) > MaxAssignmentBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func ValidateExecutionAttempt(attempt JudgeExecutionAttempt, plan JudgeExecutionPlan) error {
	if attempt.Schema != JudgeExecutionAttemptSchema || !validAssignmentRef(attempt.AttemptID) ||
		!containsAssignment(plan.JudgeOrder, attempt.AssignmentID) || !validAssignmentRef(attempt.ExecutorRef) ||
		attempt.Ordinal == 0 || attempt.Ordinal > plan.MaxAttemptsPerJudge || !validAttemptStatus(attempt.Status) ||
		(attempt.FailureCode != "" && !validAssignmentRef(attempt.FailureCode)) {
		return ErrJudgeExecutionMalformed
	}
	if attempt.Status == AttemptSucceeded {
		if attempt.FailureCode != "" || !validAssignmentRef(attempt.ResultRef) || !validAssignmentSHA256(attempt.ResultSHA256) ||
			!validAssignmentRef(attempt.ProviderReceiptRef) {
			return ErrJudgeExecutionMalformed
		}
	} else if attempt.FailureCode == "" || attempt.ResultRef != "" || attempt.ResultSHA256 != "" || attempt.ProviderReceiptRef != "" {
		return ErrJudgeExecutionMalformed
	}
	return nil
}

func ValidateQuorumResult(result JudgeQuorumResult, plans ...JudgeExecutionPlan) error {
	if result.Schema != JudgeQuorumResultSchema || !validAssignmentRef(result.QuorumResultID) ||
		!validAssignmentSHA256(result.QuorumResultSHA256) || !validAssignmentRef(result.PlanID) ||
		!validAssignmentSHA256(result.PlanSHA256) || !validQuorumStatus(result.Status) || !validOutcome(result.Outcome) ||
		result.EvaluatedAt.IsZero() || assignmentHasBadRefs(result.FailureCodes, MaxAssignmentRefs, true) {
		return ErrJudgeExecutionMalformed
	}
	if len(plans) > 1 {
		return ErrJudgeExecutionMalformed
	}
	if len(plans) == 1 {
		planDigest, err := ComputeExecutionPlanSHA256(plans[0])
		if err != nil || result.PlanID != plans[0].PlanID || result.PlanSHA256 != planDigest {
			return ErrJudgeExecutionPlanInvalid
		}
		for _, attempt := range result.Attempts {
			if err := ValidateExecutionAttempt(attempt, plans[0]); err != nil {
				return err
			}
		}
	}
	seenResults, seenAssignments, seenJudges := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	seenPrincipals, seenProviders := map[string]struct{}{}, map[string]struct{}{}
	for _, counted := range result.CountedResults {
		if !validAssignmentRef(counted.ResultRef) || !validAssignmentSHA256(counted.ResultSHA256) ||
			!validAssignmentRef(counted.AssignmentID) || !validAssignmentRef(counted.JudgeRef) ||
			!validAssignmentRef(counted.PrincipalRef) || !validAssignmentRef(counted.ProviderRef) || !validAssignmentRef(counted.ModelRef) {
			return ErrJudgeExecutionMalformed
		}
		if _, duplicate := seenResults[counted.ResultRef]; duplicate {
			return ErrJudgeExecutionMalformed
		}
		if _, duplicate := seenAssignments[counted.AssignmentID]; duplicate {
			return ErrJudgeExecutionMalformed
		}
		if _, duplicate := seenJudges[counted.JudgeRef]; duplicate {
			return ErrJudgeExecutionMalformed
		}
		seenResults[counted.ResultRef], seenAssignments[counted.AssignmentID] = struct{}{}, struct{}{}
		seenJudges[counted.JudgeRef], seenPrincipals[counted.PrincipalRef] = struct{}{}, struct{}{}
		seenProviders[counted.ProviderRef] = struct{}{}
	}
	if len(plans) == 1 && result.Status == QuorumMet {
		plan := plans[0]
		if uint32(len(seenProviders)) < plan.MinimumDistinctProviders ||
			(plan.RequireDistinctPrincipals && len(seenPrincipals) != len(result.CountedResults)) {
			return ErrJudgeExecutionPlanInvalid
		}
		for assignmentID := range seenAssignments {
			if !containsAssignment(plan.JudgeOrder, assignmentID) {
				return ErrJudgeExecutionPlanInvalid
			}
		}
	}
	seenAtoms := map[string]struct{}{}
	for _, decision := range result.AtomDecisions {
		if !validAssignmentRef(decision.AtomRef) || !validVerdict(decision.Verdict) || decision.SupportPPM > 1_000_000 ||
			assignmentHasBadRefs(decision.ResultRefs, MaxAssignmentRefs, false) {
			return ErrJudgeExecutionMalformed
		}
		if _, duplicate := seenAtoms[decision.AtomRef]; duplicate {
			return ErrJudgeExecutionMalformed
		}
		seenAtoms[decision.AtomRef] = struct{}{}
		for _, ref := range decision.ResultRefs {
			if _, exists := seenResults[ref]; !exists {
				return ErrJudgeExecutionMalformed
			}
		}
	}
	digest, err := computeQuorumResultSHA256Unchecked(result)
	if err != nil || digest != result.QuorumResultSHA256 {
		return ErrJudgeExecutionMalformed
	}
	return nil
}

func ComputeExecutionPlanSHA256(plan JudgeExecutionPlan) (string, error) {
	if err := ValidateExecutionPlanAt(plan, plan.EvaluationAt); err != nil {
		return "", err
	}
	return digestAssignmentContract(normalizeExecutionPlan(plan))
}

func CanonicalExecutionPlanBytes(plan JudgeExecutionPlan) ([]byte, error) {
	if err := ValidateExecutionPlanAt(plan, plan.EvaluationAt); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeExecutionPlan(plan))
}

func VerifyExecutionPlanSHA256(plan JudgeExecutionPlan, expected string) error {
	digest, err := ComputeExecutionPlanSHA256(plan)
	if err != nil || !validAssignmentSHA256(expected) || digest != expected {
		return ErrJudgeExecutionPlanInvalid
	}
	return nil
}

func ComputeQuorumResultSHA256(result JudgeQuorumResult) (string, error) {
	result.QuorumResultSHA256 = ""
	if err := validateQuorumShapeWithoutDigest(result); err != nil {
		return "", err
	}
	return computeQuorumResultSHA256Unchecked(result)
}

func CanonicalQuorumResultBytes(result JudgeQuorumResult) ([]byte, error) {
	if err := ValidateQuorumResult(result); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeQuorumResult(result))
}

func VerifyQuorumResultSHA256(result JudgeQuorumResult) error {
	return ValidateQuorumResult(result)
}

func ComputeJudgeAppealSHA256(appeal JudgeAppeal) (string, error) {
	appeal.AppealSHA256 = ""
	if err := validateAppealShape(appeal, false); err != nil {
		return "", err
	}
	return digestAssignmentContract(normalizeJudgeAppeal(appeal))
}

func VerifyJudgeAppealSHA256(appeal JudgeAppeal) error {
	if err := validateAppealShape(appeal, true); err != nil {
		return err
	}
	digest, err := ComputeJudgeAppealSHA256(appeal)
	if err != nil || digest != appeal.AppealSHA256 {
		return ErrJudgeAppealInvalid
	}
	return nil
}

func CanonicalJudgeAppealBytes(appeal JudgeAppeal) ([]byte, error) {
	if err := VerifyJudgeAppealSHA256(appeal); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJudgeAppeal(appeal))
}

func validateExecutionInput(input JudgeExecutionInput, plan JudgeExecutionPlan) error {
	if !validAssignmentRef(input.ExecutorRef) || input.View.ViewID != plan.ViewRef ||
		input.Request.ViewRef != plan.ViewRef || input.Request.ViewSHA256 != plan.ViewSHA256 ||
		input.View.InformationSetRef != plan.InformationSetRef || input.View.InformationSetSHA256 != plan.InformationSetSHA256 ||
		input.Request.InformationSetRef != plan.InformationSetRef || input.Request.InformationSetSHA256 != plan.InformationSetSHA256 ||
		!reflect.DeepEqual(input.View.Artifact, plan.Artifact) || !reflect.DeepEqual(input.Assignment.Artifact, plan.Artifact) ||
		input.View.Policy.PolicyRef != plan.PolicyRef || input.View.Policy.PolicyRevision != plan.PolicyRevision ||
		input.Request.AssignmentRef != input.Assignment.AssignmentID || input.Request.VerifierIdentityRef != input.Capability.JudgeIdentityRef {
		return ErrJudgeExecutionPlanInvalid
	}
	viewDigest, err := DigestJudgeView(input.View)
	if err != nil || viewDigest != plan.ViewSHA256 {
		return ErrJudgeExecutionPlanInvalid
	}
	if err := ValidateJudgeRequestAgainst(input.Request, input.View); err != nil {
		return err
	}
	if err := ValidateJudgeAssignmentAt(input.Assignment, input.AssignmentRequest, input.Capability, plan.EvaluationAt); err != nil {
		return err
	}
	if input.AssignmentRequest.ViewRef != plan.ViewRef || input.AssignmentRequest.ViewSHA256 != plan.ViewSHA256 ||
		input.AssignmentRequest.InformationSetRef != plan.InformationSetRef ||
		input.AssignmentRequest.InformationSetSHA256 != plan.InformationSetSHA256 || input.Assignment.Lease.ExpiresAt.Before(plan.Deadline) {
		return ErrJudgeExecutionPlanInvalid
	}
	return nil
}

func validateExecutionResponse(response JudgeExecutionResponse, input JudgeExecutionInput, plan JudgeExecutionPlan, used JudgeUsage) error {
	if !response.OutcomeKnown || !validAssignmentRef(response.ProviderReceiptRef) ||
		!usageWithinBudget(response.Usage, input.Request.Budget) || !usageFitsTotal(used, response.Usage, plan.TotalBudget) {
		return ErrJudgeBudgetExceeded
	}
	if err := ValidateJudgeResultAgainst(response.Result, input.Request, input.View); err != nil {
		return err
	}
	if response.Result.CapabilityDigest != input.Capability.CapabilityDigest ||
		response.Result.ModelProvider != input.Capability.ProviderRef || response.Result.ModelVersion != input.Capability.ModelRef ||
		response.Result.PolicyRevision != plan.PolicyRevision {
		return ErrJudgeDriftDetected
	}
	if response.Result.EvaluatedAt.After(plan.Deadline) {
		return ErrJudgeExpired
	}
	return nil
}

func finalizeQuorumResult(result JudgeQuorumResult, accepted []acceptedRuntimeResult, plan JudgeExecutionPlan, forced QuorumStatus, forcedErr error) (JudgeQuorumResult, error) {
	sort.Slice(accepted, func(i, j int) bool { return accepted[i].result.JudgeIdentityRef < accepted[j].result.JudgeIdentityRef })
	judgeResults := make([]JudgeResult, 0, len(accepted))
	for _, acceptedResult := range accepted {
		judgeResult := acceptedResult.result
		digest, err := DigestJudgeResult(judgeResult)
		if err != nil {
			return JudgeQuorumResult{}, err
		}
		result.CountedResults = append(result.CountedResults, CountedJudgeResult{ResultRef: judgeResult.ResultID,
			ResultSHA256: digest, AssignmentID: acceptedResult.assignmentID, JudgeRef: judgeResult.JudgeIdentityRef,
			PrincipalRef: acceptedResult.principalRef, ProviderRef: acceptedResult.providerRef, ModelRef: acceptedResult.modelRef})
		judgeResults = append(judgeResults, judgeResult)
	}
	sortCountedResults(result.CountedResults)
	status, outcome, decisions := aggregateJudgeResults(judgeResults, plan)
	if forced != "" {
		status = forced
	}
	if status == QuorumMet {
		providers := map[string]struct{}{}
		for _, acceptedResult := range accepted {
			providers[acceptedResult.providerRef] = struct{}{}
		}
		if uint32(len(providers)) < plan.MinimumDistinctProviders {
			status, outcome = QuorumUnmet, OutcomeBlocked
			result.FailureCodes = append(result.FailureCodes, "provider_diversity_unmet")
		}
	}
	result.Status, result.Outcome, result.AtomDecisions = status, outcome, decisions
	result.FailureCodes = uniqueSortedRuntime(result.FailureCodes)
	seed := plan.PlanID + "\x00" + plan.IdempotencyRef
	result.QuorumResultID = "judge-quorum:sha256:" + assignmentDigestString(seed)
	digest, err := ComputeQuorumResultSHA256(result)
	if err != nil {
		return JudgeQuorumResult{}, err
	}
	result.QuorumResultSHA256 = digest
	if forcedErr != nil {
		return result, forcedErr
	}
	switch status {
	case QuorumMet:
		return result, nil
	case QuorumDisputed:
		return result, ErrJudgeDisagreement
	default:
		return result, ErrJudgeQuorumUnmet
	}
}

func aggregateJudgeResults(results []JudgeResult, plan JudgeExecutionPlan) (QuorumStatus, JudgeOutcome, []QuorumAtomDecision) {
	if len(results) == 0 {
		return QuorumBlocked, OutcomeBlocked, nil
	}
	atomRefs := make([]string, 0, len(results[0].AtomDecisions))
	for _, decision := range results[0].AtomDecisions {
		atomRefs = append(atomRefs, decision.AtomRef)
	}
	sort.Strings(atomRefs)
	status := QuorumMet
	decisions := make([]QuorumAtomDecision, 0, len(atomRefs))
	for _, atomRef := range atomRefs {
		counts := map[Verdict][]string{}
		for _, result := range results {
			for _, decision := range result.AtomDecisions {
				if decision.AtomRef == atomRef {
					counts[decision.Verdict] = append(counts[decision.Verdict], result.ResultID)
				}
			}
		}
		var selected Verdict
		var refs []string
		for _, verdict := range []Verdict{VerdictSupported, VerdictRebutted, VerdictInsufficientEvidence, VerdictBlocked, VerdictDisputed} {
			if uint32(len(counts[verdict])) >= plan.Threshold {
				selected, refs = verdict, counts[verdict]
				break
			}
		}
		if selected == "" {
			selected = VerdictDisputed
			if len(counts) > 1 {
				status = QuorumDisputed
			} else if status != QuorumDisputed {
				status = QuorumUnmet
			}
			for _, result := range results {
				refs = append(refs, result.ResultID)
			}
		}
		sort.Strings(refs)
		decisions = append(decisions, QuorumAtomDecision{AtomRef: atomRef, Verdict: selected,
			SupportPPM: uint32(uint64(len(refs)) * 1_000_000 / uint64(len(plan.JudgeOrder))), ResultRefs: refs})
	}
	return status, outcomeForQuorumDecisions(decisions), decisions
}

func outcomeForQuorumDecisions(decisions []QuorumAtomDecision) JudgeOutcome {
	outcome := OutcomeVerified
	for _, decision := range decisions {
		switch decision.Verdict {
		case VerdictRebutted:
			return OutcomeRejected
		case VerdictDisputed:
			outcome = OutcomeDisputed
		case VerdictBlocked:
			if outcome != OutcomeDisputed {
				outcome = OutcomeBlocked
			}
		case VerdictInsufficientEvidence:
			if outcome == OutcomeVerified {
				outcome = OutcomeInsufficientEvidence
			}
		}
	}
	return outcome
}

func validateQuorumShapeWithoutDigest(result JudgeQuorumResult) error {
	placeholder := result.QuorumResultSHA256
	result.QuorumResultSHA256 = assignmentHexForRuntime('a')
	err := ValidateQuorumResultShape(result)
	result.QuorumResultSHA256 = placeholder
	return err
}

func ValidateQuorumResultShape(result JudgeQuorumResult) error {
	if result.Schema != JudgeQuorumResultSchema || !validAssignmentRef(result.QuorumResultID) ||
		!validAssignmentSHA256(result.QuorumResultSHA256) || !validAssignmentRef(result.PlanID) ||
		!validAssignmentSHA256(result.PlanSHA256) || !validQuorumStatus(result.Status) || !validOutcome(result.Outcome) || result.EvaluatedAt.IsZero() {
		return ErrJudgeExecutionMalformed
	}
	return nil
}

func computeQuorumResultSHA256Unchecked(result JudgeQuorumResult) (string, error) {
	result.QuorumResultSHA256 = ""
	return digestAssignmentContract(normalizeQuorumResult(result))
}

func validateAppealShape(appeal JudgeAppeal, requireDigest bool) error {
	if appeal.Schema != JudgeAppealSchema || !validAssignmentRef(appeal.AppealID) ||
		(requireDigest && !validAssignmentSHA256(appeal.AppealSHA256)) ||
		(!requireDigest && appeal.AppealSHA256 != "" && !validAssignmentSHA256(appeal.AppealSHA256)) ||
		!validAssignmentRef(appeal.QuorumResultRef) || !validAssignmentSHA256(appeal.QuorumResultSHA256) ||
		assignmentHasBadRefs(appeal.PriorResultRefs, MaxAssignmentRefs, false) || len(appeal.PriorResultRefs) != len(appeal.PriorResultSHA256s) ||
		!allRuntimeSHA256(appeal.PriorResultSHA256s) || !validAssignmentRef(appeal.PolicyRef) ||
		!validAssignmentRef(appeal.PolicyRevision) || assignmentHasBadRefs(appeal.ArbitratorRefs, MaxAssignmentRefs, false) ||
		appeal.Generation == 0 || !validAssignmentRef(appeal.ReasonCode) || appeal.CreatedAt.IsZero() || !appeal.ExpiresAt.After(appeal.CreatedAt) {
		return ErrJudgeAppealInvalid
	}
	return nil
}

func normalizeExecutionPlan(plan JudgeExecutionPlan) JudgeExecutionPlan {
	body, _ := json.Marshal(plan)
	var out JudgeExecutionPlan
	_ = json.Unmarshal(body, &out)
	normalizeAssignmentArtifact(&out.Artifact)
	return out
}

func normalizeQuorumResult(result JudgeQuorumResult) JudgeQuorumResult {
	body, _ := json.Marshal(result)
	var out JudgeQuorumResult
	_ = json.Unmarshal(body, &out)
	sortCountedResults(out.CountedResults)
	sort.Slice(out.AtomDecisions, func(i, j int) bool { return out.AtomDecisions[i].AtomRef < out.AtomDecisions[j].AtomRef })
	for i := range out.AtomDecisions {
		out.AtomDecisions[i].ResultRefs = sortedAssignmentStrings(out.AtomDecisions[i].ResultRefs)
	}
	out.FailureCodes = uniqueSortedRuntime(out.FailureCodes)
	return out
}

func normalizeJudgeAppeal(appeal JudgeAppeal) JudgeAppeal {
	body, _ := json.Marshal(appeal)
	var out JudgeAppeal
	_ = json.Unmarshal(body, &out)
	out.ArbitratorRefs = sortedAssignmentStrings(out.ArbitratorRefs)
	return out
}

func validQuorumMode(mode QuorumMode) bool {
	return mode == QuorumAll || mode == QuorumUnanimous || mode == QuorumThreshold
}
func validAttemptStatus(status AttemptStatus) bool {
	return status == AttemptSucceeded || status == AttemptFailed || status == AttemptOutcomeUnknown || status == AttemptCanceled
}
func validQuorumStatus(status QuorumStatus) bool {
	return status == QuorumMet || status == QuorumUnmet || status == QuorumDisputed || status == QuorumBlocked
}

func usageWithinBudget(usage JudgeUsage, budget JudgeBudget) bool {
	return usage.Tokens <= budget.MaxTokens && usage.MediaBytes <= budget.MaxMediaBytes &&
		usage.SpendMicros <= budget.MaxSpendMicros && usage.DurationMS <= budget.MaxDurationMS
}

func usageFitsTotal(used, next JudgeUsage, budget JudgeBudget) bool {
	if used.Tokens > budget.MaxTokens || used.MediaBytes > budget.MaxMediaBytes ||
		used.SpendMicros > budget.MaxSpendMicros || used.DurationMS > budget.MaxDurationMS {
		return false
	}
	return next.Tokens <= budget.MaxTokens-used.Tokens && next.MediaBytes <= budget.MaxMediaBytes-used.MediaBytes &&
		next.SpendMicros <= budget.MaxSpendMicros-used.SpendMicros && next.DurationMS <= budget.MaxDurationMS-used.DurationMS
}

func uniqueSortedRuntime(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func allRuntimeSHA256(values []string) bool {
	for _, value := range values {
		if !validAssignmentSHA256(value) {
			return false
		}
	}
	return len(values) > 0
}
func assignmentHexForRuntime(value byte) string { return strings.Repeat(string(value), 64) }
