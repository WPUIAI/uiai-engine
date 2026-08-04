# UIAI Cockpit Scoped Work Surfaces, Tabs, Panes, Windows, and Motion Specification

**Document number:** `UIAI-COCKPIT-004`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding numbered decisions/amendments:** `UIAI-COCKPIT-001`, `UIAI-COCKPIT-002`, `UIAI-COCKPIT-003`  
**Status:** Proposed normative foundational amendment  
**Version:** 1.0  
**Date:** 2026-08-04  
**Repository:** `WPUIAI/uiai-engine`  
**Primary implementation home:** `apps/cockpit/`  
**Primary implementation stack:** SvelteKit 2, Svelte 5, Tauri v2  
**Scope:** Corrected Focusa Workstream consumption, true Cockpit scoping, work surfaces, tabs, tab groups, split panes, multiple windows, browser-target binding, drag-and-drop, restoration, command targeting, assistant context, motion, accessibility, security, migration, implementation order, and proof

---

## 0. Decision

UIAI Cockpit SHALL implement one **scoped work-surface system** for open work across tabs, panes, windows, and browser/document/test/report surfaces.

The work-surface system SHALL be designed around the corrected Focusa identity hierarchy:

```text
ProjectRootKey
  → WorkstreamId
    → ContinuityId and AttachmentId
      → SessionId and InstanceId
        → UIAI runtime identities
          → Cockpit WorkSurfaceId
```

The following distinctions are mandatory:

1. `WorkstreamId` is the stable canonical identity of a coherent body of work inside a project.
2. `ContinuityId` is a continuation lineage, generation, rollover, or resumable epoch inside a Workstream. It is not the Workstream identity.
3. `SessionId` is temporal runtime metadata. It is not project or Workstream authority.
4. `AttachmentId` binds a Session or Instance to one Workstream and, where applicable, one continuity lineage.
5. `WorkSurfaceId` identifies a presentation of work in Cockpit. It is never canonical work authority.
6. Visual selection, keyboard focus, tab order, pane membership, window placement, and layout restoration SHALL NOT mutate Focusa project, Workstream, Workpoint, Trajectory, Focus Stack, Work Loop, Evidence, ownership, or authority.

This amendment supersedes any Cockpit interpretation that treats a thread, continuity, session, selected tab, selected sidebar row, or daemon-global `active/current/latest` value as the canonical Workstream boundary.

This amendment does not create Focusa canonical identity or state authority. Focusa owns and publishes those contracts. Cockpit consumes a versioned generated contract bundle and fails closed when required authority contracts are unavailable or ambiguous.

---

# 1. Authority, sources, and dependency posture

## 1.1 Application order

Implementation SHALL apply sources in this order:

1. The accepted Focusa canonical Workstream-scoped runtime specification and generated contract bundle.
2. `UIAI-COCKPIT-000` unified master specification.
3. `UIAI-COCKPIT-001` Review Reports integration decision.
4. `UIAI-COCKPIT-002` agent-first browser amendment and companion ledger.
5. `UIAI-COCKPIT-003` sidebar navigation, IA, and DnD implementation amendment.
6. This `UIAI-COCKPIT-004` work-surface amendment.
7. Current repository contracts, route parity, implementation code, tests, and release proof.

When an existing Cockpit contract conflicts with the accepted Focusa Workstream contract, the Focusa generated contract governs canonical identity and authority. Cockpit presentation contracts SHALL be migrated rather than patched with ambiguous aliases.

## 1.2 Upstream dependency

Before canonical multi-Workstream mutation ships, Focusa must normatively settle and generate at least:

- `ProjectRootKey` or equivalent verified project identity;
- `WorkstreamId` and `WorkstreamRef`;
- `ContinuityId` and `ContinuityRef`;
- `AttachmentId` and `AttachmentRef`;
- Session and Instance references;
- Workstream-scoped Workpoint, Trajectory, Focus Stack, Focus State, Work Loop, Evidence, event-head, snapshot, and authority references;
- exact-scope request and result envelopes;
- migration posture from continuity-as-workstream and legacy thread terminology;
- canonical, advisory, degraded, blocked, and aggregate projection states.

The TypeScript examples in this amendment are consumer mappings. They MUST NOT become hand-maintained duplicates of Focusa DTOs after generated contracts exist.

## 1.3 Current repository sources

Primary current sources include:

- `apps/cockpit/src/routes/+layout.svelte`;
- `apps/cockpit/src/routes/+page.svelte`;
- `apps/cockpit/src/lib/contracts/scope-ref.ts`;
- `apps/cockpit/src/lib/contracts/card-manifest.ts`;
- `apps/cockpit/src/lib/ui/design-tokens.css`;
- `apps/cockpit/src-tauri/tauri.conf.json`;
- `apps/cockpit/src-tauri/src/main.rs`;
- `docs/cockpit/000/UIAI_COCKPIT_000_V04.md`;
- `docs/cockpit/000/UIAI_COCKPIT_000_V08.md`;
- `docs/cockpit/000/UIAI_COCKPIT_000_V09.md`;
- `docs/cockpit/000/UIAI_COCKPIT_000_V10.md`;
- `docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`;
- `docs/UIAI_COCKPIT_003_SIDEBAR_NAVIGATION_IA_DND_IMPLEMENTATION_SPEC_2026-08-01_v1.0.md`.

---

# 2. Current implementation audit and foundational correction

## 2.1 Current Cockpit implementation

The current Cockpit is still the Slice 0 shell:

- one static scope strip;
- one primitive navigator;
- one route slot;
- one static Inspector placeholder;
- one static process ribbon;
- one Phase 0 card grid;
- no work-object tab implementation;
- no pane layout tree;
- no work-surface store;
- no tab DnD;
- no split-pane DnD;
- no multiple-window work-surface manager;
- no layout snapshot restoration;
- no transition orchestrator;
- no generated Focusa Workstream contract consumption.

`UIAI-COCKPIT-003` correctly adds a work-object tab region to the target shell, but its normative DnD scope is sidebar workspaces, Favorites, saved views, and shortcuts. It does not fully define tab, pane, group, window, runtime-binding, or layout-restoration semantics.

## 2.2 Current scope contract problem

The current `ScopeRef` permits optional project, workstream, continuity, thread, session, node, machine, endpoint, role, and authority fields in one loose object.

That shape is insufficient because it:

- allows canonical and temporal identities to be absent or ambiguous;
- permits `thread_id`, `workstream_key`, and `continuity_id` to coexist without precedence;
- mixes authority, routing, runtime correlation, placement, and display context;
- cannot distinguish a canonical Workstream target from a project aggregate or system surface;
- encourages fallback to whichever optional field happens to be available.

The current `ScopeRef` SHALL be deprecated for new canonical Cockpit work and replaced by generated Focusa references plus a discriminated Cockpit scope binding.

## 2.3 Foundational correction

The governing state hierarchy is:

