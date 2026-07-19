- release/version information;
- copy redacted support bundle.


---

# 8. Cross-workspace journeys

The value of one Cockpit is that work moves between capabilities without losing context.

## 8.1 Agent session to verified E2E flow

```text
Overview: Resume Workpoint
  → Live: watch agent reproduce issue
  → Take over briefly and correct path
  → Save operator actions as draft flow
  → Test Lab: choose runner and run repeatedly
  → Inspect synchronized video, assertions, and diagnostics
  → Evidence: attach verified result to Workpoint
  → Focusa: evaluate prediction and checkpoint next action
```

No manual re-entry of project, Workpoint, session, or artifact paths.

## 8.2 Document received to approved output

```text
Documents: import scanned contract
  → detect OCR need and extract structure
  → select clauses and ask bounded AI analysis
  → apply proposed redactions to derivative
  → compare original and proposed output
  → approve final derivative
  → Evidence: attach clause findings and output refs
  → Activity: record operation manifest and approval
```

## 8.3 Research to document/report

```text
Research: search and capture sources
  → create bounded research packet
  → Studio or Documents: generate report from approved captures
  → Documents: inspect citations and layout
  → Evidence: attach report and source refs
```

## 8.4 Visual regression investigation

```text
Test Lab: visual matrix fails
  → open failed screenshot comparison in Studio
  → open underlying browser step in Live/replay
  → inspect console/network diagnostics
  → patch outside Cockpit through agent/editor workflow
  → rerun failed step
  → mark approved result as new golden
  → Evidence: record actual verification
```

## 8.5 Interactive report to governed follow-up

```text
Test Lab: failed checkout verification
  → Evidence: compose Failure Anatomy report
  → reviewer annotates changed region and requests a fix
  → Cockpit creates a follow-up Workpoint proposal carrying selected evidence/receipts
  → Focusa verifies scope and checkpoints the accepted Workpoint
  → agent fixes and reruns
  → Report Canvas creates v2 with linked verification result
  → reviewer approves the frozen engineering version
  → optional public-safe stakeholder snapshot is generated separately
```

The report carries context but does not itself become the Workpoint, authorization, or completion record.

## 8.6 Supporting work in another repository

Cockpit must distinguish:

- active scope;
- supporting work;
- artifact link;
- explicit scope switch.

Filing a UIAI issue while working on another primary project must not silently transfer Workpoint authority. The created issue is linked as supporting evidence unless the operator explicitly switches scope.

---

# 9. Interaction architecture

## 9.1 Selection hierarchy

Only one primary object is selected in the viewport. Secondary selections—page, test step, screenshot region, table, event—remain contextual children of that object.

This prevents the inspector from becoming ambiguous.

## 9.2 Selection-to-inspector behavior

Examples:

| Selected item | Center | Inspector |
|---|---|---|
| Browser session | Live mirror | session summary, scope, diagnostics, evidence |
| Browser element | highlighted live region | selector/ref, accessibility, action availability |
| Test step | synchronized frame/video position | assertion, diagnostics, artifacts, retry |
| PDF page | page canvas | text blocks, citations, annotations, page operations |
| PDF region | highlighted region | OCR/native text, coordinates, extraction confidence |
| Research capture | clean reader | source metadata, chunks, diagnostics, evidence |
| Screenshot diff | visual comparison | thresholds, changed regions, baseline lineage |
| Review report | Report Canvas | sources, predicates, decisions, comments, integrity, share policy |
| Node | topology/health view | endpoint, transport, pairing, ownership, capacity |

## 9.3 Contextual actions

Actions appear in four places, with strict priority:

1. primary action in toolbar;
2. direct manipulation on the selected content;
3. inspector actions for selected object;
4. overflow menu or command palette for less common actions.

Do not repeat the same action prominently in all four places.

## 9.4 Command previews

Before a meaningful write, show a compact preview:

```text
Action: Apply 12 redactions
Input: vendor-agreement.pdf
Output: vendor-agreement-redacted.pdf
Scope: Acme Renewal / Contract Review
Runs on: Mac Studio / UIAI document worker
Original preserved: Yes
Approval: Required
```

Developer Mode may reveal the underlying capability ID, adapter, and command parameters.

## 9.5 Approvals

