package epwadelivery

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDeliveryContractRequiresHTTPSPortableEPWA(t *testing.T) {
	input := validInput()
	first, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !strings.HasPrefix(first.DeliveryID, "uiai-epwa-delivery:sha256:") {
		t.Fatalf("delivery is not deterministic: %#v %#v", first, second)
	}
	if err := RequireReady(first); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Delivery){
		"http viewer":        func(value *Delivery) { value.EPWA.RecordURL = "http://evidence.example/record" },
		"raw local path":     func(value *Delivery) { value.EPWA.PortableURL = "/tmp/evidence.zip" },
		"non archive":        func(value *Delivery) { value.EPWA.PortableURL = "https://evidence.example/raw.png" },
		"package drift":      func(value *Delivery) { value.EPWA.PackageSHA256 = strings.Repeat("b", 64) },
		"projection missing": func(value *Delivery) { value.EPWA.ProjectionRef = "" },
		"scope missing":      func(value *Delivery) { value.Scope.WorkItemRef = "" },
		"continuity missing": func(value *Delivery) { value.Scope.ContinuityRef = "" },
		"truth conflated":    func(value *Delivery) { value.TruthNotice = "delivered means complete" },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := first
			mutate(&mutated)
			if !errors.Is(Validate(mutated), ErrInvalidContract) {
				t.Fatalf("mutation accepted: %#v", mutated)
			}
		})
	}
}

func TestBlockedDeliveryWithholdsUnscopedPackage(t *testing.T) {
	input := validInput()
	input.State = StateBlocked
	input.Scope = ScopeBinding{Posture: ScopeBlocked, WorkpointRef: "workpoint:known"}
	input.EPWA.ProjectionRef = ""
	input.EPWA.ProjectionSHA256 = ""
	input.EPWA.RecordURL = ""
	input.EPWA.PortableURL = ""
	input.EPWA.Access = AccessRedacted
	input.RecoveryRef = "reconcile:scope-required"
	delivery, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(RequireReady(delivery), ErrDeliveryBlocked) {
		t.Fatalf("blocked delivery reported ready: %#v", delivery)
	}
	if delivery.EPWA.RecordURL != "" || delivery.EPWA.PortableURL != "" {
		t.Fatalf("blocked unscoped record exposed delivery links: %#v", delivery)
	}
}

func TestDeliveryIdentitySurvivesHTTPSReconciliation(t *testing.T) {
	ready, err := New(validInput())
	if err != nil {
		t.Fatal(err)
	}
	pendingInput := validInput()
	pendingInput.State = StatePendingReconcile
	pendingInput.EPWA.RecordURL = ""
	pendingInput.EPWA.PortableURL = ""
	pendingInput.RecoveryRef = "reconcile:https-required"
	pending, err := New(pendingInput)
	if err != nil {
		t.Fatal(err)
	}
	if pending.DeliveryID != ready.DeliveryID {
		t.Fatalf("delivery identity changed across reconciliation: pending=%s ready=%s", pending.DeliveryID, ready.DeliveryID)
	}
}

func validInput() Input {
	now := time.Date(2026, 9, 4, 5, 0, 0, 0, time.UTC)
	digest := strings.Repeat("a", 64)
	return Input{
		Producer: ProducerScreenshot,
		Artifact: ArtifactBinding{ArtifactRef: "artifact:test", Revision: 1, ManifestSHA256: digest, OutputSHA256: digest},
		EPWA: EPWABinding{
			PackageID: digest, ProjectionRef: "uiai-evidence-projection:sha256:" + digest, ProjectionSHA256: digest,
			PackageRef: "uiai-epwa-package:sha256:" + digest, PackageSHA256: digest,
			RecordURL: "https://evidence.example/records/test/", PortableURL: "https://evidence.example/records/test/portable.zip", Access: AccessPublicSafe,
		},
		Scope: ScopeBinding{Posture: ScopeComplete, ProjectRef: "project:test", WorkstreamRef: "workstream:test", WorksetRef: "workset:test", CallGraphRef: "callgraph:test", WorkpointRef: "workpoint:test", WorkItemRef: "work-item:test", ContinuityRef: "continuity:test"},
		State: StateReady, IdempotencyKey: "capture:test", CreatedAt: now, ObservedAt: now,
	}
}
