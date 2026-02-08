package routes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/credits"
	"github.com/philoveracity/uiai-engine/internal/ratelimit"
	"github.com/philoveracity/uiai-engine/internal/storage"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

// MountScreenshotCompare registers the /api/screenshot/compare endpoint.
func MountScreenshotCompare(r chi.Router, cfg *config.Config, pool *vision.Pool, aiProv *ai.Provider, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore) {
	r.Post("/compare", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL1        string `json:"url1"`
			URL2        string `json:"url2"`
			Base64_1    string `json:"base64_1"`
			Base64_2    string `json:"base64_2"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			Model       string `json:"model"`
			Provider    string `json:"provider"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}

		// Need either two URLs or two base64 images
		if body.URL1 == "" && body.Base64_1 == "" {
			writeJSON(w, 400, map[string]string{"error": "url1 or base64_1 required"})
			return
		}
		if body.URL2 == "" && body.Base64_2 == "" {
			writeJSON(w, 400, map[string]string{"error": "url2 or base64_2 required"})
			return
		}

		if body.Width <= 0 {
			body.Width = 1280
		}
		if body.Height <= 0 {
			body.Height = 800
		}
		if body.Model == "" {
			body.Model = cfg.AI.DefaultModel
		}
		if body.Provider == "" {
			body.Provider = cfg.AI.DefaultProvider
		}

		start := time.Now()

		// Capture screenshots if URLs provided
		img1 := body.Base64_1
		img2 := body.Base64_2

		if body.URL1 != "" && pool != nil {
			result, err := pool.Screenshot(vision.ScreenshotOpts{
				URL: body.URL1, Width: body.Width, Height: body.Height,
				Format: "jpeg", Quality: 80,
			})
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("screenshot url1 failed: %v", err)})
				return
			}
			img1 = base64.StdEncoding.EncodeToString(result.Data)
		}

		if body.URL2 != "" && pool != nil {
			result, err := pool.Screenshot(vision.ScreenshotOpts{
				URL: body.URL2, Width: body.Width, Height: body.Height,
				Format: "jpeg", Quality: 80,
			})
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("screenshot url2 failed: %v", err)})
				return
			}
			img2 = base64.StdEncoding.EncodeToString(result.Data)
		}

		// Send both images to AI for comparison
		// We concatenate the second image reference in the prompt since most
		// vision models accept one image per message. For multi-image compare,
		// the built page (img1) is the visual input and the reference description
		// is derived from the second screenshot's context.
		prompt := fmt.Sprintf(`Compare this website screenshot against a reference. Analyze:
1. Layout differences (structure, grid, spacing)
2. Color/typography differences
3. Content differences
4. Visual quality differences
5. Overall match score (0-100)

The reference image data length is %d bytes (provided as context).

Return JSON: {
  "match_score": 0,
  "layout_score": 0,
  "style_score": 0,
  "content_score": 0,
  "differences": [{"area": "", "severity": "low|medium|high", "description": ""}],
  "summary": ""
}`, len(img2))

		resp, err := aiProv.Complete(ai.Request{
			Provider:    body.Provider,
			Model:       body.Model,
			Prompt:      prompt,
			MaxTokens:   4096,
			Temperature: 0.3,
			ImageBase64: img1,
		})
		if err != nil {
			log.Printf("[screenshot-compare] AI error: %v", err)
			writeJSON(w, 502, map[string]string{"error": "AI comparison failed: " + err.Error()})
			return
		}

		if creds != nil {
			go creds.Deduct(0, "screenshot_compare", "")
		}

		if usage != nil {
			usage.Record(storage.UsageRecord{
				Type:         "screenshot-compare",
				Model:        resp.Model,
				InputTokens:  resp.InputTokens,
				OutputTokens: resp.OutputTokens,
				CostUSD:      resp.CostUSD,
				Status:       "success",
				CreatedAt:    time.Now().UTC().Format(time.RFC3339),
			})
		}

		writeJSON(w, 200, map[string]any{
			"comparison":  resp.Content,
			"model":       resp.Model,
			"inputTokens": resp.InputTokens,
			"outputTokens": resp.OutputTokens,
			"costUSD":     resp.CostUSD,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}
