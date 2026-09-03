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
- Bundled FPV asset build via `make fpv-assets`, served from `web/fpv/dist/` under `/m/assets/`.
- Optional FPV visual baseline comparison/update via `make fpv-visual-smoke` and `make fpv-visual-baselines`.
- Mobile bottom-sheet-style inspector, 44px touch targets, and swipe tab switching.
- Action timeline rows link to transient viewport markers, with graceful no-coordinate markers.

Remaining gaps below require explicit beads. Gaps marked implemented here remain listed for traceability.

## Gap 1 — WebRTC/CDP screencast transport

Current MJPEG transport is simpler but bandwidth-heavy. Add a lower-latency transport using Chrome DevTools Protocol `Page.startScreencast`, WebSocket frame delivery, or WebRTC.

Acceptance:

- New transport endpoint or negotiated mode.
- Measured latency/fps shown in UI.
- Fallback to MJPEG/polling remains.
- Visual smoke passes.

## Gap 2 — Bundled frontend build

Status: implemented. `make fpv-assets` copies the dependency-free FPV CSS/JS into `web/fpv/dist/`, writes an asset manifest, and fails if CDN/runtime dependencies are introduced. `/m/assets/{file}` serves generated dist files first with source fallback.

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

Status: implemented. Click/fill/key/point actions create transient viewport markers; timeline rows carry marker ids and hover/click highlights the related marker. Selector/key actions without coordinates fall back to a no-coordinate marker.

Acceptance:

- Click/fill/key events create transient visual markers.
- Timeline event hover/click highlights related marker.
- Markers degrade gracefully when no coordinates exist.

## Gap 5 — Share security controls

Status: source candidate; not installed or independently accepted. Session creation is private by default. Explicit read-only share creation uses a 192-bit URL-safe capability, exact session/origin binding, a maximum 60-minute TTL, atomic bounded-view consumption, immediate session-close revocation, sensitive-origin denial, and token-redacted status/audit payloads. Public control issuance fails closed until a separate governed-confirmation contract exists.

Acceptance:

- Installed `browser_open` returns no share or capability.
- Auth, privacy, payment, and health origins cannot mint public shares.
- Read-only and control capabilities are separate; control issuance remains blocked without governed confirmation.
- TTL and max-view enforcement are bounded and concurrency-safe.
- Closing a session immediately revokes all derived capabilities.
- Tokens are absent from diagnostics, audit events, revocation receipts, and notifications.
- Existing legacy, unbounded, or control-enabled registry entries are rejected on load and lookup.
- Real public/mobile browser proof remains required.

## Gap 6 — Persistent share registry

Status: source candidate; not installed or independently accepted. The registry persists only current-policy, unexpired, bounded, read-only entries through an atomic `0600` temporary-file rename. Creation, view consumption, explicit revocation, and session-close revocation report persistence failure instead of claiming durable success.

Acceptance:

- Restart preserves only current-policy unexpired shares.
- Legacy, unbounded, control-enabled, expired, and revoked entries never load.
- Registry-update failure injection proves fail-closed behavior.
- Registry and logs expose no capability values outside the protected storage/explicit share response boundary.

## Gap 7 — Live Focusa integration

Status: implemented. FPV status adapts session `focusa_scope` into compact Workpoint/evidence/prediction/trajectory-linked fields, optionally hydrates live Workpoint/Trajectory summaries through `FOCUSA_DAEMON_URL`, and renders graceful degraded states when scope or daemon access is unavailable.

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

Status: implemented. Mobile FPV uses a bottom-sheet-style inspector, 44px touch targets, horizontal swipe tab switching, horizontal scroll snapping for dense controls, and verified 375px visual smoke.

Acceptance:

- Bottom-sheet controls or equivalent.
- Swipe/tab gesture support if feasible.
- Touch targets meet mobile sizing.
- 375px visual smoke remains green.

## Gap 12 — Visual baseline comparison

Status: implemented. `scripts/smoke-fpv-visual-breakpoints.sh` supports `BASELINE_DIR`, `UPDATE_BASELINE=1`, `DIFF_THRESHOLD`, ImageMagick RMSE comparison, and failure output with baseline/current/diff artifact paths.

Acceptance:

- Baseline capture/update command.
- Pixel or perceptual diff threshold.
- CI/local failure report with artifact paths.
