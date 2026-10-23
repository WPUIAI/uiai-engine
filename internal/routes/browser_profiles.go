package routes

import (
	"encoding/json"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/browserprofile"
	"github.com/WPUIAI/uiai-engine/internal/captcha"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// MountBrowserProfileRoutes exposes profile discovery, resolution, profile-
// scoped browser sessions, and the existing CAPTCHA solver over those sessions.
func MountBrowserProfileRoutes(r chi.Router, manager *browserprofile.Manager, solvers ...*captcha.Solver) {
	if manager == nil {
		return
	}
	var solver *captcha.Solver
	if len(solvers) > 0 {
		solver = solvers[0]
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
			"profiles":        profiles,
			"domain_rules":    registry.Config().DomainRules,
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
				"id":          session.ID,
				"url":         session.URL,
				"title":       session.Title,
				"created_at":  session.CreatedAt,
				"last_used":   session.LastUsed,
				"runtime_pid": session.RuntimePID,
			},
			"selection": session.Selection,
			"profile":   session.Profile,
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
				"id":          session.ID,
				"url":         session.URL,
				"title":       session.Title,
				"created_at":  session.CreatedAt,
				"last_used":   session.LastUsed,
				"runtime_pid": session.RuntimePID,
				"selection":   session.Selection,
				"profile":     session.Profile,
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

		r.Post("/captcha/solve", func(w http.ResponseWriter, req *http.Request) {
			if solver == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "captcha solver unavailable"})
				return
			}
			id := chi.URLParam(req, "profileSessionID")
			page, profile, ok := manager.SessionPage(id)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile session not found"})
				return
			}
			session, _ := manager.Get(id)
			var body captcha.SolveRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
				return
			}
			facade := vision.WrapExternalPage(
				page,
				session.URL,
				profile.Identity.Viewport.Width,
				profile.Identity.Viewport.Height,
			)
			result := solver.SolveInSession(req.Context(), facade, body)
			writeJSON(w, http.StatusOK, map[string]any{
				"result": result,
				"browser_profile": map[string]any{
					"profile_id":     profile.ID,
					"profile_digest": profile.Digest,
					"mode":           profile.Mode,
					"engine":         profile.Engine,
					"network_route":  profile.Network.Route,
				},
			})
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
