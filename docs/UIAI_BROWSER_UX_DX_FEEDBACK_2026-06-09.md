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

---

## Round 2 — Additional Observations (2026-06-09)

After shipping the first cut, the audit continued and surfaced a deeper
set of friction points. These are organized by the operation they
impede.

### Session lifecycle & navigation

### 24. Viewport state doesn't persist predictably across navigations

After `uiai_browser_resize({width: 375})` succeeded, the next
`uiai_browser_navigate` occasionally reset the viewport (likely to
1280 default). Agents have to re-resize after every navigation. Fix:
make `resize` state sticky per session, or include `viewport` in
`navigate` parameters.

### 25. No way to enumerate open sessions

I had to remember session IDs like `LgwqFy4b` mentally across many
turns. `uiai_browser_list_sessions` would return `[{id, url, viewport,
last_used}]` so agents can pick the right one.

### 26. `uiai_browser_close` is not always effective

After the session entered a hung state, `uiai_browser_close` returned
"This operation was aborted" instead of confirming the close. A
"force: true" parameter or async close would help.

### 27. Snapshot refs (`@e1`) don't survive navigation

The `uiai_browser_snapshot` returns refs that are valid for that
single render. After a redirect, navigation, or even a single SPA
route change, refs are invalid and must be re-snapshotted. Fix: re-snapshot
implicitly on click failure when a ref is stale.

### 28. No "session resume" after disconnect

When a session got killed (long-running op, server restart, etc.),
all in-flight context (cookies, localStorage, scroll position) was
lost. A way to persist and resume a session's `storage_state` would
prevent re-doing the login every time.

### Selector & interaction limits

### 29. No `:has-text()` or text-based selectors

I couldn't click the "View" link in the runs table using a CSS
selector alone — I had to evaluate JS to find it by `textContent`.
Playwright's `text=...` and `role=...` selector engines would cut
this friction.

### 30. `click` fails silently on hidden elements

When I tried to click a button that was visually obscured (e.g.
behind the DevConsole panel at the bottom), the tool returned
"operation was aborted" without telling me whether the element
existed, was hidden, or was outside the viewport. Adding a
"reason" field to the error would help agents recover.

### 31. Click on the FIRST matching element only

`uiai_browser_click(".button")` always picks the first match. For
tables with multiple rows, agents have to fall back to `eval` with
`[index]`. Adding `index: 2` or `nth-of-type(n)` would be cleaner.

### 32. `fill` is not atomic

It types character-by-character, which:
- Triggers IME/IME-composition events on certain inputs
- Can be slow for 200+ char URLs
- Doesn't always survive `[type=password]` fields with autocomplete

A `set_value` (atomic) alternative would help.

### 33. No upload / drag-drop API

The Google Drive setup wizard requires uploading a service-account
JSON key. Without `uiai_browser_upload_file`, that flow cannot be
E2E-tested.

### 34. No file picker / native dialog API

Browser-level "save as" / "print" / native pickers have no API.
Agents have to skip the entire flow or workaround via mocks.

### 35. No mouse-coordinate events

Click is element-based. Some interactions (drag-from-here-to-there
without a clear target, hovering near an element) need x/y events.

### 36. No keyboard navigation primitives

There's `uiai_browser_press` for single keys but no "tab through a
form" / "shift+tab" / "ctrl+A" sequences. Accessibility audits need
these.

### 37. ARIA testing primitives missing

axe-core integration would let agents run `uiai_browser_a11y_check`
to surface accessibility violations alongside the visual audit. The
current workflow is `eval` to inject axe + read violations manually.

### State setup & environment control

### 38. No way to inject cookies / localStorage for state

For testing "logged in as plan X" or "wpuiai_dev=true", I have to run
`eval` to seed localStorage. A `uiai_browser_state({localStorage:
{key: value}, cookies: [...]})` would standardize this.

### 39. No custom user-agent string

For mobile-emulated tests (UA-switching), the viewport resize alone
isn't enough. The page may serve different HTML for desktop vs
mobile UA.

