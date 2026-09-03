package routes

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/captcha"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// MountSessionRoutes registers the persistent browser session API.
//
// LLM Tool Design Principles:
//   - Every action returns a screenshot (visual feedback on every call)
//   - JSON in, JSON out (structured for function calling)
//   - Minimal required params, sensible defaults
//   - Session ID in URL path (clear resource identity)
//   - DOM info included so LLM can reason about interactive elements
func MountSessionRoutes(r chi.Router, cfg *config.Config, sm *vision.SessionManager, solver ...*captcha.Solver) {
	if sm == nil {
		return
	}

	// List all sessions
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		sessions := sm.List()
		out := make([]map[string]any, 0, len(sessions))
		for _, s := range sessions {
			out = append(out, sessionInfoPayload(s))
		}
		writeJSON(w, 200, map[string]any{"sessions": out, "count": len(out), "max": vision.MaxSessions, "max_sessions": vision.MaxSessions})
	})

	// Open a new session
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL          string              `json:"url"`
			Width        int                 `json:"width"`
			Height       int                 `json:"height"`
			FocusaScope  *vision.FocusaScope `json:"focusa_scope"`
			WorkpointID  string              `json:"workpoint_id"`
			ContinuityID string              `json:"continuity_id"`
			ProjectRoot  string              `json:"project_root"`
			EvidenceRef  string              `json:"evidence_ref"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		if body.URL == "" {
			writeSessionContractError(w, 400, "invalid_request", "url required")
			return
		}

		sess, snap, err := sm.Open(body.URL, body.Width, body.Height)
		if err != nil {
			writeSessionError(w, 500, classifySessionError(err), err, sess)
			return
		}
		sess.SetFocusaScope(resolveFocusaScope(body.FocusaScope, body.WorkpointID, body.ContinuityID, body.ProjectRoot, body.EvidenceRef))

		fpvShare, fpvErr := fpvCreateShare(sess.ID, 60, true, false, 0)
		if fpvErr != nil {
			writeSessionError(w, 500, "fpv_share_failed", fpvErr, sess)
			return
		}
		writeJSON(w, 201, map[string]any{
			"session":     sessionInfoPayload(sess),
			"fpv_share":   fpvShare,
			"screenshot":  snap.Screenshot,
			"size":        snap.Size,
			"duration_ms": snap.Duration,
		})
	})

	// Session-scoped routes
	r.Route("/{sessionID}", func(r chi.Router) {

		// Get session info
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			writeJSON(w, 200, sessionInfoPayload(sess))
		})

		// Close session — idempotent: closing an already-reaped/closed session
		// is a success (#45/#73 contract; operators must not need to distinguish).
		r.Delete("/", func(w http.ResponseWriter, req *http.Request) {
			id := chi.URLParam(req, "sessionID")
			if err := sm.Close(id); err != nil {
				if errors.Is(err, vision.ErrSessionNotFound) {
					writeJSON(w, 200, map[string]any{
						"status":     "already_closed",
						"code":       "already_closed",
						"id":         id,
						"retryable":  false,
						"recover":    []string{},
						"state_lost": []string{},
					})
					return
				}
				writeSessionContractError(w, 500, classifySessionError(err), err.Error())
				return
			}
			writeJSON(w, 200, map[string]string{"status": "closed", "id": id})
		})

		// Screenshot — instant re-snap of current state
		r.Post("/screenshot", func(w http.ResponseWriter, req *http.Request) {
			sessionID := chi.URLParam(req, "sessionID")
			sess, ok := sm.Get(sessionID)
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Format   string `json:"format"`
				Quality  int    `json:"quality"`
				FullPage bool   `json:"fullPage"`
				Output   string `json:"output"`
			}
			json.NewDecoder(req.Body).Decode(&body)

			var snap *vision.SnapResult
			var err error
			if body.FullPage {
				snap, err = sess.ScreenshotFull(body.Format, body.Quality)
			} else {
				snap, err = sess.Screenshot(body.Format, body.Quality)
			}
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeScreenshotOutput(w, req, cfg, sessionID, snap, body.Output)
		})

		r.Get("/screenshot/artifacts/{name}", func(w http.ResponseWriter, req *http.Request) {
			path, ok := screenshotArtifactPath(cfg, chi.URLParam(req, "name"))
			if !ok {
				writeSessionContractError(w, 400, "invalid_request", "invalid artifact name")
				return
			}
			http.ServeFile(w, req, path)
		})

		// Navigate to new URL
		r.Post("/navigate", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				URL        string `json:"url"`
				AutoWaitMs int    `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.URL == "" {
				writeSessionContractError(w, 400, "invalid_request", "url required")
				return
			}

			snap, err := sess.Navigate(body.URL)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			if err := applyAutoWait(sess, body.AutoWaitMs); err != nil {
				writeSessionError(w, 500, "auto_wait_failed", err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Scroll (relative)
		r.Post("/scroll", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				DeltaX int `json:"deltaX"`
				DeltaY int `json:"deltaY"`
				X      int `json:"x"` // absolute — if set, uses ScrollTo
				Y      int `json:"y"`
			}
			json.NewDecoder(req.Body).Decode(&body)

			var snap *vision.SnapResult
			var err error
			if body.X > 0 || body.Y > 0 {
				snap, err = sess.ScrollTo(body.X, body.Y)
			} else {
				if body.DeltaY == 0 {
					body.DeltaY = 600 // default: scroll one viewport
				}
				snap, err = sess.Scroll(body.DeltaX, body.DeltaY)
			}
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Click — accepts CSS selector or @ref from snapshot
		r.Post("/click", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector   string `json:"selector"`
				AutoWaitMs int    `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector required (CSS or @ref)")
				return
			}

			resolved, resolveErr := sess.ResolveSelector(body.Selector)
			if resolveErr != nil {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "click", "selector": body.Selector})
				return
			}
			snap, err := sess.Click(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "click", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			if err := applyAutoWait(sess, body.AutoWaitMs); err != nil {
				writeSessionError(w, 500, "auto_wait_failed", err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Hover — accepts CSS selector or @ref from snapshot
		r.Post("/hover", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector   string `json:"selector"`
				AutoWaitMs int    `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector required (CSS or @ref)")
				return
			}

			resolved, resolveErr := sess.ResolveSelector(body.Selector)
			if resolveErr != nil {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "hover", "selector": body.Selector})
				return
			}
			snap, err := sess.Hover(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "hover", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			if err := applyAutoWait(sess, body.AutoWaitMs); err != nil {
				writeSessionError(w, 500, "auto_wait_failed", err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Type into input — accepts CSS selector or @ref from snapshot
		r.Post("/type", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector   string `json:"selector"`
				Text       string `json:"text"`
				AutoWaitMs int    `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || body.Text == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector (CSS or @ref) and text required")
				return
			}
			originalSelector := body.Selector
			if resolved, resolveErr := sess.ResolveSelector(body.Selector); resolveErr == nil {
				body.Selector = resolved
			} else {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "read", "selector": body.Selector})
				return
			}

			snap, err := sess.Type(body.Selector, body.Text)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "type", "selector": originalSelector, "resolved_selector": body.Selector})
				return
			}
			writeJSON(w, 200, snap)
		})

		// Eval JavaScript
		r.Post("/eval", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				JS string `json:"js"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.JS == "" {
				writeSessionContractError(w, 400, "invalid_request", "js required")
				return
			}

			jsResult, snap, err := sess.Eval(body.JS)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}

			resp := map[string]any{"result": jsResult}
			if snap != nil {
				resp["screenshot"] = snap.Screenshot
				resp["size"] = snap.Size
				resp["duration_ms"] = snap.Duration
			}
			writeJSON(w, 200, resp)
		})

		// Eval bounded async JavaScript. Prefer browser actions/snapshot for long UI flows.
		r.Post("/eval_async", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				JS        string `json:"js"`
				TimeoutMs int    `json:"timeout_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.JS == "" {
				writeSessionContractError(w, 400, "invalid_request", "js required")
				return
			}

			jsResult, snap, err := sess.EvalAsync(body.JS, body.TimeoutMs)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "eval_async", "timeout_ms": body.TimeoutMs})
				return
			}
			if strings.HasPrefix(jsResult, "error:") {
				writeSessionError(w, 500, "eval_failed", fmt.Errorf("%s", jsResult), sess, map[string]any{"action": "eval_async", "timeout_ms": body.TimeoutMs})
				return
			}
			resp := map[string]any{"result": jsResult, "bounded_async": true}
			if snap != nil {
				resp["screenshot"] = snap.Screenshot
				resp["size"] = snap.Size
				resp["duration_ms"] = snap.Duration
			}
			writeJSON(w, 200, resp)
		})

		// Resize viewport
		r.Post("/resize", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Width <= 0 || body.Height <= 0 {
				writeSessionContractError(w, 400, "invalid_request", "width and height required")
				return
			}

			snap, err := sess.Resize(body.Width, body.Height)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Inject CSS
		r.Post("/css", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				CSS string `json:"css"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.CSS == "" {
				writeSessionContractError(w, 400, "invalid_request", "css required")
				return
			}

			snap, err := sess.InjectCSS(body.CSS)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Fill — clear + type (more reliable for replacing values)
		r.Post("/fill", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector   string `json:"selector"`
				Text       string `json:"text"`
				AutoWaitMs int    `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || body.Text == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector (CSS or @ref) and text required")
				return
			}
			resolved, resolveErr := sess.ResolveSelector(body.Selector)
			if resolveErr != nil {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "fill", "selector": body.Selector})
				return
			}
			snap, err := sess.Fill(resolved, body.Text)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "fill", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			if err := applyAutoWait(sess, body.AutoWaitMs); err != nil {
				writeSessionError(w, 500, "auto_wait_failed", err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Select — choose dropdown option by value or text
		r.Post("/select", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector   string   `json:"selector"`
				Values     []string `json:"values"`
				AutoWaitMs int      `json:"auto_wait_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || len(body.Values) == 0 {
				writeSessionContractError(w, 400, "invalid_request", "selector and values required")
				return
			}
			resolved, resolveErr := sess.ResolveSelector(body.Selector)
			if resolveErr != nil {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "select", "selector": body.Selector})
				return
			}
			snap, err := sess.Select(resolved, body.Values)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "select", "selector": body.Selector, "resolved_selector": resolved, "values": body.Values})
				return
			}
			writeJSON(w, 200, snap)
		})

		// Press — keyboard key (Enter, Tab, Escape, ArrowDown, etc)
		r.Post("/press", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Key string `json:"key"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Key == "" {
				writeSessionContractError(w, 400, "invalid_request", "key required (Enter, Tab, Escape, ArrowDown, etc)")
				return
			}
			snap, err := sess.Press(body.Key)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Back — browser history back
		r.Post("/back", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			snap, err := sess.Back()
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Forward — browser history forward
		r.Post("/forward", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			snap, err := sess.Forward()
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Text — get text content of element (no screenshot)
		r.Post("/text", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector string `json:"selector"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector (CSS or @ref) required")
				return
			}
			resolved, resolveErr := sess.ResolveSelector(body.Selector)
			if resolveErr != nil {
				writeSessionError(w, 404, "selector_not_found", resolveErr, sess, map[string]any{"action": "text", "selector": body.Selector})
				return
			}
			text, err := sess.TextContent(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "text", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			writeJSON(w, 200, map[string]any{"text": text, "selector": body.Selector})
		})

		// Read — GET is canonical for bounded, side-effect-free page extraction.
		// POST remains temporarily compatible for existing clients.
		readHandler := sessionReadHandler(sm)
		r.Get("/read", readHandler)
		r.Post("/read", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Deprecation", "true")
			w.Header().Set("Warning", `299 UIAI "POST /read is deprecated; use GET with query parameters"`)
			readHandler(w, req)
		})

		// Cookies — get/set/clear
		r.Post("/cookies", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body vision.CookieAction
			json.NewDecoder(req.Body).Decode(&body)
			if body.Action == "" {
				body.Action = "get"
			}
			result, err := sess.Cookies(body)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, result)
		})

		// Auth — save/load authentication state (cookies + localStorage)
		r.Post("/auth/save", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			state, err := sess.SaveAuth()
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(state)
		})

		r.Post("/auth/load", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var state json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&state); err != nil {
				writeSessionContractError(w, 400, "invalid_request", "invalid JSON body")
				return
			}
			if err := sess.LoadAuth(state); err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "loaded"})
		})

		// Snapshot — accessibility tree with @ref selectors
		// This is the primary way LLMs should discover page elements.
		// Returns a text tree + ref map. Use refs in click/type/hover: {"selector": "@e3"}
		r.Post("/snapshot", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body vision.SnapshotOptions
			json.NewDecoder(req.Body).Decode(&body)

			snap, err := sess.Snapshot(body)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}

			// Store refs in session so click/type/hover can resolve @e3
			sess.StoreRefs(snap.Refs)

			writeJSON(w, 200, snap)
		})

		// Also support GET for simple snapshot (interactive mode)
		r.Get("/snapshot", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}

			opts := vision.SnapshotOptions{Interactive: true, Compact: true}
			snap, err := sess.Snapshot(opts)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}

			sess.StoreRefs(snap.Refs)
			writeJSON(w, 200, snap)
		})

		// Selector resolve — convert @ref, text=..., text/..., or role=...;name=... into a concrete CSS selector.
		r.Post("/selector/resolve", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector string `json:"selector"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if strings.TrimSpace(body.Selector) == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector required")
				return
			}
			resolved, err := sess.ResolveSelector(body.Selector)
			if err != nil {
				writeSessionError(w, 404, "selector_not_found", err, sess, map[string]any{"action": "selector_resolve", "selector": body.Selector})
				return
			}
			writeJSON(w, 200, map[string]string{"selector": body.Selector, "resolved_selector": resolved})
		})

		// Diagnostics — bounded console, exception, and network evidence (no screenshot)
		r.Get("/diagnostics", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
			sinceSeq, _ := strconv.ParseUint(req.URL.Query().Get("since_seq"), 10, 64)
			level := req.URL.Query().Get("level")
			failedOnly := req.URL.Query().Get("failed_only") == "true" || req.URL.Query().Get("failed_only") == "1"
			writeJSON(w, 200, sess.DiagnosticsWithOptions(vision.DiagnosticsOptions{
				Limit:      limit,
				Level:      level,
				FailedOnly: failedOnly,
				Category:   req.URL.Query().Get("category"),
				SinceSeq:   sinceSeq,
				Format:     req.URL.Query().Get("format"),
			}))
		})

		// Diagnostics clear — reset session diagnostic buffers
		r.Post("/diagnostics/clear", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			sess.ClearDiagnostics()
			writeJSON(w, 200, map[string]string{"status": "cleared", "session_id": sess.ID})
		})

		// DOM info — structured page data for LLM reasoning (legacy, prefer /snapshot)
		r.Get("/dom", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}

			dom, err := sess.DOMInfo()
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, dom)
		})

		// Wait for selector to appear
		r.Post("/wait", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeSessionContractError(w, 404, "session_not_found", "session not found")
				return
			}
			var body struct {
				Selector  string `json:"selector"`
				TimeoutMs int    `json:"timeout_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeSessionContractError(w, 400, "invalid_request", "selector required")
				return
			}

			snap, err := sess.WaitFor(body.Selector, body.TimeoutMs)
			if err != nil {
				writeSessionError(w, 408, "timeout", err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Captcha solver — POST /api/session/{sessionID}/captcha/solve
		if len(solver) > 0 && solver[0] != nil {
			MountSessionCaptchaRoute(r, solver[0], sm)
		}
	})
}

