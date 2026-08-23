package observability

import (
	"testing"
	"time"
)

func TestFreshCountWindow(t *testing.T) {
	s := &ErrorStore{}
	now := time.Now()
	mk := func(minsAgo int) ErrorEvent {
		return ErrorEvent{TS: now.Add(-time.Duration(minsAgo) * time.Minute).UTC().Format(time.RFC3339Nano)}
	}
	s.events = []ErrorEvent{mk(1), mk(5), mk(30), mk(120), {TS: "garbage"}}

	if got := s.FreshCount(15 * time.Minute); got != 2 {
		t.Fatalf("FreshCount(15m) = %d, want 2", got)
	}
	if got := s.FreshCount(time.Hour); got != 3 {
		t.Fatalf("FreshCount(1h) = %d, want 3", got)
	}
	if got := FreshCount(); got != 0 {
		t.Fatalf("global FreshCount on empty default store = %d, want 0", got)
	}
}
