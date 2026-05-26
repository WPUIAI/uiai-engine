package vision

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

const (
	maxConsoleEvents   = 200
	maxExceptionEvents = 100
	maxNetworkEvents   = 300
)

type ConsoleEvent struct {
	Seq         uint64   `json:"seq"`
	TS          string   `json:"ts"`
	Level       string   `json:"level"`
	Text        string   `json:"text"`
	ArgsPreview []string `json:"args_preview,omitempty"`
	URL         string   `json:"url,omitempty"`
	Line        int      `json:"line,omitempty"`
	Column      int      `json:"column,omitempty"`
}

type ExceptionEvent struct {
	Seq          uint64 `json:"seq"`
	TS           string `json:"ts"`
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	Line         int    `json:"line,omitempty"`
	Column       int    `json:"column,omitempty"`
	StackPreview string `json:"stack_preview,omitempty"`
}

type NetworkEvent struct {
	Seq           uint64 `json:"seq"`
	RequestID     string `json:"request_id"`
	Method        string `json:"method,omitempty"`
	URL           string `json:"url"`
	ResourceType  string `json:"resource_type,omitempty"`
	Status        int    `json:"status,omitempty"`
	MIMEType      string `json:"mime_type,omitempty"`
	Failed        bool   `json:"failed"`
	FailureReason string `json:"failure_reason,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

type DiagnosticsSummary struct {
	ConsoleErrors   int `json:"console_errors"`
	ConsoleWarnings int `json:"console_warnings"`
	Exceptions      int `json:"exceptions"`
	Requests        int `json:"requests"`
	FailedRequests  int `json:"failed_requests"`
	HTTP4xx         int `json:"http_4xx"`
	HTTP5xx         int `json:"http_5xx"`
}

type DiagnosticsSnapshot struct {
	SessionID      string             `json:"session_id"`
	URL            string             `json:"url"`
	Title          string             `json:"title"`
	Seq            uint64             `json:"seq"`
	GeneratedAt    string             `json:"generated_at"`
	Console        []ConsoleEvent     `json:"console"`
	Exceptions     []ExceptionEvent   `json:"exceptions"`
	Network        []NetworkEvent     `json:"network"`
	FailedRequests []NetworkEvent     `json:"failed_requests"`
	Summary        DiagnosticsSummary `json:"summary"`
}

type diagnosticsRecorder struct {
	mu         sync.Mutex
	seq        uint64
	console    []ConsoleEvent
	exceptions []ExceptionEvent
	network    []NetworkEvent
	requests   map[string]int
}

func newDiagnosticsRecorder() *diagnosticsRecorder {
	return &diagnosticsRecorder{requests: make(map[string]int)}
}

func (s *Session) initDiagnostics() {
	if s.diagnostics == nil {
		s.diagnostics = newDiagnosticsRecorder()
	}
	if s.page == nil {
		return
	}
	_ = proto.RuntimeEnable{}.Call(s.page)
	_ = proto.NetworkEnable{}.Call(s.page)
	eventPage, cancel := s.page.WithCancel()
	s.diagnosticsCancel = cancel
	go eventPage.EachEvent(
		func(e *proto.RuntimeConsoleAPICalled) {
			s.recordConsole(e)
		},
		func(e *proto.RuntimeExceptionThrown) {
			s.recordException(e)
		},
		func(e *proto.NetworkRequestWillBeSent) {
			s.recordRequest(e)
		},
		func(e *proto.NetworkResponseReceived) {
			s.recordResponse(e)
		},
		func(e *proto.NetworkLoadingFailed) {
			s.recordLoadingFailed(e)
		},
	)()
}

func (s *Session) ClearDiagnostics() {
	if s.diagnostics == nil {
		s.diagnostics = newDiagnosticsRecorder()
		return
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.console = nil
	s.diagnostics.exceptions = nil
	s.diagnostics.network = nil
	s.diagnostics.requests = make(map[string]int)
	s.diagnostics.seq = 0
}

func (s *Session) Diagnostics(limit int, level string, failedOnly bool) DiagnosticsSnapshot {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	if s.diagnostics == nil {
		s.diagnostics = newDiagnosticsRecorder()
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()

	console := filterConsole(s.diagnostics.console, limit, level)
	exceptions := tailExceptions(s.diagnostics.exceptions, limit)
	network := filterNetwork(s.diagnostics.network, limit, failedOnly)
	failed := filterNetwork(s.diagnostics.network, limit, true)

	return DiagnosticsSnapshot{
		SessionID:      s.ID,
		URL:            s.URL,
		Title:          s.Title,
		Seq:            s.diagnostics.seq,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		Console:        console,
		Exceptions:     exceptions,
		Network:        network,
		FailedRequests: failed,
		Summary:        summarizeDiagnostics(s.diagnostics.console, s.diagnostics.exceptions, s.diagnostics.network),
	}
}

func (s *Session) recordConsole(e *proto.RuntimeConsoleAPICalled) {
	if s.diagnostics == nil || e == nil {
		return
	}
	args := make([]string, 0, len(e.Args))
	for _, arg := range e.Args {
		args = append(args, remoteObjectPreview(arg))
	}
	url, line, col := stackTop(e.StackTrace)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	s.diagnostics.console = appendBounded(s.diagnostics.console, ConsoleEvent{
		Seq:         s.diagnostics.seq,
		TS:          time.Now().UTC().Format(time.RFC3339),
		Level:       string(e.Type),
		Text:        strings.Join(args, " "),
		ArgsPreview: args,
		URL:         url,
		Line:        line,
		Column:      col,
	}, maxConsoleEvents)
}

func (s *Session) recordException(e *proto.RuntimeExceptionThrown) {
	if s.diagnostics == nil || e == nil || e.ExceptionDetails == nil {
		return
	}
	d := e.ExceptionDetails
	text := d.Text
	if d.Exception != nil {
		if p := remoteObjectPreview(d.Exception); p != "" {
			text = p
		}
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	s.diagnostics.exceptions = appendBounded(s.diagnostics.exceptions, ExceptionEvent{
		Seq:          s.diagnostics.seq,
		TS:           time.Now().UTC().Format(time.RFC3339),
		Text:         text,
		URL:          d.URL,
		Line:         d.LineNumber,
		Column:       d.ColumnNumber,
		StackPreview: stackPreview(d.StackTrace),
	}, maxExceptionEvents)
}

func (s *Session) recordRequest(e *proto.NetworkRequestWillBeSent) {
	if s.diagnostics == nil || e == nil || e.Request == nil {
		return
	}
	rid := string(e.RequestID)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	event := NetworkEvent{
		Seq:          s.diagnostics.seq,
		RequestID:    rid,
		Method:       e.Request.Method,
		URL:          redactURL(e.Request.URL),
		ResourceType: string(e.Type),
		StartedAt:    time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	s.diagnostics.network = appendBounded(s.diagnostics.network, event, maxNetworkEvents)
	s.rebuildRequestIndexLocked()
}

func (s *Session) recordResponse(e *proto.NetworkResponseReceived) {
	if s.diagnostics == nil || e == nil || e.Response == nil {
		return
	}
	rid := string(e.RequestID)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	idx, ok := s.diagnostics.requests[rid]
	if !ok {
		s.diagnostics.seq++
		s.diagnostics.network = appendBounded(s.diagnostics.network, NetworkEvent{Seq: s.diagnostics.seq, RequestID: rid}, maxNetworkEvents)
		s.rebuildRequestIndexLocked()
		idx = s.diagnostics.requests[rid]
	}
	n := &s.diagnostics.network[idx]
	n.URL = redactURL(e.Response.URL)
	n.ResourceType = string(e.Type)
	n.Status = e.Response.Status
	n.MIMEType = e.Response.MIMEType
	n.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if e.Response.Status >= 400 {
		n.Failed = true
		n.FailureReason = fmt.Sprintf("HTTP %d", e.Response.Status)
	}
}

func (s *Session) recordLoadingFailed(e *proto.NetworkLoadingFailed) {
	if s.diagnostics == nil || e == nil {
		return
	}
	rid := string(e.RequestID)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	idx, ok := s.diagnostics.requests[rid]
	if !ok {
		s.diagnostics.seq++
		s.diagnostics.network = appendBounded(s.diagnostics.network, NetworkEvent{Seq: s.diagnostics.seq, RequestID: rid}, maxNetworkEvents)
		s.rebuildRequestIndexLocked()
		idx = s.diagnostics.requests[rid]
	}
	n := &s.diagnostics.network[idx]
	n.ResourceType = string(e.Type)
	n.Failed = true
	n.FailureReason = e.ErrorText
	n.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

func (s *Session) rebuildRequestIndexLocked() {
	s.diagnostics.requests = make(map[string]int)
	for i, n := range s.diagnostics.network {
		if n.RequestID != "" {
			s.diagnostics.requests[n.RequestID] = i
		}
	}
}

func appendBounded[T any](items []T, item T, max int) []T {
	items = append(items, item)
	if len(items) > max {
		copy(items, items[len(items)-max:])
		items = items[:max]
	}
	return items
}

func filterConsole(items []ConsoleEvent, limit int, level string) []ConsoleEvent {
	out := make([]ConsoleEvent, 0, minInt(limit, len(items)))
	level = strings.ToLower(level)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		item := items[i]
		if level == "" || level == "all" || strings.ToLower(item.Level) == level || (level == "warning" && strings.ToLower(item.Level) == "warn") {
			out = append(out, item)
		}
	}
	reverseConsole(out)
	return out
}

func tailExceptions(items []ExceptionEvent, limit int) []ExceptionEvent {
	if len(items) <= limit {
		return append([]ExceptionEvent{}, items...)
	}
	return append([]ExceptionEvent{}, items[len(items)-limit:]...)
}

func filterNetwork(items []NetworkEvent, limit int, failedOnly bool) []NetworkEvent {
	out := make([]NetworkEvent, 0, minInt(limit, len(items)))
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		if !failedOnly || items[i].Failed {
			out = append(out, items[i])
		}
	}
	reverseNetwork(out)
	return out
}

func summarizeDiagnostics(console []ConsoleEvent, exceptions []ExceptionEvent, network []NetworkEvent) DiagnosticsSummary {
	var s DiagnosticsSummary
	s.Exceptions = len(exceptions)
	s.Requests = len(network)
	for _, c := range console {
		switch strings.ToLower(c.Level) {
		case "error", "assert":
			s.ConsoleErrors++
		case "warning", "warn":
			s.ConsoleWarnings++
		}
	}
	for _, n := range network {
		if n.Failed {
			s.FailedRequests++
		}
		if n.Status >= 400 && n.Status < 500 {
			s.HTTP4xx++
		}
		if n.Status >= 500 {
			s.HTTP5xx++
		}
	}
	return s
}

func remoteObjectPreview(obj *proto.RuntimeRemoteObject) string {
	if obj == nil {
		return ""
	}
	if obj.Description != "" {
		return truncateDiag(obj.Description, 500)
	}
	if obj.Value.Raw() != nil {
		return truncateDiag(obj.Value.String(), 500)
	}
	if obj.UnserializableValue != "" {
		return string(obj.UnserializableValue)
	}
	return string(obj.Type)
}

func stackTop(st *proto.RuntimeStackTrace) (string, int, int) {
	if st == nil || len(st.CallFrames) == 0 {
		return "", 0, 0
	}
	f := st.CallFrames[0]
	return f.URL, f.LineNumber, f.ColumnNumber
}

func stackPreview(st *proto.RuntimeStackTrace) string {
	if st == nil {
		return ""
	}
	parts := make([]string, 0, minInt(5, len(st.CallFrames)))
	for i, f := range st.CallFrames {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %s:%d:%d", f.FunctionName, f.URL, f.LineNumber, f.ColumnNumber))
	}
	return truncateDiag(strings.Join(parts, "\n"), 1000)
}

func redactURL(raw string) string {
	if raw == "" {
		return raw
	}
	redacted := raw
	for _, key := range []string{"token=", "key=", "secret=", "password=", "auth=", "apikey=", "api_key="} {
		idx := strings.Index(strings.ToLower(redacted), key)
		if idx >= 0 {
			start := idx + len(key)
			end := start
			for end < len(redacted) && redacted[end] != '&' && redacted[end] != '#' {
				end++
			}
			redacted = redacted[:start] + "REDACTED" + redacted[end:]
		}
	}
	return redacted
}

func truncateDiag(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func reverseConsole(items []ConsoleEvent) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func reverseNetwork(items []NetworkEvent) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