func sessionInfoPayload(sess *vision.Session) map[string]any {
	out := map[string]any{
		"id":         sess.ID,
		"url":        sess.URL,
		"title":      sess.Title,
		"width":      sess.Width,
		"height":     sess.Height,
		"created_at": sess.CreatedAt,
		"last_used":  sess.LastUsed,
		"nav_count":  sess.NavCount,
		"snap_count": sess.SnapCount,
	}
	if sess.FocusaScope != nil {
		out["focusa_scope"] = sess.FocusaScope
	}
	return out
}

func resolveFocusaScope(scope *vision.FocusaScope, workpointID, continuityID, projectRoot, evidenceRef string) *vision.FocusaScope {
	if scope != nil {
		return scope
	}
	if workpointID == "" && continuityID == "" && projectRoot == "" && evidenceRef == "" {
		return nil
	}
	return &vision.FocusaScope{WorkpointID: workpointID, ContinuityID: continuityID, ProjectRoot: projectRoot, EvidenceRef: evidenceRef}
}

func writeSessionContractError(w http.ResponseWriter, status int, code, message string) {
	event := observability.Record(observability.ErrorEvent{
		Source:              "browser_session",
		Class:               code,
		Status:              status,
		Message:             message,
		SuggestedNextAction: suggestedNextSessionAction(code),
	})
	writeJSON(w, status, observability.NewErrorEnvelope(event, message, nil))
}

