package design

import "testing"

func TestValidateHero_Good(t *testing.T) {
	f := &Fundamentals{}
	result := f.ValidateHero(map[string]any{
		"headline":     "Build Better Websites",
		"subheadline":  "AI-powered design assistance",
		"cta_text":     "Get Started",
		"background":   "#1a365d",
		"height_ratio": 0.5,
	})
	if !result.Pass {
		t.Errorf("Good hero should pass: %v", result.Issues)
	}
}

func TestValidateHero_Missing(t *testing.T) {
	f := &Fundamentals{}
	result := f.ValidateHero(map[string]any{})
	if result.Pass {
		t.Error("Empty hero should fail")
	}
	if len(result.Issues) < 2 {
		t.Errorf("Expected ≥2 issues, got %d: %v", len(result.Issues), result.Issues)
	}
}

func TestValidatePageRhythm_ConsecutiveBg(t *testing.T) {
	f := &Fundamentals{}
	sections := []map[string]any{
		{"type": "hero", "background": "blue"},
		{"type": "features", "background": "white"},
		{"type": "about", "background": "white"},
		{"type": "stats", "background": "white"},
		{"type": "cta", "background": "blue"},
	}
	result := f.ValidatePageRhythm(sections)
	if result.Pass {
		t.Error("3 consecutive white sections should produce issues")
	}
}

func TestAuditComposition(t *testing.T) {
	f := &Fundamentals{}
	sections := []map[string]any{
		{"type": "hero", "headline": "Hello", "cta_text": "Go", "background": "#333"},
		{"type": "features", "background": "white"},
		{"type": "about", "background": "#f5f5f5"},
		{"type": "stats", "background": "white"},
		{"type": "cta", "cta_text": "Contact Us", "background": "#333"},
	}
	result := f.AuditComposition(sections, "home")
	if result.Score < 50 {
		t.Errorf("Good composition should score ≥50, got %d", result.Score)
	}
}

func TestCompositionPromptRules(t *testing.T) {
	f := &Fundamentals{}
	rules := f.CompositionPromptRules()
	if rules == "" {
		t.Error("Prompt rules should not be empty")
	}
}
