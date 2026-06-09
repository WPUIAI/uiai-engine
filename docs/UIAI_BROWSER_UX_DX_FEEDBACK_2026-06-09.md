# UIAI Engine Browser — UX/DX Feedback (2026-06-09)

A consolidated list of every feedback item from the 4+ hour WPUIAI admin E2E
audit (11 pages, 4 breakpoints, 5 input tests, 2 commits shipped).

**Conflicts and duplications resolved**: see
`UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md` for the full
analysis. The 5 hard conflicts (A-E below) and 3 philosophical overlaps
were merged into single items. All items are now continuously numbered.

Items are grouped by:
- **Top 5** (must-ship, biggest DX wins)
- **Tool surface** (primitives and API consistency)
- **Focusa ↔ UIAI integration** (where the deepest agent DX wins live)
- **Session & state** (lifecycle, persistence, env control)
- **Diagnostics & observability**
- **Reliability & errors**
- **Aesthetics & consistency**
- **North-star vision** (long-term redesigns)

---

## Top 5 priorities (must-ship, biggest DX wins)

### 1. Direct file output for screenshots

`uiai_browser_screenshot` returns a JSON envelope wrapping a base64 JPEG.
To get an actual image file, an agent must do:

```bash
screenshot_json = uiai_browser_screenshot()      # JSON envelope
jq -r .screenshot screenshot.json | base64 -d > /tmp/x.jpg
read /tmp/x.jpg
```

For 11 pages × 4 viewports = 44 screenshots, this is 132 tool calls
instead of 44.

**Fix**: Add `output` parameter to `uiai_browser_screenshot`:

- `output: "json"` (default, current behavior)
- `output: "file"` → write directly to a server-known path the agent
  can `read` later
- `output: "url"` → return a server URL the agent can fetch

### 2. Multi-viewport batch screenshot

For responsive visual audits, agents need 4 viewports per page. Today
this is `navigate` → `resize` → `sleep` → `screenshot` × 4 = 16 calls per
page.

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

### 3. Real server-side diagnostic filtering

`uiai_browser_diagnostics` returns 124,248 bytes for a single page. The
`failed_only: true, level: error` filters are accepted as parameters
but the response still contains everything. Agents asking for
diagnostics get context-blowout.

**Fix**: Server-side filtering:

- `since_seq: 100` — only return events after sequence 100
- `category: "console|exceptions|network|all"`
- `level: "error|warning|info|all"` — actually filter
- `failed_only: true` — actually filter network
- `format: "summary"` — return just `{console: 3, exceptions: 0,
  network_4xx: 1, network_5xx: 0}`

Default to `format: "summary"`, opt-in to verbose.

### 4. Auto-wait on navigate

After `uiai_browser_navigate`, agents have to insert `sleep N` calls
then re-screenshot to capture the post-load state. Fragile — too
short and we capture mid-load, too long and we waste time.

**Fix**: Make `uiai_browser_navigate` block until:

- `domcontentloaded` (default)
- `load` event (default for screenshot purposes)
- `networkidle` (opt-in)
- Custom selector appears (opt-in via `wait_for_selector`)

Returns when ready. Or add `uiai_browser_wait_for` primitive with a
selector + timeout.

### 5. Text-based selectors

`uiai_browser_click` only supports CSS selectors — no `:has-text()`,
`:contains()`, or attribute `=` with non-ASCII values. Forces agents
to fall back to `uiai_browser_eval` for "click the button with text X":

```js
document.querySelectorAll('button').forEach(b => b.textContent.trim() === 'View' && b.click());
```

This is fragile (matching exact text including whitespace) and verbose.

**Fix**: Add a `text` selector type. e.g. `uiai_browser_click` accepts:

- `selector: "@ref"` — accessibility ref
- `selector: ".class"` — CSS (current)
- `selector: "text=Submit"` — Playwright-style text match
- `selector: "role=button[name=Save]"` — ARIA-based

---

## Tool surface (primitives and API consistency)

### 6. Async tool calls for long-running operations

