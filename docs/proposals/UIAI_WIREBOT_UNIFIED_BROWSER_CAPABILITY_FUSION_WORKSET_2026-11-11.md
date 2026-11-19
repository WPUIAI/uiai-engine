# Unified Browser & Capability Fusion — One Wirebot-First Implementation Workset

**Workset ID:** UBCF-2026-11-11  
**Status:** OWNER-DIRECTED ACTIVE IMPLEMENTATION TARGET; task admission and runtime proof still outstanding  
**Product goal:** One persistent personal **Wirebot Chief of Staff**, delivered through OpenClaw and composed from Focusa, UIAI Engine, the Wirebot Fusion Engine, SOVOS routines, Portable Memory, Master Loop/Perpetua, Master Observer, and governed sovereign Go APIs.  
**Parent design:** [UIAI unified MCP/WebMCP and Cockpit PR #290](https://github.com/WPUIAI/uiai-engine/pull/290); keep its existing design files and Cockpit Lite scope intact.  
**Cross-product orchestration owner:** Wirebot Core / existing Focusa Project–Workstream admission; **no new cross-product authority**.  
**Implementation truth:** This is a dependency and acceptance workset, not source code, a merged PR, a created Focusa Workstream, an installed OpenClaw skill, a deployed service, or proof of customer readiness.

## 0. Owner decision and scope

Implement the original **MCP + WebMCP browser enhancement** and **production-bottleneck repair** together with a minimal, verified **browser-workflow → compact Wirebot/OpenClaw skill** vertical slice. All work belongs to **one coordinated workset with five owning tracks**. It does not belong to one giant code PR, one new scheduler, one new memory store, or a second agent/browser.

**Product promise:** “Give Wirebot a job; it discovers the best authorized route, completes and verifies the work, can turn a qualified successful procedure into a reusable skill, and improves with evidence.”

The live browser may have WebMCP, DOM, accessibility, CDP and visual channels. Do not assume sites implement WebMCP or treat a directory listing as live authorization. A successful tool callback is not a verified business outcome. Native API execution is an eligible alternative where already authorized.

**Status semantics:** accepted target ≠ approved external mutation ≠ task admitted ≠ code implemented ≠ deployment verified ≠ business outcome settled. Every affected contract and consumer MUST display the correct state.

## 1. Existing component owners and hard boundaries

| Concern | Existing authority / implementation home | Must not duplicate |
|---|---|---|
| Operating Partner, user interaction, skill availability | [Wirebot Core](https://github.com/Startempire-Wire/wirebot-core) + native OpenClaw runtime | Another chief-of-staff agent, conversation/reply path, skill loader, scheduler |
| Project/Workstream/Workpoint/Foreman, permissions, Evidence, receipts, outcomes | [Focusa](https://github.com/Startempire-Wire/focusa), especially Specs 34/140 and actual generated interfaces | Parallel cognitive state, authorizer, canonical receipt ledger, speculative Focusa routes |
| Browser/computer runtime, WebMCP origin tools, diagnostics, evidence candidates | [UIAI Engine](https://github.com/WPUIAI/uiai-engine), PR #290 | Second browser/CDP owner, unchecked page-tool execution, duplicate session state |
| Reusable skills/procedures | Existing Wirebot Fusion Engine direction + Focusa skill/compiler semantics + SOVOS routine doctrine | A brand-new competing skill-compiler authority or automatically trusted page-authored instructions |
| Routine templates/blueprints/instances and leverage | [SOVOS Routine Compiler doctrine](https://github.com/Startempire-Wire/agent-driven-life-business-os/blob/main/SOVOS_ROUTINE_COMPILER_AND_TEMPLATE_LIBRARY.md) | Another routine store/scheduler/owner-specific secrets in portable templates |
| Retrieval, corrections, qualified procedural learning | Existing Portable Memory WRITE/MANAGE/READ lifecycle; configured Mem0/Letta can be selected providers | Extra memory database, unilateral global promotion, retirement of configured providers by assumption |
| Loops, Perpetua, Observer | Existing Master Loop and Focusa metacognition contracts, [SOVOS Master Loop blueprint](https://github.com/Startempire-Wire/agent-driven-life-business-os/blob/main/docs/08-sovos-master-loop-modular-composition-and-evolution-blueprint.md) | Second observer scheduler, event bus, score authority or unverified causality |
| Optional user presentation | Wirebot App #20/#21, Cockpit, Discord, voice, WINS, APIs/CLI | Mandatory GUI, separate skill authorization or shadow agent |
| Host/device control | Veragensia and sovereign deploy contracts | Tailscale as authorization; shared host as full-Sovereign proof |

**Sovereign topology:** For the full-Sovereign operator profile, preserve an owner-dedicated VPS and owner-specific Tailscale mesh with at least one owner-controlled local computer. Managed lower tiers are distinct. Each customer’s skills, credentials, memory and verified outcomes remain isolated; public sharing and network listings require separate consent.

## 2. Five tracks — concrete implementation packets

### A. Production browser hardening — owner: UIAI Engine
**A01** Inventory current default branch, installed binaries, outstanding fixes, regression receipts, issue status and actual host topology. Do not infer deployed behavior from a merged commit or healthcheck.  
**A02** Session admission and bounded reads: [#292](https://github.com/WPUIAI/uiai-engine/issues/292), [#293](https://github.com/WPUIAI/uiai-engine/issues/293), [#300](https://github.com/WPUIAI/uiai-engine/issues/300). Require typed timeouts, useful diagnostics, partial research results, restart and overload behavior.  
**A03** Auth and secret-safe browser: [#302](https://github.com/WPUIAI/uiai-engine/issues/302), [#304](https://github.com/WPUIAI/uiai-engine/issues/304), [#81](https://github.com/WPUIAI/uiai-engine/issues/81), [#82](https://github.com/WPUIAI/uiai-engine/issues/82), [PR #291](https://github.com/WPUIAI/uiai-engine/pull/291). Verify visible challenges, bounded pause/resume/cancel, redaction and actual consent; no bypass. The current source fix for hidden markers still needs installed-runtime acceptance.  
**A04** Confidential evidence/control: [#147](https://github.com/WPUIAI/uiai-engine/issues/147), [#238](https://github.com/WPUIAI/uiai-engine/issues/238), [#299](https://github.com/WPUIAI/uiai-engine/issues/299) and related PRs #175/#237. Ordinary session open must be private; screenshots/evidence must reach the authorized consumer through a safe durable route, including unscoped-mode truthful results.  
**A05** Everyday/browser input parity: [#301](https://github.com/WPUIAI/uiai-engine/issues/301), [#303](https://github.com/WPUIAI/uiai-engine/issues/303), [#305](https://github.com/WPUIAI/uiai-engine/issues/305), plus [#294](https://github.com/WPUIAI/uiai-engine/issues/294). Track files, dialogs, tabs, frames, coordinate actions, clear-field and trusted input, video/audio as independently evidenced capabilities; do not block core read-only WebMCP experiments on unrelated media coverage.  
**A06** Distribution and endpoint integrity: [#295](https://github.com/WPUIAI/uiai-engine/issues/295), [#296](https://github.com/WPUIAI/uiai-engine/issues/296), [#297](https://github.com/WPUIAI/uiai-engine/issues/297), [#298](https://github.com/WPUIAI/uiai-engine/issues/298), [#249](https://github.com/WPUIAI/uiai-engine/issues/249). Native OpenClaw plugin, dynamic endpoint discovery, CLI failure semantics, portable packaging, and cross-client route parity.  
**A08** Canonical browser session lifecycle and recovery — **P0 explicit gate**: [#76](https://github.com/WPUIAI/uiai-engine/issues/76). Make one registry/lease owner authoritative for session allocation, page attachment, close/reap and crash recovery; reconcile real CDP targets versus counters; prevent split-brain, zombie pages, stale control leases and two actors mutating one session. Prove forced close, concurrent calls, lease expiry, engine restart, cross-node disconnect and recovery with independent runtime diagnostics and no accidental effect replay. This is a blocking dependency for reliable governed execution and reusable skills.  
**A09** Existing authenticated-browser/profile reuse — **P1 explicit gate**: [#143](https://github.com/WPUIAI/uiai-engine/issues/143), [#145](https://github.com/WPUIAI/uiai-engine/issues/145), #81/#82. Support narrowly approved *existing session attach* when technically feasible and separately admitted *isolated disposable auth profile* workflows; never silently clone a local owner's credentials into remote Chromium or assume a ChromeOS tab is the VPS browser. Authenticate/authorize target ownership, isolate context/cookies/storage by owner and origin, inject credentials only through brokered opaque handles, preserve explicit takeover/consent and provide revocation/expiry/zero-residue receipts. Unsupported local browser attachment must return a truthful alternative, not an unsafe fallback. Prove two owners, login/reopen, passkey/challenge handoff, revocation and cleanup with no secret-bearing logs or artifacts.  
**A10** Patched Go toolchain and standard-library vulnerability clearance — **P0 CI/release blocker**: [#306](https://github.com/WPUIAI/uiai-engine/issues/306). [PR #290 CI run 43153067939](https://github.com/WPUIAI/uiai-engine/actions/runs/43153067939) passed Go vet/tests but failed `govulncheck`: **GO-2026-6859**, reachable `net/http@go1.26.10`, scanner-reported fix `go1.26.11` (HTTP/1 `MaxBytesReader` header race). Update the canonical `go.mod` Go baseline and `docs/RELEASE_DEPLOY_RUNBOOK.md` after official patched-toolchain availability is checked; verify GitHub setup-go, AppVeyor native Windows toolchain, OVH side-by-side build, relevant Mac/ARM release lanes, actual installed engine binary and repeat exact-commit `govulncheck`. Do not suppress the scanner or claim installed remediation from a green source build. Existing [#262](https://github.com/WPUIAI/uiai-engine/issues/262) is a different, closed `x/net`/HTTP2 dependency issue.  
**A07** Commercial release licensing: respect [release blocker summary](https://github.com/WPUIAI/uiai-engine/blob/main/docs/UIAI_RELEASE_BLOCKER_SUMMARY_2026-08-01.yaml). New evaluator/customer delivery is denied until signed entitlement, universal pre-effect route/limit gating, authentication separation, and adversarial tests pass. Internal bounded proofs may continue within actual authority.

**A acceptance:** #306 Go security gate passes with an officially patched compiler and exact installed/released runtime verification; #76 canonical session lease/reconcile/recovery passes an actual crash/disconnect/concurrency test; #143/#145 authorized authenticated-session attach or isolated-profile equivalent proves cookie and actor segregation, revocation and zero-residue cleanup; authenticated internal open→read→act→screenshot→diagnose→close succeeds on real installed runtime; stalled/false-auth/unsafe-share/closed-port paths fail with truthful errors; no accidental public session control or credential disclosure; independent release evidence recorded.

### B. Unified MCP / WebMCP / discovery and Cockpit — owner: UIAI Engine PR #290
**B01** Reconcile AFB-020, Cockpit 002/004/005/008, current operation/registry types, interface versions and OpenClaw/Pi adapters before new schemas.  
**B02** Per-document WebMCP capability probe; report supported/unsupported, browser flags, origin, navigation generation, frame and exact registration revision.  
**B03** Lazy, ranked origin-tool discovery with bounded result and dynamic invalidation. Treat all page-origin metadata as untrusted and ephemeral; no globally trusted page-tool catalog.  
**B04** Governed invocation through existing browser action boundary; validate input schema, least data/egress, Focusa authority, UIAI entitlement, actor lease and approval, then independently verify effects. Preserve DOM/accessibility/CDP/visual fallback without privilege escalation.  
**B05** Reconcile HTTP/Pi/OpenClaw/CLI/MCP parity; modern MCP protocol compatibility requires real conformance, not a changed advertised version; authenticated remote MCP transport is optional and independently gated.  
**B06** **Lean federated discovery, added scope:** a small configurable list of external WebMCP-capability directories; query/cache normalized metadata via the existing Go provider/search/cache capabilities, with staleness and provenance; no new mandatory service, crawler, database, global schema registry or LLM ranker. Off-site discovery is a candidate locator; each candidate must be verified inside the actual page before any action. Deterministic ranking initially: task fit, allowed origin, freshness, historical verified reliability, estimated cost; popularity only a tiebreaker.  
**B07** Cockpit Live Tool Inspector, actual session/Workstream identity, event cursor, risk preview, receipts and fenced takeover; preserve low-RAM Chromebook web/PWA and optional Tauri implementation in PR #290. Native browser skill operation must work when Cockpit is absent.

**B acceptance:** one supported fixture exposes an origin tool and an unsupported fixture uses semantic browser fallback; navigation/tool-schema changes invalidate candidates; malicious tool descriptions cannot grant authority; Pi/MCP/OpenClaw and Cockpit observe the same authorized session/action/evidence refs.

### C. Verified workflow → compact skill / routine — owner: existing Wirebot Fusion Engine path + Focusa skill compiler
**C01** UIAI emits a **candidate**, not an installed skill, on eligible verified execution: goal, typed input slots, origin/tool and version constraints, exact actor/tenant/scope refs, preconditions, step skeleton, expected effects, postcondition predicates, receipts, fallback/recovery, idempotency class, side-effect and approval posture, redacted source trace, measured cost and confidence/limitations.  
**C02** Candidate quality gate: replay or independent validation on owned fixtures, source/DOM drift, prompt injection, tool capability/installed-availability, no embedded passwords/cookies/cross-tenant identifiers. A single successful run is insufficient for global promotion.  
**C03** Reuse Focusa Spec 34/140 generated contracts and the existing Wirebot Fusion Engine direction to compile short, progressive-disclosure OpenClaw-compatible SKILL.md plus machine-readable manifest and tests; no duplicate compiler, registry or hidden worker runtime.  
**C04** Separate *skill* from *recurring routine*: portable owner-neutral SOVOS Routine Template → owner-specific Blueprint → admitted Routine Instance with schedule/role/capability grants; ordinary one-shot skills require no new scheduler.  
**C05** Load/upgrade/revoke via the actual OpenClaw skill-loader boundary; preserve source/digest/version and rollback, scope, tool availability and approval. A generated skill must not silently execute external side effects.

**C acceptance:** skill candidate produced and independently validated → newly started Wirebot/OpenClaw session discovers only the qualified compact skill and executes its authorized UIAI/API route; negative tests prove unavailable dependency, drift, cross-tenant reuse and revoked approval do not result in unsafe execution.

### D. Wirebot organism integration — owner: Wirebot Core + Focusa + SOVOS (source-specific)
**D01** Runtime: use native OpenClaw conversations/skills with exact session and tool binding. Reconcile [Wirebot Core #79](https://github.com/Startempire-Wire/wirebot-core/issues/79) and [UIAI #249](https://github.com/WPUIAI/uiai-engine/issues/249), not an independent model-only reply path.  
**D02** Work: Focusa exact Project/Workstream/Workpoint/Foreman admission, Evidence proposals, receipts and actual receiver acceptance; direct ad-hoc Focusa usage remains legitimate. Installed daemon schema/version must actually support each invoked operation. See [UIAI #216](https://github.com/WPUIAI/uiai-engine/issues/216).  
**D03** Follow-through: Master Loop/Perpetua connects qualified skill executions to real outstanding goals, accepted outcomes, next steps and recurrence; no second task/scheduler authority.  
**D04** Remember: Portable Memory WRITE/MANAGE/READ handles eligible procedural knowledge, corrections, freshness and deletion propagation; Mem0/Letta remain selected permitted providers where configured, not deleted by inference.  
**D05** Improve: Master Observer/Focusa metacognition evaluates actual successes, stale tools, corrective feedback, false-done rates, retries, token/browser time and verified leverage. It **proposes** repairs/promotions and preserves dissent/uncertainty; it does not auto-grant authority or create new processes by existing.  
**D06** Surfaces/deployment: Wirebot App #20/#21, Discord, voice, Cockpit, WINS and API/CLI project the same owner refs and skill lifecycle; UI optional. Respect supported self-hosted Sovereign API packaging, operator authentication and tenant isolation.  
**D07** Fresh native OpenClaw connector + learned-skill proof — **P1 explicit gate**: [UIAI #249](https://github.com/WPUIAI/uiai-engine/issues/249), [Wirebot Core #79](https://github.com/Startempire-Wire/wirebot-core/issues/79). On an actually installed owner-scoped Wirebot gateway, a newly started OpenClaw session must discover the UIAI connector via its native plugin/tool loader, open/read/act/verify/close using the **same UIAI browser ownership**, then discover a validated compact skill from C03/C05 and invoke it correctly. Assert one agent reply, correct live endpoint/version/tenant, scoped Focusa receipts, image delivery, cleanup, explicit degraded mode if UIAI is absent, and no silent fallback to another browser, stale tool cache, preloaded transcript skill or model-only response. Add restart and second-tenant test witnesses.

**D acceptance:** D07 fresh installed OpenClaw UIAI plugin/skill discovery is demonstrated with a real native tool call and actual selected endpoint rather than static repository metadata; exactly one authorized Chief-of-Staff turn in OpenClaw routes through qualified skill → UIAI/Go API; Focusa governs; history, memory, follow-through and Observer analysis are available through headless path and optionally the App; no duplicate reply, outcome, scheduler, agent or memory authority.

### E. Integrated acceptance, operability and release — all source owners with independent verifier
**E01** Record baseline before optimizing: verified task success, false-done, latency, token/cost, incident recovery, browser pressure, secret leak denials, stale directory candidates, duplicate effects and operator burden; do not invent performance improvements.  
**E02** Synthetic/owned **SendyKit-like campaign draft** fixture: discover through WebMCP or semantic fallback; create a draft (no real campaign send); verify stored draft; emit candidate; qualify; compile; load from a new OpenClaw session; replay and verify; retain qualified memory; Master Loop records follow-through; Observer produces improvement proposal.  
**E03** Negative fixtures: deceptive origin tools, cross-frame mismatch, stale navigation, revoked grants, missing provider, auth wall, screenshot failure, engine restart, disconnected Focusa, duplicate submit, version/endpoint drift, customer A vs customer B and local/remote/OVH Tailscale separation.  
**E04** Prove actual deployed service + installed OpenClaw loader + authentic receiving UI/CLI/client, then repeat on a properly isolated full-Sovereign operator environment. Stage only those irreversible customer operations requiring explicit action-specific consent.  
**E05** Preserve dated evidence/receipts, exact source/installed revisions, reversible changes, runtime status and owner-visible demonstration. Never close the workset on docs, CI-only green, a healthy process, a PR merge or agent self-report.  
**E07** UIAI Go vulnerability release proof: reconcile [#306](https://github.com/WPUIAI/uiai-engine/issues/306) on the exact release SHA. Capture `go version`, the `go.mod` baseline, `govulncheck` zero reachable issues for the originally failing advisory, native Windows/macOS/Linux supported build toolchain versions, signed artifact digests, and installed binary `go version -m` or equivalent. Failed/unknown any required lane blocks public release; do not treat the existing WebMCP proposal CI as release-ready.  
**E06** Explicit three-gate release matrix — #76 canonical session ownership/recovery (P0), #143/#145 authenticated profile/session capability with secret-safe isolation (P1), and #249 fresh OpenClaw connector plus compiled-skill invocation (P1). Each gate requires an owner repo, current branch/installed binary, failing-before/passing-after acceptance fixture, negative test, immutable evidence/receiver receipt, rollback and linked original issue. Do not report the unified workset complete while any one of these is unproven; do not require unsupported local-tab adoption when a secure isolated-provider-auth flow is the approved alternative.

## 3. Dependency DAG and safe parallelism

```text
A01 current-state audit ──────┬─→ A02–A09 hardening / A08 session lifecycle P0 ─────────────┐
                              ├─→ B01 existing-contract mapping ─┬─→ B02/B03 read-only discovery
                              │                                    ├─→ B06 cache/directory lookup (independent)
                              │                                    └─→ B04 execution → B05/B07 parity/UI
                              └─→ C01 candidate schema (on existing verified DOM path)
                                         ↓
                                    C02 validation → C03 compiler → C05 OpenClaw fresh-session proof
                                         ↓                       ↓
                                    C04 optional recurring routine / D01–D07 bindings
                                         └───────────────→ E02/E03 integrated tests
A-critical release gates + E06 three explicit gates ──────────→ E04/E05 production acceptance
```

- Browser defects and WebMCP experiments can proceed concurrently. Do not block safe read-only capability discovery on unrelated feature completion.
- Candidate/skill compaction can be exercised using **existing DOM browser actions** before native WebMCP is ready.
- Do not require Cockpit, a directory provider, portable skill promotion, a media worker or GUI for ordinary safe browser operation.
- Hard pre-effect authorization and privacy gates are never bypassed to increase parallel throughput.
- Track exact availability as **available / qualified / partial / blocked / failed / not_proven** per node, with independent test witnesses.

## 4. Single end-to-end done condition

An authorized Wirebot/OpenClaw user asks to prepare a campaign draft on an owned fixture. UIAI chooses an available WebMCP tool or ordinary browser action and creates the draft; Focusa receives valid proof and verifies the intended draft exists. The existing compiler pipeline produces a redacted, compact, dependency-checked, governable skill. **A new native OpenClaw turn** discovers and reuses that skill with an actual UIAI agent tool; the expected effect is checked, and an actual receiving consumer confirms the result. Portable Memory records only qualified learning, Master Loop/Perpetua records actual follow-through, and Master Observer produces an evaluated bounded improvement proposal. Repeat with another isolated tenant; prove negative auth/staleness/side-effect paths and the full-Sovereign topology. Only then mark the complete workset accepted.

## 4.1 Owner-added acceptance gates — preserve in final release report

| Gate | Existing issue owners | Concrete independent witness | Release posture |
|---|---|---|---|
| **P0 session ownership and recovery** | UIAI [#76](https://github.com/WPUIAI/uiai-engine/issues/76) | Crash/restart, expiry, concurrent close/act, multi-node reconnect; real page/lease count reconciled; no zombie or duplicate effect | Mandatory for production browser/skill correctness |
| **P1 authenticated session and profile reuse** | UIAI [#143](https://github.com/WPUIAI/uiai-engine/issues/143), [#145](https://github.com/WPUIAI/uiai-engine/issues/145) | Authorized operator attach **or** admitted isolated disposable profile where applicable; two distinct tenants; brokered credentials, revoke/cleanup/zero residue; no secret or cookie crossover | Required for claims of authenticated-site automation; unsupported browser attach must be explicitly labeled |
| **P1 native OpenClaw connector + learned skill** | UIAI [#249](https://github.com/WPUIAI/uiai-engine/issues/249), Wirebot Core [#79](https://github.com/Startempire-Wire/wirebot-core/issues/79) | Fresh installed OpenClaw turn loads real UIAI tool and a qualified compact skill, executes, verifies, and emits scoped Focusa receipt; outage/restart/tenant negative tests pass | Mandatory for flagship Wirebot end-to-end acceptance |

These are explicit validation gates in existing tracks, **not three new services or a new architecture**. Do not prematurely claim production readiness because they are listed in the specification. Their implementation and independent runtime proof remain outstanding.

## 5. Implementation handoff and tracking hygiene

- Use **one cross-product orchestration issue** owned by Wirebot Core; make this file PR #290's browser-side planning companion. Link existing GitHub issues/Beads/Focusa Workstream refs rather than duplicating tasks.
- No *live Focusa Workstream ID* is invented. The actual owner/operator agent must bind the workset to an existing Project scope and create/admit the one Workstream with real returned IDs and current authorized operations.
- For every atomic implementation task record: owner repo, existing issue/Bead, dependency, concrete producer/consumer, expected receipt, side-effect/risk, done criterion, test/rollback and installed-version evidence.
- Existing PR #290 remains a design umbrella; source changes are separate small owning-repo PRs/commits. This workset is active for implementation planning, but not an implicit approval to merge unrelated drafts, distribute customer software, publish customer data or grant new permissions.
- **Before claiming completion:** reconcile with Wirebot Core #79, SOVOS PR #35, Wirebot App #20/#21, UIAI #65/#249/#290, Focusa Spec 140, and UIAI licensing release blockers. New repo or service may be considered only after measured evidence shows existing capabilities cannot supply the required function.
