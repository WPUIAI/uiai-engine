// Package tests contains parity tests verifying Go engine matches PHP behavior.
//
// Run:   go test ./tests/ -v
package tests

import (
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/comparison"
	"github.com/WPUIAI/uiai-engine/internal/content"
	"github.com/WPUIAI/uiai-engine/internal/design"
	"github.com/WPUIAI/uiai-engine/internal/jsonutil"
	"github.com/WPUIAI/uiai-engine/internal/mimic"
)

var fund = &design.Fundamentals{}

// ═══════════════════════════════════════════════════════════
// DESIGN FUNDAMENTALS PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_ContrastRatio(t *testing.T) {
	ratio := fund.ContrastRatio("#000000", "#FFFFFF")
	if ratio < 20.9 || ratio > 21.1 {
		t.Errorf("Black/white contrast should be ~21.0, got %f", ratio)
	}

	ratio2 := fund.ContrastRatio("#767676", "#FFFFFF")
	if ratio2 < 4.5 || ratio2 > 4.6 {
		t.Errorf("#767676 on white should be ~4.54, got %f", ratio2)
	}
}

func TestParity_BestTextColor(t *testing.T) {
	white := fund.BestTextColor("#1a365d", "#ffffff", "#000000")
	if white != "#ffffff" {
		t.Errorf("Dark bg should return white, got %s", white)
	}
	dark := fund.BestTextColor("#f7fafc", "#ffffff", "#000000")
	if dark != "#000000" {
		t.Errorf("Light bg should return dark, got %s", dark)
	}
}

func TestParity_CTA(t *testing.T) {
	result := fund.ValidateCTA("#ffffff", "#1a365d", "#ffffff", false)
	if !result.Pass {
		t.Errorf("White on dark blue should pass CTA validation: %v", result.Issues)
	}
}

func TestParity_DesignSystem_Smoke(t *testing.T) {
	ds := fund.GenerateDesignSystem(design.Sources{
		IntakeData: map[string]any{
			"business_type": "SaaS",
			"colors":        []any{"#1a365d", "#ed8936"},
		},
	})
	if len(ds.Colors) == 0 {
		t.Error("Design system should have color tokens")
	}
	if ds.Typography == nil {
		t.Error("Design system should have typography")
	}
	if ds.Spacing == nil {
		t.Error("Design system should have spacing tokens")
	}
}

// ═══════════════════════════════════════════════════════════
// JSON REPAIR PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_Repair_CodeFences(t *testing.T) {
	input := "```json\n{\"key\":\"value\"}\n```"
	obj, err := jsonutil.RepairObject(input)
	if err != nil {
		t.Fatalf("RepairObject failed: %v", err)
	}
	if obj["key"] != "value" {
		t.Errorf("Expected key=value, got %v", obj["key"])
	}
}

func TestParity_Repair_TrailingCommas(t *testing.T) {
	obj, err := jsonutil.RepairObject(`{"a":1,"b":2,}`)
	if err != nil {
		t.Fatalf("RepairObject failed: %v", err)
	}
	if obj["a"] != float64(1) {
		t.Errorf("Expected a=1, got %v", obj["a"])
	}
}

func TestParity_Repair_LLMWrapper(t *testing.T) {
	raw := "Here's the analysis:\n```json\n{\"score\": 7, \"issues\": [\"contrast\"]}\n```\nThat's my review."
	obj, err := jsonutil.RepairObject(raw)
	if err != nil {
		t.Fatalf("RepairObject failed: %v", err)
	}
	if obj["score"] != float64(7) {
		t.Errorf("Expected score=7, got %v", obj["score"])
	}
}

// ═══════════════════════════════════════════════════════════
// 5-WAY COMPARISON PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_TokenCompliance(t *testing.T) {
	html := `<div style="color: #1a365d; background: #ed8936">Test</div>`
	tokens := map[string]any{
		"colors": map[string]string{
			"primary":   "#1a365d",
			"secondary": "#ed8936",
		},
	}
	result := comparison.CalculateTokenCompliance(html, tokens)
	if result.ColorsMatch < 50 {
		t.Errorf("Exact color match should score >50%%, got %d", result.ColorsMatch)
	}
}

func TestParity_SectionCompliance(t *testing.T) {
	html := `<section class="hero"><h1>Hello</h1></section><section class="features"><h2>Features</h2></section>`
	refSections := []map[string]any{
		{"name": "hero"},
		{"name": "features"},
		{"name": "testimonials"},
	}
	result := comparison.CalculateSectionCompliance(html, refSections)
	if result.SectionsExpected != 3 {
		t.Errorf("Expected 3 sections, got %d", result.SectionsExpected)
	}
	if result.Matched < 1 {
		t.Errorf("Should match ≥1 section, got %d", result.Matched)
	}
}

