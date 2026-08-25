# UIAI-COCKPIT-007 — Governed Notebook Runtime, Jupyter Integration, and Computational Evidence

**Status:** Proposed normative amendment v1.0; documentation-first; implementation not implied  
**Date:** 2026-08-01  
**Depends on:** UIAI-COCKPIT-000 through 006; Focusa temporal, prediction, runtime-constitution, evidence, and future mathematical/physical fabric contracts  
**Companions:** UIAI-COCKPIT-007-C01 through C05

## 0. Constitutional law

A notebook is not trustworthy because its cells ran. A numerical result is not verified because a plot rendered. A simulation is not physical observation. A successful browser interaction is not proof of computational correctness. A saved `.ipynb` file is not a reproducibility record.

Notebook integration is complete only when all applicable layers are present and truthfully distinguished:

1. a typed notebook, kernel, cell, environment, dataset, output, and execution contract;
2. a structured Jupyter adapter for contents, sessions, kernels, channels, and outputs;
3. a browser adapter for JupyterLab interactions that are not representable structurally;
4. bounded execution under resource, authority, network, and secret controls;
5. exact temporal and causal lineage for every consequential execution attempt;
6. immutable output artifacts and machine-readable receipts;
7. explicit separation of exploratory, reproducible, and verification execution;
8. Focusa bindings for mathematical meaning, prediction authority, physical claims, evidence, learning, and settlement;
9. actual compiled Cockpit UI code with complete operational states;
10. interaction tests and inspected screen-capture evidence from the exact build SHA.

No implementation may collapse the following distinctions:

```text
cell executed
≠ output is valid
≠ analysis is reproducible
≠ claim is supported
≠ prediction is calibrated
≠ simulation matches reality
≠ physical conclusion is verified
≠ control action is authorized
≠ outcome is settled
```

## 1. Product purpose

UIAI Engine SHALL provide a provider-neutral **Governed Notebook Runtime** that can operate Jupyter Server and JupyterLab while remaining compatible with future notebook providers. The runtime turns computational notebooks into attributable, reproducible, evidence-producing work surfaces without making UIAI Engine or Jupyter the canonical mission, mathematical, epistemic, or physical authority.

Primary outcomes:

- create, open, edit, checkpoint, execute, interrupt, restart, and verify notebooks;
- stream cell outputs and kernel status into Cockpit;
- preserve cell-level temporal, causal, environment, dataset, and artifact lineage;
- support symbolic mathematics, numerical methods, statistics, simulation, visualization, and experiment analysis;
- coordinate structured Jupyter APIs with browser/JupyterLab actuation;
- produce Focusa-compatible evidence and computational receipts;
- prevent hidden state, stale data, environment drift, and attractive but unsupported conclusions from appearing authoritative;
- support human, agent, and mixed-initiative computational work.

## 2. Non-goals

This amendment does not:

- make notebook order the canonical task graph;
- treat Jupyter kernels as trusted arbitrary-shell authority;
- allow notebook output to directly authorize deployment, financial action, physical actuation, or other consequential side effects;
- promote simulation output into physical truth;
- promote one successful numerical run into a learned universal rule;
- expose private chain-of-thought as notebook evidence;
- permit arbitrary remote iframes, scripts, widgets, or extensions without registration and policy;
- claim Focusa mathematical or physics fabric implementation merely because notebook execution exists;
- require a permanent top-level sidebar item for all users.

## 3. Plane separation

```text
Focusa
  owns mission scope, Workpoints, mathematical/physical semantics,
  temporal authority, prediction authority, evidence linkage,
  learning candidates, verification posture, and settlement
                    │
                    ▼
UIAI Governed Notebook Runtime
  owns provider adapters, kernel/session lifecycle, bounded execution,
  output streaming, artifact capture, resource accounting, and receipts
                    │
          ┌─────────┴─────────┐
          ▼                   ▼
Structured Jupyter        UIAI browser plane
REST/WebSocket APIs       JupyterLab/UI inspection
          │                   │
          └─────────┬─────────┘
                    ▼
Cockpit Computational Workbench
  renders notebook state, variables, outputs, assumptions,
  predictions, verification, evidence, reproducibility, and activity
```

Authority rules:

- Focusa owns durable mission and epistemic state.
- UIAI Engine owns transient execution and local operational state.
- Jupyter owns kernel protocol behavior and notebook-format persistence within its configured server.
- Cockpit owns presentation and interaction state, never canonical truth.
- A notebook may reference Focusa objects; it SHALL NOT duplicate or silently rewrite them.

