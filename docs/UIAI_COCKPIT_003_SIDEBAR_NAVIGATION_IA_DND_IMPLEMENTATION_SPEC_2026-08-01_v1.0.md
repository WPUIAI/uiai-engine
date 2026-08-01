# UIAI Cockpit Sidebar Navigation, Information Architecture, and Drag-and-Drop Implementation Specification

**Document number:** `UIAI-COCKPIT-003`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding numbered decisions/amendments:** `UIAI-COCKPIT-001`, `UIAI-COCKPIT-002`  
**Status:** Proposed normative implementation amendment  
**Version:** 1.0  
**Date:** 2026-08-01  
**Repository:** `WPUIAI/uiai-engine`  
**Primary implementation home:** `apps/cockpit/`  
**Primary implementation stack:** SvelteKit 2, Svelte 5, Tauri v2, TypeScript  
**Scope:** Cockpit shell, primary sidebar, Context Control, workspace navigation, contextual collection panes, user ordering, pinning, visibility, accessibility, entitlement presentation, migration from Slice 0, test gates, and dependency-ordered implementation work

---

## 0. Decision

UIAI Cockpit SHALL replace the current Slice 0 primitive navigator with one manifest-backed, task-oriented, progressively disclosed navigation system.

The sidebar SHALL organize the real Cockpit workspaces defined by `UIAI-COCKPIT-000`:

```text
Work
  Overview
  Live
  Test Lab
  Documents
  Research

Create
  Studio
  Automations

Prove
  Evidence
  Activity

System
  Nodes & Services
  Capabilities

Footer
  Settings
  Help
```

The manifest order and groups define the recommended default layout. Users MAY reorder workspaces, pin workspaces or work objects, hide nonessential workspaces, and restore the recommended layout. Reordering is a presentation preference only. It MUST NOT modify Project, Workstream, Workpoint, ScopeRef, NodeRef, worker ownership, session leases, authority, evidence linkage, entitlement, or backend object ownership.

This amendment adds no new canonical mission store, browser runtime, evidence store, task authority, route family, or product plane. It implements and refines the existing Cockpit information architecture against the current codebase.

---

# 1. Authority and source precedence

Implementation SHALL apply sources in this order:

1. `UIAI-COCKPIT-000` unified master specification.
2. `UIAI-COCKPIT-001` Review Reports integration decision.
3. `UIAI-COCKPIT-002` agent-first browser amendment and companion ledger.
4. This `UIAI-COCKPIT-003` navigation implementation amendment.
5. August 1, 2026 UIAI entitlement and protected-worker documents for access, recovery, and feature-state presentation.
6. Current repository contracts, route mounts, parity matrices, smoke harnesses, and implementation code.

Primary repository sources:

- `apps/cockpit/src/routes/+layout.svelte`
- `apps/cockpit/src/routes/+page.svelte`
- `apps/cockpit/src/lib/cards/phase0-card-manifest.ts`
- `apps/cockpit/src/lib/contracts/card-manifest.ts`
- `apps/cockpit/src/lib/ui/design-tokens.css`
- `apps/cockpit/package.json`
- `apps/cockpit/smoke/smoke-runner.ts`
- `docs/UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md`
- `docs/cockpit/000/UIAI_COCKPIT_000_V01.md` through `V16.md`
- `docs/UIAI_COCKPIT_001_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md`
- `docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`
- `docs/contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml`
- `docs/PUBLIC_API_PARITY_MATRIX.md`
- `docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`
- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
- `docs/UIAI_ENTITLEMENT_IMPLEMENTATION_WORKBREAKDOWN_2026-08-01.md`
- `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`

No literal example project, document, session, count, limit, user, node, or Workpoint value from documentation SHALL be embedded as product data.

---

# 2. Current implementation audit

## 2.1 Current shell

The current `apps/cockpit/src/routes/+layout.svelte` is the July 1, 2026 Slice 0 shell. It currently contains:

- a static scope strip with independent chips;
- a three-column body;
- a hard-coded `Primitive navigator`;
- static rows for UIAI Engine, Focusa Local, Focusa Cloud, and AI API;
- a static viewport slot;
- a static Inspector placeholder;
- a static bottom process ribbon.

The current navigation has no route state, selected-workspace state, group state, hidden-workspace state, pinned items, resize state, compact mode, persisted preference, keyboard model, or drag-and-drop implementation.

## 2.2 Current root page

The current `apps/cockpit/src/routes/+page.svelte` renders the Phase 0 card manifest as a grid. It does not yet render the final Overview composition.

## 2.3 Current manifests

`phase0-card-manifest.ts` contains the real Phase 0 cards and contract mappings. `CardManifest` is a bounded card contract and SHALL remain supported, but cards SHALL NOT define the top-level navigation taxonomy.

## 2.4 Current dependencies

`apps/cockpit/package.json` currently contains no drag-and-drop or sortable-list dependency. DnD SHALL be added behind a Cockpit-owned adapter so product behavior is not coupled directly to a third-party component API.

## 2.5 Recent commit conclusion

After the Slice 0 shell, recent Cockpit-specific repository activity is predominantly:

- release and packaging work;
- product rename and path correction;
- numbered Cockpit documentation publication;
- agent-first browser and report specifications;
- entitlement and protected-worker specifications.

There is no later implemented Cockpit navigation surface that supersedes the current Slice 0 shell.

---

# 3. Product information architecture

## 3.1 User mental model

The navigation model SHALL preserve:

```text
Project
  Workstream
    Workpoint
      Work objects
```

Work objects include the kinds already defined by the master contracts:

- browser session;
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
- evidence;
- review report;
- report snapshot;
- Workpoint.

Backend products and technical planes support these objects and SHALL NOT replace them as ordinary navigation language.

## 3.2 Shell responsibility split

