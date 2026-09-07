package evidenceaction

import "errors"

// Reconcile records authoritative inspection of an unknown or partial action
// outcome. Only an exact, unused reconciliation that proves a consistent state
// may release the idempotency key for one explicit retry.
func (runtime *ExecutionRuntime) Reconcile(proposal ActionProposal, observed ActionResult, reconciliation ActionReconciliation) error {
	if runtime == nil {
		return ErrReconciliationRequired
	}
	if err := ValidateActionProposal(proposal); err != nil {
		return err
	}
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		return err
	}

	runtime.mu.Lock()
	record, found := runtime.records[proposal.IdempotencyKey]
	if _, used := runtime.reconciliations[reconciliation.ReconciliationID]; used {
		runtime.mu.Unlock()
		return ErrReplayDetected
	}
	if !found || record.proposalSHA256 != proposalDigest {
		runtime.mu.Unlock()
		return ErrReconciliationRequired
	}
	select {
	case <-record.done:
	default:
		runtime.mu.Unlock()
		return ErrReconciliationRequired
	}
	preview := cloneActionPreview(record.preview)
	recordErr := record.err
	storedStatus := record.result.Status
	runtime.mu.Unlock()

	if !errors.Is(recordErr, ErrOutcomeUnknown) && !errors.Is(recordErr, ErrReconciliationRequired) &&
		storedStatus != StatusOutcomeUnknown && storedStatus != StatusPartiallyApplied {
		return ErrReconciliationRequired
	}
	if resultErr := ValidateActionResultAgainst(observed, proposal, preview); resultErr != nil && !errors.Is(resultErr, ErrOutcomeUnknown) {
		return resultErr
	}
	if observed.Status != StatusOutcomeUnknown && observed.Status != StatusPartiallyApplied {
		return ErrReconciliationRequired
	}
	if err := ValidateReconciliation(reconciliation, observed); err != nil {
		return err
	}
	reconciliationDigest, err := DigestReconciliation(reconciliation)
	if err != nil {
		return err
	}

	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	current, found := runtime.records[proposal.IdempotencyKey]
	if !found || current != record || current.proposalSHA256 != proposalDigest {
		return ErrReconciliationRequired
	}
	if _, used := runtime.reconciliations[reconciliation.ReconciliationID]; used {
		return ErrReplayDetected
	}
	runtime.reconciliations[reconciliation.ReconciliationID] = reconciliationDigest
	current.result = cloneActionResult(observed)
	current.err = ErrReconciliationRequired
	if reconciliation.RetryPermitted {
		delete(runtime.records, proposal.IdempotencyKey)
	}
	return nil
}
