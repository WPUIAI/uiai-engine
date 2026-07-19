| Tooltip | Short label, shortcut, or explanation | Actions, warnings, long text |
| Menu | Compact command list | Forms or rich inspection |
| Popover | A few related controls or small contextual detail | Critical warning or long workflow |
| Inspector | Persistent contextual detail and actions | Blocking commitment |
| Sheet | Scoped task tied to the current window/object | Passive status or unrelated navigation |
| Dialog | Short decision requiring completion before continuing | Multi-page configuration |
| Alert | Brief high-consequence warning/confirmation | Routine success feedback |
| Command Palette | Search, navigation, commands, recent objects | Arbitrary settings form |
| Toast | Noncritical acknowledgment | Errors requiring recovery or irreversible outcomes |
| Banner | Persistent object/workspace issue with action | General notification history |

### Overlay quality rules

- Only one modal layer may be active at a time, except an operating-system authentication or file dialog.
- Modal content outside the dialog is inert.
- Opening a modal moves focus inside; closing returns focus to the initiator or the new logical result.
- `Escape` closes transient overlays unless closure would discard an in-progress destructive action, in which case a discard confirmation is required.
- Every dialog/sheet has a visible title and a visible close, Cancel, or Done path.
- Approval and destructive dialogs have stable button placement and default focus that does not increase accidental confirmation risk.
- Popovers close on outside interaction and `Escape`; they do not contain nested sheets or dialogs.
- A popover is limited to a small number of related tasks. If it needs navigation, long scrolling, or multiple sections, use an inspector or sheet.
- Warnings MUST use banners, sheets, dialogs, or alerts—not tooltips or transient popovers.
- Overlay width, max height, internal scroll region, and footer behavior are tokenized.
- Footer actions remain visible when long sheet/dialog content scrolls.
- The backdrop, border, shadow, and motion communicate depth without obscuring context.
- Overlay stacking and `z-index` MUST come from a shared layer scale.
- No workspace defines its own overlay root.

### Command Palette quality bar

- Opens with `Command-K` in under 100ms after warm start.
- Centers near the top of the active window rather than vertically centering over content.
- Separates Find and Do results.
- Shows object type, scope, source, and shortcut where relevant.
- Supports complete keyboard operation and type-ahead.
- Never exposes commands the current ScopeRef cannot execute without marking them blocked and explaining recovery.

## 10.13 Empty, zero, filtered, unavailable, and completion states

An empty state is a designed state, not leftover whitespace.

Each empty-state component MUST receive a typed reason:

```ts
type EmptyStateReason =
  | "first_use"
  | "no_items"
  | "no_results"
  | "filtered_out"
  | "not_configured"
  | "permission_or_scope"
  | "service_unavailable"
  | "all_complete";
```

### Required anatomy

- concise title, usually 2–6 words;
- one sentence explaining the specific condition;
- one primary next action when an action exists;
- optional quiet secondary action or documentation link;
- optional restrained symbol or lightweight illustration;
- no promotional copy.

### Guardrails

- Loading, failure, and empty are distinct states and MUST NOT be substituted for one another.
- “No results” MUST preserve the query and offer clear/reset/refine actions.
- “Filtered out” MUST show active filters and a clear-filter action.
- “Not configured” MUST explain what configuration is missing and whether it is local, node, cloud, license, or credential related.
- Scope/permission empty states MUST name the blocked authority and exact recovery action.
- “All complete” may be calm and affirmative but MUST not invent productivity claims.
- Empty-state illustrations MUST not dominate technical workspaces or consume most of the viewport.
- The first-use state SHOULD teach the mental model; repeat empty states SHOULD become more compact.
- Collection headers, search, and filters SHOULD remain visible where they help the user recover.

Every workspace MUST provide fixtures for all applicable empty-state reasons.

## 10.14 Loading, progress, feedback, and notifications

- Content expected within about 100ms renders immediately.
- Predictable content taking longer uses a structure-matching skeleton.
- Known-duration work uses determinate progress.
- Unknown-duration work uses indeterminate progress with a meaningful stage label.
- Work that can outlive the current view becomes a Job.
- Full-window loading spinners are prohibited for ordinary operations.
- Cached data remains visible with freshness and refresh status rather than disappearing during reload.
- Optimistic updates are allowed only when rollback is reliable and the UI clearly distinguishes proposed/pending from committed.