```text
Primary sidebar
  stable workspaces, saved views, Favorites, Settings, Help

Contextual collection pane
  sessions, test flows/runs, documents, captures, reports, nodes, or other objects in the active workspace

Workspace viewport
  selected object or workspace content

Universal Inspector
  Summary, Details, Scope, Evidence, History, diagnostics, Developer data

Activity Bar
  errors, running work, approvals, ownership conflicts, evidence attention, sync, entitlement warnings, updates
```

The primary sidebar MUST NOT become the object tree, Inspector, Activity feed, endpoint catalog, or service dashboard.

---

# 4. Stable shell layout

The target layout is:

```text
CockpitShell
  UnifiedToolbar
  PrimarySidebar
  WorkObjectTabs
  WorkspaceViewport
  UniversalInspector
  ActivityBar
```

Required behavior:

- sidebar, Inspector, and contextual collection pane resize independently within tokenized limits;
- Live, document, test, and Report Canvas content receive priority over chrome;
- the sidebar may be Expanded, Compact, Hidden, or Overlay;
- state is remembered per window when the available width permits restoration;
- closing a work-object tab closes the view only unless the command explicitly ends, cancels, or deletes backend work.

---

# 5. Primary sidebar anatomy

The expanded sidebar SHALL use this structure:

```text
Product header                     pinned
Context Control                    pinned
Resume Workpoint                   conditional pinned action
Search and commands                pinned action
Workspace section header/menu      pinned within navigation region
Scrollable workspace navigation    scrollable
Favorites                          scrollable, only when nonempty
Settings                           pinned footer
Help                               pinned footer
```

## 5.1 Product header

Contains:

- UIAI Cockpit identity;
- sidebar mode control;
- no backend-plane chips;
- no decorative health or entitlement badges.

## 5.2 Context Control

The current scope strip SHALL become one Context Control backed by actual resolved context.

Supported fields:

- Project;
- Workstream;
- Workpoint;
- session or thread where applicable;
- NodeRef;
- operating profile;
- role;
- authority state;
- sync state;
- route/transport in expanded disclosure.

The collapsed control SHALL present the most specific available human-readable context without raw IDs. Missing, stale, conflicting, or read-only context SHALL present the exact recovery action and SHALL block writes through the existing guard path.

No example names from documentation are allowed in fixtures except explicitly test-only fixture data stored outside production defaults.

## 5.3 Resume Workpoint

The existing `focusa.workpoint_resume` Phase 0 contract SHALL provide the conditional Resume Workpoint action.

Rules:

- shown only when a resumable Workpoint is available;
- opens the relevant Overview/Mission Deck or work object;
- does not create a parallel Workpoint state;
- unavailable or blocked state explains the exact scope, node, or authority recovery.

## 5.4 Search and commands

`Command-K` SHALL remain the global Find/Do surface defined by the master.

Find SHALL search readable scope for:

- workspaces;
- capabilities;
- browser sessions;
- test flows and runs;
- documents;
- research captures;
- artifacts and evidence;
- Workpoints and Trajectories;
- jobs;
- nodes;
- settings;
- Help.

Do SHALL expose bounded commands from registered capability and workspace manifests. It SHALL display scope and side-effect posture before meaningful mutations.

The active workspace's dominant primary action remains in the Unified Toolbar. The sidebar SHALL NOT duplicate every workspace primary action.

---

# 6. Canonical workspace registry

The first-party registry SHALL contain these IDs, labels, semantic groups, and default order values:

| workspace_id | label | semantic_group | default_order | default emphasis |
|---|---|---:|---:|---|
| `overview` | Overview | `work` | 10 | primary |
| `live` | Live | `work` | 20 | primary |
| `test_lab` | Test Lab | `work` | 30 | primary |
| `documents` | Documents | `work` | 40 | primary |
| `research` | Research | `work` | 50 | quiet |
| `studio` | Studio | `create` | 10 | quiet |
| `automations` | Automations | `create` | 20 | quiet |
| `evidence` | Evidence | `prove` | 10 | primary |
| `activity` | Activity | `prove` | 20 | quiet |
| `nodes_services` | Nodes & Services | `system` | 10 | quiet |
| `capabilities` | Capabilities | `system` | 20 | quiet |

Footer destinations:

| destination_id | label |
|---|---|
| `settings` | Settings |
| `help` | Help |

`Overview` remains the permanent workspace label. It renders the Mission Deck composition when a mission or Workpoint is active.

---

# 7. Workspace routes

The implementation SHALL use these first-party routes:

| workspace_id | route |
|---|---|
| `overview` | `/` |
| `live` | `/live` |
| `test_lab` | `/test-lab` |
| `documents` | `/documents` |
| `research` | `/research` |
| `studio` | `/studio` |
| `automations` | `/automations` |
| `evidence` | `/evidence` |
| `activity` | `/activity` |
| `nodes_services` | `/nodes-services` |
| `capabilities` | `/capabilities` |
| `settings` | `/settings` |
| `help` | `/help` |

Subsections SHALL use route data or nested routes without creating deeper than two visible navigation levels.

---

# 8. Workspace secondary navigation

## 8.1 Overview

No permanent secondary navigation.

Overview composition:

- Continue;
- Active now;
- Recent work;
- System posture;
- Suggested next actions;
- Mission Deck fields when a mission or Workpoint is active.

## 8.2 Live

Secondary views:

- Sessions;
- Recordings;
- Shares.

Session collection filters SHALL use documented states: active, parked, queued, recent, running, paused, takeover, idle, error, and waiting.

FPV PWA remains a projection of Live and receives no top-level workspace.

## 8.3 Test Lab

Secondary views:

- Flows;
- Runs;
- Baselines;
- Environments;
- Runners.

Registered first-party runner labels:

- UIAI Scenario;
- Maestro Web;
- Maestro Mobile;
- Tauri WebDriver;
- Visual Matrix.

## 8.4 Documents