### 40. No `prefers-color-scheme: dark` media emulation

Most modern UIs have a dark mode. There's no way to test it without
a real OS theme change. `uiai_browser_emulate_media({colorScheme:
"dark"})` would solve this.

### 41. No `prefers-reduced-motion` emulation

Same problem for the reduced-motion accessibility setting.

### 42. No way to inject response interceptors

For testing "what if the API returns 503?" I have to break the
backend. Playwright's `route.fulfill()` would be a clean win.

### 43. No CPU / network throttling

Cannot simulate slow connections or low-end devices. The
`uiai_browser_throttle` primitive would unlock perf-budget testing.

### 44. No way to disable JS / CSS

For testing no-JS fallback, or for performance isolation.

### 45. No cache-bypass header injection

Forcing a "fresh load" requires cache-busting query params on the
URL. A `--no-cache` flag or `cache-control: no-cache` injection would
standardize this.

### Diagnostic & observability

### 46. Console messages don't show source maps

Stack traces from bundled JS show `/app.js:1:1` instead of
`webpack://src/utils.ts:42:7`. Makes debugging 10x harder.

### 47. Console output is mixed with network output

`uiai_browser_diagnostics` returns console + exceptions + network
in one big array. Hard to scan. Splitting into 3 sub-calls
(`uiai_browser_console`, `uiai_browser_network`, `uiai_browser_exceptions`)
would be much more usable.

### 48. No console-message capture during navigation

Only after-the-fact. Errors that flash during the in-flight
`navigate` are sometimes missed because the diagnostics buffer
wasn't subscribed yet. Auto-subscribing on `open` would fix this.

### 49. Network responses don't show request body for 4xx/5xx

Just the URL. Agents that try to debug "why did the form submit
fail?" have to re-issue the request manually with `eval`.

### 50. Failed network requests are mixed with successful ones in
timing data

Hard to reason about TTFB for failures.

### 51. Diagnostics aren't session-scoped

A failed request from session A showed up in session B's
diagnostics buffer. This caused confusion in the audit. Fix:
diagnostics should be keyed by session_id, and `uiai_browser_diagnostics`
should default to the current session only.

### 52. No structured diff/assertion language

After filling a form, agents want to "wait for element X to contain
text Y". A `uiai_browser_expect({selector, text, timeout})` would
turn 4 tool calls (input, click, sleep, snapshot) into 1.

### 53. No console-message color coding

Error lines look identical to info lines in the diagnostic output.
Severity-based color or icon would let agents scan faster.

### 54. No focus-state reporting

After clicking an input, the snapshot doesn't tell me whether it
received focus. A `uiai_browser_focused()` returning
`{tag, id, name}` would help a11y audits.

### 55. No ARIA-landmark dump

The "no main landmark" or "no skip link" check requires a separate
axe-core invocation. A `uiai_browser_landmarks` would be one call.

### Composability

### 56. No way to test "what happens if I refresh?"

For SPA route testing, hitting "refresh" needs a separate
`uiai_browser_reload` primitive. Currently I have to `navigate` to
the same URL which doesn't really test the back/forward cache.

### 57. No way to test browser back/forward

`uiai_browser_back` and `uiai_browser_forward` exist but
diagnostics aren't captured during those transitions.

### 58. No "navigate + fill + click + screenshot" composite