func TestParity_ComponentCompliance(t *testing.T) {
	// HTML has 2 buttons, 3 images — reference expects both
	html := `<button>Click</button><button>More</button><img src="a.jpg"><img src="b.jpg"><img src="c.jpg">`
	refComponents := []map[string]any{
		{"type": "button", "count": float64(3)},
		{"type": "image", "count": float64(5)},
	}
	result := comparison.CalculateComponentCompliance(html, refComponents)
	// Should detect both types present
	if result.ComponentsFound < 2 {
		t.Errorf("Should find ≥2 component types, got %d", result.ComponentsFound)
	}

	// Test with a missing type
	html2 := `<img src="a.jpg">`
	refComponents2 := []map[string]any{
		{"type": "button"},
		{"type": "image"},
		{"type": "card"},
	}
	result2 := comparison.CalculateComponentCompliance(html2, refComponents2)
	if result2.ComplianceScore > 60 {
		t.Errorf("1/3 component types should score ≤60%%, got %d", result2.ComplianceScore)
	}
}

// ═══════════════════════════════════════════════════════════
// CONTENT MAPPER PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_PatternAlias(t *testing.T) {
	cases := map[string]string{
		"hero-banner":   "hero",
		"hero-section":  "hero",
		"features-grid": "features",
		"pricing-table": "pricing",
		"cta-block":     "cta",
		"testimonial":   "testimonials",
		"contact-form":  "contact",
		"unknown-thing": "unknown-thing",
	}
	for alias, expected := range cases {
		got := content.ResolvePatternAlias(alias)
		if got != expected {
			t.Errorf("ResolvePatternAlias(%q) = %q, want %q", alias, got, expected)
		}
	}
}

func TestParity_Similarity(t *testing.T) {
	if s := content.Similarity("features", "features-grid"); s < 0.4 {
		t.Errorf("Similar strings should score >0.4, got %f", s)
	}
	if s := content.Similarity("hero", "hero"); s < 0.99 {
		t.Errorf("Exact match should be ~1.0, got %f", s)
	}
}

func TestParity_MapDocs(t *testing.T) {
	docs := []content.Document{
		{ID: "doc-1", Name: "hero", Content: "Welcome to our site"},
		{ID: "doc-2", Name: "features", Content: "Our amazing features"},
	}
	result := content.MapDocsToPatterns(docs, []string{"hero", "features", "cta"}, "home")
	if result["hero"] == nil {
		t.Error("Should map hero doc to hero pattern")
	}
	if result["features"] == nil {
		t.Error("Should map features doc to features pattern")
	}
}

// ═══════════════════════════════════════════════════════════
// MIMIC INTELLIGENCE PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_Archetype(t *testing.T) {
	cases := map[string]string{
		"home": "home", "about": "about", "services": "listing",
		"contact": "detail", "blog": "listing", "pricing": "listing",
		"faq": "detail", "random": "detail",
	}
	for slug, expected := range cases {
		if got := mimic.InferArchetype(slug); got != expected {
			t.Errorf("InferArchetype(%q) = %q, want %q", slug, got, expected)
		}
	}
}

func TestParity_InferPages(t *testing.T) {
	pages := mimic.InferPages([]string{"hero", "features", "pricing", "contact"}, "home")
	if len(pages) < 2 {
		t.Errorf("Should infer ≥2 pages, got %d", len(pages))
	}
	if pages[0].Slug != "home" {
		t.Error("First page should be home")
	}
}

func TestParity_Blueprint(t *testing.T) {
	plan := mimic.Plan{
		Business:      mimic.Business{Name: "Test Co", Type: "SaaS"},
		ReferenceURLs: []string{"https://example.com"},
		Sources:       []string{"reference"},
		Pages:         []mimic.Page{{Title: "Home", Slug: "home", Archetype: "home", Patterns: []string{"hero"}, Source: "reference"}},
		Confidence:    mimic.Confidence{Score: 80, Sources: "reference"},
	}
	bp := mimic.GenerateBlueprint(plan)

	for _, needle := range []string{"# Blueprint", "Test Co", "80%", "SaaS"} {
		if !strings.Contains(bp, needle) {
			t.Errorf("Blueprint should contain %q", needle)
		}
	}
}

// ═══════════════════════════════════════════════════════════
// COMPOSITION / POLISH PARITY
// ═══════════════════════════════════════════════════════════

func TestParity_Hero(t *testing.T) {
	result := fund.ValidateHero(map[string]any{
		"headline":    "Welcome to our site",
		"subheadline": "We build great things",
		"cta_text":    "Get Started",
		"image":       "hero.jpg",
	})
	if len(result.Issues) > 0 {
		t.Errorf("Complete hero: no issues expected, got %d: %v", len(result.Issues), result.Issues)
	}

	result2 := fund.ValidateHero(map[string]any{
		"headline":    "Welcome",
		"subheadline": "We build great things",
	})
	if len(result2.Issues) == 0 {
		t.Error("Hero without CTA should have issues")
	}
}

func TestParity_Navigation(t *testing.T) {
	items := make([]string, 10)
	for i := range items {
		items[i] = "Item"
	}
	result := fund.ValidateNavigation(items)
	if len(result.Issues) == 0 {
		t.Error("10 nav items should trigger Miller's law issue")
	}
}