## 4. Focusa organism-loop relationship

The runtime supports the computational portion of a broader agent loop:

```text
PERCEIVE
→ CLASSIFY
→ GROUND
→ UNDERSTAND
→ QUANTIFY
→ CHALLENGE
→ DECIDE
→ ACT
→ OBSERVE CONSEQUENCES
→ SETTLE
→ LEARN
→ CONSOLIDATE
→ PROJECT
→ REPEAT
```

Notebook execution primarily serves `QUANTIFY`, `CHALLENGE`, `OBSERVE CONSEQUENCES`, `SETTLE`, and `LEARN`. It may assist other stages, but it SHALL NOT become the sole authority for any stage.

## 5. Planned Focusa fabric bindings

The following identifiers describe proposed or separately governed Focusa work and SHALL be treated as external contracts until implemented and pinned:

| Proposed Focusa family | Notebook relationship |
|---|---|
| **137B — Temporal lineage** | exact time, clock, interval, validity, horizon, supersession, and causal lineage for measurements, derivations, simulations, forecasts, and experiments |
| **138B — Adversarial forecasting** | primary forecast, adversarial model, matched control, source ablation, calibration, scoring, and leakage tests |
| **152 — Mathematical fabric** | symbols, expressions, equations, assumptions, transformations, solvers, derivations, proofs, counterexamples, and quantitative conclusions |
| **152A — Probability/statistics fabric** | distributions, sampling, uncertainty, inference, calibration, causal analysis, scoring, residuals, and sensitivity |
| **152B — System integration** | operation registry, Workpoints, Evidence, Receipts, persistence, replay, tools, UI surfaces, and reducer settlement |
| **152C — Mathematical learning/discovery** | candidate relationships, reproduction, alternate derivations, counterexample search, applicability, and governed promotion |
| **153 — Physics fabric** | analytical models, simulations, dimensional quantities, measurement comparison, and domain models |
| **153A — Physical verification/control** | simulation-to-shadow promotion, sensor comparison, skill preconditions, safety, authority reduction, and outcome verification |

UIAI SHALL expose `contract_only` or `implementation_open` states for unavailable Focusa bindings. It SHALL NOT synthesize unsupported operations.

## 6. Canonical notebook object model

The runtime SHALL define stable typed objects rather than treating `.ipynb` as an opaque blob.

### 6.1 Core objects

```text
NotebookServer
NotebookDocument
NotebookRevision
NotebookCheckpoint
NotebookEnvironment
KernelSpecification
KernelSession
NotebookSession
CellDefinition
CellRevision
CellAttempt
ExecutionDependency
ParameterSet
AssumptionSet
DatasetSnapshot
EquationDefinition
MathematicalExpression
SolverConfiguration
SimulationRun
ExperimentRun
MeasurementSeries
ResultArtifact
VisualizationArtifact
VerificationFinding
ReproducibilityManifest
NotebookEvidenceReceipt
```

### 6.2 Identity rules

- `notebook_id` remains stable across notebook revisions.
- `notebook_revision_id` changes whenever canonical notebook source changes.
- `cell_id` is stable across non-destructive edits when the provider preserves identity.
- `cell_revision_id` changes whenever source, declared metadata, parameters, or semantic role changes.
- every execution creates a new immutable `attempt_id`.
- re-execution SHALL NOT overwrite prior attempts or outputs.
- a checkpoint is not automatically a reproducible revision.
- provider execution counts are display metadata, not globally unique attempt identity.

## 7. Notebook format and metadata

The runtime SHALL support `.ipynb` format without relying on undocumented provider behavior. UIAI metadata SHALL be namespaced and round-trip safely when policy permits.

Recommended metadata namespace:

```json
{
  "metadata": {
    "uiai": {
      "schema": "uiai.notebook_metadata.v1",
      "notebook_id": "...",
      "scope_ref": {},
      "focusa_refs": [],
      "environment_ref": "...",
      "default_execution_mode": "exploratory",
      "evidence_policy": "bounded",
      "semantic_profile": "math | statistics | physics | general"
    }
  }
}
```

Cell metadata MAY declare:

- stable cell identity;
- semantic role;
- expected dependencies;
- declared inputs and outputs;
- assumptions;
- units and dimensional expectations;
- execution policy;
- timeout and resource class;
- evidence policy;
- verification policy;
- Focusa references;
- public/redaction posture.

Unknown metadata SHALL be preserved unless an explicit migration says otherwise.

## 8. Cell semantic roles

