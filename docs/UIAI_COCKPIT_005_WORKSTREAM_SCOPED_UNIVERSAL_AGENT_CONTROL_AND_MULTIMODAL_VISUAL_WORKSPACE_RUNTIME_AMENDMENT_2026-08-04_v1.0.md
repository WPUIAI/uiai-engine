# UIAI Cockpit Workstream-Scoped Universal Agent Control and Multimodal Visual Workspace Runtime Amendment

**Document number:** `UIAI-COCKPIT-005`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding numbered decisions/amendments:** `UIAI-COCKPIT-001`, `UIAI-COCKPIT-002`, `UIAI-COCKPIT-003`, `UIAI-COCKPIT-004`  
**Status:** Proposed normative architecture and implementation amendment  
**Version:** 1.0  
**Date:** 2026-08-04  
**Repository:** `WPUIAI/uiai-engine`  
**Coordination issue:** `WPUIAI/uiai-engine#7`  
**Cross-repository foundation dependency:** `Startempire-Wire/focusa#125`, planned Focusa Spec 158  
**Primary implementation homes:** UIAI Engine authority/API/CLI layers, `apps/cockpit/`, work-surface/control contracts, document and visual workers, tool discovery, MCP/Pi exposure, artifact and Evidence integration  
**Scope:** universal Cockpit agent control; GUI/CLI/API/MCP/Pi parity; exact Focusa Workstream binding; semantic Cockpit state; multimodal Studio work objects; spreadsheets; whiteboards; semantic charts; DataViews; dashboards and generated workspaces; real-time human-agent collaboration; Focusa Desktop handoff; Evidence; security; accessibility; headless/offline behavior; dependency governance; migration; testing; release proof

---

# 0. Decision

UIAI Engine Cockpit SHALL become a fully semantic, fully agent-addressable work environment.

Every meaningful Cockpit operation SHALL be available through one governed control plane shared by:

- the Svelte/Tauri Cockpit GUI;
- the UIAI CLI;
- typed local and remote APIs;
- MCP and Pi tools;
- the Focusa-powered agent;
- authorized UIAI and external agents;
- the command palette and approved automation surfaces.

No meaningful durable Cockpit capability may exist only as a graphical click path.

Cockpit SHALL also gain one shared multimodal visual-workspace runtime supporting:

- spreadsheet work objects;
- whiteboard/canvas work objects;
- semantic chart and visualization work objects;
- first-class versioned DataViews;
- dashboards;
- bounded generated workspaces and interfaces;
- documents, reports, browser/research artifacts, media, and Evidence as composable referenced objects;
- real-time collaboration between humans and authorized agents;
- bidirectional handoff between UIAI Engine Cockpit, the Focusa-powered agent, and Focusa Desktop.

This amendment builds on `UIAI-COCKPIT-004`. It consumes that document's exact Workstream-scoped Work Surfaces, tabs, panes, windows, browser-target bindings, command targeting, restoration, and per-surface Attachment semantics. It does not create a competing tab/window/surface model.

The implementation SHALL NOT create separate control, identity, revision, collaboration, or Evidence systems for spreadsheets, tldraw, Flint, generated UI, Documents, Studio, or individual Cockpit workspaces. Every adapter inherits the common contracts defined here.

---

# 1. Focusa foundation correction applied to this amendment

This specification is filtered through the foundation decision recorded in `Startempire-Wire/focusa#125` and the planned Focusa Spec 158:

> Focusa is one daemon serving many isolated Workstreams. A Workstream is the durable cognitive unit. No canonical cognitive object exists outside an exact verified Project/Scope plus Workstream. Sessions attach to Workstreams. Continuity IDs, paths, cwd, cached packets, UI focus, and daemon-global active values do not define Workstream authority.

## 1.1 Canonical terminology

The canonical durable Focusa cognitive unit is **Workstream**.

`Thread` is historical design lineage only. It SHALL NOT appear as a new canonical owner, identifier, storage partition, runtime key, Cockpit object kind, API authority field, or generated UI concept.

The required identity separation is:

```text
ProjectRootKey / verified project-scope reference
  identifies the verified project and workspace topology

WorkstreamId
  durable identity of one Focusa cognitive workspace

ContinuityId
  continuation lineage or generation inside that Workstream

SessionId / InstanceId
  temporal harness or agent execution identities

AttachmentId / AttachmentKey
  binds a client, Session, Instance, workspace binding, and Continuity generation
  to an exact Workstream

WorkpointId
  durable continuation/checkpoint identity owned by the Workstream
```

The concrete identifiers and envelopes SHALL be generated from the accepted Focusa contract bundle. TypeScript examples in this amendment are consumer mappings, not permission to maintain permanent duplicate DTOs.

## 1.2 Earlier binding shorthand is superseded

Earlier discussion and prototypes that used only:

```text
project_root + continuity_id
```

as a Focusa binding identity are corrected.

A path may participate in verified project/workspace topology, but it cannot independently identify Workstream authority. `ContinuityId` cannot substitute for `WorkstreamId`. Session, selected tab, selected sidebar item, open window, current directory, and last-active UI state cannot substitute for a verified Attachment.

## 1.3 No cognitive singleton

This amendment prohibits canonical authority derived from:

- process-global `current_workstream`;
- daemon-global `active_project`;
- Cockpit-server `last_workstream` fallback;
- one Focusa binding shared by every Cockpit window;
- one global Studio space acting as canonical project state;
- latest-verified, nearest-path, remembered, or similarly named Workstream fallback;
- `ContinuityId`-only or Session-only resolution;
- UI focus, tab order, pane membership, window placement, or sidebar selection.

Presentation state may record what a specific client, Work Surface, tab, pane, or window is showing. It is noncanonical and may not authorize a command by itself.

