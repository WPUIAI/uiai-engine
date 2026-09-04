package evidenceaction

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutionRuntimeExecutesIdempotencyKeyOnce(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:execute-once", nil })
	effects := []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append"}}
	preview, err := previews.BuildPreview(proposal, approval, effects, RiskLow, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	want := successfulRuntimeResult(t, proposal, preview, now)
	confirmation := runtimeConfirmation(t, proposal, preview, now)
	var calls atomic.Int32
	executor := ActionExecutorFunc(func(context.Context, ActionProposal, *ActionPreview) (ActionResult, error) {
		calls.Add(1)
		return want, nil
	})
	runtime := NewExecutionRuntime()
	const workers = 16
	results := make(chan ActionResult, workers)
	errorsSeen := make(chan error, workers)
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, executor)
			results <- result
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	for result := range results {
		if result.ResultID != want.ResultID {
			t.Fatalf("result = %#v", result)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("executor calls = %d, want 1", calls.Load())
	}
	conflict := proposal
	conflict.ProposalID = "proposal:conflict"
	conflictPreviews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:conflict", nil })
	conflictPreview, err := conflictPreviews.BuildPreview(conflict, approval, effects, RiskLow, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	conflictConfirmation := runtimeConfirmation(t, conflict, conflictPreview, now)
	if _, err := runtime.Execute(context.Background(), conflict, &conflictPreview, &conflictConfirmation, conflict.ActorRef, conflictPreviews, executor); !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("conflicting idempotency error = %v", err)
	}
}

func TestExecutionRuntimeAllowsConfirmationRepairBeforeSideEffect(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:confirmation", nil })
	preview, err := previews.BuildPreview(proposal, approval, []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append"}}, RiskHigh, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	want := successfulRuntimeResult(t, proposal, preview, now)
	var calls atomic.Int32
	executor := ActionExecutorFunc(func(context.Context, ActionProposal, *ActionPreview) (ActionResult, error) {
		calls.Add(1)
		return want, nil
	})
	runtime := NewExecutionRuntime()
	if _, err := runtime.Execute(context.Background(), proposal, &preview, nil, proposal.ActorRef, previews, executor); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("missing confirmation error = %v", err)
	}
	digest, err := DigestActionPreview(preview)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := &ActionConfirmation{PreviewRef: preview.PreviewID, PreviewSHA256: digest, Nonce: preview.AntiReplayNonce, ActorRef: proposal.ActorRef, ConfirmedAt: now}
	if _, err := runtime.Execute(context.Background(), proposal, &preview, confirmation, proposal.ActorRef, previews, executor); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("executor calls = %d, want 1", calls.Load())
	}
}

func TestExecutionRuntimeCachesUnknownOutcomeWithoutRetry(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:unknown", nil })
	preview, err := previews.BuildPreview(proposal, approval, []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append"}}, RiskLow, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := runtimeConfirmation(t, proposal, preview, now)
	var calls atomic.Int32
	executor := ActionExecutorFunc(func(context.Context, ActionProposal, *ActionPreview) (ActionResult, error) {
		calls.Add(1)
		return ActionResult{}, errors.New("provider response lost")
	})
	runtime := NewExecutionRuntime()
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := runtime.Execute(context.Background(), proposal, &preview, &confirmation, proposal.ActorRef, previews, executor); !errors.Is(err, ErrOutcomeUnknown) {
			t.Fatalf("attempt %d error = %v", attempt, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("executor retried unknown outcome: %d calls", calls.Load())
	}
}

func runtimeConfirmation(t *testing.T, proposal ActionProposal, preview ActionPreview, now time.Time) ActionConfirmation {
	t.Helper()
	digest, err := DigestActionPreview(preview)
	if err != nil {
		t.Fatal(err)
	}
	return ActionConfirmation{PreviewRef: preview.PreviewID, PreviewSHA256: digest, Nonce: preview.AntiReplayNonce, ActorRef: proposal.ActorRef, ConfirmedAt: now}
}

func successfulRuntimeResult(t *testing.T, proposal ActionProposal, preview ActionPreview, now time.Time) ActionResult {
	t.Helper()
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	previewDigest, err := DigestActionPreview(preview)
	if err != nil {
		t.Fatal(err)
	}
	return ActionResult{
		Schema: ActionResultSchema, ResultID: "result:runtime", ProposalRef: proposal.ProposalID,
		ProposalSHA256: proposalDigest, PreviewRef: preview.PreviewID, PreviewSHA256: previewDigest,
		IdempotencyKey: proposal.IdempotencyKey, Status: StatusSucceeded,
		AppliedEffects:      []AppliedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", ObservedVersion: proposal.ExpectedStateVersion + 1, EvidenceRef: "evidence:effect"}},
		ProviderReceiptRefs: []string{"receipt:provider"}, AuthoritativeStateRef: "state:authoritative",
		ObservedStateVersion: proposal.ExpectedStateVersion + 1, ObservedAt: now,
	}
}