Cell position and markdown prose are insufficient to establish computational meaning. Consequential cells SHALL support an explicit semantic role:

```text
definition
assumption
observation
measurement_import
data_transform
derivation
transformation
solver_invocation
simulation
visualization
prediction
scoring
verification
counterexample
sensitivity_analysis
control_proposal
conclusion
narrative
```

Roles are descriptive and policy-bearing. They do not make content correct.

Examples:

- `measurement_import` requires source and time lineage.
- `prediction` requires question, information-set, horizon, and probability/confidence bindings when Focusa prediction authority is active.
- `simulation` must be labeled separately from observation.
- `control_proposal` cannot directly actuate hardware.
- `conclusion` must reference supporting attempts and verification posture.

## 9. Structured Jupyter adapter

The initial provider SHALL support Jupyter Server/JupyterLab through documented interfaces.

### 9.1 Adapter domains

- server discovery and health;
- authentication and capability negotiation;
- contents listing, reading, writing, renaming, deleting, and checkpointing;
- session creation, listing, reconnection, and deletion;
- kernel specification discovery;
- kernel start, interrupt, restart, reconnect, and shutdown;
- WebSocket channel connection;
- execution request, input request, status, stream, error, display data, update display data, and result messages;
- comm/widget policy and registered renderer negotiation;
- terminal access only when explicitly enabled by a separate bounded capability;
- extension and server-version inspection.

### 9.2 Required provider posture

Every server connection SHALL expose:

- provider kind and version;
- base URL and transport;
- TLS posture;
- authentication method and credential label, never raw secret;
- capability set;
- extension set where observable;
- kernel specifications;
- content roots and permissions;
- health and last successful probe;
- local, Tailscale, remote, or cloud routing classification;
- compatibility warnings.

### 9.3 Authentication

Supported authentication MAY include token, password exchange, OAuth/session, reverse-proxy identity, or local trusted bridge. Credentials SHALL be stored through approved secure storage, redacted from logs, and scoped to the server.

A notebook document or output SHALL never persist raw server tokens, cookies, Authorization headers, or kernel connection secrets.

## 10. Browser/JupyterLab adapter

The structured adapter is preferred for reliable operations. The browser adapter is required for rich interactions not fully represented by the protocol, including:

- JupyterLab extension surfaces;
- interactive widgets and plots;
- debugger interfaces;
- drag-and-drop and visual layout;
- rich media inspection;
- browser-only authentication;
- human takeover;
- screenshot and video evidence;
- visual regression and accessibility checks.

Action routing law:

```text
Outcome request
→ capability and authority check
→ structured adapter when sufficient
→ registered browser/JupyterLab actuation when necessary
→ no-authority-expanding fallback
→ receipt and evidence
```

The browser fallback SHALL NOT silently broaden scope, access a different server, use a more privileged identity, or bypass notebook execution policy.

## 11. Execution modes

### 11.1 Exploratory mode

Purpose: fast human/agent investigation.

Characteristics:

- mutable cells;
- existing kernel state allowed;
- out-of-order execution allowed but visible;
- outputs are provisional;
- hidden-state warnings are active;
- evidence can be captured but cannot claim reproducibility by default;
- destructive or external side effects remain separately governed.

### 11.2 Reproducible-run mode

Purpose: execute a frozen notebook revision under declared inputs.

Required controls:

- immutable notebook revision;
- clean kernel/session;
- frozen or content-addressed environment manifest;
- pinned dataset snapshots or explicit external-data immutability claim;
- declared parameters and assumptions;
- deterministic seed policy or declared nondeterminism;
- topologically valid or explicitly ordered execution plan;
- bounded resources and timeout;
- complete output capture;
- hashes for source, inputs, environment, and results;
- reproducibility receipt.

A reproducible run SHALL fail or degrade visibly when dependencies, data, packages, kernels, secrets, or external services cannot be reproduced.

### 11.3 Verification mode

Purpose: independently challenge a result or claim.

Verification SHALL use a distinct execution context appropriate to the claim, such as:

- clean independent kernel;
- separate environment or dependency resolution;
- alternate solver or numerical method;
- alternate implementation or model;
- dimensional and unit checking;
- property-based tests;
- invariant checks;
- counterexample search;
- perturbation and sensitivity analysis;
- source/data ablation;
- matched-control evaluation;
- simulation-to-measurement comparison;
- independent human review.

Verification output SHALL state its independence class and limitations. Re-running the same cell in the same hidden state is not independent verification.

