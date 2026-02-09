package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/auth"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/credits"
	"github.com/philoveracity/uiai-engine/internal/ratelimit"
	"github.com/philoveracity/uiai-engine/internal/storage"
)

type pipelineDeps struct {
	cfg     *config.Config
	ai      *ai.Provider
	credits *credits.Service
	limiter *ratelimit.Limiter
	usage   *storage.UsageStore
}

// MountDesignSystemRoute handles /api/design-system
func MountDesignSystemRoute(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &pipelineDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		var body struct {
			Sources  map[string]any `json:"sources"`
			Model    string         `json:"model"`
			Provider string         `json:"provider"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Model == "" {
			body.Model = d.cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = d.cfg.AI.DefaultProvider
		}

		prompt := `Generate a comprehensive design system from the provided sources. Return JSON:
{
  "colors": {"primary": "#hex", "secondary": "#hex", "accent": "#hex", "neutrals": ["#hex"]},
  "typography": {"headings": {"family": "", "weights": [], "scale": []}, "body": {"family": "", "size": "", "line_height": ""}},
  "spacing": {"base": 0, "scale": [], "unit": "px"},
  "borders": {"radius": [], "widths": []},
  "shadows": [],
  "breakpoints": {}
}

Sources: ` + toJSON(body.Sources)

		start := time.Now()
		resp, err := d.ai.Complete(req.Context(), ai.Request{
			Provider: body.Provider, Model: body.Model,
			Prompt: prompt, MaxTokens: 4096, Temperature: 0.5,
		})
		if err != nil {
			log.Printf("[design-system] AI error: %v", err)
			writeJSON(w, 502, map[string]string{"error": "AI error: " + err.Error()})
			return
		}

		go d.credits.Deduct(id.LicenseID, "design_system", "")
		d.usage.Record(storage.UsageRecord{
			Type: "design_system", Model: resp.Model,
			InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
			CostUSD: resp.CostUSD, Status: "success",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})

		writeJSON(w, 200, map[string]any{
			"design_system": resp.Content,
			"model": resp.Model, "inputTokens": resp.InputTokens,
			"outputTokens": resp.OutputTokens, "costUSD": resp.CostUSD,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}

// MountContentMapRoute handles /api/content-map
func MountContentMapRoute(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &pipelineDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		var body struct {
			Pages      []map[string]any `json:"pages"`
			Blueprint  map[string]any   `json:"blueprint"`
			RunContext map[string]any   `json:"run_context"`
			Model      string           `json:"model"`
			Provider   string           `json:"provider"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Model == "" {
			body.Model = d.cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = d.cfg.AI.DefaultProvider
		}

		prompt := `Map content for these pages based on the blueprint. For each page, generate section content (headlines, body copy, CTAs). Return JSON:
{
  "pages": [{"slug": "", "title": "", "sections": [{"type": "", "headline": "", "body": "", "cta": ""}]}]
}

Pages: ` + toJSON(body.Pages) + `
Blueprint: ` + toJSON(body.Blueprint)

		start := time.Now()
		resp, err := d.ai.Complete(req.Context(), ai.Request{
			Provider: body.Provider, Model: body.Model,
			Prompt: prompt, MaxTokens: 8192, Temperature: 0.7,
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "AI error: " + err.Error()})
			return
		}

		go d.credits.Deduct(id.LicenseID, "content_map", "")
		d.usage.Record(storage.UsageRecord{
			Type: "content_map", Model: resp.Model,
			InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
			CostUSD: resp.CostUSD, Status: "success",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})

		writeJSON(w, 200, map[string]any{
			"content_map": resp.Content,
			"model": resp.Model, "inputTokens": resp.InputTokens,
			"outputTokens": resp.OutputTokens, "costUSD": resp.CostUSD,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}

// MountBlockRecipesRoute handles /api/block-recipes
func MountBlockRecipesRoute(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &pipelineDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		var body struct {
			Section      map[string]any `json:"section"`
			DesignSystem map[string]any `json:"design_system"`
			ContentMap   map[string]any `json:"content_map"`
			Model        string         `json:"model"`
			Provider     string         `json:"provider"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Model == "" {
			body.Model = d.cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = d.cfg.AI.DefaultProvider
		}

		prompt := `Generate WordPress block editor (Gutenberg) markup for this section. Use the design system tokens and content map. Return JSON:
{
  "blocks": "<!-- wp:group --> ... <!-- /wp:group -->",
  "custom_css": "/* section-specific CSS */",
  "block_count": 0,
  "patterns_used": []
}

Section: ` + toJSON(body.Section) + `
Design System: ` + toJSON(body.DesignSystem) + `
Content: ` + toJSON(body.ContentMap)

		start := time.Now()
		resp, err := d.ai.Complete(req.Context(), ai.Request{
			Provider: body.Provider, Model: body.Model,
			Prompt: prompt, MaxTokens: 8192, Temperature: 0.5,
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "AI error: " + err.Error()})
			return
		}

		go d.credits.Deduct(id.LicenseID, "block_recipes", "")
		d.usage.Record(storage.UsageRecord{
			Type: "block_recipes", Model: resp.Model,
			InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
			CostUSD: resp.CostUSD, Status: "success",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})

		writeJSON(w, 200, map[string]any{
			"blocks": resp.Content,
			"model": resp.Model, "inputTokens": resp.InputTokens,
			"outputTokens": resp.OutputTokens, "costUSD": resp.CostUSD,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}

// MountComparisonRoute handles /api/comparison (5-way)
func MountComparisonRoute(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &pipelineDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		var body struct {
			BuiltPageURL    string         `json:"built_page_url"`
			BuiltImageB64   string         `json:"built_image_base64"`
			DesignTokens    map[string]any `json:"design_tokens"`
			ReferenceSections []any        `json:"reference_sections"`
			ReferenceComponents []any      `json:"reference_components"`
			Model           string         `json:"model"`
			Provider        string         `json:"provider"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Model == "" {
			body.Model = d.cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = d.cfg.AI.DefaultProvider
		}

		prompt := `Perform a 5-way compliance comparison of this built page against the reference design tokens, sections, and components. Score each dimension 0-100. Return JSON:
{
  "token_compliance": {"score": 0, "color_match": 0, "typography_match": 0, "spacing_match": 0, "issues": []},
  "section_compliance": {"score": 0, "matched": 0, "missing": 0, "extra": 0, "issues": []},
  "component_compliance": {"score": 0, "matched": 0, "missing": 0, "issues": []},
  "layout_compliance": {"score": 0, "issues": []},
  "content_compliance": {"score": 0, "issues": []},
  "overall_score": 0,
  "priority_fixes": []
}

Design Tokens: ` + toJSON(body.DesignTokens) + `
Sections: ` + toJSON(body.ReferenceSections) + `
Components: ` + toJSON(body.ReferenceComponents)

		start := time.Now()
		resp, err := d.ai.Complete(req.Context(), ai.Request{
			Provider: body.Provider, Model: body.Model,
			Prompt: prompt, MaxTokens: 4096, Temperature: 0.3,
			ImageBase64: body.BuiltImageB64,
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "AI error: " + err.Error()})
			return
		}

		go d.credits.Deduct(id.LicenseID, "five_way_comparison", "")
		d.usage.Record(storage.UsageRecord{
			Type: "five_way_comparison", Model: resp.Model,
			InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
			CostUSD: resp.CostUSD, Status: "success",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})

		writeJSON(w, 200, map[string]any{
			"comparison": resp.Content,
			"model": resp.Model, "inputTokens": resp.InputTokens,
			"outputTokens": resp.OutputTokens, "costUSD": resp.CostUSD,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}
