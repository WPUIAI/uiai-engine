# FPV Browser Style Guide

Last updated: 2026-06-12

## Purpose

Standardize the public FPV browser viewer at `fpv.wpuiai.com/m/{token}` as a premium live-operations browser: browser-first, realtime-feeling, human-readable, and beautiful from phone to desktop.

The FPV viewer is not just a screenshot mirror. It is an operator cockpit that should explain:

1. what the agent/browser is seeing,
2. what the session is doing,
3. what changed recently,
4. what repo/project context matters,
5. what Focusa/workpoint context matters,
6. what needs attention now.

## Source of truth

Current implementation source: `internal/routes/fpv.go` inline `fpvPageTemplate`.

Implemented asset split:

- `internal/routes/fpv.go` — route/data contract and asset serving only.
- `web/fpv/index.html` — semantic FPV shell.
- `web/fpv/fpv.css` — Mac-beauty tokens, layout, components, motion.
- `web/fpv/fpv.js` — polling, state normalization, controls, context rendering.
- `docs/FPV_BROWSER_STYLE_GUIDE.md` — design contract.

## Hard visual constraints

- No colored left-border accent treatments. Use full-card borders, glyphs, chips, and background motion instead.
- Container `border-radius` must not exceed `5px`. Small signal cells may use `2px`; avoid pill-shaped containers unless explicitly approved.
- Data animations must be Nullframe-inspired signal animations, not generic decorative motion. Required primitives: seismograph bars, glyph/cell-slam grids, health matrix, sweep tags, and ticker/status cadence.


## Mac beauty standard

FPV should feel like a native, premium macOS operations cockpit:

- SF-style typography stack: `-apple-system`, `BlinkMacSystemFont`, `SF Pro Display`, `SF Pro Text`, fallback sans.
- Crisp glass surfaces, restrained gradients, no loud decorative borders.
- 5px maximum container radius, per operator constraint.
- Clean toolbar/sidebar/detail split.
- Lucide icon library when available, with local glyph fallback so icons never disappear.
- Route code remains clean; UI complexity belongs in `web/fpv` assets, not `internal/routes/fpv.go`.


## Design principles

| Principle | Meaning |
|---|---|
| Browser first | The left side is always a believable browser chrome containing the live viewport. |
| Data, not dumps | Never show raw JSON by default; transform it into labeled cards, timelines, and status chips. |
| Realtime calm | Use motion to communicate liveness, not distract. |
| Context-rich | Show session, repo, Focusa, diagnostics, and history in clean groups. |
| Progressive density | Phone gets essential cards; desktop gets full telemetry. |
| Fluid between breakpoints | Use `clamp()`, `minmax()`, container-like grids, and responsive card wrapping. |

## Responsive standard

Design must be explicitly checked at these viewport widths:

| Width | Layout target |
|---:|---|
| 375 | Single column. Browser first, then compact status cards, then stream. Hide noncritical labels. |
| 768 | Single column or soft 60/40 sections. Browser remains prominent. Data cards become 2-up where possible. |
| 1024 | Two-column layout begins. Browser 60–65%, data rail 35–40%. |
| 1440 | Full desktop cockpit. Browser 2/3, realtime data rail 1/3. More timeline/context modules visible. |

Use fluid CSS instead of hard jumps:

```css
:root {
  --space-1: clamp(0.375rem, 0.25rem + 0.4vw, 0.625rem);
  --space-2: clamp(0.625rem, 0.45rem + 0.6vw, 1rem);
  --space-3: clamp(0.875rem, 0.65rem + 0.9vw, 1.5rem);
  --radius-lg: clamp(1rem, 0.8rem + 0.8vw, 1.5rem);
  --text-xs: clamp(0.68rem, 0.64rem + 0.16vw, 0.75rem);
  --text-sm: clamp(0.78rem, 0.72rem + 0.2vw, 0.875rem);
  --text-md: clamp(0.95rem, 0.86rem + 0.28vw, 1.125rem);
  --text-xl: clamp(1.45rem, 1.1rem + 1.2vw, 2.25rem);
}

main {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: minmax(0, 1fr);
}

@media (min-width: 1024px) {
  main {
    grid-template-columns: minmax(0, 2fr) minmax(22rem, 1fr);
  }
}
```

## Layout primitives

