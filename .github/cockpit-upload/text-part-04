- Isolate parsers, OCR, conversion, and signing workers.
- Enforce byte, page, memory, render, and decompression limits.
- Verify redaction by re-extraction and object inspection.
- Warn before modifying a signed document.
- Keep passwords ephemeral or in secure storage.
- Require explicit approval before external delivery.

## 7.5 Research

Research unifies provider-neutral search, Source-to-Markdown, browser reads, public-source adapters, citations, and Focusa research packets.

### Information architecture

```text
Research
  ├── Search
  ├── Captures
  ├── Collections
  └── Packets
```

### Search

The default experience is a clean search field, source/provider selector when needed, and bounded results.

Each result can:

- open in a UIAI session;
- capture to Markdown/JSONL;
- add to a collection;
- inspect source metadata;
- create evidence;
- include in a packet.

### Capture reader

Show clean content first:

- title;
- source;
- captured timestamp;
- article/thread/transcript structure;
- citations and links;
- evidence status.

Raw adapter metadata, diagnostics, chunk JSONL, and capture routing appear in the inspector.

### Collections

Collections group captures under a goal or Workpoint without becoming parallel long-term cognition. Focusa stores the durable evidence and continuity handles; Cockpit stores the local presentation and artifact references.

### Research packet

Guided route:

```text
scope → search/open/read → diagnostics when needed → bounded packet → Focusa handoff
```

The packet view shows goal, sources, evidence refs, diagnostics status, recommended Focusa tool, and next action before expanded JSON.

## 7.6 Studio

Studio consolidates visual analysis, design extraction, screenshot output, visual comparison, and media production.

### Sections

```text
Studio
  ├── Capture
  ├── Compare
  ├── Analyze
  ├── Design
  └── Produce
```

### Capture

- one-shot screenshots;
- session screenshots;
- responsive viewport matrices;
- element and region capture;
- stable screenshot mode;
- dark/reduced-motion/media emulation;
- browser/device frames;
- golden baseline capture.

### Compare

- screenshot comparison;
- layout comparison;
- baseline diff;
- scenario diff;
- version-to-version visual review;
- threshold and changed-region inspection.

### Analyze

- critique;
- section detection;
- UI reverse/reference analysis;
- accessibility report;
- contrast, focus, landmarks, keyboard navigation, and focus-trap checks;
- diagnostics-assisted visual investigation.

### Design

- design-system extraction;
- content map;
- block recipes;
- five-way comparison;
- structured output suitable for WPUIAI or other build workflows.

### Produce

- device mockups;
- animated GIFs;
- product/E2E videos;
- illustrations where enabled;
- export presets and artifact packages.

Paid or credit-consuming actions clearly show estimated cost, model/provider, consent, and output policy before execution.

## 7.7 Automations

Automations is the home for reusable scenarios, workflow orchestration, intake, and bounded recipes.

### Sections

- Recipes
- Runs
- Schedules/Triggers later
- Intake
- Migration
- Templates

### Recipe model

A recipe is reusable procedural memory, not a canonical mission, blanket authority grant, or proof of completion. On use, it is bound to a Mission Contract, versioned inputs, verified scope, capability grants, budgets, and a task graph.

A recipe declares:

- goal;
- accepted inputs;
- ordered capabilities;
- scope requirements;
- side effects;
- approvals;
- rollback/derivative behavior;
- outputs;
- evidence policy;
- timeout and retry policy;
- supported nodes/runners.

Examples:

- responsive visual audit;
- authenticated form verification;
- source-to-report workflow;
- document packet preparation;
- migration preview;
- release proof bundle;
- recurring QA flow.

### Guardrails

- Every run has an explicit task graph or equivalently reconstructable structured plan.
- Shared resources use owners, leases, locks, or version checks.
- Side-effecting steps define idempotency or reconciliation.
- Retries are bounded and failure-class aware.
- Consequential steps emit Action Proposals and Receipts.
- Exhausted or ambiguous side effects enter review/dead-letter state rather than retrying blindly.
- No arbitrary hidden shell.
- Code Capsule remains the bounded code-execution path when required.
- A recipe preview shows what will run and what will change.
- External writes and destructive actions pause for approval.
- A run can be resumed from a safe checkpoint.

