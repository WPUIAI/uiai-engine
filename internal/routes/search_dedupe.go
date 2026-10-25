package routes

import (
	"net/url"
	"strings"
	"unicode"
)

// searchCitationURL normalizes identity without rewriting the returned citation.
// Only known tracking parameters are ignored; content-selecting queries remain.
func searchCitationURL(raw string, withoutTracking bool) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}
	if withoutTracking {
		for key := range q {
			lower := strings.ToLower(key)
			if strings.HasPrefix(lower, "utm_") || lower == "gclid" || lower == "fbclid" {
				q.Del(key)
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
func similarSearchTitles(a, b string) bool {
	tokens := func(s string) map[string]bool {
		set := map[string]bool{}
		for _, part := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
			set[part] = true
		}
		return set
	}
	left, right := tokens(a), tokens(b)
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	overlap := 0
	for word := range left {
		if right[word] {
			overlap++
		}
	}
	return overlap*5 >= (len(left)+len(right)-overlap)*4 // Jaccard >=0.8.
}
func deduplicateSearchResults(results []searchResult) []searchResult {
	kept := make([]searchResult, 0, len(results))
	groups := map[string][]int{}
	for _, result := range results {
		exact := searchCitationURL(result.URL, false)
		if exact == "" {
			continue
		}
		key := searchCitationURL(result.URL, true)
		duplicate := false
		for _, index := range groups[key] {
			old := kept[index]
			if exact == searchCitationURL(old.URL, false) || similarSearchTitles(result.Title, old.Title) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		groups[key] = append(groups[key], len(kept))
		kept = append(kept, result)
	}
	return kept
}
