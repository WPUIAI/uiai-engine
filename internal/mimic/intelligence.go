// Package mimic implements the Mimic Intelligence engine.
//
// Ports of:
//   - class-mimic-intelligence.php (1147 LOC) — plan generation + blueprint
//   - class-intake-ai-inference.php (1024 LOC) — AI enrichment of intake data
//
// The Go engine ports the INTELLIGENCE layer:
//  1. Archetype inference (slug → home/about/listing/detail)
//  2. Page inference from patterns (patterns → page structure)
//  3. Blueprint generation (plan → markdown document)
//  4. AI plan enrichment prompts (intake data → enhanced plan)
//  5. Content suggestions (page type + intake → section content)
//
// The WordPress-specific orchestration (intake DB queries, Google Drive API,
// website analyzer calls) remains in the PHP plugin. The plugin calls the
// Go engine's /api/intake endpoints to enrich plans with AI.
package mimic

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Plan represents a site build plan.
type Plan struct {
	Success           bool              `json:"success"`
	ReferenceURLs     []string          `json:"reference_urls"`
	Sources           []string          `json:"sources"`
	Business          Business          `json:"business"`
	Pages             []Page            `json:"pages"`
	Patterns          []string          `json:"patterns"`
	Plugins           []Plugin          `json:"plugins"`
	Config            map[string]any    `json:"config"`
	Branding          Branding          `json:"branding"`
	SocialLinks       map[string]string `json:"social_links"`
	ContentMapping    map[string]any    `json:"content_mapping"`
	Navigation        Navigation        `json:"navigation,omitempty"`
	ReferenceAnalysis map[string]any    `json:"reference_analysis,omitempty"`
	DesignSystem      map[string]any    `json:"design_system,omitempty"`
	Confidence        Confidence        `json:"confidence"`
}

type Business struct {
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type Page struct {
	Title     string   `json:"title"`
	Slug      string   `json:"slug"`
	Archetype string   `json:"archetype"`
	Patterns  []string `json:"patterns"`
	Source    string   `json:"source"`
}

type Plugin struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Source string `json:"source"`
}

type Branding struct {
	Colors      []string `json:"colors,omitempty"`
	ColorSource string   `json:"color_source,omitempty"`
	StyleSource string   `json:"style_source,omitempty"`
	Style       string   `json:"style,omitempty"`
}

type Navigation struct {
	PrimaryNav    []NavItem      `json:"primary_nav,omitempty"`
	FooterMenus   []FooterMenu   `json:"footer_menus,omitempty"`
	FooterContent map[string]any `json:"footer_content,omitempty"`
	CTABar        *CTABar        `json:"cta_bar,omitempty"`
}

type NavItem struct {
	Label string `json:"label"`
	Slug  string `json:"slug"`
	Order int    `json:"order"`
	Notes string `json:"notes,omitempty"`
}

type FooterMenu struct {
	Title string    `json:"title"`
	Items []NavItem `json:"items"`
}

type CTABar struct {
	Text     string `json:"text"`
	Target   string `json:"target"`
	Position string `json:"position"`
}

type Confidence struct {
	Score   int    `json:"score"`
	Sources string `json:"sources"`
}

// ═══════════════════════════════════════════════════════════
// ARCHETYPE INFERENCE
// ═══════════════════════════════════════════════════════════

// ArchetypeMap maps slugs to archetypes.
var ArchetypeMap = map[string]string{
	"home":         "home",
	"about":        "about",
	"services":     "listing",
	"products":     "listing",
	"blog":         "listing",
	"team":         "listing",
	"portfolio":    "listing",
	"contact":      "detail",
	"faq":          "detail",
	"pricing":      "listing",
	"features":     "listing",
	"gallery":      "listing",
	"careers":      "listing",
	"news":         "listing",
	"events":       "listing",
	"case-studies": "listing",
	"testimonials": "listing",
	"privacy":      "detail",
	"terms":        "detail",
}

// InferArchetype returns the archetype for a page slug.
func InferArchetype(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if arch, ok := ArchetypeMap[slug]; ok {
		return arch
	}
	return "detail"
}

// ═══════════════════════════════════════════════════════════
// PAGE INFERENCE FROM PATTERNS
// ═══════════════════════════════════════════════════════════

