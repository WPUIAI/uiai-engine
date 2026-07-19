- Ordinary body copy MUST NOT be smaller than `body` in primary workspaces.
- `micro` MUST NOT carry required instructions, errors, consent language, or sole status meaning.
- A surface MUST use no more than four type styles at once unless it is a document/code viewer.
- Weight, size, spacing, and color MUST work together; hierarchy MUST NOT rely only on font size.
- Uppercase text is prohibited for sentences and navigation. It MAY be used for short, familiar technical abbreviations.
- Labels and headings use sentence case.
- Numeric dashboards SHOULD use tabular numerals.
- Monospace is reserved for values where character identity matters: code, commands, paths, hashes, evidence refs, IDs, timings, and aligned numeric data.
- Human-readable titles MUST remain in the UI font even when the underlying object has a technical identifier.

### Readability and truncation

- Long-form readable content SHOULD remain within roughly 60–85 characters per line.
- Metadata may truncate only when the complete value is available through tooltip, inspector, copy action, or expansion.
- Primary object titles SHOULD wrap to two lines before truncating.
- Error messages MUST NOT truncate the recovery action.
- Path shortening MUST preserve the meaningful final segments and expose the full path on demand.
- Typography MUST remain usable at 200% zoom and with operating-system text enlargement.

## 10.5 Iconography

Cockpit MUST use one primary interface-icon family with consistent optical weight. The Svelte implementation SHOULD use Lucide-style line icons or an equivalent source-owned set that can be licensed and rendered consistently across platforms. Native macOS integrations MAY map equivalent actions to SF Symbols when the mapping is exact and platform appropriate.

### Icon sizes

| Context | Optical box |
|---|---:|
| Inline with caption/body | 14–16px |
| Standard control or row | 16–18px |
| Toolbar / primary navigation | 18–20px |
| Workspace identity / empty state | 24–32px |
| Illustrative empty-state symbol | 40–56px maximum |

### Icon rules

- An icon expresses one recognizable concept.
- Persistent navigation MUST include a text label in expanded mode.
- Icon-only controls MUST have an accessible name and a tooltip.
- Tooltips MUST repeat the action label and MAY include the shortcut.
- Outline icons are the default. Fill or hierarchical emphasis is reserved for active selection, recording, or a small number of strong states.
- Emoji MUST NOT be used as interface icons.
- Destructive actions MUST NOT rely on a trash icon alone; destructive meaning is reinforced through label, placement, and confirmation.
- Status icons MUST pair with text in any decision-relevant state.
- Custom icons MUST use SVG, a shared grid, consistent stroke/weight, optical alignment, and RTL/localized variants where direction or characters matter.
- Workspace icons remain stable after release; changing them requires a migration note and updated onboarding/help visuals.
- Decorative icons that do not improve recognition are prohibited.

## 10.6 Color, materials, and visual hierarchy

Color communicates selection, status, and action. It does not assign a loud brand color to every module.

### Surface hierarchy

Use four visual layers:

1. **Window** — quiet application background.
2. **Navigation/chrome** — sidebar, toolbar, and Activity Bar.
3. **Workspace** — the primary content canvas.
4. **Raised** — inspector, popover, sheet, dialog, command palette, and transient control surfaces.

Translucency and blur MAY be used on chrome and overlays when they preserve contrast and improve spatial hierarchy. They MUST NOT reduce legibility, create muddy stacking, or turn every surface into glass.

Cards are used for bounded summaries, grouped actions, comparisons, and onboarding. Cards MUST NOT become the default wrapper around every section, table, or text block.

### Status color contract

| Meaning | Semantic color | Required companion |
|---|---|---|
| Active/selected/informational | Accent/blue | selection treatment or label |
| Verified/success | Green | status text/icon |
| Attention/stale/degraded/pending | Amber | exact status and next action |
| Failed/destructive/hard block | Red | explicit cause/action |
| Neutral/offline/secondary | Gray | status text |

Color MUST never be the only signal. Contrast MUST meet WCAG 2.2 AA for text and interactive boundaries. Focus appearance SHOULD meet the stronger 2-pixel-perimeter and 3:1 contrast posture described by WCAG 2.2 Focus Appearance guidance.

