package routes

import (
	"encoding/json"
	"fmt"
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

type critiqueHandler struct {
	cfg     *config.Config
	ai      *ai.Provider
	credits *credits.Service
	limiter *ratelimit.Limiter
	usage   *storage.UsageStore
}

type critiqueRequest struct {
	WebsiteURL    string `json:"websiteUrl"`
	Model         string `json:"model"`
	Provider      string `json:"provider"`
	PageType      string `json:"pageType"`
	ReferenceURL  string `json:"referenceUrl"`
	IncludeMemory bool   `json:"includeMemory"`
}

func MountCritiqueReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	h := &critiqueHandler{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

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
	if req.WebsiteURL == "" {
		writeJSON(w, 400, map[string]string{"error": "websiteUrl is required"})
		return
	}
	if req.Model == "" {
		req.Model = h.cfg.AI.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = "openrouter" // matches Bun default for critique
	}

	// Build critique prompt (matches Bun's critiqueRouter logic)
	prompt := buildCritiquePrompt(req)

	start := time.Now()
	resp, err := h.ai.Complete(ai.Request{
		Provider:    req.Provider,
		Model:       req.Model,
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		log.Printf("[critique] AI error: %v", err)
		writeJSON(w, 502, map[string]string{"error": "AI provider error: " + err.Error()})
		return
	}

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

	duration := time.Since(start)
	log.Printf("[critique] %s %s %d+%d tokens $%.4f %s", req.WebsiteURL, resp.Model, resp.InputTokens, resp.OutputTokens, resp.CostUSD, duration.Round(time.Millisecond))

	writeJSON(w, 200, map[string]any{
		"content":      resp.Content,
		"model":        resp.Model,
		"inputTokens":  resp.InputTokens,
		"outputTokens": resp.OutputTokens,
		"costUSD":      resp.CostUSD,
		"duration_ms":  duration.Milliseconds(),
	})
}

func (h *critiqueHandler) models(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []map[string]string{
		{"id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4", "provider": "anthropic"},
		{"id": "claude-opus-4-20250514", "name": "Claude Opus 4", "provider": "anthropic"},
		{"id": "gpt-4o", "name": "GPT-4o", "provider": "openai"},
		{"id": "gpt-4o-mini", "name": "GPT-4o Mini", "provider": "openai"},
	})
}

func (h *critiqueHandler) dimensions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []string{
		"visual_hierarchy", "typography", "color_usage", "spacing_consistency",
		"layout_effectiveness", "cta_prominence", "mobile_responsiveness",
		"brand_consistency", "accessibility", "content_clarity",
	})
}

func buildCritiquePrompt(req critiqueRequest) string {
	pageContext := ""
	if req.PageType != "" {
		pageContext = fmt.Sprintf("\nThis is a %s page.", req.PageType)
	}
	return fmt.Sprintf(`You are an expert web design critic. Analyze the website at %s and provide a detailed design critique.%s

Evaluate these dimensions:
1. Visual Hierarchy - Is the most important content prominent?
2. Typography - Are fonts readable, consistent, well-paired?
3. Color Usage - Is the palette cohesive, accessible, purposeful?
4. Spacing & Layout - Is whitespace used effectively?
5. CTA Prominence - Are calls-to-action clear and compelling?
6. Mobile Responsiveness - Does it work across viewports?
7. Brand Consistency - Is the design language unified?
8. Accessibility - Color contrast, text size, interactive elements
9. Content Clarity - Is the message clear and scannable?
10. Overall Polish - Does it feel professional and trustworthy?

Respond with valid JSON:
{
  "score": <1-100>,
  "summary": "<2-3 sentence overview>",
  "findings": [
    {"dimension": "<name>", "score": <1-10>, "finding": "<specific observation>", "suggestion": "<actionable fix>"}
  ],
  "priority_fixes": ["<most impactful fix 1>", "<fix 2>", "<fix 3>"]
}`, req.WebsiteURL, pageContext)
}
