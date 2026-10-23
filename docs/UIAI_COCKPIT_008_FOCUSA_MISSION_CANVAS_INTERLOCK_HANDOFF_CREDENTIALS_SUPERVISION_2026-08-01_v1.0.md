# UIAI-COCKPIT-008 — Focusa Mission Canvas Interlock, Contextual Handoff, Scoped Credentials, and Supervisable Execution

**Status:** Proposed normative amendment v1.0; documentation-first; implementation not implied  
**Date:** 2026-08-01  
**Depends on:** UIAI-COCKPIT-000 through 007; Focusa Specs 119, 133, 135A, 135C, 135E, 135G, 135I, 135J, 136, 137, 139, and 140  
**External companion:** `Startempire-Wire/focusa/docs/current/SPEC135_UIAI_COCKPIT_MISSION_CANVAS_INTERLOCK_2026-08-01.md`  
**Machine companions:** UIAI-COCKPIT-008-C01 through C06

---

## 0. Constitutional law

UIAI Engine Cockpit and Focusa Mission Canvas are distinct GUIs. They SHALL be tightly interwoven through generated contracts, stable bindings, durable events, and typed actions, but they SHALL NOT duplicate mission authority.

```text
Focusa
  canonical mission, scope, Workpoint, authority, Evidence,
  Receipt, completion, settlement, and durable event owner

Focusa Mission Canvas
  canonical mission and professional-workspace projection:
  Work Surfaces, Work Rail, steering, follow-up, proof, and continuity

UIAI Engine
  browser, research, notebook, document, test, media,
  resource, credential-injection, execution, and proof-production plane

UIAI Engine Cockpit
  rich execution, verification, inspection, and operator-control shell
  that can host conformant Focusa projections
```

The previous phrase `Cockpit = Mission Experience layer` is superseded wherever it implies that Cockpit owns canonical mission state, Workpoints, Work Rail, steering, next-safe-action cognition, Evidence meaning, completion, or settlement.

The product rule is:

```text
Mission Canvas determines what the work is and what it means.
Cockpit performs, observes, controls, diagnoses, and proves the work.
```

A Cockpit feature is nonconformant if it creates a second:

- project or mission store;
- Workpoint or task authority;
- Work Rail;
- steering or follow-up queue;
- approval or permission system;
- mission schedule;
- credential-use authority;
- Evidence meaning or completion reducer;
- Focusa event history.

---

## 1. Product objective

This amendment converts the current loose Focusa/Cockpit relationship into a precise cross-GUI interlock that allows an operator to:

- see exact Focusa mission context while working in any Cockpit browser, research, document, notebook, test, report, visual-comparison, automation, or media object;
- open the corresponding Mission Canvas Work Surface without searching or rebinding;
- open a UIAI execution object from Mission Canvas with exact session/context/target identity;
- hand a page, selection, document, notebook result, test, or artifact to a real Focusa Workpoint through preview and commit;
- take over browser/computer execution and safely return control with a machine-readable state delta;
- authorize credential use through Focusa while UIAI retains secret custody and injection;
- run recurring mission work from one Focusa context policy and one UIAI execution manifest;
- accept work from browser, email, mobile, Agent Inbox, and other surfaces through one typed intake primitive;
- disconnect, restart, and resume both GUIs without scope adoption, duplicated work, or false completion.

These outcomes SHALL be delivered as thin integration primitives over existing Cockpit workspaces and Focusa Mission Canvas. They SHALL NOT create a new permanent workspace called Computer, Agent Inbox, Mission, Tasks, or Work Rail.

---

## 2. GUI boundary

### 2.1 Focusa Mission Canvas owns

- project, workstream, Workpoint, and work-item projections;
- Mission, Trajectory, current work, next work, and next-safe action;
- Work Surfaces and exact Attachments;
- Work Rail;
- steering and follow-up queues;
- capability, permission, approval, and authority posture;
- Evidence, Receipts, contradictions, completion, and settlement;
- cross-session contention and proposal resolution;
- canonical recurrence and temporal mission intent;
- restoration of Focusa presentation and session bindings.

### 2.2 Cockpit owns

- browser, notebook, document, research, test, media, and report execution surfaces;
- browser sessions, contexts, targets, profiles, and FPV;
- DOM/accessibility, screenshot, visual, console, network, and diagnostic detail;
- kernels, cells, variables, outputs, environments, and computational execution;
- local execution jobs, runners, resource posture, logs, and recovery;
- secret storage, opaque handles, origin-bound injection, and rotation;
- technical artifact inspection and evidence-candidate production;
- actuator-level immediate safety freeze and operator input control.

