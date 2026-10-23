# UIAI Engine Web Runtime Leap — Agentic, Autonomous, Hyper-Personal

**Document number:** `UIAI-ENGINE-010`
**Status:** proposed normative v1.0 (operator-mandated scope; activation on Wave-1 bead creation)
**Date:** 2026-08-25
**Repository:** `WPUIAI/uiai-engine`
**Primary parent:** `UIAI-COCKPIT-005` (browser product contract), `UIAI-COCKPIT-009` (identity gap-closure method)
**Also amends:** `UIAI-COCKPIT-004`, `UIAI-COCKPIT-008`, `SESSION_API.md`, `BROWSER_DIAGNOSTICS_SPEC.md`, `CAPTCHA_SOLVER_SPEC.md`, `FPV_REMAINING_GAPS_SPEC.md`, `.github/workflows/ci.yml`
**Focusa counterpart:** Focusa `docs/181-focusa-browser-runtime-integration-spec.md` (mirrored taskgraph)

## 0. Decision

The engine stops being a CDP tool server and becomes an **agent-native web runtime**: goal-level verbs, governed autonomy, persistent web-state continuity, persona composition, perception anchors, and native citizenship in the Focusa event/evidence ecosystem.

A deliverable is not complete because a type, route, flag, or document exists. Completion requires operational behavior through the **canonical agent/browser path AND the Cockpit path**, plus the regression proof named in its closure condition.

## 1. Authority and safety boundary

UIAI Engine owns browser processes, profiles, contexts, observations, actions, diagnostics, network identity, challenge handling, and browser-runtime evidence. Cockpit presents and governs them. **Focusa may authorize, scope, and link work; it does not implement browsing.**

Nothing here authorizes account abuse, access-control bypass, deceptive identity claims, or activity prohibited by a destination's rules. Persona coherence (#87/#88) represents *approved* identities coherently; challenge handling remains bounded by the existing CAPTCHA solver policy (#70/#89). Budget governors and writer-leases are safety surfaces, not conveniences: an autonomous task that cannot stay inside its cap must pause and escalate, never improvise.

## 2. Evidence basis

