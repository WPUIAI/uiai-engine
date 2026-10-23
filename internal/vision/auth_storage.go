package vision

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

var ErrInvalidAuthState = errors.New("invalid auth state")

// storageOrigin accepts only HTTP(S) origins, never paths, userinfo, or opaque URLs.
func storageOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

// parseAuthStorage validates every value before selecting only the target origin.
// Different origins are never merged into the current page's storage.
func parseAuthStorage(raw json.RawMessage, targetURL ...string) (map[string]string, string) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, "empty"
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, "unparseable"
	}
	flat := make(map[string]string, len(fields))
	allStrings := true
	for k, v := range fields {
		var text string
		if strings.TrimSpace(string(v)) == "null" || json.Unmarshal(v, &text) != nil {
			allStrings = false
			break
		}
		flat[k] = text
	}
	if allStrings {
		return flat, "flat"
	}
	target := ""
	if len(targetURL) > 0 {
		target = storageOrigin(targetURL[0])
	}
	if target == "" {
		return nil, "unparseable"
	}
	selected := map[string]string{}
	seen := map[string]bool{}
	for origin, rawItems := range fields {
		parsed, err := url.Parse(origin)
		normalized := storageOrigin(origin)
		if err != nil || normalized == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || seen[normalized] {
			return nil, "unparseable"
		}
		seen[normalized] = true
		items, shape := parseAuthStorage(rawItems)
		if shape != "flat" {
			return nil, "unparseable"
		}
		if normalized == target {
			selected = items
		}
	}
	return selected, "origin-keyed"
}
