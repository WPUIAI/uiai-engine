# UIAI Cockpit Desktop Session Presentation and Focusa Menubar Handoff Specification

**Document number:** `UIAI-COCKPIT-004`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding amendment:** `UIAI-COCKPIT-003`  
**Status:** Proposed normative implementation amendment  
**Version:** 1.0  
**Date:** 2026-08-03  
**Machine-readable companion:** [`UIAI-COCKPIT-004-C01`](contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml)  
**Call-stack design:** `019fc617-9d91-70c2-8392-3c7488e457a7`

---

# 0. Authority and application order

This amendment applies after the unified Cockpit master and prior accepted numbered decisions and amendments:

```text
UIAI-COCKPIT-000
→ UIAI-COCKPIT-001
→ UIAI-COCKPIT-002 + UIAI-COCKPIT-002-C01
→ UIAI-COCKPIT-003
→ UIAI-COCKPIT-004 + UIAI-COCKPIT-004-C01
```

It makes the browser-runtime, desktop-presentation, and Focusa Menubar handoff boundaries explicit. It refines the detailed connection-plane requirements in `UIAI_OPERATOR_BROWSER_DESKTOP_SPEC_2026-06-19.md` §17 without weakening them.

Where older prose can be interpreted as allowing Cockpit to become a second browser authority, this amendment wins: Cockpit presents and controls UIAI Engine sessions; it does not fork them.

---

# 1. Decision

UIAI ships as one seamless desktop product with three cooperating authorities:

```text
UIAI Engine (native Go)
  = browser process, session, artifact, diagnostics, and execution authority

UIAI Cockpit (Tauri desktop shell)
  = desktop window, presentation, operator intent, approval, and control surface

Focusa Menubar + Focusa daemon
  = quick local entry surface plus canonical scoped cognition/pairing authority
```

When automation does not require a visible desktop, UIAI Engine runs the canonical session without opening a desktop window.

When an operator, policy, failure, takeover, or workflow requests desktop visibility, UIAI Engine MUST present that same session in Cockpit. It MUST NOT open installed Chrome/Brave, create an unrelated Tauri WebView navigation, or copy the session into a second browser runtime.

Cockpit and Focusa Menubar MUST support bidirectional OS-mediated handoff. Deep links select an opaque object or session; they never confer authority and never carry secrets.

---

# 2. Non-negotiable requirements

- **DSP-001 — One browser authority.** UIAI Engine owns every agent browser process, context, target, page, cookie jar, storage partition, diagnostic stream, and lifecycle state.
- **DSP-002 — Cockpit is the desktop shell.** All full desktop browser presentation opens in or focuses UIAI Cockpit.
- **DSP-003 — Same-session presentation.** A desktop view attaches to the existing UIAI session ID and does not renavigate or reconstruct the page in another runtime.
- **DSP-004 — No surprise runtime download.** Production builds MUST NOT download or unpack Chromium on first browser use.
- **DSP-005 — Packaged runtime.** A signed, checksummed, pinned CDP-compatible browser runtime ships with or is installed atomically alongside UIAI Engine.
- **DSP-006 — Explicit development fallback.** Rod-managed browser download is allowed only behind an explicit development configuration and MUST be false in production.
- **DSP-007 — No installed-browser default.** UIAI MUST NOT silently select a user's Chrome, Brave, Edge, profile, cookies, extensions, or stored credentials.
- **DSP-008 — Cockpit handoff.** UIAI Engine exposes a typed presentation request and receipt for launching/focusing Cockpit on a session.
- **DSP-009 — Single Cockpit instance.** Repeated requests route to the existing Cockpit process and window unless a future explicit multi-window policy applies.
- **DSP-010 — Bidirectional app handoff.** Menubar owns `focusa://`; Cockpit owns `cockpit://`.
- **DSP-011 — Opaque deep links.** URLs carry stable opaque refs only, never bearer tokens, pairing payloads except the separately governed pairing fallback, raw project paths, page credentials, or private page content.
- **DSP-012 — Truth-plane fallback.** Missing, stale, or conflicting handoff targets fall back to authenticated daemon/Engine reads and never invent state.
- **DSP-013 — Distinct app identities.** Menubar and Cockpit retain separate bundle IDs, device IDs, Keychain services, and daemon tokens.
- **DSP-014 — Existing FPV transport first.** Cockpit Live reuses UIAI's CDP screencast/MJPEG fallback, action, audit, and diagnostics contracts before new transport is invented.
- **DSP-015 — Scoped mutation.** Navigation, input, takeover, release, close, and presentation mutations preserve ScopeRef, owner/lease, consent, and audit boundaries.
- **DSP-016 — Standalone resilience.** UIAI Engine remains useful headlessly; Cockpit remains useful without Menubar; Menubar remains useful without Cockpit.
- **DSP-017 — Typed failure.** Unavailable Cockpit, missing runtime, incompatible protocol, blocked scope, unavailable session, and failed attach are distinct recoverable states.
- **DSP-018 — Version negotiation.** Engine, Cockpit, Menubar, runtime bundle, and handoff protocol versions are observable and compatibility-checked.

