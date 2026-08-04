# UIAI Cockpit Workstream-Scoped Universal Agent Control and Multimodal Visual Workspace Runtime Amendment

**Document number:** `UIAI-COCKPIT-004`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding numbered decisions/amendments:** `UIAI-COCKPIT-001`, `UIAI-COCKPIT-002`, `UIAI-COCKPIT-003`  
**Status:** Proposed normative architecture and implementation amendment  
**Version:** 1.0  
**Date:** 2026-08-04  
**Repository:** `WPUIAI/uiai-engine`  
**Coordination issue:** `WPUIAI/uiai-engine#7`  
**Cross-repository foundation dependency:** `Startempire-Wire/focusa#125`, planned Focusa Spec 158  
**Primary implementation homes:** UIAI Engine authority/API/CLI layers, `apps/cockpit/`, document and visual workers, shared contracts, tool discovery, MCP/Pi exposure, artifact and Evidence integration  
**Scope:** universal Cockpit agent control; GUI/CLI/API/MCP/Pi parity; exact Focusa Workstream binding; semantic Cockpit state; multimodal Studio work objects; spreadsheets; whiteboards; semantic charts; DataViews; dashboards and generated workspaces; real-time human-agent collaboration; Focusa Desktop handoff; Evidence; security; accessibility; headless/offline behavior; dependency governance; migration; testing; release proof

---

# 0. Executive decision

UIAI Engine Cockpit SHALL become a fully semantic, fully agent-addressable work environment.

Every meaningful Cockpit operation SHALL be available through one governed control plane shared by:

- the Svelte/Tauri Cockpit GUI;
- the UIAI CLI;
- REST or equivalent typed local/remote APIs;
- MCP and Pi tools;
- the Focusa-powered agent;
- other authorized agents;
- command palette and automation surfaces.

No meaningful durable Cockpit capability may exist only as a graphical click path.

Cockpit SHALL also gain a shared multimodal visual-workspace runtime supporting:

- spreadsheet work objects;
- whiteboard/canvas work objects;
- semantic chart and visualization work objects;
- first-class versioned DataViews;
- dashboards;
- bounded generated workspaces and interfaces;
- documents, reports, browser/research artifacts, and Evidence as composable referenced objects;
- real-time collaboration between humans and authorized agents;
- bidirectional handoff between UIAI Engine Cockpit, the Focusa-powered agent, and Focusa Desktop.

The implementation SHALL NOT create separate, unrelated control systems for spreadsheets, tldraw, Flint, generated UI, Documents, Studio, or individual Cockpit workspaces. Every adapter inherits the common identity, command, revision, collaboration, receipt, Evidence, recovery, and Workstream contracts defined here.

This amendment is filtered through and constrained by the Focusa foundation correction recorded in `Startempire-Wire/focusa#125`:

> Focusa is one daemon serving many isolated Workstreams. A Workstream is the durable cognitive unit. No canonical cognitive object exists outside an exact Scope plus Workstream. Sessions attach to Workstreams. `ContinuityId`, cwd, path, cached packets, UI focus, and daemon-global active values do not define Workstream authority.

Accordingly, Cockpit SHALL NOT introduce or preserve a global cognitive singleton through its own GUI state, command router, Studio runtime, Focusa integration, cache, collaboration layer, or generated workspace system.

---

# 1. Authority and source precedence

Implementation SHALL apply sources in this order:

1. `UIAI-COCKPIT-000` unified master specification.
2. `UIAI-COCKPIT-001` Interactive Review Reports integration decision.
3. `UIAI-COCKPIT-002` Agent-First Browser amendment and its machine-readable companion.
4. `UIAI-COCKPIT-003` Sidebar Navigation, IA, and DnD implementation amendment.
5. This `UIAI-COCKPIT-004` amendment.
6. Current entitlement, protected-worker, API parity, security, artifact, Evidence, and interoperability specifications.
7. Current repository contracts, route mounts, manifests, schemas, tests, and implementation code.

For Focusa interoperability, this amendment adopts the foundation decision in `Startempire-Wire/focusa#125` and SHALL be reconciled with the normative Focusa Spec 158 when published.

If the eventual Focusa Spec 158 differs materially from an interop assumption in this document, cross-product implementation SHALL stop at the contract boundary until the discrepancy is resolved explicitly. UIAI Engine-local visual work may proceed only where it does not fabricate Focusa state or identity.

This amendment does not authorize UIAI Engine to own Focusa canonical cognition.

---

# 2. Focusa foundation correction applied to Cockpit

## 2.1 Canonical vocabulary

The canonical durable Focusa cognitive unit is **Workstream**.

`Thread` is historical design lineage only. It SHALL NOT appear as a new canonical owner, identifier, storage partition, runtime key, Cockpit object kind, API authority field, or generated UI concept.

The required identity separation is:

```text
ScopeRef
  identifies verified project, host, remote, worktree, and workspace scope

WorkstreamId
  durable identity of one Focusa cognitive workspace

ContinuityId
  continuation lineage or generation inside that Workstream

SessionId
  temporal harness or agent execution identity

AttachmentId / AttachmentKey
  binds a client, Session, Instance, workspace binding, and Continuity generation
  to an exact Workstream

WorkpointId
  durable continuation/checkpoint identity owned by the Workstream
```

## 2.2 Binding correction

Earlier design shorthand using only:

```text
project_root + continuity_id
```

is superseded for canonical Focusa binding.

A path may participate in `ScopeRef` or a workspace binding, but it cannot independently identify Workstream authority. `ContinuityId` cannot substitute for `WorkstreamId`.

The minimum Focusa binding used by UIAI is:

```ts
interface FocusaWorkstreamBinding {
  schema: "focusa.workstream_binding.v1";
  scope_ref: ScopeRef;
  workstream_id: string;
  continuity_id?: string;
  workpoint_id?: string;
  attachment_ref?: string;
  workspace_binding_id?: string;
  focusa_event_head?: string;
  workpoint_revision?: number;
  trajectory_revision?: number;
  authority_ref?: string;
  verified_at: string;
}
```

The concrete generated schema SHALL come from the Focusa/UIAI interoperability contract rather than a handwritten permanent duplicate.

## 2.3 No singleton or global current Workstream

The following are prohibited as canonical mutation or read authority:

- process-global `current_workstream`;
- daemon-global `active_project`;
- Cockpit-server `last_workstream` fallback;
- one global Focusa binding shared by every Cockpit window;
- one global Studio workspace acting as canonical project state;
- latest verified Workstream fallback;
- nearest project or path inference for mutation;
- `ContinuityId`-only resolution;
- Session-only resolution;
- UI focus or selected sidebar item as backend authority;
- cached prior Workstream identity after a binding becomes unavailable.

Presentation state may track what a particular Cockpit window or Work Surface is showing. That state is per client/window Attachment, noncanonical, and may never authorize a command by itself.

## 2.4 Multiplexing invariant

One UIAI Engine and one Focusa daemon may concurrently serve:

- multiple Scopes;
- multiple Workstreams inside one Scope;
- multiple Sessions attached to one Workstream;
- multiple Cockpit windows attached to different Workstreams;
- multiple humans and agents collaborating on one UIAI work object;
- multiple independent UIAI objects attached to one Workstream.

No Workstream switch may mutate another client's binding or replace a daemon-global cognitive aggregate.

## 2.5 Fail-closed behavior

A Focusa-linked canonical operation with absent, stale, conflicting, or ambiguous Scope/Workstream binding SHALL:

- fail before mutation;
- return zero foreign Workstream content;
- provide a typed denial/recovery envelope;
- expose bounded candidate information only where policy permits;
- require explicit reattachment, verification, or operator repair;
- never fall back to a prior, current, latest, nearest, or similarly named Workstream.

---

# 3. Product and authority boundaries

## 3.1 Focusa

Focusa owns:

- Scope and Project identity;
- Workstream identity;
- Focus Stack and Focus State;
- Workpoints;
- Workstream Trajectory and Work Loop;
- canonical tactical Context and memory;
- Context Authority and Capability Grants;
- canonical decisions and completion state;
- Evidence acceptance/linkage;
- Focusa Receipts and settlement meaning;
- canonical replay and migration of Focusa cognition.

