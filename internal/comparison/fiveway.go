// Package comparison implements the 5-Way Compliance Scoring system.
//
// Port of PHP class-five-way-comparison.php (793 LOC).
// Compares a built page against reference analysis across 5 dimensions:
//
//   1. Token compliance  — colors, typography, spacing, shapes
//   2. Section compliance — section inventory, order, completeness
//   3. Component compliance — component types, counts, positions
//   4. UICrit scores — 7-dimension design critique scores
//   5. Priority fixes — aggregated, ranked actionable fixes
//
// The Go implementation operates on HTML strings rather than fetching URLs,
// keeping the engine stateless. The plugin or caller is responsible for
// fetching the built page HTML.
package comparison

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// TokenCompliance is the result of comparing built-page tokens vs reference.
type TokenCompliance struct {
	Success           bool            `json:"success"`
	ColorsMatch       int             `json:"colors_match"`
	TypographyMatch   int             `json:"typography_match"`
	SpacingMatch      int             `json:"spacing_match"`
	ShapesMatch       int             `json:"shapes_match"`
	OverallCompliance int             `json:"overall_compliance"`
	Details           map[string]any  `json:"details,omitempty"`
}

// SectionCompliance is the result of comparing built sections vs reference.
type SectionCompliance struct {
	Success           bool     `json:"success"`
	SectionsExpected  int      `json:"sections_expected"`
	SectionsFound     int      `json:"sections_found"`
	Matched           int      `json:"matched"`
	Missing           []string `json:"missing"`
	Extra             []string `json:"extra"`
	OrderCorrect      bool     `json:"order_correct"`
	ComplianceScore   int      `json:"compliance_score"`
}

// ComponentCompliance is the result of comparing built components vs reference.
type ComponentCompliance struct {
	Success            bool               `json:"success"`
	ComponentsExpected int                `json:"components_expected"`
	ComponentsFound    int                `json:"components_found"`
	Matched            int                `json:"matched"`
	Missing            []MissingComponent `json:"missing"`
	ComplianceScore    int                `json:"compliance_score"`
}

type MissingComponent struct {
	Type        string `json:"type"`
	Section     string `json:"section"`
	ContentHint string `json:"content_hint"`
}

// PriorityFixes is the aggregated fix list from all comparison dimensions.
type PriorityFixes struct {
	Fixes       []string `json:"priority_fixes"`
	TotalIssues int      `json:"total_issues"`
}

