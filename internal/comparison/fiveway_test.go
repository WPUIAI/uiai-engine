package comparison

import "testing"

func TestCalculateTokenCompliance(t *testing.T) {
	html := `<div style="color: #1a1a2e; background-color: #f9fafb; font-family: 'Inter', sans-serif; font-weight: 700; padding: 24px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);"></div>`

	tokens := map[string]any{
		"colors": map[string]any{
			"text_primary":       "#1a1a2e",
			"background_primary": "#f9fafb",
		},
		"typography": map[string]any{
			"heading_font":    "Inter",
			"body_font":       "Inter",
			"heading_weights": []any{"bold"},
		},
		"spacing": map[string]any{
			"section_padding": "medium",
			"component_gap":   "24px",
		},
		"shapes": map[string]any{
			"button_radius": "small 8px",
			"shadow_style":  "subtle",
		},
	}

	result := CalculateTokenCompliance(html, tokens)
	if !result.Success {
		t.Error("Should succeed")
	}
	if result.ColorsMatch < 50 {
		t.Errorf("Colors should match well, got %d", result.ColorsMatch)
	}
	if result.OverallCompliance < 30 {
		t.Errorf("Overall compliance too low: %d", result.OverallCompliance)
	}
}

func TestCalculateSectionCompliance(t *testing.T) {
	html := `
		<header class="site-header">Nav</header>
		<div class="hero-section">Hero</div>
		<div class="features">Features</div>
		<footer>Footer</footer>
	`

	reference := []map[string]any{
		{"name": "header"},
		{"name": "hero"},
		{"name": "features"},
		{"name": "testimonials"},
		{"name": "footer"},
	}

	result := CalculateSectionCompliance(html, reference)
	if !result.Success {
		t.Error("Should succeed")
	}
	if result.Matched < 3 {
		t.Errorf("Should match ≥3 sections, got %d", result.Matched)
	}
	if len(result.Missing) == 0 {
		t.Error("Should have missing sections (testimonials)")
	}
}

func TestCalculateComponentCompliance(t *testing.T) {
	html := `
		<h1>Welcome</h1>
		<h2>Features</h2>
		<img src="hero.jpg" />
		<a class="btn-primary" href="#">Get Started</a>
		<button>Submit</button>
		<div class="card">Feature 1</div>
	`

	reference := []map[string]any{
		{"type": "heading_h1", "section": "hero", "content_hint": "main headline"},
		{"type": "button_primary", "section": "hero", "content_hint": "CTA"},
		{"type": "hero_image", "section": "hero", "content_hint": "background"},
		{"type": "feature_card", "section": "features", "content_hint": "card"},
	}

	result := CalculateComponentCompliance(html, reference)
	if !result.Success {
		t.Error("Should succeed")
	}
	if result.ComplianceScore < 50 {
		t.Errorf("Score too low: %d", result.ComplianceScore)
	}
}

func TestGeneratePriorityFixes(t *testing.T) {
	result := FiveWayResult{
		SectionCompliance: &SectionCompliance{
			Missing:      []string{"testimonials", "cta"},
			OrderCorrect: false,
		},
		TokenCompliance: &TokenCompliance{
			ColorsMatch:     40,
			TypographyMatch: 60,
			SpacingMatch:    50,
		},
		ComponentCompliance: &ComponentCompliance{
			Missing: []MissingComponent{
				{Type: "testimonial_card", Section: "testimonials", ContentHint: "quote"},
			},
		},
	}

	fixes := GeneratePriorityFixes(result)
	if len(fixes.Fixes) == 0 {
		t.Error("Should generate fixes")
	}
	if fixes.TotalIssues < 5 {
		t.Errorf("Expected ≥5 total issues, got %d", fixes.TotalIssues)
	}
	// First fix should be section-related (highest priority)
	if len(fixes.Fixes) > 0 && !contains(fixes.Fixes[0], "section") {
		t.Errorf("First fix should be section-related, got: %s", fixes.Fixes[0])
	}
}

func TestColorDistance(t *testing.T) {
	// Same color = 0
	d := colorDistance("#ff0000", "#ff0000")
	if d != 0 {
		t.Errorf("Same color distance should be 0, got %.1f", d)
	}

	// Very different
	d = colorDistance("#000000", "#ffffff")
	if d < 100 {
		t.Errorf("B/W distance should be large, got %.1f", d)
	}
}

func TestDetectSectionsInHTML(t *testing.T) {
	html := `<header>H</header><div class="hero-main">H</div><footer>F</footer>`
	sections := detectSectionsInHTML(html)
	if len(sections) < 3 {
		t.Errorf("Expected ≥3 sections, got %d: %v", len(sections), sections)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" && len(s) >= len(substr) && findSubstr(s, substr)
}
func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
