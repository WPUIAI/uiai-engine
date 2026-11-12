# Proposed Cockpit Lite: ChromeOS / Web PWA / Linux / Chrome Companion

**Status:** Proposed additive architecture and implementation decomposition — NOT owner-promoted canonical architecture, built application, or release approval.  
**Prepared:** 2026-10-30 | **Reviewed UIAI main:** `1c0a2d3a5a2724fcad7dde7981c2a0be053316c9`  
**Parent draft:** [Unified MCP/WebMCP/Cockpit Agent Browser](UIAI_MCP_WEBMCP_COCKPIT_UNIFIED_AGENT_BROWSER_SPEC_2026-10-30.md), draft PR #290  
**Project:** `WPUIAI/uiai-engine`, `apps/cockpit/`  
**Purpose:** enable the existing UIAI Cockpit to run effectively on a low-memory Chromebook as an authenticated installable web/PWA, continue full Linux desktop support, and optionally integrate a consent-based Chrome side-panel companion—without creating any second browser/session/agent authority or forking the existing Cockpit UI.

## 0. Recommended decision

Ship **one Svelte Cockpit product with shared core operations and multiple capability-gated shells**:

1. **Cockpit Web / Lite PWA — immediate priority.** Existing SvelteKit static SPA, delivered from an authenticated HTTPS origin, installable on ChromeOS and usable on any Linux desktop browser. Chromebook uses remote engine sessions, remote Focusa/Wirebot, and a lightweight event-driven browser mirror. No local Chromium automation pool, Go engine, Linux container, Tauri/WebKitGTK, local LLM, or agent runner.
2. **Cockpit Desktop — Tauri v2 on macOS/Windows/Linux.** Reuse same source and canonical engine APIs; native shell adds device discovery, local filesystem, keychain-bound secure operations, signed OTA and native integrations where actually implemented. A Linux `.deb`/AppImage packaging lane is desirable but separate from Chromebook reachability.
3. **Cockpit Companion (optional MV3 Chrome extension) — future, gated.** Side panel for contextual agent status, attach authorized local browser tab as a separately scoped actor/capture surface, share selected page/URL/element with a workstream, consented handoff to Cockpit. It must NOT become the canonical Cockpit UI, autonomous background agent/runtime, browser fleet policy controller, generic authenticated site scraper, or trust bridge bypass.
4. **Existing FPV share PWA and Evidence PWA** remain narrow user-facing projections of already-owned sessions/artifacts, not another Cockpit, agent authority or session broker.

This **does not reverse issue #79's rejection of an extension for infrastructure configuration**: only optional browser-local capture/consent would be a unique extension capability. Never build a legacy Chrome App.

## 1. As-built facts and blockers, verified from repo

