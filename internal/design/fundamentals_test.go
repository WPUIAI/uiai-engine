package design

import (
	"math"
	"testing"
)

func TestRelativeLuminance(t *testing.T) {
	f := &Fundamentals{}

	tests := []struct {
		hex  string
		want float64
	}{
		{"#000000", 0.0},
		{"#FFFFFF", 1.0},
		{"#ff0000", 0.2126}, // pure red
	}

	for _, tt := range tests {
		got := f.RelativeLuminance(tt.hex)
		if math.Abs(got-tt.want) > 0.01 {
			t.Errorf("RelativeLuminance(%s) = %.4f, want %.4f", tt.hex, got, tt.want)
		}
	}
}

func TestContrastRatio(t *testing.T) {
	f := &Fundamentals{}

	// Black on white should be 21:1
	ratio := f.ContrastRatio("#000000", "#FFFFFF")
	if math.Abs(ratio-21.0) > 0.1 {
		t.Errorf("Black/White contrast = %.1f, want 21.0", ratio)
	}

	// Same color = 1:1
	ratio = f.ContrastRatio("#336699", "#336699")
	if math.Abs(ratio-1.0) > 0.01 {
		t.Errorf("Same color contrast = %.1f, want 1.0", ratio)
	}
}

func TestValidateContrast_Pass(t *testing.T) {
	f := &Fundamentals{}

	result := f.ValidateContrast("#1a1a2e", "#FFFFFF", false)
	if !result.Pass {
		t.Errorf("Dark text on white should pass, got ratio %.2f", result.Ratio)
	}
	if result.Level != "AAA" {
		t.Errorf("Expected AAA, got %s", result.Level)
	}
}

func TestValidateContrast_Fail(t *testing.T) {
	f := &Fundamentals{}

	// Light gray on white — should fail
	result := f.ValidateContrast("#cccccc", "#FFFFFF", false)
	if result.Pass {
		t.Errorf("Light gray on white should fail, got ratio %.2f", result.Ratio)
	}
	if len(result.Issues) == 0 {
		t.Error("Expected issues for failing contrast")
	}
}

func TestValidateContrast_LargeText(t *testing.T) {
	f := &Fundamentals{}

	// Test a color that passes AA-large (3:1) but not AA (4.5:1)
	result := f.ValidateContrast("#767676", "#FFFFFF", true)
	if !result.Pass {
		t.Errorf("Gray on white should pass for large text, got ratio %.2f", result.Ratio)
	}
}

func TestCheckDangerousCombo_LightOnLight(t *testing.T) {
	f := &Fundamentals{}
	result := f.CheckDangerousCombo("#EEEEEE", "#FFFFFF")
	if !result.Blocked {
		t.Error("Light on light should be blocked")
	}
}

func TestCheckDangerousCombo_YellowBg(t *testing.T) {
	f := &Fundamentals{}
	result := f.CheckDangerousCombo("#FFFFFF", "#FFD700")
	if !result.Blocked {
		t.Error("White on yellow should be blocked")
	}
}

func TestValidateCTA_Pass(t *testing.T) {
	f := &Fundamentals{}
	result := f.ValidateCTA("#FFFFFF", "#1a365d", "#FFFFFF", false)
	if !result.Pass {
		t.Errorf("White on dark blue CTA should pass: text=%.1f button=%.1f issues=%v",
			result.TextRatio, result.ButtonRatio, result.Issues)
	}
}

func TestValidateCTA_GhostButton(t *testing.T) {
	f := &Fundamentals{}
	result := f.ValidateCTA("#1a365d", "#FFFFFF", "#FFFFFF", true)
	if result.Pass {
		t.Error("Ghost button should always produce an issue")
	}
}