1. 2026-08-25 production wedge incident (#96): proxy-pool total outage, unbounded lock, passive LB — root-caused and fixed (e5cd457).
2. Session-lifecycle audit commits already on main (13a111c, a4f8c67 reconciler).
3. Static review of current `internal/vision`, `internal/routes`, MCP bridge, FPV transport.
4. Operator directive 2026-08-25 mandating full implementation of the discussed leap, nothing omitted, fully documented.

## 3. Claim matrix (normative)

Every claim carries: owner surface, phase, regression proof, closure condition. Phases map to Waves (§5).

| ID | Claim | Owner surface | Phase | Regression proof | Closure condition |
|----|-------|---------------|-------|------------------|-------------------|
| C-010-01 | Intent verbs: `extract{schema}`, `act{do+assert}`, `task{outcome,caps}` return typed results with confidence + receipt | `internal/routes`, schemas, MCP/OpenAI/Pi projections | W2 | golden fixtures per verb; MCP projection test; cockpit panel invokes `extract` | agent completes a 3-page extraction using only intent verbs |
| C-010-02 | Engine emits `focusa.stream_event.v1` envelopes (event_id/cursor/sequence/scope/invalidate) | `internal/events` new emitter | W3 | schema-validate emitted frames against Focusa fixture | workforce notification center ingests engine events unmodified |
| C-010-03 | Budget governors: ms/pages/bytes caps; auto-pause on captcha/auth-wall emits `budget.paused`; resume token | session layer + routes | W2 | forced-cap test pauses mid-task; envelope asserts event | paused task resumes by token without duplicate side effects |
| C-010-04 | Zero bare client aborts: all long ops deadline-bounded with structured error envelopes + retry semantics | MCP bridge + routes | W1 | fault-injection: stalled upstream → envelope not abort | chaos suite shows 0 raw aborts across 200 mixed calls |
| C-010-05 | Cost stamps on every envelope: duration_ms, bytes_in/out, pages_touched | middleware | W1 | envelope snapshot test | aggregate endpoint returns per-session totals |
| C-010-06 | Warm identity fleet: keep-alive per profile class; first-action <300ms warm | pool/launcher config | W2 | bench harness cold vs warm p50 | bench artifact committed; soak 30min leak-neutral |
| C-010-07 | Web State Continuity: checkpoint/rollback/branch cookies+storage per identity; crash-resume journal | storage-state v2 + journal | W4 | round-trip test; kill -9 mid-task resume test | resumed session completes task with identical final state hash |
| C-010-08 | Differential DOM: since-revision patch stream | snapshot pipeline | W4 | payload-reduction bench ≥5× | SSE clients render identical tree vs full snapshot |
| C-010-09 | Artifact-ref screenshots default; base64 only `inline=true` | capture path | W2 | envelope shape test | transcripts contain refs, not pixels |
| C-010-10 | Raw-frame FPV over WebSocket ≤200ms 1080p glass-to-glass | FPV transport (#68) | W4 | latency probe harness | FPV_REMAINING_GAPS item closed |
| C-010-11 | Pressure-based recycle (tree RSS/heap) alongside count | pool recycler | W1 | synthetic leak test recycles before cgroup stall | no MemoryHigh thrash in 24h soak |
| C-010-12 | Queue priority classes interactive>batch | queue | W1 | mixed-load bench: interactive p95 < batch p95 under saturation | bench artifact committed |
| C-010-13 | Egress circuit-breaker + `direct_egress_fallback` + per-domain policy | egress pool | W1 | fail all proxies → typed fast-fail or direct fallback per flag | zero nav hangs during simulated outage |
| C-010-14 | Persona construct: profile(#88)+stealth(#87)+Broker JIT(#82), redaction at boundary | identity subsystem | W4 | injection redaction test; coherence checks from 009 | persona task runs end-to-end with zero secret leakage in envelopes/logs |
| C-010-15 | Perception anchors: semantic @refs + versioned auto-rebind post-SPA | vision snapshotter | W5 | re-render rebind test | wrong-element click rate 0 on fixture site |
| C-010-16 | Auth preflight per domain before task | session open hook | W4 | auth-wall fixture triggers `auth.required` pre-action | tasks never discover auth walls mid-flow |
| C-010-17 | Evidence chains: hash-linked receipts per nav/mutation/capture; exportable timeline | capture layer | W5 | tamper test breaks chain detection | timeline exports and verifies |
| C-010-18 | Consensus reads across N personas → diff + trust score | orchestrator | W5 | N=3 diff fixture | trust artifact schema validated |
| C-010-19 | Mesh workers (tailnet #93) + elastic container profile (#78) | multi-host registry | W5 | cross-host session open through registry | fleet picker lists remote worker |
| C-010-20 | Cockpit contract: every new surface ships typed API + adapter card/panel + operator control where mutation-adjacent + document-register entry + cockpit-path regression | apps/cockpit adapters | each wave | cockpit vitest/e2e per surface | UIAI_COCKPIT_DOCUMENT_REGISTER row added same PR |
| C-010-21 | Browser Fleet widgets (Focusa spec180 family) consume engine fleet/health via daemon bridge | focusa-side + bridge | W3 | widget renders live pools from fixture stream | wall-mode displays fleet without mutation authority |
| C-010-22 | Work Loop browsing beads: silent-session browser workers; lease per (page,scope); outcomes as evidence citations | focusa silent-sessions + engine leases | W3 | two agents same page → second waits on lease | bead closes with citation from real run |
| C-010-23 | Web-state checkpoints usable as Workpoints; compaction-safe | focusa workpoint + C-010-07 ids | W4 | packet contains checkpoint id; compaction retains handle | resume-after-compaction continues browsing context |
| C-010-24 | North-star prefetch: Focusa current-ask feeds Speculation-Rules prerender (opt-in) | launcher hooks | W5 | prerender hit-rate counter >0 on scripted flow | first-nav latency improvement recorded |
| C-010-25 | Identity ↔ device pairing binding; walls show acting persona read-only | pairing + personas | W5 | paired persona acts; wall labels correctly | no persona action without bound device |
| C-010-26 | Ops floor: Caddy active checks (5s/2s/fail_duration 10s); Prometheus `/metrics` (pool,queue,egress,budgets); rod pin policy + provenance note; structured dedup logs | ops configs + routes | W1/W5 | hung worker pulled ≤10s; scrape contains series | runbook updated in ENGINE_RELEASE_DEPLOY_RUNBOOK |
| C-010-27 | **Browser-native Audit Ledger**: hash-chained browsing-art ledger (session/nav/act/input-class/capture/challenge/budget/lease/state/egress/error); writer-boundary redaction; values+cookies+tokens never serialized | `internal/auditledger` | W2 | tamper→verify fails; redaction fuzz clean | `/api/audit`+SSE tail+export chain-verify green |
| C-010-28 | **Audit Time Travel**: rows pair artifact_ref+dom_revision_ref+metadata bundle; LRU evict heavy artifacts, hashes permanent | capture pipeline + `/api/audit/{seq}/context` | W4 | random-instant reconstruction test | Cockpit scrubber (filmstrip/DOM/metadata/T005 keyboard/jump-live/moment-export) |
| C-010-29 | **Licensing closure** vs wpuiai.com authority per matrix doc; fail-closed; offline grace bounded; recovery exempt | entitlement middleware + new routes | W1 gates; waves enforce rows | bypass-resistance suite (stale/refunded/unbound denied); live probe evidence | operator sign-off; zero unguarded premium paths |
| C-010-30 | **Stealth hardening**: per-persona persistent fingerprint seeds — full UA-CH set, platform/UA coherence (009 §defects), fonts, WebGL/WebGPU vendor strings, stable canvas/audio noise, plugins, hardwareConcurrency/deviceMemory realism, timezone/locale/geo consistency with egress class | browserprofile + launcher flags | W4 | 009 baseline defect suite passes zero-exposure on fixture panel (creepjs-style self-audit) | seeded persona yields byte-stable fingerprint across restarts |
| C-010-31 | **Adaptive IP rotation**: rotate-on-challenge/flag signals, per-domain reputation memory, sticky-vs-rotate policies, cooldown auto-tuning from solve rates | IPPool + policy engine | W4 | simulated flag event rotates mid-task without session loss | flagged-domain solve-rate improves vs static baseline bench |
| C-010-32 | **Challenge resolution upgrades**: provider/type coverage matrix, perception-assisted solving (P2.2 anchors), budget-aware retry ladders | captcha subsystem (#70) | W5 | coverage matrix test per supported type | solver success rate tracked in metrics; failures typed envelopes |
| C-010-33 | **Anti-detection telemetry loop**: detection-event corpus (site, leaked signal, persona build), redteam harness regressing against top detectors | observability + fixtures | W5 | injected detector page catches nothing on hardened personas | corpus drives ≥1 shipped coherence fix |

## 4. Cockpit integration contract (amends COCKPIT-004/005/008/009)

For every claim touching an agent-visible surface:

1. **API first:** typed HTTP route + JSON schema under `/api`, registered in capability/tool projections.
2. **Adapter card:** cockpit adapter (phase0-style) with non-null `contract_ref`; panel or card lands in Cockpit following T005 accessibility bar (keyboard/filter/aria/confirm pattern set by T005-08.x).
3. **Operator control:** anything mutating (personas, budgets overrides, mesh enablement) gets explicit Cockpit control with confirm semantics; read-only surfaces render as cards.
4. **Documents:** same-PR row in `UIAI_COCKPIT_DOCUMENT_REGISTER.md`.
5. **Regression through cockpit path**, not just API tests — mirrors 009 §0 completion rule.
6. **Modes respect 005:** `detect/no_detect/operator/research/auto` semantics unchanged; personas compose *within* mode boundaries; `no_detect`/`operator` remain coherent-identity representations, never deception tooling.

### 4c. Licensing closure (C-010-29 normative)

Every UIAI-ENGINE-010 surface enforces the **wpuiai.com license authority** per `docs/UIAI_ENGINE_010_LICENSING_MATRIX.md`. Premium pillars — personas/stealth (P2.1), consensus N>1 (P2.5), mesh remote workers (P2.6), captcha solver, media jobs — require an entitled tier. Enforcement is fail-closed on missing/expired/refunded/unbound leases, bounded offline grace, recovery surfaces exempt. Bypass-resistance tests mirror Focusa Spec152F patterns.

## 5. Waves

- **W1 (floor):** C-010-04,05,11,12,13,26(active-checks half)
- **W2:** C-010-01,03,06,09
- **W3:** C-010-02,21,22
- **W4:** C-010-07,08,10,14,16,23
- **W5:** C-010-15,17,18,19,24,25,26(metrics remainder)

Behavior-changing items ship behind config flags (`features.web_runtime.*`) with defaults preserving current behavior until their wave's bench+soak evidence lands.


> Licensing (C-010-29): C-010-30..33 are `optional_premium` — fail-closed without entitled tier; enforcement lands with each row.

## 6. Test & evidence plan

Per claim: producer unit tests in-package; consumer test through canonical agent path (MCP) AND cockpit path where applicable; schema-version interop test when envelopes change; live e2e on OVH workers with durable evidence handles (artifact refs, journal lines, bench files under `docs/evidence/uiai-engine-010/`). No claim closes on prose.

## 7. Non-goals

Rendering-engine replacement; credential custody outside Secrets Broker; autonomous mutations outside Focusa governance when a Focusa context exists; CAPTCHA circumvention beyond sanctioned solver policy.

## 8. Rollback

Each wave is binary+config reversible: flags default-off pre-evidence; previous release binaries retained as `.prev-YYYYMMDD` per deploy runbook; Caddy changes are config-file revert + reload.