---

# 3. Terminology

| Term | Meaning |
|---|---|
| **Browser runtime** | The packaged CDP-compatible executable controlled by UIAI Engine. It is a rendering dependency, not an authority plane. |
| **Canonical browser session** | The UIAI Engine session object that owns page/context state and stable session ID. |
| **Desktop presentation** | Cockpit's live view and controls attached to a canonical session. |
| **DesktopPresenter** | UIAI Engine adapter that requests Cockpit launch/focus and returns a typed receipt. |
| **Handoff intent** | An OS deep link containing an opaque target ref and bounded presentation intent. |
| **Truth plane** | Authenticated UIAI Engine or Focusa daemon state read used to resolve the opaque ref. |
| **Fast channel** | Optional per-user Unix socket/named pipe for local events and larger bounded payloads. |
| **FPV** | UIAI live browser projection using CDP screencast with MJPEG/polling fallback and audited controls. |

“Native UIAI product” means the Go Engine and Tauri Cockpit are packaged, launched, updated, and recovered as one signed product. It does not claim that Blink/V8 have been ported to Go.

---

# 4. Current-state audit

## 4.1 Already implemented and reusable

UIAI Engine already provides:

- persistent session IDs and a `SessionManager`;
- session open, list, inspect, navigate, screenshot, diagnostics, and close routes;
- FPV share creation with read-only/control roles and audit;
- `stream.cdp.mjpg` backed by `Session.CDPScreencast`;
- `stream.mjpg` screenshot-polling fallback;
- audited click, coordinate click, fill/type, press, annotate, and message actions;
- browser health and pressure reporting;
- Focusa scope metadata on browser operations.

Cockpit already provides:

- a Tauri/Svelte desktop shell;
- `/live` and legacy `/runs` routes sharing `LiveWorkspace`;
- Engine health, browser health, session list/open/navigate/screenshot/diagnostics/close integration;
- scope guards around session mutations;
- Bonjour discovery and Focusa phone-bridge commands;
- updater support and signed-update configuration.

Focusa Menubar already provides:

- a Tauri tray shell;
- Bonjour discovery;
- pairing bridge and Keychain token commands;
- the existing `focusa://` pairing-fallback requirement in Focusa portability documentation.

## 4.2 Missing or incomplete

The current repositories do not yet provide:

- a production packaged-browser manifest or startup preflight;
- a production prohibition on Rod's lazy browser download;
- a `DesktopPresenter` interface or `/present` route;
- a Cockpit single-instance/deep-link handler;
- `CFBundleURLTypes`/cross-platform protocol registration for `cockpit://`;
- complete `focusa://` handling in Menubar despite the existing spec;
- a Cockpit Live adapter for the existing FPV CDP stream and control channel;
- Engine-to-Cockpit launch/focus IPC;
- typed presentation receipts and failure recovery;
- a cross-app compatibility manifest;
- E2E proof that Cockpit controls the same session without state duplication;
- integration with the planned session broker, leases, parking, and reclamation.

The current Cockpit identifier is `com.wpuiai.uiaiengine.cockpit`; older prose uses `com.focusa.cockpit`. This amendment preserves the shipping identifier unless release engineering deliberately migrates it with signed-update and protocol-registration proof.

