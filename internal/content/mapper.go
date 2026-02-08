// Package content implements the Content Mapper and Block Recipes system.
//
// Ports of:
//   - class-content-mapper.php (1143 LOC) — document-to-pattern mapping
//   - class-block-recipes.php (1259 LOC) — pattern resolution + available patterns
//   - class-block-intelligence.php (876 LOC) — pattern alias resolution
//
// The Go engine ports the ALGORITHMS (mapping strategies, similarity scoring,
// pattern resolution) but NOT the WordPress block markup templates, which
// remain in the PHP plugin. The engine provides:
//   1. MapDocsToPatterns — 5-strategy content mapping
//   2. FindBestMatch — fuzzy pattern matching with Levenshtein similarity
//   3. ResolvePatternAlias — canonical pattern name resolution
//   4. AvailablePatterns — the pattern library
//   5. FillMissingPatterns — intelligent default content generation
package content

import (
	"strings"
	"unicode/utf8"
)

// Document represents a source content document (from Google Drive, etc.)
type Document struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// PatternContent is the mapped content for a pattern section.
type PatternContent struct {
	Title       string `json:"title,omitempty"`
	Headline    string `json:"headline,omitempty"`
	Subheadline string `json:"subheadline,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	CTAText     string `json:"cta_text,omitempty"`
	CTALink     string `json:"cta_link,omitempty"`
	Strategy    string `json:"strategy,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// ContentMap maps pattern names to their content.
type ContentMap map[string]*PatternContent

// PatternKeywords maps patterns to their search keywords.
var PatternKeywords = map[string][]string{
	"hero":           {"hero", "header", "banner", "landing", "main", "home"},
	"features":       {"feature", "service", "benefit", "capability", "what-we-do"},
	"services-grid":  {"service", "offering", "solution"},
	"testimonials":   {"testimonial", "review", "quote", "client", "customer"},
	"about-preview":  {"about", "story", "who-we-are", "mission"},
	"cta":            {"cta", "call-to-action", "contact", "get-started"},
	"stats":          {"stat", "number", "metric", "impact", "achievement"},
	"pricing":        {"pricing", "plan", "tier", "package"},
	"team-grid":      {"team", "people", "staff", "member"},
	"faq":            {"faq", "question", "answer", "help"},
	"gallery":        {"gallery", "image", "photo", "portfolio", "work"},
	"timeline":       {"timeline", "milestone", "history", "process", "step"},
	"contact":        {"contact", "form", "reach", "email"},
	"content":        {"content", "body", "text", "article"},
	"values":         {"value", "principle", "belief", "standard"},
	"story":          {"story", "journey", "history", "about"},
	"comparison":     {"comparison", "compare", "versus", "vs"},
}

// ═══════════════════════════════════════════════════════════
// MAIN MAPPING ALGORITHM (5 strategies)
// ═══════════════════════════════════════════════════════════

// MapDocsToPatterns maps documents to patterns using 5 strategies.
func MapDocsToPatterns(docs []Document, patterns []string, archetype string) ContentMap {
	contentMap := make(ContentMap)
	usedDocs := make(map[string]bool)

	// Strategy 1: Exact filename match
	for _, pattern := range patterns {
		for _, doc := range docs {
			if usedDocs[doc.ID] {
				continue
			}
			docSlug := normalizeName(doc.Name)
			if docSlug == pattern || docSlug == strings.ReplaceAll(pattern, "-", "_") {
				contentMap[pattern] = parseDocForPattern(doc, pattern)
				contentMap[pattern].Strategy = "exact_match"
				contentMap[pattern].Confidence = 1.0
				usedDocs[doc.ID] = true
				break
			}
		}
	}

	// Strategy 2: Fuzzy keyword match
	for _, pattern := range patterns {
		if contentMap[pattern] != nil {
			continue
		}
		keywords := PatternKeywords[pattern]
		if len(keywords) == 0 {
			keywords = []string{pattern}
		}

		for _, doc := range docs {
			if usedDocs[doc.ID] {
				continue
			}
			docName := strings.ToLower(doc.Name)
			matched := false
			for _, keyword := range keywords {
				if strings.Contains(docName, keyword) {
					contentMap[pattern] = parseDocForPattern(doc, pattern)
					contentMap[pattern].Strategy = "keyword_match"
					contentMap[pattern].Confidence = 0.8
					usedDocs[doc.ID] = true
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
	}

	// Strategy 3: Section markers in docs
	for _, doc := range docs {
		if usedDocs[doc.ID] {
			continue
		}
		sections := extractSectionsFromDoc(doc)
		for sectionType, content := range sections {
			matchedPattern := mapSectionToPattern(sectionType, patterns)
			if matchedPattern != "" && contentMap[matchedPattern] == nil {
				contentMap[matchedPattern] = content
				contentMap[matchedPattern].Strategy = "section_marker"
				contentMap[matchedPattern].Confidence = 0.7
			}
		}
	}

	// Strategy 4: Content analysis (H1 detection)
	for _, doc := range docs {
		if usedDocs[doc.ID] {
			continue
		}
		inferred := inferPatternFromContent(doc, patterns, archetype)
		if inferred != "" && contentMap[inferred] == nil {
			contentMap[inferred] = parseDocForPattern(doc, inferred)
			contentMap[inferred].Strategy = "content_analysis"
			contentMap[inferred].Confidence = 0.6
			usedDocs[doc.ID] = true
		}
	}

	// Strategy 5: Fill missing patterns with defaults
	fillMissingPatterns(contentMap, patterns, archetype)

	return contentMap
}

// ═══════════════════════════════════════════════════════════
// SIMILARITY / MATCHING
// ═══════════════════════════════════════════════════════════

// Similarity calculates Levenshtein-based similarity (0.0 to 1.0).
func Similarity(str1, str2 string) float64 {
	str1 = strings.ToLower(str1)
	str2 = strings.ToLower(str2)

	if str1 == str2 {
		return 1.0
	}

	maxLen := utf8.RuneCountInString(str1)
	if l2 := utf8.RuneCountInString(str2); l2 > maxLen {
		maxLen = l2
	}
	if maxLen == 0 {
		return 1.0
	}

	distance := levenshtein(str1, str2)
	return 1.0 - float64(distance)/float64(maxLen)
}

// FindBestMatch finds the best pattern match for a document name.
func FindBestMatch(docName string, patterns []string, threshold float64) string {
	if threshold == 0 {
		threshold = 0.6
	}

	docSlug := normalizeName(docName)
	bestMatch := ""
	bestScore := 0.0

	for _, pattern := range patterns {
		if docSlug == pattern {
			return pattern
		}

		keywords := PatternKeywords[pattern]
		if len(keywords) == 0 {
			keywords = []string{pattern}
		}

		for _, keyword := range keywords {
			score := Similarity(docSlug, keyword)
			if score > bestScore && score >= threshold {
				bestScore = score
				bestMatch = pattern
			}
		}
	}

	return bestMatch
}

// ═══════════════════════════════════════════════════════════
// PATTERN RESOLUTION
// ═══════════════════════════════════════════════════════════

// PatternAliases maps alternative names to canonical pattern names.
var PatternAliases = map[string]string{
	"header": "hero", "banner": "hero", "landing": "hero", "main-header": "hero",
	"hero-detail": "hero", "hero-about": "hero", "hero-video": "hero",
	"service": "features", "services": "features", "services-grid": "features",
	"benefit": "features", "capabilities": "features", "what-we-do": "features",
	"values": "features",
	"review": "testimonials", "reviews": "testimonials", "quotes": "testimonials",
	"about": "about-preview", "who-we-are": "about-preview", "mission": "about-preview",
	"story": "about-preview",
	"call-to-action": "cta", "get-started": "cta",
	"numbers": "stats", "metrics": "stats", "impact": "stats", "achievements": "stats",
	"plans": "pricing", "packages": "pricing", "tiers": "pricing",
	"people": "team-grid", "staff": "team-grid", "our-team": "team-grid",
	"questions": "faq", "help": "faq",
	"photos": "gallery", "portfolio": "gallery", "work": "gallery",
	"milestones": "timeline", "process": "timeline", "how-it-works": "timeline",
	"steps": "timeline",
	"form": "contact", "reach-us": "contact",
	"compare": "comparison", "versus": "comparison",
	"posts": "posts-grid", "blog": "posts-grid", "articles": "posts-grid",
	"related": "posts-grid", "specs": "faq", "filters": "content",
}

// ResolvePatternAlias returns the canonical pattern name.
func ResolvePatternAlias(pattern string) string {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	pattern = strings.ReplaceAll(pattern, " ", "-")

	if canonical, ok := PatternAliases[pattern]; ok {
		return canonical
	}
	return pattern
}

// AvailablePatterns returns all supported pattern types.
func AvailablePatterns() []string {
	return []string{
		"hero", "hero-detail", "hero-about", "hero-video", "page-header",
		"features", "services-grid", "team-grid", "testimonials", "about-preview",
		"cta", "stats", "content", "gallery", "story", "values", "timeline",
		"faq", "pricing", "contact", "posts-grid", "related", "specs", "filters",
		"comparison",
	}
}

// SectionBlocks returns the recommended block types for a section pattern.
func SectionBlocks(pattern string) []string {
	m := map[string][]string{
		"hero":          {"cover", "group", "heading", "paragraph", "buttons"},
		"features":      {"group", "columns", "column", "heading", "paragraph", "image"},
		"testimonials":  {"group", "columns", "column", "quote", "paragraph"},
		"cta":           {"cover", "group", "heading", "paragraph", "buttons"},
		"stats":         {"group", "columns", "column", "heading", "paragraph"},
		"pricing":       {"group", "columns", "column", "heading", "paragraph", "list", "buttons"},
		"team-grid":     {"group", "columns", "column", "image", "heading", "paragraph"},
		"faq":           {"group", "heading", "details", "paragraph"},
		"gallery":       {"group", "gallery", "image"},
		"contact":       {"group", "heading", "paragraph", "form"},
		"timeline":      {"group", "columns", "column", "heading", "paragraph"},
		"about-preview": {"group", "columns", "column", "heading", "paragraph", "image"},
		"content":       {"group", "heading", "paragraph", "image"},
		"comparison":    {"group", "table", "heading", "paragraph"},
	}
	if blocks, ok := m[pattern]; ok {
		return blocks
	}
	return []string{"group", "heading", "paragraph"}
}

// ═══════════════════════════════════════════════════════════
// INTERNAL HELPERS
// ═══════════════════════════════════════════════════════════

func normalizeName(name string) string {
	name = strings.ToLower(name)
	// Remove file extensions
	for _, ext := range []string{".md", ".txt", ".docx", ".pdf", ".html"} {
		name = strings.TrimSuffix(name, ext)
	}
	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func parseDocForPattern(doc Document, pattern string) *PatternContent {
	content := &PatternContent{
		Content: doc.Content,
	}

	lines := strings.Split(doc.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// First non-empty line as headline
		if content.Headline == "" {
			content.Headline = strings.TrimLeft(line, "# ")
			continue
		}
		// Second non-empty line as description
		if content.Description == "" {
			content.Description = line
			break
		}
	}

	return content
}

func extractSectionsFromDoc(doc Document) map[string]*PatternContent {
	sections := make(map[string]*PatternContent)
	lines := strings.Split(doc.Content, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check for section markers (## Section Name or [SECTION: name])
		if strings.HasPrefix(trimmed, "## ") {
			if currentSection != "" {
				sections[currentSection] = &PatternContent{
					Headline: currentSection,
					Content:  strings.TrimSpace(currentContent.String()),
				}
			}
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	if currentSection != "" {
		sections[currentSection] = &PatternContent{
			Headline: currentSection,
			Content:  strings.TrimSpace(currentContent.String()),
		}
	}

	return sections
}

func mapSectionToPattern(sectionType string, patterns []string) string {
	sectionType = strings.ToLower(sectionType)
	// Direct match
	for _, p := range patterns {
		if p == sectionType {
			return p
		}
	}
	// Keyword match
	for _, p := range patterns {
		keywords := PatternKeywords[p]
		for _, kw := range keywords {
			if strings.Contains(sectionType, kw) {
				return p
			}
		}
	}
	return ""
}

func inferPatternFromContent(doc Document, patterns []string, archetype string) string {
	content := strings.ToLower(doc.Content)

	// Check for pattern-specific indicators
	for _, p := range patterns {
		keywords := PatternKeywords[p]
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				matchCount++
			}
		}
		if matchCount >= 2 {
			return p
		}
	}

	return ""
}

func fillMissingPatterns(contentMap ContentMap, patterns []string, archetype string) {
	// Extract context from existing content
	heroHeadline := ""
	heroSub := ""
	if hero := contentMap["hero"]; hero != nil {
		heroHeadline = hero.Headline
		heroSub = hero.Description
	}

	defaults := map[string]*PatternContent{
		"services-grid": {
			Title: "Our Services", Headline: "What We Offer",
			Description: "Discover our range of services designed to meet your needs.",
			Strategy: "default", Confidence: 0.3,
		},
		"testimonials": {
			Title: "What Our Clients Say", Headline: "Client Testimonials",
			Description: "Hear from those who have experienced our services.",
			Strategy: "default", Confidence: 0.3,
		},
		"about-preview": {
			Title: "About Us", Headline: "Who We Are",
			Description: orDefault(heroSub, "Learn more about our story and mission."),
			Strategy: "default", Confidence: 0.3,
		},
		"stats": {
			Title: "By The Numbers", Headline: "Our Impact",
			Content: "100+ Happy Clients | 50+ Projects Completed | 10+ Years Experience",
			Strategy: "default", Confidence: 0.3,
		},
		"cta": {
			Title: "Ready to Get Started?", Headline: orDefault(heroHeadline, "Let's Work Together"),
			Description: "Contact us today to start your project.",
			CTAText: "Get Started", CTALink: "/contact",
			Strategy: "default", Confidence: 0.3,
		},
	}

	for _, pattern := range patterns {
		if contentMap[pattern] != nil {
			continue
		}
		if def, ok := defaults[pattern]; ok {
			contentMap[pattern] = def
		}
	}
}

// Levenshtein distance
func levenshtein(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	n := len(r1)
	m := len(r2)

	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			d[i][j] = minOf(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
		}
	}

	return d[n][m]
}

func minOf(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func orDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}
