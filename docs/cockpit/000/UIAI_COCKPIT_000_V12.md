Raw secrets remain opaque handles. A page request, document instruction, tool description, or model suggestion cannot authorize disclosure.

## 17.6 Structured tools and file safety

Origin-supplied structured tools are treated as untrusted interfaces. Their origin, registration context, schema/version hash, requested data, and side-effect classification are inspected locally; a tool change triggers re-authorization.

Document and browser file flows use isolated directories, explicit selection, MIME/size validation, hashing, provenance, scanning where configured, and deletion/retention rules. A downloaded file does not become trusted instruction or procedural memory.

## 17.7 Retention

Preserve existing configurable retention for telemetry, onboarding events, pairing caches, audit rows, and exports. Extend retention to:

- browser recordings;
- test artifacts;
- document derivatives;
- OCR/semantic intermediate outputs;
- research captures;
- media jobs;
- evidence/proof drafts.

Pinned artifacts survive automated rotation. Deletion shows dependency impact before proceeding.

## 17.8 Export and deletion

Exports are redacted by default and schema-validated on import.

A privacy/reset flow can:

- clear settings;
- clear local telemetry;
- revoke and remove keychain tokens;
- purge selected artifacts by scope;
- retain or delete audit records according to policy;
- explain what remains in external/cloud systems.

---

# 18. Performance and reliability

## 18.1 Perceived performance

Targets from the existing spec remain. Extend them with:

- workspace switch feels immediate using cached view models;
- sidebar/search index warm query under 200ms;
- inspector opens without re-fetching the entire object;
- Live adapts frame rate when backgrounded;
- document pages render progressively;
- test/video/artifact loading is lazy;
- long operations stream progress;
- large logs use virtualization and bounded windows.

## 18.2 Mission budgets and economic routing

Where available, Cockpit surfaces financial, token, model-call, browser-minute, worker-runtime, network, retry, human-attention, and wall-clock budgets from the Mission Contract.

- Hard limits stop, replan, or escalate.
- Estimated cost is shown before paid or high-variance work.
- Actual cost is attached to receipts.
- Routing prefers the least expensive eligible mechanism that meets security, reliability, evidence, and latency requirements.
- The default UI shows only material budget risks; complete accounting remains in Inspect/Developer views.

## 18.3 Resource awareness

Cockpit should explain pressure before failure:

- browser slots;
- session leases;
- test devices/runners;
- document worker queue;
- media worker queue;
- disk retention pressure;
- cloud/API rate limits;
- AI credits/cost.

## 18.4 Graceful degradation

- UIAI unavailable: unrelated Focusa/system views remain usable.
- Focusa unavailable: UIAI work can continue unscoped where safe, clearly marked, with durable Focusa writes blocked.
- Cloud unavailable: Local Only continues.
- AI API unavailable: non-AI local operations continue.
- Document semantic worker unavailable: native PDF reading/rendering can continue where possible.
- Maestro unavailable: other compatible runners remain selectable.
- FPV stream degraded: screenshot polling fallback.

## 18.5 Contract and self-tests

Preserve and extend:

- manifest validation;
- adapter smoke tests;
- route parity checks;
- docs drift checks;
- per-tool contract tests;
- runner availability tests;
- document parser fixture tests;
- artifact viewer tests;
- scope-conflict tests;
- cross-project support-work tests;
- release proof.

---

# 19. Implementation trajectory

## 19.1 Preserve the current foundation slices

### Slice 0 — Repository and app skeleton

Preserve:

- Tauri/Svelte app;
- shell;
- design tokens;
- package/build shape;
- smoke entry point;
- release metadata.

### Slice 1 — Contracts and fixtures

Extend contracts with:

- work objects;
- workspaces;
- capabilities;
- artifacts;
- jobs;
- runner adapters;
- disclosure level;
- approval policy.

### Slice 2 — Real read-only adapters

Implement actual, testable adapters for:

- UIAI health and capabilities;
- sessions/diagnostics;
- Focusa local reads;
- cloud profile reads;
- AI API health/usage;
- artifact discovery.

Replace unconditional smoke assertions with real fixture or live adapter proof.

### Slice 3 — Node graph, routing, and ScopeGuard

Implement:

- explicit ScopeRef resolver;
- authority guard;
- Mac/VPS/remote distinction;
- support-work relation;
- conflict/read-only state;
- session capacity ownership visibility;
- no singleton current Focusa.

### Slice 4 — Pairing and Cloud Profile

Preserve and implement:

- menubar-parity discovery;
- replicated pairing and auto-add paths;
- keychain;
- repair/revoke/refresh;
- OAuth/PKCE;
- consent;
- local identity proof;
- cloud node status.

### Slice 5 — Evidence and proof preview

Implement:

- artifact refs;
- evidence capture/link;
- local proof preview;
- redaction boundary;
- cancellation;
- receipt drafts;
- immutable lineage.

### Slice 6 — Beautiful unified shell

Implement the information architecture in this document:

- task-oriented sidebar;
- Context Control;
- work-object tabs;
- universal inspector;
- Activity Bar;
- global search;
- responsive panels;
- onboarding and Help;
- Apple-like visual quality.

## 19.2 Product capability tracks after/alongside the foundation

### Track A — Live and FPV unification

- native Live workspace;
- PWA share management;
- multi-agent canvas;
- recording/replay/fork;
- operator control states;
- session broker UI;
- PII mode;
- Workpoint binding.

### Track B — Test Lab

- runner contracts;
- UIAI Scenario runner;
- Maestro adapters;
- Tauri WebDriver adapter;
- flow library;
- video/report artifacts;
- baselines;
- accessibility and visual runners;
- Focusa verification handoff.

### Track C — Documents

- document artifact/job contracts;
- import and immutable originals;
- PDF viewer/render/extraction;
- OCR;
- semantic parsing;
- structural PDF operations;
- Office conversion;
- redaction/verification;
- document recipes;
- signature inspection/signing later.

### Track D — Research and Studio

- Research workspace over existing search/Markdown/packet routes;
- Studio over screenshot, compare, critique, reverse, section, layout, design, and media routes;
- clear cost and exposure gates;
- artifact viewer parity.

### Track E — Automations and shared jobs

- scenario/recipe engine;
- job lifecycle;
- progress/cancel/retry;
- intake/workflow/migration previews;
- output/evidence lineage.

### Track F — Capability catalog and extensibility

- complete static capability inventory;
- status/gating display;
- canonical registry generation later;
- declarative integration manifests;
- signed first-party modules;
- sandboxed third-party extensions only after the permission model is proven.

### Track G — Governed mission, authority, verification, and settlement

- Mission Deck over Focusa Workpoint/Trajectory first;
- additive Mission Contract and Completion Contract references;
- explicit mission/task/action lifecycle rendering;
- WorkerRef, TaskLease, CapabilityGrant, ActionProposal, and ActionReceipt contracts;
- R0–R5 risk policy and capability-lease approvals;
- actuator-neutral Action Router and route receipts;
- pre/postcondition, idempotency, retry classification, compensation, reconciliation, and dead-letter behavior;
- independent verification and predicate-linked evidence sufficiency;
- provisional completion and asynchronous settlement;
- data classification, provenance, credential mediation, and origin-bound structured-tool security;
- budget and cost-per-verified-outcome telemetry;
- worker identity, delegation, incident response, red-team, and maturity/conformance reporting.

This track must reuse Focusa as Mission Kernel and UIAI as execution/evidence plane. Cockpit is the Mission Experience and must not implement a competing canonical kernel.

### Track H — Interactive review reports and Report Canvas

- ReviewReport, ReportSection, ReportInteractionManifest, thread, and snapshot contracts;
- static first-party report templates and declarative block renderers;
- report composition from Mission/Workpoint, receipts, evidence, artifacts, tests, documents, comparisons, and activity;
- Report Canvas using the universal shell, inspector, progressive disclosure, and design system;
- actual-capture visual provenance and immutable annotation overlays;
- decision, comment, annotation, variant, recapture, reverification, and follow-up interactions;
- ReportActionController using the same guards, router, adapters, events, and receipts;
- report lifecycle, freshness, versioning, freezing, supersession, revocation, and expiry;
- audience profiles, public-safe redaction, version-bound shares, and export snapshots;
- report search/indexing and cross-workspace creation entry points;
- deterministic report composer with agent-authored summaries clearly separated from evidence;
- Focusa linkage for durable review outcomes and follow-up Workpoints without creating parallel mission state;
- signed/hashable report manifests for release, audit, incident, and high-trust external use.

