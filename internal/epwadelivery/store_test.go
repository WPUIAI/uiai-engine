package epwadelivery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeliveryStoreReplaysIdempotentReconciliation(t *testing.T) {
	root := t.TempDir()
	pendingInput := validInput()
	pendingInput.State = StatePendingReconcile
	pendingInput.EPWA.RecordURL = ""
	pendingInput.EPWA.PortableURL = ""
	pendingInput.RecoveryRef = "reconcile:https-required"
	pending, err := New(pendingInput)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Record(root, pending)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := pending
	duplicate.ObservedAt = duplicate.ObservedAt.Add(time.Minute)
	got, err := Record(root, duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != first.Revision || got.ObservedAt != first.ObservedAt {
		t.Fatalf("idempotent retry appended a revision: first=%#v got=%#v", first, got)
	}
	ready, err := New(validInput())
	if err != nil {
		t.Fatal(err)
	}
	ready.ObservedAt = pending.ObservedAt.Add(2 * time.Minute)
	settled, err := Record(root, ready)
	if err != nil {
		t.Fatal(err)
	}
	if settled.DeliveryID != first.DeliveryID || settled.Revision != 2 || settled.State != StateReady || settled.CreatedAt != first.CreatedAt {
		t.Fatalf("unexpected reconciliation revision: %#v", settled)
	}
	replayed, err := Load(root, settled.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed != settled {
		t.Fatalf("replay mismatch: got=%#v want=%#v", replayed, settled)
	}
}

func TestDeliveryStoreDoesNotDowngradeReadyWhenHTTPSDiscoveryIsTemporarilyUnavailable(t *testing.T) {
	root := t.TempDir()
	ready, err := New(validInput())
	if err != nil {
		t.Fatal(err)
	}
	ready, err = Record(root, ready)
	if err != nil {
		t.Fatal(err)
	}
	pendingInput := validInput()
	pendingInput.State = StatePendingReconcile
	pendingInput.EPWA.RecordURL = ""
	pendingInput.EPWA.PortableURL = ""
	pendingInput.RecoveryRef = "reconcile:https-required"
	pendingInput.ObservedAt = ready.ObservedAt.Add(time.Minute)
	pending, err := New(pendingInput)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Record(root, pending)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.State != StateReady || replayed.Revision != ready.Revision || replayed.EPWA.RecordURL != ready.EPWA.RecordURL {
		t.Fatalf("ready delivery was downgraded: ready=%#v replayed=%#v", ready, replayed)
	}
}

func TestDeliveryStoreDetectsLedgerTampering(t *testing.T) {
	root := t.TempDir()
	delivery, err := New(validInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Record(root, delivery); err != nil {
		t.Fatal(err)
	}
	key := delivery.DeliveryID[len("uiai-epwa-delivery:sha256:"):]
	path := filepath.Join(root, "epwa-delivery", key+".jsonl")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)/2] ^= 1
	if err := os.WriteFile(path, body, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, delivery.DeliveryID); err != ErrStoreCorrupt {
		t.Fatalf("tampered ledger error=%v, want %v", err, ErrStoreCorrupt)
	}
}