Use a single approval pattern across the app. Approval grants a narrow, expiring capability lease; it is not merely a confirmation click.

Approval sheet includes:

- risk class (R0–R5) and why;
- principal/worker receiving authority;
- capability scope, allowed origin/resource, maximum uses/value, expiration, and delegation posture;
- what will happen;
- what will change;
- where it runs;
- what data leaves the machine;
- scope and authority;
- cost where applicable;
- rollback/derivative behavior;
- evidence/receipt outcome;
- Cancel / Approve.

Types:

- local write;
- external/cloud send;
- destructive change;
- document redaction/signing;
- proof publication;
- device revoke/ownership transfer;
- paid AI action;
- bulk operation.

Report review decisions use the same pattern but preserve semantic distinctions:

- approving a report approves that version/audience, not arbitrary follow-up execution;
- accepting an outcome proposes a mission decision and does not bypass Completion Contract or settlement rules;
- requesting changes may create a follow-up proposal but does not silently mutate the current Workpoint;
- comments, annotations, and ratings never grant capability authority.

## 9.6 Undo and rollback

Three distinct concepts must not be conflated:

- **UI undo:** reverses local presentation or a not-yet-committed proposal.
- **Compensating action:** creates a new operation that reverses a prior external/local write when supported.
- **Artifact versioning:** returns to a prior immutable derivative or baseline.

The UI must say which one is available. It must never imply daemon or external state was rolled back when only local display state changed.

## 9.7 Empty states

Every empty state must:

1. explain what the area is for;
2. explain why it is empty;
3. offer one useful next action;
4. avoid sales language.

Examples:

- **No live sessions.** Start a browser session or open a recent recording.
- **No tests yet.** Import a Maestro flow, create a flow, or record one from Live.
- **No documents.** Import a PDF or Office file.
- **No evidence for this Workpoint.** Run a verification or capture the selected artifact.
- **No node selected.** Choose where this work should run.

## 9.8 Loading and long work

- Use skeletons for content that normally resolves quickly.
- Use local cached summaries with a visible freshness label when online refresh is slower.
- Use inline progress for known stages.
- Long work becomes a Job and continues without blocking the viewport.
- No full-window spinner for ordinary adapter calls.
- Navigation remains usable while independent work runs.

## 9.9 Failure and recovery

Each failure displays:

- plain-language cause;
- affected object only;
- whether work was committed;
- retryability;
- one recommended recovery;
- evidence/log reference when useful;
- technical details on demand.

A failed module must not crash the shell or unrelated workspaces.

## 9.10 State vocabulary

Use a restrained human-facing vocabulary, while preserving canonical mission/task/action distinctions from Section 4.5 in Inspector and event history.

Human-facing labels:

- Ready
- Running
- Paused
- Waiting
- Needs approval
- Needs pairing
- Needs scope
- Read-only
- Blocked
- Degraded
- Failed
- Complete
- Verified
- Provisional
- Public-safe
- Offline

Do not invent subtly different status words for each module. Do not use `Complete` when the truthful state is only Submitted, Accepted, Provisionally complete, or Settlement pending.

---

# 10. Normative design system and interaction quality guardrails

This section is a release contract, not a collection of aesthetic suggestions. A Cockpit surface that violates a **MUST** requirement is incomplete even when its underlying capability works.

The objective is not to imitate macOS decoration. The objective is to achieve the qualities that make excellent desktop software feel inevitable: clear hierarchy, precise alignment, restrained chrome, fluent interaction, predictable state, and immediate access to deeper power when it is needed.

## 10.1 Design-system authority

The Cockpit MUST maintain one source-owned design system under `apps/cockpit/src/lib/ui/`.

It MUST contain:

- semantic design tokens;
- accessible headless primitives;
- composed controls and patterns;
- workspace shell components;
- state and interaction contracts;
- representative fixtures for every state;
- automated accessibility, visual, and performance checks.

The design system MUST be used by every first-party workspace, card, inspector, overlay, approval, empty state, and extension-facing host surface.

The following are prohibited:

- workspace-local color systems;
- workspace-local spacing scales;
- arbitrary animation durations;
- mixing unrelated icon families;
- direct use of unreviewed third-party component styling;
- copying a control into a workspace and modifying it independently;
- raw API responses used as the primary UI;
- one-off modal, table, notification, or empty-state patterns.