The most common E2E pattern is 5 sequential calls. A scenario
runner (mentioned in #16) would absorb this.

### 59. No way to record a video

For visual E2E demos and walkthroughs, a `.webm` of the session
would be invaluable.

### 60. No way to compare a screenshot against a reference image

For visual regression, a hash-based diff (`uiai_browser_diff({
reference: "screenshot-2026-05-01.png" })`) would be a 10x speedup
over manual `compare` runs.

### 61. No reusable session profiles

After the first E2E audit, I knew the WPUIAI admin selectors
(`.wpuiai-settings-card`, `.wpuiai-tab-link`, etc.). A way to save
and reload these as a named profile would let subsequent audits
start in 1 call.

### 62. No domain presets

WordPress admin, Stripe dashboard, GitHub settings — each has a
known structure. A "domain preset" that auto-configures selectors,
auth patterns, and known breakpoints would be a huge productivity
win.

### Error message quality

### 63. "This operation was aborted" gives no context

A 504-style error with `code`, `retryable`, `last_tool`, and
`target_selector` fields would let agents reason about what to do
next. Currently I have to retry the same call 3+ times.

### 64. No 5xx-vs-4xx distinction in error envelopes

"Failed" booleans don't say why. Adding an `error_class` field
(timeout, network, abort, oom, etc.) would help.

### 65. No "stuck session" auto-detection

If a session hasn't had tool calls in 60s, maybe surface "session
idle" so the agent knows to close it before re-opening.

### Positives (what's working well)

### 66. `uiai_browser_snapshot` (a11y) tree is well-structured

The `@ref` system is intuitive and surfaces roles + names + states
correctly. Don't change it; just make refs persistent.

### 67. `uiai_browser_eval` is fast

Even with multi-line scripts wrapped in `new Promise()`, eval
returns are sub-second. No complaints.

### 68. `uiai_browser_open` returns useful metadata

`focusa_scope`, viewport size, last_used — all present and correct.

### 69. The existing docs in `/docs` (AGENT_UX_COOKBOOK, BROWSER_DIAGNOSTICS_SPEC, UIAI_FOR_AGENTS_QUICKSTART) are great

These three documents are a model. Any new feature should update
them in the same commit.

### Summary of Round 2

Round 1 captured the highest-impact DX wins. Round 2 fills in the
~50 medium-to-low priority items that would each save 1-3 tool calls
per audit. Together, the two rounds would let a typical 11-page
responsive visual regression audit drop from:

- Current: ~200 tool calls, ~500KB context bloat, 30+ minutes wall time
- After Round 1: ~30 calls, ~50KB, 8 minutes
- After Round 2: ~15 calls, ~20KB, 4 minutes

The top 5 from Round 2 (in priority order):

1. **Server-side diagnostic filtering** (re-listed, since the
   `level=error` filter is currently broken)
2. **Viewport state sticky across navigations** (no more surprise
   resize re-runs)
3. **`uiai_browser_expect` primitive** (single-call form assertion)
4. **`uiai_browser_state` primitive** (set localStorage/cookies
   for state setup)
5. **axe-core integration** (`uiai_browser_a11y_check` for one-call
   a11y audits)

---

## Round 3 — Focusa ↔ UIAI Integration (and remaining items)

After 4+ hours of intensive E2E work, the largest remaining DX win is
**deepening the Focusa↔UIAI integration**. Today the two systems are
loosely coupled — Focusa knows about workpoints, evidence, and
predictions, but UIAI is a black box that emits base64 JPEGs that
agents manually translate into Focusa evidence refs.

This round catalogues the integration gaps, the remaining tool-surface
items, and a north-star vision for what UIAI could become as a
first-class agent browser.

### Focusa ↔ UIAI integration gaps

### 70. Every UIAI screenshot needs manual translation into Focusa evidence

**Today**:
```python
screenshot_json = uiai_browser_screenshot()           # returns JSON envelope
b64 = jq -r .screenshot screenshot.json               # extract
open file.jpg with base64 -d                          # write file
read file.jpg                                        # load into context
focusa_evidence_capture(target_ref, result, evidence_ref)  # link by hand
```

That's 4-5 tool calls per screenshot, plus a string-typed
`evidence_ref` that means nothing to the system.

**Fix**: Add a `uiai_browser_capture_evidence` that:
- Takes the screenshot
- Writes to a server-known path
- Returns a structured `evidence_handle` that Focusa can resolve
- Auto-invokes `focusa_evidence_capture` with the right `target_ref`

```json
{
  "uiai_browser_capture_evidence": {
    "viewport": [1280, 800],
    "target_ref": "https://wpuiai.com/admin.php?page=wpuiai",
    "result": "WPUIAI Settings page rendered with DevConsole settings card",
    "auto_attach_to_workpoint": true
  }
}
```

Returns:
```json
{
  "evidence_handle": "uiai:session=X:screenshot=N",
  "workpoint_attached": true,
  "context_window_saved_bytes": 46000
}
```

### 71. UIAI sessions are not first-class Focusa objects

`uiai_browser_open` returns `LgwqFy4b` as a session ID. Focusa has
no knowledge of it. If an agent switches contexts, it has to
remember which session was open, what page it was on, what
viewports it was testing, and which auth profile was in use.

**Fix**: Treat UIAI sessions as `focusa_active_object_resolve`-able
resources:

```json
{
  "object_type": "uiai_session",
  "object_ref": "LgwqFy4b",
  "properties": {
    "url": "https://wpuiai.com/wp-admin/admin.php?page=wpuiai-settings",
    "viewport": [1280, 800],
    "auth_profile": "wpuiai-test-admin",
    "created_at": "2026-06-09T22:01:00Z",
    "last_used": "2026-06-09T22:14:32Z"
  }
}
```

Then `focusa_active_object_resolve` can return sessions as
inspectable, linkable objects in the trajectory. A new "WebSession"
object type would integrate naturally with the existing ontology.

### 72. No bi-directional workpoint → browser context

When Focusa has an active workpoint with `current_action: "submit_form_for_tab=ai&setting=max_iterations"`, UIAI has no way to know this. The agent must manually translate workpoint state into UIAI tool calls.

**Fix**: Implement `focusa_to_uiai_bridge` that, given a workpoint
current_action, generates the right sequence of UIAI tool calls:

```python
focusa_to_uiai_bridge(workpoint_id)
# Returns:
{
  "action_sequence": [
    {"uiai_browser_navigate": {"url": "https://wpuiai.com/wp-admin/admin.php?page=wpuiai-settings&tab=ai"}},
    {"uiai_browser_fill": {"selector": "[name=uiai_autopilot_max_iterations]", "value": "5"}},
    {"uiai_browser_click": {"selector": "input[name=submit]"}},
    {"uiai_browser_expect": {"selector": ".notice", "text": "Saved"}}
  ]
}
```

### 73. Diagnostics are anonymous to Focusa

When UIAI reports "1 failed network request" in diagnostics, Focusa
has no way to correlate it to the action that caused it. The
diagnostic event has no `cause_action_id`, no `workpoint_id`, no
`active_object_ref`. The agent has to be the bridge.

**Fix**: When UIAI emits a diagnostic event, it should be tagged with
the action that produced it:

```json
{
  "diagnostic_event": {
    "type": "network_failure",
    "url": "/admin-ajax.php",
    "status": 400,
    "action_id": "uiai:fill:uiai_unsplash_app_id:999888777",
    "workpoint_id": "019eae3c-...",
    "session_id": "LgwqFy4b"
  }
}
```

`focusa_browser_diagnostics_intake` should auto-link each event to the
right Focusa evidence/workpoint.

### 74. `focusa_tool_doctor` doesn't know about UIAI

When UIAI hangs or returns "operation was aborted", the agent
should be able to ask `focusa_tool_doctor("uiai_browser_click hung after Start button")` and get back a structured recovery plan.

**Fix**: `focusa_tool_doctor` should accept a `tool_family: "uiai"`
parameter and return:

```json
{
  "diagnosis": "uiai_session_hang",
  "root_cause": "long-running_backend_op",
  "recovery_actions": [
    {"action": "uiai_browser_close", "params": {"session_id": "...", "force": true}},
    {"action": "uiai_browser_open", "params": {"auth_profile": "wpuiai-test-admin"}},
    {"action": "uiai_browser_navigate", "params": {"url": "..."}}
  ]
}
```

### 75. No shared auth model between Focusa and UIAI

UIAI has no concept of `auth_profile`. Every UIAI session requires
the agent to manually re-authenticate with username/password. Focusa
has a `continuity_id` and knows the agent's identity, but the
browser never inherits that.

**Fix**: Introduce auth profiles that both systems share:

```json
{
  "uiai_browser_open": {
    "auth_profile": "wpuiai-test-admin",
    "url": "https://wpuiai.com/wp-admin/"
  }
}
```

Focusa would store the profile under `focusa://auth/wpuiai-test-admin`
and pass credentials automatically. CI systems configure the
profile once, every subsequent audit reuses it.

### 76. Evidence capture is one-way

I can `focusa_evidence_capture(screenshot, target_ref)` but I can't
`focusa_evidence_retrieve(target_ref, type='screenshot')` and have
it open a UIAI session to re-shoot for comparison.

**Fix**: Add a `uiai_browser_recapture({evidence_ref: "..."})` that
finds the evidence, parses out the URL/viewport/selector context,
opens a session, navigates, and re-screenshots. Returns a new
evidence handle and a visual diff.

### 77. No "what changed" between evidence points

When I do `focusa_evidence_capture` twice, Focusa has both but no
diff. To compare I have to download both, run `compare -metric`
manually, interpret. A `focusa_evidence_diff` would do this
natively.

### 78. No way to scope UIAI sessions to Focusa workpoints

When a workpoint is the current Focusa context, UIAI should know:
- Which auth profile to use
- Which URL to start at (from workpoint.target_objects)
- Which evidence_handle to attach new evidence to
- When the workpoint advances, the session should follow (e.g. close
  when workpoint ends)

**Fix**: `uiai_browser_open({workpoint: "019eae3c-..."})` would
pre-fill all this.

### 79. No way to report UIAI state into Focusa trajectory

When UIAI does a long action and the workpoint is "verify
critique-page-375px", the trajectory should know:
- UIAI started the action
- UIAI captured screenshot evidence
- UIAI reported OK/FAIL

Today this is all manual. `uiai_action_log` should auto-feed
`focusa_trajectory_view` so the trajectory shows what the agent did
on the browser.

### 80. No "session memory" between Focusa sessions

When a Focusa continuity resumes (after compaction or model swap),
UIAI sessions opened by the previous agent are forgotten. The
agent has to re-discover, re-open, re-navigate, re-auth.

**Fix**: When `focusa_workpoint_resume` is called, also resume the
UIAI session that was last active. Store `last_uiai_session_id` in
the workpoint packet.

### 81. No "uiai session as Focusa context" pattern

Focusa has a powerful "context" abstraction (workpoint, current
action, evidence, predictions). UIAI has session state. They're
disjoint. A unified mental model would be:

```
Workpoint
├── trajectory
│   ├── waypoints[]
│   ├── workpoints[]
├── active_object (current_url)
├── evidence (uiai screenshots + uiai DOM snapshots)
├── state (uiai session_id, viewport, auth_profile, localStorage)
└── next_action (uiai tool sequence)
```

Currently the agent has to manually maintain all of these in their
context window.

### Remaining tool-surface items

### 82. No way to detect if a button is disabled before clicking

`uiai_browser_click(disabled_button)` silently fails. An
`is_enabled` / `is_visible` / `is_in_viewport` check before click
would prevent wasted attempts.

### 83. No "double-click" or "right-click" primitives

Some admin UIs (e.g. context menus, double-click edit) need these.

### 84. No "key sequence" primitive

`uiai_browser_press('Tab')` works, but `uiai_browser_press_seq(['Tab','Tab','Tab','Enter'])` for form submission via keyboard would be useful.

### 85. No way to mock `Date.now()` / `setTimeout` for time-traveling tests

For testing scheduled actions, autopilot runs, cron jobs.

### 86. No "freeze UI" / "throttle animations" mode

Animations make screenshots non-deterministic. UIAI should
auto-pause CSS animations / Web Animations API during screenshots.

### 87. No "stable screenshot" primitive

`uiai_browser_screenshot({stable: true})` should wait for:
- `requestAnimationFrame` quiescent for 100ms
- No pending `setTimeout` / `fetch`
- No CSS animations running
- All images loaded
- All web fonts loaded

This eliminates the need for explicit `sleep` calls.

### 88. `fill` with select elements requires `select_option`, not `fill`

Inconsistent API surface for different form controls. A unified
`uiai_browser_set({selector, value})` with smart field-type
detection would be cleaner.

### 89. No "find all interactive elements" primitive

`uiai_browser_list_buttons` / `_list_inputs` / `_list_links` would
be a fast way to discover a page's affordances without evaluating
JS or full snapshot.

### 90. No "scroll to element" before screenshot

I had to do `eval(el.scrollIntoView())` before screenshot. A
`screenshot({selector, scroll_to: true})` would be one call.

### 91. Console-message capture is lossy

Only the last N messages are kept. Long-running operations can
lose context.

### 92. Network response bodies are not captured for success

Only failure responses show their body. For perf audits or
"what's the API returning?" questions, success-body capture would
help.

### 93. No way to take a screenshot of a specific element only

`uiai_browser_screenshot({selector: ".uiai-settings-card"})` would
clip to a region, save tokens, and focus the audit.

### 94. No "compare against saved baseline" workflow

A `.uiai baseline/` directory convention where `uiai_browser_screenshot` automatically diffs against a saved baseline and returns the diff percentage.

### 95. No "generate accessibility report" primitive

axe-core integration would be a one-call way to surface
accessibility issues alongside visual audits.

### 96. No "test keyboard nav" primitive

`uiai_browser_keyboard_nav({start: "first_input", direction: "tab"})` would test that tab order is sensible.

### 97. No "check focus trap" for modals

`uiai_browser_focus_trap({modal: ".uiai-modal"})` would test
that focus stays within the modal.

### 98. No "check color contrast" for text

`uiai_browser_contrast({selector: "h1", min_aa: 4.5})` would
audit WCAG compliance on rendered text.

### Workflow / DX patterns

### 99. The skill at `/root/.pi/skills/wpuiai-workflow` doesn't have a "common UIAI patterns" section

The skill is named `wpuiai-workflow` not `uiai-engine` — confusing.
It also lacks a "UIAI cookbook" subsection for common E2E patterns
(responsive audit, form test, navigation tree, etc.).

### 100. No "starter template" for E2E audit scenarios

A canonical "responsive visual audit" JSON template with placeholders
for URL, selectors, viewports would let new agents ramp up
immediately.

### 101. No replay/redo for failed interactions

When `uiai_browser_click` fails, the agent has to manually retry.
A `uiai_browser_retry` with exponential backoff and selector
relaxation (e.g. drop `:nth-child(2)`) would be a huge win.

### 102. No structured "scenario diff"

After running scenario A and scenario B on the same page, agents
have no way to know what changed. `uiai_scenario_diff({a, b})`
would surface a structured diff (which elements changed, which
events fired, which network calls happened).

### 103. No way to export a session as a replayable script

A UIAI audit should be able to:
```bash
uiai audit run --url=... --viewports=375,768,1024,1440 \
  --output=audit-2026-06-09.json
uiai audit replay --input=audit-2026-06-09.json
```

So the audit is reproducible. Today, the audit is one-shot and
not replayable.

### 104. No "lint" for the test session itself

If an agent's E2E pattern has 5 redundant tool calls (e.g.
"check state, check state again, check state again, click"), a
linter would flag it.

