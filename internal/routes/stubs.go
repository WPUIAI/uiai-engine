// Package routes provides HTTP route handlers for every API module.
// Health mount and writeJSON helper. Stub mounts kept for fallback only.
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

// HandleShareViewer serves a public share page with live screenshot.
func HandleShareViewer(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		// Try to load share entry from in-memory store
		v, ok := shareStore.Load(token)
		if !ok {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(404)
			w.Write([]byte(`<!DOCTYPE html><html><body><h1>Share Not Found</h1><p>This share link has expired or does not exist.</p></body></html>`))
			return
		}
		entry := v.(*shareEntry)
		if time.Now().After(entry.ExpiresAt) {
			shareStore.Delete(token)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(410)
			w.Write([]byte(`<!DOCTYPE html><html><body><h1>Share Expired</h1><p>This share link has expired.</p></body></html>`))
			return
		}
		entry.Views++

		title := entry.Title
		if title == "" {
			title = "WPUIAI Shared Design"
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>` + title + `</title>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{font-family:system-ui;max-width:1200px;margin:2em auto;padding:0 1em;color:#1e293b;background:#f8fafc}
h1{color:#059669}iframe{width:100%;height:80vh;border:1px solid #e2e8f0;border-radius:8px}
.meta{color:#64748b;font-size:13px;margin-bottom:1em}</style></head><body>
<h1>` + title + `</h1>
<div class="meta">Shared URL: <a href="` + entry.URL + `">` + entry.URL + `</a> · Views: ` + time.Now().Format("2006-01-02 15:04") + `</div>
<iframe src="` + entry.URL + `" loading="lazy"></iframe>
</body></html>`))
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
