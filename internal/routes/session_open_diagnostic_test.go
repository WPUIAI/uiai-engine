package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// #292: a wedged browser launch must answer with a typed refusal naming the
// cause (browser_launch_failed, 503, suggested action), never a bare 503.
func TestSessionOpenClassifiesBrowserLaunchFailures(t *testing.T) {
	wrapped := errors.Join(vision.ErrLaunchTimeout, errors.New("on-demand launch failed"))
	if class := classifySessionError(wrapped); class != "browser_launch_failed" {
		t.Fatalf("ErrLaunchTimeout classified %q, want browser_launch_failed", class)
	}
	msg := errors.New("on-demand launch failed: browser launch timed out after 35s")
	if class := classifySessionError(msg); class != "browser_launch_failed" {
		t.Fatalf("launch failure classified %q, want browser_launch_failed", class)
	}
	if class := classifySessionError(errors.New("queue full")); class != "session_capacity" {
		t.Fatalf("queue full classified %q, want session_capacity", class)
	}
	if class := classifySessionError(vision.ErrQueueTimeout); class != "session_capacity" {
		t.Fatalf("queue timeout classified %q, want session_capacity", class)
	}
}

func TestWriteSessionErrorLaunchFailureIsTyped503(t *testing.T) {
	rec := httptest.NewRecorder()
	writeSessionError(rec, 500, classifySessionError(vision.ErrLaunchTimeout), vision.ErrLaunchTimeout, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503 for a browser dependency failure", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"browser_launch_failed", "retry", "browser/health", "error_id"} {
		if !strings.Contains(body, want) {
			t.Fatalf("typed launch refusal missing %q: %s", want, body)
		}
	}
}

// #292: a deadline trip must emit one journal event naming the route and
// answer with a machine-readable reason_code, not a bare degraded body.
func TestWithDeadlineRecordsJournalEventAndReasonCode(t *testing.T) {
	deadlineEvents := observability.Recent(50, "http_deadline", "")
	for _, e := range deadlineEvents {
		if e.Path == "/api/test-deadline-probe" {
			t.Fatalf("stale probe event present; test isolation expected clean ledger")
		}
	}
	handler := WithDeadline(30 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(200)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/api/test-deadline-probe", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"reason_code":"route_deadline_exceeded"`) {
		t.Fatalf("deadline body missing reason_code: %s", body)
	}
	if !strings.Contains(body, "browser/health") {
		t.Fatalf("deadline body missing capacity pointer: %s", body)
	}
	events := observability.Recent(10, "http_deadline", "deadline_exceeded")
	found := false
	for _, e := range events {
		if e.Path == "/api/test-deadline-probe" && e.Method == "POST" {
			found = true
			if !strings.Contains(e.Message, "route deadline exceeded") {
				t.Fatalf("journal message does not name the deadline: %q", e.Message)
			}
		}
	}
	if !found {
		t.Fatalf("no deadline_exceeded journal event for the probe route; got %+v", events)
	}
}
