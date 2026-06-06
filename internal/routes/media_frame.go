package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/media/deviceframes"
	"github.com/go-chi/chi/v5"
)

var (
	frameRenderer     *deviceframes.Renderer
	frameRendererErr  error
	frameRendererOnce sync.Once
)

func mountFrameRoutes(r chi.Router) {
	r.Get("/catalog", handleFrameCatalog)
	r.Post("/render", handleFrameRender)
}

func getFrameRenderer() (*deviceframes.Renderer, error) {
	frameRendererOnce.Do(func() {
		frameRenderer, frameRendererErr = deviceframes.NewRenderer("")
	})
	return frameRenderer, frameRendererErr
}

func handleFrameCatalog(w http.ResponseWriter, r *http.Request) {
	renderer, err := getFrameRenderer()
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "frame renderer unavailable: " + err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{
		"frames": frameCatalogResponse(renderer.Catalog()),
		"count":  len(renderer.Catalog()),
	})
}

func frameCatalogResponse(frames []deviceframes.FrameConfig) []map[string]any {
	out := make([]map[string]any, 0, len(frames))
	for _, frame := range frames {
		out = append(out, map[string]any{
			"frame_id":      frame.FrameID,
			"frameId":       frame.FrameID,
			"title":         frame.Title,
			"source":        frame.Source,
			"source_ref":    frame.SourceRef,
			"license":       frame.License,
			"svg":           frame.SVG,
			"safe_area":     frame.SafeArea,
			"output":        frame.Output,
			"corner_radius": frame.CornerRadius,
			"active":        frame.Active,
		})
	}
	return out
}

func handleFrameRender(w http.ResponseWriter, r *http.Request) {
	renderer, err := getFrameRenderer()
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "frame renderer unavailable: " + err.Error()})
		return
	}

	var body struct {
		FrameID     string `json:"frameId"`
		FrameIDAlt  string `json:"frame_id"`
		ImageBase64 string `json:"imageBase64"`
		Fit         string `json:"fit"`
		Format      string `json:"format"`
		Quality     int    `json:"quality"`
		Scale       int    `json:"scale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.FrameID == "" {
		body.FrameID = body.FrameIDAlt
	}
	if body.FrameID == "" {
		writeJSON(w, 400, map[string]string{"error": "frameId required"})
		return
	}
	if body.ImageBase64 == "" {
		writeJSON(w, 400, map[string]string{"error": "imageBase64 required"})
		return
	}

	imgB64 := body.ImageBase64
	if idx := strings.Index(imgB64, ","); strings.HasPrefix(imgB64, "data:") && idx > 0 {
		imgB64 = imgB64[idx+1:]
	}
	imgBytes, err := base64.StdEncoding.DecodeString(imgB64)
	if err != nil {
		imgBytes, err = base64.RawStdEncoding.DecodeString(imgB64)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid imageBase64"})
			return
		}
	}

	start := time.Now()
	out, err := renderer.Render(deviceframes.RenderRequest{
		FrameID: body.FrameID,
		Image:   imgBytes,
		Fit:     body.Fit,
		Format:  body.Format,
		Quality: body.Quality,
		Scale:   body.Scale,
	})
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]any{
		"frameId":     body.FrameID,
		"format":      out.Format,
		"width":       out.Width,
		"height":      out.Height,
		"imageBase64": base64.StdEncoding.EncodeToString(out.Image),
		"cacheHit":    false,
		"source":      out.Source,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}
