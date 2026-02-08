package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/auth"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/credits"
	"github.com/philoveracity/uiai-engine/internal/design"
	"github.com/philoveracity/uiai-engine/internal/jsonutil"
	"github.com/philoveracity/uiai-engine/internal/ratelimit"
	"github.com/philoveracity/uiai-engine/internal/storage"
)

type critiqueHandler struct {
	cfg          *config.Config
	ai           *ai.Provider
	credits      *credits.Service
	limiter      *ratelimit.Limiter
	usage        *storage.UsageStore
	fundamentals *design.Fundamentals
}

type critiqueRequest struct {
	WebsiteURL    string         `json:"websiteUrl"`
	Model         string         `json:"model"`
	Provider      string         `json:"provider"`
	PageType      string         `json:"pageType"`
	ReferenceURL  string         `json:"referenceUrl"`
	IncludeMemory bool           `json:"includeMemory"`
	CritiqueMode  string         `json:"critiqueMode"`  // "mimic" or "public"
	DesignTokens  map[string]any `json:"designTokens"`  // Design system for fundamentals audit
	ImageBase64   string         `json:"imageBase64"`   // Screenshot for vision critique
	ImageType     string         `json:"imageType"`     // image/jpeg, image/png, image/webp
}

// UICrit dimensions — matches PHP UICRIT_DIMENSIONS constant exactly
var uiCritDimensions = []struct {
	Key   string
	Name  string
	Focus string
}{
	{"layout", "Layout", "Element positioning, alignment grids, whitespace distribution, visual flow."},
	{"color_contrast", "Color Contrast", "WCAG contrast ratios, text legibility, element boundaries, color harmony."},
	{"text_readability", "Text Readability", "Font sizes, line heights, paragraph widths, heading hierarchy."},
	{"button_usability", "Button Usability", "Button affordance, tap target sizes, hover states, CTA prominence."},
	{"learnability", "Learnability", "Icon clarity, navigation patterns, interaction cues, mental models."},
	{"design_cohesion", "Overall Design Cohesion", "Background textures/gradients, decorative icons, visual balance, contrast variety, design polish."},
	{"responsivity", "Responsivity / Adaptability", "Mobile-first design, breakpoint handling, touch targets, readable text at all sizes."},
}

func MountCritiqueReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	h := &critiqueHandler{
		cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage,
		fundamentals: &design.Fundamentals{},
	}

	r.Post("/", h.critique)
	r.Get("/models", h.models)
	r.Get("/dimensions", h.dimensions)
}