Saved views:

- Inbox;
- Recent;
- Pinned;
- Forms;
- Contracts;
- Reports;
- Templates;
- Generated.

Supported object families remain those listed in `UIAI-COCKPIT-000`, including PDF, Office/OpenDocument files, scans, images, attachments, generated reports, and packets.

## 8.5 Research

Secondary views:

- Search;
- Captures;
- Collections;
- Packets.

## 8.6 Studio

Secondary views:

- Capture;
- Compare;
- Analyze;
- Design;
- Produce.

## 8.7 Automations

Secondary views:

- Recipes;
- Runs;
- Intake;
- Migration;
- Templates;
- Schedules/Triggers.

`Schedules/Triggers` remains feature-gated until implemented.

## 8.8 Evidence

Saved views:

- Current Workpoint;
- Recent;
- Needs capture;
- Needs review;
- Verified;
- Provisional/Surrogate;
- Public-safe;
- Receipts;
- Reports.

Review Reports remain Evidence work objects and contextual outputs; they receive no top-level workspace.

## 8.9 Activity

Segments:

- Now;
- Approvals;
- History;
- Jobs;
- Notifications;
- Audit.

## 8.10 Nodes & Services

Secondary views:

- Nodes;
- UIAI Engine;
- Focusa Local;
- Focusa Cloud;
- AI API;
- Pairing & Devices;
- Capacity;
- Sync;
- Updates & Compatibility.

## 8.11 Capabilities

Capabilities is a catalog with filters, not a fixed deep subsection tree.

Required filters:

- task;
- workspace;
- status;
- source plane;
- side effect;
- required scope;
- local/cloud;
- license;
- experimental state;
- artifact type.

---

# 9. Real route and capability placement

The sidebar and workspaces SHALL present only current or explicitly planned functionality, with truthful implementation states.

| workspace | current route/capability integration | exposure posture |
|---|---|---|
| Overview | Phase 0 Project Identity, Project Card, Workpoint Resume, Trajectory View, Work-loop Status, health/status summaries | current contracts; workspace composition to implement |
| Live | `/api/session/*`, `/api/share/*`, `/api/health/browser`, browser diagnostics, screenshots, errors | current core browser routes; Cockpit workspace to implement |
| Test Lab | UIAI Scenario, Maestro, Tauri WebDriver, Visual Matrix contracts | planned first-party workspace/backend track; show backend/adapter state truthfully |
| Documents | document object/runtime requirements from master | planned first-party backend track; show backend missing until implemented |
| Research | `/api/search`, `/api/search/providers`, `/api/markdown`, `/api/agent/research-packet` | current route families with existing agent parity |
| Studio/Capture | `/api/screenshot`, `/api/screenshot/*`, `/api/media/frame/catalog`, `/api/media/frame/render` | current routes |
| Studio/Analyze | `/api/reference/analyze`, `/api/critique`, `/api/critique/models`, `/api/critique/dimensions`, `/api/ui-reverse`, `/api/section-detect` | route-specific current/guarded states from parity inventory |
| Studio/Design | `/api/design-system`, `/api/content-map`, `/api/block-recipes`, `/api/style-enhance`, applicable `/api/copilot/*` | current guarded routes; generic agent exposure deferred |
| Studio/Produce | `/api/media/produce`, `/api/media/status/*`, `/api/media/jobs` | current long-running/guarded routes |
| Automations | `/api/workflow/*`, `/api/intake/*`, `/api/migration/*` | current guarded/mutating routes; workflow UI and safety proof required |
| Evidence | `focusa.evidence_link`, UIAI artifact/evidence handles, receipts, Review Reports contracts | Phase 0 write contract plus planned report/evidence composition |
| Activity | `/api/errors`, bounded event/job projections, approvals | errors current; bounded event/activity UI to implement |
| Nodes & Services | `/health`, `/api/health`, `/api/status`, `/api/metrics/browser`, pairing contracts, entitlement state, protected workers | current health/pairing foundations plus entitlement implementation work |
| Capabilities | `/api/tools/*`, Phase 0 CardManifest, future CapabilityManifest registry | current discovery routes; static first-party manifest before generated registry |

Raw admin, memory, intelligence, training, captcha/IP-pool, extension-token, and unbounded event-stream surfaces SHALL NOT receive ordinary navigation without a separately accepted operator workflow and safety proof.

---

# 10. Phase 0 card migration

The existing card grid SHALL be removed as the default product taxonomy. Existing cards SHALL be preserved in these placements:

| card_id | placement |
|---|---|
| `uiai.health` | Overview system posture; Nodes & Services; Capabilities |
| `uiai.diagnostics` | Live Inspector; Test Lab Inspector; Activity; Capabilities |
| `focusa.project_identity` | Context Control; Overview; Inspector |
| `focusa.project_card` | Overview; Scope Inspector |
| `focusa.workpoint_resume` | Resume Workpoint; Overview Continue |
| `focusa.trajectory_view` | Overview; Inspector |
| `focusa.tool_doctor` | Nodes & Services; contextual recovery |
| `focusa.dxux_requirement` | contextual recovery; Help; Capabilities |
| `focusa.work_loop_status` | Overview; Activity; Nodes & Services |
| `focusa.device_pair_status` | Nodes & Services → Pairing & Devices |
| `focusa.evidence_link` | contextual Capture Evidence action; Evidence |
| `cloud.node_status` | Nodes & Services |
| `cloud.device_pairing` | Nodes & Services → Pairing & Devices |
| `ai_api.health_usage` | Nodes & Services → AI API; paid-action approval surfaces |

Card manifests remain backend-to-UI presentation contracts and MAY be used in Overview, Inspectors, empty states, and contextual summaries.

---

# 11. Workspace manifest contract

Add `apps/cockpit/src/lib/contracts/workspace-manifest.ts`:

```ts
import type { CockpitWorkObject } from "./cockpit-work-object";

export type WorkspaceGroup = "work" | "create" | "prove" | "system";

export interface WorkspaceSubsectionManifest {
  subsection_id: string;
  label: string;
  route: string;
  order: number;
  default_visible: boolean;
  feature_flag?: string;
  developer_only?: boolean;
}

export interface WorkspaceManifest {
  workspace_id: string;
  label: string;
  icon: string;
  semantic_group: WorkspaceGroup;
  default_order: number;
  default_visible: boolean;
  default_emphasis: "primary" | "quiet";
  supported_object_kinds: CockpitWorkObject["kind"][];
  capability_ids: string[];
  route: string;
  subsections: WorkspaceSubsectionManifest[];
  feature_flag?: string;
  local_only_behavior: "works" | "partial" | "blocked";
  extension_source: "core" | "first_party" | "third_party";
  reorder_policy: "within_group" | "custom_group_allowed" | "fixed";
  pinnable: boolean;
  hideable: boolean;
}
```

First-party manifests SHALL be static and source-controlled before dynamic capability generation is introduced.

`Settings` and `Help` are fixed footer destinations and are not reorderable into workspace groups.

---

# 12. Sidebar preference and persistence contract

Add an additive sidebar field to the existing `UserSettings` model.

```ts
export type SidebarMode = "expanded" | "compact" | "hidden";
export type SidebarLayoutMode = "recommended" | "custom";
export type SidebarDensity = "comfortable" | "compact";

export interface SidebarWorkspacePlacement {
  workspace_id: string;
  display_group: "work" | "create" | "prove" | "system";
  order: number;
}

export interface SidebarPinnedRef {
  ref: string;
  kind: "workspace" | "saved_view" | "work_object";
  workspace_id: string;
  order: number;
}

export interface SidebarPreferencesV1 {
  schema: "uaiengine.cockpit.sidebar_preferences.v1";
  mode: SidebarMode;
  layout_mode: SidebarLayoutMode;
  density: SidebarDensity;
  width_px: number;
  collapsed_groups: Array<"work" | "create" | "prove" | "system">;
  workspace_placements: SidebarWorkspacePlacement[];
  hidden_workspace_ids: string[];
  pinned_refs: SidebarPinnedRef[];
  last_updated_at: string;
}
```

Persistence rules:

1. Recommended mode derives group/order from `WorkspaceManifest`.
2. First successful reorder switches to Custom mode.
3. Reset clears placement overrides, hidden workspaces, group-collapse overrides, and custom width only when explicitly selected by the user.
4. Unknown or removed workspace IDs are ignored but retained in migration diagnostics until settings cleanup.
5. Newly added first-party workspaces appear in their manifest-defined group/order without corrupting existing custom placement.
6. Preferences remain local presentation state and SHALL NOT be written into Focusa mission, Workpoint, Evidence, or canonical event history.
7. Preferences MAY synchronize through a future approved user-settings sync path, but local behavior cannot depend on cloud availability.

---

# 13. Drag-and-drop behavior

## 13.1 Required DnD surfaces

DnD SHALL support:

- reordering workspace rows within a group;
- moving workspaces between groups only when `reorder_policy="custom_group_allowed"`;
- pinning a workspace by dropping onto Favorites;
- reordering Favorites;
- pinning and reordering saved views;
- pinning and reordering work-object shortcuts;
- reordering user-created saved views inside a workspace collection.

Core first-party workspaces SHALL initially use `custom_group_allowed`, except any future security/recovery-only workspace explicitly marked fixed. Settings and Help remain fixed footer items.

## 13.2 DnD does not mutate canonical work

A navigation drag SHALL NOT:

- change Project;
- change Workstream;
- change Workpoint;
- change ScopeRef or NodeRef;
- transfer worker/task/session ownership;
- move or cancel a backend job;
- modify a session lease;
- alter authority or Capability Grants;
- attach/detach Evidence;
- change entitlement;
- delete or move an artifact;
- change a work object's canonical backend scope.

Dragging a work object into Favorites creates only a shortcut reference.

## 13.3 Interaction states

- drag handle appears on row hover and keyboard focus;
- pointer drag lifts the row using transform/elevation tokens;
- valid insertion targets show a placeholder gap;
- group targets show a bounded group highlight;
- Favorites shows a pin drop target;
- invalid targets do not accept the item and expose the reason through accessible text;
- movement uses 180–220ms transform-based animation unless reduced motion is enabled;
- order is committed only after a valid drop;
- Escape cancels an active keyboard or pointer drag;
- failed preference persistence restores the prior order and reports a recoverable error.

## 13.4 Keyboard alternative

Every reorderable row SHALL expose commands through keyboard and row menu:

- Move up;
- Move down;
- Move to group;
- Move to beginning;
- Move to end;
- Pin;
- Unpin;
- Hide, when hideable;
- Restore recommended position.

Keyboard DnD SHALL announce pickup, target position, group, and completion through an appropriate live region without flooding screen-reader output.

## 13.5 DnD adapter boundary

Add a Cockpit-owned adapter:

```ts
export interface SidebarDndAdapter {
  beginDrag(itemRef: string): void;
  previewDrop(target: SidebarDropTarget): SidebarDropDecision;
  commitDrop(target: SidebarDropTarget): Promise<void>;
  cancelDrag(): void;
}
```

A compatible Svelte DnD library MAY implement pointer mechanics, but components SHALL consume the Cockpit adapter and shared state rather than importing library-specific event contracts throughout the shell.

Dependency selection acceptance:

- compatible with Svelte 5 and the current Vite/SvelteKit build;
- pointer and touch support;
- no remote runtime dependency;
- supports stable item identity;
- permits custom keyboard behavior;
- passes Tauri/WebView smoke and reduced-motion tests;
- license is acceptable for distribution.

---

# 14. Workspace section header and hierarchical menu

