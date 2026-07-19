- Session/thread
- Node
- Role
- authority state
- sync state
- route/transport
- change context
- open full scope inspector

When scope is stale, missing, conflicting, or read-only, the control becomes prominent and names the exact recovery action.

## 6.5 Work-object tabs

Tabs represent open work, not product categories.

Examples:

```text
Checkout FPV
Login flow #42
Vendor Agreement.pdf
Homepage comparison
Research: OAuth providers
```

Rules:

- Keep approximately six tabs visible; use overflow for more.
- Preserve workspace affiliation through icon and subtle label.
- Tabs maintain local UI state independently.
- Closing a tab closes only the view unless the action explicitly says **End session**, **Cancel job**, or **Delete artifact**.
- Pinned tabs survive restart.
- Recently closed objects are recoverable.

## 6.6 Universal inspector

The right inspector adapts to the selected object but keeps consistent sections:

1. **Summary** — human-readable state and primary metadata.
2. **Details** — type-specific fields and configuration.
3. **Scope** — project, Workpoint, node, role, authority, and route.
4. **Evidence** — artifacts, evidence refs, receipt state, and capture action.
5. **History** — bounded events and operation lineage.
6. **Raw** — redacted JSON, manifests, logs, and correlation IDs; Developer Mode only.

The inspector docks, overlays, or collapses based on window width. It must never permanently steal the main viewport.

## 6.7 Activity Bar

The existing bottom ribbon becomes the **Activity Bar**.

Collapsed state shows only actionable signals, in priority order:

1. unresolved errors;
2. running jobs/actions;
3. pending approvals;
4. scope or ownership conflicts;
5. evidence awaiting capture;
6. sync backlog;
7. pairing or token warnings;
8. predictions awaiting evaluation;
9. update available.

Selecting an item expands Activity to the relevant filtered view.

Avoid decorative counters. An empty Activity Bar simply says **All clear** or remains visually quiet.

## 6.8 Global search and command palette

`Command-K` opens one universal surface with two modes:

### Find

Searches, within current readable scope:

- workspaces;
- capabilities;
- browser sessions;
- test flows and runs;
- documents;
- research captures;
- artifacts and evidence refs;
- Workpoints and Trajectories;
- jobs;
- nodes and settings;
- help topics.

### Do

Executes bounded commands such as:

- Start browser session
- Run test
- Import PDF
- Capture URL as Markdown
- Compare screenshots
- Open diagnostics
- Resume Workpoint
- Pair node

Commands show scope and side-effect labels before execution.

## 6.9 Menu bar and keyboard model

The desktop app should provide conventional menus:

- File
- Edit
- View
- Work
- Window
- Help

Power features remain reachable by keyboard without becoming visible clutter. Existing documented shortcuts remain preserved and are expanded by workspace-specific commands.

---

# 7. Workspace specifications

## 7.1 Overview

### Purpose

Overview answers five questions:

1. Where am I?
2. What is happening now?
3. What needs my attention?
4. What should happen next?
5. What proof already exists?


When a mission or Workpoint is active, Overview is the Mission Deck defined in Section 4.2. Its primary summary adds:

- desired external outcome and Mission Contract version;
- mission lifecycle and settlement posture;
- verified versus remaining completion predicates;
- active workers/leases and pending decisions;
- current authority lease and remaining hard budgets;
- strongest contradiction, uncertainty, or missing evidence;
- Pause, Stop, Take over, Amend, Fork, and Resume actions when allowed.

The default composition remains calm: only fields relevant to the current decision are visible. Full task graphs, grant envelopes, receipts, and route details remain in Inspect or Developer disclosure.

### Default composition

#### Continue

A prominent continuation card shows:

- current Workpoint mission;
- current action;
- exact next action;
- last verified evidence;
- **Resume** primary action.

#### Active now

Compact live rows for:

- active browser sessions;
- running test/document/media/workflow jobs;
- pending operator approvals;
- agent questions or FPV interventions.

#### Recent work

A mixed recent-object list:

- sessions;
- tests;
- documents;
- research captures;
- comparisons;
- evidence.

#### System posture

A quiet one-line summary:

```text
UIAI healthy · Focusa verified · 1/4 browser slots · Local Only
```