// FiveWayResult combines all 5 comparison dimensions.
type FiveWayResult struct {
	TokenCompliance     *TokenCompliance     `json:"token_compliance,omitempty"`
	SectionCompliance   *SectionCompliance   `json:"section_compliance,omitempty"`
	ComponentCompliance *ComponentCompliance  `json:"component_compliance,omitempty"`
	CritiqueScores      map[string]any       `json:"critique_scores,omitempty"`
	PriorityFixes       *PriorityFixes       `json:"priority_fixes,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// TOKEN COMPLIANCE (colors, typography, spacing, shapes)
// ═══════════════════════════════════════════════════════════

// CalculateTokenCompliance compares built HTML against reference design tokens.
func CalculateTokenCompliance(html string, designTokens map[string]any) TokenCompliance {
	refColors := extractStringMap(designTokens, "colors")
	refTypography := extractStringMap(designTokens, "typography")
	refSpacing := extractStringMap(designTokens, "spacing")
	refShapes := extractStringMap(designTokens, "shapes")

	colorScore := compareColors(html, refColors)
	typoScore := compareTypography(html, refTypography)
	spacingScore := compareSpacing(html, refSpacing)
	shapesScore := compareShapes(html, refShapes)

	overall := int(math.Round(
		float64(colorScore)*0.30 +
			float64(typoScore)*0.25 +
			float64(spacingScore)*0.25 +
			float64(shapesScore)*0.20,
	))

	return TokenCompliance{
		Success:           true,
		ColorsMatch:       colorScore,
		TypographyMatch:   typoScore,
		SpacingMatch:      spacingScore,
		ShapesMatch:       shapesScore,
		OverallCompliance: overall,
	}
}

// ═══════════════════════════════════════════════════════════
// SECTION COMPLIANCE
// ═══════════════════════════════════════════════════════════

// CalculateSectionCompliance compares detected sections against reference.
func CalculateSectionCompliance(html string, referenceSections []map[string]any) SectionCompliance {
	detected := detectSectionsInHTML(html)
	expected := referenceSections

	foundNames := make([]string, len(detected))
	for i, d := range detected {
		foundNames[i] = strings.ToLower(d)
	}

	matched := 0
	var missing, extra []string

	for _, sec := range expected {
		name := strings.ToLower(fmt.Sprintf("%v", sec["name"]))
		found := false
		for _, fn := range foundNames {
			if fn == name {
				found = true
				break
			}
		}
		if found {
			matched++
		} else {
			missing = append(missing, name)
		}
	}

	expectedNames := make(map[string]bool)
	for _, sec := range expected {
		expectedNames[strings.ToLower(fmt.Sprintf("%v", sec["name"]))] = true
	}
	for _, fn := range foundNames {
		if !expectedNames[fn] {
			extra = append(extra, fn)
		}
	}

	// Check order
	orderCorrect := checkSectionOrder(foundNames, expected)

	// Score
	baseScore := 100
	if len(expected) > 0 {
		baseScore = int(math.Round(float64(matched) / float64(len(expected)) * 100))
	}
	penalty := len(extra)*5 + boolToInt(!orderCorrect)*10
	score := max(0, baseScore-penalty)

	return SectionCompliance{
		Success:          true,
		SectionsExpected: len(expected),
		SectionsFound:    len(detected),
		Matched:          matched,
		Missing:          missing,
		Extra:            extra,
		OrderCorrect:     orderCorrect,
		ComplianceScore:  score,
	}
}

// ═══════════════════════════════════════════════════════════
// COMPONENT COMPLIANCE
// ═══════════════════════════════════════════════════════════

// CalculateComponentCompliance compares detected components against reference.
func CalculateComponentCompliance(html string, referenceComponents []map[string]any) ComponentCompliance {
	// Count detected component types in HTML
	detected := map[string]int{
		"button":  countMatches(html, `<(?:button|a[^>]*class="[^"]*btn)`),
		"heading": countMatches(html, `<h[1-6]`),
		"image":   countMatches(html, `<img`),
		"card":    countMatches(html, `class="[^"]*card`),
	}

	// Count expected by type
	expectedByType := map[string]int{}
	for _, comp := range referenceComponents {
		t := strings.ToLower(fmt.Sprintf("%v", comp["type"]))
		// Normalize to detectable types
		normalized := normalizeComponentType(t)
		expectedByType[normalized]++
	}

	matched := 0
	totalTypes := len(expectedByType)
	for t, count := range expectedByType {
		if d, ok := detected[t]; ok && d > 0 {
			// Type exists; check if count is close enough (≥50% of expected)
			threshold := count / 2
			if threshold < 1 {
				threshold = 1
			}
			if d >= threshold {
				matched++
			}
		}
	}

	score := 0
	if totalTypes > 0 {
		score = int(math.Round(float64(matched) / float64(totalTypes) * 100))
	}

	// Find missing
	var missing []MissingComponent
	for _, comp := range referenceComponents {
		t := strings.ToLower(fmt.Sprintf("%v", comp["type"]))
		normalized := normalizeComponentType(t)
		if detected[normalized] == 0 {
			missing = append(missing, MissingComponent{
				Type:        fmt.Sprintf("%v", comp["type"]),
				Section:     fmt.Sprintf("%v", comp["section"]),
				ContentHint: fmt.Sprintf("%v", comp["content_hint"]),
			})
		}
	}

	totalFound := 0
	for _, v := range detected {
		totalFound += v
	}

	return ComponentCompliance{
		Success:            true,
		ComponentsExpected: len(referenceComponents),
		ComponentsFound:    totalFound,
		Matched:            matched,
		Missing:            missing,
		ComplianceScore:    score,
	}
}

// ═══════════════════════════════════════════════════════════
// PRIORITY FIXES
// ═══════════════════════════════════════════════════════════

// GeneratePriorityFixes aggregates all comparison results into ranked fixes.
func GeneratePriorityFixes(result FiveWayResult) PriorityFixes {
	var fixes []priorityFix

	// Section compliance gaps
	if sc := result.SectionCompliance; sc != nil {
		for _, m := range sc.Missing {
			fixes = append(fixes, priorityFix{
				text:     fmt.Sprintf("Add missing section: %s (between %s and %s)", m, guessBefore(m), guessAfter(m)),
				priority: 100,
			})
		}
		if !sc.OrderCorrect {
			fixes = append(fixes, priorityFix{
				text:     "Reorder sections to match reference layout",
				priority: 90,
			})
		}
	}

	// Token compliance gaps
	if tc := result.TokenCompliance; tc != nil {
		if tc.SpacingMatch < 70 {
			fixes = append(fixes, priorityFix{
				text:     "Increase section padding to match reference (currently too tight)",
				priority: 60,
			})
		}
		if tc.TypographyMatch < 70 {
			fixes = append(fixes, priorityFix{
				text:     "Update font families to match reference",
				priority: 50,
			})
		}
		if tc.ColorsMatch < 70 {
			fixes = append(fixes, priorityFix{
				text:     "Adjust color palette to match reference hex values",
				priority: 40,
			})
		}
	}

	// Component gaps
	if cc := result.ComponentCompliance; cc != nil {
		for _, m := range cc.Missing {
			fixes = append(fixes, priorityFix{
				text:     fmt.Sprintf("Add component: %s in %s section (%s)", m.Type, m.Section, m.ContentHint),
				priority: 70,
			})
		}
	}

	// UICrit score gaps
	if cs := result.CritiqueScores; cs != nil {
		for dim, data := range cs {
			if dm, ok := data.(map[string]any); ok {
				if score, ok := dm["score"].(float64); ok && score < 7.0 {
					fixes = append(fixes, priorityFix{
						text:     fmt.Sprintf("Improve %s (currently %.1f/10)", dim, score),
						priority: 30,
					})
				}
			}
		}
	}

	// Sort by priority (highest first)
	sort.Slice(fixes, func(i, j int) bool {
		return fixes[i].priority > fixes[j].priority
	})

	// Limit to top 5
	top := make([]string, 0, 5)
	for i := 0; i < len(fixes) && i < 5; i++ {
		top = append(top, fixes[i].text)
	}

	return PriorityFixes{
		Fixes:       top,
		TotalIssues: len(fixes),
	}
}

// ═══════════════════════════════════════════════════════════
// INTERNAL HELPERS
// ═══════════════════════════════════════════════════════════

type priorityFix struct {
	text     string
	priority int
}

var (
	cssColorRe  = regexp.MustCompile(`(?i)(?:color|background-color|border-color):\s*(#[0-9a-fA-F]{3,6})`)
	fontFamilyRe = regexp.MustCompile(`(?i)font-family:\s*([^;]+)`)
	fontWeightRe = regexp.MustCompile(`(?i)font-weight:\s*(\d+)`)
	paddingRe    = regexp.MustCompile(`(?i)padding:\s*([^;]+)`)
	marginRe     = regexp.MustCompile(`(?i)margin:\s*([^;]+)`)
	pxValueRe    = regexp.MustCompile(`(\d+)px`)
	radiusRe     = regexp.MustCompile(`(?i)border-radius:\s*([^;]+)`)
	shadowRe     = regexp.MustCompile(`(?i)box-shadow:\s*([^;]+)`)
)