func writeSessionError(w http.ResponseWriter, status int, class string, err error, sess *vision.Session, context ...map[string]any) {
	if class == "url_not_allowed" && status >= 500 {
		status = http.StatusBadRequest
	}
	details := map[string]any{}
	for _, ctx := range context {
		for k, v := range ctx {
			details[k] = v
		}
	}
	var diag *vision.DiagnosticsSnapshot
	if sess != nil {
		d := sess.Diagnostics(20, "all", false)
		diag = &d
		details["session_id"] = diag.SessionID
		details["url"] = diag.URL
		details["title"] = diag.Title
		if diag.FocusaScope != nil {
			details["focusa_scope"] = diag.FocusaScope
		}
		details["diagnostics_summary"] = diag.Summary
		if len(diag.FailedRequests) > 0 {
			details["failed_requests"] = diag.FailedRequests
		}
		if len(diag.Console) > 0 {
			details["console"] = diag.Console
		}
		if len(diag.Exceptions) > 0 {
			details["exceptions"] = diag.Exceptions
		}
	}
	event := observability.ErrorEvent{
		Source:              "browser_session",
		Class:               class,
		Status:              status,
		Message:             err.Error(),
		SuggestedNextAction: suggestedNextSessionAction(class),
	}
	if diag != nil {
		event.SessionID = diag.SessionID
		event.URL = diag.URL
		event.Context = map[string]any{
			"title":             diag.Title,
			"console_errors":    diag.Summary.ConsoleErrors,
			"console_warnings":  diag.Summary.ConsoleWarnings,
			"exceptions":        diag.Summary.Exceptions,
			"failed_requests":   diag.Summary.FailedRequests,
			"diagnostics_seq":   diag.Seq,
			"focusa_workpoint":  "",
			"focusa_continuity": "",
		}
		if diag.FocusaScope != nil {
			event.Context["focusa_workpoint"] = diag.FocusaScope.WorkpointID
			event.Context["focusa_continuity"] = diag.FocusaScope.ContinuityID
		}
	}
	event = observability.Record(event)
	resp := observability.NewErrorEnvelope(event, err.Error(), details)
	writeJSON(w, status, resp)
}

