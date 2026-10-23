package evidencejudge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

type runtimeFakeExecutor struct {
	responses []JudgeExecutionResponse
	errors    []error
	calls     int
	envelopes []FrozenJudgeEnvelope
}

func (executor *runtimeFakeExecutor) ExecuteJudge(_ context.Context, envelope FrozenJudgeEnvelope) (JudgeExecutionResponse, error) {
	executor.envelopes = append(executor.envelopes, envelope)
	index := executor.calls
	executor.calls++
	var response JudgeExecutionResponse
	var err error
	if index < len(executor.responses) {
		response = executor.responses[index]
	}
	if index < len(executor.errors) {
		err = executor.errors[index]
	}
	return response, err
}

type runtimeMutatingExecutor struct {
	response JudgeExecutionResponse
}

func (executor runtimeMutatingExecutor) ExecuteJudge(_ context.Context, envelope FrozenJudgeEnvelope) (JudgeExecutionResponse, error) {
	if len(envelope.Citations) > 0 && len(envelope.Citations[0].SupportsAtoms) > 0 {
		envelope.Citations[0].SupportsAtoms[0] = "atom:hostile-citation-mutation"
	}
	if len(envelope.Request.PolicyRefs) > 0 {
		envelope.Request.PolicyRefs[0] = "policy:hostile-mutation"
	}
	if len(envelope.Request.AcceptanceAtomRefs) > 0 {
		envelope.Request.AcceptanceAtomRefs[0] = "atom:hostile-request-mutation"
	}
	if len(envelope.Request.RequiredModalities) > 0 && len(envelope.Request.RequiredModalities[0].CitationIDs) > 0 {
		envelope.Request.RequiredModalities[0].CitationIDs[0] = "citation:hostile-mutation"
	}
	if len(envelope.Assignment.Artifact.AttestationRefs) > 0 {
		envelope.Assignment.Artifact.AttestationRefs[0] = "attestation:hostile-mutation"
	}
	if len(envelope.Assignment.Artifact.TrustRefs) > 0 {
		envelope.Assignment.Artifact.TrustRefs[0] = "trust:hostile-mutation"
	}
	if len(envelope.Assignment.Artifact.SecurityRefs) > 0 {
		envelope.Assignment.Artifact.SecurityRefs[0] = "security:hostile-mutation"
	}
	if len(envelope.Assignment.IndependenceEvidenceRefs) > 0 {
		envelope.Assignment.IndependenceEvidenceRefs[0] = "independence:hostile-mutation"
	}
	if len(envelope.Assignment.CalibrationRefs) > 0 {
		envelope.Assignment.CalibrationRefs[0] = "calibration:hostile-mutation"
	}
	return executor.response, nil
}

