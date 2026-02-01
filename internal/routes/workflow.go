package routes

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

var (
	siteStore    sync.Map // siteId → site
	workflowRuns sync.Map // runId → run
)

func MountWorkflowReal(r chi.Router, _ *config.Config) {
	// Site registration
	r.Post("/sites", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			SiteURL       string `json:"site_url"`
			AdminUsername string `json:"admin_username"`
			AdminPassword string `json:"admin_password"`
			PluginSlug    string `json:"plugin_slug"`
			PluginZipURL  string `json:"plugin_zip_url"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.SiteURL == "" || body.AdminUsername == "" {
			writeJSON(w, 400, map[string]string{"error": "site_url and admin_username required"})
			return
		}
		id := "site_" + time.Now().Format("20060102150405")
		site := map[string]any{
			"id": id, "site_url": body.SiteURL, "plugin_slug": body.PluginSlug,
			"status": "registered", "created_at": time.Now().UTC().Format(time.RFC3339),
		}
		siteStore.Store(id, site)
		writeJSON(w, 201, map[string]any{"success": true, "site": site})
	})

	r.Get("/sites/{id}", func(w http.ResponseWriter, req *http.Request) {
		if v, ok := siteStore.Load(chi.URLParam(req, "id")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
		}
	})

	r.Get("/sites/{id}/status", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if _, ok := siteStore.Load(id); ok {
			writeJSON(w, 200, map[string]any{"site_id": id, "status": "ready"})
		} else {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
		}
	})

	// Workflow operations
	for _, action := range []string{"create-run", "set-active-run", "start", "execute", "run", "complete", "skip"} {
		act := action
		r.Post("/sites/{id}/workflow/"+act, func(w http.ResponseWriter, req *http.Request) {
			siteId := chi.URLParam(req, "id")
			var body map[string]any
			json.NewDecoder(req.Body).Decode(&body)
			writeJSON(w, 200, map[string]any{
				"success": true, "site_id": siteId, "action": act,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		})
	}

	// Templates
	r.Get("/templates", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"templates": []any{
			map[string]any{"id": "default", "name": "Default 29-Step", "steps": 29},
		}})
	})

	r.Get("/templates/{id}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"id": chi.URLParam(req, "id"), "name": "Default 29-Step", "steps": 29})
	})

	// Direct execution
	r.Post("/execute", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"status": "accepted", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	})

	r.Post("/run", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		runId := "run-" + time.Now().Format("20060102-150405")
		writeJSON(w, 200, map[string]any{"run_id": runId, "status": "started"})
	})

	r.Get("/status/{runId}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"run_id": chi.URLParam(req, "runId"), "status": "unknown"})
	})
}
