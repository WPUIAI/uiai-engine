package vision

import (
	"encoding/json"

	"github.com/WPUIAI/uiai-engine/internal/secrettext"
)

// Console arguments and exception message/stack fields share one privacy
// boundary: classify complete JSON roles and aliases before persisting previews.
func redactedDiagnosticPreviews(raw []string, limit int) []string {
	projections := make([]any, 0, len(raw)*3)
	for _, text := range raw {
		prefix, value, suffix := secrettext.DecodeFramedProjection(text)
		projections = append(projections, prefix, value, suffix)
	}
	clean := secrettext.Value(projections).([]any)
	out := make([]string, len(raw))
	for i := range raw {
		value := clean[3*i+1]
		text, ok := value.(string)
		if !ok {
			if encoded, err := json.Marshal(value); err == nil {
				text = string(encoded)
			} else {
				text = secrettext.Marker
			}
		}
		text = clean[3*i].(string) + text + clean[3*i+2].(string)
		out[i] = truncateDiag(sanitizeDiagnosticText(text), limit)
	}
	return out
}