func TestRunJudgeQuorumGoldenDeterministicAndReadOnly(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 2)
	beforeInputs := runtimeJSON(t, inputs)
	beforePlan := runtimeJSON(t, plan)
	executors := runtimeExecutors(inputs, responses)

	result, err := RunJudgeQuorum(context.Background(), plan, inputs, executors)
	if err != nil {
		t.Fatalf("RunJudgeQuorum() result = %#v, error = %v", result, err)
	}
	if result.Status != QuorumMet || result.Outcome != OutcomeVerified || len(result.CountedResults) != 2 {
		t.Fatalf("quorum result = %#v", result)
	}
	if err := ValidateQuorumResult(result, plan); err != nil {
		t.Fatalf("ValidateQuorumResult() error = %v", err)
	}
	if err := VerifyExecutionPlanSHA256(plan, result.PlanSHA256); err != nil {
		t.Fatalf("VerifyExecutionPlanSHA256() error = %v", err)
	}
	if err := VerifyQuorumResultSHA256(result); err != nil {
		t.Fatalf("VerifyQuorumResultSHA256() error = %v", err)
	}
	if runtimeJSON(t, inputs) != beforeInputs || runtimeJSON(t, plan) != beforePlan {
		t.Fatal("runtime mutated frozen inputs")
	}
	for ref, executor := range executors {
		fake := executor.(*runtimeFakeExecutor)
		if len(fake.envelopes) != 1 {
			t.Fatalf("executor %s envelopes = %d", ref, len(fake.envelopes))
		}
		envelope := fake.envelopes[0]
		if envelope.EvidenceClass != "untrusted_evidence_data" || envelope.TrustedPolicyRef != plan.PolicyRef {
			t.Fatalf("unsafe envelope = %#v", envelope)
		}
		body := runtimeJSON(t, envelope)
		if !strings.Contains(body, "ignore_previous_instructions") {
			t.Fatal("prompt-like evidence fixture was not delivered as bounded untrusted data")
		}
		for _, forbidden := range []string{"credential", "private_path", "completion_operation", "provider_close", "settlement"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("envelope contains forbidden field %q", forbidden)
			}
		}
	}

	body := append([]byte(runtimeJSON(t, result)), '\n')
	fixturePath := "testdata/quorum-run.golden.json"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(fixturePath, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(body, want) {
		t.Fatalf("golden drift\n got: %s\nwant: %s", body, want)
	}

	slices.Reverse(inputs)
	for i := 0; i < 30; i++ {
		replay, err := RunJudgeQuorum(context.Background(), plan, inputs, runtimeExecutors(inputs, responses))
		if err != nil || replay.QuorumResultSHA256 != result.QuorumResultSHA256 {
			t.Fatalf("replay %d = %s, %v", i, replay.QuorumResultSHA256, err)
		}
	}
}

func TestRunJudgeQuorumFreezesNestedExecutorEnvelope(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 1)
	before := runtimeJSON(t, inputs)
	executor := runtimeMutatingExecutor{response: responses[0]}

	result, err := RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: executor})
	if err != nil || result.Status != QuorumMet {
		t.Fatalf("mutating executor result = %#v, error = %v", result, err)
	}
	if got := runtimeJSON(t, inputs); got != before {
		t.Fatalf("executor mutated frozen caller input\n got: %s\nwant: %s", got, before)
	}
}

func TestRunJudgeQuorumDisagreementBuildsImmutableAppeal(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 2)
	responses[1].Result.AtomDecisions[0].Verdict = VerdictRebutted
	responses[1].Result.AtomDecisions[0].ReasonCode = "cited_mismatch"
	responses[1].Result.Outcome = OutcomeRejected

	result, err := RunJudgeQuorum(context.Background(), plan, inputs, runtimeExecutors(inputs, responses))
	if !errors.Is(err, ErrJudgeDisagreement) || result.Status != QuorumDisputed || result.Outcome != OutcomeDisputed {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	policy := AppealPolicy{PolicyRef: "appeal-policy:strict", PolicyRevision: "1", ArbitratorRefs: []string{"judge:arbitrator-b", "judge:arbitrator-a"},
		MaxAppealGeneration: 1, Deadline: plan.Deadline.Add(time.Hour)}
	appeal, err := BuildJudgeAppeal(result, policy, plan.EvaluationAt)
	if err != nil {
		t.Fatalf("BuildJudgeAppeal() error = %v", err)
	}
	if err := VerifyJudgeAppealSHA256(appeal); err != nil {
		t.Fatalf("VerifyJudgeAppealSHA256() error = %v", err)
	}
	tampered := appeal
	tampered.PriorResultSHA256s[0] = runtimeHex('f')
	if err := VerifyJudgeAppealSHA256(tampered); !errors.Is(err, ErrJudgeAppealInvalid) {
		t.Fatalf("tampered appeal error = %v", err)
	}
}

