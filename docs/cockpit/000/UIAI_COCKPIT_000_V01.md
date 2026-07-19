# UIAI Cockpit Unified Product, Information Architecture, UI/UX, and Extensibility Spec

**Document number:** `UIAI-COCKPIT-000`  
**Document family:** UIAI Cockpit numbered specification series  
**Status:** Replacement-candidate master specification  
**Version:** 0.5 draft  
**Date:** 2026-07-16  
**Repository home:** `WPUIAI/uiai-engine`  
**Primary implementation home:** `apps/cockpit/`  
**Audience:** UIAI Engine, Cockpit, Focusa, Focusa Cloud, AI API, Pi, QA, design, and product engineering

---

## 0. Document authority and preservation rule

This document combines and reorganizes the current UIAI Engine and Cockpit plans with the additional product direction established in the related design discussions:

- the existing UIAI Operator Browser Desktop specification;
- the UIAI UX/DX/FPV consolidation plan;
- the FPV PWA and FPV Co-Pilot specifications;
- the UIAI × Focusa × Pi hand-in-glove specification;
- the current UIAI Engine route and capability inventory;
- the current Cockpit contracts, card manifest, shell, pairing bridge, and smoke harness;
- the browser UX/DX recommendations and north-star trajectory;
- the proposed Maestro-based Test Lab;
- the proposed first-class PDF and Office document runtime;
- the requirement that Cockpit beautifully surface everything built within UIAI Engine while remaining extensible;
- the Agentic Browser Best Practices Specification (`ABPS-1.0`) governing mission contracts, deterministic authority, action routing, receipts, verification, settlement, security, reliability, memory, and operational conformance.
- the proposed visual and interactive agent-work report capability, conformed here as a derived, evidence-backed review object rather than a parallel task system, memory store, or completion authority.

### 0.1 No-feature-removal rule

This is an information architecture and experience consolidation, not a feature reduction.

1. Every capability, safety requirement, release requirement, accessibility requirement, integration contract, and future feature documented in the current source specifications remains preserved unless this document explicitly marks it as replaced by an equivalent or broader contract.
2. Moving a capability to a deeper disclosure level does not remove it.
3. Renaming a surface for human comprehension does not change its authority plane or backend owner.
4. Consolidating duplicate requirements does not cancel either requirement; the stricter acceptance condition wins.
5. Existing Slice 0–6 foundation work remains valid and is extended rather than discarded.
6. Existing card manifests remain a backend-to-UI contract pattern, but cards no longer define the entire top-level information architecture.
7. The full preservation crosswalk appears in Annex A.

### 0.2 Relationship to the current desktop specification

After operator acceptance, this document should replace the product, information architecture, and experience sections of `docs/UIAI_OPERATOR_BROWSER_DESKTOP_SPEC_2026-06-19.md`, while retaining its detailed release, pairing, security, testing, hard-parts, and implementation requirements through direct incorporation or normative reference.

No repository file is changed merely by creating this replacement candidate.

### 0.3 Normative design and UX enforcement

The visual, interaction, accessibility, motion, component, and content requirements in Section 10 are normative product requirements. They are enforced through shared components, tokens, fixtures, automated checks, visual regression, accessibility testing, performance gates, and human design review.

A capability is not complete merely because its backend call succeeds. The normal, loading, empty, blocked, degraded, error, approval, success, keyboard, dark-mode, compact-density, reduced-motion, and constrained-window states must also meet the design-system contract.

### 0.4 Numbered document family and amendment precedence

This specification is the numbered master document `UIAI-COCKPIT-000`. Its accepted integration decisions and amendments use a stable, sortable sequence:

| Document number | Document | Relationship |
|---|---|---|
| `UIAI-COCKPIT-000` | This unified master specification | Master product, architecture, UX, contract, implementation, testing, and acceptance authority |
| `UIAI-COCKPIT-001` | [`UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md`](UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md) | Integrated Review Reports and Report Canvas decision |
| `UIAI-COCKPIT-002` | [`UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`](UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md) | Agent-first browser truth, verification, provenance, structured actuation, execution attestation, and exchange-efficiency amendment |
| `UIAI-COCKPIT-002-C01` | [`contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml`](contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml) | Machine-readable requirement, capability, schema, phase, and metric ledger for Amendment 002 |

