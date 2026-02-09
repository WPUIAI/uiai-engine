# Browser Session API — LLM Vision Tool

**Persistent browser sessions for AI agents.** Open a page once, interact with it continuously — like a human with a browser tab.

## Why This Exists

Traditional screenshot APIs are **transactional**: navigate → snap → forget. Every call pays the full navigation cost (~1.7s). An LLM doing visual QA must re-navigate the same page 5–10 times per iteration.

Session API is **persistent**: open once → snap/scroll/click/type/eval instantly. Re-screenshots take **30ms** instead of 1.7s. That's **57x faster**.

## Quick Start

```bash
# 1. Open a session
SID=$(curl -s -X POST http://localhost:7456/api/session \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","width":1280,"height":800}' \
  | jq -r '.session.id')

# 2. Snapshot — get accessibility tree with @ref selectors
curl -s http://localhost:7456/api/session/$SID/snapshot | jq '{tree, stats}'
# Output:
#   - link "Learn more" [ref=e1]
#   - textbox "Search" [ref=e2]
#   - button "Submit" [ref=e3]

# 3. Click by @ref (from snapshot)
curl -s -X POST http://localhost:7456/api/session/$SID/click \
  -H "Content-Type: application/json" -d '{"selector":"@e1"}'

# 4. Screenshot current state (instant — no navigation)
curl -s -X POST http://localhost:7456/api/session/$SID/screenshot \
  -H "Content-Type: application/json" -d '{}' \
  | jq '{duration_ms, size, url, title}'

# 5. Scroll down
curl -s -X POST http://localhost:7456/api/session/$SID/scroll \
  -H "Content-Type: application/json" -d '{"deltaY":600}'

# 6. Click a button (CSS selector also still works)
curl -s -X POST http://localhost:7456/api/session/$SID/click \
  -H "Content-Type: application/json" -d '{"selector":"button.submit"}'

# 6. Inject CSS to test a design change
curl -s -X POST http://localhost:7456/api/session/$SID/css \
  -H "Content-Type: application/json" \
  -d '{"css":".header { background: red; }"}'

# 7. Run JavaScript
curl -s -X POST http://localhost:7456/api/session/$SID/eval \
  -H "Content-Type: application/json" \
  -d '{"js":"return document.querySelectorAll(\"img\").length + \" images\""}'

# 8. Close when done
curl -s -X DELETE http://localhost:7456/api/session/$SID
```

## API Reference

### Session Management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/session` | List all active sessions |
| `POST` | `/api/session` | Open new session (navigate + screenshot) |
| `GET` | `/api/session/{id}` | Get session info |
| `DELETE` | `/api/session/{id}` | Close session, release page |

### Session Actions (all return screenshot except snapshot/dom)

| Method | Path | Description | Typical Latency |
|--------|------|-------------|-----------------|
| `GET/POST` | `/api/session/{id}/snapshot` | **A11y tree with @ref selectors** | **~50ms** |
| `POST` | `/api/session/{id}/screenshot` | Re-snap current state | **30ms** |
| `POST` | `/api/session/{id}/scroll` | Scroll + screenshot | **150ms** |
| `POST` | `/api/session/{id}/click` | Click element (CSS or @ref) + screenshot | **300ms** |
| `POST` | `/api/session/{id}/hover` | Hover element (CSS or @ref) + screenshot | **200ms** |
| `POST` | `/api/session/{id}/type` | Type into input (CSS or @ref) + screenshot | **150ms** |
| `POST` | `/api/session/{id}/eval` | Run JS + screenshot | **150ms** |
| `POST` | `/api/session/{id}/navigate` | Go to new URL + screenshot | **1-2s** |
| `POST` | `/api/session/{id}/resize` | Change viewport + screenshot | **300ms** |
| `POST` | `/api/session/{id}/css` | Inject CSS + screenshot | **150ms** |
| `POST` | `/api/session/{id}/wait` | Wait for selector + screenshot | **varies** |
| `POST` | `/api/session/{id}/fill` | Clear + type (reliable value replace) | **150ms** |
| `POST` | `/api/session/{id}/select` | Choose dropdown option by value/text | **150ms** |
| `POST` | `/api/session/{id}/press` | Keyboard key (Enter, Tab, Escape…) | **200-800ms** |
| `POST` | `/api/session/{id}/back` | Browser history back + screenshot | **1-2s** |
| `POST` | `/api/session/{id}/forward` | Browser history forward + screenshot | **1-2s** |
| `POST` | `/api/session/{id}/text` | Get element text content (no screenshot) | **instant** |
| `POST` | `/api/session/{id}/cookies` | Get/set/clear cookies (no screenshot) | **instant** |
| `POST` | `/api/session/{id}/auth/save` | Save cookies + localStorage to JSON | **instant** |
| `POST` | `/api/session/{id}/auth/load` | Restore auth state from saved JSON | **instant** |
| `GET` | `/api/session/{id}/dom` | DOM structure (legacy, prefer snapshot) | **instant** |

