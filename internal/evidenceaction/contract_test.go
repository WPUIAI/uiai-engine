package evidenceaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestGoldenContractsRoundTripAndDigestStability(t *testing.T) {
	proposal, _, _, _, _, _, thread, _ := validContracts(t)
	assertGolden(t, "action-proposal.golden.json", proposal)
	assertGolden(t, "review-thread.golden.json", thread)
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	threadDigest, err := DigestReviewThread(thread)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		currentProposal, _ := DigestActionProposal(proposal)
		currentThread, _ := DigestReviewThread(thread)
		if currentProposal != proposalDigest || currentThread != threadDigest {
			t.Fatal("contract digest drift")
		}
	}
}

func TestExactItemAtomArtifactAndLineageScope(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	if err := ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*ActionProposal){
		func(p *ActionProposal) { p.Scope.ProjectRef = "project:other" },
		func(p *ActionProposal) { p.Scope.WorkItemRef = "item:other" },
		func(p *ActionProposal) { p.ArtifactRef = "artifact:other" },
		func(p *ActionProposal) { p.TargetAcceptanceAtomRefs = []string{"atom:other"} },
	}
	for index, mutate := range mutations {
		candidate := proposal
		mutate(&candidate)
		if !errors.Is(ValidateActionProposalAgainst(candidate, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now), ErrActionScopeMismatch) {
			t.Fatalf("scope mutation %d accepted", index)
		}
	}
}

func TestOperationApprovalAndAutonomyFailClosed(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	approval.Approved = false
	if !errors.Is(ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now), ErrCapabilityStale) {
		t.Fatal("unapproved operation accepted")
	}
	proposal, approval, _, _, _, _, _, now = validContracts(t)
	proposal.SourceTrust = SourceImportedUntrusted
	if !errors.Is(ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now), ErrImportedActionUntrusted) {
		t.Fatal("imported proposal granted authority")
	}
	proposal, approval, _, _, _, _, _, now = validContracts(t)
	proposal.HumanReviewMandated = true
	proposal.AutonomousEligibility = AutonomousIneligible
	if !errors.Is(ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now), ErrAutonomousIneligible) {
		t.Fatal("human-mandated proposal treated as autonomous")
	}
}

func TestChangedRequestTargetCapabilityAndStateInvalidatePreview(t *testing.T) {
	proposal, _, preview, _, _, _, _, now := validContracts(t)
	mutations := []func(*ActionPreview){
		func(p *ActionPreview) { p.NormalizedRequestSHA256 = digestA },
		func(p *ActionPreview) { p.TargetRefs = []string{"atom:other"} },
		func(p *ActionPreview) { p.CapabilitySnapshotSHA256 = digestA },
		func(p *ActionPreview) { p.ExpectedStateVersion++ },
	}
	for index, mutate := range mutations {
		candidate := preview
		mutate(&candidate)
		if !errors.Is(ValidateActionPreviewAgainst(candidate, proposal, now), ErrPreviewMismatch) {
			t.Fatalf("preview mutation %d accepted", index)
		}
	}
}

func TestExpiredReplayedAndChangedConfirmationFails(t *testing.T) {
	proposal, _, preview, confirmation, _, _, _, now := validContracts(t)
	if err := ValidateActionPreviewAgainst(preview, proposal, now); err != nil {
		t.Fatal(err)
	}
	checkAt := now.Add(2 * time.Minute)
	if err := ValidateConfirmationAgainst(confirmation, preview, proposal.ActorRef, nil, checkAt); err != nil {
		t.Fatal(err)
	}
	used := map[string]struct{}{confirmation.Nonce: {}}
	if !errors.Is(ValidateConfirmationAgainst(confirmation, preview, proposal.ActorRef, used, checkAt), ErrReplayDetected) {
		t.Fatal("replayed nonce accepted")
	}
	changed := confirmation
	changed.PreviewSHA256 = digestA
	if !errors.Is(ValidateConfirmationAgainst(changed, preview, proposal.ActorRef, nil, checkAt), ErrPreviewMismatch) {
		t.Fatal("changed confirmation accepted")
	}
	if !errors.Is(ValidateConfirmationAgainst(confirmation, preview, proposal.ActorRef, nil, preview.ExpiresAt), ErrActionExpired) {
		t.Fatal("expired confirmation accepted")
	}
}