func TestRunJudgeQuorumRetriesOnlySafeKnownFailures(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 1)
	plan.MaxAttemptsPerJudge = 2
	plan.MaxConsecutiveFailures = 2
	fake := &runtimeFakeExecutor{responses: []JudgeExecutionResponse{{}, responses[0]}, errors: []error{ErrJudgeExecutorUnavailable, nil}}
	result, err := RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: fake})
	if err != nil || result.Status != QuorumMet || len(result.Attempts) != 2 || fake.calls != 2 {
		t.Fatalf("safe retry result = %#v, calls = %d, error = %v", result, fake.calls, err)
	}

	unknown := &runtimeFakeExecutor{responses: []JudgeExecutionResponse{{OutcomeKnown: false}, responses[0]}}
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: unknown})
	if !errors.Is(err, ErrJudgeOutcomeUnknown) || result.Status != QuorumBlocked || unknown.calls != 1 ||
		len(result.Attempts) != 1 || result.Attempts[0].Status != AttemptOutcomeUnknown {
		t.Fatalf("unknown result = %#v, calls = %d, error = %v", result, unknown.calls, err)
	}

	unknownError := &runtimeFakeExecutor{errors: []error{ErrJudgeOutcomeUnknown, nil}, responses: []JudgeExecutionResponse{{}, responses[0]}}
	_, _ = RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: unknownError})
	if unknownError.calls != 1 {
		t.Fatalf("outcome-unknown error retried %d times", unknownError.calls)
	}

	effectBearingError := &runtimeFakeExecutor{errors: []error{ErrJudgeExecutorUnavailable, nil}, responses: []JudgeExecutionResponse{responses[0], responses[0]}}
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: effectBearingError})
	if !errors.Is(err, ErrJudgeOutcomeUnknown) || effectBearingError.calls != 1 || len(result.Attempts) != 1 ||
		result.Attempts[0].Status != AttemptOutcomeUnknown || result.Attempts[0].ProviderReceiptRef != responses[0].ProviderReceiptRef ||
		result.Attempts[0].Usage != responses[0].Usage || result.Usage != responses[0].Usage {
		t.Fatalf("effect-bearing error result = %#v, calls = %d, error = %v", result, effectBearingError.calls, err)
	}

	unboundedEffect := responses[0]
	unboundedEffect.ProviderReceiptRef = "invalid receipt ref"
	unboundedEffect.Usage.Tokens = plan.TotalBudget.MaxTokens + 1
	unboundedError := &runtimeFakeExecutor{errors: []error{ErrJudgeExecutorUnavailable, nil}, responses: []JudgeExecutionResponse{unboundedEffect, responses[0]}}
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{inputs[0].ExecutorRef: unboundedError})
	if !errors.Is(err, ErrJudgeOutcomeUnknown) || unboundedError.calls != 1 || len(result.Attempts) != 1 ||
		result.Attempts[0].Status != AttemptOutcomeUnknown || result.Attempts[0].ProviderReceiptRef != "" ||
		result.Attempts[0].Usage != (JudgeUsage{}) || result.Usage != (JudgeUsage{}) {
		t.Fatalf("unbounded effect metadata result = %#v, calls = %d, error = %v", result, unboundedError.calls, err)
	}
}