func applyAutoWait(sess *vision.Session, timeoutMs int) error {
	if timeoutMs <= 0 {
		return nil
	}
	return sess.AutoWait(timeoutMs)
}

func suggestedNextSessionAction(class string) string {
	switch class {
	case "selector_not_found":
		return "Call /api/session/{id}/snapshot or /diagnostics, then retry using a current @ref or more specific selector."
	case "timeout", "navigation_failed":
		return "Read diagnostics for console/network failures; retry once if transient, otherwise fix the page/navigation target."
	case "page_unavailable":
		return "Close the stale session and open a new session before retrying the action."
	case "url_not_allowed":
		return "Use an http:// or https:// URL allowed by the engine URL safety policy; private/internal URLs require allow_private_urls."
	case "screenshot_failed":
		return "Retry screenshot once, then reopen the session if the page target is unavailable."
	case "eval_failed":
		return "Read browser_diagnostics console/exceptions, keep browser_eval_async bounded, and split long UI work into snapshot plus direct actions."
	default:
		return "Inspect error_class, diagnostics_summary, console, exceptions, and failed_requests before retrying."
	}
}

func classifySessionError(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "selector") || strings.Contains(msg, "element"):
		return "selector_not_found"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") || strings.Contains(msg, "timed out"):
		return "timeout"
	case strings.Contains(msg, "url scheme not allowed") || strings.Contains(msg, "url not allowed"):
		return "url_not_allowed"
	case strings.Contains(msg, "navigation") || strings.Contains(msg, "navigate"):
		return "navigation_failed"
	case strings.Contains(msg, "screenshot"):
		return "screenshot_failed"
	case strings.Contains(msg, "click"):
		return "click_failed"
	case strings.Contains(msg, "eval") || strings.Contains(msg, "javascript"):
		return "eval_failed"
	case strings.Contains(msg, "closed") || strings.Contains(msg, "crash") || strings.Contains(msg, "target"):
		return "page_unavailable"
	default:
		return "action_failed"
	}
}

