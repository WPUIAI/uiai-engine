# FPV Browser Style Guide

Last updated: 2026-06-12

## Purpose

Standardize the public FPV browser viewer at `fpv.wpuiai.com/m/{token}` so it feels like a premium live operations surface: browser-first, realtime, human-readable, and calm under load.

## Current CSS inventory

Source: `internal/routes/fpv.go` inline `fpvPageTemplate`.

### Layout primitives

- `header` — fixed top application bar.
- `main` — desktop two-column shell.
- `.stage` — left column wrapper, 2/3 desktop width.
- `.browser` — browser chrome card and screenshot viewport.
- `.chrome`, `.lights`, `.addr`, `.viewport` — visual browser frame.
- `.rail` — right realtime data rail, 1/3 desktop width.
- `.rail-head`, `.section`, `.grid`, `.card` — data hierarchy.
- `.stream`, `.stream-row` — realtime activity feed.

### Design tokens

| Token | Role |
|---|---|
| `--bg0`, `--bg1` | Deep navy/black gradient base. |
| `--panel`, `--panel2`, `--card` | Glass panels and nested telemetry cards. |
| `--text`, `--muted` | Primary and secondary text. |
| `--line`, `--line2` | Hairline borders and elevated edge strokes. |
| `--blue`, `--violet` | Primary brand gradient / CTA / focus color. |
| `--green`, `--amber`, `--red` | Healthy/warning/error status semantics. |
| `--shadow`, `--radius` | Premium elevation and rounding. |

### Components

#### Browser chrome

Left column must look like an actual browser, not a raw image:

- macOS traffic lights
- lock/status glyph
- current URL pill
- metadata chip
- contained screenshot viewport with `object-fit: contain`

#### Realtime data rail

Right column streams readable state:

- Live signal: FPS, mode, title, viewport, expiry, views
- Diagnostics: console errors, failed requests, HTTP 4xx/5xx, request count
- Operator action: audited note entry
- Activity stream: frame/control/audit/status events

#### Responsive behavior

- Desktop: `grid-template-columns: minmax(0, 2fr) minmax(380px, 1fr)`.
- Mobile/tablet: stack browser above rail, hide nonessential chrome metadata.

## Nullframe animation patterns to adapt

Reference: `https://project-nullframe.vercel.app/`.

Observed patterns:

1. **Live pulse LEDs** — small status dots pulse at a slow cadence.
2. **Card shimmer/sweep** — cards show a diagonal shine on hover or data update.
3. **Bento telemetry cards** — compact cards with uppercase micro-labels and large values.
4. **Activity stream cadence** — tiny timestamped events communicate liveness without noise.
5. **Seismograph/glyph feel** — data changes should feel like signals, not static tables.
6. **RAF/FPS metrics** — the UI explains its own realtime cadence.
7. **Composite-friendly motion** — use opacity/transform/filter, avoid layout-heavy animations.

## FPV motion system

### Motion tokens

```css
--ease-out: cubic-bezier(.16,1,.3,1);
--ease-spring: cubic-bezier(.34,1.56,.64,1);
--pulse-slow: 2.4s;
--sweep-fast: 1.05s;
--stream-in: .42s;
```

### Required keyframes

- `fpv-pulse` — status LEDs and live chips.
- `fpv-shimmer` — card/rail sweep on hover or update.
- `fpv-stream-in` — new activity rows slide/fade in.
- `fpv-scan` — subtle browser viewport scanline.
- `fpv-glow` — diagnostics warning glow for nonzero errors.

### Application rules

- Animate only when it clarifies live state.
- Keep screenshot polling separate from CSS animation; CSS should make the UI feel alive without hiding frame latency.
- Respect `prefers-reduced-motion` by disabling repeated animation.
- Use warning/error animation sparingly; never make error states decorative.

## Next implementation targets

1. Move inline CSS/JS to versioned static assets when route architecture supports it.
2. Add a small canvas/sparkline for frame cadence and diagnostics events.
3. Add a “stream quality” indicator based on frame success/error counters.
4. Add operator-selectable quality modes:
   - Smooth: 250ms poll
   - Balanced: 500ms poll
   - Saver: 1000ms poll
5. Add a visual event timeline for click/fill/press/actions.
