# UIAI Operator Browser Desktop Spec (Iterable Draft) — 2026-06-19

**Status:** Draft / intended to be edited in place  
**Audience:** UIAI Engine, Focusa, Focusa Cloud, and AI API product/engineering work  
**Normative refinement:** [`UIAI-COCKPIT-004`](UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md) governs packaged browser runtime, same-session Cockpit presentation, and Cockpit ↔ Focusa Menubar handoff. Section 17 remains the detailed connection-plane basis where Amendment 004 does not refine it.
**Related docs:**
- `docs/UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`
- `docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`
- `docs/UIAI_UX_DX_FPV_CONSOLIDATION_2026-06-12.md`
- `docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`
- UIAI research capture: shadcn-svelte homepage/components, Bits UI introduction, Tauri v2 SvelteKit/develop/distribute docs, `tauri-apps/tauri-action` README
- Focusa repo: `docs/43-multi-device-sync.md` — multi-device local-first sync, observation import, thread ownership
- Focusa repo: `docs/53-focusa-device-pairing-spec.md` — OAuth-like device pairing and trust boundary
- Focusa repo: `docs/54-focusa-pairing-room-plan.md` — phone/PWA-mediated pairing room flow
- Focusa repo: `docs/56-focusa-pairing-wizard-spec.md` — pairing wizard UX
- Focusa repo: `docs/57-focusa-pairing-revoke-and-repair.md` — revoke, repair, and re-pair behavior
- Focusa repo: `docs/90-ontology-backed-tool-contracts-parity-spec.md` — canonical `focusa_*` tool contracts and parity
- Focusa repo: `docs/98-project-root-crdt-reconciliation-foundation-spec.md` — project-root/workstream CRDT reconciliation foundation
- Focusa repo: `docs/104-typed-scoped-runtime-and-singleton-elimination-spec.md` — scoped runtime, anti-singleton authority model
- Focusa repo: `docs/105-agent-dx-ux-merged-scope-spec.md` — DX/UX surfaces for scope, recovery, and operator clarity
- Focusa repo: `docs/111-agent-context-bootstrap-and-delivery-spec.md` — preload packets and agent readiness receipts
- Focusa repo: `docs/112-install-binary-architecture-spec.md` — installer/license/update authority
- Focusa repo: `docs/production-deployment-guide.md` — daemon production deployment pattern
- Focusa repo: `docs/deploy-runbook.md` — deploy incident/operator runbook
- Focusa repo: `docs/live-release-automation.md` — tag-driven release/deploy pipeline
- Focusa repo: `docs/self-heal-chain.md` — layered deploy self-heal model
- Focusa repo: `docs/11-menubar-ui-spec.md` — Apple-like menubar pairing precedent
- Focusa repo: `docs/113-agent-benchmark-spec.md` — measured private benchmark/eval ledger evidence
- Focusa repo: `docs/114-public-benchmark-flywheel-spec.md` — public bench/evals/proof planes and redaction boundaries
- Focusa repo: `docs/115-focusa-cloud-control-plane-tool-gateway-master-spec.md` — SaaS/control-plane, tool gateway, relay, proof, benchmark, and multi-node plan

---

## 0. Status

**Status:** iterable draft, in active editing.
**Scope:** developer/operator cockpit for the browser/desktop client of Focusa Cloud/Spec 115, the UIAI Engine API, and the AI API.
**Authority:** extends UIAI Engine product direction, Focusa Spec 43/53/90/98/104/105/111/112/113/114/115, and the Focusa menubar app precedent. Does not redefine Focus State, Trajectory, Workpoint, CRDT, or Scope semantics.
**Owns:**
- cockpit runtime shell (`apps/cockpit/`),
- integration adapter layer for UIAI Engine, Focusa API/Tool Gateway, and AI API,
- backend-first build order Slice 0–6,
- pairing + token model for the cockpit as a Focusa device,
- card manifest, capability naming, smoke harness,
- menubar-parity release pipeline (tag/dev/release/deploy/auto-retry/audit),
- pre-UI acceptance and per-Slice gates.

**Does not own:**
- UIAI Engine server code (`internal/`),
- Focusa daemon code (`crates/`, `apps/menubar/`),
- WPUIAI WordPress plugin admin UI,
- cloud cognition authority,
- silent CRDT merges,
- AI API server implementation,
- cloud account/billing UX.

---

## 1. Normative basis

This spec depends on and preserves these existing surfaces:

| Existing surface | This spec uses it for |
| --- | --- |
| UIAI Engine HTTP/session/diagnostics APIs | browser execution cards and FPV/mirror primitives |
| Focusa Spec 43 — multi-device sync / CRDT | node graph, observation/proposal UX, thread ownership |
| Focusa Spec 53 — device pairing | cockpit as a first-class paired device, token model |
| Focusa Spec 90 — tool contracts | `focusa_*` Phase 0 cards must derive from canonical contracts |
| Focusa Spec 98 — CRDT reconciliation foundation | Mac + VPS dual-node model, scope-keyed writes |
| Focusa Spec 104 — typed scope | every Focusa/SaaS card carries ScopeRef |
| Focusa Spec 105 — DX/UX merged scope | cockpit DX/scope surfaces |
| Focusa Spec 111 — agent context bootstrap | preload/readiness card surface |
| Focusa Spec 112 — installer/license | dev-key + AI API key storage in OS keychain |
| Focusa Spec 113 — private benchmark evidence | cockpit bench/preview cards |
| Focusa Spec 114 — public benchmark flywheel | public-safe snapshot rendering only |
| Focusa Spec 115 — cloud control plane | cloud profile, node registry, pairing, proof, relay |
| AI API (`ai.wpuiai.com`) | hosted AI cards (read-only in MVP) |
| Mac menubar app pattern | pairing flow, Apple-like aesthetic, tagged-release pipeline |
| Focusa daemon release automation | tag-driven cockpit releases with self-heal |

---

## 2. Purpose

The current FPV/PWA work proves that operators want to **see what the agent/browser sees** and eventually **steer it in real time**. The next iteration is not just a passive FPV mirror and not a brand-new browser engine. It is a **desktop operator surface** for the agent-first stack.

This spec is intentionally **iterable**:
- sections are expected to be revised rather than treated as frozen truth;
- product boundaries are stated explicitly so later edits can preserve them;
- open questions are kept in the document instead of hidden in chat history.

### 2.1 Product vision

Build a **beautiful, app-like developer cockpit** where a human can see and operate the agent-first stack without needing to remember CLI commands, raw endpoints, or hidden daemon state.

The cockpit should feel like a professional desktop control surface:
- visual first, with browser state, node state, scope state, and API state visible at a glance;
- functional and powerful, with cards that can actually perform bounded actions;
- friendly to humans, with plain-language blocked states, recovery hints, and proof/receipt trails;
- local-first by default, with cloud convenience layered on top rather than cloud custody;
- honest about authority, so the operator always knows which node, API, token, and scope a card will use;
- Apple-like in beauty and simplicity: calm layout, native-feeling controls, high-quality typography, restrained motion, generous spacing, obvious primary actions, and advanced detail hidden until needed.

North-star experience:

```text
Open cockpit → choose Local Only or Cloud Profile → pair once → select node/scope → see browser + Focusa + Cloud + AI API state → run card → capture evidence/proof → hand control back to the agent.
```

### 2.2 Grounding map (this spec → Focusa/UIAI authorities)

This browser/desktop spec is a client-experience layer over existing Focusa authority specs. It must not invent conflicting semantics.

| Browser concern | Normative Focusa source | Browser requirement |
| --- | --- | --- |
| Multi-node Focusa | Spec 43 + Spec 98 | Show nodes, peers, CRDT sync, observations, thread ownership, and conflicts as first-class UI states. |
| Scoped authority | Spec 104 + Spec 105 | Key every Focusa/SaaS card by ScopeRef; stale/conflicting scope becomes visible read-only state. |
| Device trust | Spec 53 | Pair browser/desktop clients as devices with scoped, revocable tokens. |
| Tool cards | Spec 90 | Generate or validate `focusa_*` cards from canonical tool contracts, not ad-hoc UI definitions. |
| Agent readiness | Spec 111 | Surface preload packet/receipt state and missing readiness fields. |
| Install/license/cloud | Spec 112 + Spec 115 | Treat cloud as account/license/node/control-plane coordinator, not cognition authority. |
| Bench/proof | Spec 113 + Spec 114 + Spec 115 | Render public-safe receipts/snapshots only; never expose raw evals, logs, prompts, or private browser diagnostics by default. |

---

## 3. Product thesis

Build a **UIAI-first developer/operator browser** with **extension-style cards** that can also interact with the Focusa SaaS/control-plane surfaces.

Meaning:
- **UIAI Engine** remains the browser execution backend;
- the desktop app is a **control + observability shell** over existing UIAI Engine APIs;
- **Focusa** integrates as the local context/evidence/continuity authority;
- **Focusa Cloud / Spec 115** integrates as the hosted SaaS control plane for accounts, nodes, devices, relay, proof, benchmarks, and tool gateway policy;
- **AI API** integrates as the AI service surface when cockpit cards need hosted AI capabilities;
- **Wirebot** can integrate later as a conversational/agent companion surface, but it is pending and not a Phase 0 priority;
- **WPUIAI the WordPress plugin is not surfaced in this browser MVP**; it is a separate WordPress product that consumes AI API;
- these integrations should feel closer to **browser extension cards / operator panels** than to a monolithic all-in-one app.

### 3.1 Architectural boundary

The desktop product should **not** start by embedding a second browser runtime or forking browser execution away from UIAI Engine.

The desktop product **should**:
- connect to UIAI Engine sessions;
- render FPV / mirror / screenshot / diagnostics data;
- issue agent-safe control actions;
- host optional integration panels for Focusa, Focusa Cloud, and AI API first, with Wirebot as a later companion integration;
- interact with Focusa SaaS through explicit cloud/client APIs, not by treating cloud as local authority;
- run fully useful in Local Only mode without requiring a SaaS login.

### 3.2 Spec 104 scoped authority boundary

The browser must fully honor Focusa Spec 104.

Requirements:
- every project-sensitive browser action carries an explicit ScopeRef / scope envelope;
- the scope strip is always visible when Focusa or Focusa Cloud cards are active;
- cards, caches, session state, and receipts are keyed by scope instead of global singleton state;
- ScopeRef must distinguish `cloud_node_id`, `machine_id`, daemon endpoint, `project_root_key`, `workstream_key` / continuity id, thread id, session id, client type, and role;
- the browser must support more than one Focusa daemon at once, e.g. a local Mac node plus a VPS node;
- unknown, stale, or conflicting scope downgrades the UI to read-only health/orientation actions;
- browser cards route mutations only to the thread-owner / locally authoritative node for that scoped workstream;
- non-owner nodes may show observations, proposals, sync state, and receipt summaries, but must not silently write canonical Focus State;
- browser cards never promote cloud data into canonical local state without local node authority;
- relay, proof, and SaaS responses display redacted scoped summaries unless the local node approves richer output.

### 3.3 Multiplexed Focusa topology boundary

The desktop product must model Focusa as a **node graph**, not a singleton daemon.

It should make these dimensions visible and routable:
- multiple Focusa nodes under one account or team;
- multiple daemons reachable by local loopback, SSH/BYO tunnel, or Focusa Cloud relay;
- multiple sessions per node, including Pi, Claude Code, Codex, Cursor, OpenCode, MCP, menubar, CLI, and CI;
- multiple project roots and workstreams per node;
- thread ownership and assistant/observer roles;
- CRDT sync status, backlog, duplicate-event skips, stale packets, and ownership conflicts.

The browser is allowed to coordinate and visualize reconciliation, but canonical reconciliation remains governed by Focusa CRDT and local node authority.

### 3.4 Authority planes

The desktop/browser must keep three authority planes visually and technically separate:

```text
UIAI Engine          = browser execution authority
Local Focusa Node(s) = cognition, Workpoint, Evidence, CRDT, and tool execution authority
Focusa Cloud         = SaaS coordination, entitlement, relay, receipt, proof, and benchmark authority
```

Consequences:
- UIAI browser actions target UIAI sessions and return browser artifacts;
- Focusa cards route to the selected scoped Focusa node / daemon;
- Focusa Cloud cards route to SaaS control-plane APIs and may request local node execution through allowed capabilities;
- AI API cards route to authenticated hosted AI API endpoints;
- no plane may silently impersonate another plane.

### 3.5 Operating profiles

The browser must support two first-class operating profiles.

#### Local Only

Local Only is the default trust posture.

It provides:
- local UIAI Engine browser sessions;
- local Focusa daemon/cards where available;
- local node selection for every reachable daemon;
- local evidence/proof preview where supported;
- no Focusa Cloud sign-in requirement;
- no SaaS heartbeat requirement;
- clear offline/unconfigured states for cloud-only cards.

Local Only must never feel like a broken trial mode. It is the core local-first product.

#### Cloud Profile

Cloud Profile adds Focusa SaaS convenience.

It provides:
- account/license/profile state;
- cloud node registry and heartbeat;
- device registry and pairing recovery;
- relay/BYO tunnel visibility;
- proof receipt hosting and benchmark snapshot flows;
- team/multi-node dashboard surfaces;
- MCP/tool gateway policy and entitlement state.

Cloud Profile must not become cognition authority. It coordinates access, receipts, relay, billing, and visibility while local nodes retain Workpoint/Focus State/Evidence authority.

### 3.6 API triad to surface

The browser must make three API planes visible as first-class developer surfaces.

| API plane | Primary endpoint style | What the browser surfaces | Authority boundary |
| --- | --- | --- | --- |
| UIAI Engine API | local UIAI Engine HTTP/session APIs | browser sessions, FPV/mirror, screenshot, snapshot, click/fill/press/eval, diagnostics, health, capacity | UIAI owns browser execution only. |
| Focusa API / Tool Gateway | local Focusa daemon API plus Spec 115 cloud/tool-gateway routes | `focusa_*` cards, Workpoint, Trajectory, Evidence, ScopeRef, node registry, pairing, relay, proof, preload, benchmark, MCP/tool gateway | Local node owns cognition/tool execution; cloud coordinates. |
| AI API | hosted AI API (`ai.wpuiai.com`) | AI health, usage/entitlement, hosted AI actions, model/provider-facing cards where available | Hosted AI service owns AI request/response execution; it is not WPUIAI plugin admin UI. |

The UI should not bury these APIs behind generic “integrations.” A developer should be able to see which API a card will call, which credential/token it needs, whether it is local/offline-capable, and what proof/receipt/audit trail it produces.

Wirebot may appear later as a companion surface, but it is pending and not a Phase 0 dependency. When added, it must not be a hidden fourth authority plane: Wirebot requests should display the same scope, node, API target, and receipt/proof behavior as equivalent cards.

### 3.7 Backend integration layers before UI

Before investing in polished UI screens, the boring integration middle must be explicit. The cockpit MVP is blocked until these layers have names, contracts, owner APIs, fallback behavior, and test/smoke proof.

| Layer | Purpose | MVP contract |
| --- | --- | --- |
| Runtime discovery | Find reachable UIAI Engine, Focusa daemon(s), Focusa Cloud profile, and AI API. | `discover()` returns status, version, endpoint, mode, and capability flags per API plane. |
| Pairing/auth | Establish trust for local nodes, cloud profile, relay, and AI API. | Tokens are scoped, revocable, per-device, per-node/profile; first grant is read-only. |
| Scope authority | Apply Spec 104 everywhere before any meaningful action. | Every Focusa/SaaS card receives ScopeRef or is blocked/read-only with a visible reason. |
| Node graph | Model multiple Focusa daemons on Mac/VPS/remote nodes. | Node selector shows node id, machine id, endpoint, role, health, sync state, and authority role. |
| Transport/router | Decide local loopback vs BYO tunnel vs Focusa relay vs direct cloud/API call. | Router selects target per card and records why; no raw daemon proxying through cloud. |
| API adapters | Normalize UIAI Engine API, Focusa API/Tool Gateway, Focusa Cloud, and AI API. | Each adapter exposes `health`, `capabilities`, `call`, `errors`, and redaction metadata. |
| Card manifest | Turn backend primitives into UI cards. | Cards derive from tool contracts/API metadata with side-effect class, required scope, offline behavior, and receipt behavior. |
| Event stream | Keep cockpit live without polling chaos. | Unified event envelope for browser actions, Focusa node events, sync/proposals, cloud receipts, and AI API calls. |
| Command execution | Run bounded actions safely. | Reads execute immediately when authorized; writes require scope + token + owner/approval; code runs only through Code Capsule. |
| Sync/reconciliation | Expose CRDT state without merging cognition in UI/cloud. | Show observations, proposals, ownership, backlog, duplicate skips, stale packets, and conflicts. |
| Evidence/proof | Preserve outputs as evidence/proof without leaking private state. | Results can become local evidence refs; cloud proof publishing requires redaction/public-safe gates. |
| Error/recovery | Make failures friendly and actionable. | Typed error envelope with human message, technical detail, retry path, and responsible API plane. |
| Local cache/store | Keep UI responsive and local-first. | Cache is scope-keyed and node-keyed; no singleton “current Focusa” cache. |
| Deploy/update | Ship the cockpit like a product. | Tauri artifacts, signed/notarized where needed, checksums, smoke checks, update visibility, rollback path. |

Pre-UI acceptance:
- one smoke test per API adapter;
- one local-only happy path: UIAI health + Focusa project identity/workpoint resume without cloud login;
- one cloud-profile happy path: cloud profile/node status without local cognition upload;
- one multi-node routing test: Mac node and VPS node are distinguishable and cannot cross-write;
- one scope-conflict test: stale/missing ScopeRef blocks writes and explains why;
- one proof/evidence test: browser diagnostics or screenshot result becomes a local evidence ref;
- one pairing repair test: revoked/expired token becomes a repair/re-pair UI state.

UI design may proceed only after these contracts are stable enough to mock accurately.

### 3.8 Integration contract sketch

The MVP should create a small shared TypeScript contract package before visual implementation. UI components consume these contracts; adapters implement them.

```ts
type OperatingProfile = "local_only" | "cloud_profile";

type ApiPlane =
  | "uiai_engine"
  | "focusa_local"
  | "focusa_cloud"
  | "ai_api";

type AuthorityPlane =
  | "browser_execution"
  | "local_node"
  | "cloud_control_plane"
  | "hosted_ai";

type HealthState = "unknown" | "ok" | "degraded" | "offline" | "blocked";

type SideEffectClass =
  | "read"
  | "local_write"
  | "cloud_write"
  | "code_capsule"
  | "proof_publish"
  | "benchmark_run";

interface EndpointStatus {
  plane: ApiPlane;
  endpoint: string;
  version?: string;
  health: HealthState;
  capabilities: string[];
  auth_state: "none" | "paired" | "expired" | "revoked" | "missing";
  last_checked_at: string;
  human_status: string;
}

interface ScopeRef {
  project_root_key?: string;
  project_label?: string;
  workstream_key?: string;
  continuity_id?: string;
  thread_id?: string;
  session_id?: string;
  cloud_node_id?: string;
  machine_id?: string;
  daemon_endpoint?: string;
  role?: "owner" | "assistant" | "observer" | "reviewer" | "ci_runner" | "support";
  authority_state: "verified" | "missing" | "stale" | "conflict" | "read_only";
}

interface NodeRef {
  cloud_node_id?: string;
  machine_id: string;
  display_name: string;
  endpoint: string;
  transport: "loopback" | "ssh" | "byo_tunnel" | "focusa_relay" | "cloud_only";
  health: HealthState;
  sync_state?: "unknown" | "current" | "backlog" | "conflict" | "offline";
  authority_role?: ScopeRef["role"];
  version?: string;
}

interface PairingState {
  device_id?: string;
  device_name: string;
  device_type: "desktop_cockpit" | "browser_session" | "mac_app" | "pi_session" | "mcp_client";
  token_state: "none" | "pending" | "active" | "expired" | "revoked" | "repair_required";
  granted_scopes: string[];
  mutation_grant: boolean;
  node?: NodeRef;
  repair_action?: "re_pair" | "rotate_token" | "revoke" | "open_connect";
}

interface ApiAdapter {
  plane: ApiPlane;
  discover(): Promise<EndpointStatus>;
  capabilities(): Promise<string[]>;
  call<TInput, TOutput>(request: AdapterRequest<TInput>): Promise<AdapterResult<TOutput>>;
}

interface AdapterRequest<TInput = unknown> {
  plane: ApiPlane;
  capability: string;
  input: TInput;
  scope?: ScopeRef;
  node?: NodeRef;
  side_effect: SideEffectClass;
  idempotency_key?: string;
}

interface AdapterResult<TOutput = unknown> {
  ok: boolean;
  output?: TOutput;
  error?: CockpitError;
  evidence_ref?: string;
  receipt_ref?: string;
  redaction_state?: "none" | "redacted" | "blocked" | "public_safe";
}

interface CardManifest {
  card_id: string;
  label: string;
  product_surface: ApiPlane | "wirebot";
  authority_plane: AuthorityPlane;
  normative_source: string;
  required_scope: "none" | "project" | "workstream" | "thread" | "session" | "node" | "team";
  side_effect_class: SideEffectClass;
  capabilities: string[];
  offline_behavior: "works" | "read_only" | "hidden" | "blocked_with_reason";
  receipt_behavior: "none" | "local_receipt" | "cloud_receipt" | "proof_receipt";
  visual_priority: "phase0" | "phase1" | "later";
}

interface CockpitEvent {
  event_id: string;
  at: string;
  plane: ApiPlane | "cockpit";
  kind:
    | "health_changed"
    | "scope_changed"
    | "node_changed"
    | "pairing_changed"
    | "card_started"
    | "card_completed"
    | "card_failed"
    | "sync_changed"
    | "proof_changed";
  scope?: ScopeRef;
  node?: NodeRef;
  summary: string;
  evidence_ref?: string;
  receipt_ref?: string;
}

interface CockpitError {
  code: string;
  plane: ApiPlane | "cockpit";
  human_message: string;
  technical_detail?: string;
  retry_strategy?: string;
  recovery_action?: "pair" | "repair_pairing" | "select_scope" | "select_node" | "retry" | "open_logs";
  correlation_id?: string;
}
```