```text
Daemon infrastructure
└── Project scope
    └── Workstream
        ├── Focus Stack
        ├── Focus State
        ├── Workpoints
        ├── Workstream Trajectory
        ├── Work Loop
        ├── Context and cognition
        ├── Evidence links
        ├── canonical event head
        ├── continuity lineages
        └── Attachments
            └── Sessions and Instances
                └── UIAI runtimes
                    └── Cockpit work surfaces
```

One daemon process MAY host many projects and Workstreams. One Cockpit window MAY display many projects and Workstreams. Neither process identity nor window identity establishes canonical scope.

---

# 3. Canonical terminology

## 3.1 Project scope

A Project scope is the verified root authority boundary. Cockpit SHALL display a human-readable project label but retain the exact generated `ProjectRootKey` for scope resolution.

## 3.2 Workstream

A Workstream is the durable canonical identity of a coherent body of work within one Project scope.

A Workstream owns or references its own:

- Focus Stack and active Focus Frame;
- Focus State;
- Workpoints and active Workpoint selection;
- Workstream Trajectory;
- Work Loop and scoped writer ownership;
- context and cognitive projections;
- Evidence linkage;
- event head and materialized snapshot;
- continuity lineages;
- attachments and authority relationships.

A Project MAY contain many active Workstreams simultaneously.

## 3.3 Continuity

Continuity is a resumable lineage or generation inside a Workstream. A continuity rollover does not create another Workstream unless an explicit governed fork operation does so.

## 3.4 Session and Instance

A Session is temporal execution context. An Instance is an executing client or agent runtime. They may restart, reconnect, move nodes, or rotate while the Workstream remains stable.

Session and Instance identities are correlation and attachment identities. They do not establish project or Workstream authority by themselves.

## 3.5 Attachment

An Attachment explicitly binds a Session or Instance to a Workstream with role, authority, lifecycle, lease, and optional continuity metadata.

Every scope-bearing Cockpit command originating from a runtime-backed surface SHALL resolve through an explicit Attachment or a generated direct authority grant. Ambient last-session or last-workstream fallback is prohibited.

## 3.6 Workspace

A Workspace is a stable Cockpit product domain such as Live, Test Lab, Documents, Research, Studio, Evidence, Activity, Nodes & Services, or Capabilities.

A Workspace is navigation taxonomy, not canonical work identity.

## 3.7 Work Object

A Work Object is a referencable object such as:

- browser session;
- browser target;
- FPV share;
- test flow;
- test run;
- document;
- research capture;
- research packet;
- visual comparison;
- media job;
- workflow run;
- artifact;
- Evidence item;
- Review Report;
- report snapshot;
- Workpoint;
- Workstream summary or projection.

## 3.8 Work Surface

A Work Surface is one Cockpit presentation of a Work Object in a pane or window.

Several Work Surfaces may present the same Work Object. A Work Surface may be closed without ending the underlying Work Object, Session, browser target, job, or Workstream.

## 3.9 Surface Group

A Surface Group is a presentation-only grouping of tabs or surfaces. It replaces the ambiguous proposed term `Workset`.

A Surface Group is not a Workstream and never owns canonical work.

## 3.10 Pane Layout and Layout Snapshot

A Pane Layout is the recursive spatial arrangement of panes in one window.

A Layout Snapshot is a saved or crash-recovery presentation arrangement containing surfaces, groups, pane geometry, selected surface, Inspector state, and window placement. It does not snapshot or restore canonical Focusa authority.

## 3.11 Deprecated canonical term: Thread

`Thread` MUST NOT remain a parallel canonical identity beside Workstream.

Where a legacy API or stored record uses `thread_id`, it SHALL be treated as one of:

- a migration input;
- a compatibility alias resolving to one explicit Workstream;
- a conversational UI object that carries no canonical authority;
- an unsupported ambiguous value that blocks canonical mutation.

No new Cockpit contract SHALL use `thread_id` as Workstream authority.

---

# 4. Non-negotiable invariants

## 4.1 Authority invariants

1. No canonical mutation without an exact Project scope and Workstream target.
2. No visual or keyboard state creates canonical authority.
3. No Session, Continuity, browser target, selected tab, selected sidebar item, or route creates Workstream authority by itself.
4. Missing or ambiguous scope blocks canonical mutation.
5. A Project aggregate is read-only or advisory unless an explicit command is decomposed into independently authorized Workstream operations.
6. Cross-Workstream mutation is never represented as one ambient mutation.
7. Each Workstream sub-operation receives its own authority result, event, Evidence linkage, and Receipt where required.

## 4.2 Presentation invariants

The following operations change presentation only:

- selecting a tab;
- reordering tabs;
- renaming a tab label;
- pinning or unpinning a tab;
- moving a tab between panes;
- moving a tab between windows;
- placing a tab in a Surface Group;
- creating, resizing, rotating, maximizing, merging, or closing a split pane;
- saving or restoring a Layout Snapshot;
- hiding or showing the sidebar or Inspector.

These operations MUST NOT change Project, Workstream, Continuity, Attachment, Workpoint, Trajectory, Focus Stack, Work Loop, Evidence, Session lease, browser ownership, capability grant, or entitlement.

## 4.3 Lifecycle invariants

Commands SHALL use distinct language and effects:

```text
Close view
Detach Session
Park browser target
End Session
Stop run
Close Workstream
Archive Workstream
Delete artifact
```

`Close view` is the default tab/pane close operation. Destructive or runtime-ending actions require separately named commands and existing authority/approval paths.

## 4.4 Focus terminology invariant

Cockpit UI implementation SHALL use:

- `active_surface`;
- `selected_surface`;
- `keyboard_focus`;
- `visual_focus`.

It SHALL NOT use a selected tab or pane as a synonym for Focusa Focus Stack, Focus Frame, Focus State, active Workpoint, or canonical focus.

---

# 5. Scope-binding model

## 5.1 Required discriminated binding

Replace the loose all-optional scope object with a discriminated binding:

```ts
export type CockpitScopeBinding =
  | {
      kind: "workstream";
      workstream: FocusaWorkstreamRef;
      authority_state: ScopeAuthorityState;
    }
  | {
      kind: "attachment";
      attachment: FocusaAttachmentRef;
      authority_state: ScopeAuthorityState;
    }
  | {
      kind: "project_aggregate";
      project_root_key: string;
      posture: "read_only" | "advisory";
    }
  | {
      kind: "system";
      system_scope: "settings" | "help" | "host";
    };
```

Generated Focusa types SHALL replace the provisional names above when available.

## 5.2 Binding laws

- A `workstream` binding may issue canonical commands only after current authority validation.
- An `attachment` binding may issue only commands permitted by that Attachment role and grant.
- A `project_aggregate` binding cannot silently select a Workstream for mutation.
- A `system` binding cannot mutate Workstream cognition.
- Scope binding is immutable for the lifetime of one Work Surface. Rebinding creates a new surface or an explicit governed rebind event.
- A browser navigation does not change the Workstream binding.
- A route change does not change the Workstream binding unless the newly opened object declares and verifies a different binding.