func TestMutationRequiresPreviewAndConfirmation(t *testing.T) {
	proposal, _, preview, _, result, _, _, now := validContracts(t)
	if !errors.Is(ValidateActionResultAgainst(result, proposal, nil), ErrPreviewRequired) {
		t.Fatal("mutation result accepted without preview")
	}
	preview.ConfirmationRequired = false
	if !errors.Is(ValidateActionPreviewAgainst(preview, proposal, now), ErrConfirmationRequired) {
		t.Fatal("mutation preview accepted without confirmation")
	}
	proposal, _, preview, _, _, _, _, now = validContracts(t)
	preview.ExpectedEffects = nil
	if !errors.Is(ValidateActionPreviewAgainst(preview, proposal, now), ErrPreviewMismatch) {
		t.Fatal("provider receipt-only mutation accepted")
	}
	proposal, _, _, _, _, _, _, _ = validContracts(t)
	proposal.Action = ActionInspect
	if !errors.Is(ValidateActionProposal(proposal), ErrActionContractInvalid) {
		t.Fatal("inspect action advertised mutation authority")
	}
}

func TestUnknownAndPartialOutcomesReconcileBeforeRetry(t *testing.T) {
	proposal, _, preview, _, result, reconciliation, _, _ := validContracts(t)
	result.Status = StatusOutcomeUnknown
	result.AppliedEffects = nil
	result.ObservedStateVersion = 0
	result.ErrorCodes = []string{"response_lost"}
	resultDigest, _ := DigestActionResult(result)
	reconciliation.ResultSHA256 = resultDigest
	if !errors.Is(ValidateActionResultAgainst(result, proposal, &preview), ErrOutcomeUnknown) {
		t.Fatal("ambiguous result accepted as known")
	}
	if err := ValidateReconciliation(reconciliation, result); err != nil {
		t.Fatalf("valid reconciliation: %v", err)
	}
	bad := reconciliation
	bad.AuthoritativeInspectionRefs = nil
	if !errors.Is(ValidateReconciliation(bad, result), ErrActionContractInvalid) {
		t.Fatal("retry accepted without authoritative inspection")
	}
	bad = reconciliation
	bad.State = ReconciliationConflict
	if !errors.Is(ValidateReconciliation(bad, result), ErrReconciliationRequired) {
		t.Fatal("retry accepted with conflict")
	}
}

func TestPartialEffectsCannotSerializeAsSuccess(t *testing.T) {
	proposal, _, preview, _, result, _, _, _ := validContracts(t)
	result.AppliedEffects = result.AppliedEffects[:1]
	if !errors.Is(ValidateActionResultAgainst(result, proposal, &preview), ErrActionContractInvalid) {
		t.Fatal("partial effects serialized as success")
	}
	result.Status = StatusPartiallyApplied
	if err := ValidateActionResultAgainst(result, proposal, &preview); err != nil {
		t.Fatalf("typed partial result rejected: %v", err)
	}
}

func TestImportedReviewCannotGrantAuthority(t *testing.T) {
	_, _, _, _, _, _, thread, _ := validContracts(t)
	thread.Entries[0].Imported = true
	thread.Entries[0].SourceTrust = SourceImportedUntrusted
	if err := ValidateReviewThread(thread); err != nil {
		t.Fatalf("untrusted import not representable: %v", err)
	}
	if !errors.Is(ValidateReviewAuthority(thread), ErrImportedActionUntrusted) {
		t.Fatal("imported review granted authority")
	}
}

func TestReviewApprovalCannotSerializeCompletionOrSettlement(t *testing.T) {
	_, _, _, _, _, _, thread, _ := validContracts(t)
	if err := ValidateReviewAuthority(thread); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(thread)
	for _, forbidden := range []string{"completion_receipt", "provider_close", "settlement", `"completed"`} {
		if bytes.Contains(body, []byte(forbidden)) {
			t.Fatalf("review leaked completion authority: %s", forbidden)
		}
	}
}

