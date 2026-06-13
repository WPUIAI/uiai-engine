package routes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

type fpvShare struct {
	Token     string        `json:"token"`
	SessionID string        `json:"session_id"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	Views     int           `json:"views"`
	Controls  bool          `json:"controls"`
	Audit     []fpvAuditLog `json:"audit,omitempty"`
}

type fpvAuditLog struct {
	TS       time.Time      `json:"ts"`
	Action   string         `json:"action"`
	Selector string         `json:"selector,omitempty"`
	Key      string         `json:"key,omitempty"`
	Message  string         `json:"message,omitempty"`
	OK       bool           `json:"ok"`
	Error    string         `json:"error,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

var fpvShares sync.Map

func MountFPVRoutes(r chi.Router, sm *vision.SessionManager) {
	r.Post("/share", func(w http.ResponseWriter, req *http.Request) {
		if sm == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session manager unavailable"})
			return
		}
		var body struct {
			SessionID      string `json:"session_id"`
			ExpiresMinutes int    `json:"expires_minutes"`
			Controls       bool   `json:"controls"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		if body.SessionID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id required"})
			return
		}
		if _, ok := sm.Get(body.SessionID); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		minutes := body.ExpiresMinutes
		if minutes <= 0 || minutes > 240 {
			minutes = 60
		}
		token, err := fpvToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		entry := &fpvShare{Token: token, SessionID: body.SessionID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(minutes) * time.Minute), Controls: body.Controls}
		fpvShares.Store(token, entry)
		writeJSON(w, http.StatusOK, map[string]any{
			"token":                 token,
			"session_id":            body.SessionID,
			"mirror_url":            "/m/" + token,
			"status_url":            "/m/" + token + "/status",
			"screenshot_url":        "/m/" + token + "/screenshot.jpg",
			"mirror_url_expires_at": entry.ExpiresAt,
			"mode":                  fpvMode(entry.Controls),
		})
	})
}

func MountFPVPublicRoutes(r chi.Router, sm *vision.SessionManager) {
	r.Get("/assets/{file}", func(w http.ResponseWriter, req *http.Request) {
		file := filepath.Base(chi.URLParam(req, "file"))
		if file != "fpv.css" && file != "fpv.js" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "asset not found"})
			return
		}
		http.ServeFile(w, req, filepath.Join("web", "fpv", file))
	})
	r.Get("/{token}", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		entry.Views++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl, err := template.ParseFiles(filepath.Join("web", "fpv", "index.html"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fpv template unavailable"})
			return
		}
		_ = tmpl.Execute(w, map[string]any{"Token": entry.Token, "SessionID": entry.SessionID})
	})
	r.Get("/{token}/status", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		sess, ok := sm.Get(entry.SessionID)
		if !ok {
			writeJSON(w, http.StatusGone, map[string]string{"error": "session closed"})
			return
		}
		diag := sess.DiagnosticsWithOptions(vision.DiagnosticsOptions{Format: "summary", Limit: 1})
		writeJSON(w, http.StatusOK, map[string]any{
			"token":       entry.Token,
			"session_id":  entry.SessionID,
			"mode":        fpvMode(entry.Controls),
			"controls":    entry.Controls,
			"audit_count": len(entry.Audit),
			"audit":       entry.Audit,
			"views":       entry.Views,
			"created_at":  entry.CreatedAt,
			"url":         sess.URL,
			"title":       sess.Title,
			"width":       sess.Width,
			"height":      sess.Height,
			"expires_at":  entry.ExpiresAt,
			"diagnostics": diag.Summary,
			"context":     fpvContext(),
		})
	})

	r.Post("/{token}/control", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		if !entry.Controls {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "fpv share is read-only"})
			return
		}
		sess, ok := sm.Get(entry.SessionID)
		if !ok {
			writeJSON(w, http.StatusGone, map[string]string{"error": "session closed"})
			return
		}
		var body struct {
			Action   string `json:"action"`
			Selector string `json:"selector"`
			Text     string `json:"text"`
			Key      string `json:"key"`
			Message  string `json:"message"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		log := fpvAuditLog{TS: time.Now().UTC(), Action: body.Action, Selector: body.Selector, Key: body.Key, Message: body.Message}
		var err error
		switch body.Action {
		case "message", "annotate":
			log.OK = true
		case "click":
			var resolved string
			resolved, err = sess.ResolveSelector(body.Selector)
			if err == nil {
				_, err = sess.Click(resolved)
			}
		case "fill", "type":
			var resolved string
			resolved, err = sess.ResolveSelector(body.Selector)
			if err == nil {
				_, err = sess.Fill(resolved, body.Text)
			}
		case "press":
			_, err = sess.Press(body.Key)
		default:
			err = fmt.Errorf("unsupported fpv action %q", body.Action)
		}
		if err != nil {
			log.Error = err.Error()
		} else {
			log.OK = true
		}
		entry.Audit = append(entry.Audit, log)
		if len(entry.Audit) > 100 {
			entry.Audit = entry.Audit[len(entry.Audit)-100:]
		}
		fpvShares.Store(entry.Token, entry)
		status := http.StatusOK
		if err != nil {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{"ok": log.OK, "audit": log, "mode": fpvMode(entry.Controls)})
	})

	r.Get("/{token}/screenshot.jpg", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		sess, ok := sm.Get(entry.SessionID)
		if !ok {
			writeJSON(w, http.StatusGone, map[string]string{"error": "session closed"})
			return
		}
		snap, err := sess.Screenshot("jpeg", 60)
		if err != nil {
			writeSessionError(w, http.StatusInternalServerError, classifySessionError(err), err, sess)
			return
		}
		data, err := base64.StdEncoding.DecodeString(snap.Screenshot)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "decode screenshot"})
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
}

