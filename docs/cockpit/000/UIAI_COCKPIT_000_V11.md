  | "ready_for_review"
  | "in_review"
  | "changes_requested"
  | "approved"
  | "frozen"
  | "published"
  | "failed"
  | "superseded"
  | "revoked"
  | "expired";

interface CockpitReviewReport {
  schema: "uaiengine.cockpit.review_report.v1";
  report_id: string;
  version: number;
  kind: ReportKind;
  title: string;
  state: ReportState;
  freshness: "live" | "current" | "stale" | "historical";
  audience_profile: string;
  scope: ScopeRef;
  mission_ref?: MissionRef;
  workpoint_ref?: string;
  source_object_refs: string[];
  source_artifact_refs: string[];
  evidence_refs: string[];
  receipt_refs: string[];
  predicate_refs: string[];
  sections: ReportSection[];
  interactions: ReportInteractionManifest[];
  thread_ref?: string;
  redaction_state: "none" | "redacted" | "blocked" | "public_safe";
  integrity?: {
    manifest_sha256?: string;
    signature_ref?: string;
    frozen_at?: string;
  };
  created_by: WorkerRef | { operator_id: string };
  created_at: string;
  updated_at: string;
  supersedes_ref?: string;
}

interface ReportSection {
  section_id: string;
  kind:
    | "executive_summary"
    | "outcome"
    | "finding"
    | "visual_evidence"
    | "comparison"
    | "predicate_status"
    | "timeline"
    | "table"
    | "chart"
    | "code_diff"
    | "document_citation"
    | "recommendation"
    | "limitation"
    | "custom_registered";
  title?: string;
  source_refs: string[];
  block_renderer_id: string;
  data: unknown;
  collapsed_by_default?: boolean;
}

interface ReportInteractionManifest {
  interaction_id: string;
  label: string;
  action_kind:
    | "approve_report"
    | "reject_report"
    | "request_changes"
    | "comment"
    | "annotate"
    | "choose_variant"
    | "review_evidence"
    | "request_recapture"
    | "request_reverification"
    | "create_followup"
    | "rerun_capability"
    | "share"
    | "export";
  capability_id?: string;
  required_scope: CapabilityManifest["required_scope"];
  side_effect_class: SideEffectClass;
  risk_class: RiskClass;
  target_refs: string[];
  context_ref_policy: "report_only" | "selected_sources" | "explicit_selection";
  approval_policy: "none" | "conditional" | "required";
  expected_receipt?: string;
}

