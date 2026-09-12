# 106q — UIAI Evidence PWA Completion CallGraph

**Parent:** Issue #106 — Evidence PWA authority. **Mission:** reach a fully installed, cross-surface, independently verified EPWA without conflating evidence, review, completion, provider closure, or settlement.

## CallGraph contract

Each node has stable ID + descriptor, owner surface, dependencies, done condition, and required evidence. Candidate publication is not completion. A join passes only after an independent verifier consumes committed hashes and immutable evidence refs. Maximum two repair/reproof cycles per join.

```text
EPWA-CG-ROOT
├─ FOUNDATION-JOIN [T01–T05 implemented; independent settlement pending]
├─ RUNTIME-FANOUT
│  ├─ T06 Judge runtime → T06-J
│  ├─ T07 Action/collaboration runtime → T07-J
│  ├─ T08 PWA runtime → T08-J
│  └─ T09 Derivative runtime → T09-J
├─ SURFACE-FANOUT (after T08 runtime)
│  ├─ S01 UIAI packet API + registry
│  ├─ S02 Pi parity
│  ├─ S03 Cockpit board + settings
│  ├─ S04 Chrome viewer + settings
│  ├─ S05 Desktop Canvas object + settings
│  └─ S06 Veragensia public/private preview + durable mount
├─ PLATFORM-FANOUT
│  ├─ T10 connectors
│  ├─ T11 API/CLI/MCP/OpenAPI parity
│  └─ T12 operations/migration/release
├─ CLOSURE-CONTROL-FANOUT
│  ├─ CG-33 Review/Adjudication + truthful-proof gate
│  ├─ CG-34 working-agent closure awareness/parity
│  ├─ CG-35 autonomous work-state binding/recovery
│  └─ CG-36 project registry/bidirectional closure index
├─ T13 Focusa authority integration
├─ T14 installed cross-environment dogfood
└─ T15 standards/benchmark/public claims
```

## Node table

