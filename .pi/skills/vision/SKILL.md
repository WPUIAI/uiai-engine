---
name: vision
description: Screenshot and browser automation via local Rod pool on localhost:7456.
---
# Vision Skill — Screenshot & Browser Automation

Take screenshots, interact with pages, and automate browser tasks via the local Go engine's Rod browser pool. No API key needed — runs on `localhost:7456`.

**Three modes:**
1. **One-shot** (`/api/screenshot`) — navigate → snap → forget. Best for single checks.
2. **Session** (`/api/session`) — open → interact repeatedly → close. **Best for iterative QA** (30ms re-snaps vs 1.7s re-navigations).
3. **Snapshot + @ref** (`/api/session/{id}/snapshot`) — get a11y tree with element refs for precise clicking/typing. **Best for LLM agents.**
4. **Diagnostics** (`/api/session/{id}/diagnostics`) — get console errors, JS exceptions, network failures, failed requests, and summary counts without taking a screenshot.
5. **Bounded async eval** (`/api/session/{id}/eval_async`) — await small async checks with `timeout_ms`; use direct actions for long UI workflows.
6. **Focusa packet composer** (`/api/agent/research-packet`) — compose bounded `uiai.focusa_research_diagnostics_packet.v1` packets from existing UIAI search/read/snapshot/diagnostics/error/screenshot/share responses.

**Use when:** You need to see what a webpage looks like, verify a deployment, audit a site, check mobile layout, fill forms, test auth flows, or automate browser interactions.

**Full API docs:** `/home/wpuiai/uiai-engine/docs/SESSION_API.md`

**Diagnostics:** `/home/wpuiai/uiai-engine/docs/BROWSER_DIAGNOSTICS_SPEC.md` documents implemented lightweight console/exception/network capture via `GET /api/session/{id}/diagnostics` and `browser_diagnostics`. During troubleshooting, call diagnostics after `browser_open` and after broken screenshots, failed clicks/waits, JS errors, CORS/API/network suspicion, blank pages, broken pages, or visual failures.

**Reliability runbook:** `/home/wpuiai/uiai-engine/docs/BROWSER_RELIABILITY_RUNBOOK.md` covers diagnostics stress, mixed soak, release gates, artifacts, and long-async eval mitigation.

**Focusa packet handoff:** after search/read/snapshot/diagnostics proof, call `POST /api/agent/research-packet` (or Pi `uiai_focusa_packet_compose`) with the existing UIAI responses, then pass `recommended_focusa.args_preview` to `focusa_evidence_capture` or `focusa_browser_diagnostics_intake`.

---

## Quick Screenshot

```bash
# Screenshot → save to file (decode base64 → binary)
curl -s -X POST http://localhost:7456/api/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}' \
  | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('/tmp/shot.png','wb').write(base64.b64decode(d['screenshot'])); print(f'Saved: {d[\"size\"]}B {d[\"width\"]}x{d[\"height\"]} {d[\"duration\"]}ms')"
```

Then view the image:
```bash
read /tmp/shot.png
```

## One-liner helper

```bash
# Define once per session:
viz() {
  local url="${1:?url required}" out="${2:-/tmp/viz.jpg}" w="${3:-1280}" h="${4:-800}"
  curl -s -X POST http://localhost:7456/api/screenshot \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"$url\",\"width\":$w,\"height\":$h,\"format\":\"jpeg\",\"quality\":60}" \
    | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('$out','wb').write(base64.b64decode(d['screenshot'])); print(f'✅ {d[\"size\"]}B {d[\"width\"]}x{d[\"height\"]} {d[\"duration\"]}ms → $out')"
}

# Usage:
viz https://wpuiai.com /tmp/home.jpg
viz https://wpuiai.com/pricing /tmp/pricing.jpg
viz https://example.com /tmp/mobile.jpg 375 812   # mobile viewport
```

---

## Session Mode — Continuous Vision (Recommended for QA)

### Open → Snapshot → Act → Screenshot → Diagnostics when debugging