Rules:
- UI never calls raw endpoints directly; it calls adapters.
- Adapters never infer ScopeRef from global state; ScopeRef is passed in or the action blocks.
- Card manifests are data, not component code.
- Events are append-only from the UI perspective; views derive state from latest event + adapter status.
- Error envelopes are user-facing first and technical second.

### 3.9 Pre-UI backend MVP sequence

Build the integration middle in this order:

1. **Contract package** — define the interfaces above and fixture examples for local-only and cloud-profile modes.
2. **Discovery adapters** — read-only status for UIAI Engine, local Focusa daemon(s), Focusa Cloud profile, and AI API.
3. **Pairing/auth state** — represent paired/unpaired/expired/revoked/repair-required without yet building pretty flows.
4. **ScopeRef resolver** — select/verify scope and prove stale/conflict states block writes.
5. **Node graph/router** — distinguish Mac node, VPS node, relay path, and cloud-only surfaces.
6. **Phase 0 card manifest** — generate/static-declare the first card set with side-effect and offline behavior.
7. **Unified error envelope** — normalize failures from all three API planes.
8. **Event bus** — append card lifecycle, scope changes, node changes, and proof/evidence results.
9. **Smoke harness** — CLI/script that runs the pre-UI acceptance list without the polished app UI.
10. **Mock UI fixtures** — only then start beautiful screens against stable fixtures.

This keeps the team from painting a beautiful shell over undefined integration behavior.

### 3.10 Integration call stack

Every card action should travel through the same boring middle path:

```text
Card UI
  ↓
CardController
  ↓
ScopeGuard / AuthorityGuard
  ↓
NodeRouter / TransportRouter
  ↓
ApiAdapter: UIAI Engine | Focusa Local | Focusa Cloud | AI API
  ↓
ResultNormalizer / RedactionBoundary
  ↓
EventBus + LocalStore
  ↓
Card ViewModel
```

Responsibilities:
- **Card UI** renders state and collects intent only.
- **CardController** validates the card manifest and creates an `AdapterRequest`.
- **ScopeGuard** blocks missing/stale/conflicting ScopeRef before writes.
- **AuthorityGuard** blocks local writes unless the selected node/thread role allows them.
- **NodeRouter** chooses Mac/VPS/relay/cloud target and records the route decision.
- **ApiAdapter** owns endpoint-specific details and typed error translation.
- **ResultNormalizer** converts raw API output into card-safe data.
- **RedactionBoundary** prevents cloud/public surfaces from receiving private cognition by default.
- **EventBus** records what happened for timeline, debugging, proof, and UI replay.
- **LocalStore** caches only scope-keyed/node-keyed summaries, never singleton global authority.

Hard rule:

```text
No Svelte component calls fetch/curl/raw API directly for production card actions.
```

### 3.11 Pre-UI smoke matrix

The first useful backend milestone is a smoke harness that exercises integration without polished UI.

| Smoke | Mode | Expected result |
| --- | --- | --- |
| UIAI health | Local Only | UIAI adapter reports endpoint/version/capabilities or friendly offline error. |
| UIAI diagnostics read | Local Only | Diagnostics read works or returns typed unavailable/capacity error. |
| Focusa local discovery | Local Only | At least one daemon can be registered as a `NodeRef`, or no-node state explains next pairing/connect step. |
| Focusa ScopeRef resolve | Local Only | Project/workstream/thread scope resolves or blocks writes with `select_scope`. |
| Focusa workpoint read | Local Only | Workpoint resume/card read succeeds when scope verified; no cloud required. |
| Multi-node distinction | Local Only or Cloud Profile | Mac and VPS nodes have distinct machine/node ids and cannot cross-write. |
| Pairing repair | Local Only or Cloud Profile | Expired/revoked token produces repair/re-pair action, not generic failure. |
| Cloud profile read | Cloud Profile | Account/node heartbeat reads without uploading Workpoint/Focus State/raw evidence. |
| Relay route preview | Cloud Profile | Relay-capable card shows route/allowlist before execution. |
| AI API health/usage | Local Only or Cloud Profile | AI API adapter reports health/usage/credential state with hosted-AI authority label. |
| Evidence capture | Local Only | Browser artifact result can become local evidence ref. |
| Proof preview | Cloud Profile | Proof card shows redaction/public-safe gate before publish. |

A visual MVP should not claim readiness until this smoke matrix is runnable in CI or a repeatable local script.

### 3.12 Phase 0 package/file layout

Proposed layout for a fresh Svelte/Tauri cockpit inside this repo:

```text
apps/cockpit/
  package.json
  svelte.config.js
  vite.config.ts
  tsconfig.json
  components.json
  src-tauri/
    tauri.conf.json
    Cargo.toml
    src/
      main.rs
      commands.rs
      secure_store.rs
  src/
    app.css
    routes/
      +layout.svelte
      +page.svelte
    lib/
      contracts/
        api-plane.ts
        scope-ref.ts
        node-ref.ts
        pairing-state.ts
        card-manifest.ts
        cockpit-event.ts
        cockpit-error.ts
      adapters/
        uiai-engine-adapter.ts
        focusa-local-adapter.ts
        focusa-cloud-adapter.ts
        ai-api-adapter.ts
      router/
        node-router.ts
        transport-router.ts
        scope-guard.ts
        authority-guard.ts
      cards/
        phase0-card-manifest.ts
        card-controller.ts
      store/
        local-store.ts
        event-bus.ts
      smoke/
        smoke-runner.ts
        smoke-fixtures.ts
      ui/
        design-tokens.css
        components/
        shell/
        cards/
        panels/
```

Notes:
- `contracts/` is the first deliverable; it should be UI-independent.
- `adapters/` should be testable without Tauri window chrome.
- `src-tauri/` should hold only native shell, secure storage, deep-link/pairing helpers, updater hooks, and safe command bridges.
- Svelte components live under `ui/` and consume view models, not raw adapter responses.
- shadcn-svelte/Bits UI components are copied/customized into `ui/components/` as owned source, not treated as opaque vendor magic.

### 3.13 Phase 0 smoke script shape

The first script should run without opening the polished app window.

Proposed command:

```bash
cd apps/cockpit
pnpm cockpit:smoke --mode local-only
pnpm cockpit:smoke --mode cloud-profile --allow-cloud-readonly
```

Proposed smoke runner flow:

```text
1. Load env/profile: local-only or cloud-profile.
2. Discover UIAI Engine endpoint.
3. Discover local Focusa daemon endpoints.
4. Optionally read Focusa Cloud profile/token state.
5. Optionally read AI API credential/health state.
6. Resolve or mock ScopeRef.
7. Build NodeGraph.
8. Load Phase 0 card manifest.
9. Validate each card has adapter, authority plane, side-effect class, offline behavior, and error mapping.
10. Execute read-only cards.
11. Attempt blocked write with stale/missing scope and assert friendly block.
12. Emit smoke report JSON.
```

Smoke report schema:

```json
{
  "schema": "uaiengine.cockpit.smoke.v1",
  "mode": "local_only",
  "started_at": "...",
  "ended_at": "...",
  "endpoints": [],
  "nodes": [],
  "scope_state": "verified|missing|stale|conflict|read_only",
  "cards_checked": 0,
  "cards_passed": 0,
  "cards_failed": 0,
  "blocked_write_proven": true,
  "private_cognition_uploaded": false,
  "failures": []
}
```

### 3.14 Phase 0 GitHub Actions shape

Initial workflow should be boring and proof-oriented:

```yaml
name: Cockpit Phase 0 Smoke
on:
  pull_request:
    paths:
      - 'apps/cockpit/**'
      - 'docs/UIAI_OPERATOR_BROWSER_DESKTOP_SPEC_2026-06-19.md'
  push:
    branches: [main]
    paths:
      - 'apps/cockpit/**'
      - '.github/workflows/cockpit-smoke.yml'

jobs:
  contracts-and-smoke:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
      - uses: actions/setup-node@v4
      - run: pnpm --dir apps/cockpit install --frozen-lockfile
      - run: pnpm --dir apps/cockpit check
      - run: pnpm --dir apps/cockpit test
      - run: pnpm --dir apps/cockpit cockpit:smoke --mode local-only --mock-external
      - uses: actions/upload-artifact@v4
        with:
          name: cockpit-smoke-report
          path: apps/cockpit/smoke-report.json
```

Release pipeline mirrors the Focusa menubar model: each browser build is a tagged release.

### 3.14.1 Per-platform tagged release

Adopt the menubar CI shape directly:

```yaml
cockpit:
  name: Cockpit
  runs-on: macos-latest
  steps:
    - uses: actions/checkout@v6
    - uses: actions/setup-node@v6
      with:
        node-version: '22'
        cache: 'npm'
        cache-dependency-path: apps/cockpit/package-lock.json
    - uses: dtolnay/rust-toolchain@stable
    - name: Install dependencies
      working-directory: apps/cockpit
      run: npm ci
    - name: Typecheck
      working-directory: apps/cockpit
      run: npm run check
    - name: Web build (SvelteKit static/adapter)
      working-directory: apps/cockpit
      run: npm run build
    - name: Tauri package proof
      working-directory: apps/cockpit
      run: npm run tauri build -- --bundles app
    - name: Validate macOS bundle
      run: |
        set -euo pipefail
        APP_PATH=$(find apps/cockpit/src-tauri/target/release/bundle/macos -name "*.app" | head -1)
        test -n "$APP_PATH"
        echo "Found bundle: $APP_PATH"
        test -f "$APP_PATH/Contents/Info.plist"
        /usr/bin/plutil -lint "$APP_PATH/Contents/Info.plist"
```

Release/deploy workflow mirrors `deploy-live-daemon.yml` and `dev-release-tag.yml` patterns:

```text
1. Dev releases use the existing dev-release-tag pattern (e.g. `cockpit-v0.1.0-dev`).
2. Production releases use the tag-driven release flow: tag → build → notarize/sign → publish GitHub Release.
3. `tauri-apps/tauri-action` may later own cross-platform builds (macOS first, then Linux/Windows).
4. Auto-retry-deploy.yml may wrap the cockpit deploy step if flaky CI occurs.
5. The smoke report from §2.13 is uploaded as a release artifact and attached to the release proof.
```

Each release also produces:

```text
apps/cockpit/smoke-report.json
apps/cockpit/release-checksums.txt
apps/cockpit/release-metadata.json
```

Where `release-metadata.json` includes at minimum:

```json
{
  "schema": "uaiengine.cockpit.release.v1",
  "app": "uaiengine-cockpit",
  "version": "0.1.0",
  "channel": "stable|preview|dev",
  "tauri_version": "2.x",
  "frontend_commit": "...",
  "tauri_commit": "...",
  "signed": true,
  "notarized": true,
  "artifacts": [
    { "platform": "macos-aarch64", "path": "Focusa-Cockpit-0.1.0.dmg", "sha256": "..." }
  ],
  "smoke_report_ref": "..."
}
```

### 3.14.2 Release rules copied from menubar

- macOS first because the menubar app already proves the codesign/notarize path on that host.
- Tauri CLI `tauri build --bundles app` is the first proof gate; DMG/NSIS/DEB come later.
- macOS bundle Info.plist lint is mandatory before publish.
- Each dev tag (`cockpit-vX.Y.Z-dev`) must be reproducible from a known commit; no floating builds.
- Public/production tag must reference a green CI run that includes the cockpit job.
- Auto-retry-deploy covers flaky network/release uploads, not build correctness.

### 3.15 Phase 0 card manifest draft

Phase 0 should hardcode a small manifest first; dynamic registry generation can come after the shape is proven.

Every `focusa_*` card below is mapped to a real Spec 90 contract in
`docs/current/focusa-tool-contracts.json` (97 contracts verified
2026-06-30). Capability strings follow the proposed `focusa.<family>.<verb>` convention. Cards without a Spec 90 contract yet are flagged `adapter_only` until the contract ships.

```ts
export const phase0Cards: CardManifest[] = [
  {
    card_id: "uiai.health",
    label: "UIAI Engine Health",
    product_surface: "uiai_engine",
    authority_plane: "browser_execution",
    normative_source: "UIAI Engine /api/health/browser",
    contract_ref: null, // adapter_only until UIAI tooling registry exposes this
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["uiai.health.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "uiai.diagnostics",
    label: "Browser Diagnostics",
    product_surface: "uiai_engine",
    authority_plane: "browser_execution",
    normative_source: "UIAI Engine /api/session/{id}/diagnostics",
    contract_ref: null, // adapter_only
    required_scope: "session",
    side_effect_class: "read",
    capabilities: ["uiai.session.diagnostics.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.project_identity",
    label: "Project Identity",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 104 + Spec 90 contract focusa_project_identity",
    contract_ref: "focusa_project_identity",
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["focusa.project.identity.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.project_card",
    label: "Project Card",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_project_card",
    contract_ref: "focusa_project_card",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.project.card.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.workpoint_resume",
    label: "Workpoint Resume",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_workpoint_resume",
    contract_ref: "focusa_workpoint_resume",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.workpoint.resume.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.trajectory_view",
    label: "Trajectory View",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_trajectory_view",
    contract_ref: "focusa_trajectory_view",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.trajectory.view.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.tool_doctor",
    label: "Tool Doctor",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_tool_doctor",
    contract_ref: "focusa_tool_doctor",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.tool.doctor.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.dxux_requirement",
    label: "DXUX Requirement",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_dxux_requirement",
    contract_ref: "focusa_dxux_requirement",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.dxux.requirement.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.work_loop_status",
    label: "Work-loop Status",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_work_loop_status (parity_status=domain)",
    contract_ref: "focusa_work_loop_status",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.workloop.status.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.device_pair_status",
    label: "Device Pair Status",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_device_pair_status + Spec 53",
    contract_ref: "focusa_device_pair_status",
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["focusa.device.pair.status.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.evidence_link",
    label: "Capture Evidence",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_workpoint_link_evidence",
    contract_ref: "focusa_workpoint_link_evidence",
    required_scope: "workstream",
    side_effect_class: "local_write",
    capabilities: ["focusa.evidence.link.write"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "cloud.node_status",
    label: "Cloud Node Status",
    product_surface: "focusa_cloud",
    authority_plane: "cloud_control_plane",
    normative_source: "Spec 115 §9.2 Node Registry",
    contract_ref: null, // cloud-only; not a Spec 90 contract yet
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["node.health.read", "node.version.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "cloud_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "cloud.device_pairing",
    label: "Device Pairing",
    product_surface: "focusa_cloud",
    authority_plane: "cloud_control_plane",
    normative_source: "Spec 53 + Spec 115 §9.3",
    contract_ref: null, // cloud-only; not a Spec 90 contract yet
    required_scope: "node",
    side_effect_class: "cloud_write",
    capabilities: ["device.pair", "device.repair", "device.revoke"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "cloud_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "ai_api.health_usage",
    label: "AI API Health & Usage",
    product_surface: "ai_api",
    authority_plane: "hosted_ai",
    normative_source: "AI API health/usage endpoints (hosted)",
    contract_ref: null, // adapter-only
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["ai_api.health.read", "ai_api.usage.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
];
```

Spec 90 contract coverage matrix for Phase 0:

| card_id | contract_ref | parity_status | notes |
| --- | --- | --- | --- |
| focusa.project_identity | focusa_project_identity | full | core read |
| focusa.project_card | focusa_project_card | full | advisory read |
| focusa.workpoint_resume | focusa_workpoint_resume | full | core read |
| focusa.trajectory_view | focusa_trajectory_view | full | advisory read |
| focusa.tool_doctor | focusa_tool_doctor | full | readiness read |
| focusa.dxux_requirement | focusa_dxux_requirement | full | recovery read |
| focusa.work_loop_status | focusa_work_loop_status | domain | domain-only parity |
| focusa.device_pair_status | focusa_device_pair_status | full | pairing read |
| focusa.evidence_link | focusa_workpoint_link_evidence | full | the one Phase 0 write card |
| uiai.* | null (adapter_only) | n/a | UIAI card parity deferred to Spec 90 for UIAI |
| cloud.* | null (cloud-only) | n/a | mapped via Spec 115 capability catalogue |
| ai_api.* | null (hosted) | n/a | mapped via AI API readiness endpoints |

Audit date: 2026-06-30. Any future contract addition (e.g. focusa_sync_status, focusa_node_selector) is added to this matrix in the same PR. Phase 0 must not ship cards marked `adapter_only` without an exemption PR rationale.

Phase 0 deliberately avoids:
- arbitrary shell/code execution cards;
- proof publishing writes;
- benchmark run writes;
- ownership transfer writes;
- hidden AI-agent action cards;
- WPUIAI WordPress plugin admin cards.

### 3.16 Endpoint/capability naming rule

Every card must name the capability it needs before naming a raw endpoint.

Good:

```text
capability = focusa.workpoint.resume.read
adapter maps it to local daemon route/tool/CLI fallback
```

Bad:

```text
component calls http://127.0.0.1:8787/v1/workpoint/resume directly
```

This keeps the cockpit compatible with:
- local loopback;
- multiple local/remote nodes;
- Focusa Cloud relay;
- MCP/tool gateway parity;
- future endpoint reshaping without rewriting UI components.

### 3.17 Browser release parity with menubar

The menubar app defines the canonical desktop release contract for Focusa products. The browser cockpit must adopt it as a sibling product, not invent a parallel flow.

Shared elements:
- working directory per app: `apps/cockpit/`
- Node 22 + npm ci + `npm run check` + `npm run build`
- `npm run tauri build -- --bundles app` as first packaging step
- macOS `Info.plist` plutil lint before any publish
- dev tag pattern (`<app>-vX.Y.Z-dev`) using the menubar dev-release-tag flow
- production tag pattern with GitHub Release + checksums + smoke report
- deploy audit row appended on every release success
- self-heal chain: runner, script, workflow, audit (same as menubar)

Differences vs menubar (allowed):
- additional Linux/Windows build matrix (later, after macOS MVP)
- additional `cockpit:smoke` script in the CI job before the Tauri package proof
- additional `release-metadata.json` schema (above) attached to release notes

Forbidden:
- shipping bundles without plutil-linted Info.plist
- publishing without a smoke report artifact
- publishing without a `release-metadata.json`
- silently changing bundle IDs without updating menubar parity tests

### 3.18 Bead-to-spec anchors

All cockpit work is tracked in UIAI Engine beads (`/home/wpuiai/uiai-engine/.beads/`). The parent epic is `uiai-engine-3n3`. Bead counts and ids are intentionally not duplicated here; see `bd list --json` for ground truth.

| Slice / domain | Spec anchors |
| --- | --- |
| Epic (everything) | §0–§16 |
| Slice 0 repo/app skeleton | §3.12, §3.13, §3.14, §3.14.1, §10 |
| Slice 1 contracts and fixtures | §3.8, §3.15, §3.16 |
| Slice 2 read-only adapters | §3.6, §3.8, §3.13 |
| Slice 3 node graph + ScopeRef guard | §3.2, §3.3, §3.4, §3.10, §3.38 H1/H2 |
| Slice 4 pairing + cloud profile read | §3.5, §3.6, §3.38 H3/H5/H7 |
| Slice 5 evidence/proof preview | §3.6, §3.8 (CockpitEvent/RedactionBoundary), §3.38 H16 |
| Slice 6 beautiful shell begins | §2.1, §3.20, §6.3, §3.38 H10 |
| Release pipeline (tag/dev/release/deploy) | §3.14.1, §3.17, §3.32, §3.33 |
| Rollback / on-call / bundle integrity | §3.33, §3.34, §3.35 |
| Audit / dependency policy | §3.27, §3.36, §3.38 H19/H20 |
| Safe storage | §3.20 (keychain rule), §3.38 H5 |
| UI stack hardening | §6.4, §3.38 H10 |
| Wirebot (deferred) | §2.1, §3.6, §3.38 H14 |
| Cockpit ↔ Menubar connection planes | §17 (entire) |

Cockpit ↔ Menubar gap-resolution beads (G1–G18, all attached to `uiai-engine-3n3`):

| Gap | Spec anchors |
| --- | --- |
| G1 cloud profile auto-add parity | §17.3, §17.0 |
| G2 ScopeRef inheritance policy | §17.2, §17.13 |
| G3 multi-daemon fleet picker | §17.10, §17.17 |
| G4 bridge ownership per room | §17.6, §17.4 |
| G5 token expiry + re-pair loop | §17.11, §17.12 |
| G6 pair-status card surfaces auto-add | §17.12, §3.15 |
| G7 tool class + audit row | §17.15, §3.8 |
| G8 UserSettings.pairing_provenance | §17.13, §3.26 |
| G9 macOS sandbox + entitlements | §17.14, §3.38 H11 |
| G10 cross-platform Keychain | §17.8, §3.20 |
| G11 non-interactive CI cockpit | §17.9, §17.17 |
| G12 extended menubar Keychain entry | §17.3, §17.2 |
| G13 shared focusa-platform Keychain hint | §17.3 |
| G14 a11y for auto-add notification | §17.16, §3.22 |
| G15 operating profile lock during auto-add | §17.16, §3.5 |
| G16 card "Re-pair from menubar" affordance | §17.12, §17.16 |
| G17 cross-Mac repair propagation | §17.19 |
| G18 client_type taxonomy reserved slots | §17.10 |

Gap-closure beads (H21–H52, all attached to `uiai-engine-3n3`):

