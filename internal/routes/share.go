package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

type shareEntry struct {
	ID        string         `json:"id"`
	URL       string         `json:"url"`
	Title     string         `json:"title"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"createdAt"`
	ExpiresAt time.Time      `json:"expiresAt"`
	Views     int            `json:"views"`
}

var shareStore sync.Map

func MountShareReal(r chi.Router, _ *config.Config, pool *vision.Pool) {
	// Create share
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL       string         `json:"url"`
			Title     string         `json:"title"`
			ExpiresIn int            `json:"expiresIn"` // minutes
			Data      map[string]any `json:"data"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		idBytes := make([]byte, 8)
		rand.Read(idBytes)
		id := hex.EncodeToString(idBytes)

		expiresIn := body.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440 // 24 hours default
		}

		entry := &shareEntry{
			ID:        id,
			URL:       body.URL,
			Title:     body.Title,
			Data:      body.Data,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Minute),
		}
		shareStore.Store(id, entry)

		writeJSON(w, 200, map[string]any{
			"id":        id,
			"shareUrl":  "/api/share/" + id,
			"expiresAt": entry.ExpiresAt,
		})
	})

	// Create multi-page share
	r.Post("/multi", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URLs      []string       `json:"urls"`
			Title     string         `json:"title"`
			ExpiresIn int            `json:"expiresIn"`
			Data      map[string]any `json:"data"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if len(body.URLs) == 0 {
			writeJSON(w, 400, map[string]string{"error": "urls required"})
			return
		}

		idBytes := make([]byte, 8)
		rand.Read(idBytes)
		id := "multi-" + hex.EncodeToString(idBytes)

		expiresIn := body.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440
		}

		entry := &shareEntry{
			ID:    id,
			URL:   body.URLs[0],
			Title: body.Title,
			Data: map[string]any{
				"urls":     body.URLs,
				"custom":   body.Data,
				"pageCount": len(body.URLs),
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Minute),
		}
		shareStore.Store(id, entry)

		writeJSON(w, 200, map[string]any{
			"id":        id,
			"shareUrl":  "/api/share/" + id,
			"pages":     len(body.URLs),
			"expiresAt": entry.ExpiresAt,
		})
	})

	// Get share
	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		v, ok := shareStore.Load(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "share not found"})
			return
		}
		entry := v.(*shareEntry)
		if time.Now().After(entry.ExpiresAt) {
			shareStore.Delete(id)
			writeJSON(w, 410, map[string]string{"error": "share expired"})
			return
		}
		entry.Views++
		writeJSON(w, 200, entry)
	})

	// Delete share
	r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
		shareStore.Delete(chi.URLParam(req, "id"))
		writeJSON(w, 200, map[string]string{"message": "share deleted"})
	})

	// Screenshot for share
	r.Post("/{id}/screenshot", func(w http.ResponseWriter, req *http.Request) {
		if pool == nil {
			writeJSON(w, 503, map[string]string{"error": "vision pool not available"})
			return
		}
		id := chi.URLParam(req, "id")
		v, ok := shareStore.Load(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "share not found"})
			return
		}
		entry := v.(*shareEntry)

		result, err := pool.Screenshot(vision.ScreenshotOpts{
			URL:    entry.URL,
			Width:  1280,
			Height: 800,
			Format: "jpeg",
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(result.Data)
	})
}
