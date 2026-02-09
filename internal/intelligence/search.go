package intelligence

import (
	"math"
	"regexp"
	"strings"
)

// Search performs weighted keyword search across documents.
// Field weighting: title=4x, summary=2x, keywords=3x, body=1x.
// Boost factor applied per document. ExcludeFromSearch respected.
func Search(docs []Document, query string, limit int) []SearchResult {
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}

	type scored struct {
		doc     Document
		score   float64
		matches []string
		snippet string
	}

	var results []scored

	for _, doc := range docs {
		if doc.ExcludeFromSearch {
			continue
		}

		titleLower := strings.ToLower(doc.Title)
		bodyLower := strings.ToLower(doc.Body)
		summaryLower := strings.ToLower(doc.Summary)
		keywordsLower := strings.ToLower(strings.Join(doc.Keywords, " "))

		var score float64
		var matches []string

		for _, token := range tokens {
			titleHits := float64(countOccurrences(titleLower, token)) * 4
			summaryHits := float64(countOccurrences(summaryLower, token)) * 2
			bodyHits := float64(countOccurrences(bodyLower, token))
			keywordHits := float64(countOccurrences(keywordsLower, token)) * 3

			tokenScore := titleHits + summaryHits + bodyHits + keywordHits
			if tokenScore > 0 {
				matches = append(matches, token)
				score += tokenScore
			}
		}

		if score <= 0 {
			continue
		}

		// Apply per-document boost factor
		boost := doc.Boost
		if boost == 0 {
			boost = 1
		}
		score *= boost

		snippet := buildSnippet(doc.Body, matches, tokens)

		results = append(results, scored{
			doc:     doc,
			score:   score,
			matches: matches,
			snippet: snippet,
		})
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{
			ID:         r.doc.ID,
			RunID:      r.doc.RunID,
			Title:      r.doc.Title,
			Summary:    r.doc.Summary,
			SourceType: r.doc.SourceType,
			SourceURL:  r.doc.SourceURL,
			Score:      math.Round(r.score*100) / 100,
			Matches:    r.matches,
			Snippet:    r.snippet,
			Metadata: SearchResultMeta{
				Category:     r.doc.Category,
				DocumentType: r.doc.DocumentType,
				PageType:     r.doc.PageType,
				Keywords:     r.doc.Keywords,
			},
		}
	}
	return out
}

// tokenize splits a query into lowercase tokens.
func tokenize(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var out []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// countOccurrences counts case-insensitive occurrences of needle in haystack.
func countOccurrences(haystack, needle string) int {
	if needle == "" || len(haystack) < len(needle) {
		return 0
	}
	escaped := regexp.QuoteMeta(needle)
	re, err := regexp.Compile("(?i)" + escaped)
	if err != nil {
		return 0
	}
	return len(re.FindAllStringIndex(haystack, -1))
}

// buildSnippet extracts a ~200-char snippet around the first match.
func buildSnippet(body string, matches []string, tokens []string) string {
	if body == "" {
		return ""
	}

	// Find position of first match
	target := ""
	if len(matches) > 0 {
		target = matches[0]
	} else if len(tokens) > 0 {
		target = tokens[0]
	}

	lower := strings.ToLower(body)
	idx := strings.Index(lower, target)
	if idx < 0 {
		// No match found, return beginning
		if len(body) > 200 {
			return body[:200]
		}
		return body
	}

	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := start + 200
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}