---

### `POST /api/session` — Open

```json
{
  "url": "https://example.com",    // required
  "width": 1280,                    // default: 1280
  "height": 800                     // default: 800
}
```

**Response (201):**
```json
{
  "session": {
    "id": "abc12345",
    "url": "https://example.com",
    "title": "Example Domain",
    "width": 1280,
    "height": 800
  },
  "screenshot": "<base64>",
  "size": 45230,
  "duration_ms": 1200
}
```

### `GET/POST /api/session/{id}/snapshot` — Accessibility Tree with @refs

**The recommended way for LLMs to discover page elements.** Returns a text tree with `@ref` IDs that can be used in click/type/hover actions.

```json
// POST body (all optional)
{
  "interactive": true,   // only buttons, links, inputs (default: false)
  "compact": true,       // remove empty structural nodes (default: false)
  "max_depth": 5,        // limit tree depth (default: unlimited)
  "selector": "#main"    // scope to CSS selector (default: "body")
}
```

`GET` variant always returns interactive+compact snapshot.

**Response:**
```json
{
  "tree": "  - link \"Join Free\" [ref=e4]\n  - textbox \"Search...\" [ref=e6]\n  - button \"Search\" [ref=e7]",
  "refs": {
    "e4": {"selector": "a.join-btn", "role": "link", "name": "Join Free", "tag": "a"},
    "e6": {"selector": "input[placeholder=\"Search...\"]", "role": "textbox", "name": "Search...", "tag": "input"},
    "e7": {"selector": "button.search-btn", "role": "button", "name": "Search", "tag": "button"}
  },
  "stats": {
    "lines": 78,
    "chars": 3820,
    "tokens": 955,
    "ref_count": 78,
    "interactive": 78
  }
}
```

**Using refs in actions:**
```bash
# Click by ref (resolves to stored CSS selector)
curl -X POST /api/session/$SID/click -d '{"selector":"@e4"}'

# Type by ref
curl -X POST /api/session/$SID/type -d '{"selector":"@e6","text":"startup ideas"}'

# CSS selectors still work
curl -X POST /api/session/$SID/click -d '{"selector":"button.submit"}'
```

**Optimal LLM workflow:**
1. `POST /snapshot {"interactive":true}` → parse tree + refs (~955 tokens)
2. Identify target from tree (e.g., `@e7` is Search button)
3. `POST /click {"selector":"@e7"}` → get screenshot result
4. Re-snapshot if page changed significantly

---

### `POST /api/session/{id}/screenshot` — Snap

```json
{
  "format": "jpeg",     // "jpeg" (default) or "png"
  "quality": 80,        // 1-100, default 80
  "fullPage": false     // capture entire scrollable page
}
```

**Response:**
```json
{
  "screenshot": "<base64>",
  "width": 1280,
  "height": 800,
  "format": "jpeg",
  "size": 38445,
  "url": "https://example.com",
  "title": "Example Domain",
  "duration_ms": 30
}
```

### `POST /api/session/{id}/scroll` — Scroll

```json
// Relative scroll:
{ "deltaY": 600 }                  // scroll down 600px
{ "deltaX": 200, "deltaY": 0 }    // scroll right

// Absolute scroll:
{ "x": 0, "y": 2000 }             // scroll to y=2000
```

