package routes

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/auth"
	"github.com/philoveracity/uiai-engine/internal/config"
)

var intakeStore sync.Map

func MountIntakeReal(r chi.Router, cfg *config.Config, aiProv *ai.Provider) {
	r.Post("/analyze", func(w http.ResponseWriter, req *http.Request) {
		id := auth.FromContext(req.Context())
		if id == nil {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}

		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)

		resp, err := aiProv.Complete(ai.Request{
			Provider: cfg.AI.DefaultProvider,
			Model:    cfg.AI.DefaultModel,
			Prompt:   "Analyze this business intake data and provide recommendations for website design. Data: " + toJSON(body),
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, 200, map[string]any{"analysis": resp.Content, "model": resp.Model, "tokens": resp.InputTokens + resp.OutputTokens})
	})

	r.Post("/submit", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		intakeId := "intake-" + time.Now().Format("20060102-150405")
		body["id"] = intakeId
		body["status"] = "submitted"
		body["submitted_at"] = time.Now().UTC().Format(time.RFC3339)
		intakeStore.Store(intakeId, body)
		writeJSON(w, 200, map[string]any{"success": true, "id": intakeId})
	})

	r.Get("/status/{id}", func(w http.ResponseWriter, req *http.Request) {
		intakeId := chi.URLParam(req, "id")
		if v, ok := intakeStore.Load(intakeId); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "intake not found"})
		}
	})
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
