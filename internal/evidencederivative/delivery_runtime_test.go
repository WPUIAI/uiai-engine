package evidencederivative

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
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
	outputSHA := hex.EncodeToString(sum[:])
	digest := strings.Repeat("a", 64)
	createdAt := time.Unix(9, 0).UTC()
	delivery, err := epwadelivery.New(epwadelivery.Input{
		Producer: epwadelivery.ProducerDerivative,
		Artifact: epwadelivery.ArtifactBinding{ArtifactRef: "derivative:1", Revision: 1, ManifestSHA256: digest, OutputSHA256: outputSHA},
		EPWA:     epwadelivery.EPWABinding{PackageID: digest, ProjectionRef: "uiai-evidence-projection:sha256:" + digest, ProjectionSHA256: digest, PackageRef: "uiai-epwa-package:sha256:" + digest, PackageSHA256: digest, RecordURL: "https://evidence.example/derivative/1/", PortableURL: "https://evidence.example/derivative/1/portable.zip", Access: epwadelivery.AccessPublicSafe},
		Scope:    epwadelivery.ScopeBinding{Posture: epwadelivery.ScopeComplete, ProjectRef: "project:1", WorkstreamRef: "workstream:1", WorksetRef: "workset:1", CallGraphRef: "callgraph:1", WorkpointRef: "workpoint:1", WorkItemRef: "work-item:1", ContinuityRef: "continuity:1"},
		State:    epwadelivery.StateReady, IdempotencyKey: "derivative-epwa:1", CreatedAt: createdAt, ObservedAt: createdAt,
	})
	if err != nil {
		panic(err)
	}
	return DeliveryCommand{DerivativeRef: "derivative:1", DerivativeSHA256: outputSHA, DestinationRef: "mailbox:1", IdempotencyKey: "key:1", Payload: body, EmailPolicy: &policy, EPWADelivery: delivery}
}
func TestDeliveryRuntimeRequiresReadyBoundEPWA(t *testing.T) {
	transport := &fakeDeliveryTransport{}
	command := deliveryCommand()
	command.EPWADelivery = epwadelivery.Delivery{}
	if _, err := NewDeliveryRuntime().Deliver(context.Background(), command, transport); !errors.Is(err, ErrDerivativeContractInvalid) || transport.calls != 0 {
		t.Fatalf("raw derivative reached connector without EPWA: error=%v calls=%d", err, transport.calls)
	}
	command = deliveryCommand()
	command.EPWADelivery.Artifact.OutputSHA256 = strings.Repeat("b", 64)
	if _, err := NewDeliveryRuntime().Deliver(context.Background(), command, transport); !errors.Is(err, ErrDerivativeContractInvalid) || transport.calls != 0 {
		t.Fatalf("mismatched EPWA derivative reached connector: error=%v calls=%d", err, transport.calls)
	}
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

func TestDeliveryRuntimeCachesEveryPostExecutionUncertainty(t *testing.T) {
	for name, result := range map[string]ProviderDeliveryResult{
		"explicit unknown":  {State: DeliveryOutcomeUnknown, ProviderReceiptRef: "provider:maybe", EvidenceRefs: []string{"evidence:maybe"}, ObservedAt: time.Unix(10, 0).UTC()},
		"invalid accepted":  {State: DeliveryAccepted, ObservedAt: time.Unix(10, 0).UTC()},
		"unsupported state": {State: DeliveryDelivered, ProviderReceiptRef: "provider:too-soon", EvidenceRefs: []string{"evidence:too-soon"}, ObservedAt: time.Unix(10, 0).UTC()},
	} {
		t.Run(name, func(t *testing.T) {
			transport := &fakeDeliveryTransport{result: result}
			runtime := NewDeliveryRuntime()
			command := deliveryCommand()
			receipt, err := runtime.Deliver(context.Background(), command, transport)
			if !errors.Is(err, ErrDeliveryRetryBlocked) || receipt.State != DeliveryOutcomeUnknown || receipt.ProviderReceiptRef != "" || len(receipt.EvidenceRefs) != 1 {
				t.Fatalf("receipt=%#v err=%v", receipt, err)
			}
			again, err := runtime.Deliver(context.Background(), command, transport)
			if !errors.Is(err, ErrDeliveryRetryBlocked) || again.DeliveryID != receipt.DeliveryID || transport.calls != 1 {
				t.Fatalf("uncertain result retried: receipt=%#v err=%v calls=%d", again, err, transport.calls)
			}
		})
	}
}

func TestDeliveryRuntimeUnknownRequiresReconciliation(t *testing.T) {
	tr := &fakeDeliveryTransport{result: ProviderDeliveryResult{ObservedAt: time.Unix(10, 0).UTC()}, err: errors.New("timeout")}
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
	reconcileAt := time.Unix(20, 0).UTC()
	reconcileEvidence := []string{"evidence:imap"}
	done, e := r.Reconcile(c.IdempotencyKey, DeliveryDelivered, "imap:1", reconcileEvidence, reconcileAt)
	if e != nil || done.State != DeliveryDelivered || done.RetryPermitted {
		t.Fatalf("reconcile=%#v err=%v", done, e)
	}
	replayed, e := r.Reconcile(c.IdempotencyKey, DeliveryDelivered, "imap:1", reconcileEvidence, reconcileAt)
	if e != nil || replayed.DeliveryID != done.DeliveryID || !reflect.DeepEqual(replayed, done) {
		t.Fatalf("reconciliation replay = %#v err=%v", replayed, e)
	}
	if _, e = r.Reconcile(c.IdempotencyKey, DeliveryRejected, "imap:2", []string{"evidence:changed"}, time.Unix(30, 0).UTC()); !errors.Is(e, ErrDeliveryConflict) {
		t.Fatalf("terminal reconciliation mutation error = %v", e)
	}
}