func fpvContext() map[string]any {
	branch := fpvGit("rev-parse", "--abbrev-ref", "HEAD")
	head := fpvGit("rev-parse", "--short", "HEAD")
	status := fpvGit("status", "--short")
	recent := fpvGit("log", "-5", "--pretty=format:%h%x09%s")
	dirty := strings.TrimSpace(status) != ""
	return map[string]any{
		"project": map[string]any{
			"name":   "uiai-engine",
			"root":   "/home/wpuiai/uiai-engine",
			"branch": branch,
			"head":   head,
			"dirty":  dirty,
			"status": strings.Split(strings.TrimSpace(status), "\n"),
		},
		"tree": []map[string]any{
			{"path": "internal/routes/fpv.go", "kind": "go", "active": true, "label": "FPV route + viewer"},
			{"path": "docs/FPV_BROWSER_STYLE_GUIDE.md", "kind": "doc", "active": true, "label": "Design contract"},
			{"path": "/etc/cloudflared/wpuiai.yml", "kind": "ops", "active": false, "label": "Cloudflare /m/* ingress"},
		},
		"history": fpvHistory(recent),
		"focusa": map[string]any{
			"objective":   "Professional realtime FPV browser cockpit",
			"next_step":   "Validate streaming quality and operator controls",
			"evidence":    []string{"fpv.wpuiai.com/m/{token}", "git:" + head},
			"drift_guard": "Keep fpv.wpuiai.com path-gated to /m/*; do not expose /api/*",
		},
	}
}

func fpvGit(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = "/home/wpuiai/uiai-engine"
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func fpvHistory(logLines string) []map[string]string {
	items := []map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(logLines), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			items = append(items, map[string]string{"type": "git", "ref": parts[0], "title": parts[1]})
		}
	}
	return items
}

func fpvMode(controls bool) string {
	if controls {
		return "control"
	}
	return "read_only"
}

func fpvEntry(token string) (*fpvShare, bool) {
	v, ok := fpvShares.Load(token)
	if !ok {
		return nil, false
	}
	entry := v.(*fpvShare)
	if time.Now().UTC().After(entry.ExpiresAt) {
		fpvShares.Delete(token)
		return nil, false
	}
	return entry, true
}

func fpvToken() (string, error) {
	adjectives := []string{"bright", "calm", "clear", "cosmic", "gentle", "golden", "nimble", "quiet", "rapid", "steady", "sunny", "vivid"}
	nouns := []string{"atlas", "beacon", "comet", "harbor", "meadow", "orbit", "river", "signal", "summit", "trail", "voyage", "window"}
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate fpv token: %w", err)
	}
	return fmt.Sprintf("%s-%s-%x", adjectives[int(buf[0])%len(adjectives)], nouns[int(buf[1])%len(nouns)], buf[2:]), nil
}
