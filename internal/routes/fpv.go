package routes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

type fpvShare struct {
	Token     string    `json:"token"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Views     int       `json:"views"`
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
		entry := &fpvShare{Token: token, SessionID: body.SessionID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(minutes) * time.Minute)}
		fpvShares.Store(token, entry)
		writeJSON(w, http.StatusOK, map[string]any{
			"token":                 token,
			"session_id":            body.SessionID,
			"mirror_url":            "/m/" + token,
			"status_url":            "/m/" + token + "/status",
			"screenshot_url":        "/m/" + token + "/screenshot.jpg",
			"mirror_url_expires_at": entry.ExpiresAt,
			"mode":                  "read_only",
		})
	})
}

func MountFPVPublicRoutes(r chi.Router, sm *vision.SessionManager) {
	r.Get("/{token}", func(w http.ResponseWriter, req *http.Request) {
		entry, ok := fpvEntry(chi.URLParam(req, "token"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "fpv share not found or expired"})
			return
		}
		entry.Views++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = fpvPageTemplate.Execute(w, map[string]any{"Token": entry.Token, "SessionID": entry.SessionID})
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
			"mode":        "read_only",
			"url":         sess.URL,
			"title":       sess.Title,
			"width":       sess.Width,
			"height":      sess.Height,
			"expires_at":  entry.ExpiresAt,
			"diagnostics": diag.Summary,
		})
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
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate fpv token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

var fpvPageTemplate = template.Must(template.New("fpv").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>UIAI FPV</title><style>body{font-family:system-ui;margin:0;background:#111;color:#eee}header{padding:12px;background:#1d1d1d;position:sticky;top:0}main{padding:12px}img{width:100%;height:auto;border:1px solid #333;background:#000}.muted{color:#aaa;font-size:12px}pre{white-space:pre-wrap;background:#181818;padding:8px;border-radius:8px}</style></head>
<body><header><strong>● Watching UIAI session {{.SessionID}}</strong><div class="muted">read-only FPV share</div></header><main><img id="shot" alt="live browser screenshot"><pre id="status">loading…</pre></main>
<script>
const token={{printf "%q" .Token}};
async function tick(){
  const st=await fetch('/m/'+token+'/status',{cache:'no-store'}).then(r=>r.json());
  document.getElementById('status').textContent=JSON.stringify(st,null,2);
  document.getElementById('shot').src='/m/'+token+'/screenshot.jpg?t='+Date.now();
}
tick(); setInterval(tick, 2000);
</script></body></html>`))
