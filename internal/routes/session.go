package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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
func MountSessionRoutes(r chi.Router, _ *config.Config, sm *vision.SessionManager) {
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
			writeJSON(w, 500, map[string]string{"error": err.Error()})
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
			"screenshot": snap.Screenshot,
			"size":       snap.Size,
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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

			snap, err := sess.Click(sess.ResolveRef(body.Selector))
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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

			snap, err := sess.Hover(sess.ResolveRef(body.Selector))
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
			body.Selector = sess.ResolveRef(body.Selector)

			snap, err := sess.Type(body.Selector, body.Text)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
			snap, err := sess.Fill(sess.ResolveRef(body.Selector), body.Text)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
			snap, err := sess.Select(sess.ResolveRef(body.Selector), body.Values)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
			text, err := sess.TextContent(sess.ResolveRef(body.Selector))
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}

			sess.StoreRefs(snap.Refs)
			writeJSON(w, 200, snap)
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
				writeJSON(w, 500, map[string]string{"error": err.Error()})
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
				writeJSON(w, 408, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, snap)
		})
	})
}
