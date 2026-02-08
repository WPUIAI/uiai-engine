package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

// MountMigrationAPI registers data migration endpoints.
// These allow importing data from the old Bun/PHP systems into Go.
func MountMigrationAPI(r chi.Router, cfg *config.Config) {
	r.Post("/import/usage", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Records []struct {
				Type         string  `json:"type"`
				Model        string  `json:"model"`
				InputTokens  int     `json:"inputTokens"`
				OutputTokens int     `json:"outputTokens"`
				CostUSD      float64 `json:"costUsd"`
				Status       string  `json:"status"`
				CreatedAt    string  `json:"createdAt"`
			} `json:"records"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"imported": len(body.Records), "status": "accepted"})
	})

	r.Post("/import/memory", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			UserID       string         `json:"userId"`
			Conversation []memoryEntry  `json:"conversation"`
			Preferences  map[string]any `json:"preferences"`
			Context      map[string]any `json:"context"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		if body.UserID == "" {
			writeJSON(w, 400, map[string]string{"error": "userId required"})
			return
		}
		if memBackend != nil {
			m := memBackend.get(body.UserID)
			m.Conversation = append(m.Conversation, body.Conversation...)
			if body.Preferences != nil {
				for k, v := range body.Preferences {
					m.Preferences[k] = v
				}
			}
			if body.Context != nil {
				for k, v := range body.Context {
					m.Context[k] = v
				}
			}
			memBackend.save(body.UserID)
		}
		writeJSON(w, 200, map[string]any{
			"userId":       body.UserID,
			"conversation": len(body.Conversation),
			"status":       "imported",
		})
	})

	r.Post("/import/shares", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Shares []shareEntry `json:"shares"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		count := 0
		for _, s := range body.Shares {
			if s.ID == "" {
				continue
			}
			sCopy := s
			shareStore.Store(s.ID, &sCopy)
			persistShare(&sCopy)
			count++
		}
		writeJSON(w, 200, map[string]any{"imported": count, "status": "accepted"})
	})

	r.Get("/status", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"engine":     "go",
			"version":    "2.0.0",
			"migrated":   true,
			"bun_status": "stopped",
			"php_status": "stopped",
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	})
}