## 3.2 UIAI Engine

UIAI Engine owns:

- browser and OS actuation under its authority contracts;
- UIAI work-object identity and storage;
- spreadsheet, whiteboard, chart, DataView, generated workspace, media, report-artifact, and execution-object production;
- visual-object revisions and collaboration state;
- rendering, import/export, transforms, validation, and diagnostics;
- UIAI operation receipts and artifacts;
- live object access and object-level permission enforcement;
- proposing UIAI artifacts and immutable snapshots to Focusa.

## 3.3 Cockpit

Cockpit owns presentation and human interaction:

- workspace navigation;
- tabs and Work Surfaces;
- object selection and visual editing;
- participant and agent presence;
- proposals, previews, approvals, and review;
- semantic state presentation;
- routing intent through registered commands.

Cockpit SHALL NOT own canonical Focusa cognition, action authority, Workpoint settlement, or durable browser memory merely because it renders them.

## 3.4 Focusa Desktop

Focusa Desktop remains a separate Focusa product and distribution. It may present UIAI-owned visual Work Surfaces through shared contracts, but SHALL NOT:

- become a second UIAI browser authority;
- clone UIAI canonical visual-object state without an explicit snapshot or fork;
- treat a screenshot as the control protocol;
- bypass UIAI command, permission, revision, or receipt contracts;
- invent or infer Workstream binding.

## 3.5 Third-party engines

Flint, tldraw, Univer, Hucre, Vega-Lite, ECharts, Plotly, Chart.js, A2UI renderers, and other libraries are implementation engines or codecs.

They SHALL NOT define UIAI canonical:

- object identity;
- Focusa Workstream binding;
- authority;
- collaboration policy;
- Evidence semantics;
- cross-product handoff;
- stable public command contracts.

All third-party APIs SHALL remain behind UIAI-owned adapters.

---

# 4. Governing product model

The Cockpit architecture is:

```text
Focusa-powered agent / human operator / authorized external agent
                              │
                    CLI / MCP / API / GUI
                              │
                              ▼
              Cockpit Universal Control Plane
       ┌─────────────────────────────────────────┐
       │ capability and workspace registry       │
       │ exact Scope + Workstream context         │
       │ queries, commands, events, receipts      │
       │ revisions, leases, proposals, approvals  │
       │ policy, entitlement, recovery            │
       │ semantic state and presentation routing  │
       └─────────────────────┬───────────────────┘
                             ▼
                    Cockpit workspaces
       Overview · Live · Test Lab · Documents · Research
       Studio · Automations · Evidence · Activity
       Nodes & Services · Capabilities · Settings · Help
                             │
                             ▼
                 Shared work-object runtime
       spreadsheets · whiteboards · charts · DataViews
       dashboards · generated workspaces · reports
       browser/research/document/media/artifact refs
                             │
                             ▼
                Focusa Workstream handoff
       live reference · Evidence snapshot · fork · continuation
```

Studio is the primary composition and creative collaboration workspace, but universal agent control applies to every Cockpit workspace and subsection.

---

# 5. Universal Cockpit control plane

## 5.1 Non-negotiable parity invariant

> There shall be no meaningful Cockpit capability available only through the graphical interface.

The GUI, CLI, API, MCP/Pi tools, command palette, and authorized agents SHALL invoke the same semantic operations through the same guard, revision, receipt, event, and recovery contracts.

## 5.2 Domain commands versus presentation commands

### Domain commands

Domain commands inspect or mutate durable UIAI objects or external systems.

Examples:

```text
studio.workspace.create
spreadsheet.apply
whiteboard.apply
visualization.chart.patch
dataview.refresh
testlab.run.start
research.capture.create
evidence.snapshot.capture
automation.run.start
```

Domain commands SHALL work without an open Cockpit window where the capability's declared runtime posture permits.

### Presentation commands

Presentation commands control a particular Cockpit client/window without becoming canonical work authority.

Examples:

```text
cockpit.workspace.open
cockpit.subsection.open
cockpit.object.reveal
cockpit.tab.activate
cockpit.inspector.show
cockpit.panel.focus
cockpit.viewport.fit
cockpit.follow_agent.set
```

Presentation commands require an explicit Cockpit client/window target or verified Attachment. They do not imply a Focusa Workstream switch unless a separate attachment command succeeds.

## 5.3 UI implementation rule

Every actionable durable Cockpit control SHALL bind to a registered capability ID.

Allowed pattern:

```svelte
<Button
  data-capability="testlab.run.start"
  onclick={() => commandBus.execute("testlab.run.start", input)}
>
  Run test
</Button>
```

Prohibited pattern:

```svelte
<Button onclick={() => localStore.startRun()}>
  Run test
</Button>
```

Svelte stores may cache presentation/view-model state. They may not become an independent command authority or canonical work store.

## 5.4 Single registry

One source-controlled registry SHALL describe every first-party workspace capability.

```ts
interface CockpitCapabilityDescriptor {
  capability_id: string;
  title: string;
  description: string;
  workspace_ids: string[];
  object_kinds: string[];
  operation_kind: "query" | "command" | "presentation";
  context_requirement:
    | "none"
    | "uiai_object"
    | "focusa_workstream"
    | "focusa_workpoint"
    | "focusa_attachment"
    | "node";
  input_schema_ref: string;
  output_schema_ref: string;
  event_types: string[];
  side_effect_class:
    | "read"
    | "presentation"
    | "reversible"
    | "persistent"
    | "destructive"
    | "external";
  supports_preview: boolean;
  supports_undo: boolean;
  supports_idempotency: boolean;
  headless_posture: "works" | "partial" | "blocked";
  offline_posture: "works" | "partial" | "blocked";
  required_workers: string[];
  required_permissions: string[];
  required_entitlements: string[];
  approval_policy_ref?: string;
  receipt_policy_ref: string;
  cli_namespace: string;
  recovery_contract_ref: string;
}
```

The exact schema SHALL be generated and versioned. This illustrative shape does not authorize duplicate handwritten DTOs.

## 5.5 Workspace manifest extension

`WorkspaceManifest` from `UIAI-COCKPIT-003` SHALL be extended to reference:

- registered queries;
- registered commands;
- registered presentation operations;
- emitted/consumed events;
- semantic state projections;
- supported object kinds;
- CLI namespace;
- object and Workstream context requirements;
- capability-level offline/headless posture;
- approval, receipt, and recovery policies.

The existing workspace-level `local_only_behavior` field is insufficient for the new runtime and SHALL remain only as a coarse summary.

---

# 6. Exact context and Attachment model

## 6.1 Cockpit client Attachment

Every Cockpit window or headless agent control session that operates under Focusa context SHALL have an explicit client Attachment.

```ts
interface CockpitClientAttachment {
  schema: "uiai.cockpit_client_attachment.v1";
  cockpit_client_id: string;
  window_id?: string;
  agent_session_id?: string;
  focusa_binding: FocusaWorkstreamBinding;
  uiai_workspace_binding_id?: string;
  status: "verified" | "stale" | "conflicting" | "detached" | "blocked";
  created_at: string;
  verified_at: string;
}
```

This Attachment is presentation/control-session state. Focusa remains authoritative for the underlying Workstream identity and authority.

## 6.2 Explicit switching

Selecting a Project, Workstream, Workpoint, card, Work Surface, or deep link SHALL:

1. resolve the exact target;
2. verify Scope and Workstream identity;
3. create or update the client Attachment;
4. emit a typed Attachment receipt;
5. refresh all affected semantic projections;
6. invalidate scoped caches and subscriptions from the prior Attachment;
7. never mutate another window's Attachment implicitly.

## 6.3 Commands carry context

A Focusa-bound domain command SHALL carry an exact verified Workstream envelope or an Attachment reference that resolves to one.

```ts
interface CockpitCommandEnvelope<TInput = unknown> {
  schema: "uiai.cockpit_command.v1";
  operation_id: string;
  capability_id: string;
  actor: ActorRef;
  cockpit_attachment_ref?: string;
  focusa_binding?: FocusaWorkstreamBinding;
  target: {
    workspace_id: string;
    object_ref?: string;
    subobject_ref?: string;
  };
  base_revision?: number;
  idempotency_key: string;
  intent: string;
  mode: "preview" | "commit";
  input: TInput;
  requested_presentation?: PresentationRequest;
}
```

