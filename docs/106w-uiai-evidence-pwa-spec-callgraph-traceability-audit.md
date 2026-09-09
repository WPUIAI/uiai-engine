# 106w — UIAI Evidence PWA specification-to-CallGraph traceability audit

**Parent:** Issue #106 — Evidence PWA authority  
**Audit date:** 2026-09-09  
**Scope:** canonical EPWA specification family (`106a`–`106s`) versus `106q` Completion CallGraph

## Finding

The decomposition did not preserve bidirectional traceability. The original `106q` graph compressed later normative amendments into broad T06/T07/T08/T11/T13 rows without carrying their exact state machines, reviewer authority, acceptance conditions, evidence atoms, and failure/reopen paths. The operator approval/rejection requirement was therefore present in planning but absent as an executable closure node.

This is a planning/decomposition defect. It is not evidence that the omitted behavior is complete.

## Confirmed omitted or only umbrella-covered requirements

| Source requirement | Existing `106q` coverage | Finding | Required graph repair |
|---|---|---|---|
| `106k` lines 4–21, 23–55, 137–160: Review Case, reviewer assignment/lease, item-scoped approve/reject, partial decisions, quorum/dispute, decision Receipts, fail-closed `bd`/`br` closure, reproof lineage | CG-08/09, CG-10/11, CG-18, CG-29 | **Missing as an explicit node.** The graph had judge/action/Focusa rows but no Review Case or decision-UI contract and no acceptance path for appointed human review through PWA → CLI/API/MCP. | CG-33: policy-gated Review/Adjudication + truthful-proof closure gate |
| `106l` lines 34–65: autonomous LLM review, human-optional policy, rejection/repair/reproof, quorum, cycle/budget/circuit-breaker limits | CG-08/09 and CG-29 | **Partial and ambiguous.** The graph did not state that human approval is policy-dependent, that LLM decisions use the same canonical Review Case/Receipt, or that rejection routes bounded repair instead of false success. | CG-33 plus explicit policy-dependent reviewer and repair/reproof acceptance |
| `106m` lines 16–57, 95–104: Project → Workstream → Trajectory → Workset → CallGraph → Workpoint → provider item → artifact → judge → Completion → provider close → settlement; revision/generation binding and missing-primitive fail-closed behavior | CG-16/17/22/29 | **Partial.** The rows name packet/registry/parity but do not require the complete bidirectional binding envelope, generation invalidation, `binding_blocked`/`capture_candidate`, or exact citation/Receipt lineage. | Extend CG-16, CG-22, CG-29 with the full binding envelope and stale/orphan failure path |
| `106n` lines 4–50, 52–88: bounded closure-awareness projection, review/completion/provider/settlement state machines, `outcome_unknown`, no-assumption rule, asynchronous refresh, parity across Pi/CLI/MCP/PWA/Cockpit/Desktop/provider, ten failure fixtures | CG-27/29 | **Missing as an explicit join.** “Generated parity” and “Focusa completion integration” do not define the awareness projection or the unknown/reconcile/reopen behavior. | CG-34: working-agent closure awareness and submission-state parity |
| `106o` lines 8–32, 40–80: evidence as autonomous-work substrate; Work Loop/assignment/generation/budget/retry/circuit-breaker binding; restart/failover/reassignment/reproof; no hidden mutable UIAI state; unattended acceptance | CG-03/06/10/28/29/30 | **Partial.** Reliability and dogfood rows exist, but the graph lacks a required autonomous-state binding/recovery join and does not prove all restart/failover/unknown cases. | CG-35: autonomous work-state binding and recovery join |
| `106s` lines 35–55, 153–176: project registry, forward/reverse edge index, closure projection, missing/rejected/stale/unverified eligibility, rebuild parity, public leakage, bulk preview/confirm/idempotency | CG-12/13/16/21/22 | **Partial.** Registry/PWA rows do not require reverse closure edges, exact task/Acceptance/Completion links, corruption rebuild parity, or “not closure-eligible” rendering. | CG-36: project registry and bidirectional closure index join |
| `106i` gaps 141–150: separate validity layers, standards/conformance matrix, public corpus, comparative metrics, independent assessments, interoperability, dated claim packet, challenge/correction/expiry governance | CG-31 | **Partial.** The standards row is too broad to prove the distinct validity layers and claim-governance lifecycle. | Expand CG-31 with typed validity/claim states, corpus, interop, challenge, expiry, and withdrawal evidence |
| `106h` gaps 111–140: custody/selection and omission, independent ground truth, anti-cherry-picking, event continuity, actor/delegation, imported-action distrust, corrections/retractions, legal/eDiscovery, redaction/media fidelity, air-gapped verification, link rot | CG-04/05/06/28/31 | **Partial.** Some words appear in T03/T04/T05/T12 scope, but the graph does not require per-gap proof atoms or failure paths. | Expand CG-04/05/06/28 acceptance and evidence atoms; no gap may be satisfied by a label alone |
| `106b` lines 38–67, 71–90: bundle execution/provenance/verification/receipts/policy/integrity/links, automatic publication before cleanup, typed blocked/failed results, uncertainty rendered as uncertainty | CG-12/16/17/21/22 | **Partial.** Automatic publication and failure-result semantics are not explicit in the node done conditions or cross-surface evidence. | Expand CG-16/17/22 with before-cleanup publication and blocked/failed/not-applicable proofs |
| `106d` lines 39–80: governed Action Deck operations (reproof, follow-up, adjudication, export), untrusted artifact content, no local completion mutation, canonical Focusa operation calls | CG-10/11/18/29 | **Partial.** Action runtime mentions registry/preview/reconciliation but not the full operation vocabulary or the non-authority rule as a join criterion. | Expand CG-10/11/29 with operation coverage and authority-separation tests |

## Why approval disappeared

`106j` decomposed the work into T01–T15 and `106q` converted those workstreams into broad runtime/surface/authority joins. The later closure amendments were treated as prose attached to T07/T13 rather than recompiled into node-level acceptance. In particular:

1. “review transport” in CG-10 was mistaken for a Review Case and decision contract;
2. “verification, Completion Receipt, provider sync and reopen” in CG-29 was mistaken for the operator decision and its failure path;
3. “Focusa Completion Authority” in CG-32 was treated as sufficient closure semantics without requiring the browser-visible appointed reviewer operation;
4. no decomposition check required every normative amendment to have a node, evidence atom, and negative-path test.

## Correct closure semantics

A reviewer may be an appointed human or an authorized LLM judge according to the Completion Contract. The EPWA must expose the same governed Review Case and decision state; it must not create a second completion authority. For every closure attempt:

- the responsible model supplies truthful, scope-matched proof for every required step;
- the canonical gate verifies evidence, freshness, scope, independence, policy, and Receipt lineage;
- an absent, stale, contradictory, withheld, or unverifiable proof yields `pending_proof`/`returned_to_model`, not closure;
- the task remains open or reopens and receives exact proof gaps and a bounded next action;
- only a terminal, policy-authorized review decision plus all other Completion gates can reach provider closure and settlement;
- an operator browser approval is available when the reviewer assignment/policy authorizes it; it cannot waive missing model proof.

## Required decomposition acceptance

The repaired graph is not accepted until every normative source row above has:

1. one explicit node or an explicitly enumerated acceptance clause;
2. exact dependencies and owner authority;
3. positive and negative evidence atoms;
4. typed state/Receipt semantics, including unknown and reopen paths;
5. producer, consumer, cross-surface, independent-review, and installed proof;
6. a reverse citation from the node back to the source requirement.

Until that matrix is green, no EPWA task may be represented as fully closed or settled.