### Toasts

- Use for noncritical acknowledgments such as “Copied” or “Saved locally.”
- Maximum three visible toasts.
- Default duration about 4–6 seconds, paused on hover/focus.
- Toasts never contain the sole recovery path.
- Important events also persist in Activity.

### Banners

Use a banner for a persistent, scoped condition such as lost connection, stale scope, paused takeover, or document signature invalidation. A banner MUST include the affected scope/object and recovery action.

### Live/running indicators

Running state uses restrained, non-distracting feedback. Repeating pulse animations are limited to truly live/recording states and MUST stop when offscreen or reduced motion is enabled.

## 10.15 Motion, transitions, and animation

Motion explains relationship, state, and causality. It MUST NOT be used merely to make the interface feel active.

### Motion tokens

```css
--motion-instant: 80ms;
--motion-fast: 120ms;
--motion-standard: 220ms;
--motion-emphasized: 300ms;
--motion-slow: 360ms;

--ease-standard: cubic-bezier(0.2, 0, 0, 1);
--ease-enter: cubic-bezier(0.0, 0, 0, 1);
--ease-exit: cubic-bezier(0.3, 0, 1, 1);
```

### Timing guide

| Interaction | Target duration |
|---|---:|
| Hover/focus/color feedback | 80–120ms |
| Menu/tooltip/popover | 120–180ms |
| Tab/content state transition | 120–220ms |
| Sidebar/inspector collapse | 180–240ms |
| Sheet/dialog entrance | 180–260ms |
| Major workspace/object transition | 220–300ms |
| Rare onboarding/emphasized transition | 300–360ms maximum |

### Motion rules

- Routine UI motion MUST remain under 400ms.
- Motion MUST be interruptible; rapid input must not queue stale animations.
- Prefer opacity and transform. Layout-property animation requires measured proof that it remains smooth.
- Animate from the initiating control or selected object when that relationship improves orientation.
- Do not animate live browser frames, videos, document pages, logs, or rapidly updating tables merely because surrounding UI changes.
- Avoid bounce, overshoot, parallax, spinning icons for ordinary progress, and decorative looping.
- Do not animate multiple large regions in opposing directions simultaneously.
- Sidebar, inspector, and overlay transitions MUST preserve focus and not cause unexpected scroll jumps.
- Success/error feedback SHOULD use subtle icon/color transition rather than celebratory motion.
- Motion MUST not delay interaction; controls become interactive when visually available.

### Reduced motion

When `prefers-reduced-motion: reduce` or the Cockpit setting is active:

- remove spatial movement and zooming;
- disable smooth scrolling and parallax;
- replace slide/scale transitions with an opacity change of 0–100ms or no transition;
- stop nonessential looping indicators;
- preserve immediate state feedback;
- never require animation to understand change.

Reduced-motion behavior MUST be covered by visual and E2E fixtures.

## 10.16 Composability and component architecture

The design system is composable at four levels:

```text
Primitives  → Button, Input, Tabs, Dialog, Splitter, Disclosure
Components  → ScopePicker, StatusBadge, ObjectRow, ApprovalSummary
Patterns    → InspectorPanel, JobProgress, EmptyState, ArtifactTimeline
Workspaces  → Live, Test Lab, Documents, Research, Studio
```

### Component contract

Every reusable component MUST declare:

- purpose and non-goals;
- inputs/props and emitted intents;
- controlled/uncontrolled state posture;
- visual states;
- loading/empty/error/blocked behavior;
- keyboard interaction;
- focus behavior;
- screen-reader semantics;
- responsive behavior;
- density behavior;
- theming/token dependencies;
- slots/composition points;
- fixture and test coverage.

### Separation of concerns

- Visual components MUST NOT call UIAI, Focusa, Cloud, AI API, or Tauri endpoints directly.
- Components receive view models and emit user intent.
- ScopeRef, NodeRef, authority, consent, and adapter execution remain outside visual components.
- Components MUST NOT infer project or node from singleton global state.
- Headless interaction/state logic SHOULD be separated from presentation when reused.
- Workspaces compose shared patterns rather than duplicating markup and behavior.
- A component MUST support all declared states rather than relying on the parent to hide broken states.