## 5.3 Window Context Lens

The shell MAY expose a Window Context Lens such as:

```text
Project Alpha · All Workstreams
```

or:

```text
Project Alpha · Checkout Workstream
```

This lens is a filter and navigation preference. It is not authority creation and does not override the active surface binding.

## 5.4 Active Surface Scope

The toolbar and Inspector SHALL expose the active Work Surface's exact scope in human-readable form, with technical details behind disclosure:

```text
Project Alpha
Checkout Workstream
Continuity generation 3
Pi Session 8 attached
UIAI browser target 19
Authority verified
```

## 5.5 Command Scope Receipt

Before a meaningful mutation, Cockpit SHALL resolve a command target and present a bounded scope receipt appropriate to risk:

```text
Apply to
Project Alpha
Checkout Workstream
Workpoint: Verify payment flow
Attachment: Pi Session 8
Authority: verified
```

A command MUST NOT fall back to:

- daemon-global active state;
- the most recently used Workstream;
- the latest Session;
- the selected sidebar row;
- another pane's active surface;
- a similar Workpoint or Trajectory;
- the latest continuity without verified lineage.

---

# 6. Work-surface architecture

## 6.1 Stable shell composition

The target shell becomes:

```text
CockpitWindow
  UnifiedToolbar
  PrimarySidebar
  WindowContextLens
  WorkSurfaceRegion
    PaneLayoutTree
      Pane
        SurfaceTabStrip
        ActiveSurfaceHost
  UniversalInspector
  ActivityBar
  OverlayLayer
```

`WindowContextLens` MAY be visually integrated into the Unified Toolbar or Context Control, but its semantics remain distinct from active surface authority.

## 6.2 Work Surface contract

Add a Cockpit-owned presentation contract:

```ts
export interface WorkSurfaceV1 {
  schema: "uaiengine.cockpit.work_surface.v1";

  surface_id: string;
  window_id: string;
  pane_id: string;

  workspace_id: string;
  object_ref: string;
  object_kind: CockpitWorkObject["kind"] | "browser_target" | "workstream_projection";

  scope_binding: CockpitScopeBinding;
  attachment_ref?: FocusaAttachmentRef;
  runtime_ref?: CockpitRuntimeRef;

  presentation: {
    generated_label: string;
    label_override?: string;
    label_override_locked: boolean;
    icon_ref?: string;
    pinned: boolean;
    preview: boolean;
    protected_from_bulk_close: boolean;
  };

  state:
    | "loading"
    | "ready"
    | "running"
    | "waiting"
    | "attention"
    | "blocked"
    | "suspended"
    | "crashed";

  local_view_state_ref?: string;
  opened_at: string;
  last_activated_at: string;

  restore: {
    policy: "live" | "lazy" | "snapshot";
    restore_ref?: string;
  };
}
```

## 6.3 Runtime reference

A runtime reference MAY contain:

```ts
export interface CockpitRuntimeRef {
  node_id?: string;
  uiai_session_id?: string;
  browser_context_id?: string;
  target_id?: string;
  document_id?: string;
  navigation_id?: string;
  frame_id?: string;
}
```

A runtime reference is not a ScopeBinding. It identifies where current execution occurs.

## 6.4 Work Surface state

Each Work Surface SHALL maintain independent presentation state where applicable:

- scroll position;
- zoom;
- selected region;
- document page;
- table sorting and visible columns;
- active subview;
- local filter/query;
- Inspector tab selection for that object type;
- back/forward view history;
- browser address and load state projection;
- expanded/collapsed local disclosures.

Local view state SHALL be bounded and versioned. It MUST NOT contain secrets, raw credentials, unredacted private values, or canonical cognition.

---

# 7. Tab system

## 7.1 Core behavior

Tabs represent open Work Surfaces, not product categories.

Required actions:

- open;
- activate;
- close view;
- reopen recently closed;
- duplicate presentation;
- pin and unpin;
- protect and unprotect from bulk close;
- rename presentation label;
- restore generated label;
- move to beginning or end;
- move left or right;
- move to another pane;
- move to another window;
- open as split;
- copy object reference;
- copy URL where applicable;
- reveal object in owning workspace;
- open Inspector;
- close other unprotected surfaces;
- close surfaces to the right;
- close unpinned surfaces;
- suspend or discard presentation cache when eligible.

Approximately six tabs SHOULD remain directly visible at ordinary widths. Overflow MUST preserve keyboard and pointer access to every tab.

## 7.2 Rename semantics

Inline rename MAY be invoked by double-click, keyboard shortcut, or menu.

Renaming a tab updates only `presentation.label_override`.

It does not rename:

- the Workstream;
- the Workpoint;
- the browser target;
- the document;
- the test run;
- the Work Object.

Canonical rename commands, when supported, SHALL be separately labeled and routed through the owning controller.

An AI MAY propose a tab label. A user-authored label remains locked until the user explicitly restores automatic naming.

## 7.3 Preview tabs

A collection selection MAY open a temporary preview tab. A preview becomes permanent when the user:

- edits or mutates through it;
- pins it;
- renames it;
- moves it;
- explicitly keeps it;
- opens another preview while preservation rules require the first to remain.

Preview behavior MUST NOT discard an active runtime or unsaved proposal.

## 7.4 Status indicators

A tab MAY expose bounded indicators for:

- loading;
- running work;
- waiting for approval;
- agent reading;
- agent controlling;
- operator takeover;
- verification running;
- blocked scope;
- error;
- unsaved proposal;
- audio or recording;
- download;
- disconnected or suspended runtime.

A tab receives at most one dominant status treatment. Accessible text summarizes combined state. Color alone is prohibited.

## 7.5 Switching and discovery

Required switching surfaces:

- pointer activation;
- keyboard next/previous tab;
- most-recently-used switcher;
- last-tab toggle;
- numbered pinned-tab shortcuts where configured;
- Command Palette search;
- semantic search over open Work Surfaces;
- filters by project, Workstream, workspace, object kind, state, domain, owner, or attention requirement.

Switching tabs changes only `active_surface_id` in the relevant pane.

---

# 8. Surface Groups

## 8.1 Group types

Cockpit SHALL support:

- manual Surface Groups;
- system-suggested groups requiring user acceptance;
- dynamic groups derived from explicit external state;
- saved presentation groups inside Layout Snapshots.

Examples:

- Checkout investigation;
- OAuth research;
- Release verification;
- Current Workpoint surfaces;
- Active test suite;
- Evidence requiring review.

## 8.2 Group laws

- Group membership is presentation-only.
- A Surface Group may contain surfaces from more than one Workstream only when the UI labels each scope and treats aggregate AI/mutation behavior safely.
- The system MUST NOT silently reorganize manual groups.
- Dynamic group disappearance does not close protected, running, dirty, or explicitly retained surfaces.
- Group labels may be user-authored, generated, or AI-proposed. User-authored names remain locked.
- Surface Group color is presentation metadata and not a status or authority signal.