interface ReportSnapshotManifest {
  schema: "uaiengine.cockpit.report_snapshot.v1";
  report_ref: string;
  report_version: number;
  artifact_refs: string[];
  source_manifest_ref: string;
  audience_profile: string;
  interaction_state_summary: object;
  redaction_state: CockpitArtifactRef["redaction_state"];
  sha256: string;
  signed_ref?: string;
  created_at: string;
}
```

Architecture rules:

- `CockpitReviewReport` is a versioned presentation/review object, not Mission Kernel state.
- `ReportSnapshotManifest` is immutable and share/export oriented.
- report source references are resolved through adapters and scope guards;
- report blocks are declarative and rendered by registered first-party or approved extension renderers;
- report interactions use `ReportActionController` and the standard guard/router/adapter/event path;
- the report thread stores review activity, while durable mission decisions are promoted explicitly through Focusa;
- a report can be regenerated when sources change, but approved/frozen versions remain immutable;
- source manifests and report exports receive hashes and optional signatures according to risk/audience policy.

First-party capabilities SHOULD include:

```text
uiai.report.compose
uiai.report.refresh
uiai.report.freeze
uiai.report.render
uiai.report.comment
uiai.report.review_action
uiai.report.create_followup
uiai.report.share
uiai.report.revoke_share
uiai.report.export
```

Candidate backend routes are implementation details mapped by the UIAI adapter. The Cockpit UI names capabilities first and MUST NOT bind components directly to route shapes.

Potential later endpoints:

```text
GET /api/cockpit/manifest
GET /api/cockpit/workspaces
GET /api/capabilities
GET /api/capabilities/{id}
GET /api/artifact-viewers
GET /api/commands
```

---

# 14. Capability placement and route preservation

The Cockpit must surface all UIAI Engine capabilities through a sensible user workflow or the Capabilities catalog. The table below preserves current route families while preventing raw-route clutter.

| Current UIAI family | Primary Cockpit placement | Default exposure |
|---|---|---|
| `/api/session/*` | Live; Test Lab | First-class |
| `/api/fpv/*`, `/m/{token}` | Live; share controls | First-class |
| `/api/screenshot/*` | Studio; Live; Test Lab | First-class |
| `/api/share/*`, `/v/{token}` | Evidence/Studio share | Contextual |
| `/api/search*` | Research | First-class |
| `/api/markdown` | Research | First-class |
| `/api/agent/research-packet` | Research; Evidence | First-class |
| `/api/errors`, browser diagnostics/metrics | Inspector; Activity; Nodes & Services | Contextual |
| `/api/tools/*` | Capabilities; Developer Mode | First-class discovery, quiet visually |
| `/api/media/frame/*` | Studio | First-class helper |
| `/api/media/produce`, status, jobs | Studio; Activity | Gated until lifecycle/cancel/artifact proof |
| `/api/critique*` | Studio / Analyze | Gated paid action; metadata visible |
| `/api/ui-reverse` | Studio / Analyze | Gated paid action |
| `/api/section-detect` | Studio / Analyze | Gated paid action |
| `/api/layout-compare` | Studio / Compare | Gated paid action |
| `/api/style-enhance` | Studio / Analyze or Design | Gated paid action |
| `/api/copilot/*` | Contextual assistant panels; future Wirebot compatibility | Gated, visible and attributable |
| `/api/reference/analyze` | Studio / Analyze; Research | Gated paid action |
| `/api/intake/*` | Automations / Intake | Gated mutating workflow |
| `/api/workflow/*` | Automations | Gated mutating workflow |
| `/api/design-system` | Studio / Design | Gated pipeline |
| `/api/content-map` | Studio / Design | Gated pipeline |
| `/api/block-recipes` | Studio / Design; Automations templates | Gated pipeline |
| `/api/comparison` | Studio / Compare | Gated pipeline |
| `/api/migration/*` | Automations / Migration | Preview-first, gated mutation |
| `/api/intelligence/*` | Research/Capabilities; later intelligence workspace if justified | Health/index/search visible by policy; sensitive operations gated |
| `/api/training/*` | Capabilities/System; later specialist surface | Operator/service-token only |
| `/api/memory/*` | Capabilities/System; not generic user memory UI | Sensitive, per-user gated |
| `/api/admin/*`, `/api/usage/*` | Nodes & Services / System | Least-privilege and redacted |
| `/api/captcha/*` | Live advanced recovery; Capabilities/System | Operationally sensitive and gated |
| `/api/2fa/*` | Live auth flow; security settings | Contextual |
| `/api/extension/*` | Nodes & Services / Integrations | Specific extension lifecycle only |
| `/api/events` | Activity/EventBus adapter | Internal live data source |
| `/dashboard` | Legacy/maintenance | Not embedded as product UI |
| Future `/api/documents/*` | Documents | First-class after backend implementation |
| Future test-runner APIs | Test Lab | First-class after runner contracts |

Every row remains discoverable in Capabilities with status and next gate.

---

# 15. Extensibility model

## 15.1 Extension goal

Cockpit must have room to grow without becoming a generic plugin host before its safety model is ready.

## 15.2 Extension levels

### Level A — Declarative capability integration

Provides:

- manifest;
- input/output schemas;
- adapter mapping;
- commands;
- artifact kinds;
- side-effect and approval policy;
- related capabilities;
- Focusa evidence handoff.

Uses standard Cockpit views and forms. No custom code loaded into the UI.

Best for APIs, converters, research adapters, runners, and recipes.

### Level B — Signed first-party module

Provides:

- workspace or specialized view;
- custom artifact viewer;
- inspector panels;
- event rendering;
- trusted native bridges where explicitly granted.

Examples: Documents, Test Lab, Advanced FPV, visual diff viewer.

### Level C — Sandboxed third-party extension, later

Requires:

- signed package;
- explicit permissions;
- isolated runtime;
- capability-only API;
- no raw token access;
- no arbitrary filesystem or shell access;
- per-project enablement;
- audit and uninstall behavior.

Potential permissions:

```text
project.files.read
project.files.write_derivative
browser.session.observe
browser.session.control
runner.execute
artifact.create
network.domain:<host>
focusa.evidence.propose
```

Third-party extensions must not directly write Focusa canonical cognition or bypass Cockpit guards.

## 15.3 Artifact viewer registry

First-party viewers:

- browser/FPV;
- image;
- visual comparison;
- video;
- PDF;
- Office-derived preview;
- Markdown/research capture;
- diagnostics;
- test report;
- JSON/log Developer Mode.

A capability declares output artifact kinds; Cockpit chooses the registered viewer.

## 15.4 Workspace customization

Users may:

- reorder workspaces;
- hide optional workspaces;
- pin favorite capabilities;
- save sidebar filters;
- choose default inspector state;
- create workspace-specific saved views.

Core safety and system access cannot be permanently hidden from search or Help.

---

# 16. Focusa integration throughout the Cockpit

## 16.1 Pervasive, not a separate destination

Focusa context should appear where useful instead of forcing the user to constantly navigate to a “Focusa” area.

Examples:

### Live

- current Workpoint;
- current action;
- active-object hint;
- evidence count;
- prediction status;
- operator intervention update.

### Test Lab

- verification goal;
- prediction awaiting evaluation;
- evidence capture state;
- exact failed proof condition;
- Workpoint checkpoint suggestion.

### Documents

- verified project ownership;
- document purpose in current Workpoint;
- sensitive action approval;
- findings and artifact refs;
- next document action.

### Research

- research goal;
- relevant trajectory gap;
- source evidence state;
- packet and next action.

### Evidence

- Workpoint link;
- verification class;
- prediction/evaluation;
- receipt state;
- future metacognitive lesson proposal.

## 16.2 Focusa cards remain available

Existing Phase 0 cards remain preserved:

- Project Identity
- Project Card
- Workpoint Resume
- Trajectory View
- Tool Doctor
- DXUX Requirement
- Work-loop Status
- Device Pair Status
- Capture Evidence
- Cloud Node Status
- Device Pairing
- AI API Health & Usage
- UIAI Health
- Browser Diagnostics

They are placed contextually:

- Overview: Project/Workpoint/Trajectory summaries;
- Nodes & Services: health, pairing, nodes, tool doctor;
- inspector: scope, evidence, Workpoint context;
- Evidence: capture/link;
- Capabilities: complete card catalog and technical contract.

No existing card is removed.

## 16.3 Scope verification

Before durable evidence or Focusa writes:

1. resolve Project Identity;
2. verify project;
3. resume Workpoint/Trajectory as appropriate;
4. construct explicit ScopeRef;
5. verify authority state;
6. route to owning node;
7. execute through adapter;
8. capture bounded result/evidence.

A session ID remains temporal metadata, not project authority.

## 16.4 Feedback into Focusa

Current supported path:

- stable UIAI artifact/evidence refs;
- browser diagnostics intake;
- Workpoint evidence link;
- active-object hints;
- prediction record/evaluate;
- optional metacognition through explicit Focusa tools;
- Workpoint checkpoint/resume.

Future path:

- generalized document/test/visual verification receipts;
- operation-level provenance;
- synchronized operator interventions;
- Workpoint-bound session/run restoration;
- recapture and evidence diff;
- learned reusable recipes with validity checks.

Focusa stores bounded durable meaning and handles. UIAI retains large binaries, raw recordings, page renders, and detailed execution artifacts according to policy.

---

# 17. Trust, security, consent, and data lifecycle

This section applies the hostile-content and deterministic-policy model from Section 4.12 to every workspace. Websites, documents, emails, tool descriptions, model output, API responses, and files are untrusted inputs until policy and verification say otherwise.

## 17.1 Secure storage

Secrets belong in OS secure storage/keychain:

- pairing tokens;
- Focusa Cloud OAuth tokens;
- AI API keys;
- document signing credentials or references;
- auth-profile secrets.

Settings stores contain only non-secret labels, preferences, and provenance.

## 17.2 Local identity

Preserve the current local-host trust posture:

- keychain-bound token;
- signed Cockpit identity/attestation for sensitive writes where implemented;
- read-only degradation when identity proof fails;
- no raw token values in events.

## 17.3 Cloud consent

Before any Focusa Cloud or AI API call that sends user/project data, disclose:

- capability;
- destination;
- bounded payload summary;
- what is not sent;
- expected output;
- cost if applicable;
- session trust option where policy allows.

## 17.4 Artifact privacy

Artifacts declare:

- local/private/public-safe;
- redaction state;
- retention;
- content URL access policy;
- source scope;
- whether external upload occurred.

Screenshots, videos, documents, diagnostics, HAR-like data, raw prompts, and secrets never upload automatically.

## 17.5 Data classification, provenance, and egress

Every sensitive object or field SHOULD carry classification, mission scope, source provenance, permitted destinations, retention, and redaction state.

Before egress, the adapter/authority path checks:

- destination origin/service;
- mission and Workpoint scope;
- Capability Grant;
- data classes and minimum necessary payload;
- provider/model routing policy;
- cross-origin or cross-mission transfer;
- operator consent requirement;
- receipt and audit requirement.