| Surface | As built | Gap for Chromebook |
|---|---|---|
| SPA output | `apps/cockpit/svelte.config.js` uses `@sveltejs/adapter-static`, `build/` fallback `index.html`. | No verified authenticated hosted Cockpit/PWA install path; manifest, service worker, web release and web E2E gates not shown. |
| Tauri | `apps/cockpit/src-tauri/tauri.conf.json` v0.3.2 packages `../build`; native updater/CSP. | Linux package qualification/distribution and ChromeOS Crostini performance unproven; cannot assume Tauri-only bridge works on web. |
| Native bridge | `apps/cockpit/src/lib/bridge/tauri.ts` tests `isTauri()` and no-ops on the web. | Explicit adapter-level availability/denial and alternate secure browser pairing needed rather than pretending native operations succeeded. |
| Live | `LiveWorkspace.svelte` reads real Engine health/sessions, open, navigate, close, screenshot, diagnostics, FPV share; polls every 5 seconds. | Full browser viewer and agent conversation/workstream/activity/approvals/takeover are not yet a proven integrated web surface; polling ignores #287 preference until fixed. |
| Live video | `apps/cockpit/src/lib/adapters/fpv-live-adapter.ts` resolves `/m/{token}/stream.cdp.mjpg` and screenshot fallback; `docs/SESSION_API.md` documents bounded FPS stream. | Need authenticated private PWA transport, scoped actor lease and low-bandwidth/backpressure rules; a tokenized public FPV share is not a broad authorization to UIAI APIs. |
| Cockpit catalog | `routes/capabilities/+page.svelte` builds card manifest locally and shows read-only catalog. | Engine live ephemeral WebMCP tools need separate origin/session-scoped inspector (PR #290) and can't be made permanent app privileges. |
| Discovery | `engine-client.ts` defaults to `127.0.0.1:7456`; engine discovery probes local/default endpoints. | Chromebook `localhost` means **Chromebook**, not remote VPS; authenticate/select a remote gateway and never silently switch to localhost. |
| Secret store | `secure-store.ts` imports Tauri plugin-store and has a memory fallback. File-store must not be assumed to be an OS keychain; web fallback is not durable authenticated identity. | Web login should use server-issued HttpOnly Secure SameSite cookies or short-lived scoped tokens held server-side; never localStorage permanent API keys, session bearer tokens or FPV controls. |
| Existing events | `internal/routes/websocket.go` has a generic broadcast SSE `eventBus`, wildcard Access-Control-Allow-Origin and no per-tenant/Workstream cursor in shown code. | **Do not expose or reuse this global SSE unchanged** for private Cockpit/agent data. Implement authenticated, audience-filtered, replayable scoped projection in canonical owner; never stream all sessions to every subscriber. |
| Public routes | `docs/FPV_PUBLIC_ROUTE_DEPLOY_PLAN.md` documents `fpv.wpuiai.com/m/*` only, explicitly excluding `/api/*`. | Do not reconfigure the FPV hostname to proxy all Engine APIs. Stand up a separate authenticated Cockpit gateway/domain, no guess about its public URL until deployed. |
| Feature policy | `docs/ENDPOINT_AUTH_MATRIX.md` says evaluator/customer deployment is blocked pending signed entitlement coverage and endpoint hardening. | Preview for authorized operator/internal fixture only until release gates independently pass. |
| Chrome extension precedent | Issue #79 rejects extension for fleet/infra config with no unique capability. | Companion remains a separate owner-decision, not prerequisite for PWA. |

Read existing `AGENTS.md`, `docs/ARCHITECTURE_AUTHORITY_POLICY.md`, Cockpit 000/002/004/005/008, `docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md` and PR #290 before edits. Do not create conflicting canonical Workstream, session, tabs, command, message or browser-tool authorities.

## 2. Runtime topology and browser identity

    Chromebook / ChromeOS
    └─ installed Cockpit Web/PWA (small Svelte shell)
       ├─ Workstream/mission, conversation and approval surfaces
       ├─ Live remote session mirror (on-demand image/MJPEG)
       └─ scoped commands, events, capture requests
           │ HTTPS, authenticated, CSRF-protected, rate-limited
           ▼
       Cockpit Operator Gateway (private backends, separate from FPV public share)
           ├─ session/actor/capability/entitlement enforcement
           ├─ verified Focusa pairing and exact Workstream Attachment
           ├─ targeted engine + agent event projection, cursor and resync
           └─ scoped stream/command proxy (no open reverse proxy)
           │
           ├─ UIAI Go Engine, Rod/Chromium, MCP + WebMCP broker
           ├─ Focusa daemon, CallGraph, Master Loop/Portable Memory evidence settlement
           └─ Wirebot/OpenClaw/Pi conversation/agent service where authorized

One physical UIAI browser session is not recreated in the PWA. The user *views* that remote Chromium session and sends authenticated intents to its engine owner; browser chrome/back/forward/navigation/URL bar in Cockpit are engine commands and session projections, not a local WebView's history. The remote browser may have different cookies/origin/IP than the Chromebook's Chrome browser. Do not misrepresent the two as the same browser.

If the user explicitly wants to operate their **local ChromeOS tab**, that is a separate opt-in companion-extension integration with distinct target/identity and equivalent policy gates; it does not magically adopt an already-authenticated remote UIAI session.

For existing Tailscale sovereign installations, the gateway can be on a tailnet-protected node reachable through an authorized HTTPS ingress/relay. Do not assume ChromeOS supports every Android/VM networking variant or expose the loopback engine on 0.0.0.0; prove endpoint/TLS/client identity/route reachability on the target Chromebook.

## 3. Build modes: shared app, selective capability bundles

`apps/cockpit/` remains the single product and design-system source. Create platform capability contracts, not copied components:

    apps/cockpit/
      src/lib/platform/
        capabilities.ts          # web/pwa/tauri/extension features with truthful unsupported state
        transport.ts             # authenticated gateway vs native engine adapter
        session-projection.ts    # common engine session/view model
        command-gateway.ts       # one governed action/approval route
        agent-conversation.ts    # Focusa/Wirebot scoped conversation adapter
        browser-stream.ts       # responsive FPS, pause, resume, error/refresh
      src/lib/ui/
        LiveWorkspace.svelte     # keep/reuse, not fork
        AgentInteraction.svelte  # scoped messages, intervention, status
        LiveBrowserViewport.svelte
        SessionTimeline.svelte
        ToolInspector.svelte
      static/manifest.webmanifest
      static/icons/             # existing licensed assets, maskable icon
      static/sw.js or build pipeline generated SW
      src-tauri/                 # same frontend for native packages

Paths above are proposed loci, not existing files. Keep actual current controllers/Work Surface code where available, use existing design tokens and the Cockpit 004 tab/pane contract rather than inventing a parallel "lite tabs" state. Platform-specific adapters can be lazy imports; web build must never invoke Tauri APIs. No destructive default or disabled-button misrepresentation; show native-only capability as unavailable with a reason and safe alternative.

Core modes:
- `web_lite`: default on ChromeOS/low memory. Essential mission, agent conversation, Live, approval, evidence, tool inspector; no local heavy editor engine.
- `web_full`: optional and lazily loaded richer Cockpit Studio/Documents/Test Lab on capable devices.
- `tauri_desktop`: same app + exact native capability adapters as supported.
- `chrome_companion`: separate MV3 side-panel bundle sharing typed contracts/very small components, never duplicating the app.

Switching mode must not alter canonical permissions, risk classes, approval semantics, workstream binding, completion status, event order, or browser ownership; it only changes presentation/performance and device-specific capabilities.

## 4. What an operator must actually be able to do in Lite

**LITE-001 Mission/agent awareness:** See which authorized agent (Wirebot/Pi/OpenClaw or other participant) is active, exact Focusa Project/Workstream/Workpoint/CallGraph reference when available, current action/goal, remaining steps, blockers and last verified outcome. If scope attachment or agent identity is absent, show unbound/unavailable; never fill from last-opened tab, last-active Workstream or user-visible similarity.

**LITE-002 Bidirectional conversation:** Send a message to an authorized agent within exact Workstream/agent-session attachment, read its responses, ask for status, provide corrections, pause/resume under canonical authority. Messages use existing conversation/command owners in Focusa/agent harness; never store a parallel Cockpit mission transcript or claim that typing to UIAI navigates an agent by magic. Agent service disconnected => visibly unavailable, not fake success.

**LITE-003 Real browser observation:** Select an engine node and session; show actual remote page mirror, URL/title/origin, active actor, navigation/spinner, current tool, diagnostics and latest frames with timestamps/freshness. Support manual address navigation, semantic click/fill, and optional coordinates only with exact viewport/observation binding. Do not iframe arbitrary websites or pretend a screenshot is an interactive DOM.

**LITE-004 Human co-control:** Operator pause/freeze, request take-control, single fenced control lease, changes audited; no simultaneous agent mutations. Human release triggers re-observe/reconcile and only then agent continuation. Explicit approvals and sensitive actions require same backend grant/entitlement/secret checks regardless of web/mobile/desktop/agent origin.

**LITE-005 Agent timeline:** Show event-cursor-backed action/diagnostics/MCP/WebMCP browser tool activity, agent utterance/action proposals, observed effect, evidence refs, operator approvals, pending/partial/unknown, source and timestamp. A tool callback resolving is not task completion.

**LITE-006 Evidence/verification:** Screenshot, existing EPWA viewer/share, safe artifact ref, recorded browser evidence and evidence candidate to Focusa. Show verified/failed/inconclusive and settled/provisional distinctly. The PWA must not independently adjudicate Focusa completion.

**LITE-007 Session and fleet controls:** Discover only authenticated/verified engine nodes, start/reconnect/close/park permitted sessions, health/pressure/queue and exact scope. Never create sessions simply because a PWA loads. No silent engine endpoint switching, no assuming unavailable means zero work.

**LITE-008 WebMCP:** Reuse the PR #290 ephemeral origin-bound WebMCP Inspector/Execute contracts; PWA displays origin/frame/toolset/risk/revision, offers governed call, and sees the same evidence/action receipt as MCP/Pi/Tauri.

**LITE-009 UI compliance:** Accessible keyboard/touch controls, responsive Chromebook split layout and smaller-screen stacked view, text labels, status and focus semantics, reduced-motion preference, no always-on animations. Provide an easy **Open Full Cockpit**/expand view, no loss of mission context.

## 5. Authenticated web delivery and secret boundary

**GATE-001** Serve Cockpit from **HTTPS with same-origin operator gateway**, a separately protected domain/path from existing `fpv.wpuiai.com/m/*`. The gateway proxies ONLY approved typed operation paths to private UIAI and Focusa services; never a generic `/api/*` passthrough, arbitrary URL target, arbitrary local daemon proxy, or direct internet-accessible engine port.

**GATE-002** Explicit operator login/device-pairing flow binds browser session, human principal, owner/tenant, exact permitted Engine node, Focusa daemon/ScopeRef/Workstream attachment and capability grants. If using a Cloudflare Access/Tunnel front door, Cloudflare identity alone is not UIAI entitlement or Focusa authority.

**GATE-003** HttpOnly, Secure, appropriately scoped SameSite cookies with CSRF protection for same-origin write requests and strict Origin checking, or an equivalently reviewed per-device short-lived delegated mechanism; backend holds private engine and Focusa credentials. No long-lived API keys or share-control URLs in localStorage, service-worker CacheStorage, browser analytics, logs, URL query, copied messages or extension sync storage. UIAI secret broker remains authoritative for action secrets.

**GATE-004** The existing broad SSE bus is NOT acceptable as the owner-private stream. Build/verify authenticated scoped event feed: `engine_node + session + Workstream attachment + actor grant + last event seq`; redact payloads by visibility; allow cursor resume and snapshot recovery when a gap is detected; no wildcard CORS for private streams; never trust an EventSource connection merely because a frontend is on a recognized origin.

**GATE-005** Session stream read capability distinct from action/keyboard/takeover capability, short TTL, targeted session/tenant/source identity, revoke on logout/lease handback/origin change. Reuse existing FPV CDP screencast where suitable but proxy via authorized private channel; tokenized public share links do not grant full Cockpit control.

**GATE-006** CSP/connect/frame/worker policies, trusted proxy, HSTS, secure cache, service-worker update integrity, no user-supplied HTML scripts, clickjacking protection, XSS/CSRF/SSRF checks, logout revocation. Any separate public evidence/share PWA remains behind its existing narrower tokenized/share visibility policy.

**GATE-007** Respect licensing README/ENDPOINT_AUTH_MATRIX: current code includes legacy unauthenticated loopback behaviors; new evaluator/customer distribution is blocked until owner-approved signed entitlement enforcement on all actual execution routes. Personal/internal prototype is not production release proof.

## 6. Low-memory and bandwidth design

**PERF-001:** The laptop must not load browser workers, Chromium automation, local LLM, Go daemon, Tauri/WebKitGTK or Crostini container for ordinary Cockpit Web use. Keep compute remote.

**PERF-002:** Make Live/browser mirror adaptive. Default inexpensive event summary; load first frame only for selected session. Suggested initial experiment: 1–2 fps idle/overview, 5–8 fps focused active viewing, explicit higher-quality mode only if device/network telemetry shows benefit. These are **provisional test profiles**, not delivered-performance claims. Pause MJPEG/frame polling on hidden tabs or collapsed viewport and keep metadata/events where permitted. Avoid multiple open per-session streams; bounded buffers and cancellation on navigation/unmount.

**PERF-003:** Lazy-load heavy Studio/3D/video/document/Remotion/Three modules only when requested. Tree-shake and route-split; minimal fonts/icons; reduce motion; avoid perpetual refresh of all screen panels. Reuse scoped browser-side query cache with expiration; no large session DOM/snapshot blobs in state. Virtualize session/event lists and bound cache limits.

**PERF-004:** Cache only versioned static public PWA shell assets with network-first/no-store auth and operation responses. Offline PWA = shell + last safe nonsecret metadata with an unmistakable **disconnected/read-only/unverified** state, not browser or agent continuity. Service worker must never replay queued mutations automatically.

**PERF-005:** On disconnect, stop stale control affordances, report last received sequence/time, re-authenticate and rehydrate from canonical engine+Focusa snapshot. Engine/agent continues remotely if its own lease/task permits; PWA disappearance never grants or silently revokes canonical work authority. No "agent stopped" inference from a closed Chromebook lid.

**PERF-006:** Measure on actual target device (include low-RAM 4 GiB test profile if device spec unavailable) and actual ChromeOS version. Collect JS bundle and network waterfall, renderer resident memory where observable, idle CPU, battery, first usable Live, stream latency, frame drops, reconnect time, background tab behavior, accessibility input latency, concurrent session thumbnails, 20-session fixture and 60-minute soak. Set acceptance budgets from measurements; do not assert hard MB targets without device proof.

## 7. Chrome companion: only where unique browser-local value exists

Use Manifest V3 `chrome.sidePanel` for small contextual agent UI; optional per-site `activeTab`/permission-gated content scripts for selected user tab context, local page/snapshot handoff and explicit share to current authenticated Workstream. Ask permission at point of use, do not request `<all_urls>` or full cookies/debugger permissions merely to display Cockpit. Never read/write web pages without user-granted scope and action policy. An extension can assist an existing local Chrome tab but cannot claim to be the remote Rod browser session.

The MV3 service worker is ephemeral; it must not be the sole owner of tasks, browser sessions, agent conversations, approvals, replay logs or event continuity. Panel open/close and browser restart recover from canonical backend event cursor. Chrome extension injection/content scripts have different isolation, site access and platform policies from remote WebMCP, so route each observation through the same provenance/authority gates. Do not use the extension to bypass unsupported browser WebMCP APIs, cross-origin restrictions or credential custody.

Product choice gate: build companion **only after** one requested use case (such as “share this Chrome tab with Wirebot and watch it in a side panel”) has measurable unique value vs PWA and is verified to respect the existing Cockpit/Veragensia browser identity contracts. Issue #79’s fleet-panel rejection remains intact.

## 8. Linux route and packaging

The installed PWA works on Linux Chrome/Chromium without extra packaging and should be the fast path. Independently qualify the existing Tauri v2 shell on supported Linux distributions. Tauri v2 requires WebKitGTK 4.1 dependencies, and `.deb`/AppImage packaging needs clean CI, signed/checksummed release artifact, actual graphical-launch and browser-session/Focusa tests, secure updater/support/rollback policy. ChromeOS Crostini (Linux VM/container) incurs extra memory and disk overhead, so do **not** make it a Chromebook prerequisite. Do not claim a Linux release just because the CI installs Linux Rust/WebKit dependencies; the exact package must be built and independently run.

## 9. Implementation task graph

| Packet | Source and dependency | Outcome and acceptance |
|---|---|---|
| CL-00 | Freeze main/PR #290 and Cockpit 000/002/004/005/008 contract status | Stable requirement-to-owner matrix, no competing spec or UI identity |
| CL-01 | `apps/cockpit/` shared platform/transport | Same Svelte build works in direct web browser and Tauri; web native operations explicitly disabled; no runtime crash |
| CL-02 | PWA assets/deployment | HTTPS authenticated Cockpit page with manifest, installable icon, correct static route fallbacks, no private API exposed |
| CL-03 | Gateway device auth, protected API, entitlement | Login/pair/revoke/device binding, web cookie/CSRF/Origin, signed entitlement, correct negative tests |
| CL-04 | Canonical scoped event feed | Active agent and browser events filtered to exact tenant/Workstream/session, gap recovery, bounded payload and cleanup |
| CL-05 | Live selected-session remote viewer | FPV stream and screenshot fallback with stale indicators, per-device resource budget, read-only safety |
| CL-06 | Agent interaction and action governance | Focusa/Wirebot messaging adapter, operator pause/takeover/approval and receipts; no shadow chat/store |
| CL-07 | WebMCP/MCP parity PR #290 | Tool inspector and governed browser tool execution have same effect, event, evidence across web/desktop/MCP |
| CL-08 | Constrained UI performance | Lazy bundles, pause hidden streams, low-memory Chromebook proof, keyboard/mobile UI, reduced motion |
| CL-09 | Linux Tauri package | Actual `.deb` and/or AppImage user run, signed/checksummed release, rollback and test evidence |
| CL-10 | Optional MV3 extension (after owner value approval) | minimal sidePanel+activeTab-only, per-origin consent, event synchronization, no new engine/Workstream or CDP authority |
| CL-11 | Independent release gate | Clean build, auth/tenancy/adversarial tests, ChromeOS real-device E2E, Linux VM E2E, multi-session/handoff, production rollback |

Every packet must be decomposed to exact existing files/contracts, minimal diffs, positive and negative tests, versioned receipts, independent verification and explicit release effect. Per current operator workflow, deploy/testing in production should occur only through approved guarded canaries and rollback; do not expose unfinished operator authority or browser session access.

## 10. Acceptance tests (must fail closed)

- Open the installable Cockpit PWA on a Chromebook **without enabling Linux**; show an exact authorized UIAI session created on the VPS, including changing screenshot/stream and current action.
- Chromebook `127.0.0.1:7456` never silently resolves as remote engine; missing gateway results in understandable disconnected state.
- Attach same session from Cockpit Lite, native Cockpit and MCP agent; all see matching IDs/current observation/revision/effect status; one authorized actor at a time.
- Agent active in Workstream A never leaks events/messages/evidence from Workstream B; switching tabs cannot authorize a mutation.
- New operator device cannot view private browser frames without authenticated permission; expired/revoked/read-only session denied before control; logout revokes active viewers.
- Press Take control while agent is mid-action: no concurrent writes; after handback, agent must get fresh observation and lease.
- Browser stream hidden, network slow, device sleeps and reloads: no memory leak, no duplicate session, no fabricated agent stop, bounded event resume.
- PWA offline: static shell remains optional; private evidence, cached secrets and queued mutations are absent; all actions visibly blocked.
- A malicious WebMCP tool/website/extension message cannot escalate to agent instruction, system authorization, credential export, or egress.
- Chrome companion not installed: **all normal Cockpit Lite remote agent/browser/operator features still work.**
- Linux Tauri package independently launches and connects to the same engine; no local UIAI/Focusa authority changes.
- Signed entitlement + protected-worker + operator/tenant/ScopeRef checks pass independently across gateway, streams, execution, FPV and WebMCP.
- ChromeOS low-memory test has recorded baseline and performance proof, no arbitrary “lightweight” claim based on bundle size alone.

## 11. Reference and owner decision

Original repository policies are the authority and this is merely proposed implementation guidance. The owner may promote a narrow decision: “Cockpit Web/PWA is the first low-memory ChromeOS access mode, sharing one Svelte, engine session, Focusa Workstream and protected command/event contract with Tauri; optional Chrome extension only when tab-local consent/context adds unique value.” Any delegation to an implementer must be scoped and cannot itself promote new canonical architecture.

References:
- https://developers.google.com/chromeos/app-development/learn/desktop-progressive-web-apps
- https://developer.chrome.com/docs/apps/migration/
- https://developer.chrome.com/docs/extensions/reference/api/sidePanel
- https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle
- https://v2.tauri.app/start/prerequisites/
- https://v2.tauri.app/distribute/debian/
- https://github.com/WPUIAI/uiai-engine/issues/79
- https://github.com/WPUIAI/uiai-engine/pull/290