## 8.3 No `Workset` canonical term

The term `Workset` SHALL NOT be introduced as a Cockpit data object because it is easily confused with Workstream. Use `SurfaceGroup`, `PaneLayout`, and `LayoutSnapshot`.

---

# 9. Split panes

## 9.1 Pane tree

The layout SHALL use a recursive tree:

```ts
export type PaneNodeV1 =
  | {
      type: "leaf";
      pane_id: string;
      surface_ids: string[];
      active_surface_id?: string;
    }
  | {
      type: "split";
      split_id: string;
      direction: "row" | "column";
      ratio: number;
      first: PaneNodeV1;
      second: PaneNodeV1;
    };
```

## 9.2 Required operations

- split left;
- split right;
- split above;
- split below;
- open link or object as split;
- drag a surface to an edge drop target;
- resize using a splitter;
- move active surface to another pane;
- swap panes;
- rotate split direction;
- equalize sibling panes;
- temporarily maximize one pane;
- restore prior pane arrangement;
- merge panes;
- close an empty pane;
- close a pane while preserving or explicitly closing its surfaces;
- save layout as a Layout Snapshot.

## 9.3 Pane independence

Each pane has an independent active Work Surface. Browser, document, test, and report surfaces retain their own local view state.

The Inspector follows the keyboard-active Work Surface unless pinned to an explicit comparison or multi-selection mode.

## 9.4 Same-object splits

Two Work Surfaces may present the same Work Object, including:

- different pages of one document;
- two regions of one image;
- live and semantic views of one browser target;
- report and source Evidence;
- baseline and current visual comparison.

Creating a second presentation does not automatically clone the runtime or canonical object.

## 9.5 Linked behavior

Linked scrolling, synchronized navigation, shared zoom, or comparison cursors MUST be opt-in, visibly indicated, and reversible. They remain presentation features unless a separately named canonical action is invoked.

---

# 10. Multiple windows and Picture-in-Picture

## 10.1 Window model

A window owns:

- one `window_id`;
- one Pane Layout tree;
- local sidebar/Inspector state;
- one Window Context Lens;
- window geometry and display identity;
- a collection of Work Surfaces.

A window does not own a canonical Workstream.

## 10.2 Cross-window movement

Dragging or commanding a surface into another window changes `window_id` and pane membership only.

Dropping a surface onto empty desktop space MAY create a new Tauri window where platform support and accessibility requirements are satisfied.

## 10.3 Mixed-scope windows

One window MAY display several Workstreams or Projects simultaneously. Toolbar mutation commands SHALL bind to the active Work Surface, not the Window Context Lens.

## 10.4 Picture-in-Picture

A PiP surface MAY host:

- one Live browser target;
- one active test run;
- one recording or verification status;
- minimal pause, takeover, and restore controls.

PiP is a Work Surface projection. It does not receive independent authority or a duplicate canonical Session.

---

# 11. Drag-and-drop

## 11.1 Required drop targets

A Work Surface tab MAY be dropped onto:

- another position in its tab strip;
- another pane's tab strip;
- left, right, top, or bottom viewport edges to create a split;
- an existing pane center to join that pane;
- a Surface Group;
- another Cockpit window;
- empty desktop space to create a window where supported;
- Favorites to create a presentation shortcut;
- an Assistant context target to add a context reference without moving the surface.

## 11.2 Semantic drop targets

Dropping a Work Object onto a Workstream, Workpoint, Evidence target, or other canonical object MUST NOT perform a presentation move and a canonical mutation as one ambiguous operation.

The UI MAY offer a separately named semantic command after validation, such as:

- Link as supporting work;
- Attach Evidence candidate;
- Open in Workstream;
- Propose transfer;
- Fork into new Workstream.

The command SHALL show source and destination scope, authority, consequence, and proof posture.

## 11.3 DnD adapter

Add a Cockpit-owned adapter boundary:

```ts
export interface WorkSurfaceDndAdapter {
  beginDrag(surfaceId: string): void;
  previewDrop(target: WorkSurfaceDropTarget): WorkSurfaceDropDecision;
  commitDrop(target: WorkSurfaceDropTarget): Promise<void>;
  cancelDrag(): void;
}
```

Components SHALL not import third-party DnD contracts throughout the shell.

Pointer Events or an equivalent Cockpit-owned input layer SHOULD be preferred over browser-native HTML5 DnD so behavior remains controllable across Tauri platforms and native webview constraints.

## 11.4 Keyboard alternatives

Every DnD operation SHALL have menu and keyboard alternatives. At minimum:

- Move left/right;
- Move to pane;
- Split left/right/above/below;
- Move to window;
- Move to group;
- Move to beginning/end;
- Pin/unpin;
- Cancel.

Announcements SHALL name the surface, destination pane/window/group, position, and completion without flooding assistive technology.

---

# 12. Browser runtime binding

## 12.1 Ownership boundary

UIAI Engine owns:

- browser process;
- browser session;
- browser context/container;
- target/tab;
- page, document, navigation, and frame identity;
- observations and stale-state detection;
- actions, diagnostics, artifacts, verification, recording, and browser Evidence.

Cockpit owns presentation and intent.

A Cockpit tab and a UIAI browser target SHALL be bound but never conflated.

## 12.2 Target binding

A browser Work Surface SHALL carry:

- exact Workstream or Attachment binding;
- UIAI session identity;
- browser context identity;
- target identity;
- current document/navigation identity where available;
- node and route identity;
- control ownership and lease state;
- observation freshness;
- agent/operator control state.

## 12.3 Open disposition

Add one explicit disposition contract:

```ts
export type SurfaceOpenDisposition =
  | "current"
  | "new_foreground"
  | "new_background"
  | "new_group"
  | "split"
  | "new_window"
  | "external_browser";
```

It governs:

- command-click and middle-click;
- agent-opened links;
- popups and `target=_blank`;
- authentication windows;
- downloads;
- external protocols;
- Help/documentation links;
- Work Object open commands.

Agent-initiated surface creation SHALL be budgeted and policy-bound. An agent cannot create unbounded foreground tabs or windows.

## 12.4 Canonical and direct-webview surfaces

The default browser surface SHOULD remain a projection of the canonical UIAI runtime:

```text
Cockpit Work Surface
  → BrowserSurfaceAdapter
    → UIAI session/context/target
      → FPV frames and semantic observations
      → operator input
      → UIAI action, verification, and Evidence path
```

An optional direct Tauri child-webview surface MAY be implemented only behind the same adapter and a separately reviewed security posture. Direct remote-content webviews MUST NOT receive privileged Cockpit IPC by default.

---

# 13. Command routing and true scoping

## 13.1 Intent path

All Work Surface commands follow:

```text
Work Surface intent
  → resolve exact scope binding
  → resolve Attachment or direct authority grant
  → ScopeGuard / AuthorityGuard / ConsentGuard
  → owning controller
  → NodeRouter / TransportRouter
  → UIAI or Focusa adapter
  → result normalizer
  → scoped Event / Artifact / Evidence / Receipt
  → Work Surface view model
```

