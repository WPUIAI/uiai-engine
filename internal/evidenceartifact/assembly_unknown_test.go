package evidenceartifact

import (
	"errors"
	"testing"
)

func TestUnknownCaptureCannotProveCompleteWindow(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	request.Observations = append(request.Observations, CaptureObservation{ObservationRef: "observation:unknown", Ordinal: 1, OccurredAt: "2026-08-29T12:00:01Z", ActionRef: "action:capture", ReceiptRef: "receipt:unknown"})
	request.Omissions = []CaptureOmission{{ObservationRef: "observation:unknown", Reason: "capture result could not be verified", PolicyRef: "policy:omission", Outcome: "unknown", ProofRef: "receipt:unknown"}}
	if _, err := AssembleCapture(request); !errors.Is(err, ErrCoverageIncomplete) {
		t.Fatalf("complete unknown window: %v", err)
	}
	request.WindowComplete = false
	manifest, err := AssembleCapture(request)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Capture.Omissions[0].Outcome != "unknown" {
		t.Fatal("unknown posture lost")
	}
	request.Omissions[0].ProofRef = ""
	if _, err := AssembleCapture(request); !errors.Is(err, ErrAssemblyInvalid) {
		t.Fatalf("missing proof: %v", err)
	}
	request.Omissions[0].ProofRef = "receipt:unknown"
	request.Omissions[0].Outcome = "invented"
	if _, err := AssembleCapture(request); !errors.Is(err, ErrAssemblyInvalid) {
		t.Fatalf("invalid outcome: %v", err)
	}
}