### 105. No way to mark a screenshot as "golden" / "baseline"

Once a designer approves a screenshot, the agent should be able to
mark it as the baseline for future diffs. `uiai_browser_save_golden({evidence_ref: "..."})` would do this.

### Aesthetics of the tool itself

### 106. The `format: jpeg` default is silent

When `uiai_browser_screenshot` is called, the response includes
`"format": "jpeg"` but not "this is base64-encoded". A new agent
has to read the docs to find out.

### 107. Tool names are inconsistent

`uiai_browser_screenshot` (verb), `uiai_browser_diagnostics`
(noun), `uiai_browser_open` (verb), `uiai_browser_health` (noun).
Pick one convention.

### 108. Parameter naming inconsistent

`viewport: [375, 800]` vs `width: 375, height: 800` (used elsewhere)
vs `at_size: 375,800`. Pick one.

### 109. The "screenshot" tool's success response is just metadata

A new agent would expect the screenshot bytes. Returning
`width, height, format, size, url, title, duration_ms` looks like
an error. A first-line note would help: "screenshot captured; use
jq to extract".

### 110. No `--help` parameter to see tool schema

Agents have to read README or skill docs to learn parameters.
Inline help would speed onboarding.

### Reliability concerns

### 111. Session pool is bounded at 2

`max_pages: 2` is a hard limit. Agents with parallel investigations
need to wait. A queue with `priority` parameter would help.