| Bead | Section | Domain |
| --- | --- | --- |
| H21 comprehensive test plan | §18.1.1 | process |
| H22 app-level rollback policy | §18.1.2 | process |
| H23 settings migration / schema versioning | §18.1.3 | process |
| H24 release notes template | §18.1.4 | process |
| H25 in-app help + first-use tour | §18.1.5 | process |
| H26 card inventory justification | §18.2.1 | functional |
| H27 global search | §18.2.2 | functional |
| H28 notifications system catalog | §18.2.3 | functional |
| H29 bottom ribbon event catalog | §18.2.4 | functional |
| H30 inspector rail content map | §18.2.5 | functional |
| H31 window management | §18.2.6 | functional |
| H32 empty-state catalog | §18.3.1 | UX |
| H33 keyboard shortcut map | §18.3.2 | UX |
| H34 loading state policy | §18.3.3 | UX |
| H35 confirmation dialogs | §18.3.4 | UX |
| H36 undo/redo | §18.3.5 | UX |
| H37 bulk actions | §18.3.6 | UX |
| H38 export/import | §18.3.7 | UX |
| H39 CI alternatives | §18.4.1 | integration |
| H40 Tailscale/Cloudflare specifics | §18.4.2 | integration |
| H41 MCP bridge role | §18.4.3 | integration |
| H42 OAuth flow for Cloud Profile | §18.4.4 | integration |
| H43 multi-tenant cockpit scope | §18.4.5 | integration |
| H44 real-time collaboration | §18.4.6 | integration |
| H45 multi-user on same Mac | §18.4.7 | integration |
| H46 per-feature gating | §18.5.1 | business |
| H47 trial experience | §18.5.2 | business |
| H48 compliance posture | §18.5.3 | business |
| H49 data retention / cleanup | §18.5.4 | business |
| H50 non-English locales | §18.6.1 | i18n |
| H51 translated error messages | §18.6.2 | i18n |
| H52 RTL and CJK layouts | §18.6.3 | i18n |

Dual-mode pairing beads (V1–V10, all attached to `uiai-engine-3n3`):

| Bead | Section | Notes |
| --- | --- | --- |
| V1 replicated menubar pairing as first choice | §17.3.1 | Path A primary; mirrors FirstRunWizard exactly |
| V2 less-friction auto-add as conditional side path | §17.3.2 | Path B fires only after Path A discovery |
| V3 cockpit standalone (menubar not on same machine) | §17.3.3 | Path A only; Path B suppressed |
| V4 cockpit standalone (menubar installed but not running) | §17.3.3 | Path A works; Path B may fire from Keychain |
| V5 operator-facing comparison UI | §17.3.4 | defaults: profile + env vars |
| V6 menubar-first-run parity test matrix | §17.3 | nightly CI integration test |
| V7 Path B dismiss + per-URL memory | §17.3.2 | 30-day TTL; settings toggle |
| V8 Path A/B unified completion state | §17.3.4 | Keychain + daemon ledger + settings converged |
| V9 cross-Mac Tailscale hint propagation | §17.3.3 | MagicDNS / Cloud device registry as fallback |
| V10 audit row for replicated pairing | §17.3 | pairing_via_replicated_flow vs pairing_via_auto_add |

Updates to a slice must reference its corresponding spec section; updates to a spec section must update the bead acceptance criteria.

### 3.19 Scope ladder and investigated decisions

This section freezes the scope stack before beads are filed.

| Scope level | Decision | Evidence / basis |
| --- | --- | --- |
| Product home | Keep the cockpit in this repo under `apps/cockpit/` unless later extracted. | Operator: “same”; UIAI Engine owns browser execution backend. |
| Product mode | Local Only and Cloud Profile are both first-class. | Spec 115 local-first promise; operator explicitly required local-only or cloud profile. |
| Backend before UI | Do not jump to polished UI until contracts/adapters/router/smoke harness exist. | Operator: “No spec only”; current §2.7–§2.16. |
| UIAI Engine API | Surface local browser/session/diagnostics/capacity primitives. | UIAI docs: `docs/UIAI_FOR_AGENTS_QUICKSTART.md`, `docs/ENDPOINT_AUTH_MATRIX.md`. |
| Focusa API / Tool Gateway | Surface scoped `focusa_*` cards through contracts, not raw singleton calls. | Focusa Specs 90, 104, 115. |
| AI API | Investigate and support dev/API-key based read-only health/usage first. | UIAI auth docs expose `UIAI_API_KEY`, bearer token, accepted auth headers; no secret values go in spec/logs. |
| Multi-node Focusa | Mac/local and VPS daemons are first-class nodes; no singleton “current Focusa”. | Specs 43/98/104; operator emphasized local+VPS reconciliation. |
| Pairing | Follow Mac menubar flow and improve it for cockpit. | `apps/menubar/src/lib/components/PairingPanel.svelte`, `FirstRunWizard.svelte`, Spec 53. |
| Secret storage | Store tokens/dev keys in OS secure storage/keychain, not localStorage. | Menubar pairing store already says secret tokens are stored in macOS Keychain via Tauri commands; localStorage only metadata/token preview. |
| Cloud endpoints | Treat Spec 115 cloud surfaces as read-only/mockable until real endpoints exist. | Spec 115 is master plan; cockpit must not invent cloud authority. |
| Deploy | Follow Focusa daemon-style discipline: CI, smoke, artifacts, proof, release/update visibility. | Focusa production deployment guide, deploy runbook, self-heal chain, Tauri `tauri-action`. |
| UI beauty | Apple-like simplicity is mandatory, but after backend contract stability. | Operator emphasis; §1.1/§5.3. |
| Wirebot | Pending/future, not Phase 0. | Operator: Wirebot integration can be pending/not top priority. |

Investigated implementation notes:
- Menubar pairing has an explicit state machine: idle → starting → waiting/completed/expired/error/revoking, plus paired-device list and revoke affordance.
- Menubar first-run wizard documents discovery priority: Tailscale MagicDNS, Bonjour/mDNS, env/localStorage hint, then advanced CLI paste fallback.
- Menubar token policy is better than raw browser storage: secret tokens via Tauri/keychain commands; localStorage only non-secret metadata and token preview.
- UIAI remote/auth docs support API-key and bearer-token patterns; cockpit should model “dev key/API key present/missing/expired” without ever displaying secret values.

Strict scope recommendations:
1. **Do not implement UI first.** Build contracts, fixtures, adapters, router, and smoke harness first.
2. **Do not make cloud mandatory.** Local Only must feel complete.
3. **Do not store dev keys/tokens in localStorage.** Use OS keychain via `tauri-plugin-keyring` (Rust crate + JS bridge) when available; fall back to encrypted local store only if the OS keychain is unavailable; localStorage may hold non-secret labels/previews only. Tauri v2's `@tauri-apps/plugin-store` is for non-secret persistent settings, not secrets.
4. **Do not let cards call endpoints directly.** Cards call controllers/adapters through capabilities.
5. **Do not flatten Mac+VPS daemons.** NodeRef + ScopeRef + thread ownership must always be visible for Focusa-backed actions.
6. **Do not ship write cards in Phase 0 except tightly scoped local evidence capture.** Start read-first.
7. **Do not surface WPUIAI plugin admin UI.** AI API only.
8. **Do not prioritize Wirebot before Phase 0 backend and cockpit shell.** Keep it compatible with the same card/scope model.

### 3.20 Cockpit shell information architecture

Apple-like shell is a hard product goal, but it must be defined before Slice 6 produces pixels.

```text
+------------------------------------------------------+
| Top scope strip                                       |
|   Operating Profile | Selected Node | ScopeRef       |
|   UIAI health       |   Focusa health |   Cloud profile |
+------+---------------------------------+-------------+
| Left | Center viewport                | Right       |
| nav  |   card grid / live view         | inspector   |
|      |                                 |             |
|      |   - UIAI browser                | - input     |
|      |   - Focusa cards                | - state     |
|      |   - Cloud profile               | - history   |
|      |   - AI API                      | - raw JSON  |
+------+---------------------------------+-------------+
| Bottom process ribbon                                 |
|   routines | decisions pending | predictions | proofs |
+------------------------------------------------------+
```

Rules:
- scope strip always visible when any Focusa or SaaS card is selected;
- left nav groups primitives by product surface (UIAI Engine, Focusa Local, Focusa Cloud, AI API);
- center viewport renders the selected primitive/card;
- inspector never blocks the viewport; it docks or pushes by design token;
- bottom ribbon only surfaces actionable events, not decorative stats;
- empty states teach, they do not sell.

First-run flow:

```text
welcome → choose Operating Profile (Local Only default)
  → if Cloud Profile: account → node status → pairing
  → if Local Only: discover Focusa daemons → pick primary node → pair (optional) → enter cockpit
```

Pairing follows menubar-first-run ordering:
1. discover local Mac daemon via Tailscale MagicDNS / Bonjour / saved connection / advanced CLI paste;
2. show QR/code on VPS (or local daemon) → scan/approve;
3. receive scoped token → store in keychain;
4. enter cockpit at selected node scope.

Discovery priority preserved from menubar:
1. Tailscale MagicDNS (recommended);
2. Bonjour / mDNS `_focusa._tcp.local` on LAN;
3. saved connection / env hint;
4. advanced CLI paste fallback.

### 3.21 Feature flags and gating

Default-on features:
- Local Only operating profile.
- UIAI Engine adapter read-only cards (health, diagnostics).
- Focusa local adapter read-only cards (project identity, workpoint resume, nodes).

Default-off features (must be explicitly enabled by user/OS):
- Cloud Profile operating profile.
- Focusa Cloud adapter read cards (node status, device pairing).
- AI API adapter read cards.
- Mutating cards (local evidence capture).
- Wirebot companion integration (deferred).
- Public proof publishing (post-Phase 0).

Flag sources (priority order, last wins):
1. compile-time `COKPIT_ENABLE_*` constants from `tauri.conf.json` or feature flags;
2. OS/package-level (channel: stable | preview | dev);
3. user setting in `lib/settings/` (non-secret).

Rules:
- flags never store secrets;
- disabling a flag must remove its card from the navigator, not just hide it with blank UI;
- adding a new online-dependent flag must be paired with an offline-behavior fixture.

### 3.22 Accessibility bar

Minimum a11y for Apple-like simplicity:
- keyboard reachable end-to-end (Tab/Shift+Tab cycle, `?` keyboard help, `cmd+K` command palette);
- visible focus rings on every interactive surface;
- ARIA roles for inspector, scope strip, process ribbon, card navigator;
- light and dark themes both meet WCAG AA contrast;
- reduced-motion respected in panel transitions;
- no status information conveyed by color alone (badge text + icon);
- error messages link to recovery actions (`recovery_action` field on `CockpitError`).

### 3.23 Observability hooks

Local-first product must still surface issues without leaking private state.

Local hooks:
- structured app log (`app://log`) with append-only stream and rotating file;
- in-app error toast for typed failures;
- breadcrumb list per session (viewModel + adapter calls, no request bodies);
- crash report handshake with optional local storage and never automatic upload.

Avoid by default:
- automatic cloud crash uploads;
- analytics beacons in Phase 0;
- screenshots bundled with diagnostics unless the operator explicitly shares them.

Public/proof boundary:
- any telemetry exported outside the cockpit must go through `RedactionBoundary` (§2.10);
- never include raw dev-key, token, or local-only path data in exported payloads.

### 3.24 Graceful failure bar

The cockpit must never crash the user's whole agent context because one card failed.

Rules:
- adapter failures are caught at `ResultNormalizer` and turned into `CockpitError`;
- a failed card never breaks neighboring cards or the shell;
- a failed node is marked offline in node graph without other nodes being affected;
- a failed pairing repair becomes a retry action, not a modal trap;
- Svelte components must render a `blocked_with_reason` view when manifest offline_behavior demands it, instead of exception bubbles.

### 3.25 Internationalization readiness

Even if Phase 0 ships English-only, internal strings must be i18n-friendly from day one to avoid a Phase 1 rewrite.

Rules:
- no raw user-facing strings inside `.svelte` files; use message keys in `lib/i18n/strings.ts`;
- adapters return `CockpitError` with stable `code`, locale-agnostic; the view-model layer maps `code` → message key → translated string;
- number/date/time formatting uses the system locale via `Intl.*` or Svelte i18n helper, never manual string concat;
- the AI API product surface displays credit and usage in the user's currency and locale once exposed.

### 3.26 Settings model

User-controlled, non-secret settings live in `lib/settings/` and are persisted via `@tauri-apps/plugin-store`.

```ts
interface UserSettings {
  schema: "uaiengine.cockpit.settings.v1";
  operating_profile: "local_only" | "cloud_profile";
  selected_node_id?: string;
  selected_scope?: {
    project_root_key?: string;
    workstream_key?: string;
    continuity_id?: string;
    thread_id?: string;
  };
  theme: "system" | "light" | "dark";
  reduced_motion: "system" | "off" | "on";
  command_palette_recent: string[]; // capability ids, not secrets
  card_filters: {
    uiai_engine: boolean;
    focusa_local: boolean;
    focusa_cloud: boolean;
    ai_api: boolean;
  };
  last_updated: string;
}
```

Rules:
- settings never contain secrets (no tokens, dev keys, pairing codes);
- settings updates emit a `CockpitEvent` of `kind: "settings_changed"`;
- reset-to-default wipes user settings but never OS keychain entries.

### 3.27 CI matrix and channel strategy

The cockpit job in `.github/workflows/ci.yml` should mirror menubar's job exactly for stable channel, plus a preview channel job.

```text
cockpit-stable:
  trigger: push to main, PR to main
  env: COCKPIT_CHANNEL=stable
  runs-on: macos-latest
  steps: same as menubar (npm ci, check, build, tauri build --bundles app, Info.plist lint)
  required: true

cockpit-preview:
  trigger: tag matching cockpit-v*.-preview
  env: COCKPIT_CHANNEL=preview
  runs-on: macos-latest + ubuntu-latest (Linux smoke)
  steps: stable steps + Linux smoke (no Tauri bundle on Linux yet)
  required: false

cockpit-smoke:
  trigger: every PR and push to main
  runs-on: ubuntu-latest
  steps: pnpm install, pnpm check, pnpm test, pnpm cockpit:smoke --mode local-only --mock-external, upload smoke-report.json artifact
  required: true
```

Channel rules:
- stable: tagged `cockpit-vX.Y.Z`, GitHub Release, macOS notarized dmg;
- preview: tagged `cockpit-vX.Y.Z-preview`, internal sharing only;
- dev: tagged `cockpit-vX.Y.Z-dev`, never published externally;
- the smoke job must pass for stable promotion.

### 3.28 Onboarding telemetry

Phase 0 stays local-first with no analytics. The cockpit must still let the operator see whether onboarding works locally.

Local-only telemetry hooks:
- `OnboardingEvent` (kinds: `welcome_shown`, `profile_chosen`, `node_selected`, `pair_started`, `pair_completed`, `pair_repair_required`, `cockpit_entered`);
- stored in `app://onboarding.jsonl` append-only on disk;
- never uploaded automatically;
- viewable via a small `view onboarding events` action in inspector for debugging UX regressions.

### 3.29 Versioning policy

The cockpit follows SemVer with explicit channel suffixes that mirror menubar.

```text
cockpit-vX.Y.Z          stable release
cockpit-vX.Y.Z-preview  preview release
cockpit-vX.Y.Z-dev      dev/internal release
```

Rules:
- `X` (major) only when a breaking change requires the operator to re-pair or re-configure (auth flow change, scope model change, file layout change);
- `Y` (minor) when a new card family or major surface is added;
- `Z` (patch) for bug fixes, dependency updates, and Polish UI fixes;
- release-metadata.json must echo the version, channel, and tagged commit;
- the in-app "About" surface shows version + channel + last update timestamp;
- auto-update visibility tells the operator when their cockpit is behind stable.

### 3.30 Performance bar

Local-first Apple-like simplicity requires fast perceived performance.

Targets for Phase 0 (measured on M-series Mac baseline):
- cold start (Tauri window → cockpit shell) < 1.5s;
- open UIAI session card < 250ms after click;
- switch operating profile < 200ms;
- inspect a node/scope < 100ms;
- smoke harness runs < 10s end-to-end on a warm cache.

Rules:
- no Svelte component performs synchronous blocking work > 50ms;
- adapters stream long operations and emit progress events rather than blocking the UI thread;
- bundle size budget: macOS .app < 80MB initial, < 120MB with default fonts/icons;
- the smoke harness must include a `--bench` flag (later) to record these numbers per build.

### 3.31 Anti-features / out-of-scope guard

The cockpit MVP must explicitly say "no" to:

- second embedded browser runtime;
- WPUIAI WordPress-plugin admin UI;
- cloud cognition authority / hosted Focus State / hosted Workpoints;
- automatic cloud upload of screenshots, diagnostics, or raw logs;
- direct fetch/curl/raw API from Svelte components (use adapters);
- hidden AI agent actions (all AI API calls visible as cards);
- automatic productivity claims ("this saved you 10 minutes");
- tunnel-style raw daemon proxy through cloud;
- silent CRDT/cognitive merges;
- shipped Phase 0 write cards beyond tightly scoped local evidence capture;
- shipping without smoke harness green;
- shipping without release-metadata.json and Info.plist lint pass.

Every PR description must state which anti-feature (if any) the change touches and why.

### 3.32 Browser live-release automation (parity with Focusa daemon)

The browser cockpit must mirror the Focusa live-release model end-to-end so the cockpit ships with the same operator confidence as the daemon.

Reference (Focusa model, not to be copied verbatim):
- `docs/live-release-automation.md`
- `.github/workflows/release.yml`
- `.github/workflows/dev-release-tag.yml`
- `.github/workflows/deploy-live-daemon.yml`
- `.github/workflows/auto-retry-deploy.yml`
- `.github/workflows/audit-recorder.yml`
- `scripts/create-dev-release-tag.sh`
- `scripts/release.sh`
- `scripts/stamp-release-version/`
- `scripts/auto-heal-audit.py`
- `scripts/audit-schema.py`
- `release-proof/audit/audit.jsonl`

Browser release policy (sibling, not duplicate):

```text
1. Edits land on main in this repo.
2. CI proves typecheck, lint, unit tests, contract validity, and cockpit smoke.
3. scripts/create-cockpit-dev-release-tag.sh --push creates cockpit-vX.Y.Z-dev.
4. .github/workflows/cockpit-release.yml:
     - gates: cockpit-stable job (macOS bund app + plutil lint + smoke),
     - creates the GitHub Release for cockpit-vX.Y.Z,
     - attaches:
         apps/cockpit/release-metadata.json
         apps/cockpit/release-checksums.txt
         apps/cockpit/smoke-report.json
         dist/Focusa-Cockpit-*.dmg (macOS notarized)
         dist/Focusa-Cockpit-*.app.tar.gz (when applicable)
5. .github/workflows/deploy-cockpit.yml:
     - runs only on self-hosted macOS runner with labels self-hosted, macos, focusa-deploy, cockpit-sign,
     - performs preflight (free disk, signing identity present, macOS notarize creds present),
     - signs + notarizes the .app + .dmg,
     - publishes artifacts to the GitHub Release,
     - attaches a release-proof row to release-proof/cockpit/audit.jsonl.
6. .github/workflows/auto-retry-deploy.yml re-dispatches cockpit deploy at most once on flaky release uploads.
7. .github/workflows/audit-recorder.yml runs scripts/auto-heal-audit.py + scripts/audit-schema.py against both daemon and cockpit audit streams.
```

Required scripts (browser sibling):

```text
scripts/create-cockpit-dev-release-tag.sh    (mirrors create-dev-release-tag.sh, tag pattern cockpit-vX.Y.Z-dev)
scripts/cockpit-release.sh                   (mirrors release.sh)
scripts/stamp-cockpit-version/               (mirrors stamp-release-version, writes version into apps/cockpit/src/lib/version.ts and apps/cockpit/src-tauri/tauri.conf.json)
scripts/auto-heal-cockpit-audit.py           (mirrors auto-heal-audit.py for cockpit rows)
scripts/deploy-smoke-check.sh                (already shared; reused for cockpit smoke)
```

Required artifacts:

```text
apps/cockpit/release-metadata.json   (schema uaiengine.cockpit.release.v1)
apps/cockpit/release-checksums.txt   (sha256 per artifact)
apps/cockpit/smoke-report.json       (schema uaiengine.cockpit.smoke.v1)
dist/Focusa-Cockpit-*.dmg            (signed, notarized)
dist/Focusa-Cockpit-*.app.tar.gz     (optional)
release-proof/cockpit/audit.jsonl    (append-only cockpit audit)
```

Required secrets/vars:

```text
secrets.FOCUSA_DEPLOY_INSTALL_ROOT        (already present, reusable for cockpit runner path)
secrets.MACOS_SIGNING_IDENTITY            (codesign identity)
secrets.MACOS_NOTARY_KEYCHAIN_PROFILE     (notarytool credentials)
secrets.GITHUB_TOKEN                      (release upload, already present)
vars.FOCUSA_DEPLOY_SERVICE_NAME           (existing, reusable label)
```

Self-heal coverage:

```text
runner:  memory cap + restart policy on self-hosted macOS runner
script:  each step has explicit recovery hints
workflow:auto-retry-deploy wraps cockpit release/deploy at most once
audit:   cockpit audit rows flow through auto-heal-cockpit-audit.py
```

Required acceptance gates before promoting a cockpit tag from dev → preview → stable:

```text
- cockpit-stable job green (typecheck, build, tauri build --bundles app, plutil lint, smoke)
- contract validator green (validateCardManifest)
- scope guard tests green (Mac/VPS distinguish, scope_conflict blocks writes)
- pairing flow test green (token via OS keychain, no localStorage secrets)
- smoke-report.json uploaded as release artifact
- release-metadata.json sha256 matches release-checksums.txt
- Info.plist plutil lint pass for every macOS artifact
- audit row appended with kind=cockpit_release_published
```