The section header SHALL be labeled `Workspaces` and display the active layout mode when customized.

The floating menu contains:

## Layout

- Recommended;
- Custom.

## Sidebar

- Expanded;
- Compact;
- Hidden.

## Visibility

- Show hidden workspaces;
- Open Workspaces & Sidebar settings.

## Groups

- Expand all;
- Collapse all.

## Reset

- Reset sidebar layout.

The menu MUST use an accessible popover/menu pattern appropriate to its controls. Exclusive choices use radio semantics; persistent toggles use check semantics; commands use menu-item semantics. Nested panels open to the right when space permits and below or in-place on constrained widths.

`Mark all as read` does not belong in the workspace menu. It belongs in Activity/Notifications where implemented.

---

# 15. Sidebar visual and token contract

Add semantic tokens rather than local raw values.

Required geometry:

```css
--sidebar-width-expanded: 240px;
--sidebar-width-min: 208px;
--sidebar-width-max: 320px;
--sidebar-width-compact: 64px;
--sidebar-row-height: 32px;
--sidebar-row-height-compact-density: 28px;
--sidebar-active-bar-width: 2px;
```

Required semantic token families:

- sidebar/window surface;
- hover surface;
- selected surface;
- drop-target surface;
- primary and muted text;
- divider and border;
- focus ring;
- running, attention, verified, failed, and offline states;
- row height and indentation;
- resize handle;
- drag elevation;
- transition duration and easing.

Typography:

- workspace label: 14px body;
- selected label: medium weight;
- subsection/menu item: 13–14px;
- metadata/count: 12px;
- group header: 12–13px medium muted;
- no uppercase navigation labels;
- no raw technical IDs in ordinary navigation.

Icons:

- one source-owned outline family;
- 16–18px in rows;
- 18–20px for primary navigation controls;
- stable workspace icons;
- labels visible in Expanded mode;
- accessible names and tooltips in Compact mode.

Motion:

- hover/state transition: 120–150ms;
- group disclosure: 120–200ms;
- overlay/menu entrance: approximately 120ms;
- reorder movement: 180–220ms;
- GPU-friendly transform/opacity;
- reduced-motion disables nonessential interpolation.

---

# 16. Sidebar modes and responsive behavior

## Expanded

- icons, labels, groups, subsections, Favorites;
- default 240px;
- user-resizable 208–320px;
- width persisted per window.

## Compact

- 64px icon rail;
- selected workspace remains identifiable;
- actionable error/approval state remains identifiable without color alone;
- activating the current workspace icon MAY open a temporary labeled overlay for subsections/Favorites;
- tooltips are supplemental, not the only access path.

## Hidden

- workspace consumes the region;
- `[` and View menu restore the sidebar;
- Command Palette remains able to open every workspace.

## Overlay

- used on constrained windows;
- focus moves into the overlay;
- Escape and outside click close it;
- focus returns to the invoking control;
- previous scroll, selected workspace, group disclosure, and custom order are preserved.

---

# 17. Badges and activity integration

Sidebar badges SHALL show actionable state only, in this priority:

1. unresolved error;
2. pending approval;
3. scope or ownership conflict;
4. running work;
5. Evidence needing capture/review;
6. entitlement warning;
7. update available.

A row receives at most one primary visual badge. Accessible text SHALL summarize combined state. Color or a status dot cannot be the only signal.

The Activity Bar remains the global actionable signal surface. Its collapsed order remains:

1. unresolved errors;
2. running jobs/actions;
3. pending approvals;
4. scope/ownership conflicts;
5. evidence awaiting capture;
6. sync backlog;
7. pairing/token/entitlement warnings;
8. predictions awaiting evaluation;
9. update available.

Selecting a sidebar badge or Activity Bar signal opens the owning workspace or Activity with an explicit filter.

---

# 18. Entitlement and recovery integration

The August 1 entitlement work SHALL integrate without adding a permanent License workspace.

## 18.1 Normal Cockpit

Entitlement state appears through:

- Nodes & Services → UIAI Engine;
- Capabilities filters and capability states;
- contextual locked-capability presentation;
- Activity Bar when operator action is required;
- approval/denial/recovery surfaces.

## 18.2 Capability state

Capability metadata SHALL support:

- license feature;
- evaluation availability;
- limit-consumption posture;
- current entitlement state;
- exact stable denial code;
- recovery action.

Locked capabilities remain discoverable. Execution SHALL fail before browser, model, media, job, or mutating storage allocation.

## 18.3 Recovery-only state

For `unactivated`, `expired`, `revoked`, or `invalid` states, the shell SHALL preserve access to:

- entitlement recovery;
- safe diagnostics;
- local artifacts;
- Evidence;
- reports;
- export;
- uninstall guidance.

Entitled execution is blocked. The workspace registry remains available to explain capability state, but blocked workspaces MUST NOT imply executable availability.

## 18.4 Protected workers

Protected-worker and feature-capsule state belongs in:

- Nodes & Services → UIAI Engine;
- Capacity;
- affected capability details;
- Developer diagnostics.

It receives no top-level navigation item.

---

# 19. Accessibility contract

Primary navigation SHALL use:

- `<nav aria-label="Cockpit workspaces">`;
- semantic lists;
- semantic buttons for disclosures;
- `aria-expanded` for group state;
- `aria-current="page"` for active routes;
- visible high-contrast focus indication.

Ordinary workspace navigation SHALL NOT use ARIA tree semantics. A true hierarchical object tree uses a separately reviewed tree component.

Keyboard behavior:

- Up/Down move through visible rows;
- Left collapses a group or returns from a subsection overlay;
- Right expands a group or enters available subsection navigation;
- Enter/Space activates;
- Escape closes menus/overlays or cancels DnD;
- Shift+F10 opens row commands;
- `Command-K` opens Search and commands;
- `Command-,` opens Settings;
- `?` opens Help;
- `[` toggles the sidebar;
- `Command-1…9` opens configured Favorites.

