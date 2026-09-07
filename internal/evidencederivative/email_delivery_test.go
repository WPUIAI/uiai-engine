package evidencederivative

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEmailDeliveryPolicyFailsClosed(t *testing.T) {
	command := deliveryCommand()
	if err := ValidateEmailDeliveryPolicy(*command.EmailPolicy); err != nil {
		t.Fatal(err)
	}
	cases := []func(*EmailDeliveryPolicy){
		func(policy *EmailDeliveryPolicy) { policy.TLSRequired = false },
		func(policy *EmailDeliveryPolicy) { policy.RecipientConsentRef = "" },
		func(policy *EmailDeliveryPolicy) { policy.SuppressionEvidenceRef = "" },
		func(policy *EmailDeliveryPolicy) { policy.DKIMEvidenceRef = "" },
		func(policy *EmailDeliveryPolicy) { policy.SPFEvidenceRef = "" },
		func(policy *EmailDeliveryPolicy) { policy.DMARCEvidenceRef = "" },
		func(policy *EmailDeliveryPolicy) { policy.NoTracking = false },
		func(policy *EmailDeliveryPolicy) { policy.MaxMessageBytes = MaxEmailMessageBytes + 1 },
		func(policy *EmailDeliveryPolicy) { policy.BounceReconciliationRef = "" },
	}
	for index, mutate := range cases {
		policy := *command.EmailPolicy
		mutate(&policy)
		if err := ValidateEmailDeliveryPolicy(policy); !errors.Is(err, ErrDerivativeContractInvalid) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestDeliveryRuntimeRequiresProviderAcceptanceEvidence(t *testing.T) {
	command := deliveryCommand()
	transport := &fakeDeliveryTransport{result: ProviderDeliveryResult{
		State: DeliveryAccepted, ProviderReceiptRef: "smtp:1", ObservedAt: time.Unix(10, 0).UTC(),
	}}
	runtime := NewDeliveryRuntime()
	receipt, err := runtime.Deliver(context.Background(), command, transport)
	if !errors.Is(err, ErrDeliveryRetryBlocked) || receipt.State != DeliveryOutcomeUnknown || transport.calls != 1 {
		t.Fatalf("evidence-free acceptance receipt = %#v error = %v calls = %d", receipt, err, transport.calls)
	}
	if _, err = runtime.Deliver(context.Background(), command, transport); !errors.Is(err, ErrDeliveryRetryBlocked) || transport.calls != 1 {
		t.Fatalf("evidence-free acceptance retried: error = %v calls = %d", err, transport.calls)
	}
}

func TestDeliveryRuntimeRejectsOversizePolicyBoundPayload(t *testing.T) {
	command := deliveryCommand()
	command.EmailPolicy.MaxMessageBytes = 1
	transport := &fakeDeliveryTransport{}
	if _, err := NewDeliveryRuntime().Deliver(context.Background(), command, transport); !errors.Is(err, ErrDerivativeContractInvalid) || transport.calls != 0 {
		t.Fatalf("oversize delivery error = %v calls = %d", err, transport.calls)
	}
}
