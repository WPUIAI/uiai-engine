package routes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
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

var fpvPageTemplate = template.Must(template.New("fpv").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>UIAI FPV</title><style>
:root{color-scheme:dark;--bg:#050913;--bg2:#0b1220;--panel:rgba(11,18,32,.82);--panel-strong:rgba(15,23,42,.95);--card:rgba(2,6,23,.58);--text:#f8fafc;--sub:#b6c2d2;--muted:#7d8aa0;--line:rgba(148,163,184,.18);--line2:rgba(148,163,184,.30);--cyan:#67e8f9;--blue:#60a5fa;--violet:#a78bfa;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--radius:5px;--gap:clamp(10px,1vw,16px);--pad:clamp(13px,1vw,17px);--fs-xs:clamp(11px,.72vw,12.5px);--fs-sm:clamp(13px,.86vw,14.5px);--fs-md:clamp(15px,1vw,17px);--fs-lg:clamp(19px,1.35vw,24px);--fs-xl:clamp(30px,2.15vw,42px);--ease:cubic-bezier(.16,1,.3,1);--spring:cubic-bezier(.34,1.56,.64,1);--shadow:0 28px 90px rgba(0,0,0,.52)}*{box-sizing:border-box}html,body{height:100%;margin:0;overflow:hidden}body{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:radial-gradient(900px 620px at 14% -14%,rgba(103,232,249,.20),transparent 58%),radial-gradient(820px 560px at 95% 5%,rgba(167,139,250,.18),transparent 54%),linear-gradient(135deg,var(--bg),var(--bg2) 60%,#020617);color:var(--text);font-size:var(--fs-sm);line-height:1.45;letter-spacing:-.012em;text-rendering:geometricPrecision;-webkit-font-smoothing:antialiased}body:before{content:"";position:fixed;inset:0;pointer-events:none;background-image:linear-gradient(rgba(255,255,255,.035) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.026) 1px,transparent 1px);background-size:56px 56px;mask-image:linear-gradient(to bottom,rgba(0,0,0,.65),transparent 72%)}header{height:68px;padding:12px clamp(14px,1.7vw,24px);display:flex;align-items:center;justify-content:space-between;gap:16px;border-bottom:1px solid var(--line);background:rgba(2,6,23,.68);backdrop-filter:blur(24px);position:relative;z-index:5}.brand{display:flex;align-items:center;gap:12px;min-width:0}.mark{width:38px;height:38px;border-radius:5px;display:grid;place-items:center;background:linear-gradient(135deg,var(--cyan),var(--violet));color:#020617;font-weight:950;box-shadow:0 14px 38px rgba(103,232,249,.22)}.title{font-size:var(--fs-lg);font-weight:900;letter-spacing:-.045em;line-height:1.02;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.sub{color:var(--muted);font-size:var(--fs-xs);line-height:1.25;margin-top:3px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.statusbar{display:flex;gap:9px;align-items:center;flex:0 0 auto}.chip{display:inline-flex;align-items:center;gap:7px;padding:7px 10px;border:1px solid var(--line);border-radius:5px;background:rgba(15,23,42,.72);font-size:var(--fs-xs);font-weight:800;color:#dbeafe}.led{width:8px;height:8px;border-radius:5px;background:var(--green);box-shadow:0 0 0 5px rgba(52,211,153,.10),0 0 18px var(--green);animation:pulse 2.4s var(--ease) infinite}main{height:calc(100vh - 68px);display:grid;grid-template-columns:minmax(0,1.65fr) minmax(400px,.95fr);gap:var(--gap);padding:var(--gap)}.stage{min-width:0;display:flex}.browser{width:100%;border:1px solid var(--line2);border-radius:var(--radius);overflow:hidden;background:linear-gradient(180deg,rgba(15,23,42,.92),rgba(2,6,23,.96));box-shadow:var(--shadow);display:flex;flex-direction:column;position:relative}.browser:before{content:"";position:absolute;inset:0;pointer-events:none;border-radius:inherit;background:linear-gradient(135deg,rgba(255,255,255,.13),transparent 20%,transparent 74%,rgba(103,232,249,.10));mix-blend-mode:screen}.browser:after{content:"";position:absolute;inset:52px 0 auto;height:24%;pointer-events:none;background:linear-gradient(180deg,transparent,rgba(103,232,249,.08),transparent);animation:scan 4.8s linear infinite;opacity:.38}.chrome{height:52px;display:grid;grid-template-columns:76px 1fr auto;gap:12px;align-items:center;padding:0 14px;background:rgba(15,23,42,.88);border-bottom:1px solid var(--line);backdrop-filter:blur(18px)}.lights{display:flex;gap:8px}.light{width:12px;height:12px;border-radius:5px;}.red{background:#ff5f57}.yellow{background:#febc2e}.green{background:#28c840}.addr{height:31px;display:flex;align-items:center;gap:8px;padding:0 13px;border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.72);color:#dbeafe;font-size:var(--fs-xs);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.lock{color:var(--green)}.chrome-meta{font-size:10px;color:var(--muted);font-weight:900;text-transform:uppercase;letter-spacing:.08em}.viewport{flex:1;min-height:0;display:grid;place-items:center;background:#000}.viewport img{width:100%;height:100%;object-fit:contain;background:#000;transition:opacity .12s var(--ease)}.rail{height:100%;border:1px solid var(--line2);border-radius:var(--radius);overflow:hidden;background:var(--panel);backdrop-filter:blur(24px);box-shadow:var(--shadow);display:flex;flex-direction:column}.rail-head{padding:var(--pad);border-bottom:1px solid var(--line);background:rgba(15,23,42,.90)}.rail-title{display:flex;align-items:center;gap:9px;font-size:var(--fs-lg);font-weight:950;letter-spacing:-.055em}.rail-sub{color:var(--muted);font-size:var(--fs-xs);margin-top:3px}.signal{height:5px;margin-top:12px;border-radius:5px;background:linear-gradient(90deg,var(--green),var(--cyan),var(--violet),var(--green));background-size:220% 100%;animation:flow 3.2s linear infinite}.tabs{display:grid;grid-template-columns:repeat(3,1fr);gap:7px;padding:10px;border-bottom:1px solid var(--line);background:rgba(2,6,23,.22)}.tab{border:1px solid transparent;border-radius:5px;background:transparent;color:var(--sub);padding:9px 8px;font-weight:900;font-size:clamp(11.5px,.76vw,12.5px);line-height:1;display:flex;gap:6px;align-items:center;justify-content:center;cursor:pointer}.tab.active{color:#fff;background:rgba(103,232,249,.10);border-color:rgba(103,232,249,.34);box-shadow:inset 0 1px rgba(255,255,255,.06)}.tab-panel{display:none;flex:1;overflow:auto;padding:var(--pad);scrollbar-width:thin}.tab-panel.active{display:block}.section-title{display:flex;align-items:center;justify-content:space-between;margin:0 0 12px;font-size:11.5px;line-height:1.2;text-transform:uppercase;letter-spacing:.12em;color:#dbeafe}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.card{position:relative;overflow:hidden;background:var(--card);border:1px solid var(--line);border-radius:5px;padding:13px;min-height:116px;transition:transform .28s var(--ease),border-color .2s var(--ease),background .28s var(--ease)}.card:hover{transform:translateY(-1px);border-color:rgba(103,232,249,.38)}.card:after{content:"";position:absolute;inset:0;background:linear-gradient(115deg,transparent 38%,rgba(255,255,255,.14) 50%,transparent 62%);transform:translateX(-110%) skewX(-12deg);opacity:0;pointer-events:none}.card:hover:after,.card.sweep:after{animation:null-shimmer 1.05s var(--ease)}.label{display:flex;align-items:center;gap:6px;color:var(--muted);font-size:11px;line-height:1.15;font-weight:950;text-transform:uppercase;letter-spacing:.09em}.value{margin-top:8px;font-size:var(--fs-md);font-weight:850;line-height:1.22;word-break:break-word}.value.hero{font-size:var(--fs-xl);letter-spacing:-.06em}.hint{margin-top:6px;color:var(--muted);font-size:var(--fs-xs);line-height:1.35}.ok{color:var(--green)}.warn{color:var(--amber)}.bad{color:var(--red)}.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:11px}.span2{grid-column:1/-1}.tree{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;line-height:1.65;color:#cbd5e1;background:rgba(2,6,23,.45);border:1px solid var(--line);border-radius:5px;padding:12px}.timeline{display:grid;gap:9px}.event{display:grid;grid-template-columns:34px 1fr;gap:9px;padding:10px;border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.46);animation:stream-in .42s var(--spring)}.glyph{width:28px;height:28px;border-radius:5px;display:grid;place-items:center;background:rgba(103,232,249,.10);border:1px solid rgba(103,232,249,.20)}.event-title{font-weight:900;line-height:1.2}.event-meta{color:var(--muted);font-size:var(--fs-xs);line-height:1.35;margin-top:3px}.controlbox{display:grid;gap:10px}.quality{display:grid;grid-template-columns:repeat(3,1fr);gap:6px}.quality button,.keyrow button{border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.55);color:var(--sub);padding:9px;font-weight:900}.quality button.active{border-color:rgba(103,232,249,.55);color:#fff;background:rgba(103,232,249,.12)}.fieldgrid{display:grid;gap:8px}.fieldgrid input{width:100%;border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.62);color:var(--text);padding:10px}.keyrow{display:grid;grid-template-columns:repeat(4,1fr);gap:6px}.filterbar{display:grid;grid-template-columns:repeat(5,1fr);gap:6px;margin-bottom:10px}.filterbar button{border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.48);color:var(--sub);padding:7px;font-size:11px;font-weight:900}.filterbar button.active{color:#fff;border-color:rgba(103,232,249,.45)}textarea{width:100%;min-height:96px;resize:vertical;border:1px solid var(--line);border-radius:5px;background:rgba(2,6,23,.62);color:var(--text);padding:11px;outline:none}textarea:focus{border-color:rgba(103,232,249,.62);box-shadow:0 0 0 4px rgba(103,232,249,.10)}button.action{border:0;border-radius:5px;padding:12px 14px;background:linear-gradient(135deg,var(--cyan),var(--violet));color:#020617;font-weight:950;cursor:pointer}.empty{color:var(--muted);font-size:var(--fs-sm);padding:16px;border:1px dashed var(--line);border-radius:5px;background:rgba(2,6,23,.30)}.tag{letter-spacing:.12em;color:var(--muted);opacity:.72;transition:opacity .25s var(--ease);font:800 9px/1 ui-monospace,SFMono-Regular,Menlo,monospace;position:absolute;top:8px;right:9px;max-width:72px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.card:hover .tag,.tag.always{opacity:1}.shine{display:none}.card.sweep:after,.card:hover:after{animation:null-shimmer 1.05s var(--ease) forwards}.data-viz{margin-top:10px}.seismo-grid{height:74px;display:grid;grid-template-columns:repeat(48,1fr);align-items:end;gap:2px;padding:10px;background:rgba(2,6,23,.48);border:1px solid var(--line);border-radius:5px;overflow:hidden}.seismo-grid i{display:block;min-height:3px;background:linear-gradient(180deg,var(--cyan),rgba(96,165,250,.45));border-radius:2px;transform-origin:bottom;animation:seg-in .28s var(--ease) both}.glyph-grid,.health-grid{display:grid;grid-template-columns:repeat(12,1fr);gap:3px;padding:10px;background:rgba(2,6,23,.48);border:1px solid var(--line);border-radius:5px;}.glyph-grid i,.health-grid i{aspect-ratio:1;background:#1c1c1c;border-radius:2px;display:block;transform:scale(0);animation:cell-slam .38s var(--spring) forwards;transition:background .18s var(--ease),filter .22s var(--ease)}.glyph-grid i.l1,.health-grid i.l1{background:#153345}.glyph-grid i.l2,.health-grid i.l2{background:#1d5b73}.glyph-grid i.l3,.health-grid i.l3{background:#2a8ead}.glyph-grid i.l4,.health-grid i.l4{background:#67e8f9}.glyph-grid i.warn,.health-grid i.warn{background:var(--amber);animation:streak-wave 1.4s var(--ease) infinite}.glyph-grid i.bad,.health-grid i.bad{background:var(--red);animation:streak-wave .9s var(--ease) infinite}.ticker{display:flex;gap:6px;align-items:center;overflow:hidden;white-space:nowrap;color:var(--muted);font-size:var(--fs-xs);font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.ticker b{color:var(--text)}@keyframes pulse{0%,100%{transform:scale(1);opacity:1}50%{transform:scale(.72);opacity:.58}}@keyframes scan{0%{transform:translateY(-35%)}100%{transform:translateY(380%)}}@keyframes flow{0%{background-position:0 50%}100%{background-position:220% 50%}}@keyframes shimmer{0%{opacity:0;transform:translateX(-55%) skewX(-12deg)}18%{opacity:1}100%{opacity:0;transform:translateX(55%) skewX(-12deg)}}@keyframes stream-in{0%{opacity:0;transform:translateY(10px) scale(.985)}100%{opacity:1;transform:none}}@media (prefers-reduced-motion:reduce){*,*:before,*:after{animation:none!important;transition:none!important}}@media (max-width:1199px) and (min-width:1024px){main{grid-template-columns:minmax(0,1.45fr) minmax(380px,.95fr)}.rail .grid{grid-template-columns:1fr}.tabs{grid-template-columns:repeat(2,1fr)}}@media (max-width:1023px){html,body{height:auto;overflow:auto}main{height:auto;grid-template-columns:1fr}.browser{height:clamp(390px,62vh,660px)}.rail{height:auto;min-height:620px}.tab-panel{max-height:none}}@media (max-width:767px){header{height:auto;padding:12px}.statusbar{display:none}main{padding:10px}.chrome{grid-template-columns:54px 1fr}.chrome-meta{display:none}.tabs{grid-template-columns:repeat(2,1fr)}.grid{grid-template-columns:1fr}.browser{height:min(58vh,520px)}.rail-title{font-size:18px}.card{min-height:112px;padding:12px}}@media (max-width:420px){.rail-title{font-size:18px}.tab{justify-content:flex-start}.browser{height:58vh}}
</style></head>
<body><header><div class="brand"><div class="mark">◈</div><div><div class="title">UIAI First‑Person View</div><div class="sub">Browser session <span class="mono">{{.SessionID}}</span> · <span id="mode">connecting</span></div></div></div><div class="statusbar"><div class="chip"><span class="led"></span><span id="fps">starting</span></div><div class="chip" id="modeChip">Live</div></div></header>
<main><section class="stage"><div class="browser"><div class="chrome"><div class="lights"><span class="light red"></span><span class="light yellow"></span><span class="light green"></span></div><div id="address" class="addr"><span class="lock">●</span><span>loading URL…</span></div><div id="chromeMeta" class="chrome-meta">FPV Live</div></div><div class="viewport"><img id="shot" alt="live browser screenshot"></div></div></section>
<aside class="rail"><div class="rail-head"><div class="rail-title"><span>✦</span><span>Realtime Control Center</span></div><div class="rail-sub">Categorized browser, diagnostics, repo, Focusa, and audit context.</div><div class="signal"></div><div class="ticker" id="ticker"><b>LIVE</b><span>waiting for telemetry frames</span></div></div><nav class="tabs"><button class="tab active" data-tab="overview">◉ Live</button><button class="tab" data-tab="diagnostics">◇ Health</button><button class="tab" data-tab="control">✎ Actions</button><button class="tab" data-tab="repo">⌘ Repo</button><button class="tab" data-tab="focusa">✦ Focusa</button><button class="tab" data-tab="timeline">↯ History</button></nav>
<section id="overview" class="tab-panel active"><h3 class="section-title">Live signal <span class="mono">/m/{{.Token}}</span></h3><div class="grid"><div class="card sweep"><span class="shine"></span><span class="tag always">RAF</span><div class="label">↻ Frame rate</div><div id="fpsNum" class="value hero">0.0</div><div class="hint">Screenshot polling cadence</div></div><div class="card"><span class="shine"></span><span class="tag">MODE</span><div class="label">◉ Access mode</div><div id="modeCard" class="value">loading</div><div class="hint">Read-only or operator control</div></div><div class="card span2"><span class="shine"></span><span class="tag">URL</span><div class="label">🌐 Current page</div><div id="pageUrl" class="value">loading</div><div id="pageTitle" class="hint">loading title</div></div><div class="card"><div class="label">▣ Viewport</div><div id="viewportSize" class="value">loading</div></div><div class="card"><div class="label">⏱ Expires</div><div id="expires" class="value">loading</div></div><div class="card"><div class="label">👁 Viewer opens</div><div id="views" class="value">loading</div></div><div class="card"><span class="shine"></span><span class="tag">SIGNAL</span><div class="label">⌁ Stream quality</div><div id="streamQuality" class="value ok">Healthy</div></div></div><div class="data-viz"><div class="section-title">Input seismograph <span class="mono">CH 01</span></div><div id="seismo" class="seismo-grid"></div></div><div class="data-viz"><div class="section-title">Glyph signal grid <span class="mono">LIVE</span></div><div id="glyphGrid" class="glyph-grid"></div></div></section>
<section id="diagnostics" class="tab-panel"><h3 class="section-title">Runtime health</h3><div class="grid"><div class="card"><div class="label">⚠ Console errors</div><div id="consoleErrors" class="value hero ok">0</div></div><div class="card"><div class="label">⇣ Failed requests</div><div id="failedRequests" class="value hero ok">0</div></div><div class="card"><div class="label">4xx / 5xx</div><div id="httpErrors" class="value hero ok">0 / 0</div></div><div class="card"><div class="label">⇄ Requests</div><div id="requests" class="value hero">0</div></div></div><div class="data-viz"><div class="section-title">Health matrix <span class="mono">ERR/NET</span></div><div id="healthGrid" class="health-grid"></div></div></section>
<section id="repo" class="tab-panel"><h3 class="section-title">Repository context</h3><div id="repoGrid" class="grid"></div><h3 class="section-title">Active tree</h3><div id="repoTree" class="tree">Loading repo tree…</div></section>
<section id="focusa" class="tab-panel"><h3 class="section-title">Focusa context</h3><div id="focusaGrid" class="grid"></div></section>
<section id="timeline" class="tab-panel"><h3 class="section-title">History timeline</h3><div class="filterbar"><button class="active" onclick="setFilter('all')">All</button><button onclick="setFilter('frame')">Frames</button><button onclick="setFilter('action')">Actions</button><button onclick="setFilter('git')">Git</button><button onclick="setFilter('focusa')">Focusa</button></div><div id="stream" class="timeline"></div></section>
<section id="control" class="tab-panel"><h3 class="section-title">Operator actions</h3><div class="controlbox"><div class="label">Stream quality</div><div class="quality"><button id="qSmooth" class="active" onclick="setQuality('smooth')">Smooth</button><button id="qBalanced" onclick="setQuality('balanced')">Balanced</button><button id="qSaver" onclick="setQuality('saver')">Saver</button></div><textarea id="msg" placeholder="Send a note to the audited action stream"></textarea><button class="action" onclick="sendMsg()">Send audited note</button><div class="fieldgrid"><input id="selector" placeholder="CSS or text selector, e.g. text=Learn more"><input id="fillText" placeholder="Text for fill/type action"></div><button class="action" onclick="sendClick()">Click selector</button><button class="action" onclick="sendFill()">Fill selector</button><div class="keyrow"><button onclick="sendKey('Enter')">Enter</button><button onclick="sendKey('Escape')">Esc</button><button onclick="sendKey('Tab')">Tab</button><button onclick="sendKey('ArrowDown')">↓</button></div><div id="controlResult" class="hint"></div></div></section>
</aside></main>
<script>
const token='{{.Token}}'; let frames=0; let started=Date.now(); let stream=[]; let active=true; let frameErrors=0; let frameMs=250; let frameTimer=null; let eventFilter='all';
const $=id=>document.getElementById(id); function setText(id,v){const el=$(id); if(el) el.textContent=(v ?? '—')}
function seedCells(id,count=48){const el=$(id); if(!el||el.dataset.ready)return; el.dataset.ready='1'; el.innerHTML=Array.from({length:count},(_,i)=>'<i style="animation-delay:'+((i%18)*18)+'ms"></i>').join('')}
function renderSeismo(){const el=$('seismo'); if(!el)return; seedCells('seismo',48); [...el.children].forEach((bar,i)=>{const h=12+Math.abs(Math.sin((frames+i)*.42))*62+(i%7)*2; bar.style.height=Math.min(72,h)+'px'; bar.className=frameErrors?'warn':''})}
function renderGlyph(id,level=1,bad=false){const el=$(id); if(!el)return; seedCells(id,72); [...el.children].forEach((cell,i)=>{const hot=((i+frames)%13)<level+2; cell.className=bad&&hot?'bad':hot?'l'+(1+((i+frames)%4)):'';})}
function sweepCards(){document.querySelectorAll('.card').forEach((c,i)=>{if((frames+i)%11===0){c.classList.add('sweep');setTimeout(()=>c.classList.remove('sweep'),1100)}})}
document.querySelectorAll('.tab').forEach(btn=>btn.onclick=()=>{document.querySelectorAll('.tab').forEach(b=>b.classList.toggle('active',b===btn));document.querySelectorAll('.tab-panel').forEach(p=>p.classList.toggle('active',p.id===btn.dataset.tab));});
function eventType(kind){kind=kind.toLowerCase(); if(kind.includes('frame'))return 'frame'; if(kind.includes('git'))return 'git'; if(kind.includes('focusa'))return 'focusa'; if(kind.includes('operator')||kind.includes('audit')||kind.includes('control'))return 'action'; return 'status'}
function visibleEvents(){return eventFilter==='all'?stream:stream.filter(e=>e.type===eventFilter)}
function setFilter(f){eventFilter=f;document.querySelectorAll('.filterbar button').forEach(b=>b.classList.toggle('active',b.textContent.toLowerCase().startsWith(f)||f==='all'&&b.textContent==='All'));renderStream()}
function renderStream(){const el=$('stream'); if(el) el.innerHTML=visibleEvents().length?visibleEvents().map(e=>'<div class="event" onclick="alert('+JSON.stringify('Event: ').replace(/'/g,'&#39;')+'+this.innerText)"><div class="glyph">'+e.glyph+'</div><div><div class="event-title">'+e.kind+'</div><div class="event-meta">'+e.t.toLocaleTimeString()+' · '+e.msg+'</div></div></div>').join(''):'<div class="empty">No events in this filter.</div>'}
function addStream(kind,msg,glyph='↯'){stream.unshift({t:new Date(),kind,msg,glyph,type:eventType(kind)}); stream=stream.slice(0,24); renderStream()}
async function fpvControl(action,payload={}){return fetch('/m/'+token+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,...payload})}).then(r=>r.json())}
async function sendMsg(){const message=$('msg').value.trim(); if(!message)return; const r=await fpvControl('message',{message}); setText('controlResult',r.ok?'Saved to audit':'Rejected: '+(r.error||'not allowed')); addStream('Operator note',r.ok?'Saved to audit':'Rejected','✎'); $('msg').value=''; await statusTick()}
async function sendClick(){const selector=$('selector').value.trim(); if(!selector)return; const r=await fpvControl('click',{selector}); setText('controlResult',r.ok?'Click sent':'Click rejected: '+(r.error||'failed')); addStream('Control click',selector,'⌖'); await statusTick()}
async function sendFill(){const selector=$('selector').value.trim(), text=$('fillText').value; if(!selector)return; const r=await fpvControl('fill',{selector,text}); setText('controlResult',r.ok?'Fill sent':'Fill rejected: '+(r.error||'failed')); addStream('Control fill',selector,'✍'); await statusTick()}
async function sendKey(key){const r=await fpvControl('press',{key}); setText('controlResult',r.ok?key+' sent':'Key rejected: '+(r.error||'failed')); addStream('Control key',key,'⌨'); await statusTick()}
function setQuality(q){frameMs=q==='smooth'?250:q==='balanced'?500:1000; document.querySelectorAll('.quality button').forEach(b=>b.classList.remove('active')); $('q'+q[0].toUpperCase()+q.slice(1)).classList.add('active'); clearInterval(frameTimer); frameTimer=setInterval(frameTick,frameMs); addStream('Stream quality',q+' mode','⌁')}
function tone(id,good,bad=false){const el=$(id); if(el) el.className='value hero '+(bad?'bad':good?'ok':'warn')}
function render(st){const mode=st.mode==='control'?'Control enabled':'Read-only'; setText('mode',mode); setText('modeChip',mode); setText('modeCard',mode); $('address').innerHTML='<span class="lock">●</span><span>'+(st.url||'—')+'</span>'; setText('pageUrl',st.url); setText('pageTitle',st.title); setText('viewportSize',(st.width||'?')+' × '+(st.height||'?')); setText('expires',new Date(st.expires_at).toLocaleString()); setText('views',st.views); setText('chromeMeta',st.title||'FPV Live'); const d=st.diagnostics||{}; const errTotal=(d.console_errors||0)+(d.failed_requests||0)+(d.http_4xx||0)+(d.http_5xx||0); renderGlyph('healthGrid',Math.min(8,errTotal||1),(d.http_5xx||0)>0); setText('consoleErrors',d.console_errors||0); setText('failedRequests',d.failed_requests||0); setText('httpErrors',(d.http_4xx||0)+' / '+(d.http_5xx||0)); setText('requests',d.requests||0); tone('consoleErrors',(d.console_errors||0)===0); tone('failedRequests',(d.failed_requests||0)===0); tone('httpErrors',(d.http_4xx||0)+(d.http_5xx||0)===0,(d.http_5xx||0)>0); renderContext(st.context||{}); const audit=st.audit||[]; const last=audit[audit.length-1]; if(last && (!stream[0]||stream[0].msg!==last.action+': '+(last.message||last.selector||last.key||last.error||''))) addStream('Audit event',last.action+': '+(last.message||last.selector||last.key||last.error||''),'✓')}
function card(label,value,hint=''){return '<div class="card"><div class="label">'+label+'</div><div class="value">'+(value??'—')+'</div>'+ (hint?'<div class="hint">'+hint+'</div>':'') +'</div>'}
function renderContext(ctx){const p=ctx.project||{}; const repo=$('repoGrid'); if(repo) repo.innerHTML=card('⌘ Project',p.name,'Root: '+(p.root||'unknown'))+card('⑂ Branch',p.branch)+card('● Head',p.head,p.dirty?'Working tree modified':'Working tree clean')+card('◧ Public host','fpv.wpuiai.com','Path-gated to /m/*'); const tree=$('repoTree'); if(tree) tree.textContent=(ctx.tree||[]).map(i=>'├─ '+i.path+'  '+(i.active?'active':'')).join('\n')||'No tree context available'; const f=ctx.focusa||{}; const fg=$('focusaGrid'); if(fg) fg.innerHTML=card('✦ Current objective',f.objective,'Compact Focusa summary')+card('➜ Next step',f.next_step)+card('✓ Evidence',(f.evidence||[]).join(', '))+card('⛨ Drift guard',f.drift_guard); (ctx.history||[]).slice(0,3).forEach(h=>{if(!stream.some(e=>e.msg===h.ref+': '+h.title))addStream('Git history',h.ref+': '+h.title,'⌘')})}
async function statusTick(){if(!active)return;try{const r=await fetch('/m/'+token+'/status',{cache:'no-store'}); const st=await r.json(); if(!r.ok||st.error){active=false; setText('mode','share expired'); setText('fps','stopped'); addStream('Status',st.error||'share unavailable','⏱'); return} render(st)}catch(e){setText('mode','offline');addStream('Status','metadata retrying','⚠')}}
function frameTick(){if(!active)return; const img=new Image(); img.onload=()=>{frameErrors=0;frames++; $('shot').style.opacity=.985; $('shot').src=img.src; requestAnimationFrame(()=>$('shot').style.opacity=1); const fps=(frames/((Date.now()-started)/1000)).toFixed(1); setText('fps',fps+' FPS'); setText('fpsNum',fps); setText('streamQuality',Number(fps)>2?'Excellent':'Healthy'); renderSeismo(); renderGlyph('glyphGrid',Math.max(1,Math.min(8,Math.round(Number(fps)||1)))); sweepCards(); setText('ticker','LIVE · '+fps+' FPS · '+(frameErrors?'RECOVERING':'COMPOSITE OK')); if(frames%16===0)addStream('Frame refresh','Browser frame received','↻')}; img.onerror=()=>{frameErrors++; setText('fps','retrying'); setText('streamQuality','Retrying'); if(frameErrors>8){active=false;setText('fps','stopped');setText('streamQuality','Stopped');addStream('Frame stream','Stopped after repeated frame errors','⚠')}}; img.src='/m/'+token+'/screenshot.jpg?t='+Date.now()}
addStream('Viewer connected','FPV cockpit online','◉'); statusTick(); frameTick(); setInterval(statusTick,1500); frameTimer=setInterval(frameTick,frameMs);
</script></body></html>`))
