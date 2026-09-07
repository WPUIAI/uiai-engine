package epwadelivery

import "testing"

func TestProducerInventoryIsCompleteAndFailClosed(t *testing.T) {
	want := map[ProducerID]bool{
		ProducerScreenshot: true, ProducerSessionVisual: true, ProducerInteractive: true,
		ProducerMedia: true, ProducerRuntimeArtifact: true, ProducerShareScreenshot: true, ProducerFPV: true,
		ProducerArtifactCommit: true, ProducerDerivative: true, ProducerDiagnostics: true,
		ProducerSourceSnapshot: true, ProducerDOMSnapshot: true, ProducerCritique: true,
		ProducerVisualComparison: true, ProducerResearch: true, ProducerGeneratedReport: true,
	}
	seen := map[ProducerID]bool{}
	for _, policy := range ProducerInventory() {
		if seen[policy.ID] || !want[policy.ID] {
			t.Fatalf("duplicate or unknown producer: %#v", policy)
		}
		seen[policy.ID] = true
		if policy.DeliverySchema != Schema || policy.RawOnlySuccess || !policy.HTTPSViewerRequired || !policy.PortableZIPRequired || !policy.CompleteScopeRequired {
			t.Fatalf("producer is not fail-closed: %#v", policy)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("producer inventory incomplete: got=%d want=%d", len(seen), len(want))
	}
}

func TestUnknownProducerCannotCreateDelivery(t *testing.T) {
	input := validInput()
	input.Producer = "unregistered.output"
	if _, err := New(input); err == nil {
		t.Fatal("unregistered producer created a delivery contract")
	}
}