Focus and selection are distinct. DnD has complete keyboard alternatives. Required status, recovery, and actions never depend only on hover, color, tooltip, animation, or drag.

---

# 20. Component and module layout

Add:

```text
apps/cockpit/src/lib/ui/sidebar/
  CockpitSidebar.svelte
  SidebarHeader.svelte
  ContextControl.svelte
  ResumeWorkpointButton.svelte
  SearchCommandButton.svelte
  WorkspaceNavigation.svelte
  WorkspaceGroup.svelte
  WorkspaceRow.svelte
  WorkspaceSubnavigation.svelte
  WorkspaceMenu.svelte
  FavoritesSection.svelte
  SidebarFooter.svelte
  SidebarResizeHandle.svelte
  SidebarDropIndicator.svelte
  SidebarLiveRegion.svelte
```

Add shell components:

```text
apps/cockpit/src/lib/ui/shell/
  UnifiedToolbar.svelte
  WorkObjectTabs.svelte
  UniversalInspector.svelte
  ActivityBar.svelte
  ContextualCollectionPane.svelte
```

Add contracts/state:

```text
apps/cockpit/src/lib/contracts/
  workspace-manifest.ts
  cockpit-work-object.ts
  sidebar-preferences.ts
  sidebar-dnd.ts
  sidebar-badge.ts

apps/cockpit/src/lib/navigation/
  core-workspace-manifest.ts
  derive-sidebar-model.ts
  sidebar-controller.ts
  sidebar-dnd-adapter.ts
  sidebar-preference-store.ts
```

Shared UI primitives and tokens remain under `apps/cockpit/src/lib/ui/`. Workspace-local forks are prohibited.

---

# 21. Controller and authority integration

No Svelte component SHALL call raw HTTP routes directly.

All actions preserve the integration spine:

```text
Workspace / Card / Command UI
  → Controller
  → ScopeGuard / AuthorityGuard / ConsentGuard
  → NodeRouter / TransportRouter
  → ApiAdapter or RunnerAdapter
  → ResultNormalizer / RedactionBoundary
  → ArtifactStore / EventBus / LocalStore
  → ViewModel
```

Sidebar navigation and preference writes remain local presentation actions. Context changes, Resume Workpoint, capability commands, evidence operations, pairing, entitlement recovery, and backend mutations continue through their owning controllers and guards.

---

# 22. Implementation task graph

Tasks are intentionally ordered for direct agent decomposition. A task MUST NOT be marked complete solely because a component renders; its contract, accessibility, tests, and migration acceptance must pass.

## T003-00 — Register amendment and freeze source map

**Depends on:** none  
**Outputs:** this document; document-register entry; source map  
**Acceptance:** register resolves `UIAI-COCKPIT-003`; no conflicting document number exists.

## T003-01 — Current-state code and fixture inventory

**Depends on:** T003-00  
**Files:** current `+layout.svelte`, `+page.svelte`, contracts, tokens, package, smoke harness  
**Work:**

- record current shell DOM and screenshots;
- record current route tree;
- record current Phase 0 card IDs and contract mappings;
- record package dependencies and build commands;
- add baseline fixtures for Expanded, Compact, Hidden, Overlay, empty context, missing scope, and blocked entitlement states.

**Acceptance:** no current component, card, route, or contract is removed without a mapped destination.

## T003-02 — Core workspace and work-object contracts

**Depends on:** T003-01  
**Work:**

- implement `CockpitWorkObject` from master;
- implement `WorkspaceManifest` and subsection manifest;
- add all first-party workspace registry entries exactly as Section 6;
- validate unique IDs, routes, groups, order, object kinds, and capability references;
- add manifest unit tests.

**Acceptance:** static registry validates; every top-level workspace is generated from the manifest, not hard-coded Svelte markup.

## T003-03 — Sidebar preference schema and migration

**Depends on:** T003-02  
**Work:**

- add `SidebarPreferencesV1` to UserSettings additively;
- implement local default derivation;
- implement schema validation and migration;
- implement unknown/removed workspace handling;
- implement recommended-layout reset;
- implement persistence failure rollback.

**Acceptance:** restart preserves mode, width, groups, custom order, visibility, and Favorites; Local Only works without cloud.

## T003-04 — Design tokens and shared primitives

**Depends on:** T003-01  
**Work:**

- extend design tokens with sidebar geometry, selection, DnD, resize, badges, and overlay layers;
- implement shared row, disclosure, tooltip, popover, resize handle, focus, badge, and empty-state primitives;
- remove raw sidebar values from workspace components;
- add light, dark, compact-density, high-contrast, and reduced-motion fixtures.

**Acceptance:** token lint/allowlist passes; no sidebar-local raw color, radius, shadow, spacing, or duration values.

## T003-05 — Shell replacement scaffold

**Depends on:** T003-02, T003-03, T003-04  
**Work:**

- replace primitive navigator in `+layout.svelte` with `CockpitShell` regions;
- add Unified Toolbar, Primary Sidebar, work-object tab region, viewport, Inspector, Activity Bar;
- retain a functioning slot/route outlet;
- preserve shell stability when individual workspaces fail.

**Acceptance:** shell renders from registry; no backend-plane navigation remains; unrelated workspace failure does not crash shell.

## T003-06 — Context Control

**Depends on:** T003-05  
**Integration:** Project Identity, Project Card, Workpoint, NodeRef, ScopeRef, pairing, authority, sync  
**Work:**

- replace independent scope chips;
- derive human-readable collapsed context;
- implement expanded context panel;
- implement missing/stale/conflict/read-only states;
- wire change actions through owning guards/controllers;
- add keyboard/focus behavior.

**Acceptance:** writes block with exact recovery when scope/authority is invalid; no raw example values or IDs appear by default.

