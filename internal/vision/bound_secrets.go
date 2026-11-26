package vision

import (
	"strings"
)

// BoundSecretMarker replaces every exact occurrence of a bound injected
// value in textual output channels.
const BoundSecretMarker = "[secret]"

// maxBoundSecrets bounds the per-session injected-value registry so a
// runaway client cannot grow an unbounded list.
const maxBoundSecrets = 32

// boundSecret is one injected value the session promises never to surface
// verbatim (#81): snapshots, reads, eval results, diagnostics values, and
// error messages all replace exact occurrences with the marker. Binding is
// the value-level guarantee pattern redaction cannot give - an injected
// secret that does not look secret-shaped still never leaks.
type boundSecret struct {
	Value string
	Label string
}

// boundSecrets is the per-session registry. Guarded by the session mutex.
type boundSecrets struct {
	values []string
	labels map[string]string
}

func newBoundSecrets() *boundSecrets {
	return &boundSecrets{labels: map[string]string{}}
}

// bind registers an injected value. Empty or whitespace-only values are
// ignored; duplicates re-bind harmlessly.
func (b *boundSecrets) bind(value, label string) {
	if b == nil {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	for _, existing := range b.values {
		if existing == value {
			return
		}
	}
	if len(b.values) >= maxBoundSecrets {
		// Bounded: keep the most recent bindings by dropping the oldest.
		dropped := b.values[0]
		delete(b.labels, dropped)
		b.values = b.values[1:]
	}
	b.values = append(b.values, value)
	b.labels[value] = strings.TrimSpace(label)
}

// scrub replaces exact occurrences of every bound value with the marker.
// The scrub is deliberately exact-match: it never guesses, only honors the
// session's own injection promises.
func (b *boundSecrets) scrub(text string) string {
	if b == nil || len(b.values) == 0 {
		return text
	}
	for _, value := range b.values {
		if strings.Contains(text, value) {
			text = strings.ReplaceAll(text, value, BoundSecretMarker)
		}
	}
	return text
}

// BindSecret registers an injected credential value for this session; every
// textual output channel replaces its exact occurrences with [secret].
func (s *Session) BindSecret(value, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.boundSecrets.bind(value, label)
}

// ScrubBoundSecrets applies the bound-value scrub to arbitrary text on
// session-owned channels.
func (s *Session) ScrubBoundSecrets(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.boundSecrets.scrub(text)
}
