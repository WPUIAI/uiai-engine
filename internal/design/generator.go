// generator.go — Port of class-design-system-generator.php (1865 LOC)
// Generates a complete design system from multiple sources: intake data,
// reference analysis, AI suggestions, and logo analysis.
//
// Core capabilities:
//   - Color generation with 7-shade system (ACSS-inspired HSL interpolation)
//   - Font pairing library (20+ curated pairings, vibe-matched to business type)
//   - Fluid type scale using clamp() (Typetura-inspired intrinsic typography)
//   - Spacing, shadows, radii, transitions, breakpoints, z-index tokens
//   - Dark mode generation
//   - Design Fundamentals enforcement (WCAG auto-fix)
//   - CSS custom property output
package design

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// DesignSystem is the complete generated token set.
type DesignSystem struct {
	Version         string              `json:"version"`
	GeneratedAt     string              `json:"generated_at"`
	Prefix          string              `json:"prefix"`
	Colors          map[string]string   `json:"colors"`
	ColorPartials   map[string]any      `json:"color_partials"`
	Typography      map[string]any      `json:"typography"`
	Spacing         map[string]string   `json:"spacing"`
	SpacingTShirt   map[string]string   `json:"spacing_tshirt"`
	SpacingContext  map[string]string   `json:"spacing_contextual"`
	Shadows         map[string]string   `json:"shadows"`
	Radii           map[string]string   `json:"radii"`
	Transitions     map[string]string   `json:"transitions"`
	Breakpoints     map[string]string   `json:"breakpoints"`
	ZIndex          map[string]int      `json:"z_index"`
	Cards           map[string]string   `json:"cards"`
	DarkMode        map[string]string   `json:"dark_mode"`
	Fundamentals    *FundamentalsAudit  `json:"_fundamentals,omitempty"`
}

// FundamentalsAudit records what was auto-fixed during generation.
type FundamentalsAudit struct {
	ValidatedAt string   `json:"validated_at"`
	Fixes       []string `json:"fixes"`
	FixCount    int      `json:"fix_count"`
}

// Sources is the input data for design system generation.
type Sources struct {
	LogoAnalysis      map[string]any `json:"logo_analysis,omitempty"`
	IntakeData        map[string]any `json:"intake_data,omitempty"`
	ReferenceAnalysis map[string]any `json:"reference_analysis,omitempty"`
	AISuggestions     map[string]any `json:"ai_suggestions,omitempty"`
}

// GenerateDesignSystem creates a complete design system from sources.
func (f *Fundamentals) GenerateDesignSystem(sources Sources) DesignSystem {
	prefix := determinePrefix(sources)

	ds := DesignSystem{
		Version:       "2.0",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Prefix:        prefix,
		Colors:        f.generateColors(sources),
		Typography:    generateTypography(sources),
		Spacing:       generateSpacing(),
		SpacingTShirt: generateSpacingTShirt(),
		SpacingContext: generateSpacingContextual(),
		Shadows:       make(map[string]string),
		Radii:         generateRadii(),
		Transitions:   generateTransitions(),
		Breakpoints:   generateBreakpoints(),
		ZIndex:        generateZIndex(),
		Cards:         generateCardTokens(),
	}

	// Color partials (HSL/RGB components for runtime manipulation)
	ds.ColorPartials = generateColorPartials(ds.Colors)

	// Shadows (needs primary color for hue-matching)
	shadowTokens := f.GenerateShadowTokens(ds.Colors)
	ds.Shadows = map[string]string{
		"sm": shadowTokens.SM,
		"md": shadowTokens.MD,
		"lg": shadowTokens.LG,
	}

	// Dark mode
	ds.DarkMode = generateDarkMode(ds.Colors)

	// Enforce Design Fundamentals — auto-fix WCAG violations
	ds = f.enforceDesignFundamentals(ds)

	return ds
}

// ═══════════════════════════════════════════════════════════
// COLOR GENERATION
// ═══════════════════════════════════════════════════════════

