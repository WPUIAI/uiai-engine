package routes

import (
	"encoding/json"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/browserprofile"
	"github.com/go-chi/chi/v5"
)

// MountBrowserProfileRoutes exposes profile discovery, resolution, and
// profile-scoped browser sessions.
func MountBrowserProfileRoutes(r chi.Router, manager *browserprofile.Manager) {
	if manager == nil {
		return
	}

	r.Get("/profiles", func(w http.ResponseWriter, req *http.Request) {
		registry := manager.Registry()
		profiles := make([]browserprofile.ResolvedProfile, 0, len(registry.Names()))
		for _, name := range registry.Names() {
			profile, err := registry.Resolve(name)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "profile": name})
				return
			}
			profiles = append(profiles, profile)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"default_profile": registry.Config().DefaultProfile,
			"profiles": profiles,
			"domain_rules": registry.Config().DomainRules,
		})
	})

	r.Post("/resolve", func(w http.ResponseWriter, req *http.Request) {
		var body browserprofile.OpenRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
			return
		}
		profile, selection, err := manager.Registry().Select(body.URL, body.BrowserProfile, body.BrowserMode)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"selection": selection, "profile": profile})
	})

	r.Get("/sessions", func(w http.ResponseWriter, req *http.Request) {
		sessions := manager.List()
		writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions, "count": len(sessions)})
	})

	r.Post("/sessions", func(w http.ResponseWriter, req *http.Request) {
		var body browserprofile.OpenRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
			return
		}
		session, err := manager.Open(req.Context(), body)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"session": map[string]any{
				"id": session.ID,
				"url": session.URL,
				"title": session.Title,
				"created_at": session.CreatedAt,
				"last_used": session.LastUsed,
				"runtime_pid": session.RuntimePID,
			},
			"selection": session.Selection,
			"profile": session.Profile,
		})
	})

	r.Route("/sessions/{profileSessionID}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			id := chi.URLParam(req, "profileSessionID")
			session, ok := manager.Get(id)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile session not found"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"id": session.ID,
				"url": session.URL,
				"title": session.Title,
				"created_at": session.CreatedAt,
				"last_used": session.LastUsed,
				"runtime_pid": session.RuntimePID,
				"selection": session.Selection,
				"profile": session.Profile,
			})
		})

		r.Post("/navigate", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				URL string `json:"url"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.URL == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
				return
			}
			session, err := manager.Navigate(req.Context(), chi.URLParam(req, "profileSessionID"), body.URL)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, session)
		})

		r.Delete("/", func(w http.ResponseWriter, req *http.Request) {
			id := chi.URLParam(req, "profileSessionID")
			if err := manager.Close(id); err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "closed", "id": id})
		})
	})
}
