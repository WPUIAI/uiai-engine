package evidenceaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrActionContractInvalid   = errors.New("evidence action contract invalid")
	ErrActionScopeMismatch     = errors.New("evidence action scope mismatch")
	ErrCapabilityStale         = errors.New("evidence action capability stale")
	ErrPreviewRequired         = errors.New("evidence action preview required")
	ErrPreviewMismatch         = errors.New("evidence action preview mismatch")
	ErrConfirmationRequired    = errors.New("evidence action confirmation required")
	ErrReplayDetected          = errors.New("evidence action replay detected")
	ErrActionExpired           = errors.New("evidence action expired")
	ErrStateVersionMismatch    = errors.New("evidence action state version mismatch")
	ErrOutcomeUnknown          = errors.New("evidence action outcome unknown")
	ErrReconciliationRequired  = errors.New("evidence action reconciliation required")
	ErrImportedActionUntrusted = errors.New("imported evidence action untrusted")
	ErrAutonomousIneligible    = errors.New("evidence action autonomous ineligible")
)

func DigestActionProposal(value ActionProposal) (string, error)       { return digest(value) }
func DigestActionPreview(value ActionPreview) (string, error)         { return digest(value) }
func DigestActionResult(value ActionResult) (string, error)           { return digest(value) }
func DigestReviewThread(value ReviewThread) (string, error)           { return digest(value) }
func DigestReconciliation(value ActionReconciliation) (string, error) { return digest(value) }

func ValidateActionProposal(proposal ActionProposal) error {
	if proposal.Schema != ActionProposalSchema || blank(proposal.ProposalID) || !validScope(proposal.Scope) ||
		blank(proposal.ArtifactRef) || !validSHA256(proposal.ArtifactSHA256) ||
		len(proposal.TargetAcceptanceAtomRefs) == 0 || len(proposal.TargetAcceptanceAtomRefs) > MaxRefs ||
		hasBlankOrDuplicate(proposal.TargetAcceptanceAtomRefs) || blank(proposal.CapabilitySnapshotRef) ||
		!validSHA256(proposal.CapabilitySnapshotSHA256) || blank(proposal.OperationRef) ||
		blank(proposal.OperationVersion) || !validAction(proposal.Action) || blank(proposal.ActorRef) ||
		blank(proposal.DelegationRef) || blank(proposal.IdempotencyKey) || blank(proposal.ExpectedStateRef) ||
		proposal.ExpectedStateVersion == 0 || proposal.ExpiresAt.IsZero() || !validSideEffect(proposal.SideEffect) ||
		!validSourceTrust(proposal.SourceTrust) || !validEligibility(proposal.AutonomousEligibility) {
		return ErrActionContractInvalid
	}
	if proposal.HumanReviewMandated != (proposal.AutonomousEligibility == AutonomousIneligible) {
		return ErrActionContractInvalid
	}
	if (proposal.Action == ActionInspect || proposal.Action == ActionExport) && proposal.SideEffect != EffectReadOnly {
		return ErrActionContractInvalid
	}
	return sizeOK(proposal)
}

func ValidateActionProposalAgainst(proposal ActionProposal, scope ScopeBinding, artifactRef, artifactSHA256 string, expectedAtomRefs []string, approval OperationApproval, now time.Time) error {
	if err := ValidateActionProposal(proposal); err != nil {
		return err
	}
	if proposal.Scope != scope || proposal.ArtifactRef != artifactRef || proposal.ArtifactSHA256 != artifactSHA256 ||
		!reflect.DeepEqual(proposal.TargetAcceptanceAtomRefs, expectedAtomRefs) {
		return ErrActionScopeMismatch
	}
	if !now.Before(proposal.ExpiresAt) {
		return ErrActionExpired
	}
	if proposal.SourceTrust == SourceImportedUntrusted {
		return ErrImportedActionUntrusted
	}
	if proposal.AutonomousEligibility == AutonomousIneligible {
		return ErrAutonomousIneligible
	}
	if !validApproval(approval) || !approval.Approved || !now.Before(approval.ExpiresAt) ||
		approval.OperationRef != proposal.OperationRef || approval.OperationVersion != proposal.OperationVersion ||
		approval.CapabilitySnapshotRef != proposal.CapabilitySnapshotRef ||
		approval.CapabilitySnapshotSHA256 != proposal.CapabilitySnapshotSHA256 {
		return ErrCapabilityStale
	}
	return nil
}