### `POST /api/session/{id}/click` — Click

```json
{ "selector": ".buy-button" }
{ "selector": "#submit" }
{ "selector": "nav a:nth-child(3)" }
```

### `POST /api/session/{id}/hover` — Hover

```json
{ "selector": ".dropdown-trigger" }
```

### `POST /api/session/{id}/type` — Type

```json
{
  "selector": "input[name=email]",
  "text": "user@example.com"
}
```

### `POST /api/session/{id}/eval` — JavaScript

```json
{ "js": "return document.querySelectorAll('h1').length + ' headings'" }
```

**Response:**
```json
{
  "result": "3 headings",
  "screenshot": "<base64>",
  "size": 38445,
  "duration_ms": 95
}
```

### `POST /api/session/{id}/resize` — Viewport

```json
{ "width": 375, "height": 812 }    // switch to mobile
{ "width": 1440, "height": 900 }   // switch to desktop
```

### `POST /api/session/{id}/css` — Inject CSS

```json
{ "css": ".header { background: red; } .hero { display: none; }" }
```

Replaces any previously injected CSS (identified by `#llm-injected-css`).

### `POST /api/session/{id}/wait` — Wait for Selector

```json
{
  "selector": ".lazy-loaded-content",
  "timeout_ms": 5000                // default: 5000
}
```

### `GET /api/session/{id}/dom` — DOM Info

No screenshot. Returns structured page data for LLM reasoning:

```json
{
  "url": "https://example.com",
  "title": "Example Domain",
  "scroll": { "x": 0, "y": 800, "maxY": 6788 },
  "viewport": { "width": 1280, "height": 800, "scrollHeight": 7588 },
  "headings": [
    { "tag": "H1", "text": "Welcome" },
    { "tag": "H2", "text": "Features" }
  ],
  "links": 72,
  "buttons": 4,
  "images": { "total": 12, "broken": 0 },
  "forms": 1,
  "inputs": 3,
  "interactive": [
    {
      "tag": "a",
      "type": "",
      "text": "Sign In",
      "selector": "a.sign-in-btn",
      "visible": true
    },
    {
      "tag": "button",
      "type": "submit",
      "text": "Subscribe",
      "selector": "button.subscribe",
      "visible": true
    }
  ]
}
```

The `interactive` array lists up to 30 clickable/typeable elements with their CSS selectors — so the LLM knows exactly what it can interact with.

> **Prefer `POST /snapshot`** for LLM agents — it returns an a11y tree with `@ref` selectors that are more reliable than DOM's CSS selectors.

### `POST /api/session/{id}/fill` — Clear + Type

More reliable than `type` for replacing existing input values. Select-all → delete → type.

```json
{ "selector": "@e5", "text": "new value" }
```

Accepts CSS selector or `@ref` from snapshot.

### `POST /api/session/{id}/select` — Dropdown Option

```json
{ "selector": "@e8", "values": ["California"] }
```

Selects option by visible text or value. Multiple values for multi-select.

### `POST /api/session/{id}/press` — Keyboard Key

```json
{ "key": "Enter" }
```

Supported keys: `Enter`, `Tab`, `Escape`, `Backspace`, `Delete`, `Space`, `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`, `Home`, `End`, `PageUp`, `PageDown`.

Waits for DOM stability after keypress (handles form submissions, modal dismissals).

### `POST /api/session/{id}/back` — Browser History Back

No body required. Returns screenshot of previous page.

### `POST /api/session/{id}/forward` — Browser History Forward

No body required. Returns screenshot after navigating forward.

### `POST /api/session/{id}/text` — Get Element Text

```json
{ "selector": "@e12" }
```

No screenshot. Returns `{"text": "element text content", "selector": "@e12"}`.

### `POST /api/session/{id}/cookies` — Cookie Management

```json
// Get all
{ "action": "get" }

// Get by name
{ "action": "get", "name": "wp_logged_in" }

// Set
{ "action": "set", "name": "theme", "value": "dark", "domain": "example.com" }

// Clear all
{ "action": "clear" }

// Clear by name
{ "action": "clear", "name": "tracking" }
```