### 112. "This operation was aborted" is the only error

There's no 504 vs 502 vs timeout distinction. Agents retry blindly.

### 113. No automatic session recovery

When a session dies mid-action, the agent has to manually re-open
and re-navigate. A "auto-resume" mode would re-open with the same
cookies/URL.

### 114. The `127.0.0.1:7456` only works from one machine

If the agent's tool call goes through a different host (e.g. a CI
runner), it fails. The skill doesn't document this constraint.

### 115. No way to know the engine's health before opening a session

`uiai_browser_health` exists, but it returns capacity stats, not
readiness (e.g. "is the engine process running?"). Agents open
sessions blind.

### North-star vision

### 116. **UIAI as a "do this on the web" primitive, not a browser controller**

The biggest shift would be moving from "control a Chromium browser"
to "do this on the web". Higher-level primitives:
- `uiai_do({intent: "log in to wpuiai.com", then: "click Settings", then: "fill license key"})`
- `uiai_scenario({actions: [...]})` with semantic intent per action
- `uiai_extract({url, query: "what's the user's subscription plan?"})`

### 117. **Focusa-aware UIAI: the UIAI session IS the workpoint's web context**

Today: Workpoint is abstract, UIAI is a separate tool.
Tomorrow: A UIAI session is the concrete realization of a workpoint's
"web context". Workpoint advances → browser navigates. Browser
captures evidence → workpoint gains evidence. Browser hits an
error → workpoint knows to backtrack.