## 7.8 Evidence

Evidence is the project proof center, not merely a file browser.

### Saved views

- Current Workpoint
- Recent
- Needs capture
- Needs review
- Verified
- Provisional/Surrogate
- Public-safe
- Receipts
- Reports

### Evidence item

Evidence is typed and predicate-linked. A screenshot or video is not treated as self-explanatory proof.

Show first:

- what the evidence proves;
- source object;
- verification class;
- captured time;
- project/Workpoint;
- primary artifact;
- approval/public state.

Inspector reveals:

- artifact hashes;
- lineage;
- scope;
- tool and version;
- diagnostics;
- redaction;
- Focusa ref;
- receipt;
- raw manifest.

### Action receipts and settlement

Evidence groups may include a machine-readable Action Receipt showing:

- action, worker, grant, target, actuator, and bounded input;
- before/action/after state;
- external confirmation identifier;
- evidence items and related completion predicates;
- verification, contradiction, and settlement status;
- cost, retries, duration, and uncertainty.

The Evidence workspace clearly separates:

- agent claim;
- immediate action result;
- accepted external result;
- independently verified outcome;
- provisionally complete outcome;
- settled outcome.

Video and screenshots supplement receipts but never replace machine-readable history.

### Interactive review reports

The Cockpit supports **Review Reports**: versioned, evidence-backed, interactive projections of completed, active, failed, or proposed work. The user-facing renderer is the **Report Canvas**.

A Review Report is not a second mission store, a replacement for Focusa, a free-form agent webpage, or proof merely because it looks polished. It composes canonical references from:

- Mission Contract and Completion Contract state;
- Project, Workstream, Workpoint, and Trajectory references;
- task and action lifecycle events;
- Action Proposals and Action Receipts;
- typed evidence items and immutable artifacts;
- browser/session recordings and operator interventions;
- test assertions, diagnostics, diffs, and runner reports;
- document findings, citations, derivatives, and verification results;
- approved human comments, decisions, and follow-up references.

The underlying sources remain authoritative. The report stores bounded summaries, layout, annotations, interaction manifests, audience policy, and source references. It MUST NOT silently rewrite the referenced mission, Workpoint, receipt, evidence, or external state.

#### Product role

Review Reports make agent work legible and actionable without changing the primary Cockpit shape:

```text
Mission Deck / workspaces produce work
  → UIAI produces artifacts and receipts
  → Focusa preserves mission and evidence linkage
  → Evidence composes a Review Report
  → Report Canvas supports review and bounded decisions
  → typed interactions return through guards/orchestrator
  → new Workpoints/tasks/receipts are created when authorized
```

Reports therefore belong primarily in **Evidence**, open as normal work-object tabs, appear contextually in Test Lab, Documents, Live, Research, Studio, Automations, and Overview, and remain searchable through global search. They do not require another permanent top-level sidebar item. A user MAY pin a Reports saved view, but Reports is not a separate product.

#### Report families

The same schema supports purpose-specific templates:

| Report family | Primary purpose | Typical sources |
|---|---|---|
| **Run Review** | Explain what an agent or automation attempted and what happened. | session events, receipts, screenshots, video, diagnostics |
| **Verification Report** | Demonstrate whether explicit predicates were satisfied. | Completion Contract, verifier results, evidence sufficiency |
| **Failure Anatomy** | Make a failed or partial run diagnosable and recoverable. | failed step, logs, network, console, screenshots, uncertainty |
| **Decision Brief** | Compare options or variants and collect a bounded decision. | comparisons, tables, prototypes, metrics, recommendations |
| **Release Proof** | Summarize readiness and attach release/CI evidence. | test runs, bundle manifests, visual matrices, receipts |
| **Benchmark Report** | Explain performance and failure patterns across runs. | metrics, variants, model/runner/actuator dimensions |
| **Document Review** | Present findings, citations, annotations, changes, and approvals. | pages, regions, clauses, diffs, derivatives, signatures |
| **Stakeholder Update** | Communicate progress externally using public-safe evidence. | approved summaries, selected visuals, redacted artifacts |
| **Incident Report** | Preserve impact, timeline, causes, mitigations, and linked missions. | security/operational events, affected receipts, recovery actions |

