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
:root{color-scheme:dark;--bg0:#020617;--bg1:#07111f;--panel:rgba(15,23,42,.78);--panel2:rgba(30,41,59,.72);--card:rgba(2,6,23,.58);--muted:#94a3b8;--text:#f8fafc;--line:rgba(148,163,184,.18);--line2:rgba(148,163,184,.28);--blue:#7dd3fc;--violet:#a78bfa;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--shadow:0 30px 100px rgba(0,0,0,.50);--radius:24px;--ease-out:cubic-bezier(.16,1,.3,1);--ease-spring:cubic-bezier(.34,1.56,.64,1);--pulse-slow:2.4s;--sweep-fast:1.05s;--stream-in:.42s}*{box-sizing:border-box}html,body{margin:0;height:100%;overflow:hidden}body{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:radial-gradient(1100px 720px at 16% -18%,rgba(56,189,248,.22),transparent 58%),radial-gradient(900px 620px at 94% 8%,rgba(167,139,250,.18),transparent 55%),linear-gradient(135deg,var(--bg0),var(--bg1) 58%,#020617);color:var(--text)}body:before{content:"";position:fixed;inset:0;pointer-events:none;background-image:linear-gradient(rgba(255,255,255,.035) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.028) 1px,transparent 1px);background-size:56px 56px;mask-image:linear-gradient(to bottom,rgba(0,0,0,.65),transparent 72%)}header{height:72px;padding:14px 22px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid var(--line);background:rgba(2,6,23,.62);backdrop-filter:blur(24px);position:relative;z-index:5}.brand{display:flex;gap:12px;align-items:center}.logo{width:38px;height:38px;border-radius:14px;background:linear-gradient(135deg,var(--blue),var(--violet));box-shadow:0 14px 40px rgba(125,211,252,.28);display:grid;place-items:center;color:#020617;font-weight:950}.title{font-weight:900;letter-spacing:-.035em;font-size:18px}.sub{color:var(--muted);font-size:12px;margin-top:2px}.actions{display:flex;gap:10px;align-items:center}.chip{display:inline-flex;gap:8px;align-items:center;border:1px solid var(--line);border-radius:999px;padding:8px 12px;background:rgba(15,23,42,.70);box-shadow:inset 0 1px rgba(255,255,255,.06);color:#cbd5e1;font-size:12px;font-weight:700}.dot{width:8px;height:8px;border-radius:50%;background:var(--green);box-shadow:0 0 0 5px rgba(52,211,153,.10),0 0 18px var(--green);animation:fpv-pulse var(--pulse-slow) var(--ease-out) infinite}main{height:calc(100vh - 72px);display:grid;grid-template-columns:minmax(0,2fr) minmax(380px,1fr);gap:18px;padding:18px}.stage{min-width:0;display:flex;align-items:center;justify-content:center}.browser{width:100%;height:100%;border:1px solid var(--line2);border-radius:var(--radius);overflow:hidden;background:linear-gradient(180deg,rgba(15,23,42,.9),rgba(2,6,23,.94));box-shadow:var(--shadow);display:flex;flex-direction:column;position:relative}.browser:before{content:"";position:absolute;inset:0;border-radius:inherit;pointer-events:none;background:linear-gradient(135deg,rgba(255,255,255,.16),transparent 22%,transparent 72%,rgba(125,211,252,.10));mix-blend-mode:screen}.browser:after{content:"";position:absolute;inset:54px 0 0;pointer-events:none;background:linear-gradient(180deg,transparent 0,rgba(125,211,252,.10) 49%,transparent 52%);height:24%;animation:fpv-scan 4.8s linear infinite;opacity:.35}.chrome{height:54px;display:grid;grid-template-columns:78px 1fr auto;gap:12px;align-items:center;padding:0 14px;background:rgba(15,23,42,.86);border-bottom:1px solid var(--line);backdrop-filter:blur(20px)}.lights{display:flex;gap:8px}.light{width:12px;height:12px;border-radius:999px;box-shadow:inset 0 -1px rgba(0,0,0,.30)}.red{background:#ff5f57}.yellow{background:#febc2e}.green{background:#28c840}.addr{height:32px;display:flex;align-items:center;gap:8px;padding:0 13px;border:1px solid var(--line);border-radius:999px;background:rgba(2,6,23,.74);color:#dbeafe;font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;box-shadow:inset 0 1px rgba(255,255,255,.04)}.lock{color:var(--green);font-size:11px}.chrome-meta{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.04em;text-transform:uppercase}.viewport{flex:1;display:grid;place-items:center;background:#000;min-height:0;padding:0}.viewport img{display:block;max-width:100%;max-height:100%;width:100%;height:100%;object-fit:contain;background:#000;transition:opacity .12s ease,filter .12s ease}.rail{height:100%;overflow:auto;border:1px solid var(--line2);border-radius:var(--radius);background:var(--panel);backdrop-filter:blur(24px);box-shadow:var(--shadow)}.rail-head{position:sticky;top:0;z-index:3;padding:16px;border-bottom:1px solid var(--line);background:rgba(15,23,42,.88);backdrop-filter:blur(24px)}.rail-title{font-size:18px;font-weight:950;letter-spacing:-.04em}.rail-sub{margin-top:3px;color:var(--muted);font-size:12px}.section{padding:15px;border-bottom:1px solid var(--line)}.section h3{margin:0 0 11px;display:flex;align-items:center;justify-content:space-between;font-size:11px;text-transform:uppercase;letter-spacing:.13em;color:#e2e8f0}.grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}.card{background:var(--card);border:1px solid var(--line);border-radius:18px;padding:12px;box-shadow:inset 0 1px rgba(255,255,255,.04);position:relative;overflow:hidden;transition:border-color .2s var(--ease-out),transform .35s var(--ease-out),background .35s var(--ease-out)}.card:after{content:"";position:absolute;inset:-40% -60%;background:linear-gradient(115deg,transparent 42%,rgba(255,255,255,.18) 50%,transparent 58%);transform:translateX(-55%) skewX(-12deg);opacity:0}.card:hover{border-color:rgba(125,211,252,.45);transform:translateY(-1px)}.card:hover:after,.card.sweep:after{animation:fpv-shimmer var(--sweep-fast) var(--ease-out)}.label{color:var(--muted);font-size:10px;text-transform:uppercase;letter-spacing:.08em;font-weight:850}.value{font-weight:850;margin-top:6px;word-break:break-word;letter-spacing:-.015em}.big{font-size:22px}.ok{color:var(--green)}.warn{color:var(--amber)}.bad{color:var(--red)}textarea{width:100%;background:rgba(2,6,23,.66);color:var(--text);border:1px solid var(--line);border-radius:16px;padding:12px;outline:none;resize:vertical}textarea:focus{border-color:rgba(125,211,252,.7);box-shadow:0 0 0 4px rgba(125,211,252,.12)}button.action{margin-top:10px;width:100%;border:0;border-radius:16px;padding:12px 14px;color:#020617;font-weight:950;background:linear-gradient(135deg,var(--blue),var(--violet));box-shadow:0 15px 35px rgba(125,211,252,.18);cursor:pointer}.stream{display:grid;gap:9px}.stream-row{display:grid;grid-template-columns:78px 1fr;gap:10px;align-items:start;border:1px solid var(--line);border-radius:16px;padding:10px;background:rgba(2,6,23,.48);animation:fpv-stream-in var(--stream-in) var(--ease-spring);position:relative;overflow:hidden}.stream-row:before{content:"";position:absolute;left:0;top:0;bottom:0;width:2px;background:linear-gradient(var(--blue),var(--violet));opacity:.8}.time{color:var(--muted);font-size:11px;font-variant-numeric:tabular-nums}.small{font-size:12px;color:#cbd5e1}.kbd{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:11px;color:#dbeafe}.spark{height:6px;border-radius:999px;background:linear-gradient(90deg,var(--green),var(--blue),var(--violet),var(--green));background-size:220% 100%;box-shadow:0 0 22px rgba(125,211,252,.28);margin-top:12px;transform-origin:left;animation:fpv-flow 3.2s linear infinite}.hero-stat{display:flex;align-items:baseline;gap:8px}.hero-stat .num{font-size:30px;font-weight:950;letter-spacing:-.05em}.hero-stat .unit{color:var(--muted);font-size:12px;font-weight:850}@keyframes fpv-pulse{0%,100%{transform:scale(1);opacity:1}50%{transform:scale(.72);opacity:.58}}@keyframes fpv-shimmer{0%{opacity:0;transform:translateX(-55%) skewX(-12deg)}18%{opacity:1}100%{opacity:0;transform:translateX(55%) skewX(-12deg)}}@keyframes fpv-stream-in{0%{opacity:0;transform:translateY(10px) scale(.985)}100%{opacity:1;transform:none}}@keyframes fpv-scan{0%{transform:translateY(-30%)}100%{transform:translateY(360%)}}@keyframes fpv-flow{0%{background-position:0% 50%}100%{background-position:220% 50%}}@keyframes fpv-glow{0%,100%{box-shadow:0 0 0 rgba(251,191,36,0)}50%{box-shadow:0 0 28px rgba(251,191,36,.22)}}@media (prefers-reduced-motion:reduce){*,*:before,*:after{animation:none!important;transition:none!important}}@media (max-width:980px){html,body{height:auto;overflow:auto}header{height:auto;padding:12px 14px}.actions{display:none}main{height:auto;grid-template-columns:1fr;padding:12px}.browser{height:64vh;min-height:460px}.rail{height:auto;overflow:visible}.grid{grid-template-columns:1fr}.chrome{grid-template-columns:54px 1fr}.chrome-meta{display:none}}
</style></head>
<body><header><div class="brand"><div class="logo">F</div><div><div class="title">UIAI First‑Person View</div><div class="sub">{{.SessionID}} · <span id="mode">connecting</span></div></div></div><div class="actions"><div class="chip"><span class="dot"></span><span id="fps">starting</span></div><div class="chip" id="modeChip">Live</div></div></header>
<main><section class="stage"><div class="browser"><div class="chrome"><div class="lights"><span class="light red"></span><span class="light yellow"></span><span class="light green"></span></div><div id="address" class="addr"><span class="lock">●</span><span>loading URL…</span></div><div id="chromeMeta" class="chrome-meta">FPV Live</div></div><div class="viewport"><img id="shot" alt="live browser screenshot"></div></div></section>
<aside class="rail"><div class="rail-head"><div class="rail-title">Realtime Control Center</div><div class="rail-sub">Streaming browser state, diagnostics, and audited operator actions.</div><div class="spark"></div></div>
<section class="section"><h3><span>Live signal</span><span class="kbd">/m/{{.Token}}</span></h3><div class="grid"><div class="card"><div class="label">Frame rate</div><div class="hero-stat"><span id="fpsNum" class="num">0.0</span><span class="unit">FPS</span></div></div><div class="card"><div class="label">Mode</div><div id="modeCard" class="value big">loading</div></div><div class="card"><div class="label">Title</div><div id="pageTitle" class="value">loading</div></div><div class="card"><div class="label">Viewport</div><div id="viewportSize" class="value">loading</div></div><div class="card"><div class="label">Expires</div><div id="expires" class="value">loading</div></div><div class="card"><div class="label">Views</div><div id="views" class="value">loading</div></div></div></section>
<section class="section"><h3>Diagnostics</h3><div class="grid"><div class="card"><div class="label">Console errors</div><div id="consoleErrors" class="value big ok">0</div></div><div class="card"><div class="label">Failed requests</div><div id="failedRequests" class="value big ok">0</div></div><div class="card"><div class="label">HTTP 4xx / 5xx</div><div id="httpErrors" class="value big ok">0 / 0</div></div><div class="card"><div class="label">Requests</div><div id="requests" class="value big">0</div></div></div></section>
<section class="section"><h3>Operator action</h3><textarea id="msg" rows="3" placeholder="Send a note to the audit stream"></textarea><button class="action" onclick="sendMsg()">Send audited note</button><div id="controlResult" class="rail-sub"></div></section>
<section class="section"><h3>Activity stream</h3><div id="stream" class="stream"></div></section>
</aside></main>
<script>
const token='{{.Token}}'; let frames=0; let started=Date.now(); let stream=[]; let active=true; let frameErrors=0;
const $=id=>document.getElementById(id); function setText(id,v){const el=$(id); if(el) el.textContent=(v ?? '—')}
function addStream(kind,msg){stream.unshift({t:new Date(),kind,msg}); stream=stream.slice(0,20); $('stream').innerHTML=stream.map(e=>'<div class="stream-row"><div class="time">'+e.t.toLocaleTimeString()+'</div><div><b>'+e.kind+'</b><div class="small">'+e.msg+'</div></div></div>').join('')}
async function fpvControl(action,payload={}){return fetch('/m/'+token+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,...payload})}).then(r=>r.json())}
async function sendMsg(){const message=$('msg').value.trim(); if(!message)return; const r=await fpvControl('message',{message}); setText('controlResult',r.ok?'Saved to audit':'Rejected: '+(r.error||'not allowed')); addStream('control',r.ok?'Operator note saved':'Control rejected'); $('msg').value=''; await statusTick()}
function tone(id,good){$(id).className='value big '+(good?'ok':'warn')}
function render(st){const mode=st.mode==='control'?'Control enabled':'Read-only'; setText('mode',mode); setText('modeChip',mode); setText('modeCard',mode); $('address').innerHTML='<span class="lock">●</span><span>'+(st.url||'—')+'</span>'; setText('pageTitle',st.title); setText('viewportSize',(st.width||'?')+' × '+(st.height||'?')); setText('expires',new Date(st.expires_at).toLocaleString()); setText('views',st.views); setText('chromeMeta',st.title||'FPV Live'); const d=st.diagnostics||{}; setText('consoleErrors',d.console_errors||0); setText('failedRequests',d.failed_requests||0); setText('httpErrors',(d.http_4xx||0)+' / '+(d.http_5xx||0)); setText('requests',d.requests||0); tone('consoleErrors',(d.console_errors||0)===0); tone('failedRequests',(d.failed_requests||0)===0); tone('httpErrors',(d.http_4xx||0)+(d.http_5xx||0)===0); const audit=st.audit||[]; const last=audit[audit.length-1]; if(last && (!stream[0]||stream[0].msg!==last.action+': '+(last.message||last.selector||last.key||last.error||''))) addStream('audit',last.action+': '+(last.message||last.selector||last.key||last.error||''))}
async function statusTick(){if(!active)return;try{const r=await fetch('/m/'+token+'/status',{cache:'no-store'}); const st=await r.json(); if(!r.ok||st.error){active=false; setText('mode','share expired'); setText('fps','stopped'); addStream('status',st.error||'share unavailable'); return} render(st)}catch(e){setText('mode','offline');addStream('status','metadata retrying')}}
function frameTick(){if(!active)return; const img=new Image(); img.onload=()=>{frameErrors=0;frames++; $('shot').style.opacity=.985; $('shot').src=img.src; requestAnimationFrame(()=>$('shot').style.opacity=1); const fps=(frames/((Date.now()-started)/1000)).toFixed(1); setText('fps',fps+' FPS'); setText('fpsNum',fps); if(frames%16===0)addStream('frame','browser frame refreshed')}; img.onerror=()=>{frameErrors++; setText('fps','retrying'); if(frameErrors>8){active=false;setText('fps','stopped');addStream('frame','stream stopped after repeated frame errors')}}; img.src='/m/'+token+'/screenshot.jpg?t='+Date.now()}
addStream('viewer','FPV control center connected'); statusTick(); frameTick(); setInterval(statusTick,1500); setInterval(frameTick,250);
</script></body></html>`))