## 12. Cell execution state machine

```text
DRAFT
→ READY
→ QUEUED
→ DISPATCHED
→ RUNNING
→ AWAITING_INPUT | STREAMING_OUTPUT
→ SUCCEEDED | FAILED | INTERRUPTED | TIMED_OUT | CANCELLED
→ OUTPUT_CAPTURED
→ PROVISIONAL | REPRODUCIBLE | VERIFIED | DISPUTED | SUPERSEDED
```

Additional conditions:

- `BLOCKED_AUTHORITY`
- `BLOCKED_RESOURCE`
- `BLOCKED_DEPENDENCY`
- `BLOCKED_SECRET`
- `BLOCKED_NETWORK`
- `KERNEL_LOST`
- `SERVER_OFFLINE`
- `OUTPUT_STALE`
- `ENVIRONMENT_DRIFT`
- `DATA_STALE`
- `AMBIGUOUS_SIDE_EFFECT`

The client SHALL preserve the latest known state and recovery recommendation without inventing completion.

## 13. Temporal and causal lineage

Every consequential execution attempt SHALL distinguish applicable times:

```text
event_time
observation_time
ingestion_time
execution_requested_at
execution_started_at
execution_completed_at
effective_time
valid_from / valid_to
forecast_origin_time
forecast_horizon_start / forecast_horizon_end
resolution_time
verification_time
settlement_time
```

Required clock posture:

- UTC wall-clock timestamp with declared precision;
- monotonic start and end values for durations;
- clock source and synchronization posture where available;
- uncertainty/error bound where material;
- causal parent attempt/event references;
- no conversion from provider execution count into temporal authority.

For market, control, scientific, or other high-temporal-integrity use, the runtime SHALL preserve microsecond-or-better source values when supplied and SHALL NOT silently round them. UI display may abbreviate while inspectors and receipts retain full precision.

## 14. Cell-attempt record

Each attempt SHALL produce a record equivalent to:

```yaml
schema: uiai.notebook_cell_attempt.v1
notebook_id: "..."
notebook_revision_id: "..."
cell_id: "..."
cell_revision_id: "..."
attempt_id: "..."

scope_ref:
  project_root: "..."
  continuity_id: "..."
  workpoint_id: "..."

semantic_role: derivation
language: python
kernel_ref: "..."
environment_ref: "..."
execution_mode: reproducible

execution_requested_at: "..."
execution_started_at: "..."
execution_completed_at: "..."
monotonic_started_ns: 0
monotonic_completed_ns: 0

source_hash: "sha256:..."
input_dataset_refs: []
parameter_set_ref: "..."
assumption_set_ref: "..."
dependency_attempt_refs: []

stdout_ref: "..."
stderr_ref: "..."
display_output_refs: []
result_refs: []

exit_status: succeeded
interrupted: false
timeout: false
resource_usage_ref: "..."

evidence_refs: []
receipt_ref: "..."
verification_status: provisional
canonical_status: noncanonical
```

Provider-specific fields SHALL be isolated under a namespaced extension object.

## 15. Kernel and session lifecycle

### 15.1 Kernel authority

A kernel is an execution resource, not an identity or principal. Every kernel session SHALL have:

- stable UIAI kernel-session ID;
- provider kernel ID;
- owning principal and scope;
- notebook/session leases;
- environment and kernel-spec refs;
- creation and last-activity times;
- current execution owner;
- resource limits;
- network policy;
- secret policy;
- idle/park/shutdown policy;
- recovery and reconnect posture.

### 15.2 Concurrency

The runtime SHALL prevent ambiguous cross-writes and hidden-state races through:

- per-kernel execution serialization unless the provider explicitly supports safe parallelism;
- notebook revision checks before writes;
- lease or ownership checks;
- idempotency keys for submitted execution requests where feasible;
- deduplication of repeated transport messages;
- explicit handling of lost acknowledgment and unknown execution status.

### 15.3 Interruption and shutdown

Interrupt, restart, and shutdown are distinct operations. The UI SHALL disclose:

- which execution attempts may be affected;
- whether outputs may be partial;
- whether in-memory state will be lost;
- whether external side effects may already have occurred;
- whether restart creates a clean verification context or merely resets the current kernel.

## 16. Environment and reproducibility

A reproducibility manifest SHALL capture enough information to reconstruct or truthfully explain the run:

