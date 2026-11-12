# UIAI Engine + Cockpit: Unified MCP / WebMCP Agent Browser
**Design specification:** UIAI-INTEROP-WEBMCP-2026-10-30, revision 0.1 (proposed, review required)  
**Authoring posture:** architecture recommendation and implementation handoff; NOT canonical approval, shipped behavior, or a deployment claim.  
**Repository/source reviewed:** WPUIAI/uiai-engine, main at 1c0a2d3a5a2724fcad7dde7981c2a0be053316c9 (2026-10-30).  
**External baselines:** MCP specification 2026-07-28; WebMCP Draft Community Group Report 2026-10-27. Both require version-pinned conformance tests.  
**Scope:** UIAI Engine Go/Rod Chromium, Node stdio MCP bridge, optional remote MCP, Pi/OpenClaw adapters, Svelte/Tauri Cockpit, FPV/Evidence PWA, Focusa-bound browser missions, and an optional first-party Cockpit WebMCP exposure surface.

## 0. Executive decision proposal

Make UIAI **one authoritative browser execution and observation substrate** with three complementary actuation channels: (a) existing DOM/accessibility/CDP/visual browser controls; (b) origin-bound WebMCP page tools discovered and invoked in the actual browser document; (c) agent-facing MCP/HTTP/Pi/OpenClaw/CLI interfaces forwarding to the same engine action boundary. Cockpit is the synchronized human-facing view and command surface, not a second browser or second authority. Focusa continues to own Workstream, mission/CallGraph intent, permissions and completion; UIAI owns browser sessions, tool execution, observed effects, and evidence candidates.

Do NOT replace existing semantic browser interaction with WebMCP. Do NOT represent page-authored WebMCP metadata as trusted capabilities or Focusa operations. Do NOT dump a website's full tool registry into each agent's global MCP catalog. Do NOT create another mutable session or Workstream store in Cockpit.

This proposal extends, does not supersede:
- docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md, especially Section 8 Origin-Bound WebMCP Actuator Broker; existing AFB-020 in its C01 ledger.
- docs/UIAI_COCKPIT_005_WORKSTREAM_SCOPED_UNIVERSAL_AGENT_CONTROL_AND_MULTIMODAL_VISUAL_WORKSPACE_RUNTIME_AMENDMENT_2026-08-04_v1.0.md (exact Workstream binding, shared GUI/API/MCP/Pi control).
- docs/UIAI_COCKPIT_004_SCOPED_WORK_SURFACES_TABS_PANES_WINDOWS_MOTION_SPEC_2026-08-04_v1.0.md (do not invent a new tabs/panes/work-surface model).
- docs/UIAI_COCKPIT_008_FOCUSA_MISSION_CANVAS_INTERLOCK_HANDOFF_CREDENTIALS_SUPERVISION_2026-08-01_v1.0.md and docs/contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml.
- docs/ARCHITECTURE_AUTHORITY_POLICY.md, docs/PUBLIC_API_PARITY_MATRIX.md, docs/ENDPOINT_AUTH_MATRIX.md, docs/SESSION_API.md, docs/RELEASE_DEPLOY_RUNBOOK.md.
- Issue #65 and Focusa #281/#293/#294/#295/#296 cross-process IR0/IR5 authority gates.

The root architecture owner must approve promotion to canonical normative architecture. Engineering spikes, tests and proposed PRs may proceed without claiming this promotion.

## 1. Verified as-built baseline and gap classification

