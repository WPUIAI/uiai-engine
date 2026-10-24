package routes

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestActionErrorHTTPPreservesEffectAndCancellationSemantics(t *testing.T) {
	for _, test := range []struct {
		name    string
		cause   error
		effects bool
		class   string
		status  int
	}{
		{"uncertain dispatch", context.DeadlineExceeded, true, "pending_side_effect_unknown", 409},
		{"cancelled before dispatch", context.Canceled, false, "operation_cancelled", 408},
		{"cancelled after dispatch", context.Canceled, true, "operation_cancelled", 408},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := &vision.SessionActionError{Operation: "click", Phase: "dispatch", Cause: test.cause, EffectsPossible: test.effects}
			class := classifySessionError(failure)
			if class != test.class {
				t.Fatalf("error class=%s want=%s", class, test.class)
			}
			w := httptest.NewRecorder()
			writeSessionError(w, 500, class, failure, nil, map[string]any{"action": "click"})
			var response observability.ErrorEnvelope
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != test.status || response.Retryable {
				t.Fatalf("unsafe replay posture: status=%d retryable=%v", w.Code, response.Retryable)
			}
			if response.Details["side_effects_may_have_occurred"] != test.effects || response.Details["retry_requires_reobservation"] != test.effects {
				t.Fatal("HTTP response lost the effect uncertainty flag")
			}
			if test.effects && !strings.Contains(response.SuggestedNextAction, "inspect") && !strings.Contains(response.SuggestedNextAction, "verify") {
				t.Fatal("effect-bearing failure lacks observation guidance")
			}
		})
	}
}