## T003-07 — Primary workspace navigation

**Depends on:** T003-05  
**Work:**

- render Work/Create/Prove/System groups from manifest;
- implement active route, disclosure, badges, quiet/primary emphasis;
- implement workspace subsections;
- implement Workspaces menu;
- implement hide/restore and group collapse;
- implement Settings and Help fixed footer.

**Acceptance:** every enabled workspace is reachable by pointer and keyboard; no more than two visible levels.

## T003-08 — Sidebar modes, resize, and responsive overlay

**Depends on:** T003-05, T003-07  
**Work:**

- Expanded 208–320px resize;
- Compact 64px rail;
- Hidden state;
- constrained-window Overlay;
- persistence and responsive restoration;
- `[` command and View-menu hook;
- focus return and Escape behavior.

**Acceptance:** mode transitions preserve selection, scroll, group state, badges, and custom order.

## T003-09 — DnD adapter and user ordering

**Depends on:** T003-03, T003-04, T003-07  
**Work:**

- select and document a compatible DnD dependency or implement equivalent pointer mechanics;
- add Cockpit adapter boundary;
- implement within-group and permitted cross-group workspace movement;
- implement Favorites drop zone;
- implement saved-view/work-object shortcut pinning;
- implement keyboard reorder commands and announcements;
- persist custom placements transactionally;
- add cancel and failed-save rollback.

**Acceptance:** all DnD requirements in Section 13 pass; no canonical scope, authority, evidence, or backend object state changes.

## T003-10 — Resume Workpoint and global search integration

**Depends on:** T003-06, T003-07  
**Work:**

- render Resume Workpoint only from real contract/view-model state;
- open correct Overview/work object;
- implement Search and commands entry;
- index workspace manifests and registered objects/capabilities;
- avoid duplicating active-workspace primary actions.

**Acceptance:** no fake resume data; Search opens all enabled workspaces and registered capabilities.

## T003-11 — Workspace route skeletons and secondary navigation

**Depends on:** T003-02, T003-05, T003-07  
**Work:**

- create routes in Section 7;
- create secondary navigation in Section 8;
- add contextual collection pane host;
- implement truthful empty/backend-missing/adapter-missing/gated states;
- preserve object-tab affiliation.

**Acceptance:** all routes build; planned workspaces do not overstate backend implementation.

## T003-12 — Phase 0 card migration and Overview

**Depends on:** T003-06, T003-10, T003-11  
**Work:**

- remove default Phase 0 card grid;
- implement Overview Continue, Active now, Recent work, System posture, Suggested next actions;
- place every Phase 0 card according to Section 10;
- preserve card contract validation;
- add Overview/Mission Deck progressive disclosure.

**Acceptance:** all 14 cards remain reachable or represented; card grid no longer defines product IA.

## T003-13 — Live and current browser-route integration

**Depends on:** T003-11  
**Work:**

- integrate session list/state, shares, recordings, diagnostics, screenshots, errors, health;
- add current action and agent-first Inspector sections;
- add session capacity/lease posture when available;
- preserve entitlement gating and no direct component-route calls.

**Acceptance:** current browser routes are operable through adapters; stale/blocked/degraded states are truthful.

## T003-14 — Research, Studio, Automations route placement

**Depends on:** T003-11  
**Work:**

- wire current Research routes;
- wire current screenshot/frame routes;
- represent guarded analysis/design/media/workflow routes with exact capability state;
- route long-running media/workflow work to Activity Jobs;
- keep intentionally omitted agent exposure omitted.

**Acceptance:** route parity matrix and Cockpit placement agree; no route becomes visible merely because it exists.

## T003-15 — Evidence, Reports, Activity, Nodes, Capabilities

**Depends on:** T003-11, T003-12  
**Work:**

- implement Evidence saved views and Phase 0 evidence action;
- host Review Reports under Evidence;
- implement Activity segments and badge routing;
- implement Nodes & Services secondary views;
- implement static Capability catalog from current cards/routes/manifests;
- expose adapter/backend/gating state truthfully.

**Acceptance:** Reports are not top-level; jobs/approvals/notifications remain Activity; backend planes remain Nodes & Services/metadata.

## T003-16 — Entitlement and protected-worker UX

**Depends on:** T003-07, T003-11, entitlement WP1–WP4 foundations  
**Work:**

- consume canonical entitlement state;
- add capability license/limit metadata;
- implement locked capability and recovery states;
- implement recovery-only shell behavior;
- expose protected-worker/capsule status under UIAI Engine and capability detail;
- preserve local artifacts/Evidence after expiry.

**Acceptance:** no execution allocation before entitlement success; no new License workspace; stable denial envelope used.

## T003-17 — Accessibility, i18n, and interaction completeness

**Depends on:** T003-07 through T003-16  
**Work:**

- keyboard navigation and reorder;
- focus management;
- screen-reader labels/live regions;
- dragging alternatives;
- RTL logical properties;
- message-key conversion;
- 200% zoom and text enlargement;
- high contrast and reduced motion.

**Acceptance:** automated and manual accessibility checks pass; no essential operation is drag-only or hover-only.

## T003-18 — Test, smoke, visual regression, and performance gates

**Depends on:** all implementation tasks  
**Work:**

- unit tests for manifests, preferences, migration, DnD decisions, badge priority;
- component tests for disclosure, keyboard, overlays, resize, Compact mode;
- route tests for all workspace paths;
- integration tests for Context Control, Resume, Activity routing, entitlement recovery;
- visual fixtures for normal/loading/empty/blocked/degraded/error/approval/success;
- required screenshots: default, dark, constrained, non-happy, overlay, DnD;
- performance checks for navigation render and reorder under representative object counts;
- update `cockpit:smoke` to validate workspace registry and Phase 0 migration.

**Acceptance:** CI-blocking gates pass; no regression to existing contract and backend smokes.