Numbering rules:

1. The master remains `000`; amendments and integrated decisions increment `001`, `002`, `003`, and so on.
2. Machine-readable companions use `C01`, `C02`, and so on under their parent document number.
3. Stable requirement IDs retain their local prefix but resolve fully as `<document-number>/<requirement-id>`.
4. An amendment remains normative beside the master until its requirements are incorporated into a later approved master revision.
5. When an amendment and older master wording conflict, the newer accepted amendment governs only its explicitly amended subject.
6. The complete document register is [`UIAI_COCKPIT_DOCUMENT_REGISTER.md`](UIAI_COCKPIT_DOCUMENT_REGISTER.md).


---

# 1. Executive decision

## 1.1 One Cockpit, not a collection of products

UIAI Cockpit is the single desktop operating environment for the UIAI agent-first stack.

```text
UIAI Cockpit
  ├── Live agent/browser oversight and FPV
  ├── Test execution and verification
  ├── PDF and Office document work
  ├── Research and source capture
  ├── Visual analysis and media production
  ├── Reusable workflows and automations
  ├── Evidence, proof, receipts, interactive reports, and activity
  ├── Nodes, services, pairing, scope, and capacity
  └── Capability discovery and future extensions
```

The product must not become:

```text
FPV app + test runner app + PDF app + API dashboard + Focusa cards
```

It must feel like:

```text
one coherent workspace with multiple kinds of work
```

## 1.2 Product roles

| Component | Product responsibility |
|---|---|
| **UIAI Engine** | Executes browser, research, visual, media, workflow, test-support, and future document operations; emits bounded artifacts, diagnostics, and stable evidence handles. |
| **UIAI Cockpit** | Mission Experience and operator control surface: human intent, orientation, observability, configuration, bounded orchestration requests, review, approval, comparison, takeover, and evidence handoff. It does not own canonical mission truth or scheduling. |
| **FPV PWA** | Zero-install mobile and shareable projection of Cockpit Live sessions, using the same session, event, control, and audit contracts. |
| **Focusa** | Mission Kernel and cognitive authority: project identity, verified scope, versioned mission continuity, Workpoint, Trajectory, constraints, decisions, evidence linkage, predictions, metacognition, handoff, recovery, and—when implemented—Completion Contract and settlement state. |
| **Focusa Cloud** | Account, entitlement, node registry, pairing coordination, relay, public-safe proof/receipt hosting, and team coordination without becoming cognition authority. |
| **AI API** | Hosted model execution, provider metadata, usage, entitlement, and bounded AI actions. |
| **Pi** | Agent/operator harness that chooses tools and guided workflows while remaining thin and avoiding parallel memory. |
| **Maestro** | One black-box web/mobile execution adapter inside Test Lab, not the product architecture, mission kernel, verifier, or completion authority. |
| **Tauri WebDriver runner** | Packaged native Tauri application verification inside Test Lab. |

## 1.3 Product statement

> **UIAI Cockpit is a local-first, project-scoped operating environment where humans and agents browse, test, read, create, transform, inspect, verify, and prove work through UIAI Engine—while Focusa preserves scope, continuity, evidence, and learning.**

## 1.4 Experience promise

The Cockpit should be:

- **simple at first glance;**
- **complete when searched;**
- **deep when inspected;**
- **safe when acted upon;**
- **provable when finished.**

---

# 2. Product boundaries and non-negotiable invariants

## 2.1 Preserve the current authority model

```text
UIAI Engine          = execution and artifact authority
Local Focusa Node(s) = cognition, scope, Workpoint, evidence, and continuity authority
Focusa Cloud         = coordination, entitlement, relay, public receipt/proof authority
AI API               = hosted AI request/response authority
Cockpit              = operator intent, orchestration, visualization, and approval surface
```

No plane may silently impersonate another.

## 2.2 Local Only remains complete

Local Only is the default and must feel like a complete product, not a damaged trial.

A user without Focusa Cloud or AI API credentials can still:

- operate local UIAI Engine capabilities;
- use local Focusa scope and Workpoints where available;
- run browser sessions and FPV;
- run local test runners;
- process local documents;
- capture and inspect local artifacts;
- create local evidence references;
- use local jobs, activity, settings, and diagnostics.

Cloud Profile adds coordination and hosted capabilities; it does not unlock the basic integrity of the Cockpit.

## 2.3 No second browser runtime

Cockpit renders UIAI session output and sends bounded UIAI actions. It does not fork browser execution into a second embedded automation runtime.

A WebView may render Cockpit UI or a safe document viewer. It is not a competing agent browser authority.

## 2.4 No parallel memory

Cockpit local state may cache scope-keyed summaries, layout preferences, recent objects, and events. It must not become a second Workpoint, Trajectory, prediction, or metacognitive store.

## 2.5 No raw-route product design

An HTTP route existing in UIAI Engine does not automatically make it a visible button.

Every user-facing capability requires:

- a concrete user workflow;
- an authority plane;
- an authentication posture;
- a required scope;
- an input and output contract;
- a side-effect classification;
- a redaction boundary;
- a failure and recovery model;
- an artifact/evidence model;
- smoke or contract proof.

## 2.6 No hidden mutations

Reads may execute immediately when authorized. Writes, external sends, destructive changes, proof publishing, signing, and sensitive document actions must be explicit, previewable, and attributable.

## 2.7 No forced exposure of implementation complexity

Technical truth must remain available, but raw endpoint names, daemon internals, JSON, tokens, route decisions, CRDT details, and provider configuration are not the default first layer for ordinary work.

---

# 3. Experience principles

## 3.1 Make the complex simple without making it shallow

The product should not achieve simplicity by deleting power. It should achieve simplicity by controlling **when**, **where**, and **why** power appears.

The basic pattern is:

```text
Orient → Act → Inspect → Configure → Diagnose
```

A user should see only the level needed for the current decision.

## 3.2 Progressive disclosure model

Every Cockpit capability uses five disclosure levels.

### Level 0 — Glance

Visible without interaction:

- current project/workstream;
- current Workpoint or task;
- selected node when relevant;
- one-line object status;
- primary action;
- important error, approval, or blocked state;
- running work indicator.

### Level 1 — Work

The normal workspace:

- content or live output;
- task-specific controls;
- recent artifacts;
- essential options;
- one dominant primary action;
- one or two secondary actions.

### Level 2 — Inspect

A contextual inspector reveals:

- metadata;
- scope and authority;
- inputs and output lineage;
- diagnostics summary;
- evidence links;
- history;
- runner or engine details;
- non-default options relevant to the selected object.

### Level 3 — Configure

An explicit configuration surface reveals:

- advanced execution parameters;
- environment profiles;
- redaction and retention policy;
- retry, timeout, resource, and transport settings;
- runner-specific configuration;
- feature flags and experimental options.

### Level 4 — Developer

Developer Mode reveals:

- raw JSON;
- API plane and endpoint mapping;
- capability manifests;
- adapter routing;
- correlation IDs;
- event envelopes;
- raw logs after redaction;
- contract and normative-source links;
- command preview where safe;
- implementation diagnostics.

Developer Mode is discoverable and powerful, but off by default.

## 3.3 One obvious primary action

Each view or card has one visually dominant action. Examples:

- **Start session**
- **Run test**
- **Import document**
- **Capture source**
- **Compare**
- **Approve output**
- **Resume Workpoint**

Secondary actions use menus, contextual toolbars, or the inspector rather than competing button rows.

## 3.4 Context follows work

Project, Workstream, Workpoint, node, and authority are selected once and inherited by opened work objects. They should not be repeatedly re-entered.

The UI must still expose scope before any meaningful mutation and block stale, missing, conflicting, or read-only authority.

## 3.5 Reveal exceptions, not constant noise