Cross-product invariants:

```text
- cockpit release reuses the daemon self-heal runner attributes (no second fleet)
- cockpit audit stream lives alongside the daemon audit stream but never merges with daemon state
- cockpit artifacts are signed with the Focusa project signing identity so dev/repo trust alignment is preserved
- cockpit release tag never reuses a daemon tag lane; cockpit uses cockpit-vX.Y.Z, daemon uses vX.Y.Z
```

PR requirements:

```text
- PR description declares if it touches the browser release pipeline
- PR description declares which anti-feature list (§2.17.12) item, if any, it touches
- PR description lists which scripts and workflows were added or modified
- PR includes:
    smoke-report.json diff (or note "no smoke-affecting change")
    release-metadata.json diff (or note "no release-affecting change")
    stamp-cockpit-version diff (or note "no version change")

### 3.33 Cockpit release rollback runbook

A bad cockpit tag must be reversible without breaking the operator's local cockpit.

```text
Detect:
  - in-app "Update available" badge not advancing
  - crash handshake firing > 0.5% within 1h of release
  - smoke-report upload missing sha256 in release-checksums.txt
  - plutil lint fails after publish (caught late)

Decide:
  - minor patch (cosmetic): ship a follow-up cockpit-vX.Y.(Z+1) patch
  - functional regression: pin recommended tag to last known good cockpit-vX.Y.Z
  - signing/notarize leak: revoke the dmg via Apple notary history and publish a NEW tag

Recover:
  1. Trigger workflow_dispatch on cockpit-release.yml with the previous good tag.
  2. Append release-proof/cockpit/audit.jsonl row kind=cockpit_release_reverted.
  3. Open a follow-up PR with the bad release marked in CHANGELOG.md.
  4. Notify operators via the in-app ribbon that an update is recommended.
  5. If the broken release signed with the Focusa signing identity, file Apple ticket + rotate identity (post-MVP).

Prevent:
  - stable promotion requires green cockpit-stable + green cockpit-smoke
  - dev/preview tags never touch signing identity (signed only at stable)
  - the audit row is the source of truth for "what shipped when"
```

### 3.34 On-call note for cockpit release

Until the cockpit has its own SaaS-side telemetry, the operator still has to answer one question quickly: "is the latest cockpit safe to upgrade to?". The on-call runbook:

```text
1. Open cockpit-release workflow in GitHub Actions for the suspected tag.
2. Confirm:
   - cockpit-stable job green
   - cockpit-smoke job green
   - release-metadata.json sha256 matches release-checksums.txt
   - audit row appended within 60s of release publish
3. If any check fails:
   - escalate using the daemon self-heal contact list (re-use existing Focusa on-call)
   - trigger rollback via cockpit-release.yml workflow_dispatch with previous good tag
   - post status in the cockpit Discord release channel
