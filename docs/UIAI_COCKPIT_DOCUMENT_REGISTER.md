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

> Historical identifier collisions are preserved by distinct canonical filenames; consumers must resolve by filename, not the short register identifier alone.
| `UIAI-COCKPIT-000` | [`UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md`](UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md) | Replacement-candidate v0.5 | Unified master product, IA, UX, architecture, contracts, implementation, testing, and acceptance specification |
| `UIAI-COCKPIT-001` | [`UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md`](UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md) | Integrated decision | Review Reports and Report Canvas conformance decision |
| `UIAI-COCKPIT-002` | [`UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`](UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md) | Adopted v1.0 | Agent-first browser exchange, observation truth, verification, provenance, execution attestation, token efficiency, and origin-bound structured actuation |
| `UIAI-COCKPIT-002-C01` | [`contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml) | Machine-readable companion v1 | Requirements, schemas, capabilities, implementation phases, metrics, ownership and stable IDs for Amendment 002 |
| `UIAI-COCKPIT-003` | [`UIAI_COCKPIT_003_SIDEBAR_NAVIGATION_IA_DND_IMPLEMENTATION_SPEC_2026-08-01_v1.0.md`](UIAI_COCKPIT_003_SIDEBAR_NAVIGATION_IA_DND_IMPLEMENTATION_SPEC_2026-08-01_v1.0.md) | Adopted v1.0 | Manifest-backed Cockpit shell/sidebar migration, real workspace IA, exact Focusa Project (ScopeRef) → Trajectory Ladder → Workset → TaskGraph → Individual Tasks authority model, user ordering and DnD, persistence, entitlement integration, task graph, dependencies, testing, and rollout |
| `UIAI-COCKPIT-004` | [`UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md`](UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md) | Adopted v1.0 | Engine-owned packaged browser runtime, same-session Cockpit presentation, bidirectional Focusa Menubar handoff, lifecycle, security, task graph, and release proof |
| `UIAI-COCKPIT-004-C01` | [`contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml) | Machine-readable companion v1 | Stable DSP requirements, schemas, routes, phases, task dependencies, metrics, ownership, prohibited patterns, and proof requirements for Amendment 004 |
| `UIAI-COCKPIT-004-NEXT` | [`UIAI_COCKPIT_004_SCOPED_WORK_SURFACES_TABS_PANES_WINDOWS_MOTION_SPEC_2026-08-04_v1.0.md`](UIAI_COCKPIT_004_SCOPED_WORK_SURFACES_TABS_PANES_WINDOWS_MOTION_SPEC_2026-08-04_v1.0.md) | Proposed v1.0 (origin/main) | Corrected Workstream-scoped Cockpit foundation for tabs, Surface Groups, split panes, multiple windows, browser-target binding, true command scoping, assistant context, restoration, Tauri security, and motion |
| `UIAI-COCKPIT-005` | [`UIAI_COCKPIT_005_FOCUSA_DAEMON_PAIRING_MULTI_DAEMON_SCOPE_RECONCILIATION_SPEC_2026-08-03_v1.0.md`](UIAI_COCKPIT_005_FOCUSA_DAEMON_PAIRING_MULTI_DAEMON_SCOPE_RECONCILIATION_SPEC_2026-08-03_v1.0.md) | Adopted v1.0 | Focusa daemon discovery and pairing, Menubar-assisted auto-add, secure credentials, authenticated fleet profiles, project/scope reconciliation, repair, TaskGraph, tests, and signed rollout |
| `UIAI-COCKPIT-005-C01` | [`contracts/UIAI_COCKPIT_005_C01_FOCUSA_PAIRING_RECONCILIATION_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_005_C01_FOCUSA_PAIRING_RECONCILIATION_LEDGER_v1.yaml) | Machine-readable companion v1 | Stable pairing invariants, contracts, exact stream dependencies, acceptance criteria, critical path, and proof requirements for Amendment 005 |
| `UIAI-COCKPIT-005-C02` | [`contracts/UIAI_COCKPIT_005_C02_ATOMIC_TASKGRAPH_v1.yaml`](contracts/UIAI_COCKPIT_005_C02_ATOMIC_TASKGRAPH_v1.yaml) | Executable atomic TaskGraph v1 | Eighty-four bounded work packages with implementation-to-proof ordering and detailed acceptance outcomes for T005-02 through T005-14 |
| `UIAI-COCKPIT-006` | [`UIAI_COCKPIT_006_STUDIO_CREATIVE_WORKBENCH_AMENDMENT_2026-08-11_v0.1.md`](UIAI_COCKPIT_006_STUDIO_CREATIVE_WORKBENCH_AMENDMENT_2026-08-11_v0.1.md) | Iterable Draft v0.1 | Studio creative workbench — Capture/Compare/Analyze/Design/Produce + whiteboard (tldraw-offline) + generative GUIs + Google-Docs-like Report Canvas collaboration, workstream-scoped, universal agent control |
| `UIAI-COCKPIT-005-NEXT` | [`UIAI_COCKPIT_005_WORKSTREAM_SCOPED_UNIVERSAL_AGENT_CONTROL_AND_MULTIMODAL_VISUAL_WORKSPACE_RUNTIME_AMENDMENT_2026-08-04_v1.0.md`](UIAI_COCKPIT_005_WORKSTREAM_SCOPED_UNIVERSAL_AGENT_CONTROL_AND_MULTIMODAL_VISUAL_WORKSPACE_RUNTIME_AMENDMENT_2026-08-04_v1.0.md) | Proposed v1.0 (origin/main) | Universal semantic Cockpit control; GUI/CLI/API/MCP/Pi parity; Workstream-scoped spreadsheets, whiteboards, DataViews, Flint visualizations, dashboards and generated workspaces; live human-agent collaboration; Focusa Desktop handoff; Evidence; security; accessibility; headless and adversarial proof |
| `UIAI-COCKPIT-003` | [`UIAI_COCKPIT_003_OPERATIONAL_CONSTITUTION_2026-08-01_v1.0.md`](UIAI_COCKPIT_003_OPERATIONAL_CONSTITUTION_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Cross-plane commits, ambiguity, recovery, resource governance, operational truth, identity, intervention, compatibility, evidence storage, and proof verification |
| `UIAI-COCKPIT-003-C01` | [`contracts/UIAI_COCKPIT_003_C01_OPERATIONAL_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_003_C01_OPERATIONAL_CONTRACT_LEDGER_v1.yaml) | Machine-readable companion v1 | Operational states, recovery classes, requirements, phases, ownership, and metrics |
| `UIAI-COCKPIT-004` | [`UIAI_COCKPIT_004_PERCEPTION_VERIFICATION_EVAL_2026-08-01_v1.0.md`](UIAI_COCKPIT_004_PERCEPTION_VERIFICATION_EVAL_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Perception fusion, verification-independence levels, reproducible UIAI Engine Eval, challenge evaluation, and autonomy calibration |
| `UIAI-COCKPIT-004-C01` | [`contracts/UIAI_COCKPIT_004_C01_PERCEPTION_VERIFICATION_EVAL_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_004_C01_PERCEPTION_VERIFICATION_EVAL_LEDGER_v1.yaml) | Machine-readable companion v1 | Perception channels, verification levels, requirements, phases, and eval metrics |
| `UIAI-COCKPIT-005` | [`UIAI_COCKPIT_005_BROWSER_PROFILES_CHALLENGE_RESEARCH_2026-08-01_v1.0.md`](UIAI_COCKPIT_005_BROWSER_PROFILES_CHALLENGE_RESEARCH_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Detect/no-detect/operator/research browser profiles, engine adapters, fingerprint consistency, network identity, shared challenge subsystem, research lab, and Cockpit settings |
| `UIAI-COCKPIT-005-C01` | [`contracts/UIAI_COCKPIT_005_C01_BROWSER_PROFILE_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_005_C01_BROWSER_PROFILE_LEDGER_v1.yaml) | Machine-readable companion v1 | Browser modes, engines, route adapters, challenge policies, contracts, requirements, phases, and metrics |
| `UIAI-COCKPIT-006` | [`UIAI_COCKPIT_006_FOCUSA_LIVE_BINDING_AND_VISUAL_PROOF_2026-08-01_v1.0.md`](UIAI_COCKPIT_006_FOCUSA_LIVE_BINDING_AND_VISUAL_PROOF_2026-08-01_v1.0.md) | Proposed normative amendment v1.0 | Focusa contract locking, live projections, mandatory functional UI code, event-reactive states, screen capture evidence, and release-blocking visual proof |
| `UIAI-COCKPIT-006-C01` | [`contracts/UIAI_COCKPIT_006_C01_FOCUSA_UI_VISUAL_PROOF_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_006_C01_FOCUSA_UI_VISUAL_PROOF_LEDGER_v1.yaml) | Machine-readable companion v1 | Functional UI, event binding, visual capture, evidence manifest, and release-verification requirements |
| `UIAI-COCKPIT-006-C02` | [`contracts/UIAI_COCKPIT_006_C02_FOCUSA_FUNCTIONAL_UI_SURFACE_MATRIX_v1.yaml`](contracts/UIAI_COCKPIT_006_C02_FOCUSA_FUNCTIONAL_UI_SURFACE_MATRIX_v1.yaml) | Machine-readable companion v1 | Per-spec functional workbenches, operation coverage, exact UI obligations, capture sets, truthful implementation status, and closure rules |
| `UIAI-COCKPIT-007` | [`UIAI_COCKPIT_007_GOVERNED_NOTEBOOK_RUNTIME_JUPYTER_COMPUTATIONAL_EVIDENCE_2026-08-01_v1.0.md`](UIAI_COCKPIT_007_GOVERNED_NOTEBOOK_RUNTIME_JUPYTER_COMPUTATIONAL_EVIDENCE_2026-08-01_v1.0.md) | Proposed normative amendment v1.0; documentation-first | Provider-neutral governed notebook runtime, Jupyter Server/JupyterLab adapters, bounded kernels, computational evidence, reproducibility, verification, and Focusa math/physics integration boundaries |
| `UIAI-COCKPIT-007-C01` | [`contracts/UIAI_COCKPIT_007_C01_NOTEBOOK_KERNEL_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_007_C01_NOTEBOOK_KERNEL_CONTRACT_LEDGER_v1.yaml) | Machine-readable companion v1 | Notebook objects, semantic roles, execution modes, capability IDs, proposed routes, requirements, phases, authority, resource, and evidence contracts |
| `UIAI-COCKPIT-007-C02` | [`contracts/UIAI_COCKPIT_007_C02_NOTEBOOK_CELL_OUTPUT_EVIDENCE_EVENT_SCHEMA_v1.yaml`](contracts/UIAI_COCKPIT_007_C02_NOTEBOOK_CELL_OUTPUT_EVIDENCE_EVENT_SCHEMA_v1.yaml) | Machine-readable companion v1 | Event envelope, cell attempts, output artifacts, verification findings, notebook receipts, invalidations, replay, and Activity semantics |
| `UIAI-COCKPIT-007-C03` | [`contracts/UIAI_COCKPIT_007_C03_NOTEBOOK_ENVIRONMENT_REPRODUCIBILITY_MANIFEST_v1.yaml`](contracts/UIAI_COCKPIT_007_C03_NOTEBOOK_ENVIRONMENT_REPRODUCIBILITY_MANIFEST_v1.yaml) | Machine-readable companion v1 | Environment, package, data, service, determinism, resource, security, math, statistics, physics, drift, and reproducibility manifest requirements |
| `UIAI-COCKPIT-007-C04` | [`contracts/UIAI_COCKPIT_007_C04_NOTEBOOK_FUNCTIONAL_UI_VISUAL_PROOF_MATRIX_v1.yaml`](contracts/UIAI_COCKPIT_007_C04_NOTEBOOK_FUNCTIONAL_UI_VISUAL_PROOF_MATRIX_v1.yaml) | Machine-readable companion v1 | Computational Workbench surfaces, complete functional states, interaction assertions, capture families, networked provider suites, and release-blocking visual proof |
| `UIAI-COCKPIT-007-C05` | [`contracts/UIAI_COCKPIT_007_C05_FOCUSA_MATH_PHYSICS_NOTEBOOK_BINDING_MATRIX_v1.yaml`](contracts/UIAI_COCKPIT_007_C05_FOCUSA_MATH_PHYSICS_NOTEBOOK_BINDING_MATRIX_v1.yaml) | Machine-readable companion v1 | Current and proposed Focusa temporal, epistemic, runtime, mathematical, statistical, learning, physics, safety, handoff, parity, and truth-status bindings |
| `UIAI-COCKPIT-008` | [`UIAI_COCKPIT_008_FOCUSA_MISSION_CANVAS_INTERLOCK_HANDOFF_CREDENTIALS_SUPERVISION_2026-08-01_v1.0.md`](UIAI_COCKPIT_008_FOCUSA_MISSION_CANVAS_INTERLOCK_HANDOFF_CREDENTIALS_SUPERVISION_2026-08-01_v1.0.md) | Proposed normative amendment v1.0; documentation-first | Focusa Mission Canvas/Cockpit GUI boundary, generated cross-GUI projection, contextual handoff, typed intake, operator takeover, credential brokerage, workflow context, restoration, and proof |
| `UIAI-COCKPIT-008-C01` | [`contracts/UIAI_COCKPIT_008_C01_CROSS_GUI_PROJECTION_WORK_SURFACE_BINDING_v1.yaml`](contracts/UIAI_COCKPIT_008_C01_CROSS_GUI_PROJECTION_WORK_SURFACE_BINDING_v1.yaml) | Machine-readable companion v1 | Exact Focusa Work Surface to UIAI Work Object binding, projection levels, deep links, adapter, events, and no-duplication requirements |
| `UIAI-COCKPIT-008-C02` | [`contracts/UIAI_COCKPIT_008_C02_CONTEXTUAL_HANDOFF_TYPED_INTAKE_v1.yaml`](contracts/UIAI_COCKPIT_008_C02_CONTEXTUAL_HANDOFF_TYPED_INTAKE_v1.yaml) | Machine-readable companion v1 | Browser/document/notebook handoff, TaskIntakeEnvelope, trusted-instruction boundary, deduplication, preview, commit, and result bindings |
| `UIAI-COCKPIT-008-C03` | [`contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml`](contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml) | Machine-readable companion v1 | Focusa intervention, UIAI control leases, safety freeze, operator delta, reobservation, fencing, return-control, and recovery |
| `UIAI-COCKPIT-008-C04` | [`contracts/UIAI_COCKPIT_008_C04_CREDENTIAL_GRANT_SECRET_PROXY_USE_RECEIPT_v1.yaml`](contracts/UIAI_COCKPIT_008_C04_CREDENTIAL_GRANT_SECRET_PROXY_USE_RECEIPT_v1.yaml) | Machine-readable companion v1 | Focusa credential authority, UIAI secret custody, opaque proxy injection, origin/operation admission, standing grants, receipts, and secret-leak testing |
| `UIAI-COCKPIT-008-C05` | [`contracts/UIAI_COCKPIT_008_C05_WORKFLOW_CONTEXT_EXECUTION_MANIFEST_v1.yaml`](contracts/UIAI_COCKPIT_008_C05_WORKFLOW_CONTEXT_EXECUTION_MANIFEST_v1.yaml) | Machine-readable companion v1 | One Focusa recurrence authority, frozen/refreshed/carried/prohibited context, UIAI execution manifests, preflight, retry, and settlement boundary |
| `UIAI-COCKPIT-008-C06` | [`contracts/UIAI_COCKPIT_008_C06_FUNCTIONAL_UI_CROSS_GUI_VISUAL_PROOF_MATRIX_v1.yaml`](contracts/UIAI_COCKPIT_008_C06_FUNCTIONAL_UI_CROSS_GUI_VISUAL_PROOF_MATRIX_v1.yaml) | Machine-readable companion v1 | Exact Cockpit components, complete states, interactions, cross-GUI suites, Focusa Mission Canvas captures, accessibility, secret scans, manifests, and release gates |
| `UIAI-COCKPIT-009` | [`009-uiai-cockpit-browser-identity-gap-closure-matrix.md`](009-uiai-cockpit-browser-identity-gap-closure-matrix.md) | Proposed normative gap-closure amendment v1.0 | Cross-spec browser identity, canonical session integration, operator persona, challenge, extraction, FPV security, evidence, test, and false-completion closure matrix |

## 3. Application order

```text
UIAI-COCKPIT-000 master
→ accepted numbered decisions and amendments in ascending order
→ machine-readable companions for implementation and proof
→ generated contracts, tasks, tests, Evidence, Receipts, and release proof
```

A machine-readable companion must agree with its normative parent. A discrepancy blocks implementation until the parent or companion is corrected explicitly.