## T003-19 — Rollout and Slice 0 cleanup

**Depends on:** T003-18  
**Work:**

- enable new shell behind a temporary migration flag only if needed;
- migrate existing settings;
- remove primitive navigator, scope-chip strip, old card-grid home, and old process-ribbon markup;
- remove temporary flag after acceptance;
- update docs, Help, screenshots, release notes, and support diagnostics.

**Acceptance:** one production shell remains; no duplicate navigation or dead Slice 0 UI path remains.

---

# 23. Dependency summary

```text
T003-00
  → T003-01
      → T003-02
          → T003-03
          → T003-05
      → T003-04
          → T003-05

T003-05
  → T003-06
  → T003-07
      → T003-08
      → T003-09
      → T003-10
      → T003-11

T003-06 + T003-10 + T003-11
  → T003-12

T003-11
  → T003-13
  → T003-14
  → T003-15

T003-07 + T003-11 + entitlement foundations
  → T003-16

T003-07 through T003-16
  → T003-17
  → T003-18
  → T003-19
```

Parallelizable work:

- T003-03 and T003-04 after their dependencies;
- T003-06 and T003-07 after shell scaffold;
- T003-08, T003-09, T003-10, and T003-11 after primary navigation foundations;
- T003-13, T003-14, and T003-15 after route skeletons.

Tasks that must remain sequential:

- contract registry before manifest-driven shell;
- preference schema before DnD persistence;
- shared tokens/primitives before workspace-local visual implementation;
- entitlement foundations before executable entitlement UX;
- accessibility completion before release acceptance;
- cleanup only after migration and regression proof.

---

# 24. Agent decomposition instructions

The implementing agent SHALL:

1. Decompose each `T003-*` task into repository-native tracked tasks while retaining the parent task ID.
2. Preserve dependencies from Sections 22–23.
3. Include exact files, contracts, tests, and acceptance criteria in every child task.
4. Mark route/capability state as current, guarded/deferred, adapter missing, or backend missing; never infer implementation from documentation alone.
5. Avoid introducing a second state store, route registry, card registry, or navigation taxonomy.
6. Add shared components before local forks.
7. Keep DnD changes presentation-only and prove no canonical mutation.
8. Update manifests, tests, docs, screenshots, and parity metadata in the same change when a workspace/capability placement changes.
9. Stop and record a blocker when a required authority contract is absent rather than fabricating data or bypassing guards.
10. Treat the task complete only when normal and non-happy UI states pass the design and accessibility contract.

Recommended change boundaries:

- one PR/change set for contracts/manifests/preferences;
- one for shared shell/sidebar primitives;
- one for DnD and persistence;
- one for Context Control/Search/Resume;
- one or more workspace integration changes grouped by current route ownership;
- one entitlement/recovery integration change coordinated with entitlement work packages;
- one final migration/cleanup and release-proof change.

---

# 25. Prohibited patterns

Implementation MUST NOT introduce:

- backend products as primary workspace rows;
- a separate FPV workspace;
- a separate Reports workspace;
- a separate Jobs, Approvals, Notifications, or Errors workspace;
- a separate License workspace solely for entitlement state;
- raw API endpoints as ordinary navigation;
- raw Memory, Intelligence, Training, Captcha, or Admin navigation without accepted workflows;
- hard-coded workspace lists in Svelte components;
- drag-only operation;
- automatic reorder based on recency;
- DnD that mutates canonical work;
- example names or fake counts as production defaults;
- raw JSON as the primary UI;
- disabled controls without a reason;
- icon-only persistent navigation without discoverable labels;
- nested navigation deeper than required;
- workspace-local design systems;
- direct raw API calls from production Svelte components.

---

# 26. Unified acceptance criteria

This amendment is complete when:

1. The current primitive navigator is removed.
2. The exact Work/Create/Prove/System taxonomy renders from manifests.
3. Overview remains the permanent label and becomes Mission Deck when context requires.
4. Context Control replaces independent scope chips.
5. Resume Workpoint consumes real contract state only.
6. Search and commands reaches every enabled workspace and registered capability.
7. Expanded, Compact, Hidden, and Overlay modes work and persist.
8. Users can reorder workspaces, permitted groups, Favorites, saved views, and pinned work objects with pointer and keyboard.
9. Recommended layout remains recoverable from manifest defaults.
10. DnD changes presentation preferences only.
11. Every workspace and subsection in Sections 6–8 is reachable.
12. Phase 0 cards are preserved in their mapped destinations.
13. Current routes are placed according to Section 9 and parity documentation.
14. Planned workspaces display truthful backend/adapter state.
15. Reports remain under Evidence and contextual entry points.
16. FPV remains under Live.
17. Runners remain under Test Lab.
18. Jobs, approvals, notifications, history, and audit remain under Activity.
19. Backend planes, protected workers, and entitlement status remain under Nodes & Services, Capabilities, Inspector, or recovery UX.
20. Locked capabilities remain discoverable and fail before resource allocation.
21. Expired/revoked/invalid states preserve local artifacts, Evidence, reports, safe diagnostics, export, and recovery.
22. Navigation is fully keyboard accessible and no operation is drag-only.
23. Light, dark, compact density, reduced motion, high contrast, RTL, 200% zoom, and constrained-window states pass.
24. Normal, loading, empty, blocked, degraded, error, approval, and success fixtures exist for every major workspace/shell surface.
25. Manifest, preference, DnD, route, accessibility, visual, smoke, and performance gates are CI-blocking.
26. No duplicate Slice 0 shell path remains after rollout.
27. No fictional product data or example values are shipped as defaults.

---

# 27. Final implementation principle

> **Cockpit navigation presents the real kinds of work UIAI Engine and Focusa support, preserves complete discoverability through manifests and Capabilities, and lets the user shape presentation through safe local ordering without ever changing canonical mission, authority, execution, or evidence state.**