func TestRunJudgeQuorumEnforcesCircuitBudgetDriftAndCancellation(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 1)
	plan.MaxConsecutiveFailures = 1
	result, err := RunJudgeQuorum(context.Background(), plan, inputs, map[string]JudgeExecutor{})
	if !errors.Is(err, ErrJudgeCircuitOpen) || result.Status != QuorumBlocked {
		t.Fatalf("circuit result = %#v, error = %v", result, err)
	}

	overBudget := responses[0]
	overBudget.Usage.Tokens = inputs[0].Request.Budget.MaxTokens + 1
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, runtimeExecutors(inputs, []JudgeExecutionResponse{overBudget}))
	if !errors.Is(err, ErrJudgeCircuitOpen) || !slices.Contains(result.FailureCodes, "budget_exhausted") {
		t.Fatalf("budget result = %#v, error = %v", result, err)
	}

	drift := responses[0]
	drift.Result.ModelVersion = "model:drifted"
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, runtimeExecutors(inputs, []JudgeExecutionResponse{drift}))
	if !errors.Is(err, ErrJudgeCircuitOpen) || !slices.Contains(result.FailureCodes, "drift_detected") {
		t.Fatalf("drift result = %#v, error = %v", result, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = RunJudgeQuorum(ctx, plan, inputs, runtimeExecutors(inputs, responses))
	if !errors.Is(err, ErrJudgeExecutorUnavailable) || len(result.Attempts) != 1 || result.Attempts[0].Status != AttemptCanceled {
		t.Fatalf("canceled result = %#v, error = %v", result, err)
	}
}

func TestRunJudgeQuorumThresholdDiversityBudgetAndMalformedResult(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 3)
	baseline, err := RunJudgeQuorum(context.Background(), plan, inputs, runtimeExecutors(inputs, responses))
	if err != nil {
		t.Fatalf("baseline quorum error = %v", err)
	}
	for index, permutation := range runtimePermutations(inputs) {
		replay, err := RunJudgeQuorum(context.Background(), plan, permutation, runtimeExecutors(permutation, responses))
		if err != nil || replay.QuorumResultSHA256 != baseline.QuorumResultSHA256 {
			t.Fatalf("permutation %d digest = %s, error = %v", index, replay.QuorumResultSHA256, err)
		}
	}

	plan.QuorumMode, plan.Threshold, plan.MinimumDistinctProviders = QuorumThreshold, 2, 2
	executors := runtimeExecutors(inputs, responses)
	delete(executors, inputs[2].ExecutorRef)
	result, err := RunJudgeQuorum(context.Background(), plan, inputs, executors)
	if err != nil || result.Status != QuorumMet || len(result.CountedResults) != 2 {
		t.Fatalf("threshold result = %#v, error = %v", result, err)
	}

	unanimousPlan := plan
	unanimousPlan.QuorumMode, unanimousPlan.Threshold, unanimousPlan.MinimumDistinctProviders = QuorumUnanimous, 3, 3
	unanimous, err := RunJudgeQuorum(context.Background(), unanimousPlan, inputs, runtimeExecutors(inputs, responses))
	if err != nil || unanimous.Status != QuorumMet || len(unanimous.CountedResults) != 3 {
		t.Fatalf("unanimous result = %#v, error = %v", unanimous, err)
	}

	plan.MinimumDistinctProviders = 3
	executors = runtimeExecutors(inputs, responses)
	delete(executors, inputs[2].ExecutorRef)
	result, err = RunJudgeQuorum(context.Background(), plan, inputs, executors)
	if !errors.Is(err, ErrJudgeQuorumUnmet) || result.Status != QuorumUnmet ||
		!slices.Contains(result.FailureCodes, "provider_diversity_unmet") {
		t.Fatalf("diversity result = %#v, error = %v", result, err)
	}

	oneInput, oneResponse, onePlan := runtimeTestInputs(t, 1)
	onePlan.TotalBudget.MaxTokens = oneInput[0].Request.Budget.MaxTokens - 1
	fake := &runtimeFakeExecutor{responses: oneResponse}
	result, err = RunJudgeQuorum(context.Background(), onePlan, oneInput, map[string]JudgeExecutor{oneInput[0].ExecutorRef: fake})
	if !errors.Is(err, ErrJudgeCircuitOpen) || fake.calls != 0 || !slices.Contains(result.FailureCodes, "budget_exhausted") {
		t.Fatalf("pre-call budget result = %#v, calls = %d, error = %v", result, fake.calls, err)
	}

	badResponse := oneResponse[0]
	badResponse.Result.AtomDecisions[0].CitationIDs = []string{"citation:missing"}
	result, err = RunJudgeQuorum(context.Background(), onePlanWithFullBudget(oneInput, onePlan), oneInput,
		runtimeExecutors(oneInput, []JudgeExecutionResponse{badResponse}))
	if !errors.Is(err, ErrJudgeCircuitOpen) || !slices.Contains(result.FailureCodes, "result_invalid") {
		t.Fatalf("malformed result = %#v, error = %v", result, err)
	}

	correlatedInputs, correlatedResponses, correlatedPlan := runtimeTestInputs(t, 2)
	correlatedPlan.DisallowedCorrelationRefs = []string{correlatedInputs[0].Capability.CorrelationRefs[0]}
	correlatedExecutors := runtimeExecutors(correlatedInputs, correlatedResponses)
	if _, err := RunJudgeQuorum(context.Background(), correlatedPlan, correlatedInputs, correlatedExecutors); !errors.Is(err, ErrJudgeExecutionPlanInvalid) {
		t.Fatalf("correlation error = %v", err)
	}
	for ref, executor := range correlatedExecutors {
		if calls := executor.(*runtimeFakeExecutor).calls; calls != 0 {
			t.Fatalf("correlated executor %s received %d calls", ref, calls)
		}
	}
}

