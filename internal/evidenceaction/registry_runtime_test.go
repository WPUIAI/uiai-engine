package evidenceaction

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestOperationRegistryResolvesExactApprovedOperation(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	operations := []RegisteredOperation{{Approval: approval, Action: proposal.Action, SideEffect: proposal.SideEffect}}
	original := append([]RegisteredOperation(nil), operations...)
	registry, err := NewOperationRegistry(operations)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(operations, original) {
		t.Fatal("registry construction mutated input")
	}
	registryRef, registrySHA256 := registry.Identity()
	if registryRef != approval.RegistryRef || registrySHA256 != approval.RegistrySHA256 {
		t.Fatalf("registry identity = %q %q", registryRef, registrySHA256)
	}
	resolved, err := registry.Resolve(proposal, now)
	if err != nil || resolved != approval {
		t.Fatalf("Resolve() = %#v, %v", resolved, err)
	}

	mutations := map[string]func(*ActionProposal){
		"operation version": func(value *ActionProposal) { value.OperationVersion = "v2" },
		"capability digest": func(value *ActionProposal) { value.CapabilitySnapshotSHA256 = digestA },
		"action":            func(value *ActionProposal) { value.Action = ActionFollowUp },
		"side effect":       func(value *ActionProposal) { value.SideEffect = EffectExternalMutation },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := proposal
			mutate(&candidate)
			if _, err := registry.Resolve(candidate, now); !errors.Is(err, ErrCapabilityStale) {
				t.Fatalf("Resolve() error = %v, want ErrCapabilityStale", err)
			}
		})
	}
	if _, err := registry.Resolve(proposal, approval.ExpiresAt); !errors.Is(err, ErrCapabilityStale) {
		t.Fatalf("expired Resolve() error = %v, want ErrCapabilityStale", err)
	}
	if _, err := registry.Resolve(proposal, time.Time{}); !errors.Is(err, ErrCapabilityStale) {
		t.Fatalf("zero-time Resolve() error = %v, want ErrCapabilityStale", err)
	}
}

func TestOperationRegistryRejectsAmbiguousOrUnsafeDefinitions(t *testing.T) {
	proposal, approval, _, _, _, _, _, _ := validContracts(t)
	valid := RegisteredOperation{Approval: approval, Action: proposal.Action, SideEffect: proposal.SideEffect}
	tests := map[string][]RegisteredOperation{
		"empty":     nil,
		"duplicate": {valid, valid},
		"unapproved": func() []RegisteredOperation {
			candidate := valid
			candidate.Approval.Approved = false
			return []RegisteredOperation{candidate}
		}(),
		"mixed registry": func() []RegisteredOperation {
			candidate := valid
			candidate.Approval.OperationRef = "operation:other"
			candidate.Approval.RegistrySHA256 = digestB
			return []RegisteredOperation{valid, candidate}
		}(),
		"unsafe inspect": func() []RegisteredOperation {
			candidate := valid
			candidate.Action = ActionInspect
			candidate.SideEffect = EffectExternalMutation
			return []RegisteredOperation{candidate}
		}(),
	}
	for name, operations := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewOperationRegistry(operations); !errors.Is(err, ErrActionContractInvalid) {
				t.Fatalf("NewOperationRegistry() error = %v, want ErrActionContractInvalid", err)
			}
		})
	}
}

func TestPreviewRuntimeBuildsOnlyFromRegisteredOperation(t *testing.T) {
	proposal, approval, _, _, _, _, _, now := validContracts(t)
	registry, err := NewOperationRegistry([]RegisteredOperation{{Approval: approval, Action: proposal.Action, SideEffect: proposal.SideEffect}})
	if err != nil {
		t.Fatal(err)
	}
	previews := NewPreviewRuntime(func() time.Time { return now }, func() (string, error) { return "nonce:registered", nil })
	effects := []ExpectedEffect{{EffectRef: "effect:1", TargetRef: "atom:1", Kind: "append_capture"}}
	preview, err := previews.BuildRegisteredPreview(proposal, registry, effects, RiskModerate, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateActionPreviewAgainst(preview, proposal, now); err != nil {
		t.Fatal(err)
	}
	if _, err := previews.BuildRegisteredPreview(proposal, nil, effects, RiskModerate, time.Minute); !errors.Is(err, ErrCapabilityStale) {
		t.Fatalf("nil registry error = %v, want ErrCapabilityStale", err)
	}
}