func TestValidateTypography(t *testing.T) {
	f := &Fundamentals{}

	// Good typography
	result := f.ValidateTypography(map[string]string{
		"text_base":      "1rem",
		"leading_normal": "1.5",
		"measure_ideal":  "65ch",
		"text_3xl":       "2rem",
	})
	if !result.Valid {
		t.Errorf("Good typography should be valid: %v", result.Issues)
	}

	// Bad: small body, cramped leading
	result = f.ValidateTypography(map[string]string{
		"text_base":      "12px",
		"leading_normal": "1.1",
		"measure_ideal":  "90ch",
	})
	if result.Valid {
		t.Error("Bad typography should be invalid")
	}
	if len(result.Issues) < 3 {
		t.Errorf("Expected ≥3 issues, got %d: %v", len(result.Issues), result.Issues)
	}
}

func TestAuditDesignSystem(t *testing.T) {
	f := &Fundamentals{}

	// Good design system
	ds := map[string]any{
		"colors": map[string]any{
			"bg_page":      "#FFFFFF",
			"text_primary": "#1a1a2e",
			"primary":      "#1a365d",
			"secondary":    "#2d3748",
		},
		"typography": map[string]any{
			"text_base":      "1rem",
			"leading_normal": "1.5",
			"measure_ideal":  "65ch",
			"text_3xl":       "2.5rem",
		},
	}

	result := f.AuditDesignSystem(ds)
	if !result.Pass {
		t.Errorf("Good design system should pass: score=%d issues=%v", result.Score, result.Issues)
	}
	if result.Score < 80 {
		t.Errorf("Good design system should score ≥80, got %d", result.Score)
	}
}

func TestAuditDesignSystem_BadColors(t *testing.T) {
	f := &Fundamentals{}

	// Bad: yellow primary, light text on light bg
	ds := map[string]any{
		"colors": map[string]any{
			"background": "#FFFFFF",
			"text":       "#cccccc", // very light text on white
			"primary":    "#FFD700", // yellow
		},
	}

	result := f.AuditDesignSystem(ds)
	if result.Pass {
		t.Error("Bad color system should fail")
	}
	if len(result.Issues) == 0 {
		t.Error("Should have contrast issues")
	}
}

func TestGenerateSafePalette(t *testing.T) {
	f := &Fundamentals{}

	palette := f.GenerateSafePalette("#2563eb") // blue-600
	if !palette.ContrastVerified {
		t.Error("Safe palette should be contrast-verified")
	}

	// Verify CTA text passes on CTA bg
	ratio := f.ContrastRatio(palette.CTAText, palette.CTABg)
	if ratio < 4.5 {
		t.Errorf("CTA text/bg ratio %.1f below 4.5:1", ratio)
	}

	// Verify text passes on background
	ratio = f.ContrastRatio(palette.Text, palette.Background)
	if ratio < 4.5 {
		t.Errorf("Text/bg ratio %.1f below 4.5:1", ratio)
	}
}

func TestBuildShadow(t *testing.T) {
	f := &Fundamentals{}
	colors := map[string]string{"primary": "#1a365d"}

	sm := f.BuildShadow("sm", colors)
	if sm.Layers != 1 {
		t.Errorf("SM shadow should have 1 layer, got %d", sm.Layers)
	}

	md := f.BuildShadow("md", colors)
	if md.Layers != 3 {
		t.Errorf("MD shadow should have 3 layers, got %d", md.Layers)
	}

	lg := f.BuildShadow("lg", colors)
	if lg.Layers != 5 {
		t.Errorf("LG shadow should have 5 layers, got %d", lg.Layers)
	}
}

func TestShadowForElement(t *testing.T) {
	f := &Fundamentals{}

	if f.ShadowForElement("modal") != "lg" {
		t.Error("Modal should get lg shadow")
	}
	if f.ShadowForElement("card") != "sm" {
		t.Error("Card should get sm shadow")
	}
	if f.ShadowForElement("section") != "none" {
		t.Error("Section should get no shadow")
	}
	if f.ShadowForElement("unknown") != "none" {
		t.Error("Unknown should get no shadow")
	}
}

func TestParseFontSize(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"16px", 16},
		{"1rem", 16},
		{"1.125rem", 18},
		{"clamp(1rem, 2vw, 1.5rem)", 16},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseFontSize(tt.input)
		if math.Abs(got-tt.want) > 0.1 {
			t.Errorf("parseFontSize(%q) = %.1f, want %.1f", tt.input, got, tt.want)
		}
	}
}