func TestRuntimeRejectsPlanInputResultAndPolicyDrift(t *testing.T) {
	inputs, responses, plan := runtimeTestInputs(t, 1)

	badThreshold := plan
	badThreshold.Threshold = 2
	if _, err := RunJudgeQuorum(context.Background(), badThreshold, inputs, runtimeExecutors(inputs, responses)); !errors.Is(err, ErrJudgeExecutionPlanInvalid) {
		t.Fatalf("threshold error = %v", err)
	}

	badView := append([]JudgeExecutionInput(nil), inputs...)
	badView[0].Request.InformationSetSHA256 = runtimeHex('e')
	if _, err := RunJudgeQuorum(context.Background(), plan, badView, runtimeExecutors(badView, responses)); !errors.Is(err, ErrJudgeCircuitOpen) {
		t.Fatalf("input drift error = %v", err)
	}

	unsafe := plan
	unsafe.AppealPolicyRef = "https://user:secret@example.test/policy?token=x#fragment"
	if err := ValidateExecutionPlanAt(unsafe, unsafe.EvaluationAt); !errors.Is(err, ErrJudgeExecutionPlanInvalid) {
		t.Fatalf("unsafe plan error = %v", err)
	}

	_, _, unsorted := runtimeTestInputs(t, 2)
	slices.Reverse(unsorted.JudgeOrder)
	if err := ValidateExecutionPlanAt(unsorted, unsorted.EvaluationAt); !errors.Is(err, ErrJudgeExecutionPlanInvalid) {
		t.Fatalf("unsorted plan error = %v", err)
	}

	unboundedAttempt := newRuntimeAttempt(inputs[0], 1, AttemptOutcomeUnknown, "outcome_unknown")
	unboundedAttempt.Usage.Tokens = plan.TotalBudget.MaxTokens + 1
	if err := ValidateExecutionAttempt(unboundedAttempt, plan); !errors.Is(err, ErrJudgeExecutionMalformed) {
		t.Fatalf("unbounded attempt error = %v", err)
	}
}

