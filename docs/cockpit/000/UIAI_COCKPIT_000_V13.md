16. incident linkage across mission, worker, grant, receipt, and external target.
17. malicious report content, remote embeds, script injection, comment injection, forged visual provenance, unauthorized interaction widgets, and report-text attempts to expand authority;
18. approval/report-state confusion, including proof that approving a report does not settle a mission or authorize an unrelated follow-up.

The expected security result is deterministic prevention by guards/policy, not merely model refusal.

## 20.2 Governed outcome metrics

Release and benchmark reports SHOULD distinguish:

- perfect, partial, verified, and settled success;
- false-done rate;
- duplicate side-effect rate;
- unauthorized-action attempt/block rate;
- human rescue/takeover rate;
- recovery success;
- evidence completeness and contradiction rate;
- retries by failure class;
- cost per verified and settled outcome;
- time to verification and settlement;
- performance by model, site, risk class, actuator, procedure version, and verifier.

A benchmark reaching the end of a flow is not reported as real external completion unless the Completion Contract and settlement requirements were met.

---

# 21. Unified acceptance criteria

The unified Cockpit may claim the intended product experience only when all of the following are true.

## 21.1 Comprehension

- A new user can identify project, current work, active state, and next action within 30 seconds.
- The default view does not require understanding UIAI/Focusa route names.
- Every advanced technical detail remains findable through Inspector, Capabilities, search, or Developer Mode.
- No feature in Annex A becomes inaccessible because it moved out of the default view.

## 21.2 Navigation

- All first-party capabilities are reachable through a workspace or Capabilities.
- Search finds capabilities, objects, evidence, settings, and Help.
- Work-object tabs preserve context across workspaces.
- Closing a view never silently destroys backend work.

## 21.3 Context and authority

- Context Control is always understandable and expands to full ScopeRef.
- Missing/stale/conflicting scope blocks writes with exact recovery.
- Supporting work does not transfer active scope.
- Node and thread ownership are visible before writes.
- Local Only remains functional without cloud.

## 21.4 Consistency

- Browser, test, document, research, and media work use common object, job, artifact, approval, event, and evidence patterns.
- One action never has materially different safety semantics merely because it appears in another workspace.
- Technical errors use the shared recovery envelope.

## 21.5 Progressive disclosure

- Ordinary work can be completed at Levels 0–2.
- Advanced configuration is hidden until requested but no more than two deliberate interactions away.
- Raw JSON/logs are not the default UI and remain available in Developer Mode.
- Healthy technical details remain quiet; risks become prominent.

## 21.6 Beauty and responsiveness

- The shell feels native and calm at default window size.
- Live/document/test content receives priority over chrome.
- Light, dark, compact, reduced-motion, keyboard, and accessibility modes work.
- No workspace looks like a generic admin dashboard or raw API console.

## 21.7 Proof

- Every meaningful run or transformation can produce bounded artifact/evidence refs.
- Evidence preserves scope, lineage, tool/version, redaction, and verification class.
- Focusa handoff uses verified scope.
- Public/cloud proof requires explicit redaction and consent.

## 21.8 Extensibility

- New first-party capabilities can register placement, commands, objects, artifacts, and inspectors without rewriting the shell.
- A missing custom workspace does not prevent basic declarative capability exposure.
- Third-party code cannot bypass adapters, scope guards, consent, or Focusa authority.

## 21.9 Governed execution and verified outcomes

- Consequential missions have a versioned Mission Contract or explicit documented degraded equivalent.
- Completion is evaluated against explicit predicates and evidence requirements.
- The UI distinguishes submitted, accepted, verified, provisionally complete, and settled.
- Deterministic policy—not model self-classification—enforces authority at the action boundary.
- Consequential actions expose an Action Proposal and emit a machine-readable receipt.
- Every receipt identifies worker, grant, target, actuator, result, evidence, verification, cost, retries, and settlement.
- Fallback routing never silently expands authority, origin access, data disclosure, or spending.
- Retries are bounded, classified, and reconcile ambiguous side effects before repeating.
- Shared resources use owners, leases, locks, version checks, or equivalent conflict prevention.
- Untrusted web/document/tool content cannot grant authority, disclose unrelated data, disable evidence, or become durable procedure automatically.
- Secrets use brokered or opaque-handle paths wherever possible.
- Contradictory or insufficient evidence blocks settlement.
- Cockpit remains Mission Experience; Focusa remains canonical Mission Kernel; UIAI remains execution/evidence plane.

