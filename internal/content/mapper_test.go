package content

import (
	"testing"
)

func TestMapDocsToPatterns_ExactMatch(t *testing.T) {
	docs := []Document{
		{ID: "1", Name: "hero.md", Content: "# Welcome\nBuild something amazing"},
		{ID: "2", Name: "features.md", Content: "# Our Features\nWe offer great tools"},
	}
	patterns := []string{"hero", "features", "cta"}

	result := MapDocsToPatterns(docs, patterns, "marketing")

	if result["hero"] == nil {
		t.Error("hero should be mapped")
	}
	if result["hero"].Strategy != "exact_match" {
		t.Errorf("hero strategy should be exact_match, got %s", result["hero"].Strategy)
	}
	if result["features"] == nil {
		t.Error("features should be mapped")
	}
}

func TestMapDocsToPatterns_KeywordMatch(t *testing.T) {
	docs := []Document{
		{ID: "1", Name: "customer-reviews.md", Content: "Great product!\n5 stars!"},
	}
	patterns := []string{"testimonials"}

	result := MapDocsToPatterns(docs, patterns, "business")

	if result["testimonials"] == nil {
		t.Error("testimonials should match via keyword 'review'")
	}
	if result["testimonials"].Strategy != "keyword_match" {
		t.Errorf("Strategy should be keyword_match, got %s", result["testimonials"].Strategy)
	}
}

func TestMapDocsToPatterns_FillDefaults(t *testing.T) {
	docs := []Document{}
	patterns := []string{"hero", "cta", "stats"}

	result := MapDocsToPatterns(docs, patterns, "business")

	// CTA and stats should get defaults
	if result["cta"] == nil {
		t.Error("cta should get default content")
	}
	if result["stats"] == nil {
		t.Error("stats should get default content")
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		s1, s2 string
		min    float64
	}{
		{"hero", "hero", 1.0},
		{"hero", "heros", 0.7},
		{"features", "feature", 0.8},
		{"completely-different", "hero", 0.0},
	}

	for _, tt := range tests {
		score := Similarity(tt.s1, tt.s2)
		if score < tt.min {
			t.Errorf("Similarity(%q, %q) = %.2f, want ≥ %.2f", tt.s1, tt.s2, score, tt.min)
		}
	}
}

func TestFindBestMatch(t *testing.T) {
	patterns := []string{"hero", "features", "testimonials", "cta", "stats"}

	// Exact match
	m := FindBestMatch("hero", patterns, 0.6)
	if m != "hero" {
		t.Errorf("Expected hero, got %s", m)
	}

	// Fuzzy match
	m = FindBestMatch("customer-reviews", patterns, 0.3)
	if m != "testimonials" {
		t.Errorf("Expected testimonials for 'customer-reviews', got %s", m)
	}
}

func TestResolvePatternAlias(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"header", "hero"},
		{"banner", "hero"},
		{"service", "features"},
		{"review", "testimonials"},
		{"hero", "hero"}, // already canonical
		{"unknown-thing", "unknown-thing"},
	}

	for _, tt := range tests {
		got := ResolvePatternAlias(tt.input)
		if got != tt.want {
			t.Errorf("ResolvePatternAlias(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAvailablePatterns(t *testing.T) {
	patterns := AvailablePatterns()
	if len(patterns) < 20 {
		t.Errorf("Expected ≥20 patterns, got %d", len(patterns))
	}
}

func TestSectionBlocks(t *testing.T) {
	blocks := SectionBlocks("hero")
	if len(blocks) == 0 {
		t.Error("hero should have recommended blocks")
	}

	blocks = SectionBlocks("unknown")
	if len(blocks) == 0 {
		t.Error("Unknown pattern should get fallback blocks")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		got := levenshtein(tt.s1, tt.s2)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
		}
	}
}

func TestExtractSectionsFromDoc(t *testing.T) {
	doc := Document{
		ID:   "1",
		Name: "all-content.md",
		Content: `# Main Page

## Features
We have great features.
Feature 1, Feature 2.

## Testimonials
"Great product!" - Customer
"Love it!" - User
`,
	}

	sections := extractSectionsFromDoc(doc)
	if len(sections) < 2 {
		t.Errorf("Expected ≥2 sections, got %d", len(sections))
	}
	if sections["features"] == nil {
		t.Error("Should extract features section")
	}
	if sections["testimonials"] == nil {
		t.Error("Should extract testimonials section")
	}
}