## 1.4 Multiplexing invariant

One Focusa daemon and one UIAI Engine may concurrently serve:

- multiple verified projects/scopes;
- multiple Workstreams in one project;
- multiple Sessions attached to one Workstream;
- multiple Cockpit windows attached to different Workstreams;
- multiple humans and agents collaborating on one UIAI work object;
- multiple independent UIAI work objects attached to one Workstream.

A switch in one Attachment or Work Surface cannot replace daemon-global cognition or silently switch another client.

## 1.5 Fail closed

A Focusa-linked operation with absent, stale, conflicting, or ambiguous project/Workstream binding SHALL:

- fail before mutation;
- return zero foreign Workstream content;
- provide typed denial and recovery;
- expose bounded candidates only where policy permits;
- require explicit attachment, verification, or repair;
- never fall back to prior, current, latest, nearest, or path-similar work.

---

# 2. Authority and source precedence

Implementation SHALL apply sources in this order:

1. The accepted Focusa Workstream-rooted runtime specification and generated contract bundle.
2. `UIAI-COCKPIT-000` unified master specification.
3. `UIAI-COCKPIT-001` Interactive Review Reports integration decision.
4. `UIAI-COCKPIT-002` Agent-First Browser amendment and companion ledger.
5. `UIAI-COCKPIT-003` Sidebar Navigation, IA, and DnD amendment.
6. `UIAI-COCKPIT-004` Scoped Work Surfaces, Tabs, Panes, Windows, and Motion amendment.
7. This `UIAI-COCKPIT-005` amendment.
8. Current entitlement, protected-worker, API parity, security, artifact, Evidence, and interoperability specifications.
9. Current repository contracts, route mounts, manifests, schemas, tests, and implementation code.

Until Focusa Spec 158 is published, this amendment consumes the invariants in `Startempire-Wire/focusa#125`. A material discrepancy with the accepted Spec 158 blocks cross-product implementation until corrected explicitly. UIAI-local visual work may proceed only where it does not fabricate Focusa identity, authority, Workpoints, Evidence meaning, or canonical state.

---

# 3. Product and authority boundaries

## 3.1 Focusa owns canonical cognition

Focusa owns:

- verified Project/Scope identity and workspace topology;
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

## 3.2 UIAI Engine owns visual execution objects

UIAI Engine owns:

- browser and OS actuation under its authority contracts;
- UIAI work-object identity and storage;
- spreadsheet, whiteboard, chart, DataView, generated workspace, media, report-artifact, and execution-object production;
- visual-object revisions and collaboration state;
- rendering, import/export, transforms, validation, diagnostics, and artifact production;
- UIAI operation receipts;
- live object access and object-level permissions;
- proposing UIAI artifacts and immutable snapshots to Focusa.

## 3.3 Cockpit owns presentation and steering

Cockpit owns:

- workspace navigation;
- Work Surfaces, tabs, panes, windows, and object presentation defined by `UIAI-COCKPIT-004`;
- object selection and visual editing;
- participant and agent presence;
- proposals, previews, approvals, and review;
- semantic application-state presentation;
- routing intent through registered commands.

Cockpit does not own Focusa canonical cognition, Workpoint settlement, action authority, or durable browser memory merely because it renders them.

## 3.4 Focusa Desktop is a separate rich host

Focusa Desktop remains a separate Focusa product and distribution. It may present UIAI-owned visual Work Surfaces through shared contracts, but SHALL NOT:

- become a second UIAI browser authority;
- clone mutable UIAI object state without an explicit snapshot or fork;
- treat screenshots as control protocol;
- bypass UIAI command, permission, revision, or receipt contracts;
- infer Workstream binding;
- introduce a competing generated-UI portability model.

## 3.5 External engines remain adapters

Flint, tldraw, Univer, Hucre, Vega-Lite, ECharts, Plotly, Chart.js, A2UI renderers, and other dependencies are engines or codecs. They SHALL NOT define UIAI canonical object identity, Focusa binding, authority, collaboration policy, Evidence semantics, or stable public commands.

---

# 4. Governing architecture