func (f *Fundamentals) generateColors(sources Sources) map[string]string {
	colors := make(map[string]string)

	logoColors := extractMap(sources.LogoAnalysis, "colors")
	intakeColors := parseIntakeColors(getStr(sources.IntakeData, "brand_colors"))
	refColors := extractBrandColors(sources.ReferenceAnalysis)
	aiColors := extractMap(sources.AISuggestions, "colors")

	// Primary
	colors["primary"] = resolveColor([]string{
		getStr(logoColors, "primary"),
		safeIdx(intakeColors, 0),
		getStr(refColors, "primary"),
		getStr(aiColors, "primary"),
		"#1a365d",
	})

	// Generate 7 shades
	for k, v := range generateColorShades(colors["primary"]) {
		colors["primary_"+k] = v
	}

	// Secondary
	colors["secondary"] = resolveColor([]string{
		getStr(logoColors, "secondary"),
		safeIdx(intakeColors, 1),
		getStr(refColors, "secondary"),
		getStr(aiColors, "secondary"),
		"#ed8936",
	})
	for k, v := range generateColorShades(colors["secondary"]) {
		colors["secondary_"+k] = v
	}

	// Accent
	colors["accent"] = resolveColor([]string{
		getStr(logoColors, "accent"),
		safeIdx(intakeColors, 2),
		getStr(refColors, "accent"),
		getStr(aiColors, "accent"),
		"#38b2ac",
	})
	for k, v := range generateColorShades(colors["accent"]) {
		colors["accent_"+k] = v
	}

	// Base (neutral for text/backgrounds)
	colors["base"] = resolveColor([]string{
		getStr(refColors, "text"),
		"#1a202c",
	})
	for k, v := range generateColorShades(colors["base"]) {
		colors["base_"+k] = v
	}

	// Semantic
	colors["success"] = "#48bb78"
	colors["warning"] = "#ecc94b"
	colors["error"] = "#f56565"
	colors["info"] = "#4299e1"

	// Text
	colors["text_primary"] = colors["base"]
	colors["text_secondary"] = lightenColorPct(colors["base"], 30)
	colors["text_muted"] = lightenColorPct(colors["base"], 50)
	colors["text_inverse"] = "#ffffff"

	// Backgrounds
	bgPage := resolveColor([]string{getStr(refColors, "background"), "#f7fafc"})
	colors["bg_page"] = bgPage
	colors["bg_surface"] = "#ffffff"
	colors["bg_elevated"] = "#ffffff"
	colors["bg_muted"] = darkenColorPct(bgPage, 5)

	// Borders
	colors["border_default"] = resolveColor([]string{getStr(refColors, "border"), "#e2e8f0"})
	colors["border_light"] = lightenColorPct(colors["border_default"], 10)
	colors["border_focus"] = colors["primary"]

	// Neutrals
	colors["white"] = "#ffffff"
	colors["black"] = "#000000"
	neutrals := map[string]string{
		"neutral_100": "#f7fafc", "neutral_200": "#edf2f7", "neutral_300": "#e2e8f0",
		"neutral_400": "#cbd5e0", "neutral_500": "#a0aec0", "neutral_600": "#718096",
		"neutral_700": "#4a5568", "neutral_800": "#2d3748", "neutral_900": "#1a202c",
	}
	for k, v := range neutrals {
		colors[k] = v
	}

	return colors
}

// ═══════════════════════════════════════════════════════════
// COLOR SHADE GENERATION (7 shades via HSL interpolation)
// ═══════════════════════════════════════════════════════════

func generateColorShades(hex string) map[string]string {
	h, s, _ := hexToHSL(hex)
	return map[string]string{
		"ultra_light": hslToHex(h, s, 85),
		"light":       hslToHex(h, s, 70),
		"semi_light":  hslToHex(h, s, 55),
		"semi_dark":   hslToHex(h, s, 35),
		"dark":        hslToHex(h, s, 25),
		"ultra_dark":  hslToHex(h, s, 15),
		"hover":       hslToHex(h, s, math.Max(0, getLightness(hex)-10)),
	}
}