---

# 5. Target architecture

```text
Agent / API / workflow
        │
        ▼
UIAI Engine session broker
        │ owns
        ├── packaged CDP browser runtime
        ├── canonical session/context/page
        ├── diagnostics/evidence/audit
        └── DesktopPresenter
                │ presentation request
                ▼
       Cockpit single-instance shell
                │
                ├── Live FPV stream
                ├── bounded input/control
                ├── diagnostics/inspection
                └── Focusa context cards

Focusa Menubar ── focusa:// / cockpit:// ── Cockpit
        │                                      │
        └──────── Focusa daemon truth plane ───┘
```

Cockpit's WebView renders Cockpit UI. The browser page itself appears through the Engine-owned live transport. A future platform-native accelerated surface may replace the image transport only if it preserves the same Engine-owned session and authority boundary.

---

# 6. Packaged browser runtime

## 6.1 Runtime modes

```yaml
vision:
  browser_runtime: packaged_cdp       # production default
  browser_bundle_path: auto           # resolved from signed install layout
  allow_runtime_download: false       # production invariant
  runtime_manifest_required: true
```

Permitted modes:

| Mode | Environment | Behavior |
|---|---|---|
| `packaged_cdp` | production/default | Launch the signed runtime declared by the UIAI package manifest. |
| `explicit_path` | controlled development/CI | Launch an operator-provided path after compatibility checks. |
| `rod_managed_dev` | development only | Permit Rod download with explicit opt-in and conspicuous diagnostics. |

No “search installed browser and silently use it” production mode is permitted.

## 6.2 Runtime manifest

Each platform package includes `browser-runtime.json`:

```json
{
  "schema": "uiai.browser_runtime_manifest.v1",
  "runtime_id": "uiai-chromium-macos-x86_64",
  "engine": "chromium",
  "version": "128.0.6568.0",
  "cdp_protocol": "1.3",
  "platform": "darwin",
  "arch": "x86_64",
  "executable_relpath": "UIAI Browser Runtime.app/Contents/MacOS/Chromium",
  "sha256": "...",
  "signed": true,
  "source": "uiai-release",
  "built_at": "..."
}
```

Startup preflight verifies manifest schema, platform, architecture, file existence, checksum, executable permissions, signature where supported, and a bounded CDP version probe. Failure returns `browser_runtime_missing`, `browser_runtime_corrupt`, `browser_runtime_incompatible`, or `browser_runtime_untrusted`; it never starts an implicit download.

## 6.3 Packaging and update

- Browser artifacts are prepared during CI/release, not at first request.
- Cockpit, Engine, and runtime compatibility are published in release metadata.
- Updates stage side-by-side, verify, then switch atomically.
- Rollback retains the last compatible Engine/runtime pair.
- Runtime binaries are never embedded into transcript/tool output or copied into project worktrees.
- macOS signing/notarization covers the complete app/runtime bundle.

---

# 7. Desktop presentation contract

## 7.1 Engine route

Planned route:

```http
POST /api/session/{session_id}/present
```

Request:

```json
{
  "schema": "uiai.desktop_presentation_request.v1",
  "mode": "full|pip|focus_existing",
  "reason": "operator_request|takeover_required|policy_confirmation|failure_recovery|workflow",
  "scope_ref": { "project_root_key": "...", "continuity_id": "..." },
  "requested_by": { "client_type": "pi|cockpit|menubar|api", "client_id": "..." },
  "focus": true,
  "expires_in_ms": 30000,
  "idempotency_key": "..."
}
```

Response:

```json
{
  "schema": "uiai.desktop_presentation_receipt.v1",
  "presentation_id": "opaque",
  "session_id": "opaque",
  "status": "visible|launching|already_visible|blocked|unavailable|failed",
  "cockpit_instance_id": "opaque-or-null",
  "handoff_ref": "opaque-or-null",
  "reason_code": "...",
  "created_at": "...",
  "expires_at": "..."
}
```

The route does not return a secret-bearing deep link. It creates a short-lived handoff ref resolvable only through the local authenticated Engine.