Returns `{"cookies": [...], "count": N}`.

### `POST /api/session/{id}/auth/save` — Save Auth State

No body. Returns JSON with cookies + localStorage + sessionStorage.

```json
{
  "url": "https://example.com",
  "cookies": [...],
  "localStorage": { "token": "abc123" },
  "sessionStorage": { "cart": "{...}" },
  "savedAt": "2026-02-09T11:08:13Z"
}
```

Save to file: `curl -s -X POST .../auth/save -o /tmp/auth-state.json`

### `POST /api/session/{id}/auth/load` — Load Auth State

Body: the JSON from `auth/save`.

```bash
curl -s -X POST .../auth/load -H "Content-Type: application/json" -d @/tmp/auth-state.json
```

Returns `{"status": "loaded"}`. Navigate after loading to trigger auth.

---

## LLM Tool Definitions

### OpenAI Function Calling Format

```json
[
  {
    "name": "browser_open",
    "description": "Open a persistent browser session on a URL. Returns the session ID and an initial screenshot. Use this to start browsing a webpage.",
    "parameters": {
      "type": "object",
      "properties": {
        "url": { "type": "string", "description": "URL to open" },
        "width": { "type": "integer", "description": "Viewport width (default: 1280)" },
        "height": { "type": "integer", "description": "Viewport height (default: 800)" }
      },
      "required": ["url"]
    }
  },
  {
    "name": "browser_screenshot",
    "description": "Take an instant screenshot of the current page state. No navigation — captures whatever is visible right now. Use for re-checking after changes.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string", "description": "Session ID from browser_open" },
        "fullPage": { "type": "boolean", "description": "Capture entire scrollable page" }
      },
      "required": ["session_id"]
    }
  },
  {
    "name": "browser_scroll",
    "description": "Scroll the page and take a screenshot. Use deltaY for relative scrolling (positive=down) or x/y for absolute position.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "deltaY": { "type": "integer", "description": "Pixels to scroll down (negative=up)" },
        "deltaX": { "type": "integer", "description": "Pixels to scroll right" },
        "y": { "type": "integer", "description": "Absolute scroll position" }
      },
      "required": ["session_id"]
    }
  },
  {
    "name": "browser_click",
    "description": "Click an element by CSS selector. Returns a screenshot after the click completes. Use browser_dom first to find available selectors.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "selector": { "type": "string", "description": "CSS selector of element to click" }
      },
      "required": ["session_id", "selector"]
    }
  },
  {
    "name": "browser_type",
    "description": "Type text into an input field. Clears existing text first.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "selector": { "type": "string", "description": "CSS selector of input element" },
        "text": { "type": "string", "description": "Text to type" }
      },
      "required": ["session_id", "selector", "text"]
    }
  },
  {
    "name": "browser_eval",
    "description": "Execute JavaScript on the page. Returns the result value and a screenshot. The JS runs inside an anonymous function — use 'return' for output.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "js": { "type": "string", "description": "JavaScript to execute (use 'return' for output)" }
      },
      "required": ["session_id", "js"]
    }
  },
  {
    "name": "browser_dom",
    "description": "Get the DOM structure of the current page without a screenshot. Returns headings, links, buttons, images, forms, and interactive elements with their CSS selectors. Use this to understand what you can click/type.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" }
      },
      "required": ["session_id"]
    }
  },
  {
    "name": "browser_navigate",
    "description": "Navigate to a new URL within the same session. Returns a screenshot of the new page.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "url": { "type": "string", "description": "URL to navigate to" }
      },
      "required": ["session_id", "url"]
    }
  },
  {
    "name": "browser_resize",
    "description": "Resize the browser viewport. Use to test responsive design (e.g., switch between mobile 375x812 and desktop 1440x900).",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "width": { "type": "integer" },
        "height": { "type": "integer" }
      },
      "required": ["session_id", "width", "height"]
    }
  },
  {
    "name": "browser_css",
    "description": "Inject CSS into the page to test visual changes. Replaces any previously injected CSS. Returns a screenshot showing the result.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" },
        "css": { "type": "string", "description": "CSS rules to inject" }
      },
      "required": ["session_id", "css"]
    }
  },
  {
    "name": "browser_close",
    "description": "Close a browser session and release resources. Always close sessions when done.",
    "parameters": {
      "type": "object",
      "properties": {
        "session_id": { "type": "string" }
      },
      "required": ["session_id"]
    }
  }
]
```