A template controls default composition and audience policy, not canonical semantics. Report sections remain typed and rearrangeable within allowed template constraints.

#### Report Canvas information architecture

A Report Canvas uses the same calm shell, Context Control, work-object tabs, inspector, Activity Bar, and progressive-disclosure system as every other Cockpit object.

The default canvas presents:

1. **Outcome header** — human title, report kind, mission/Workpoint, status, verified-versus-provisional posture, audience, version, and timestamps.
2. **Executive summary** — attempted outcome, achieved result, blockers, strongest recommendation, and most material uncertainty.
3. **Primary evidence** — one hero visual, comparison, table, or verification result selected because it best supports the current review decision.
4. **Key findings** — concise evidence-linked observations, each labeled as fact, inference, prediction, contradiction, or missing evidence.
5. **Decision or next action** — at most one prominent review decision area at a time.
6. **Supporting sections** — step timeline, visuals, structured analysis, recommendations, artifacts, and references, disclosed as needed.

The report MUST NOT display every screenshot, event, metric, comment, and widget simultaneously. It uses progressive disclosure:

- **Glance:** outcome, status, executive summary, hero evidence, primary decision.
- **Work:** findings, selected visuals, comparisons, recommendations, comments.
- **Inspect:** predicates, receipts, source lineage, annotations, timeline, uncertainty.
- **Configure:** template, audience, section order, interaction policy, retention, sharing.
- **Developer:** manifests, hashes, source refs, event ranges, renderer and widget schemas.

The inspector contains stable tabs such as **Summary**, **Sources**, **Evidence**, **Predicates**, **Decisions**, **Comments**, **Integrity**, **Share**, and **Developer**. Comments and metadata should normally live in the inspector rather than visually cluttering the report body.

#### Visual evidence integrity

Visuals are valuable because they are attributable captures, not decoration.

- Verification visuals MUST originate from actual UIAI/browser/test/document/media capture tools or an explicitly identified external evidence source.
- AI-generated or reconstructed imagery MUST be labeled **Illustrative** and MUST NOT satisfy a verification predicate.
- Originals remain immutable. Highlights, arrows, masks, callouts, numbered markers, and captions are stored as separate annotation overlays with author, time, geometry, and source reference.
- Every screenshot or clip SHOULD identify session/run, action or event range, viewport/device, capture time, redaction state, and integrity hash where practical.
- A clip or replay link MUST reference the underlying event and receipt range. Video supplements but never replaces structured action history.
- Before/after comparisons MUST preserve both source refs, comparison method, thresholds, and whether the difference was independently reviewed.
- Public or client-facing visuals pass through the same RedactionBoundary and public-safe approval path as proof publishing.

#### Narrative and execution history

A report may describe why a route or action was chosen, but it MUST NOT expose private chain-of-thought or claim that hidden reasoning is evidence.

Use:

- explicit Mission/Workpoint goals;
- recorded decisions and constraints;
- Action Proposal purpose and expected effect;
- route-selection rationale;
- observable tool/action timeline;
- post-run self-assessment labeled **Agent assessment**;
- limitations, uncertainty, and recommended recovery.

Do not use:

- raw private reasoning traces;
- unbounded prompts or model transcripts;
- credentials, cookies, private headers, or hidden form values;
- a retrospective narrative that contradicts the receipts or evidence ledger.

The step-by-step section references typed events and visuals rather than duplicating raw logs. Technical details expand on demand.

#### Structured analysis and custom visualizations

Reports support declarative blocks such as:

- evidence-linked prose;
- status and predicate summaries;
- tables and data grids;
- charts with source datasets;
- before/after and multi-variant comparisons;
- annotated screenshots and document pages;
- timeline and sequence views;
- architecture or flow diagrams;
- code/diff excerpts with artifact refs;
- document citations and extracted tables;