```

### 3.35 Proof/bundle integrity contract

Every public cockpit artifact must be checkable from a single source of truth.

```json
{
  "schema": "uaiengine.cockpit.bundle_manifest.v1",
  "app": "uaiengine-cockpit",
  "version": "0.1.0",
  "channel": "stable",
  "tag": "cockpit-v0.1.0",
  "frontend_commit": "...",
  "tauri_commit": "...",
  "signed": true,
  "notarized": true,
  "artifacts": [
    {
      "name": "Focusa-Cockpit-0.1.0.dmg",
      "platform": "macos-aarch64",
      "sha256": "...",
      "signature": "...",
      "notarization": "..."
    }
  ],
  "smoke_report_ref": "apps/cockpit/smoke-report.json",
  "release_metadata_ref": "apps/cockpit/release-metadata.json",
  "audit_row_ref": "release-proof/cockpit/audit.jsonl#<row>"
}
```

Rules:
- the manifest sha256 is recomputed by `scripts/release.sh` and the operator's `cockpit doctor` action;
- mismatched sha256 = rolled-back tag (force operator to choose previous good tag);
- notarization credentials live only on the self-hosted macOS runner.

### 3.36 Dependency and supply chain policy

The cockpit must inherit Focusa's pinned, vendored, signature-checked dependency stance.

```text
- npm + pnpm lockfile checked into git, frozen for CI installs.
- no floating caret/tilde versions in package.json without a policy note.
- @tauri-apps/* pinned to a known compatible major.
- Svelte 5 + SvelteKit pinned.
- Bits UI / shadcn-svelte pinned (or copied source-owned under lib/ui/components/).
- node_modules is NEVER shipped inside the .app; Tauri build outputs a clean bundle.
- npm registry downloads are verified against lockfile + optional integrity policy.
- If a future supply-chain attack is detected, the affected pinned version is yanked, audit row appended, dev/preview/stable retested.
```

PR rule: any dependency bump must include the lockfile diff, a why-change note, and a smoke harness run with the bumped dependency.

### 3.37 Hard-parts watchlist

This section is the operator's truth-detector for places where the spec currently asserts direction without pinning decisions. Each item below is either resolved in a follow-up edit, or explicitly deferred.

| # | Hard part | Status | Required decision / next step |
| --- | --- | --- | --- |
| H1 | Multi-node CRDT conflict UX | not yet decided | Define UI states: write_blocked_canonical_conflict, ownership_transfer_requested, observation_attached_to_local. Add CockpitEvent kinds for each. |
| H2 | ScopeRef propagation path | not yet wired | Decide single source of truth (active session > UserSettings.selected_scope > explicit card-level confirmation). Add getScopeForCard(cardId) helper contract. |
| H3 | First-run keychain bootstrap | partial | Decide cold-install flow, keychain-locked flow, and retry-after-fail flow. Pair with menubar FirstRunWizard ordering. |
| H4 | AI API auth posture | partial | Decide dev-key acquisition story (signup link, in-app purchase, or trial-only). Surface license state, key expiry, and rate-limit headers in the AI API card. |
| H5 | Local-host daemon trust | not yet decided | Decide how cockpit proves identity to local UIAI/Focusa daemons. Candidates: keychain-bound token, macOS code-signature attestation, mTLS, or localhost-only with explicit user confirm. |
| H6 | Tauri auto-update flow | not yet wired | Adopt tauri-plugin-updater; require signed update manifest; document emergency disable path; show operator-visible update available and updated-restart-to-apply states. |
| H7 | Cloud profile consent UX | not yet designed | Every cloud-side effect must surface a consent dialog before sending, with human_message, payload summary, and reject option. Cloud cannot act silently even for read cards that include operator identity. |
| H8 | Failure recovery semantics | partial | Define behaviors for: UIAI daemon unreachable mid-session, Focusa daemon crash mid-write, cloud API timeout mid-receipt, AI API rate-limited. Map each to typed CockpitError.recovery_action. |
| H9 | Performance budget measurement | not yet wired | Decide CI vs release-only benchmark; add regression gate; add operator-visible perf badge in inspector for the active session. |
| H10 | Apple-like visual tokens | not yet designed | Author lib/ui/design-tokens.css: corner radii, shadow scale, motion easing, spacing scale, type ramp, iconography rules. Provide before/after mocks for at least one card. |
| H11 | macOS signing/notarization specifics | partial | Pin team ID, hardened runtime, entitlements, notarization profile name, staple behavior, signing identity reuse policy. Document in docs/cockpit-signing.md. |
| H12 | SvelteKit static-adapter edge cases | not yet captured | Provide canonical svelte.config.js + vite.config.ts + tauri.conf.json example. Cover CSP, asset base path, prerender, deep links, custom protocol handler. |
| H13 | Local telemetry viewer | partial | Add an inspector surface for app://log and app://onboarding.jsonl so the operator can read what the cockpit is doing without external tools. Redact secrets automatically. |
| H14 | Wirebot future integration contract | deferred | When revived, Wirebot must consume CardManifest and ScopeRef, emit CockpitEvent, and never bypass adapters. Add explicit no-parallel-authority-plane guard. |
| H15 | Mac + VPS simultaneous writes | not yet designed | If operator opens two cockpits (Mac + VPS) on the same workstream, define who wins, how ownership transfers, and how UI reflects the race. |
| H16 | Proof publish cancellation | partial | If the operator cancels a proof publish mid-flight, define rollback semantics (e.g. snapshot draft retained, server-side artifact abandoned, audit row of kind=proof_publish_aborted). |
| H17 | Onboarding regression testing | not yet wired | Add a Playwright flow that replays the first-run flow and asserts deterministic ScopeRef/NodeRef state. |
| H18 | Multi-platform telemetry parity | deferred | macOS-only cockpit telemetry is acceptable for MVP; Linux/Windows telemetry must wait for those targets. |
| H19 | Open-source license compatibility | not yet decided | Confirm shadcn-svelte / Bits UI / tauri-plugin-keyring licenses are compatible with Focusa source-available license. |
| H20 | Dependency update SLAs | not yet decided | How fast do we ship a patch if a critical npm advisory lands? Decide on-call paging and patch SLA for cockpit vs daemon. |

Rule:

```text
Hard-parts watchlist items are first-class spec concerns. They are not "nice-to-have"
polish. Each gets a bead, an owner, and a yes/deferred decision before Slice 6 ships.
```

Next action: convert the open items into P1/P2 beads and resolve them in priority order.

### 3.38 Hard-parts resolutions (this iteration)

#### H5 — Local-host daemon trust

Decision (2026-06-30):

```text
Loopback default = keychain-bound token only.
For sensitive operations (pairing writes, ownership transfer, evidence capture writes), require:
  1. macOS code-signature attestation of the cockpit binary (signature must include the Focusa Team ID),
  2. keychain-bound token (token id bound to the binary's signing identifier),
  3. optional mTLS only if available on the local daemon side (Focusa daemon v0.10+).
```

Implementation sketch:

```ts
async function proveLocalIdentity(daemon: NodeRef): Promise<LocalIdentityProof> {
  const sig = await invoke("tauri_attest_cockpit_signature");
  const token = await keychainRead(`cockpit.token.${daemon.machine_id}`);
  return { sig, token, daemon_endpoint: daemon.endpoint };
}
```

Failure mode:

```text
- invalid signature → daemon returns 403 + typed CockpitError "untrusted_local_caller"
- missing token → keychain bootstrap flow opens
- token expired → re-pair flow opens
- attestation fails to load → only allow read cards; block all writes
```

Audit:

```text
- every local-host call appends a CockpitEvent of kind=local_identity_proven or kind=local_identity_rejected
- never include raw token values in events
```

#### H7 — Cloud profile consent UX

Decision (2026-06-30):

```text
Every Focusa Cloud or AI API call must pass through a ConsentDialog before send.
The dialog shows:
  - capability being invoked,
  - one-line human summary of what is sent and what is NOT sent,
  - "Trust this card for the rest of the session" toggle (per-card session grant),
  - Reject / Allow buttons.
```

Consent state lives in `UserSettings.card_consent[card_id]`:

```ts
interface UserSettings {
  ...
  card_consent: {
    [card_id: string]: "ask" | "session_grant" | "denied";
  };
}
```

Implementation rules:

```text
- "ask" → ConsentDialog shows every invocation,
- "session_grant" → skipped for the current session,
- "denied" → card renders blocked_with_reason and recovery_action=open_consent.
- consent decisions are stored in non-secret settings (no tokens),
- cloud consent is the only place where "Trust this card" appears; local-host reads do not.
```

#### H6 — Tauri auto-update flow

Decision (2026-06-30):

```text
Adopt @tauri-apps/plugin-updater v2 with the Focusa project signing identity.
Update manifest must be signed with the same identity used for the dmg.
The updater endpoint URL is fixed at install time (config); switch to a different
endpoint requires a new install.
Emergency disable env: FOCUSA_COCKPIT_DISABLE_UPDATER=1 (also stored in settings).
```

Operator-visible states:

```text
- "Update available: cockpit-vX.Y.Z (current: cockpit-vX.Y.Z-dev) → Download"
- "Downloading... 65%"
- "Ready to install: cockpit-vX.Y.Z → Restart now"
- "Up to date"
- "Update disabled by environment variable"
```

Smoke harness additions:

```text
- mock_update_manifest.json fixture with a future version
- mock_signed_update_signature.bin fixture
- asserts the updater accepts signed manifests and rejects unsigned ones
- asserts emergency disable env var disables the updater
```

#### H1 — Multi-node CRDT conflict UX

Decision (2026-06-30):

```text
If two cockpits (Mac + VPS) target the same workstream, the local Focusa CRDT module
owns reconciliation. The cockpit UI surfaces reconciliation state but never merges
cognition. New CockpitEvent kinds:

  - canonical_write_observed     // another node wrote canonical
  - canonical_write_conflict     // two nodes tried to write simultaneously
  - ownership_transfer_requested  // non-owner requested ownership
  - ownership_transfer_required  // current owner must approve

UI states (per NodeRef row):

  - "up to date"
  - "backlog: 3 events"
  - "conflict: view detail"
  - "ownership transfer: approve or reject"
  - "read-only: not thread owner"
```

Card-level behavior:

```text
- local writes blocked when status != "up to date"
- read cards always allowed but display "stale" badge if backlog > 0
- pairing/ownership transfer surface a confirm dialog with thread id and remote node name
```

#### H3 — First-run keychain bootstrap

Decision (2026-06-30):

```text
First-run flow (Local Only):
  1. welcome
  2. choose profile (default: Local Only)
  3. discover local Focusa daemon via Tailscale MagicDNS → Bonjour → saved → CLI paste (menubar ordering preserved)
  4. optional: pair to daemon via pairing card
  5. write pairing token to keychain via tauri-plugin-keyring
  6. enter cockpit

Cold-install edge case:
  - keychain locked → show platform-native unlock prompt; do not bypass
  - keychain empty → step 5 runs; keychain entry created on success
  - keychain access denied → block write cards; show recovery_action=open_keychain_settings
  - first write failure → retry-with-backoff (3 attempts); on persistent failure, open repair/re-pair flow
```

#### H4 — AI API auth posture

Decision (2026-06-30):

```text
- Phase 0 supports developer API key only.
- Acquisition story: dev key is acquired from Focusa Cloud portal (not part of cockpit MVP).
- Cockpit reads the dev key from keychain (entry: `cockpit.ai_api.dev_key`).
- AI API card surfaces:
    - license tier (from /api/usage response),
    - key expiry (if exposed by AI API),
    - rate-limit headers (X-RateLimit-Remaining, Retry-After),
    - recovery_action when rate-limited or key invalid.
- The dev key is NEVER displayed in plain text; only its label/prefix is shown.
- If dev key acquisition flow is later in-app, the user is redirected to Focusa Cloud portal
  (not implemented in cockpit MVP).
```

#### H10 — Apple-like visual tokens

Decision (2026-06-30) — initial token set:

```css
:root {
  --radius-card: 12px;
  --radius-button: 8px;
  --radius-input: 8px;
  --shadow-card: 0 1px 2px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.08);
  --shadow-overlay: 0 8px 32px rgba(0,0,0,0.18);
  --motion-fast: 120ms;
  --motion-normal: 220ms;
  --motion-slow: 360ms;
  --easing-standard: cubic-bezier(0.2, 0.0, 0, 1);
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --font-display: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Inter", system-ui;
  --font-body: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Inter", system-ui;
  --type-ramp-display: 28/32;
  --type-ramp-title: 18/24;
  --type-ramp-body: 14/20;
  --type-ramp-caption: 12/16;
  --color-bg: #ffffff;
  --color-surface: #f6f6f7;
  --color-text: #1c1c1e;
  --color-text-muted: #6b6b70;
  --color-accent: #0a84ff;
  --color-success: #30d158;
  --color-warn: #ff9f0a;
  --color-error: #ff453a;
  --color-focus-ring: rgba(10,132,255,0.45);
}
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #1c1c1e;
    --color-surface: #2c2c2e;
    --color-text: #f2f2f7;
    --color-text-muted: #98989f;
    --color-accent: #0a84ff;
  }
}
@media (prefers-reduced-motion: reduce) {
  :root {
    --motion-fast: 0ms;
    --motion-normal: 0ms;
    --motion-slow: 0ms;
  }
}
```

Rule:

```text
Phase 0 components must consume only design tokens. No raw hex/px/ms values inline.
A lint rule fails the build on raw token violations.
```

#### H11 — macOS signing/notarization specifics

Decision (2026-06-30) — pinned values go to docs/cockpit-signing.md:

```text
Team ID:               <filled at provisioning time>
Hardened runtime:      true
Code-signing identity: Developer ID Application: <name> (<TEAMID>)
Notarization:          notarytool with App Store Connect API key
                       Keychain profile: focusa-notary
Staple:                true (stapler staple -m dmg)
Bundle identifier:     com.focusa.cockpit
Minimum macOS:         12.0
Entitlements:
  com.apple.security.app-sandbox:               true
  com.apple.security.network.client:            true
  com.apple.security.files.user-selected.read-write: true
  com.apple.security.application-groups:         (none in MVP)
Hardened entitlements:
  com.apple.security.cs.allow-jit:               false
  com.apple.security.cs.allow-unsigned-executable-memory: false
Signing identity reuse policy:
  - Daemon and cockpit share the Developer ID Application identity.
  - Mac App Store identity is reserved for App Store builds (not MVP).
  - Self-hosted runner must present the identity and the notary keychain profile.
```

#### H12 — SvelteKit static-adapter edge cases

Decision (2026-06-30) — canonical configs land in apps/cockpit:

```js
// apps/cockpit/svelte.config.js
import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: "build",
      assets: "build",
      fallback: "index.html",
      precompress: false,
      strict: true,
    }),
    prerender: { entries: [] }, // SPA: no prerender
  },
};
```

```ts
// apps/cockpit/vite.config.ts
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: { port: 1420, strictPort: true },
  build: { target: "es2022", sourcemap: true },
});
```

```json
// apps/cockpit/src-tauri/tauri.conf.json (relevant blocks)
{
  "build": {
    "frontendDist": "../build",
    "devUrl": "http://localhost:1420",
    "beforeDevCommand": "npm run dev",
    "beforeBuildCommand": "npm run build"
  },
  "app": {
    "windows": [
      { "label": "main", "title": "UIAI Engine Cockpit", "width": 1280, "height": 800, "resizable": true, "decorations": true, "fullscreen": false }
    ],
    "security": {
      "csp": "default-src 'self'; img-src 'self' asset: data:; style-src 'self' 'unsafe-inline'; connect-src ipc: http://ipc.localhost https://api.focusa.dev http://127.0.0.1:7456 http://127.0.0.1:8787"
    }
  },
  "plugins": {
    "updater": {
      "active": true,
      "endpoints": ["https://api.focusa.dev/cockpit/updates/{{target}}/{{arch}}/{{current_version}}"],
      "pubkey": "<FOCUSA_UPDATER_PUBLIC_KEY>",
      "windows": { "installMode": "passive" }
    }
  },
  "bundle": {
    "macOS": { "minimumSystemVersion": "12.0", "hardenedRuntime": true, "entitlements": "entitlements.plist" }
  }
}
```

#### H8 — Failure recovery semantics

Decision (2026-06-30):

```text
- UIAI daemon unreachable mid-session:
    CockpitError code = "uiai_unreachable"
    recovery_action = "retry" or "open_health"
    Card state = "degraded"
    Other cards keep working; the offline card shows retry-with-backoff (1s, 2s, 5s).

- Focusa daemon crash mid-write:
    CockpitError code = "focusa_crash_mid_write"
    recovery_action = "open_logs" and "repair_pairing"
    Card state = "blocked_with_reason"
    Local evidence draft is preserved as a draft; the write is NOT retried automatically.
    Audit row kind=focusa_write_lost appended with scope and target node.

- Cloud API timeout mid-receipt:
    CockpitError code = "cloud_timeout_mid_receipt"
    recovery_action = "retry"
    Card state = "degraded"
    Receipt draft remains local; resync attempted on next online tick.

- AI API rate-limited:
    CockpitError code = "ai_api_rate_limited"
    recovery_action = "wait" with Retry-After value displayed
    Card state = "blocked_with_reason" until window passes.
```

Audit rule:

```text
Each scenario appends a CockpitEvent with the matching kind and a typed CockpitError.
The event includes the scope, node, card_id, and the recovery_action taken.
```

#### H9 — Performance budget measurement

Decision (2026-06-30):

```text
- Performance gate runs on every PR that touches apps/cockpit/.
- Cold start, FPV open, profile switch, inspect latency measured via a headless harness
  (no UI window; loads the same SvelteKit bundle under Playwright).
- Regression gate: any metric > 1.2x of the last stable baseline fails the PR.
- Operator-visible perf badge: small monospace field in the inspector showing last-measured
  metrics for the active session (cold start, FPV open, last card latency).
- Baseline file: apps/cockpit/perf-baseline.json (updated on stable release).
```

#### H13 — Local telemetry viewer

Decision (2026-06-30):

```text
- Inspector includes a "Logs" tab listing the last N (default 200) CockpitEvents.
- A separate "Onboarding" tab lists OnboardingEvents.
- Both tabs perform a final secret redaction pass before render.
- A "Copy as redacted JSON" action exists for sharing with the cockpit team.
- A "Clear local telemetry" action requires explicit confirmation.
```

#### H15 — Mac + VPS simultaneous writes

Decision (2026-06-30):

```text
- If two cockpits (Mac + VPS) target the same workstream:
    1. The thread owner is the canonical writer.
    2. Non-owner cockpit renders write cards as "blocked_with_reason" + recovery_action=open_ownership_transfer.
    3. The owner cockpit shows a banner: "Another cockpit is connected. Approve its transfer?"
    4. Owner clicks Approve/Reject. Rejection closes the dialog; approval writes the ownership-transfer event.
- CRDT sync status is always visible in the bottom ribbon.
- Cross-node writes without ownership transfer are rejected with typed CockpitError.
```

#### H16 — Proof publish cancellation

Decision (2026-06-30):

```text
- Operator cancels mid-flight:
    Server-side artifact is abandoned (best-effort DELETE if server supports it).
    Draft is retained locally.
    Audit row kind=proof_publish_aborted appended with operator_id, scope, target node.
- Cancellation is idempotent; a subsequent cancel returns the same audit row.
- The card UI returns to draft state with a "Retry publish" action.
```

#### H17 — Onboarding regression testing

Decision (2026-06-30):

```text
- Add tests/onboarding.spec.ts using @playwright/test.
- The flow replays: welcome → choose profile → discover node → pair (mocked) → enter cockpit.
- Asserts: ScopeRef.authority_state == "verified"; NodeRef.health == "ok"; OnboardingEvent "cockpit_entered" present.
- The spec runs on every PR touching apps/cockpit first-run code.
- Mocked keychain adapter is required so the test does not touch the real OS keychain.
```

#### H19 — Open-source license compatibility

Decision (2026-06-30):

```text
- shadcn-svelte: MIT — compatible.
- Bits UI: MIT — compatible.
- tauri-plugin-keyring: Apache-2.0 / MIT — compatible.
- @tauri-apps/plugin-store: MIT — compatible.
- @tauri-apps/plugin-updater: MIT — compatible.
- Svelte / SvelteKit: MIT — compatible.
- All copy-in components live under apps/cockpit/lib/ui/components/ as owned source.
- No copyleft (GPL/AGPL) dependencies permitted in MVP.
- License audit (npm-license-checker or equivalent) runs in CI and fails on copyleft.
```

#### H20 — Dependency update SLAs

Decision (2026-06-30):

```text
- Critical CVE in any runtime dependency: patch within 24 hours.
- High CVE: patch within 7 days.
- Medium: 30 days.
- Low: next scheduled cockpit release.
- Audit script: apps/cockpit/scripts/audit-deps.mjs (npm audit --production + license + age).
- On-call paging reuses Focusa daemon on-call list.
- Audit row kind=cockpit_dep_patch appended on every patch.
```

Resolution summary:

```text
Resolved in this iteration: H1, H3, H4, H5, H6, H7, H8, H9, H10, H11, H12, H13, H15, H16, H17, H19, H20.
Already resolved (deferred): H14, H18.
Each resolution bead is now closable when its corresponding implementation lands.
```
### 3.39 Backend-first implementation slices

Ship the boring middle in small vertical slices.

#### Slice 0 — repo/app skeleton

Deliverables:
- `apps/cockpit` SvelteKit/Tauri scaffold;
- static adapter / SPA Tauri config (`frontendDist=build`);
- package scripts: `check`, `test`, `cockpit:smoke`, `cockpit:smoke --mode local-only`, `cockpit:smoke --mode cloud-profile`;
- no polished UI beyond a plain status page.

Acceptance gates:
- `npm ci` succeeds with frozen lockfile in CI;
- `npm run check` passes typecheck and lint;
- `npm run build` produces static `build/` directory;
- `npm run tauri build -- --bundles app` produces a Tauri bundle;
- macOS `Info.plist` passes `plutil -lint`;
- `cockpit:smoke` exits 0 even when external adapters are mocked offline;
- no Svelte component or Tauri command bypasses the `lib/contracts/` types;
- README documents how to run `cockpit:smoke` against UIAI + local Focusa when available.

#### Slice 1 — contracts and fixtures

Deliverables:
- TypeScript interfaces from §2.8 in `lib/contracts/`;
- fixture data for Local Only, Cloud Profile, Mac node, VPS node, missing scope, stale scope, expired token, repair-required token;
- static Phase 0 card manifest in `lib/cards/phase0-card-manifest.ts`;
- `validateCardManifest(manifest, contracts)` guard.

Acceptance gates:
- `validateCardManifest` is invoked at build time via Vite plugin or smoke harness;
- `pnpm cockpit:test --contracts` asserts every Phase 0 card has required fields, capability map, and offline behavior;
- fixtures are usable from smoke runner without network access;
- no card references raw endpoints directly (lint rule).

#### Slice 2 — read-only adapters

Deliverables:
- UIAI Engine health/diagnostics adapter;
- Focusa local discovery + project identity/workpoint-read adapter;
- AI API health/usage adapter (dev key via `tauri-plugin-keyring`);
- typed error envelope mapping per §2.8 `CockpitError`.

Acceptance gates:
- `cockpit:smoke --mode local-only` passes when UIAI + local Focusa are reachable;
- `cockpit:smoke --mode local-only --mock-external` passes offline without warnings;
- AI API adapter surfaces `missing_credential` typed error, never an empty 401;
- all adapter errors map to `CockpitError` with `human_message` and `recovery_action`;
- no adapter writes raw HTML or stack traces into the UI.

#### Slice 3 — node graph + ScopeRef guard

Deliverables:
- node graph store (Map<machine_id, NodeRef>);
- node/daemon selector logic with authority-role awareness;
- ScopeRef resolver returning `verified | missing | stale | conflict | read_only`;
- blocked-write proof for missing/stale/conflicting scope;
- Mac/VPS dual-node fixture;
- `selectNode(nodeId)` and `setScope(scope)` actions guarded by tests.

Acceptance gates:
- `cockpit:test --scope` covers verified/missing/stale/conflict/read-only paths;
- attempting a write card against a non-owner node returns `blocked_node_role` typed error;
- attempting a write with missing scope returns `select_scope` recovery action;
- cross-node writes (Mac daemon pretending to write VPS thread) are rejected.

#### Slice 4 — pairing state + cloud profile read

Deliverables:
- pairing state model derived from menubar `PairingPanel` state machine;
- local/cloud profile token state display;
- read-only Focusa Cloud node heartbeat adapter;
- repair/re-pair state fixtures and adapter returns;
- `pairing_action()` reducer for `idle | starting | waiting | completed | expired | error | revoking | repair_required`.

Acceptance gates:
- pairing flow never persists tokens in localStorage;
- cloud profile reads account/node status without uploading Workpoint/Focus State/raw evidence (verified via contract test);
- revoked/expired token produces `repair_required` state with `re_pair` recovery action;
- Tauri commands only mutate OS keychain via `tauri-plugin-keyring`;
- no logs/diagnostics include raw token values.

#### Slice 5 — evidence/proof preview

Deliverables:
- local evidence capture path for browser artifact summary (`focusa.evidence.capture.write`);
- proof preview card with redaction/public-safety gate;
- typed error for any public publish attempt before redaction status is `public_safe`;
- no public publish yet.

Acceptance gates:
- `cockpit:smoke --evidence` captures screenshot + diagnostics summary as local evidence ref;
- proof publish preview displays redaction status, secret-scan status, and `publish_allowed` flag;
- attempting public publish when `publish_allowed=false` returns `proof_publish_blocked` typed error;
- no public URL is generated in Phase 0.

#### Slice 6 — beautiful shell begins

Deliverables:
- Apple-like Svelte/Tauri app shell (top scope strip / left nav / center viewport / right inspector / bottom ribbon per §2.17.1);
- first-run flow per §2.17.1;
- card grid + node selector + scope strip + process ribbon;
- cards consume only view models from the backend middle;
- empty states teach, not sell;
- light/dark themes from day one.

Acceptance gates:
- `pnpm cockpit:test --view-models` enforces that Svelte components only import from `lib/cards` and `lib/router` view models;
- Local Only mode shows real value without cloud sign-in;
- Cloud Profile mode is reachable only via explicit user action, not a banner/upsell;
- accessibility: keyboard navigation, focus rings, ARIA labels for inspector/ribbon;
- design tokens centralized in `lib/ui/design-tokens.css`.

---

## 4. Core decision

Treat the desktop app as an **operator cockpit**, not a generic consumer browser.

Primary job:
1. show browser state clearly;
2. let the operator intervene deliberately;
3. return control back to automation cleanly;
4. keep evidence, diagnostics, and workflow context visible.

---

## 5. Reuse from current system

The desktop direction is feasible because major primitives already exist.

### 4.1 UIAI Engine already provides

- browser sessions;
- browser read / snapshot / click / fill / press / eval / screenshot / diagnostics;
- FPV share surfaces (`mirror_url`, `status_url`, `screenshot_url`);
- health / metrics / session APIs;
- agent-facing tooling already used in Pi.

### 4.2 Focusa already provides

- project context and continuity;
- Workpoint, Trajectory, Evidence, Prediction, Metacog, and DXUX primitives;
- Spec 43 CRDT / multi-device sync primitives;
- Spec 104 typed scope/context authority;
- Spec 90 tool contracts for the `focusa_*` Pi tool surface;
- desktop/Tauri precedent via the existing menubar app;
- pairing / saved connection / runtime panel patterns.

### 4.3 Focusa Cloud / Spec 115 can provide

- account, license, node, device, and SSH identity surfaces;
- pairing, relay, proof receipt, benchmark, preload receipt, and support workflows;
- Focusa Tool Gateway / MCP parity policy for `focusa_*` cards;
- Code Capsule routing metadata for local-first execution cards;
- SaaS dashboard state that the browser can interact with while local nodes retain authority.

### 4.4 AI API can provide

- hosted AI actions for cards that need AI API capabilities;
- usage / entitlement state where exposed by the API;
- public API interactions separate from the WPUIAI WordPress plugin.

---

## 6. Product shape

### 6.1 Base app

The base desktop app is a **UIAI Operator Browser**.

Base responsibilities:
- session list;
- selected-session live view;
- action controls;
- diagnostics panes;
- artifact timeline;
- capacity/health indicators;
- share and handoff controls.

### 6.2 Integration model

The app supports integration surfaces similar to lightweight extensions.

#### Focusa integration surface
- project/trajectory/workpoint context;
- evidence capture;
- prediction / diagnosis helpers;
- handoff and continuity panels;
- `focusa_*` Pi tool cards generated from tool contracts.

#### Focusa Cloud / SaaS integration surface
- account, license, node, and device cards;
- connect/pairing and relay status cards;
- Tool Gateway / MCP client cards;
- preload, proof receipt, and benchmark cards;
- support-bundle and audit-summary cards.

#### AI API integration surface
- AI API health, usage, and entitlement cards where exposed;
- hosted AI action cards that complement local UIAI/Focusa execution;
- no WPUIAI WordPress-plugin admin surface in the browser MVP.

#### Wirebot interaction surface — pending
- not a Phase 0 priority;
- later ask/talk-to-Wirebot companion panel;
- later visible handoff between human, agent, and browser session;
- card suggestions remain operator-confirmed before mutation;
- same ScopeRef/node/session display as Focusa cards;
- evidence/proof capture for Wirebot-assisted actions where applicable.

The base app must remain useful **without** cloud sign-in or AI API credentials; local UIAI browser operation stays valuable offline.

### 6.3 Design quality bar

The app must be beautiful and simple before it is broad.

Design rules:
- one obvious primary action per card;
- native-feeling window chrome, keyboard shortcuts, and permissions;
- progressive disclosure: summary first, inspector details second, raw JSON/logs last;
- calm status language: Connected, Needs pairing, Scope unclear, Read-only, Running, Proof ready;
- never show raw daemon jargon as the first explanation when a human-readable reason exists;
- make dangerous actions visually distinct without making the whole app feel alarming;
- preserve trust through visible source labels: Local Node, Focusa Cloud, UIAI Engine, AI API, and later Wirebot.

### 6.4 Recommended MVP UI stack

Research-backed recommendation:

```text
Runtime shell: Tauri v2
Frontend: SvelteKit 2 + Svelte 5
Rendering mode: SPA/static adapter for Tauri
Design system base: Svelte-native components, starting with shadcn-svelte
Primitive layer: Bits UI
Styling: Tailwind CSS + CSS variables/design tokens
Icons: Svelte-native Lucide-style line icons
Charts: Svelte-native chart layer only where needed
Build/release: GitHub Actions + tauri-apps/tauri-action
```

Rationale:
- Tauri gives the native desktop shell without replacing UIAI Engine as browser execution backend.
- Svelte/SvelteKit keeps the UI fast, small, and pleasant for app-like desktop work.
- Tauri's SvelteKit guidance requires static-adapter / SPA behavior because Tauri does not support server-based frontend solutions inside the app; use `build/` as `frontendDist`.
- Choose what integrates best with Svelte/Tauri; do not force React/core shadcn patterns when a Svelte-native path is cleaner.
- shadcn-svelte provides beautiful, customizable, open-code Svelte components and a large component inventory useful for dashboards and cockpit workflows.
- Bits UI provides accessible, headless Svelte primitives with full styling control; it is the right primitive layer when Apple-like polish matters more than a pre-baked theme.
- Tailwind/CSS variables let the cockpit create a Focusa-specific visual language instead of looking like a generic admin template.
- `tauri-action` builds native binaries for macOS, Linux, and Windows and can publish bundles to GitHub Releases.

Selection rule:
- Prefer the Svelte-native library/component that best fits the cockpit's needs.
- shadcn-svelte is the initial visible component layer because it integrates directly with Svelte and gives fast, beautiful MVP velocity.
- Use Bits UI directly when custom behavior or accessibility primitives matter more than pre-composed components.
- Use other Svelte-native libraries only when they integrate better for a specific component.

Alternatives considered:
- Skeleton is a strong Tailwind adaptive design-system option with turnkey components; keep it as inspiration/reference, but not the MVP base because the cockpit needs tighter custom Apple-like control.
- Flowbite Svelte has many ready Svelte/Tailwind components; keep it as fallback/reference for commodity controls, but not the MVP base because its Flowbite visual language is more generic SaaS/admin than bespoke desktop cockpit.
- Bits UI alone is excellent for primitives, but shadcn-svelte gives faster MVP velocity by providing a composed component layer on top of accessible primitives.

Avoid for MVP:
- heavy admin templates that make the product feel generic;
- WordPress/WPUIAI plugin UI components;
- a second embedded browser runtime;
- visual libraries that fight Svelte/Tauri or make custom Apple-like polish difficult.

### 6.5 MVP component map

Use Svelte-native components only. Start with shadcn-svelte as the visible layer, back it with Bits UI primitives when custom behavior is needed, and substitute another Svelte-native component only when it is a better fit.

| Cockpit need | Recommended component pattern |
| --- | --- |
| App shell | Sidebar + resizable panels + top scope strip + bottom process ribbon |
| Primitive/card navigator | Sidebar, command palette, tabs, badge, tooltip |
| Card grid | Card, button, badge, progress, skeleton, empty state |
| Node/scope selector | Combobox/select, command, breadcrumb, popover |
| Pairing flow | Dialog/sheet, QR/code block, progress, alert, copy button |
| Browser controls | Button group, input, tooltip, dropdown menu, kbd |
| Diagnostics | Tabs, table/data table, accordion, scroll area, alert |
| Evidence/proof timeline | Card list, separator, badge, progress, hover card |
| SaaS/account state | Avatar, dropdown, badge, alert, sheet |
| Dangerous actions | Alert dialog with explicit scope/node summary |
| First-run onboarding | Step cards, empty state, dialog/sheet, copyable commands |

Visual defaults:
- translucent/macOS-inspired surfaces only where they improve hierarchy;
- light/dark themes from day one;
- small refined status pills instead of noisy banners;
- soft shadows, subtle borders, and high text contrast;
- keyboard-first command palette for power users;
- no raw JSON as default UI.

### 6.6 Card contract

A browser card is the user-facing representation of one bounded primitive. Cards may represent UIAI browser actions, Focusa local tools, Focusa Cloud control-plane actions, or AI API actions.

Each card should declare:

```text
card_id
label
product_surface: uiai_engine | focusa_local | focusa_cloud | ai_api | wirebot
normative_source: spec/doc/contract path
required_scope: none | project | workstream | thread | session | node | team
authority_plane: browser_execution | local_node | cloud_control_plane | hosted_ai
side_effect_class: read | local_write | cloud_write | code_capsule | proof_publish | benchmark_run
device_token_required
thread_owner_required
offline_behavior
receipt_behavior
proof_behavior
redaction_boundary
visual_states: idle | loading | blocked_scope | blocked_auth | running | done | degraded | failed
```

Rules:
- `focusa_*` cards must derive from Spec 90 tool contracts where a contract exists.
- Local write cards must show scope, target node, thread role, and approval state before execution.
- Cloud cards must show what is cloud-owned versus node-owned before execution.
- Code Capsule cards must show command preview, file/network boundary, timeout, and artifact/evidence output.
- Proof/benchmark cards must show redaction and public-safety state before publish.

### 6.7 Phase 0 card set

Phase 0 should prove the cockpit can be useful without broad scope creep.

Recommended first cards:

| Card | Surface | Why first |
| --- | --- | --- |
| UIAI session open/reuse | UIAI Engine | Core browser primitive. |
| UIAI diagnostics | UIAI Engine | Makes browser failures visible. |
| Focusa project identity | Focusa local | Establishes ScopeRef and prevents cross-project drift. |
| Focusa workpoint resume | Focusa local | Shows current mission/next action. |
| Focusa tool doctor | Focusa local | Explains degraded Focusa/tool readiness. |
| Focusa evidence capture | Focusa local | Turns browser/action results into durable proof refs. |
| Focusa node selector | Focusa local/cloud | Makes Mac/VPS/multi-daemon routing explicit. |
| Focusa sync/topology | Focusa local/cloud | Shows CRDT health, thread ownership, and observation/proposal state. |
| Focusa Cloud node status | Focusa Cloud | Proves SaaS interaction without private cognition upload. |
| Focusa Cloud pairing/device status | Focusa Cloud | Makes browser/client trust visible. |
| AI API health/usage | AI API | Shows hosted AI availability/entitlement without involving WPUIAI plugin UI. |
Phase 0 acceptance must include at least one live/read-only card from each API plane:
- UIAI Engine API: session health or diagnostics card;
- Focusa API / Tool Gateway: project identity or workpoint resume card;
- AI API: health, usage, or entitlement card.

Phase 0 must also prove both operating profiles:
- Local Only: UIAI + local Focusa cards operate without cloud sign-in;
- Cloud Profile: cloud profile card shows account/node/pairing state without taking local cognition authority.

Wirebot is intentionally deferred from Phase 0. Its future integration should reuse the same card/scope/authority model instead of introducing a parallel interaction model.

### 6.8 Desktop deploy pipeline

The browser/desktop cockpit needs a deploy pipeline with the same level of operator confidence as the Focusa daemon pipeline: explicit artifacts, signed/verifiable releases, smoke checks, rollback guidance, proof, and self-heal/runbook hooks.

Pipeline shape:

```text
commit
  ↓
CI checks: typecheck, lint, unit tests, contract/card manifest validation
  ↓
SvelteKit SPA/static build: adapter-static, frontendDist=build/
  ↓
Tauri build: tauri-action matrix for macOS first, then Linux/Windows
  ↓
sign/notarize where platform requires it
  ↓
generate checksums, release metadata, and changelog/proof notes
  ↓
publish preview/stable artifacts through GitHub Releases
  ↓
run smoke checks against UIAI Engine API, Focusa API/Tool Gateway, and AI API read-only cards
  ↓
publish release proof / deployment receipt
  ↓
show update availability inside cockpit
```

Required deploy states in the cockpit:
- current app version and channel;
- latest available version and release channel;
- artifact verification status;
- pairing/token compatibility status;
- API compatibility status for UIAI Engine, Focusa, Focusa Cloud, and AI API;
- last successful smoke/proof receipt;
- rollback/reinstall command or link.

The deploy pipeline must not deploy or mutate user nodes by surprise. Updates may install the cockpit client; local Focusa node upgrades remain explicit Focusa install/update flows governed by Focusa installer/license docs.

### 6.9 Pairing model

Browser/desktop pairing should follow the Mac menubar app precedent: Apple-like, explicit, portable, and three-party by default.

Primary flow:

```text
Connect Cockpit to Focusa

[ Cockpit-generated QR / pairing code ]

Scan or approve from Focusa Connect
```

Requirements:
- the cockpit is a first-class device type (`browser_desktop` / `desktop_cockpit`), not a reused generic token;
- pairing uses the Focusa Connect page / pairing room pattern where available;
- QR/code pairing is portable across local Mac, VPS, BYO tunnel, and Focusa Cloud relay contexts;
- pairing records device id, device name, node id, machine id, granted scopes, token expiry, and audit trail;
- first connection is read-first; mutation grants are explicit and visible;
- the UI must support revoke, repair, re-pair, and token rotation flows;
- pairing state is shown per node, not globally, because one cockpit may connect to Mac and VPS daemons at once;
- pairing failures must distinguish environment mismatch, unreachable daemon, expired token, revoked device, scope mismatch, and cloud relay denial.

---

## 7. Operator modes

The desktop app should make session ownership explicit.

### 6.1 Observe
- read-only;
- FPV + screenshots + diagnostics;
- no session mutation.

### 6.2 Assist
- operator can send bounded actions;
- agent remains primary driver;
- action log records human interventions.

### 6.3 Take control
- operator directly controls the session;
- automation is paused or gated;
- clear visual state change required.

### 6.4 Return control
- operator exits direct control;
- app records what changed;
- agent/runtime can resume from updated state.

This mode model is a core safety boundary, not an optional UX detail.

---

## 8. UX surfaces

### 7.1 Main viewport
- live mirror / FPV view;
- current URL/title/session id;
- quality and latency indicators;
- optional annotation / inspection overlays later.

### 7.2 Action rail
- open/reuse/close session;
- navigate;
- screenshot;
- diagnostics capture;
- click/fill/press/eval controls;
- share / copy mirror link.

### 7.3 Observability rail
- console errors;
- failed requests;
- diagnostics summaries;
- accessibility / DOM snapshot summaries;
- recent actions and outcomes.

### 7.4 Integration panels
- Focusa local-authority panel;
- Focusa node/topology panel;
- Focusa sync/reconciliation panel;
- Focusa Cloud / SaaS panel;
- AI API panel;
- optional future product-specific panels;
- no WPUIAI WordPress-plugin panel in the browser MVP.

---

## 9. API and transport expectations

The desktop app should prefer **existing UIAI Engine surfaces** first for browser execution, and **Focusa Cloud / Spec 115 surfaces** for SaaS interactions.

### 9.1 Required current surfaces
- UIAI Engine API: session open/close/navigate/control;
- UIAI Engine API: screenshot and diagnostics endpoints;
- UIAI Engine API: FPV share / mirror endpoints;
- UIAI Engine API: health and capacity status;
- Focusa API / Tool Gateway: ScopeRef / Context Authority status for every Focusa or SaaS-backed card;
- Focusa API / Tool Gateway: node registry / daemon discovery status for every connected local or remote Focusa node;
- Focusa API / Tool Gateway: CRDT sync status and thread ownership status for every scoped workstream shown in the browser;
- AI API: health, usage, entitlement, and hosted-AI action metadata where exposed.

### 9.2 Likely additions
- richer session event stream for actions + diagnostics + artifacts;
- explicit session ownership/lock endpoint;
- operator audit trail endpoint;
- card/panel metadata surface for integrations;
- Focusa Tool Gateway card manifest derived from tool contracts;
- Focusa Cloud client endpoints for account, node, device, relay, proof, benchmark, and preload receipt interactions;
- AI API client metadata for authenticated hosted-AI cards;
- multiplexed routing metadata for selecting node, daemon, thread, session, and authority role per card action.

### 9.3 Avoid initially
- inventing a separate desktop-only browser protocol;
- embedding a second execution engine if UIAI Engine can stay authoritative;
- hiding session ownership behind ambiguous UI;
- hiding node/thread ownership behind a single global “current Focusa” state;
- surfacing the WPUIAI WordPress plugin as a first-party browser panel;
- bypassing Spec 104 scope checks for convenience;
- letting cloud or browser UI perform silent CRDT/cognitive merges.

---

## 10. Implementation path

### 10.1 Phase A — desktop shell over today's APIs
- Tauri desktop shell;
- connect to UIAI Engine;
- show FPV/mirror/screenshot/diagnostics;
- expose bounded action controls;
- add health/capacity visibility;
- show a ScopeRef/scope-status strip before any Focusa-backed action;
- show selected Focusa node/daemon and authority role for any Focusa-backed card.

### 10.2 Phase B — bidirectional operator flow
- operator mode switching;
- structured scoped action history;
- pause/resume / return-control semantics;
- better artifact/event timeline.

### 10.3 Phase C — extension-style card model
- Focusa local-authority card contract;
- Focusa multi-node/topology card contract;
- Focusa Cloud / SaaS card contract;
- AI API card contract;
- pluggable actions / panel registration;
- optional saved workflow packs.

### 10.4 Phase C.5 — multi-node reconciliation visibility
- node graph / daemon selector;
- thread ownership display;
- CRDT sync/backlog/conflict indicators;
- scoped proposal inbox for non-owner observations;
- route every mutating card through the selected authoritative node.

### 10.5 Phase D — deeper desktop affordances
- global hotkeys;
- floating mini-view / tray mode;
- local notifications;
- richer annotation/inspection overlays.

---

## 11. Non-goals

- replacing UIAI Engine as the browser backend;
- shipping a full general-purpose end-user browser;
- making Focusa the browser product itself;
- making Focusa Cloud the local authority for private cognition;
- flattening multiple Focusa nodes into one implicit global daemon;
- surfacing WPUIAI WordPress-plugin admin/workflow UI in the browser MVP.

---

## 12. Open questions

1. What is the minimal event stream needed for the desktop UI to feel live without polling too much?
2. How should session ownership be represented on the server: lock, lease, mode flag, or command queue?
3. Should Focusa, Focusa Cloud, and AI API cards be server-declared manifests, desktop-loaded modules, or hardcoded first-party panels at first?
4. Does FPV stay as the render surface for desktop MVP, or should the app attach to a richer internal stream when available?
5. What capacity constraints in UIAI Engine (page pool, queue depth, diagnostics cost) need explicit operator feedback before this can feel trustworthy?
6. What is the minimal node/thread/session selector needed so a Mac daemon, VPS daemon, and cloud relay can reconcile visibly without confusing the operator?
7. Which CRDT/sync fields should be first-class UI states versus drill-down diagnostics?

---

## 13. Edit guidance

When updating this spec:
- preserve the architecture boundary unless intentionally changing it;
- record whether a change affects **base UIAI app**, **Focusa local integration**, **Focusa Cloud/SaaS integration**, or **AI API integration**;
- prefer editing existing sections over creating parallel conflicting docs;
- if this spec supersedes the older Mac-app subsection in the FPV co-pilot doc, note that explicitly in a future revision.


---

## 14. Acceptance criteria index

This section consolidates the pre-UI, Slice 0–6, hard-parts, and release-acceptance gates so the cockpit MVP can be claimed "done" against a single checklist.

### 14.1 Pre-UI backend gates (§3.11 smoke matrix)

- [ ] Local Only smoke covers UIAI health, UIAI diagnostics, Focusa discovery, ScopeRef resolve, workpoint read.
- [ ] Cloud Profile smoke covers account, node heartbeat, relay status without uploading cognition.
- [ ] Multi-node routing distinguishes Mac + VPS nodes and rejects cross-write attempts.
- [ ] Scope conflict blocks writes with `select_scope` recovery action.
- [ ] Evidence capture produces local evidence ref for a browser artifact summary.
- [ ] Proof preview renders redaction + public-safe gate; no public publish yet.
- [ ] Pairing repair returns the right repair action, not a generic failure.

### 14.2 Slice gates

- [ ] Slice 0 (§3.39) — apps/cockpit scaffold installs, typechecks, builds, tauri builds app, plutil lint passes.
- [ ] Slice 1 — `validateCardManifest` runs in CI; Phase 0 manifest validates against contracts.
- [ ] Slice 2 — read-only adapters pass with `--mode local-only --mock-external` and against real daemons when reachable.
- [ ] Slice 3 — ScopeRef + node-graph tests cover verified/missing/stale/conflict/read-only states.
- [ ] Slice 4 — pairing never stores tokens in localStorage; cloud profile reads return repair_required when revoked/expired.
- [ ] Slice 5 — proof preview shows redaction status; public publish is blocked without `publish_allowed=true`.
- [ ] Slice 6 — Svelte components import only from `lib/cards` + `lib/router` view models; Local Only mode feels complete.

### 14.3 Hard-parts gates (§3.38)

Each resolved hard part (H1–H20) has a corresponding acceptance gate landed in code, tests, or docs. Implementation beads:

| H | Resolution section | Acceptance landed when |
| --- | --- | --- |
| H1 | §3.38 H1 — multi-node CRDT UX | conflict UX events emitted + tested |
| H2 | §3.38 H2 — ScopeRef propagation path | `getScopeForCard` helper used + lint enforced |
| H3 | §3.38 H3 — first-run keychain bootstrap | cold/locked/denied flows tested |
| H4 | §3.38 H4 — AI API auth posture | license state + rate-limit header surfaced |
| H5 | §3.38 H5 — local-host daemon trust | signature+keychain proof verified; untrusted caller rejected |
| H6 | §3.38 H6 — Tauri auto-update flow | signed manifest enforced; emergency disable verified |
| H7 | §3.38 H7 — cloud profile consent UX | consent dialog blocks silent cloud actions |
| H8 | §3.38 H8 — failure recovery semantics | 4 scenarios emit typed errors + audit rows |
| H9 | §3.38 H9 — performance budget | 1.2× regression gate green in CI |
| H10 | §3.38 H10 — Apple-like visual tokens | lint rule fails on raw token violations |
| H11 | §3.38 H11 — signing/notarization | docs/cockpit-signing.md lands; release refuses without notary creds |
| H12 | §3.38 H12 — SvelteKit static-adapter | canonical configs land; reproducible builds |
| H13 | §3.38 H13 — local telemetry viewer | inspector surfaces redacted events |
| H14 | (deferred) | Wirebot integration contract enforced when revived |
| H15 | §3.38 H15 — Mac + VPS simultaneous writes | ownership transfer dialog tested |
| H16 | §3.38 H16 — proof publish cancellation | cancellation path appends audit row |
| H17 | §3.38 H17 — onboarding regression testing | Playwright suite green in CI |
| H18 | (deferred) | multi-platform telemetry deferred to those targets |
| H19 | §3.38 H19 — license compatibility | license audit passes in CI; no copyleft |
| H20 | §3.38 H20 — dependency update SLAs | on-call policy + audit row on patch |

### 14.4 Release gates (§3.32, §3.33)

- [ ] cockpit-stable job green (typecheck/build/tauri build/plutil lint/smoke).
- [ ] validateCardManifest green.
- [ ] scope guard tests green.
- [ ] pairing flow test green (keychain only).
- [ ] smoke-report.json attached as release artifact.
- [ ] release-metadata.json sha256 matches release-checksums.txt.
- [ ] audit row kind=`cockpit_release_published` appended.
- [ ] rollback runbook reproducible from a single workflow_dispatch.
- [ ] bundle manifest `uaiengine.cockpit.bundle_manifest.v1` re-validates on operator machine.

### 14.5 Operator-visible acceptance

- [ ] Cold start < 1.5s on M-series baseline (§3.30).
- [ ] UIAI health card < 250ms after click.
- [ ] Operating-profile switch < 200ms.
- [ ] Inspect latency < 100ms.
- [ ] Smoke harness < 10s end-to-end warm cache.
- [ ] Bundle size < 80MB initial / < 120MB with fonts+icons.

### 14.6 Cockpit ↔ Menubar bridge gates (§17)

- [ ] Auto-add detects menubar Keychain entries and inherits `daemon_url` (G12, G13).
- [ ] `focusa://` reserved for menubar; `cockpit://` registered by cockpit (G4).
- [ ] Bridge ownership per room; only one listener per nonce (G4).
- [ ] Bridge envelope preserves Spec 104 MBN-01 ScopeContext end-to-end.
- [ ] Cloud Profile auto-add parity registered in Spec 115 §9.3 device registry (G1).
- [ ] Multi-daemon fleet picker shown when menubar has >1 pair; ★ marker on active (G3).
- [ ] Token expiry triggers "Refresh from menubar" affordance (G5).
- [ ] `focusa_device_pair_status` Phase 0 card surfaces menubar+cockpit pair state (G6).
- [ ] Every auto-add appends audit row kind=`auto_pair_observed_via_menubar` (G7).
- [ ] `UserSettings.pairing_provenance` persists across restart (G8).
- [ ] macOS hardened runtime + sandbox allow both apps to read shared Keychain entries (G9).
- [ ] Linux/Windows use Secret Service / Credential Manager via tauri-plugin-keyring (G10).
- [ ] CI runner pairs silently when env vars set; `AUTO_PAIR_DENY=1` overrides (G11).
- [ ] Auto-add notification passes axe-core / WAVE / manual screen reader test (G14).
- [ ] Local Only mode never auto-pairs; explicit operator action required (G15).
- [ ] Every Phase 0 pair-status card exposes "Re-pair from menubar" button (G16).
- [ ] Cross-Mac repair via `/v1/sync/events` kind=`pairing_refreshed` (G17).
- [ ] Device ledger schema reserves `client_type` slots for `menubar|cockpit|pi|claude|codex|cursor|opencode|mcp|ssh_identity|ci` (G18).

### 14.7 Gap closure acceptance gates (cross-cutting §18)

All §18.7 gap-closure gates must pass before the cockpit MVP is claimed "done":

- [ ] Test plan, rollback policy, settings migration, release notes, in-app help (H21–H25).
- [ ] Card inventory, search, notifications, ribbon, inspector rail, window management (H26–H31).
- [ ] Empty states, shortcuts, loading states, confirmation dialogs, undo/redo, bulk actions, export/import (H32–H38).
- [ ] CI alternatives, Tailscale/Cloudflare, MCP bridge, OAuth, multi-tenant scope, real-time collab, multi-user same Mac (H39–H45).
- [ ] Per-feature gating, trial experience, compliance, data retention (H46–H49).
- [ ] Locales, translated errors, RTL+CJK (H50–H52).

Each gate corresponds 1:1 to a bead (H21–H52) tracked in `.beads/`.

### 14.8 Dual-mode pairing acceptance gates (V1–V10)

- [ ] V1: Path A (replicated menubar pairing) implemented end-to-end; same step ordering, Tailscale hosts, Bonjour service, env var, CLI paste fallback, bridge command names, payload schema, and Keychain service pattern as menubar.
- [ ] V2: Path B (less-friction auto-add) only fires after Path A has discovered a daemon URL and the menubar Keychain hint matches; Add path skips bridge + phone + VPS pair flow.
- [ ] V3: Cockpit on a Mac with no menubar install runs Path A successfully; Path B is suppressed (no Keychain hint).
- [ ] V4: Cockpit on a Mac with menubar installed but not running runs Path A; Path B may fire from Keychain; bridge ownership respects per-nonce ownership.
- [ ] V5: Pairing panel shows the two paths side-by-side; defaults: local_only suppresses Path B; cloud_profile surfaces Path B; AUTO_PAIR_DENY=1 forces Path A; AUTO_PAIR_QUIET=1 auto-accepts Path B; AUTO_PAIR=1 explicit CI opt-in for Path B without TTY check.
- [ ] V6: Nightly integration test covers menubar + cockpit + daemon in three roles.
- [ ] V7: Per-URL Path B dismiss persists 30 days; settings panel exposes re-enable auto-add suggestions toggle.
- [ ] V8: After either path, Keychain has cockpit token under UIAI Engine Cockpit Token; daemon ledger has cockpit device row; UserSettings.pairing_provenance is set.
- [ ] V9: Mac B cockpit can resolve menubar's pairing via daemon's devices list after Mac B pairs; no Mac-A-specific Keychain needed.
- [ ] V10: Both paths append the appropriate CockpitEvent with audit row schema validated.

Each gate corresponds 1:1 to a bead (V1–V10) tracked in `.beads/`.

---

## 15. Bead index

Parent epic and all cockpit child beads live in this repo at `.beads/`. See §3.18 for the spec-anchor mapping. New beads must reference a §3.x anchor and vice versa.

Approximate counts (refreshed per release):
- Epic: 1 (`uiai-engine-3n3`)
- Slices: 7 (Slice 0–6)
- Release automation: 4 (scripts + workflows)
- Audit / rollback / integrity: 4
- Hard-parts: 18 (H1–H20 minus H14 and H18 already tracked elsewhere)
- Plus: UI hardening, secret policy, deferred Wirebot

Counts change as beads close; this index is informative, not authoritative.

---

## 18. Gap closure plan (every prior gap)

This section closes every gap identified during the iterative review. Every sub-section is an implementable rule with a corresponding bead and acceptance gate.

### 18.1 Process completeness

#### 18.1.1 Comprehensive test plan

Cockpit test layers:

| Layer | Tool / framework | Coverage target | Owner |
| --- | --- | --- | --- |
| Unit | Vitest (Svelte/TS) | 80% of `lib/contracts`, `lib/cards`, `lib/router`, `lib/store` | CI |
| Adapter contract | Vitest + mock fixtures | 100% of `ApiAdapter` calls per plane | CI |
| Card contract | custom `validateCardManifest` + Vitest | 100% of `phase0Cards` | CI |
| Smoke | Bun/Node script in `apps/cockpit/smoke/` | §3.11 matrix + §14.6 bridge gates | CI per PR |
| E2E | Playwright on SvelteKit static build | first-run, profile switch, pair-status card, auto-add prompt, repair flow | CI nightly |
| Load | Playwright + Vitest benchmark | cold start <1.5s, smoke harness <10s | release gate |
| Security | `npm audit --production`, `pip-audit` for Python helpers, `osv-scanner` | zero critical/high | CI |
| Compatibility | Tauri matrix (macos-latest, ubuntu-latest, windows-latest) | build green + smoke green | release gate |

Required test files:

```
apps/cockpit/tests/
  unit/
    contracts.test.ts
    cards.test.ts
    router.test.ts
    store.test.ts
    settings.test.ts
    telemetry.test.ts
    a11y.test.ts
  adapter/
    uiai-engine-adapter.test.ts
    focusa-local-adapter.test.ts
    focusa-cloud-adapter.test.ts
    ai-api-adapter.test.ts
  card/
    phase0-card-manifest.test.ts
  e2e/
    first-run.spec.ts
    profile-switch.spec.ts
    pair-status-card.spec.ts
    auto-add.spec.ts
    refresh-from-menubar.spec.ts
    repair.spec.ts
  perf/
    cold-start.bench.ts
    smoke.bench.ts
```

Acceptance gate (added to §14.7):
- [ ] All test layers green in CI.
- [ ] Vitest coverage ≥ 80% on lib/contracts/lib/cards/lib/router/lib/store.
- [ ] E2E suite covers first-run + auto-add + repair paths.
- [ ] Performance benchmark runs on every release tag.

#### 18.1.2 App-level rollback policy

| Channel | Auto-update? | Rollback? | Recovery path |
| --- | --- | --- | --- |
| stable | opt-in (operator toggles) | yes (updater blocklist + prior good tag) | `cockpit doctor --rollback-stable-to <tag>` |
| preview | opt-in | yes (manual downgrade via release picker) | `cockpit doctor --rollback-preview-to <tag>` |
| dev | default on | yes (last known good tag, no signature required) | `cockpit doctor --rollback-dev-to <tag>` |

Rules:
- Rollback never crosses major versions (operator must manually uninstall).
- A "Last known good" entry is appended to `release-proof/cockpit/audit.jsonl` after every successful install.
- The in-app "About" surface exposes the last-10 tags so operator can pin to one.
- A bad tag is reverted by appending `kind=cockpit_release_reverted` audit row.
- No silent auto-rollback on dev channel; explicit operator click.

Acceptance gate:
- [ ] `cockpit doctor --rollback-*-to` works in CI smoke harness.
- [ ] Stable blocklist file is documented and honored.
- [ ] Last known good audit row written for every successful install.

#### 18.1.3 Settings migration / schema versioning

Schema versioning for UserSettings:

```ts
interface UserSettingsV1 { schema: "uaiengine.cockpit.settings.v1"; /* ... */ }
// future: UserSettingsV2 { schema: "uaiengine.cockpit.settings.v2"; /* ... */ }
```

Rules:
- Every `UserSettings` schema carries a `schema` field with semver-like version.
- A migration function per schema bump runs at app startup before settings take effect.
- Migrations are additive when possible; destructive only when the operator has explicitly approved.
- The settings panel shows the current schema version and warns before destructive migration.
- A backup of the previous settings is written to `app://settings.<old-schema>.bak.json` before applying.

