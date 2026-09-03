package evidencederivative

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeDeliveryTransport struct {
	calls  int
	result ProviderDeliveryResult
	err    error
}

func (f *fakeDeliveryTransport) Send(_ context.Context, _ DeliveryCommand) (ProviderDeliveryResult, error) {
	f.calls++
	return f.result, f.err
}
func deliveryCommand() DeliveryCommand {
	body := []byte("mail body")
	sum := sha256.Sum256(body)
	policy := EmailDeliveryPolicy{
		Schema: EmailDeliveryPolicySchema, PolicyRef: "email-policy:1", Transport: EmailTransportSMTP,
		TLSRequired: true, RecipientConsentRef: "consent:1", SuppressionEvidenceRef: "suppression:1",
		DKIMEvidenceRef: "dkim:1", SPFEvidenceRef: "spf:1", DMARCEvidenceRef: "dmarc:1", NoTracking: true,
		MaxMessageBytes: 1024, BounceReconciliationRef: "bounce-policy:1",
	}
	return DeliveryCommand{DerivativeRef: "derivative:1", DerivativeSHA256: hex.EncodeToString(sum[:]), DestinationRef: "mailbox:1", IdempotencyKey: "key:1", Payload: body, EmailPolicy: &policy}
}
func TestDeliveryRuntimeIdempotencyAndConflict(t *testing.T) {
	now := time.Unix(10, 0).UTC()
	tr := &fakeDeliveryTransport{result: ProviderDeliveryResult{State: DeliveryAccepted, ProviderReceiptRef: "smtp:1", EvidenceRefs: []string{"evidence:smtp"}, ObservedAt: now}}
	r := NewDeliveryRuntime()
	c := deliveryCommand()
	a, e := r.Deliver(context.Background(), c, tr)
	if e != nil {
		t.Fatal(e)
	}
	if err := ValidateEmailDeliveryReceipt(a, *c.EmailPolicy); err != nil {
		t.Fatal(err)
	}
	*a.AcceptedAt = time.Unix(99, 0).UTC()
	b, e := r.Deliver(context.Background(), c, tr)
	if e != nil || a.DeliveryID != b.DeliveryID || b.AcceptedAt.Equal(*a.AcceptedAt) || tr.calls != 1 {
		t.Fatal("idempotency failed")
	}
	bad := c
	bad.DestinationRef = "mailbox:2"
	if _, e = r.Deliver(context.Background(), bad, tr); !errors.Is(e, ErrDeliveryConflict) {
		t.Fatalf("error=%v", e)
	}
	other := c
	other.DestinationRef = "mailbox:2"
	other.IdempotencyKey = "key:2"
	otherReceipt, e := r.Deliver(context.Background(), other, tr)
	if e != nil || otherReceipt.DeliveryID == b.DeliveryID {
		t.Fatalf("delivery identity did not bind destination: %#v %v", otherReceipt, e)
	}
}
func TestDeliveryRuntimeConcurrentIdempotency(t *testing.T) {
	now := time.Unix(10, 0).UTC()
	transport := &fakeDeliveryTransport{result: ProviderDeliveryResult{State: DeliveryAccepted, ProviderReceiptRef: "smtp:1", EvidenceRefs: []string{"evidence:smtp"}, ObservedAt: now}}
	runtime := NewDeliveryRuntime()
	command := deliveryCommand()
	var wait sync.WaitGroup
	ids := make(chan string, 32)
	errorsSeen := make(chan error, 32)
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			receipt, err := runtime.Deliver(context.Background(), command, transport)
			if err != nil {
				errorsSeen <- err
				return
			}
			ids <- receipt.DeliveryID
		}()
	}
	wait.Wait()
	close(ids)
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	var expected string
	for id := range ids {
		if expected == "" {
			expected = id
		} else if id != expected {
			t.Fatalf("delivery IDs differ: %s != %s", id, expected)
		}
	}
	if transport.calls != 1 {
		t.Fatalf("transport calls = %d", transport.calls)
	}
}

func TestDeliveryRuntimeUnknownRequiresReconciliation(t *testing.T) {
	tr := &fakeDeliveryTransport{err: errors.New("timeout")}
	r := NewDeliveryRuntime()
	c := deliveryCommand()
	receipt, e := r.Deliver(context.Background(), c, tr)
	if !errors.Is(e, ErrDeliveryRetryBlocked) || receipt.State != DeliveryOutcomeUnknown || receipt.RetryPermitted {
		t.Fatalf("receipt=%#v err=%v", receipt, e)
	}
	again, e := r.Deliver(context.Background(), c, tr)
	if !errors.Is(e, ErrDeliveryRetryBlocked) || again.State != DeliveryOutcomeUnknown || tr.calls != 1 {
		t.Fatalf("unknown replay lost warning or retried: %#v %v calls=%d", again, e, tr.calls)
	}
	done, e := r.Reconcile(c.IdempotencyKey, DeliveryDelivered, "imap:1", []string{"evidence:imap"}, time.Unix(20, 0).UTC())
	if e != nil || done.State != DeliveryDelivered || done.RetryPermitted {
		t.Fatalf("reconcile=%#v err=%v", done, e)
	}
}
