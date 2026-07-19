- metric cards and failure-class matrices;
- recommendations and follow-up options.

Custom visualization freedom is provided through registered **ReportBlockRenderer** manifests, not arbitrary embedded HTML/JavaScript. First-party and approved extension renderers inherit Cockpit tokens, accessibility, motion, export, security, and responsive contracts. Unknown script, iframe, remote embed, or executable content is blocked by default.

#### Interactive review actions

Interactivity is the differentiator, but it remains governed.

Core review actions MAY include:

- Approve report for its stated audience;
- Reject report;
- Request changes;
- Add or resolve a comment;
- Annotate a visual or document region;
- choose a preferred variant;
- accept or reject a proposed baseline;
- mark evidence as reviewed;
- request recapture or re-verification;
- create a follow-up Workpoint/task/mission proposal;
- rerun a bounded test or recipe;
- open the source Live session, Test run, Document, comparison, or artifact;
- export or request a public-safe share.

Every interactive control is a declarative `ReportInteractionManifest` routed through:

```text
Report widget
  → ReportActionController
  → ScopeGuard / AuthorityGuard / ConsentGuard
  → Action Proposal or review event
  → NodeRouter / Adapter / Orchestrator
  → Receipt / Event / Focusa handoff
  → refreshed Report version or source projection
```

No report button executes arbitrary prompts, code, tools, or browser actions directly. Each widget declares capability ID, required scope, side-effect class, risk class, destination, context refs, idempotency/reconciliation posture, and expected receipt.

#### Review semantics

The UI MUST distinguish these decisions:

- **Approve report:** reviewer accepts this version for its stated audience or workflow stage.
- **Accept outcome:** reviewer proposes that the mission outcome be accepted; Focusa/verifier rules still determine completion or settlement.
- **Authorize follow-up:** grants bounded authority for a new task or action.
- **Request changes:** records review feedback and optionally creates a proposed follow-up Workpoint.
- **Mark evidence reviewed:** records human review; it does not automatically make evidence independently verified.
- **Rate usefulness:** product-quality feedback only; it cannot mutate mission truth.

Comments, reactions, ratings, and visual annotations carry no execution authority by themselves. A comment such as “go ahead and deploy” is untrusted review text until converted into an explicit typed proposal and authorized at the action boundary.

#### Follow-up work and context carryover

A reviewer can create follow-up work without re-explaining context because the report supplies bounded references:

- source report and version;
- mission/Workpoint and scope;
- selected findings, comments, and visual regions;
- relevant receipts and evidence refs;
- unresolved predicates or contradictions;
- proposed objective and expected evidence.

The follow-up flow creates or proposes a real Workpoint/task through Focusa/orchestration contracts. It MUST NOT treat the report’s layout or prose as the canonical task definition.

#### Versioning, lifecycle, and freezing

Review Reports are durable, searchable, and versioned.

Recommended lifecycle:

```text
DRAFT
→ ASSEMBLING
→ READY_FOR_REVIEW
→ IN_REVIEW
→ CHANGES_REQUESTED
→ APPROVED
→ FROZEN
→ PUBLISHED
```

Exceptional states include `FAILED`, `SUPERSEDED`, `REVOKED`, and `EXPIRED`.

Rules:

- Editing report composition, summary, annotations, or audience creates a new version or append-only delta.
- Source receipts and evidence are referenced, never overwritten.
- An approved/frozen version is immutable.
- A newer version supersedes rather than rewrites an approved version.
- Mission evidence changing after report generation marks the report **stale** until recomposed or explicitly accepted as a historical snapshot.
- Approval of a report version does not retroactively approve later source changes.
- Public shares identify the exact frozen version.

The Cockpit supports both:

- **Live report projection:** updates from current canonical sources and clearly displays freshness.
- **Frozen report snapshot:** immutable review/share artifact with hashes and source manifest.

#### Threaded review

Each report MAY have a persistent review thread. Thread entries include author, role, time, scope, target section/region, status, and optional resolution reference.

