# UIAI-COCKPIT-006 Studio Creative Workbench Amendment

**Number:** `UIAI-COCKPIT-006`  
**File:** `UIAI_COCKPIT_006_STUDIO_CREATIVE_WORKBENCH_AMENDMENT_2026-08-11_v0.1.md`  
**Status:** Iterable Draft v0.1 — not normative until Adopted  
**Parent:** `UIAI-COCKPIT-000 v0.5` (§7.6 Studio), `UIAI-COCKPIT-004`* (scoped work surfaces, tabs/panes/windows/motion) + `UIAI-COCKPIT-005`* (universal agent control, multimodal visual workspace runtime) — where * denotes origin/main 004/005-NEXT merged 2026-08-11 (85246c9)  
**Repo:** `WPUIAI/uiai-engine` branch `feat/cockpit-ota-updates`  
**Date:** 2026-08-11  

> **Intent:** Make Studio the human-visible *Create* workbench for all visual/creative production — not only video — by projecting Engine capabilities through Workstream-scoped surfaces (004) and Universal agent control (005), without swallowing Documents or duplicating Focusa Mission Kernel.

## 0. Decision summary

Studio remains `000 V05 §7.6` five-section workbench (Capture → Compare → Analyze → Design → Produce) but is now **workstream-scoped, multimodally extensible, and agent-human collaborative**:

- **Produce** extends HyperFrames video (WorkRouter product videos via Three/Theatre/GSAP postprocessing) plus mockups/GIFs;
- **Capture/Compare/Analyze/Design** stay first-class per `V11` route table;
- **Whiteboard/tldraw-offline** is the offline-first canvas primitive for human↔agent sketch/canvas collaboration (sits under 005 whiteboards/DataViews);
- **Generative GUIs** are block-recipes/design-system/content-map pipelines that emit versioned artifacts into Studio/Design and Evidence;
- **Google-Docs-like collaboration** is delivered via Documents + Report Canvas (001) + live SSE/BroadcastChannel sync (H44), not by embedding Google Docs.

Cockpit never becomes canonical store: Focusa Daemon (Mission Kernel) owns Mission/Workpoint/Trajectory/Evidence; Engine owns browser execution; Studio owns *presentation & bounded edit*.

## 1. Thesis

> One Studio, five sections, many surfaces — same-session, scope-bound, settlement-aware.

- Engine = headless browser + media produce (screenshot/frame/media/produce/critique/reverse);
- Cockpit Studio = desktop projection (tabs/panes/windows per 004) with universal control (per 005);
- Document truth stays in Engine + Evidence; Studio edits are proposals until verified/settled.

## 2. Scope (in / out)

**In (§7.6):**
- Capture: one-shot/session/responsive/region, stable-mode, frames, baselines
- Compare: screenshot/layout/scenario diff, threshold/region inspection
- Analyze: critique, section-detect, ui-reverse, a11y/contrast, diagnostics
- Design: design-system, content-map, block-recipes → WPUIAI/Automations
- Produce: device mockups, GIFs, HyperFrames product/E2E videos, illustrations

**Whiteboard/offline canvas (tldraw-class):** Workstream-scoped, offline-first, CRDT/LWW via SSE + BroadcastChannel (H44), explicit presence/cursor, undo/redo per surface — maps to 005 whiteboards/DataViews/Flint.

**Generative GUIs:** manifest-declared Level A capabilities (V11/V15) — input schemas → pipelines (`/api/design-system`, `/api/content-map`, `/api/block-recipes`) → generated workspaces/dashboards — always consent-gated, cost-visible, provenance-tagged.

**Google-Docs-like:** reuse Documents runtime + Report Canvas (001) for versioned, evidence-backed interactive docs; no direct Google Docs drive sync in v0.1 (future connector is Level A adapter if approved).

**Out:** canonical Mission/Workpoint storage (Focusa), raw route clutter (V11), Execution Truth forgery (must preserve observationVersion/provenance/VRP).

## 3. Architecture

```text
Focusa Mission Kernel (Daemon) — mission/workpoint/trajectory/evidence/settlement
        ↕ pairing/scope-reconciliation (005 pairing)
UIAI Engine (Go) — sessions/targets/observations/actions/diagnostics/media/produce
        ↕ same-session presentation + packaged runtime (004)
Cockpit Studio (Tauri SvelteKit, workstream-scoped surfaces) — Capture/Compare/Analyze/Design/Produce + whiteboard + generated GUIs
```

