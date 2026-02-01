package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

func MountScreenshotReal(r chi.Router, _ *config.Config, pool *vision.Pool) {
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL      string `json:"url"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			FullPage bool   `json:"fullPage"`
			Format   string `json:"format"`
			Quality  int    `json:"quality"`
			WaitFor  string `json:"waitFor"`
			Delay    int    `json:"delay"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		if pool == nil {
			writeJSON(w, 503, map[string]string{"error": "vision pool not initialized"})
			return
		}

		result, err := pool.Screenshot(vision.ScreenshotOpts{
			URL:      body.URL,
			Width:    body.Width,
			Height:   body.Height,
			FullPage: body.FullPage,
			Format:   body.Format,
			Quality:  body.Quality,
			WaitFor:  body.WaitFor,
			Delay:    body.Delay,
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		resp := map[string]any{
			"screenshot": base64.StdEncoding.EncodeToString(result.Data),
			"width":      result.Width,
			"height":     result.Height,
			"format":     result.Format,
			"size":       len(result.Data),
			"duration":   result.Duration.Milliseconds(),
		}
		if result.DOMReport != "" {
			resp["dom_report"] = result.DOMReport
		}
		writeJSON(w, 200, resp)
	})

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		if pool == nil {
			writeJSON(w, 503, map[string]string{"status": "unavailable", "error": "vision pool not initialized"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"status": "healthy",
			"pool":   pool.Stats(),
		})
	})
}