Visual components SHALL NOT call raw Focusa, UIAI, Cloud, AI API, or Tauri execution endpoints directly.

## 13.2 Multi-selection

Multi-selection across Workstreams or Projects is read-only by default.

A command that supports cross-scope application SHALL:

1. enumerate target surfaces and canonical objects;
2. group them by exact Workstream;
3. resolve authority independently for every group;
4. present the decomposed plan;
5. execute or reject each sub-operation independently;
6. return a compound summary without inventing one canonical cross-Workstream mutation.

## 13.3 No singleton fallback

No Cockpit controller, store, adapter, or Tauri command may use:

- `activeWorkstream` as process-global authority;
- `lastWorkstream` as mutation fallback;
- one global active Workpoint;
- one global active Trajectory;
- one global active browser target;
- one global current task;
- one window-selected Workstream as authority for all panes;
- a Session's most recent attachment when an operation lacks an Attachment ref.

Convenience selectors may exist as presentation queries only and must return explicit scope-bearing references.

---

# 14. Assistant and agent context

## 14.1 Explicit context references

The Assistant SHALL support explicit bounded context references such as:

```text
@this-surface
@this-pane
@selected-surfaces
@surface-group
@this-workstream
@project-aggregate
@current-workpoint
@selection
@inspector
@recent-activity
```

## 14.2 Context receipt

Before sending or executing, Cockpit SHALL show or make inspectable a context receipt:

```text
Context

1. Checkout browser
   Project Alpha / Checkout Workstream
   Read + control permitted

2. Payment specification
   Project Alpha / Checkout Workstream
   Read permitted

3. Authentication test run
   Project Alpha / Authentication Workstream
   Read only

Excluded
All other open surfaces
```

The receipt SHALL identify:

- source surface/object;
- Project and Workstream;
- data classification where material;
- read/control/mutation posture;
- untrusted browser-content posture;
- exclusions;
- egress destination when content leaves the local machine;
- reasons for blocked sources.

## 14.3 Cross-Workstream assistant behavior

The Assistant may reason over explicitly selected read context across Workstreams. It may not convert that context into one ambient mutation scope.

Mutation proposals SHALL be emitted as one or more scope-explicit intents. Each receives independent authority and approval handling.

## 14.4 Control state

Browser and executable surfaces SHALL make these states unmistakable:

- Viewing;
- Agent reading;
- Agent controlling;
- Waiting for approval;
- Operator takeover;
- Paused;
- Verification running;
- Scope blocked;
- Disconnected.

---

# 15. Motion and transitions

## 15.1 Purpose

Motion explains relationship, hierarchy, state, and causality. It is presentation-only and SHALL NOT imply that canonical scope changed.

## 15.2 Transition taxonomy

| Interaction | Target behavior | Duration |
|---|---|---:|
| Hover/focus/state | color/opacity | 80–120ms |
| Tab switch | restrained crossfade | 90–140ms |
| Tab reorder | FLIP/transform movement | 160–220ms |
| Workspace change | subtle shared-axis transition | 220–280ms |
| Open Work Object | forward relationship transition | 180–240ms |
| Back navigation | reversed relationship transition | 180–240ms |
| Split or merge pane | FLIP/transform layout transition | 180–240ms |
| Sidebar/Inspector | transform/opacity | 180–240ms |
| Sheet/dialog | restrained movement + opacity | 180–260ms |
| Restore Layout Snapshot | coordinated layout reveal | 240–300ms |

Live browser frames, videos, document pages, logs, and rapidly updating tables SHALL NOT receive decorative page transitions.

## 15.3 Transition orchestrator

Add a shared transition contract:

```ts
export type NavigationIntent =
  | "tab_switch"
  | "workspace_change"
  | "open_object"
  | "back"
  | "forward"
  | "restore"
  | "split"
  | "merge"
  | "move_window";

export interface TransitionRequest {
  intent: NavigationIntent;
  source_surface_id?: string;
  destination_surface_id?: string;
  direction?: "forward" | "backward" | "neutral";
  reduced_motion: boolean;
}
```

Routes and components emit intent. They do not invent local timings or easing curves.

## 15.4 Motion laws

- The shell remains mounted during route and Work Surface changes.
- Previous content remains until incoming content has a renderable state.
- No blank-window flash during switching, closing, or restoration.
- Motion is interruptible and does not queue stale transitions.
- Controls become interactive when visually available.
- Prefer opacity and transform.
- Layout-property animation requires performance proof.
- Directional movement is reserved for true hierarchy.
- Repeated tab switching uses a crossfade rather than dramatic sliding.
- Focus follows the completed logical navigation, not animation duration.
- Scroll and view state are preserved per Work Surface.

## 15.5 Reduced motion

Reduced-motion mode SHALL:

- remove spatial movement and zooming;
- replace transitions with an opacity change of 0–100ms or no transition;
- disable smooth scrolling and decorative interpolation;
- preserve immediate state feedback;
- remain covered by visual and E2E tests.

---

# 16. Persistence, restoration, and crash recovery

## 16.1 Persistence boundary

Cockpit may persist presentation state:

- windows and display placement;
- Pane Layout trees;
- Work Surface order and membership;
- Surface Groups;
- pinned and protected surfaces;
- local view-state refs;
- sidebar and Inspector state;
- transition/reduced-motion preferences;
- Layout Snapshots;
- recently closed views.

Cockpit SHALL NOT persist a local substitute for Focusa canonical Workstream state.

## 16.2 Restore policies

- `live`: reconnect to the exact runtime when permitted and available.
- `lazy`: restore the tab shell and connect only on activation.
- `snapshot`: restore a read-only artifact or bounded last-known view when live runtime cannot be resumed.

Restore failure SHALL preserve the surface with an exact recovery state rather than silently binding another runtime or Workstream.

## 16.3 Scope validation on restore

Before a restored surface becomes executable, Cockpit SHALL verify:

- Project identity;
- Workstream identity;
- Attachment validity where required;
- continuity lineage where required;
- node/route availability;
- runtime identity and freshness;
- entitlement;
- authority and control ownership.

A stale, missing, migrated, quarantined, or contradictory binding becomes blocked/read-only with a recovery action.

## 16.4 Recently closed

Recently closed entries MAY restore:

- one Work Surface;
- one Surface Group;
- one Pane Layout branch;
- one window;
- one Layout Snapshot.

Reopening restores presentation. It does not revive a terminated runtime without an explicit supported restore operation.

---

# 17. Tauri architecture and security

## 17.1 Tauri responsibilities

Tauri is appropriate for:

- native windows;
- application and context menus;
- global and local shortcuts;
- file and directory dialogs;
- window-state persistence;
- notifications;
- updater integration;
- secure local storage;
- deep links;
- child webviews where explicitly approved;
- operating-system drag/window integration where reliable.

## 17.2 Recommended plugin posture

Candidate official plugins include:

- window-state;
- store or SQL for bounded presentation persistence;
- global-shortcut;
- dialog;
- notification;
- deep-link;
- updater;
- Stronghold or an approved secret store where secrets are required.

Dependency adoption requires license, maintenance, platform, security, bundle-size, test, and failure-mode review.

## 17.3 Remote-content isolation

Any remote-content child webview SHALL:

- have no privileged Cockpit IPC by default;
- use a narrowly scoped Tauri capability;
- have explicit profile/data-store identity;
- intercept navigation and popup disposition;
- implement download, permission, external-protocol, and authentication-window policy;
- expose crash/recovery state;
- avoid direct Focusa or UIAI canonical mutation;
- retain content provenance and untrusted-content posture;
- pass platform-specific security and isolation tests.

Overlapping Tauri capabilities SHALL be audited because their permissions compose.

## 17.4 Browser consistency

Tauri's platform webviews do not create one consistent Chromium browser runtime across macOS, Windows, and Linux. The canonical UIAI browser runtime remains the cross-platform source for agent execution, observation, verification, recording, replay, and Evidence unless a separate accepted browser-runtime decision supersedes it.

---

# 18. Accessibility and keyboard contract

## 18.1 Tabs

Tab strips SHALL follow an accessible tabs pattern when they switch one active surface in one pane. Overflow, close, pin, rename, status, and context-menu controls must remain keyboard accessible without breaking tab semantics.

## 18.2 Splitters

Splitters SHALL follow an accessible window-splitter/separator pattern:

- keyboard resize;
- current value/position semantics;
- visible focus;
- minimum/maximum constraints;
- reset/equalize commands;
- no precision-drag-only behavior.

## 18.3 Windows and overlays

- Focus returns logically after closing tabs, panes, windows, menus, sheets, or dialogs.
- Cross-window moves announce the destination.
- Modal content uses one shared overlay root per window.
- No modal-on-modal workflow.
- Constrained windows retain access through overlays and Command Palette.

## 18.4 DnD alternatives

Every drag action has keyboard and menu equivalents. Invalid targets expose the reason through accessible text.

## 18.5 Zoom, text, and internationalization

The shell, tabs, groups, panes, menus, scope receipts, and transitions SHALL pass:

- 200% zoom;
- operating-system text enlargement;
- localization expansion;
- RTL logical layout;
- high contrast;
- light and dark mode;
- comfortable and compact density;
- reduced motion.

---

# 19. Migration and compatibility

## 19.1 Scope migration principle

The migration MUST NOT merely rename `continuity_id` to `workstream_id`.

Legacy records require evidence-based mapping:

```text
legacy project root + continuity/thread/session records
  → inspect rollover and transfer lineage
  → inspect Attachments, Workpoints, Trajectories, and events
  → establish stable WorkstreamId
  → retain ContinuityIds as child lineages
  → emit migration result and proof
```

## 19.2 Migration laws

- Never merge Workstreams based on similar title, goal, transcript, Workpoint, or Trajectory.
- Verified rollover lineage may show that multiple ContinuityIds belong to one Workstream.
- Project mismatch blocks migration.
- Missing or contradictory ownership enters quarantine.
- Quarantined records remain inspectable and noncanonical.
- Legacy `thread_id` is an alias/migration input only after exact resolution.
- Global active/current/latest pointers are hints, not authority.
- Every migrated canonical object receives explicit Project and Workstream identity.
- Every migration decision emits durable evidence and a reversible mapping record where feasible.

## 19.3 Cockpit compatibility adapter

During migration, Cockpit MAY use a compatibility adapter that returns:

```ts
export type LegacyScopeResolution =
  | { status: "resolved"; workstream: FocusaWorkstreamRef; evidence_refs: string[] }
  | { status: "ambiguous"; candidates: FocusaWorkstreamRef[]; recovery: string }
  | { status: "quarantined"; reason: string; recovery: string }
  | { status: "unsupported"; reason: string };
```

Only `resolved` may proceed to canonical mutation after current authority checks.

---

# 20. Component and store architecture

## 20.1 Required Cockpit modules

Recommended modules:

```text
apps/cockpit/src/lib/workbench/
  contracts/
    work-surface.ts
    pane-layout.ts
    surface-group.ts
    layout-snapshot.ts
    scope-binding.ts
    open-disposition.ts
    transition.ts
  state/
    workbench-store.ts
    window-store.ts
    surface-registry.ts
    recently-closed-store.ts
  controllers/
    surface-controller.ts
    pane-controller.ts
    window-controller.ts
    command-scope-controller.ts
    restore-controller.ts
    transition-controller.ts
  dnd/
    work-surface-dnd-adapter.ts
  adapters/
    browser-surface-adapter.ts
    focusa-contract-adapter.ts
    persistence-adapter.ts
```

Exact paths MAY change, but responsibilities and boundaries SHALL remain explicit.

## 20.2 Workbench store

The Workbench Store owns presentation state only. It may hold:

- windows;
- panes;
- surfaces;
- groups;
- local selections;
- restoration metadata;
- presentation preferences.

It SHALL NOT hold canonical Workstream cognition or synthesize authority.

## 20.3 No god component

`CockpitShell`, `WorkSurfaceRegion`, `SurfaceTabStrip`, `PaneHost`, and workspace viewers SHALL remain composable. One component must not own routes, DnD, authority, persistence, browser execution, Inspector content, and transition policy together.

## 20.4 Event naming

Presentation events SHOULD use explicit names:

- `surface_opened`;
- `surface_activated`;
- `surface_closed`;
- `surface_moved`;
- `surface_label_overridden`;
- `surface_group_changed`;
- `pane_split`;
- `pane_merged`;
- `window_created`;
- `layout_snapshot_saved`;
- `layout_snapshot_restored`.

These events are local presentation history unless explicitly promoted through an accepted user-settings sync path. They are not Focusa canonical cognitive events.

---

# 21. Implementation tasks

## T004-00 — Upstream contract readiness gate

**Work:**

- identify the accepted Focusa Workstream-scoped runtime specification;
- pin the generated contract bundle by commit SHA or release digest;
- verify distinct Workstream and Continuity identities;
- verify generated scope, Attachment, authority, Workpoint, Trajectory, Work Loop, and Evidence refs;
- document blocked or provisional fields.

**Acceptance:** no canonical Cockpit implementation relies on continuity-as-Workstream or a hand-maintained duplicate Focusa authority DTO.

## T004-01 — Scope contract migration

**Depends on:** T004-00  
**Work:**

- add generated Focusa contract adapter;
- add discriminated `CockpitScopeBinding`;
- deprecate loose `ScopeRef` for new canonical paths;
- remove `thread_id` authority use;
- add project aggregate and system bindings;
- add exact scope validation and human-readable formatting.

**Acceptance:** missing or ambiguous Workstream blocks canonical mutation; project aggregates remain advisory/read-only.

## T004-02 — Work Surface contracts and registry

**Depends on:** T004-01  
**Work:**

