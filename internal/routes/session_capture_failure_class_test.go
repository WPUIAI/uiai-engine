package routes

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// #293: a completed navigation whose capture failed classifies as
// page_capture_failed, never as navigation_failed; genuine navigation
// failures keep their own class.
func TestNavigateCaptureFailureClassification(t *testing.T) {
	underlying := errors.New("{-32603 Internal error }")
	wrapped := fmt.Errorf("%w after 40ms: navigation to %s completed, but the page target failed to capture: %w",
		vision.ErrPageCaptureFailed, "https://docs.openclaw.ai/cli/doctor/sqlite-maintenance", underlying)

	if got := classifySessionError(wrapped); got != "page_capture_failed" {
		t.Fatalf("wrapped capture error classified %q, want page_capture_failed", got)
	}
	if strings.Contains(wrapped.Error(), "navigation failed") {
		t.Fatalf("wrapped error must not read as a navigation failure: %v", wrapped)
	}
	// Regression guard: the honest message mentions navigation; the typed
	// sentinel must outrank the message buckets, and genuine navigation
	// failures must keep their class.
	if got := classifySessionError(errors.New("navigation failed: bad url")); got != "navigation_failed" {
		t.Fatalf("genuine navigation failures keep their class, got %q", got)
	}
	// Unwrapped raw CDP internal errors stay in their historic generic class
	// until a producer wraps them; no silent reclassification of raw errors.
	if got := classifySessionError(underlying); got != "action_failed" {
		t.Fatalf("raw internal errors keep the generic class, got %q", got)
	}
}
