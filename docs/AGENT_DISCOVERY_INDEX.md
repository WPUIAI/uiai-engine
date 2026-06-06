# UIAI Agent Discovery Index

Purpose: one-page index for how agents discover UIAI Engine features without loading the whole repo or guessing tool names.

## Highest-value public benefits

Public docs and cards should lead with these benefits:

| Benefit | Why agents care | Discovery path |
|---|---|---|
| Instant feature discovery | Agents can orient without dumping every schema into context | `/api/tools/agent-card`, `/api/tools/search`, `/api/tools/graph`, `/api/tools/docs`, Pi/MCP cards |
| Source-to-Markdown / Web Memory Capture | Public URLs become clean Markdown, records, metadata, diagnostics, Focusa refs, and WPUIAI research cards | `/api/markdown`, `uiai_source_to_markdown`, `source_to_markdown`, `scripts/uiai markdown` |
| Persistent browser workflows | Agents keep state across navigation, forms, reads, snapshots, @refs, cookies, and auth save/load | `/api/session/*`, `uiai_browser_*`, `browser_*` |
| Diagnostics-first debugging | Console, exceptions, failed network, and structured error proof replace guesswork | `/api/session/{id}/diagnostics`, `/api/errors`, `uiai_errors` |
| Focusa/Pi handoff | Scoped evidence packets give Focusa bounded refs instead of transcript blobs | `/api/agent/research-packet`, `uiai_focusa_packet_build`, `uiai_focusa_packet_compose` |
| Pi/MCP/CLI parity | Agents can use the same workflows through Pi, MCP, HTTP, or shell | `pi_uiai_*`, MCP `tools/list`, `scripts/uiai` |
| Visual proof | Screenshots, frames, shares, and reliability smokes support QA and demos | `/api/screenshot`, `/api/media/frame/*`, `uiai_frame_render` |

Legacy/backward-compatibility and old API retirement notes belong in maintenance docs, not above these benefits.

## Primary discovery surfaces

| Surface | Entry point | What agents get | Use when |
|---|---|---|---|
| HTTP agent card | `GET /api/tools/agent-card` | Compact bootstrap card, workflows, discovery links, search hints, reliability rules | First contact with a running engine |
| HTTP tool search | `GET /api/tools/search?q=<keyword>` | Low-context matching tool names/descriptions | Agent knows intent but not tool name |
| HTTP tool graph | `GET /api/tools/graph` | Workflow routes, related tools, Focusa evidence handoff metadata | Agent needs next-step routing |
| HTTP docs card | `GET /api/tools/docs` | Lightweight public docs/examples metadata | Agent needs docs links without loading all Markdown |
| HTTP full schemas | `GET /api/tools/openai`, `GET /api/tools/mcp`, `GET /api/tools` | Full OpenAI/MCP tool definitions | Tool runner needs schemas |
| Pi card tools | `pi_uiai_agent_card`, `pi_uiai_tool_search`, `pi_uiai_tool_graph` | Pi-native wrappers for card/search/graph | Pi session orientation |
| Pi command card | `/uiai` menu → “Show agent card” | Interactive Pi card/menu with hide/off support | Human/Pi operator UX |
| MCP discovery | `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph`, `tools/list` | MCP-native card/search/graph plus normalized tool list | MCP client orientation |
| CLI | `scripts/uiai tools agent-card`, `scripts/uiai tools search <q>`, `scripts/uiai tools graph`, `scripts/uiai tools docs` | Shell-friendly discovery and examples | Operator/headless usage |
| Public docs | `README.md` documentation map; this file; `docs/PUBLIC_API_PARITY_MATRIX.md` | Durable repo docs and parity status | Offline review, GitHub readers, release proof |

## Are we using Pi cards?

Yes. UIAI exposes a compact agent card at `/api/tools/agent-card`, mirrors it as Pi tool `pi_uiai_agent_card`, and exposes it in the Pi `/uiai` menu as “Show agent card.” The Pi extension also registers `pi_uiai_tool_search` and `pi_uiai_tool_graph` so agents can orient without loading full schemas.

## How new features become discoverable

A feature is considered agent-discoverable when the appropriate rows are complete:

1. **HTTP route** mounted in `internal/server/server.go`.
2. **Auth mode** documented in `docs/ENDPOINT_AUTH_MATRIX.md`.
3. **Tool definition** added to `internal/routes/tools.go` when it is a public agent surface.
4. **Agent card/search/graph/docs** updated when the feature changes workflow routing.
5. **Pi exposure** added in `.pi/extensions/uiai-engine.ts` or an explicit omission documented.
6. **MCP exposure** added in `mcp/browser-session-mcp.mjs` or an explicit omission documented.
7. **CLI exposure** added in `scripts/uiai` when useful for operators/headless agents.
8. **Parity row** added in `docs/PUBLIC_API_PARITY_MATRIX.md`.
9. **Public docs** linked from `README.md` and relevant task docs.
10. **Smokes/checkers** updated: `scripts/check-tool-parity.sh`, `scripts/check-docs-completeness.py`, and route-specific smokes.

## Current public agent feature index

| Feature | HTTP | Pi | MCP | CLI | Evidence handle | Status |
|---|---|---|---|---|---|---|
| Discovery/card/search/graph/docs | `/api/tools/*` | `pi_uiai_agent_card`, `pi_uiai_tool_search`, `pi_uiai_tool_graph` | `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph` | `scripts/uiai tools ...` | docs/tool graph refs | Full parity |
| Source-to-Markdown | `/api/markdown`, `/api/session/{id}/read format=markdown` | `uiai_source_to_markdown`, browser read markdown | `source_to_markdown`, `browser_read format=markdown` | `scripts/uiai markdown <url>` | `uiai-source-markdown:sha256:<prefix>` | Full MVP parity |
| Browser sessions/actions/read/snapshot/diagnostics | `/api/session/*` | `uiai_browser_*` | `browser_*` | core session/read/diagnostics/close | `uiai-browser:*`, `uiai-diagnostics:*` | Full core parity |
| Provider-neutral search | `/api/search` | `uiai_search` | `browser_search` | `scripts/uiai-open-result.sh` | `uiai-search:<provider>:<query-hash>:<rank>` | Full parity |
| Focusa research packet | `/api/agent/research-packet` | `uiai_focusa_packet_build` | `uiai_focusa_packet_compose` | `scripts/uiai research packet`, `scripts/uiai packet compose` | packet `evidence_refs[]` | Full parity |
| Errors/diagnostics | `/api/errors`, session diagnostics | `uiai_errors`, `uiai_browser_diagnostics` | `uiai_errors`, `browser_diagnostics` | `scripts/uiai errors` | `uiai-error:*`, `uiai-diagnostics:*` | Full parity |
| Screenshots/frame rendering | `/api/screenshot`, `/api/media/frame/*` | `uiai_screenshot`, `uiai_frame_catalog`, `uiai_frame_render` | `screenshot`, `frame_catalog`, `frame_render` | HTTP examples | `uiai-screenshot:*` | HTTP + Pi/MCP |
| Critique metadata | `/api/critique/models`, `/api/critique/dimensions` | metadata tools | metadata tools | HTTP examples | metadata JSON | HTTP + Pi/MCP |

See `docs/PUBLIC_API_PARITY_MATRIX.md` for full status and intentionally omitted plugin/admin/service surfaces.

## Public docs emphasis

Lead public docs with the highest-value agent/product features:

1. Agent discovery cards/search/graph/docs endpoint.
2. Source-to-Markdown / Web Memory Capture with Focusa evidence and WPUIAI research cards.
3. Persistent browser sessions with @refs, reads, snapshots, and diagnostics.
4. Search-to-browse and diagnostics-first workflows.
5. Pi/MCP/CLI parity and redacted Focusa packet handoff.

Deemphasize legacy/backward-compatibility topics. Keep them as maintenance references, not top-level product positioning.

## Known gaps worth filling

| Gap | Why it matters | Suggested next gate |
|---|---|---|
| WPUIAI plugin UI/save for `wpuiai.research_card` | Engine returns product cards and engine-side save/display contract is documented; plugin-side implementation remains follow-up | See `docs/WPUIAI_RESEARCH_CARD_INTEGRATION_CONTRACT.md` and create plugin repo tasks before editing the live plugin |
| Public website/docs deployment freshness is unknown | GitHub docs are updated, but live marketing/docs pages may lag | Verify any public docs site and link this index if one exists |
| Paid/mutating AI/admin/intelligence routes are intentionally omitted from generic agents | Prevents surprise credit spend and unsafe mutations | Add read-only summaries first, then cost/auth/rollback proof before tools |
| CLI remains lighter than Pi/MCP for full browser actions | CLI is adequate for core loops, but not every browser action | Add only operator-useful commands; avoid bloating CLI with every Pi/MCP action |

## Verification checklist

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-source-markdown-e2e.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
```

## Secret safety

Discovery outputs must use placeholders only. Never expose literal `Authorization`, cookies, API keys, bearer tokens, webhook secrets, private query values, or URL fragments in cards, docs examples, packet previews, diagnostics, or evidence refs.
