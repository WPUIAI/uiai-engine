package routes

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
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
	OneTime   bool          `json:"one_time"`
	MaxViews  int           `json:"max_views"`
	Revoked   bool          `json:"revoked"`
	Audit     []fpvAuditLog `json:"audit,omitempty"`
}

type fpvAuditLog struct {
	Seq      uint64         `json:"seq"`
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

var fpvAuditMu sync.Mutex
var fpvAuditSeq uint64
var fpvAuditEvents []fpvAuditLog

var fpvRegistryMu sync.Mutex
var fpvRegistryOnce sync.Once
var fpvStreamMu sync.Mutex
var fpvStreamViewers = map[string]int{}

func markFPVOperationalMirror(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-UIAI-Artifact-Posture", "ephemeral_non_evidence")
	w.Header().Set("X-Robots-Tag", "noindex, noarchive")
}

func MountFPVRoutes(r chi.Router, cfg *config.Config, sm *vision.SessionManager) {
	fpvLoadRegistry()
	r.Post("/share", func(w http.ResponseWriter, req *http.Request) {
		if sm == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session manager unavailable"})
			return
		}
		var body struct {
			SessionID      string `json:"session_id"`
			ExpiresMinutes int    `json:"expires_minutes"`
			Controls       bool   `json:"controls"`
			OneTime        bool   `json:"one_time"`
			MaxViews       int    `json:"max_views"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		if body.SessionID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id required"})
			return
		}
		sess, ok := sm.Get(body.SessionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		snap, err := sess.Screenshot("jpeg", 80)
		if err != nil {
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "fpv_snapshot_failed", err, "", "", "reconcile:fpv-session-snapshot")
			return
		}
		delivery, err := publishSessionSnapshotEPWA(req, cfg, sess, snap, epwadelivery.ProducerFPV)
		if err != nil {
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, "", "", "reconcile:fpv-epwa-publication")
			return
		}
		share, err := fpvCreateShare(body.SessionID, body.ExpiresMinutes, body.Controls, body.OneTime, body.MaxViews)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		status := http.StatusAccepted
		if delivery.State == epwadelivery.StateReady {
			status = http.StatusCreated
		}
		response := map[string]any{
			"schema": "uiai.fpv_share_result.v2", "delivery_state": delivery.State, "epwa_delivery": delivery,
			"operational_mirror": share, "operational_mirror_posture": "ephemeral_non_evidence",
			"token": share["token"], "session_id": share["session_id"], "expires_at": share["mirror_url_expires_at"],
		}
		if delivery.State == epwadelivery.StateReady {
			response["artifact_url"] = delivery.EPWA.RecordURL
			response["portable_url"] = delivery.EPWA.PortableURL
		}
		writeJSON(w, status, response)
	})
	r.Get("/events", func(w http.ResponseWriter, req *http.Request) {
		since := uint64(0)
		if raw := strings.TrimSpace(req.URL.Query().Get("since_seq")); raw != "" {
			_, _ = fmt.Sscan(raw, &since)
		}
		limit := 50
		if raw := strings.TrimSpace(req.URL.Query().Get("limit")); raw != "" {
			var parsed int
			if _, err := fmt.Sscan(raw, &parsed); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}
		events, latest := fpvAuditEventsSince(since, limit)
		writeJSON(w, http.StatusOK, map[string]any{"events": events, "latest_seq": latest, "count": len(events)})
	})

	r.Post("/revoke", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		if body.Token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
			return
		}
		entry, ok := fpvEntry(body.Token)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "share not found"})
			return
		}
		entry.Revoked = true
		fpvShares.Store(entry.Token, entry)
		fpvSaveRegistry()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": entry.Token, "revoked": true})
	})
}

func MountFPVPublicRoutes(r chi.Router, sm *vision.SessionManager) {
	fpvLoadRegistry()
	r.Get("/assets/{file}", func(w http.ResponseWriter, req *http.Request) {
		assetPath, ok := fpvAssetPath(filepath.Base(chi.URLParam(req, "file")))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "asset not found"})
			return
		}
		http.ServeFile(w, req, assetPath)
	})
	r.Get("/{token}", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		if entry.Revoked {
			writeJSON(w, http.StatusGone, map[string]string{"error": "fpv share revoked"})
			return
		}
		if (entry.OneTime || entry.MaxViews > 0) && entry.Views >= fpvAllowedViews(entry) {
			writeJSON(w, http.StatusGone, map[string]string{"error": "fpv share view limit reached"})
			return
		}
		entry.Views++
		fpvShares.Store(entry.Token, entry)
		fpvSaveRegistry()
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
			"token":            entry.Token,
			"session_id":       entry.SessionID,
			"mode":             fpvMode(entry.Controls),
			"controls":         entry.Controls,
			"audit_count":      len(entry.Audit),
			"audit":            entry.Audit,
			"views":            entry.Views,
			"one_time":         entry.OneTime,
			"max_views":        entry.MaxViews,
			"revoked":          entry.Revoked,
			"viewers":          fpvViewerCount(entry.Token),
			"created_at":       entry.CreatedAt,
			"url":              sess.URL,
			"title":            sess.Title,
			"width":            sess.Width,
			"height":           sess.Height,
			"expires_at":       entry.ExpiresAt,
			"evidence_posture": "ephemeral_non_evidence",
			"diagnostics":      diag.Summary,
			"transport": map[string]any{
				"primary":       "cdp_screencast",
				"stream_url":    "/m/" + entry.Token + "/stream.cdp.mjpg",
				"mjpeg_url":     "/m/" + entry.Token + "/stream.mjpg",
				"fallback_url":  "/m/" + entry.Token + "/screenshot.jpg",
				"quality_modes": []string{"smooth", "balanced", "saver"},
				"max_viewers":   3,
			},
			"context": fpvContext(sess),
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
			X        int    `json:"x"`
			Y        int    `json:"y"`
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
		case "click_xy":
			var resolved string
			_, resolved, err = sess.ClickAt(body.X, body.Y)
			log.Selector = resolved
			log.Meta = map[string]any{"x": body.X, "y": body.Y}
		case "selector_at":
			var resolved string
			resolved, err = sess.SelectorAt(body.X, body.Y)
			log.Selector = resolved
			log.Meta = map[string]any{"x": body.X, "y": body.Y}
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
		log = fpvRecordAuditEvent(entry.Token, entry.SessionID, log)
		entry.Audit = append(entry.Audit, log)
		if len(entry.Audit) > 100 {
			entry.Audit = entry.Audit[len(entry.Audit)-100:]
		}
		fpvShares.Store(entry.Token, entry)
		fpvSaveRegistry()
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
		data, err := fpvScreenshotJPEG(sess)
		if err != nil {
			writeSessionError(w, http.StatusInternalServerError, classifySessionError(err), err, sess)
			return
		}
		markFPVOperationalMirror(w)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(data)
	})

	r.Get("/{token}/stream.cdp.mjpg", func(w http.ResponseWriter, req *http.Request) {
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
		if !fpvAcquireViewer(entry.Token) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many viewers for share"})
			return
		}
		defer fpvReleaseViewer(entry.Token)
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
			return
		}
		markFPVOperationalMirror(w)
		w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
		w.Header().Set("X-Accel-Buffering", "no")
		frames, stop, err := sess.CDPScreencast(req.Context(), 60, 1)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer stop()
		for {
			select {
			case <-req.Context().Done():
				return
			case data, ok := <-frames:
				if !ok {
					return
				}
				if _, alive := fpvEntry(entry.Token); !alive {
					return
				}
				_, _ = fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(data))
				_, _ = w.Write(data)
				_, _ = fmt.Fprint(w, "\r\n")
				flusher.Flush()
			}
		}
	})

	r.Get("/{token}/stream.mjpg", func(w http.ResponseWriter, req *http.Request) {
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
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
			return
		}
		if !fpvAcquireViewer(entry.Token) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many viewers for share"})
			return
		}
		defer fpvReleaseViewer(entry.Token)
		markFPVOperationalMirror(w)
		w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
		w.Header().Set("X-Accel-Buffering", "no")
		ticker := time.NewTicker(fpvStreamInterval(entry.Token))
		defer ticker.Stop()
		for {
			select {
			case <-req.Context().Done():
				return
			case <-ticker.C:
				if _, alive := fpvEntry(entry.Token); !alive {
					return
				}
				data, err := fpvScreenshotJPEG(sess)
				if err != nil {
					return
				}
				_, _ = fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(data))
				_, _ = w.Write(data)
				_, _ = fmt.Fprint(w, "\r\n")
				flusher.Flush()
			}
		}
	})
}

func fpvAssetPath(file string) (string, bool) {
	switch file {
	case "fpv.css", "fpv.js", "fpv-assets.json":
	default:
		return "", false
	}
	for _, dir := range []string{filepath.Join("web", "fpv", "dist"), filepath.Join("web", "fpv")} {
		path := filepath.Join(dir, file)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func fpvCreateShare(sessionID string, expiresMinutes int, controls, oneTime bool, maxViews int) (map[string]any, error) {
	minutes := expiresMinutes
	if minutes <= 0 || minutes > 240 {
		minutes = 60
	}
	token, err := fpvToken()
	if err != nil {
		return nil, err
	}
	entry := &fpvShare{Token: token, SessionID: sessionID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(minutes) * time.Minute), Controls: controls, OneTime: oneTime, MaxViews: maxViews}
	fpvShares.Store(token, entry)
	fpvSaveRegistry()
	return fpvSharePayload(entry), nil
}

func fpvSharePayload(entry *fpvShare) map[string]any {
	return map[string]any{
		"token":                 entry.Token,
		"session_id":            entry.SessionID,
		"mirror_url":            "/m/" + entry.Token,
		"status_url":            "/m/" + entry.Token + "/status",
		"screenshot_url":        "/m/" + entry.Token + "/screenshot.jpg",
		"stream_url":            "/m/" + entry.Token + "/stream.cdp.mjpg",
		"public_url":            "https://fpv.wpuiai.com/m/" + entry.Token,
		"mirror_url_expires_at": entry.ExpiresAt,
		"mode":                  fpvMode(entry.Controls),
		"controls":              entry.Controls,
		"one_time":              entry.OneTime,
		"max_views":             entry.MaxViews,
	}
}

func fpvRecordAuditEvent(token, sessionID string, log fpvAuditLog) fpvAuditLog {
	fpvAuditMu.Lock()
	defer fpvAuditMu.Unlock()
	fpvAuditSeq++
	log.Seq = fpvAuditSeq
	if log.Meta == nil {
		log.Meta = map[string]any{}
	}
	log.Meta["token"] = token
	log.Meta["session_id"] = sessionID
	fpvAuditEvents = append(fpvAuditEvents, log)
	if len(fpvAuditEvents) > 500 {
		fpvAuditEvents = fpvAuditEvents[len(fpvAuditEvents)-500:]
	}
	return log
}

func fpvAuditEventsSince(since uint64, limit int) ([]fpvAuditLog, uint64) {
	fpvAuditMu.Lock()
	defer fpvAuditMu.Unlock()
	latest := fpvAuditSeq
	items := []fpvAuditLog{}
	for _, event := range fpvAuditEvents {
		if event.Seq > since {
			items = append(items, event)
		}
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items, latest
}

func fpvAllowedViews(entry *fpvShare) int {
	if entry.OneTime {
		return 1
	}
	if entry.MaxViews > 0 {
		return entry.MaxViews
	}
	return 1 << 30
}

func fpvViewerCount(token string) int {
	fpvStreamMu.Lock()
	defer fpvStreamMu.Unlock()
	return fpvStreamViewers[token]
}

func fpvAcquireViewer(token string) bool {
	fpvStreamMu.Lock()
	defer fpvStreamMu.Unlock()
	if fpvStreamViewers[token] >= 3 {
		return false
	}
	fpvStreamViewers[token]++
	return true
}

func fpvReleaseViewer(token string) {
	fpvStreamMu.Lock()
	defer fpvStreamMu.Unlock()
	if fpvStreamViewers[token] > 0 {
		fpvStreamViewers[token]--
	}
}

func fpvStreamInterval(token string) time.Duration {
	viewers := fpvViewerCount(token)
	if viewers >= 3 {
		return 750 * time.Millisecond
	}
	if viewers == 2 {
		return 500 * time.Millisecond
	}
	return 250 * time.Millisecond
}

func fpvRegistryPath() string {
	wd, err := os.Getwd()
	if err == nil {
		for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return filepath.Join(dir, "data", "fpv-shares.json")
			}
		}
	}
	return filepath.Join("data", "fpv-shares.json")
}

func fpvLoadRegistry() {
	fpvRegistryOnce.Do(func() {
		data, err := os.ReadFile(fpvRegistryPath())
		if err != nil {
			return
		}
		var entries []*fpvShare
		if json.Unmarshal(data, &entries) != nil {
			return
		}
		now := time.Now().UTC()
		for _, entry := range entries {
			if entry != nil && entry.Token != "" && now.Before(entry.ExpiresAt) && !entry.Revoked {
				fpvShares.Store(entry.Token, entry)
			}
		}
	})
}

func fpvSaveRegistry() {
	fpvRegistryMu.Lock()
	defer fpvRegistryMu.Unlock()
	entries := []*fpvShare{}
	now := time.Now().UTC()
	fpvShares.Range(func(_, v any) bool {
		entry, ok := v.(*fpvShare)
		if ok && entry != nil && now.Before(entry.ExpiresAt) && !entry.Revoked {
			entries = append(entries, entry)
		}
		return true
	})
	_ = os.MkdirAll(filepath.Dir(fpvRegistryPath()), 0750)
	data, _ := json.MarshalIndent(entries, "", "  ")
	_ = os.WriteFile(fpvRegistryPath(), data, 0600)
}

func fpvScreenshotJPEG(sess *vision.Session) ([]byte, error) {
	snap, err := sess.Screenshot("jpeg", 60)
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(snap.Screenshot)
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}
	return data, nil
}

func fpvContext(sess *vision.Session) map[string]any {
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
		"focusa":  fpvFocusaContext(sess, head),
		"lifecycle": map[string]any{
			"expired_state": "Shows Session ended and stops polling/stream retries",
			"reopen_hint":   "Create a fresh share from a live local /api/fpv/share call",
		},
	}
}

func fpvFocusaContext(sess *vision.Session, head string) map[string]any {
	base := map[string]any{
		"source":      "session.focusa_scope",
		"evidence":    []string{"fpv.wpuiai.com/m/{token}", "git:" + head},
		"drift_guard": "Keep fpv.wpuiai.com path-gated to /m/*; do not expose /api/*",
	}
	if sess == nil || sess.FocusaScope == nil || (sess.FocusaScope.WorkpointID == "" && sess.FocusaScope.ContinuityID == "" && sess.FocusaScope.ProjectRoot == "" && sess.FocusaScope.EvidenceRef == "") {
		base["status"] = "degraded"
		base["degraded"] = true
		base["objective"] = "No Focusa scope attached to this UIAI session"
		base["next_step"] = "Open or share the session with focusa_scope for Workpoint/evidence/prediction/trajectory linkage"
		base["workpoint"] = "unavailable"
		base["trajectory"] = "unavailable"
		base["prediction"] = "unavailable"
		return base
	}
	scope := sess.FocusaScope
	evidence := []string{"fpv.wpuiai.com/m/{token}", "git:" + head}
	if scope.EvidenceRef != "" {
		evidence = append(evidence, scope.EvidenceRef)
	}
	base["status"] = "linked"
	base["degraded"] = false
	base["objective"] = "Linked Focusa scope for this UIAI browser session"
	base["next_step"] = "Resolve compact Workpoint/evidence/prediction/trajectory surfaces by project_root plus continuity_id"
	base["workpoint"] = scope.WorkpointID
	base["continuity_id"] = scope.ContinuityID
	base["project_root"] = scope.ProjectRoot
	base["trajectory"] = map[string]string{"project_root": scope.ProjectRoot, "continuity_id": scope.ContinuityID, "status": "scope_linked"}
	base["prediction"] = map[string]string{"status": "scope_linked", "continuity_id": scope.ContinuityID}
	base["evidence"] = evidence
	live := fpvLiveFocusaContext(scope)
	base["live"] = live
	if live["status"] == "linked" {
		base["status"] = "live"
		base["objective"] = "Live Focusa adapter linked for this UIAI browser session"
		base["trajectory"] = live["trajectory"]
		base["prediction"] = map[string]any{"status": "live_adapter", "continuity_id": scope.ContinuityID, "live_status": live["status"]}
	}
	return base
}

func fpvLiveFocusaContext(scope *vision.FocusaScope) map[string]any {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FOCUSA_DAEMON_URL")), "/")
	if baseURL == "" {
		return map[string]any{"status": "disabled", "degraded": true, "reason": "FOCUSA_DAEMON_URL unset"}
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return map[string]any{"status": "degraded", "degraded": true, "reason": "FOCUSA_DAEMON_URL must be http(s)"}
	}
	payload := map[string]any{
		"project_root":  scope.ProjectRoot,
		"continuity_id": scope.ContinuityID,
		"workpoint_id":  scope.WorkpointID,
	}
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	workpoint := fpvPostFocusa(client, baseURL+"/v1/workpoint/resume", withPayload(payload, "mode", "compact_prompt"))
	trajectory := fpvPostFocusa(client, baseURL+"/v1/trajectory/view", withPayload(payload, "mode", "summary"))
	status := "linked"
	if workpoint["status"] != "ok" && trajectory["status"] != "ok" {
		status = "degraded"
	}
	return map[string]any{
		"status":     status,
		"degraded":   status != "linked",
		"source":     "FOCUSA_DAEMON_URL",
		"workpoint":  workpoint,
		"trajectory": trajectory,
	}
}

func withPayload(src map[string]any, key string, value any) map[string]any {
	copy := map[string]any{}
	for k, v := range src {
		if v != "" {
			copy[k] = v
		}
	}
	copy[key] = value
	return copy
}

func fpvPostFocusa(client *http.Client, url string, payload map[string]any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{"status": "degraded", "error": "encode_failed"}
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return map[string]any{"status": "degraded", "error": "request_failed"}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{"status": "degraded", "error": "unreachable"}
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&decoded)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return map[string]any{"status": "degraded", "http_status": resp.StatusCode, "summary": fpvCompactFocusaSummary(decoded)}
	}
	return map[string]any{"status": "ok", "http_status": resp.StatusCode, "summary": fpvCompactFocusaSummary(decoded)}
}

func fpvCompactFocusaSummary(decoded map[string]any) string {
	if len(decoded) == 0 {
		return "no compact payload"
	}
	for _, key := range []string{"rendered_summary", "summary", "status", "trajectory_id", "workpoint_id", "current_action", "next_action"} {
		if v, ok := decoded[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(v))
			if len(text) > 180 {
				return text[:180] + "…"
			}
			if text != "" {
				return text
			}
		}
	}
	return "compact payload available"
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
	if time.Now().UTC().After(entry.ExpiresAt) || entry.Revoked {
		fpvShares.Delete(token)
		fpvSaveRegistry()
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
