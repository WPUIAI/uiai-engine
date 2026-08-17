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

## 3. Application order

```text
UIAI-COCKPIT-000 master
→ accepted numbered decisions and amendments in ascending order
→ machine-readable companions for implementation and proof
→ generated contracts, tasks, tests, Evidence, Receipts, and release proof
```

A machine-readable companion must agree with its normative parent. A discrepancy blocks implementation until the parent or companion is corrected explicitly.