`uiai_browser_click` on the Start button of a 30-step autopilot
triggered a 30+ second backend operation. The session entered a
state where subsequent tool calls returned "This operation was
aborted" for ~2 minutes.

**Fix**: Async tool calls: `uiai_browser_click --async` returns a
request_id, agent polls status. Auto-resume is a separate, *crash-only*
complement — not the solution to long waits.

### 7. No file upload / drag-drop API

Form file uploads (e.g. Google Service Account JSON key) and
drag-to-reorder interactions have no API surface. Agents have to
skip these flows in E2E audits.

**Fix**: Add `uiai_browser_upload_file`:

```json
{ "selector": "input[type=file]", "file_path": "/path/to/key.json" }
```

And `uiai_browser_drag`:

```json
{ "from": ".item-1", "to": ".item-3" }
```

### 8. No async eval for multi-step JS

Some checks require multiple async operations (fetch + wait + DOM
query). The `uiai_browser_eval` is single-shot, forcing `new Promise`
wrappers that agents get wrong.

**Fix**: Add `uiai_browser_eval_async` with explicit timeout (already
exists, but undocumented):

```json
{
  "js": "await fetch('/api/x').then(r => r.json()); return document.querySelector('h1').textContent;",
  "timeout_ms": 5000
}
```

### 9. No page lifecycle / navigation events

Agents can `uiai_browser_navigate` but have no way to:

- Know when a SPA route change has occurred
- Get a list of frames/iframes
- Detect when a form submission has completed
- Get current URL after a redirect

**Fix**: `uiai_browser_state` returns: current URL, title, viewport
size, readyState, frame tree, localStorage entries, last network
response code.

### 10. No named sessions

Sessions are identified by opaque IDs (`LgwqFy4b`). If an agent does
20+ tool calls, they may forget which session they're in. Multiple
sessions look identical.

**Fix**: `uiai_browser_open({name: "wpuiai-settings-audit"})` and
the session shows up as "wpuiai-settings-audit" in diagnostics.

### 11. No structured way to save HTML / DOM

Agents can screenshot, but if a render bug is in the DOM, they can
only get a partial eval snippet. Useful additions:

- `uiai_browser_get_html` → return rendered HTML as a file
- `uiai_browser_get_console` → filtered by severity, returned as
  file

### 12. Login is manual

Every test session has to re-authenticate with username/password. The
test admin credentials must be reset periodically.

**Fix**: `uiai_browser_open` accepts a `auth_profile:
"wpuiai-test-admin"` that names a stored credential. CI/CD systems
pre-configure these once.

### 13. Tab clicks via ref need a snapshot dance

To click a Settings tab, agents must:

1. `uiai_browser_snapshot` to get the a11y tree
2. Find the ref for "Content" tab
3. `uiai_browser_click` with `selector:
   ".wpuiai-tab-link[data-tab='content']"`