func runtimeTestInputs(t *testing.T, count int) ([]JudgeExecutionInput, []JudgeExecutionResponse, JudgeExecutionPlan) {
	t.Helper()
	baseView, baseRequest, baseResult := validContracts(t)
	baseView.Artifact.Scope.WorkItemRef = "work-item:t06d"
	baseView.Omissions[0].ReasonCode = "ignore_previous_instructions"
	viewDigest, err := DigestJudgeView(baseView)
	if err != nil {
		t.Fatal(err)
	}
	baseRequest.ViewSHA256 = viewDigest
	inputs := make([]JudgeExecutionInput, 0, count)
	responses := make([]JudgeExecutionResponse, 0, count)
	created := baseView.CreatedAt
	for i := 0; i < count; i++ {
		suffix := string(rune('a' + i))
		request := deepCopy(baseRequest)
		request.RequestID = "judge-request:" + suffix
		request.IdempotencyRef = "idempotency:judge-" + suffix
		request.VerifierIdentityRef = "agent:judge-" + suffix
		capability := assignmentTestCapability(t, request.VerifierIdentityRef, "principal:judge-"+suffix, "instance:judge-"+suffix)
		capability.ProviderRef = "provider:example-" + suffix
		capability.ModelRef = "model:multimodal-v1"
		capability.CorrelationRefs = []string{"correlation:judge-" + suffix}
		capability.SupportedModalities = []Modality{ModalityStaticImage, ModalitySynchronizedVideo}
		capability.SupportedResultDetail = []string{"standard"}
		capability.PolicyRefs = []string{baseView.Policy.PolicyRef}
		capability.MaxBudget = request.Budget
		capability.ValidFrom = created.Add(-time.Minute)
		capability.ValidUntil = created.Add(50 * time.Minute)
		capability.Calibration.ValidUntil = created.Add(50 * time.Minute)
		capability = assignmentSealCapability(t, capability)

		assignmentRequest := assignmentTestRequest()
		assignmentRequest.RequestID = "assignment-request:" + suffix
		assignmentRequest.IdempotencyRef = "idempotency:assignment-" + suffix
		assignmentRequest.Artifact = baseView.Artifact
		assignmentRequest.ViewRef, assignmentRequest.ViewSHA256 = baseView.ViewID, viewDigest
		assignmentRequest.InformationSetRef, assignmentRequest.InformationSetSHA256 = baseView.InformationSetRef, baseView.InformationSetSHA256
		assignmentRequest.ExecutorIdentityRef = request.ExecutorIdentityRef
		assignmentRequest.ExecutorPrincipalRef = "principal:executor"
		assignmentRequest.ExecutorInstanceRef = "instance:executor"
		assignmentRequest.ExecutorCorrelationRefs = []string{"provider:executor"}
		assignmentRequest.VerifierPolicyRef, assignmentRequest.VerifierPolicyRevision = baseView.Policy.PolicyRef, baseView.Policy.PolicyRevision
		assignmentRequest.RequiredModalities = []Modality{ModalityStaticImage, ModalitySynchronizedVideo}
		assignmentRequest.ResultDetail = request.ResultDetail
		assignmentRequest.Budget = request.Budget
		assignmentRequest.RequestedAt = created.Add(time.Minute)
		assignmentRequest.ExpiresAt = created.Add(40 * time.Minute)
		assignmentRequest.LeaseDurationMS = uint64((30 * time.Minute).Milliseconds())
		assignmentRequest.DisallowedCorrelationRefs = []string{"provider:executor"}
		if err := ValidateJudgeAssignmentRequest(assignmentRequest); err != nil {
			t.Fatalf("ValidateJudgeAssignmentRequest() error = %v", err)
		}
		if err := ValidateJudgeCapabilityAt(capability, assignmentRequest.RequestedAt); err != nil {
			t.Fatalf("ValidateJudgeCapabilityAt() error = %v", err)
		}
		leaseExpires := assignmentRequest.RequestedAt.Add(time.Duration(assignmentRequest.LeaseDurationMS) * time.Millisecond)
		if err := admitJudgeCandidate(assignmentRequest, capability, leaseExpires); err != nil {
			t.Fatalf("admitJudgeCandidate() error = %v", err)
		}
		assignment, err := AssignJudge(assignmentRequest, []JudgeCapability{capability})
		if err != nil {
			t.Fatalf("AssignJudge() error = %v", err)
		}
		request.AssignmentRef = assignment.AssignmentID
		requestDigest, err := DigestJudgeRequest(request)
		if err != nil {
			t.Fatal(err)
		}
		result := deepCopy(baseResult)
		result.ResultID = "judge-result:" + suffix
		result.RequestID, result.RequestSHA256 = request.RequestID, requestDigest
		result.ViewRef, result.ViewSHA256 = request.ViewRef, request.ViewSHA256
		result.JudgeIdentityRef = request.VerifierIdentityRef
		result.CapabilityDigest = capability.CapabilityDigest
		result.ModelProvider, result.ModelVersion = capability.ProviderRef, capability.ModelRef
		inputs = append(inputs, JudgeExecutionInput{ExecutorRef: "executor:judge-" + suffix, View: baseView, Request: request,
			AssignmentRequest: assignmentRequest, Assignment: assignment, Capability: capability})
		responses = append(responses, JudgeExecutionResponse{Result: result,
			Usage:              JudgeUsage{Tokens: 1000, MediaBytes: 1024, SpendMicros: 1000, DurationMS: 1000},
			ProviderReceiptRef: "provider-receipt:" + suffix, OutcomeKnown: true})
	}
	judgeOrder := make([]string, len(inputs))
	for i := range inputs {
		judgeOrder[i] = inputs[i].Assignment.AssignmentID
	}
	sortStrings(judgeOrder)
	plan := JudgeExecutionPlan{Schema: JudgeExecutionPlanSchema, PlanID: "judge-plan:epwa-t06d", Artifact: baseView.Artifact,
		ViewRef: baseView.ViewID, ViewSHA256: viewDigest, InformationSetRef: baseView.InformationSetRef, InformationSetSHA256: baseView.InformationSetSHA256,
		PolicyRef: baseView.Policy.PolicyRef, PolicyRevision: baseView.Policy.PolicyRevision, QuorumMode: QuorumAll,
		Threshold: uint32(count), JudgeOrder: judgeOrder, MinimumDistinctProviders: uint32(count), RequireDistinctPrincipals: true,
		DisallowedCorrelationRefs: []string{"provider:executor"}, MaxAttemptsPerJudge: 1, RetryCooldownMS: 1000,
		TotalBudget:  JudgeBudget{MaxTokens: uint64(count) * 8000, MaxMediaBytes: uint64(count) * 20 << 20, MaxSpendMicros: uint64(count) * 500000, MaxDurationMS: uint64(count) * 300000},
		EvaluationAt: created.Add(5 * time.Minute), Deadline: created.Add(20 * time.Minute), MaxConsecutiveFailures: uint32(count),
		IdempotencyRef: "idempotency:quorum", AppealPolicyRef: "appeal-policy:strict", DriftPolicyRef: "drift-policy:strict"}
	return inputs, responses, plan
}