Rules:

- UIAI-local unbound object operations may omit `focusa_binding` where the capability permits.
- Linking, proposing Evidence, checkpointing, or mutating Focusa relationships requires exact binding.
- Server-side global current state may not fill a missing binding.
- Idempotency, lease, cache, and recovery keys SHALL include exact Scope + Workstream when bound.

## 6.4 Result contract

```ts
interface CockpitOperationResult<T = unknown> {
  schema: "uiai.cockpit_result.v1";
  operation_id: string;
  capability_id: string;
  status:
    | "completed"
    | "previewed"
    | "blocked"
    | "conflict"
    | "failed"
    | "cancelled";
  target_ref?: string;
  base_revision?: number;
  new_revision?: number;
  changed_refs: string[];
  receipt_ref?: string;
  render_refs: string[];
  evidence_proposals: string[];
  warnings: TypedWarning[];
  recovery?: TypedRecovery;
  output?: T;
}
```

---

# 7. Semantic Cockpit state

Agents SHALL verify Cockpit state semantically rather than relying on screenshots.

A bounded semantic projection SHALL expose, per client/window:

- protocol version;
- connection and compatibility state;
- client Attachment and exact Workstream binding;
- active workspace and subsection;
- active work object and revision;
- open tabs and Work Surfaces;
- current selection or bounded subobject reference;
- visible registered commands;
- participant and agent activity;
- pending proposals and approvals;
- blocked/recovery states;
- jobs and renders;
- layout/panel state;
- freshness and event cursor.

Screenshots remain useful for visual Evidence and debugging. They are not the command or state protocol.

---

# 8. CLI, API, MCP, Pi, and GUI parity

## 8.1 CLI surface

The stable low-level CLI SHALL support:

```text
uiai cockpit discover
uiai cockpit status
uiai cockpit state
uiai cockpit workspaces list
uiai cockpit capabilities list|search|describe
uiai cockpit attach|detach|switch
uiai cockpit workspace open
uiai cockpit object list|open|inspect
uiai cockpit query <capability-id>
uiai cockpit call <capability-id>
uiai cockpit watch
uiai cockpit proposal list|preview|accept|reject
uiai cockpit receipt show
uiai cockpit undo
uiai cockpit retry
uiai cockpit present
uiai cockpit doctor
```

JSON mode SHALL be stable and complete. Commands SHALL return operation and receipt references. Typed denial/recovery classes SHALL use distinct exit behavior.

Local CLI selection convenience is nonauthoritative. A CLI command requiring Focusa Workstream context must either carry exact context, reference a verified Attachment, or fail closed.

## 8.2 MCP and Pi

MCP and Pi tools SHALL wrap the same registry and command/query contracts. They SHALL NOT maintain separate per-tool authority logic or a second object taxonomy.

Progressive discovery SHALL expose only capabilities relevant to the current task, object type, entitlement, policy, and verified Attachment.

## 8.3 GUI parity gates

For every implemented capability:

- registry entry exists;
- typed input and output exist;
- GUI can invoke or present it where applicable;
- CLI can invoke or present it;
- agent tool can invoke or present it;
- API contract is documented/generated;
- scope, permission, entitlement, side-effect, and approval behavior match;
- result is semantically verifiable;
- receipt and recovery behavior are tested;
- headless/offline posture is truthful.

---

# 9. Shared Cockpit work-object model

## 9.1 Canonical UIAI object

```ts
interface CockpitWorkObject {
  schema: "uiai.cockpit_work_object.v1";
  object_id: string;
  object_ref: string;
  owner_plane: "uiai_engine";
  kind:
    | "spreadsheet"
    | "whiteboard"
    | "chart"
    | "dataview"
    | "dashboard"
    | "generated_workspace"
    | "document"
    | "research"
    | "browser_session"
    | "image_asset"
    | "report"
    | "artifact";
  title: string;
  revision: number;
  lifecycle: "active" | "archived" | "deleted";
  focusa_binding?: FocusaWorkstreamBinding & {
    role:
      | "reference"
      | "working_material"
      | "active_deliverable"
      | "decision_record"
      | "evidence_candidate";
  };
  snapshot_ref: string;
  operation_log_ref: string;
  preview_refs: string[];
  export_refs: string[];
  created_by: ActorRef;
  updated_by: ActorRef;
  created_at: string;
  updated_at: string;
}
```

UIAI Engine owns the object's canonical revision. Focusa binding attaches project meaning without transferring UIAI object ownership.

## 9.2 Binding states

A UIAI object may be:

- **unbound:** UIAI-local, not linked to Focusa cognition;
- **Workstream-bound:** linked to exact `ScopeRef + WorkstreamId`;
- **Workpoint-bound:** linked to an exact Workpoint in that Workstream;
- **Evidence candidate:** immutable snapshot proposed to Focusa;
- **detached:** prior link preserved in history, no current active binding.

A project-only Focusa binding is prohibited. UIAI-local organization may use its own project metadata, but it may not masquerade as canonical Focusa association.

## 9.3 Shared lifecycle

```text
create
→ optionally attach to Workstream/Workpoint
→ inspect
→ edit or propose
→ review
→ checkpoint object revision
→ render/export
→ capture immutable Evidence candidate
→ Focusa accepts/rejects/associates
→ continue from newer mutable revisions
```

---

# 10. Studio workspace architecture

## 10.1 Studio is a composition environment

Studio SHALL be a shared creative and analytical environment where humans and agents work on multiple structured visual objects under one exact Workstream Attachment.

A Studio space may contain:

- spreadsheets;
- whiteboards;
- charts;
- DataViews;
- dashboards;
- generated workspaces;
- documents;
- browser/research captures;
- report sections;
- Evidence snapshots;
- media and artifacts.

## 10.2 Studio workspace contract

```ts
interface StudioWorkspace {
  schema: "uiai.studio_workspace.v1";
  workspace_id: string;
  title: string;
  revision: number;
  focusa_binding?: FocusaWorkstreamBinding;
  participants: Participant[];
  object_refs: string[];
  active_object_ref?: string;
  policies: {
    default_agent_access: "observe" | "annotate" | "propose" | "edit";
    destructive_actions: "deny" | "confirm" | "delegated";
    external_publish: "confirm";
    executable_content: "deny" | "sandboxed" | "trusted_extension";
  };
  created_at: string;
  updated_at: string;
}
```

A Studio workspace is a UIAI composition object. It does not own Focusa Focus State, Workpoints, Trajectory, or Work Loop.

## 10.3 Studio IA amendment to `UIAI-COCKPIT-003`

The permanent Studio object taxonomy SHALL become:

```text
Studio
  Spaces
  Whiteboards
  Visualizations
  Dashboards & Generated
  Assets
```

`Capture`, `Compare`, `Analyze`, `Design`, and `Produce` SHALL remain supported as creation intents, commands, recipes, filters, or command-palette entry points. They SHALL NOT be the durable object taxonomy or separate canonical stores.

Documents SHALL own document and spreadsheet library lifecycle, including import, inbox, recent, pinned, templates, generated, and file management. Spreadsheet objects may be opened and composed inside Studio spaces.

This amendment does not create top-level Charts, Whiteboards, Dashboards, Generated UI, or Reports workspaces. Reports remain under Evidence and contextual work-object tabs.

## 10.4 Studio layout

The default Studio composition SHALL include:

```text
Context / Workstream header
Object library or collection pane
Work-object tabs
Primary editor/viewport
Universal Inspector
Participant/agent presence
Activity and proposal bar
```

The Universal Inspector SHALL consistently expose:

- Summary;
- Focusa Workstream/Workpoint binding;
- Participants;
- Structure or Data;
- Agent access;
- Evidence;
- Versions;
- Activity;
- Exports;
- Developer data.

---

# 11. Participants, presence, and collaboration modes

```ts
interface Participant {
  actor_id: string;
  actor_type: "human" | "focusa_agent" | "uiai_agent" | "external_agent";
  role:
    | "owner"
    | "coordinator"
    | "creator"
    | "researcher"
    | "reviewer"
    | "observer";
  permissions: {
    read: boolean;
    annotate: boolean;
    propose: boolean;
    edit: boolean;
    export: boolean;
    publish: boolean;
  };
  presence?: {
    object_ref?: string;
    subobject_ref?: string;
    activity?: string;
    last_seen_at: string;
  };
}
```