func TestUnknownVocabulariesBoundsAndDanglingRefsFail(t *testing.T) {
	proposal, _, _, _, _, _, thread, _ := validContracts(t)
	proposal.Action = "bulk"
	if !errors.Is(ValidateActionProposal(proposal), ErrActionContractInvalid) {
		t.Fatal("unknown action accepted")
	}
	_, _, _, _, result, _, _, _ := validContracts(t)
	result.Status = "complete"
	if !errors.Is(ValidateActionResult(result), ErrActionContractInvalid) {
		t.Fatal("completion-like status accepted")
	}
	_, _, _, _, _, _, thread, _ = validContracts(t)
	thread.Entries[0].AtomRefs = []string{"atom:other"}
	if !errors.Is(ValidateReviewThread(thread), ErrActionContractInvalid) {
		t.Fatal("dangling atom accepted")
	}
	proposal, _, _, _, _, _, _, _ = validContracts(t)
	proposal.TargetAcceptanceAtomRefs = make([]string, MaxRefs+1)
	if !errors.Is(ValidateActionProposal(proposal), ErrActionContractInvalid) {
		t.Fatal("oversized targets accepted")
	}
}

func TestValidatorsDoNotMutateInputs(t *testing.T) {
	proposal, _, _, _, _, _, thread, _ := validContracts(t)
	proposalBefore := deepCopy(proposal)
	threadBefore := deepCopy(thread)
	for i := 0; i < 30; i++ {
		_ = ValidateActionProposal(proposal)
		_ = ValidateReviewThread(thread)
	}
	if !reflect.DeepEqual(proposal, proposalBefore) || !reflect.DeepEqual(thread, threadBefore) {
		t.Fatal("validation mutated caller input")
	}
}