```text
Focusa-powered agent / human operator / authorized agent
                              │
                    CLI / MCP / API / GUI
                              │
                              ▼
              Cockpit Universal Control Plane
       ┌─────────────────────────────────────────┐
       │ capability and workspace registry       │
       │ exact Workstream and Attachment context │
       │ queries, commands, events, receipts      │
       │ revisions, leases, proposals, approvals  │
       │ policy, entitlement, recovery            │
       │ semantic state and presentation routing  │
       └─────────────────────┬───────────────────┘
                             ▼
              Work Surfaces from Cockpit 004
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

Studio is the primary composition and creative collaboration workspace. Universal semantic control applies to every Cockpit workspace and subsection.

---

# 5. Universal Cockpit control plane

## 5.1 Parity invariant

> There shall be no meaningful Cockpit capability available only through the graphical interface.

The GUI, CLI, API, MCP/Pi tools, command palette, and authorized agents SHALL invoke the same semantic operations through the same guard, revision, receipt, event, and recovery contracts.

## 5.2 Domain versus presentation commands

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

Domain commands SHALL work without an open Cockpit window where the declared capability posture permits.

### Presentation commands

Presentation commands control a specific Cockpit Work Surface, tab, pane, or window under `UIAI-COCKPIT-004`.

Examples:

```text
cockpit.workspace.open
cockpit.surface.open
cockpit.surface.focus
cockpit.tab.activate
cockpit.pane.split
cockpit.object.reveal
cockpit.inspector.show
cockpit.viewport.fit
cockpit.follow_agent.set
```

Presentation commands require explicit presentation targets and do not become canonical work authority.

## 5.3 UI command rule

Every actionable durable Cockpit control SHALL bind to a registered capability ID and route through the command/controller/guard spine. Svelte stores may cache presentation state; they may not become independent command authority or canonical work stores.

## 5.4 Single capability registry

One source-controlled registry SHALL define every first-party query, command, and presentation operation. A capability descriptor SHALL include:

- stable capability ID;
- title and purpose;
- workspace and object kinds;
- operation kind;
- exact context requirement;
- typed input/output schema refs;
- events;
- side-effect class;
- preview, undo, idempotency, and compensation posture;
- offline/headless posture;
- worker/backend/renderer dependencies;
- permissions and entitlement;
- approval, receipt, and recovery policies;
- CLI namespace;
- visibility and progressive-discovery policy.

The registry SHALL drive or generate GUI bindings, CLI, API/OpenAPI, MCP/Pi tools, Capabilities UI, parity fixtures, and Help. No client or workspace maintains a separate command registry.

## 5.5 Workspace manifest extension

`WorkspaceManifest` from `UIAI-COCKPIT-003` SHALL reference registered queries, commands, presentation operations, events, semantic state projections, supported object kinds, context requirements, and capability-level runtime posture.

The existing workspace-level `local_only_behavior` remains only a coarse summary and cannot replace per-capability truth.

---

# 6. Exact Workstream and Attachment context

`UIAI-COCKPIT-004` defines scoped Work Surfaces and their relation to Focusa identities. This amendment extends that foundation to every control-plane command and visual object.

A generated Focusa consumer envelope SHALL carry, as applicable:

```ts
interface FocusaWorkstreamContext {
  project_scope_ref: string;
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

Every Focusa-bound command carries an exact generated context or an explicit verified Attachment reference. Server-global current state may not fill a missing field.

A UIAI-local object may remain unbound where its capability permits. Linking to a Focusa Workstream, proposing Evidence, checkpointing, or modifying Focusa relationships requires exact context.

Idempotency keys, leases, event subscriptions, caches, render caches, recovery sidecars, exports, telemetry, and collaboration room keys SHALL include exact Workstream context when bound.

---

# 7. Command, query, result, event, and semantic-state contracts

## 7.1 Command envelope

A common command envelope SHALL include:

- operation and capability IDs;
- actor identity;
- Cockpit Work Surface/client Attachment ref where applicable;
- exact Focusa context where applicable;
- target workspace/object/subobject refs;
- base revision;
- idempotency key;
- intent;
- preview or commit mode;
- typed input;
- requested presentation target.

## 7.2 Result envelope

A common result SHALL include:

- operation/capability ID;
- completed, previewed, blocked, conflict, failed, or cancelled state;
- target and changed refs;
- base and new revisions;
- receipt ref;
- render/export refs;
- Evidence proposals;
- typed warnings;
- typed recovery;
- output payload.

## 7.3 Semantic Cockpit state

Agents SHALL verify Cockpit state semantically rather than relying on screenshots.

Per client/window/Work Surface, the semantic state SHALL expose:

- protocol and compatibility state;
- exact Attachment and Workstream context;
- active workspace/subsection;
- active object and revision;
- open surfaces, tabs, panes, and windows;
- bounded selection or subobject ref;
- visible registered commands;
- participants and agent activity;
- proposals and approvals;
- blocks/recovery;
- jobs and renders;
- layout and event cursor;
- freshness.

Screenshots remain visual Evidence or debugging artifacts, not the state/control protocol.

---

# 8. CLI, API, MCP, Pi, and GUI parity

The stable low-level CLI SHALL support:

```text
uiai cockpit discover
uiai cockpit status
uiai cockpit state
uiai cockpit workspaces list
uiai cockpit capabilities list|search|describe
uiai cockpit attach|detach|switch
uiai cockpit surface list|open|focus|split|move|close
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

JSON output SHALL be stable and complete. Commands return operation and receipt refs. Typed denial/recovery classes use distinct exit behavior. Local selection convenience is nonauthoritative.

MCP and Pi tools wrap the same registry and contracts. Progressive discovery exposes only relevant and authorized capabilities.

For every implemented capability:

- registry entry exists;
- typed input/output exist;
- GUI, CLI, agent tool, and API behave equivalently;
- scope, permission, entitlement, side-effect, and approval behavior match;
- state is semantically verifiable;
- receipt/recovery are tested;
- headless/offline posture is truthful.

---

# 9. Shared UIAI work-object model

A common `CockpitWorkObject` SHALL identify:

- UIAI owner plane;
- stable object ID/ref;
- kind;
- title and lifecycle;
- current revision;
- optional exact Focusa Workstream/Workpoint binding and role;
- snapshot and operation-log refs;
- previews and exports;
- creator/updater provenance;
- timestamps.

Required visual/data kinds include:

```text
spreadsheet
whiteboard
chart
dataview
dashboard
generated_workspace
document
research
browser_session
image_asset
report
artifact
```

Binding states:

- **unbound:** UIAI-local object;
- **Workstream-bound:** exact verified project + Workstream;
- **Workpoint-bound:** exact Workpoint in that Workstream;
- **Evidence candidate:** immutable snapshot proposed to Focusa;
- **detached:** prior link retained in history, no current active relationship.

A project-only Focusa binding is prohibited. UIAI-local project organization cannot masquerade as canonical Focusa association.

Shared lifecycle:

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

Studio SHALL be a shared creative and analytical environment where humans and agents work on multiple structured visual objects under exact Attachments.

A Studio space may contain spreadsheets, whiteboards, charts, DataViews, dashboards, generated workspaces, documents, browser/research captures, report sections, Evidence snapshots, media, and artifacts.

A Studio workspace is a UIAI composition object with:

- stable ID/ref and revision;
- optional exact Focusa binding;
- participants;
- object refs and active object;
- agent access policy;
- destructive/publish/executable-content policy;
- timestamps and operation history.

It does not own Focusa Focus State, Workpoints, Trajectory, or Work Loop.

## 10.1 IA amendment to `UIAI-COCKPIT-003`

The permanent Studio object taxonomy SHALL become:

```text
Studio
  Spaces
  Whiteboards
  Visualizations
  Dashboards & Generated
  Assets
```

`Capture`, `Compare`, `Analyze`, `Design`, and `Produce` remain creation intents, commands, recipes, or filters. They are not durable object stores.

Documents owns document and spreadsheet library lifecycle, import, inbox, recent, pinned, templates, generated, and file management. Spreadsheet objects may be opened and composed in Studio.

No new top-level Charts, Whiteboards, Dashboards, Generated UI, or Reports workspace is created. Reports remain under Evidence and normal Work Surface tabs.

## 10.2 Studio composition

Default Studio composition:

```text
Workstream/Attachment header
object library or collection pane
Work Surfaces/tabs/panes from Cockpit 004
primary editor/viewport
Universal Inspector
participant and agent presence
activity/proposal bar
```

Universal Inspector sections are consistent across objects:

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

# 11. Human-agent collaboration

Participants include human, Focusa agent, UIAI agent, and authorized external agent identities with explicit roles and read/annotate/propose/edit/export/publish permissions.

Collaboration modes:

- **Observe** — inspect only.
- **Annotate** — comments/overlays.
- **Propose** — preview/proposal required.
- **Edit** — bounded reversible operations commit directly.
- **Delegated** — approved capability class commits within explicit limits.

Agent presence SHALL be visually distinct from human presence.

Human controls include follow agent, stop following, pause new commands, review proposals, take object/region/range control, return control, revoke delegation, and inspect receipts and bounded rationale.

Raw private model chain-of-thought is not displayed or stored as Evidence. Observable intent, constraints, actions, receipts, limitations, and bounded assessments may be presented.

---

# 12. Shared transaction, revision, lease, and conflict model

Every visual/data mutation uses one transaction envelope carrying:

- transaction/capability ID;
- exact workspace/object/actor;
- Work Surface/client Attachment ref;
- Focusa context when bound;
- base revision;
- idempotency key;
- intent and preview/commit mode;
- optional lease ref;
- typed operations;
- approval posture.

Execution path:

```text
resolve exact object and Attachment
→ verify Workstream context when bound
→ authorize capability and side effect
→ validate revision and dependencies
→ acquire object/region/range lease when required
→ preview and validate
→ execute atomically or through bounded subtransactions
→ recalculate/render
→ append event
→ emit receipt
→ update semantic state
→ optionally propose Evidence/checkpoint/continuation
```

Conflict rules:

- compatible non-overlapping operations may commit optimistically;
- overlapping or decisional conflicts create explicit conflict/proposal state;
- same-Workstream compatible replicated operations may converge through the selected engine;
- same-Workstream conflicting decisions require explicit review/PRE-equivalent resolution when Focusa meaning is affected;
- different Workstreams never merge canonical state;
- cross-Workstream reuse requires explicit reference, snapshot, or fork and defaults to `authority_transfers=false`;
- stale agent commands never silently overwrite newer human work.

Leases may target an object, whiteboard page/frame/region, workbook/sheet/range, chart spec/annotation layer, generated component subtree, or external publish/export action.

---

# 13. DataView foundation

Cockpit SHALL implement a first-class versioned `DataView` between source objects and visualizations.

A DataView records:

- source object refs, selectors, and source revisions;
- transform recipe;
- output schema;
- semantic annotations;
- output snapshot and digest;
- row count and warnings;
- freshness and refresh policy;
- lineage;
- optional exact Focusa binding.

Transform classes include selection, filtering, sorting, grouping/aggregation, derivation, joins, pivot/unpivot, time bucketing, unit/category normalization, validation, anomaly flags, and explicitly disclosed sampling.

Agents SHALL inspect actual values, distributions, cardinality, embedded totals, units, missing values, duplicates, and freshness—not only column names—before proposing charts.

---

# 14. Spreadsheet runtime

The initial spreadsheet vertical slice SHALL use:

- Univer as the active grid/workbook/formula interaction engine where suitable;
- Hucre as an import/export and round-trip codec adapter where suitable;
- UIAI-owned object identity, revisions, transactions, collaboration, receipts, Evidence, and Workstream binding.

No library-native ID becomes a stable public UIAI identity without adapter mapping. Commercial collaboration services do not silently become UIAI canonical authority.

Required capabilities:

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

Agent operations use cells, ranges, tables, names, formulas, formatting, charts, and workbook structure—not raw DOM interaction.

Initial collaboration may use workbook revisions, sheet/range leases, atomic transactions, optimistic non-overlapping edits, proposal mode, undo/compensation, and before/after previews. Overlapping range edits cannot use arbitrary last-writer-wins.

Imports SHALL classify formulas, external links/connections, macros, hidden structures, named ranges, embedded files, CSV formula injection, unsupported features, and round-trip loss. Executable or externally connected content is disabled/quarantined by default.

---

# 15. Whiteboard runtime

The initial whiteboard vertical slice SHALL use the tldraw SDK behind a UIAI-owned adapter.

The separate tldraw offline application is not an embeddable/forkable Cockpit dependency.

Because Cockpit uses Svelte and tldraw uses React, the preferred integration is a bounded React island mounted inside a Cockpit-owned host. Cockpit is not rewritten around React.

Self-hosted synchronization may use `@tldraw/sync-core` or an approved equivalent in a UIAI-authenticated document worker. Room IDs derive from opaque UIAI object handles, not paths or project names.

Agent control order:

1. semantic tldraw Editor operations;
2. structured store transactions;
3. `@tldraw/driver` for parity tests or unrepresented gestures;
4. bounded visual/DOM fallback only exceptionally.

Required capabilities:

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

Structured snapshots expose pages, shapes/bindings, text index, spatial clusters, selected objects, viewport, assets, recent operations, embedded object refs, Focusa entity refs, render ref, and exact revision.

Custom projections may represent Workpoint, Task, Decision, Assumption, Blocker, Evidence, Deliverable, Agent Assignment, Browser Research, Spreadsheet Range, File Artifact, and Checkpoint. They store refs and projection revisions, not duplicate Focusa state.

Imported document scripts are disabled/quarantined by default. Declarative behavior is safe default; sandboxed scripts require explicit restricted capabilities; trusted extensions are signed and installed at application level.

---

# 16. Flint semantic visualization runtime

The initial visualization vertical slice SHALL use the `flint-chart` library directly behind a pinned UIAI adapter.

The Flint MCP server remains optional external interoperability. Internal Cockpit execution routes through UIAI object, revision, policy, receipt, and Evidence contracts.

The canonical editable chart is a semantic object containing:

- stable chart ref/revision;
- DataView ref and revision;
- Flint semantic types and chart spec;
- field display names/options;
- preferred backend;
- derived backend spec refs;
- render refs;
- interaction and annotation refs;
- warnings;
- optional exact Focusa binding.

Three forms remain distinct:

1. semantic chart object;
2. derived Vega-Lite/ECharts/Plotly/Chart.js/Excel/SVG/PNG output;
3. immutable Evidence snapshot with exact dependencies, render, annotations, warnings, actor, receipts, and digest.

Required capabilities:

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

Agents edit Flint/DataView semantics first. Backend-specific edits are explicitly labeled nonportable and cannot silently replace the semantic source.

Humans and agents may switch supported chart types, bind fields, filter/select data, resize, annotate, request explanations/alternates, compare revisions, embed charts, and freeze Evidence.

Flint receives bounded inline rows or a UIAI-brokered DataView snapshot—not unrestricted filesystem paths or remote URLs.

---

# 17. Generated workspaces and A2UI compatibility

Cockpit SHALL support bounded generated task-specific workspaces such as data investigations, architecture comparisons, pricing decision dashboards, launch planning, incident review, and research synthesis.

Generated workspace manifests SHALL remain compatible with the approved A2UI direction and Focusa Desktop's existing A2UI 0.9.1 Lit renderer and Focusa custom elements. UIAI may host them inside its Svelte shell through product-neutral adapters/custom elements, but SHALL NOT invent a competing portability format.

Allowed registered components include:

- Flint chart;
- table/data grid;
- spreadsheet view/range;
- whiteboard/frame;
- document/Markdown;
- metric/KPI;
- timeline;
- comparison matrix;
- Evidence list;
- receipt viewer;
- research sources;
- browser artifact/capture;
- agent status;
- form/parameter controls;
- approval/review controls;
- report section;
- media/artifact preview.

Every interactive control binds to an existing capability ID.

Lifecycle:

```text
ephemeral → pinned → promoted → templated → archived
```

Generated workspaces SHALL NOT execute arbitrary JavaScript, create capabilities, modify grants, bypass guards, issue arbitrary network/filesystem requests, hide consequential operations, own Focusa cognition, treat prose as authorization, mutate another Workstream through reused refs, or ship inaccessible controls.

---

# 18. Cross-surface references and embedding

Stable refs SHALL support whiteboard shapes/frames, spreadsheet ranges, chart selections, DataView revisions, research, browser sessions, Workstreams, Workpoints, reports, and Evidence.

Every embed declares:

- **live reference** — source remains authoritative and updates may flow;
- **frozen snapshot** — exact revision pinned;
- **fork** — new object with explicit lineage.

Embed contracts define source/revision, permissions, edit routing, refresh, staleness, missing-source behavior, cycle prevention, selection ownership, export, Evidence, and cross-Workstream policy.

Flattened screenshots are outputs, not the normal collaboration model. Different-Workstream live mutation is prohibited. Cross-Workstream use requires explicit reference/snapshot/fork with no authority transfer by default.

---

# 19. Focusa and Focusa Desktop bidirectional loop

Required flow:

```text
Focusa Workstream/Workpoint intent
→ verified Cockpit Work Surface/Attachment
→ work-object creation or inspection
→ human-agent collaboration
→ UIAI revisions, renders, receipts, findings, annotations
→ Evidence candidate and continuation packet
→ Focusa reducer accepts/rejects/associates meaning
→ next Workpoint/Work Loop action
→ optional presentation in Cockpit or Focusa Desktop
```

Transfer modes:

### Live reference

UIAI remains authoritative for the mutable object. Focusa Desktop/Cockpit presents by stable ref and routes commands to UIAI.

### Immutable snapshot

Exact object revision, dependencies, render, digest, provenance, and receipts are proposed as Focusa Evidence.

### Fork

A new UIAI object records source lineage, source Workstream/revision, reason, and authority-transfer posture.

### Continuation packet

A bounded packet carries source object/revision, exact Focusa context, active selection/subobject, viewport/range/frame, human annotations, accepted findings, unresolved questions, Evidence/receipt refs, dependency revisions, and suggested next actions.

A continuation packet is advisory. It does not become Focus State or a Workpoint merely because it exists.

UIAI may propose attach as working material, active deliverable, decision artifact, Evidence capture, Workpoint checkpoint, or follow-up. Focusa accepts/rejects/associates through the exact Workstream reducer. UIAI never silently canonizes output.

---

# 20. Mutable work versus immutable Evidence

Live objects continue changing; Evidence pins an exact revision.

Evidence capture SHALL include, as applicable:

- object ref/revision;
- exact project/Workstream/Workpoint binding;
- source object/DataView revisions;
- transform recipe and output digest;
- semantic chart/document spec;
- whiteboard page/frame/shape refs;
- spreadsheet sheet/range/formula refs;
- generated component graph;
- render/export refs;
- actor/participant provenance;
- human annotations and decisions;
- operation receipts;
- warnings/limitations;
- freshness/staleness;
- hashes;
- redaction/audience posture.

Evidence remains immutable if the source is edited, detached, archived, or deleted.

---

# 21. Security and trust boundaries

Spreadsheet cells/formulas, shape text/links, chart labels/annotations, imported documents/comments, browser captures, SVG/HTML/media, generated content, external suggestions, scripts, and macros are untrusted unless verified.

Untrusted content cannot modify Workstream binding, Capability Grants, approval policy, tool schemas, permission/entitlement, command routing, or authority.

Common controls:

- content sanitation;
- CSP/sandboxing;
- origin/network allowlists;
- path/file-broker restrictions;
- no chart/data SSRF;
- formula/macro/external-link quarantine;
- CSV injection handling;
- SVG/HTML script/event stripping;
- prompt-injection provenance boundaries;
- bounded data/shape/component/render sizes;
- cancellation/timeouts;
- worker isolation;
- destructive/external-publish approvals;
- secrets redaction;
- audit/receipts.

Executable tiers are declarative, explicitly sandboxed, or trusted signed application extension. Document-contained code never inherits Focusa authority.

---

# 22. Accessibility

Charts SHALL provide semantic titles/descriptions, source/freshness disclosure, table fallback, keyboard data navigation where interactive, non-color distinctions, accessible filter/selection, warnings, reflow/zoom, and export alt text/data.

Whiteboards SHALL provide keyboard creation/selection/edit/group/connect/delete, a structured outline of pages/frames/shapes/text/bindings, meaningful custom-shape names, accessible comments/proposals, reduced motion/high contrast, and nonvisual property inspection.

Spreadsheets SHALL provide complete grid semantics, keyboard navigation/editing, formula/error announcements, range status, accessible sheets/tables, and nonvisual inspection/export.

Generated workspaces SHALL use registered accessible components, deterministic focus order, approved labels, reflow, keyboard operation, and bounded live regions. Models may not invent arbitrary ARIA or hide required operations behind hover/drag/color.

---

# 23. Headless, offline, remote, and restart behavior

Domain commands SHALL run with Cockpit closed where the capability declares headless support, including spreadsheet operations, whiteboard semantic operations through workers, DataView refresh, chart validation/render, generated manifest composition, export, and Evidence capture.

When Cockpit opens/reconnects, it attaches to current revisions, event cursor, collaboration, jobs/renders, exact Work Surface/Attachment, proposals/approvals, and recovery state. It does not reconstruct authority from the last visible UI.

Remote hosts/worktrees use explicit verified project/workspace bindings. Identical paths on different hosts and different worktrees remain distinguishable.

Every capability declares local/offline, network, renderer, worker/backend, Focusa, remote, degraded, and recovery posture. Missing/planned backends are presented truthfully.

---

# 24. Dependency, licensing, and adapter governance

External dependencies follow:

```text
Adopt → Wrap → Configure → Extend → Custom
```

Before production use:

- pin version;
- review license/distribution obligations;
- add notices/SBOM;
- document security;
- define adapter boundary;
- add conformance fixtures;
- define fallback/degraded behavior;
- add upgrade/rollback procedure;
- prevent third-party IDs from leaking into stable contracts.

Specific rules:

- tldraw offline is not the shipped Cockpit runtime;
- tldraw SDK licensing must be satisfied;
- Flint is pinned behind UIAI adapter; MCP is optional interop;
- Univer commercial collaboration does not silently become UIAI authority;
- Hucre requires codec quality/loss/license/security review;
- Focusa Desktop retains approved A2UI 0.9.1 Lit until coordinated upgrade;
- no competing generated-UI portability format.

---

# 25. Integration with numbered documents

## `UIAI-COCKPIT-000`

The master remains authoritative for product shape, Focusa/UIAI boundaries, Workpoints, Evidence, and progressive disclosure. This amendment adds universal control and multimodal runtimes.

## `UIAI-COCKPIT-001`

Review Reports remain Evidence-centered presentation/review objects. Charts, tables, diagrams, whiteboard frames, spreadsheet ranges, and generated views reference the work-object/DataView contracts here rather than creating report-local truth. Live reports show freshness; frozen reports pin revisions/digests.

## `UIAI-COCKPIT-002`

Agent-first browser objects/observations/actions/provenance may embed through stable refs. Browser authority remains UIAI-owned and does not transfer to Focusa Desktop/generated UI.

## `UIAI-COCKPIT-003`

This amendment modifies Studio secondary navigation/object taxonomy, Documents spreadsheet placement, `WorkspaceManifest` capability metadata, Studio/Documents route skeletons, Search/Command integration, Universal Inspector sections, and task ordering for `T003-02`, `T003-11`, and `T003-14`.

## `UIAI-COCKPIT-004`

This amendment consumes `004` Work Surfaces, tabs, panes, windows, WorkSurfaceId, exact command targeting, Attachment binding, restoration, multiple-window isolation, DnD, motion, accessibility, and Tauri security. `005` does not define a second surface/layout system.

---

# 26. Implementation boundaries

Normative ownership boundaries:

```text
UIAI authority/control
  capability registry
  command/query router
  Workstream/Attachment resolver
  policy/approval/entitlement
  revisions/leases/idempotency
  receipts/events/recovery
  semantic Cockpit state

Shared work-object runtime
  object registry/storage
  operation log/snapshots/exports
  embeds/lineage
  collaboration/presence
  Evidence proposal adapter

Visual/document workers
  spreadsheet adapter
  whiteboard/tldraw adapter
  DataView transforms
  Flint/chart render adapter
  generated workspace validation/rendering

Cockpit
  control-plane client
  Workspace/Work Surface integration
  object library/editors
  React/Lit islands
  Universal Inspector
  presence/proposals/activity

Interop
  CLI
  REST/OpenAPI
  MCP/Pi
  Focusa binding/Evidence proposal
  Focusa Desktop presenter/handoff
```

No Svelte component calls raw production routes directly. No adapter bypasses the command router. No client maintains a separate registry.

---

# 27. Machine-readable companion

Create:

```text
UIAI-COCKPIT-005-C01
```

It SHALL enumerate stable requirement IDs, schemas, object kinds, commands/queries/presentation capabilities, events, context requirements, side effects, approvals/receipts, entitlement/permissions, adapter/version inventory, tasks/dependencies, security/accessibility/performance gates, adversarial fixtures, ownership, and release Evidence.

Disagreement with this normative amendment blocks implementation.

---

# 28. Implementation task graph

## T005-00 — Register and freeze terminology

Register this document; map sources; prohibit new canonical Thread, project-root-only binding, Continuity-as-Workstream identity, and global/current Workstream authority.

## T005-01 — Focusa contract alignment

Consume generated project/Workstream/Continuity/Attachment/Workpoint/event-head/authority contracts; define fail-closed behavior and discrepancy handling until Spec 158 publishes.

## T005-02 — Universal capability registry and envelopes

Implement command/query/result/event schemas, side effects, runtime posture, approvals, receipts, recovery, and registry validation.

## T005-03 — Work Surface semantic state

Extend `004` surfaces/windows/Attachments with exact per-client semantic state, events, presentation commands, and zero global current Workstream.

## T005-04 — CLI/API/MCP/Pi/GUI parity

Generate/wrap CLI, OpenAPI, MCP/Pi tools, command palette, GUI bindings, and parity fixtures from the registry.

## T005-05 — Shared object/transaction/collaboration contracts

Implement work objects, Studio spaces, revisions, transactions, operation logs, leases, participants, embeds, lineage, retention, and Evidence proposals.

## T005-06 — Studio/Documents IA reconciliation

Apply object taxonomy, spreadsheet placement, routes, Inspector, Search, and `003` task integration without creating new top-level products.

## T005-07 — DataView runtime

Implement source bindings, transforms, semantics, freshness, lineage, resource limits, and agent inspection.

## T005-08 — Spreadsheet vertical slice

Implement Univer/Hucre adapters, import/edit/recalc/render/export, transactions/leases, agent tools, security, headless operation, and Evidence.

## T005-09 — Whiteboard vertical slice

Implement tldraw React island, sync worker, semantic operations/snapshots, custom ref shapes, script quarantine, collaboration, agent tools, and Evidence.

## T005-10 — Flint visualization vertical slice

Implement pinned Flint adapter, semantic chart object, DataView binding, live view, validation, rendering, annotations, exports, collaboration, agent tools, and Evidence.

## T005-11 — Generated A2UI-compatible workspace slice

Implement component grammar, lifecycle, validator, capability bindings, Focusa Desktop conformance, accessibility, and no arbitrary code.

## T005-12 — Cross-surface composition

Implement spreadsheet→DataView→chart, chart→whiteboard/report/dashboard, browser/research refs, and live/frozen/fork semantics.

## T005-13 — Focusa/Focusa Desktop handoff

Implement live presenter, Evidence proposal, explicit fork, continuation packet, Workpoint/checkpoint proposals, opaque refs, and receipts.

## T005-14 — Security/accessibility/dependency/performance gates

Implement trust boundaries, resource caps, a11y, license/SBOM/pins, conformance, and CI-blocking fixtures.

## T005-15 — Headless/offline/remote/restart proof

Prove closed-Cockpit operation, reattachment, worker/renderer outages, remote/worktree identity, stale handles, recovery, and rollback.

## T005-16 — Adversarial acceptance and rollout

Run full multi-Workstream/multi-session/multi-agent/conflict/contamination/Evidence/restart/compatibility proof; update docs/help/release evidence; remove temporary compatibility paths.

Dependency order:

```text
T005-00 → T005-01 → T005-02
T005-02 → T005-03 → T005-04
T005-02 → T005-05 → T005-06
T005-05 → T005-07 → T005-08 and T005-10
T005-05 → T005-09
T005-05 + T005-10 → T005-11
T005-08 + T005-09 + T005-10 + T005-11 → T005-12
T005-01 + T005-03 + T005-05 + T005-12 → T005-13
all slices → T005-14 → T005-15 → T005-16
```

---

# 29. Mandatory adversarial acceptance matrix

## Focusa foundation

- [ ] Two Workstreams in one project attach to two Cockpit windows without contamination.
- [ ] Switching one Work Surface/window does not switch another.
- [ ] No global active/current/latest/last Workstream authorizes reads/writes.
- [ ] Continuity, Session, path, cwd, WorkSurfaceId, or UI focus cannot identify authority alone.
- [ ] Unbound/ambiguous operations return zero foreign payload and typed recovery.
- [ ] Thread remains historical terminology only.
- [ ] Exact Workstream survives restart, reconnect, provider/model change, remote host, and Desktop handoff.

## Universal control

- [ ] Every durable GUI action has a capability ID.
- [ ] Every capability has typed context/input/output/side-effect/approval/receipt/recovery metadata.
- [ ] CLI/API/MCP/Pi/GUI use equivalent fixtures and guards.
- [ ] Domain commands work headlessly where declared.
- [ ] Presentation commands target exact Work Surfaces/windows.
- [ ] No Svelte-local store becomes authority.

## Collaboration

- [ ] Stable object identity survives renderer/adapter restart.
- [ ] Non-overlapping edits commit safely.
- [ ] Overlapping edits create explicit conflict/proposal.
- [ ] Human takeover/return control works.
- [ ] Agent presence is distinct and semantically inspectable.
- [ ] Stale agent operations cannot overwrite newer human work silently.
- [ ] Different Workstreams never merge implicitly.

## Spreadsheet

- [ ] Import/edit/recalc/render/export is versioned and receipted.
- [ ] Lossy/unsupported features are disclosed.
- [ ] Formula/macro/external-link/CSV injection controls pass.
- [ ] Range leases/conflicts are proven.
- [ ] Agent uses semantic operations, not DOM automation.

## Whiteboard

- [ ] Structured snapshots/semantic operations support agent use.
- [ ] Human and agent collaborate live on separate regions.
- [ ] Scripts are disabled/quarantined by default.
- [ ] Focusa shapes store refs/projections, not canonical duplicates.
- [ ] Driver/DOM is fallback, not primary.
- [ ] Accessibility outline/keyboard operations pass.

## DataView and chart

- [ ] DataView reproduces output from exact source revisions/transforms.
- [ ] Agent inspects values/cardinality/units before charting.
- [ ] Flint semantic spec remains editable source.
- [ ] Backends are derived/labeled.
- [ ] Live refresh discloses revision/freshness.
- [ ] Warnings/truncation/aggregation/sampling are visible.
- [ ] Accessible summary/table fallback pass.
- [ ] Flint has no unrestricted path/URL authority.

## Generated workspace

- [ ] Only registered A2UI-compatible components are used.
- [ ] Every control binds to a capability.
- [ ] Arbitrary code/network/filesystem is blocked.
- [ ] Lifecycle works.
- [ ] Workspace works in Cockpit and presents in Focusa Desktop without competing renderer.
- [ ] Focus/keyboard/reflow/labels pass.

## Cross-surface and handoff

- [ ] Spreadsheet range drives DataView/live chart.
- [ ] Chart embeds live in whiteboard and freezes into report/Evidence.
- [ ] Browser/research refs embed by stable ref.
- [ ] Live/frozen/fork is explicit.
- [ ] Focusa Desktop presents and routes edits without owning UIAI object.
- [ ] Continuation resumes exact object/selection/questions in correct Workstream.
- [ ] Focusa accepts/rejects UIAI Evidence proposals.
- [ ] Later live revisions cannot alter prior Evidence.

## Security/performance/recovery

- [ ] Untrusted content cannot change authority/tool schemas.
- [ ] Resource limits isolate unrelated Workstreams.
- [ ] Cancellation/rollback/compensation preserve revision truth.
- [ ] Worker/renderer outages produce typed recovery.
- [ ] Restart/replay restore independent objects/Attachments.
- [ ] Licenses/notices/SBOM/pins/conformance pass.

---

# 30. Closure prohibition

Do not claim completion because one Studio page, chart, spreadsheet, canvas, MCP tool, project path, `workstream_id` field, visual GUI agent, cursor demo, adapter-native collaboration, generated HTML page, Evidence export, grep, or unit test works.

Do not claim completion while:

- a global fallback remains;
- different Workstreams are separated only by filters over mixed state;
- third-party engines define canonical authority;
- Focusa Desktop and Cockpit use competing generated-UI formats;
- UIAI silently promotes artifacts into Focusa truth;
- Evidence changes with mutable source objects;
- accessibility/security/headless/restart proof is incomplete.

Closure requires exact Workstream contracts, one universal registry, complete semantic state, all-surface parity, shared object/revision/collaboration contracts, complete spreadsheet/whiteboard/DataView-chart/generated slices, Focusa Desktop handoff, immutable Evidence/receipts, and all Section 29 gates with stable proof.

---

# 31. Prohibited patterns

Implementation MUST NOT introduce:

- Thread as a new canonical owner;
- project path plus Continuity as Workstream identity;
- daemon-global or Cockpit-global cognition;
- global current/active/latest/last mutation authority;
- project-only Focusa binding for canonical handoff;
- UI focus/WorkSurfaceId as authority;
- separate command registries per workspace/client;
- direct raw production API calls from Svelte;
- third-party libraries as canonical identity/authority;
- screenshot-only agent control;
- raw DOM automation as normal data/canvas/chart API;
- arbitrary generated JavaScript/remote embeds;
- unrestricted Flint path/URL access;
- tldraw scripts enabled by default;
- hidden spreadsheet executable/external content;
- cross-Workstream implicit merge;
- mutable Evidence snapshots;
- silent UIAI canonization into Focusa;
- second browser authority in Focusa Desktop;
- competing generated-UI portability format;
- top-level Charts/Whiteboards/Dashboards/Reports fragmentation;
- inaccessible visual-only operations/Evidence.

---

# 32. Final architectural principle

> **UIAI Engine Cockpit is the shared visual execution environment of Focusa-powered work, not a collection of disconnected tabs. Every workspace and visual object inherits one Workstream-scoped control plane, one work-surface and object model, one governed collaboration model, and one proof-and-continuation path. Focusa remembers why the work exists and canonizes meaning; UIAI Engine acts, renders, and produces inspectable artifacts; Cockpit makes that work visible and steerable; Focusa Desktop carries the exact same Workstream forward without reintroducing a singleton.**