func generateColorPartials(colors map[string]string) map[string]any {
	partials := make(map[string]any)
	mainColors := []string{"primary", "secondary", "accent", "base", "success", "warning", "error", "info"}

	for _, name := range mainColors {
		hex, ok := colors[name]
		if !ok {
			continue
		}
		h, s, l := hexToHSL(hex)
		r, g, b := hexToRGBInts(hex)

		partials[name+"_h"] = int(math.Round(h))
		partials[name+"_s"] = fmt.Sprintf("%.0f%%", s)
		partials[name+"_l"] = fmt.Sprintf("%.0f%%", l)
		partials[name+"_hsl"] = fmt.Sprintf("%.0f %.0f%% %.0f%%", h, s, l)
		partials[name+"_r"] = r
		partials[name+"_g"] = g
		partials[name+"_b"] = b
		partials[name+"_rgb"] = fmt.Sprintf("%d %d %d", r, g, b)
	}

	return partials
}

// ═══════════════════════════════════════════════════════════
// TYPOGRAPHY GENERATION
// ═══════════════════════════════════════════════════════════

type fontPairing struct {
	Heading  string
	Body     string
	Contrast string
	Scale    string
	Vibe     string
}

var fontPairings = []fontPairing{
	{"'Playfair Display', Georgia, serif", "'Lato', sans-serif", "serif+sans", "perfect_fourth", "editorial"},
	{"'DM Serif Display', Georgia, serif", "'DM Sans', sans-serif", "serif+sans", "perfect_fourth", "magazine"},
	{"'Space Grotesk', sans-serif", "'Inter', sans-serif", "geometric+humanist", "major_third", "modern-tech"},
	{"'Sora', sans-serif", "'Inter', sans-serif", "geometric+humanist", "major_third", "geometric-tech"},
	{"'Outfit', sans-serif", "'Source Sans 3', sans-serif", "geometric+humanist", "major_third", "clean-saas"},
	{"'Plus Jakarta Sans', sans-serif", "'Nunito Sans', sans-serif", "geometric+humanist", "minor_third", "friendly-tech"},
	{"'Manrope', sans-serif", "'Inter', sans-serif", "weight", "major_third", "corporate-modern"},
	{"'Poppins', sans-serif", "'Nunito Sans', sans-serif", "weight", "minor_third", "approachable-corp"},
	{"'Lexend', sans-serif", "'Lato', sans-serif", "weight", "major_second", "accessible-corp"},
	{"'Clash Display', sans-serif", "'Satoshi', sans-serif", "display+text", "perfect_fourth", "bold-agency"},
	{"'General Sans', sans-serif", "'Inter', sans-serif", "display+text", "major_third", "creative-studio"},
	{"'Cabinet Grotesk', sans-serif", "'General Sans', sans-serif", "display+text", "perfect_fourth", "startup"},
	{"'Cormorant Garamond', Georgia, serif", "'Raleway', sans-serif", "serif+sans", "perfect_fifth", "luxury"},
	{"'Instrument Serif', Georgia, serif", "'Instrument Sans', sans-serif", "serif+sans", "major_third", "refined-creative"},
	{"'Playfair Display', Georgia, serif", "'Nunito Sans', sans-serif", "serif+sans", "perfect_fourth", "elegant"},
	{"'DM Sans', sans-serif", "'Inter', sans-serif", "weight", "minor_third", "ecommerce-clean"},
	{"'Fraunces', Georgia, serif", "'Commissioner', sans-serif", "serif+sans", "augmented_fourth", "organic-editorial"},
	{"'Inter', sans-serif", "'Inter', sans-serif", "superfamily", "major_second", "minimal"},
}

