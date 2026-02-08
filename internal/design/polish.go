// polish.go — Port of class-polish-fundamentals.php (605 LOC)
// Validates visual polish: decorative elements, section dividers, micro-interactions.
// Note: SVG icon library + animation CSS are build-time assets, not ported to API.
// This ports the AUDIT and PROMPT RULES logic only.
package design

import (
	"fmt"
	"strings"
)

// PolishResult describes a polish audit.
type PolishResult struct {
	Score    int      `json:"score"`
	Pass     bool     `json:"pass"`
	Issues   []string `json:"issues"`
	Warnings []string `json:"warnings"`
}

// AuditPolish audits page content for visual polish fundamentals.
// content is the raw HTML content string.
func (f *Fundamentals) AuditPolish(content string, pageType string) PolishResult {
	var issues, warnings []string
	lower := strings.ToLower(content)

	// Rule P-1: Check for decorative icons (SVG or icon classes)
	hasIcons := strings.Contains(lower, "<svg") ||
		strings.Contains(lower, "dashicons") ||
		strings.Contains(lower, "fa-") ||
		strings.Contains(lower, "icon-") ||
		strings.Contains(lower, "lucide")
	if !hasIcons {
		issues = append(issues, "No decorative icons found — pages need visual anchors (Rule P-1)")
	}

	// Rule P-2: Check for section dividers / separators
	hasDividers := strings.Contains(lower, "divider") ||
		strings.Contains(lower, "separator") ||
		strings.Contains(lower, "wave") ||
		strings.Contains(lower, "section-transition")
	if !hasDividers {
		warnings = append(warnings, "No section dividers/transitions — consider SVG wave or angled separators (Rule P-2)")
	}

	// Rule P-3: Check for background variety
	bgCount := strings.Count(lower, "background")
	gradientCount := strings.Count(lower, "gradient")
	if bgCount < 2 && gradientCount == 0 {
		issues = append(issues, "Minimal background variety — use alternating backgrounds, gradients, or patterns (Rule P-3)")
	}

	// Rule P-4: Check for visual rhythm elements
	hasCards := strings.Contains(lower, "card") || strings.Contains(lower, "wp-block-column")
	hasGrid := strings.Contains(lower, "grid") || strings.Contains(lower, "columns")
	if !hasCards && !hasGrid {
		warnings = append(warnings, "No card/grid layouts found — consider structured content presentation (Rule P-4)")
	}

	// Rule P-5: Check for hover/interactive styling
	hasHover := strings.Contains(lower, ":hover") ||
		strings.Contains(lower, "transition") ||
		strings.Contains(lower, "transform")
	if !hasHover {
		warnings = append(warnings, "No hover/transition effects detected — add micro-interactions for polish (Rule P-5)")
	}

	// Rule P-6: Check for shadow usage
	hasShadows := strings.Contains(lower, "box-shadow") || strings.Contains(lower, "shadow")
	if !hasShadows {
		warnings = append(warnings, "No shadows detected — layered shadows add depth and professionalism (Rule P-6)")
	}

	// Rule P-7: Check image treatment
	hasImages := strings.Contains(lower, "<img") || strings.Contains(lower, "wp-image")
	if pageType != "contact" && !hasImages {
		warnings = append(warnings, "No images detected — visual content increases engagement (Rule P-7)")
	}

	score := 100 - len(issues)*15 - len(warnings)*5
	if score < 0 {
		score = 0
	}

	return PolishResult{
		Score: score, Pass: score >= 70 && len(issues) == 0,
		Issues: issues, Warnings: warnings,
	}
}

// PolishPromptRules returns the polish rules for injection into AI prompts.
func (f *Fundamentals) PolishPromptRules() string {
	return `VISUAL POLISH RULES (programmatic checks applied post-build):
- Every page MUST have decorative icons (SVG preferred — Lucide icon set)
- Sections SHOULD use dividers/transitions (SVG waves, angled separators)
- Background variety required — no all-white pages. Use gradients, patterns, brand colors.
- Card and grid layouts for structured content presentation
- Hover effects and transitions on interactive elements (300ms ease timing)
- Layered shadows on cards (hue-matched, not pure black) — sm/md/lg elevation system
- Images on all non-utility pages (hero images, illustrations, photos)
- Border-radius consistency: use 8px/12px/16px scale, never mix`
}