Details open in Nodes & Services rather than occupying the home screen.

#### Suggested next actions

Bounded, explainable suggestions based on current Workpoint and available capabilities. Suggestions do not execute automatically.

### Progressive disclosure

- Level 0: Workpoint and important alerts.
- Level 1: active and recent work.
- Level 2: scope, evidence, and system posture.
- Level 3: dashboard layout and suggestion rules.
- Level 4: source events and adapter state.

## 7.2 Live

Live is the native Cockpit home for FPV and agent/browser session operations.

### Collection view

Display active, parked, queued, and recent sessions.

Each session tile shows:

- live thumbnail;
- human label;
- current URL/domain;
- running, paused, takeover, idle, error, parked, or waiting state;
- current action;
- owning project/Workpoint;
- viewer/control count;
- error indicator;
- lease/capacity posture when the session broker exists.

Support list and multi-agent canvas modes.

### Session view

The center canvas prioritizes the live browser mirror. The header also identifies the mission/Workpoint, current task owner, action lifecycle state, selected actuator, and whether the observed result is merely submitted, accepted, verified, or settled.

Toolbar actions:

- Run / Pause / Take over / Release;
- send message;
- annotate;
- capture evidence;
- record;
- share to phone;
- open diagnostics;
- end or park session.

### Live side data

The inspector provides tabs or sections for:

- Current action
- Recent tools
- Network
- Console
- Snapshot/accessibility tree
- Evidence
- Focusa context
- Session state
- Viewers and audit

Only Current action and failures are shown by default. Full network and console detail are deeper disclosure.

### FPV PWA relationship

The PWA is a projection of the same Live session:

```text
Cockpit Live session
  ├── native full workspace
  ├── mobile PWA share
  ├── stakeholder read-only share
  └── temporary steering/takeover share
```

Cockpit controls share creation, role, TTL, redaction, revoke, and audit. The PWA retains mobile affordances, touch controls, reconnect behavior, push notifications, and zero-install distribution.

### Recording and replay

Preserve:

- tool calls;
- frames before/after;
- navigation;
- network and console summaries;
- operator messages and actions;
- annotations;
- evidence events;
- duration;
- replay;
- fork from a selected step;
- compare to baseline or human run.

### Session broker and capacity

Live should surface cooperative capacity rather than a generic pool-full error:

- owner/client;
- project and Workpoint;
- lease expiration;
- safe-to-close or park posture;
- queue position;
- recommended action;
- release, extend, park, and restore when implemented.

## 7.3 Test Lab

Test Lab unifies scenario execution, Maestro, native Tauri E2E, visual regression, accessibility checks, and replay.

### Information architecture

```text
Test Lab
  ├── Flows
  ├── Runs
  ├── Baselines
  ├── Environments
  └── Runners
```

These are secondary navigation inside one workspace, not top-level app tabs.

### Flow library

A flow contains:

- name and purpose;
- project scope;
- tags;
- runner compatibility;
- environment requirements;
- ordered semantic steps;
- assertions;
- fixture/data requirements;
- artifact policy;
- video policy;
- evidence policy;
- last result and reliability trend.

Support:

- create from template;
- import Maestro YAML;
- create from recorded Live session;
- create from UIAI scenario JSON;
- duplicate;
- version;
- lint;
- replay;
- parameterize.

### Runner adapters

First-party runners:

| Runner | Best use |
|---|---|
| **UIAI Scenario** | Browser-native semantic/primitive flows using UIAI sessions and diagnostics. |
| **Maestro Web** | Black-box web flows with Maestro artifact/report support. |
| **Maestro Mobile** | iOS/Android or Capacitor/mobile UI flows. |
| **Tauri WebDriver** | Packaged native Tauri shell E2E on supported desktop platforms. |
| **Visual Matrix** | Multi-viewport screenshot, golden baseline, and visual comparison workflows. |

A flow declares compatible runners; the UI does not pretend one runner covers every platform.

### Run setup

The normal run sheet asks only for:

- flow;
- environment;
- target/device;
- runner;
- **Run**.

Advanced options—tags, retries, shards, network profile, media emulation, intercepts, time travel, auth profile, artifact retention—are collapsed under **Run options**.