### Composition rules

- Prefer explicit slots/regions such as `header`, `toolbar`, `content`, `footer`, and `details` over a large set of styling props.
- Avoid “god components” with unrelated modes selected through dozens of booleans.
- Use discriminated unions/state machines for materially different states.
- Variants are semantic (`primary`, `destructive`, `quiet`, `selected`), not visual (`blue`, `red`, `shadowed`).
- Extension content is hosted inside approved shell regions and MUST inherit tokens, overlay roots, focus management, and event/approval patterns.
- Workspace modules MAY provide specialized viewers, but they MUST use shared chrome, jobs, approvals, empty states, notifications, and inspectors.

## 10.17 Microcopy and technical communication

Cockpit writing is concise, literal, and calm.

- Use sentence case.
- Lead with the human meaning, then expose technical detail.
- Use verbs for actions and nouns for objects/views.
- Avoid “Oops,” “magic,” “AI-powered” filler, and congratulatory language for routine work.
- Do not blame the user.
- State whether work was committed, still running, blocked, or safely preserved.
- Error copy follows: **what happened → impact → next action**.
- Confirmation copy names the exact object, destination, and consequence.
- Buttons use the action result: `Apply redactions`, `Run flow`, `Revoke device`, not `Yes`.
- Labels remain consistent with the shared object and state vocabulary.
- Acronyms and implementation jargon are expanded or explained on first use unless universally expected for the target technical audience.

## 10.18 Visual data, code, and raw technical detail

Technical depth remains available without overwhelming the normal view.

- Summaries are the default.
- Structured detail appears in Inspector.
- Raw JSON, request/response envelopes, stack traces, and route metadata appear in Developer Mode.
- Raw content MUST be redacted before render and copy.
- Code and JSON viewers provide syntax highlighting, wrapping toggle, line numbers where useful, copy, and bounded expand/load behavior.
- Large logs are virtualized and filterable.
- Charts are used only when they reveal trend, distribution, comparison, or capacity better than text.
- Every chart includes a textual summary and accessible data path.
- Decorative dashboard charts are prohibited.

## 10.19 Design QA and automated enforcement

A feature is not complete until its experience states are proven.

### Required component fixtures

Each shared component and major workspace MUST provide fixtures for applicable states:

- default;
- hover/focus/active;
- selected;
- loading;
- empty first-use;
- empty filtered/no-result;
- blocked scope/auth/license;
- degraded/offline;
- error and recovery;
- long content;
- localization expansion;
- light/dark;
- comfortable/compact;
- reduced motion;
- keyboard-only;
- 200% zoom.

### CI-blocking design gates

1. **Token lint** — no unapproved raw visual values.
2. **Component contract lint** — public shared components include documentation and fixtures.
3. **Accessibility tests** — automated axe checks plus keyboard/focus tests for interactive patterns.
4. **Visual regression** — baseline screenshots for shell, overlays, empty states, major workspaces, light/dark, density, and reduced motion.
5. **Responsive matrix** — desktop at minimum 1024×768, 1280×800, 1440×900, and 1920×1080; PWA retains 375/768/1024/1440 coverage.
6. **Zoom/text expansion** — critical workflows pass at 200% zoom and representative long translations.
7. **Motion test** — reduced-motion mode contains no prohibited spatial/looping motion.
8. **Performance test** — no interaction or transition regression beyond the established budget; overlay and sidebar transitions remain frame-stable.
9. **Focus test** — focus is visible, not obscured, trapped only inside modal surfaces, and returned correctly.
10. **Overlay test** — no nested modal roots, background inertness and Escape behavior proven.
11. **Target/drag test** — pointer targets and non-drag alternatives satisfy the accessibility posture.
12. **No-console-error gate** — shared component fixtures and critical E2E flows produce no unexpected warnings/errors.

### Human design review gate

A reviewer MUST verify:

- clear primary action;
- hierarchy at first glance;
- consistency with existing patterns;
- progressive disclosure level;
- empty/loading/error/blocked states;
- keyboard and screen-reader posture;
- focus and overlay behavior;
- dark/compact/reduced-motion behavior;