| ID + descriptor | Depends | Owner | Exact done condition | Evidence atoms |
|---|---|---|---|---|
| CG-01 — Freeze current authority baseline | #106 specs | UIAI | Canonical spec family, current commits/PRs/issues, dirty-state exclusions and compatibility floor recorded | baseline hash; exact file inventory |
| CG-02 — T01 immutable artifact independent join | CG-01 | UIAI verifier | Independent validator confirms artifact contract and golden identity | commit + fixture digest + review result |
| CG-03 — T02 store independent join | CG-02 | UIAI verifier | crash/restart/quarantine/retention/backup proof independently accepted | power-loss, restore, GC receipts |
| CG-04 — T03 hostile-content independent join | CG-03 | security verifier | adversarial fixtures, sanitization, redaction, active-content/secret/PII/OCR/transcript/media leak checks, CSP/CSRF/CORS/SSRF/archive controls, and public/private disclosure checks pass | threat corpus + scan report + redaction/active-content evidence — **accepted** (two-cycle budget exhausted; `106x` re-review VERDICT: PASS at `21c67290` with hardening carried forward) |
| CG-05 — T04 capture assembly independent join | CG-04 | multimodal verifier | deterministic capture/media/omission/selection manifest, independent-ground-truth/anti-cherry-pick, event-range continuity, modality fidelity, permission, and unavailable/unknown proof accepted | 30-run digest + modality/omission/ground-truth evidence | — **accepted** (cycle-1 FAIL → cycle-2 ACCEPT; two-cycle budget used; repair PR #229 at ; record ; next join CG-06)
| CG-06 — T05 crypto/time/custody independent join | CG-05 | crypto verifier | target/account identity, actor/delegation, canonical-state snapshot, corrections/retractions, signing/time/federation/custody, imported-action distrust, proof-of-absence, air-gapped/link-rot, and human-readable identity proofs accepted | key-rotation/import/revocation/custody/absence proof |
| CG-07 — Foundation join | CG-02..06 | independent join | all five exact results valid, unexpired, scope-matched | joined Evidence Artifact |
| CG-08 — T06 Judge runtime | CG-07 | UIAI | request/view/result runtime, isolated execution, appeals, budgets, drift/calibration, frozen hash-bound Judge View over untrusted content, capability-mismatch/indeterminate outcomes, exact citations, no judge-to-completion write, human/machine projection parity | producer + consumer tests; freeze/mismatch/citation/no-write fixtures |
| CG-09 — T06 independent join | CG-08 | independent judge | frozen-information review passes | judge result + exact citations + freeze/mismatch/no-completion-write proofs |
| CG-10 — T07 Action runtime | CG-09 | UIAI + Focusa | operation registry, preview/confirm, anti-replay, result/reconciliation, item-scoped review/adjudication/reproof/follow-up/export transport, untrusted-content handling, and no-local-completion authority implemented | ambiguity/partial/replay/rejection/reproof E2E + authority audit |
| CG-11 — T07 independent join | CG-10 | independent judge | no action/review path can assert completion | action trace + authority audit |
| CG-12 — T08 PWA runtime | CG-07 | UIAI | registry, Overview/Evidence/Timeline/Inspect/Developer, PWA/offline/subpath/LowMem/localization/access states, four-viewport responsive matrix, relative/same-origin portable routes, reverse-proxy origin derivation with no hard-coded hostname, restart/session-closure survival | browser matrix + offline replay + origin/portability-survival artifacts |
| CG-13 — T08 independent join | CG-12 | accessibility/security verifier | WCAG 2.2 AA, CSP, performance, portability, reverse-proxy origin derivation, and restart/session-closure survival accepted | axe/manual/diagnostic artifacts + viewport/origin/survival matrix |
| CG-14 — T09 derivative runtime | CG-13 | UIAI | print/PDF/email/Markdown/HTML/JSON/CSV/archive/slides with deterministic identities, audience/redaction/destination/approval/idempotency/Receipt governance, external-retention warnings, and typed negative/partial-delivery results | viewer/client matrix + hashes + negative/partial-delivery proofs |
| CG-15 — T09 independent join | CG-14 | independent document verifier | accessibility, licensing, delivery truth, archive safety, retention-warning, and negative/partial-delivery behavior accepted | PDF/email/archive proof + partial/negative-delivery evidence |
| CG-16 — S01 canonical packet API and registry | CG-13 | UIAI | automatic evidence-turn publication before cleanup; bounded list/search/inspect/verify/resolve/edges/closure projection/serve; complete binding refs and digests; typed published/blocked/failed/not-applicable results; corrupt/degraded/restart/rebuild states | API E2E + restart/rebuild + failure-result proof |
| CG-17 — S02 Pi complete parity | CG-16 | Pi | URL-first capture plus list/inspect/verify/resolve/settings/explain; typed published/blocked/failed/not-applicable delivery results; confirmation for mutations | extension contract tests + blocked/failed delivery fixtures |
| CG-18 — S03 Cockpit board/settings | CG-16 | Cockpit | preview cards, filters, provenance, verification, all settings domains, conflict/reset/degraded states | component + UIAI visual proof |
| CG-19 — S04 Chrome viewer/settings | CG-16 | Focusa extension | View Evidence after capture, trust badge/details, project scope, canonical settings round-trip | Chrome build + real browser proof |
| CG-20 — S05 Desktop Canvas object/settings | CG-16 | Focusa Desktop | Canvas evidence object, recent board, scope binding, preview/details/settings/offline handoff | Desktop tests + visual proof |
| CG-21 — S06 Veragensia durable EPWA mount | CG-16 | Veragensia | `/evidence/` survives container recreate, uses canonical packet API, public fixture/private live modes separated, desktop navigation present | recreate + HTTP + zero-residue proof |
| CG-22 — Surface parity join | CG-17..21 | independent consumer verifier | same packet/settings revisions, binding identity, review/closure posture, forward/reverse edges, uncertainty, and safe next action render consistently across five surfaces without UIAI shadow authority | cross-surface contract matrix + closure/unknown parity proof |
| CG-23 — Settings completeness | CG-16 | UIAI | lifecycle/storage/image/video/presentation/access/privacy/verification/performance/offline/integration schema + inheritance + receipts | global/project/workstream round-trip |
| CG-24 — Lifecycle/retention executor | CG-23 | UIAI | pin/archive/expiry/quota/GC/legal-hold semantics implemented; canonical evidence never silently deleted | clock/quota/restart tests |
| CG-25 — Image/video optimization | CG-23 | UIAI media | browser-native derivatives, DPR, budgets, metadata stripping, poster/caption/keyframe requirements | performance and media matrix |
| CG-26 — T10 neutral connectors | CG-11,15,22 | UIAI + Focusa | issue/document/chat adapters, exact destination authority, idempotency/dead-letter/webhook/safe-unfurl, external-retention warnings, and typed negative/partial-delivery results | three live consumer proofs + retention-warning/negative-delivery evidence |
| CG-27 — T11 generated API parity | CG-08..26 | UIAI | one contract drives REST/OpenAPI/CLI/MCP/Pi clients, jobs/cancel/resume/cursors/content negotiation, distribution/review operations with negative/partial results bound to the same contract | cross-harness conformance + negative/partial vectors |
| CG-28 — T12 operations/migration/reliability | CG-24,25,27 | UIAI ops | SLO/doctor/telemetry, resource scheduling/backpressure, legacy migration, format survival, air-gapped/recovery/link-rot handling, fuzz/chaos/power-loss/load/backup/rollback/release and ownership/runbooks | production-consistency five proofs + recovery/compatibility matrix |
| CG-33 — Operator disposition and truthful-proof closure gate | CG-16,18 | Operator via browser + model completion authority | Before any closure decision, the model must supply a typed, scope-matched proof bundle for every required step. A Review Case uses an appointed human or authorized LLM judge according to the Completion Contract; browser controls appear only for an assigned, unexpired, authorized reviewer. Evidence records carry append-only approve/reject/returned-to-model dispositions (authority ref, required reason on rejection, task/work-item refs, supersede chain); disposition state is visible on every record surface (record page, envelope, portable zip). Missing, stale, contradictory, withheld, or unverifiable proof forces pending/returned-to-model, keeps or reopens the task, emits exact proof gaps, and forbids Completion or provider-close receipts; only a proof-gated terminal Review Case decision authorized by the Completion Contract may permit closure; an assigned human can make that decision in the browser and an authorized LLM judge uses the same Receipt path; public record page stays read-only and mutations require authenticated authority; partial decisions, quorum, dispute/escalation, and reviewer lease/expiry are typed states with no silent defaults | proof-gap/return receipts + disposition receipts + record UI visual proof + route tests |
| CG-34 — Working-agent closure awareness and submission-state parity | CG-27,33 | Focusa + Pi/CLI/MCP/PWA/Cockpit/Desktop | Every action-selection, review/closure event, reconnect/resume, handoff, and completion-language boundary exposes the same bounded open-item, review, completion, provider, settlement, blocker, Receipt-cursor, and next-action posture; pending/failed/unknown submissions reconcile and never become success; rejection/changes-requested returns exact bounded repair/reproof work | cross-surface awareness matrix + ten typed failure fixtures covering unknown/reconcile/reopen/rejected/partial/lease-expiry |
| CG-35 — Autonomous work-state binding and recovery join | CG-03,06,10,16,27,28 | Focusa + UIAI | Every artifact binds the exact Project/Workstream/Trajectory/Workset/CallGraph/Workpoint/provider item, assignment/generation, action effects, budget/retry/circuit-breaker, review, Completion, Receipt, settlement, and rehydrate refs; restart/failover/reassignment/unknown effects preserve lineage, avoid duplicate effects, and continue or block without false completion | binding-envelope vectors + restart/failover/reproof dogfood |
| CG-36 — Project registry and bidirectional closure-index join | CG-12,13,16,22 | UIAI + Focusa | Registry/detail views resolve forward and reverse artifact/task/Acceptance/Completion edges; missing/rejected/stale/unverified requirements are visibly ineligible for closure; index rebuild preserves identities/digests; public, bulk, keyboard, offline, and scope/authority behavior remains safe | forward/reverse index proof + corruption rebuild + public/bulk/accessibility matrix |
| CG-29 — Focusa completion integration | CG-09,11,13,15,22,26..28,33..36 | Focusa | Project→Workpoint lineage, truthful proof, policy-authorized Review Case decision, bounded closure awareness, complete binding envelope, bidirectional closure index, Completion Receipt, provider sync and reopen remain separate and exact; failed or missing proof keeps/reopens the task and returns exact gaps to the responsible model | consumer-side authority E2E + canonical continuation proof |
| CG-30 — Installed dogfood | CG-29 | independent team | publish→restart→judge→action/reproof→derive/connect→export/import→revoke/reopen→settle across environments | installed binary + live E2E packet |
| CG-31 — Standards/public claim governance | CG-30 | external/independent | separate integrity/provenance/observation/sufficiency/verification/completion/settlement/legal-admissibility states; standards/conformance matrix, hostile/public corpus, reproducible benchmark and metrics, independent assessments, cross-implementation interop, dated claim packet, challenge/correction/expiry/withdrawal governance | external reports + corpus/interop/claim packet + challenge/expiry proof |
| CG-32 — Final closure join | CG-31,33..36 | Focusa Completion Authority | all required artifacts and every required step have valid/current/scope-matched truthful proof, complete canonical bindings, converged awareness/index state, and a terminal policy-authorized Review Case decision; otherwise no Completion Receipt or provider receipt is emitted, the task remains/reopens, and a typed return-to-model receipt names the exact gaps | Completion Receipt only after proof-gated terminal decision; provider receipt; return-to-model receipt; settlement proof |

## Dependency-safe execution waves

1. **Wave A:** CG-01..07 independent foundation settlement.
2. **Wave B parallel:** CG-08/10/12/14 implementation and joins where dependencies permit.
3. **Wave C parallel:** CG-16..25 cross-surface/settings/media/lifecycle.
4. **Wave D:** CG-26..28 connector/parity/operations convergence.
5. **Wave E:** CG-29 Focusa authority integration.
6. **Wave F:** CG-30 installed dogfood.
7. **Wave F2:** CG-33..36 review/disposition, closure awareness, autonomous binding, and registry/index joins (after their stated dependencies; before CG-29/32).
8. **Wave G:** CG-31 standards and CG-32 closure (requires truthful proof, converged canonical state, and a policy-authorized terminal decision).

## Global gates

Every candidate requires versioned contract, exact allowlist, stable identity/idempotency, deterministic producer tests, consumer-side tests, cross-version proof, immutable evidence, independent verification, rollback, and no secret/private-path leakage. HTTP 200, green tests, PR publication, visual state, artifact existence, or an unattended review approval alone never closes a node. Closure of operator-directed work requires CG-33's truthful-proof gate followed by a policy-authorized terminal Review Case disposition; when human review is required, that includes browser approval or rejection. Operator direction cannot substitute for missing, stale, contradictory, withheld, or unverifiable model proof.

### Truthful-proof closure failure path

This path is mandatory, not advisory:

1. The model submits one typed proof reference for each required step, with exact scope, target, result, freshness, and consumer-verification evidence.
2. The EPWA closure gate validates completeness, provenance, scope, freshness, consistency, and consumer-visible truth. A missing, stale, contradictory, withheld, or unverifiable proof reference is a failed gate, not an inferred pass.
3. On failure, the record stays `pending_proof` or becomes `returned_to_model`; the operator cannot approve closure, and neither a Completion Receipt nor provider-close receipt may be emitted.
4. The task provider keeps the task open or reopens it and emits a durable return-to-model receipt naming the responsible model, exact missing proof, and next required evidence.
5. Only a fresh proof bundle that passes the gate can return to the authorized reviewer; when human review is assigned, the EPWA browser exposes approval or rejection. Rejection returns the task to the model with the reviewer's reason; it never silently closes the task.