| Primitive | Role |
|---|---|
| `header` | Fixed/sticky app bar with product name, session id, and live status. |
| `main` | Responsive shell. Stacked on mobile, two-column on desktop. |
| `.stage` | Browser area wrapper. |
| `.browser` | Premium browser chrome card. |
| `.chrome` | Browser toolbar: traffic lights, URL pill, page metadata. |
| `.viewport` | Live screenshot/image area. Must preserve aspect ratio and avoid cropping important page content. |
| `.rail` | Realtime data rail. Scrollable on desktop; stacked content on mobile. |
| `.section` | Labeled data group. |
| `.card` | Atomic data card. |
| `.stream` | Timeline/activity stream. |

## Design tokens

| Token | Role |
|---|---|
| `--bg0`, `--bg1` | Deep navy/black gradient base. |
| `--panel`, `--panel2`, `--card` | Glass panels and nested telemetry cards. |
| `--text`, `--muted` | Primary and secondary text. |
| `--line`, `--line2` | Hairline borders and elevated edge strokes. |
| `--blue`, `--violet` | Primary gradient / CTA / focus color. |
| `--green` | Healthy/live status. |
| `--amber` | Warning/degraded status. |
| `--red` | Error/stopped status. |
| `--shadow`, `--radius` | Premium elevation and rounding. |
| `--ease-out`, `--ease-spring` | Motion curves. |

## Data labeling taxonomy

Every datum needs a plain-English label, a short value, and optional helper text. Avoid internal names unless useful for copy/paste.

### Label rules

- Prefer human labels: `Current page`, not `url`; `Frame rate`, not `fpsNum`.
- Keep labels 1–3 words.
- Values should be scannable; long values wrap in monospace only when they are identifiers.
- Use helper text for why it matters.
- Use status words consistently: `Live`, `Read-only`, `Control enabled`, `Degraded`, `Stopped`, `Expired`.

### Required card shape

```html
<div class="card metric-card" data-state="ok">
  <div class="label">Frame rate</div>
  <div class="value">3.8 FPS</div>
  <div class="hint">Screenshot polling cadence</div>
</div>
```

### Core session labels

| Data source | Label | Example value | Helper |
|---|---|---|---|
| share mode | Access mode | Control enabled | Whether operator actions are allowed. |
| screenshot poll | Frame rate | 3.8 FPS | Client-side image refresh cadence. |
| session id | Browser session | `Zht_aitS` | Copy-safe UIAI session id. |
| share token | Share link | `vivid-signal-5ad2` | Human-friendly public token. |
| URL | Current page | `https://stripe.com/` | Page the agent/browser is viewing. |
| title | Page title | Stripe | Browser page title. |
| dimensions | Viewport | `1440 × 900` | Source browser viewport. |
| expiry | Share expires | `11:23 PM` | Token validity window. |
| views | Viewer opens | `3` | Number of times share page opened. |

### Diagnostics labels

| Data source | Label | State logic |
|---|---|---|
| console errors | Console errors | `0 = ok`, `>0 = warning`. |
| failed requests | Failed requests | `0 = ok`, `>0 = warning`. |
| HTTP 4xx | Client errors | `0 = ok`, `>0 = warning`. |
| HTTP 5xx | Server errors | `0 = ok`, `>0 = error`. |
| request count | Network requests | informational. |
| exceptions | JS exceptions | `0 = ok`, `>0 = error`. |


## Tabbed data rail standard

The realtime rail must not be one long scrolling dump. It is a professional tabbed cockpit.

Required tabs:

| Tab | Purpose |
|---|---|
| Overview | Live signal, access mode, current page, viewport, expiry, stream quality. |
| Health | Diagnostics metrics and Nullframe-style health matrix. |
| Repo | Project, branch/head, public host, active file tree. |
| Focusa | Objective, evidence, drift guard, compact cognitive context. |
| Timeline | Realtime frame/action/audit/status history. |
| Control | Safe operator actions and audited notes. |

Typography and spacing rules:

- Tabs use 11.5–12.5px bold text with glyph icons and 31–32px height.
- Cards use 13px base text, 15–17px values, and at least 112px mobile / 116px desktop vertical rhythm.
- Labels are uppercase, 11px, high weight, and never crowd values.
- Helper text uses 1.35 line-height.
- Header title/subtitle must truncate cleanly instead of wrapping awkwardly.
- At 1024px, the rail uses single-column cards for readability; 2-column card grids are reserved for wider desktop rails.
- Shimmer/sweep effects must be contained inside card bounds and must not create layout overflow.

## Required data panels

### 1. Browser panel

Always left/top. Contains:

- browser traffic lights,
- current URL pill,
- page title/meta,
- live screenshot viewport,
- subtle scanline/liveness overlay.

### 2. Live signal panel

Contains:

- Frame rate,
- Access mode,
- Stream quality,
- Last frame time,
- Share expiry.

