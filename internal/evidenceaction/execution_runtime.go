package evidenceaction

import (
	"context"
	"errors"
	"sync"
)

type ActionExecutor interface {
	Execute(context.Context, ActionProposal, *ActionPreview) (ActionResult, error)
}

type ActionExecutorFunc func(context.Context, ActionProposal, *ActionPreview) (ActionResult, error)

func (function ActionExecutorFunc) Execute(ctx context.Context, proposal ActionProposal, preview *ActionPreview) (ActionResult, error) {
	return function(ctx, proposal, preview)
}

type executionRecord struct {
	proposalSHA256 string
	previewSHA256  string
	preview        *ActionPreview
	done           chan struct{}
	result         ActionResult
	err            error
}

type ExecutionRuntime struct {
	mu              sync.Mutex
	records         map[string]*executionRecord
	reconciliations map[string]string
}

func NewExecutionRuntime() *ExecutionRuntime {
	return &ExecutionRuntime{
		records:         map[string]*executionRecord{},
		reconciliations: map[string]string{},
	}
}

func (runtime *ExecutionRuntime) Execute(ctx context.Context, proposal ActionProposal, preview *ActionPreview, confirmation *ActionConfirmation, actorRef string, previews *PreviewRuntime, executor ActionExecutor) (ActionResult, error) {
	if executor == nil {
		return ActionResult{}, ErrActionContractInvalid
	}
	if err := ValidateActionProposal(proposal); err != nil {
		return ActionResult{}, err
	}
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		return ActionResult{}, err
	}
	previewDigest := ""
	if preview != nil {
		if previews == nil {
			return ActionResult{}, ErrPreviewMismatch
		}
		if err := previews.ValidateIssued(*preview); err != nil {
			return ActionResult{}, err
		}
		if err := ValidateActionPreviewAgainst(*preview, proposal, previews.now().UTC()); err != nil {
			return ActionResult{}, err
		}
		previewDigest, err = DigestActionPreview(*preview)
		if err != nil {
			return ActionResult{}, err
		}
	} else if proposal.SideEffect != EffectReadOnly {
		return ActionResult{}, ErrPreviewRequired
	}

	record, owner, err := runtime.reserve(proposal.IdempotencyKey, proposalDigest, previewDigest, preview)
	if err != nil {
		return ActionResult{}, err
	}
	if !owner {
		select {
		case <-record.done:
			return cloneActionResult(record.result), record.err
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		}
	}

	if preview != nil && preview.ConfirmationRequired {
		if previews == nil || confirmation == nil {
			runtime.releaseFailedValidation(proposal.IdempotencyKey, record, ErrConfirmationRequired)
			return ActionResult{}, ErrConfirmationRequired
		}
		if err := previews.ConsumeConfirmation(*confirmation, *preview, actorRef); err != nil {
			runtime.releaseFailedValidation(proposal.IdempotencyKey, record, err)
			return ActionResult{}, err
		}
	}

	result, executionErr := executor.Execute(ctx, proposal, preview)
	result = cloneActionResult(result)
	validationErr := ValidateActionResultAgainst(result, proposal, preview)
	if executionErr != nil {
		if (result.Status == StatusOutcomeUnknown || result.Status == StatusPartiallyApplied) &&
			(validationErr == nil || errors.Is(validationErr, ErrOutcomeUnknown)) {
			runtime.finish(record, result, ErrOutcomeUnknown)
			return cloneActionResult(result), ErrOutcomeUnknown
		}
		runtime.finish(record, ActionResult{}, ErrOutcomeUnknown)
		return ActionResult{}, ErrOutcomeUnknown
	}
	if errors.Is(validationErr, ErrOutcomeUnknown) {
		runtime.finish(record, result, ErrOutcomeUnknown)
		return cloneActionResult(result), ErrOutcomeUnknown
	}
	if validationErr != nil {
		runtime.finish(record, ActionResult{}, ErrOutcomeUnknown)
		return ActionResult{}, ErrOutcomeUnknown
	}
	if result.Status == StatusPartiallyApplied {
		runtime.finish(record, result, ErrReconciliationRequired)
		return cloneActionResult(result), ErrReconciliationRequired
	}
	runtime.finish(record, result, nil)
	return cloneActionResult(result), nil
}

func (runtime *ExecutionRuntime) reserve(key, proposalDigest, previewDigest string, preview *ActionPreview) (*executionRecord, bool, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if record, exists := runtime.records[key]; exists {
		if record.proposalSHA256 != proposalDigest || record.previewSHA256 != previewDigest {
			return nil, false, ErrReplayDetected
		}
		return record, false, nil
	}
	record := &executionRecord{
		proposalSHA256: proposalDigest,
		previewSHA256:  previewDigest,
		preview:        cloneActionPreview(preview),
		done:           make(chan struct{}),
	}
	runtime.records[key] = record
	return record, true, nil
}

func (runtime *ExecutionRuntime) releaseFailedValidation(key string, record *executionRecord, err error) {
	runtime.mu.Lock()
	if runtime.records[key] == record {
		delete(runtime.records, key)
	}
	record.err = err
	close(record.done)
	runtime.mu.Unlock()
}

func (runtime *ExecutionRuntime) finish(record *executionRecord, result ActionResult, err error) {
	runtime.mu.Lock()
	record.result = cloneActionResult(result)
	record.err = err
	close(record.done)
	runtime.mu.Unlock()
}

func cloneActionPreview(preview *ActionPreview) *ActionPreview {
	if preview == nil {
		return nil
	}
	cloned := *preview
	cloned.TargetRefs = append([]string(nil), preview.TargetRefs...)
	cloned.ExpectedEffects = append([]ExpectedEffect(nil), preview.ExpectedEffects...)
	return &cloned
}

func cloneActionResult(result ActionResult) ActionResult {
	result.AppliedEffects = append([]AppliedEffect(nil), result.AppliedEffects...)
	result.Compensations = append([]Compensation(nil), result.Compensations...)
	result.ProviderReceiptRefs = append([]string(nil), result.ProviderReceiptRefs...)
	result.ErrorCodes = append([]string(nil), result.ErrorCodes...)
	return result
}