func runtimePermutations(inputs []JudgeExecutionInput) [][]JudgeExecutionInput {
	var out [][]JudgeExecutionInput
	var visit func(int)
	values := append([]JudgeExecutionInput(nil), inputs...)
	visit = func(index int) {
		if index == len(values) {
			out = append(out, append([]JudgeExecutionInput(nil), values...))
			return
		}
		for candidate := index; candidate < len(values); candidate++ {
			values[index], values[candidate] = values[candidate], values[index]
			visit(index + 1)
			values[index], values[candidate] = values[candidate], values[index]
		}
	}
	visit(0)
	return out
}

func onePlanWithFullBudget(inputs []JudgeExecutionInput, plan JudgeExecutionPlan) JudgeExecutionPlan {
	plan.TotalBudget = inputs[0].Request.Budget
	return plan
}

func runtimeExecutors(inputs []JudgeExecutionInput, responses []JudgeExecutionResponse) map[string]JudgeExecutor {
	byJudge := map[string]JudgeExecutionResponse{}
	for _, response := range responses {
		byJudge[response.Result.JudgeIdentityRef] = response
	}
	out := map[string]JudgeExecutor{}
	for _, input := range inputs {
		out[input.ExecutorRef] = &runtimeFakeExecutor{responses: []JudgeExecutionResponse{byJudge[input.Capability.JudgeIdentityRef]}}
	}
	return out
}

func sortStrings(values []string)  { slices.Sort(values) }
func runtimeHex(value byte) string { return strings.Repeat(string(value), 64) }

func runtimeJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