```bash
# 1. Open session
SID=$(curl -s -X POST http://localhost:7456/api/session \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","width":390,"height":844}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['session']['id'])")

# 2. Snapshot — see what you can interact with (~50ms, ~955 tokens)
curl -s http://localhost:7456/api/session/$SID/snapshot | python3 -c "
import json,sys; d=json.load(sys.stdin)
print(d['tree'][:500])
print(f'--- {d[\"stats\"][\"ref_count\"]} refs, ~{d[\"stats\"][\"tokens\"]} tokens ---')"

# 3. Click by @ref
curl -s -X POST http://localhost:7456/api/session/$SID/click \
  -H "Content-Type: application/json" -d '{"selector":"@e3"}' \
  | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('/tmp/clicked.jpg','wb').write(base64.b64decode(d['screenshot'])); print(f'✅ {d[\"duration_ms\"]}ms → {d[\"url\"]}')"

# 4. Debug evidence when page behavior is wrong — console/errors/network, no screenshot
curl -s http://localhost:7456/api/session/$SID/diagnostics | jq '{summary, console, exceptions, failed_requests}'

# 5. Close
curl -s -X DELETE http://localhost:7456/api/session/$SID
```

### All 22 Session Actions

| Action | Method | Path | Returns | Typical Latency |
|--------|--------|------|---------|-----------------|
| **Snapshot** | `GET/POST` | `/snapshot` | A11y tree + @ref map | ~50ms |
| **Screenshot** | `POST` | `/screenshot` | Base64 image | **30ms** |
| **Scroll** | `POST` | `/scroll` | Screenshot | 150ms |
| **Click** | `POST` | `/click` | Screenshot | 300ms |
| **Hover** | `POST` | `/hover` | Screenshot | 200ms |
| **Type** | `POST` | `/type` | Screenshot | 150ms |
| **Fill** | `POST` | `/fill` | Screenshot | 150ms |
| **Select** | `POST` | `/select` | Screenshot | 150ms |
| **Press** | `POST` | `/press` | Screenshot | 200-800ms |
| **Eval** | `POST` | `/eval` | Short sync JS result + screenshot | 150ms |
| **Eval Async** | `POST` | `/eval_async` | Bounded awaited JS result + screenshot | timeout-bounded |
| **Navigate** | `POST` | `/navigate` | Screenshot | 1-2s |
| **Resize** | `POST` | `/resize` | Screenshot | 300ms |
| **CSS** | `POST` | `/css` | Screenshot | 150ms |
| **Wait** | `POST` | `/wait` | Screenshot | varies |
| **Back** | `POST` | `/back` | Screenshot | 1-2s |
| **Forward** | `POST` | `/forward` | Screenshot | 1-2s |
| **Text** | `POST` | `/text` | Element text (no screenshot) | instant |
| **Cookies** | `POST` | `/cookies` | Cookie list (no screenshot) | instant |
| **Auth Save** | `POST` | `/auth/save` | Cookies + localStorage JSON | instant |
| **Auth Load** | `POST` | `/auth/load` | Status (no screenshot) | instant |
| **DOM** | `GET` | `/dom` | DOM structure (legacy) | instant |
| **Diagnostics** | `GET` | `/diagnostics` | Console/errors/exceptions/network summary | instant |
| **Diagnostics Clear** | `POST` | `/diagnostics/clear` | Reset diagnostic buffers | instant |
| **Close** | `DELETE` | (session root) | — | instant |

All paths prefixed with `/api/session/{id}/`.

---

## @ref Workflow (Best for LLM Agents)

The **snapshot → act by ref** pattern is the most efficient way for LLMs to interact with pages:

```bash
# 1. Get snapshot (interactive-only = buttons/links/inputs)
curl -s -X POST http://localhost:7456/api/session/$SID/snapshot \
  -H "Content-Type: application/json" -d '{"interactive":true}'
# Returns:
#   - link "Sign In" [ref=e3]
#   - textbox "Email" [ref=e5]
#   - textbox "Password" [ref=e6]
#   - button "Log In" [ref=e7]

# 2. Type into email field by ref
curl -s -X POST http://localhost:7456/api/session/$SID/fill \
  -H "Content-Type: application/json" \
  -d '{"selector":"@e5","text":"user@example.com"}'

# 3. Type password
curl -s -X POST http://localhost:7456/api/session/$SID/fill \
  -H "Content-Type: application/json" \
  -d '{"selector":"@e6","text":"secret123"}'

# 4. Press Enter (or click submit button)
curl -s -X POST http://localhost:7456/api/session/$SID/press \
  -H "Content-Type: application/json" -d '{"key":"Enter"}'

# 5. Save auth state for reuse
curl -s -X POST http://localhost:7456/api/session/$SID/auth/save \
  -o /tmp/auth-state.json
```

