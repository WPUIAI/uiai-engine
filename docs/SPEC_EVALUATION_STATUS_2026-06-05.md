# UIAI Spec Evaluation Status — 2026-06-05

Purpose: current implementation-grounded register for remaining UIAI spec work after Source-to-Markdown, agent discovery, Pi/MCP/CLI, and Focusa packet MVP work.

## Authority notes

- Project scope: `/home/wpuiai/uiai-engine`.
- Beads root epic for live remaining work: `uiai-engine-44k`.
- Focusa-related implementation must be grounded in Focusa local draft repairs:
  - `/home/wirebot/focusa/docs/98-project-root-crdt-reconciliation-foundation-spec.md`
  - `/home/wirebot/focusa/docs/99-original-intent-vs-implementation-audit.md`
- Focusa spec 98/99 core rule for UIAI: UIAI outputs are proposal/evidence execution artifacts, never Focusa cognitive authority; durable authority begins only after Focusa capture/intake/link succeeds.
- Spec 98/99 also require cross-surface parity for `canonical`, `advisory`, `degraded`, `stale`, `scope_status`, `evidence_refs`, `preferred_focusa_tool`, `next_tools`, and `recovery_hint` semantics across UIAI, Pi, MCP, CLI, Focusa tools, and docs.

## Implemented / mostly implemented

| Surface | Current status | Evidence |
|---|---|---|
| Agent discovery | Implemented | README agent section; `docs/AGENT_DISCOVERY_INDEX.md`; `/api/tools/*`; Pi and MCP card/search/graph tools. |
| Source-to-Markdown MVP | Implemented | `POST /api/markdown`; `browser_read format=markdown`; Pi/MCP/CLI exposure; generic browser capture plus GitHub/Reddit/X/Hacker News/YouTube adapters. |
| Source-to-Markdown records | Implemented in current work slice | `internal/routes/markdown.go` emits `uiai.source_markdown_record.v1` records for MVP adapters including Hacker News/YouTube plus `format=jsonl`, JSONL lines, and `uiai.source_markdown_chunk.v1` refs. |
| WPUIAI research-card/report objects | Engine implemented; contract documented; plugin pending | `/api/markdown` includes `wpuiai.research_card` / `wpuiai.report`; `docs/WPUIAI_RESEARCH_CARD_INTEGRATION_CONTRACT.md` defines plugin save/display contract. |
| Focusa ResearchDiagnosticsPacket MVP | Implemented enough to supersede older planning matrix | Pi builder, HTTP endpoint `/api/agent/research-packet`, MCP composer, CLI compose/smoke, packet drift/smoke scripts. |
| Browser diagnostics baseline | Implemented | Diagnostics route, diagnostics docs, Focusa diagnostics intake path, smoke/proof scripts. |
| Search provider baseline | Implemented/deepened | Provider-neutral `/api/search`, `/api/search/providers`, Brave default, keyless Wikipedia second provider, cache TTL, bounded timeout, redaction, evidence refs. |
| Agent pressure summary | Implemented/deepened in current work slice | `/api/health/browser` and `/api/metrics/browser` expose `uiai.agent_pressure.v1` noncanonical operational telemetry with browser/search/cache/error pressure and bounded actions. |

## Stale / superseded docs to reconcile

| Doc/spec | Status | Action |
|---|---|---|
| `docs/RESEARCH_DIAGNOSTICS_PACKET_SCHEMA_DECISION_MATRIX.md` | Reconciled as historical implementation-resolution artifact. | Header/checklist now mark implemented surfaces and Focusa specs 98/99 authority grounding. |
| `docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md` §8/§9 gap tables | Mixed: packet/Pi workflow gaps mostly implemented; observability/provider/drift/foundation cautions remain. | Refresh gap table; ground all Focusa integration language in Focusa specs 98/99. |
| `docs/SOURCE_TO_MARKDOWN_AGENT_SPEC.md` status language | Mixed: product spec still useful; some MVP/draft language stale. | Update status, mark implemented adapters, retain future adapters/output modes. |
| Older parity/gap inventories from March/early planning | Potentially stale. | Treat as history unless current public benefit or proof gap still exists. |

## Live incomplete work mapped to beads

| Bead | Live gap | Notes |
|---|---|---|
| `uiai-engine-44k.1` | Explicit Source-to-Markdown JSONL/chunked output mode | Implemented in this slice; pending tests/smoke before closing. |
| `uiai-engine-44k.2` | Hacker News public adapter | Implemented in this slice; pending tests/smoke before closing. |
| `uiai-engine-44k.3` | YouTube transcript/metadata adapter | Implemented in this slice; pending tests/smoke before closing. |
| `uiai-engine-44k.4` | WPUIAI plugin UI/save contract for research cards | Implemented as engine-side integration contract; plugin repo implementation remains separate follow-up. |
| `uiai-engine-44k.5` | Observability pressure summary deepening | Implemented in this slice; pending tests/smoke before closing. |
| `uiai-engine-44k.6` | Second search provider | Implemented in this slice with keyless Wikipedia provider; pending tests/smoke before closing. |
| `uiai-engine-44k.7` | Docs/public-benefit drift automation | Implemented in this slice: CI now runs tool parity; docs completeness guards adapters, providers, agent_pressure, WPUIAI contract, and packet reconciliation. |
| `uiai-engine-44k.8` | Stale ResearchDiagnosticsPacket matrix reconciliation | Implemented in this slice; pending docs checks before closing. |

## Focusa specs 98/99 grounding checklist for any Focusa-facing UIAI change

1. Treat `focusa_scope` as metadata; validate against Focusa ProjectIdentity/Workpoint when making capture recommendations.
2. Render UIAI packets as `proposal_only` until Focusa capture/intake/link returns success.
3. Keep stable evidence handles; avoid raw screenshots, HAR-like data, cookies, auth headers, full bodies, and raw SERP blobs in Focusa.
4. Preserve explicit `project_root`, `continuity_id`, `workpoint_id`, and evidence refs in packet/capture handoffs.
5. Keep UIAI browser/search/cache pressure as noncanonical operational telemetry that may narrow/block workflows but never mutates cognition truth.
6. Maintain Pi/MCP/HTTP/CLI parity for packet schema, diagnostics redaction, scope echo, and evidence refs.
7. For foundational Focusa contract shifts, require cross-repo proof: Focusa tool contract checks plus UIAI parity, Pi rendering, packet drift/smoke, and diagnostics smoke.

## Recommended next evaluation sequence

1. Finish `uiai-engine-44k.1` validation: Go route tests, tool parity, and Source-to-Markdown smoke.
2. Close `uiai-engine-44k.4` after docs checks.
3. Close `uiai-engine-44k.8` after docs checks, then continue adapter/provider/observability follow-ups.