- Threads are append-only from the client perspective.
- Editing a comment preserves revision history where policy requires it.
- A resolved comment links to the action, report version, receipt, or evidence that addressed it.
- External reviewers receive only the sections and thread visibility allowed by audience policy.
- Report threads do not replace Focusa decisions, Workpoint checkpoints, or incident records; important outcomes are promoted through explicit typed actions.

#### Audience and sharing profiles

Report generation starts with an audience profile:

| Audience profile | Typical visibility | Allowed interaction |
|---|---|---|
| **Operator internal** | full scoped technical view | comments, decisions, follow-up proposals, reruns |
| **Engineering review** | evidence, diagnostics, diffs, bounded technical details | comments, request changes, baseline/variant decisions |
| **Executive/stakeholder** | summary, outcome, selected visuals, material risks | approve/reject/request changes, comments |
| **Client review** | public-safe summary, selected evidence, deliverables | approve/reject/request changes, comments |
| **Public proof** | frozen redacted evidence and receipt summary | normally view-only or bounded feedback |
| **Audit/export** | immutable machine-readable manifest plus approved human view | no live execution controls |

Sharing is explicit, revocable, scoped, expiring where appropriate, and version-bound. The share preview shows exactly what leaves the machine, which interactions remain enabled, and which underlying sources remain private.

#### Export formats

The canonical interactive experience remains a Cockpit/approved web Report Canvas. Export MAY produce:

- frozen HTML/web snapshot;
- PDF;
- Markdown;
- JSON manifest;
- ZIP evidence bundle;
- public-safe hosted snapshot;
- presentation or document packet generated through Documents/Studio.

Static exports replace widgets with their current state, decision history, and a link or reference to the interactive source when policy permits. A PDF MUST NOT imply that disabled interactive controls remain actionable.

#### Report generation pipeline

Agents propose content; a deterministic composer validates and renders it.

```text
Select report family and audience
  → gather canonical source refs
  → verify scope and source freshness
  → validate evidence integrity and sufficiency labels
  → compose typed sections/blocks
  → generate bounded agent assessment and recommendations
  → apply redaction and audience policy
  → validate interaction manifests
  → render Report Canvas
  → human review
  → freeze/hash/sign where required
  → share/export/link to Focusa
```

The composer SHOULD fail or degrade visibly when source refs are missing, stale, contradictory, unauthorized, or non-public-safe. It MUST NOT fill missing evidence with generated imagery or confident prose.

#### Cross-workspace entry points

- **Overview/Mission Deck:** latest report, pending decision, or stale-report warning.
- **Live:** create Run Review or Incident Report from a session/event range.
- **Test Lab:** create Verification, Failure Anatomy, Benchmark, or Release Proof report.
- **Documents:** create Document Review, approval packet, or public-safe client report.
- **Research:** create evidence-backed research brief from approved captures.
- **Studio:** create Decision Brief from comparisons, critiques, and variants.
- **Automations:** generate/update a report at checkpoint, completion, failure, or settlement.
- **Evidence:** report library, version history, review, freezing, integrity, sharing, and export.
- **Activity:** review events, comments, approvals, shares, follow-up dispatch, and report jobs.

#### Success metrics

Measure report value using:

- median time from report open to informed decision;
- evidence sufficiency and source-integrity rate;
- percentage of reports reopened without context re-explanation;
- successful follow-up completion using carried context;
- comments resolved with linked receipts/evidence;
- stale/superseded report detection rate;
- public-share redaction failures prevented;
- reviewer correction and disagreement rate;
- report generation/render cost and latency;
- reduction in false-done acceptance;
- usefulness ratings, kept separate from mission truth.

### Comparison and recapture

Evidence supports:

- recapture from prior context;
- compare to current state;
- diff two evidence points;
- mark a screenshot as golden;
- evaluate a prediction;
- attach or detach from a Workpoint;
- generate a public-safe proof preview.

### Proof boundary

Cloud publishing always shows:

- what will be uploaded;
- what remains local;
- redaction status;
- public-safe state;
- destination;
- cancellation behavior;
- resulting receipt.

## 7.9 Activity

Activity combines running work, approvals, notifications, event history, and jobs in one place to avoid a proliferation of top-level utilities.

### Segments

