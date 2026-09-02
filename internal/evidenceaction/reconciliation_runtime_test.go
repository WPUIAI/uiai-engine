package evidenceaction

import (
	"errors"
	"testing"
)

func TestReconcileActionResultAllowsRetryOnlyAfterAuthoritativeNoEffect(t *testing.T) {
	_, _, _, _, result, _, _, now := validContracts(t)
	result.Status = StatusOutcomeUnknown
	result.ErrorCodes = []string{"provider_outcome_unknown"}
	observations := []ReconciliationObservation{
		{InspectionRef: "inspection:z", RetrySafe: true},
		{InspectionRef: "inspection:a", RetrySafe: true},
	}
	reconciliation, err := ReconcileActionResult(result, observations, now)
	if err != nil {
		t.Fatal(err)
	}
	if reconciliation.State != ReconciliationConsistent || !reconciliation.RetryPermitted || reconciliation.RetryReasonCode != "authoritative_no_effect" {
		t.Fatalf("reconciliation = %#v", reconciliation)
	}
	if reconciliation.AuthoritativeInspectionRefs[0] != "inspection:a" || observations[0].InspectionRef != "inspection:z" {
		t.Fatalf("inspection ordering mutated or nondeterministic: result=%#v input=%#v", reconciliation.AuthoritativeInspectionRefs, observations)
	}
}

func TestReconcileActionResultBlocksRetryOnConflictOrPartialEffect(t *testing.T) {
	_, _, _, _, result, _, _, now := validContracts(t)
	result.Status = StatusOutcomeUnknown
	result.ErrorCodes = []string{"provider_outcome_unknown"}
	conflict, err := ReconcileActionResult(result, []ReconciliationObservation{
		{InspectionRef: "inspection:1", RetrySafe: true},
		{InspectionRef: "inspection:2", EffectOccurred: true},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.State != ReconciliationConflict || conflict.RetryPermitted {
		t.Fatalf("conflict reconciliation = %#v", conflict)
	}

	result.Status = StatusPartiallyApplied
	partial, err := ReconcileActionResult(result, []ReconciliationObservation{{InspectionRef: "inspection:3", RetrySafe: true}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if partial.State != ReconciliationConsistent || partial.RetryPermitted {
		t.Fatalf("partial reconciliation = %#v", partial)
	}
}

func TestReconcileActionResultRejectsUnsupportedOrAmbiguousEvidence(t *testing.T) {
	_, _, _, _, result, _, _, now := validContracts(t)
	if _, err := ReconcileActionResult(result, []ReconciliationObservation{{InspectionRef: "inspection:1"}}, now); !errors.Is(err, ErrReconciliationRequired) {
		t.Fatalf("successful result error = %v", err)
	}
	result.Status = StatusOutcomeUnknown
	result.ErrorCodes = []string{"provider_outcome_unknown"}
	if _, err := ReconcileActionResult(result, nil, now); !errors.Is(err, ErrReconciliationRequired) {
		t.Fatalf("missing observations error = %v", err)
	}
	if _, err := ReconcileActionResult(result, []ReconciliationObservation{{InspectionRef: "inspection:1"}, {InspectionRef: "inspection:1"}}, now); !errors.Is(err, ErrActionContractInvalid) {
		t.Fatalf("duplicate observations error = %v", err)
	}
}