func compareColors(html string, refColors map[string]string) int {
	matches := cssColorRe.FindAllStringSubmatch(html, -1)
	extracted := make(map[string]bool)
	for _, m := range matches {
		extracted[strings.ToLower(m[1])] = true
	}

	total := 0
	matched := 0
	for _, refHex := range refColors {
		if refHex == "" || !strings.HasPrefix(refHex, "#") {
			continue
		}
		total++
		refHex = strings.ToLower(refHex)
		for ex := range extracted {
			if colorDistance(refHex, ex) < 5 {
				matched++
				break
			}
		}
	}

	if total == 0 {
		return 100
	}
	return int(math.Round(float64(matched) / float64(total) * 100))
}

func colorDistance(hex1, hex2 string) float64 {
	r1, g1, b1 := hexToRGB(hex1)
	r2, g2, b2 := hexToRGB(hex2)
	return math.Sqrt(
		math.Pow(float64(r1-r2), 2) +
			math.Pow(float64(g1-g2), 2) +
			math.Pow(float64(b1-b2), 2),
	)
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	}
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(r), int(g), int(b)
}

func compareTypography(html string, refTypo map[string]string) int {
	families := fontFamilyRe.FindAllStringSubmatch(html, -1)
	familySet := make(map[string]bool)
	for _, m := range families {
		familySet[strings.ToLower(strings.Trim(m[1], "'\"` "))] = true
	}

	headingMatch := 0
	if hf := strings.ToLower(refTypo["heading_font"]); hf != "" {
		for f := range familySet {
			if strings.Contains(f, hf) || strings.Contains(hf, strings.ReplaceAll(f, " ", "")) {
				headingMatch = 100
				break
			}
		}
	}

	bodyMatch := 0
	if bf := strings.ToLower(refTypo["body_font"]); bf != "" {
		for f := range familySet {
			if strings.Contains(f, bf) || strings.Contains(bf, strings.ReplaceAll(f, " ", "")) {
				bodyMatch = 100
				break
			}
		}
	}

	weightMatch := 0
	weights := fontWeightRe.FindAllStringSubmatch(html, -1)
	foundWeights := make(map[string]bool)
	for _, m := range weights {
		foundWeights[m[1]] = true
	}
	// Check any reference weight present
	for _, w := range []string{"300", "400", "500", "600", "700", "900"} {
		if foundWeights[w] {
			weightMatch = 100
			break
		}
	}

	return int(math.Round(float64(headingMatch+bodyMatch+weightMatch) / 3))
}