---

## Common Patterns

### Fill a form
```bash
# Fill clears + types (more reliable than type for replacing values)
curl -s -X POST http://localhost:7456/api/session/$SID/fill \
  -H "Content-Type: application/json" \
  -d '{"selector":"@e5","text":"new value"}'
```

### Select a dropdown option
```bash
curl -s -X POST http://localhost:7456/api/session/$SID/select \
  -H "Content-Type: application/json" \
  -d '{"selector":"@e8","values":["California"]}'
```

### Press keyboard keys
```bash
# Submit form with Enter
curl -s -X POST http://localhost:7456/api/session/$SID/press \
  -H "Content-Type: application/json" -d '{"key":"Enter"}'

# Navigate with Tab
curl -s -X POST http://localhost:7456/api/session/$SID/press \
  -H "Content-Type: application/json" -d '{"key":"Tab"}'

# Dismiss modal with Escape
curl -s -X POST http://localhost:7456/api/session/$SID/press \
  -H "Content-Type: application/json" -d '{"key":"Escape"}'

# Supported: Enter, Tab, Escape, Backspace, Delete, Space,
#            ArrowUp, ArrowDown, ArrowLeft, ArrowRight,
#            Home, End, PageUp, PageDown
```

### Browser history
```bash
# Go back
curl -s -X POST http://localhost:7456/api/session/$SID/back

# Go forward
curl -s -X POST http://localhost:7456/api/session/$SID/forward
```

### Bounded async eval
```bash
# Small awaited checks only. For long UI flows, use snapshot + direct actions.
curl -s -X POST http://localhost:7456/api/session/$SID/eval_async \
  -H "Content-Type: application/json" \
  -d '{"js":"await new Promise(r => setTimeout(r, 250)); return document.title","timeout_ms":2000}' \
  | jq '{result,bounded_async,duration_ms}'
```

### Extract text from element
```bash
curl -s -X POST http://localhost:7456/api/session/$SID/text \
  -H "Content-Type: application/json" -d '{"selector":"@e12"}'
# Returns: {"text": "Welcome back, Verious!", "selector": "@e12"}
```

### Manage cookies
```bash
# Get all cookies
curl -s -X POST http://localhost:7456/api/session/$SID/cookies \
  -H "Content-Type: application/json" -d '{"action":"get"}'

# Get specific cookie
curl -s -X POST http://localhost:7456/api/session/$SID/cookies \
  -H "Content-Type: application/json" -d '{"action":"get","name":"wp_logged_in"}'

# Set a cookie
curl -s -X POST http://localhost:7456/api/session/$SID/cookies \
  -H "Content-Type: application/json" \
  -d '{"action":"set","name":"test","value":"hello","domain":"example.com"}'

# Clear all cookies
curl -s -X POST http://localhost:7456/api/session/$SID/cookies \
  -H "Content-Type: application/json" -d '{"action":"clear"}'
```

### Save and restore auth state
```bash
# Save: captures cookies + localStorage + sessionStorage
curl -s -X POST http://localhost:7456/api/session/$SID/auth/save \
  -o /tmp/auth-mysite.json

# Load into a new session later:
SID2=$(curl -s -X POST http://localhost:7456/api/session \
  -H "Content-Type: application/json" \
  -d '{"url":"https://mysite.com"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['session']['id'])")

curl -s -X POST http://localhost:7456/api/session/$SID2/auth/load \
  -H "Content-Type: application/json" -d @/tmp/auth-mysite.json

# Navigate to trigger auth (cookies now set)
curl -s -X POST http://localhost:7456/api/session/$SID2/navigate \
  -H "Content-Type: application/json" -d '{"url":"https://mysite.com/dashboard"}'
```

### Desktop screenshot
```bash
curl -s -X POST http://localhost:7456/api/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","width":1280,"height":800,"format":"jpeg","quality":60}' \
  | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('/tmp/desktop.jpg','wb').write(base64.b64decode(d['screenshot']))"
```

### Mobile screenshot
```bash
curl -s -X POST http://localhost:7456/api/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","width":375,"height":812,"format":"jpeg","quality":60}' \
  | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('/tmp/mobile.jpg','wb').write(base64.b64decode(d['screenshot']))"
```

