package observability

import (
	"encoding/json"
	"testing"
)

func TestNonRetryablePostureIsExplicitOnWire(t *testing.T) {
	for _, class := range []string{"operation_cancelled", "pending_side_effect_unknown", "selector_not_found"} {
		envelope := NewErrorEnvelope(ErrorEvent{Class: class}, "synthetic failure", nil)
		encoded, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(encoded, &wire); err != nil {
			t.Fatal(err)
		}
		retryable, present := wire["retryable"]
		if !present || retryable != false {
			t.Fatalf("%s omitted explicit retryable=false", class)
		}
	}
}
