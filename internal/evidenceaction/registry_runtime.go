package evidenceaction

import (
	"sync"
	"time"
)

type RegisteredOperation struct {
	Approval   OperationApproval
	Action     ActionType
	SideEffect SideEffectClass
}

// OperationRegistry is an immutable snapshot of explicitly approved operations.
// Resolution binds an action proposal to the exact operation, capability, action,
// side effect, and registry identity that was approved; callers cannot synthesize
// an approval or reinterpret an operation name.
type operationKey struct {
	operationRef     string
	operationVersion string
	capabilityRef    string
	capabilitySHA256 string
}

type OperationRegistry struct {
	mu             sync.RWMutex
	registryRef    string
	registrySHA256 string
	operations     map[operationKey]RegisteredOperation
}

func NewOperationRegistry(operations []RegisteredOperation) (*OperationRegistry, error) {
	if len(operations) == 0 || len(operations) > MaxRefs {
		return nil, ErrActionContractInvalid
	}
	registry := &OperationRegistry{operations: make(map[operationKey]RegisteredOperation, len(operations))}
	for index, operation := range operations {
		approval := operation.Approval
		if !validApproval(approval) || !approval.Approved || !validAction(operation.Action) || !validSideEffect(operation.SideEffect) ||
			((operation.Action == ActionInspect || operation.Action == ActionExport) && operation.SideEffect != EffectReadOnly) {
			return nil, ErrActionContractInvalid
		}
		if index == 0 {
			registry.registryRef = approval.RegistryRef
			registry.registrySHA256 = approval.RegistrySHA256
		} else if approval.RegistryRef != registry.registryRef || approval.RegistrySHA256 != registry.registrySHA256 {
			return nil, ErrActionContractInvalid
		}
		key := operationApprovalKey(approval.OperationRef, approval.OperationVersion, approval.CapabilitySnapshotRef, approval.CapabilitySnapshotSHA256)
		if _, duplicate := registry.operations[key]; duplicate {
			return nil, ErrActionContractInvalid
		}
		registry.operations[key] = operation
	}
	return registry, nil
}

func (registry *OperationRegistry) Resolve(proposal ActionProposal, now time.Time) (OperationApproval, error) {
	if registry == nil || now.IsZero() {
		return OperationApproval{}, ErrCapabilityStale
	}
	if err := ValidateActionProposal(proposal); err != nil {
		return OperationApproval{}, err
	}
	key := operationApprovalKey(proposal.OperationRef, proposal.OperationVersion, proposal.CapabilitySnapshotRef, proposal.CapabilitySnapshotSHA256)
	registry.mu.RLock()
	operation, found := registry.operations[key]
	registry.mu.RUnlock()
	if !found || operation.Action != proposal.Action || operation.SideEffect != proposal.SideEffect || !now.UTC().Before(operation.Approval.ExpiresAt) {
		return OperationApproval{}, ErrCapabilityStale
	}
	if err := ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, operation.Approval, now.UTC()); err != nil {
		return OperationApproval{}, err
	}
	return operation.Approval, nil
}

func (registry *OperationRegistry) Identity() (string, string) {
	if registry == nil {
		return "", ""
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.registryRef, registry.registrySHA256
}

func operationApprovalKey(operationRef, operationVersion, capabilityRef, capabilitySHA256 string) operationKey {
	return operationKey{
		operationRef:     operationRef,
		operationVersion: operationVersion,
		capabilityRef:    capabilityRef,
		capabilitySHA256: capabilitySHA256,
	}
}