func ValidateActionPreview(preview ActionPreview) error {
	if preview.Schema != ActionPreviewSchema || blank(preview.PreviewID) || blank(preview.ProposalRef) ||
		!validSHA256(preview.ProposalSHA256) || !validSHA256(preview.NormalizedRequestSHA256) ||
		blank(preview.CapabilitySnapshotRef) || !validSHA256(preview.CapabilitySnapshotSHA256) ||
		len(preview.TargetRefs) == 0 || len(preview.TargetRefs) > MaxRefs || hasBlankOrDuplicate(preview.TargetRefs) ||
		len(preview.ExpectedEffects) > MaxEffects || preview.ExpectedStateVersion == 0 || !validRisk(preview.Risk) ||
		preview.ExpiresAt.IsZero() || blank(preview.AntiReplayNonce) {
		return ErrActionContractInvalid
	}
	seen := make(map[string]struct{}, len(preview.ExpectedEffects))
	for _, effect := range preview.ExpectedEffects {
		if blank(effect.EffectRef) || blank(effect.TargetRef) || blank(effect.Kind) || !addUnique(seen, effect.EffectRef) {
			return ErrActionContractInvalid
		}
	}
	return sizeOK(preview)
}

func ValidateActionPreviewAgainst(preview ActionPreview, proposal ActionProposal, now time.Time) error {
	if err := ValidateActionPreview(preview); err != nil {
		return err
	}
	if err := ValidateActionProposal(proposal); err != nil {
		return err
	}
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		return err
	}
	if preview.ProposalRef != proposal.ProposalID || preview.ProposalSHA256 != proposalDigest ||
		preview.NormalizedRequestSHA256 != proposalDigest || preview.CapabilitySnapshotRef != proposal.CapabilitySnapshotRef ||
		preview.CapabilitySnapshotSHA256 != proposal.CapabilitySnapshotSHA256 ||
		!reflect.DeepEqual(preview.TargetRefs, proposal.TargetAcceptanceAtomRefs) ||
		preview.ExpectedStateVersion != proposal.ExpectedStateVersion {
		return ErrPreviewMismatch
	}
	if !now.Before(preview.ExpiresAt) || preview.ExpiresAt.After(proposal.ExpiresAt) {
		return ErrActionExpired
	}
	if proposal.SideEffect != EffectReadOnly && len(preview.ExpectedEffects) == 0 {
		return ErrPreviewMismatch
	}
	if proposal.SideEffect != EffectReadOnly && !preview.ConfirmationRequired {
		return ErrConfirmationRequired
	}
	return nil
}

func ValidateConfirmationAgainst(confirmation ActionConfirmation, preview ActionPreview, actorRef string, usedNonces map[string]struct{}, now time.Time) error {
	if !preview.ConfirmationRequired {
		return nil
	}
	if blank(confirmation.PreviewRef) || !validSHA256(confirmation.PreviewSHA256) || blank(confirmation.Nonce) ||
		blank(confirmation.ActorRef) || confirmation.ConfirmedAt.IsZero() {
		return ErrConfirmationRequired
	}
	previewDigest, err := DigestActionPreview(preview)
	if err != nil {
		return err
	}
	if confirmation.PreviewRef != preview.PreviewID || confirmation.PreviewSHA256 != previewDigest ||
		confirmation.Nonce != preview.AntiReplayNonce || confirmation.ActorRef != actorRef {
		return ErrPreviewMismatch
	}
	if !now.Before(preview.ExpiresAt) || confirmation.ConfirmedAt.After(preview.ExpiresAt) || confirmation.ConfirmedAt.After(now) {
		return ErrActionExpired
	}
	if _, used := usedNonces[confirmation.Nonce]; used {
		return ErrReplayDetected
	}
	return nil
}