## 21.10 Interactive report integrity and usefulness

- Reports are derived from canonical source refs and never become parallel mission truth.
- The default Report Canvas is understandable without raw logs and remains fully inspectable.
- Every verification visual is an actual attributable capture or explicitly labeled Illustrative.
- Annotations do not modify immutable originals and preserve author/time/geometry provenance.
- Report prose distinguishes fact, inference, prediction, contradiction, limitation, and missing evidence.
- Raw chain-of-thought, secrets, hidden values, and unbounded transcripts are excluded.
- Report widgets are declarative, permissioned, and routed through standard guards; no report can directly execute arbitrary scripts, prompts, tools, or browser actions.
- Approve report, accept outcome, authorize follow-up, request changes, review evidence, and usefulness rating have distinct semantics and event/receipt behavior.
- Follow-up work uses selected bounded refs and creates/proposes a real Workpoint or task.
- Approved/frozen versions are immutable, stale source changes are visible, and published shares are version-bound, revocable, and audience-scoped.
- Interactive and static exports preserve source attribution, decision state, redaction, and integrity manifests.
- Reports can be created contextually from every major work-producing workspace and found through Evidence/global search without adding default navigation clutter.

## 21.11 Design-system enforcement

- Every user-facing surface consumes shared semantic tokens and approved components.
- Token lint, accessibility checks, visual regression, responsive fixtures, reduced-motion checks, focus tests, overlay tests, and performance checks are CI-blocking.
- Typography, iconography, sidebars, inspectors, overlays, transitions, animations, forms, tables, empty states, loading, notifications, and technical-detail disclosure comply with Section 10.
- Every major workspace proves normal, loading, empty, blocked, degraded, error, approval, and success states.
- No persistent navigation, required status, warning, recovery, or action depends on icon, color, hover, tooltip, toast, animation, or drag alone.
- User-facing pull requests include the required screenshot/video evidence and identify shared components or an approved design exception.
- A feature cannot claim complete status while its UX states are unimplemented, visually inconsistent, inaccessible, or represented only by raw API output.


---

# Annex A — Feature preservation crosswalk

This annex is normative for the no-feature-removal rule.

## A.1 Current Cockpit foundation preserved

| Existing feature/requirement | Preserved placement |
|---|---|
| Local Only and Cloud Profile | Context Control, onboarding, Settings, Nodes & Services |
| UIAI, Focusa Local, Focusa Cloud, AI API planes | Capability metadata, inspector source labels, Nodes & Services |
| ScopeRef and explicit scope guard | Context Control, every adapter request, approval sheets |
| Multi-node NodeRef graph | Nodes & Services, Context Control, scope inspector |
| Pairing, revoke, repair, token rotation | Nodes & Services / Pairing; contextual recovery |
| Tailscale, Bonjour, saved/env, CLI discovery ladder | Onboarding and Nodes & Services |
| Keychain and local identity proof | Security settings and guarded adapter layer |
| CardManifest and contract mapping | Capability/card registry; dashboard and contextual cards |
| Unified error envelope | Every workspace and Activity |
| EventBus and append-only events | Activity, inspector history, audit |
| Scope/node-keyed local store | Shared platform architecture |
| RedactionBoundary | Cloud consent, export, evidence, support bundle |
| Evidence/proof preview | Evidence workspace and contextual capture |
| Cloud consent | Approval/consent pattern and settings grants |
| Release channels and metadata | Updates, About, release pipeline |
| Signed/notarized builds and rollback | Existing release contract unchanged |
| Performance budgets and regression gate | Platform and workspace performance tests |
| Local telemetry viewer | Activity / Audit and Developer inspector |
| Search, notifications, ribbon | Global search, Activity Bar, Activity workspace |
| Inspector content map | Universal inspector |
| Empty states, loading, confirmations | Shared interaction system |
| Undo/redo, bulk, export/import | Contextual object actions and Settings/Activity |
| Accessibility and i18n | Cross-cutting requirements |
| Retention and privacy deletion | Evidence/Activity retention settings |
| MCP client role | Capability parity and adapter metadata |
| Wirebot future slot | Future signed first-party module, same authority model |
| Multi-user and multi-operator posture | Node/scope/ownership behavior |
| Help and first-use tour | Help workspace and onboarding |
| Feature flags and gating | Capability states and Settings |
| Strict design-system guardrails | Section 10 tokens, components, motion, overlays, empty states, composability, and CI gates |
| Trial/license behavior | Nodes & Services and capability gating |

