package routes

import (
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func mountSessionPaceRoutes(r chi.Router, cfg *config.Config, manager *vision.SessionManager) {
	mountSessionSequenceRoute(r, cfg, manager)
	r.Post("/pace", func(w http.ResponseWriter, req *http.Request) {
		session, ok := manager.Get(chi.URLParam(req, "sessionID"))
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "session not found"})
			return
		}
		var config *vision.PaceConfig
		if err := decodeStrictJSON(req, &config); err != nil || config == nil {
			writeJSON(w, 400, map[string]string{"error": "invalid pace request"})
			return
		}
		if err := session.SetPace(*config); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		stats := session.Pacing()
		writeJSON(w, 200, map[string]any{"pacing": stats, "instant_default": stats == nil})
	})
	r.Get("/pace", func(w http.ResponseWriter, req *http.Request) {
		session, ok := manager.Get(chi.URLParam(req, "sessionID"))
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "session not found"})
			return
		}
		stats := session.Pacing()
		writeJSON(w, 200, map[string]any{"pacing": stats, "instant_default": stats == nil})
	})
}
