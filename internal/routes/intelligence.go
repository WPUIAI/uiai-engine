package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

var docStore sync.Map // runId → []document

func MountIntelligenceReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider) {
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "healthy", "module": "intelligence"})
	})

	// --- Document indexing ---

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

	// --- Semantic search ---

	r.Post("/search", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Query string `json:"query"`
			RunID string `json:"runId"`
			Limit int    `json:"limit"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.Limit <= 0 {
			body.Limit = 5
		}

		// Simple keyword search across stored docs (no embedding required)
		var results []any
		if v, ok := docStore.Load(body.RunID); ok {
			docs := v.([]any)
			for i, doc := range docs {
				if i >= body.Limit {
					break
				}
				docBytes, _ := json.Marshal(doc)
				docStr := string(docBytes)
				// Score based on keyword overlap
				if body.Query == "" || contains(docStr, body.Query) {
					results = append(results, map[string]any{
						"index":    i,
						"document": doc,
						"score":    1.0 - float64(i)*0.1,
					})
				}
			}
		}
		if results == nil {
			results = []any{}
		}
		writeJSON(w, 200, map[string]any{"results": results, "query": body.Query, "runId": body.RunID, "count": len(results)})
	})

	// --- Embeddings (ss-api-8ag) ---

	r.Post("/embed", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Input    []string `json:"input"`
			Model    string   `json:"model"`
			Provider string   `json:"provider"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		if len(body.Input) == 0 {
			writeJSON(w, 400, map[string]string{"error": "input texts required"})
			return
		}
		if body.Model == "" {
			body.Model = "text-embedding-3-small"
		}
		if body.Provider == "" {
			body.Provider = "openai"
		}

		// Call OpenAI-compatible embeddings API
		embeddings, err := computeEmbeddings(cfg, aiProv, body.Input, body.Model, body.Provider)
		if err != nil {
			log.Printf("[intelligence] embed error: %v", err)
			// Return zero-vectors as fallback (graceful degradation)
			zeros := make([][]float64, len(body.Input))
			for i := range zeros {
				zeros[i] = make([]float64, 256)
			}
			writeJSON(w, 200, map[string]any{
				"embeddings": zeros,
				"model":      body.Model,
				"status":     "degraded",
				"error":      err.Error(),
			})
			return
		}

		writeJSON(w, 200, map[string]any{
			"embeddings": embeddings,
			"model":      body.Model,
			"dimensions": len(embeddings[0]),
			"count":      len(embeddings),
			"status":     "ok",
		})
	})

	// Raw executable artifact routes are retired; creation returns mandatory EPWA delivery.
	for _, kind := range []string{"wasm", "js"} {
		kind := kind
		r.Get("/"+kind+"/{runId}", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, http.StatusGone, map[string]any{
				"schema": "uiai.epwa_delivery_error.v1", "code": "legacy_raw_artifact_removed",
				"artifact_kind": kind, "runId": chi.URLParam(req, "runId"),
				"message": "raw executable artifacts are available only through their EPWA viewer and portable package",
			})
		})
	}

	r.Post("/wasm/{runId}", func(w http.ResponseWriter, req *http.Request) {
		runID := chi.URLParam(req, "runId")
		const maxWASMBytes = 50 * 1024 * 1024
		data, err := io.ReadAll(io.LimitReader(req.Body, maxWASMBytes+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body failed"})
			return
		}
		if len(data) > maxWASMBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "WASM artifact exceeds 50 MiB"})
			return
		}
		writeBinaryArtifactEPWA(w, req, cfg, "runtime:wasm:"+runID, "WASM runtime artifact "+runID, "application/wasm", "wasm", data, http.StatusCreated)
	})

	r.Post("/js/{runId}", func(w http.ResponseWriter, req *http.Request) {
		runID := chi.URLParam(req, "runId")
		const maxJavaScriptBytes = 20 * 1024 * 1024
		data, err := io.ReadAll(io.LimitReader(req.Body, maxJavaScriptBytes+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body failed"})
			return
		}
		if len(data) > maxJavaScriptBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "JavaScript artifact exceeds 20 MiB"})
			return
		}
		writeBinaryArtifactEPWA(w, req, cfg, "runtime:javascript:"+runID, "JavaScript runtime artifact "+runID, "application/javascript", "js", data, http.StatusCreated)
	})
}

// computeEmbeddings calls an OpenAI-compatible embeddings endpoint.
func computeEmbeddings(cfg *config.Config, aiProv *ai.Provider, input []string, model, provider string) ([][]float64, error) {
	var baseURL, apiKey string

	switch provider {
	case "openai":
		baseURL = "https://api.openai.com/v1"
		if p, ok := cfg.AI.Providers["openai"]; ok && p.APIURL != "" {
			baseURL = p.APIURL
		}
		// Get key from the ai provider's cached keys
		apiKey = os.Getenv("OPENAI_API_KEY") // Embeddings use env var or WP key
	default:
		return nil, fmt.Errorf("embeddings provider %q not supported (only openai supported)", provider)
	}

	body, _ := json.Marshal(map[string]any{
		"input": input,
		"model": model,
	})

	req, _ := http.NewRequest("POST", baseURL+"/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embeddings API %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var data struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("parse embeddings response: %w", err)
	}

	embeddings := make([][]float64, len(data.Data))
	for i, d := range data.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}

// contains does a case-insensitive substring check.
func contains(s, sub string) bool {
	return len(sub) == 0 || len(s) >= len(sub) && (s == sub || containsCI(s, sub))
}

func containsCI(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			c1, c2 := s[i+j], sub[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
