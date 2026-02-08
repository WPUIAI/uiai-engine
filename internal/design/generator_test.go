package design

import (
	"strings"
	"testing"
)

func TestGenerateDesignSystem_Defaults(t *testing.T) {
	f := &Fundamentals{}
	ds := f.GenerateDesignSystem(Sources{})

	if ds.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", ds.Version)
	}

	// Should have primary color
	if ds.Colors["primary"] == "" {
		t.Error("Missing primary color")
	}

	// Should have 7 primary shades
	for _, shade := range []string{"ultra_light", "light", "semi_light", "semi_dark", "dark", "ultra_dark", "hover"} {
		key := "primary_" + shade
		if ds.Colors[key] == "" {
			t.Errorf("Missing shade: %s", key)
		}
	}

	// Should have CTA colors (from fundamentals enforcement)
	if ds.Colors["cta_bg"] == "" {
		t.Error("Missing cta_bg")
	}
	if ds.Colors["cta_text"] == "" {
		t.Error("Missing cta_text")
	}

	// CTA text should pass contrast
	ratio := f.ContrastRatio(ds.Colors["cta_text"], ds.Colors["cta_bg"])
	if ratio < 4.5 {
		t.Errorf("CTA contrast %.1f below 4.5:1", ratio)
	}

	// Typography should have font_heading
	if ds.Typography["font_heading"] == nil {
		t.Error("Missing font_heading")
	}

	// Typography should have fluid scale
	if ds.Typography["text_base"] == nil {
		t.Error("Missing text_base scale")
	}
	base, ok := ds.Typography["text_base"].(string)
	if !ok || !strings.HasPrefix(base, "clamp(") {
		t.Errorf("text_base should be clamp(), got %v", ds.Typography["text_base"])
	}

	// Should have spacing
	if len(ds.Spacing) < 10 {
		t.Errorf("Expected ≥10 spacing tokens, got %d", len(ds.Spacing))
	}

	// Should have shadows
	if ds.Shadows["sm"] == "" {
		t.Error("Missing sm shadow")
	}

	// Should have fundamentals audit
	if ds.Fundamentals == nil {
		t.Error("Missing fundamentals audit")
	}
}

func TestGenerateDesignSystem_WithIntake(t *testing.T) {
	f := &Fundamentals{}
	ds := f.GenerateDesignSystem(Sources{
		IntakeData: map[string]any{
			"brand_colors":  "#FF6B35, #004E89",
			"brand_fonts":   "Montserrat, Open Sans",
			"business_type": "saas",
		},
	})

	if ds.Colors["primary"] != "#FF6B35" {
		t.Errorf("Primary should be intake color, got %s", ds.Colors["primary"])
	}
	if ds.Colors["secondary"] != "#004E89" {
		t.Errorf("Secondary should be intake color, got %s", ds.Colors["secondary"])
	}

	heading, _ := ds.Typography["font_heading"].(string)
	if !strings.Contains(heading, "Montserrat") {
		t.Errorf("Heading should be Montserrat, got %s", heading)
	}
}

func TestGenerateDesignSystem_VibeMatching(t *testing.T) {
	f := &Fundamentals{}

	// Agency business type should get bold-agency pairing
	ds := f.GenerateDesignSystem(Sources{
		IntakeData: map[string]any{
			"business_type": "agency",
		},
	})

	heading, _ := ds.Typography["font_heading"].(string)
	if !strings.Contains(heading, "Clash Display") {
		t.Errorf("Agency should get Clash Display heading, got %s", heading)
	}
}

func TestGenerateColorShades(t *testing.T) {
	shades := generateColorShades("#1a365d")
	if len(shades) != 7 {
		t.Errorf("Expected 7 shades, got %d", len(shades))
	}

	// Ultra light should be lighter than base
	ulH, _, ulL := hexToHSL(shades["ultra_light"])
	_, _, baseL := hexToHSL("#1a365d")
	if ulL <= baseL {
		t.Errorf("Ultra light (L=%.0f) should be lighter than base (L=%.0f)", ulL, baseL)
	}
	_ = ulH // suppress unused
}

func TestHSLConversion_Roundtrip(t *testing.T) {
	colors := []string{"#1a365d", "#ff6b35", "#38b2ac", "#000000", "#ffffff"}
	for _, hex := range colors {
		h, s, l := hexToHSL(hex)
		back := hslToHex(h, s, l)
		// Allow small rounding differences
		r1, g1, b1 := hexToRGBInts(hex)
		r2, g2, b2 := hexToRGBInts(back)
		if abs(r1-r2) > 1 || abs(g1-g2) > 1 || abs(b1-b2) > 1 {
			t.Errorf("HSL roundtrip failed: %s → (%.1f, %.1f, %.1f) → %s", hex, h, s, l, back)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestToCSSVars(t *testing.T) {
	f := &Fundamentals{}
	ds := f.GenerateDesignSystem(Sources{})
	css := ds.ToCSSVars()

	if !strings.Contains(css, ":root {") {
		t.Error("CSS should start with :root")
	}
	if !strings.Contains(css, "--brand-primary:") && !strings.Contains(css, "--ref-primary:") {
		t.Error("CSS should contain color variables")
	}
}

func TestDarkMode(t *testing.T) {
	colors := map[string]string{
		"primary": "#1a365d",
		"bg_page": "#f7fafc",
	}
	dark := generateDarkMode(colors)

	if dark["bg_page"] != "#0f172a" {
		t.Errorf("Dark bg_page should be dark, got %s", dark["bg_page"])
	}
	if dark["primary"] != "#1a365d" {
		t.Error("Dark primary should keep brand color")
	}
}