### Dark mode

Dark mode is designed, not inverted.

- Preserve hierarchy and contrast without pure-white text everywhere.
- Reduce shadow dependence; use borders and surface luminance.
- Avoid saturated status backgrounds behind long text.
- Test screenshots, live video, PDF pages, charts, and code blocks independently.
- Images and browser/document content MUST not be globally color-inverted.

## 10.7 Layout, alignment, and density

The default desktop layout uses a stable alignment system:

```text
Window
  ├── unified toolbar
  ├── primary sidebar
  ├── work-object tabs
  ├── workspace canvas
  ├── optional inspector
  └── Activity Bar
```

### Layout guardrails

- Align major regions and repeated controls to shared edges.
- Use whitespace to establish groups before adding borders or cards.
- A workspace MUST have one dominant content region.
- Control clusters MUST reflect task order.
- Primary actions appear in predictable trailing/toolbar positions.
- Do not center ordinary desktop forms or technical work in large unused canvases.
- Do not present dense comparative data as unrelated cards when a table or list is more appropriate.
- Do not present a small set of distinct, visual objects as a dense table when a gallery or canvas is more appropriate.
- Nesting beyond two visible container levels SHOULD be avoided.

### Density modes

- **Comfortable** is the default.
- **Compact** reduces list/table row height and horizontal spacing but does not reduce minimum text size, focus visibility, or pointer target safety.
- Density changes MUST be implemented through tokens, not component-specific overrides.
- A workspace MAY force comfortable density for approval, onboarding, or sensitive forms.

### Control heights

| Control | Desktop | PWA/touch |
|---|---:|---:|
| Compact | 28px | not used |
| Standard | 32px | 44px minimum |
| Prominent | 36px | 48px |

Desktop controls smaller than 24×24 CSS pixels require sufficient spacing or an equivalent accessible target. Dragging interactions MUST provide a non-drag alternative.

## 10.8 Collapsible and resizable sidebars

The Cockpit uses sidebars to expose broad, relatively flat navigation and contextual collections. Sidebars MUST NOT become general-purpose dumping grounds.

### Primary sidebar states

```text
Expanded  → icons + labels + groups, default width 240px
Compact   → icon rail with tooltips, default width 64px
Hidden    → workspace consumes the region
Overlay   → temporary sidebar on constrained windows
```

### Primary sidebar rules

- Expanded width SHOULD be user-resizable from 208–320px.
- The selected workspace MUST remain identifiable in every state.
- Expanded/compact/hidden preference is saved per window and restored when the window remains large enough.
- Responsive adaptation MAY temporarily override the preference but MUST restore it when space returns.
- Collapsing MUST preserve scroll position, selection, badges, and expanded group state.
- A collapsed section with an active descendant MUST retain an active indicator.
- Sidebar group disclosure buttons MUST use semantic buttons with `aria-expanded` and keyboard support.
- Navigation nesting SHOULD remain at two levels or fewer. A true hierarchical object tree uses a reviewed tree pattern rather than improvised indentation.
- Workspace navigation, object collections, and settings navigation MUST not be mixed without clear grouping.
- Badges show actionable counts only; decorative counts are prohibited.
- `[` toggles the primary sidebar and the menu command remains available.

### Contextual sidebar and collection panes

A workspace MAY provide a secondary collection/outline pane, such as PDF thumbnails, test flows, or Live sessions.

- It MUST collapse before the primary canvas is compromised.
- It SHOULD be independently resizable.
- Minimum and maximum widths MUST be tokenized.
- It MUST expose a clear title and selected item.
- It MUST not duplicate the universal inspector’s purpose.

### Inspector behavior

The inspector is not a navigation sidebar. It contains details and actions for the current selection.

```text
Docked:   320–420px, resizable
Overlay:  medium-width windows
Sheet:    narrow windows / mobile projection
Hidden:   user-controlled
```

The inspector MUST remember the last selected inspector tab per object type, not globally across unrelated objects.

## 10.9 Toolbars, controls, menus, and direct manipulation

### Toolbar composition

A toolbar uses predictable zones:

