package vision

import (
	"strings"
	"testing"
)

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

func TestDiagnosticsRedactsURLQueryAndFragment(t *testing.T) {
	if got, want := redactURL("https://example.test/api?token=secret&q=public#frag"), "https://example.test/api"; got != want {
		t.Fatalf("redacted URL=%q want %q", got, want)
	}
}

func TestDiagnosticsSnapshotRedactsSessionURL(t *testing.T) {
	s := &Session{ID: "sid", URL: "https://example.test/app?token=secret&ok=1#frag", diagnostics: newDiagnosticsRecorder()}
	snap := s.Diagnostics(10, "all", false)
	if strings.Contains(snap.URL, "secret") || strings.Contains(snap.URL, "#frag") || strings.Contains(snap.URL, "token=") {
		t.Fatalf("diagnostics URL leaked secret: %q", snap.URL)
	}
}

func TestSanitizeDiagnosticTextRedactsSecrets(t *testing.T) {
	raw := "Authorization Bearer SECRET123 url=https://example.test/a?token=abc&ok=1#frag api_key=BAD cookie=sessionid"
	got := sanitizeDiagnosticText(raw)
	for _, leaked := range []string{"SECRET123", "token=abc", "#frag", "api_key=BAD", "sessionid"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("diagnostic text leaked %q in %q", leaked, got)
		}
	}
	if !strings.Contains(got, "Bearer REDACTED") || !strings.Contains(got, "token=REDACTED") {
		t.Fatalf("diagnostic text not redacted as expected: %q", got)
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

func TestDiagnosticsIncludesFocusaScope(t *testing.T) {
	s := &Session{ID: "sid", diagnostics: newDiagnosticsRecorder(), FocusaScope: &FocusaScope{WorkpointID: "wp1", ContinuityID: "cont1"}}
	diag := s.Diagnostics(10, "all", false)
	if diag.FocusaScope == nil || diag.FocusaScope.WorkpointID != "wp1" || diag.FocusaScope.ContinuityID != "cont1" {
		t.Fatalf("missing focusa scope: %+v", diag.FocusaScope)
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

func TestBuildReadFocusaMetadata(t *testing.T) {
	meta := buildReadFocusaMetadata("sess1", 2, "https://example.com/page?token=secret&ok=1#frag", "Example", "main", 1234, true, "text", &FocusaScope{ProjectRoot: "/repo", ContinuityID: "cont"})
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.TargetRef != "browser:https://example.com/page?ok=1&token=REDACTED" {
		t.Fatalf("target_ref not sanitized: %s", meta.TargetRef)
	}
	if meta.EvidenceRef != "uiai-browser:session=sess1:read:2" {
		t.Fatalf("evidence_ref = %s", meta.EvidenceRef)
	}
	if meta.PreferredTool != "focusa_evidence_capture" || meta.FocusaScopeStatus != "present" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if len(meta.NextTools) == 0 || meta.Summary == "" || !strings.Contains(meta.Summary, "truncated") {
		t.Fatalf("incomplete summary/next tools: %+v", meta)
	}
	if strings.Contains(meta.TargetRef, "secret") || strings.Contains(meta.TargetRef, "#frag") || strings.Contains(meta.Summary, "secret") || strings.Contains(meta.Summary, "#frag") {
		t.Fatalf("secret leaked: %+v", meta)
	}

	md := buildReadFocusaMetadata("sess1", 3, "https://example.com/doc", "Doc", "", 4321, false, "markdown", nil)
	if md == nil || !strings.Contains(md.Summary, "Read Markdown") || md.FocusaScopeStatus != "missing" {
		t.Fatalf("markdown metadata mismatch: %+v", md)
	}
}

func TestBuildDiagnosticsFocusaMetadata(t *testing.T) {
	meta := buildDiagnosticsFocusaMetadata("sess1", 7, "https://example.com/app?jwt=secret#frag", "App", DiagnosticsSummary{ConsoleErrors: 1, Exceptions: 2, FailedRequests: 3, HTTP4xx: 4, HTTP5xx: 5}, &FocusaScope{ContinuityID: "cont"})
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.TargetRef != "browser:https://example.com/app?jwt=REDACTED" {
		t.Fatalf("target_ref not sanitized: %s", meta.TargetRef)
	}
	if meta.EvidenceRef != "uiai-diagnostics:session=sess1:seq=7" {
		t.Fatalf("evidence_ref = %s", meta.EvidenceRef)
	}
	if meta.PreferredTool != "focusa_browser_diagnostics_intake" || meta.FocusaScopeStatus != "partial" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if len(meta.NextTools) == 0 || !strings.Contains(meta.Summary, "failed_requests=3") {
		t.Fatalf("incomplete metadata: %+v", meta)
	}
	if strings.Contains(meta.TargetRef, "secret") || strings.Contains(meta.TargetRef, "#frag") || strings.Contains(meta.Summary, "secret") || strings.Contains(meta.Summary, "#frag") {
		t.Fatalf("secret leaked: %+v", meta)
	}
}

func TestBuildSnapshotFocusaMetadata(t *testing.T) {
	meta := buildSnapshotFocusaMetadata("sess1", 3, "https://example.com/page?token=secret#frag", "Example", "main", SnapshotStats{Lines: 5, RefCount: 2, Interactive: 1}, &FocusaScope{ProjectRoot: "/repo", ContinuityID: "cont"})
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.TargetRef != "browser:https://example.com/page?token=REDACTED" {
		t.Fatalf("target_ref not sanitized: %s", meta.TargetRef)
	}
	if meta.EvidenceRef != "uiai-browser:session=sess1:snapshot:3" {
		t.Fatalf("evidence_ref = %s", meta.EvidenceRef)
	}
	if meta.PreferredTool != "focusa_evidence_capture" || meta.FocusaScopeStatus != "present" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if len(meta.NextTools) == 0 || !strings.Contains(meta.Summary, "Snapshot 2 refs") {
		t.Fatalf("incomplete metadata: %+v", meta)
	}
	if strings.Contains(meta.TargetRef, "secret") || strings.Contains(meta.TargetRef, "#frag") || strings.Contains(meta.Summary, "secret") || strings.Contains(meta.Summary, "#frag") {
		t.Fatalf("secret leaked: %+v", meta)
	}
}

func TestDiagnosticsOptionsCategorySinceAndSummary(t *testing.T) {
	s := &Session{ID: "sid", URL: "https://example.test", Title: "Example"}
	s.diagnostics = newDiagnosticsRecorder()
	s.diagnostics.seq = 6
	s.diagnostics.console = []ConsoleEvent{
		{Seq: 1, Level: "log", Text: "old"},
		{Seq: 2, Level: "error", Text: "boom"},
	}
	s.diagnostics.exceptions = []ExceptionEvent{{Seq: 3, Text: "TypeError"}}
	s.diagnostics.network = []NetworkEvent{
		{Seq: 4, URL: "https://example.test/ok", Status: 200},
		{Seq: 5, URL: "https://example.test/fail", Failed: true, FailureReason: "blocked"},
		{Seq: 6, URL: "https://example.test/bad", Status: 500},
	}

	network := s.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 10, Category: "network", SinceSeq: 4})
	if len(network.Console) != 0 || len(network.Exceptions) != 0 {
		t.Fatalf("network category leaked non-network events: %+v", network)
	}
	if got := len(network.Network); got != 2 {
		t.Fatalf("network events after since_seq: got %d want 2", got)
	}
	if network.Network[0].Seq != 5 || network.Network[1].Seq != 6 {
		t.Fatalf("unexpected network seqs: %+v", network.Network)
	}

	summary := s.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 10, Category: "all", Format: "summary"})
	if summary.Mode != "summary" {
		t.Fatalf("mode=%q want summary", summary.Mode)
	}
	if len(summary.Console) != 0 || len(summary.Exceptions) != 0 || len(summary.Network) != 0 || len(summary.FailedRequests) != 0 {
		t.Fatalf("summary mode should omit event arrays: %+v", summary)
	}
	if summary.Summary.ConsoleErrors != 1 || summary.Summary.Exceptions != 1 || summary.Summary.HTTP5xx != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary.Summary)
	}
}