### 2.3 Shared information, separate ownership

The same work appears in both GUIs through stable refs:

```text
Focusa Work Surface
  ↔ UIAI Work Object

Focusa Attachment
  ↔ UIAI session/context/target/job

Focusa Evidence/Receipt refs
  ↔ UIAI artifact/diagnostic/execution refs

Focusa operator intervention
  ↔ UIAI control lease and operator delta

Focusa CredentialUseGrant
  ↔ UIAI SecretBinding

Focusa WorkflowContextPolicy
  ↔ UIAI ExecutionContextManifest
```

The refs connect objects; they do not merge their authority domains.

---

## 3. Cockpit Focusa projection levels

Cockpit SHALL expose Focusa information with progressive disclosure.

### 3.1 Mission Context Strip

Every Focusa-bound Cockpit work object SHALL include a compact persistent strip showing:

- project and workstream;
- Workpoint and work item;
- Focusa Work Surface;
- mission/session status;
- one next-safe action supplied by Focusa;
- authority or approval posture;
- proof posture;
- source revision, event cursor, and freshness;
- `Open Mission Canvas`.

The strip MUST NOT calculate next work, display a second Work Rail, or treat Cockpit focus as canonical scope.

Example:

```text
Focusa / Main · WP 019… Verify authenticated checkout
UIAI Work Surface: Billing Admin · Running · Proof incomplete · 1 approval
Next safe action: Capture generated invoice
[Open Mission Canvas]
```

### 3.2 Mission Inspector

Every Focusa-bound work object MAY expose a native Cockpit inspector with bounded tabs:

```text
Mission
Current work
Next safe action
Authority
Proof
Attachments
Contention
Receipts
```

All actions use generated Focusa action bindings. Cockpit cannot infer routes, permissions, confirmation posture, or completion from labels.

### 3.3 Hosted Mission Canvas dock

Cockpit MAY host an actual Focusa-owned Mission Canvas projection as a dock or inspector region. It MUST use the approved Focusa generated client, A2UI web core, Lit renderer, Focusa Svelte Custom Elements, and generated action bindings. It MUST NOT implement a second Focusa SurfaceModel, reducer, operation catalog, or Work Rail.

### 3.4 Full Mission Canvas work-object tab

Cockpit MAY open the complete Focusa Mission Canvas as a first-class work-object tab for project overview, multiplexed Work Surfaces, Work Rail, steering, follow-up, contention, Evidence, Receipts, and governed controls.

Cockpit remains the window shell. The tab remains a Focusa-owned projection.

---

## 4. Mission Canvas UIAI projection

Mission Canvas SHALL surface enough UIAI operational truth to govern work without recreating Cockpit.

A UIAI Work Surface projection includes:

```text
UIAI Browser · Billing Admin
Running · healthy · agent controlled
Operator profile · isolated_authenticated · 3 targets
Current: billing.example.com/invoices/123
Credential grant: active until 12:30
Submit action: approval required
Proof: 2 screenshots · network verification pending
[Open in Cockpit] [Take control] [Pause] [Inspect artifacts]
```

Required projected fields:

- UIAI work-object kind and stable reference;
- UIAI session/job, browser context/target, notebook, document, test, or media refs;
- lifecycle and health;
- browser profile, isolation, authentication-sharing, or environment posture;
- node/runner/resource posture;
- current observed target/object and last observation time;
- operator-control state;
- CredentialUseGrant posture without secret material;
- artifact, diagnostic, and proof status;
- retention and cleanup posture;
- exact `Open in Cockpit` deep link.

Mission Canvas MUST NOT embed full browser chrome, raw DOM/network inspectors, secret configuration, notebook editors, complete Test Lab, or full comparison canvases.

---

## 5. Work Surface binding

Every mission-bound UIAI work object SHALL have a versioned `focusa.uiai_work_surface_binding.v1` record.

The binding preserves:

- exact `project_root`, `continuity_id`, Workpoint, and work item;
- Focusa Instance, Session, Attachment, Work Surface, and mission refs;
- UIAI work-object kind and object ID;
- session, context, target, notebook, document, test, job, or report refs;
- access mode and authority posture;
- operator-control state;
- isolation and authentication-sharing posture;
- CredentialUseGrant refs;
- resource and retention policy;
- source revision, event cursor, observation time, and staleness.