### 3. Session panel

Contains:

- Browser session id,
- Share token,
- View count,
- Viewport dimensions,
- Current URL,
- Page title.

### 4. Diagnostics panel

Contains:

- Console errors,
- Failed requests,
- HTTP 4xx/5xx,
- JS exceptions,
- Request count.

### 5. Repo context panel

Purpose: orient the operator to the code/project behind the session.

Minimum fields:

| Label | Example | Source |
|---|---|---|
| Project | `uiai-engine` | Project identity / configured page context. |
| Root | `/home/wpuiai/uiai-engine` | Focusa/project scope. |
| Branch | `main` | Git context. |
| Head | `de8cb9b` | Git context. |
| Dirty state | `clean` / `modified` | Git status. |
| Recent commit | `feat: add fpv design guide...` | Git log. |

### 6. Repo tree panel

Purpose: show where current work lives without overwhelming the viewer.

Design:

- Collapsed tree by default.
- Show max 2–3 levels.
- Highlight active files.
- Use file-type chips.

Example:

```text
uiai-engine/
├─ internal/routes/
│  └─ fpv.go        active
├─ docs/
│  └─ FPV_BROWSER_STYLE_GUIDE.md active
└─ mcp/
```

### 7. History timeline panel

Purpose: make recent changes understandable.

Timeline event types:

| Type | Icon/color | Example |
|---|---|---|
| frame | blue | Browser frame refreshed. |
| action | violet | Operator note sent. |
| diagnostic | amber/red | 2 failed requests detected. |
| git | green | Commit `de8cb9b` pushed. |
| focusa | cyan | Evidence captured. |
| session | gray | Share opened / expired / closed. |

Timeline cards should show:

- timestamp,
- event type,
- short title,
- one-line details,
- optional copyable ref.

### 8. Focusa context panel

Purpose: show helpful cognitive state without dumping Focusa internals.

Recommended fields:

| Label | Meaning |
|---|---|
| Current objective | Human-readable active goal. |
| Workpoint | Short id + mission summary. |
| Next step | Immediate next action. |
| Evidence | Latest proof handles. |
| Prediction | Current bounded prediction. |
| Drift guard | What not to drift into. |

Rules:

- Never display raw huge Focusa payloads in the default UI.
- Show compact summaries and copyable handles.
- Prefer “Evidence captured: git:de8cb9b” over internal JSON.

## Repo/Focusa data contract

The FPV status route can be extended with optional context:

```json
{
  "context": {
    "project": {
      "name": "uiai-engine",
      "root": "/home/wpuiai/uiai-engine",
      "branch": "main",
      "head": "de8cb9b",
      "dirty": false
    },
    "tree": [
      { "path": "internal/routes/fpv.go", "kind": "go", "active": true },
      { "path": "docs/FPV_BROWSER_STYLE_GUIDE.md", "kind": "doc", "active": true }
    ],
    "history": [
      { "ts": "2026-06-13T05:59:00Z", "type": "git", "title": "Design guide pushed", "ref": "de8cb9b" }
    ],
    "focusa": {
      "objective": "Improve FPV browser design system",
      "workpoint": "019...",
      "next_step": "Implement context panels",
      "evidence": ["git:de8cb9b"],
      "drift_guard": "Keep FPV host path-gated to /m/*"
    }
  }
}
```

UI must tolerate missing context and render graceful empty states.

## Visual hierarchy

### Typography

- Product/title: bold, tight letter spacing.
- Section labels: uppercase micro-labels.
- Metric values: large, tabular, high contrast.
- Identifiers: monospace, subdued background.

Recommended fluid scale:

```css
--font-label: clamp(0.62rem, 0.58rem + 0.15vw, 0.72rem);
--font-body: clamp(0.82rem, 0.78rem + 0.18vw, 0.95rem);
--font-value: clamp(1.05rem, 0.92rem + 0.55vw, 1.55rem);
--font-hero: clamp(1.75rem, 1.35rem + 1.4vw, 3rem);
```

### Density

- Use 8–12px gaps at mobile.
- Use 14–18px gaps on desktop.
- Cards should never look cramped; if a value is long, wrap or collapse.

## Nullframe-inspired motion system

Reference: `https://project-nullframe.vercel.app/`.

Observed patterns to adapt:

