// Package secrettext owns content redaction before browser/error/packet output.
// Fixtures are synthetic; redaction never grants credential-use authority.
package secrettext

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const Marker = "[REDACTED_SECRET]"

type Summary struct {
	Count   int      `json:"count"`
	Classes []string `json:"classes,omitempty"`
}

var (
	privateKey = regexp.MustCompile(`(?s)-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----.*?(?:-----END (?:[A-Z]+ )?PRIVATE KEY-----|$)`)
	neighbor   = regexp.MustCompile(`(?i)((?:api[\s_-]+key|(?:bearer|access|refresh|auth)[\s_-]+token)(?:\s*[:=]\s*|\s+)[\x60"']{0,3}\s*)([A-Za-z0-9._~+/=-]{12,})`)
	// Bare token must be a label, not a suffix of code types like ErrorToken.
	// Credential-role aliases still cover camelCase and prefixed assignments.
	assignment      = regexp.MustCompile(`(?i)((?:api[\s_-]*key|access[\s_-]*token|refresh[\s_-]*token|(?:auth|bearer|session|csrf|id)[\s_-]*token|(?:^|[^\pL\pN])token|secret|password|passwd|cookie|set-cookie)\s*[:=]\s*)([^\s,;&]+)`)
	bearer          = regexp.MustCompile(`(?i)(bearer\s+)([A-Za-z0-9._~+/=-]+)`)
	recoverySection = regexp.MustCompile(`(?is)((?:recovery|backup)[ _-]+codes?\s*:)([\s\S]*)`)
)

func placeholder(value string) bool {
	lower := strings.ToLower(strings.Trim(value, "\"'`<>"))
	for _, prefix := range []string{"your_", "example_", "redacted", "placeholder"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
func opaque(value string) bool {
	if placeholder(value) {
		return false
	}
	upper, small, symbol := false, false, false
	for _, char := range value {
		upper = upper || unicode.IsUpper(char)
		small = small || unicode.IsLower(char)
		symbol = symbol || unicode.IsDigit(char) || strings.ContainsRune("._~+/=-", char)
	}
	return symbol || len(value) >= 32 || (len(value) >= 20 && upper && small)
}
func redact(raw string, collect func(string)) (string, Summary) {
	report := Summary{}
	classes := map[string]bool{}
	mark := func(class, value string) {
		report.Count++
		classes[class] = true
		if collect != nil {
			collect(value)
		}
	}
	clean := privateKey.ReplaceAllStringFunc(raw, func(value string) string {
		mark("private_key", value)
		if collect != nil {
			for _, line := range strings.Split(value, "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "-----") {
					collect(line)
				}
			}
		}
		return Marker
	})
	for _, rule := range []struct {
		pattern *regexp.Regexp
		class   string
	}{
		{neighbor, "semantic_token"}, {assignment, "credential_field"}, {bearer, "bearer_token"}, {recoverySection, "break_glass_material_withheld"},
	} {
		clean = rule.pattern.ReplaceAllStringFunc(clean, func(match string) string {
			parts := rule.pattern.FindStringSubmatch(match)
			value := parts[2]
			if strings.Contains(value, Marker) || placeholder(value) {
				return match
			}
			if rule.pattern == neighbor && !opaque(value) {
				return match
			}
			if rule.pattern == bearer && strings.EqualFold(value, "token") {
				return match
			}
			mark(rule.class, value)
			return parts[1] + Marker
		})
	}
	for class := range classes {
		report.Classes = append(report.Classes, class)
	}
	sort.Strings(report.Classes)
	return clean, report
}
func Redact(raw string) (string, Summary) { return redact(raw, nil) }
func Text(raw string) string              { clean, _ := Redact(raw); return clean }

// Fragments preserves element/argument boundaries while removing values inferred
// from neighboring labels; the transient set never escapes or becomes a cache.
func Fragments(input []string) []string {
	joined := strings.Join(input, "\n")
	if recoverySection.MatchString(joined) {
		out := make([]string, len(input))
		for i, fragment := range input {
			if fragment != "" {
				out[i] = Marker
			}
		}
		return out
	}
	values := map[string]bool{}
	redact(joined, func(value string) {
		if value != "" {
			values[value] = true
			// A known credential can appear escaped in Markdown and raw in
			// another field (or vice versa). These are rendering aliases of an
			// already detected value, not new credential-detection rules.
			plain := markdownAliasUnescape.Replace(value)
			values[plain] = true
			values[markdownAliasEscape.Replace(plain)] = true
		}
	})
	ordered := make([]string, 0, len(values))
	for value := range values {
		ordered = append(ordered, value)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if len(ordered[i]) == len(ordered[j]) {
			return ordered[i] < ordered[j]
		}
		return len(ordered[i]) > len(ordered[j])
	})
	out := make([]string, len(input))
	for i, fragment := range input {
		for _, value := range ordered {
			if len(fragment) >= 12 && strings.HasPrefix(value, fragment) {
				fragment = Marker
				break
			}
			fragment = strings.ReplaceAll(fragment, value, Marker)
		}
		out[i] = Text(fragment)
	}
	return out
}
func SensitiveField(key string) bool {
	key = strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(key))
	for _, part := range []string{"token", "secret", "apikey", "privatekey", "accesskey", "password", "passwd", "authorization", "cookie"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

// Match the Markdown punctuation escaping used by browser text serialization.
// Replacers are immutable; inferred values remain local to Fragments.
var markdownAliasUnescape = strings.NewReplacer(
	`\\`, `\`, `\_`, `_`, `\*`, `*`, "\\`", "`", `\~`, `~`, `\[`, `[`, `\]`, `]`, `\#`, `#`,
)
var markdownAliasEscape = strings.NewReplacer(
	`\`, `\\`, `_`, `\_`, `*`, `\*`, "`", "\\`", `~`, `\~`, `[`, `\[`, `]`, `\]`, `#`, `\#`,
)
