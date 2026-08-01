# UIAI Cockpit Numbered Document Register

**Register ID:** `UIAI-COCKPIT-REGISTER`  
**Status:** Active document-family index  
**Repository:** `WPUIAI/uiai-engine`

## 0. Purpose

This register provides stable, sortable identifiers for the UIAI Cockpit master specification, integrated decisions, amendments, and machine-readable companions. It does not replace the authority or version metadata inside each document.

## 1. Numbering rules

```text
UIAI-COCKPIT-000
  Master Cockpit specification.

UIAI-COCKPIT-001, 002, 003...
  Integrated decisions and normative amendments in adoption order.

UIAI-COCKPIT-002-C01, C02...
  Machine-readable companions belonging to a numbered decision/amendment.

UIAI-COCKPIT-002/AFB-001
  Fully qualified stable requirement identifier inside Amendment 002.
```

Rules:

1. Document numbers never change when a title or filename is refined.
2. A replacement master revision remains `000` and changes its document version.
3. Amendments increment monotonically and are never renumbered to close gaps.
4. Machine-readable companions inherit the parent number and increment `C01`, `C02`, and so on.
5. Local requirement IDs remain readable; external references SHOULD use the fully qualified form.
6. Accepted amendments govern their explicit subject until incorporated into a later approved master revision.

## 2. Current register

| Number | File | Status | Role |
|---|---|---|---|
| `UIAI-COCKPIT-000` | [`UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md`](UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md) | Replacement-candidate v0.5 | Unified master product, IA, UX, architecture, contracts, implementation, testing, and acceptance specification |
| `UIAI-COCKPIT-001` | [`UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md`](UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md) | Integrated decision | Review Reports and Report Canvas conformance decision |
| `UIAI-COCKPIT-002` | [`UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`](UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md) | Proposed normative amendment v1.0 | Agent-first browser exchange, observation truth, verification, provenance, execution attestation, token efficiency, and origin-bound structured actuation |
| `UIAI-COCKPIT-002-C01` | [`contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml) | Machine-readable companion v1 | Requirements, schemas, capabilities, implementation phases, metrics, ownership and stable IDs for Amendment 002 |
| `UIAI-COCKPIT-003` | [`UIAI_COCKPIT_003_OPERATIONAL_CONSTITUTION_2026-08-01_v1.0.md`](UIAI_COCKPIT_003_OPERATIONAL_CONSTITUTION_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Cross-plane commits, ambiguity, recovery, resource governance, operational truth, identity, intervention, compatibility, evidence storage, and proof verification |
| `UIAI-COCKPIT-003-C01` | [`contracts/UIAI_COCKPIT_003_C01_OPERATIONAL_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_003_C01_OPERATIONAL_CONTRACT_LEDGER_v1.yaml) | Machine-readable companion v1 | Operational states, recovery classes, requirements, phases, ownership, and metrics |
| `UIAI-COCKPIT-004` | [`UIAI_COCKPIT_004_PERCEPTION_VERIFICATION_EVAL_2026-08-01_v1.0.md`](UIAI_COCKPIT_004_PERCEPTION_VERIFICATION_EVAL_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Perception fusion, verification-independence levels, reproducible UIAI Engine Eval, challenge evaluation, and autonomy calibration |
| `UIAI-COCKPIT-004-C01` | [`contracts/UIAI_COCKPIT_004_C01_PERCEPTION_VERIFICATION_EVAL_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_004_C01_PERCEPTION_VERIFICATION_EVAL_LEDGER_v1.yaml) | Machine-readable companion v1 | Perception channels, verification levels, requirements, phases, and eval metrics |
| `UIAI-COCKPIT-005` | [`UIAI_COCKPIT_005_BROWSER_PROFILES_CHALLENGE_RESEARCH_2026-08-01_v1.0.md`](UIAI_COCKPIT_005_BROWSER_PROFILES_CHALLENGE_RESEARCH_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Detect/no-detect/operator/research browser profiles, engine adapters, fingerprint consistency, network identity, shared challenge subsystem, research lab, and Cockpit settings |
| `UIAI-COCKPIT-005-C01` | [`contracts/UIAI_COCKPIT_005_C01_BROWSER_PROFILE_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_005_C01_BROWSER_PROFILE_LEDGER_v1.yaml) | Machine-readable companion v1 | Browser modes, engines, route adapters, challenge policies, contracts, requirements, phases, and metrics |
| `UIAI-COCKPIT-006` | [`UIAI_COCKPIT_006_FOCUSA_LIVE_BINDING_AND_VISUAL_PROOF_2026-08-01_v1.0.md`](UIAI_COCKPIT_006_FOCUSA_LIVE_BINDING_AND_VISUAL_PROOF_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Focusa contract locking, live projections, mandatory functional UI code, event-reactive states, screen capture evidence, and release-blocking visual proof |
| `UIAI-COCKPIT-006-C01` | [`contracts/UIAI_COCKPIT_006_C01_FOCUSA_UI_VISUAL_PROOF_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_006_C01_FOCUSA_UI_VISUAL_PROOF_LEDGER_v1.yaml) | Machine-readable companion v1 | Functional UI, event binding, visual capture, evidence manifest, and release-verification requirements |

## 3. Application order

```text
UIAI-COCKPIT-000 master
→ accepted numbered decisions and amendments in ascending order
→ machine-readable companions for implementation and proof
→ generated contracts, tasks, tests, Evidence, Receipts, and release proof
```

A machine-readable companion must agree with its normative parent. A discrepancy blocks implementation until the parent or companion is corrected explicitly.