func validContracts(t *testing.T) (ActionProposal, OperationApproval, ActionPreview, ActionConfirmation, ActionResult, ActionReconciliation, ReviewThread, time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	scope := ScopeBinding{ProjectRef: "project:1", WorkstreamRef: "workstream:1", WorksetRef: "workset:1", CallGraphRef: "callgraph:1", WorkpointRef: "workpoint:1", WorkItemRef: "github:WPUIAI/uiai-engine#117"}
	proposal := ActionProposal{Schema: ActionProposalSchema, ProposalID: "proposal:1", Scope: scope, ArtifactRef: "artifact:1", ArtifactSHA256: digestA, TargetAcceptanceAtomRefs: []string{"atom:1", "atom:2"}, CapabilitySnapshotRef: "capability:1", CapabilitySnapshotSHA256: digestB, OperationRef: "operation:evidence.reproof", OperationVersion: "v1", Action: ActionReproof, ActorRef: "actor:1", DelegationRef: "delegation:1", IdempotencyKey: "idempotency:1", ExpectedStateRef: "state:1", ExpectedStateVersion: 7, ExpiresAt: now.Add(time.Hour), SideEffect: EffectEvidenceAppend, SourceTrust: SourceVerified, AutonomousEligibility: AutonomousEligible}
	approval := OperationApproval{RegistryRef: "registry:1", RegistrySHA256: digestA, OperationRef: proposal.OperationRef, OperationVersion: proposal.OperationVersion, CapabilitySnapshotRef: proposal.CapabilitySnapshotRef, CapabilitySnapshotSHA256: proposal.CapabilitySnapshotSHA256, Approved: true, ExpiresAt: now.Add(time.Hour)}
	proposalDigest, _ := DigestActionProposal(proposal)
	preview := ActionPreview{Schema: ActionPreviewSchema, PreviewID: "preview:1", ProposalRef: proposal.ProposalID, ProposalSHA256: proposalDigest, NormalizedRequestSHA256: proposalDigest, CapabilitySnapshotRef: proposal.CapabilitySnapshotRef, CapabilitySnapshotSHA256: proposal.CapabilitySnapshotSHA256, TargetRefs: append([]string(nil), proposal.TargetAcceptanceAtomRefs...), ExpectedEffects: []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append_capture"}, {EffectRef: "effect:2", TargetRef: "atom:2", Kind: "append_capture"}}, ExpectedStateVersion: proposal.ExpectedStateVersion, Risk: RiskModerate, ConfirmationRequired: true, ExpiresAt: now.Add(30 * time.Minute), AntiReplayNonce: "nonce:1"}
	previewDigest, _ := DigestActionPreview(preview)
	confirmation := ActionConfirmation{PreviewRef: preview.PreviewID, PreviewSHA256: previewDigest, Nonce: preview.AntiReplayNonce, ActorRef: proposal.ActorRef, ConfirmedAt: now.Add(time.Minute)}
	result := ActionResult{Schema: ActionResultSchema, ResultID: "result:1", ProposalRef: proposal.ProposalID, ProposalSHA256: proposalDigest, PreviewRef: preview.PreviewID, PreviewSHA256: previewDigest, IdempotencyKey: proposal.IdempotencyKey, Status: StatusSucceeded, AppliedEffects: []AppliedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", ObservedVersion: 8, EvidenceRef: "evidence:1"}, {EffectRef: "effect:2", TargetRef: "atom:2", ObservedVersion: 9, EvidenceRef: "evidence:2"}}, ProviderReceiptRefs: []string{"receipt:1"}, AuthoritativeStateRef: "state:2", ObservedStateVersion: 9, ObservedAt: now.Add(2 * time.Minute)}
	resultDigest, _ := DigestActionResult(result)
	reconciliation := ActionReconciliation{Schema: ActionReconciliationSchema, ReconciliationID: "reconciliation:1", ResultRef: result.ResultID, ResultSHA256: resultDigest, IdempotencyKey: result.IdempotencyKey, AuthoritativeInspectionRefs: []string{"inspection:1"}, State: ReconciliationConsistent, RetryPermitted: true, RetryReasonCode: "no_effect_observed", ReconciledAt: now.Add(3 * time.Minute)}
	thread := ReviewThread{Schema: ReviewThreadSchema, ThreadID: "thread:1", Scope: scope, ArtifactRef: proposal.ArtifactRef, ArtifactSHA256: proposal.ArtifactSHA256, AtomRefs: append([]string(nil), proposal.TargetAcceptanceAtomRefs...), Revision: 2, SourceTrust: SourceVerified, AutonomousEligibility: AutonomousEligible, Entries: []ReviewEntry{{EntryID: "entry:1", Revision: 1, Kind: EntryComment, Decision: DecisionNone, Message: "Evidence inspected against the frozen atom.", ItemRef: scope.WorkItemRef, AtomRefs: []string{"atom:1"}, ArtifactRef: proposal.ArtifactRef, CitationRefs: []string{"citation:1"}, ActorRef: "reviewer:1", DelegationRef: "delegation:reviewer", OccurredAt: now, ProvenanceRef: "provenance:1", SourceTrust: SourceVerified}, {EntryID: "entry:2", Revision: 2, Kind: EntryDecision, Decision: DecisionApproved, Message: "The cited evidence supports this item-scoped review decision.", ItemRef: scope.WorkItemRef, AtomRefs: []string{"atom:1", "atom:2"}, ArtifactRef: proposal.ArtifactRef, CitationRefs: []string{"citation:1", "citation:2"}, ActorRef: "reviewer:1", DelegationRef: "delegation:reviewer", OccurredAt: now.Add(time.Minute), ProvenanceRef: "provenance:2", SourceTrust: SourceVerified}}}
	return proposal, approval, preview, confirmation, result, reconciliation, thread, now
}

func assertGolden(t *testing.T, name string, value any) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, want) {
		t.Fatalf("%s drifted; run UPDATE_GOLDEN=1", name)
	}
}
func deepCopy[T any](value T) T {
	body, _ := json.Marshal(value)
	var out T
	_ = json.Unmarshal(body, &out)
	return out
}
func TestStableErrorsAreValueFree(t *testing.T) {
	for _, err := range []error{ErrActionContractInvalid, ErrActionScopeMismatch, ErrCapabilityStale, ErrPreviewRequired, ErrPreviewMismatch, ErrConfirmationRequired, ErrReplayDetected, ErrActionExpired, ErrStateVersionMismatch, ErrOutcomeUnknown, ErrReconciliationRequired, ErrImportedActionUntrusted, ErrAutonomousIneligible} {
		if strings.ContainsAny(err.Error(), "\n\r") || len(err.Error()) > 100 {
			t.Fatalf("unsafe error: %q", err)
		}
	}
}