// InferPages creates a page structure from detected patterns.
func InferPages(patterns []string, archetype string) []Page {
	pages := []Page{
		{Title: "Home", Slug: "home", Archetype: "home", Patterns: patterns, Source: "reference"},
	}

	// Infer additional pages from patterns
	patternToPage := map[string]Page{
		"pricing":   {Title: "Pricing", Slug: "pricing", Archetype: "listing"},
		"contact":   {Title: "Contact", Slug: "contact", Archetype: "detail"},
		"faq":       {Title: "FAQ", Slug: "faq", Archetype: "detail"},
		"team-grid": {Title: "Team", Slug: "team", Archetype: "listing"},
		"gallery":   {Title: "Gallery", Slug: "gallery", Archetype: "listing"},
	}

	seen := map[string]bool{"home": true}
	for _, p := range patterns {
		if page, ok := patternToPage[p]; ok && !seen[page.Slug] {
			page.Patterns = []string{p}
			page.Source = "inferred"
			pages = append(pages, page)
			seen[page.Slug] = true
		}
	}

	return pages
}

// ═══════════════════════════════════════════════════════════
// BLUEPRINT GENERATION
// ═══════════════════════════════════════════════════════════

// GenerateBlueprint creates a markdown Blueprint document from a plan.
func GenerateBlueprint(plan Plan) string {
	var b strings.Builder
	b.Grow(4000)

	fmt.Fprintf(&b, "# Blueprint - Site Planning Document\n\n")
	fmt.Fprintf(&b, "**Generated:** %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**Confidence:** %d%% (sources: %s)\n\n", plan.Confidence.Score, strings.Join(plan.Sources, ", "))

	// Reference Analysis
	fmt.Fprintf(&b, "## Reference Analysis\n")
	fmt.Fprintf(&b, "**URLs:** %s\n\n", strings.Join(plan.ReferenceURLs, ", "))

	// Business Info
	if plan.Business.Name != "" {
		fmt.Fprintf(&b, "## Business\n")
		fmt.Fprintf(&b, "- **Name:** %s\n", plan.Business.Name)
		if plan.Business.Phone != "" {
			fmt.Fprintf(&b, "- **Phone:** %s\n", plan.Business.Phone)
		}
		if plan.Business.Email != "" {
			fmt.Fprintf(&b, "- **Email:** %s\n", plan.Business.Email)
		}
		if plan.Business.Type != "" {
			fmt.Fprintf(&b, "- **Type:** %s\n", plan.Business.Type)
		}
		b.WriteString("\n")
	}

	// Plugins
	if len(plan.Plugins) > 0 {
		fmt.Fprintf(&b, "## Required Plugins\n")
		fmt.Fprintf(&b, "| Plugin | Slug | Source |\n")
		fmt.Fprintf(&b, "|--------|------|--------|\n")
		for _, p := range plan.Plugins {
			fmt.Fprintf(&b, "| %s | `%s` | %s |\n", p.Name, p.Slug, p.Source)
		}
		b.WriteString("\n")
	}

	// Site Structure
	fmt.Fprintf(&b, "## Site Structure\n")
	for _, page := range plan.Pages {
		sourceBadge := ""
		if page.Source != "reference" {
			sourceBadge = fmt.Sprintf(" _(%s)_", page.Source)
		}
		fmt.Fprintf(&b, "- **%s** (`/%s`) - %s%s\n", page.Title, page.Slug, page.Archetype, sourceBadge)
	}
	b.WriteString("\n")

	// Page Layouts
	fmt.Fprintf(&b, "## Page Layouts\n")
	fmt.Fprintf(&b, "| Page | Archetype | Patterns |\n")
	fmt.Fprintf(&b, "|------|-----------|----------|\n")
	for _, page := range plan.Pages {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", page.Title, page.Archetype, strings.Join(page.Patterns, ", "))
	}
	b.WriteString("\n")

	// Branding
	fmt.Fprintf(&b, "## Branding\n")
	if len(plan.Branding.Colors) > 0 {
		fmt.Fprintf(&b, "### Colors (from %s)\n", plan.Branding.ColorSource)
		for i, c := range plan.Branding.Colors {
			fmt.Fprintf(&b, "- Color %d: `%s`\n", i+1, c)
		}
	}
	if plan.Branding.Style != "" {
		fmt.Fprintf(&b, "### Style: %s\n", plan.Branding.Style)
	}
	b.WriteString("\n")

	// Social Links
	if len(plan.SocialLinks) > 0 {
		fmt.Fprintf(&b, "## Social Links\n")
		for platform, url := range plan.SocialLinks {
			fmt.Fprintf(&b, "- **%s:** %s\n", strings.Title(platform), url)
		}
		b.WriteString("\n")
	}

	// Navigation
	b.WriteString("## Navigation Architecture\n\n")
	b.WriteString("Step 18 MUST implement everything below.\n\n")

	if len(plan.Navigation.PrimaryNav) > 0 {
		b.WriteString("### Primary Navigation (Header)\n")
		b.WriteString("| Order | Label | Target | Notes |\n")
		b.WriteString("|-------|-------|--------|-------|\n")
		for _, item := range plan.Navigation.PrimaryNav {
			fmt.Fprintf(&b, "| %d | %s | `%s` | %s |\n", item.Order, item.Label, item.Slug, item.Notes)
		}
	} else {
		b.WriteString("### Primary Navigation\n")
		b.WriteString("**TODO:** Builder must infer from Site Structure above.\n")
	}
	b.WriteString("\n")

	return b.String()
}

