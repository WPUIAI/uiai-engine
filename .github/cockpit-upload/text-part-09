- technical detail placement;
- no duplicated component or one-off token;
- visual polish at actual application scale, not only isolated component scale.

### Screenshot evidence in pull requests

PRs that change a user-facing surface MUST attach or link screenshots/video for:

- default state;
- dark mode;
- a constrained window;
- the most important non-happy state;
- any overlay or animation changed.

The PR MUST identify which shared component/pattern is used and whether a design exception is requested.

## 10.20 Prohibited-pattern catalog

The following patterns MUST fail review unless covered by an approved exception:

- generic admin-dashboard appearance;
- card-inside-card-inside-card layouts;
- multiple competing primary buttons;
- raw JSON as the default result;
- icon-only persistent navigation with no discoverable label;
- unlabeled or placeholder-only inputs;
- unexplained disabled controls;
- critical information only in color, hover, toast, or tooltip;
- modal-on-modal flows;
- popovers used for warnings or complex workflows;
- blocking spinners over the entire app;
- empty states that are actually loading or error states;
- animation for decoration, celebration, or constant ambient activity;
- arbitrary shadows, radii, colors, or durations;
- nested navigation deeper than the information hierarchy requires;
- destructive actions placed adjacent to primary actions without separation;
- hidden mutations or automatic cloud sends;
- hover-only canvas controls;
- drag-only operation;
- custom component forks for a local workspace;
- truncation that hides the only recovery or authority information;
- using technical jargon before human meaning;
- placing configuration and diagnostics in the ordinary work path when they can be progressively disclosed.

## 10.21 External design standards

Implementation and review MUST consult current official guidance, including:

- Apple Human Interface Guidelines for design principles, macOS, typography, icons/SF Symbols, sidebars, popovers, sheets, alerts, loading, motion, and accessibility;
- WCAG 2.2 for contrast, focus visibility/appearance, target size, dragging alternatives, keyboard access, and accessible authentication;
- WAI-ARIA Authoring Practices for disclosure, dialog, tabs, toolbar, tooltip, tree, and window-splitter behavior.

Official guidance informs the Cockpit system; it does not replace Cockpit-specific tokens, contracts, or release gates.

---

# 11. Accessibility, keyboard, and internationalization

## 11.1 Keyboard-first without keyboard-only assumptions

Preserve current global shortcuts and add semantic workspace shortcuts.

Global:

- `Command-K` — search/commands
- `Command-,` — settings
- `?` — help
- `[` — toggle sidebar
- `]` — toggle inspector
- `Escape` — close transient UI
- `Command-1…9` — configurable favorite workspaces/objects

Contextual actions must be listed in the Help overlay and menus.

## 11.2 Focus behavior

- Visible focus indicator on every interactive element.
- Logical focus order follows visual/task order.
- Opening a sheet/dialog moves focus into it.
- Closing returns focus to the initiator.
- Live takeover has an unmistakable keyboard focus and exit mechanism.
- Keyboard navigation cannot accidentally issue destructive browser/document actions.

## 11.3 Targets and pointer behavior

- Comfortable pointer/touch targets.
- PWA mobile controls retain at least the current 44px posture.
- Dense desktop tables may use smaller visual rows only when the actionable hit region remains adequate.
- Hover-only information must also be keyboard/focus accessible.

## 11.4 Screen readers

- Live mirror exposes an alternate accessibility-tree mode.
- Dynamic job and failure updates use appropriate live regions without flooding.
- Canvas selections have semantic descriptions.
- Charts and comparisons include text summaries.
- Scope, authority, and warning states are spoken explicitly.

## 11.5 Internationalization

Preserve the existing i18n requirements:

- all user-facing strings use message keys;
- locale-aware formatting;
- en-US initial locale;
- planned de-DE, fr-FR, es-ES, ja-JP, zh-CN, and pt-BR;
- logical CSS properties for RTL;
- CJK fallback support;
- stable error codes mapped to translated messages.

---

# 12. Window, PWA, and multi-device behavior

## 12.1 Desktop window

- Regular desktop app, not menubar-only.
- Default 1280×800, resizable.
- Remember window size and panel state per user.
- Optional multiple windows in a later slice.
- A work object may open in a new window without changing backend ownership.

## 12.2 Picture-in-picture

A later first-party PiP window supports:

- one Live session;
- one active test run;
- minimal status and pause/takeover controls;
- always-on-top where supported.

PiP does not expose full inspector or configuration complexity.

## 12.3 FPV PWA

Preserve:

- signed short-lived links;
- read-only, steer, and takeover roles;
- phone/tablet/desktop browser support;
- pinch/zoom/pan;
- feedback input;
- touch annotation and control;
- push alerts;
- reconnect and missed-event replay;
- one active controller with multiple viewers;
- background frame throttling;
- installability;
- PII redaction mode;
- audit trail;
- Focusa Workpoint binding.

The PWA uses a smaller, mobile-appropriate disclosure model but the same capability and authority contracts.

## 12.4 Multi-node and multiple operators

- Cockpit visualizes CRDT and ownership state; it never merges cognition itself.
- Thread owner remains canonical writer.
- Non-owner writes block with ownership-transfer recovery.
- Supporting work does not become active scope.
- A second operator may observe; takeover and write authority remain explicit.

---

# 13. Architecture and contract evolution

The existing Cockpit contracts remain the integration spine. They are extended with governed-execution contracts rather than replaced by UI-local state.

Required additive contracts include:

- `MissionRef` and versioned `MissionContractRef`;
- `CompletionPredicateRef` and evidence sufficiency state;
- `WorkerRef` and `TaskLease`;
- `CapabilityGrantRef`;
- `ActionProposalRef`;
- `ActionReceiptRef`;
- `RiskClass`;
- `ActuatorRef` and route decision;
- `VerificationResult` and `SettlementState`;
- `BudgetSnapshot`;
- `DataClassification` and provenance/egress state.

## 13.1 Preserve the current integration spine

All actions continue through:

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

No production Svelte component calls raw API endpoints directly.

## 13.2 Existing contracts remain

Preserve and extend:

- `OperatingProfile`
- `ApiPlane`
- `AuthorityPlane`
- `ScopeRef`
- `NodeRef`
- `PairingState`
- `ApiAdapter`
- `AdapterRequest`
- `AdapterResult`
- `CardManifest`
- `CockpitEvent`
- `CockpitError`
- `UserSettings`

## 13.3 Authority plane evolution

The current name `browser_execution` is too narrow for the documented breadth of UIAI Engine.

Use an additive migration:

```ts
export type AuthorityPlaneV2 =
  | "uiai_execution"
  | "local_node"
  | "cloud_control_plane"
  | "hosted_ai";

export type ExecutionDomain =
  | "browser"
  | "test"
  | "document"
  | "research"
  | "visual"
  | "media"
  | "workflow"
  | "intelligence"
  | "training"
  | "system";
```

Migration rule:

```text
browser_execution → uiai_execution + execution_domain=browser
```

Existing data remains readable during schema migration.

## 13.4 Cockpit work object

```ts
interface CockpitWorkObject {
  schema: "uaiengine.cockpit.work_object.v1";
  object_ref: string;
  kind:
    | "browser_session"
    | "fpv_share"
    | "test_flow"
    | "test_run"
    | "document"
    | "research_capture"
    | "research_packet"
    | "visual_comparison"
    | "media_job"
    | "workflow_run"
    | "artifact"
    | "evidence"
    | "review_report"
    | "report_snapshot"
    | "workpoint";
  title: string;
  subtitle?: string;
  status: string;
  workspace_id: string;
  capability_id?: string;
  authority_plane: AuthorityPlaneV2;
  execution_domain?: ExecutionDomain;
  scope?: ScopeRef;
  node?: NodeRef;
  artifact_refs: string[];
  evidence_refs: string[];
  created_at: string;
  updated_at: string;
}
```

## 13.5 Workspace manifest

```ts
interface WorkspaceManifest {
  workspace_id: string;
  label: string;
  icon: string;
  group: "work" | "create" | "prove" | "system";
  order: number;
  default_visible: boolean;
  supported_object_kinds: CockpitWorkObject["kind"][];
  capability_ids: string[];
  route: string;
  feature_flag?: string;
  local_only_behavior: "works" | "partial" | "blocked";
  extension_source: "core" | "first_party" | "third_party";
}
```

## 13.6 Capability manifest

```ts
interface CapabilityManifest {
  schema: "uaiengine.cockpit.capability.v1";
  capability_id: string;
  label: string;
  summary: string;
  workspace_ids: string[];
  api_plane: ApiPlane;
  authority_plane: AuthorityPlaneV2;
  execution_domain?: ExecutionDomain;
  adapter_id: string;
  required_scope:
    | "none"
    | "project"
    | "workstream"
    | "thread"
    | "session"
    | "node"
    | "team";
  side_effect_class: SideEffectClass;
  approval_policy: "none" | "conditional" | "required";
  input_schema: object;
  output_schema: object;
  artifact_kinds: string[];
  related_capabilities: string[];
  offline_behavior: "works" | "read_only" | "hidden" | "blocked_with_reason";
  exposure_state:
    | "available"
    | "gated"
    | "adapter_missing"
    | "backend_missing"
    | "experimental"
    | "deprecated";
  normative_source: string;
  parity?: {
    http?: string;
    pi?: string;
    mcp?: string;
    cli?: string;
  };
  focusa_handoff?: {
    preferred_tool: string;
    evidence_type: string;
  };
}
```