- add `WorkSurfaceV1`, runtime ref, open disposition, and local view-state contracts;
- add one Work Surface registry;
- add stable IDs and schema migration;
- define object-kind/viewer mappings;
- define close-view versus runtime lifecycle commands.

**Acceptance:** all open objects are represented by Work Surfaces; no tab object owns canonical work.

## T004-03 — Pane Layout contracts and store

**Depends on:** T004-02  
**Work:**

- add recursive pane tree;
- add invariant validation;
- add active surface per pane;
- add split ratios and constraints;
- add maximize/restore and merge history.

**Acceptance:** deterministic pane operations preserve all retained surfaces and scope bindings.

## T004-04 — Window and Layout Snapshot contracts

**Depends on:** T004-02, T004-03  
**Work:**

- add window registry;
- add Window Context Lens;
- add Layout Snapshot schema;
- add display/window geometry persistence;
- add platform-safe restoration and missing-display recovery.

**Acceptance:** one window may safely display several Workstreams and Projects without ambient authority.

## T004-05 — Shared tab primitives

**Depends on:** T004-02  
**Work:**

- implement tab strip, overflow, close, pin, protect, rename, generated-label restore, status, preview, and context menu;
- implement pointer and keyboard switching;
- implement MRU switcher and Command Palette integration;
- add long-label, icon, loading, error, blocked, running, and agent-control states.

**Acceptance:** all required tab actions work without mutating canonical object identity.

## T004-06 — Split-pane interaction

**Depends on:** T004-03, T004-05  
**Work:**

- implement split commands and edge targets;
- implement accessible splitter;
- implement swap, rotate, equalize, maximize, merge, and close-pane flows;
- preserve focus and scroll/view state;
- add constrained-window behavior.

**Acceptance:** pane manipulation has pointer and keyboard paths and remains presentation-only.

## T004-07 — Work Surface DnD

**Depends on:** T004-03, T004-04, T004-05  
**Work:**

- implement Cockpit DnD adapter;
- reorder tabs;
- move across panes/windows;
- drag to split;
- create window where supported;
- add Favorites and Assistant-context drop targets;
- add semantic-drop decision boundary;
- implement cancel and persistence rollback.

**Acceptance:** DnD cannot mutate Project, Workstream, Continuity, Attachment, Workpoint, Evidence, Session, or runtime ownership.

## T004-08 — Surface Groups

**Depends on:** T004-05, T004-07  
**Work:**

- implement manual groups;
- implement generated/user labels and locking;
- implement accepted suggested groups;
- implement dynamic groups from explicit state;
- add group collapse, move, save, and restore;
- prove mixed-scope labeling and safe behavior.

**Acceptance:** no group is called or treated as a Workstream or canonical work container.

## T004-09 — BrowserSurfaceAdapter

**Depends on:** T004-01, T004-02  
**Work:**

- bind browser surfaces to UIAI session/context/target/document/navigation refs;
- implement current open dispositions;
- implement back/forward/address/load projection;
- expose observation freshness and control ownership;
- separate close view, park target, and end Session;
- preserve agent-first browser contracts and Evidence.

**Acceptance:** Cockpit tab identity and browser target identity remain separate and traceable.

## T004-10 — Command Scope Controller

**Depends on:** T004-01, T004-02  
**Work:**

- resolve active surface target;
- create command scope receipts;
- block missing/ambiguous scope;
- decompose supported multi-selection commands by Workstream;
- add independent authority/result handling;
- remove singleton/last-active fallback.

**Acceptance:** every canonical mutation has exact Project and Workstream proof.

## T004-11 — Assistant context receipts

**Depends on:** T004-01, T004-02, T004-10  
**Work:**

- implement explicit context references;
- render context receipts;
- show read/control/mutation and egress posture;
- preserve browser content provenance;
- decompose mutation proposals by Workstream;
- add blocked/excluded context explanations.

**Acceptance:** assistant context never creates ambient cross-Workstream mutation authority.

## T004-12 — Transition orchestrator

**Depends on:** T004-03, T004-05, T004-06  
**Work:**

- add shared transition intent contract;
- implement persistent shell and keyed hosts;
- implement tab, workspace, object, split, merge, overlay, and restore transitions;
- prevent blank flashes;
- implement interruption and reduced motion;
- add performance fixtures.

**Acceptance:** transitions are smooth, interruptible, tokenized, accessible, and presentation-only.

## T004-13 — Persistence and crash recovery

**Depends on:** T004-02 through T004-12  
**Work:**

- persist bounded presentation state;
- implement live/lazy/snapshot restore;
- validate scope and runtime on restore;
- implement recently closed surfaces/groups/panes/windows;
- implement stale/quarantined/migrated recovery states;
- prevent canonical fallback during restore.

**Acceptance:** crash recovery restores layout without borrowing another Workstream or reviving terminated runtimes implicitly.

## T004-14 — Tauri window, menu, and plugin integration

**Depends on:** T004-04, T004-07, T004-13  
**Work:**

- implement native window manager;
- implement File/Edit/View/Work/Window/Help commands;
- add shortcuts and native dialogs;
- integrate approved persistence/window-state plugins;
- add cross-window focus and display recovery;
- review child-webview capability isolation.

**Acceptance:** macOS, Windows, and Linux window behavior passes platform smoke; remote content has no unintended privilege.

## T004-15 — Accessibility and internationalization

**Depends on:** T004-05 through T004-14  
**Work:**

- implement accessible tabs, splitters, menus, DnD alternatives, announcements, and focus return;
- convert strings to message keys;
- add RTL logical layout;
- test 200% zoom, text enlargement, high contrast, dark mode, density, and reduced motion.

**Acceptance:** no operation is drag-only, hover-only, color-only, or precision-pointer-only.

## T004-16 — Migration adapter and quarantine UX

**Depends on:** T004-00, T004-01  
**Work:**

- consume Focusa legacy-scope resolution results;
- handle resolved, ambiguous, quarantined, and unsupported states;
- remove thread authority from Cockpit;
- display mapping evidence and recovery;
- keep quarantined data inspectable and noncanonical.

**Acceptance:** Cockpit never guesses a Workstream from continuity, thread, session, title, or similarity.

## T004-17 — Proof, performance, and release gates

**Depends on:** all prior tasks  
**Work:**

- unit tests for scope bindings, Work Surface invariants, pane tree, groups, DnD decisions, restoration, and transitions;
- integration tests for browser binding, command receipts, assistant context, multi-selection decomposition, and migration states;
- multi-project/multi-Workstream isolation tests;
- visual tests for default/dark/constrained/error/blocked/quarantined/running/control/reduced-motion states;
- performance tests for representative tab/pane/window counts;
- update `cockpit:smoke`;
- attach screenshots/video to user-facing changes.

**Acceptance:** all unified criteria in Section 25 pass as CI-blocking gates.

---

# 22. Dependency summary