### Active run view

Combine:

- FPV/live stage;
- current step and progress;
- step timeline;
- assertions;
- diagnostics;
- screenshots;
- synchronized video;
- logs;
- network/console failures;
- operator intervention;
- pause, stop, retry, and fork.

The user should not switch to Live merely to observe a Test Lab run. The run embeds the same Live viewer component while preserving a direct link to the underlying session object.

### Result view

Start with:

```text
Passed / Failed / Blocked
Duration
Failed step or strongest proof
Primary artifact
Recommended next action
```

Deeper levels reveal:

- full steps;
- JUnit/HTML reports;
- video markers;
- screenshot diffs;
- diagnostics;
- raw runner output;
- exact command and versions.

### Verification and Focusa

A test runner is an executor and evidence producer; it is not automatically the independent verifier or completion authority.

Each flow SHOULD declare:

- preconditions;
- expected postconditions;
- Completion Contract predicates it evaluates;
- acceptable evidence per assertion;
- whether the runner result is native, surrogate, simulated, or human-reviewed;
- idempotency/reconciliation posture for side effects;
- settlement window for asynchronous behavior.

A completed run can:

- attach bounded result and artifact refs to the active Workpoint;
- evaluate a prediction;
- propose metacognitive learning;
- produce a future `e2e_verification` or `visual_ui_verification` receipt when that Focusa contract exists;
- distinguish actual proof, blocked proof, surrogate proof, and missing native proof.

## 7.4 Documents

Documents is one first-class Cockpit workspace for PDF and Office work.

### Supported object families

- PDF;
- DOCX;
- XLSX;
- PPTX;
- ODT/ODS/ODP;
- scans and images;
- email/document attachments;
- generated reports and packets.

### Collection view

Organize by:

- Inbox
- Recent
- Pinned
- Forms
- Contracts
- Reports
- Templates
- Generated

These are saved views and tags, not rigid filesystem replacement.

### Document view

```text
┌──────────────┬──────────────────────────────────┬────────────────────┐
│ Pages/files  │ Document canvas                  │ Inspector          │
│ thumbnails   │ text/region selection            │ Summary            │
│ outline      │ overlays and comparison          │ Fields/Tables      │
│ attachments  │                                  │ Changes/Evidence   │
└──────────────┴──────────────────────────────────┴────────────────────┘
```

### Core reading capabilities

Preserve and implement:

- local file, upload, scanner, and approved URL import;
- immutable original artifact;
- page rendering and thumbnails;
- native text, word, block, coordinate, and reading-order extraction;
- page and region citations;
- outline, links, annotations, comments, attachments, and metadata;
- tables to structured data, CSV, Markdown, and XLSX;
- form and key-value extraction;
- digital-signature inspection;
- encryption and permissions inspection;
- OCR-needed detection and OCR;
- Markdown, JSON, and JSONL semantic outputs;
- text, structure, and visual comparison;
- document classification and summarization through explicit AI actions.

### Core writing capabilities

Preserve and implement:

- merge, split, reorder, duplicate, remove, rotate, crop, and resize pages;
- headers, footers, numbering, stamps, and watermarks;
- fill, reset, and flatten forms;
- annotations and comments;
- attachments and metadata;
- encryption and permissions;
- optimize and linearize;
- searchable OCR PDF;
- PDF/A conversion;
- HTML/Markdown/template/image/Office-to-PDF generation;
- document packet composition;
- real redaction with verification;
- comparison and marked-up review output;
- signature validation and later operator-approved signing/timestamping.

### Interaction model

Read-only inspection is immediate. Mutations create a **proposed derivative**:

```text
Original → Proposed output → Preview/diff → Approve → Final derivative
```

The original remains immutable. The user can always inspect lineage and tool/version metadata.

### Document recipes

Reusable recipes support workflows such as:

- redact sensitive identifiers;
- combine exhibits;
- invoice extraction;
- contract clause review;
- form filling;
- convert report packet to PDF/A;
- table extraction;
- compare signed vs approved document.

Recipes use bounded capabilities and approval gates, not arbitrary shell execution.

### Safety

- Never execute embedded PDF JavaScript.
- Never auto-launch external actions or attachments.