- OS, architecture, container/VM/runtime identity;
- language and kernel versions;
- package manager and lockfiles;
- installed package set and hashes where practical;
- environment variables by approved name with secret values redacted;
- locale, timezone, encoding, and numeric settings;
- hardware and accelerator details;
- solver/library versions;
- deterministic seed policy;
- notebook and cell source hashes;
- dataset and model refs;
- external services and versions;
- network and filesystem posture;
- resource limits;
- known nondeterminism;
- parent manifest and drift comparison.

`same notebook` and `same kernel name` SHALL NOT imply the same environment.

## 17. Data and measurement lineage

Datasets and measurements SHALL be represented by explicit snapshots or external-source claims.

Required data posture:

- source URI or source object reference;
- content hash when obtainable;
- schema and units;
- acquisition/observation/ingestion times;
- sampling frame and exclusions where applicable;
- transformation lineage;
- licensing/privacy classification;
- redaction/public-safe posture;
- mutable-source warning;
- stale-data and supersession handling.

A notebook variable or dataframe existing in kernel memory is not sufficient dataset provenance.

## 18. Mathematical-fabric integration

When the Focusa mathematical fabric is active, notebook cells SHALL bind computational outputs to typed mathematical objects rather than only text.

Supported concepts SHOULD include:

- symbols and domains;
- expressions and equations;
- assumptions and constraints;
- units and dimensions;
- transformations and derivation steps;
- exact versus approximate results;
- solver selection and tolerances;
- convergence and conditioning;
- proofs, proof obligations, and counterexamples;
- applicability regions;
- uncertainty and error bounds;
- alternate derivations;
- contradiction and supersession.

A symbolic solver output MAY support a derivation claim but SHALL NOT automatically become a formal proof.

## 19. Probability, statistics, and forecasting integration

Notebook support for statistical and prediction work SHOULD include:

- descriptive summaries;
- distributions and generative assumptions;
- sampling and missingness posture;
- Bayesian and frequentist methods;
- uncertainty propagation;
- Monte Carlo simulation;
- sensitivity and robustness analysis;
- calibration and reliability diagrams;
- proper scoring rules and utility;
- residual and error analysis;
- causal assumptions and diagnostics;
- matched controls and ablations;
- regime and horizon breakdown;
- temporal leakage checks;
- forecast commitment and outcome resolution bindings.

A probability displayed in a notebook SHALL NOT become a Focusa prediction commitment until submitted through the applicable prediction-authority operation.

## 20. Physics-fabric integration

The runtime SHALL preserve separate classes for:

```text
analytical_prediction
numerical_simulation
sensor_observation
experimental_measurement
estimated_state
control_proposal
verified_physical_finding
```

Physics notebook support SHOULD include:

- dimensional quantities and unit conversion;
- mechanics and dynamics;
- differential-equation solvers;
- control-system models;
- circuit, fluid, thermal, optical, orbital, structural, and robotics analyses;
- sensor calibration;
- uncertainty and error propagation;
- simulation-to-measurement residuals;
- hardware-in-loop and shadow-mode experiments;
- validity regions and model breakdown;
- safety-envelope analysis.

Simulation output SHALL never silently populate an observation or measurement field.

## 21. Physical control boundary

Notebook output SHALL NOT directly control actuators. The required path is:

```text
Notebook result
→ typed physical claim or control proposal
→ verification and uncertainty review
→ Focusa skill/action proposal
→ authority and safety guards
→ independent controller or approved actuator
→ physical execution receipt
→ sensor observation
→ outcome verification and settlement
```

Emergency stops, hard real-time constraints, and safety interlocks SHALL remain outside the notebook kernel.

## 22. Learning and discovery boundary

A discovered relationship SHALL pass through a governed promotion path:

```text
Notebook observation
→ candidate relationship
→ reproduction
→ alternate derivation or implementation
→ adversarial challenge
→ counterexample search
→ applicability analysis
→ independent verification
→ scoped learning candidate
→ Focusa reducer-governed promotion
```

One successful run, high correlation, visually compelling chart, or model-generated explanation is insufficient for durable learning.

## 23. Computational Workbench information architecture

The primary UI is the **Computational Workbench**, opened as a normal Cockpit work object. It is discoverable from Research, Studio, Test Lab, Evidence, Capabilities, Mission/Workpoint context, and global search. Users MAY pin it to the sidebar; it need not be a permanent default top-level item.

```text
Computational Workbench
├── Notebook
├── Files
├── Running
├── Kernels
├── Variables
├── Equations
├── Data
├── Simulation
├── Predictions
├── Verification
├── Evidence
├── Reproducibility
└── Activity
```

### 23.1 Default notebook surface

