// composition.go — Port of class-composition-fundamentals.php (367 LOC)
// Validates page composition: hero sections, page rhythm, feature grids, stats.
package design

import (
	"fmt"
	"strings"
)

// CompositionResult describes a composition audit.
type CompositionResult struct {
	Score    int      `json:"score"`
	Pass     bool     `json:"pass"`
	Issues   []string `json:"issues"`
	Warnings []string `json:"warnings"`
}

// ValidateHero checks a hero section for composition fundamentals.
func (f *Fundamentals) ValidateHero(section map[string]any) CompositionResult {
	var issues, warnings []string

	// Rule H-1: Hero must have headline
	headline := getStr(section, "headline")
	if headline == "" {
		issues = append(issues, "Hero section missing headline (Rule H-1)")
	} else if len(headline) > 80 {
		warnings = append(warnings, fmt.Sprintf("Hero headline %d chars — ideal max 60-80 (Rule H-2)", len(headline)))
	}

	// Rule H-3: Hero should have subheadline/description
	sub := getStr(section, "subheadline")
	if sub == "" {
		sub = getStr(section, "description")
	}
	if sub == "" {
		warnings = append(warnings, "Hero section missing subheadline/description (Rule H-3)")
	}

	// Rule H-4: Hero should have CTA
	cta := getStr(section, "cta_text")
	if cta == "" {
		issues = append(issues, "Hero section missing CTA button (Rule H-4)")
	}

	// Rule H-5: Hero height ratio
	heightRatio := getFloat(section, "height_ratio")
	if heightRatio > 0 && heightRatio < 0.3 {
		issues = append(issues, fmt.Sprintf("Hero height ratio %.2f too small — minimum 30%% viewport (Rule H-5)", heightRatio))
	}

	// Rule H-6: Background
	bg := getStr(section, "background")
	if bg == "" || bg == "white" || bg == "#ffffff" || bg == "#FFFFFF" {
		warnings = append(warnings, "Hero has plain white background — consider gradient, image, or brand color (Rule H-6)")
	}

	score := 100 - len(issues)*20 - len(warnings)*5
	if score < 0 {
		score = 0
	}

	return CompositionResult{
		Score: score, Pass: len(issues) == 0,
		Issues: issues, Warnings: warnings,
	}
}

// ValidatePageRhythm checks section alternation and spacing.
func (f *Fundamentals) ValidatePageRhythm(sections []map[string]any) CompositionResult {
	var issues, warnings []string
	n := len(sections)

	if n < 3 {
		warnings = append(warnings, fmt.Sprintf("Only %d sections — most pages need 5-8 for good rhythm", n))
	}

	// Check alternating backgrounds
	consecutiveSame := 0
	lastBg := ""
	for i, s := range sections {
		bg := strings.ToLower(getStr(s, "background"))
		if bg == "" {
			bg = "white"
		}
		if bg == lastBg && i > 0 {
			consecutiveSame++
			if consecutiveSame >= 2 {
				issues = append(issues, fmt.Sprintf(
					"3+ consecutive sections with same background ('%s') starting at section %d — breaks visual rhythm",
					bg, i-1,
				))
			}
		} else {
			consecutiveSame = 0
		}
		lastBg = bg
	}

	// Check that hero is first
	if n > 0 {
		firstType := strings.ToLower(getStr(sections[0], "type"))
		if firstType != "" && firstType != "hero" && firstType != "cover" {
			warnings = append(warnings, fmt.Sprintf("First section is '%s' — expected hero/cover", firstType))
		}
	}

	// Check CTA presence near end
	if n >= 3 {
		lastSection := sections[n-1]
		lastType := strings.ToLower(getStr(lastSection, "type"))
		hasCTA := getStr(lastSection, "cta_text") != ""
		if !hasCTA && lastType != "cta" && lastType != "footer" {
			warnings = append(warnings, "Page doesn't end with CTA section — missed conversion opportunity")
		}
	}

	score := 100 - len(issues)*15 - len(warnings)*5
	if score < 0 {
		score = 0
	}

	return CompositionResult{
		Score: score, Pass: len(issues) == 0,
		Issues: issues, Warnings: warnings,
	}
}

// AuditComposition audits all sections for composition fundamentals.
func (f *Fundamentals) AuditComposition(sections []map[string]any, pageType string) CompositionResult {
	var allIssues, allWarnings []string

	// Page rhythm check
	rhythm := f.ValidatePageRhythm(sections)
	allIssues = append(allIssues, rhythm.Issues...)
	allWarnings = append(allWarnings, rhythm.Warnings...)

	// Per-section validation
	for _, s := range sections {
		sType := strings.ToLower(getStr(s, "type"))
		switch sType {
		case "hero", "cover":
			r := f.ValidateHero(s)
			allIssues = append(allIssues, r.Issues...)
			allWarnings = append(allWarnings, r.Warnings...)
		}
	}

	score := 100 - len(allIssues)*15 - len(allWarnings)*5
	if score < 0 {
		score = 0
	}

	return CompositionResult{
		Score: score, Pass: score >= 70 && len(allIssues) == 0,
		Issues: allIssues, Warnings: allWarnings,
	}
}

// CompositionPromptRules returns the composition rules for injection into AI prompts.
func (f *Fundamentals) CompositionPromptRules() string {
	return `COMPOSITION RULES (enforced programmatically — violations detected automatically):
- Hero section MUST have: headline (≤80 chars), subheadline, CTA button, ≥30% viewport height
- Hero MUST NOT have plain white background — use gradient, image, or brand color
- Page sections MUST alternate backgrounds (no 3+ consecutive same-bg sections)
- Page MUST end with CTA section for conversion
- Minimum 5 sections for full pages (hero, features, social proof, content, CTA)`
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}
