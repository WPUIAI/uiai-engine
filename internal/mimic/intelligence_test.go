package mimic

import (
	"strings"
	"testing"
)

func TestInferArchetype(t *testing.T) {
	tests := []struct {
		slug, want string
	}{
		{"home", "home"},
		{"about", "about"},
		{"services", "listing"},
		{"contact", "detail"},
		{"blog", "listing"},
		{"pricing", "listing"},
		{"random-page", "detail"}, // default
	}

	for _, tt := range tests {
		got := InferArchetype(tt.slug)
		if got != tt.want {
			t.Errorf("InferArchetype(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

func TestInferPages(t *testing.T) {
	patterns := []string{"hero", "features", "pricing", "testimonials", "cta", "contact"}
	pages := InferPages(patterns, "home")

	if len(pages) < 2 {
		t.Errorf("Expected ≥2 pages, got %d", len(pages))
	}

	// Home should always be first
	if pages[0].Slug != "home" {
		t.Errorf("First page should be home, got %s", pages[0].Slug)
	}

	// Should infer pricing and contact pages
	slugs := make(map[string]bool)
	for _, p := range pages {
		slugs[p.Slug] = true
	}
	if !slugs["pricing"] {
		t.Error("Should infer pricing page from pricing pattern")
	}
	if !slugs["contact"] {
		t.Error("Should infer contact page from contact pattern")
	}
}

func TestGenerateBlueprint(t *testing.T) {
	plan := Plan{
		Success:       true,
		ReferenceURLs: []string{"https://example.com"},
		Sources:       []string{"reference", "intake"},
		Business:      Business{Name: "Test Corp", Type: "SaaS"},
		Pages: []Page{
			{Title: "Home", Slug: "home", Archetype: "home", Patterns: []string{"hero", "features", "cta"}, Source: "reference"},
			{Title: "Pricing", Slug: "pricing", Archetype: "listing", Patterns: []string{"pricing"}, Source: "inferred"},
		},
		Branding: Branding{
			Colors:      []string{"#1a365d", "#ed8936"},
			ColorSource: "intake",
			Style:       "modern-tech",
		},
		Confidence: Confidence{Score: 85, Sources: "reference, intake"},
	}

	blueprint := GenerateBlueprint(plan)

	if !strings.Contains(blueprint, "# Blueprint") {
		t.Error("Blueprint should have header")
	}
	if !strings.Contains(blueprint, "Test Corp") {
		t.Error("Blueprint should contain business name")
	}
	if !strings.Contains(blueprint, "SaaS") {
		t.Error("Blueprint should contain business type")
	}
	if !strings.Contains(blueprint, "85%") {
		t.Error("Blueprint should contain confidence score")
	}
	if !strings.Contains(blueprint, "hero, features, cta") {
		t.Error("Blueprint should contain patterns")
	}
}

func TestBuildEnrichmentPrompt(t *testing.T) {
	plan := Plan{
		Business: Business{Name: "Test", Type: "agency"},
		Pages:    []Page{{Title: "Home", Slug: "home"}},
	}
	prompt := BuildEnrichmentPrompt(plan)
	if !strings.Contains(prompt, "website architect") {
		t.Error("Enrichment prompt should set architect role")
	}
	if !strings.Contains(prompt, "Test") {
		t.Error("Prompt should contain business data")
	}
}

func TestBuildContentSuggestionsPrompt(t *testing.T) {
	prompt := BuildContentSuggestionsPrompt("home", "SaaS", []string{"hero", "features"})
	if !strings.Contains(prompt, "copywriter") {
		t.Error("Content prompt should set copywriter role")
	}
	if !strings.Contains(prompt, "SaaS") {
		t.Error("Prompt should contain business type")
	}
}

func TestBuildInferencePrompt(t *testing.T) {
	data := map[string]any{
		"business_name": "Acme Corp",
		"industry":      "technology",
	}
	prompt := BuildInferencePrompt(data)
	if !strings.Contains(prompt, "Acme Corp") {
		t.Error("Inference prompt should contain intake data")
	}
}
