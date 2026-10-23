// Package routes provides HTTP route handlers for every API module.
// Health mount and writeJSON helper. Stub mounts kept for fallback only.
package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

// --- Health (implemented) ---

func MountHealth(r chi.Router, cfg *config.Config, aiProv *ai.Provider) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":    "healthy",
			"service":   "uiai-engine",
			"version":   "2.0.0",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	r.Get("/providers", func(w http.ResponseWriter, req *http.Request) {
		providerSet := map[string]bool{}
		for _, m := range aiProv.AvailableModels() {
			providerSet[m.Provider] = true
		}
		result := map[string]any{}
		for _, name := range []string{"anthropic", "openai", "openrouter", "fireworks", "kimi", "qwen"} {
			result[name] = map[string]any{"available": providerSet[name]}
		}
		writeJSON(w, 200, result)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(InjectCost(data, w))
}