1. **Pulse LEDs** — small live indicators pulse calmly; not a substitute for data animations.
2. **Card shimmer/sweep** — Nullframe-style `shine` strip on hover or fresh data, using `null-shimmer` left-to-right sweep.
3. **Bento telemetry cards** — dense but clean cards with uppercase labels, glyphs, and strong values.
4. **Activity stream cadence** — timestamped events create a sense of life via compact ticker/timeline rows.
5. **Seismograph/glyph feel** — required for realtime data: animated bar channels, glyph grids, and health matrices.
6. **RAF/FPS metrics** — show the live cadence transparently.
7. **Composite-safe animation** — animate opacity, transform, and background-position; avoid layout-heavy animation.

### Motion tokens

```css
--ease-out: cubic-bezier(.16, 1, .3, 1);
--ease-spring: cubic-bezier(.34, 1.56, .64, 1);
--pulse-slow: 2.4s;
--sweep-fast: 1.05s;
--stream-in: .42s;
```

### Required Nullframe data animations

| Animation | FPV use |
|---|---|
| Seismograph bars | Frame cadence / realtime browser signal. |
| Glyph grid with `cell-slam` | Live browser data energy / session heartbeat. |
| Health matrix with `streak-wave` | Diagnostics severity and network health. |
| Sweep tags / shine strip | Card hover and fresh data updates. |
| Ticker cadence | Compact live text stream for frame/control/status events. |

### Required keyframes

- `fpv-pulse` — status LEDs and live chips.
- `fpv-shimmer` — card/rail sweep on hover or update.
- `fpv-stream-in` — new activity rows slide/fade in.
- `fpv-scan` — subtle browser viewport scanline.
- `fpv-flow` — animated signal bar.
- `fpv-glow` — diagnostics warning glow.

### Motion rules

- Animate on data arrival, status changes, hover, and warning escalation.
- Do not animate large layout changes continuously.
- Disable repeated motion under `prefers-reduced-motion: reduce`.
- Show “stopped/expired” as a calm state; do not keep polling aggressively.


## Implemented interaction standard

Current FPV UI implements these cockpit interactions:

- Quality modes: Smooth, Balanced, Saver.
- Safe operator controls: audited note, selector click, selector fill, common keypresses.
- Dynamic context in `/m/{token}/status`: repo project/branch/head/dirty status, active tree, recent git history, compact Focusa context.
- Timeline filtering: All, Frames, Actions, Git, Focusa.
- Expired/stopped state: polling stops after unavailable status or repeated frame errors.

Rules:

- Click/fill/press controls must remain audited server-side.
- Repo and Focusa context must render graceful empty states if unavailable.
- Timeline entries must be short, filterable, and copy-friendly.
- Stream quality controls change client polling cadence only; they do not widen public API exposure.
- On mobile, inspector controls behave like a bottom sheet, touch targets stay at least 44px tall, and horizontal swipes switch tabs.

## Implemented product upgrades

- Primary transport: `/m/{token}/stream.mjpg` MJPEG stream.
- Fallback transport: `/m/{token}/screenshot.jpg` polling.
- Status contract advertises `transport.primary`, `stream_url`, `fallback_url`, and quality modes.
- Visual regression smoke: `scripts/smoke-fpv-visual-breakpoints.sh` captures 375 / 768 / 1024 / 1440 FPV views, checks route isolation, and compares optional baselines with RMSE threshold.
- Baseline capture/update command: `make fpv-visual-baselines` (or `UPDATE_BASELINE=1 scripts/smoke-fpv-visual-breakpoints.sh`). Failures print `baseline=`, `current=`, and `diff=` artifact paths.
- UI assets are split into `web/fpv/index.html`, `web/fpv/fpv.css`, and `web/fpv/fpv.js`; `make fpv-assets` generates `/m/assets/`-served files in `web/fpv/dist/` and rejects CDN/runtime dependencies.

## Implementation checklist

Before marking FPV UI work complete:

- [ ] 375px viewport reviewed.
- [ ] 768px viewport reviewed.
- [ ] 1024px viewport reviewed.
- [ ] 1440px viewport reviewed.
- [ ] Layout fluid between breakpoints using `clamp()`/`minmax()`.
- [ ] Browser panel remains primary at all widths.
- [ ] Data labels are human-readable.
- [ ] Repo context panel has graceful empty state.
- [ ] Repo tree panel has max-depth/collapse behavior.
- [ ] History timeline shows frame/action/diagnostic/git/focusa events.
- [ ] Focusa context is summarized, not raw JSON.
- [ ] Animation respects reduced-motion.
- [ ] `/api/*` remains unavailable on `fpv.wpuiai.com`.

## Next implementation targets

Remaining work is tracked in [FPV Remaining Gaps Spec](FPV_REMAINING_GAPS_SPEC.md) and corresponding beads.
