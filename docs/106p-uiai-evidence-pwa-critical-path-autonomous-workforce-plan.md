> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5464665848

## Slice-by-slice critical path and autonomous workforce layout

T01–T05 are implemented on `main` through `e9736e244360f3fb2c70f555003f3e79a64d28dd`. This plan decomposes T06–T15 into independently admissible executor lanes with explicit all-joins, independent verification, bounded repair, and no inferred completion.

A typed candidate CallGraph has been generated locally: 65 frames / 97 edges, SHA-256 `2c34710e2ff1474bd07c4afd22adaf2d6872b13bb6f4b3323489dbdd22fb73e6`. It is **not executable** because the canonical validator is broken; Startempire-Wire/focusa#424 owns that P0. No dispatch or fallback graph authority is claimed.

### Program critical path

```text
                         ┌─ T06 Judge runtime ─┬─ T07 Actions/collab ─┐
T01–T05 complete ─ launch┤                     │                      ├─ T10 connectors
                         └─ T08 Evidence PWA ──┴─ T09 derivatives ───┘
                                                                │
                         T10 → T11 parity → T12 reliability → T13 Focusa integration
                              → T14 installed dogfood → T15 public standards/claims
```

Hard critical path: `T06 → T07 → T10 → T11 → T12 → T13 → T14 → T15`.
Parallel feeder path: `T08 → T09 → T10`.
Independent authority feeder: approved Focusa operation registry feeds T07; approved Focusa integration packets feed T13. Missing approval blocks only the dependent join, never the ready UIAI lane.

### Wave 1 — ready now, two independent worktrees

#### T06 Judge View and independent verification

1. **T06-A Contract** — freeze Judge View/request/result, modality, citation, rationale, error, fixture, rollback, and evidence contracts.
2. **T06-B Information set** — bounded selection, omission truth, canonical information-set hashing, immutable citations. Depends A.
3. **T06-C Assignment** — executor/verifier separation, capability and modality checks, calibration, scoped expiry, budgets. Depends A; parallel with B.
4. **T06-D Quorum runtime** — judge fanout, all/majority policy, disagreement, appeal, drift, failover, circuit breaker. Depends B+C join.
5. **T06-E Adversarial proof** — golden sets, bias/order tests, provider/model drift, hostile inputs, high-consequence policy. Depends A; parallel with B+C+D where fixtures permit.
6. **T06-J Independent join** — consumer tests, deterministic replay, stale/failure behavior, evidence review. Requires D+E; verifier cannot be an executor.

File ownership should remain under a new `internal/evidencejudge/` boundary; T01–T05 files change only through an explicitly frozen compatibility slice.

#### T08 Self-hosted Evidence PWA

1. **T08-A Contract/shell** — freeze semantic sections, projection schema, relative paths, hosting/access states, budgets, fixtures, rollback.
2. **T08-B Renderer** — Overview/Evidence/Timeline/Inspect/Developer semantic article. Depends A.
3. **T08-C Registry** — implement `106s`: rebuildable per-Project registry DB, compact list/search/filter/facets/collections, immutable exact lookup, stable cursors, typed bidirectional Artifact ↔ Work Item ↔ Acceptance Atom ↔ Completion Case edges, closure-eligibility projection, and registry-to-detail navigation. Depends A; parallel with B.
4. **T08-D Hosting/PWA** — localhost/LAN/tailnet/private/unlisted/public-safe, non-root subpaths, offline/update behavior. Depends A; parallel.
5. **T08-E Accessibility/i18n** — WCAG 2.2 AA, keyboard, reduced motion, localization, RTL, 375/768/1024/1440. Depends A; parallel.
6. **T08-F Scale/degradation** — `106s` 10k/100k fixtures, virtualized rows, lazy media, stable keyset paging, LowMem/offline snapshots, index rebuild/recovery, bounded payloads, performance budgets, and explicit unavailable/blocked/corrupt/stale-index states. Depends A; parallel.
7. **T08-J Independent join** — browser, accessibility, offline, restart, subpath, pressure, public-redaction, registry authorization, bidirectional edge, closure-gate, bulk-operation, index-rebuild, and 10k/100k performance matrices. Requires B+C+D+E+F.

Use a new `internal/evidencepwa/` plus bounded `web/evidence/` ownership surface; legacy share-handler hardening remains separately owned by #107.

### Authority feeder lanes

- **R-A Operation-registry contract** — freeze only approved declarative operations, capability refresh, guards, idempotency, and Receipts. No unavailable Focusa implementation.
- **R-B Integration readiness** — verify exact approved Focusa packets required by T13. Unapproved packets remain typed blockers.
- These lanes never write completion, provider closure, or settlement state.

### Wave 2 — after joins

- **T07 after T06-J + R-A:** A contract; B action projection; C preview/confirmation/anti-replay/reconciliation; D item-scoped collaboration; independent all-join.
- **T09 after T08-J:** A derivative contract; parallel B PDF, C email, D slides, E archive/text/data exports; consumer-matrix all-join.
- **T10 after T07-J + T09-J:** A neutral block contract; parallel issue-tracker, document/wiki, chat/card adapters plus delivery-safety lane; all-join.

### Waves 3–7

- **T11 after T10:** generated contract → parallel CLI/MCP/REST/Pi+LLM lanes → cross-harness all-join.
- **T12 after T11:** operations contract → parallel security/fuzz, chaos/power-loss, migration/compatibility, SLO/runbook lanes → five-proof all-join.
- **T13 after T12 + R-B:** integration contract → parallel verification, execution/continuation, Completion/provider reconciliation, operator-surface lanes → authority-separation all-join.
- **T14 after T13:** parallel happy path, rejection/repair, federation/revoke/reopen, derivatives/connectors, hostile/recovery/cross-version installed dogfood → all-join.
- **T15 after T14:** parallel standards, benchmark, external review, claim-governance lanes → all-join; no public superlative before settlement.

### Autonomous execution policy per atomic lane

```text
admit frozen slice
  → executor worktree
  → deterministic local tests
  → immutable evidence publication
  → independent verifier assignment
  → accept | changes_requested | blocked | outcome_unknown
  → bounded repair/reproof (maximum 2 cycles)
  → all-join
  → canonical completion evaluation only at the owning authority
```

- Stable work-item, intent, attempt, and idempotency refs on every frame.
- One executor owns one exact file allowlist; parallel lanes may not overlap files.
- Joins consume committed hashes and evidence refs, never dirty shared worktrees.
- A lane failure does not cancel independent ready lanes; blocked work is deferred.
- Lost responses become `outcome_unknown` and reconcile before retry.
- Dispatch, HTTP success, test success, push, review acceptance, or provider state never independently means complete.
- Maximum initial fanout: **6 lanes** (T06 B/C/E + T08 B/C/D after contracts); expand only after resource/validator health and non-overlap proof.
- Every join has an independent verifier and consumer-side acceptance tests.

### Immediate admission order

1. Repair and prove canonical CallGraph validation (#424).
2. Freeze T06-A and T08-A as separate issues with exact contracts/files/tests/rollback/evidence.
3. Validate this graph revision; bind concrete issue IDs and immutable graph digest.
4. Dispatch T06-A and T08-A in parallel.
5. On each contract join, fan the non-overlapping implementation lanes; never dispatch future waves early.