Supported collaboration modes:

- **Observe:** inspect only.
- **Annotate:** comments and overlays only.
- **Propose:** agent operations create a preview/proposal.
- **Edit:** bounded reversible operations commit directly.
- **Delegated:** approved capability class may commit within explicit limits.

Agent presence SHALL be visually distinct from human presence.

Human controls SHALL include:

- follow agent;
- stop following;
- pause new agent commands;
- review proposals;
- take object/region/range control;
- return control;
- revoke delegation;
- inspect receipts and reasoning summary.

Raw hidden model chain-of-thought is not displayed or stored as Evidence. Observable intent, constraints, actions, receipts, limitations, and bounded assessments may be displayed.

---

# 12. Shared transaction, revision, lease, and event model

## 12.1 Transaction envelope

```ts
interface StudioTransaction {
  schema: "uiai.studio_transaction.v1";
  transaction_id: string;
  capability_id: string;
  workspace_id: string;
  object_ref: string;
  actor: ActorRef;
  cockpit_attachment_ref?: string;
  focusa_binding?: FocusaWorkstreamBinding;
  base_revision: number;
  idempotency_key: string;
  intent: string;
  mode: "preview" | "commit";
  lease_ref?: string;
  operations: unknown[];
  approval: {
    required: boolean;
    policy_ref?: string;
    reason?: string;
  };
}
```

## 12.2 Execution path

```text
resolve exact object and Attachment
→ verify Scope + Workstream when bound
→ authorize capability and side effect
→ validate base revision and dependencies
→ acquire object/region/range lease when required
→ preview and validate
→ execute atomically or through bounded subtransactions
→ recalculate/render
→ append operation event
→ emit receipt
→ update semantic state
→ optionally propose Evidence/checkpoint/continuation
```

## 12.3 Conflict rules

- Non-overlapping, compatible same-object operations MAY commit optimistically.
- Overlapping or decisional conflicts SHALL produce an explicit conflict/proposal state.
- Same Workstream compatible replicated operations MAY converge through the selected object engine.
- Same Workstream conflicting decisions require explicit review or PRE-equivalent resolution where Focusa meaning is affected.
- Different Workstreams never merge canonical state.
- Cross-Workstream reuse requires explicit reference, snapshot, or fork and does not transfer authority by default.
- Stale agent commands never silently rebase over newer human work.

## 12.4 Leases

Leases may apply to:

- whole object;
- whiteboard page/frame/region;
- spreadsheet workbook/sheet/range;
- chart spec or annotation layer;
- generated workspace component subtree;
- external publish/export operation.

Leases SHALL carry exact object identity, revision, actor, expiry, and Focusa Workstream context when bound.

---

# 13. DataView foundation

Flint and other visualization engines are not data-wrangling authorities. Cockpit SHALL implement a first-class versioned `DataView` between source objects and visualizations.

```ts
interface DataView {
  schema: "uiai.dataview.v1";
  object_ref: string;
  revision: number;
  source_bindings: Array<{
    object_ref: string;
    source_revision: number;
    selector?: string;
  }>;
  transform_recipe: DataTransformStep[];
  output_schema_ref: string;
  semantic_annotations: Record<string, unknown>;
  output_snapshot_ref: string;
  output_digest: string;
  row_count: number;
  freshness: "current" | "stale" | "blocked" | "failed";
  refresh_policy: "manual" | "on_source_change" | "scheduled";
  warnings: TypedWarning[];
  lineage_ref: string;
  focusa_binding?: FocusaWorkstreamBinding;
}
```

Supported transform classes include:

- select/project fields;
- filter;
- sort;
- group/aggregate;
- derive/calculation;
- join;
- pivot/unpivot;
- time bucketing;
- unit normalization;
- category normalization;
- sampling with explicit disclosure;
- validation and anomaly flags.

Agents SHALL inspect actual values and distributions, not only column names, before proposing a chart. Embedded totals, units, missing values, duplicate entities, cardinality, and source freshness must be evaluated and disclosed.

---

# 14. Spreadsheet work-object runtime

## 14.1 Adapter decision

The initial spreadsheet vertical slice SHALL use:

- Univer as the active grid, workbook, and formula interaction engine where suitable;
- Hucre as an import/export and round-trip codec adapter where suitable;
- UIAI-owned object identity, revisions, transactions, collaboration, receipts, Evidence, and Workstream binding.

No Univer or Hucre internal identifier becomes a permanent public UIAI object identity without an adapter mapping.

The implementation SHALL NOT depend on a commercial real-time collaboration service as canonical UIAI authority unless a separately approved licensing and authority decision is made.

## 14.2 Spreadsheet capabilities

Required semantic capabilities:

```text
spreadsheet.create
spreadsheet.import
spreadsheet.snapshot
spreadsheet.read
spreadsheet.query
spreadsheet.apply
spreadsheet.recalculate
spreadsheet.render
spreadsheet.export
spreadsheet.validate
spreadsheet.compare
spreadsheet.link_focusa
```

Agent operations SHALL use cells, ranges, tables, names, formulas, formatting, charts, and workbook structure—not raw DOM interaction.

## 14.3 Collaboration

Initial collaboration may use:

- workbook revisions;
- sheet/range leases;
- atomic transactions;
- optimistic non-overlapping edits;
- proposal mode for conflicts;
- undo/compensation;
- before/after render previews.

Arbitrary last-writer-wins over overlapping spreadsheet ranges is prohibited.

## 14.4 Import security

Imports SHALL classify and handle:

- formulas;
- external links and data connections;
- macros/executable content;
- hidden sheets/rows/columns;
- named ranges;
- embedded files;
- CSV formula injection;
- unsupported features and round-trip loss.

Executable or externally connected content is disabled or quarantined by default and must never inherit Focusa authority from the workbook binding.

---

# 15. Whiteboard and canvas runtime

## 15.1 Adapter decision

The initial whiteboard vertical slice SHALL use the tldraw SDK behind a UIAI-owned adapter.

The separate tldraw offline application is not an embeddable or forkable Cockpit dependency and SHALL NOT be treated as the product runtime.

Because Cockpit uses Svelte and tldraw uses React, the preferred implementation is a bounded React island/root mounted inside a Cockpit-owned DOM host. Cockpit SHALL NOT be rewritten around React.

Self-hosted synchronization may use `@tldraw/sync-core` or an approved equivalent inside a Node/TypeScript document worker, authenticated and scoped by UIAI. Collaboration room IDs SHALL derive from opaque UIAI object handles, not Focusa paths or human-readable project names.

## 15.2 Agent control order

Whiteboard agent control SHALL prefer:

1. semantic tldraw Editor operations;
2. structured store transactions;
3. `@tldraw/driver` for parity tests or gestures not represented semantically;
4. bounded visual/DOM automation only as an exceptional fallback.

Raw DOM automation is not the normal whiteboard API.

## 15.3 Whiteboard capabilities

```text
whiteboard.create
whiteboard.snapshot
whiteboard.query
whiteboard.apply
whiteboard.transform
whiteboard.group
whiteboard.connect
whiteboard.render
whiteboard.export
whiteboard.compare
whiteboard.validate
whiteboard.link_focusa
```

Structured snapshots SHALL expose:

- pages;
- shapes and bindings;
- text index;
- spatial clusters;
- selected objects;
- viewport;
- assets;
- recent operations;
- embedded object refs;
- Focusa-linked entity refs;
- render reference;
- exact revision.

Agents shall receive structured state plus bounded renders, not screenshot-only context.

## 15.4 Focusa-aware shapes

Custom projections may represent:

- Workpoint;
- Task;
- Decision;
- Assumption;
- Blocker;
- Evidence;
- Deliverable;
- Agent Assignment;
- Browser Research;
- Spreadsheet Range;
- File Artifact;
- Checkpoint.

These shapes store canonical refs and projection revisions only. They do not duplicate or own Focusa canonical state.

## 15.5 Executable content

Imported document scripts are disabled or quarantined by default.

Three behavior tiers apply:

1. safe declarative behaviors;
2. explicitly approved sandboxed scripts with restricted capabilities, resource limits, provenance, and no arbitrary filesystem/network access;
3. trusted signed application extensions installed and governed separately.

A document-contained script cannot grant itself capabilities or Focusa authority.

---

# 16. Flint semantic chart and visualization runtime

## 16.1 Adapter decision

The initial visualization vertical slice SHALL use the `flint-chart` library directly behind a UIAI-owned adapter.

The raw Flint MCP server remains useful for external interoperability and prototyping, but internal Cockpit execution SHALL route through UIAI object, revision, policy, receipt, and Evidence contracts rather than opening disconnected MCP chart views.

The adapter SHALL pin a reviewed Flint version and prevent Flint API churn from becoming a Cockpit public contract.

## 16.2 Canonical chart object

```ts
interface SemanticChartObject {
  schema: "uiai.semantic_chart.v1";
  object_ref: string;
  revision: number;
  dataview_ref: string;
  dataview_revision: number;
  flint_input: {
    semantic_types: Record<string, unknown>;
    chart_spec: Record<string, unknown>;
    field_display_names?: Record<string, string>;
    options?: Record<string, unknown>;
  };
  preferred_backend: "vegalite" | "echarts" | "plotly" | "chartjs" | "excel";
  derived_spec_refs: string[];
  render_refs: string[];
  interaction_state_ref?: string;
  annotation_layer_ref?: string;
  warnings: TypedWarning[];
  focusa_binding?: FocusaWorkstreamBinding;
}
```

The canonical editable object is the semantic chart and DataView binding—not a PNG, screenshot, or backend-specific configuration.

## 16.3 Artifact forms

Every chart distinguishes:

1. **Semantic chart object:** editable, portable source.
2. **Derived backend specification/render:** Vega-Lite, ECharts, Plotly, Chart.js, Excel, SVG, or PNG.
3. **Immutable Evidence snapshot:** exact semantic spec, DataView/source revisions, render, annotations, warnings, actor, Workstream/Workpoint binding, receipts, and digest.

## 16.4 Chart capabilities

```text
visualization.chart.create
visualization.chart.inspect
visualization.chart.validate
visualization.chart.patch
visualization.chart.bind_dataview
visualization.chart.backend.set
visualization.chart.filter.set
visualization.chart.selection.set
visualization.chart.annotation.add
visualization.chart.render
visualization.chart.export
visualization.chart.compare
visualization.chart.embed
visualization.chart.fork
visualization.chart.snapshot.capture
```

Agents SHALL edit Flint/DataView semantics first. Backend JSON may be modified only through an explicit backend-specific presentation layer after a valid semantic chart exists, and such modifications must be labeled nonportable.

## 16.5 Live collaboration

Humans and agents may:

- switch supported chart types;
- bind fields to channels;
- filter or select data;
- resize the visual slot;
- add annotations;
- request explanations or alternate views;
- compare chart revisions;
- embed the chart in a whiteboard, report, document, or dashboard;
- freeze a revision as Evidence.

Human selection and annotation become structured state that agents can consume without interpreting pixels.

## 16.6 File and network access

Flint SHALL receive inline bounded rows or a UIAI-brokered DataView snapshot. It SHALL NOT receive unrestricted filesystem paths or remote URLs from untrusted agent content.

Remote data access, if implemented, routes through approved UIAI connectors and DataView provenance rather than Flint directly.

---

# 17. Generated workspaces and A2UI compatibility

## 17.1 Purpose

Cockpit SHALL support real-time generated task-specific workspaces composed from approved components.

Examples:

- data investigation workspace;
- architecture comparison workspace;
- pricing decision dashboard;
- launch planning workspace;
- incident review workspace;
- research synthesis workspace.

## 17.2 Portability format

Generated workspace manifests SHALL use the existing approved A2UI direction and remain compatible with the Focusa Desktop A2UI 0.9.1 Lit renderer and Focusa custom elements.

UIAI may host the generated surface inside its Svelte shell through product-neutral adapters or custom elements, but it SHALL NOT invent a competing portability format or force Focusa Desktop to implement a second generated-UI renderer.

Version upgrades require coordinated conformance fixtures across UIAI Cockpit, Focusa Desktop, and Focusa.work where applicable.

## 17.3 Registered component grammar

Allowed component families include:

- Flint chart;
- table/data grid;
- spreadsheet view/range;
- whiteboard/frame;
- document/Markdown;
- metric/KPI;
- timeline;
- comparison matrix;
- evidence list;
- receipt viewer;
- research sources;
- browser artifact/capture;
- agent status;
- form and parameter controls;
- approval/review controls;
- report section;
- media/artifact preview.

Every interactive control SHALL bind to an existing registered capability ID.

## 17.4 Lifecycle

```text
ephemeral
  generated for the current task or question

pinned
  retained by the user

promoted
  becomes a named reusable workspace

templated
  reusable across compatible Workstreams under explicit creation

archived
  preserved as a snapshot or Evidence-linked historical surface
```

## 17.5 Prohibitions

Generated workspaces SHALL NOT:

- execute arbitrary JavaScript;
- create new capabilities;
- modify Capability Grants;
- bypass Scope, Workstream, authority, consent, or entitlement guards;
- issue arbitrary network or filesystem requests;
- hide consequential operations;
- own canonical Focusa cognition;
- treat prose or comments as execution authorization;
- mutate another Workstream through reused references;
- ship inaccessible or visually uninspectable controls.

---

# 18. Cross-surface references and embedding

Stable reference grammar SHALL support examples such as:

```text
@whiteboard:architecture#shape:focusa-daemon
@whiteboard:launch-plan#frame:phase-2
@spreadsheet:forecast#range:Forecast!B4:H42
@chart:revenue-growth#selection:enterprise-q3
@dataview:quarterly-revenue@11
@research:competitor-analysis
@browser-session:pricing-review
@focusa:workstream:<opaque>
@focusa:workpoint:<opaque>
```

The final grammar SHALL use opaque stable IDs and generated schemas.

Every embed declares one lifecycle mode:

- **live reference:** source remains authoritative and updates may flow;
- **frozen snapshot:** exact source revision is pinned;
- **fork:** a new object is created with explicit lineage.

Embed contracts SHALL define:

- source object and revision;
- permission inheritance;
- edit routing;
- refresh policy;
- staleness presentation;
- missing-source behavior;
- cycle prevention;
- selection ownership;
- export behavior;
- Evidence capture behavior;
- cross-Workstream policy.

Flattened screenshots are render outputs, not the normal collaboration model.

Different-Workstream live mutation is prohibited. Cross-Workstream use requires explicit reference, frozen snapshot, or fork with `authority_transfers=false` by default.

---

# 19. Focusa and Focusa Desktop bidirectional work loop

## 19.1 Required flow

```text
Focusa Workstream and Workpoint intent
→ verified UIAI Cockpit Attachment
→ Studio/other Cockpit work-object creation or inspection
→ human and agent collaboration
→ UIAI revisions, renders, receipts, findings, and annotations
→ Evidence candidate and continuation packet
→ Focusa reducer accepts/rejects/associates canonical meaning
→ next Workpoint or Work Loop action
→ optional presentation in Cockpit or Focusa Desktop
```

## 19.2 Four transfer modes

### Live reference

UIAI Engine remains authoritative for the mutable work object. Focusa Desktop or Cockpit presents it by stable ref and routes commands back to UIAI.

### Immutable snapshot

An exact object revision, dependencies, render, digest, provenance, and receipts are captured as a candidate for Focusa Evidence.

### Fork

A new UIAI object is created with explicit source lineage, source Workstream, source revision, reason, and authority-transfer posture.

### Continuation packet

A bounded packet carries enough semantic context for another Focusa-powered agent or Desktop Work Surface to resume.

## 19.3 Continuation packet

```ts
interface VisualContinuationPacket {
  schema: "uiai.visual_continuation_packet.v1";
  source_object_ref: string;
  source_revision: number;
  focusa_binding: FocusaWorkstreamBinding;
  active_subobject_ref?: string;
  active_selection?: unknown;
  viewport_or_range?: unknown;
  human_annotations: AnnotationRef[];
  accepted_findings: FindingRef[];
  unresolved_questions: QuestionRef[];
  evidence_refs: string[];
  receipt_refs: string[];
  dependency_revisions: Array<{ ref: string; revision: number }>;
  suggested_next_actions: SuggestedAction[];
  created_at: string;
}
```

