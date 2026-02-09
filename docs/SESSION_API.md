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

# 2. See what's on the page (DOM structure)
curl -s http://localhost:7456/api/session/$SID/dom | jq

# 3. Screenshot current state (instant — no navigation)
curl -s -X POST http://localhost:7456/api/session/$SID/screenshot \
  -H "Content-Type: application/json" -d '{}' \
  | jq '{duration_ms, size, url, title}'

# 4. Scroll down
curl -s -X POST http://localhost:7456/api/session/$SID/scroll \
  -H "Content-Type: application/json" -d '{"deltaY":600}'

# 5. Click a button
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

### Session Actions (all return screenshot)

| Method | Path | Description | Typical Latency |
|--------|------|-------------|-----------------|
| `POST` | `/api/session/{id}/screenshot` | Re-snap current state | **30ms** |
| `POST` | `/api/session/{id}/scroll` | Scroll + screenshot | **150ms** |
| `POST` | `/api/session/{id}/click` | Click element + screenshot | **300ms** |
| `POST` | `/api/session/{id}/hover` | Hover element + screenshot | **200ms** |
| `POST` | `/api/session/{id}/type` | Type into input + screenshot | **150ms** |
| `POST` | `/api/session/{id}/eval` | Run JS + screenshot | **150ms** |
| `POST` | `/api/session/{id}/navigate` | Go to new URL + screenshot | **1-2s** |
| `POST` | `/api/session/{id}/resize` | Change viewport + screenshot | **300ms** |
| `POST` | `/api/session/{id}/css` | Inject CSS + screenshot | **150ms** |
| `POST` | `/api/session/{id}/wait` | Wait for selector + screenshot | **varies** |
| `GET` | `/api/session/{id}/dom` | DOM structure (no screenshot) | **instant** |

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

## Port & Auth

- **Port:** 7456 (localhost only, behind Cloudflare tunnel externally)
- **Auth:** No auth required for localhost calls (auth bypass for `/api/session*`)
- **External:** Requires `X-Webhook-Secret` header through Cloudflare tunnel