| Surface | Verified code/docs as of above SHA | Outstanding work / non-claim |
|---|---|---|
| Engine | Go Chi HTTP server and Rod/Chromium browser sessions under internal/vision and internal/server; read, snapshot, semantic refs, navigation, actions, screenshots, diagnostics, Focusa/Evidence handoff. | No verified production WebMCP discovery/invocation broker. Browser feature capability must be probed at runtime. |
| MCP | mcp/browser-session-mcp.mjs is a Node **stdio** JSON-RPC MCP-to-Go HTTP bridge; code advertises protocolVersion 2025-03-26; /api/tools/mcp feeds the catalog; route smoke exists. | No verified 2026-07-28 conformance or authenticated Streamable HTTP MCP endpoint. Upgrade requires a real protocol adapter, not editing a version string. |
| Other agents | Pi extension, CLI; integrations/openclaw plugin source for #249 with host session binding. | #249 is open, README explicitly disclaims production activation/Discord delivery proof. Keep the engine as sole browser session owner. |
| Cockpit | Svelte/Tauri package v0.3.2, route /live -> LiveWorkspace.svelte, engine-client.ts; real engine health/sessions, open/navigate/close, screenshot, diagnostics, FPV attachment; /capabilities is a read-only card-manifest-backed catalog. | Not yet a full tabbed browser with WebMCP inspector, authoritative event timeline, tool execution approval/receipt UI, or a unified live registry projection. Do not imply otherwise. |
| Cockpit session refresh | LiveWorkspace currently polls engine state about every 5 seconds. | Real-time event subscription and global session consistency are not established by this poll. Honor settings (#287); protect selected engine identity. |
| Security | Existing entitlement, EPWA delivery/redaction, origin and secret work; #280/#281 closed with code changes; #81 still open. | No WebMCP mutation/exfiltration privilege until applicable entitlement, Focusa authority, egress and secret gates actually pass. |
| Existing WebMCP design | Agent-First Browser spec Section 8 already defines origin-bound ephemeral tool candidates and untrusted metadata. | This document decomposes runtime and Cockpit delivery; do not create a parallel tool-candidate doctrine. |

Related issues: #65 integration; #76 session lease/reconciliation; #81 secret-safe browser surface; #92 browser-as-evidence; #93 tailnet mesh; #102 modes; #106 EPWA; #143 browser adoption; #238 delivery; #249 OpenClaw; #287 Live updates; #288 endpoint selection (closed). A closed issue is not proof that the combined target works.

## 2. Standards and scope: avoid category confusion

**Backend MCP** connects independent agent clients to tool/resource/prompt services. Current version 2026-07-28 is a stateless JSON-RPC core, supports stdio and Streamable HTTP transport, with per-request metadata, method/name HTTP headers, MRTR, cache hints, auth and version negotiation. Preserve compatibility with older clients using a deliberately versioned adapter. UIAI already acts as an MCP **server**; optional UIAI outbound MCP **client** support is a separately scoped feature.

**WebMCP** is an experimental browser Document API. It is not a network MCP endpoint and cannot be assumed to expose MCP resources or prompts. In the 2026-10-27 draft, the target document API has registerTool, getTools, executeTool, toolchange, secure-context and origin/permissions-policy semantics. Distinguish imperative JS tools and declarative form annotations. A site may not expose any tools. Chromium support is experimental (Chrome origin trial/flag; Edge origin trial); Tauri webviews and headless/remote configurations must be individually probed. Never silently inject a fake WebMCP implementation into uncooperative third-party pages to claim conformance.

Four integration directions must be distinct:
1. **UIAI MCP server:** external agents invoke UIAI's stable engine operations.
2. **UIAI WebMCP consumer:** UIAI discovers and invokes untrusted ephemeral tools from the page currently owned by its browser session.
3. **Cockpit optional WebMCP provider:** on supported secure first-party Cockpit pages, expose a restricted allowlist of registered, already-governed UIAI commands to in-browser agents; never expose local secrets or ad hoc Svelte callbacks as privileged tools.
4. **UIAI optional MCP client:** deliberately configured service connectors for supported MCP servers, with separate per-connector authorization. Do not turn the browser runtime into an unrestricted generic remote MCP gateway or duplicate Focusa/OpenClaw orchestration.

## 3. One authority, one session, one event lineage

Canonical flow:

    Focusa scope / Workstream / CallGraph / approved action
      -> UIAI engine action gateway (auth + entitlement + policy + lease)
      -> session identity + observation + capability candidates
      -> select page WebMCP OR semantic/DOM/CDP/visual actuator
      -> execute at exact session/target/frame
      -> browser/network/visual observation and diagnostics
      -> immutable result, action/effect uncertainty, evidence refs
      -> Focusa verification and completion adjudication
      -> Cockpit and agent channels project the same receipts/events

An agent may interact through Pi, OpenClaw, MCP, HTTP or Cockpit without spinning up a second browser, cloning profile cookies, adopting another tenant's session or creating another Workstream identity. Multi-agent observers may read with scope; one actor may own a conflicting mutation per lease generation. A human takeover invalidates agent action rights until explicit handback and fresh observation. Focusa holds canonical work authorization; UIAI holds browser runtime truth; Veragensia may constrain OS-level capability; engine entitlement is an independent required gate.

## 4. Required typed contracts (names proposals; reconcile with existing C01 contracts first)

### UMC-001: SessionCapabilitySnapshot

    schema: uiai.session_capability_snapshot.v1
    engine_node_ref: opaque
    engine_version: string
    browser_runtime: {engine_family, version, channel, headless, feature_flags_digest}
    session_ref: opaque
    context_ref: opaque
    target_ref: opaque
    document_ref: opaque
    navigation_generation: integer
    frame_ref: opaque
    top_level_origin: origin
    registration_origin: origin
    observation_ref: opaque
    observed_at: time
    expires_at: time
    webmcp_support: supported|unsupported|disabled|unknown|error
    support_reason: safe_enum
    toolset_digest: sha256
    tools: bounded_candidate_summaries
    truncated: boolean
    continuation_ref: optional

Candidate summaries include stable-for-observation opaque candidate_ref, tool name, short page-provided description, input/output schema digest, page-provided annotations marked UNTRUSTED, engine-assigned risk/side-effect estimate, registration frame/origin, feature version, availability and stale state. Never use a globally permanent page-tool identity.

### UMC-002: OriginToolCandidate and request

Reuse the proposed uiai.origin_tool_candidate.v1 from UIAI-COCKPIT-002 Section 8 and add only justified adapter fields. Mutation request must bind:

    schema: uiai.origin_tool_execution_request.v1
    action_intent_ref: opaque
    focusa_context_ref: opaque_or_absent_for_exploration
    workstream_attachment_ref: optional_validated
    uiai_session_ref: opaque
    target_ref: opaque
    expected_document_ref: opaque
    expected_navigation_generation: integer
    expected_observation_ref: opaque
    candidate_ref: opaque
    candidate_manifest_sha256: sha256
    input: validated_bounded_json
    action_lease_ref: required_for_mutation
    idempotency_ref: optional_based_on_effect
    approval_ref: required_when_policy_demands
    egress_policy_ref: opaque
    response_profile: compact|expanded
    timeout_ms: bounded

The server, not Cockpit or an agent prompt, validates the exact observed candidate, origin/frame, current tool version, schema, input limits, action lease, session principal, entitlement, allowlisted destination/data classes and approval. A tool description's readOnlyHint/consequentialHint is advisory, never sufficient to downgrade risk.

### UMC-003: Execution receipt and observation

    schema: uiai.origin_tool_execution_result.v1
    execution_ref: opaque
    action_intent_ref: opaque
    session_ref: opaque
    candidate_ref: opaque
    state: blocked|not_executed|succeeded|failed_before_effect|partial|effect_unknown|cancelled
    side_effect: none|observed|possible|unknown
    request_manifest_sha256: sha256
    browser_pre_observation_ref: opaque
    browser_post_observation_ref: opaque_optional
    page_result_ref: redacted_content_addressed_ref
    diagnostics_ref: optional
    network_ref: optional
    evidence_refs: []
    verification: observed|unverified|contradictory|pending
    retry_safe: boolean
    reason_code: stable_enum
    suggested_next_action: stable_action
    completed_at: time

Do not return success merely because the page callback resolved. After navigation or tool removal, invalidate old candidate references. After a possible side effect or timeout, re-observe before deciding whether to retry. Cancellation does not imply rollback. Focusa alone promotes evidence to completion.

### UMC-004: Engine event cursor

    schema: uiai.browser_session_event.v1
    event_id: monotonic_per_session_or_unique
    event_seq: monotonic_per_session
    engine_node_ref: opaque
    session_ref: opaque
    event_type: session|document|navigation|toolset|action|approval|takeover|diagnostics|evidence|health
    previous_revision: integer
    current_revision: integer
    actor_ref: redacted_opaque
    causation_ref: opaque
    correlation_ref: opaque
    timestamp: time
    visibility_scope_ref: opaque
    payload_ref_or_bounded_delta: optional

Subscription must be authenticated and scope filtered; on gaps/reconnect ask for snapshot+resume instead of merging stale client arrays. If a production-safe event stream already exists, reuse it. Prefer a single engine-owned cursor; Cockpit's 5-second polling is an explicitly labeled degraded fallback, not an authority. Distinguish live, disconnected, stale, unverified, expired, and unsupported states.

## 5. Engine work: WebMCP detection, discovery and execution

**UMC-010 feature probe:** For every browser build/provisioning profile record actual Chromium version, flags, context secure status, whether document.modelContext exists and getTools/executeTool are supported. This is per-session/target; do not hardcode browser-brand success. Experimental WebMCP opt-in must not change normal browser profiles globally or assume universal headless support.

**UMC-011 page tool discovery:** Run through the existing owned Rod/CDP session and exact current document execution context. Access document.modelContext.getTools() only where supported and permitted; produce a bounded schema-validated summary, not executable JS closures, full webpage dumps or raw private data. Register toolchange observation where supported and reconcile with document/target navigation. Must handle dynamic SPA registrations, abort/removals, same-name replacements, iframe topology, page crash, navigation and browser restart.

**UMC-012 frame/origin policy:** Treat same-origin by default. Cross-origin tools require current standard's explicit embedder Permissions Policy and registering tool's exposedTo opt-in, plus UIAI's stricter operator policy. Never inspect foreign iframe private content through injected cross-origin bypasses. Include top-level and registration origins in every scoped candidate. Enforce sandbox, insecure-context, target and tenant isolation.

**UMC-013 invoke through browser API:** Resolve the exact RegisteredTool from a fresh current registry in the correct document context; call document.modelContext.executeTool(registeredTool, parsedValidatedInput, options) per detected draft support. Never trust serialized page tool descriptors from a prior navigation. Harden JS result transport, async cancellation, bounded sizes, timeouts, structured errors, Unicode preservation and redaction. Capture provenance on both metadata and output. Page scripts are untrusted even though calls are via a standard API.

**UMC-014 selector/visual fallback:** If a page exposes no WebMCP, or capability is unsupported or policy blocks it, retain ordinary semantic snapshot/ref, DOM/CDP/visual and human takeover. Re-evaluate policy and effect class for fallback; no fallback silently escalates privilege, ignores a confirmation, or invents success.

**UMC-015 resilience:** Establish bounded per-session capacity, tool concurrency and operation limits; neither WebMCP discovery nor tool callbacks may stall the engine's main browser health/cleanup loop. Preserve restart reconciliation; don't leak persistent event listeners, content worlds, session references or callbacks.

## 6. MCP interfaces and version strategy

**UMC-020:** Keep the existing stdio MCP server and its current tools/routes and compatibility with installed Pi clients. Route to canonical engine operations, not a new JavaScript browser backend. Add a compatibility-tested 2026-07-28 MCP server adapter (stateless request metadata, version negotiation, appropriate discovery, header routing and auth requirements for HTTP, MRTR where supported), retaining an explicit legacy path as needed. The currently hardcoded 2025-03-26 initialization is a real modernization gap. Versioned adapters should share the same engine tool registry and policy.

**UMC-021:** Define a small stable MCP surface to operate page candidates:
- uiai_browser_webmcp_status(session_ref)
- uiai_browser_webmcp_discover(session_ref, query?, cursor?, max_results?)
- uiai_browser_webmcp_invoke(session_ref, candidate_ref, manifest_digest, observation_ref, input, intent/grant/approval refs)
- uiai_browser_webmcp_observe(session_ref, since_event_seq?)
Names are provisional until canonical registry and API route parity review. Avoid emitting one globally registered MCP tool for every arbitrary page registration; page tool names may be malicious, huge, rapidly changing or tenant-specific.

**UMC-022:** Add authenticated MCP Streamable HTTP only if a concrete remote-client need is approved; isolate clients/tenants and enforce Origin checks, auth, scopes, TLS, rate limits and denial of DNS rebinding/SSRF. No unauthenticated 0.0.0.0 endpoint. The transport layer does not mint product entitlement or Focusa authority.

**UMC-023:** Optional outbound MCP client is separately configured, allowlisted and scope-bound. Its service connectors are not browser-origin WebMCP tools. Explicitly account for upstream server auth, connector audit, user consent, transport errors, quotas, output prompt injection and credential custody; Focusa/OpenClaw keep orchestration authority.

**UMC-024:** Update canonical tool discovery, exact-name ranking, cache invalidation, Pi mirror, OpenClaw adapter, CLI help, HTTP route matrix, MCP route parity test and docs. A page's registry version change invalidates its candidate cache without flushing unrelated engine tools.

## 7. Cockpit: full agent-browser synchronization and GUI completion

Cockpit is a presentation and governed-command surface on top of the engine session. Maintain exact engine-node, session, document, frame and Workstream Attachment identities. Reuse existing Work Surfaces/tabs/panes/windows contracts and control-lease specifications; never establish a second tab model or actor scheduler.

**UMC-030 / Live Browser:** Enhance apps/cockpit/src/lib/ui/LiveWorkspace.svelte and its route/controller/adapters to present real engine-backed tabs and document transitions through existing Cockpit surface architecture, readable address/navigation, loading/network state, secure origin details, visual viewport/stream, semantic element inspector, agent activity timeline and manual control. The displayed tab is an engine target projection, not a local browser webview unless it explicitly attaches to the identical target. Don't advertise a streaming browser when only periodically refreshed screenshots exist.

**UMC-031 / Tool Inspector:** In Live session inspector add WebMCP availability status, registration origin/frame, current tool count, ranked tool search, input/output schema preview, risk/authority posture, freshness, manifest digest, reason for disabled tools, recent executions and proof links. Read-only inspection works without an action lease; invocation opens a structured preview and uses the same engine action gateway. Unsupported browsers clearly say unsupported, not 'no tools'.

**UMC-032 / Controlled execution:** Read-only tools may run within authorized scope; write/secret/spend/destructive tools require full gates and user approval where applicable. Surface preview/confirmation, actual affected targets, pending/blocked/unverified states, postcondition evidence, operator takeover and resumability. UI input must not treat tool annotations as grants. Cockpit button clicks, WebMCP calls and agent MCP calls share one action receipt and one session event timeline.

**UMC-033 / Capabilities:** Current apps/cockpit/src/routes/capabilities/+page.svelte displays read-only static card catalog plus entitlement projection. Add an explicitly separate **Live Website Tools** pane scoped to an attached active browser session, backed by UMC-001; do not silently merge ephemeral WebMCP candidates into permanent registered product capabilities. Display availability, supported transport and selected engine identity. Global catalog remains versioned and entitlement-aware.

**UMC-034 / Settings, Nodes and Test Lab:** Add opt-in WebMCP experimental runtime profile, per-site/frame approval policy and supported-browser diagnostics; use existing Settings and Nodes & Services identity/configuration. Test Lab provides an owned fixture website that registers both read-only and mutation tools, dynamic registration and cross-origin cases; automated browser/agent/Cockpit parity tests render actual page outputs. Honor Live updates/reduced motion state (#287), keep healthy configured engine selected (#288), and never imply health from stale data (#285).

**UMC-035 / Human collaboration:** Use existing FPV attach, revoke, stream and exact control-lease/takeover contracts. Agent must stop mutating immediately when the human takes control; after handback, force re-observation and candidate re-discovery. Operator pause/local freeze is not Focusa canonical pause. Provide explicit controls for reject/approve/revoke, view evidence and see the active agent/Workpoint.

**UMC-036 / Optional Cockpit as a WebMCP provider:** For a secure supported first-party Cockpit document, expose a curated allowlist of existing canonical commands (e.g. current session status, select inspected session, request read-only diagnostics, request controlled navigation). Registration must call real existing command controllers with exact scope and grant checks, not arbitrary Svelte state mutators. Feature-detect support and unregister on component teardown. No assumed WebMCP on Tauri webviews. This is separate from consuming page tools in remote browser sessions. Treat externally invoked first-party UI commands as actions in the same engine event/audit lineage.

**UMC-037 / Cross-system portability:** Identical live session and event identity is projected in Wirebot app, Focusa Desktop, FPV PWA and Cockpit as authorized; UIAI owns session reality and bounded browser artifacts. Focusa owns master-loop progress/settlement. Tailscale remote nodes and Veragensia host constraints do not license taking over sessions from another account/Workstream.

## 8. Security and non-negotiable gates

- Untrusted origin-provided metadata, schema, tool hints, callback result and page audio/visual content NEVER become instructions, Focusa authority, operational capability grants, or trusted spoken approval.
- User/site permission and UIAI entitlement are independent from Focusa scope/operation approval and Veragensia OS capability. Check all applicable gates before effects.
- Prevent cross-origin data laundering: cross-site context assembled by an agent cannot be forwarded as tool input merely because a page asks for it. Data classification and destination/egress allowlist precede invocation.
- Enforce tool parameter minimum necessity, JSON schema limits, max lengths, secret redaction, timeouts, origin-specific allowances, file/upload/download path restrictions and browser credential/cookie isolation.
- One conflicting actor at a time; audit actor, session, frame, action lease generation, human confirmation, actual effect and evidence. Refuse stale navigation/frame/tool schema revision and revoked/expired lease.
- Use secret broker references and blind fill where authorized, never copy raw credentials into model context, screenshots, diagnostics, tool results or HTML. #81 remains open.
- WebMCP page promise resolved != successful business effect. Record unknown/partial effects honestly, verify browser/network/external confirmation, never blind retry payments, messages, exports or destructive actions.
- The current README blocks new evaluator/customer distribution until signed entitlement and all endpoint gates are verified. This integration cannot be represented as production-ready customer release solely by merging docs or tests.
- Treat screenshots, advertisements, counterfeit UI, prompt injection, malicious JSON schemas, nested iframes, dynamic tool replacement and agent-to-agent propagation as adversarial inputs.

## 9. Dependency-ordered implementation packets

**P0 — Foundation inventory and freeze** (UMC-001 through 004). Confirm current main, all existing versioned contracts, exact engine session/target/observation identity, Cockpit controller/FPV routes, current Go/Pi/MCP parity, Tauri runtime browser behavior. Audit open #65/#76/#81/#102/#249/#287 and existing Cockpit document/contract ledgers. Produce architecture conflict matrix and Focusa IR5 readiness gate; no guessed Focusa API route.

**P1 — Experiment and tests (read-only)** (UMC-010/011/012). Add isolated browser capability probe and owned WebMCP fixture; capture actual Chrome experimental support and unsupported default. Add session/frame/origin-bound discovery and invalidation. No mutation surface yet.

**P2 — Governed execution** (UMC-013/014/015). Add typed engine tool execution under current authority/lease, post-observation, artifact refs and structured error/uncertainty. Test malicious tool descriptions/results and dynamic registration replacement. No separate browser pool or context identity.

**P3 — Agent protocol parity** (UMC-020/021/024). Add narrowly-scoped MCP/Pi/HTTP/CLI routes to the same engine. Modernize MCP protocol as a versioned compatibility project and separately prove 2026-07-28 conformance. Test both legacy installed clients and modern clients against one fixture. Optional remote MCP HTTP and outbound MCP client are gated independent packages (UMC-022/023).

**P4 — Cockpit Live + Capabilities** (UMC-030 through 035). Bind Live Tool Inspector, event timeline, execution preview/receipts, current engine/node and full browser surface to canonical runtime. Test same session across an MCP agent, Cockpit human, and FPV observer. Do not overwrite 004 Work Surfaces/tabs/panes/windows implementation.

**P5 — Cockpit optional WebMCP provider and ecosystem** (UMC-036/037). Expose restricted Cockpit-local commands only on supported runtimes; integrate Focusa/Wirebot/Veragensia without new cognitive stores. Validate secure Tauri/browser support separately.

**P6 — Release and independent evidence.** Execute targeted Go/Node/Svelte/Rust tests, clean dependency install, actual Chromium browser E2E, local/remote authorization, multi-engine/Tailscale, production-config rollback, FPV takeover, signed entitlement and EPWA proofs. Stage behind feature flags; enable for developer-owned fixtures before any trusted-site allowlist, then canary where explicitly permitted. Production approval remains separate.

Every task packet must include: owner/source file/contract version, preconditions, dependency IDs, action, expected result, negative tests, evidence handles, rollback, and independent verifier. Do not call an IR0 epic directly executable.

## 10. Acceptance matrix (blocking)

| ID | Required observable proof |
|---|---|
| A1 | Same webpage with and without browser WebMCP support: discovered-tools vs explicit unsupported with semantic fallback; zero false claims. |
| A2 | Tool registered, schema changed, aborted/unregistered, navigation and iframe replacement: prior candidate is stale and execution denied before mutation. |
| A3 | Cross-origin frame without two-sided opt-in: no access. With opt-in: still blocked unless UIAI policy/authority independently allows. |
| A4 | Read-only page tool called from MCP; exact same execution/session/ref/evidence visible in Cockpit and Pi, with bounded output. |
| A5 | Write tool called from agent while Cockpit holds human control: denied; handback requires fresh observation and lease. |
| A6 | Malicious tool description, output, schema and audio do not override trusted intent, egress or privacy rules. |
| A7 | Tool callback resolves but visible postcondition fails: result says unverified/contradictory, not completed. |
| A8 | Timeout after possible write: effect_unknown, no blind retry, diagnostics and reconciliation path. |
| A9 | Multiple nodes/tenants/Workstreams: no tool, cookie, event, identity, clipboard, artifact or grant crossover. |
| A10 | MCP modern and legacy clients pass feature subset without protocol misrepresentation; stdio and HTTP have their own documented proof and auth. |
| A11 | Cockpit Live, Capabilities, Settings, Test Lab and FPV show current engine endpoint, session events, provenance, entitlement and truthful stale/offline states. |
| A12 | Unsupported Tauri/headless build has functional baseline browser and no fabricated WebMCP registry. |
| A13 | Authorized page tool results settle only as evidence candidates in Focusa; no UIAI-made CallGraph completion. |
| A14 | All public agent/HTTP/Pi/MCP/CLI tools have route parity, error/repair semantics and clean smokes; old browser tests do not regress. |
| A15 | Build/release evidence includes explicit license, secrets, security, Cockpit OTA distribution and rollback gates before production claim. |

## 11. Metrics and resource budgets

Compare real tasks on (a) WebMCP-enabled page, (b) semantic browser fallback, (c) human takeover. Track: verified task success and false-success rate, tool discovery time, invocation time, postcondition verification time, observed UI divergence, recovered stale candidates, secret/egress denials, token count for discovery/results, browser CPU/RAM and page pool impact, per-origin error rate, Cockpit event lag, user approval time, cost per *verified* outcome and reconciled side-effect uncertainty. Define realistic baseline and regression thresholds from measurements rather than inventing target latency numbers. Run sustained multi-session soak and interruption/reconnect trials.

## 12. Concrete implementation loci / ownership

- Engine: internal/vision/session.go, session_eval.go, snapshot.go, session_actions.go and existing browser session handlers; new internal WebMCP adapter only if cross-cutting capability cannot fit existing vision ownership; internal/server and route/auth/entitlement gates; avoid duplicating Rod or launching a second CDP owner.
- MCP: mcp/browser-session-mcp.mjs; mcp/browser-session-mcp.test.mjs; /api/tools/mcp and scripts/smoke-mcp-tool-routes.sh; Pi parity through integrations/pi and canonical installer. MCP protocol modernization should use verified current SDK/contracts.
- Cockpit: apps/cockpit/src/lib/ui/LiveWorkspace.svelte, routes/live/+page.svelte, routes/capabilities/+page.svelte, existing engine-client, fpv-live-adapter, discovery, controllers, cards/entitlement and tests. Honor current Tauri security/deep-link and update rules.
- Interop: integrations/openclaw/ issue #249, Focusa #281/#293/#294/#295/#296, UIAI #65, EPWA contracts, Cockpit document register/004/005/008 and Veragensia browser enforcement contract.
- Documentation: revise parity, session API, agent discovery, security, release proof, and C01 ledgers ONLY after reviewed architecture acceptance. No duplicate authoritative spec number before owner promotion.

## 13. Sources / design verification

- WebMCP live draft (2026-10-27): https://webmachinelearning.github.io/webmcp/
- WebMCP implementation status: https://github.com/webmachinelearning/webmcp/blob/main/implementation-status.md
- Chrome experimental guide: https://developer.chrome.com/docs/ai/webmcp
- MCP 2026-07-28 specification: https://modelcontextprotocol.io/specification/2026-07-28
- MCP release note: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- UIAI own contracts: https://github.com/WPUIAI/uiai-engine/blob/main/docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md
- UIAI cross-process authority issue: https://github.com/WPUIAI/uiai-engine/issues/65
- UIAI Cockpit universal agent control issue: https://github.com/WPUIAI/uiai-engine/issues/7

**Decision requested for owner promotion:** Accept the single unified execution/observation/Cockpit projection design, with WebMCP as an optional ephemeral origin-bound actuator; delegate engineering task decomposition under current Focusa IR5 and Cockpit 004/005/008 contracts. This branch/PR intentionally contains NO runtime changes, approvals or release claims.