A continuation packet is advisory context. It does not become canonical Focus State or a Workpoint merely because it exists.

## 19.4 Evidence proposal boundary

UIAI may propose:

- attach object as working material;
- set as active deliverable;
- record a decision artifact;
- capture revision as Evidence;
- checkpoint a Workpoint;
- create a follow-up proposal.

Focusa accepts, rejects, or associates these through its exact Workstream reducer and authority contracts. UIAI must not silently promote output into canonical Focusa truth.

## 19.5 No cross-Workstream contamination

A handoff SHALL include exact Scope, Workstream, Workpoint where applicable, event head/revisions, and Attachment provenance.

A missing or stale binding fails closed. UIAI and Focusa Desktop may not use the daemon's last Workstream, another window's active Workstream, path similarity, or ContinuityId-only inference.

---

# 20. Mutable work versus immutable Evidence

A live object continues changing:

```text
object revision 37
  ├── revision 38
  ├── revision 39
  └── immutable Evidence snapshot of revision 37
```

Evidence capture SHALL include, as applicable:

- object ref and revision;
- exact Focusa Scope/Workstream/Workpoint binding;
- source object and DataView revisions;
- transform recipe and output digest;
- semantic chart or document specification;
- whiteboard page/frame/shape refs;
- spreadsheet sheet/range/formula refs;
- generated workspace component graph;
- render/export refs;
- actor and participant provenance;
- human annotations and decisions;
- operation receipts;
- warnings and limitations;
- freshness/staleness state;
- content hashes;
- redaction and audience posture.

Evidence snapshots remain immutable even if the source object is later edited, detached, archived, or deleted.

---

# 21. Security and trust boundaries

## 21.1 Untrusted visual content

The following are untrusted content unless separately verified:

- spreadsheet cells and formulas;
- shape text and embedded links;
- chart labels and annotations;
- imported documents and comments;
- browser/research captures;
- SVG/HTML/media metadata;
- generated component content;
- external agent suggestions;
- imported scripts and macros.

Untrusted content may inform reasoning but cannot modify:

- Focusa binding;
- Capability Grants;
- approval policy;
- tool schemas;
- permission or entitlement state;
- command routing;
- Workstream authority.

## 21.2 Common controls

The visual runtime SHALL implement:

- content sanitation;
- CSP and sandboxing;
- origin/network allowlists;
- path and file-broker restrictions;
- no SSRF through chart/data adapters;
- formula/macro/external-link quarantine;
- CSV injection handling;
- SVG/HTML event and script stripping;
- prompt-injection provenance boundaries;
- bounded data, shape, component, and render sizes;
- cancellation and timeouts;
- worker isolation;
- explicit destructive and external-publish approvals;
- secrets redaction;
- audit and receipts.

## 21.3 Executable behavior tiers

- **Declarative:** safe default.
- **Sandboxed:** explicit restricted capabilities and limits.
- **Trusted extension:** signed, installed at application level, separately governed.

Document-contained executable code never inherits the containing object's Focusa authority.

---

# 22. Accessibility contract

## 22.1 Charts

Charts SHALL provide:

- semantic title and description;
- source and freshness disclosure;
- tabular fallback;
- keyboard data navigation where interactive;
- non-color distinctions;
- accessible selected/filter state;
- warnings for truncation, sampling, or aggregation;
- reflow/zoom behavior;
- export alt text and data attachment where policy permits.

## 22.2 Whiteboards

Whiteboards SHALL provide:

- keyboard-accessible creation, selection, move, edit, group, connect, and delete;
- an inspectable structured outline/list of pages, frames, shapes, text, and bindings;
- meaningful names for custom Focusa-linked shapes;
- accessible comments and agent proposals;
- reduced-motion and high-contrast behavior;
- nonvisual access to selected object properties.

## 22.3 Spreadsheets

Spreadsheets SHALL provide complete grid semantics, keyboard navigation/editing, formula and error announcements, selection/range status, accessible sheets/tables, and nonvisual export/inspection paths.

## 22.4 Generated workspaces

Generated workspaces SHALL use registered accessible components, deterministic focus order, approved labels, reflow, keyboard operation, and bounded live-region output.

Models may not invent arbitrary ARIA roles or hide required actions behind hover, drag, animation, or color alone.

---

# 23. Headless, offline, remote, and restart behavior

## 23.1 Headless operation

Domain commands SHALL run with Cockpit closed where the capability declares `headless_posture="works"`.

Examples include:

- create or update a spreadsheet;
- apply whiteboard semantic operations through a worker;
- refresh a DataView;
- create/validate/render a Flint chart;
- compose a generated workspace manifest;
- export artifacts;
- capture an Evidence candidate.

Presentation commands may launch or focus Cockpit through a registered single-instance presenter.

## 23.2 Reattachment

When Cockpit opens or reconnects, it SHALL attach to:

- current object revisions;
- operation/event cursor;
- collaboration state;
- jobs/renders;
- exact client Attachment;
- pending proposals/approvals;
- recovery states.

It SHALL NOT reconstruct authority from the last visible UI alone.

## 23.3 Remote and worktree identity

Remote hosts and worktrees participate in explicit `ScopeRef` and workspace bindings. Identical paths on different hosts, and different worktrees of one project, SHALL remain distinguishable.

## 23.4 Capability-level posture

Every capability declares:

- local/offline operation;
- network dependency;
- renderer dependency;
- worker/backend dependency;
- Focusa dependency;
- remote support;
- degraded behavior;
- recovery action.

Planned or missing backends must be presented truthfully.

---

# 24. Dependency, licensing, and adapter governance

Every external dependency follows:

```text
Adopt → Wrap → Configure → Extend → Custom
```

Before production use:

- pin version;
- review license and distribution obligations;
- add notices and SBOM entry;
- document security posture;
- define adapter boundary;
- add conformance fixtures;
- define fallback/degraded behavior;
- add upgrade and rollback procedure;
- prevent third-party IDs from leaking into stable public contracts.

Specific requirements:

- tldraw offline is not shipped as the Cockpit runtime.
- tldraw SDK licensing must be reviewed and satisfied for production distribution.
- Flint is consumed through a pinned UIAI adapter; its MCP remains optional interoperability.
- Univer commercial collaboration features do not silently become canonical UIAI authority.
- Hucre use requires codec quality, loss, license, and security review.
- Focusa Desktop retains its approved A2UI 0.9.1 Lit renderer until a coordinated upgrade is accepted.
- No separate generated-UI portability format is introduced casually.

---

# 25. Integration with existing numbered documents

## 25.1 `UIAI-COCKPIT-000`

The master remains authoritative for product shape, workspaces, Focusa/UIAI boundaries, Workpoints, Evidence, and progressive disclosure. This amendment adds universal control and multimodal object runtimes.

## 25.2 `UIAI-COCKPIT-001`

Review Reports remain Evidence-centered presentation/review objects. Report charts, tables, diagrams, whiteboard frames, spreadsheet ranges, and generated views SHALL reference the work-object and DataView contracts here rather than creating report-local truth.

Live reports show source freshness. Frozen reports pin exact revisions and digests.

## 25.3 `UIAI-COCKPIT-002`

Agent-first browser objects, observations, actions, provenance, and verification may be embedded in Studio/visual workspaces through stable refs. Browser authority remains UIAI-owned and does not transfer to Focusa Desktop or generated UI.

## 25.4 `UIAI-COCKPIT-003`

This amendment modifies:

- Studio secondary navigation and object taxonomy;
- Documents spreadsheet placement;
- `WorkspaceManifest` capability metadata;
- route-skeleton work for Studio and Documents;
- Search/Command integration;
- Universal Inspector sections;
- task ordering for `T003-02`, `T003-11`, and `T003-14`.

Implementation of those `003` tasks SHALL incorporate this amendment rather than freezing the prior verb-only Studio IA.

---

# 26. Target implementation boundaries

The exact package layout may adapt to repository reality, but these ownership boundaries are normative:

```text
UIAI authority/control layer
  capability registry
  command/query router
  context/Attachment resolver
  policy/approval/entitlement
  revisions/leases/idempotency
  receipts/events/recovery
  semantic Cockpit state

Shared work-object runtime
  object registry and storage
  operation log
  snapshots and exports
  embeds and lineage
  collaboration/presence
  Evidence proposal adapter

Visual/document workers
  spreadsheet adapter
  whiteboard/tldraw adapter
  DataView transforms
  Flint/chart render adapter
  generated workspace validation/rendering

Cockpit application
  control-plane client
  workspace manifests
  object library/tabs
  editors and React/Lit islands
  Universal Inspector
  presence/proposals/activity

Interop
  CLI
  REST/OpenAPI
  MCP/Pi tools
  Focusa binding and Evidence proposal
  Focusa Desktop presenter/handoff
```

No Svelte component calls raw production routes directly. No adapter bypasses the command router. No client maintains a separate command registry.

---

# 27. Machine-readable companion

A companion ledger SHALL be created as:

```text
UIAI-COCKPIT-004-C01
```

It SHALL enumerate:

- stable requirement IDs;
- schemas;
- object kinds;
- command/query/presentation capability IDs;
- events;
- context requirements;
- side-effect classes;
- approval and receipt policies;
- entitlement and permission mappings;
- adapter/version inventory;
- task graph and dependencies;
- security/accessibility/performance gates;
- adversarial acceptance fixtures;
- implementation ownership;
- release Evidence requirements.

Disagreement between this normative amendment and the companion blocks implementation until corrected explicitly.

---

# 28. Implementation task graph

## T004-00 — Register amendment and freeze terminology

**Outputs:** this document; document-register entry; coordination issue; source map.  
**Work:** prohibit new canonical `Thread`, project-root-only Focusa binding, ContinuityId-as-Workstream identity, and global/current Workstream authority.  
**Acceptance:** register resolves `UIAI-COCKPIT-004`; CI/lint plan exists; no conflicting number.

## T004-01 — Focusa Spec 158 alignment contract

**Depends on:** T004-00  
**Work:** define generated `ScopeRef`, `WorkstreamId`, `ContinuityId`, `AttachmentKey`, Workpoint, event-head, and authority envelopes; define discrepancy handling until Spec 158 publishes.  
**Acceptance:** no handwritten permanent duplicate; unbound/ambiguous requests fail closed.

## T004-02 — Universal capability registry and envelopes

**Depends on:** T004-01  
**Work:** command/query/result/event schemas; capability descriptors; side effects; headless/offline posture; approvals; receipts; recovery.  
**Acceptance:** registry validates; every implemented workspace action maps to a capability or explicit presentation-only state.

## T004-03 — Cockpit client Attachment and semantic state

**Depends on:** T004-01, T004-02  
**Work:** per-window/agent Attachment; explicit switching; semantic state endpoint/stream; no global current Workstream.  
**Acceptance:** two windows remain independently attached to different Workstreams.

## T004-04 — CLI/API/MCP/Pi/GUI parity generation

**Depends on:** T004-02, T004-03  
**Work:** CLI grammar; OpenAPI/schema generation; MCP/Pi tools; command-palette bindings; parity fixtures.  
**Acceptance:** one fixture invokes the same capability through every supported surface with equivalent guard/result behavior.

## T004-05 — Shared work-object, revision, transaction, lease, and embed contracts

**Depends on:** T004-01, T004-02  
**Work:** `CockpitWorkObject`, `StudioWorkspace`, transaction envelope, operation log, snapshots, leases, participants, live/frozen/fork embeds, deletion/retention.  
**Acceptance:** adapters cannot invent parallel identity or history models.

## T004-06 — Studio/Documents IA reconciliation

**Depends on:** T004-05; coordinated with `T003-02`, `T003-11`, `T003-14`  
**Work:** new Studio object taxonomy; Documents spreadsheet placement; intents/recipes; routes; Inspector; Search.  
**Acceptance:** one Studio workspace; no top-level Charts/Whiteboards/Reports; two-level navigation limit remains.

## T004-07 — DataView runtime

**Depends on:** T004-05  
**Work:** source bindings, transform recipes, semantic annotations, freshness, lineage, caching, resource limits, agent inspection.  
**Acceptance:** chart can be reproduced from exact source revisions and transform recipe.

## T004-08 — Spreadsheet vertical slice

**Depends on:** T004-05, T004-07  
**Work:** Univer/Hucre adapters; import; edit; recalc; render; export; transactions; leases; agent tools; security; Evidence.  
**Acceptance:** headless and GUI edits share receipts/revisions; round-trip loss is disclosed.

## T004-09 — Whiteboard vertical slice

**Depends on:** T004-05  
**Work:** tldraw React island; sync worker; semantic ops; structured snapshots; custom ref shapes; scripts quarantine; agent tools; Evidence.  
**Acceptance:** agent can inspect/edit without DOM automation; human sees live proposal/presence.

## T004-10 — Flint chart vertical slice

**Depends on:** T004-05, T004-07  
**Work:** pinned Flint adapter; semantic chart object; live Svelte-hosted view; validation; multi-backend render; annotations; exports; agent tools; Evidence.  
**Acceptance:** spreadsheet/DataView change refreshes chart with explicit revision/freshness; static render remains reproducible.

## T004-11 — Generated A2UI-compatible workspace vertical slice

**Depends on:** T004-05, T004-07, T004-10  
**Work:** registered component grammar; lifecycle; validator; capability bindings; Focusa Desktop conformance; no arbitrary code.  
**Acceptance:** agent generates a bounded decision dashboard usable in Cockpit and presentable in Focusa Desktop.

## T004-12 — Cross-surface composition

**Depends on:** T004-08 through T004-11  
**Work:** chart in whiteboard; spreadsheet range in chart/dashboard; browser/research refs; report integration; live/frozen/fork semantics.  
**Acceptance:** dependencies, permissions, staleness, cycles, missing source, and exports are tested.

## T004-13 — Focusa/Focusa Desktop handoff

**Depends on:** T004-01, T004-03, T004-05, T004-12  
**Work:** live-reference presenter; Evidence proposal; explicit fork; continuation packet; Workpoint/checkpoint proposal; deep links/opaque refs; receipts.  
**Acceptance:** exact Workstream survives Cockpit/Desktop switching, restart, and multiple simultaneous Workstreams.

## T004-14 — Security, accessibility, dependency, and performance gates

**Depends on:** adapter vertical slices  
**Work:** trust boundaries; scripts/macros; CSP; file/network broker; resource caps; a11y; licenses; SBOM; conformance; reduced motion/high contrast/zoom.  
**Acceptance:** CI-blocking normal and adversarial fixtures pass.

## T004-15 — Headless/offline/remote/restart proof

**Depends on:** T004-04 through T004-13  
**Work:** closed-Cockpit execution; event reattachment; worker outages; renderer outages; remote hosts/worktrees; stale handles; rollback.  
**Acceptance:** objects restore independently; no prior Workstream fallback.

## T004-16 — Full adversarial acceptance and rollout

**Depends on:** all tasks  
**Work:** multi-Workstream, multi-session, multi-agent, conflict, contamination, Evidence, migration, compatibility, performance, release receipts; documentation/help/screenshots; remove temporary compatibility paths.  
**Acceptance:** all Section 30 gates pass and one production control/object architecture remains.

---

# 29. Dependency summary

```text
T004-00
  → T004-01
      → T004-02
          → T004-03
          → T004-04
          → T004-05
              → T004-06
              → T004-07
                  → T004-08
                  → T004-10
              → T004-09
              → T004-11

T004-08 + T004-09 + T004-10 + T004-11
  → T004-12

T004-01 + T004-03 + T004-05 + T004-12
  → T004-13

adapter slices + interop
  → T004-14
  → T004-15
  → T004-16
```

Partitioning and exact Workstream binding precede broad cross-product collaboration. Adapter demos do not justify bypassing the common runtime.

---

# 30. Mandatory adversarial acceptance matrix

Closure requires live semantic and visual proof, not only source inspection.

## 30.1 Focusa foundation alignment

