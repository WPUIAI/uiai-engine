package routes

import (
	"encoding/json"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// MountAttachRoutes registers the #143 CDP-attach surface:
//
//	GET  /api/attach            → bounded discovery of allow-listed endpoints
//	POST /api/attach            → open a session in an attached operator browser
//
// Attaching is strictly allow-list driven: the caller selects a configured
// endpoint by name and never supplies a debugger URL.
func MountAttachRoutes(r chi.Router, cfg *config.Config, sm *vision.SessionManager, registry *vision.AttachRegistry) {
	if sm == nil {
		return
	}
	if registry == nil {
		registry = vision.NewAttachRegistry()
	}

	// Bounded discovery: names plus reachability, never endpoint internals.
	r.Get("/attach", func(w http.ResponseWriter, _ *http.Request) {
		out := make([]map[string]any, 0, len(cfg.Attach.Endpoints))
		for _, ep := range cfg.Attach.Endpoints {
			pool := registry.GetOrCreate(ep.Name, ep.URL, ep.AllowPrivate)
			out = append(out, map[string]any{
				"name":  ep.Name,
				"alive": pool.IsBrowserAlive(),
			})
		}
		writeJSON(w, 200, map[string]any{
			"enabled":   cfg.Attach.Enabled,
			"endpoints": out,
			"count":     len(out),
		})
	})

	r.Post("/attach", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Endpoint     string              `json:"endpoint"`
			URL          string              `json:"url"`
			Width        int                 `json:"width"`
			Height       int                 `json:"height"`
			FocusaScope  *vision.FocusaScope `json:"focusa_scope"`
			WorkpointID  string              `json:"workpoint_id"`
			ContinuityID string              `json:"continuity_id"`
			ProjectRoot  string              `json:"project_root"`
			EvidenceRef  string              `json:"evidence_ref"`
			ActorRef     string              `json:"actor_ref"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		if !cfg.Attach.Enabled {
			writeJSON(w, 404, map[string]string{"error": "attach is disabled on this deployment (set attach.enabled and name endpoints in config)", "code": "attach_disabled"})
			return
		}
		if body.Endpoint == "" || body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "endpoint (a configured attach name) and url are required", "code": "attach_request_invalid"})
			return
		}
		var selected *config.AttachEndpoint
		for i := range cfg.Attach.Endpoints {
			if cfg.Attach.Endpoints[i].Name == body.Endpoint {
				selected = &cfg.Attach.Endpoints[i]
				break
			}
		}
		if selected == nil {
			writeJSON(w, 400, map[string]string{"error": "unknown attach endpoint; only operator-configured names are attachable", "code": "attach_endpoint_unknown"})
			return
		}

		pool := registry.GetOrCreate(selected.Name, selected.URL, selected.AllowPrivate)
		sess, snap, err := sm.OpenAttached(pool, body.URL, body.Width, body.Height, vision.DefaultHostScope())
		if err != nil {
			writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "attach", "endpoint": selected.Name})
			return
		}
		// The attached session is a full native session: same scope, actor,
		// and challenge treatment as browser_open.
		w.Header().Set("X-UIAI-Created-Session-ID", sess.ID)
		sess.SetActorRef(body.ActorRef)
		sess.SetFocusaScope(resolveFocusaScope(body.FocusaScope, body.WorkpointID, body.ContinuityID, body.ProjectRoot, body.EvidenceRef))
		challenge, _ := sess.DetectAuthChallenge()
		if !writeSessionSnapshot(w, req, cfg, sess, snap, http.StatusCreated, map[string]any{"session": sessionInfoPayload(sess)}) {
			// Idempotent cleanup: the caller never received the id, so the
			// attached tab is released rather than left to TTL.
			_ = sm.Close(sess.ID)
		}
		_ = challenge
	})
}

var _ = chi.URLParam // reserved for endpoint-path variants
