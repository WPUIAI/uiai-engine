### Diagnostics and observability

| Original item | Preserved capability |
|---:|---|
| 3 | Server-side diagnostics filtering and bounded summary. |
| 14 | Failed request body capture with redaction and bounds. |
| 20 | Diagnostics keyed per session. |
| 30 | Longer-lived/loss-aware console capture. |
| 31/73 | Optional bounded success response-body capture. |
| 52 | Diagnostic cause/action/Workpoint correlation. |
| 72 | Source-map-aware stack traces. |
| 74 | Separate failure timing from successful request timing. |
| 77 | Severity coding and readable diagnostic presentation. |
| 80 | Capture diagnostics during navigation transitions. |
| 82 | Rich error envelopes with code, retryability, tool, target, and recovery. |
| 84 | Engine readiness before session creation. |
| 101 | Per-tool contract/self-tests surfaced in health. |

Placement: contextual inspector, Activity, Nodes & Services, Developer Mode.

### Reliable interaction primitives

| Original item | Preserved capability |
|---:|---|
| 4 | Auto-wait on navigate/action with explicit advanced waits. |
| 5 | Text, role/name, ref, and CSS selector support. |
| 6 | Async long-running action/job model. |
| 7 | File upload and drag/drop/reorder primitives. |
| 8 | Documented bounded async eval. |
| 9 | Page lifecycle/state including URL, title, ready state, frames, response. |
| 13 | Reduced snapshot dance through selector shortcuts and semantic actions. |
| 21 | Expect/wait-for-text assertion. |
| 22 | Double-click and right-click primitives. |
| 23 | Key-sequence primitive. |
| 27 | Unified smart set-value across control types. |
| 28 | Enumerate interactive elements. |
| 38 | Visibility/enabled/in-viewport preflight. |
| 39 | Indexed/nth target selection. |
| 40 | Atomic set-value alternative. |
| 41 | Pointer/coordinate input. |
| 42 | Native file picker/save/print boundary through safe Tauri/UIAI bridges. |
| 45 | Force/async close and safe cleanup. |
| 46 | Stale ref detection and re-snapshot/recovery. |
| 47 | Precise hidden/obscured/out-of-viewport failure reasons. |
| 70 | True reload primitive. |
| 71 | Back/forward with diagnostics. |
| 78 | Focused-element state. |

Placement: Live advanced controls, UIAI Scenario runner, Test Lab assertions, Capability catalog.

### Session, authentication, and environment state

| Original item | Preserved capability |
|---:|---|
| 10 | Named sessions. |
| 12 | Reusable secure auth profiles. |
| 43 | Sticky viewport state. |
| 44 | Enumerate open sessions. |
| 54 | Shared/compatible Focusa–UIAI auth-profile reference without secret leakage. |
| 57 | Open/session binding to verified Workpoint scope. |
| 59/85 | Resume last relevant UIAI session across continuity. |
| 61 | Park/persist/restore browser state after disconnect/restart. |
| 62 | Controlled cookie/localStorage/state injection. |
| 63 | Custom user-agent. |
| 64 | Dark color-scheme emulation. |
| 65 | Reduced-motion emulation. |
| 66 | Auditable response interception/stubbing. |
| 67 | Per-session CPU/network throttling with clear indicator. |
| 68 | Disable JS/CSS for fallback testing. |
| 69 | Cache bypass. |
| 81 | Capacity queue and cooperative broker behavior. |
| 83 | Remote-host/endpoint constraints and recovery documentation. |

Placement: Live session setup/inspector, Test Lab Environments, Nodes & Services, secure Settings.

### Scenario, flow, and reuse

| Original item | Preserved capability |
|---:|---|
| 16 | Stateful scenario runner. |
| 17 | Reusable selectors/step library. |
| 19 | Domain presets for common admin/product patterns. |
| 24 | Date/time/timer mocking for tests. |
| 76 | Structured scenario comparison. |
| 92 | Starter templates for E2E audits. |
| 93 | Replay/retry with bounded recovery strategies. |
| 94 | Flow/session lint for redundant or fragile steps. |
| 100 | Audit history, replay suggestion, and validity/state-hash checks. |

Placement: Test Lab Flows/Environments, Automations Recipes, Capabilities.

### Accessibility verification

| Original item | Preserved capability |
|---:|---|
| 34/48 | Accessibility/axe-core report primitive. |
| 35 | Keyboard navigation test. |
| 36 | Focus-trap verification. |
| 37 | Contrast check. |
| 79 | ARIA landmark dump. |

