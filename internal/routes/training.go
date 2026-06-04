package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

type signedUploadToken struct {
	DatasetID string
	Filename  string
	ExpiresAt time.Time
}

var signedTokens sync.Map // token → *signedUploadToken

// In-memory stores for training system (matches Bun behavior — all in-memory Maps)
var (
	trainingJobs sync.Map
	evalRuns     sync.Map
	datasets     sync.Map
	modelReg     sync.Map
)

func MountTrainingReal(r chi.Router, cfg *config.Config) {
	// Service token auth check — matches Bun behavior:
	// 503 if no service token configured, 401 if wrong token
	serviceToken := cfg.WordPress.WebhookSecret

	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if serviceToken == "" {
				writeJSON(w, 503, map[string]string{"error": "Service not configured"})
				return
			}
			token := req.Header.Get("X-API-Token")
			if token == "" {
				auth := req.Header.Get("Authorization")
				if len(auth) > 7 {
					token = auth[7:]
				}
			}
			if token != serviceToken {
				writeJSON(w, 401, map[string]string{"error": "Unauthorized - valid service token required"})
				return
			}
			next(w, req)
		}
	}

	// Jobs CRUD
	r.Post("/jobs", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		id := time.Now().Format("20060102-150405")
		body["id"] = id
		body["status"] = "pending"
		body["createdAt"] = time.Now().UTC().Format(time.RFC3339)
		trainingJobs.Store(id, body)
		writeJSON(w, 201, body)
	}))

	r.Get("/jobs", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var jobs []any
		trainingJobs.Range(func(_, v any) bool { jobs = append(jobs, v); return true })
		writeJSON(w, 200, map[string]any{"jobs": jobs, "count": len(jobs)})
	}))

	r.Get("/jobs/{id}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if v, ok := trainingJobs.Load(id); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "job not found"})
		}
	}))

	r.Post("/jobs/{id}/cancel", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if v, ok := trainingJobs.Load(id); ok {
			m := v.(map[string]any)
			m["status"] = "cancelled"
			writeJSON(w, 200, m)
		} else {
			writeJSON(w, 404, map[string]string{"error": "job not found"})
		}
	}))

	// Evals CRUD
	r.Post("/evals", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		id := "eval-" + time.Now().Format("20060102-150405")
		body["id"] = id
		body["completedAt"] = time.Now().UTC().Format(time.RFC3339)
		evalRuns.Store(id, body)
		writeJSON(w, 201, body)
	}))

	r.Get("/evals", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var evals []any
		evalRuns.Range(func(_, v any) bool { evals = append(evals, v); return true })
		writeJSON(w, 200, map[string]any{"evals": evals, "count": len(evals)})
	}))

	r.Get("/evals/{id}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		if v, ok := evalRuns.Load(chi.URLParam(req, "id")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "eval not found"})
		}
	}))

	// Registry
	r.Get("/registry/models", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var models []any
		modelReg.Range(func(_, v any) bool { models = append(models, v); return true })
		writeJSON(w, 200, map[string]any{"models": models, "count": len(models)})
	}))

	r.Get("/registry/models/{id}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		if v, ok := modelReg.Load(chi.URLParam(req, "id")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "model not found"})
		}
	}))

	r.Post("/registry/models", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		id, _ := body["id"].(string)
		if id == "" {
			id = "model-" + time.Now().Format("20060102-150405")
			body["id"] = id
		}
		modelReg.Store(id, body)
		writeJSON(w, 201, body)
	}))

	r.Post("/registry/promote", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			ModelID     string `json:"model_id"`
			TargetStage string `json:"target_stage"` // staging, production
			Reason      string `json:"reason"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.ModelID == "" {
			writeJSON(w, 400, map[string]string{"error": "model_id required"})
			return
		}
		if body.TargetStage == "" {
			body.TargetStage = "staging"
		}

		// Verify model exists in registry
		v, ok := modelReg.Load(body.ModelID)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "model not found in registry"})
			return
		}
		model := v.(map[string]any)
		prevStage, _ := model["stage"].(string)
		model["stage"] = body.TargetStage
		model["promoted_at"] = time.Now().UTC().Format(time.RFC3339)
		model["promoted_by"] = body.Reason

		// Record in audit log
		auditEntry := map[string]any{
			"model_id":   body.ModelID,
			"from_stage": prevStage,
			"to_stage":   body.TargetStage,
			"reason":     body.Reason,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}

		writeJSON(w, 200, map[string]any{
			"allowed":    true,
			"model_id":   body.ModelID,
			"from_stage": prevStage,
			"to_stage":   body.TargetStage,
			"reason":     body.Reason,
			"promotedAt": time.Now().UTC().Format(time.RFC3339),
			"audit":      auditEntry,
		})
	}))

	r.Get("/registry/audit", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"records": []any{}, "count": 0})
	}))

	// Datasets
	r.Post("/datasets", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		id := "ds-" + time.Now().Format("20060102-150405")
		body["id"] = id
		datasets.Store(id, body)
		writeJSON(w, 201, body)
	}))

	r.Get("/datasets", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var ds []any
		datasets.Range(func(_, v any) bool { ds = append(ds, v); return true })
		writeJSON(w, 200, map[string]any{"datasets": ds, "count": len(ds)})
	}))

	r.Get("/datasets/{id}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		if v, ok := datasets.Load(chi.URLParam(req, "id")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "dataset not found"})
		}
	}))

	r.Get("/datasets/{id}/shards", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"shards": []any{}, "count": 0})
	}))

	r.Get("/datasets/shards/{id}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"shard": nil, "error": "shard not found"})
	}))

	r.Post("/datasets/signed-url", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			DatasetID string `json:"dataset_id"`
			Filename  string `json:"filename"`
			ExpiresIn int    `json:"expires_in"` // minutes
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.ExpiresIn <= 0 {
			body.ExpiresIn = 60
		}

		// Generate a signed upload URL. In production this would use R2/S3 pre-signed URLs.
		// For now, generate a token that the /datasets/confirm-upload endpoint accepts.
		tokenBytes := make([]byte, 16)
		rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)

		uploadURL := fmt.Sprintf("https://ai.wpuiai.com/api/training/datasets/upload?token=%s&dataset=%s", token, body.DatasetID)
		expiry := time.Now().Add(time.Duration(body.ExpiresIn) * time.Minute)

		// Store token for validation
		signedTokens.Store(token, &signedUploadToken{
			DatasetID: body.DatasetID,
			Filename:  body.Filename,
			ExpiresAt: expiry,
		})

		writeJSON(w, 200, map[string]any{
			"url":        uploadURL,
			"token":      token,
			"expires_at": expiry.Format(time.RFC3339),
			"method":     "PUT",
		})
	}))

	r.Post("/datasets/confirm-upload", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"confirmed": true})
	}))

	// Teacher labeling
	r.Post("/teacher/label", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"status": "accepted", "jobId": "label-" + time.Now().Format("150405")})
	}))

	// Eval configs
	r.Get("/eval-config/{engineId}", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"engineId": chi.URLParam(req, "engineId"), "config": map[string]any{}})
	}))

	r.Get("/eval-configs", requireAuth(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"configs": []any{}, "count": 0})
	}))
}