### MCP (Model Context Protocol) Tool Format

```json
{
  "tools": [
    {
      "name": "browser_open",
      "description": "Open a persistent browser session on a URL. Returns session_id + initial screenshot.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "url": { "type": "string" },
          "width": { "type": "integer", "default": 1280 },
          "height": { "type": "integer", "default": 800 }
        },
        "required": ["url"]
      }
    },
    {
      "name": "browser_screenshot",
      "description": "Instant re-screenshot of current page state (~30ms). No navigation.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "fullPage": { "type": "boolean", "default": false }
        },
        "required": ["session_id"]
      }
    },
    {
      "name": "browser_scroll",
      "description": "Scroll the page. deltaY>0 scrolls down. Returns screenshot.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "deltaY": { "type": "integer", "default": 600 }
        },
        "required": ["session_id"]
      }
    },
    {
      "name": "browser_click",
      "description": "Click element by CSS selector. Returns screenshot after click.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "selector": { "type": "string" }
        },
        "required": ["session_id", "selector"]
      }
    },
    {
      "name": "browser_type",
      "description": "Type text into input. Clears existing text.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "selector": { "type": "string" },
          "text": { "type": "string" }
        },
        "required": ["session_id", "selector", "text"]
      }
    },
    {
      "name": "browser_eval",
      "description": "Execute JavaScript. Returns result + screenshot.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "js": { "type": "string" }
        },
        "required": ["session_id", "js"]
      }
    },
    {
      "name": "browser_dom",
      "description": "Get page DOM structure: headings, links, interactive elements with selectors.",
      "inputSchema": {
        "type": "object",
        "properties": { "session_id": { "type": "string" } },
        "required": ["session_id"]
      }
    },
    {
      "name": "browser_navigate",
      "description": "Navigate to new URL in same session. Returns screenshot.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "url": { "type": "string" }
        },
        "required": ["session_id", "url"]
      }
    },
    {
      "name": "browser_resize",
      "description": "Resize viewport. Common: mobile 375x812, tablet 768x1024, desktop 1440x900.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "width": { "type": "integer" },
          "height": { "type": "integer" }
        },
        "required": ["session_id", "width", "height"]
      }
    },
    {
      "name": "browser_css",
      "description": "Inject CSS to test visual changes. Replaces previous injection.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "session_id": { "type": "string" },
          "css": { "type": "string" }
        },
        "required": ["session_id", "css"]
      }
    },
    {
      "name": "browser_close",
      "description": "Close browser session. Always call when done browsing.",
      "inputSchema": {
        "type": "object",
        "properties": { "session_id": { "type": "string" } },
        "required": ["session_id"]
      }
    }
  ]
}
```

---

## Architecture

```
LLM Agent
  │  browser_open("https://example.com")
  │  → POST /api/session {url}
  ▼
Session Manager
  │  Holds up to 4 persistent Chrome pages
  │  Each session: page + state + auto-expire timer (10min)
  ▼
Session abc12345
  │  browser_screenshot() → 30ms (just snap, no navigate)
  │  browser_scroll()     → 150ms
  │  browser_click()      → 300ms
  │  browser_dom()        → instant (no screenshot)
  │  browser_navigate()   → 1-2s (new URL, same page)
  ▼
Chrome (headless, shared with screenshot pool)
  │  System Chromium, memory-optimized flags
  │  Pages shared with transactional /api/screenshot pool
  ▼
Response: screenshot (base64) + metadata + DOM info
```

## Limits

| Resource | Limit | Notes |
|----------|-------|-------|
| Max sessions | 4 | Each holds ~50-100MB Chrome page |
| Idle timeout | 10 minutes | Auto-closes unused sessions |
| Session lifetime | unlimited | Active sessions never expire |
| Actions per session | unlimited | Counter tracked in session stats |
| Pages shared with | `/api/screenshot` pool | Same 4-page pool, sessions hold pages longer |