- **Workstream scope (004):** every Studio surface/tab/pane/window binds to a Workstream ScopeRef; cross-scope drag = explicit move with confirmation.
- **Universal control (005):** single semantic inventory (`controlId` + `kind` + `provenance`) drives GUI/CLI/API/MCP/Pi parity; Studio commands surface through same registry as 002 agent-first.
- **Agent-first (002):** compact observation → typed action → authority check → delta → predicate → settlement; never `tool call == completion`.

## 4. Studio sections → Engine routes

| Studio | Primary routes | Cockpit placement | Gate |
|---|---|---|---|
| Capture | `/api/screenshot/*`, `/api/session/*/screenshot` | Studio/Capture + Live | first-class |
| Compare | `/api/comparison`, `/api/layout-compare`, `/api/media/frame/*` | Studio/Compare | gated pipeline |
| Analyze | `/api/critique*`, `/api/ui-reverse`, `/api/section-detect`, `/api/style-enhance`, `/api/reference/analyze` | Studio/Analyze | gated paid |
| Design | `/api/design-system`, `/api/content-map`, `/api/block-recipes` | Studio/Design + Automations | gated pipeline |
| Produce | `/api/media/produce`, jobs/status, artifacts | Studio/Produce + Activity | gated lifecycle/cancel |
| Whiteboard | `/api/events/publish`, SSE/BroadcastChannel, `/api/session/*/eval/css` (canvas ops) | Studio/whiteboard surface | scoped, undo/presence |
| Generated GUI | pipeline outputs → generated workspace | Studio/Design, Evidence | manifest, versioned |

All rows stay discoverable via Capabilities (`V11 §14`), not raw route list.

## 5. Whiteboard / tldraw-offline detail

- Offline-first LWW CRDT (H44), SSE publish/subscribe + BroadcastChannel intra-window.
- Strokes/shapes as ``surfaceId`` objects with `ownerScopeRef`, `provenance`, `observationVersion`-guarded actions.
- Human and agent share cursors/selections; agent edits via semantic ``whiteboard.addStroke / updateShape`` commands, never raw DOM.
- Persisted as artifact (JSON + PNG export) → Evidence; Report Canvas can embed live or frozen view.

## 6. Generative GUIs

- Level A declarative capability: manifest + input/output schemas + cost/consent policy.
- Pipelines emit versioned ``design-system.json``, ``content-map.json``, ``block-recipes/`` → Studio Design preview + Evidence receipt.
- Human approves before Automations/Live consumption; every generation tagged with model/provider, cost, source observationRef.

## 7. Google-Docs-like collaboration

- Primary = Documents (PDF/Office) + Report Canvas (001) + H44 live sync. Studio/Analyze outputs can ``generate Report Canvas`` for review/decisions.
- No silent Google Drive sync in 006 v0.1. Future `GoogleDocsAdapter` would be Level A (adapter mapping + side-effect policy) and must respect Focusa Evidence settlement.

## 8. Security / authority

- Same-session only (004 DSP-001/003): Studio never re-navigates Engine session to present; uses packaged runtime via `/present`.
- All mutating actions scoped to Workstream; cross-scope needs explicit confirmation.
- Signed capability grants per 002 ledger; media produce respects concurrency/limits.
- Presence/collab never leaks across ScopeRefs.

## 9. Acceptance (v0.1 iterable)

- [ ] Studio shell renders five sections with workstream-scoped tab/pane/window binding (proof: 004 motion spec tabs open/close/split/restore without scope leak).
- [ ] Whiteboard surface opens offline, syncs SSE+BroadCastChannel, undo/redo works, agent stroke via semantic command lands, persists to Evidence (headless test).
- [ ] Design pipeline (`block-recipes`) manifests visibly, consent-gated, produces artifact that Report Canvas embeds.
- [ ] Produce pipeline (`media/produce`) lifecycle/cancel/artifact proof under Studio/Produce (stubs allowed in v0.1).
- [ ] All Studio routes discoverable via Capabilities, gated appropriately (V11 table passes).
- [ ] Docs register patched with 006; no 000 drift (generated DTOs via contracts, not hand copy).

## 10. Rollout

- Merge `origin/main` (done 85246c9) → branch stays 192 ahead — next rebase is merge-only until 004/005 number collision formally amended (follow-up renumber 004-NEXT/005-NEXT → 007/008 or Adopted supersede).
- v0.1 lands as iterable draft; proof via `focusa bg` lanes: `cargo check`, `pnpm build`, `cargo test --workspace`, headless pane/whiteboard sim.
- No new canonical storage; all persistence goes through existing Evidence/artifact paths.