func ValidateActionResult(result ActionResult) error {
	if result.Schema != ActionResultSchema || blank(result.ResultID) || blank(result.ProposalRef) ||
		!validSHA256(result.ProposalSHA256) || blank(result.IdempotencyKey) || !validStatus(result.Status) ||
		len(result.AppliedEffects) > MaxEffects || len(result.Compensations) > MaxEffects ||
		len(result.ProviderReceiptRefs) > MaxRefs || hasBlankOrDuplicate(result.ProviderReceiptRefs) ||
		len(result.ErrorCodes) > MaxRefs || hasBlankOrDuplicate(result.ErrorCodes) || result.ObservedAt.IsZero() {
		return ErrActionContractInvalid
	}
	seen := make(map[string]struct{}, len(result.AppliedEffects))
	for _, effect := range result.AppliedEffects {
		if blank(effect.EffectRef) || blank(effect.TargetRef) || effect.ObservedVersion == 0 || blank(effect.EvidenceRef) || !addUnique(seen, effect.EffectRef) {
			return ErrActionContractInvalid
		}
	}
	for _, compensation := range result.Compensations {
		if blank(compensation.EffectRef) || blank(compensation.CompensationRef) ||
			(compensation.Status != "pending" && compensation.Status != "succeeded" && compensation.Status != "failed" && compensation.Status != "outcome_unknown") ||
			(compensation.Status == "succeeded" && blank(compensation.EvidenceRef)) {
			return ErrActionContractInvalid
		}
	}
	if result.Status == StatusSucceeded && len(result.ErrorCodes) > 0 {
		return ErrActionContractInvalid
	}
	if result.Status == StatusPartiallyApplied && len(result.AppliedEffects) == 0 {
		return ErrActionContractInvalid
	}
	if result.Status == StatusOutcomeUnknown && len(result.ErrorCodes) == 0 {
		return ErrActionContractInvalid
	}
	return sizeOK(result)
}

func ValidateActionResultAgainst(result ActionResult, proposal ActionProposal, preview *ActionPreview) error {
	if err := ValidateActionResult(result); err != nil {
		return err
	}
	if err := ValidateActionProposal(proposal); err != nil {
		return err
	}
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		return err
	}
	if result.ProposalRef != proposal.ProposalID || result.ProposalSHA256 != proposalDigest || result.IdempotencyKey != proposal.IdempotencyKey {
		return ErrActionScopeMismatch
	}
	if result.Status != StatusOutcomeUnknown && result.ObservedStateVersion < proposal.ExpectedStateVersion {
		return ErrStateVersionMismatch
	}
	if proposal.SideEffect != EffectReadOnly && preview == nil {
		return ErrPreviewRequired
	}
	if preview != nil {
		if err := ValidateActionPreview(*preview); err != nil {
			return err
		}
		previewDigest, err := DigestActionPreview(*preview)
		if err != nil {
			return err
		}
		if result.PreviewRef != preview.PreviewID || result.PreviewSHA256 != previewDigest {
			return ErrPreviewMismatch
		}
		if result.Status == StatusSucceeded && !effectsMatch(preview.ExpectedEffects, result.AppliedEffects) {
			return ErrActionContractInvalid
		}
	}
	if result.Status == StatusOutcomeUnknown {
		return ErrOutcomeUnknown
	}
	return nil
}