var vibeMap = map[string]string{
	"saas": "modern-tech", "software": "modern-tech", "tech": "geometric-tech",
	"agency": "bold-agency", "creative": "creative-studio", "design": "refined-creative",
	"consulting": "corporate-modern", "finance": "corporate-modern", "legal": "classic-content",
	"ecommerce": "ecommerce-clean", "retail": "retail-friendly",
	"blog": "editorial", "media": "magazine", "news": "magazine",
	"luxury": "luxury", "fashion": "elegant", "beauty": "elegant",
	"health": "approachable-corp", "medical": "accessible-corp", "education": "friendly-tech",
	"startup": "startup", "nonprofit": "approachable-corp", "restaurant": "organic-editorial",
	"portfolio": "refined-creative", "photography": "elegant", "architecture": "luxury",
}

var scaleRatios = map[string]float64{
	"minor_second": 1.067, "major_second": 1.125, "minor_third": 1.200,
	"major_third": 1.250, "perfect_fourth": 1.333, "augmented_fourth": 1.414,
	"perfect_fifth": 1.500, "golden": 1.618,
}

func generateTypography(sources Sources) map[string]any {
	typography := make(map[string]any)

	// Determine business type for vibe matching
	businessType := strings.ToLower(getStr(sources.IntakeData, "business_type"))
	if businessType == "" {
		businessType = strings.ToLower(getStr(sources.ReferenceAnalysis, "detected_industry"))
	}

	// Priority waterfall: intake → reference → AI → curated library
	intakeFonts := getStr(sources.IntakeData, "brand_fonts")
	refTypo := extractStringMap(sources.ReferenceAnalysis, "typography")

	var selectedPair *fontPairing

	if intakeFonts != "" {
		parts := strings.SplitN(intakeFonts, ",", 2)
		typography["font_heading"] = "'" + strings.TrimSpace(parts[0]) + "', sans-serif"
		if len(parts) > 1 {
			typography["font_body"] = "'" + strings.TrimSpace(parts[1]) + "', sans-serif"
		} else {
			typography["font_body"] = typography["font_heading"]
		}
	} else if refTypo["heading_font"] != "" || refTypo["heading"] != "" {
		h := refTypo["heading_font"]
		if h == "" {
			h = refTypo["heading"]
		}
		typography["font_heading"] = h
		b := refTypo["body_font"]
		if b == "" {
			b = refTypo["body"]
		}
		if b == "" {
			b = h
		}
		typography["font_body"] = b
	} else {
		// Vibe-matched selection from curated library
		targetVibe := vibeMap[businessType]
		if targetVibe == "" {
			for k, v := range vibeMap {
				if strings.Contains(businessType, k) {
					targetVibe = v
					break
				}
			}
		}

		if targetVibe != "" {
			for i := range fontPairings {
				if fontPairings[i].Vibe == targetVibe {
					selectedPair = &fontPairings[i]
					break
				}
			}
		}
		if selectedPair == nil {
			// Default to modern-tech
			selectedPair = &fontPairings[2] // Space Grotesk + Inter
		}
		typography["font_heading"] = selectedPair.Heading
		typography["font_body"] = selectedPair.Body
	}

	typography["font_mono"] = "'JetBrains Mono', 'Fira Code', 'SF Mono', monospace"

	// Type scale
	scaleName := "major_third"
	if selectedPair != nil {
		scaleName = selectedPair.Scale
	}
	ratio := scaleRatios[scaleName]
	if ratio == 0 {
		ratio = 1.250
	}

	typography["scale_ratio"] = ratio
	typography["scale_name"] = scaleName

	// Fluid scale using clamp()
	baseMobile := 1.0
	baseDesktop := 1.15
	steps := map[string]int{
		"text_xs": -2, "text_sm": -1, "text_base": 0,
		"text_lg": 1, "text_xl": 2, "text_2xl": 3,
		"text_3xl": 4, "text_4xl": 5, "text_5xl": 6, "text_6xl": 7,
	}
	for key, step := range steps {
		mobile := baseMobile * math.Pow(ratio, float64(step))
		desktop := baseDesktop * math.Pow(ratio, float64(step))
		slope := (desktop - mobile) / (75 - 20)
		intercept := mobile - (slope * 20)
		typography[key] = fmt.Sprintf("clamp(%.3frem, %.4frem + %.4fvw, %.3frem)",
			mobile, intercept, slope*100, desktop)
	}

	// Line heights, weights, tracking
	typography["leading_display"] = "1.05"
	typography["leading_tight"] = "1.15"
	typography["leading_snug"] = "1.3"
	typography["leading_normal"] = "1.6"
	typography["leading_relaxed"] = "1.75"

	typography["weight_normal"] = "400"
	typography["weight_medium"] = "500"
	typography["weight_semibold"] = "600"
	typography["weight_bold"] = "700"
	typography["weight_extrabold"] = "800"

	typography["tracking_tight"] = "-0.025em"
	typography["tracking_normal"] = "0"
	typography["tracking_wide"] = "0.025em"

	// Rhythm
	typography["paragraph_gap"] = "1.5rem"
	typography["heading_gap_above"] = "2.5rem"
	typography["heading_gap_below"] = "1rem"
	typography["section_gap"] = "4rem"

	// Measure
	typography["measure_ideal"] = "65ch"

	return typography
}