```text
T004-00
  → T004-01
      → T004-02
          → T004-03
              → T004-04
              → T004-06
          → T004-05
              → T004-06
              → T004-07
              → T004-08
          → T004-09
          → T004-10
              → T004-11

T004-03 + T004-05 + T004-06
  → T004-12

T004-02 through T004-12
  → T004-13

T004-04 + T004-07 + T004-13
  → T004-14

T004-05 through T004-14
  → T004-15

T004-00 + T004-01
  → T004-16

All prior tasks
  → T004-17
```

`T003-05` shell replacement SHALL reserve the Work Surface region, but final tab/pane/window behavior is governed by this amendment. `T003-11` route skeletons and `T003-13` Live integration SHALL consume `T004` contracts rather than introducing local tab or pane state.

---

# 23. Required proof matrix

## 23.1 Foundational isolation scenario

The release proof SHALL demonstrate concurrently:

```text
Project A / Workstream 1 in pane 1
Project A / Workstream 2 in pane 2
Project B / Workstream 3 in another window
```

Each has independent:

- Focus Stack projection;
- active Workpoint projection;
- Trajectory projection;
- Work Loop projection;
- Attachments and runtimes;
- events and snapshots;
- browser/document/test surfaces;
- authority and control state.

The operator then:

- switches tabs rapidly;
- changes keyboard focus;
- drags tabs;
- creates and merges splits;
- moves a surface to another window;
- restores a Layout Snapshot;
- selects cross-Workstream read context for the Assistant;
- attempts same- and cross-Workstream mutations;
- closes views without ending runtimes;
- ends one runtime explicitly.

Proof must show no operation:

- borrows another Workstream's authority;
- changes canonical scope through visual focus;
- reads a daemon-global active/current/latest pointer;
- silently merges continuity;
- leaks state across Projects or Workstreams;
- terminates runtime through close-view;
- mutates canonical work through presentation DnD.

## 23.2 Required test classes

- same Project, different Workstreams;
- different Projects, same ContinuityId string;
- one Workstream, several continuities;
- one Workstream, several Sessions and Attachments;
- one Session attached to several Workstreams;
- several Work Surfaces presenting one Work Object;
- browser target navigation/document replacement;
- stale Attachment;
- revoked authority;
- missing node;
- migrated scope;
- ambiguous legacy thread;
- quarantined legacy record;
- terminated runtime with restorable snapshot;
- cross-window DnD;
- DnD persistence failure rollback;
- reduced motion;
- keyboard-only split and move;
- 200% zoom;
- multi-display removal and restoration.

## 23.3 Performance posture

Representative tests SHALL cover:

- 30 open Work Surfaces;
- 8 Surface Groups;
- 4 panes;
- 3 windows;
- 5 simultaneously updating surfaces;
- large overflow search;
- rapid MRU switching;
- restore after unclean shutdown.

No limit above is a product entitlement or hard runtime cap. It is a minimum proof fixture.

---

# 24. Prohibited patterns

Implementation MUST NOT introduce:

- continuity-as-Workstream identity in new contracts;
- `thread_id` as parallel Workstream authority;
- Session identity as project or Workstream authority;
- one process-global active Workstream used for mutation;
- one window-global Workstream authority applied to all panes;
- one global active browser target;
- tab selection that updates Focusa active Workpoint or Focus Stack;
- tab rename that renames the canonical object implicitly;
- close tab that ends a Session or browser target implicitly;
- DnD that changes canonical ownership or scope;
- Surface Groups called Workstreams or Worksets;
- layout restoration that silently binds a different Workstream/runtime;
- multi-selection mutation without Workstream decomposition;
- Assistant context that silently includes every open tab;
- AI-generated labels that overwrite user-locked labels;
- raw remote content with privileged Tauri IPC;
- direct API calls from visual Svelte components;
- browser-native HTML5 DnD dependency spread across components;
- decorative page animation over live browser/video/document/log content;
- blank-window flashes during navigation;
- queued stale animations;
- drag-only or hover-only actions;
- hand-maintained Focusa authority DTOs after generated contracts are available;
- fake production projects, Workstreams, tabs, counts, or sessions.

---

# 25. Unified acceptance criteria

This amendment is complete when:

1. Cockpit consumes a Focusa contract with distinct Workstream and Continuity identities.
2. The loose legacy `ScopeRef` is not used for new canonical mutation paths.
3. `thread_id` is removed as Cockpit Workstream authority.
4. Every executable Work Surface has an exact Workstream or Attachment binding.
5. Project aggregate and system surfaces cannot silently mutate a Workstream.
6. One window can display multiple Workstreams and Projects safely.
7. Each pane owns an independent active Work Surface.
8. Work Surface tabs open, close, reopen, duplicate, pin, protect, rename, overflow, search, and switch by pointer and keyboard.
9. Tab labels remain presentation metadata unless a separate canonical rename is invoked.
10. Surface Groups remain presentation-only and are never named or treated as Workstreams.
11. Split panes support all required commands, resizing, keyboard control, maximize, merge, and restoration.
12. Surfaces move across panes and windows without changing canonical scope.
13. Drag-to-split and cross-window DnD have accessible alternatives.
14. Semantic drop operations are separate explicit governed commands.
15. Browser Work Surfaces bind to exact UIAI session/context/target identities without conflating tab and target identity.
16. Close view is distinct from park target, end Session, stop run, close Workstream, archive, and delete.
17. Every meaningful mutation resolves an exact command scope receipt.
18. No daemon-global or UI-global active/current/latest fallback participates in canonical mutation.
19. Supported cross-Workstream commands are decomposed into independently authorized operations.
20. Assistant context is explicit, inspectable, scoped, provenance-aware, and exclusion-aware.
21. Assistant mutation proposals remain Workstream-explicit.
22. Tab, object, workspace, split, merge, overlay, and restore transitions follow the shared orchestrator.
23. Transitions are interruptible, frame-stable, nonblank, and reduced-motion compliant.
24. Persistence stores presentation state only.
25. Restore validates Project, Workstream, Attachment, runtime, entitlement, and authority before execution.
26. Missing/stale/migrated/quarantined bindings fail safely with exact recovery.
27. Tauri windows, menus, shortcuts, persistence, and security pass platform tests.
28. Remote child webviews receive no unintended privileged IPC.
29. Tabs, splitters, DnD, windows, overlays, and commands pass keyboard and screen-reader review.
30. Light, dark, compact, comfortable, high contrast, RTL, localization expansion, 200% zoom, and reduced motion pass.
31. Foundational three-Workstream isolation proof passes without state bleed.
32. No duplicate work-surface, pane, window, route, authority, or scope store exists.
33. No fake production scope or work data ships.
34. `cockpit:smoke`, unit, integration, accessibility, visual, performance, migration, and isolation gates are CI-blocking.

---

# 26. Final implementation principle

> **Cockpit may display and arrange many kinds of work at once, but every canonical action remains bound to one explicit Project and Workstream. Tabs, panes, groups, windows, visual focus, and beautiful motion organize presentation; they never create, transfer, merge, or replace authority.**
