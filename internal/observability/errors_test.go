package observability

import "testing"

func TestErrorStoreRedactsAndBounds(t *testing.T) {
	store := &ErrorStore{}
	for i := 0; i < maxErrorEvents+5; i++ {
		store.Record(ErrorEvent{
			Source:  "browser session!",
			Class:   "selector:not-found",
			Status:  500,
			Path:    "/api/session?id=secret",
			URL:     "https://example.test/path?token=secret#frag",
			Message: "boom",
			Context: map[string]any{"api_key": "secret", "duration_ms": 12},
		})
	}
	if got := store.Count(); got != maxErrorEvents {
		t.Fatalf("store count=%d want %d", got, maxErrorEvents)
	}
	events := store.Recent(1, "", "")
	if len(events) != 1 {
		t.Fatalf("events len=%d", len(events))
	}
	e := events[0]
	if e.Path != "/api/session" {
		t.Fatalf("path not redacted: %q", e.Path)
	}
	if e.URL != "https://example.test/path" {
		t.Fatalf("url not redacted: %q", e.URL)
	}
	if _, ok := e.Context["api_key"]; ok {
		t.Fatalf("secret context key was retained: %+v", e.Context)
	}
}

func TestErrorStoreFiltersRecent(t *testing.T) {
	store := &ErrorStore{}
	store.Record(ErrorEvent{Source: "http", Class: "client_error"})
	store.Record(ErrorEvent{Source: "browser_session", Class: "selector_not_found"})
	items := store.Recent(10, "browser_session", "selector_not_found")
	if len(items) != 1 || items[0].Source != "browser_session" {
		t.Fatalf("unexpected filter result: %+v", items)
	}
}