// ═══════════════════════════════════════════════════════════
// SPACING / SHADOWS / RADII / TRANSITIONS / BREAKPOINTS
// ═══════════════════════════════════════════════════════════

func generateSpacing() map[string]string {
	return map[string]string{
		"space_0": "0", "space_1": "0.25rem", "space_2": "0.5rem",
		"space_3": "0.75rem", "space_4": "1rem", "space_5": "1.25rem",
		"space_6": "1.5rem", "space_8": "2rem", "space_10": "2.5rem",
		"space_12": "3rem", "space_16": "4rem", "space_20": "5rem", "space_24": "6rem",
	}
}

func generateSpacingTShirt() map[string]string {
	return map[string]string{
		"xs": "0.25rem", "sm": "0.5rem", "md": "1rem",
		"lg": "1.5rem", "xl": "2rem", "2xl": "3rem",
		"3xl": "4rem", "4xl": "6rem", "5xl": "8rem",
	}
}

func generateSpacingContextual() map[string]string {
	return map[string]string{
		"section_padding":  "clamp(3rem, 5vw, 6rem)",
		"container_max":    "1200px",
		"container_gutter": "clamp(1rem, 3vw, 2rem)",
		"card_padding":     "clamp(1rem, 2vw, 1.5rem)",
		"button_padding_x": "1.5rem",
		"button_padding_y": "0.75rem",
	}
}

func generateRadii() map[string]string {
	return map[string]string{
		"none": "0", "sm": "0.25rem", "md": "0.5rem",
		"lg": "0.75rem", "xl": "1rem", "2xl": "1.5rem",
		"full": "9999px",
	}
}

func generateTransitions() map[string]string {
	return map[string]string{
		"fast":    "150ms ease",
		"default": "300ms ease",
		"slow":    "500ms ease",
		"spring":  "300ms cubic-bezier(0.34, 1.56, 0.64, 1)",
	}
}

func generateBreakpoints() map[string]string {
	return map[string]string{
		"sm": "640px", "md": "768px", "lg": "1024px",
		"xl": "1280px", "2xl": "1536px",
	}
}

func generateZIndex() map[string]int {
	return map[string]int{
		"behind": -1, "base": 0, "dropdown": 1000,
		"sticky": 1020, "fixed": 1030, "modal_backdrop": 1040,
		"modal": 1050, "popover": 1060, "tooltip": 1070,
	}
}

func generateCardTokens() map[string]string {
	return map[string]string{
		"padding":      "1.5rem",
		"radius":       "0.75rem",
		"shadow":       "sm",
		"shadow_hover": "md",
		"border":       "1px solid var(--border-default)",
		"bg":           "var(--bg-surface)",
		"gap":          "1rem",
	}
}