Placement: Test Lab verification and Studio / Analyze.

### Focusa integration

| Original item | Preserved capability |
|---:|---|
| 49 | Screenshot/evidence capture with structured handle and optional Workpoint link. |
| 50 | UIAI sessions represented as resolvable active work objects. |
| 51 | Focusa Workpoint-to-UIAI action bridge. |
| 53 | Tool Doctor understands UIAI failures and pressure. |
| 55 | Recapture from prior evidence context. |
| 56/75 | Evidence-to-evidence structured diff. |
| 58 | UIAI action/run state contributes to Trajectory/Workpoint proof. |
| 60 | Unified Workpoint web/session context model. |
| 99 | Long-term Focusa-aware UIAI where session concretely realizes Workpoint web context. |

Placement: pervasive Focusa context, Evidence, Live, Test Lab, Workpoint restoration.

### Agent and developer experience

| Original item | Preserved capability |
|---:|---|
| 11 | Save rendered HTML/DOM and filtered console as artifacts. |
| 86 | Explicit screenshot encoding/output clarity. |
| 87 | Consistent tool naming policy. |
| 88 | Consistent parameter naming/viewport shape. |
| 89 | Clear first-line screenshot success result. |
| 90 | Inline schema/help discovery. |
| 91 | Canonical UIAI skill/cookbook patterns. |
| 98 | Intent-level `do/scenario/extract` primitives above atomic primitives. |
| 102 | UIAI playground/fixture environment for onboarding and smoke. |

Placement: Capabilities, Help, Pi/MCP/CLI parity, Test Lab templates, Developer Mode.

## A.5 Existing UIAI route families preserved

All current route families in Section 14 remain present in the capability catalog even when default Cockpit exposure is gated. This includes browser/session, FPV, screenshots, shares, search, Markdown, agent packets, errors, tools, critique, UI reverse, section detection, layout/style, copilot, reference, intake, workflow, usage/admin, extension, memory, intelligence, training, 2FA, media, design system, content map, block recipes, comparison, migration, CAPTCHA, events, and legacy dashboard maintenance state.

## A.6 Documents feature preservation

The unified spec preserves all previously proposed strong document capabilities:

- PDF/Office/image/email ingestion;
- native extraction and OCR;
- page/region citations;
- table/form/annotation/attachment/signature/metadata inspection;
- Markdown/JSON/JSONL semantic output;
- merge/split/reorder/rotate/crop/resize;
- stamps/watermarks/numbers;
- forms and flattening;
- encryption, optimization, PDF/A;
- redaction and redaction verification;
- document generation and packet assembly;
- comparison;
- signatures and later signing;
- immutable originals and derivatives;
- job lifecycle;
- recipes;
- hostile-input isolation;
- Focusa evidence handoff.

## A.7 Test Lab feature preservation

The unified spec preserves:

- Maestro web/mobile integration;
- packaged Tauri E2E through a native-appropriate runner;
- UIAI browser scenarios;
- per-project flow libraries;
- run environments;
- live FPV during execution;
- synchronized video, steps, assertions, screenshots, logs, diagnostics, and reports;
- baseline and visual regression;
- accessibility verification;
- operator intervention;
- replay, retry, fork, and flow generation from recording;
- artifacts scoped per project/run;
- Focusa prediction/evidence/Workpoint feedback;
- future verification receipts.

---

## A.8 Interactive visual and feedback-report feature preservation

The generic visual/interactive report proposal is preserved in a conformed form:

| Generic feature | Conformed Cockpit placement and guardrail |
|---|---|
| Structured self-documenting report | ReviewReport schema and Report Canvas in Evidence/work-object tabs |
| Metadata header | Outcome header sourced from mission, Workpoint, worker, timing, status, and artifact refs |
| Executive summary | Bounded summary with uncertainty and strongest recommendation |
| Real screenshots and recordings | Typed actual-capture evidence with source/event/viewport/hash and immutable originals |
| Annotations | Separate overlay objects with author, geometry, timestamp, and original ref |
| Before/after comparison | Studio/Evidence comparison block with lineage and method |
| Step-by-step execution log | Receipt/event-linked timeline, not unbounded raw transcript |
| Tool calls and code | Expandable bounded technical detail with redaction and artifact refs |
| Agent self-reflection | Labeled post-run agent assessment; no raw private chain-of-thought |
| Tables, charts, diagrams, custom visuals | Declarative registered report blocks using Cockpit design/security contracts |
| Approve/reject/request changes | Distinct typed review actions; report approval does not settle mission |
| Comments and threads | Persistent report thread with explicit promotion of durable decisions |
| Follow-up task submission | Bounded context bundle creates/proposes Focusa Workpoint/orchestrator task |
| Buttons that run prompts/subtasks | Capability-bound interaction manifests through guards; no arbitrary direct execution |
| Editable canvases and comparisons | Report Canvas with governed annotations, selections, variants, and approved renderers |
| Durability, search, versioning | ReviewReport lifecycle, versions, freshness, frozen snapshots, global search |
| Shareable links | Audience-scoped, redacted, expiring, revocable, exact-version shares |
| PDF/HTML/Markdown export | Static report snapshots preserving attribution and interaction state |
| Cryptographic integrity | Source manifest hashes and optional signatures |
| Task-system attachment | Workpoint/evidence linkage without becoming parallel task authority |
| Success metrics | Review time, context carryover, evidence sufficiency, follow-up success, false-done reduction |

The generic proposal is intentionally narrowed where it conflicts with the greater architecture: reports do not own mission state, raw reasoning is not exposed, arbitrary HTML/JS is not accepted, comments do not grant authority, and visual polish does not substitute for receipts or independent verification.

## A.9 Governed agentic-browser best-practice preservation

The following ABPS-1.0 concerns are preserved and integrated:

- Mission Experience / Mission Deck → Overview, Context Control, Workpoints, work-object tabs, Activity, Inspector;
- Mission Kernel → Focusa authority boundary;
- task graph, workers, leases, locks, retries, budgets → Automations, Activity, Nodes & Services, Session Broker, Track G;
- deterministic Authority Kernel and Capability Grants → ScopeGuard, AuthorityGuard, approval leases, risk classes;
- actuator-neutral Action Router → Capabilities and route decision contracts;
- Mission and Completion Contracts → governed mission model and Mission Deck;
- action proposals and receipts → approval previews, events, Evidence;
- verification, false-done prevention, contradiction, and settlement → Test Lab, Evidence, Mission Deck;
- hostile-content, data-flow, credential, structured-tool, and file security → Trust/Security and document/browser guardrails;
- idempotency, classified retry, compensation, dead-letter, checkpoint, concurrency → Automations, Jobs, Session Broker, Track G;
- memory separation and controlled promotion → Focusa boundary and recipe/procedural-memory rules;
- worker identity and delegation → Nodes & Services and capability grants;
- budgets/economic routing and verified-outcome metrics → Performance and operational tests;
- incident response, red-team, maturity, and conformance → testing/release requirements and Annex E.

No ABPS responsibility is reassigned to Cockpit merely because Cockpit renders it. Canonical ownership remains with Focusa, UIAI, an explicit orchestrator/policy service, or an authorized external verifier as defined in Section 4.1.

---

# Annex B — Information architecture rationale

## B.1 Why task-oriented workspaces replace product-plane navigation

The current product-plane navigation—UIAI Engine, Focusa Local, Focusa Cloud, AI API—is technically honest but forces the user to understand backend ownership before completing work.

The revised model keeps ownership visible in Context, approvals, inspector, and Capabilities while organizing normal work by recognizable goals:

- watch;
- test;
- read/write documents;
- research;
- create/compare;
- automate;
- prove;
- maintain systems.

This reduces cognitive translation without hiding technical truth.

## B.2 Why Activity includes Jobs

Jobs, approvals, notifications, and event history are different views of “what is happening or happened.” Combining them prevents four utility destinations while preserving filtered segments and deep job views.

## B.3 Why Capabilities is first-class

A capability catalog solves two competing goals:

1. the default UI must remain calm;
2. everything built in UIAI Engine must remain visible and discoverable.

The catalog is the complete technical map. Workspaces are the curated task map.

## B.4 Why cards remain but do not define navigation

Cards are excellent for:

- bounded actions;
- dashboard summaries;
- onboarding;
- system readiness;
- approval previews;
- reusable manifest-driven UI.

They are poor as the only structure for live mirrors, documents, videos, test timelines, diff canvases, and multi-object work. The unified architecture keeps cards as a component and contract, not the whole application metaphor.

## B.5 Why technical detail lives in the inspector

The inspector provides a stable location for metadata, scope, evidence, history, and raw detail. This prevents each workspace from inventing a different “advanced” drawer and lets the main canvas remain focused on the task.

---

# Annex C — Normative source map

