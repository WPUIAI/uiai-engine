package vision

import "testing"

func TestAppendBoundedKeepsTail(t *testing.T) {
	items := []int{}
	for i := 0; i < 10; i++ {
		items = appendBounded(items, i, 3)
	}
	if got, want := len(items), 3; got != want {
		t.Fatalf("len=%d want=%d", got, want)
	}
	want := []int{7, 8, 9}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("items=%v want=%v", items, want)
		}
	}
}

func TestDiagnosticsLimitAndSummary(t *testing.T) {
	s := &Session{ID: "sid", URL: "https://example.test", Title: "Example"}
	s.diagnostics = newDiagnosticsRecorder()
	for i := 0; i < 5; i++ {
		s.diagnostics.console = append(s.diagnostics.console, ConsoleEvent{Seq: uint64(i + 1), Level: "log", Text: "log"})
	}
	s.diagnostics.console = append(s.diagnostics.console,
		ConsoleEvent{Seq: 6, Level: "error", Text: "boom"},
		ConsoleEvent{Seq: 7, Level: "warn", Text: "careful"},
	)
	s.diagnostics.exceptions = append(s.diagnostics.exceptions, ExceptionEvent{Seq: 8, Text: "TypeError"})
	s.diagnostics.network = append(s.diagnostics.network,
		NetworkEvent{Seq: 9, URL: "https://example.test/ok", Status: 200},
		NetworkEvent{Seq: 10, URL: "https://example.test/missing", Status: 404, Failed: true},
		NetworkEvent{Seq: 11, URL: "https://example.test/api", Status: 500, Failed: true},
	)
	s.diagnostics.seq = 11

	snap := s.Diagnostics(1, "error", false)
	if got, want := len(snap.Console), 1; got != want {
		t.Fatalf("console len=%d want=%d", got, want)
	}
	if snap.Console[0].Text != "boom" {
		t.Fatalf("console[0]=%q want boom", snap.Console[0].Text)
	}
	if got, want := len(snap.Network), 1; got != want {
		t.Fatalf("network len=%d want=%d", got, want)
	}
	if got, want := snap.Summary.ConsoleErrors, 1; got != want {
		t.Fatalf("console errors=%d want=%d", got, want)
	}
	if got, want := snap.Summary.ConsoleWarnings, 1; got != want {
		t.Fatalf("console warnings=%d want=%d", got, want)
	}
	if got, want := snap.Summary.Exceptions, 1; got != want {
		t.Fatalf("exceptions=%d want=%d", got, want)
	}
	if got, want := snap.Summary.FailedRequests, 2; got != want {
		t.Fatalf("failed requests=%d want=%d", got, want)
	}
	if got, want := snap.Summary.HTTP4xx, 1; got != want {
		t.Fatalf("http4xx=%d want=%d", got, want)
	}
	if got, want := snap.Summary.HTTP5xx, 1; got != want {
		t.Fatalf("http5xx=%d want=%d", got, want)
	}
}

func TestDiagnosticsFailedOnly(t *testing.T) {
	s := &Session{ID: "sid"}
	s.diagnostics = newDiagnosticsRecorder()
	s.diagnostics.network = append(s.diagnostics.network,
		NetworkEvent{Seq: 1, URL: "https://example.test/ok", Status: 200},
		NetworkEvent{Seq: 2, URL: "https://example.test/fail", Status: 503, Failed: true},
	)
	snap := s.Diagnostics(10, "", true)
	if got, want := len(snap.Network), 1; got != want {
		t.Fatalf("failed-only network len=%d want=%d", got, want)
	}
	if snap.Network[0].URL != "https://example.test/fail" {
		t.Fatalf("failed-only URL=%q", snap.Network[0].URL)
	}
}

func TestClearDiagnosticsResetsRecorder(t *testing.T) {
	s := &Session{ID: "sid", diagnostics: newDiagnosticsRecorder()}
	s.diagnostics.seq = 42
	s.diagnostics.console = append(s.diagnostics.console, ConsoleEvent{Text: "x"})
	s.diagnostics.exceptions = append(s.diagnostics.exceptions, ExceptionEvent{Text: "x"})
	s.diagnostics.network = append(s.diagnostics.network, NetworkEvent{RequestID: "r1"})
	s.diagnostics.requests["r1"] = 0

	s.ClearDiagnostics()
	if s.diagnostics.seq != 0 || len(s.diagnostics.console) != 0 || len(s.diagnostics.exceptions) != 0 || len(s.diagnostics.network) != 0 || len(s.diagnostics.requests) != 0 {
		t.Fatalf("diagnostics not cleared: %+v", s.diagnostics)
	}
}