The default view SHALL prioritize:

- notebook title, revision, scope, and mode;
- server, kernel, environment, and connection state;
- cells and outputs;
- current execution and queue;
- hidden-state or stale-output warnings;
- one primary action at a time;
- evidence and verification posture;
- Activity rail for meaningful state changes.

### 23.2 Inspector

Stable inspector tabs SHOULD include:

- Summary
- Scope
- Inputs
- Assumptions
- Dependencies
- Variables
- Environment
- Data lineage
- Outputs
- Verification
- Evidence
- Integrity
- Developer

### 23.3 Activity law

The UI SHALL feel alive only when actual computational state changes. Kernel transport heartbeats, WebSocket pings, and polling SHALL NOT appear as work.

Meaningful activity includes:

- server connected/disconnected;
- kernel started/restarted/interrupted/lost;
- cell queued/started/completed/failed;
- output created/updated;
- environment drift detected;
- dataset changed or became stale;
- verification finding created;
- evidence captured;
- Focusa handoff accepted/rejected;
- reproducibility status changed.

## 24. Functional UI states

Every implemented surface SHALL include applicable states:

```text
unconfigured
probing
unauthorized
server_offline
server_degraded
connected
kernel_starting
kernel_idle
kernel_busy
kernel_restarting
kernel_lost
cell_draft
cell_queued
cell_running
cell_streaming
cell_succeeded
cell_failed
cell_interrupted
cell_timed_out
output_stale
environment_drift
data_stale
replaying
verification_running
verified
disputed
blocked
read_only
offline
```

No static card or happy-path-only view satisfies implementation.

## 25. Proposed UIAI operations

Canonical capability IDs SHOULD include:

```text
uiai.notebook.server.discover
uiai.notebook.server.probe
uiai.notebook.contents.list
uiai.notebook.contents.read
uiai.notebook.contents.write
uiai.notebook.contents.rename
uiai.notebook.contents.delete
uiai.notebook.checkpoint.create
uiai.notebook.checkpoint.restore
uiai.notebook.session.create
uiai.notebook.session.list
uiai.notebook.session.close
uiai.notebook.kernel.specs
uiai.notebook.kernel.start
uiai.notebook.kernel.interrupt
uiai.notebook.kernel.restart
uiai.notebook.kernel.shutdown
uiai.notebook.cell.execute
uiai.notebook.cell.cancel
uiai.notebook.run.reproducible
uiai.notebook.run.verify
uiai.notebook.environment.snapshot
uiai.notebook.environment.diff
uiai.notebook.output.capture
uiai.notebook.evidence.prepare
uiai.notebook.evidence.commit
uiai.notebook.browser.open
uiai.notebook.browser.inspect
```

Every operation SHALL declare scope, authority, side effects, idempotency/reconciliation, timeout, resource class, evidence, and recovery posture.

## 26. Proposed engine API

The implementation MAY expose equivalent routes under `/api/notebooks`:

```text
GET    /api/notebooks/servers
POST   /api/notebooks/servers/probe
GET    /api/notebooks/contents
GET    /api/notebooks/contents/item
PUT    /api/notebooks/contents/item
POST   /api/notebooks/contents/rename
DELETE /api/notebooks/contents/item
POST   /api/notebooks/checkpoints
POST   /api/notebooks/checkpoints/restore
GET    /api/notebooks/kernel-specs
GET    /api/notebooks/sessions
POST   /api/notebooks/sessions
DELETE /api/notebooks/sessions/{id}
POST   /api/notebooks/kernels
POST   /api/notebooks/kernels/{id}/interrupt
POST   /api/notebooks/kernels/{id}/restart
DELETE /api/notebooks/kernels/{id}
POST   /api/notebooks/cells/execute
POST   /api/notebooks/cells/{attempt_id}/cancel
POST   /api/notebooks/runs/reproducible
POST   /api/notebooks/runs/verify
POST   /api/notebooks/environments/snapshot
POST   /api/notebooks/environments/diff
POST   /api/notebooks/outputs/capture
POST   /api/notebooks/evidence/prepare
POST   /api/notebooks/evidence/commit
GET    /api/notebooks/events/stream
```

Exact routes remain implementation-defined until registered. Generated contracts, Cockpit adapters, CLI/Pi/MCP tools, and docs SHALL derive from one operation registry.

## 27. Events and streaming

The runtime SHALL publish durable or replayable notebook events where practical. Events SHALL carry:

- stable event ID and sequence;
- exact timestamp and precision;
- notebook, revision, cell, attempt, kernel, server, scope, and principal refs as applicable;
- event type and schema version;
- source-state revision;
- causation and correlation IDs;
- payload/artifact refs;
- invalidated projections;
- redaction posture.

Core event families:

```text
notebook.server.*
notebook.document.*
notebook.session.*
notebook.kernel.*
notebook.cell.*
notebook.output.*
notebook.environment.*
notebook.dataset.*
notebook.run.*
notebook.verification.*
notebook.evidence.*
notebook.focusa_handoff.*
```

## 28. Outputs and artifacts

Supported output classes MAY include:

- text and structured JSON;
- tables/dataframes;
- images and plots;
- HTML sanitized through registered renderers;
- LaTeX/math;
- audio/video;
- interactive widgets through approved manifests;
- files and archives;
- model checkpoints;
- simulation traces;
- measurement series;
- logs and diagnostics.

Rules:

- original outputs are immutable once captured as evidence;
- display updates create versioned output deltas;
- untrusted HTML/JavaScript is sanitized or isolated;
- unknown widgets degrade to a safe placeholder with metadata;
- outputs identify source attempt and environment;
- large outputs use artifact references rather than unbounded event payloads;
- public export passes redaction and audience policy.

## 29. Evidence and receipts

A Notebook Evidence Receipt SHALL answer:

- what notebook revision ran;
- which cells and attempts ran;
- who or what initiated the run;
- under which scope and authority;
- which server, kernel, environment, data, parameters, assumptions, and dependencies were used;
- exact start/end times and durations;
- resource use and limits;
- outputs and hashes;
- errors, retries, interruption, or unknown side effects;
- reproducibility and verification posture;
- linked mathematical, prediction, physical, Workpoint, and evidence refs;
- what the run does and does not prove.

Screenshots and video supplement this receipt; they do not replace it.

## 30. Security and isolation

Notebook execution SHALL be considered code execution.

Minimum controls:

- explicit trusted server configuration;
- secure credential storage and redaction;
- filesystem roots and path traversal protection;
- network egress policy;
- environment and package-install policy;
- secret allowlist and ephemeral injection;
- process, CPU, memory, disk, output, and duration limits;
- kernel/session ownership and leases;
- extension and widget allowlists;
- untrusted notebook/output handling;
- browser origin isolation;
- audit and receipt generation;
- operator approval for privilege expansion;
- no automatic terminal or shell capability inheritance.

Package installation, native compilation, system access, and external writes SHALL be separate typed operations.

## 31. Resource Governor integration

The Resource Governor SHALL account for:

- server connections;
- kernel processes;
- memory and CPU;
- accelerator allocation;
- notebook sessions;
- queued and running cell attempts;
- output bytes and artifact storage;
- network transfer;
- browser pages and JupyterLab sessions;
- simulation and verification workloads;
- idle kernels and parkability.

The UI SHALL show resource pressure and recommended recovery without silently killing consequential executions. Forced termination SHALL produce a receipt and partial-output posture.

## 32. Multi-agent and human collaboration

Multiple agents MAY analyze the same notebook only through explicit ownership and revision controls.

Required controls:

- principal identity;
- notebook revision checks;
- cell-level or notebook-level leases;
- conflict detection;
- proposal versus committed edit distinction;
- review and approval for consequential changes;
- append-only attempt history;
- no anonymous kernel ownership;
- human takeover and return-to-agent handoff.

A notebook shared through Jupyter collaboration features SHALL still map remote edits to observable revisions and principals where possible. Unattributed changes degrade authority.

## 33. Focusa handoff

The runtime SHALL support bounded handoffs such as:

- attach a result to a Workpoint;
- submit a mathematical claim candidate;
- submit a prediction commitment or scoring result;
- submit a physical finding or simulation result;
- create an Evidence item;
- propose a learning candidate;
- request verification;
- create a follow-up task or experiment.

Every handoff SHALL preview scope, data leaving the machine, authority, side effects, evidence, and expected receipt.

## 34. Functional and visual proof

Every activated Notebook Runtime surface SHALL have actual compiled Cockpit code and release evidence under UIAI-COCKPIT-006 law.

Minimum capture families:

- server unconfigured and connected;
- authentication failure;
- notebook empty and loaded;
- kernel starting, idle, busy, interrupted, restarted, lost, and recovered;
- cell queued, streaming, succeeded, failed, timed out, and cancelled;
- rich output and safe unsupported-output fallback;
- variables and data lineage;
- environment snapshot and drift;
- exploratory hidden-state warning;
- reproducible-run preview and result;
- verification comparison and dispute;
- simulation-versus-observation distinction;
- Focusa handoff preview, accepted, and rejected;
- offline/replay state;
- narrow viewport where supported.

