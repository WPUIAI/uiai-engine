package observability

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

type ErrorStore struct {
	mu     sync.Mutex
	events []ErrorEvent
	seq    uint64
}

var defaultStore = &ErrorStore{}

func Record(event ErrorEvent) ErrorEvent { return defaultStore.Record(event) }
func Recent(limit int, source, class string) []ErrorEvent {
	return defaultStore.Recent(limit, source, class)
}
func Count() int { return defaultStore.Count() }
func Clear()     { defaultStore.Clear() }

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
		default:
			out[key] = truncate(fmt.Sprint(val), 300)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
