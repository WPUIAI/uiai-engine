# FPV Remaining Gaps Spec

Last updated: 2026-06-13

This spec tracks remaining FPV product/architecture upgrades after the MJPEG streaming + Mac-style cockpit pass.

## Status summary

Implemented baseline:

- Cloudflare `/m/*` public hostname.
- Mac-style FPV assets in `web/fpv/`.
- MJPEG stream with screenshot polling fallback.
- Quality modes.
- Audited note/click/fill/key controls.
- Dynamic repo/Focusa context summary.
- Visual breakpoint smoke at 375 / 768 / 1024 / 1440.

Remaining gaps below require explicit beads.

## Gap 1 — WebRTC/CDP screencast transport

Current MJPEG transport is simpler but bandwidth-heavy. Add a lower-latency transport using Chrome DevTools Protocol `Page.startScreencast`, WebSocket frame delivery, or WebRTC.

Acceptance:

- New transport endpoint or negotiated mode.
- Measured latency/fps shown in UI.
- Fallback to MJPEG/polling remains.
- Visual smoke passes.

## Gap 2 — Bundled frontend build

Current assets are plain CSS/JS plus CDN Lucide fallback. Introduce a build pipeline only if it keeps deployment simple.

Acceptance:

- Bundled icons; no CDN runtime dependency.
- Build command documented.
- Generated assets served under `/m/assets/`.
- Go route remains clean.

## Gap 3 — Selector picking overlay

Operator should click/tap the mirrored viewport to suggest/select a target instead of hand-writing selectors.

Acceptance:

- Overlay mode toggles on/off.
- Click coordinate maps to a selector candidate or element context.
- Suggested selector can be used by click/fill controls.
- Audit event records target.

## Gap 4 — Visual action overlay

Actions should appear over the browser viewport and be linked to timeline events.

Acceptance:

- Click/fill/key events create transient visual markers.
- Timeline event hover/click highlights related marker.
- Markers degrade gracefully when no coordinates exist.

## Gap 5 — Share security controls

Public bearer URLs need stronger lifecycle controls.

Acceptance:

- Revoke share endpoint.
- Optional one-time share flag or max-view count.
- Short TTL presets.
- UI shows revoked/expired state clearly.

## Gap 6 — Persistent share registry

Shares are currently in-memory. Restart loses active links.

Acceptance:

- Durable share registry with expiry cleanup.
- Restart preserves unexpired shares when safe.
- Registry excludes secrets from logs.

## Gap 7 — Live Focusa integration

Current Focusa panel uses server-generated summary. Integrate real Focusa surfaces.

Acceptance:

- Workpoint/evidence/prediction/trajectory adapter.
- Compact UI only; raw payload hidden.
- Graceful degraded state when Focusa unavailable.

## Gap 8 — Live repo context depth

Repo panel needs actual active files/diffs/bead status/deploy info.

Acceptance:

- Branch/head/dirty state.
- Active files and bounded tree from actual repo state.
- Recent commits and bead links.
- Dirty diff summary without huge payloads.

## Gap 9 — Performance hardening

Public streaming needs resource controls.

Acceptance:

- Per-share/viewer frame throttle.
- Max viewers per share.
- Adaptive quality under pressure.
- UI pressure indicator.

## Gap 10 — Accessibility pass

UI must be keyboard and screen-reader usable.

Acceptance:

- Keyboard tab navigation.
- ARIA labels for tabs/panels/controls.
- Focus states visible.
- Reduced-motion verified.

## Gap 11 — Mobile interaction design

Responsive layout exists, but mobile controls need native-feeling interaction.

Acceptance:

- Bottom-sheet controls or equivalent.
- Swipe/tab gesture support if feasible.
- Touch targets meet mobile sizing.
- 375px visual smoke remains green.

## Gap 12 — Visual baseline comparison

Smoke captures screenshots but does not compare to approved baselines.

Acceptance:

- Baseline capture/update command.
- Pixel or perceptual diff threshold.
- CI/local failure report with artifact paths.