Acceptance gate:
- [ ] Schema version field present in every UserSettings write.
- [ ] Migration function exists per schema bump; tested.
- [ ] Backup is written before applying destructive migrations.

#### 18.1.4 Release notes template

Stable releases must include:

```markdown
# cockpit-vX.Y.Z

## Highlights
- (1–3 lines)

## Operator-visible changes
- (bulleted, plain English)

## Breaking changes
- (none / list with mitigation)

## Required actions
- (none / list with explicit steps)

## Compatibility
- UIAI Engine: >=X.Y.Z
- Focusa daemon: >=X.Y.Z
- macOS: >=X.Y

## Proof
- smoke-report.json (link)
- bundle-manifest.json (link)
- audit row (link)
```

Template ships at `apps/cockpit/templates/RELEASE_NOTES.template.md`; CI substitutes fields from release-metadata.json.

Acceptance gate:
- [ ] Every stable release contains all template sections.
- [ ] Release notes link smoke-report.json + bundle-manifest.json + audit row.

#### 18.1.5 In-app help / first-use tour

Rules:
- The cockpit ships a help button (`?` shortcut) that opens a help overlay.
- Help overlay lists keyboard shortcuts, common actions, and links to docs.focusa.dev.
- First-use tooltip tour (skippable) walks through: scope strip, navigator, viewport, inspector, ribbon.
- Help docs are vendored inside the app (`apps/cockpit/lib/help/`); not loaded from network.
- Tour state persists in UserSettings (`tour.completed_at`).
- All help text uses message keys from `lib/i18n/strings.ts`.

Acceptance gate:
- [ ] `?` opens help overlay; Esc closes.
- [ ] First-use tour appears once per `tour.completed_at` missing.
- [ ] Help content is vendored and offline-capable.

### 18.2 Functional completeness

#### 18.2.1 Card inventory justification

Phase 0 manifest picks these 9 cards (all bound to Spec 90 contracts verified 2026-06-30):

| Card | Contract | Why Phase 0 |
| --- | --- | --- |
| focusa.project_identity | focusa_project_identity | Core read; establishes ScopeRef. |
| focusa.project_card | focusa_project_card | Advisory read. |
| focusa.workpoint_resume | focusa_workpoint_resume | Core read; shows mission + next action. |
| focusa.trajectory_view | focusa_trajectory_view | Advisory read; shows HLT/MLG/STG. |
| focusa.tool_doctor | focusa_tool_doctor | Readiness read. |
| focusa.dxux_requirement | focusa_dxux_requirement | Recovery read. |
| focusa.work_loop_status | focusa_work_loop_status | Domain-parity read. |
| focusa.device_pair_status | focusa_device_pair_status | Pairing read. |
| focusa.evidence_link | focusa_workpoint_link_evidence | Phase 0's one write card. |

Deferred to Phase 1+ with reason:

