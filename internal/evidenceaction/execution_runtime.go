package evidenceaction

import (
	"context"
	"fmt"
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
	done           chan struct{}
	result         ActionResult
	err            error
}

type ExecutionRuntime struct {
	mu      sync.Mutex
	records map[string]*executionRecord
}

func NewExecutionRuntime() *ExecutionRuntime {
	return &ExecutionRuntime{records: map[string]*executionRecord{}}
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
	} else if proposal.SideEffect != EffectReadOnly {
		return ActionResult{}, ErrPreviewRequired
	}

	record, owner, err := runtime.reserve(proposal.IdempotencyKey, proposalDigest)
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
	if executionErr != nil {
		executionErr = fmt.Errorf("%w: %v", ErrOutcomeUnknown, executionErr)
		runtime.finish(record, ActionResult{}, executionErr)
		return ActionResult{}, executionErr
	}
	if err := ValidateActionResultAgainst(result, proposal, preview); err != nil {
		executionErr = fmt.Errorf("%w: invalid executor result", ErrOutcomeUnknown)
		runtime.finish(record, ActionResult{}, executionErr)
		return ActionResult{}, executionErr
	}
	result = cloneActionResult(result)
	runtime.finish(record, result, nil)
	return cloneActionResult(result), nil
}

func (runtime *ExecutionRuntime) reserve(key, proposalDigest string) (*executionRecord, bool, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if record, exists := runtime.records[key]; exists {
		if record.proposalSHA256 != proposalDigest {
			return nil, false, ErrReplayDetected
		}
		return record, false, nil
	}
	record := &executionRecord{proposalSHA256: proposalDigest, done: make(chan struct{})}
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

func cloneActionResult(result ActionResult) ActionResult {
	result.AppliedEffects = append([]AppliedEffect(nil), result.AppliedEffects...)
	result.Compensations = append([]Compensation(nil), result.Compensations...)
	result.ProviderReceiptRefs = append([]string(nil), result.ProviderReceiptRefs...)
	result.ErrorCodes = append([]string(nil), result.ErrorCodes...)
	return result
}