- **Now** — running actions and jobs.
- **Approvals** — pending operator decisions.
- **History** — append-only bounded events.
- **Jobs** — queued, running, completed, failed, cancelled, and blocked jobs.
- **Notifications** — unread and archived notices.
- **Audit** — local redacted audit and release/proof records.

### Job detail

Activity preserves meaningful lifecycle transitions rather than only the latest status. Task and action timelines distinguish proposed, authorized, submitted, accepted, verified, settled, retryable failure, authorization failure, semantic failure, external failure, irreversible/unknown failure, and dead-letter review.

A job view shows:

- human purpose;
- stage and progress;
- inputs;
- current operation;
- cancellation posture;
- approval gate;
- outputs;
- logs/diagnostics;
- evidence status;
- retries;
- tool versions;
- resource use.

## 7.10 Nodes & Services

Nodes & Services contains technical orientation that should not dominate ordinary work.

### Sections

- Nodes
- UIAI Engine
- Focusa Local
- Focusa Cloud
- AI API
- Pairing & Devices
- Capacity
- Sync
- Updates & Compatibility

### Node graph

Display local Mac, VPS, remote, relay, and cloud-only nodes with:

- machine/node ID;
- endpoint and transport;
- health;
- version;
- role and thread ownership;
- sync state;
- backlog/conflict;
- pairing state;
- capabilities;
- last seen.

### Workers and leases

Nodes & Services exposes attributable workers and resource ownership:

- stable worker/client ID;
- runtime/model and version;
- owning principal and organizational role;
- current mission/task lease;
- granted capability classes and expiration;
- health, priority, last heartbeat, and attestation where available;
- delegation chain and revocation state;
- supported execution domains and actuators.

Anonymous or unverified workers cannot receive high-risk grants.

### Session capacity

Show:

- browser processes/pages;
- persistent sessions;
- leases;
- queue;
- active owners/scopes;
- parkable sessions;
- media/document/test worker capacity;
- recommended recovery.

### Services

Each service begins with human status and a single recovery action. Endpoint, credential label, headers, raw health, and contract details are deeper disclosure.

## 7.11 Capabilities

Capabilities guarantees that everything implemented in UIAI Engine can be found without forcing everything into the sidebar.

### Catalog behavior

Search and filter by:

- task;
- workspace;
- status;
- source plane;
- side effect;
- scope;
- local/cloud;
- license;
- experimental state;
- artifact type.

### Capability states

- Available
- Needs project scope
- Needs pairing
- Needs configuration
- Needs license/credits
- Needs cloud profile
- Needs operator approval
- Experimental
- Backend available, Cockpit adapter missing
- Documented, backend not implemented
- Disabled by policy
- Deprecated

### Action routing and actuator policy

Capabilities can advertise multiple actuator implementations. Cockpit presents one outcome-level capability while the Action Router selects among connector, API, MCP, structured browser tool, DOM/accessibility, visual interaction, or human takeover according to policy.

The detail view exposes:

- eligible and prohibited actuators;
- selected actuator and reason;
- origin/registration context for structured tools;
- reliability, cost, latency, evidence, and reversibility posture;
- fallback path and confirmation that fallback does not expand authority;
- current manifest/schema hash and freshness where applicable.

### Capability detail

Default:

- human purpose;
- where it appears;
- requirements;
- inputs;
- outputs;
- safety/side effect;
- related capabilities;
- **Open workspace** or **Configure**.

Developer Mode adds:

- capability ID;
- normative source;
- adapter;
- API plane;
- route/tool mapping;
- Pi/MCP/CLI parity;
- contract schema;
- smoke proof;
- known gaps.

## 7.12 Settings and Help

### Settings groups

- General
- Appearance
- Workspaces & Sidebar
- Scope & Project Defaults
- Nodes & Connections
- Pairing & Security
- Test Runners
- Documents
- AI & Costs
- Evidence & Retention
- Notifications
- Accessibility
- Developer
- Updates

Use forms with summary rows; open advanced subpages only when selected. Avoid one immense settings screen.

### Help

- first-use tour;
- keyboard shortcuts;
- common workflows;
- capability help;
- troubleshooting;
- local vendored docs;
