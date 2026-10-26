package focusapacket

import (
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"regexp"
	"strings"
	"unicode"
)

var textURL = regexp.MustCompile(`https?://`)

// SanitizeText applies canonical URL and credential redaction before a caller
// truncates, serializes or persists free-form telemetry.
func SanitizeText(raw string) string {
	var out strings.Builder
	out.Grow(len(raw))
	for {
		prefix := textURL.FindStringIndex(raw)
		if prefix == nil {
			out.WriteString(raw)
			break
		}
		out.WriteString(raw[:prefix[0]])
		raw = raw[prefix[0]:]
		end := textURLBoundary(raw)
		out.WriteString(SanitizeURL(raw[:end]))
		raw = raw[end:]
	}
	return secrettext.Text(out.String())
}

// Retain balanced parentheses in URL paths and brackets in IPv6 authorities,
// but leave surrounding Markdown/prose delimiters outside the redacted URL.
// Prefix matching and boundary scanning each advance once: adjacent citations
// cannot cause a greedy match to consume the rest of a paragraph.
func textURLBoundary(raw string) int {
	parens, brackets := 0, 0
	for i, char := range raw {
		if unicode.IsSpace(char) || strings.ContainsRune("\"'<>`", char) {
			return i
		}
		switch char {
		case '(':
			parens++
		case ')':
			if parens == 0 {
				return i
			}
			parens--
		case '[':
			brackets++
		case ']':
			if brackets == 0 {
				return i
			}
			brackets--
		}
	}
	return len(raw)
}