func ValidateReconciliation(reconciliation ActionReconciliation, result ActionResult) error {
	if err := ValidateActionResult(result); err != nil {
		return err
	}
	if reconciliation.Schema != ActionReconciliationSchema || blank(reconciliation.ReconciliationID) ||
		blank(reconciliation.ResultRef) || !validSHA256(reconciliation.ResultSHA256) ||
		blank(reconciliation.IdempotencyKey) || len(reconciliation.AuthoritativeInspectionRefs) == 0 ||
		len(reconciliation.AuthoritativeInspectionRefs) > MaxRefs || hasBlankOrDuplicate(reconciliation.AuthoritativeInspectionRefs) ||
		!validReconciliationState(reconciliation.State) || reconciliation.ReconciledAt.IsZero() ||
		reconciliation.ReconciledAt.Before(result.ObservedAt) {
		return ErrActionContractInvalid
	}
	resultDigest, err := DigestActionResult(result)
	if err != nil {
		return err
	}
	if reconciliation.ResultRef != result.ResultID || reconciliation.ResultSHA256 != resultDigest ||
		reconciliation.IdempotencyKey != result.IdempotencyKey {
		return ErrActionScopeMismatch
	}
	if result.Status != StatusOutcomeUnknown && result.Status != StatusPartiallyApplied {
		return ErrReconciliationRequired
	}
	if reconciliation.RetryPermitted && (reconciliation.State != ReconciliationConsistent || blank(reconciliation.RetryReasonCode)) {
		return ErrReconciliationRequired
	}
	return sizeOK(reconciliation)
}

func ValidateReviewThread(thread ReviewThread) error {
	if thread.Schema != ReviewThreadSchema || blank(thread.ThreadID) || !validScope(thread.Scope) ||
		blank(thread.ArtifactRef) || !validSHA256(thread.ArtifactSHA256) || len(thread.AtomRefs) == 0 ||
		len(thread.AtomRefs) > MaxRefs || hasBlankOrDuplicate(thread.AtomRefs) || thread.Revision == 0 ||
		len(thread.Entries) > MaxEntries || !validSourceTrust(thread.SourceTrust) || !validEligibility(thread.AutonomousEligibility) ||
		thread.HumanReviewMandated != (thread.AutonomousEligibility == AutonomousIneligible) {
		return ErrActionContractInvalid
	}
	seen := make(map[string]struct{}, len(thread.Entries))
	for index, entry := range thread.Entries {
		if blank(entry.EntryID) || entry.Revision != uint64(index+1) || !validEntryKind(entry.Kind) ||
			!validDecision(entry.Decision) || blank(entry.Message) || utf8.RuneCountInString(entry.Message) > MaxMessageRunes ||
			entry.ItemRef != thread.Scope.WorkItemRef || entry.ArtifactRef != thread.ArtifactRef ||
			len(entry.AtomRefs) == 0 || hasBlankOrDuplicate(entry.AtomRefs) || !subset(entry.AtomRefs, thread.AtomRefs) ||
			hasBlankOrDuplicate(entry.CitationRefs) || blank(entry.ActorRef) || blank(entry.DelegationRef) ||
			entry.OccurredAt.IsZero() || blank(entry.ProvenanceRef) || !validSourceTrust(entry.SourceTrust) || !addUnique(seen, entry.EntryID) {
			return ErrActionContractInvalid
		}
		if entry.Kind != EntryDecision && entry.Decision != DecisionNone {
			return ErrActionContractInvalid
		}
		if entry.Kind == EntryDecision && entry.Decision == DecisionNone {
			return ErrActionContractInvalid
		}
		if entry.Imported != (entry.SourceTrust == SourceImportedUntrusted) {
			return ErrActionContractInvalid
		}
		if !blank(entry.SupersedesRef) {
			if _, exists := seen[entry.SupersedesRef]; !exists {
				return ErrActionContractInvalid
			}
		}
	}
	if thread.Revision != uint64(len(thread.Entries)) {
		return ErrActionContractInvalid
	}
	return sizeOK(thread)
}

