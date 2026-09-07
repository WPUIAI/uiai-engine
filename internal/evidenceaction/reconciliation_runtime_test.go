package evidenceaction

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExecutionRuntimeRequiresReconciliationBeforeUnknownOutcomeRetry(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	nonceIndex := 0
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) {
		nonceIndex++
		if nonceIndex == 1 {
			return "nonce:unknown-first", nil
		}
		return "nonce:unknown-retry", nil
	})
	effects := []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append_capture"}}
	preview, err := previews.BuildPreview(proposal, approval, effects, RiskModerate, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := runtimeConfirmation(t, proposal, preview, now)
	runtime := NewExecutionRuntime()
	calls := 0
	unknownExecutor := ActionExecutorFunc(func(context.Context, ActionProposal, *ActionPreview) (ActionResult, error) {
		calls++
		return ActionResult{}, errors.New("provider detail must not escape: secret-value")
	})
	if _, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, unknownExecutor); !errors.Is(err, ErrOutcomeUnknown) || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("first Execute() error = %v", err)
	}
	if _, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, unknownExecutor); !errors.Is(err, ErrOutcomeUnknown) {
		t.Fatalf("unreconciled replay error = %v, want ErrOutcomeUnknown", err)
	}
	if calls != 1 {
		t.Fatalf("executor calls before reconciliation = %d, want 1", calls)
	}

	observed := successfulRuntimeResult(t, proposal, preview, now)
	observed.Status = StatusOutcomeUnknown
	observed.AppliedEffects = nil
	observed.ErrorCodes = []string{"transport_outcome_unknown"}
	observedDigest, err := DigestActionResult(observed)
	if err != nil {
		t.Fatal(err)
	}
	reconciliation := ActionReconciliation{
		Schema: ActionReconciliationSchema, ReconciliationID: "reconciliation:unknown-first",
		ResultRef: observed.ResultID, ResultSHA256: observedDigest, IdempotencyKey: proposal.IdempotencyKey,
		AuthoritativeInspectionRefs: []string{"inspection:provider-idempotency"}, State: ReconciliationConsistent,
		RetryPermitted: true, RetryReasonCode: "no_effect_observed", ReconciledAt: now.Add(3 * time.Minute),
	}
	if err := runtime.Reconcile(proposal, observed, reconciliation); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Reconcile(proposal, observed, reconciliation); !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("reconciliation replay error = %v, want ErrReplayDetected", err)
	}

	retryPreview, err := previews.BuildPreview(proposal, approval, effects, RiskModerate, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	retryConfirmation := runtimeConfirmation(t, proposal, retryPreview, now)
	successExecutor := ActionExecutorFunc(func(_ context.Context, current ActionProposal, currentPreview *ActionPreview) (ActionResult, error) {
		calls++
		return successfulRuntimeResult(t, current, *currentPreview, now), nil
	})
	result, err := runtime.Execute(context.Background(), proposal, &retryPreview, &retryConfirmation, proposal.ActorRef, previews, successExecutor)
	if err != nil || result.Status != StatusSucceeded {
		t.Fatalf("retry Execute() = %#v, %v", result, err)
	}
	if calls != 2 {
		t.Fatalf("executor calls after reconciliation = %d, want 2", calls)
	}
}

func TestExecutionRuntimeRetainsPartialOutcomeUntilReconciled(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:partial", nil })
	effects := []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append_capture"}, {EffectRef: "effect:2", TargetRef: "atom:2", Kind: "append_capture"}}
	preview, err := previews.BuildPreview(proposal, approval, effects, RiskModerate, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := runtimeConfirmation(t, proposal, preview, now)
	calls := 0
	executor := ActionExecutorFunc(func(_ context.Context, current ActionProposal, currentPreview *ActionPreview) (ActionResult, error) {
		calls++
		result := successfulRuntimeResult(t, current, *currentPreview, now)
		result.Status = StatusPartiallyApplied
		result.AppliedEffects = result.AppliedEffects[:1]
		return result, nil
	})
	runtime := NewExecutionRuntime()
	partial, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, executor)
	if !errors.Is(err, ErrReconciliationRequired) || partial.Status != StatusPartiallyApplied {
		t.Fatalf("partial Execute() = %#v, %v", partial, err)
	}
	if _, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, executor); !errors.Is(err, ErrReconciliationRequired) {
		t.Fatalf("partial replay error = %v, want ErrReconciliationRequired", err)
	}
	if calls != 1 {
		t.Fatalf("partial executor calls = %d, want 1", calls)
	}

	partialDigest, err := DigestActionResult(partial)
	if err != nil {
		t.Fatal(err)
	}
	reconciliation := ActionReconciliation{
		Schema: ActionReconciliationSchema, ReconciliationID: "reconciliation:partial",
		ResultRef: partial.ResultID, ResultSHA256: partialDigest, IdempotencyKey: proposal.IdempotencyKey,
		AuthoritativeInspectionRefs: []string{"inspection:partial-effect"}, State: ReconciliationConflict,
		ReconciledAt: now.Add(3 * time.Minute),
	}
	if err := runtime.Reconcile(proposal, partial, reconciliation); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, executor); !errors.Is(err, ErrReconciliationRequired) {
		t.Fatalf("blocked post-reconciliation replay error = %v, want ErrReconciliationRequired", err)
	}
	if calls != 1 {
		t.Fatalf("blocked reconciliation re-executed action: %d calls", calls)
	}
}

func TestValidateReconciliationRejectsTimeInversion(t *testing.T) {
	_, _, _, _, result, reconciliation, _, _ := validContracts(t)
	reconciliation.ReconciledAt = result.ObservedAt.Add(-time.Nanosecond)
	if err := ValidateReconciliation(reconciliation, result); !errors.Is(err, ErrActionContractInvalid) {
		t.Fatalf("ValidateReconciliation() error = %v, want ErrActionContractInvalid", err)
	}
}
