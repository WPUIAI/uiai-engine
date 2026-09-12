package evidenceartifact

import (
	"errors"
	"testing"
)

func TestPermissionDeniedCaptureCannotProveCompleteWindow(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	request.Observations = append(request.Observations, CaptureObservation{ObservationRef: "observation:denied", Ordinal: 1, OccurredAt: "2026-08-29T12:00:01Z", ActionRef: "action:capture", ReceiptRef: "receipt:denied"})
	request.Omissions = []CaptureOmission{{ObservationRef: "observation:denied", Reason: "capture permission denied", PolicyRef: "policy:omission", Outcome: "denied", ProofRef: "receipt:denied"}}
	if _, err := AssembleCapture(request); !errors.Is(err, ErrCoverageIncomplete) {
		t.Fatalf("permission-denied capture must not prove a complete window: %v", err)
	}
	request.WindowComplete = false
	manifest, err := AssembleCapture(request)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Capture.Omissions[0].Outcome != "denied" {
		t.Fatal("denied posture lost")
	}
	request.Omissions[0].ProofRef = ""
	if _, err := AssembleCapture(request); !errors.Is(err, ErrAssemblyInvalid) {
		t.Fatalf("permission denial without proof ref: %v", err)
	}
}