func compareSpacing(html string, refSpacing map[string]string) int {
	var values []int
	for _, re := range []*regexp.Regexp{paddingRe, marginRe} {
		for _, m := range re.FindAllStringSubmatch(html, -1) {
			for _, pm := range pxValueRe.FindAllStringSubmatch(m[1], -1) {
				v, _ := strconv.Atoi(pm[1])
				if v > 0 {
					values = append(values, v)
				}
			}
		}
	}

	if len(values) == 0 {
		return 50
	}

	sum := 0
	for _, v := range values {
		sum += v
	}
	avg := float64(sum) / float64(len(values))

	refSection := parseSpacingRange(refSpacing["section_padding"])
	refComponent := parseSpacingRange(refSpacing["component_gap"])

	sectionMatch := 50
	if avg >= float64(refSection)*0.8 && avg <= float64(refSection)*1.2 {
		sectionMatch = 100
	}
	componentMatch := 50
	if avg >= float64(refComponent)*0.8 && avg <= float64(refComponent)*1.2 {
		componentMatch = 100
	}

	return (sectionMatch + componentMatch) / 2
}

func parseSpacingRange(s string) int {
	re := regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*px`)
	if m := re.FindStringSubmatch(s); len(m) > 2 {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return (a + b) / 2
	}
	if m := pxValueRe.FindStringSubmatch(s); len(m) > 1 {
		v, _ := strconv.Atoi(m[1])
		return v
	}
	s = strings.ToLower(s)
	if strings.Contains(s, "small") {
		return 16
	}
	if strings.Contains(s, "medium") {
		return 24
	}
	if strings.Contains(s, "large") {
		return 32
	}
	return 24
}

func compareShapes(html string, refShapes map[string]string) int {
	radii := radiusRe.FindAllStringSubmatch(html, -1)
	shadows := shadowRe.FindAllStringSubmatch(html, -1)

	radiusMatch := checkRadiusMatch(radii, refShapes["button_radius"])
	shadowMatch := checkShadowMatch(shadows, refShapes["shadow_style"])

	return (radiusMatch + shadowMatch) / 2
}

func checkRadiusMatch(radii [][]string, reference string) int {
	if len(radii) == 0 {
		return 50
	}
	refValue := parseRadiusValue(reference)
	for _, r := range radii {
		v := strings.TrimSpace(strings.ToLower(r[1]))
		if v == "0" || v == "none" {
			if refValue == 0 {
				return 100
			}
		} else if strings.Contains(v, "%") && refValue == 50 && strings.Contains(v, "50") {
			return 100
		} else if m := pxValueRe.FindStringSubmatch(v); len(m) > 1 {
			px, _ := strconv.Atoi(m[1])
			if abs(px-refValue) <= 4 {
				return 100
			}
		}
	}
	return 50
}

func parseRadiusValue(s string) int {
	m := map[string]int{
		"sharp": 0, "none": 0, "small": 6, "medium": 14, "large": 24, "pill": 50,
	}
	s = strings.ToLower(s)
	for name, px := range m {
		if strings.Contains(s, name) {
			return px
		}
	}
	if pm := pxValueRe.FindStringSubmatch(s); len(pm) > 1 {
		v, _ := strconv.Atoi(pm[1])
		return v
	}
	return 6
}

func checkShadowMatch(shadows [][]string, reference string) int {
	if len(shadows) == 0 {
		if strings.ToLower(reference) == "none" {
			return 100
		}
		return 50
	}
	if strings.ToLower(reference) != "none" {
		return 100 // Has shadows and reference expects shadows
	}
	return 50
}

func detectSectionsInHTML(html string) []string {
	var sections []string
	lower := strings.ToLower(html)

	patterns := []struct {
		pattern string
		name    string
	}{
		{`<header|role="banner"|class="[^"]*header`, "header"},
		{`class="[^"]*hero`, "hero"},
		{`class="[^"]*feature`, "features"},
		{`class="[^"]*testimonial`, "testimonials"},
		{`class="[^"]*cta`, "cta"},
		{`<footer|role="contentinfo"|class="[^"]*footer`, "footer"},
	}

	for _, p := range patterns {
		re := regexp.MustCompile(`(?i)` + p.pattern)
		if re.MatchString(lower) {
			sections = append(sections, p.name)
		}
	}

	return sections
}

func checkSectionOrder(foundNames []string, expected []map[string]any) bool {
	keySections := []string{"header", "hero", "footer"}
	for _, key := range keySections {
		foundPos := indexOf(foundNames, key)
		expectedPos := -1
		for i, sec := range expected {
			if strings.ToLower(fmt.Sprintf("%v", sec["name"])) == key {
				expectedPos = i
				break
			}
		}
		if foundPos >= 0 && expectedPos >= 0 && abs(foundPos-expectedPos) > 2 {
			return false
		}
	}
	return true
}

func normalizeComponentType(t string) string {
	if strings.HasPrefix(t, "button") || strings.HasPrefix(t, "icon_button") {
		return "button"
	}
	if strings.HasPrefix(t, "heading") {
		return "heading"
	}
	if strings.Contains(t, "image") || strings.Contains(t, "avatar") || strings.Contains(t, "logo") {
		return "image"
	}
	if strings.Contains(t, "card") {
		return "card"
	}
	return t
}

var sectionOrder = []string{"header", "nav", "hero", "features", "services", "testimonials", "team", "pricing", "cta", "footer"}

func guessBefore(section string) string {
	for i, s := range sectionOrder {
		if strings.EqualFold(s, section) && i > 0 {
			return sectionOrder[i-1]
		}
	}
	return "previous section"
}

func guessAfter(section string) string {
	for i, s := range sectionOrder {
		if strings.EqualFold(s, section) && i < len(sectionOrder)-1 {
			return sectionOrder[i+1]
		}
	}
	return "next section"
}

func countMatches(html string, pattern string) int {
	re := regexp.MustCompile(`(?i)` + pattern)
	return len(re.FindAllString(html, -1))
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func extractStringMap(m map[string]any, key string) map[string]string {
	result := make(map[string]string)
	if sub, ok := m[key].(map[string]any); ok {
		for k, v := range sub {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
	}
	return result
}