### Full-page screenshot
```bash
curl -s -X POST http://localhost:7456/api/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","fullPage":true,"format":"png"}' \
  | python3 -c "import json,sys,base64; d=json.load(sys.stdin); open('/tmp/fullpage.png','wb').write(base64.b64decode(d['screenshot']))"
```

---

## API Reference

### POST /api/screenshot (One-shot)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | **required** | URL to capture |
| `width` | int | 1280 | Viewport width |
| `height` | int | 800 | Viewport height |
| `fullPage` | bool | false | Capture entire scrollable page |
| `format` | string | `png` | `png` or `jpeg` |
| `quality` | int | 60 | JPEG quality (1-100) |
| `waitFor` | string | — | CSS selector to wait for before capture |
| `delay` | int | — | Extra ms to wait after page load |
| `cookies` | string | — | `"name=val; name2=val2"` — set before navigation |

**Response:** `{screenshot, width, height, format, size, duration, dom_report}`

### POST /api/session (Open)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | **required** | URL to open |
| `width` | int | 1280 | Viewport width |
| `height` | int | 800 | Viewport height |

**Response (201):** `{session: {id, url, title, width, height}, screenshot, size, duration_ms}`

### POST /api/session/{id}/snapshot

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `interactive` | bool | false | Only buttons/links/inputs (smaller tree) |
| `compact` | bool | false | Remove empty structural nodes |
| `max_depth` | int | unlimited | Limit tree depth |
| `selector` | string | `body` | CSS selector to scope snapshot |

**Response:** `{tree, refs, stats: {lines, chars, tokens, ref_count, interactive}}`

### GET /api/screenshot/health

```bash
curl -s http://localhost:7456/api/screenshot/health
# {"status":"healthy","pool":{...}}
```

---

## Tool Discovery (for MCP / LLM integration)

```bash
# List tools (names + descriptions only, bounded output)
curl -s http://localhost:7456/api/tools/search

# Search by keyword
curl -s 'http://localhost:7456/api/tools/search?q=click'

# Full OpenAI function definitions
curl -s http://localhost:7456/api/tools/openai

# Full MCP tool definitions
curl -s http://localhost:7456/api/tools/mcp
```

---

## Architecture

```
Agent (any tool)
  │  One-shot: POST localhost:7456/api/screenshot
  │  Session:  POST localhost:7456/api/session (session actions incl diagnostics/eval_async)
  │  Tools:    GET  localhost:7456/api/tools/search
  ▼
uiai-engine v23 (Go, port 7456)
  │  internal/routes/screenshot.go    (one-shot)
  │  internal/routes/session.go       (persistent — 22 actions)
  │  internal/routes/tools.go         (LLM tool discovery)
  │  internal/vision/snapshot.go      (a11y tree + @ref)
  │  internal/vision/session_actions.go (fill, select, press, etc)
  ▼
vision.Pool + SessionManager (Rod browser pool)
  │  4 pooled Chromium pages (headless, pre-warmed)
  │  Sessions hold pages alive between calls
  ▼
Screenshot JPEG/PNG → base64 in JSON response
  + DOM report / a11y snapshot / text extraction
```

- **Rod** = Go-native Chrome DevTools Protocol client (no Node.js)
- **Pool** = 4 pre-warmed pages, auto-recovers stale pages
- **Sessions** = up to 4 persistent pages, 10min idle auto-expire
- **No external API key** — local Chromium binary
- **Accessible from any process on the VPS** via `localhost:7456`

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `vision pool not initialized` | Engine couldn't find/launch Chromium. `journalctl -u uiai-engine` |
| Timeout on complex pages | Add `"delay": 3000` or `"waitFor": ".content"`; prefer session wait/actions for QA |
| Blank screenshot | Page may require JS. Add `"delay": 2000` |
| Large response | Use `"format":"jpeg","quality":60}` to reduce size |
| Need auth'd page | Use session + `/auth/load` with saved state, or `"cookies":"wp_cookie=value"` |
| `@ref` not found | Re-snapshot first — refs are from the last snapshot call |
| `unknown key` | Check supported keys: Enter, Tab, Escape, Arrow*, Backspace, Delete, Space, Home, End, PageUp, PageDown |
| Long async eval flakes | Split into direct browser actions; use `/eval_async` only for bounded awaits and read diagnostics before patching |
