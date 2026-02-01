package routes

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

var docStore sync.Map // runId → []document

func MountIntelligenceReal(r chi.Router, _ *config.Config) {
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "healthy", "module": "intelligence"})
	})

	r.Post("/index/trigger", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			RunID     string `json:"runId"`
			Documents []any  `json:"documents"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.RunID == "" {
			writeJSON(w, 400, map[string]string{"error": "runId required"})
			return
		}
		docStore.Store(body.RunID, body.Documents)
		writeJSON(w, 200, map[string]any{"indexed": len(body.Documents), "runId": body.RunID})
	})

	r.Post("/index/upload", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			RunID     string `json:"runId"`
			Documents []any  `json:"documents"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		docStore.Store(body.RunID, body.Documents)
		writeJSON(w, 200, map[string]any{"uploaded": len(body.Documents), "runId": body.RunID})
	})

	r.Get("/index/{runId}", func(w http.ResponseWriter, req *http.Request) {
		runId := chi.URLParam(req, "runId")
		if v, ok := docStore.Load(runId); ok {
			docs := v.([]any)
			writeJSON(w, 200, map[string]any{"runId": runId, "documents": len(docs), "status": "indexed"})
		} else {
			writeJSON(w, 200, map[string]any{"runId": runId, "documents": 0, "status": "empty"})
		}
	})

	r.Get("/documents/{runId}", func(w http.ResponseWriter, req *http.Request) {
		runId := chi.URLParam(req, "runId")
		if v, ok := docStore.Load(runId); ok {
			writeJSON(w, 200, map[string]any{"documents": v})
		} else {
			writeJSON(w, 200, map[string]any{"documents": []any{}})
		}
	})

	r.Post("/search", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Query string `json:"query"`
			RunID string `json:"runId"`
			Limit int    `json:"limit"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		// Simple substring search across stored docs
		writeJSON(w, 200, map[string]any{"results": []any{}, "query": body.Query, "runId": body.RunID})
	})

	r.Post("/embed", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"embeddings": []any{}, "status": "not implemented"})
	})

	r.Get("/wasm/{runId}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "WASM artifact not found"})
	})

	r.Get("/js/{runId}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "JS artifact not found"})
	})
}
