package routes

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

// In-memory stores for training system (matches Bun behavior — all in-memory Maps)
var (
	trainingJobs sync.Map
	evalRuns     sync.Map
	datasets     sync.Map
	modelReg     sync.Map
)

func MountTrainingReal(r chi.Router, cfg *config.Config) {
	// Service token auth check
	serviceToken := cfg.WordPress.WebhookSecret

	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			token := req.Header.Get("X-API-Token")
			if token == "" {
				auth := req.Header.Get("Authorization")
				if len(auth) > 7 {
					token = auth[7:]
				}
			}
			if token == "" || (serviceToken != "" && token != serviceToken) {
				writeJSON(w, 401, map[string]string{"error": "valid service token required"})
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
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"allowed": true, "reason": "promotion accepted", "promotedAt": time.Now().UTC().Format(time.RFC3339)})
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
		writeJSON(w, 200, map[string]any{"url": "", "error": "not implemented"})
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
