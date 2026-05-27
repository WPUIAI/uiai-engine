package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/captcha"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

// MountSessionRoutes registers the persistent browser session API.
//
// LLM Tool Design Principles:
//   - Every action returns a screenshot (visual feedback on every call)
//   - JSON in, JSON out (structured for function calling)
//   - Minimal required params, sensible defaults
//   - Session ID in URL path (clear resource identity)
//   - DOM info included so LLM can reason about interactive elements
func MountSessionRoutes(r chi.Router, _ *config.Config, sm *vision.SessionManager, solver ...*captcha.Solver) {
	if sm == nil {
		return
	}

	// List all sessions
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		sessions := sm.List()
		out := make([]map[string]any, 0, len(sessions))
		for _, s := range sessions {
			out = append(out, map[string]any{
				"id":         s.ID,
				"url":        s.URL,
				"title":      s.Title,
				"width":      s.Width,
				"height":     s.Height,
				"created_at": s.CreatedAt,
				"last_used":  s.LastUsed,
				"nav_count":  s.NavCount,
				"snap_count": s.SnapCount,
			})
		}
		writeJSON(w, 200, map[string]any{"sessions": out, "count": len(out), "max": vision.MaxSessions})
	})

	// Open a new session
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		sess, snap, err := sm.Open(body.URL, body.Width, body.Height)
		if err != nil {
			writeSessionError(w, 500, classifySessionError(err), err, sess)
			return
		}

		writeJSON(w, 201, map[string]any{
			"session": map[string]any{
				"id":     sess.ID,
				"url":    sess.URL,
				"title":  sess.Title,
				"width":  sess.Width,
				"height": sess.Height,
			},
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			writeJSON(w, 200, map[string]any{
				"id":         sess.ID,
				"url":        sess.URL,
				"title":      sess.Title,
				"width":      sess.Width,
				"height":     sess.Height,
				"created_at": sess.CreatedAt,
				"last_used":  sess.LastUsed,
				"nav_count":  sess.NavCount,
				"snap_count": sess.SnapCount,
			})
		})

		// Close session
		r.Delete("/", func(w http.ResponseWriter, req *http.Request) {
			id := chi.URLParam(req, "sessionID")
			if err := sm.Close(id); err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]string{"status": "closed", "id": id})
		})

		// Screenshot — instant re-snap of current state
		r.Post("/screenshot", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Format   string `json:"format"`
				Quality  int    `json:"quality"`
				FullPage bool   `json:"fullPage"`
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
			writeJSON(w, 200, snap)
		})

		// Navigate to new URL
		r.Post("/navigate", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				URL string `json:"url"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.URL == "" {
				writeJSON(w, 400, map[string]string{"error": "url required"})
				return
			}

			snap, err := sess.Navigate(body.URL)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess)
				return
			}
			writeJSON(w, 200, snap)
		})

		// Scroll (relative)
		r.Post("/scroll", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string `json:"selector"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeJSON(w, 400, map[string]string{"error": "selector required (CSS or @ref)"})
				return
			}

			resolved := sess.ResolveRef(body.Selector)
			snap, err := sess.Click(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "click", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			writeJSON(w, 200, snap)
		})

		// Hover — accepts CSS selector or @ref from snapshot
		r.Post("/hover", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string `json:"selector"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeJSON(w, 400, map[string]string{"error": "selector required (CSS or @ref)"})
				return
			}

			resolved := sess.ResolveRef(body.Selector)
			snap, err := sess.Hover(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "hover", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			writeJSON(w, 200, snap)
		})

		// Type into input — accepts CSS selector or @ref from snapshot
		r.Post("/type", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string `json:"selector"`
				Text     string `json:"text"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || body.Text == "" {
				writeJSON(w, 400, map[string]string{"error": "selector (CSS or @ref) and text required"})
				return
			}
			originalSelector := body.Selector
			body.Selector = sess.ResolveRef(body.Selector)

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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				JS string `json:"js"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.JS == "" {
				writeJSON(w, 400, map[string]string{"error": "js required"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				JS        string `json:"js"`
				TimeoutMs int    `json:"timeout_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.JS == "" {
				writeJSON(w, 400, map[string]string{"error": "js required"})
				return
			}

			jsResult, snap, err := sess.EvalAsync(body.JS, body.TimeoutMs)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "eval_async", "timeout_ms": body.TimeoutMs})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Width <= 0 || body.Height <= 0 {
				writeJSON(w, 400, map[string]string{"error": "width and height required"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				CSS string `json:"css"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.CSS == "" {
				writeJSON(w, 400, map[string]string{"error": "css required"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string `json:"selector"`
				Text     string `json:"text"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || body.Text == "" {
				writeJSON(w, 400, map[string]string{"error": "selector (CSS or @ref) and text required"})
				return
			}
			resolved := sess.ResolveRef(body.Selector)
			snap, err := sess.Fill(resolved, body.Text)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "fill", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			writeJSON(w, 200, snap)
		})

		// Select — choose dropdown option by value or text
		r.Post("/select", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string   `json:"selector"`
				Values   []string `json:"values"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" || len(body.Values) == 0 {
				writeJSON(w, 400, map[string]string{"error": "selector and values required"})
				return
			}
			resolved := sess.ResolveRef(body.Selector)
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Key string `json:"key"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Key == "" {
				writeJSON(w, 400, map[string]string{"error": "key required (Enter, Tab, Escape, ArrowDown, etc)"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector string `json:"selector"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeJSON(w, 400, map[string]string{"error": "selector (CSS or @ref) required"})
				return
			}
			resolved := sess.ResolveRef(body.Selector)
			text, err := sess.TextContent(resolved)
			if err != nil {
				writeSessionError(w, 500, classifySessionError(err), err, sess, map[string]any{"action": "text", "selector": body.Selector, "resolved_selector": resolved})
				return
			}
			writeJSON(w, 200, map[string]any{"text": text, "selector": body.Selector})
		})

		// Cookies — get/set/clear
		r.Post("/cookies", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var state json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&state); err != nil {
				writeJSON(w, 400, map[string]string{"error": "invalid JSON body"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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

		// Diagnostics — bounded console, exception, and network evidence (no screenshot)
		r.Get("/diagnostics", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
			level := req.URL.Query().Get("level")
			failedOnly := req.URL.Query().Get("failed_only") == "true" || req.URL.Query().Get("failed_only") == "1"
			writeJSON(w, 200, sess.Diagnostics(limit, level, failedOnly))
		})

		// Diagnostics clear — reset session diagnostic buffers
		r.Post("/diagnostics/clear", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			sess.ClearDiagnostics()
			writeJSON(w, 200, map[string]string{"status": "cleared", "session_id": sess.ID})
		})

		// DOM info — structured page data for LLM reasoning (legacy, prefer /snapshot)
		r.Get("/dom", func(w http.ResponseWriter, req *http.Request) {
			sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "session not found"})
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
				writeJSON(w, 404, map[string]string{"error": "session not found"})
				return
			}
			var body struct {
				Selector  string `json:"selector"`
				TimeoutMs int    `json:"timeout_ms"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			if body.Selector == "" {
				writeJSON(w, 400, map[string]string{"error": "selector required"})
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

func writeSessionError(w http.ResponseWriter, status int, class string, err error, sess *vision.Session, context ...map[string]any) {
	resp := map[string]any{
		"error":                 err.Error(),
		"error_class":           class,
		"suggested_next_action": suggestedNextSessionAction(class),
	}
	for _, ctx := range context {
		for k, v := range ctx {
			resp[k] = v
		}
	}
	if sess != nil {
		diag := sess.Diagnostics(20, "all", false)
		resp["session_id"] = diag.SessionID
		resp["url"] = diag.URL
		resp["title"] = diag.Title
		resp["diagnostics_summary"] = diag.Summary
		if len(diag.FailedRequests) > 0 {
			resp["failed_requests"] = diag.FailedRequests
		}
		if len(diag.Console) > 0 {
			resp["console"] = diag.Console
		}
		if len(diag.Exceptions) > 0 {
			resp["exceptions"] = diag.Exceptions
		}
	}
	writeJSON(w, status, resp)
}

func suggestedNextSessionAction(class string) string {
	switch class {
	case "selector_not_found":
		return "Call /api/session/{id}/snapshot or /diagnostics, then retry using a current @ref or more specific selector."
	case "timeout", "navigation_failed":
		return "Read diagnostics for console/network failures; retry once if transient, otherwise fix the page/navigation target."
	case "page_unavailable":
		return "Close the stale session and open a new session before retrying the action."
	case "screenshot_failed":
		return "Retry screenshot once, then reopen the session if the page target is unavailable."
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
