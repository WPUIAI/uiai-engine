package evidencederivative

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	return DeliveryCommand{DerivativeRef: "derivative:1", DerivativeSHA256: hex.EncodeToString(sum[:]), DestinationRef: "mailbox:1", IdempotencyKey: "key:1", Payload: body}
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
	if e != nil || again.State != DeliveryOutcomeUnknown || tr.calls != 1 {
		t.Fatalf("blind retry occurred: %#v %v calls=%d", again, e, tr.calls)
	}
	done, e := r.Reconcile(c.IdempotencyKey, DeliveryDelivered, "imap:1", []string{"evidence:imap"}, time.Unix(20, 0).UTC())
	if e != nil || done.State != DeliveryDelivered || done.RetryPermitted {
		t.Fatalf("reconcile=%#v err=%v", done, e)
	}
}