func writeScreenshotOutput(w http.ResponseWriter, req *http.Request, cfg *config.Config, sessionID string, snap *vision.SnapResult, output string) {
	mode := strings.ToLower(strings.TrimSpace(output))
	if mode == "" || mode == "json" {
		writeJSON(w, 200, snap)
		return
	}
	if mode != "file" && mode != "url" {
		writeSessionContractError(w, 400, "invalid_request", "output must be json, file, or url")
		return
	}
	name, path, err := saveScreenshotArtifact(cfg, sessionID, snap)
	if err != nil {
		writeSessionContractError(w, 500, classifySessionError(err), err.Error())
		return
	}
	artifactURL := fmt.Sprintf("/api/session/%s/screenshot/artifacts/%s", sessionID, name)
	resp := map[string]any{
		"artifact_path": path,
		"artifact_url":  artifactURL,
		"format":        snap.Format,
		"size":          snap.Size,
		"width":         snap.Width,
		"height":        snap.Height,
		"url":           snap.URL,
		"title":         snap.Title,
		"duration":      snap.Duration,
		"output":        mode,
	}
	if mode == "file" {
		writeJSON(w, 200, resp)
		return
	}
	writeJSON(w, 200, resp)
	_ = req
}

func saveScreenshotArtifact(cfg *config.Config, sessionID string, snap *vision.SnapResult) (string, string, error) {
	data, err := base64.StdEncoding.DecodeString(snap.Screenshot)
	if err != nil {
		return "", "", fmt.Errorf("decode screenshot: %w", err)
	}
	dir := screenshotArtifactDir(cfg)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", "", fmt.Errorf("create screenshot artifact dir: %w", err)
	}
	ext := strings.ToLower(snap.Format)
	if ext != "png" {
		ext = "jpeg"
	}
	name := fmt.Sprintf("%s-%d.%s", safeArtifactName(sessionID), time.Now().UnixNano(), ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return "", "", fmt.Errorf("write screenshot artifact: %w", err)
	}
	return name, path, nil
}

func screenshotArtifactDir(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Vision.ShareDir) != "" {
		return filepath.Join(cfg.Vision.ShareDir, "session-screenshots")
	}
	return filepath.Join(os.TempDir(), "uiai-session-screenshots")
}

func screenshotArtifactPath(cfg *config.Config, name string) (string, bool) {
	clean := filepath.Base(name)
	if clean == "." || clean == string(filepath.Separator) || clean != name {
		return "", false
	}
	return filepath.Join(screenshotArtifactDir(cfg), clean), true
}

func safeArtifactName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "session"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := b.String()
	if out == "" {
		return "session"
	}
	return out
}