3 calls to click 1 button. With named sessions + selector shortcuts
(#5), this collapses to 1 call.

### 14. Failed requests show URL but not body

`uiai_browser_diagnostics` for a 400 admin-ajax response shows the
URL but not the request body. Agents have to repeat the request via
eval to see what payload was sent.

**Fix**: `uiai_browser_diagnostics` includes request body for failed
network requests, with a `truncate: 1024` parameter to keep response
sizes bounded.

### 15. Resize takes a screenshot as a side effect

Calling `uiai_browser_resize` always includes a full screenshot in
the response. For agents that just want to change viewport size, the
screenshot is wasted tokens.

**Fix**: `uiai_browser_resize({return_screenshot: false})`.

### 16. No stateful scripting (scenario runner)

Complex E2E audits involve 100+ tool calls in a fixed sequence:
"open session → login → for each page: navigate, screenshot 375,
resize, screenshot 768, ...".

A "scenario runner" would accept a JSON array of actions and run them
in sequence, returning only the final result set:

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

After running E2E on Dashboard, the agent has identified
`.wpuiai-settings-card` as the main content selector. On a new
session, this knowledge is lost.

**Fix**: `uiai_browser_save_session({name: "wpuiai", selectors:
{main: ".wpuiai-settings-card", tabs: ".wpuiai-tab-link"}})` and
`uiai_browser_load_session({name: "wpuiai"})`.

### 18. No diffing vs baseline

A visual regression is a diff between today's screenshot and last
week's. The agent has to download both, run `compare`, interpret. A
built-in `uiai_browser_diff` against a reference image would be 10x
faster.

### 19. No domain presets

WordPress, Shopify, Stripe, GitHub — each has its own admin
structure. A "domain preset" that auto-configures selectors and auth
patterns would be a huge productivity win.

### 20. Diagnostics aren't per-session

A failed network request from session A bleeds into session B's
diagnostics buffer. This caused confusion. Fix: diagnostics should
be keyed by `session_id`.

### 21. No "uiai browser wait for text"

After filling a form, agents want to "wait for element X to contain
text Y". A `uiai_browser_expect({selector, text, timeout})` would
turn 4 tool calls (input, click, sleep, snapshot) into 1.

### 22. No double-click or right-click primitives

Some admin UIs (e.g. context menus, double-click edit) need these.

### 23. No key sequence primitive

`uiai_browser_press('Tab')` works, but
`uiai_browser_press_seq(['Tab','Tab','Tab','Enter'])` for form
submission via keyboard would be useful.

### 24. No way to mock `Date.now()` / `setTimeout` for time-traveling
tests

For testing scheduled actions, autopilot runs, cron jobs.

### 25. No "freeze UI" / "throttle animations" mode

Animations make screenshots non-deterministic. UIAI should
auto-pause CSS animations / Web Animations API during screenshots.

### 26. No "stable screenshot" primitive

`uiai_browser_screenshot({stable: true})` should wait for:

- `requestAnimationFrame` quiescent for 100ms
- No pending `setTimeout` / `fetch`
- No CSS animations running
- All images loaded
- All web fonts loaded

This eliminates the need for explicit `sleep` calls.

### 27. `fill` with select elements requires `select_option`, not `fill`

Inconsistent API surface for different form controls. A unified
`uiai_browser_set({selector, value})` with smart field-type detection
would be cleaner.

### 28. No "find all interactive elements" primitive

`uiai_browser_list_buttons` / `_list_inputs` / `_list_links` would
be a fast way to discover a page's affordances without evaluating
JS or full snapshot.

### 29. No "scroll to element" before screenshot

I had to do `eval(el.scrollIntoView())` before screenshot. A
`screenshot({selector, scroll_to: true})` would be one call.

### 30. Console-message capture is lossy

Only the last N messages are kept. Long-running operations can lose
context.

### 31. Network response bodies are not captured for success

Only failure responses show their body. For perf audits or "what's
the API returning?" questions, success-body capture would help.

### 32. No way to take a screenshot of a specific element only

`uiai_browser_screenshot({selector: ".uiai-settings-card"})` would
clip to a region, save tokens, and focus the audit.

### 33. No "compare against saved baseline" workflow

A `.uiai baseline/` directory convention where
`uiai_browser_screenshot` automatically diffs against a saved
baseline and returns the diff percentage.

### 34. No "generate accessibility report" primitive

axe-core integration would be a one-call way to surface
accessibility issues alongside visual audits.

### 35. No "test keyboard nav" primitive

`uiai_browser_keyboard_nav({start: "first_input", direction:
"tab"})` would test that tab order is sensible.

### 36. No "check focus trap" for modals

`uiai_browser_focus_trap({modal: ".uiai-modal"})` would test that
focus stays within the modal.

### 37. No "check color contrast" for text

`uiai_browser_contrast({selector: "h1", min_aa: 4.5})` would
audit WCAG compliance on rendered text.

### 38. No way to detect if a button is disabled before clicking

`uiai_browser_click(disabled_button)` silently fails. An
`is_enabled` / `is_visible` / `is_in_viewport` check before click
would prevent wasted attempts.

### 39. Click on the FIRST matching element only

`uiai_browser_click(".button")` always picks the first match. For
tables with multiple rows, agents have to fall back to `eval` with
`[index]`. Adding `index: 2` or `nth-of-type(n)` would be cleaner.

### 40. `fill` is not atomic

It types character-by-character, which:

- Triggers IME/IME-composition events on certain inputs
- Can be slow for 200+ char URLs
- Doesn't always survive `[type=password]` fields with autocomplete

A `set_value` (atomic) alternative would help.

### 41. No mouse-coordinate events

Click is element-based. Some interactions (drag-from-here-to-there
without a clear target, hovering near an element) need x/y events.

### 42. No file picker / native dialog API

Browser-level "save as" / "print" / native pickers have no API.
Agents have to skip the entire flow or workaround via mocks.

### 43. Viewport state sticky across navigations

After `uiai_browser_resize({width: 375})` succeeded, the next
`uiai_browser_navigate` occasionally reset the viewport (likely to
1280 default). Make `resize` state sticky per session, or include
`viewport` in `navigate` parameters.

### 44. No way to enumerate open sessions

No `uiai_browser_list_sessions` returning `[{id, url, viewport,
last_used}]` so agents can pick the right one.

### 45. `uiai_browser_close` is not always effective

After a session entered a hung state, `uiai_browser_close` returned
"This operation was aborted" instead of confirming. A `force: true`
parameter or async close would help.

### 46. Snapshot refs (`@e1`) don't survive navigation

The `uiai_browser_snapshot` returns refs valid for that single
render. After a redirect, navigation, or SPA route change, refs are
invalid. Fix: re-snapshot implicitly on click failure when a ref is
stale.

### 47. `click` fails silently on hidden elements

When I tried to click a button visually obscured (e.g. behind the
DevConsole panel at the bottom), the tool returned "operation was
aborted" without telling me whether the element existed, was
hidden, or was outside the viewport. Adding a `reason` field to
the error would help agents recover.

### 48. ARIA testing primitives missing

axe-core integration would let agents run
`uiai_browser_a11y_check` to surface accessibility violations
alongside the visual audit. Current workflow is `eval` to inject axe
+ read violations manually.

---

## Focusa ↔ UIAI integration (deepest agent DX wins)

### 49. Every UIAI screenshot needs manual translation into Focusa
evidence

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

### 50. UIAI sessions are not first-class Focusa objects

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

### 51. No bi-directional workpoint → browser context

When Focusa has an active workpoint with `current_action:
"submit_form_for_tab=ai&setting=max_iterations"`, UIAI has no way
to know this. The agent must manually translate workpoint state
into UIAI tool calls.

**Fix**: Implement `focusa_to_uiai_bridge` that, given a workpoint
current_action, generates the right sequence of UIAI tool calls.

### 52. Diagnostics are anonymous to Focusa

When UIAI reports "1 failed network request" in diagnostics, Focusa
has no way to correlate it to the action that caused it. The
diagnostic event has no `cause_action_id`, no `workpoint_id`, no
`active_object_ref`.

**Fix**: When UIAI emits a diagnostic event, it should be tagged
with the action that produced it. `focusa_browser_diagnostics_intake`
should auto-link each event to the right Focusa evidence/workpoint.

### 53. `focusa_tool_doctor` doesn't know about UIAI

When UIAI hangs or returns "operation was aborted", the agent
should be able to ask `focusa_tool_doctor("uiai_browser_click
hung after Start button")` and get back a structured recovery plan.

**Fix**: `focusa_tool_doctor` should accept a `tool_family:
"uiai"` parameter and return diagnosis, root_cause, and
recovery_actions.

### 54. No shared auth model between Focusa and UIAI

UIAI has no concept of `auth_profile`. Every UIAI session requires
the agent to manually re-authenticate with username/password.
Focusa has a `continuity_id` and knows the agent's identity, but
the browser never inherits that.

**Fix**: Introduce auth profiles that both systems share:

```json
{
  "uiai_browser_open": {
    "auth_profile": "wpuiai-test-admin",
    "url": "https://wpuiai.com/wp-admin/"
  }
}
```

Focusa stores the profile under `focusa://auth/wpuiai-test-admin` and
passes credentials automatically.

### 55. Evidence capture is one-way

I can `focusa_evidence_capture(screenshot, target_ref)` but I
can't `focusa_evidence_retrieve(target_ref, type='screenshot')`
and have it open a UIAI session to re-shoot for comparison.

**Fix**: Add a `uiai_browser_recapture({evidence_ref: "..."})` that
finds the evidence, parses out the URL/viewport/selector context,
opens a session, navigates, and re-screenshots.

### 56. No "what changed" between evidence points

When I do `focusa_evidence_capture` twice, Focusa has both but no
diff. A `focusa_evidence_diff` would do this natively.

### 57. No way to scope UIAI sessions to Focusa workpoints

When a workpoint is the current Focusa context, UIAI should know
which auth profile, URL, evidence_handle, and lifecycle to use.

**Fix**: `uiai_browser_open({workpoint: "019eae3c-..."})` would
pre-fill all this.

### 58. No way to report UIAI state into Focusa trajectory

When UIAI does a long action and the workpoint is "verify
critique-page-375px", the trajectory should know UIAI started the
action, captured screenshot evidence, and reported OK/FAIL.

**Fix**: `uiai_action_log` should auto-feed `focusa_trajectory_view`.

### 59. No "session memory" between Focusa sessions

When a Focusa continuity resumes (after compaction or model swap),
UIAI sessions opened by the previous agent are forgotten.

**Fix**: When `focusa_workpoint_resume` is called, also resume the
UIAI session that was last active. Store `last_uiai_session_id` in
the workpoint packet.

### 60. No "uiai session as Focusa context" pattern

Focusa has a powerful "context" abstraction (workpoint, current
action, evidence, predictions). UIAI has session state. They're
disjoint. A unified mental model would be:

```
Workpoint
├── trajectory (waypoints, workpoints)
├── active_object (current_url)
├── evidence (uiai screenshots + uiai DOM snapshots)
├── state (uiai session_id, viewport, auth_profile, localStorage)
└── next_action (uiai tool sequence)
```

---

## Session & state (lifecycle, persistence, env control)

### 61. No "session resume" after disconnect

When a session got killed (long-running op, server restart, etc.),
all in-flight context (cookies, localStorage, scroll position) was
lost. A way to persist and resume a session's `storage_state`
would prevent re-doing the login every time.

### 62. No way to inject cookies / localStorage for state

For testing "logged in as plan X" or "wpuiai_dev=true", I have to
run `eval` to seed localStorage. A `uiai_browser_state({localStorage:
{key: value}, cookies: [...]})` would standardize this.

### 63. No custom user-agent string

For mobile-emulated tests (UA-switching), the viewport resize
alone isn't enough. The page may serve different HTML for desktop
vs mobile UA.

### 64. No `prefers-color-scheme: dark` media emulation

Most modern UIs have a dark mode. There's no way to test it without
a real OS theme change. `uiai_browser_emulate_media({colorScheme:
"dark"})` would solve this.

### 65. No `prefers-reduced-motion` emulation

Same problem for the reduced-motion accessibility setting.

### 66. No way to inject response interceptors (security/auditability)

For testing "what if the API returns 503?" I have to break the
backend. Playwright's `route.fulfill()` would be a clean win.

**Constraint**: The intercept primitive must record every stub
applied to a `focusa_evidence` record so the test is auditable.
The workpoint's `not_done_if` should include "intercept was used".

### 67. No CPU / network throttling (per-session flag)

Cannot simulate slow connections or low-end devices. The
`uiai_browser_throttle` primitive would unlock perf-budget testing.

**Constraint**: Throttle must be a per-session flag, not a global
toggle. Clear visual indication that the session is in perf-mode.
Otherwise, throttled interactions may time out and be misreported as
hangs.

### 68. No way to disable JS / CSS

For testing no-JS fallback, or for performance isolation.

### 69. No cache-bypass header injection

Forcing a "fresh load" requires cache-busting query params on the
URL. A `--no-cache` flag or `cache-control: no-cache` injection
would standardize this.

### 70. No way to test "what happens if I refresh?"

For SPA route testing, hitting "refresh" needs a separate
`uiai_browser_reload` primitive. Currently I have to `navigate`
to the same URL which doesn't really test the back/forward cache.

### 71. No way to test browser back/forward

`uiai_browser_back` and `uiai_browser_forward` exist but
diagnostics aren't captured during those transitions.

---

## Diagnostics & observability

### 72. Console messages don't show source maps

Stack traces from bundled JS show `/app.js:1:1` instead of
`webpack://src/utils.ts:42:7`. Makes debugging 10x harder.

### 73. Network response bodies are not captured for success

(Already covered as #31 — duplicated here for completeness of
diagnostics section.)

### 74. Failed network requests are mixed with successful ones in
timing data

Hard to reason about TTFB for failures.

### 75. No structured "what changed" between two evidence points

A `focusa_evidence_diff` would do this natively (also #56).

### 76. No way to diff two scenarios

`uiai_scenario_diff({a, b})` would surface a structured diff (which
elements changed, which events fired, which network calls happened).

### 77. No console-message color coding

Error lines look identical to info lines in the diagnostic output.
Severity-based color or icon would let agents scan faster.

### 78. No focus-state reporting

After clicking an input, the snapshot doesn't tell me whether it
received focus. A `uiai_browser_focused()` returning `{tag, id,
name}` would help a11y audits.

### 79. No ARIA-landmark dump

The "no main landmark" or "no skip link" check requires a separate
axe-core invocation. A `uiai_browser_landmarks` would be one call.

### 80. Console-message capture during navigation is missed

Only after-the-fact. Errors that flash during the in-flight
`navigate` are sometimes missed because the diagnostics buffer
wasn't subscribed yet. Auto-subscribing on `open` would fix this.

---

## Reliability & errors

### 81. Session pool is bounded at 2

`max_pages: 2` is a hard limit. Agents with parallel investigations
need to wait. A queue with `priority` parameter would help.

### 82. "This operation was aborted" is the only error

There's no 504 vs 502 vs timeout distinction. Agents retry blindly.

**Fix**: Richer error envelopes with `code` (enum: `timeout`,
`network`, `abort`, `oom`, `server_error`, `client_error`),
`retryable` (bool), `last_tool`, `target_selector`.

### 83. The `127.0.0.1:7456` only works from one machine

If the agent's tool call goes through a different host (e.g. a CI
runner), it fails. The skill doesn't document this constraint.

### 84. No way to know the engine's health before opening a session

`uiai_browser_health` exists, but it returns capacity stats, not
readiness (e.g. "is the engine process running?"). Agents open
sessions blind.

### 85. No "session memory" between continuities

(Already covered as #59 — see Focusa integration section.)

---

## Aesthetics & consistency

### 86. The `format: jpeg` default is silent

When `uiai_browser_screenshot` is called, the response includes
`"format": "jpeg"` but not "this is base64-encoded". A new agent
has to read the docs to find out. (Note: this is a sub-fix of #1;
remove if #1 lands.)

### 87. Tool names are inconsistent

`uiai_browser_screenshot` (verb), `uiai_browser_diagnostics`
(noun), `uiai_browser_open` (verb), `uiai_browser_health` (noun).
Pick one convention.

### 88. Parameter naming inconsistent

`viewport: [375, 800]` vs `width: 375, height: 800` (used
elsewhere) vs `at_size: 375,800`. Pick one.

### 89. The "screenshot" tool's success response is just metadata

A new agent would expect the screenshot bytes. Returning `width,
height, format, size, url, title, duration_ms` looks like an error. A
first-line note would help: "screenshot captured; use jq to extract".
(Resolved by #1.)

### 90. No `--help` parameter to see tool schema

Agents have to read README or skill docs to learn parameters.
Inline help would speed onboarding.

### 91. The skill at `/root/.pi/skills/wpuiai-workflow` doesn't have a
"common UIAI patterns" section

The skill is named `wpuiai-workflow` not `uiai-engine` —
confusing. It also lacks a "UIAI cookbook" subsection for common
E2E patterns (responsive audit, form test, navigation tree, etc.).

### 92. No "starter template" for E2E audit scenarios

A canonical "responsive visual audit" JSON template with
placeholders for URL, selectors, viewports would let new agents
ramp up immediately.

### 93. No replay/redo for failed interactions

When `uiai_browser_click` fails, the agent has to manually retry.
A `uiai_browser_retry` with exponential backoff and selector
relaxation (e.g. drop `:nth-child(2)`) would be a huge win.

### 94. No "lint" for the test session itself

If an agent's E2E pattern has 5 redundant tool calls (e.g. "check
state, check state again, check state again, click"), a linter
would flag it.

### 95. No way to mark a screenshot as "golden" / "baseline"

Once a designer approves a screenshot, the agent should be able to
mark it as the baseline for future diffs.
`uiai_browser_save_golden({evidence_ref: "..."})` would do this.

### 96. No way to record a video

For visual E2E demos and walkthroughs, a `.webm` of the session
would be invaluable.

### 97. No way to export a session as a replayable script

A UIAI audit should be able to:

```bash
uiai audit run --url=... --viewports=375,768,1024,1440 \
  --output=audit-2026-06-09.json
uiai audit replay --input=audit-2026-06-09.json
```

So the audit is reproducible. Today, the audit is one-shot and not
replayable.

---

## North-star vision (long-term redesigns)

### 98. UIAI as a "do this on the web" primitive, not a browser
controller

The biggest shift would be moving from "control a Chromium
browser" to "do this on the web". Higher-level primitives:

- `uiai_do({intent: "log in to wpuiai.com", then: "click Settings",
  then: "fill license key"})`
- `uiai_scenario({actions: [...]})` with semantic intent per action
- `uiai_extract({url, query: "what's the user's subscription
  plan?"})`

Position intent-primitives as the "default" for new agents. Keep
atomic primitives as "advanced" for power users and edge cases. A
"level" parameter on session-open could choose: `level: "intent"`
(high-level) vs `level: "primitive"` (low-level).

### 99. Focusa-aware UIAI: the UIAI session IS the workpoint's web
context (6-month redesign)

Today: Workpoint is abstract, UIAI is a separate tool.
Tomorrow: A UIAI session is the concrete realization of a
workpoint's "web context". Workpoint advances → browser
navigates. Browser captures evidence → workpoint gains evidence.
Browser hits an error → workpoint knows to backtrack.

**Note**: This obsoletes #50 (sessions as first-class objects) once
implemented. Ship #50 first; plan #99.

### 100. UIAI should learn from past audits (with validity checks)

The same E2E audit (11 pages, 4 viewports, 5 input tests) shouldn't
be re-discovered every time. A `uiai_audit_history` that suggests
"you ran this scenario yesterday; replay?" would be 10x faster
than re-deriving the action list.

**Constraint**: When suggesting replay, always include a validity
check: re-verify the underlying state matches the original audit
(e.g. compare to a state-hash from the previous run). If
different, surface "stale" and prompt a fresh audit. Otherwise,
false-positive regressions will slip through.

### 101. Per-tool surface contract tests

Each UIAI tool should have a small self-test that runs on engine
startup. If `uiai_browser_diagnostics` is broken (e.g. `level`
filter ignored), an admin should see a warning, not just a
confused agent.

### 102. A "UIAI playground" in the engine repo

A static HTML page in `uiai-engine/playground/` with all primitives
exercised in a single page, useful for both human onboarding and
CI smoke tests. Currently agents have to set up a real WordPress
admin to test UIAI workflows.

---

## Workflow cookbook (quick recipes for common audit patterns)

These concrete patterns emerged from the WPUIAI E2E audit and can be
shipped as the `UIAI_FOR_AGENTS_QUICKSTART` addendum.

### Pattern A: Responsive visual regression audit (~15 calls)

```python
session = uiai_browser_open({auth_profile: "wpuiai-test-admin"})
for page in ["dashboard", "mimic-runs", "intakes", "copilot", "critique", "settings&tab=ai", "settings&tab=development"]:
    for viewport in [375, 768, 1024, 1440]:
        uiai_browser_screenshot({viewport, output: "file"})
        uiai_browser_evidence_diff({golden: page + "-" + viewport + ".png"})
uiai_browser_close()
```

### Pattern B: Form input + submit + verify (~3 calls)

```python
uiai_browser_navigate({url: "...&tab=ai"})
uiai_browser_fill({selector: "[name=uiai_autopilot_max_iterations]", value: "5"})
uiai_browser_click({selector: "input[name=submit]"})
uiai_browser_expect({selector: ".notice", text: "Saved"})
uiai_browser_screenshot({output: "file"})
```

### Pattern C: Filter only network failures (~1KB response)

```python
uiai_browser_diagnostics({
    level: "error",
    category: "network",
    failed_only: true,
    format: "summary",
    since_seq: last_seq
})
```

### Pattern D: Set up auth state for a complex scenario

```python
uiai_browser_state({
    localStorage: {wpuiai_admin: "1", plan: "enterprise"},
    cookies: {wpuiai_session: "..."},
    cache: "bypass"
})
```

### Pattern E: Re-shoot from prior evidence

```python
# UIAI looks up the prior evidence, parses URL/viewport/selector,
# opens a session, navigates, takes the same shot, returns diff.
result = uiai_browser_recapture({evidence_ref: "uiai:session=X:8"})
```

### Pattern F: Crash recovery

```python
try:
    uiai_browser_click({selector: ".uiai-autopilot-start"})
except TimeoutError:
    # auto-resume picks up cookies, auth, URL, viewport
    uiai_browser_recover()
    uiai_browser_navigate({url: "..."})
```

---

## Final tally

- **102 unique feedback items** (down from 120 after conflict
  resolution)
- **Hard conflicts resolved**: 5 (A-E in the conflicts doc)
- **Philosophical overlaps**: 3 (#50 vs #99, #52 vs #20, #49 vs
  agent control)
- **Subtle contradictions**: 5 (sticky viewport, state/cache/baseline,
  intercept auditability, throttle per-session, replay validity)
- **Big-picture tension**: 1 (intent vs primitive philosophy)

### Top 5 must-ship (cuts audit from 200 calls to ~15)

1. **Direct file output for screenshots** (output: file|url|json)
2. **Multi-viewport batch screenshot** (one-call responsive)
3. **Real server-side diagnostic filtering** (level/category/format)
4. **Auto-wait on navigate** (load/networkidle)
5. **Text-based selectors** (text=, role=)

### Top 5 Focusa ↔ UIAI integration (deepest agent DX)

1. **`uiai_browser_capture_evidence`** — auto-route to Focusa evidence
2. **UIAI sessions as first-class Focusa objects** (defer #99 obsolescence)
3. **`focusa_tool_doctor` understands UIAI failures**
4. **Shared `auth_profile` between Focusa and UIAI**
5. **`focusa_to_uiai_bridge`** — generate UIAI actions from workpoint

### Top 5 Reliability & errors

1. **Async tool calls** (return request_id, poll status)
2. **Richer error envelopes** (code/retryable/target_selector)
3. **Crash auto-recover** (resume cookies/URL, not just wait)
4. **Engine readiness check** (before opening session)
5. **Per-tool contract tests** (surface broken filters at startup)

Implementing these 15 items would cut a typical 11-page responsive
visual regression audit from ~200 calls + 500KB + 30min down to
~15 calls + 5KB + 3min. That is the north star for the engine.

### Pre-requisite: doc updates

Every code change must update:

- `uiai-engine/README.md`
- `uiai-engine/docs/AGENT_UX_COOKBOOK.md`
- `uiai-engine/docs/UIAI_FOR_AGENTS_QUICKSTART.md`
- `uiai-engine/docs/SESSION_API.md`
- `/root/.pi/skills/wpuiai-workflow/SKILL.md`

Without coordinated doc updates, the recommendations make the engine
*worse* for new agents, because they'll have stale docs and try
old patterns.

See also: `UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md` for
the full conflict analysis.
