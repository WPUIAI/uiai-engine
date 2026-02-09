package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/storage"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

func MountScreenshotReal(r chi.Router, _ *config.Config, pool *vision.Pool, usage *storage.UsageStore) {
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
			Cookies  string `json:"cookies"`  // "name=value; name2=value2"
			Timeout  int    `json:"timeout"`  // overall timeout in seconds (default: 30)
			NoCache  bool   `json:"nocache"`  // skip cache, always take fresh screenshot
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
			Cookies:  body.Cookies,
			Timeout:  body.Timeout,
			NoCache:  body.NoCache,
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		// Record usage
		if usage != nil {
			usage.Record(storage.UsageRecord{
				Type:      "screenshot",
				Status:    "success",
				CostUSD:   0.005, // flat per-screenshot cost
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			})
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
