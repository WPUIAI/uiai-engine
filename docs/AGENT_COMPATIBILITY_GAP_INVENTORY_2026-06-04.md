# Agent Compatibility Gap Inventory — 2026-06-04

Purpose: track remaining work under the operator HLT: improve UIAI Engine as an agent-compatible browser/intelligence platform for Pi, MCP, Focusa, portability, security, performance, stability, and public documentation.

## Current verified slice

- Provider-neutral search route exists: `GET/POST /api/search`.
- Brave provider is implemented behind the generic search contract.
- Search provider metadata exists: `GET /api/search/providers`.
- Pi extension exposes `uiai_search` and `/uiai off` / menu **Hide UIAI card**.
- MCP bridge exposes and fallback-normalizes `browser_search`.
- Tool discovery and graph advertise `browser_search` and `search_then_browse`.
- Search is loopback-public and remote-authenticated, aligned with browser/session/screenshot tool boundaries.
- Public docs mention search flow, Pi/MCP surfaces, and auth boundaries.

## Proof handles

- `go test ./...` passed.
- `node --check mcp/browser-session-mcp.mjs` passed.
- `uiai-engine.service` active after rebuild/restart.
- Loopback no-auth search smoke: `POST /api/search` returned `count=2`, `provider=brave`.
- Provider metadata smoke: `/api/search/providers` returned `configured=true`.
- MCP JSON-RPC smoke: `tools/list` includes `browser_search`; `tools/call browser_search` returned `provider=brave`.
- MCP core smoke: `tools/call uiai_health` and `tools/call uiai_status` returned valid health/status payloads.
- MCP critique metadata smoke: `tools/call critique_models` and `tools/call critique_dimensions` returned valid read-only metadata payloads.
- MCP media-frame smoke: `tools/call frame_catalog` returned catalog metadata and `tools/call frame_render` rendered a valid PNG frame payload.
- Focusa handoff smoke: UIAI session opened with `focusa_scope`; diagnostics echoed scope; `focusa_browser_diagnostics_intake` completed with evidence `uiai-diagnostics:health-focusa-scope-smoke`.
- Ownership check: no root-owned files under `/home/wpuiai/uiai-engine`.

## Remaining gaps by waypoint

### 1. Pi plugin support

- Verify updated `/uiai off` behavior in a freshly reloaded Pi session; current running sessions may retain old extension code.
- Add a small static/runtime test for `.pi/extensions/uiai-engine.ts` command registration if the project adopts a Pi extension test harness.
- Consider a persistent user setting for showing/hiding the UIAI widget by default, instead of command-only clearing.

### 2. MCP access to core features

- Completed: MCP bridge exposes browser/session/search plus non-browser `uiai_health` and `uiai_status` core tools.
- Completed: MCP smoke coverage verifies `uiai_health` and `uiai_status` calls.
- Completed: read-only critique metadata tools are exposed through MCP/Pi (`critique_models`, `critique_dimensions`; Pi `uiai_critique_*`).
- Completed: MCP bridge now routes existing frame helpers (`frame_catalog`, `frame_render`) with loopback-public/remote-auth parity.
- Remaining: inventory deeper non-browser APIs agents may need next: reference analysis and usage-safe admin reads.
- Remaining: document client reconnect requirements where MCP metadata is cached.

### 3. Agent web surfing

- Search → open → read → snapshot → diagnostics workflow is now advertised and smoke-tested.
- Add optional `open_result` helper only if repeated agent workflows show the separate search/open/read sequence is too verbose.
- Consider search result caching/rate limiting to reduce provider cost and protect Brave quota.

### 4. Focusa integration

- Completed: scoped UIAI browser diagnostic session with `focusa_scope` ingested via `focusa_browser_diagnostics_intake`.
- Remaining: link successful search/browser proof to a canonical Focusa Workpoint when evidence hot routes are stable.
- Remaining: add docs for using UIAI search results as Focusa evidence handles without pasting raw SERP blobs.

### 5. Portability

- Ensure installer/smoke scripts mention `BRAVE_SEARCH_API_KEY`, `UIAI_ENGINE_URL`, `UIAI_API_KEY`, `UIAI_BEARER_TOKEN`, and MCP config location.
- Add a degraded-mode check showing `/api/search/providers` reports `configured=false` when Brave is missing.
- Keep provider-specific secrets out of repo and public docs.

### 6. Security, performance, and stability

- Add a remote-auth negative/positive test for `/api/search*`, parallel to browser/session boundaries.
- Consider provider request timeout/rate-limit controls beyond the current 12s HTTP timeout.
- Consider bounded response truncation/redaction if future providers return richer metadata.
- Review service memory/CPU behavior under repeated search + browser workflows.

### 7. Public and related documentation

- README and Session API are updated for search and `/uiai off`.
- Add a dedicated search API section if the route gains more providers or parameters.
- Update MCP/Pi install smoke docs after adding broader non-browser MCP tools.

## Recommended next slice

Next agent-core expansion: evaluate reference-analysis and usage-safe admin helpers; avoid paid or mutating tools unless a clear agent workflow and smoke proof exist.