## 7.2 Presenter interface

```go
type DesktopPresenter interface {
    EnsureVisible(ctx context.Context, req PresentationRequest) (PresentationReceipt, error)
    Status(ctx context.Context, presentationID string) (PresentationReceipt, error)
}
```

Platform adapters:

- macOS: Launch Services opens `cockpit://live/session/<id>?handoff=<opaque>`;
- Windows: registered protocol activation and named-pipe handoff;
- Linux: `xdg-open` protocol activation and Unix-socket handoff;
- headless: typed `desktop_unavailable` plus optional FPV share recovery.

## 7.3 Presentation state machine

```text
requested
  → resolving_session
  → resolving_cockpit
  → launching_or_focusing
  → attaching
  → visible
  → focused

terminal/recovery states:
blocked_scope | session_missing | cockpit_missing | incompatible |
attach_failed | expired | cancelled | desktop_unavailable
```

Every transition is bounded, observable, idempotent, and safe to retry.

---

# 8. Cockpit single-instance and session attach

Cockpit MUST:

1. register and exclusively own `cockpit://` for its shipping bundle ID;
2. use a single-instance plugin/guard;
3. parse only allowlisted routes and fields;
4. focus or show the existing main window;
5. route to `/live` and select the canonical `session_id`;
6. resolve the handoff ref through authenticated loopback Engine APIs;
7. verify session/scope/lease state before enabling controls;
8. attach the existing CDP FPV stream;
9. fall back to MJPEG/polling when CDP streaming degrades;
10. send controls through existing audited Engine endpoints;
11. show explicit blocked/degraded/reconnect states;
12. never navigate a second WebView to the target page.

Suggested modules:

```text
apps/cockpit/src-tauri/src/
  deep_link.rs
  single_instance.rs
  engine_process.rs
  desktop_handoff.rs

apps/cockpit/src/lib/
  contracts/desktop-presentation.ts
  adapters/desktop-presentation-adapter.ts
  adapters/fpv-live-adapter.ts
  controllers/live-session-controller.ts
  stores/presentation-store.ts
```

---

# 9. Focusa Menubar handoff

## 9.1 Scheme ownership

- Menubar owns `focusa://` under `com.focusa.menubar`.
- Cockpit owns `cockpit://` under `com.wpuiai.uiaiengine.cockpit` unless a separately approved bundle migration changes it.
- Both applications register schemes in signed package metadata and install deep-link handlers.

## 9.2 Canonical routes

```text
focusa://mission/<opaque-ref>
focusa://card/<opaque-ref>
focusa://workpoint/<opaque-ref>
focusa://connect?payload=<governed-pairing-payload>   # existing pairing exception

cockpit://live/session/<opaque-session-id>?handoff=<opaque-ref>
cockpit://focusa/<opaque-ref>
cockpit://evidence/<opaque-ref>
cockpit://settings/pairing
```

New route additions require protocol-version negotiation and allowlist tests.

## 9.3 Directional behaviors

Cockpit → Menubar:

- “Open Mission/Workpoint in Focusa” opens the relevant `focusa://` target.
- If Menubar is absent, Cockpit reads daemon truth and offers installation or an in-Cockpit read-only context surface.

Menubar → Cockpit:

- “Open browser session in Cockpit” requests Engine presentation and opens the resulting `cockpit://live/session/...` handoff.
- “Open Evidence/Diagnostics” opens the scoped Cockpit object.
- Menubar does not directly control a browser page or reuse Cockpit's token.

## 9.4 Authority boundary

A deep link is an intent envelope, not authentication. On receipt, the target app:

1. validates scheme, route, ref syntax, size, and TTL;
2. resolves the ref through its own authenticated Engine/daemon client;
3. verifies scope and ownership;
4. asks for consent when mutation/takeover requires it;
5. records an action/presentation receipt;
6. rejects stale or mismatched refs without fallback mutation.

---

# 10. Connection planes

The six existing connection planes remain distinct:

| Plane | Purpose | Mechanism |
|---|---|---|
| Truth | canonical pairing/scope/session state | UIAI Engine and Focusa daemon authenticated APIs |
| Discovery | find Engine, Cockpit, Menubar, daemon | loopback, Bonjour, Tailscale, saved hints |
| Handoff | open/focus sibling app on an opaque object | `focusa://`, `cockpit://` |
| Health | liveness/version/capabilities | loopback well-known manifest |
| Bridge | phone pairing completion | nonce-bound bridge callback/status poll |
| Fast channel | local bounded events/payloads | Unix socket or named pipe |

No plane may silently substitute for another. In particular, the fast channel and deep links cannot mutate canonical Focusa state.

---

# 11. Compatibility manifest

Cockpit and Menubar expose a local read-only manifest equivalent to:

```json
{
  "schema": "focusa.app.manifest.v2",
  "app": "focusa-menubar|uaiengine-cockpit",
  "version": "...",
  "channel": "stable|preview|dev",
  "protocols": {
    "focusa_deep_link": "1",
    "cockpit_deep_link": "1",
    "desktop_presentation": "1",
    "fpv": "1"
  },
  "capabilities": [
    "session.present",
    "session.attach",
    "mission.open",
    "pair.start"
  ]
}
```

Compatibility mismatch produces a bounded recovery action: update Cockpit, update Menubar, use FPV fallback, or continue headlessly.

---

# 12. Security and privacy

- Deep links MUST reject unknown routes, oversized refs, malformed encoding, path traversal, nested URLs, and replay after TTL.
- Page URLs, titles, cookies, DOM, screenshots, tokens, raw project roots, and credentials MUST NOT appear in deep links.
- Pairing's existing `focusa://connect` payload remains separately size-bounded, nonce-bound, schema-validated, and audited.
- Cockpit and Menubar MUST use their own daemon tokens and Keychain services.
- Local socket/pipe endpoints MUST enforce current-user ownership and restrictive filesystem permissions.
- Presentation requests MUST be origin/client identified and rate limited.
- A background agent MAY request presentation but MUST NOT silently acquire takeover authority.
- Window focus behavior must respect operator settings for foreground, notification-only, or failure-only presentation.
- Logs contain opaque IDs and reason codes, not private page content.

---

# 13. Reliability and recovery

| Failure | Required behavior |
|---|---|
| Runtime absent/corrupt | Fail startup preflight with repair/update command; no implicit download. |
| Cockpit not installed | Return `cockpit_missing`; offer signed installer or FPV fallback. |
| Cockpit installed but stopped | Launch it and await bounded attach receipt. |
| Cockpit already running | Focus existing instance and select session. |
| Session missing/parked | Resolve broker state; restore if authorized or show truthful missing/parked state. |
| FPV CDP stream fails | Fall back to MJPEG, then screenshot polling. |
| Menubar missing | Cockpit continues standalone. |
| Engine unavailable | Cockpit shows reconnect/repair; deep link remains non-mutating. |
| Protocol mismatch | Block handoff and identify required app update. |
| Scope conflict | Read-only orientation only; no takeover or navigation. |
| Duplicate request | Return the existing idempotent presentation receipt. |

---

# 14. Observability and receipts

Metrics:

- presentation requests/success/failure by reason code;
- time to Cockpit visible and time to first live frame;
- existing-instance vs cold-launch ratio;
- CDP→MJPEG→poll fallback count;
- handoff route success/failure;
- protocol mismatch and stale-ref count;
- browser runtime preflight duration/failure;
- unexpected runtime-download attempts, which MUST remain zero in production.

Evidence artifacts:

- runtime manifest/checksum/signing proof;
- presentation receipt;
- same-session proof before/after attach;
- deep-link routing test matrix;
- cross-app compatibility matrix;
- E2E video or frame evidence plus machine-readable action receipts;
- release metadata proving packaged runtime inclusion.

---

# 15. API and contract additions

Required schemas:

- `uiai.browser_runtime_manifest.v1`
- `uiai.desktop_presentation_request.v1`
- `uiai.desktop_presentation_receipt.v1`
- `uiai.desktop_presentation_status.v1`
- `uiai.app_handoff_intent.v1`
- `uiai.app_handoff_receipt.v1`
- `focusa.app.manifest.v2`

Required Engine surfaces:

```text
POST /api/session/{id}/present
GET  /api/presentation/{presentation_id}
POST /api/presentation/{presentation_id}/ack
GET  /.well-known/uiai-desktop.json
```

Required Cockpit controller capabilities:

```text
cockpit.presentation.open
cockpit.presentation.focus
cockpit.session.attach
cockpit.session.takeover.request
cockpit.handoff.focusa.open
```

Required Menubar capabilities:

```text
focusa.handoff.mission.open
focusa.handoff.workpoint.open
focusa.handoff.cockpit.open
focusa.handoff.session.present
```

---

# 16. Implementation layout

UIAI Engine repository:

```text
internal/browserruntime/
  provider.go
  packaged.go
  manifest.go
  preflight.go
  managed_dev.go

internal/desktop/
  presenter.go
  protocol.go
  launch_darwin.go
  launch_windows.go
  launch_linux.go

internal/routes/
  presentation.go

apps/cockpit/src-tauri/src/
  deep_link.rs
  single_instance.rs
  engine_process.rs
  desktop_handoff.rs

apps/cockpit/src/lib/
  contracts/desktop-presentation.ts
  adapters/fpv-live-adapter.ts
  controllers/live-session-controller.ts
```

Focusa repository:

```text
apps/menubar/src-tauri/src/
  deep_link.rs
  cockpit_handoff.rs

apps/menubar/src/lib/
  contracts/app-handoff.ts
  controllers/cockpit-handoff-controller.ts
```

Exact names may vary, but the authority and dependency boundaries may not.

---

# 17. Migration and compatibility

1. Preserve existing session and FPV routes.
2. Add presentation contracts without changing existing browser-tool responses.
3. Introduce packaged runtime preflight while keeping `rod_managed_dev` explicitly available to developers.
4. Add Cockpit URI registration and single-instance routing before exposing Engine presentation actions.
5. Wire Cockpit Live to existing FPV stream/control before creating new transport.
6. Add Menubar deep-link registration and route handlers while preserving tray behavior and pairing.
7. Integrate broker leases/parking once `uiai-engine-roadmap.6` and `.7` land.
8. Gate production release on no-download and same-session E2E proofs.

---

# 18. Implementation task graph

The canonical local Beads parent is `uiai-engine-roadmap.12`. IDs below are stable decomposition handles.

## T004-00 — Register specification and freeze authority map

**Bead:** `uiai-engine-roadmap.12.00`  
**Scope:** add this amendment and companion ledger; update document register/master/legacy notice; record current code and task graph.  
**Acceptance:** all references resolve; YAML parses; task IDs exist; no conflicting scheme ownership remains.  
**Can close immediately after this documentation change.**

## T004-01 — Define generated contracts and compatibility manifest

**Bead:** `uiai-engine-roadmap.12.01`  
**Depends on:** T004-00.  
**Scope:** Go/TypeScript/Rust contract types, route vocabulary, reason codes, protocol versions, fixtures, schema parity tests.  
**Acceptance:** identical fixtures parse across Engine/Cockpit/Menubar; no secrets permitted in handoff fixtures.

## T004-02 — Add browser-runtime provider and production no-download gate

**Bead:** `uiai-engine-roadmap.12.02`  
**Depends on:** T004-01 and browser-pressure recovery `uiai-engine-roadmap.1`.  
**Scope:** provider interface, modes, manifest preflight, typed failures, Rod-managed dev opt-in.  
**Acceptance:** production config cannot trigger download; missing runtime fails quickly and diagnostically.

## T004-03 — Package, sign, update, and roll back the runtime

**Bead:** `uiai-engine-roadmap.12.03`  
**Depends on:** T004-02.  
**Scope:** platform artifacts, checksums, release manifest, Tauri/Engine packaging, atomic update, rollback.  
**Acceptance:** clean-machine install opens a browser session offline from browser-download hosts.

## T004-04 — Implement Engine DesktopPresenter and presentation routes

**Bead:** `uiai-engine-roadmap.12.04`  
**Depends on:** T004-01 and completion of Cockpit-003 rollout `uiai-engine-roadmap.11.19`.  
**Scope:** presenter interface, route/status/ack, idempotency, platform launch adapters, typed fallback.  
**Acceptance:** repeated present calls converge on one receipt and never create a second session.