Each capture SHALL be paired with interaction assertions and a SHA-bound manifest. Screenshot existence alone is insufficient.

## 35. Test requirements

### 35.1 Contract tests

- Jupyter version and capability negotiation;
- contents round trip and metadata preservation;
- session/kernel lifecycle;
- channel message parsing;
- event and receipt schema validation;
- generated operation parity;
- unknown provider field preservation.

### 35.2 Functional tests

- open/edit/save/checkpoint notebook;
- execute and stream each supported output class;
- interrupt/restart/shutdown behavior;
- reconnect after transport loss;
- stale revision and cross-write blocking;
- duplicate message and lost-ack handling;
- hidden-state detection;
- clean reproducible run;
- environment/data drift detection;
- verification independence labeling;
- Focusa handoff preview and receipt;
- no fake Activity events.

### 35.3 Security tests

- credential and secret redaction;
- path traversal;
- untrusted HTML/widget isolation;
- unauthorized server/kernel access;
- network egress denial;
- resource exhaustion;
- package-install and shell boundary;
- malicious notebook metadata;
- oversized/decompression-bomb outputs;
- cross-origin browser isolation.

### 35.4 Math and physics tests

- exact versus approximate result distinction;
- units and dimensions;
- solver tolerance and convergence reporting;
- uncertainty propagation;
- calibration/scoring reproducibility;
- simulation cannot satisfy observation predicates;
- physical control proposal cannot actuate directly;
- learning candidate cannot self-promote.

## 36. Implementation phases

### Phase N0 — contracts and fixtures

- operation registry entries;
- schemas and generated types;
- provider test server and deterministic fixtures;
- object/event/receipt persistence decisions;
- security and resource policy.

### Phase N1 — server, contents, sessions, and kernels

- server profiles and health;
- contents adapter;
- session and kernel lifecycle;
- Cockpit connection, files, running, and kernel surfaces;
- functional and visual proof.

### Phase N2 — cell execution and outputs

- channel client;
- immutable attempts;
- output streaming and artifact capture;
- cancellation/recovery;
- Activity integration;
- functional and visual proof.

### Phase N3 — reproducibility and evidence

- environment/data manifests;
- clean-run executor;
- receipts;
- Evidence and Focusa handoff;
- drift detection;
- functional and visual proof.

### Phase N4 — verification, math, statistics, and physics bindings

- verification runner;
- alternate solver and comparison adapters;
- mathematical/epistemic/physical object bindings as external contracts become available;
- simulation-versus-observation and learning boundaries;
- functional and visual proof.

### Phase N5 — rich JupyterLab/browser integration

- browser routing;
- registered widgets/extensions;
- debugger/rich visual surfaces;
- human takeover;
- accessibility and visual regression proof.

## 37. Success metrics

Measure:

- successful server discovery and connection rate;
- kernel startup/reconnect/interrupt reliability;
- cell execution acceptance and terminal-state accuracy;
- duplicate/lost-message recovery rate;
- output capture completeness;
- hidden-state and stale-output detection rate;
- reproducible-run success rate;
- environment/data drift detection rate;
- independent verification coverage;
- evidence receipt completeness;
- time from notebook result to informed review;
- false-authority claims prevented;
- simulation/observation classification errors prevented;
- resource-limit recovery quality;
- user correction and dispute rate;
- visual-proof coverage for activated surfaces.

Metrics SHALL distinguish exploratory productivity from verified computational reliability.

## 38. Closure rule

UIAI-COCKPIT-007 remains documentation-only until implementation evidence exists.

A Notebook Runtime capability may move to `release_verified` only when:

- its operation is registered and generated across required surfaces;
- the provider adapter works against a supported Jupyter version;
- authority, security, resource, temporal, and recovery behavior is implemented;
- Cockpit contains actual functional UI code for all claimed states;
- interaction tests pass;
- evidence and receipts are machine-readable;
- screen captures have been inspected and bound to the exact implementation SHA;
- contract-faithful fixtures and at least one approved networked Jupyter run pass;
- Focusa-specific claims are limited to the exact pinned external contracts available at that time.

No notebook, mathematical, statistical, predictive, simulation, physics, learning, or control claim may be marked complete by documentation, a static card, a successful cell, or an attractive plot alone.
