package routes

import (
	"crypto/rand"
	"encoding/base64"
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
:root{color-scheme:dark;--bg:#0d1117;--panel:#151b23;--muted:#8b949e;--text:#f0f6fc;--line:#30363d;--accent:#58a6ff;--ok:#3fb950;--warn:#d29922}*{box-sizing:border-box}body{font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;margin:0;background:var(--bg);color:var(--text)}header{padding:14px 16px;background:rgba(21,27,35,.96);position:sticky;top:0;z-index:2;border-bottom:1px solid var(--line)}.top{display:flex;justify-content:space-between;gap:12px;align-items:start}.title{font-weight:800}.pill{display:inline-flex;gap:6px;align-items:center;border:1px solid var(--line);border-radius:999px;padding:4px 9px;color:var(--muted);font-size:12px}.dot{width:8px;height:8px;border-radius:50%;background:var(--ok);box-shadow:0 0 10px var(--ok)}main{padding:12px;display:grid;gap:12px}.screen{border:1px solid var(--line);border-radius:16px;overflow:hidden;background:#000;box-shadow:0 18px 60px rgba(0,0,0,.35)}img{display:block;width:100%;height:auto;background:#000}.tabs{display:grid;grid-template-columns:repeat(4,1fr);gap:6px}.tabs button{border:1px solid var(--line);background:var(--panel);color:var(--text);padding:10px 6px;border-radius:10px;font-weight:700}.tabs button.active{outline:2px solid var(--accent);color:#fff}.panel{display:none;border:1px solid var(--line);border-radius:16px;background:var(--panel);padding:12px}.panel.active{display:block}.grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}.card{background:#0d1117;border:1px solid var(--line);border-radius:12px;padding:10px}.label{color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.06em}.value{font-weight:750;margin-top:4px;word-break:break-word}.muted{color:var(--muted);font-size:12px}.diag{display:grid;grid-template-columns:repeat(3,1fr);gap:8px}.metric{text-align:center}.metric .value{font-size:22px}.ok{color:var(--ok)}.warn{color:var(--warn)}input,textarea{width:100%;background:#0d1117;color:var(--text);border:1px solid var(--line);border-radius:10px;padding:10px}button.action{margin-top:8px;width:100%;background:var(--accent);border:0;border-radius:10px;padding:10px;color:#07111f;font-weight:800}.audit{display:grid;gap:8px}.audit-row{border:1px solid var(--line);border-radius:10px;padding:8px;background:#0d1117}.small{font-size:12px}@media (max-width:520px){.grid,.diag{grid-template-columns:1fr}.tabs{grid-template-columns:repeat(2,1fr)}}
</style></head>
<body><header><div class="top"><div><div class="title">● UIAI FPV</div><div class="muted">{{.SessionID}} · <span id="mode">loading</span></div></div><div class="pill"><span class="dot"></span><span id="fps">starting</span></div></div></header>
<main><section class="screen"><img id="shot" alt="live browser screenshot"></section><nav class="tabs"><button data-tab="view" class="active">View</button><button data-tab="session">Session</button><button data-tab="diag">Diagnostics</button><button data-tab="control">Controls</button></nav>
<section id="view" class="panel active"><div class="grid"><div class="card"><div class="label">Live cadence</div><div id="cadence" class="value">~4 FPS fast screenshot poll</div></div><div class="card"><div class="label">Last frame</div><div id="lastFrame" class="value">waiting</div></div><div class="card"><div class="label">Page title</div><div id="pageTitle" class="value">loading</div></div><div class="card"><div class="label">URL</div><div id="pageUrl" class="value">loading</div></div></div></section>
<section id="session" class="panel"><div class="grid"><div class="card"><div class="label">Session</div><div id="sessionId" class="value">{{.SessionID}}</div></div><div class="card"><div class="label">Share token</div><div class="value">{{.Token}}</div></div><div class="card"><div class="label">Expires</div><div id="expires" class="value">loading</div></div><div class="card"><div class="label">Viewer opens</div><div id="views" class="value">loading</div></div></div></section>
<section id="diag" class="panel"><div class="diag"><div class="card metric"><div class="label">Console errors</div><div id="consoleErrors" class="value ok">0</div></div><div class="card metric"><div class="label">Failed requests</div><div id="failedRequests" class="value ok">0</div></div><div class="card metric"><div class="label">HTTP 4xx/5xx</div><div id="httpErrors" class="value ok">0</div></div></div></section>
<section id="control" class="panel"><div class="card"><div class="label">Operator note</div><textarea id="msg" rows="3" placeholder="Send a note to the audit stream"></textarea><button class="action" onclick="sendMsg()">Send audited note</button><div id="controlResult" class="muted"></div></div><h3>Recent audit</h3><div id="audit" class="audit muted">No audit events yet.</div></section></main>
<script>
const token='{{.Token}}'; let statusCache={}; let frames=0; let started=Date.now();
function $(id){return document.getElementById(id)}
function setText(id,v){const el=$(id); if(el) el.textContent=(v ?? '—')}
function tab(name){document.querySelectorAll('.panel').forEach(p=>p.classList.toggle('active',p.id===name));document.querySelectorAll('.tabs button').forEach(b=>b.classList.toggle('active',b.dataset.tab===name))}
document.querySelectorAll('.tabs button').forEach(b=>b.onclick=()=>tab(b.dataset.tab));
async function fpvControl(action, payload={}){ return fetch('/m/'+token+'/control', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({action, ...payload})}).then(r=>r.json()); }
async function sendMsg(){const message=$('msg').value.trim(); if(!message)return; const r=await fpvControl('message',{message}); setText('controlResult', r.ok?'Saved to audit':'Rejected: '+(r.error||'not allowed')); $('msg').value=''; await statusTick();}
function render(st){statusCache=st; setText('mode', st.mode==='control'?'control enabled':'read-only'); setText('pageTitle', st.title); setText('pageUrl', st.url); setText('expires', new Date(st.expires_at).toLocaleString()); setText('views', st.views); const d=st.diagnostics||{}; setText('consoleErrors', d.console_errors||0); setText('failedRequests', d.failed_requests||0); setText('httpErrors', (d.http_4xx||0)+' / '+(d.http_5xx||0)); ['consoleErrors','failedRequests','httpErrors'].forEach(id=>$(id).className='value '+(($(id).textContent==='0'||$(id).textContent==='0 / 0')?'ok':'warn')); const audit=st.audit||[]; $('audit').innerHTML=audit.length?audit.slice(-8).reverse().map(a=>'<div class="audit-row"><b>'+a.action+'</b> '+(a.ok?'✅':'⚠️')+'<div class="small">'+new Date(a.ts).toLocaleTimeString()+' '+(a.message||a.selector||a.key||a.error||'')+'</div></div>').join(''):'No audit events yet.';}
async function statusTick(){try{const st=await fetch('/m/'+token+'/status',{cache:'no-store'}).then(r=>r.json()); render(st);}catch(e){setText('mode','offline');}}
function frameTick(){const img=new Image(); img.onload=()=>{frames++; $('shot').src=img.src; setText('lastFrame', new Date().toLocaleTimeString()); const elapsed=(Date.now()-started)/1000; if(elapsed>1)setText('fps',(frames/elapsed).toFixed(1)+' FPS');}; img.onerror=()=>setText('fps','frame retrying'); img.src='/m/'+token+'/screenshot.jpg?t='+Date.now();}
statusTick(); frameTick(); setInterval(statusTick,1500); setInterval(frameTick,250);
</script></body></html>`))
