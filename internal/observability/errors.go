package observability

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
)

const maxErrorEvents = 500

// ErrorEvent is a bounded, redacted diagnostic record for engine/browser failures.
type ErrorEvent struct {
	ID                  string         `json:"id"`
	TS                  string         `json:"ts"`
	Source              string         `json:"source"`
	Class               string         `json:"class,omitempty"`
	Status              int            `json:"status,omitempty"`
	Method              string         `json:"method,omitempty"`
	Path                string         `json:"path,omitempty"`
	URL                 string         `json:"url,omitempty"`
	Message             string         `json:"message,omitempty"`
	SessionID           string         `json:"session_id,omitempty"`
	SuggestedNextAction string         `json:"suggested_next_action,omitempty"`
	Context             map[string]any `json:"context,omitempty"`
}

type ErrorEnvelope struct {
	Error               string               `json:"error"`
	Message             string               `json:"message"`
	ErrorID             string               `json:"error_id,omitempty"`
	ErrorClass          string               `json:"error_class,omitempty"`
	Status              int                  `json:"status,omitempty"`
	SuggestedNextAction string               `json:"suggested_next_action,omitempty"`
	Retryable           bool                 `json:"retryable,omitempty"`
	Recover             []string             `json:"recover,omitempty"`
	Diagnostics         string               `json:"diagnostics,omitempty"`
	Details             map[string]any       `json:"details,omitempty"`
	Focusa              *ErrorFocusaMetadata `json:"focusa,omitempty"`
}

type ErrorFocusaMetadata struct {
	TargetRef         string   `json:"target_ref"`
	EvidenceRef       string   `json:"evidence_ref"`
	PreferredTool     string   `json:"preferred_tool"`
	Summary           string   `json:"summary"`
	NextTools         []string `json:"next_tools"`
	FocusaScopeStatus string   `json:"focusa_scope_status"`
}

type ErrorStore struct {
	mu     sync.Mutex
	events []ErrorEvent
	seq    uint64
}

var defaultStore = &ErrorStore{}

// DefaultFreshWindow is how far back "fresh" errors count for pressure
// signals (#75): stale noise must never read as current saturation.
const DefaultFreshWindow = 15 * time.Minute

func Record(event ErrorEvent) ErrorEvent { return defaultStore.Record(event) }

// FreshCount counts errors recorded within DefaultFreshWindow.
func FreshCount() int { return defaultStore.FreshCount(DefaultFreshWindow) }
func Recent(limit int, source, class string) []ErrorEvent {
	return defaultStore.Recent(limit, source, class)
}
func Count() int { return defaultStore.Count() }
func Clear()     { defaultStore.Clear() }

// classRecovery maps error classes to agent-actionable recovery semantics (#73).
func classRecovery(class string) (bool, []string) {
	switch class {
	case "selector_not_found":
		return false, []string{"snapshot", "resync_refs"}
	case "url_not_allowed":
		return false, []string{"adjust_target"}
	case "page_unavailable":
		return true, []string{"reopen_session"}
	case "timeout", "navigation_failed", "screenshot_failed", "click_failed", "eval_failed":
		return true, []string{"retry", "diagnostics"}
	default:
		return true, []string{"retry", "diagnostics"}
	}
}

func NewErrorEnvelope(event ErrorEvent, fallbackMessage string, details map[string]any) ErrorEnvelope {
	message := strings.TrimSpace(event.Message)
	if message == "" {
		message = fallbackMessage
	}
	if message == "" {
		message = "UIAI request failed"
	}
	retryable, recover := classRecovery(event.Class)
	return ErrorEnvelope{
		Error:               message,
		Message:             message,
		ErrorID:             event.ID,
		ErrorClass:          event.Class,
		Retryable:           retryable,
		Recover:             recover,
		Status:              event.Status,
		SuggestedNextAction: event.SuggestedNextAction,
		Diagnostics:         "/api/errors?limit=20" + diagnosticsFilter(event),
		Details:             sanitizeContext(details),
		Focusa:              buildErrorFocusaMetadata(event, message),
	}
}

func buildErrorFocusaMetadata(event ErrorEvent, message string) *ErrorFocusaMetadata {
	if strings.TrimSpace(event.ID) == "" {
		return nil
	}
	preferred := "focusa_evidence_capture"
	nextTools := []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"}
	if event.Source == "browser_session" && event.SessionID != "" {
		preferred = "focusa_browser_diagnostics_intake"
		nextTools = []string{"focusa_browser_diagnostics_intake", "focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"}
	}
	targetRef := "engine:error"
	if event.URL != "" {
		targetRef = "browser:" + focusapacket.SanitizeURL(event.URL)
	} else if event.Path != "" {
		targetRef = "endpoint:" + sanitizePath(event.Path)
	}
	summary := fmt.Sprintf("%s error %s status=%d: %s", firstNonEmpty(event.Source, "engine"), firstNonEmpty(event.Class, "unknown"), event.Status, message)
	return &ErrorFocusaMetadata{
		TargetRef:         targetRef,
		EvidenceRef:       "uiai-error:" + event.ID,
		PreferredTool:     preferred,
		Summary:           focusapacket.Truncate(summary, focusapacket.MaxCaptureSummaryChars),
		NextTools:         nextTools,
		FocusaScopeStatus: errorFocusaScopeStatus(event.Context),
	}
}