## Typical LLM Workflow

```
1. browser_open("https://mysite.com", width=390, height=844)
   → See the mobile homepage (screenshot + session_id)

2. browser_dom(session_id)
   → Know what's on the page (headings, links, buttons, selectors)

3. browser_scroll(session_id, deltaY=600)
   → See below the fold

4. browser_click(session_id, selector="nav a:nth-child(2)")
   → Navigate via click, see result

5. browser_css(session_id, css=".hero { padding: 20px; }")
   → Test a CSS tweak, see it instantly

6. browser_resize(session_id, width=1440, height=900)
   → Check desktop version of same page

7. browser_screenshot(session_id)
   → Quick re-check (30ms)

8. browser_close(session_id)
   → Done, release resources
```

## Performance Comparison

| Operation | Transactional API | Session API | Speedup |
|-----------|------------------|-------------|---------|
| First view | 1.7s | 1.7s | same |
| Re-check same page | 1.7s | **30ms** | **57x** |
| Scroll + check | impossible | **150ms** | ∞ |
| Click + check | impossible | **300ms** | ∞ |
| CSS change + check | impossible | **150ms** | ∞ |
| Resize + check | 1.7s | **300ms** | **6x** |
| 10 checks on 1 page | 17s | **1.7s + 9×30ms = 2s** | **8.5x** |

## Tool Discovery (Context-Efficient)

Tools are **never auto-loaded** into LLM context. Agents discover on demand:

```bash
# Minimal: just names + one-line descriptions (~200 tokens for 14 tools)
curl -s http://localhost:7456/api/tools/search | jq '.tools[]'

# Search: only matching tools returned
curl -s "http://localhost:7456/api/tools/search?q=click" | jq

# Full definitions when needed:
curl -s http://localhost:7456/api/tools/openai | jq   # OpenAI format
curl -s http://localhost:7456/api/tools/mcp | jq      # MCP format
```

### Pattern: Search → Discover → Call

```
Agent: "I need to click a button on a page"
  1. GET /api/tools/search?q=click  → finds browser_click
  2. Reads browser_click params     → needs session_id + selector
  3. POST /api/session/{id}/click   → clicks + returns screenshot
```

Cost: **~200 tokens** for tool discovery vs **~2000** if all 14 definitions were loaded upfront.

## MCP Integration

### Pi (Recommended)

Add to `~/.pi/agent/mcp.json`:

```json
{
  "mcpServers": {
    "browser": {
      "command": "node",
      "args": ["/home/wpuiai/uiai-engine/mcp/browser-session-mcp.mjs"],
      "lifecycle": "lazy",
      "idleTimeout": 60
    }
  }
}
```

Then in pi:
```
mcp({ search: "browser" })         → see available browser tools
mcp({ tool: "browser_open", args: '{"url": "https://example.com"}' })
mcp({ tool: "browser_screenshot", args: '{"session_id": "abc123"}' })
mcp({ tool: "browser_close", args: '{"session_id": "abc123"}' })
```

The bridge is **lazy** — Node process only starts when you first call a browser tool. Pi-mcp-adapter caches tool metadata, so `tools/list` is called once.

### Claude Desktop

Add to Claude Desktop MCP config:
```json
{
  "mcpServers": {
    "browser": {
      "command": "node",
      "args": ["/path/to/browser-session-mcp.mjs"],
      "env": { "UIAI_ENGINE_URL": "http://localhost:7456" }
    }
  }
}
```

### Any MCP Client

The bridge speaks MCP JSON-RPC over stdio:
- `initialize` → returns server capabilities
- `tools/list` → fetches tool definitions from Go engine
- `tools/call` → routes to session HTTP endpoints
- Screenshots returned as MCP `image` content blocks

## Port & Auth

- **Port:** 7456 (localhost only, behind Cloudflare tunnel externally)
- **Auth:** No auth required for localhost calls (auth bypass for `/api/session*`, `/api/tools*`)
- **External:** Requires `X-Webhook-Secret` header through Cloudflare tunnel