// IAResult describes an information architecture audit.
type IAResult struct {
	Score    int      `json:"score"`
	Pass     bool     `json:"pass"`
	Issues   []string `json:"issues"`
	Warnings []string `json:"warnings"`
}

// ValidateNavigation checks navigation items for IA fundamentals.
func (f *Fundamentals) ValidateNavigation(navItems []string) IAResult {
	var issues, warnings []string

	n := len(navItems)
	if n == 0 {
		issues = append(issues, "No navigation items — every site needs primary navigation")
	} else if n > 7 {
		issues = append(issues, fmt.Sprintf("Navigation has %d items — maximum 7 for cognitive load (Miller's law)", n))
	} else if n < 3 {
		warnings = append(warnings, fmt.Sprintf("Only %d nav items — most sites need 4-6", n))
	}

	// Check for Home link
	hasHome := false
	for _, item := range navItems {
		lower := strings.ToLower(item)
		if lower == "home" || lower == "/" {
			hasHome = true
			break
		}
	}
	if !hasHome && n > 0 {
		warnings = append(warnings, "No explicit 'Home' link — consider adding for clarity")
	}

	// Check for CTA in nav
	hasCTA := false
	ctaWords := []string{"contact", "get started", "sign up", "try", "demo", "book", "schedule"}
	for _, item := range navItems {
		lower := strings.ToLower(item)
		for _, cta := range ctaWords {
			if strings.Contains(lower, cta) {
				hasCTA = true
				break
			}
		}
	}
	if !hasCTA && n > 0 {
		warnings = append(warnings, "No CTA in navigation — consider adding 'Contact' or 'Get Started'")
	}

	score := 100 - len(issues)*20 - len(warnings)*5
	if score < 0 {
		score = 0
	}
	return IAResult{Score: score, Pass: len(issues) == 0, Issues: issues, Warnings: warnings}
}

// ValidateSitemap checks a list of page names for IA fundamentals.
func (f *Fundamentals) ValidateSitemap(pages []string) IAResult {
	var issues, warnings []string

	n := len(pages)
	if n == 0 {
		issues = append(issues, "No pages defined")
		return IAResult{Score: 0, Pass: false, Issues: issues, Warnings: warnings}
	}

	// Required pages
	hasHome := false
	hasAbout := false
	hasContact := false
	for _, p := range pages {
		lower := strings.ToLower(p)
		if lower == "home" || lower == "homepage" {
			hasHome = true
		}
		if lower == "about" || lower == "about us" {
			hasAbout = true
		}
		if lower == "contact" || lower == "contact us" {
			hasContact = true
		}
	}

	if !hasHome {
		issues = append(issues, "Missing Home page — every site needs a landing page")
	}
	if !hasAbout {
		warnings = append(warnings, "Missing About page — builds trust and credibility")
	}
	if !hasContact {
		warnings = append(warnings, "Missing Contact page — essential for conversions")
	}

	if n > 15 {
		warnings = append(warnings, fmt.Sprintf("%d pages may overwhelm navigation — consider grouping", n))
	}

	score := 100 - len(issues)*20 - len(warnings)*5
	if score < 0 {
		score = 0
	}
	return IAResult{Score: score, Pass: len(issues) == 0, Issues: issues, Warnings: warnings}
}

// IAPromptRules returns the IA rules for injection into AI prompts.
func (f *Fundamentals) IAPromptRules() string {
	return `INFORMATION ARCHITECTURE RULES:
- Primary navigation: 4-7 items maximum (Miller's law)
- Must include CTA in navigation (Contact, Get Started, etc.)
- Required pages: Home, About (or equivalent), Contact
- Page depth: max 2 clicks from homepage to any page
- Consistent naming: use user-facing language, not internal jargon`
}
