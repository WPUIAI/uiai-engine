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
| CG-12 — T08 PWA runtime | CG-07 | UIAI | registry, Overview/Evidence/Timeline/Inspect/Developer, PWA/offline/subpath/LowMem/localization/access states | browser matrix + offline replay |
| CG-13 — T08 independent join | CG-12 | accessibility/security verifier | WCAG 2.2 AA, CSP, performance and portability accepted | axe/manual/diagnostic artifacts |
| CG-14 — T09 derivative runtime | CG-13 | UIAI | print/PDF/email/Markdown/HTML/JSON/CSV/archive/slides with deterministic identities | viewer/client matrix + hashes |
| CG-15 — T09 independent join | CG-14 | independent document verifier | accessibility, licensing, delivery truth and archive safety accepted | PDF/email/archive proof |
| CG-16 — S01 canonical packet API and registry | CG-13 | UIAI | automatic screenshot/video packet creation; bounded list/inspect/verify/resolve/serve; corrupt/degraded/restart states | API E2E + restart proof |
| CG-17 — S02 Pi complete parity | CG-16 | Pi | URL-first capture plus list/inspect/verify/resolve/settings/explain; confirmation for mutations | extension contract tests |
| CG-18 — S03 Cockpit board/settings | CG-16 | Cockpit | preview cards, filters, provenance, verification, all settings domains, conflict/reset/degraded states | component + UIAI visual proof |
| CG-19 — S04 Chrome viewer/settings | CG-16 | Focusa extension | View Evidence after capture, trust badge/details, project scope, canonical settings round-trip | Chrome build + real browser proof |
| CG-20 — S05 Desktop Canvas object/settings | CG-16 | Focusa Desktop | Canvas evidence object, recent board, scope binding, preview/details/settings/offline handoff | Desktop tests + visual proof |
| CG-21 — S06 Veragensia durable EPWA mount | CG-16 | Veragensia | `/evidence/` survives container recreate, uses canonical packet API, public fixture/private live modes separated, desktop navigation present | recreate + HTTP + zero-residue proof |
| CG-22 — Surface parity join | CG-17..21 | independent consumer verifier | same packet/settings revisions render consistently across five surfaces | cross-surface contract matrix |
| CG-23 — Settings completeness | CG-16 | UIAI | lifecycle/storage/image/video/presentation/access/privacy/verification/performance/offline/integration schema + inheritance + receipts | global/project/workstream round-trip |
| CG-24 — Lifecycle/retention executor | CG-23 | UIAI | pin/archive/expiry/quota/GC/legal-hold semantics implemented; canonical evidence never silently deleted | clock/quota/restart tests |
| CG-25 — Image/video optimization | CG-23 | UIAI media | browser-native derivatives, DPR, budgets, metadata stripping, poster/caption/keyframe requirements | performance and media matrix |
| CG-26 — T10 neutral connectors | CG-11,15,22 | UIAI + Focusa | issue/document/chat adapters, exact destination authority, idempotency/dead-letter/webhook/safe-unfurl | three live consumer proofs |
| CG-27 — T11 generated API parity | CG-08..26 | UIAI | one contract drives REST/OpenAPI/CLI/MCP/Pi clients, jobs/cancel/resume/cursors/content negotiation | cross-harness conformance |
| CG-28 — T12 operations/migration/reliability | CG-24,25,27 | UIAI ops | SLO/doctor/telemetry, legacy migration, fuzz/chaos/power-loss/load/backup/rollback/release | production-consistency five proofs |
| CG-33 — Operator disposition and truthful-proof closure gate | CG-16,18 | Operator via browser + model completion authority | Before approval, the model must supply a typed, scope-matched proof bundle for every required step. Evidence records carry append-only approve/reject/returned-to-model dispositions (authority ref, required reason on rejection, task/work-item refs, supersede chain); disposition state is visible on every record surface (record page, envelope, portable zip). Missing, stale, contradictory, withheld, or unverifiable proof forces pending/returned-to-model, keeps or reopens the task, emits exact proof gaps, and forbids Completion or provider-close receipts; only a proof-gated operator approval may permit closure; public record page stays read-only and mutations require authenticated authority | proof-gap/return receipts + disposition receipts + record UI visual proof + route tests |
| CG-29 — Focusa completion integration | CG-09,11,13,15,22,26..28,33 | Focusa | Project→Workpoint lineage, truthful proof, proof-gated operator disposition, Completion Receipt, provider sync and reopen remain separate and exact; failed or missing proof keeps/reopens the task and returns exact gaps to the responsible model | consumer-side authority E2E |
| CG-30 — Installed dogfood | CG-29 | independent team | publish→restart→judge→action/reproof→derive/connect→export/import→revoke/reopen→settle across environments | installed binary + live E2E packet |
| CG-31 — Standards/public claim governance | CG-30 | external/independent | standards matrix, corpus, interop implementation, security/privacy/accessibility review, dated claim packet and expiry | external reports + challenge flow |
| CG-32 — Final closure join | CG-31,33 | Focusa Completion Authority | all required artifacts and every required step have valid/current/scope-matched truthful proof plus an approved CG-33 disposition; otherwise no Completion Receipt or provider receipt is emitted, the task remains/reopens, and a typed return-to-model receipt names the exact gaps | Completion Receipt only after proof-gated approval; provider receipt; return-to-model receipt |

## Dependency-safe execution waves

1. **Wave A:** CG-01..07 independent foundation settlement.
2. **Wave B parallel:** CG-08/10/12/14 implementation and joins where dependencies permit.
3. **Wave C parallel:** CG-16..25 cross-surface/settings/media/lifecycle.
4. **Wave D:** CG-26..28 connector/parity/operations convergence.
5. **Wave E:** CG-29 Focusa authority integration.
6. **Wave F:** CG-30 installed dogfood.
7. **Wave F2:** CG-33 proof-gated operator disposition (after CG-16/18; before CG-29/32).
8. **Wave G:** CG-31 standards and CG-32 closure (requires truthful proof plus CG-33 approval for operator-directed work).

## Global gates

Every candidate requires versioned contract, exact allowlist, stable identity/idempotency, deterministic producer tests, consumer-side tests, cross-version proof, immutable evidence, independent verification, rollback, and no secret/private-path leakage. HTTP 200, green tests, PR publication, visual state, artifact existence, or an unattended review approval alone never closes a node. Closure of operator-directed work requires CG-33's truthful-proof gate followed by an approved browser disposition; operator direction cannot substitute for missing, stale, contradictory, withheld, or unverifiable model proof.

### Truthful-proof closure failure path

This path is mandatory, not advisory:

1. The model submits one typed proof reference for each required step, with exact scope, target, result, freshness, and consumer-verification evidence.
2. The EPWA closure gate validates completeness, provenance, scope, freshness, consistency, and consumer-visible truth. A missing, stale, contradictory, withheld, or unverifiable proof reference is a failed gate, not an inferred pass.
3. On failure, the record stays `pending_proof` or becomes `returned_to_model`; the operator cannot approve closure, and neither a Completion Receipt nor provider-close receipt may be emitted.
4. The task provider keeps the task open or reopens it and emits a durable return-to-model receipt naming the responsible model, exact missing proof, and next required evidence.
5. Only a fresh proof bundle that passes the gate can return to the browser for operator approval or rejection. Rejection also returns the task to the model with the operator's reason; it never silently closes the task.
