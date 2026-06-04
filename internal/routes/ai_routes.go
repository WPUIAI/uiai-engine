package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/credits"
	"github.com/WPUIAI/uiai-engine/internal/ratelimit"
	"github.com/WPUIAI/uiai-engine/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Shared deps for all AI routes
type aiDeps struct {
	cfg     *config.Config
	ai      *ai.Provider
	credits *credits.Service
	limiter *ratelimit.Limiter
	usage   *storage.UsageStore
}

// --- Generic vision-AI request (shared by ui-reverse, section-detect, layout-compare, style-enhance) ---

type visionAIRequest struct {
	WebsiteURL       string `json:"websiteUrl,omitempty"`
	ImageURL         string `json:"image_url,omitempty"`
	ImageBase64      string `json:"image_base64,omitempty"`
	BuiltImageURL    string `json:"built_image_url,omitempty"`
	BuiltImageBase64 string `json:"built_image_base64,omitempty"`
	RefImageURL      string `json:"reference_image_url,omitempty"`
	RefImageBase64   string `json:"reference_image_base64,omitempty"`
	ExistingCSS      string `json:"existing_css,omitempty"`
	Operation        string `json:"operation,omitempty"`
	Model            string `json:"model"`
	Provider         string `json:"provider"`
	Prompt           string `json:"prompt,omitempty"`
}

func (d *aiDeps) handleVisionAI(w http.ResponseWriter, r *http.Request, opName, creditOp, prompt string) {
	id := auth.FromContext(r.Context())
	if id == nil {
		writeJSON(w, 401, map[string]string{"error": "authentication required"})
		return
	}

	key := id.LicenseKey
	if key == "" {
		key = id.APIKey
	}
	if err := d.limiter.Check(key, id.Tier); err != nil {
		writeJSON(w, 429, map[string]string{"error": err.Error()})
		return
	}

	var req visionAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Model == "" {
		req.Model = d.cfg.AI.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = d.cfg.AI.DefaultProvider
	}

	start := time.Now()
	resp, err := d.ai.Complete(r.Context(), ai.Request{
		Provider:    req.Provider,
		Model:       req.Model,
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.7,
		ImageBase64: req.ImageBase64,
	})
	if err != nil {
		log.Printf("[%s] AI error: %v", opName, err)
		writeJSON(w, 502, map[string]string{"error": "AI provider error: " + err.Error()})
		return
	}

	go d.credits.Deduct(id.LicenseID, creditOp, "")

	d.usage.Record(storage.UsageRecord{
		Type:         opName,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
		Status:       "success",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	})

	log.Printf("[%s] %s %d+%d tokens $%.4f %s", opName, resp.Model, resp.InputTokens, resp.OutputTokens, resp.CostUSD, time.Since(start).Round(time.Millisecond))

	writeJSON(w, 200, map[string]any{
		"content":      resp.Content,
		"model":        resp.Model,
		"inputTokens":  resp.InputTokens,
		"outputTokens": resp.OutputTokens,
		"costUSD":      resp.CostUSD,
		"duration_ms":  time.Since(start).Milliseconds(),
	})
}

// --- UI Reverse ---

func MountUIReverseReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &aiDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		d.handleVisionAI(w, req, "ui-reverse", "ui_reverse", `Analyze this website screenshot and extract design tokens. Return JSON with: {
  "tokens": {"colors": [], "typography": [], "spacing": [], "borders": []},
  "components": [{"name": "", "description": "", "css_properties": {}}],
  "spacing": {"base_unit": 0, "scale": []},
  "page_analysis": {"layout_type": "", "sections": [], "grid_system": ""}
}`)
	})

	r.Get("/models", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, aiProv.AvailableModels())
	})

	r.Get("/operations", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, []string{
			"analyze-reference", "extract-components", "extract-tokens", "extract-spacing", "full-pipeline",
		})
	})
}

// --- Section Detect ---

func MountSectionDetectReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &aiDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		d.handleVisionAI(w, req, "section-detect", "section_detect", `Analyze this page screenshot and identify all distinct visual sections. Return JSON: {
  "sections": [{"name": "", "type": "", "y_start": 0, "y_end": 0, "description": "", "elements": []}]
}`)
	})
}

// --- Layout Compare ---

func MountLayoutCompareReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &aiDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		d.handleVisionAI(w, req, "layout-compare", "layout_compare", `Compare these two page screenshots (built vs reference). Return JSON: {
  "match_score": 0.0,
  "differences": [{"area": "", "severity": "", "description": "", "suggestion": ""}],
  "recommendations": [""]
}`)
	})
}

// --- Style Enhance ---

func MountStyleEnhanceReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &aiDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		d.handleVisionAI(w, req, "style-enhance", "style_enhance", `Analyze this page screenshot and suggest CSS improvements. Return JSON: {
  "suggestions": [{"property": "", "current": "", "recommended": "", "reason": ""}],
  "css_improvements": ""
}`)
	})
}

// --- Copilot ---

func MountCopilotReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	d := &aiDeps{cfg: cfg, ai: aiProv, credits: creds, limiter: lim, usage: usage}

	r.Post("/chat", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		key := id.LicenseKey
		if key == "" {
			key = id.APIKey
		}
		if err := d.limiter.Check(key, id.Tier); err != nil {
			writeJSON(w, 429, map[string]string{"error": err.Error()})
			return
		}

		var body struct {
			Message  string `json:"message"`
			Model    string `json:"model"`
			Provider string `json:"provider"`
			Context  string `json:"context"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Message == "" {
			writeJSON(w, 400, map[string]string{"error": "message is required"})
			return
		}
		if body.Model == "" {
			body.Model = d.cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = d.cfg.AI.DefaultProvider
		}

		prompt := fmt.Sprintf("You are WPUIAI Copilot, an AI assistant for WordPress website building.\n\nUser message: %s", body.Message)
		if body.Context != "" {
			prompt = fmt.Sprintf("You are WPUIAI Copilot, an AI assistant for WordPress website building.\n\nContext: %s\n\nUser message: %s", body.Context, body.Message)
		}

		start := time.Now()
		resp, err := d.ai.Complete(req.Context(), ai.Request{
			Provider:    body.Provider,
			Model:       body.Model,
			Prompt:      prompt,
			MaxTokens:   4096,
			Temperature: 0.7,
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "AI error: " + err.Error()})
			return
		}

		go d.credits.Deduct(id.LicenseID, "copilot", "")

		d.usage.Record(storage.UsageRecord{
			Type:         "copilot",
			Model:        resp.Model,
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
			CostUSD:      resp.CostUSD,
			Status:       "success",
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		})

		writeJSON(w, 200, map[string]any{
			"response":     resp.Content,
			"model":        resp.Model,
			"inputTokens":  resp.InputTokens,
			"outputTokens": resp.OutputTokens,
			"costUSD":      resp.CostUSD,
			"duration_ms":  time.Since(start).Milliseconds(),
		})
	})

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "healthy"})
	})
}