func errorFocusaScopeStatus(ctx map[string]any) string {
	if len(ctx) == 0 {
		return string(focusapacket.ScopeMissing)
	}
	projectRoot, hasProject := ctx["focusa_project_root"].(string)
	continuity, hasContinuity := ctx["focusa_continuity"].(string)
	workpoint, hasWorkpoint := ctx["focusa_workpoint"].(string)
	if hasProject && projectRoot != "" && hasContinuity && continuity != "" {
		return string(focusapacket.ScopePresent)
	}
	if (hasContinuity && continuity != "") || (hasWorkpoint && workpoint != "") || (hasProject && projectRoot != "") {
		return string(focusapacket.ScopePartial)
	}
	return string(focusapacket.ScopeMissing)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func diagnosticsFilter(event ErrorEvent) string {
	if event.Source != "" {
		return "&source=" + event.Source
	}
	if event.Class != "" {
		return "&class=" + event.Class
	}
	return ""
}

func (s *ErrorStore) Record(event ErrorEvent) ErrorEvent {
	if event.Source == "" {
		event.Source = "engine"
	}
	event.Source = safeToken(event.Source, 48)
	event.Class = safeToken(event.Class, 80)
	event.Method = safeToken(event.Method, 16)
	event.Path = sanitizePath(event.Path)
	event.URL = sanitizeURL(event.URL)
	event.Message = truncate(strings.TrimSpace(event.Message), 500)
	event.SessionID = safeToken(event.SessionID, 80)
	event.SuggestedNextAction = truncate(strings.TrimSpace(event.SuggestedNextAction), 300)
	event.Context = sanitizeContext(event.Context)
	event.TS = time.Now().UTC().Format(time.RFC3339Nano)
	n := atomic.AddUint64(&s.seq, 1)
	event.ID = fmt.Sprintf("uiai-error-%d", n)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	if len(s.events) > maxErrorEvents {
		s.events = s.events[len(s.events)-maxErrorEvents:]
	}
	return event
}

// FreshCount counts events whose TS parses within maxAge of now.
// Unparseable timestamps are treated as stale (never inflate pressure).
func (s *ErrorStore) FreshCount(maxAge time.Duration) int {
	if maxAge <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	n := 0
	for i := len(s.events) - 1; i >= 0; i-- {
		ts, err := time.Parse(time.RFC3339Nano, s.events[i].TS)
		if err != nil {
			continue // unparseable: skip, never inflate
		}
		if !ts.Before(cutoff) {
			n++
		}
	}
	return n
}

func (s *ErrorStore) Recent(limit int, source, class string) []ErrorEvent {
	if limit <= 0 || limit > maxErrorEvents {
		limit = 100
	}
	source = strings.ToLower(strings.TrimSpace(source))
	class = strings.ToLower(strings.TrimSpace(class))
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ErrorEvent, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		e := s.events[i]
		if source != "" && strings.ToLower(e.Source) != source {
			continue
		}
		if class != "" && strings.ToLower(e.Class) != class {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (s *ErrorStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *ErrorStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

func sanitizeContext(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		key := safeToken(k, 64)
		if key == "" || sensitiveKey(key) {
			continue
		}
		switch val := v.(type) {
		case string:
			if strings.HasPrefix(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "url") {
				out[key] = sanitizeURL(val)
			} else {
				out[key] = truncate(val, 300)
			}
		case int, int64, uint64, bool, float64:
			out[key] = val
		case map[string]any:
			out[key] = sanitizeContext(val)
		case []any:
			out[key] = sanitizeList(val)
		default:
			if structured, ok := sanitizeStructured(val); ok {
				out[key] = structured
			} else {
				out[key] = truncate(fmt.Sprint(val), 300)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeList(in []any) []any {
	out := make([]any, 0, min(len(in), 20))
	for i, v := range in {
		if i >= 20 {
			break
		}
		if structured, ok := sanitizeStructured(v); ok {
			out = append(out, structured)
		} else if s, ok := v.(string); ok {
			out = append(out, sanitizeStringValue(s))
		} else {
			out = append(out, v)
		}
	}
	return out
}

func sanitizeStructured(v any) (any, bool) {
	data, err := json.Marshal(v)
	if err != nil || len(data) > 10000 {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, false
	}
	return sanitizeDecoded(decoded), true
}

func sanitizeDecoded(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return sanitizeContext(val)
	case []any:
		return sanitizeList(val)
	case string:
		return sanitizeStringValue(val)
	default:
		return val
	}
}

func sanitizeStringValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(trimmed), "http://") || strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		return sanitizeURL(trimmed)
	}
	return truncate(value, 300)
}

func sensitiveKey(k string) bool {
	k = strings.ToLower(k)
	return strings.Contains(k, "token") || strings.Contains(k, "secret") || strings.Contains(k, "key") || strings.Contains(k, "password") || strings.Contains(k, "authorization") || strings.Contains(k, "cookie")
}

func sanitizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[:i]
	}
	return truncate(path, 240)
}

func sanitizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return truncate(raw, 240)
	}
	u.RawQuery = ""
	u.Fragment = ""
	return truncate(u.String(), 240)
}

func safeToken(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == '/' {
			return r
		}
		return '_'
	}, s)
	return truncate(s, max)
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