When a new visual or interaction need cannot be served by the shared system, the component MUST be added to or deliberately extended in the shared system before it is used in a product workspace.

## 10.2 Normative language and exception process

Within design and experience sections:

- **MUST / MUST NOT** means merge- and release-blocking.
- **SHOULD / SHOULD NOT** means required unless a documented exception is approved.
- **MAY** means optional and context-dependent.

A design exception MUST appear in the pull request as:

```text
Design exception
Requirement:
Reason:
Affected surfaces:
Accessibility impact:
Performance impact:
Owner:
Expiration or follow-up bead:
```

An exception MUST NOT be permanent by omission. It either receives an expiry/follow-up bead or is incorporated into the design system as a deliberate new rule.

## 10.3 Semantic token contract

Components MUST consume semantic tokens rather than raw visual values. Tokens are organized in four layers:

```text
Foundation tokens  → raw scale values
Semantic tokens    → meaning: text, surface, border, status, focus
Component tokens   → control-specific mappings
Context overrides  → dark mode, compact density, reduced motion, high contrast
```

### Required token families

- color and contrast;
- typography;
- spacing;
- dimensions and control heights;
- corner radii;
- borders and dividers;
- shadows and elevation;
- opacity;
- motion duration and easing;
- z-index/layer ordering;
- sidebar, inspector, toolbar, and overlay geometry;
- focus-ring appearance;
- data visualization series and states.

No production `.svelte`, `.ts`, or `.css` file outside the token and visualization-definition files may introduce arbitrary hexadecimal colors, RGB/HSL values, box shadows, radii, spacing values, or transition durations.

A token-lint gate MUST fail CI when a raw value is introduced outside an approved allowlist.

### Base geometry

Use a 4-point spacing foundation. The initial scale is:

```css
--space-0: 0;
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;
--space-5: 20px;
--space-6: 24px;
--space-8: 32px;
--space-10: 40px;
--space-12: 48px;
--space-16: 64px;
```

Use compact increments only for optical alignment inside icons or tightly controlled data visualization. Layout spacing MUST use the shared scale.

### Initial shape scale

```css
--radius-control-sm: 6px;
--radius-control: 8px;
--radius-surface: 10px;
--radius-card: 12px;
--radius-overlay: 14px;
--radius-pill: 999px;
```

Pills are reserved for compact statuses, segmented selections, and tags. Ordinary buttons and fields MUST NOT default to pill shapes.

### Initial elevation scale

```css
--shadow-resting: 0 1px 2px rgb(0 0 0 / 0.06);
--shadow-raised: 0 4px 14px rgb(0 0 0 / 0.10);
--shadow-overlay: 0 12px 40px rgb(0 0 0 / 0.20);
--shadow-focus: 0 0 0 3px var(--focus-halo);
```

Elevation MUST communicate layering or interactivity. It MUST NOT be applied to every container.

## 10.4 Typography system

Cockpit typography MUST use the operating-system UI font through a system stack. Apple font files MUST NOT be bundled or redistributed. On macOS, the system resolves to San Francisco; on other platforms it resolves to the platform UI font.

```css
--font-ui: -apple-system, BlinkMacSystemFont, "Segoe UI", Inter, system-ui, sans-serif;
--font-mono: ui-monospace, "SFMono-Regular", "Cascadia Code", "Roboto Mono", monospace;
```

### Semantic type ramp

| Token | Initial size / line height | Weight | Primary use |
|---|---:|---:|---|
| `display` | 28 / 34 | 650 | Rare workspace or onboarding hero heading |
| `title-1` | 22 / 28 | 650 | Workspace title |
| `title-2` | 18 / 24 | 600 | Work object, sheet, and major section title |
| `headline` | 15 / 20 | 600 | Group headings, prominent row labels |
| `body` | 14 / 20 | 400 | Default readable text and control labels |
| `callout` | 13 / 18 | 450 | Secondary UI copy, inspector content |
| `caption` | 12 / 16 | 450 | Metadata, timestamps, compact status copy |
| `micro` | 11 / 14 | 500 | Rare nonessential technical labels only |
| `code` | 12 / 18 | 400 | Code, refs, paths, compact structured values |

Guardrails:

