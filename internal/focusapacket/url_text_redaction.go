package focusapacket

import (
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"regexp"
)

var textURL = regexp.MustCompile(`https?://[^\s"'<>]+`)

// SanitizeText applies canonical URL and credential redaction before a caller
// truncates, serializes or persists free-form telemetry.
func SanitizeText(raw string) string {
	return secrettext.Text(textURL.ReplaceAllStringFunc(raw, SanitizeURL))
}