The initial release should ship Review Reports as an Evidence saved view and contextual action, not as a new default top-level workspace. Promote Reports to a permanent sidebar item only if observed use demonstrates that it is a frequent primary destination rather than a cross-workspace review object.

## 19.3 Build order inside every product track

```text
workflow definition
  → contract
  → auth/scope/redaction/side-effect policy
  → adapter/runner
  → fixtures and smoke
  → job/artifact/evidence behavior
  → basic UI
  → polished UI
  → docs/parity/release proof
```

No polished surface should hide an undefined backend contract.

---

# 20. Testing, release, and operational requirements

All current detailed requirements remain preserved, including:

- contract, unit, adapter, E2E, accessibility, and performance tests;
- coverage targets;
- first-run, pairing, auto-add, repair, scope-conflict, and multi-node flows;
- macOS-first signed/notarized Tauri release;
- stable/preview/dev channels;
- checksums, release metadata, bundle manifest, smoke report, and audit row;
- updater signing and emergency disable;
- rollback and last-known-good behavior;
- settings migration and backup;
- release notes template;
- dependency pinning and supply-chain policy;
- local telemetry and support bundle;
- GitHub Actions plus documented local/GitLab/Bitbucket alternatives;
- internationalization and accessibility smoke;
- no release without a green real smoke harness.

Add track-specific release gates:

### Live

- stream fallback;
- control authorization;
- replay/audit correctness;
- share expiry/revoke;
- session ownership/park/restore.

### Test Lab

- runner discovery;
- report/video artifact integrity;
- cancellation;
- failed-step correlation;
- native Tauri and Maestro fixture coverage.

### Documents

- hostile PDF fixtures;
- extraction/citation fixtures;
- OCR and table fixtures;
- redaction verification;
- immutable original/derivative lineage;
- signed-document warning;
- conversion reproducibility.

### Studio and Automations

- paid-action cost guard;
- deterministic artifact references;
- retry/cancel behavior;
- recipe preview and approval;
- migration rollback/preview.

### Interactive reports

- schema validation and source-reference resolution;
- report generation with missing, stale, contradictory, and unauthorized sources;
- actual-capture provenance and illustrative-image labeling;
- immutable original visual plus annotation overlay integrity;
- report lifecycle, versioning, freezing, supersession, revocation, and expiry;
- report freshness change after source evidence updates;
- comment/thread permissions, revision history, and resolution links;
- report approval versus outcome acceptance versus follow-up authorization semantics;
- interaction manifests routed through guards with no direct script/tool execution;
- follow-up Workpoint/task creation with bounded selected context;
- public-safe audience policy, visual redaction, share expiry, and version-bound links;
- HTML/PDF/Markdown/JSON/ZIP export parity and static-widget state rendering;
- custom block renderer sandbox, CSP, accessibility, responsive, and reduced-motion tests;
- report hash/signature and source-manifest reproducibility;
- cross-workspace creation from Live, Test Lab, Documents, Research, Studio, Automations, and Evidence.

---

## 20.1 Governed-execution and security test categories

In addition to the existing workspace tests, production claims require fixtures for:

1. Mission Contract compilation and versioned amendment;
2. Completion predicates and missing-evidence behavior;
3. submitted versus accepted versus verified versus settled state;
4. capability expiry, revocation, maximum-use/value, and action-boundary recheck;
5. actuator selection, prohibited actuator, and fallback without authority expansion;
6. precondition failure and postcondition failure;
7. idempotency and ambiguous-result reconciliation before retry;
8. dead-letter and human-review recovery;
9. concurrent workers, lease expiry, ownership transfer, stale state, and shared budgets;
10. prompt injection in visible text, hidden DOM, accessibility labels, images, documents, downloaded files, and tool descriptions;
11. lookalike redirects, form substitution, fake confirmation pages, tool manifest replacement, and cross-origin data requests;
12. credential expiration, origin-bound credential release, and secret-handle non-disclosure;
13. cross-mission data isolation and egress-policy denial;
14. independent verifier disagreement and contradiction blocking settlement;
15. asynchronous settlement reversal or failure;
