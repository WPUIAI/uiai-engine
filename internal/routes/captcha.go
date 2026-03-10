package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/captcha"
	visionPkg "github.com/philoveracity/uiai-engine/internal/vision"
)

// MountCaptchaRoutes registers captcha solver endpoints.
//
//	POST /api/captcha/solve-image   — stateless image-only solve
//	GET  /api/captcha/status        — solver capabilities and stats
//
// Session-scoped solve is mounted under /api/session/{id}/captcha/solve
// via MountSessionCaptchaRoutes.
func MountCaptchaRoutes(r chi.Router, solver *captcha.Solver) {
	if solver == nil {
		return
	}

	// POST /api/captcha/solve-image — stateless
	r.Post("/solve-image", func(w http.ResponseWriter, req *http.Request) {
		var body captcha.ImageSolveRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if body.ImageBase64 == "" {
			http.Error(w, `{"error":"image_base64 is required"}`, http.StatusBadRequest)
			return
		}

		result, err := solver.SolveImage(req.Context(), body)
		if err != nil {
			log.Printf("[captcha-route] solve-image failed: %v", err)
		}
		if result == nil {
			http.Error(w, `{"error":"solve failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// POST /api/captcha/solve-proxied — solve via proxy-rotated browser
	r.Post("/solve-proxied", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL     string               `json:"url"`
			Width   int                  `json:"width"`
			Height  int                  `json:"height"`
			Fields  map[string]string    `json:"fields"`   // name→value to fill
			Selects map[string]string    `json:"selects"`  // select name→value
			Captcha captcha.SolveRequest `json:"captcha"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if body.URL == "" {
			http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
			return
		}
		if body.Width <= 0 {
			body.Width = 1280
		}
		if body.Height <= 0 {
			body.Height = 900
		}

		result := solver.SolveViaProxy(req.Context(), body.URL, body.Width, body.Height,
			func(sess *visionPkg.Session) error {
				// Fill input fields
				for name, val := range body.Fields {
					selector := `input[name="` + name + `"]`
					sess.Fill(selector, val)
				}
				// Fill selects via eval
				for name, val := range body.Selects {
					js := `var el=document.querySelector('select[name="` + name + `"]');if(el){el.value="` + val + `";el.dispatchEvent(new Event('change',{bubbles:true}));return "ok"}return "not_found"`
					sess.Eval(js)
				}
				return nil
			},
			body.Captcha,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// GET /api/captcha/status
	r.Get("/status", func(w http.ResponseWriter, req *http.Request) {
		status := solver.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// ─── IP Pool management ──────────────────────────────────────────

	// GET /api/captcha/pool — full pool status with per-IP health
	r.Get("/pool", func(w http.ResponseWriter, req *http.Request) {
		pool := solver.Pool()
		if pool == nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "IP pool not enabled"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pool.Status())
	})

	// POST /api/captcha/pool/add — add an IP at runtime
	r.Post("/pool/add", func(w http.ResponseWriter, req *http.Request) {
		pool := solver.Pool()
		if pool == nil {
			http.Error(w, `{"error":"IP pool not enabled"}`, http.StatusBadRequest)
			return
		}
		var body struct {
			Endpoint string `json:"endpoint"` // "local:1.2.3.4" or "socks5://..."
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Endpoint == "" {
			http.Error(w, `{"error":"endpoint is required"}`, http.StatusBadRequest)
			return
		}
		if err := pool.AddIP(body.Endpoint); err != nil {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added", "endpoint": body.Endpoint})
	})

	// POST /api/captcha/pool/remove — remove an IP at runtime
	r.Post("/pool/remove", func(w http.ResponseWriter, req *http.Request) {
		pool := solver.Pool()
		if pool == nil {
			http.Error(w, `{"error":"IP pool not enabled"}`, http.StatusBadRequest)
			return
		}
		var body struct {
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Endpoint == "" {
			http.Error(w, `{"error":"endpoint is required"}`, http.StatusBadRequest)
			return
		}
		if err := pool.RemoveIP(body.Endpoint); err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed", "endpoint": body.Endpoint})
	})
}

// MountSessionCaptchaRoute adds POST /captcha/solve under a session route group.
// Called from session.go route setup.
func MountSessionCaptchaRoute(r chi.Router, solver *captcha.Solver, sm *visionPkg.SessionManager) {
	if solver == nil {
		return
	}

	r.Post("/captcha/solve", func(w http.ResponseWriter, req *http.Request) {
		sessionID := chi.URLParam(req, "sessionID")
		sess, ok := sm.Get(sessionID)
		if !ok {
			http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
			return
		}

		var body captcha.SolveRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		result := solver.SolveInSession(req.Context(), sess, body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
}
