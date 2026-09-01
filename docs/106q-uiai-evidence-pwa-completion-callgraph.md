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
| CG-04 — T03 hostile-content independent join | CG-03 | security verifier | adversarial fixtures, sanitization, redaction and leak checks pass | threat corpus + scan report |
| CG-05 — T04 capture assembly independent join | CG-04 | multimodal verifier | deterministic capture/media/omission/anti-curation proof accepted | 30-run digest + modality evidence |
| CG-06 — T05 crypto/time/custody independent join | CG-05 | crypto verifier | identity, signing, time-confidence, federation and custody accepted | key-rotation/import/revocation proof |
| CG-07 — Foundation join | CG-02..06 | independent join | all five exact results valid, unexpired, scope-matched | joined Evidence Artifact |
| CG-08 — T06 Judge runtime | CG-07 | UIAI | request/view/result runtime, isolated execution, appeals, budgets, drift/calibration | producer + consumer tests; fixtures |
| CG-09 — T06 independent join | CG-08 | independent judge | frozen-information review passes | judge result + citations |
| CG-10 — T07 Action runtime | CG-09 | UIAI + Focusa | operation registry, preview/confirm, anti-replay, result/reconciliation, review transport implemented | ambiguity/partial/replay E2E |
| CG-11 — T07 independent join | CG-10 | independent judge | no action/review path can assert completion | action trace + authority audit |
| CG-12 — T08 PWA runtime | CG-07 | UIAI | `106s` project registry DB, compact list/search/filter/facets/collections, bidirectional Artifact↔Work Item↔Acceptance Atom↔Completion Case edges and closure projection; Overview/Evidence/Timeline/Inspect/Developer detail; PWA/offline/subpath/LowMem/localization/access states | 10k/100k browser/search/edge/closure matrix + offline replay |
| CG-13 — T08 independent join | CG-12 | accessibility/security/performance verifier | WCAG 2.2 AA, CSP, registry authorization/no-enumeration, bulk-operation authority, index recovery, 10k/100k performance and portability accepted | axe/manual/diagnostic/load/rebuild artifacts |
| CG-14 — T09 derivative runtime | CG-13 | UIAI | print/PDF/email/Markdown/HTML/JSON/CSV/archive/slides with deterministic identities | viewer/client matrix + hashes |
| CG-15 — T09 independent join | CG-14 | independent document verifier | accessibility, licensing, delivery truth and archive safety accepted | PDF/email/archive proof |
| CG-16 — S01 canonical packet API and registry | CG-13 | UIAI | automatic screenshot/video packet creation; bounded list/search/facets/inspect/verify/resolve/serve/forward-edge/reverse-edge/closure-projection operations with stable cursors; corrupt/degraded/restart/rebuild states | API E2E + restart/rebuild/edge parity proof |
| CG-17 — S02 Pi complete parity | CG-16 | Pi | URL-first capture plus list/inspect/verify/resolve/settings/explain; confirmation for mutations | extension contract tests |
| CG-18 — S03 Cockpit board/settings | CG-16 | Cockpit | preview cards, filters, provenance, verification, all settings domains, conflict/reset/degraded states | component + UIAI visual proof |
| CG-19 — S04 Chrome viewer/settings | CG-16 | Focusa extension | View Evidence after capture, trust badge/details, project scope, canonical settings round-trip | Chrome build + real browser proof |
| CG-20 — S05 Desktop Canvas object/settings | CG-16 | Focusa Desktop | Canvas evidence object, recent board, scope binding, preview/details/settings/offline handoff | Desktop tests + visual proof |
| CG-21 — S06 Veragensia durable EPWA mount | CG-16 | Veragensia | `/evidence/` defaults to the compact registry, row selection opens immutable detail, survives container recreate, uses canonical packet API, and separates public fixture/private live databases and trust classes | recreate + registry/detail HTTP + zero-residue proof |
| CG-22 — Surface parity join | CG-17..21 | independent consumer verifier | same packet/settings/index revisions and Artifact↔Task↔Acceptance↔Closure edges render consistently across five surfaces | cross-surface registry/detail/edge contract matrix |
| CG-23 — Settings completeness | CG-16 | UIAI | lifecycle/storage/image/video/presentation/access/privacy/verification/performance/offline/integration schema + inheritance + receipts | global/project/workstream round-trip |
| CG-24 — Lifecycle/retention executor | CG-23 | UIAI | pin/archive/expiry/quota/GC/legal-hold semantics implemented; canonical evidence never silently deleted | clock/quota/restart tests |
| CG-25 — Image/video optimization | CG-23 | UIAI media | browser-native derivatives, DPR, budgets, metadata stripping, poster/caption/keyframe requirements | performance and media matrix |
| CG-26 — T10 neutral connectors | CG-11,15,22 | UIAI + Focusa | issue/document/chat adapters, exact destination authority, idempotency/dead-letter/webhook/safe-unfurl | three live consumer proofs |
| CG-27 — T11 generated API parity | CG-08..26 | UIAI | one contract drives REST/OpenAPI/CLI/MCP/Pi clients, jobs/cancel/resume/cursors/content negotiation | cross-harness conformance |
| CG-28 — T12 operations/migration/reliability | CG-24,25,27 | UIAI ops | SLO/doctor/telemetry, legacy migration, fuzz/chaos/power-loss/load/backup/rollback/release | production-consistency five proofs |
| CG-29 — Focusa completion integration | CG-09,11,13,15,22,26..28 | Focusa | Project→Workpoint lineage, Verification, Completion Receipt, provider sync and reopen remain separate and exact | consumer-side authority E2E |
| CG-30 — Installed dogfood | CG-29 | independent team | publish→restart→judge→action/reproof→derive/connect→export/import→revoke/reopen→settle across environments | installed binary + live E2E packet |
| CG-31 — Standards/public claim governance | CG-30 | external/independent | standards matrix, corpus, interop implementation, security/privacy/accessibility review, dated claim packet and expiry | external reports + challenge flow |
| CG-32 — Final closure join | CG-31 | Focusa Completion Authority | all required artifacts valid/current/scope-matched; no open blocker; provider synchronization reconciled | Completion Receipt; provider receipt |

## Dependency-safe execution waves

1. **Wave A:** CG-01..07 independent foundation settlement.
2. **Wave B parallel:** CG-08/10/12/14 implementation and joins where dependencies permit.
3. **Wave C parallel:** CG-16..25 cross-surface/settings/media/lifecycle.
4. **Wave D:** CG-26..28 connector/parity/operations convergence.
5. **Wave E:** CG-29 Focusa authority integration.
6. **Wave F:** CG-30 installed dogfood.
7. **Wave G:** CG-31 standards and CG-32 closure.

## Global gates

Every candidate requires versioned contract, exact allowlist, stable identity/idempotency, deterministic producer tests, consumer-side tests, cross-version proof, immutable evidence, independent verification, rollback, and no secret/private-path leakage. HTTP 200, green tests, PR publication, visual state, review approval, or artifact existence alone never closes a node.