| Contract | Reason for deferral |
| --- | --- |
| focusa_trajectory_checkpoint | write-state; needs thread-authority guard; Phase 1 |
| focusa_workpoint_checkpoint | write-state; same |
| focusa_metacog_capture | local write; needs i18n lesson schema; Phase 1 |
| focusa_predict_record | local write; needs prediction schema; Phase 1 |
| focusa_predict_evaluate | read; Phase 1 |
| focusa_context_cognition | read; large; needs scope guards; Phase 1 |
| focusa_preload_verify | adapter-only (no Spec 90 contract yet) |
| focusa_sync_status | adapter-only (no Spec 90 contract yet) |
| focusa_node_selector | adapter-only (no Spec 90 contract yet) |

Acceptance gate:
- [ ] Phase 0 manifest matches the 9 cards listed.
- [ ] Every deferred card is annotated with reason + Phase target.
- [ ] Each Phase 0 card has a real Spec 90 contract_ref or is marked `adapter_only` / `cloud-only` / `hosted`.

#### 18.2.2 Search across cockpit

Rules:
- `cmd+K` opens a global search overlay.
- Search covers: card inventory, settings, telemetry events, evidence refs, workpoints, trajectories.
- Results respect scope: only data the active ScopeRef can read.
- Search is local-first; no upload to cloud.
- Search uses a simple prefix index built at app start (≤ 200ms warm).

Acceptance gate:
- [ ] cmd+K opens overlay.
- [ ] All listed result categories covered.
- [ ] Latency <200ms after warm.

#### 18.2.3 Notifications system

Event catalog surfaced via in-app notifications + ribbon updates:

| Event | Severity | Where | Origin |
| --- | --- | --- | --- |
| pairing_refreshed | info | ribbon + inspector | truth plane |
| pairing_repaired | info | ribbon + inspector | truth plane |
| pairing_revoked | warn | modal | truth plane |
| pairing_expiring_soon | info | ribbon | local timer |
| sync_backlog_growing | warn | ribbon | local check |
| ai_api_rate_limited | warn | inline card | adapter |
| evidence_capture_succeeded | info | ribbon | adapter |
| evidence_capture_failed | warn | inline card + ribbon | adapter |
| workpoint_drift_detected | warn | inline card | adapter |
| cockpit_update_available | info | tray + ribbon | updater |
| cockpit_update_ready | info | modal | updater |
| cockpit_emergency_disable | warn | modal | env var |
| auto_pair_observed_via_menubar | info | modal + ribbon | menubar bridge |

Every notification has:
- `kind`
- `human_message` (i18n key)
- `recovery_action` (when applicable)
- `audience` (operator / agent / system)
- `expires_at` (default 24h)

Acceptance gate:
- [ ] Every event kind in the catalog is implementable.
- [ ] Each event surfaces in the correct location.
- [ ] All human messages are i18n-keyed.

#### 18.2.4 Bottom ribbon event catalog

Bottom ribbon shows (in priority order):
1. Unresolved errors (red badge, count)
2. Active card actions (running)
3. Pending audit rows (count)
4. Unread pair-status warnings (badge)
5. Pairing expiring soon (within 7 days)
6. Update available (count)
7. Decisions pending (count)
8. Predictions awaiting eval (count)

Ribbon state machine:

```ts
type RibbonState =
  | { kind: "idle" }
  | { kind: "busy"; active_actions: number }
  | { kind: "warning"; warnings: RibbonWarning[] }
  | { kind: "error"; errors: RibbonError[] };
```

Acceptance gate:
- [ ] Ribbon surfaces each item from the catalog.
- [ ] Ribbon state has a typed schema.
- [ ] Tapping a ribbon item opens the relevant inspector/craft panel.

#### 18.2.5 Inspector rail content

Inspector rail content per selected primitive:

| Primitive | Inspector content |
| --- | --- |
| Card | inputs (locked by scope), state, history (last 20), redacted JSON, evidence refs |
| Node | endpoint, machine_id, transport, health, sync_state, authority_role, last sync |
| Scope | project_root_key, workstream_key, continuity_id, thread_id, session_id, role, authority_state |
| Pairing device | device_id, client_type, scopes, expires_at, paired_at, source (auto/manual) |
| Evidence ref | content_url, scope, captured_at, redaction_status |
| Proof preview | receipt_id, redaction_status, publish_allowed |

Acceptance gate:
- [ ] Every primitive has its inspector rail content list.
- [ ] Inspector never displays secrets.

#### 18.2.6 Window management

Rules:
- macOS: dock icon hidden (menubar-only behavior for the menubar app; cockpit is a regular app).
- Cockpit may show a menubar accessory icon (optional, default OFF in Phase 0).
- Window size: 1280×800 default, resizable.
- Multiple windows: optional Phase 1.
- Picture-in-picture: optional Phase 1.
- Tray icon: out of scope for cockpit MVP (menubar owns the tray).

Acceptance gate:
- [ ] Default window size enforced.
- [ ] Dock icon policy honored on macOS.
- [ ] Multi-window and PiP documented as Phase 1.

### 18.3 UX completeness

#### 18.3.1 Empty-state catalog

Each empty state must be specific, teach, and offer a next action.

| Empty state | Copy | Next action |
| --- | --- | --- |
| No daemons discovered | "We couldn't find a Focusa daemon. Pick how to connect." | (Tailscale / Bonjour / CLI / env) buttons |
| No node selected | "Pick a Focusa node to start." | opens node picker |
| No workpoint | "No active workpoint. Create one." | opens focusa_workpoint_checkpoint (Phase 1) |
| No evidence yet | "Run a card to capture evidence." | highlights first card |
| No notifications | "All clear." | (none) |
| No search results | "Nothing matched '<query>'." | suggests scopes |
| No devices paired | "Pair your first device." | runs pair flow |

Every empty state copy lives in `lib/i18n/strings.ts`.

Acceptance gate:
- [ ] Each empty state from the catalog is implementable in Phase 0.
- [ ] All copy is i18n-keyed.
- [ ] Every empty state has a next action.

#### 18.3.2 Keyboard shortcut map

| Shortcut | Action | Scope |
| --- | --- | --- |
| `?` | Open help overlay | global |
| `cmd+K` | Command palette / global search | global |
| `cmd+,` | Settings | global |
| `cmd+/` | Toggle scope strip | global |
| `cmd+1..9` | Switch active profile / pinned card | global |
| `[` | Toggle left navigator | global |
| `]` | Toggle inspector | global |
| `g` then `c` | Jump to "card" view | navigation |
| `g` then `n` | Jump to "node" view | navigation |
| `g` then `s` | Jump to "scope" view | navigation |
| `e` | Open evidence timeline (when in card view) | view-specific |
| `r` | Refresh current primitive (when applicable) | view-specific |
| `Escape` | Close overlay / modal / picker | global |

Acceptance gate:
- [ ] Every shortcut in the map is bound.
- [ ] Shortcuts listed in help overlay and onboarding tour.
- [ ] Shortcuts respect accessibility (no system-wide overrides that conflict).

#### 18.3.3 Loading state policy

- Skeleton screens for content >100ms (perceptually instant).
- Inline progress for known-duration actions (≥500ms).
- Indeterminate progress for unknown-duration actions.
- No blocking full-window spinners.
- Network actions have a "fallback to local cached state" path.

Acceptance gate:
- [ ] Skeleton components defined per content type.
- [ ] Indeterminate progress component available for adapter calls.

#### 18.3.4 Confirmation dialogs for destructive actions

Destructive actions requiring confirmation:

| Action | Confirmation copy |
| --- | --- |
| Clear local telemetry | "Delete <count> log entries and <count> onboarding events? Local only — not cloud." |
| Reset settings | "Reset all cockpit settings to defaults? You'll need to re-pair." |
| Revoke device | "Revoke <device_name> from <daemon_url>? You'll need to re-pair." |
| Cancel proof publish mid-flight | "Cancel proof publish? Draft retained locally; server-side artifact abandoned." |
| Force re-pair | "Discard current pairing for <daemon_url> and re-pair? Brief lockout expected." |
| Clear local evidence | "Delete <count> local evidence refs? Affects <count> workpoints." |

Pattern:
- Modal with the exact confirmation copy
- typed `CockpitError` if rejected
- typed `CockpitEvent` if accepted

Acceptance gate:
- [ ] Each destructive action has the documented modal.
- [ ] Audit row appended for every accept/reject.

#### 18.3.5 Undo/redo for state-modifying cards

Rules:
- Each Phase 0 write card (only `focusa.evidence_link` in Phase 0) supports undo for ≤30s after execution.
- Undo restores the prior local state; it does not roll back the daemon.
- Redo re-issues the action.
- Undo history is per-session and in-memory.
- A persistent undo (across sessions) is out of scope for Phase 0.

Acceptance gate:
- [ ] Undo works within 30s for `focusa.evidence_link`.
- [ ] Undo history cleared on app restart.

#### 18.3.6 Bulk actions

Phase 0 bulk actions:

| Bulk action | Cards affected | Confirm? |
| --- | --- | --- |
| Capture evidence from multiple cards | focusa_evidence_link × N | yes |
| Refresh all read cards in current view | all read cards | no (idempotent) |
| Revoke multiple paired devices | focusa_device_pair_revoke × N | yes |

Acceptance gate:
- [ ] Bulk actions defined for Phase 0.
- [ ] Each bulk action has progress + audit row.

#### 18.3.7 Export / import

Phase 0 export/import:

| Item | Format | Destination |
| --- | --- | --- |
| UserSettings | JSON | local file picker, JSON |
| Pairing provenance | JSON | local file picker, JSON |
| Onboarding events | JSONL | local file picker |
| Local evidence refs | JSON | local file picker |
| Audit row subset (filtered) | JSONL | local file picker |

Rules:
- Exports are redacted by default (no secrets, no token values).
- Imports validate schema before applying.
- Imports of evidence refs respect scope.

Acceptance gate:
- [ ] Each export is implementable and redacted.
- [ ] Each import validates schema.

### 18.4 Integration completeness

#### 18.4.1 CI alternatives

Primary CI: GitHub Actions.
Supported alternatives:
- GitLab CI: `.gitlab-ci.yml` provided as a sibling.
- Bitbucket Pipelines: `bitbucket-pipelines.yml` provided.
- Local CI: `scripts/local-ci.sh` mirrors the GitHub workflow for offline / VPS operators.

Each alternative must include: typecheck, lint, smoke, e2e (when runnable), security audit.

Acceptance gate:
- [ ] `.gitlab-ci.yml` ships with the cockpit.
- [ ] `bitbucket-pipelines.yml` ships with the cockpit.
- [ ] `scripts/local-ci.sh` runs in CI dry-run mode.

#### 18.4.2 Tailscale / Cloudflare specifics

Tailscale:
- MagicDNS hostname probe list: `focusa-vps`, `focusa`, `focusa-daemon`, plus user-configured (localStorage hint).
- ACL: cockpit needs no Tailscale admin scope; the existing daemon token is the trust boundary.
- Coordinated sharing: not in cockpit's scope.

Cloudflare:
- Tunnel: optional via Focusa Cloud relay (Spec 115 §9.4). Cockpit does not host its own tunnel.
- Cloudflare Access: optional; integrate via bearer-token sent with each request.
- Cache rules: irrelevant (cockpit is a desktop app).

Acceptance gate:
- [ ] MagicDNS probe list is documented and matches FirstRunWizard ordering.
- [ ] Bearer-token path for Cloudflare Access documented.

#### 18.4.3 MCP bridge role for cockpit

Decision:
- Cockpit is **an MCP client** (consumes MCP tools from `focusa-mcp` server on the local daemon).
- Cockpit is **not an MCP host** in Phase 0.
- Future Phase 1+ may let cockpit expose its cards as MCP tools.

Implications:
- Cards that have MCP equivalents are noted in the manifest with `mcp_equivalent: "<server.tool>"`.
- The cockpit CLI (`cockpit doctor`, `cockpit smoke`, etc.) can be invoked via MCP from external tools.

Acceptance gate:
- [ ] Cockpit runs as an MCP client; verified in smoke harness.
- [ ] At least 5 Phase 0 cards have documented `mcp_equivalent` mappings.

#### 18.4.4 OAuth flow for Cloud Profile

OAuth provider: Focusa Cloud identity (`cloud.focusa.dev/oauth`).
Scopes:
- `openid` (required)
- `profile` (display name, email)
- `focusa.cloud.read` (account, nodes, devices, license)
- `focusa.cloud.write` (only when explicitly granted by operator)

Rules:
- Authorization Code + PKCE flow.
- Tokens stored in OS keychain (`focusa-cloud-token` service).
- Refresh tokens rotated; access tokens ≤15min TTL.
- MFA supported via TOTP per existing Focusa Cloud account rules.
- Logout revokes refresh token server-side and clears keychain entry.

Acceptance gate:
- [ ] OAuth flow implementable in cockpit OAuth card (Phase 0 cloud profile).
- [ ] Tokens stored in keychain only.

#### 18.4.5 Multi-tenant cockpit

Decision:
- Phase 0 cockpit serves one operator per install (per-user Keychain scope enforces this).
- Multi-tenant UI in cockpit (e.g. team dashboard) is **out of scope** for Phase 0.
- The cloud account can still represent a team; cockpit just shows the operator's view.

Acceptance gate:
- [ ] Phase 0 explicitly excludes team UI in cockpit.
- [ ] Per-user Keychain scoping enforced.

#### 18.4.6 Real-time collaboration

Rules:
- Two operators on two Macs viewing the same workstream see CRDT events via truth plane.
- Cockpit shows: node graph, sync status, ownership role, last canonical write.
- Conflicts surface as the H1 multi-node CRDT UX states.
- No direct app-to-app transport added.

Acceptance gate:
- [ ] Conflict UX states implementable per §3.38 H1.
- [ ] Subscribes to /v1/sync/events for cross-machine events.

#### 18.4.7 Multi-user on same Mac

Rules:
- macOS Keychain scoping per-user ensures tokens don't leak.
- Tauri app launches per-user session.
- Per-user `~/Library/Application Support/Focusa` directories.
- When user switches via fast user switching, the running cockpit stays bound to the launching user's session; switching users must relaunch.

Acceptance gate:
- [ ] Per-user data isolation enforced.
- [ ] Documentation explains fast-user-switching behavior.

### 18.5 Business completeness

#### 18.5.1 Per-feature gating for cloud profile

| Feature | Local Only | Cloud Profile |
| --- | --- | --- |
| UIAI Engine cards | ✓ | ✓ |
| Focusa local cards | ✓ | ✓ |
| Cloud node status | blocked | ✓ |
| Cloud device pairing | blocked | ✓ |
| AI API cards | blocked | ✓ (with API key) |
| Proof publishing | local preview only | public snapshot requires plan |
| Benchmark snapshots | local preview only | public snapshots require plan |

Rules:
- Free features are implementable in Local Only mode without any cloud sign-in.
- Cloud-only features render as `blocked_with_reason` in Local Only mode with the message "Switch to Cloud Profile to enable."

Acceptance gate:
- [ ] Each feature in the matrix is gated correctly.
- [ ] Local Only mode is complete (not a trial).

#### 18.5.2 Trial experience

Trial mode (Phase 0 + Spec 115 §14.1):
- Local self-hosted Focusa evaluation license works for 30 days without an account.
- After 30 days, cockpit prompts: "Trial ended. Activate a Focusa license to continue."
- Trial features match Free tier.
- No payment info required for trial.

Acceptance gate:
- [ ] Trial countdown visible in inspector.
- [ ] Post-trial prompt honors the activation flow.

#### 18.5.3 Compliance

Posture:
- **SOC2 Type II**: targeted for Focusa Cloud by Q4 2026; cockpit's role is to surface audit data.
- **GDPR**: cockpit is local-first; personal data minimization by default. Audit log supports data subject access requests.
- **HIPAA / FedRAMP**: not in MVP scope; documented as Phase 2+ via enterprise plan.

Rules:
- No telemetry uploaded without explicit operator action (per §3.4 anti-feature).
- Audit log + telemetry export supports data subject access.
- Right-to-deletion: settings clear + keychain wipe + telemetry purge, all via single button.

Acceptance gate:
- [ ] Audit + telemetry export implementable.
- [ ] Right-to-deletion button documented.

#### 18.5.4 Data retention / cleanup

| Data | Default retention | Operator configurable |
| --- | --- | --- |
| Local telemetry (`app://log`) | 30 days rolling | yes (settings) |
| Onboarding events | 90 days rolling | yes |
| Pairing ledger cache | per-token TTL | n/a |
| Audit rows (local copy) | 365 days | yes |
| Telemetry exported to cloud | only on explicit operator action | n/a |

Auto-rotation rules:
- Older rows are deleted on schedule; no manual cleanup needed.
- Operator can pin specific rows via "star" button.

Acceptance gate:
- [ ] Each retention default is implementable.
- [ ] Settings panel exposes retention controls.

### 18.6 i18n completeness

#### 18.6.1 Non-English locales

Phase 0 ships `en-US`. Phase 1 target locales:
- `de-DE`
- `fr-FR`
- `es-ES`
- `ja-JP`
- `zh-CN`
- `pt-BR`

Each locale ships with translated strings + locale-aware number/date/time formatting.

Acceptance gate:
- [ ] en-US strings ship with Phase 0.
- [ ] Phase 1 locales are listed; no implementation required for Phase 0 beyond the i18n key infrastructure.

#### 18.6.2 Translated error messages

- Every `CockpitError.human_message` is an i18n key.
- Translation files live in `apps/cockpit/src/lib/i18n/strings.<locale>.ts`.
- Missing translation falls back to `en-US` with a console warn.

Acceptance gate:
- [ ] Every error code maps to a translated message.
- [ ] Missing translation logs a warn.

#### 18.6.3 RTL and CJK layouts

- Layout system is logical-property-based (`margin-inline-start`, etc.).
- Component-level mirror handled by Bits UI / shadcn-svelte primitives.
- CJK fonts loaded on demand when `Intl.Locale` reports CJK script.

Acceptance gate:
- [ ] Smoke harness verifies an RTL render passes.
- [ ] CJK font fallback works on macOS.

### 18.7 Gap closure acceptance gates (cross-cutting)

All of the following must be green for the cockpit MVP to claim "done":

- [ ] Test plan (§18.1.1) implementable; coverage targets met.
- [ ] App-level rollback policy (§18.1.2) implementable.
- [ ] Settings migration (§18.1.3) implementable.
- [ ] Release notes template (§18.1.4) used for stable releases.
- [ ] In-app help (§18.1.5) shipped.
- [ ] Card inventory (§18.2.1) matches Phase 0 + deferral table.
- [ ] Search (§18.2.2) implementable.
- [ ] Notifications (§18.2.3) implementable.
- [ ] Bottom ribbon (§18.2.4) implementable.
- [ ] Inspector rail (§18.2.5) implementable.
- [ ] Window management (§18.2.6) implementable.
- [ ] Empty states (§18.3.1) implementable.
- [ ] Keyboard shortcuts (§18.3.2) bound.
- [ ] Loading states (§18.3.3) implementable.
- [ ] Confirmation dialogs (§18.3.4) implementable.
- [ ] Undo/redo (§18.3.5) implementable.
- [ ] Bulk actions (§18.3.6) implementable.
- [ ] Export/import (§18.3.7) implementable.
- [ ] CI alternatives (§18.4.1) shipped.
- [ ] Tailscale / Cloudflare (§18.4.2) documented.
- [ ] MCP bridge role (§18.4.3) defined.
- [ ] OAuth flow (§18.4.4) implementable.
- [ ] Multi-tenant scope (§18.4.5) defined.
- [ ] Real-time collab (§18.4.6) implementable.
- [ ] Multi-user same Mac (§18.4.7) implementable.
- [ ] Per-feature gating (§18.5.1) implementable.
- [ ] Trial experience (§18.5.2) implementable.
- [ ] Compliance posture (§18.5.3) documented.
- [ ] Data retention (§18.5.4) implementable.
- [ ] Locales (§18.6.1) listed.
- [ ] Translated errors (§18.6.2) implementable.
- [ ] RTL + CJK (§18.6.3) implementable.


---

## 16. Provenance and change log

| Date | Change | Section anchors |
| --- | --- | --- |
| 2026-06-19 | Iterable draft created (UIAI Engine docs). | original |
| 2026-06-30 | Reframed as Spec 115 client; Spec 104 scope + multi-node CRDT guard; SaaS interaction; Local Only + Cloud Profile; menubar-style pairing; Svelte/Tauri UI stack; backend-first implementation slices; tagged-release pipeline mirroring Focusa daemon; Phase 0 card manifest bound to real Spec 90 contracts; hard-parts watchlist H1–H20; resolutions for H1, H3–H13, H15–H17, H19, H20; canonical §0–§13 numbering aligned with Focusa peer specs. | §0–§16 |

---

## 17. Cockpit ↔ Menubar connection planes

This section defines exactly how the cockpit (Tauri desktop app in this UIAI Engine repo) and the existing Mac menubar app (`apps/menubar`, Focusa repo) coexist, discover each other, and exchange context. Every gap identified during brainstorming is addressed here as an implementable contract.

### 17.0 Operating principle

```text
1. Reuse what menubar has already proven. If menubar is paired to a daemon, the
   cockpit auto-detects that and offers to add the same machine to the same
   pairing ledger — with zero new device-data entry from the operator.

2. If menubar is missing or has no active pairing, the cockpit runs the full
   first-run flow end-to-end (matches Spec 53 §6.2 menubar-first-run ladder)
   and then exposes the resulting pair to a future menubar install.

3. Path-of-least-friction: every choice the operator can re-enter later is
   cached once (Keychain or truth-plane read) and never asked twice unless
   the cache is invalidated by an explicit event.

4. Fleet-aware: a single Mac may have a Mac menubar pair, a cockpit pair, and
   downstream cloud-registered devices. The cockpit does not silo one of these.

5. Each plane is layered, not copied: truth/discovery/handoff/health/fast.
   No plane bypasses authority.
```

### 17.1 Six connection planes (canonical)