## 13.7 Card manifest evolution

Cards remain useful for bounded actions, summaries, and dashboard compositions. Extend them additively:

```ts
interface CardManifestV2 extends CardManifest {
  workspace_ids: string[];
  visibility_tier: "glance" | "work" | "inspect" | "configure" | "developer";
  object_kinds?: CockpitWorkObject["kind"][];
  capability_id?: string;
  primary_action?: string;
  inspector_sections?: string[];
  artifact_kinds?: string[];
}
```

Cards are no longer the top-level navigation taxonomy.

## 13.8 Artifact reference

```ts
interface CockpitArtifactRef {
  schema: "uaiengine.cockpit.artifact_ref.v1";
  artifact_ref: string;
  kind: string;
  title: string;
  media_type?: string;
  sha256?: string;
  byte_size?: number;
  source_object_ref?: string;
  parent_refs?: string[];
  scope?: ScopeRef;
  node?: NodeRef;
  local_path_ref?: string;
  content_url?: string;
  redaction_state: "none" | "redacted" | "blocked" | "public_safe";
  verification_class?: "actual" | "blocked" | "surrogate" | "missing_native";
  created_at: string;
  retention_policy?: string;
}
```

Raw server-local paths must not be exposed to remote/untrusted clients.

## 13.9 Shared job contract

```ts
interface CockpitJob {
  schema: "uaiengine.cockpit.job.v1";
  job_id: string;
  title: string;
  capability_id: string;
  object_ref?: string;
  status:
    | "queued"
    | "validating"
    | "running"
    | "waiting_for_approval"
    | "complete"
    | "failed"
    | "cancelled"
    | "blocked";
  scope?: ScopeRef;
  node?: NodeRef;
  progress?: {
    current?: number;
    total?: number;
    percent?: number;
    stage?: string;
    message?: string;
  };
  input_refs: string[];
  output_refs: string[];
  evidence_refs: string[];
  cancellable: boolean;
  retryable: boolean;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error?: CockpitError;
}
```

## 13.10 Runner adapter

```ts
interface RunnerAdapter {
  runner_id: string;
  label: string;
  supported_targets: string[];
  discover(): Promise<EndpointStatus>;
  validateFlow(flow: unknown): Promise<AdapterResult>;
  startRun(request: AdapterRequest): Promise<AdapterResult<CockpitJob>>;
  pauseRun?(jobId: string): Promise<AdapterResult>;
  cancelRun(jobId: string): Promise<AdapterResult>;
  streamEvents(jobId: string): AsyncIterable<CockpitEvent>;
  collectArtifacts(jobId: string): Promise<CockpitArtifactRef[]>;
}
```

## 13.11 Event model expansion

Preserve existing event kinds and add domain-neutral lifecycle events:

- object_opened / object_closed;
- action_started / progress / completed / failed / blocked / cancelled;
- approval_requested / approved / rejected;
- artifact_created / artifact_compared;
- evidence_captured / evidence_failed;
- live_control_changed / operator_intervened;
- test_step_started / test_step_completed;
- document_derivative_proposed / document_derivative_approved;
- job_queued / job_progress / job_completed;
- session_parked / restored / lease_expiring;
- scope_supporting_work_linked;
- canonical ownership events already defined by the current spec.

All events remain append-only from the UI perspective and scope/node aware.

## 13.12 Canonical capability registry trajectory

Long term, one internal UIAI capability registry should generate or feed:

```text
HTTP discovery
OpenAI tool schema
MCP tool schema
Pi tool metadata
CLI discovery
Cockpit capability manifest
Cockpit command palette
Tool/workflow graph
Docs and parity checks
```

Do not start with a dynamic third-party registry. First prove static first-party manifests, adapters, guards, artifacts, jobs, and smoke tests.

## 13.13 Interactive report contracts

The report system separates the live review object from immutable export artifacts.

```ts
type ReportKind =
  | "run_review"
  | "verification"
  | "failure_anatomy"
  | "decision_brief"
  | "release_proof"
  | "benchmark"
  | "document_review"
  | "stakeholder_update"
  | "incident";

type ReportState =
  | "draft"
  | "assembling"
