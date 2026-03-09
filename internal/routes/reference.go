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
	"github.com/philoveracity/uiai-engine/internal/reference"
	"github.com/philoveracity/uiai-engine/internal/storage"
)

type referenceHandler struct {
	cfg      *config.Config
	analyzer *reference.Analyzer
	credits  *credits.Service
	limiter  *ratelimit.Limiter
	usage    *storage.UsageStore
}

type referenceRequest struct {
	ImageBase64 string `json:"imageBase64"`
	ImageType   string `json:"imageType"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Pass        string `json:"pass"` // "analyze", "components", "tokens", "spacing", "text", "full"
	// For pass 2 (components) — sections from pass 1
	Sections []reference.Section `json:"sections,omitempty"`
	// For pass 4 (spacing) — components from pass 2
	Components []reference.Component `json:"components,omitempty"`
}

func MountReferenceReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	h := &referenceHandler{
		cfg:      cfg,
		analyzer: reference.NewAnalyzer(aiProv),
		credits:  creds,
		limiter:  lim,
		usage:    usage,
	}

	r.Post("/analyze", h.analyze)
}

func (h *referenceHandler) analyze(w http.ResponseWriter, r *http.Request) {
	id := auth.FromContext(r.Context())
	if id == nil {
		writeJSON(w, 401, map[string]string{"error": "authentication required"})
		return
	}

	key := id.LicenseKey
	if key == "" {
		key = id.APIKey
	}
	if err := h.limiter.Check(key, id.Tier); err != nil {
		writeJSON(w, 429, map[string]string{"error": err.Error()})
		return
	}

	var req referenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if req.ImageBase64 == "" {
		writeJSON(w, 400, map[string]string{"error": "imageBase64 is required"})
		return
	}
	providerProvided := req.Provider != ""
	modelProvided := req.Model != ""
	if req.Model == "" {
		req.Model = h.cfg.AI.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = h.cfg.AI.DefaultProvider
	}
	if req.Pass == "" {
		req.Pass = "full"
	}

	start := time.Now()

	ctx := r.Context()

	switch req.Pass {
	case "analyze":
		result, err := h.analyzer.AnalyzeReference(ctx, req.ImageBase64, req.ImageType, req.Provider, req.Model)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_analyze", "")
		h.logUsage("reference_analyze", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	case "components":
		result, err := h.analyzer.ExtractComponents(ctx, req.ImageBase64, req.ImageType, req.Provider, req.Model, req.Sections)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_components", "")
		h.logUsage("reference_components", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	case "tokens":
		result, err := h.analyzer.ExtractTokens(ctx, req.ImageBase64, req.ImageType, req.Provider, req.Model)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_tokens", "")
		h.logUsage("reference_tokens", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	case "spacing":
		result, err := h.analyzer.ExtractSpacing(ctx, req.ImageBase64, req.ImageType, req.Provider, req.Model, req.Components)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_spacing", "")
		h.logUsage("reference_spacing", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	case "text":
		provider := req.Provider
		model := req.Model
		if !providerProvided {
			provider = ""
		}
		if !modelProvided {
			model = ""
		}
		result, err := h.analyzer.ExtractText(ctx, req.ImageBase64, req.ImageType, provider, model)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_text", "")
		h.logUsage("reference_text", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	case "full":
		result, err := h.analyzer.AnalyzeFull(ctx, req.ImageBase64, req.ImageType, req.Provider, req.Model)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		go h.credits.Deduct(id.LicenseID, "reference_full", "")
		h.logUsage("reference_full", req.Model, start)
		writeJSON(w, 200, map[string]any{"success": true, "data": result})

	default:
		writeJSON(w, 400, map[string]string{"error": "pass must be: analyze, components, tokens, spacing, text, or full"})
	}
}

func (h *referenceHandler) logUsage(opType, model string, start time.Time) {
	duration := time.Since(start)
	log.Printf("[reference] %s %s %s", opType, model, duration.Round(time.Millisecond))
	h.usage.Record(storage.UsageRecord{
		Type:      opType,
		Model:     model,
		Status:    "success",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