// ═══════════════════════════════════════════════════════════
// AI ENRICHMENT PROMPTS
// ═══════════════════════════════════════════════════════════

// BuildEnrichmentPrompt creates the AI prompt for plan enrichment.
func BuildEnrichmentPrompt(plan Plan) string {
	planJSON, _ := json.MarshalIndent(plan, "", "  ")

	return fmt.Sprintf(`You are a professional website architect. Analyze this site build plan and enhance it.

## CURRENT PLAN
%s

## ENHANCE WITH
1. **Navigation Architecture**: Design primary nav and footer menus based on pages and business type
2. **Content Strategy**: Suggest hero headlines, CTA text, and section copy based on business type
3. **Page Flow**: Ensure logical user journey from homepage through conversion
4. **Missing Pages**: Identify any critical pages missing (About, Contact, etc.)
5. **Plugin Recommendations**: Suggest WordPress plugins needed for the planned features

## OUTPUT FORMAT
Return valid JSON with these additions to the plan:
{
  "navigation": {
    "primary_nav": [{"label": "...", "slug": "...", "order": 1}],
    "footer_menus": [{"title": "...", "items": [{"label": "...", "slug": "..."}]}],
    "footer_content": {"copyright": "...", "tagline": "..."}
  },
  "content_suggestions": {
    "hero_headline": "...",
    "hero_subheadline": "...",
    "cta_primary": "...",
    "value_proposition": "..."
  },
  "missing_pages": [{"title": "...", "slug": "...", "reason": "..."}],
  "plugin_suggestions": [{"name": "...", "slug": "...", "reason": "..."}]
}`, string(planJSON))
}

// BuildContentSuggestionsPrompt creates the AI prompt for content suggestions.
func BuildContentSuggestionsPrompt(pageType string, businessType string, patterns []string) string {
	return fmt.Sprintf(`You are a professional copywriter. Generate content suggestions for a %s page of a %s business.

## SECTION PATTERNS
The page will use these section patterns: %s

## FOR EACH PATTERN, PROVIDE
- headline: Main section heading (5-8 words)
- subheadline: Supporting text (15-25 words)  
- cta_text: Call-to-action button text (2-4 words)
- content_notes: Brief notes on content to include

## OUTPUT FORMAT
Return valid JSON:
{
  "sections": {
    "<pattern_name>": {
      "headline": "...",
      "subheadline": "...",
      "cta_text": "...",
      "content_notes": "..."
    }
  }
}`, pageType, businessType, strings.Join(patterns, ", "))
}

// BuildInferencePrompt creates the AI prompt for intake data inference.
func BuildInferencePrompt(intakeData map[string]any) string {
	dataJSON, _ := json.MarshalIndent(intakeData, "", "  ")

	return fmt.Sprintf(`You are a business analyst. Analyze this client intake data and infer missing information.

## INTAKE DATA
%s

## INFER
1. Business type and industry if not explicit
2. Target audience demographics
3. Key differentiators / value propositions
4. Recommended page structure
5. Color palette suggestions based on industry
6. Font style suggestions based on brand voice

## OUTPUT FORMAT
Return valid JSON:
{
  "business_type": "...",
  "industry": "...",
  "target_audience": "...",
  "value_propositions": ["..."],
  "recommended_pages": ["home", "about", "services", "contact"],
  "color_suggestions": ["#hex1", "#hex2", "#hex3"],
  "font_style": "...",
  "confidence": 0.0-1.0
}`, string(dataJSON))
}