- leading: navigation and view controls;
- center: object title, breadcrumbs, search, or mode selector;
- trailing: primary action, run/control state, share, and overflow.

A toolbar MUST avoid more than one visually dominant action. Low-frequency actions move to overflow, context menus, Inspector, or Command Palette.

Toolbars SHOULD allow customization only after the base command model is stable. Customization MUST not remove essential recovery or safety controls.

### Buttons

- Button labels begin with a verb when they perform an action.
- Primary buttons represent the current surface’s most likely next action.
- Secondary buttons are visually quieter.
- Destructive buttons use destructive styling only at the final commitment step.
- A disabled button MUST have an discoverable reason nearby or in its tooltip; prefer a blocked state with recovery over unexplained disablement.
- Loading buttons preserve their width, label context, and cancellation posture.
- Buttons MUST not move when their state changes.

### Menus

- Menus contain commands, not arbitrary content.
- Group commands by intent and separate destructive items.
- Include shortcuts when available.
- Checkmarks indicate persistent state; radio behavior indicates exclusive choice.
- A context menu MUST not be the only path to an essential action.

### Direct manipulation

Direct manipulation is encouraged for browser, document, visual, and timeline canvases, but every drag, spatial, or gesture action MUST have a keyboard and single-pointer alternative.

Selection handles, resize affordances, drop targets, and active control ownership MUST be visually clear. An agent-controlled or operator-controlled canvas state MUST be unmistakable.

## 10.10 Forms and configuration surfaces

Forms MUST support comprehension before density.

- Use visible labels; placeholders are examples, not labels.
- Place labels consistently above fields unless a compact inspector row has a proven need for side-by-side layout.
- Group fields by user decision, not backend request shape.
- Put advanced, uncommon, or dangerous fields behind a labeled disclosure section.
- Explain units, defaults, inheritance, and data destination.
- Validate on blur or submit; avoid noisy validation on every keystroke unless the input is inherently live.
- Preserve valid input after an error.
- Move focus to the first invalid field only after a submitted validation failure, and provide an error summary for long forms.
- Passwords, tokens, and secrets are never displayed in full after storage.
- File/path pickers use native Tauri/OS dialogs when appropriate.
- Every mutation form shows the effective project, node, scope, and output before commitment.

Configuration pages MUST distinguish:

- effective value;
- inherited/default value;
- local override;
- project override;
- environment-enforced value;
- unavailable/locked value.

## 10.11 Tables, lists, timelines, trees, and canvases

Use the component that matches the information model.

### Tables

Use tables for comparison across consistent fields.

- Headers remain visible during long scrolling when useful.
- Sorting state is explicit and keyboard accessible.
- Row actions do not crowd every column; select a row and use toolbar/inspector actions when many commands exist.
- Columns may resize, hide, or reorder only when the preference is persisted and reversible.
- Horizontal scrolling is a last resort; prioritize and progressively disclose columns.
- Dense tables MUST preserve at least the minimum pointer-target/spacing posture.
- Empty, loading, filtered, and error states belong inside the table region without destroying headers unnecessarily.

### Lists

Use lists for heterogeneous objects with one dominant label and a small set of metadata. Selection, focus, hover, unread, running, and error states MUST be visually distinct.

### Timelines

Timelines show causal or chronological work. They MUST distinguish:

- agent action;
- operator intervention;
- system event;
- artifact/evidence capture;
- failure/recovery;
- approval/decision.

Time compression and grouping MUST preserve access to individual events.

### Trees

Use a tree only for true parent-child hierarchy. Tree keyboard behavior MUST follow a tested accessible pattern. Ordinary navigation SHOULD use disclosure sections instead of ARIA tree semantics.

### Canvases

Browser, document, image, and comparison canvases MUST provide:

- zoom and reset;
- keyboard-reachable controls;
- current scale/page/frame;
- selection description;
- alternate semantic representation where possible;
- no essential action available only through hover or precision drag.

## 10.12 High-quality overlays and modality

Cockpit MUST use a consistent overlay taxonomy. The smallest suitable surface wins.

| Surface | Appropriate use | Prohibited use |
|---|---|---|