Binding laws:

1. Focusa mints canonical scope and binding identity.
2. UIAI supplies stable runtime references.
3. Cockpit-local unbound work is explicitly labeled and cannot appear as canonical project work.
4. Rebinding requires Focusa preview/commit, expected revision, and idempotency.
5. Closing a view is separate from terminating a runtime.
6. Aggregate views remain read-only until an explicit target is selected.
7. Cross-project binding is forbidden by default.
8. Missing or stale bindings produce visible recovery, never silent adoption.

C01 defines the complete binding and projection contract.

---

## 6. Contextual handoff and typed intake

### 6.1 One reusable handoff

Cockpit SHALL provide one outcome-oriented action:

```text
Hand Off to Focusa
```

It is available from browser, Research, Documents, Notebooks, Test Lab, Studio, Evidence, Reports, Automations, and other eligible work objects.

The composer captures:

- current page/object and selected region;
- relevant session/context/target/environment refs;
- screenshots, snapshots, diagnostics, outputs, and artifact refs;
- requested outcome;
- expected evidence;
- trusted operator instruction;
- untrusted page, quoted, document, attachment, model, and tool content;
- retention and cleanup posture.

The user may propose:

- attach to current Workpoint;
- attach to another Workpoint;
- create a new Workpoint proposal;
- send as steering;
- send as follow-up;
- capture as Context/Evidence candidate;
- create a verification task.

Focusa decides and commits the canonical result. Cockpit does not create a local task and synchronize it later.

### 6.2 General `TaskIntakeEnvelope`

Browser, email, mobile, menubar, Slack, Agent Inbox, document, and notebook intake SHALL produce one `focusa.task_intake_envelope.v1` contract.

Only a verified principal's explicit instruction can carry authority. Forwarded text, quotes, page content, attachments, document content, tool output, and model output remain untrusted context.

Cockpit may display `proposed`, `scope review`, `accepted`, `attached`, `rejected`, `duplicate`, or `blocked`, but Focusa owns the lifecycle.

C02 defines handoff, trust, intake, deduplication, preview, commit, and deep-link behavior.

---

## 7. Reversible operator takeover

Takeover is a coordinated protocol, not a local browser toggle.

### 7.1 Focusa intervention state

```text
running
→ pause_requested
→ paused_for_operator
→ operator_intervention
→ resume_proposed
→ resumed | redirected | stopped | blocked
```

Focusa owns the durable session decision and mission consequences.

### 7.2 UIAI actuator-control state

```text
agent_controlled
→ local_freeze
→ operator_controlled
→ operator_delta_capture
→ reobservation_required
→ agent_controlled | terminated
```

UIAI owns the browser/computer control lease and immediate actuator state.

### 7.3 Safety freeze

Cockpit MAY immediately freeze local actuation when continuing could create risk. Until Focusa acknowledges the session pause, the UI MUST say:

```text
Local safety freeze · Focusa reconciliation pending
```

It MUST NOT claim a canonical pause.

### 7.4 Return-control reconciliation

Before agent control resumes, UIAI emits an `uiai.operator_delta_receipt.v1` with observable navigation, form, account, attachment, target, and pending-side-effect changes plus current screenshot/DOM/URL refs.

The agent MUST re-observe. It cannot continue from pre-takeover assumptions.

Mission Canvas `Take control` opens the exact Cockpit work object. Cockpit `Pause` and `Stop` route through Focusa, while the actuator-level freeze remains local and explicitly unreconciled until acknowledged.

C03 defines leases, fencing, state transitions, deltas, reobservation, failure, and restoration.

---

## 8. Scoped Credential Broker

Credential handling SHALL separate authority from custody.

### 8.1 Focusa `CredentialUseGrant`

Focusa owns exact project/workstream/Attachment scope, requesting actor, purpose, capability, origins, operation classes, side-effect ceiling, spend ceiling, use limits, issuance, expiry, approval, revocation, and receipt policy.

The grant contains no raw secret.

### 8.2 UIAI `SecretBinding`

UIAI owns vault/keychain storage, opaque credential refs, target origins, route/method restrictions, injection mechanism, active lease, use counters, rotation, and revocation.

Agents and Focusa receive only opaque refs and availability posture.

### 8.3 Broker law

```text
Focusa can authorize use but never receives the secret.
UIAI can possess and inject the secret but cannot authorize itself.
```

Every consequential use emits a secret-safe receipt connecting the Focusa grant, UIAI binding, origin, operation class, time, result, and revocation posture.