### 118. **UIAI should learn from past audits**

The same E2E audit (11 pages, 4 viewports, 5 input tests) shouldn't
be re-discovered every time. A `uiai_audit_history` that suggests
"you ran this scenario yesterday; replay?" would be 10x faster than
re-deriving the action list.

### 119. **Per-tool surface contract tests**

Each UIAI tool should have a small self-test that runs on engine
startup. If `uiai_browser_diagnostics` is broken (e.g. `level`
filter ignored), an admin should see a warning, not just a confused
agent.

### 120. **A "UIAI playground" in the engine repo**

A static HTML page in `uiai-engine/playground/` with all primitives
exercised in a single page, useful for both human onboarding and
CI smoke tests. Currently agents have to set up a real WordPress
admin to test UIAI workflows.

### Final tally

- **Rounds 1+2+3**: 120 distinct feedback items
- **Top 5 from Round 3 (Focusa↔UIAI integration)**:
  1. `uiai_browser_capture_evidence` — auto-route to Focusa evidence
  2. UIAI sessions as first-class Focusa objects
  3. `focusa_tool_doctor` understands UIAI failures
  4. Shared `auth_profile` between Focusa and UIAI
  5. `focusa_to_uiai_bridge` — generate UIAI action sequences from workpoints

If UIAI implemented the top 5 from each round (15 total), an agent
E2E audit would drop from:

- Today: ~200 calls, ~500KB context, 30 min wall
- Tomorrow: ~15 calls, ~5KB context, 3 min wall

That is the "north star" for the engine.
