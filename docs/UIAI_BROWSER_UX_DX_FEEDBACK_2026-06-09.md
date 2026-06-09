# UIAI Engine Browser — UX/DX Feedback (2026-06-09)

After a multi-hour E2E audit of the WPUIAI plugin at 4 canonical breakpoints
(375/768/1024/1440) across 11 admin pages, here are concrete UIAI Engine
browser improvements that would speed up agent workflows.

## Tier 1 — High-impact, low-effort

### 1. Screenshot workflow is 3 tool calls

**Today**: `uiai_browser_screenshot` returns a JSON envelope wrapping a base64-encoded
JPEG. To get an actual image file, the agent must do:

```bash
# 1. screenshot tool call
# 2. extract: jq -r .screenshot | base64 -d > /tmp/x.jpg
# 3. read the file
```

For 11 pages × 4 viewports = 44 screenshots, this is 132 tool calls instead
of 44.

**Fix**: Add `output` parameter to `uiai_browser_screenshot`:

- `output: "json"` (default, current behavior)
- `output: "file"` → write directly to a path the agent can `read` later
- `output: "url"` → return a server URL the agent can fetch

### 2. No multi-viewport batch screenshot

For responsive visual audits, agents need 4 viewports per page. Today this is
`navigate` → `resize` → `sleep` → `screenshot` × 4 = 16 calls per page.

**Fix**: Add `uiai_browser_screenshot_responsive`:

```json
{
  "url": "https://example.com",
  "viewports": [375, 768, 1024, 1440],
  "selector": ".main-content",
  "return": "files"
}
```

Returns 4 screenshot files matching the canonical breakpoints. Cuts a
responsive audit from ~64 tool calls to ~11.

### 3. Diagnostics buffer is unbounded

`uiai_browser_diagnostics` returned 124,248 bytes for a single page. The
`failed_only: true, level: error` filters don't actually filter the response —
they send the entire buffer. Agents that ask for diagnostics get
context-blowout.

**Fix**: Server-side filtering:

- `since_seq: 100` — only return events after sequence 100
- `category: "console|exceptions|network|all"`
- `level: "error|warning|info|all"` — actually filter
- `failed_only: true` — actually filter network
- `format: "summary"` — return just `{console: 3, exceptions: 0, network_4xx: 1, network_5xx: 0}`

Default to `format: "summary"`, opt-in to verbose.

### 4. No wait_for_stable between navigation and screenshot

After `uiai_browser_navigate`, agents have to insert `sleep N` calls then
re-screenshot to capture the post-load state. This is fragile — too short
and we capture mid-load, too long and we waste time.

**Fix**: Make `uiai_browser_navigate` block until:

- `domcontentloaded` (default)
- `load` event (default for screenshot purposes)
- `networkidle` (opt-in)
- Custom selector appears (opt-in via `wait_for_selector`)

Returns when ready. Or add `uiai_browser_wait_for` primitive with a
selector + timeout.

### 5. Selectors are limited

`uiai_browser_click` only supports CSS selectors — no `:has-text()`,
`:contains()`, or attribute `=` with non-ASCII values. Forces agents to
fall back to `uiai_browser_eval` for "click the button with text X":

```js
document.querySelectorAll('button').forEach(b => b.textContent.trim() === 'View' && b.click());
```

This is fragile (matching exact text including whitespace) and verbose.

**Fix**: Add a `text` selector type. e.g. `uiai_browser_click` accepts:

- `selector: "@ref"` — accessibility ref
- `selector: ".class"` — CSS (current)
- `selector: "text=Submit"` — Playwright-style text match
- `selector: "role=button[name=Save]"` — ARIA-based

## Tier 2 — High-impact, medium-effort

### 6. Sessions abort during long-running operations

`uiai_browser_click` on the Start button of a 30-step autopilot triggered a
30+ second backend operation. The session entered a state where subsequent
tool calls returned "This operation was aborted" for ~2 minutes.

**Fix**:

- Async tool calls: `uiai_browser_click --async` returns a request ID, agent polls status
- Auto-restart: detect session hangs and re-spawn with the same cookies/auth
- Or surface a "session timed out" error with the recovery URL/cookies so the agent can re-open

### 7. No file upload / drag-drop API

Form file uploads (e.g. Google Service Account JSON key) and drag-to-reorder
interactions have no API surface. Agents have to skip these flows in E2E
audits.

**Fix**: Add `uiai_browser_upload_file`:

```json
{
  "selector": "input[type=file]",
  "file_path": "/path/to/key.json"
}
```

And `uiai_browser_drag`:

```json
{
  "from": ".item-1",
  "to": ".item-3"
}
```

### 8. No async eval for multi-step JS

Some checks require multiple async operations (fetch + wait + DOM query). The
`uiai_browser_eval` is single-shot, forcing `new Promise` wrappers that
agents get wrong (got "Cannot read properties of undefined" several times).

**Fix**: Add `uiai_browser_eval_async` with explicit timeout:

```json
{
  "js": "await fetch('/api/x').then(r => r.json()); return document.querySelector('h1').textContent;",
  "timeout_ms": 5000
}
```

Already exists actually! But it's hard to discover and the documentation
doesn't make the async/Promise return value pattern obvious.

### 9. No page lifecycle / navigation events

Agents can `uiai_browser_navigate` but have no way to:

- Know when a SPA route change has occurred
- Get a list of frames/iframes
- Detect when a form submission has completed
- Get current URL after a redirect

**Fix**: `uiai_browser_state` returns:

- current URL
- current title
- viewport size
- readyState
- frame tree
- localStorage entries (if relevant)
- last network response code

## Tier 3 — Polish, medium-effort

### 10. No named sessions

Sessions are identified by opaque IDs (`LgwqFy4b`). If an agent does 20+
tool calls, they may forget which session they're in. Multiple sessions
look identical.

**Fix**: `uiai_browser_open({name: "wpuiai-settings-audit"})` and the session
shows up as "wpuiai-settings-audit" in diagnostics.

### 11. No structured way to save HTML / DOM

Agents can screenshot, but if a render bug is in the DOM, they can only
get a partial eval snippet. Would be useful to:

- `uiai_browser_get_html` → return rendered HTML as a file
- `uiai_browser_get_console` → filtered by severity, returned as file

### 12. Login is manual

Every test session has to re-authenticate with username/password. The
test admin credentials must be reset periodically (the audit had to do
this twice).

**Fix**: `uiai_browser_open` accepts a `auth_profile: "wpuiai-test-admin"`
that names a stored credential. CI/CD systems pre-configure these once.

### 13. Tab clicks via ref need a snapshot dance

To click a Settings tab, agents must:

1. `uiai_browser_snapshot` to get the a11y tree
2. Find the ref for "Content" tab
3. `uiai_browser_click` with `selector: ".wpuiai-tab-link[data-tab='content']"`

3 calls to click 1 button. With named sessions + selector shortcuts (#5),
this collapses to 1 call.

### 14. Failed requests show URL but not body

`uiai_browser_diagnostics` for a 400 admin-ajax response shows the URL
but not the request body. Agents have to repeat the request via eval to
see what payload was sent.

**Fix**: `uiai_browser_diagnostics` includes request body for failed
network requests, with a `truncate: 1024` parameter to keep response
sizes bounded.

### 15. Resize takes a screenshot as a side effect

Calling `uiai_browser_resize` always includes a full screenshot in the
response. For agents that just want to change viewport size, the
screenshot is wasted tokens.

**Fix**: `uiai_browser_resize({return_screenshot: false})`.

## Tier 4 — Power-user features

### 16. No stateful scripting

Complex E2E audits involve 100+ tool calls in a fixed sequence:
"open session → login → for each page: navigate, screenshot 375, resize, screenshot 768, ...".

A "scenario runner" would accept a JSON array of actions and run them in
sequence, returning only the final result set:

```json
{
  "scenarios": [
    {"action": "open", "url": "https://...", "name": "settings"},
    {"action": "navigate", "url": "https://..."},
    {"action": "input", "selector": "...", "value": "..."},
    {"action": "screenshot", "viewports": [375, 768, 1024, 1440]}
  ]
}
```

### 17. No reusable selectors / step library

After running E2E on Dashboard, the agent has identified `.wpuiai-settings-card`
as the main content selector. On a new session, this knowledge is lost.

**Fix**: `uiai_browser_save_session({name: "wpuiai", selectors: {main: ".wpuiai-settings-card", tabs: ".wpuiai-tab-link"}})`
and `uiai_browser_load_session({name: "wpuiai"})`.

### 18. No diffing vs baseline

A visual regression is a diff between today's screenshot and last week's.
The agent has to download both, run diff, interpret. A built-in
`uiai_browser_diff` against a reference image would be 10x faster.

### 19. No domain presets

WordPress, Shopify, Stripe, GitHub — each has its own admin structure.
A "domain preset" would auto-configure selectors and auth patterns.

### 20. Diagnostics aren't per-session

A failed network request from session A bleeds into session B's
diagnostics buffer. This caused confusion. Fix: diagnostics are
session-scoped (currently they appear to be global).

## Tier 5 — Agent ergonomics

### 21. "This operation was aborted" is unhelpful

The error message gives no context. A `code: 504, retryable: true,
last_tool: "uiai_browser_click", target_selector: ".uiai-autopilot-start"`
would let agents make informed decisions.

### 22. No batched input+submit+screenshot

```json
{
  "uiai_browser_action": "submit_form",
  "selector": "#mainform",
  "expect_success": true,
  "capture_before": true,
  "capture_after": true
}
```

Cuts 5 calls to 1 for the most common audit interaction.

### 23. Documentation surface

The skill at `/root/.pi/skills/wpuiai-workflow` is good but doesn't include
a "common UIAI patterns" section. An agent learning the tool chain would
benefit from a cookbook of:

- "How to do a responsive visual audit" (current 64 calls → 16 calls)
- "How to test a form" (current 5 calls → 1 call)
- "How to capture only errors" (current 124KB → 1KB)

## Summary

The biggest DX wins (in priority order):

1. **Direct file output for screenshots** — eliminates curl/base64 dance
2. **Multi-viewport batch screenshot** — 4× speedup for visual audits
3. **Real diagnostic filtering** — server-side, not just response filtering
4. **Auto-wait on navigate** — no manual sleep calls
5. **Text-based selectors** — `text=Submit`, `role=button[name=Save]`

These five changes alone would cut a typical 11-page responsive audit from
~200 tool calls and 500KB of context bloat down to ~30 calls and 50KB.