func generateDarkMode(lightColors map[string]string) map[string]string {
	dark := make(map[string]string)
	dark["bg_page"] = "#0f172a"
	dark["bg_surface"] = "#1e293b"
	dark["bg_elevated"] = "#334155"
	dark["text_primary"] = "#f1f5f9"
	dark["text_secondary"] = "#94a3b8"
	dark["text_muted"] = "#64748b"
	dark["border_default"] = "#334155"
	dark["border_light"] = "#475569"

	// Keep brand colors, lighten for dark backgrounds
	for _, role := range []string{"primary", "secondary", "accent"} {
		if c, ok := lightColors[role]; ok {
			dark[role] = c
			h, s, _ := hexToHSL(c)
			dark[role+"_light"] = hslToHex(h, s, 70)
		}
	}

	return dark
}

// ═══════════════════════════════════════════════════════════
// DESIGN FUNDAMENTALS ENFORCEMENT
// ═══════════════════════════════════════════════════════════

func (f *Fundamentals) enforceDesignFundamentals(ds DesignSystem) DesignSystem {
	var fixes []string
	bg := ds.Colors["bg_page"]
	if bg == "" {
		bg = "#FFFFFF"
	}

	// 1. Text-on-background contrast
	textPrimary := ds.Colors["text_primary"]
	if textPrimary != "" {
		check := f.ValidateContrast(textPrimary, bg, false)
		if !check.Pass {
			best := f.BestTextColor(bg, "", "")
			fixes = append(fixes, fmt.Sprintf("Fixed text_primary: %s → %s (was %.1f:1 on %s)", textPrimary, best, check.Ratio, bg))
			ds.Colors["text_primary"] = best
		}
	}

	// 2. Block dangerous CTA combos
	for _, role := range []string{"accent", "secondary", "primary"} {
		c := ds.Colors[role]
		if c == "" {
			continue
		}
		combo := f.CheckDangerousCombo("#FFFFFF", c)
		if combo.Blocked {
			fixes = append(fixes, fmt.Sprintf("Blocked: %s (%s) unsafe as CTA bg with white text — %s", role, c, combo.Reason))
			ds.Colors[role+"_cta_text"] = f.BestTextColor(c, "", "")
		}
	}

	// 3. Generate safe CTA colors
	ctaBg := ds.Colors["primary"]
	if ctaBg == "" {
		ctaBg = "#1a365d"
	}
	ds.Colors["cta_bg"] = ctaBg
	ds.Colors["cta_text"] = f.BestTextColor(ctaBg, "", "")

	// Section accent bg
	accentBg := lightenColor(ctaBg, 0.85)
	ds.Colors["section_accent_bg"] = accentBg
	ds.Colors["section_accent_text"] = f.BestTextColor(accentBg, "", "")

	// Dark section
	ds.Colors["section_dark_bg"] = "#111827"
	ds.Colors["section_dark_text"] = "#f9fafb"

	// 4. Typography validation
	typoCheck := f.ValidateTypography(extractTypoStrings(ds.Typography))
	for _, issue := range typoCheck.Issues {
		fixes = append(fixes, "Typography: "+issue)
	}

	ds.Fundamentals = &FundamentalsAudit{
		ValidatedAt: time.Now().UTC().Format(time.RFC3339),
		Fixes:       fixes,
		FixCount:    len(fixes),
	}

	return ds
}

// ═══════════════════════════════════════════════════════════
// CSS OUTPUT
// ═══════════════════════════════════════════════════════════

