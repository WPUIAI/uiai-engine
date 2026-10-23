package evidenceaction

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPreviewRuntimeBuildsBoundedApprovedPreview(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	effects := []ExpectedEffect{
		{EffectRef: "effect:z", TargetRef: "atom:2", Kind: "append"},
		{EffectRef: "effect:a", TargetRef: "atom:1", Kind: "inspect"},
	}
	runtime := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:runtime", nil })
	preview, err := runtime.BuildPreview(proposal, approval, effects, RiskHigh, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ConfirmationRequired || preview.ExpiresAt.After(now.Add(maxPreviewTTL)) {
		t.Fatalf("preview policy = %#v", preview)
	}
	if preview.ExpectedEffects[0].EffectRef != "effect:a" || effects[0].EffectRef != "effect:z" {
		t.Fatalf("effects not normalized immutably: preview=%#v input=%#v", preview.ExpectedEffects, effects)
	}
	if err := ValidateActionPreviewAgainst(preview, proposal, now); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewRuntimeRequiresConfirmationForUntrustedOrExternalActions(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	tests := map[string]func(*ActionProposal){
		"external": func(p *ActionProposal) { p.SideEffect = EffectExternalMutation },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := proposal
			mutate(&candidate)
			runtime := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:" + name, nil })
			preview, err := runtime.BuildPreview(candidate, approval, []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "apply"}}, RiskLow, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if !preview.ConfirmationRequired {
				t.Fatal("unsafe action omitted confirmation")
			}
		})
	}
}

func TestPreviewRuntimeConsumesConfirmationNonceOnce(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	runtime := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:single-use", nil })
	preview, err := runtime.BuildPreview(proposal, approval, []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "apply"}}, RiskHigh, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := DigestActionPreview(preview)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := ActionConfirmation{PreviewRef: preview.PreviewID, PreviewSHA256: digest, Nonce: preview.AntiReplayNonce, ActorRef: proposal.ActorRef, ConfirmedAt: now}
	const workers = 16
	results := make(chan error, workers)
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- runtime.ConsumeConfirmation(confirmation, preview, proposal.ActorRef)
		}()
	}
	wait.Wait()
	close(results)
	succeeded, replayed := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrReplayDetected):
			replayed++
		default:
			t.Fatalf("unexpected confirmation error: %v", err)
		}
	}
	if succeeded != 1 || replayed != workers-1 {
		t.Fatalf("confirmation outcomes: succeeded=%d replayed=%d", succeeded, replayed)
	}
}

func TestPreviewRuntimeRejectsDuplicateAndUnissuedNonces(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	runtime := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:duplicate", nil })
	preview, err := runtime.BuildPreview(proposal, approval, []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "apply"}}, RiskHigh, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.BuildPreview(proposal, approval, []ExpectedEffect{{EffectRef: "effect:2", TargetRef: "atom:2", Kind: "apply"}}, RiskHigh, time.Minute); !errors.Is(err, ErrNonceUnavailable) {
		t.Fatalf("duplicate nonce BuildPreview() error = %v", err)
	}
	forged := preview
	forged.AntiReplayNonce = "nonce:forged"
	forged.PreviewID = previewID(forged.ProposalSHA256, forged.AntiReplayNonce)
	digest, err := DigestActionPreview(forged)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := ActionConfirmation{PreviewRef: forged.PreviewID, PreviewSHA256: digest, Nonce: forged.AntiReplayNonce, ActorRef: proposal.ActorRef, ConfirmedAt: now}
	if err := runtime.ConsumeConfirmation(confirmation, forged, proposal.ActorRef); !errors.Is(err, ErrPreviewMismatch) {
		t.Fatalf("unissued confirmation error = %v", err)
	}
}

func TestPreviewRuntimeFailsClosedOnApprovalAndEntropy(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	approval.Approved = false
	runtime := NewPreviewRuntime(func() time.Time { return now }, nil)
	if _, err := runtime.BuildPreview(proposal, approval, nil, RiskLow, time.Minute); !errors.Is(err, ErrCapabilityStale) {
		t.Fatalf("unapproved BuildPreview() error = %v", err)
	}
	approval.Approved = true
	imported := proposal
	imported.SourceTrust = SourceImportedUntrusted
	imported.HumanReviewMandated = true
	imported.AutonomousEligibility = AutonomousIneligible
	if _, err := runtime.BuildPreview(imported, approval, nil, RiskLow, time.Minute); !errors.Is(err, ErrImportedActionUntrusted) {
		t.Fatalf("imported BuildPreview() error = %v", err)
	}

	runtime = NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "", errors.New("entropy offline") })
	if _, err := runtime.BuildPreview(proposal, approval, nil, RiskLow, time.Minute); !errors.Is(err, ErrNonceUnavailable) {
		t.Fatalf("entropy BuildPreview() error = %v", err)
	}
}
