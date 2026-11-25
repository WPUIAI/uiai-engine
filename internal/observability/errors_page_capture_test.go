package observability

import (
	"reflect"
	"testing"
)

// #293: page_capture_failed is retryable and points callers at the honest
// recovery path — the same session may still serve reads because the
// navigation completed.
func TestClassRecoveryPageCaptureFailed(t *testing.T) {
	retryable, steps := classRecovery("page_capture_failed")
	if !retryable {
		t.Fatal("page_capture_failed must be retryable")
	}
	want := []string{"read_or_diagnostics_same_session", "reopen_fresh_session_if_repeated", "check_browser_health"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("recovery steps %v, want %v", steps, want)
	}
}