Mission Canvas shows availability, approval, expiry, revocation, origin mismatch, and usage boundaries. Secret setup and rotation remain Cockpit-native.

C04 defines the credential, proxy, lease, receipt, egress, logging, and redaction contract.

---

## 9. Workflow context and recurrence

Mission-bound recurring work SHALL have one Focusa recurrence authority and one UIAI execution account.

### 9.1 Focusa `WorkflowContextPolicy`

Focusa owns:

- mission purpose and scope;
- schedule or trigger;
- temporal authority;
- Workpoint policy;
- completion contract;
- frozen context;
- context refreshed each run;
- context carried forward;
- prohibited stale reuse;
- freshness requirements;
- Evidence and settlement policy.

### 9.2 UIAI `ExecutionContextManifest`

UIAI records:

- actual runner and node;
- browser contexts and targets;
- notebook environment;
- document, data, and model versions;
- credential grant and binding refs;
- tool/provider versions;
- actual inputs and outputs;
- resource use;
- diagnostics and receipts.

Focusa decides what context is valid. UIAI records what was actually used.

An unscoped local utility automation may remain Cockpit-local. Binding it to a project, mission, Workpoint, Evidence, or outcome transfers canonical recurrence and continuity to Focusa.

C05 defines context freshness, recurrence, run binding, execution manifests, retry/reconciliation, and cross-GUI presentation.

---

## 10. Adapter and event architecture

### 10.1 Generated client only

The UIAI Focusa Projection Adapter SHALL be generated from the published Focusa OpenAPI 3.0.3 and JSON Schema contracts. UIAI MUST NOT hand-author duplicate Focusa DTOs, operation descriptors, confirmation policies, permission maps, or action bindings when generation can represent them.

### 10.2 One Focusa history

```text
Focusa SQLite canonical events
→ cursor replay
→ broadcast live tail
→ UIAI Focusa Projection Adapter
→ Cockpit projection cache
→ Context Strip / Inspector / hosted Focusa surface
```

Cockpit cache is a projection, not canonical state.

### 10.3 UIAI result flow

```text
UIAI execution observation
→ stable artifact, diagnostic, control-delta, or execution-manifest ref
→ generated Focusa preview/commit operation
→ Focusa validation and reducer
→ canonical Focusa event
→ both GUIs update
```

UIAI cannot inject canonical Focusa events directly.

### 10.4 Truthful intermediate states

Before Focusa acknowledgment, Cockpit may show only explicit states such as:

- binding pending;
- evidence capture pending;
- Focusa reconciliation pending;
- local safety freeze;
- offline projection;
- stale Focusa state;
- origin mismatch;
- grant expired or revoked;
- external contract unavailable.

It cannot display canonical completed or settled state optimistically.

---

## 11. Deep links and restoration

Deep links use stable opaque refs and optional expected revisions. They never include secrets, cookies, raw artifacts, or inline canonical state.

Opening a Focusa or UIAI deep link SHALL validate:

- exact project/workstream scope;
- capability and permission;
- object existence;
- binding and object revision;
- current runtime generation;
- freshness and retention;
- credential and control posture where applicable.

A stale or ended object opens a recovery surface. It never silently substitutes a different project, session, context, target, Workpoint, notebook, or job.

Cross-GUI restoration SHALL preserve:

- Focusa project/workstream/Attachment/Work Surface identity;
- UIAI work object, runtime, context, target, environment, or job identity;
- event cursor and source revision;
- unread and intervention state;
- operator-control lease posture;
- credential grant posture;
- pending handoff/intake state;
- local unsent drafts separately from canonical state.

---

## 12. Activity and notification separation

Focusa Mission Canvas Activity/History represents mission and project events, Workpoints, Attachments, authority, Evidence, Receipts, and settlement.

Cockpit Activity represents UIAI jobs, actions, technical progress, logs, diagnostics, resource use, and operator-control events.

Cockpit MAY include Focusa-linked entries, but each entry must state whether it is:

- a canonical Focusa event;
- a UIAI operational event;
- a local pending action;
- an evidence candidate;
- a reconciled Receipt.

One event cannot be silently duplicated into two apparently independent histories.

---

## 13. Security requirements