func ValidateReviewAuthority(thread ReviewThread) error {
	if err := ValidateReviewThread(thread); err != nil {
		return err
	}
	if thread.SourceTrust == SourceImportedUntrusted {
		return ErrImportedActionUntrusted
	}
	for _, entry := range thread.Entries {
		if entry.SourceTrust == SourceImportedUntrusted {
			return ErrImportedActionUntrusted
		}
	}
	if thread.AutonomousEligibility == AutonomousIneligible {
		return ErrAutonomousIneligible
	}
	return nil
}

func digest(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", ErrActionContractInvalid
	}
	if len(body) > MaxContractBytes {
		return "", ErrActionContractInvalid
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
func sizeOK(value any) error {
	body, err := json.Marshal(value)
	if err != nil || len(body) > MaxContractBytes {
		return ErrActionContractInvalid
	}
	return nil
}
func validScope(scope ScopeBinding) bool {
	return !blank(scope.ProjectRef) && !blank(scope.WorkstreamRef) && !blank(scope.WorksetRef) && !blank(scope.CallGraphRef) && !blank(scope.WorkpointRef) && !blank(scope.WorkItemRef)
}
func validApproval(value OperationApproval) bool {
	return !blank(value.RegistryRef) && validSHA256(value.RegistrySHA256) && !blank(value.OperationRef) && !blank(value.OperationVersion) && !blank(value.CapabilitySnapshotRef) && validSHA256(value.CapabilitySnapshotSHA256) && !value.ExpiresAt.IsZero()
}
func validAction(value ActionType) bool {
	return value == ActionInspect || value == ActionLink || value == ActionCapture || value == ActionReproof || value == ActionFollowUp || value == ActionAdjudication || value == ActionShare || value == ActionExport
}
func validSideEffect(value SideEffectClass) bool {
	return value == EffectReadOnly || value == EffectEvidenceAppend || value == EffectFocusaHandoff || value == EffectExternalMutation
}
func validSourceTrust(value SourceTrust) bool {
	return value == SourceVerified || value == SourceImportedUntrusted
}
func validEligibility(value AutonomousEligibility) bool {
	return value == AutonomousEligible || value == AutonomousIneligible
}
func validRisk(value RiskClass) bool {
	return value == RiskLow || value == RiskModerate || value == RiskHigh
}
func validStatus(value ActionStatus) bool {
	return value == StatusSucceeded || value == StatusRejected || value == StatusBlocked || value == StatusPartiallyApplied || value == StatusOutcomeUnknown
}
func validReconciliationState(value ReconciliationState) bool {
	return value == ReconciliationPending || value == ReconciliationConsistent || value == ReconciliationConflict || value == ReconciliationBlocked
}
func validEntryKind(value ReviewEntryKind) bool {
	return value == EntryComment || value == EntryAnnotation || value == EntrySuggestion || value == EntryDecision || value == EntrySupersession || value == EntryNotificationIntent
}
func validDecision(value ReviewDecision) bool {
	return value == DecisionNone || value == DecisionApproved || value == DecisionChangesRequested || value == DecisionRejected || value == DecisionBlocked
}
func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}
func blank(value string) bool { return strings.TrimSpace(value) == "" }
func addUnique(values map[string]struct{}, value string) bool {
	if _, exists := values[value]; exists {
		return false
	}
	values[value] = struct{}{}
	return true
}
func hasBlankOrDuplicate(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if blank(value) || !addUnique(seen, value) {
			return true
		}
	}
	return false
}
func subset(values, allowed []string) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}
func effectsMatch(expected []ExpectedEffect, actual []AppliedEffect) bool {
	if len(expected) != len(actual) {
		return false
	}
	byRef := make(map[string]ExpectedEffect, len(expected))
	for _, effect := range expected {
		byRef[effect.EffectRef] = effect
	}
	for _, effect := range actual {
		want, ok := byRef[effect.EffectRef]
		if !ok || want.TargetRef != effect.TargetRef {
			return false
		}
	}
	return true
}
