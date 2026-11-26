package vision

import (
	"encoding/json"

	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"github.com/go-rod/rod/lib/proto"
)

const (
	maxConsoleEvents   = 200
	maxExceptionEvents = 100
	maxNetworkEvents   = 300
	maxDialogEvents    = 50
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

// DialogEvent records one JavaScript dialog (alert/confirm/prompt/
// beforeunload) the page opened. Unanswered dialogs block the renderer main
// thread, so the session listener auto-dismisses and keeps the event here for
// the agent (#301).
type DialogEvent struct {
	Seq           uint64 `json:"seq"`
	TS            string `json:"ts"`
	Type          string `json:"type"`
	Message       string `json:"message"`
	DefaultPrompt string `json:"default_prompt,omitempty"`
	URL           string `json:"url,omitempty"`
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

// MediaElementInfo is one detected page media element in the diagnostics
// inventory, so an agent knows what video/audio the page carries and whether
// it consumed any of it (#294 diagnose slice).
type MediaElementInfo struct {
	Ref         string  `json:"ref"`
	Tag         string  `json:"tag"`
	HasSource   bool    `json:"has_source"`
	CrossOrigin bool    `json:"cross_origin,omitempty"`
	DurationS   float64 `json:"duration_s,omitempty"`
	ReadyState  int     `json:"ready_state"`
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
	Pacing                *PaceStats                 `json:"pacing,omitempty"`
	SessionID             string                     `json:"session_id"`
	ActorRef              string                     `json:"actor_ref,omitempty"`
	URL                   string                     `json:"url"`
	Title                 string                     `json:"title"`
	Seq                   uint64                     `json:"seq"`
	GeneratedAt           string                     `json:"generated_at"`
	Mode                  string                     `json:"mode,omitempty"`
	Filters               map[string]any             `json:"filters,omitempty"`
	FocusaScope           *FocusaScope               `json:"focusa_scope,omitempty"`
	OriginDiagnostic      map[string]any             `json:"origin_diagnostic,omitempty"`
	Console               []ConsoleEvent             `json:"console"`
	Exceptions            []ExceptionEvent           `json:"exceptions"`
	JavascriptDialogs     []DialogEvent              `json:"javascript_dialogs"`
	MediaElements         []MediaElementInfo         `json:"media_elements,omitempty"`
	WebAuthnHint          string                     `json:"webauthn_hint,omitempty"`
	ProviderRejectionHint string                     `json:"provider_rejection_hint,omitempty"`
	MediaIssues           []string                   `json:"media_issues,omitempty"`
	Network               []NetworkEvent             `json:"network"`
	FailedRequests        []NetworkEvent             `json:"failed_requests"`
	Summary               DiagnosticsSummary         `json:"summary"`
	Focusa                *DiagnosticsFocusaMetadata `json:"focusa,omitempty"`
}

type DiagnosticsOptions struct {
	Limit      int
	Level      string
	FailedOnly bool
	Category   string
	SinceSeq   uint64
	Format     string
}

type DiagnosticsFocusaMetadata struct {
	TargetRef         string   `json:"target_ref"`
	EvidenceRef       string   `json:"evidence_ref"`
	PreferredTool     string   `json:"preferred_tool"`
	Summary           string   `json:"summary"`
	NextTools         []string `json:"next_tools"`
	FocusaScopeStatus string   `json:"focusa_scope_status"`
}

type diagnosticsRecorder struct {
	mu         sync.Mutex
	seq        uint64
	console    []ConsoleEvent
	exceptions []ExceptionEvent
	dialogs    []DialogEvent
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
	_ = proto.PageEnable{}.Call(s.page)
	eventPage, cancel := s.page.WithCancel()
	s.diagnosticsCancel = cancel
	go eventPage.EachEvent(
		func(e *proto.RuntimeConsoleAPICalled) {
			s.recordConsole(e)
		},
		func(e *proto.PageDownloadWillBegin) {
			s.noteDownloadBegin(e)
		},
		func(e *proto.PageDownloadProgress) {
			s.noteDownloadProgress(e)
		},
		func(e *proto.PageJavascriptDialogOpening) {
			// #301: an unanswered JavaScript dialog blocks the renderer main
			// thread; record it for the agent and auto-dismiss so the page
			// stays usable. The event remains visible in diagnostics
			// (javascript_dialogs) with its message and default prompt.
			s.recordDialog(e)
			if derr := (proto.PageHandleJavaScriptDialog{Accept: false, PromptText: ""}).Call(eventPage); derr != nil {
				log.Printf("[vision] dialog auto-dismiss failed (session %s): %v", s.ID, derr)
			}
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
	s.diagnostics.dialogs = nil
	s.diagnostics.network = nil
	s.diagnostics.requests = make(map[string]int)
	s.diagnostics.seq = 0
}

// probePageMedia inventories the page's <video>/<audio> elements (bounded
// to 20) and builds the honest "present but not consumed" hints, so an
// agent knows a page carried media it did not sample or transcribe. A
// failing probe never breaks the diagnostics snapshot.
func (s *Session) probePageMedia() ([]MediaElementInfo, []string) {
	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return nil, nil
	}
	res, err := page.Eval(`() => {
		const els = document.querySelectorAll('video, audio');
		const out = [];
		let vi = 0, ai = 0;
		for (const el of els) {
			if (out.length >= 20) break;
			const idx = el.tagName === 'VIDEO' ? vi++ : ai++;
			let src = '';
			try { src = el.currentSrc || el.src || (el.querySelector('source') || {}).src || ''; } catch (e) {}
			let cross = false;
			try { cross = !!src && new URL(src, location.href).origin !== location.origin; } catch (e) {}
			out.push({
				ref: el.tagName.toLowerCase() + '[' + idx + ']',
				tag: el.tagName.toLowerCase(),
				has_source: !!src,
				cross_origin: cross,
				duration_s: isFinite(el.duration) ? el.duration : 0,
				ready_state: el.readyState || 0,
			});
		}
		return JSON.stringify(out);
	}`)
	if err != nil || res == nil {
		return nil, nil
	}
	var elements []MediaElementInfo
	if jerr := json.Unmarshal([]byte(res.Value.Str()), &elements); jerr != nil {
		return nil, nil
	}
	var issues []string
	for _, el := range elements {
		if !el.HasSource {
			continue
		}
		state := ""
		if el.CrossOrigin {
			state = " (cross-origin source)"
		}
		issues = append(issues, fmt.Sprintf(
			"media present but not sampled/transcribed: %s%s duration=%.1fs ready_state=%d - sample with POST /api/session/{id}/video/frames, transcribe with POST /media/transcript, or combine with POST /media/understand",
			el.Ref, state, el.DurationS, el.ReadyState))
	}
	return elements, issues
}

func (s *Session) Diagnostics(limit int, level string, failedOnly bool) DiagnosticsSnapshot {
	return s.DiagnosticsWithOptions(DiagnosticsOptions{Limit: limit, Level: level, FailedOnly: failedOnly})
}

func (s *Session) DiagnosticsWithOptions(opts DiagnosticsOptions) DiagnosticsSnapshot {
	s.mu.Lock()
	title := s.Title
	pacing := s.pacingStatsLocked()
	s.mu.Unlock()
	limit := opts.Limit
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = "full"
	}
	category := strings.ToLower(strings.TrimSpace(opts.Category))
	if category == "" {
		category = "all"
	}
	if s.diagnostics == nil {
		s.diagnostics = newDiagnosticsRecorder()
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()

	console := filterConsoleSince(s.diagnostics.console, limit, opts.Level, opts.SinceSeq)
	exceptions := tailExceptionsSince(s.diagnostics.exceptions, limit, opts.SinceSeq)
	dialogs := tailDialogsSince(s.diagnostics.dialogs, limit, opts.SinceSeq)
	network := filterNetworkSince(s.diagnostics.network, limit, opts.FailedOnly, opts.SinceSeq)
	failed := filterNetworkSince(s.diagnostics.network, limit, true, opts.SinceSeq)

	switch category {
	case "console":
		exceptions = []ExceptionEvent{}
		network = []NetworkEvent{}
		failed = []NetworkEvent{}
	case "exceptions", "exception":
		console = []ConsoleEvent{}
		network = []NetworkEvent{}
		failed = []NetworkEvent{}
	case "network":
		console = []ConsoleEvent{}
		exceptions = []ExceptionEvent{}
	case "failed", "failed_requests":
		console = []ConsoleEvent{}
		exceptions = []ExceptionEvent{}
		network = []NetworkEvent{}
	case "all":
	default:
		category = "all"
	}

	summary := summarizeDiagnostics(console, exceptions, network)
	if category == "all" && opts.SinceSeq == 0 {
		// Backward compatibility: legacy diagnostics summaries counted the full
		// buffers even when event arrays were limited or level-filtered.
		summary = summarizeDiagnostics(s.diagnostics.console, s.diagnostics.exceptions, s.diagnostics.network)
	}
	if category == "failed" || category == "failed_requests" {
		summary = summarizeDiagnostics(nil, nil, failed)
	}
	seq := s.diagnostics.seq
	if format == "summary" {
		console = []ConsoleEvent{}
		exceptions = []ExceptionEvent{}
		network = []NetworkEvent{}
		failed = []NetworkEvent{}
	} else {
		format = "full"
	}
	// #320: when the page reports failed 4xx traffic, attach a precise origin
	// metadata diagnostic so an agent can distinguish "the engine strips
	// request metadata" from "the page/provider context is the cause". The
	// engine does not rewrite Origin/Referer for engine-launched pages; a
	// null-Origin rejection against a committed page origin points at an
	// opaque (sandboxed iframe / data:) context on the page or the provider's
	// own domain policy — not at engine transport.
	var originDiagnostic map[string]any
	if summary.HTTP4xx > 0 {
		committedOrigin, originErr := s.CurrentOriginContext(context.Background())
		posture := "committed_https"
		if originErr != nil || committedOrigin == "" {
			posture = "unavailable_or_opaque"
		}
		originDiagnostic = map[string]any{
			"page_origin":             committedOrigin,
			"origin_posture":          posture,
			"engine_metadata_posture": "engine sessions preserve page Origin/Referer request metadata; the engine does not rewrite outbound subresource headers",
			"hint":                    "A provider rejection citing a null/missing Origin while page_origin is committed means the request came from an opaque page context (sandboxed iframe, data: document) or the provider's domain allowlist — not engine stripping. Compare an identical side-browser flow before treating the page as broken, and complete third-party login challenges through the approved private handoff lane rather than replaying them.",
		}
	}
	var mediaElements []MediaElementInfo
	var mediaIssues []string
	var webAuthnHint string
	var providerRejectionHint string
	if category == "all" && format == "full" {
		mediaElements, mediaIssues = s.probePageMedia()
		webAuthnHint = FormatWebAuthnHint(s.WebAuthnAttempts())
		if challenge := s.CurrentAuthChallenge(); challenge != nil && challenge.Class == ProviderBrowserRejectedClass {
			providerRejectionHint = FormatProviderRejectionHint()
		}
	}
	return DiagnosticsSnapshot{
		WebAuthnHint:          webAuthnHint,
		ProviderRejectionHint: providerRejectionHint,
		JavascriptDialogs:     dialogs,
		MediaElements:         mediaElements,
		MediaIssues:           mediaIssues,
		Pacing:                pacing,
		SessionID:             s.ID,
		ActorRef:              s.ActorRef,
		URL:                   redactURL(s.URL),
		Title:                 title,
		Seq:                   seq,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
		Mode:                  format,
		Filters: map[string]any{
			"limit":       limit,
			"level":       opts.Level,
			"failed_only": opts.FailedOnly,
			"category":    category,
			"since_seq":   opts.SinceSeq,
		},
		FocusaScope:      s.FocusaScope,
		OriginDiagnostic: originDiagnostic,
		Console:          console,
		Exceptions:       exceptions,
		Network:          network,
		FailedRequests:   failed,
		Summary:          summary,
		Focusa:           buildDiagnosticsFocusaMetadata(s.ID, seq, s.URL, title, summary, s.FocusaScope),
	}
}

func buildDiagnosticsFocusaMetadata(sessionID string, seq uint64, pageURL, title string, summary DiagnosticsSummary, scope *FocusaScope) *DiagnosticsFocusaMetadata {
	targetRef := "browser:" + focusapacket.SanitizeURL(pageURL)
	if pageURL == "" {
		targetRef = "browser:session=" + focusapacket.Truncate(sessionID, 80)
	}
	result := fmt.Sprintf("Diagnostics for %s: console_errors=%d exceptions=%d failed_requests=%d http_4xx=%d http_5xx=%d", focusapacket.Truncate(safePageLabel(title, pageURL, sessionID), 160), summary.ConsoleErrors, summary.Exceptions, summary.FailedRequests, summary.HTTP4xx, summary.HTTP5xx)
	return &DiagnosticsFocusaMetadata{
		TargetRef:         targetRef,
		EvidenceRef:       fmt.Sprintf("uiai-diagnostics:session=%s:seq=%d", focusapacket.Truncate(sessionID, 80), seq),
		PreferredTool:     "focusa_browser_diagnostics_intake",
		Summary:           focusapacket.Truncate(result, focusapacket.MaxCaptureSummaryChars),
		NextTools:         []string{"focusa_browser_diagnostics_intake", "focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		FocusaScopeStatus: readFocusaScopeStatus(scope),
	}
}

func (s *Session) recordConsole(e *proto.RuntimeConsoleAPICalled) {
	if s.diagnostics == nil || e == nil {
		return
	}
	raw := make([]string, 0, len(e.Args))
	for _, arg := range e.Args {
		raw = append(raw, fullRemoteObjectPreview(arg))
	}
	args := redactedDiagnosticPreviews(raw, 500)
	url, line, col := stackTop(e.StackTrace)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	s.diagnostics.console = appendBounded(s.diagnostics.console, ConsoleEvent{
		Seq:         s.diagnostics.seq,
		TS:          time.Now().UTC().Format(time.RFC3339),
		Level:       string(e.Type),
		Text:        sanitizeDiagnosticText(strings.Join(args, " ")),
		ArgsPreview: args,
		URL:         redactURL(url),
		Line:        line,
		Column:      col,
	}, maxConsoleEvents)
}

// truncateDialogText bounds dialog text to max runes without splitting a rune.
func truncateDialogText(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// recordDialog appends one dialog event to the bounded ring.
func (s *Session) recordDialog(e *proto.PageJavascriptDialogOpening) {
	if s.diagnostics == nil {
		return
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	event := DialogEvent{
		Seq:           s.diagnostics.seq,
		TS:            time.Now().UTC().Format(time.RFC3339),
		Type:          string(e.Type),
		Message:       truncateDialogText(e.Message, 400),
		DefaultPrompt: truncateDialogText(e.DefaultPrompt, 200),
		URL:           focusapacket.SanitizeURL(e.URL),
	}
	s.diagnostics.dialogs = append(s.diagnostics.dialogs, event)
	if len(s.diagnostics.dialogs) > maxDialogEvents {
		s.diagnostics.dialogs = s.diagnostics.dialogs[len(s.diagnostics.dialogs)-maxDialogEvents:]
	}
}

func (s *Session) recordException(e *proto.RuntimeExceptionThrown) {
	if s.diagnostics == nil || e == nil || e.ExceptionDetails == nil {
		return
	}
	d := e.ExceptionDetails
	text := d.Text
	textLimit := 1000
	if d.Exception != nil {
		if p := fullRemoteObjectPreview(d.Exception); p != "" {
			text = p
			textLimit = 500
		}
	}
	// Labels may occur beyond the preview budget, while their aliases appear
	// earlier in the message or stack; classify the complete event first.
	clean := redactedDiagnosticPreviews([]string{text, stackPreview(d.StackTrace)}, 1000)
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	s.diagnostics.exceptions = appendBounded(s.diagnostics.exceptions, ExceptionEvent{
		Seq:          s.diagnostics.seq,
		TS:           time.Now().UTC().Format(time.RFC3339),
		Text:         truncateDiag(clean[0], textLimit),
		URL:          redactURL(d.URL),
		Line:         d.LineNumber,
		Column:       d.ColumnNumber,
		StackPreview: clean[1],
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
	return filterConsoleSince(items, limit, level, 0)
}

func filterConsoleSince(items []ConsoleEvent, limit int, level string, sinceSeq uint64) []ConsoleEvent {
	out := make([]ConsoleEvent, 0, minInt(limit, len(items)))
	level = strings.ToLower(level)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		item := items[i]
		if sinceSeq > 0 && item.Seq <= sinceSeq {
			continue
		}
		if level == "" || level == "all" || strings.ToLower(item.Level) == level || (level == "warning" && strings.ToLower(item.Level) == "warn") {
			out = append(out, item)
		}
	}
	reverseConsole(out)
	return out
}

func tailExceptions(items []ExceptionEvent, limit int) []ExceptionEvent {
	return tailExceptionsSince(items, limit, 0)
}

func tailExceptionsSince(items []ExceptionEvent, limit int, sinceSeq uint64) []ExceptionEvent {
	out := make([]ExceptionEvent, 0, minInt(limit, len(items)))
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		if sinceSeq > 0 && items[i].Seq <= sinceSeq {
			continue
		}
		out = append(out, items[i])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func tailDialogsSince(items []DialogEvent, limit int, sinceSeq uint64) []DialogEvent {
	out := make([]DialogEvent, 0, minInt(limit, len(items)))
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		if sinceSeq > 0 && items[i].Seq <= sinceSeq {
			continue
		}
		out = append(out, items[i])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func filterNetwork(items []NetworkEvent, limit int, failedOnly bool) []NetworkEvent {
	return filterNetworkSince(items, limit, failedOnly, 0)
}

func filterNetworkSince(items []NetworkEvent, limit int, failedOnly bool, sinceSeq uint64) []NetworkEvent {
	out := make([]NetworkEvent, 0, minInt(limit, len(items)))
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		if sinceSeq > 0 && items[i].Seq <= sinceSeq {
			continue
		}
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

func fullRemoteObjectPreview(obj *proto.RuntimeRemoteObject) string {
	if obj == nil {
		return ""
	}
	if obj.Value.Raw() != nil {
		if obj.Type == proto.RuntimeRemoteObjectTypeString {
			return obj.Value.Str()
		}
		if encoded, err := obj.Value.MarshalJSON(); err == nil {
			return string(encoded)
		}
		return secrettext.Marker
	}
	if obj.Description != "" {
		return obj.Description
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
	return strings.Join(parts, "\n")
}

func redactURL(raw string) string {
	raw = focusapacket.SanitizeURL(raw)
	if raw == "" {
		return raw
	}
	for _, sep := range []string{"?", "#"} {
		if idx := strings.Index(raw, sep); idx >= 0 {
			raw = raw[:idx]
		}
	}
	return raw
}

func sanitizeBrowserContent(raw string) string {
	if raw == "" {
		return raw
	}
	return focusapacket.SanitizeText(raw)
}

// Diagnostics are previews; full browser reads must retain their own character budget.
func sanitizeDiagnosticText(raw string) string {
	return truncateDiag(sanitizeBrowserContent(raw), 1000)
}

func sanitizeBrowserValue(value any) any {
	return sanitizeBrowserValueText(secrettext.Value(value))
}

func sanitizeBrowserValueText(value any) any {
	switch typed := value.(type) {
	case string:
		return sanitizeBrowserContent(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeBrowserValueText(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = sanitizeBrowserValueText(item)
		}
		return out
	default:
		return value
	}
}

func safePageLabel(title, pageURL, sessionID string) string {
	if strings.TrimSpace(title) != "" {
		return sanitizeDiagnosticText(title)
	}
	if strings.TrimSpace(pageURL) != "" {
		return focusapacket.SanitizeURL(pageURL)
	}
	return sanitizeDiagnosticText(sessionID)
}

func truncateDiag(s string, max int) string {
	if len(s) <= max {
		return s
	}
	end := max
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end] + "…"
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