func (h *critiqueHandler) critique(w http.ResponseWriter, r *http.Request) {
	id := auth.FromContext(r.Context())
	if id == nil {
		writeJSON(w, 401, map[string]string{"error": "authentication required"})
		return
	}

	// Rate limit
	key := id.LicenseKey
	if key == "" {
		key = id.APIKey
	}
	if err := h.limiter.Check(key, id.Tier); err != nil {
		writeJSON(w, 429, map[string]string{"error": err.Error()})
		return
	}

	// Parse request
	var req critiqueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if req.WebsiteURL == "" && req.ImageBase64 == "" {
		writeJSON(w, 400, map[string]string{"error": "websiteUrl or imageBase64 is required"})
		return
	}
	if req.Model == "" {
		req.Model = h.cfg.AI.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = h.cfg.AI.DefaultProvider
	}
	if req.CritiqueMode == "" {
		req.CritiqueMode = "mimic"
	}

	// Build UICrit prompt
	prompt := h.buildUICritPrompt(req)

	// Inject Design Fundamentals violations as measured facts
	if req.DesignTokens != nil {
		fundamentalsNote := h.getFundamentalsViolations(req.DesignTokens)
		if fundamentalsNote != "" {
			prompt += "\n\n" + fundamentalsNote
		}
	}

	start := time.Now()

	// Build AI request — with or without vision
	aiReq := ai.Request{
		Provider:    req.Provider,
		Model:       req.Model,
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
	if req.ImageBase64 != "" {
		aiReq.ImageBase64 = req.ImageBase64
		aiReq.ImageType = req.ImageType
	}

	resp, err := h.ai.Complete(aiReq)
	if err != nil {
		log.Printf("[critique] AI error: %v", err)
		writeJSON(w, 502, map[string]string{"error": "AI provider error: " + err.Error()})
		return
	}

	// Parse and repair the LLM's JSON response
	parsed, parseErr := jsonutil.RepairObject(resp.Content)
	duration := time.Since(start)

	// Fire-and-forget credit deduction
	go h.credits.Deduct(id.LicenseID, "critique", "")

	// Record usage
	h.usage.Record(storage.UsageRecord{
		Type:         "critique",
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
		Status:       "success",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	})

	log.Printf("[critique] %s %s %d+%d tokens $%.4f %s parsed=%v",
		req.WebsiteURL, resp.Model, resp.InputTokens, resp.OutputTokens, resp.CostUSD,
		duration.Round(time.Millisecond), parseErr == nil)

	if parseErr != nil {
		// Fallback: return raw content with metadata so plugin can attempt its own repair
		writeJSON(w, 200, map[string]any{
			"content":      resp.Content,
			"model":        resp.Model,
			"model_used":   resp.Model,
			"inputTokens":  resp.InputTokens,
			"outputTokens": resp.OutputTokens,
			"costUSD":      resp.CostUSD,
			"duration_ms":  duration.Milliseconds(),
			"parse_error":  parseErr.Error(),
		})
		return
	}

	// Validate required structure
	critique, _ := parsed["critique"].(map[string]any)
	scores, _ := parsed["scores"].(map[string]any)
	priorityFixes, _ := parsed["priority_fixes"].([]any)
	summary, _ := parsed["summary"].(string)

	// Return the STRUCTURED response the plugin expects
	writeJSON(w, 200, map[string]any{
		// Plugin-expected fields (fixes uiai-790)
		"critique":      critique,
		"scores":        scores,
		"priority_fixes": priorityFixes,
		"summary":       summary,
		"success":       true,

		// Metadata
		"model":        resp.Model,
		"model_used":   resp.Model,
		"inputTokens":  resp.InputTokens,
		"outputTokens": resp.OutputTokens,
		"costUSD":      resp.CostUSD,
		"duration_ms":  duration.Milliseconds(),
	})
}

// buildUICritPrompt constructs the full UICrit 7-dimension critique prompt.
// This matches the PHP build_critique_prompt() method exactly.
func (h *critiqueHandler) buildUICritPrompt(req critiqueRequest) string {
	pageType := req.PageType
	if pageType == "" {
		pageType = "webpage"
	}

	contextNote := "MIMIC CRITIQUE MODE: full internal context allowed."
	if req.CritiqueMode == "public" {
		contextNote = "PUBLIC LEAD CRITIQUE MODE: visual-only. Provide concise summary and top 3-5 fixes."
	}

	var b strings.Builder
	b.Grow(8000)

	fmt.Fprintf(&b, `You are an EXTREMELY CRITICAL professional UI/UX design critic. Analyze this %s using the UICrit framework with BRUTAL standards.

%s

BE RUTHLESSLY HARSH. Most websites are amateur garbage that score 2-4. A score of 5 means "barely mediocre". Only exceptional professional designs score above 6.

YOUR DEFAULT ASSUMPTION: This design is probably a 3.0 until proven otherwise.

Evaluate these 7 dimensions, scoring each 0.0-10.0 (USE DECIMALS for precision):
`, pageType, contextNote)

	// Dimension prompts — abbreviated for token efficiency while keeping scoring rigor
	b.WriteString(`
## 1. LAYOUT (positioning, alignment, visual hierarchy, grouping, simplicity)
Assess: element positioning, grid alignment, padding/margin consistency, whitespace distribution, visual hierarchy clarity.

## 2. COLOR CONTRAST (WCAG accessibility)
Assess: cohesive palette (3-5 colors), text contrast ratios (4.5:1 normal, 3:1 large), element boundaries, color harmony.

## 3. TEXT READABILITY (font size, weight, line-height)
Assess: body ≥16px, line-height 1.4-1.6, heading hierarchy, paragraph max-width 65-75ch, font pairing (max 2-3).

## 4. BUTTON USABILITY (affordance, tap targets, clarity)
Assess: visual weight, touch targets ≥44x44px, styling consistency, CTA hierarchy, interactive affordance.

## 5. LEARNABILITY (icon clarity, navigation intuitiveness)
Assess: navigation clarity, icon meaning, information architecture, user flow logic, interaction cues.

## 6. DESIGN COHESION (overall polish — THE CRITICAL DIFFERENTIATOR)
Score caps: plain white bg = 2.0-2.5 MAX. No decorative elements = 2.5-3.0 MAX. Generic template = 3.0-3.5 MAX.
Above 5 requires: intentional backgrounds, custom icons, visual rhythm, consistent design language.

## 7. RESPONSIVITY / ADAPTABILITY (60% of traffic is mobile!)
Score caps: fixed-width = 1.0-2.0. Desktop-only = 2.0-3.0. No mobile nav = 3.0-4.0.
`)

	b.WriteString(`
SCORING: 0-1.9 broken | 2-2.9 severely flawed | 3-3.9 poor amateur | 4-4.9 below average | 5-5.9 mediocre | 6-6.9 acceptable | 7-7.9 good | 8-8.9 very good | 9-10 exceptional

OVERALL_QUALITY formula: ((responsivity*4) + (design_cohesion*4) + (layout*2) + (text_readability*2) + (button_usability*2) + color_contrast + learnability) / 16
Caps: if responsivity < 4 OR design_cohesion < 4 OR ANY dimension < 3 → overall_quality ≤ 3.5

Return valid JSON:
{
  "critique": {
    "layout": { "score": <0.0-10.0>, "findings": [{"element":"...","expected":"...","gap":"...","remediation":"..."}] },
    "color_contrast": { "score": <0.0-10.0>, "findings": [...] },
    "text_readability": { "score": <0.0-10.0>, "findings": [...] },
    "button_usability": { "score": <0.0-10.0>, "findings": [...] },
    "learnability": { "score": <0.0-10.0>, "findings": [...] },
    "design_cohesion": { "score": <0.0-10.0>, "findings": [...] },
    "responsivity": { "score": <0.0-10.0>, "findings": [...] }
  },
  "scores": {
    "aesthetics": <0.0-10.0>,
    "learnability": <0.0-10.0>,
    "efficiency": <0.0-10.0>,
    "design_cohesion": <0.0-10.0>,
    "responsivity": <0.0-10.0>,
    "overall_quality": <0.0-10.0>
  },
  "priority_fixes": [
    {"priority":1,"action":"...","description":"..."},
    {"priority":2,"action":"...","description":"..."},
    {"priority":3,"action":"...","description":"..."}
  ],
  "summary": "<2-3 sentence brutally honest assessment>"
}

Use DECIMAL scores. Be SPECIFIC about elements. Include 3-5 findings per dimension. NO GENEROSITY.
`)

	if req.ReferenceURL != "" {
		fmt.Fprintf(&b, "\nNote: This page is attempting to mimic the structure of %s. Consider alignment with that reference.\n", req.ReferenceURL)
	}

	if req.WebsiteURL != "" && req.ImageBase64 == "" {
		fmt.Fprintf(&b, "\nAnalyze the website at: %s\n", req.WebsiteURL)
	}

	return b.String()
}

// getFundamentalsViolations runs the Design Fundamentals audit on the provided
// design tokens and returns a prompt fragment with measured violations as FACTS.
// This matches the PHP get_fundamentals_violations() method.
func (h *critiqueHandler) getFundamentalsViolations(designTokens map[string]any) string {
	if len(designTokens) == 0 {
		return ""
	}

	audit := h.fundamentals.AuditDesignSystem(designTokens)

	if len(audit.Issues) == 0 && len(audit.Warnings) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("PROGRAMMATIC DESIGN FUNDAMENTALS AUDIT (measured values — these are FACTS):\n")
	fmt.Fprintf(&b, "Audit score: %d/100\n\n", audit.Score)

	if len(audit.Issues) > 0 {
		b.WriteString("CRITICAL VIOLATIONS (must reduce color_contrast and button_usability scores):\n")
		for _, issue := range audit.Issues {
			fmt.Fprintf(&b, "  ❌ %s\n", issue)
		}
	}
	if len(audit.Warnings) > 0 {
		b.WriteString("\nWARNINGS:\n")
		for _, w := range audit.Warnings {
			fmt.Fprintf(&b, "  ⚠️ %s\n", w)
		}
	}

	b.WriteString("\nThese are MEASURED values from the actual design system. Score the color_contrast and button_usability dimensions accordingly.")
	return b.String()
}

func (h *critiqueHandler) models(w http.ResponseWriter, r *http.Request) {
	models := []map[string]string{}
	seen := map[string]bool{}

	// Default model first
	if dm := h.cfg.AI.DefaultModel; dm != "" {
		models = append(models, map[string]string{
			"id": dm, "name": dm, "provider": h.cfg.AI.DefaultProvider, "default": "true",
		})
		seen[dm] = true
	}

	// Known models
	known := []struct{ id, name, provider string }{
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", "anthropic"},
		{"claude-opus-4-20250514", "Claude Opus 4", "anthropic"},
		{"anthropic/claude-sonnet-4", "Claude Sonnet 4 (OR)", "openrouter"},
		{"gpt-4o", "GPT-4o", "openai"},
		{"gpt-4o-mini", "GPT-4o Mini", "openai"},
	}
	for _, m := range known {
		if !seen[m.id] {
			models = append(models, map[string]string{
				"id": m.id, "name": m.name, "provider": m.provider,
			})
		}
	}
	writeJSON(w, 200, models)
}

func (h *critiqueHandler) dimensions(w http.ResponseWriter, r *http.Request) {
	dims := make([]map[string]string, len(uiCritDimensions))
	for i, d := range uiCritDimensions {
		dims[i] = map[string]string{
			"key":   d.Key,
			"name":  d.Name,
			"focus": d.Focus,
		}
	}
	writeJSON(w, 200, dims)
}