## T004-05 — Implement Cockpit deep-link registration and single-instance routing

**Bead:** `uiai-engine-roadmap.12.05`  
**Depends on:** T004-01 and `uiai-engine-roadmap.11.19`.  
**Scope:** `cockpit://`, signed bundle metadata, strict parser, existing-window focus, `/live` selection, settings.  
**Acceptance:** cold and warm activation select the requested opaque session; malformed links are rejected.

## T004-06 — Attach Cockpit Live to existing FPV stream and control

**Bead:** `uiai-engine-roadmap.12.06`  
**Depends on:** T004-04 and T004-05.  
**Scope:** CDP MJPEG stream, fallback ladder, input/control audit, diagnostics, reconnect, read-only/takeover states.  
**Acceptance:** Cockpit displays and controls the exact Engine session; no second navigation occurs.

## T004-07 — Implement Menubar `focusa://` registration and handlers

**Bead:** `uiai-engine-roadmap.12.07`  
**Depends on:** T004-01.  
**Repository scope:** Focusa `apps/menubar` with UIAI contract fixtures mirrored or generated.  
**Acceptance:** existing `focusa://connect` remains governed; mission/card/workpoint routes focus the Menubar safely.

## T004-08 — Implement bidirectional Cockpit/Menubar handoff

**Bead:** `uiai-engine-roadmap.12.08`  
**Depends on:** T004-05 and T004-07.  
**Scope:** Cockpit→Menubar and Menubar→Cockpit controllers, truth-plane fallback, compatibility manifest, app-absent recovery.  
**Acceptance:** both directions work cold/warm without token sharing or authority expansion.

## T004-09 — Add Engine/Cockpit lifecycle and local fast channel

**Bead:** `uiai-engine-roadmap.12.09`  
**Depends on:** T004-03 and T004-05.  
**Scope:** Cockpit-managed Engine startup/health/restart, per-user socket/pipe, version negotiation, update coordination.  
**Acceptance:** one installer launches a healthy product; lifecycle recovery does not orphan sessions or sockets.

## T004-10 — Integrate broker ownership, leases, parking, and takeover

**Bead:** `uiai-engine-roadmap.12.10`  
**Depends on:** T004-04, session broker `uiai-engine-roadmap.6`, and parking `uiai-engine-roadmap.7`.  
**Scope:** presentation eligibility, owner-aware controls, parked-session restore, lease expiry, explicit takeover.  
**Acceptance:** Cockpit never bypasses broker ownership and truthfully handles parked/reclaimed sessions.

## T004-11 — Harden security, privacy, accessibility, and degraded UX

**Bead:** `uiai-engine-roadmap.12.11`  
**Depends on:** T004-06, T004-08, T004-09, and T004-10.  
**Scope:** fuzz parsers, secret scans, rate limits, scope/consent guards, keyboard/screen-reader flows, app-missing/mismatch/reconnect UX.  
**Acceptance:** security and accessibility matrices pass; deep links contain only opaque bounded refs.

## T004-12 — Prove same-session E2E and release readiness

**Bead:** `uiai-engine-roadmap.12.12`  
**Depends on:** T004-11.  
**Scope:** cold/warm Cockpit launch, Menubar handoff, offline install, runtime preflight, fallback ladder, cross-platform CI, signed release artifacts.  
**Acceptance:** all §21 acceptance criteria pass with stable evidence refs and no production runtime downloads.

---

# 19. Dependency summary and shortest safe path

```text
T004-00 → T004-01 → T004-07
                   │
Current Cockpit-003 work
  T003-10/12/15/16/17/18/19
        ├→ T003-19 → T004-04 ─┐
        │            T004-05 ─┼→ T004-06
        │                     └→ T004-08 ← T004-07
        └→ browser pressure .1 → T004-02 → T004-03 → T004-09

Broker .6 + Parking .7 + T004-04 → T004-10

T004-06 + T004-08 + T004-09 + T004-10
        → T004-11 → T004-12
```

Fastest safe sequence:

- now: complete contract generation and compatibility fixtures (T004-01);
- after T004-01: Menubar scheme work (T004-07) may proceed in its verified Focusa worktree;
- after Cockpit-003 rollout: Engine presenter and Cockpit URI/single-instance work (T004-04 and T004-05) may run in parallel;
- after browser-pressure recovery: runtime provider and packaging work (T004-02 and then T004-03) may proceed;
- only after those foundations: Live attachment, cross-app handoff, lifecycle, broker integration, hardening, and release proof.

Do not begin Live replacement, lifecycle integration, or release proof before their dependencies. Existing Cockpit-003 tasks retain implementation priority and are not bypassed; only T004 documentation/contracts and independent Menubar protocol work can overlap safely.

`uiai-engine-roadmap.12` MUST block the existing final release/Cockpit visibility bead `uiai-engine-roadmap.10` so the architecture cannot be omitted from final closure.

---

# 20. Agent decomposition rules

Each implementation bead MUST record:

- exact project root and repository;
- governing DSP requirement IDs;
- target files/modules;
- authority and mutation boundary;
- dependencies and blockers;
- tests and evidence handles;
- compatibility behavior;
- rollback/recovery path;
- exact next action.

Cross-repository Menubar work remains represented in the UIAI local graph and references the Focusa commit/PR/evidence. Agents MUST verify project identity before switching repositories and MUST NOT treat copied fixtures as independent authority.

---

# 21. Unified acceptance criteria

This amendment is complete only when:

1. UIAI Engine starts from a clean signed install without downloading a browser at first use.
2. Production configuration cannot enable Rod-managed download accidentally.
3. A headless Engine session can be presented in Cockpit without changing session ID, URL, cookies, storage, DOM, or diagnostics lineage.
4. Cold presentation launches Cockpit; warm presentation focuses its existing instance.
5. Cockpit Live uses CDP FPV with MJPEG/polling fallback and audited controls.
6. Cockpit never creates a competing automation WebView session.
7. Menubar owns and handles `focusa://`; Cockpit owns and handles `cockpit://`.
8. Menubar can present a session in Cockpit and Cockpit can open scoped Focusa objects in Menubar.
9. Deep links contain no tokens, raw project paths, private URLs, page data, or credentials.
10. Each app keeps a distinct daemon token/device ID and resolves refs through truth-plane reads.
11. Missing app, missing session, parked session, protocol mismatch, scope conflict, and degraded stream all have tested recovery.
12. Session broker ownership, leases, parking, and takeover cannot be bypassed by desktop presentation.
13. Signed package metadata registers schemes without bundle-ID collision.
14. Compatibility manifests and protocol negotiation block unsafe version combinations.
15. Accessibility, keyboard, screen-reader, reduced-motion, and focus behavior pass Cockpit gates.
16. Cross-repository tests prove Focusa Menubar and UIAI Cockpit handoff in both directions.
17. Release proof includes runtime manifest/checksum/signing, offline clean-install, same-session E2E, and zero unexpected downloads.
18. The Beads graph is cycle-free and `uiai-engine-roadmap.10` cannot close before `uiai-engine-roadmap.12`.

---

# 22. Prohibited patterns

- silently launching installed Chrome, Brave, Edge, or a user profile;
- first-request production downloads;
- using Cockpit's WebView as a second agent browser;
- copying cookies/storage between Engine and Cockpit;
- embedding secrets, raw project paths, private URLs, or page content in deep links;
- sharing Menubar and Cockpit tokens or device IDs;
- direct app-to-app canonical Focusa mutation;
- focus stealing without operator policy;
- unbounded launch retries or duplicate Cockpit instances;
- treating “window opened” as proof that the correct session attached;
- bypassing session broker ownership or parked-state recovery;
- implementing new transport before reusing FPV/CDP and its tested fallback;
- closing the release bead without signed package and same-session evidence.

---

# 23. Final implementation principle

> UIAI Engine owns one canonical browser session; Cockpit makes that session visible and controllable; Focusa Menubar hands scoped intent to and from Cockpit; packaging and protocols make the whole product feel like one application without collapsing authority boundaries.