- [ ] Two Focusa Workstreams inside one Scope are attached to two Cockpit windows without cross-contamination.
- [ ] Switching one window does not switch another.
- [ ] No daemon/Cockpit global `active`, `current`, `latest`, `last`, or remembered Workstream authorizes a read or write.
- [ ] `ContinuityId`, Session, path, cwd, or UI focus cannot independently identify Workstream authority.
- [ ] Unbound/ambiguous requests return zero foreign payload and typed recovery.
- [ ] `Thread` exists only in historical/deprecation documentation.
- [ ] Exact Workstream survives restart, reconnect, model/provider change, remote host change, and Work Surface handoff.

## 30.2 Universal control

- [ ] Every implemented GUI durable action has a capability ID.
- [ ] Every capability has typed input/output, side-effect, context, approval, receipt, and recovery metadata.
- [ ] CLI, API, MCP/Pi, command palette, and GUI use the same fixture and equivalent guards.
- [ ] Domain commands work with Cockpit closed where declared.
- [ ] Presentation commands target explicit clients/windows.
- [ ] No Svelte-local store becomes command authority.

## 30.3 Work-object and collaboration

- [ ] Stable object identity survives renderer/adapter restart.
- [ ] Same-object non-overlapping edits commit safely.
- [ ] Overlapping edits create explicit conflict/proposal state.
- [ ] Human takeover and return control work.
- [ ] Agent presence is distinct and semantically inspectable.
- [ ] Stale agent commands cannot overwrite newer human work silently.
- [ ] Different-Workstream objects never merge implicitly.
- [ ] Cross-Workstream reuse records explicit provenance and no authority transfer.

## 30.4 Spreadsheet

- [ ] Import/edit/recalc/render/export round trip is versioned and receipted.
- [ ] Unsupported or lossy features are disclosed.
- [ ] Formulas, macros, external connections, and CSV injection are handled safely.
- [ ] Range leases and conflict behavior are proven.
- [ ] Agent edits use semantic range operations, not DOM automation.

## 30.5 Whiteboard

- [ ] Structured snapshots and semantic operations are complete enough for agent use.
- [ ] A human and agent collaborate live on separate regions.
- [ ] Embedded scripts are disabled/quarantined by default.
- [ ] Focusa-linked custom shapes store refs/projections, not duplicate canonical state.
- [ ] Driver/DOM control is fallback, not primary.
- [ ] Accessibility outline and keyboard operations pass.

## 30.6 DataView and chart

- [ ] DataView reproduces exact output from source revisions and transforms.
- [ ] Agent inspects values/cardinality/units before charting.
- [ ] Flint semantic spec remains the editable portable source.
- [ ] Vega-Lite/ECharts/Plotly/Chart.js/Excel outputs are derived and labeled by support.
- [ ] Live chart updates disclose DataView/source revision and staleness.
- [ ] Chart warnings, truncation, aggregation, and sampling are visible.
- [ ] Chart has accessible summary and table fallback.
- [ ] Flint receives no unrestricted path or remote URL authority.

## 30.7 Generated workspace

- [ ] Workspace is composed only from registered A2UI-compatible components.
- [ ] Every control binds to an existing capability.
- [ ] Arbitrary JS/network/filesystem access is blocked.
- [ ] Ephemeral/pinned/promoted/templated/archived lifecycle works.
- [ ] Generated workspace is usable in Cockpit and presentable through Focusa Desktop without a competing renderer.
- [ ] Focus order, keyboard access, reflow, and labels pass.

## 30.8 Cross-surface and Focusa handoff

- [ ] Spreadsheet range drives a DataView and live chart.
- [ ] Chart embeds live in a whiteboard and freezes into a report/Evidence snapshot.
- [ ] Browser/research artifacts embed by stable reference.
- [ ] Live/frozen/fork behavior is explicit and tested.
- [ ] Focusa Desktop presents a UIAI object and routes edits back without owning it.
- [ ] Continuation packet resumes exact object/selection/questions in the correct Workstream.
- [ ] UIAI Evidence proposal is accepted/rejected by Focusa rather than silently canonized.
- [ ] Mutable later revisions cannot change prior Evidence snapshots.

## 30.9 Security, performance, and recovery

- [ ] Untrusted visual content cannot alter authority or tool schemas.
- [ ] Resource limits prevent one object/Workstream from blocking unrelated work.
- [ ] Cancellation, rollback, and compensating actions preserve revision truth.
- [ ] Worker/renderer outage produces typed degraded/recovery state.
- [ ] Restart and event replay restore independent objects and Attachments.
- [ ] Dependency licenses, notices, SBOM, pins, and conformance fixtures pass.

---

# 31. Closure prohibition

Do not claim this amendment complete because:

- one Studio page renders;
- one Flint chart is visible;
- one spreadsheet can be edited;
- one tldraw canvas opens;
- one MCP tool works;
- a Focusa project path is stored;
- a `workstream_id` field exists but global fallback remains;
- one agent can click the GUI visually;
- screenshots show human-agent cursors;
- adapter-native collaboration works without UIAI receipts and revisions;
- a generated HTML page is displayed;
- one Evidence export exists;
- static grep or unit tests pass;
- global fields are labeled legacy while remaining authoritative;
- different Workstreams are separated only by filters over mixed global state;
- Cockpit and Focusa Desktop each implement separate generated-UI formats.

Closure requires:

1. exact Workstream-rooted contracts and no singleton/global cognitive authority;
2. one universal capability/control registry;
3. complete semantic state and per-client Attachment behavior;
4. GUI/CLI/API/MCP/Pi parity;
5. shared work-object/revision/transaction/collaboration contracts;
6. complete spreadsheet, whiteboard, DataView/chart, and generated-workspace vertical slices;
7. Focusa Desktop live reference, snapshot, fork, and continuation;
8. immutable Evidence and receipt lineage;
9. security, accessibility, dependency, performance, offline, restart, and remote proof;
10. all Section 30 adversarial gates with stable Evidence receipts.

---

# 32. Prohibited patterns

Implementation MUST NOT introduce:

- `Thread` as a new canonical owner;
- `project_root + continuity_id` as permanent Workstream identity;
- daemon-global or Cockpit-global cognitive state;
- global current/active/latest/last Workstream mutation authority;
- project-only Focusa binding for canonical handoff;
- UI focus as backend authority;
- separate command registries per workspace or client;
- direct raw production API calls from Svelte components;
- external libraries as canonical identity/authority stores;
- screenshot-only agent control;
- raw DOM automation as normal spreadsheet/whiteboard/chart API;
- arbitrary generated JavaScript or remote embeds;
- unrestricted Flint local-file or URL access;
- tldraw document scripts enabled by default;
- hidden spreadsheet macros/external connections;
- cross-Workstream implicit merging;
- Evidence snapshots that change with the live object;
- silent UIAI promotion into Focusa canonical truth;
- a second browser authority in Focusa Desktop;
- a competing generated-UI portability format;
- top-level Charts, Whiteboards, Dashboards, or Reports products that fragment the Cockpit;
- inaccessible visual-only operations or Evidence.

---

# 33. Unified acceptance statement

This amendment is complete only when UIAI Engine Cockpit functions as one governed visual and operational environment in which:

- Focusa provides exact Workstream-rooted purpose, continuity, authority, Workpoints, and canonical meaning;
- UIAI Engine provides actuation, structured visual work objects, collaboration, rendering, and proof;
- Cockpit provides the shared visual field where humans and agents inspect, create, steer, review, and approve;
- Focusa Desktop can present and continue UIAI-owned visual work without duplicating authority;
- every important operation is semantically controllable through GUI, CLI, API, and agent tools;
- every important output remains attached to the exact Workstream and Workpoint that caused it to exist;
- no global cognitive singleton, Thread owner, path fallback, or UI-current state can contaminate another Workstream.

---

# 34. Final architectural principle

> **UIAI Engine Cockpit is the shared visual execution environment of Focusa-powered work, not a collection of disconnected tabs. Every workspace and visual object inherits one Workstream-scoped control plane, one object and revision model, one governed collaboration model, and one proof-and-continuation path. Focusa remembers why the work exists and canonizes meaning; UIAI Engine acts, renders, and produces inspectable artifacts; Cockpit makes that work visible and steerable; Focusa Desktop carries the exact same Workstream forward without reintroducing a singleton.**
