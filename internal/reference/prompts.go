package reference

import (
	"encoding/json"
	"fmt"
	"strings"
)

// referenceAnalysisPrompt is Pass 1 — matches PHP build_reference_analysis_prompt() exactly.
const referenceAnalysisPrompt = `You are a professional UI/UX analyst. Analyze this website screenshot and extract the UI structure.

## PAGE ANALYSIS
Identify the overall page characteristics:
- type: What kind of page is this? (marketing_landing, portfolio, blog, ecommerce, saas_landing, agency, restaurant, travel, fitness, real_estate, etc.)
- aspect_ratio: Viewport dimensions and scroll behavior
- background: Base background treatment (solid color with hex, gradient description, or image)
- primary_function: What is this page designed to accomplish?

## SECTION ANALYSIS
Identify each distinct horizontal section from top to bottom. For each section provide:
- name: Semantic name (header, hero, features, services, testimonials, team, pricing, cta, footer, about, gallery, stats, partners, faq, contact, portfolio, blog_posts, etc.)
- height_ratio: Approximate percentage of initial viewport height (e.g., "8%", "85%")
- background: Background treatment (color hex, gradient, image with overlay, pattern)
- components: List of UI components visible

## OUTPUT FORMAT
Respond with valid JSON only:

{
  "page": {
    "type": "string",
    "aspect_ratio": "string",
    "background": "string",
    "primary_function": "string"
  },
  "sections": [
    {
      "name": "string",
      "height_ratio": "string",
      "background": "string",
      "components": ["string"]
    }
  ]
}

Be thorough - identify ALL visible sections. Most marketing pages have 6-12 sections.
IMPORTANT: Count every distinct visual region. Include header, footer, and transitional sections. If you see fewer than 5 sections, look more carefully.`

// tokenExtractionPrompt is Pass 3 — matches PHP build_token_extraction_prompt() exactly.
// #nosec G101 -- design tokens are UI values, not credentials.
const tokenExtractionPrompt = `You are a professional UI/UX analyst. Analyze this website screenshot and extract design system tokens.

## COLOR TOKENS
Extract hex values where visible:
- background_primary, background_secondary
- text_primary, text_secondary, text_inverse
- accent_primary, accent_secondary

## TYPOGRAPHY TOKENS
- heading_font, body_font (describe: "serif display", "geometric sans", etc.)
- heading_weights: ["bold", "semibold", etc.]
- size_scale: ("1.25 major second", "1.333 perfect fourth", "1.5 perfect fifth", "1.618 golden ratio")

## SPACING TOKENS
- section_padding (small 40-60px, medium 60-80px, large 80-120px, xl 120-160px)
- container_width (narrow ~800px, medium ~1200px, wide ~1400px)
- component_gap (tight 12-16px, normal 24-32px, loose 40-48px)
- text_line_height (tight 1.2-1.3, normal 1.5-1.6, loose 1.7-1.8)

## SHAPE TOKENS
- button_radius, card_radius, image_radius (sharp 0, small 4-8px, medium 12-16px, large 20-24px, pill 50%)
- shadow_style (none, subtle, medium, dramatic)

## OUTPUT FORMAT
Respond with valid JSON only:

{
  "colors": {
    "background_primary": "string",
    "background_secondary": "string",
    "text_primary": "string",
    "text_secondary": "string",
    "text_inverse": "string",
    "accent_primary": "string",
    "accent_secondary": "string"
  },
  "typography": {
    "heading_font": "string",
    "body_font": "string",
    "heading_weights": ["string"],
    "size_scale": "string"
  },
  "spacing": {
    "section_padding": "string",
    "container_width": "string",
    "component_gap": "string",
    "text_line_height": "string"
  },
  "shapes": {
    "button_radius": "string",
    "card_radius": "string",
    "image_radius": "string",
    "shadow_style": "string"
  }
}

Be precise - use hex values for colors when clearly visible.`

// buildComponentPrompt is Pass 2 — matches PHP build_component_extraction_prompt().
func buildComponentPrompt(sections []Section) string {
	sectionsJSON, _ := json.MarshalIndent(sections, "", "  ")

	return fmt.Sprintf(`You are a professional UI/UX analyst. Analyze this website screenshot and extract detailed component information.

## REFERENCE SECTIONS
The following sections have been identified:
%s

## COMPONENT EXTRACTION
For EACH visible UI component, document:
1. section: Which section contains this component
2. type: Component type (nav_bar, button_primary, heading_h1, hero_image, feature_card, etc.)
3. position: Position within parent (top-left, center, bottom-right, full-width, etc.)
4. geometry: Shape and aspect ratio (pill 4:1, rounded-rect 3:2, square 1:1, circle, etc.)
5. visual_attributes: fill, border, shadow, radius
6. content_hint: What content this holds

## OUTPUT FORMAT
Respond with valid JSON only:

{
  "components": [
    {
      "section": "string",
      "type": "string",
      "position": "string",
      "geometry": "string",
      "visual_attributes": {
        "fill": "string",
        "border": "string",
        "shadow": "string",
        "radius": "string"
      },
      "content_hint": "string"
    }
  ]
}

Be comprehensive - identify at least 8 components. Hero sections typically have 3+ components.`, sectionsJSON)
}

// buildSpacingPrompt is Pass 4 — matches PHP build_spacing_extraction_prompt().
func buildSpacingPrompt(components []Component) string {
	// Summarize components instead of full JSON to save tokens
	var summary strings.Builder
	for _, c := range components {
		fmt.Fprintf(&summary, "- %s in %s (%s)\n", c.Type, c.Section, c.Position)
	}

	return fmt.Sprintf(`You are a professional UI/UX analyst. Extract the spacing and alignment system from this screenshot.

## REFERENCE COMPONENTS
%s

## EXTRACT
- Spacing scale (4px or 8px base unit)
- Section gaps, card padding, button padding
- Vertical rhythm, horizontal layout, alignment patterns

## OUTPUT FORMAT
Respond with valid JSON only:

{
  "base_unit": 8,
  "scale": [8, 16, 24, 32, 48, 64, 96],
  "detected": {
    "section_gap": 64,
    "card_padding": 24,
    "text_margin": 16,
    "button_padding": "12 24"
  },
  "vertical_rhythm": {
    "section_gap": "string",
    "heading_to_content": "string",
    "paragraph_gap": "string",
    "list_item_gap": "string"
  },
  "horizontal_layout": {
    "grid_columns": "string",
    "gutter_width": "string",
    "content_alignment": "string",
    "text_max_width": "string"
  },
  "container_padding": {
    "section_horizontal": "string",
    "card_padding": "string",
    "button_padding": "string",
    "input_padding": "string"
  },
  "alignment_patterns": {
    "nav_alignment": "string",
    "hero_alignment": "string",
    "section_title_alignment": "string",
    "cta_alignment": "string"
  }
}

Be specific with pixel estimates where possible.`, summary.String())
}
