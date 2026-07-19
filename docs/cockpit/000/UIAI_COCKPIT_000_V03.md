Evidence items SHOULD include source, timestamp, origin, collection method, integrity hash, sensitivity, mission scope, confidence, related predicate, retention, redaction state, and lineage.

Cockpit’s existing `ArtifactRef`, Focusa evidence handles, job manifests, diagnostics references, screenshots, recordings, test reports, documents, and research captures become evidence inputs to this broader receipt contract.

## 4.10 Independent verification, contradiction, and settlement

Execution and verification SHOULD be logically separate for consequential outcomes.

A verifier may use deterministic checks, an independent model, external API/query, network confirmation, email, document parsing, transaction lookup, test runner, screenshot/DOM comparison, or human review.

Requirements:

- model confidence alone cannot satisfy a consequential predicate;
- the mission remains incomplete when required evidence is missing;
- conflicting evidence is preserved, marked uncertain, and reconciled rather than silently discarded;
- contradictions block settlement when they affect required predicates;
- asynchronous outcomes remain `PROVISIONALLY_COMPLETE` or `WAITING_EXTERNAL` until settlement;
- evidence sufficiency is evaluated per predicate, not as a single vague score.

## 4.11 Reliability controls

Every consequential task MUST define preconditions, expected postconditions, how each is checked, and failure behavior.

The orchestration layer and adapters MUST support:

- locks, leases, serialization, or version checks for shared resources;
- idempotency keys or reconciliation strategies for side effects;
- search-before-create or confirmation lookup where external idempotency is absent;
- bounded retries classified as transient, semantic, authorization, external, or irreversible/unknown;
- reconciliation before retrying an action that may already have succeeded;
- compensating actions where possible, without calling compensation “undo”;
- dead-letter/review state for exhausted or unsafe work;
- checkpoints that include mission version, task state, worker leases, context references, evidence, used authority, remaining budgets, uncertainty, and recovery instructions;
- worker replacement without loss of canonical history.

The existing browser Session Broker proposal is part of this requirement: session ownership, lease expiry, safe parking, restore, queueing, and cross-project visibility are not optional polish for shared execution.

## 4.12 Untrusted content, data flow, credentials, and files

The system assumes that websites, documents, emails, tool descriptions, API responses, advertisements, scripts, downloaded files, and user-generated content may be malicious.

Foundational rule:

> **Untrusted content may inform a mission, but it may never grant itself authority.**

Requirements:

- untrusted content is labeled as data, not system/user/policy instruction;
- instruction provenance is retained where practical;
- sensitive data carries classification and provenance;
- supported data classes include Public, Internal, Personal, Confidential, Credential, Financial, Health, Legal, Identity, and Restricted;
- data egress is checked against destination, mission, grant, and data policy;
- data from one mission is unavailable to another by default;
- secrets use opaque handles where practical and remain outside model context;
- credentials are origin-bound, short-lived, revocable, and brokered without plaintext disclosure where possible;
- authentication state is reverified after suspicious navigation or context changes;
- origin-supplied structured tools are versioned/hashed, origin-bound, schema-validated, locally risk-classified, and re-evaluated if changed;
- uploads require explicit file authorization and cannot expose unrelated files;
- downloads use isolated directories, size/MIME checks, hashing, provenance, scanning where appropriate, and retention policy;
- downloaded content never becomes trusted procedure automatically.

## 4.13 Memory separation and controlled learning

The system distinguishes:

| Store | Owner/purpose |
|---|---|
| Canonical mission state | Focusa objective, constraints, decisions, Workpoints, completion/settlement. |
| Task working state | Temporary worker/orchestrator state. |
| Browser/session state | UIAI pages, cookies, storage, refs, lease. |
| Evidence ledger | UIAI artifacts plus Focusa linkage and receipts. |
| Preference memory | Stable, inspectable, scoped user choices. |
| Procedural memory | Validated reusable workflows/recipes. |
| Organizational policy | Roles, approval, routing, data, and risk rules. |
| Quarantine memory | Unverified candidate learnings. |

A website statement, model inference, or single failed/successful run MUST NOT become global procedure or preference automatically.

Promotion follows:

```text
Observation → Candidate learning → Quarantine → Validation
→ Scope assignment → Sanitization → Procedural/preference memory
→ Policy-controlled retrieval
```

Users and authorized administrators must be able to inspect, correct, delete, restrict, and review provenance of durable memory. Cockpit surfaces these operations through Focusa contracts; it does not create a hidden memory store.

## 4.14 Worker identity, ownership, and delegation

Every worker SHOULD have a stable identifier, type, model/runtime identity, owner, role, capabilities, current lease, version, and audit history. High-risk organizational actions require attributable and verified worker identity.

Delegation records:

- original principal;
- delegating principal;
- receiving worker;
- scope and permitted action classes;
- expiration;
- redelegation permission and maximum depth;
- revocation path.

Every active task has a clearly identified owner or lease holder. Multiple agents operate against one canonical mission truth and shared budgets. Supporting work in another repository does not transfer active mission authority.

## 4.15 Budgets, model routing, and economic efficiency

A Mission Contract SHOULD carry budgets for financial spend, tokens, model calls, browser minutes, worker runtime, network use, retry count, human attention, and wall-clock duration.

Hard limits cause stop, replan, or escalation.

Routing SHOULD use the least expensive method that still meets reliability, security, data, evidence, and latency requirements:

- deterministic parser before model inference;
- structured data before vision;
- incremental state before full screenshots;
- small/local model for bounded classification where appropriate;
- larger or independent verifier only when justified;
- human escalation when continued autonomy is more expensive or risky.

Cockpit exposes bounded budget posture and cost per verified outcome without turning the main UI into a finance dashboard.

## 4.16 Operational conformance and maturity

The system SHOULD measure verified completion, settled completion, false-done rate, duplicate side effects, unauthorized-action blocks, intervention/rescue rate, recovery success, retries, evidence completeness, cost per verified outcome, and time to settlement.

Maturity is described as:

| Level | Description |
|---|---|
| **L0** | AI-assisted browser; no consequential autonomy. |
| **L1** | Agent-capable browser; multi-step actions and basic approvals. |
| **L2** | Governed agentic browser; Mission Contract, capability grants, isolation, deterministic policy, action receipts, bounded retries, task graph. |
| **L3** | Verified execution platform; Completion Contract, independent verification, false-done prevention, settlement, recovery, scoped memory, action routing. |
| **L4** | Distributed agentic work platform; multi-worker orchestration, device handoff, organizational authority, delegation, signed receipts, cross-application execution, economic optimization. |

Cockpit UI MUST identify whether a capability is implemented, planned, simulated, surrogate-only, or unavailable. It MUST NOT present an L1 or L2 capability as L3 verified execution merely because a polished interface exists.

---

# 5. User mental model and core objects

## 5.1 The user should think in work, not APIs

The primary mental model is:

```text
Project
  └── Workstream
       └── Workpoint / task
            └── Work objects
                 ├── browser session
                 ├── test run
                 ├── document
                 ├── research capture
                 ├── visual comparison
                 ├── workflow run
                 └── evidence/proof
```

Nodes, daemons, API planes, adapters, runners, and routes support those objects. They do not replace them as the main navigation language.

## 5.2 Core product objects

| Object | Meaning |
|---|---|
| **Project** | Verified durable project identity. |
| **Workstream** | Scoped continuity or thread of work within the project. |
| **Workpoint** | Current mission, action, evidence state, and next step. |
| **Work object** | A user-openable browser session, test, document, capture, comparison, workflow, or proof item. |
| **Artifact** | Immutable or versioned output such as screenshot, video, PDF, report, Markdown, diagnostics bundle, or diff. |
| **Review report** | Versioned, evidence-backed projection of mission/work-object state for human review, decisions, feedback, follow-up, and safe sharing. It is not canonical mission truth. |
| **Evidence ref** | Stable bounded reference that Focusa can link without ingesting raw private blobs. |
| **Job** | Long-running execution with progress, cancellation, approval, and outputs. |
| **Capability** | A bounded action exposed through an adapter and manifest. |
| **Node** | A local, VPS, relay, or cloud execution/authority endpoint. |
| **Runner** | A test execution adapter such as UIAI Scenario, Maestro, or Tauri WebDriver. |
| **Recipe** | A reusable sequence of bounded capabilities. |

## 5.3 Work objects are universal

Every work object can support, when applicable:

- open and pin;
- rename or label;
- inspect;
- attach to Workpoint;
- compare;
- create evidence;
- show history;
- export;
- duplicate or fork;
- reveal source artifact;
- show current job;
- close without destroying backend state;
- reopen from recent activity;
- compose or update a review report from selected canonical sources;
- receive comments or review decisions without granting hidden action authority.

---

# 6. Global information architecture

## 6.1 Stable shell

The current shell primitives remain and are refined:

```text
┌───────────────────────────────────────────────────────────────────────┐
│ Unified toolbar                                                      │
├──────────────┬──────────────────────────────────────┬─────────────────┤
│ Sidebar      │ Work-object tabs                     │ Inspector       │
│              ├──────────────────────────────────────┤                 │
│ Workspaces   │ Active workspace / selected object   │ Summary         │
│ Collections  │                                      │ Details         │
│ Favorites    │                                      │ Evidence        │
│              │                                      │ History         │
│              │                                      │ Raw (Developer) │
├──────────────┴──────────────────────────────────────┴─────────────────┤
│ Activity Bar: errors · running · approvals · evidence · sync         │
└───────────────────────────────────────────────────────────────────────┘
```

The shell remains recognizable across every capability.

## 6.2 Sidebar hierarchy

The sidebar is organized by user work, not backend product plane.

### WORK

- **Overview**
- **Live**
- **Test Lab**
- **Documents**
- **Research**

### CREATE

- **Studio**
- **Automations**

### PROVE

- **Evidence**
- **Activity**

### SYSTEM

- **Nodes & Services**
- **Capabilities**

### FOOTER

- **Settings**
- **Help**

Groups may collapse. Users may pin or hide nonessential workspaces, but all enabled capabilities remain discoverable through search and Capabilities.

### Default visibility

On first launch, show:

- Overview
- Live
- Test Lab
- Documents
- Evidence

Research, Studio, Automations, Activity, Nodes & Services, and Capabilities remain in the sidebar but may use quieter styling until first use. This gives the primary product a calm initial posture without hiding its breadth.

## 6.3 Unified toolbar

The toolbar follows a stable spatial grammar.

### Leading area

- sidebar toggle;
- back/forward history;
- project/context control.

### Center area

- active work-object title;
- compact state subtitle;
- optional breadcrumb for nested document/test/research objects.

### Trailing area

- context-specific primary action;
- share/export when applicable;
- global search / command palette;
- inspector toggle;
- overflow menu.

The toolbar should not contain a row of permanent status chips.

## 6.4 Context control

The current scope strip becomes a single compact **Context Control**.

Collapsed state:

```text
VoiceSpot · OAuth reconnect · Verified
```

Optional small trailing indicators:

```text
Mac Studio · Local Only
```

Expanded popover:

- Project
- Workstream / continuity
- Workpoint