// ToCSSVars converts the design system to CSS custom properties.
func (ds *DesignSystem) ToCSSVars() string {
	var b strings.Builder
	b.WriteString(":root {\n")

	for k, v := range ds.Colors {
		fmt.Fprintf(&b, "  --%s-%s: %s;\n", ds.Prefix, k, v)
	}
	for k, v := range ds.Typography {
		if s, ok := v.(string); ok {
			fmt.Fprintf(&b, "  --%s-%s: %s;\n", ds.Prefix, k, s)
		}
	}
	for k, v := range ds.Spacing {
		fmt.Fprintf(&b, "  --%s-%s: %s;\n", ds.Prefix, k, v)
	}
	for k, v := range ds.Shadows {
		fmt.Fprintf(&b, "  --%s-shadow-%s: %s;\n", ds.Prefix, k, v)
	}
	for k, v := range ds.Radii {
		fmt.Fprintf(&b, "  --%s-radius-%s: %s;\n", ds.Prefix, k, v)
	}

	b.WriteString("}\n")
	return b.String()
}

// ═══════════════════════════════════════════════════════════
// INTERNAL HELPERS
// ═══════════════════════════════════════════════════════════

func determinePrefix(sources Sources) string {
	if getStr(sources.LogoAnalysis, "colors") != "" {
		return "logo"
	}
	if getStr(sources.IntakeData, "brand_colors") != "" || getStr(sources.IntakeData, "brand_fonts") != "" {
		return "client"
	}
	if len(sources.ReferenceAnalysis) > 0 {
		return "ref"
	}
	return "brand"
}

func resolveColor(candidates []string) string {
	hexRe := hexColorRegex
	for _, c := range candidates {
		if c != "" && hexRe.MatchString(c) {
			return c
		}
	}
	return "#000000"
}

var hexColorRegex = compileRegex(`^#[0-9A-Fa-f]{6}$`)

func parseIntakeColors(s string) []string {
	if s == "" {
		return nil
	}
	var colors []string
	for _, c := range strings.Split(s, ",") {
		c = strings.TrimSpace(c)
		if hexColorRegex.MatchString(c) {
			colors = append(colors, c)
		}
	}
	return colors
}

func extractBrandColors(refAnalysis map[string]any) map[string]any {
	if branding, ok := refAnalysis["branding"].(map[string]any); ok {
		if colors, ok := branding["colors"].(map[string]any); ok {
			return colors
		}
	}
	return map[string]any{}
}

func safeIdx(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

func hexToHSL(hex string) (h, s, l float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r := hexVal(hex[0:2]) / 255
	g := hexVal(hex[2:4]) / 255
	b := hexVal(hex[4:6]) / 255

	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2

	if mx == mn {
		return 0, 0, l * 100
	}

	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}

	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6

	return h * 360, s * 100, l * 100
}

func getLightness(hex string) float64 {
	_, _, l := hexToHSL(hex)
	return l
}

func hslToHex(h, s, l float64) string {
	h /= 360
	s /= 100
	l /= 100

	if s == 0 {
		v := int(math.Round(l * 255))
		return fmt.Sprintf("#%02x%02x%02x", v, v, v)
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r := hueToRGB(p, q, h+1.0/3.0)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3.0)

	return fmt.Sprintf("#%02x%02x%02x",
		int(math.Round(r*255)),
		int(math.Round(g*255)),
		int(math.Round(b*255)),
	)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func hexToRGBInts(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r := int(hexVal(hex[0:2]))
	g := int(hexVal(hex[2:4]))
	b := int(hexVal(hex[4:6]))
	return r, g, b
}

func lightenColorPct(hex string, pct int) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "#f9fafb"
	}
	r := int(hexVal(hex[0:2]))
	g := int(hexVal(hex[2:4]))
	b := int(hexVal(hex[4:6]))

	r = minI(255, r+(255-r)*pct/100)
	g = minI(255, g+(255-g)*pct/100)
	b = minI(255, b+(255-b)*pct/100)

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func darkenColorPct(hex string, pct int) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "#111827"
	}
	r := int(hexVal(hex[0:2]))
	g := int(hexVal(hex[2:4]))
	b := int(hexVal(hex[4:6]))

	r = maxI(0, r-r*pct/100)
	g = maxI(0, g-g*pct/100)
	b = maxI(0, b-b*pct/100)

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func compileRegex(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

func extractTypoStrings(typo map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range typo {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}
