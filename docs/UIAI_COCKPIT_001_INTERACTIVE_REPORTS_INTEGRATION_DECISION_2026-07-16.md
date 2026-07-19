# UIAI Cockpit Interactive Review Reports Integration Decision

**Document number:** `UIAI-COCKPIT-001`  
**Parent document:** `UIAI-COCKPIT-000`  
**Status:** Product/architecture integration decision  
**Date:** 2026-07-16  
**Integrated master spec:** [`UIAI-COCKPIT-000`](./UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md)  
**Machine-readable companions:** None; contracts are incorporated into the master and later amendments

## 0. Decision

Adopt the generic visual and interactive agent-report concept as **Review Reports**, rendered through a **Report Canvas**, inside the existing UIAI Cockpit architecture.

Do not create a separate report product, report authority plane, task database, memory system, or arbitrary agent-generated web application.

A Review Report is:

> A versioned, evidence-backed, interactive projection of canonical mission, Workpoint, receipt, evidence, artifact, test, document, and activity sources for human review, decisions, feedback, follow-up, and safe sharing.

The source systems remain authoritative:

- Focusa owns mission continuity, Workpoints, scope, evidence linkage, decisions, and future settlement state.
- UIAI Engine owns execution artifacts, captures, diagnostics, recordings, document/test/media outputs, and report artifact production.
- Cockpit renders the Report Canvas and routes review intent.
- Guards, policy, adapters, and orchestration authorize and execute any follow-up action.
- Focusa Cloud may host public-safe frozen snapshots without becoming mission authority.

# 1. Why this preserves the primary product shape

The Cockpit remains organized around task-oriented workspaces:

- Overview / Mission Deck
- Live
- Test Lab
- Documents
- Research
- Studio
- Automations
- Evidence
- Activity
- Nodes & Services
- Capabilities

Reports are cross-workspace review objects. They are created contextually from work and live primarily in Evidence. They open as normal work-object tabs and can be found in global search. A Reports saved view may be pinned, but no new default top-level workspace is required.

This avoids fragmentation:

```text
Not:
Cockpit + report app + client portal + task-comment system

Instead:
Cockpit workspaces
  → canonical artifacts, receipts, and evidence
  → Review Report projection
  → governed review interactions
  → explicit follow-up Workpoint/task/decision
```

# 2. Conformance changes made to the generic feature

## 2.1 Reports do not own truth

The generic proposal says reports should persist as first-class objects. That is retained, but a report is first-class as a **review and presentation object**, not as the canonical mission ledger.

The report references rather than rewrites:

- Mission and Completion Contracts
- Project, Workstream, Workpoint, and Trajectory
- task/action events
- Action Proposals and Receipts
- typed evidence
- immutable artifacts
- verification and settlement state
- approved decisions and follow-up refs

## 2.2 Visual evidence is attributable

Real screenshots, clips, recordings, document pages, and comparisons are preserved as the central value proposition.

Additional requirements:

- actual capture source and event range;
- timestamp, viewport/device, session/run, redaction state, and hash;
- immutable original;
- annotation stored as a separate overlay;
- AI-generated visuals labeled Illustrative and prohibited from satisfying verification predicates;
- video supplements but does not replace receipts.

## 2.3 Interactivity is declarative and governed

The following interactions are supported:

- approve/reject report;
- request changes;
- comment and annotate;
- choose variant;
- accept/reject baseline;
- review evidence;
- request recapture or reverification;
- create follow-up Workpoint/task proposal;
- rerun a bounded capability;
- share or export.

Every interaction uses a typed manifest and routes through:

```text
Report widget
  → ReportActionController
  → ScopeGuard / AuthorityGuard / ConsentGuard
  → Action Proposal or review event
  → adapter/orchestrator
  → receipt/event/Focusa handoff
```

No report may directly execute arbitrary JavaScript, prompts, shell commands, tools, browser actions, or remote embeds.

## 2.4 Review decisions have distinct semantics

The following are deliberately separate:

- **Approve report:** accept this report version for its stated audience or review stage.
- **Accept outcome:** propose that the mission outcome be accepted; verification and settlement contracts still apply.
- **Authorize follow-up:** grant bounded authority for a new task/action.
- **Request changes:** record feedback and optionally propose follow-up work.
- **Mark evidence reviewed:** record human inspection, not automatic independent verification.
- **Rate usefulness:** product feedback only.

Comments, annotations, and prose never grant execution authority.

## 2.5 No raw chain-of-thought

The generic concept’s “reasoning” and self-reflection are narrowed to:

- explicit goals and constraints;
- recorded decision rationale;
- Action Proposal purpose;
- actuator-route rationale;
- observable action timeline;
- bounded post-run Agent assessment;
- limitations and uncertainty.

Raw private reasoning, full prompts, and hidden chain-of-thought are excluded and are not treated as evidence.

## 2.6 Custom visualization uses registered blocks

Agents retain creative flexibility through typed report sections and registered renderers:

- evidence-linked narrative;
- tables and data grids;
- charts with source datasets;
- comparisons;
- annotated captures;
- timelines;
- diagrams;
- code/diff excerpts;
- document citations;
- failure matrices;
- recommendations.

Custom blocks inherit Cockpit typography, color, spacing, iconography, motion, overlay, accessibility, responsive, export, CSP, and security contracts.

# 3. Report families

- Run Review
- Verification Report
- Failure Anatomy
- Decision Brief
- Release Proof
- Benchmark Report
- Document Review
- Stakeholder Update
- Incident Report

Templates set default composition and audience policy. They do not create different truth models.

# 4. Report Canvas progressive disclosure

## 4.1 Glance

- outcome and report status;
- executive summary;
- verified/provisional posture;
- primary evidence;
- one primary decision.

## 4.2 Work

- key findings;
- selected visuals;
- comparisons;
- recommendations;
- comments.

## 4.3 Inspect

- Completion predicates;
- receipts;
- evidence lineage;
- timeline;
- annotations;
- contradictions and uncertainty.

## 4.4 Configure

- template and audience;
- section composition;
- interaction policy;
- retention and sharing.

## 4.5 Developer

- manifests;
- source refs;
- hashes;
- event ranges;
- widget and renderer schemas.

# 5. Report lifecycle

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

Exceptional states:

- FAILED
- SUPERSEDED
- REVOKED
- EXPIRED

Approved and frozen versions are immutable. Source changes mark live reports stale. Public shares always identify the exact frozen version.

# 6. Live and frozen forms

## 6.1 Live report projection

- updates from current canonical sources;
- shows freshness and staleness;
- supports active review and bounded interactions.

## 6.2 Frozen report snapshot

- immutable source manifest;
- interaction-state summary;
- artifact refs;
- hashes and optional signature;
- version-bound sharing/export.

# 7. Audience profiles

- Operator internal
- Engineering review
- Executive/stakeholder
- Client review
- Public proof
- Audit/export

Each audience profile controls sections, technical depth, comments, interaction capabilities, data disclosure, redaction, retention, and share expiry.

# 8. Export model

Supported projections may include:

- frozen HTML/web snapshot;
- PDF;
- Markdown;
- JSON manifest;
- ZIP evidence bundle;
- public-safe hosted snapshot;
- generated document or presentation packet.

Static exports show current widget/decision state and do not imply that interactive controls remain active.

# 9. Cross-workspace creation

- Live → Run Review or Incident Report
- Test Lab → Verification, Failure Anatomy, Benchmark, or Release Proof
- Documents → Document Review or approval packet
- Research → evidence-backed research brief
- Studio → Decision Brief
- Automations → report at checkpoint, completion, failure, or settlement
- Evidence → report library, versioning, review, freeze, share, and export
- Overview → latest report and pending decision
- Activity → comments, approvals, shares, report jobs, and follow-up dispatch

# 10. Follow-up context bundle

A report can create a bounded follow-up proposal containing:

- source report/version;
- mission and Workpoint scope;
- selected findings/comments/regions;
- relevant receipts/evidence;
- unresolved predicates or contradictions;
- proposed objective and required evidence.

Focusa or the orchestrator creates the real Workpoint/task. The report is not treated as the task definition merely because it contains prose.

# 11. Initial implementation sequence

1. Define ReviewReport, ReportSection, ReportInteractionManifest, thread, and snapshot contracts.
2. Implement static first-party report templates.
3. Implement deterministic report composition from canonical refs.
4. Build Report Canvas using the existing Cockpit shell and inspector.
5. Add actual-capture provenance and annotation overlays.
6. Add comments and non-mutating review actions.
7. Add governed follow-up creation and bounded reruns.
8. Add versioning, freezing, freshness, and supersession.
9. Add audience policy, redaction preview, revocable sharing, and exports.
10. Add hashes/signatures for release, audit, incident, and external proof reports.
11. Add custom registered block renderers only after first-party blocks and sandbox rules are proven.

# 12. Non-negotiable tests

- missing/stale/contradictory source behavior;
- actual versus illustrative visual labeling;
- immutable originals and annotation integrity;
- report approval versus outcome acceptance versus action authorization;
- no direct execution from report content;
- comment and custom-block injection resistance;
- public-safe redaction and version-bound share links;
- follow-up context minimization and correct scope;
- export parity and integrity manifest reproduction;
- accessibility, keyboard, responsive, reduced-motion, and design-system conformance.

# 13. Success measures

- time to informed review decision;
- evidence sufficiency/integrity;
- follow-up success without context re-explanation;
- comment resolution linked to receipts/evidence;
- stale/superseded report detection;
- redaction failures prevented;
- reviewer correction/disagreement rate;
- report generation cost and latency;
- false-done acceptance reduction;
- usefulness rating, explicitly separated from mission truth.

# 14. Final principle

> Reports are powerful because they turn execution into something humans can see, understand, discuss, and act on. They remain trustworthy only when they are treated as governed lenses over canonical mission state, receipts, and evidence—not as beautiful replacements for them.