| # | Plane | Plane purpose | Endpoint / signal | Authority |
| --- | --- | --- | --- | --- |
| 1 | Truth | canonical state, pairing ledger | `127.0.0.1:8787` (or fleet-discovered daemon URL) — `/v1/*` | daemon token |
| 2 | Discovery | find cockit + menubar + daemons | `_focusa._tcp.local` (mDNS/Bonjour), Tailscale MagicDNS, saved hint, CLI | none |
| 3 | Handoff | user-initiated app switching | `focusa://`, `cockpit://`, `tauri://localhost/<route>` | none (OS-mediated) |
| 4 | Health | app liveness + version | `GET /.well-known/focusa.json` on loopback port | none |
| 5 | Bridge | Mac ↔ phone pairing handoff | TCP listener on `0.0.0.0:0` with `focusa-connect-v1` envelope | nonce + signed payload |
| 6 | Fast-channel | large payloads between colocalized apps | Unix domain socket (mac/linux) or named pipe (Windows); JSON-lines | per-user |

### 17.2 Truth plane (canonical state + pairing ledger)

```text
cockpit ──HTTP/WS──▶ <daemon_url> ◀──HTTP/WS── menubar
                             ▲
                             └── daemon token per app, distinct from any other
```

- Both apps have their **own** daemon-bound device token (Spec 53 / Spec 115 §9.3).
- Cockpit's token is stored under macOS Keychain service `"UIAI Engine Cockpit Token"`; menubar's remains under `"Focusa Menubar Device Token"` (Spec 53 §A.7).
- Each app stores `daemon_url`, `daemon_token`, `device_id`, `scopes`, `expires_at`, `continuity_id` (if known) under **its own** Keychain service.
- Authority (canonical vs observation) is governed by Focusa Spec 43 + Spec 104 + CRDT. No app-↔-app-talk to mutate canonical state.
- Cockpit inherits menubar's `daemon_url` only when the operator clicks "Add this machine"; the **token + device_id remain distinct**.

### 17.3 Discovery plane and dual-mode pairing (replicated first, auto-add conditional)

**Operating principle (§17.0) restated**: cockpit replicates the menubar first-run pairing flow as the first choice, end to end. The less-friction auto-add from menubar's Keychain is offered **only when a connection is detected during the replicated flow**, as a side path. Replicated pairing is never replaced by auto-add; it is supplemented.

Two interoperating paths:

| Path | Trigger | Steps required | Use case |
| --- | --- | --- | --- |
| A — Replicated menubar pairing (first choice) | first-run or `Re-pair from menubar` | Tailscale → Bonjour → env → CLI paste fallback → bridge open → phone scans QR → VPS mints token → MAC stores in Keychain (`UIAI Engine Cockpit Token`) | operator on a fresh machine, no menubar install, or operator prefers explicit pairing |
| B — Less-friction auto-add (side path) | A discovered a daemon_url AND `focusa-platform` Keychain hint matches that URL AND menubar's paired=true on that URL | 2 clicks: notification → Add | operator has menubar paired on the same Mac and wants the cockpit to inherit pairing without re-entering discovery |

#### 17.3.1 Path A — replicated menubar pairing flow (first choice)

Mirrors `apps/menubar/src/lib/components/FirstRunWizard.svelte` step by step:

```text
Step 1: welcome (focusa: welcome)
Step 2: profile (Local Only default; Cloud Profile advanced)
Step 3: vps_discover — probe in this order, stop on first hit:
   a. Tailscale MagicDNS probe list (focusa-vps, focusa, focusa-daemon, plus user-saved host).
   b. Bonjour / mDNS via `focusa_discover_via_bonjour` (browse _focusa._tcp.local, default 2s timeout).
   c. FOCUSA_DAEMON_URL env var.
   d. localStorage hint fallback (only if Keychain hint unavailable).
   e. Advanced CLI paste (always available).
Step 4: show_qr — at this moment, IF menubar's Keychain hint matches the discovered URL
              (§17.3.2), surface the auto-add side path before/with the QR code.
Step 5: waiting_phone — operator scans QR with phone PWA, phone posts to /v1/connect/rooms + /mac-offer.
Step 6: vps_wait — VPS creates room, Mac polls /v1/connect/rooms + /v1/connect/room/{id}/status every 1.5s.
Step 7: connected — token returned via bridge callback OR /status poll. Token stored under Keychain service "UIAI Engine Cockpit Token".
```

Each step mirrors the menubar-first-run ladder exactly:
- Same Tailscale hostnames.
- Same Bonjour service type.
- Same discover-time semantics.
- Same bridge command names (`focusa_start_bridge_callback`, `focusa_take_bridge_completion`).
- Same payload schema (`protocol=focusa-connect-v1`, `role=mac_completion_payload`).
- Same Spec 104 MBN-01 typed ScopeContext envelope end-to-end.
- Same Keychain service name pattern (per-app, separate name).

#### 17.3.2 Path B — less-friction auto-add (side path)

Surfaced only during the replicated flow, only after a daemon URL is discovered via Path A step 3, and only when the menubar Keychain hint matches.

Trigger conditions (all must be true):

```text
- Replicated pairing is in step 4 (show_qr) or later (does not fire pre-discover).
- A daemon URL was discovered.
- `focusa-platform` Keychain hint (or menubar's own Keychain service) reports a row at that URL with paired=true.
- The operator has not previously dismissed auto-add for this URL.
- Operating profile allows the connection (Local Only + auto-add disabled does not surface the prompt, see §17.16).
```

Prompt:

```text
"Focusa Menubar is paired to <daemon_label> on this Mac.
 Add this machine as a paired device?
 [Add this machine]  [Pair from scratch]  [Details]"
```

On `Add this machine`:

```text
- POST /v1/device/pair/start to <daemon_url> with client_type="cockpit" and the menubar Keychain device_id cross-reference (so the daemon can prove ownership).
- Receive pair_token + nonce.
- Skip the bridge / phone QR / VPS flow entirely.
- Persist pair_token under "UIAI Engine Cockpit Token" with provenance `source=auto_via_menubar`.
- Append CockpitEvent kind=auto_pair_observed_via_menubar.
- Notify the user: "Cockpit is now paired to <daemon_label>."
```

On `Pair from scratch`:

```text
- Continue Path A from step 4 with the same daemon URL.
- Do NOT inherit the menubar's device_id; create a new device_id entry.
- Persist source=fresh_discovery.
```

On `Details`: shows the discovered URL, the menubar device label, the operator-visible fields, with keyboard navigation.

#### 17.3.3 When menubar is not on the same machine or is not in use

Each Path A and Path B has a menubar-absent fallback so the cockpit works standalone:

```text
- mac with no menubar install:
    Path A: full replicated flow runs (Tailscale / Bonjour / env / CLI paste).
    Path B: never fires (no Keychain hint available).
- menubar installed on a *different* Mac:
    Path A: full replicated flow runs on this Mac.
    Path B: never fires (Keychain is per-user, per-Mac).
- menubar installed on this Mac but not running:
    Path A: full replicated flow runs on this Mac.
    Path B: Keychain hint is read; auto-add prompt fires (if conditions met). Since menubar is not currently running, neither app holds the bridge; cockpit binds its own bridge only if the operator initiates a fresh pair.
```

Multi-daemon fleet UX (§17.10) drives the picker when more than one daemon is paired by menubar.

#### 17.3.4 Operator-facing comparison (so the operator can decide)

```text
Replicated Path A:  5+ steps  (~30s typical)   "I want this machine to have its own pairing"
                       includes Tailscale / Bonjour probe, bridge open, phone QR scan, VPS mints token, Keychain stored
Less-friction Path B: 2 clicks  (<2s typical)   "I want this machine to inherit menubar's pairing"
                       requires menubar paired on this Mac; reads Keychain hint; POSTs pair/start directly
```

Defaults:

- Operating profile **local_only**: Path A only. Path B is suppressed.
- Operating profile **cloud_profile**: Path A primary; Path B surfaced as described above.
- `FOCUSA_COCKPIT_AUTO_PAIR_DENY=1`: Path B denied for this run; Path A only.
- `FOCUSA_COCKPIT_AUTO_PAIR_QUIET=1`: Path B auto-accepted without prompt (CI runners only; check TTY first).
- `FOCUSA_COCKPIT_AUTO_PAIR=1`: explicit CI opt-in for Path B; no TTY check.

### 17.4 Handoff plane (deep links)

- `focusa://` stays owned by menubar (Spec 53 §A.6, PORTABILITY_AUDIT.md §A.6). Tauri `Info.plist` registers `focusa` for `com.focusa.menubar`.
- Cockpit registers `cockpit://` for `com.focusa.cockpit`. macOS routes `cockpit://` to the second bundle id. Both bundle ids ship as separate Tauri binaries.
- Cargo / tauri.conf.json for cockpit must include `CFBundleURLTypes` declaring `cockpit` scheme.
- Reverse direction: `focusa://card/<id>` from cockpit → opens menubar Mission tab pointing at the workpoint that owns the card.
- `cockpit://focusa/<what>` from menubar → opens cockpit with that detail.
- Both apps also handle `tauri://localhost/<route>` for in-app navigation after bridge callback.
- Resolution conflict (deep-link target missing) → fall back to truth-plane read; never block.

### 17.5 Health plane (loopback manifest)

```text
GET http://127.0.0.1:<port>/.well-known/focusa.json
```

Manifest:

```json
{
  "schema": "focusa.app.manifest.v1",
  "app": "focusa-menubar" | "uaiengine-cockpit",
  "version": "0.9.51-dev",
  "channel": "stable | preview | dev",
  "build_id": "...",
  "scope_ref_supported": true,
  "capabilities": ["pair.start", "pair.complete", "card.open"],
  "auto_pair_inherited_supported": true,
  "client_type": "menubar | cockpit"
}
```

- Read-only, no auth, returns 200 if app is running.
- Used by the other app to detect "is the sibling app installed and on a compatible version?".
- Used by menubar to surface "this machine can also be opened in cockpit" affordance and vice versa.
- Manifest changes are versioned; the client_type field pins which protocol version the manifest honors.

### 17.6 Bridge plane (Mac ↔ phone for VPS pairing)

Both apps reuse the **same** Tauri commands and envelope (Spec 53 §2.0, Spec 54 §B.5):

```rust
#[tauri::command]
fn focusa_start_bridge_callback(nonce: String) -> Result<String, String>;
#[tauri::command]
fn focusa_take_bridge_completion(nonce: String) -> Result<Option<String>, String>;
```

Envelope:

```json
{ "protocol": "focusa-connect-v1", "role": "mac_completion_payload", ... }
```

Bridge ownership per room (per §17.10):

```text
- If menubar started the room (room.client_type = menubar, default): menubar holds the bridge listener.
- If cockpit started the room (room.client_type = cockpit): cockpit holds the bridge listener.
- The daemon stamps client_type on the room at creation time.
- Deep-link fallback focusa:// → menubar; cockpit:// → cockpit.
- Both apps implement the bridge commands, but only one is active for a given nonce.
- The other app polls /v1/connect/room/{room_id}/status as the canonical completion channel (matches men's FirstRunWizard pollRoomsList pattern, every 1.5s).
```

Bridge-envelope constraint (preserves Spec 104 MBN-01):

```text
All bridge messages between Mac, VPS, and Phone preserve the typed
ScopeContext (project_root, continuity_id, session_id, thread_id) end-to-end.
The bridge does NOT mutate canonical scope state; it forwards metadata alongside token/nonce payloads.
```

### 17.7 Fast-channel plane (large payloads, optional in MVP)

```text
macOS / Linux:  $TMPDIR/focusa-ipc.sock (Unix domain socket, JSON-lines)
Windows:       \\.\pipe\focusa-ipc (named pipe, JSON-lines envelope)
```

- Per-user, readable only by the same UID / SID as the apps.
- Used for FPV frame relay between UIAI Engine client and the second app, large evidence blob transport, telemetry batches.
- Persistent fallback to truth-plane HTTP chunks if the socket is unavailable.
- Not required for MVP. Phase 1+ optimization.

### 17.8 Cross-platform storage and the Linux/Windows non-Mac path

The "auto-add" principle must survive macOS-only Keychain:

```text
macOS:    macOS Keychain (per-user)
Linux:    Secret Service / libsecret via tauri-plugin-keyring (per-user)
Windows:  Windows Credential Manager via tauri-plugin-keyring (per-user)
Sandbox:  app keychain via freedesktop.org Secret Service
```

- All platforms store `{daemon_url, daemon_token, device_id, scopes, expires_at}` under service `"UIAI Engine Cockpit Token"`.
- Shared `focusa-platform` hint entry holds `{daemon_url, last_verified_at, client_types_seen}`.
- A platform must never fall back to plaintext local file storage for tokens.

### 17.9 CI / non-interactive cockpit (unattended machine)

On a CI runner or agency "headless" deployment:

```text
- FOCUSA_COCKPIT_AUTO_PAIR=1   honored only when running unattended (no TTY).
- FOCUSA_COCKPIT_AUTO_PAIR_DAEMON_URL=...  explicit daemon URL.
- FOCUSA_COCKPIT_AUTO_PAIR_SCOPE=...     optional ScopeRef seed (e.g. ci/<runner>/<repo>).
- FOCUSA_COCKPIT_AUTO_PAIR_QUIET=1       suppresses the "Add this machine?" prompt entirely; logs auto-pair event to app://log and the audit row kind=cockpit_auto_paired.
- FOCUSA_COCKPIT_AUTO_PAIR_DENY=1        explicitly disables auto-pair, even if menubar is paired; bypasses the prompt.
```

### 17.10 Multi-daemon fleet picker (not a single daemon per Mac)

Operator reality check: a Mac may be paired to multiple daemons (personal / work / client / homelab / CI). Auto-add must not silently pick the wrong one.

Behavior:

```text
- cockpit reads menubar Keychain service entries; if menubar stores
  multiple {daemon_url, paired=true} rows, cockpit enumerates them.
- For ONE row: skip picker, default to the single paired daemon.
- For >1 rows: open a small "Add to which Focusa daemon?" picker:
    [ ] Personal Focusa  (focusa-personal.local:8787)  ★ used today
    [ ] Work Focusa      (focusa-work.local:8787)
    [ ] Client Focusa    (focusa-client.example:8787)
    [ ] All of the above
- The ★ icon marks menubar's currently-active daemon (if menubar exposes it via /health or the room list).
- After pairing, cockpit surfaces the full device list via the focusa_device_pair_list Phase 0 card.
```

Daemon-side schema:

```json
{
  "schema": "focusa.cloud.device_ledger.v1",
  "client_type": "menubar | cockpit | pi | claude | codex | cursor | opencode | mcp | ssh_identity | ci",
  "device_id": "...",
  "display_name": "Verious MacBook (cockpit)",
  "daemon_url": "http://focusa-vps:8787",
  "paired_at": "...",
  "expires_at": "...",
  "scopes": ["..."]
}
```

The `client_type` field is the key gap-closer for the fleet model.

### 17.11 Token expiry + re-pair loop

Auto-add is not "set and forget." Tokens expire.

```text
- cockpit stores daemon_token + expires_at per row.
- If menubar's token expires and menubar repairs, cockpit observes via truth-plane /v1/device/pair/status card and re-validates its own.
- If cockpit's own token expires and menubar is still paired, cockpit displays:
    "Cockpit's pairing to <daemon> expired. Refresh from menubar's pairing? [Refresh] [Repair]"
    - Refresh reuses menubar's discovery state; one click.
    - Repair runs the full bridge flow.
- If both apps' tokens expire and menubar is not running, cockpit runs the full first-run ladder.
- All expiry events append CockpitEvent rows kind=pairing_refreshed|pairing_repaired.
```

### 17.12 Card affordance: pair-status card surfaces auto-add result

The Phase 0 card `focusa_device_pair_status` (§3.15) is the operator-visible summary:

```text
- on launch, card surfaces:
    - menubar paired_to:  focusa-vps:8787  ★ currently active
    - cockpit paired_to:  focusa-vps:8787  ✓ just auto-added at 14:02
    - work-vps:          not paired
- tapping the "Refresh from menubar" button next to an expired row runs the
  fleet picker (§17.10) and re-issues the auto-add.
```

The card never silently auto-pairs; it surfaces the auto-pair provenance from UserSettings (§17.13).

### 17.13 Settings + observability + provenance

`UserSettings` (§3.26) gains:

```ts
interface UserSettings {
  ...
  pairing_provenance: {
    [device_id: string]: {
      source: "auto_via_menubar" | "fresh_discovery" | "cli_paste" | "explicit_repair";
      observed_at: string;
      menubar_keychain_service?: string;
    };
  };
  last_auto_pair_event_at?: string;
}
```

`OnboardingEvent` (§3.28) gains a new kind: `auto_pair_observed_via_menubar`.

`CockpitEvent` (§3.8) gains a new kind: `pairing_observed_via_menubar` with detail `{ scope_ref, daemon_url, expires_at, source }`.

The "in-app telemetry viewer" (§3.23) lists all auto-pair events with the provenance field.

### 17.14 Sandbox + entitlements + Keychain cross-app read

macOS hardened runtime + sandbox can block a second Tauri binary from reading another's Keychain entries. Cockpit must declare:

```xml
<key>keychain-access-groups</key>
<array>
  <string>$(AppIdentifierPrefix)com.focusa.shared-focusa</string>
</array>
<key>com.apple.security.application-groups</key>
<array>
  <string>group.com.focusa.shared</string>
</array>
```

- `focusa-platform` and `UIAI Engine Cockpit Token` entries are tagged with access group `$(AppIdentifierPrefix)com.focusa.shared-focusa` to grant read to both menubar and cockpit.
- Both apps set `com.apple.security.app-sandbox=true` and add the same `application-groups` declaration.
- Verified via macOS keychain ACL test (`security cms-find-identity-participating` not required; just verify `security find-generic-password -s focusa-platform` works as either user).
- On Linux/Windows, secret-service / credential-manager isolation is per-user, which is the natural boundary.

### 17.15 Auditability + tool class assignment

Auto-add is **Class B (local write, scope required, audit log mandatory)** per Spec 115 §9.6. Each auto-add event appends:

```text
CockpitEvent {
  kind: "auto_pair_observed_via_menubar" | "pairing_refreshed" | "pairing_repaired" | "pairing_revoked"
  scope?: ScopeRef
  provenance: PairingProvenance
  audit_required: true
}
```

Class B requirements (Scope 115 §9.6) must be satisfied: device token + scope + thread authority + side-effect declaration + operator approval + audit log. The notification UI is the operator approval step.

### 17.16 A11y, profile lock, and small UX details

- Auto-add notification uses an ARIA live region `role="status" aria-live="polite"`.
- Keyboard-only operator can Tab to "Add" / "Not now" / "Details".
- Reduced-motion respected for slide-in animations.
- If the operator has set `operating_profile = "local_only"` AND menubar points at a daemon URL, the notification reveals what menubar has paired but **does not** auto-pair and shows:
    "Menubar is paired to focusa-vps. Switch to Cloud Profile to add this machine? [Switch] [Stay Local Only]"
- A subsequent "Re-pair from menubar" button on the device-card list always works.
- Cross-Mac repair propagation: when menubar repairs on Mac A, cockpit on Mac B observes via the daemon's `cross_machine_event_log` (Spec 43 §3) — needs no new transport.

### 17.17 Operator-visible flow (final summary)

```text
Mac starts.
├── menubar is installed and paired to focusa-vps:8787
│     └── cockpit reads shared Keychain hint
│           ├── if maco_quiet=0 (default):
│           │       "Add this machine to focusa-vps as a cockpit device?"
│           │       [Add] [Details] [Not now]
│           └── on Add:
│                   ├── POST /v1/device/pair/start client_type=cockpit
│                   ├── store token in "UIAI Engine Cockpit Token"
│                   ├── append CockpitEvent auto_pair_observed_via_menubar
│                   └── refresh pair-status card with ★ marker
├── menubar is installed but not paired
│     └── cockpit runs first-run ladder (mirrors menubar-first-run)
└── menubar is not installed (CI / fresh machine)
      ├── if FOCUSA_COCKPIT_AUTO_PAIR_DAEMON_URL: pair silently
      ├── else: run CLI paste fallback
      └── in either case: append source=auto_pair_event_quiet to UserSettings
```

### 17.18 Anti-patterns (re-stated)

- Direct fetch/curl between cockpit and menubar dev ports.
- Sharing tokens between menubar and cockpit.
- Auto-pairing in unattended environments without FOCUSA_COCKPIT_AUTO_PAIR_* env vars set.
- Treating a Cloud Profile pairing as identical to a local VPS pairing.
- Hidden auto-pair that bypasses the operator for Class B (local write) operations.
- Plaintext local file token storage on Linux/Windows.

### 17.19 Layered example: cross-Mac repair propagation

```text
1. Mac A's menubar repairs pairing to focusa-vps (extends daemon_token TTL).
2. Daemon appends cross_machine_event row kind=pairing_refreshed with daemon_url, device_id.
3. Mac B's cockpit subscribes to /v1/sync/events on the same daemon.
4. On receiving the event, cockpit re-validates its own daemon_token with the new TTL.
5. If cockpit's own token also expires, its pair-status card shows the refresh button.
```

Spec 43 §3 already provides the cross-machine event log. Cockpit subscribes; nothing new is required.

### 17.20 Open questions (carried into §12)

1. Should `cockpit://` deep-link targets be namespaced (`cockpit://focusa/...`) or flat (`cockpit://card/<id>`)?
2. Should fleet picker default to menubar's active daemon ★ or to the most-recently-used daemon?
3. Should auto-add be opt-out or opt-in (default ON or default OFF)?
4. Should the spec also cover a non-Unix-domain-socket fallback for Linux/Windows fast-channel?
5. Should we pre-reserve `client_type=opencode|codex|claude|cursor` to keep the device ledger uniform across Focusa integrations?