## A.2 Existing Phase 0 cards preserved

| Card | New primary presentation |
|---|---|
| UIAI Engine Health | Overview system posture; Nodes & Services; Capabilities |
| Browser Diagnostics | Live/Test Lab inspector; Activity; Capabilities |
| Project Identity | Context Control; Overview; inspector |
| Project Card | Overview and scope inspector |
| Workpoint Resume | Overview Continue card; inspector |
| Trajectory View | Overview/inspector deeper context |
| Tool Doctor | Nodes & Services and contextual recovery |
| DXUX Requirement | Recovery/help inspector and Capabilities |
| Work-loop Status | Overview/Activity/System |
| Device Pair Status | Nodes & Services / Pairing |
| Capture Evidence | Evidence and contextual action |
| Cloud Node Status | Nodes & Services |
| Device Pairing | Nodes & Services / Pairing |
| AI API Health & Usage | Nodes & Services / AI API; paid-action sheets |

Deferred Focusa cards—including Workpoint/Trajectory checkpoint, metacognition, prediction, context cognition, preload, sync, and node selection—remain preserved as later capability manifests and contextual actions.

## A.3 FPV feature preservation

| FPV feature | Preserved placement |
|---|---|
| Watchable share on session creation | Live share controls and session defaults |
| Public tokenized PWA | Mobile/share projection |
| Read-only default | Share role policy |
| Message, annotate, click, fill/type, press | Live/PWA control channel |
| Pi steering bridge | Live event/agent bridge |
| CDP screencast/MJPEG plus polling fallback | Live transport adapter |
| URL/title/status/diagnostics context | Live canvas and inspector |
| Repo/Focusa context | Scope inspector, with verified/degraded status |
| Mobile bottom sheet and responsive breakpoints | PWA experience |
| Touch, pinch, pan, long-press annotation | PWA controls |
| Push notifications | PWA and Activity notification policy |
| Reconnect and missed-event replay | FPV event protocol |
| Multi-tab viewers, one controller | Share/viewer state |
| Run/Pause/Takeover/Release | Live control state |
| Operator audit | Activity/Audit and session history |
| Multi-agent canvas | Live collection mode |
| Recording/replay | Live recordings and Test Lab |
| Fork from step | Live/Test Lab replay |
| Golden/human-run comparison | Test Lab/Studio |
| PII redaction support mode | Share/redaction policy |
| Accessibility-tree/DOM-diff/target modes | Live view modes |
| Performance/frame adaptation | Live transport policy |
| Floating/PiP window | Later desktop window feature |
| Workpoint binding and operator advancement | Focusa session/run binding |

## A.4 Browser UX/DX recommendations 1–102 preserved

Duplicate recommendations are consolidated into broader capabilities, but each requirement remains represented.

### Evidence, screenshots, and visual proof

| Original item | Preserved capability |
|---:|---|
| 1 | Direct screenshot output as JSON, file, or URL. |
| 2 | Multi-viewport responsive batch screenshot. |
| 15 | Resize without forced screenshot response. |
| 18 | Built-in visual diff against baseline/reference. |
| 25 | Freeze/throttle animations for deterministic capture. |
| 26 | Stable screenshot waiting for visual quiescence. |
| 29 | Scroll selected element into view before capture. |
| 32 | Element/region-only screenshot. |
| 33 | Saved baseline convention and automatic comparison. |
| 95 | Mark evidence screenshot as golden/baseline. |
| 96 | Session video recording. |
| 97 | Export and replay a session/audit script. |

Placement: Studio / Capture and Compare, Test Lab baselines, Live recording, Evidence.
