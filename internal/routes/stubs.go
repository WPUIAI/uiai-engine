// Package routes provides HTTP route handlers for every API module.
// Stub mounts return 501 until each module is implemented.
package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/config"
)

// --- Health (implemented) ---

func MountHealth(r chi.Router, cfg *config.Config, aiProv *ai.Provider) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":    "healthy",
			"service":   "uiai-engine",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	r.Get("/providers", func(w http.ResponseWriter, req *http.Request) {
		// Build provider availability from WP-managed models.
		// A provider is "available" if at least one model from it is in the list.
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

// --- Stub mounts: return 501 with module name until implemented ---

func MountCritique(r chi.Router, cfg *config.Config)      { stubModule(r, "critique") }
func MountUIReverse(r chi.Router, cfg *config.Config)      { stubModule(r, "ui-reverse") }
func MountSectionDetect(r chi.Router, cfg *config.Config)  { stubModule(r, "section-detect") }
func MountLayoutCompare(r chi.Router, cfg *config.Config)  { stubModule(r, "layout-compare") }
func MountStyleEnhance(r chi.Router, cfg *config.Config)   { stubModule(r, "style-enhance") }
func MountCopilot(r chi.Router, cfg *config.Config)        { stubModule(r, "copilot") }
func MountIntake(r chi.Router, cfg *config.Config)         { stubModule(r, "intake") }
func MountWorkflow(r chi.Router, cfg *config.Config)       { stubModule(r, "workflow") }
func MountUsage(r chi.Router, cfg *config.Config)          { stubModule(r, "usage") }
func MountExtension(r chi.Router, cfg *config.Config)      { stubModule(r, "extension") }
func MountMemory(r chi.Router, cfg *config.Config)         { stubModule(r, "memory") }
func MountAdmin(r chi.Router, cfg *config.Config)          { stubModule(r, "admin") }
func MountIntelligence(r chi.Router, cfg *config.Config)   { stubModule(r, "intelligence") }
func MountTraining(r chi.Router, cfg *config.Config)       { stubModule(r, "training") }
func MountScreenshot(r chi.Router, cfg *config.Config)     { stubModule(r, "screenshot") }
func MountShare(r chi.Router, cfg *config.Config)          { stubModule(r, "share") }

func HandleShareViewer(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		writeJSON(w, 501, map[string]any{
			"error":  "share viewer not yet implemented",
			"token":  token,
			"module": "share-viewer",
		})
	}
}

func HandleDashboard(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>UIAI Engine Dashboard</h1><p>Coming soon.</p></body></html>`))
	}
}

// stubModule creates catch-all GET and POST handlers that return 501.
func stubModule(r chi.Router, name string) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 501, map[string]any{
			"error":  name + " module not yet implemented",
			"module": name,
			"status": "stub",
		})
	}
	r.Get("/*", handler)
	r.Post("/*", handler)
	r.Put("/*", handler)
	r.Delete("/*", handler)
	// Also handle root of the route group
	r.Get("/", handler)
	r.Post("/", handler)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