- Treat page, document, email, attachment, tool, and model content as untrusted data.
- Do not allow contextual handoff to smuggle instructions across the verified-principal boundary.
- Never expose cookies, headers, tokens, private keys, browser storage, or secret values to Focusa or artifacts.
- Origin, route, method, operation class, project scope, and grant must all match before credential injection.
- Use expiry, revocation, use limits, and fencing for grants and control leases.
- Do not let browser fallback expand authority, origin access, or data disclosure.
- Do not use a deep link as proof of authorization.
- Redact captures and diagnostics before public or cross-project sharing.
- Keep cross-project browser context and credential reuse forbidden by default.
- Preserve exact principal, actor, node, session, and environment provenance in technical receipts.

---

## 14. Required Cockpit UI components

Implementation SHALL provide reusable components or equivalent shared primitives:

```text
FocusaMissionContextStrip
FocusaMissionInspector
FocusaProjectionHost
FocusaWorkSurfaceLink
ContextualHandoffComposer
TaskIntakeStatus
OperatorControlBar
OperatorDeltaReview
CredentialGrantStatus
SecretBindingInspector
WorkflowContextSummary
ExecutionContextManifestView
CrossGuiRecoveryCard
```

These are not separate workspace products. They are embedded where the existing Cockpit architecture requires them.

Every component supports loading, empty, live, stale, offline, blocked, approval-required, conflict, revoked, expired, unsupported-contract, and recovery states where applicable.

---

## 15. Implementation phases

```text
X0 — freeze cross-GUI ownership and contract digests
X1 — generate Focusa interlock schemas, operation client, and compatibility handshake
X2 — implement Focusa Projection Adapter, replay, cache, and freshness
X3 — implement exact Work Surface ↔ UIAI Work Object binding and deep links
X4 — implement Mission Context Strip and Mission Inspector
X5 — implement contextual handoff and TaskIntakeEnvelope
X6 — implement control lease, takeover, operator delta, and reconciliation
X7 — implement CredentialUseGrant, SecretBinding, proxy, and use receipts
X8 — implement WorkflowContextPolicy and ExecutionContextManifest integration
X9 — host conformant Focusa Mission Canvas projections in Cockpit
X10 — implement complete cross-GUI restoration, security, accessibility, and proof
```

No phase may demonstrate a static UI by bypassing production adapters, generated Focusa actions, or durable event logic.

---

## 16. Functional and visual proof

Every claimed interlock feature SHALL meet UIAI-COCKPIT-006 proof law and the C06 matrix.

Minimum cross-GUI release scenarios:

1. a real Focusa binding hydrates a Cockpit Mission Context Strip;
2. a canonical Focusa event changes the visible Cockpit projection;
3. `Open Mission Canvas` and `Open in Cockpit` resolve the exact bound objects;
4. a Cockpit page/document/notebook handoff creates or attaches real Focusa work through preview/commit;
5. Mission Canvas shows the exact UIAI session/context/target or notebook object;
6. takeover completes local freeze, Focusa intervention, operator control, delta receipt, reobservation, and resume;
7. credential use is blocked without both a matching Focusa grant and UIAI SecretBinding;
8. a secret-safe use receipt is linked without exposing secret material;
9. one Focusa WorkflowContextPolicy drives a UIAI run with one ExecutionContextManifest;
10. reconnect/restart restores exact identity without adopting a replacement scope;
11. close-view and terminate-runtime are visibly distinct;
12. stale, offline, conflict, revoked, expired, origin-mismatch, and unsupported-contract states are captured;
13. desktop, narrow, keyboard, high-contrast, and reduced-motion behavior is proven;
14. screenshots, DOM/accessibility snapshots, diagnostics, contract digests, event cursors, source revisions, build SHA, Evidence refs, and Receipt refs are bound in the release manifest.

A static Mission Context card, local mock task, fixture-only event store, manually supplied screenshot, or successful UIAI action without Focusa commitment cannot satisfy integration.

---

## 17. Closure rule

UIAI-COCKPIT-008 remains implementation-open until the same compatible Focusa and UIAI revisions prove:

```text
one Focusa canonical mission authority
+ one Focusa Mission Canvas work projection
+ one UIAI execution and proof plane
+ generated cross-repository contracts
+ exact Work Surface bindings
+ contextual handoff and typed intake
+ reversible operator takeover and reconciliation
+ Focusa credential grants plus UIAI secret custody
+ one Focusa recurrence authority plus UIAI execution manifests
+ durable event replay and exact deep links
+ actual compiled functional UI and inspected visual proof
```

Two polished GUIs that disagree about scope, Workpoint, session, authority, credentials, recurrence, proof, or completion are a release-blocking failure.