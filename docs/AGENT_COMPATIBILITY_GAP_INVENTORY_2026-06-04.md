# Agent Compatibility Gap Inventory — 2026-06-04

Purpose: track remaining work under the operator HLT: improve UIAI Engine as an agent-compatible browser/intelligence platform for Pi, MCP, Focusa, portability, security, performance, stability, and public documentation.

## Current verified slice

- WordPress plugin route parity inventory added: `docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md` maps existing plugin callers to Go routes/auth/error contracts.
- Endpoint auth matrix added: `docs/ENDPOINT_AUTH_MATRIX.md` maps route families to public, loopback-public remote-auth, authenticated, service-token, and handler-auth modes with update rules.
- Browser class/action tuning: `eval_failed` now has specific browser_diagnostics console/exception guidance, validated by the regression smoke.
- Browser error regression smoke added: `scripts/smoke-browser-error-regressions.sh` covers selector_not_found, timeout, eval_failed, stale-session not_found event, and url_not_allowed.
- MCP structured failure smoke added: `scripts/smoke-mcp-structured-failure.sh` verifies MCP error text includes id/class/Next diagnostics guidance.
- Pi rendering smoke added: `scripts/smoke-pi-rendering.sh` proves compact summaries plus expanded JSON behavior for success/error representative results.
- Pi registration smoke added: `scripts/smoke-pi-extension-registration.sh` verifies required Pi tool registrations, `/uiai` command, compact wrapper, and MCP mirrors.
- MCP route parity smoke added: `scripts/smoke-mcp-tool-routes.sh` verifies all advertised MCP tools have bridge call routes and caught/fixed `browser_eval_async`.
- CLI wrapper slice added: `scripts/uiai` supports status, health, errors, tools, session open/read/diagnostics/close, smoke, install, output modes, and stable exit code conventions.
- Interoperability quality spec added: `docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md` defines cross-surface contracts, acceptance gates, and proof requirements.
- README now links the interoperability quality spec, endpoint auth matrix, and WordPress plugin route parity matrix from the task index and matrix section.
- Agent-surface release proof checklist added: `docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md` lists HTTP/Pi/MCP/CLI/browser/search/auth/Focusa/doc gates and evidence handles.
- Provider-neutral search route exists: `GET/POST /api/search`.
- Brave provider is implemented behind the generic search contract.
- Search provider metadata exists: `GET /api/search/providers`.
- Search timeout/quota behavior documented: Brave provider calls use a bounded 12s timeout; upstream quota/rate limits stay provider-account concerns; UIAI reports configured/degraded readiness without exposing secrets.
- Pi extension exposes `uiai_search` and `/uiai off` / menu **Hide UIAI card**.
- MCP bridge exposes and fallback-normalizes `browser_search`.
- MCP metadata cache/reconnect behavior documented: restart/reconnect MCP clients and reload Pi MCP-adapter sessions after tool/schema/call-route changes.
- Focusa evidence handles documented for diagnostics, errors, search results, browser read/snapshot, screenshot, and share artifacts.
- Search results now include deterministic `rank` and `evidence_ref` fields shaped as `uiai-search:<provider>:<query-hash>:<rank>`; tool graph Focusa metadata advertises the same handle.
- Screenshot and share artifact evidence now echoes request `focusa_scope`, extending scope propagation beyond browser-session diagnostics.
- Tool discovery and graph advertise `browser_search` and `search_then_browse`.
- Search is loopback-public and remote-authenticated, aligned with browser/session/screenshot tool boundaries.
- Remote positive auth proof covers search, errors, media frame, session, and screenshot loopback-public route families with `X-API-Key` and Bearer local-token credentials.
- Public docs mention search flow, Pi/MCP surfaces, and auth boundaries.

## Proof handles

Recent pushed proof commits:

- `68f64c6` — README interoperability matrix links.
- `f84c638` — remote-auth positive coverage for loopback-public route families.
- `aec92b7` — search missing-key degraded-mode provider test.
- `8ef5ee2` — failed-network diagnostics smoke.
- `6068667` — Focusa `uiai-error:*` evidence smoke.
- `fdbcb81` — stable Focusa evidence-handle docs.
- `3835866` — MCP metadata cache/reconnect docs.
- `fc21f2f` — `/uiai off` verification smoke.
- `bf9afeb` — WP auth/retry policy docs.
- `b2734af` — diagnostics redaction hardening.
- `106edf4` — WordPress plugin route parity matrix.

Current proof commands:

- `go test ./...` passed.
- `node --check mcp/browser-session-mcp.mjs` passed.
- `uiai-engine.service` active after rebuild/restart; public deploy unit template now mirrors the live secret-safe env-file/resource-limit pattern without literal secrets.
- Loopback no-auth search smoke: `POST /api/search` returned `count=2`, `provider=brave`.
- Provider metadata smoke: `/api/search/providers` returns `configured`, `status`, and `degraded_reason`; missing Brave key is covered by a degraded-mode unit test.
- MCP JSON-RPC smoke: `tools/list` includes `browser_search`; `tools/call browser_search` returned `provider=brave`.
- MCP core smoke: `tools/call uiai_health` and `tools/call uiai_status` returned valid health/status payloads.
- MCP critique metadata smoke: `tools/call critique_models` and `tools/call critique_dimensions` returned valid read-only metadata payloads.
- MCP media-frame smoke: `tools/call frame_catalog` returned catalog metadata and `tools/call frame_render` rendered a valid PNG frame payload.
- Engine error tracking smoke: `/api/errors?limit=1` returned bounded redacted event envelope; MCP/Pi advertise `uiai_errors`; structured errors include id/class/message/next-action diagnostics links.
- Focusa handoff smoke: UIAI session opened with `focusa_scope`; diagnostics echoed scope; `focusa_browser_diagnostics_intake` completed with evidence `uiai-diagnostics:health-focusa-scope-smoke`.
- Ownership check: no root-owned files under `/home/wpuiai/uiai-engine`.

## Remaining gaps by waypoint

### 1. Pi plugin support

- Completed: `scripts/smoke-pi-uiai-off.sh` verifies `/uiai off|hide|clear|disable` clears the widget before engine fetch in freshly loaded extension source.
- Completed: `scripts/smoke-pi-extension-registration.sh` statically verifies Pi tool registrations, `/uiai` command registration, compact wrapper, and MCP mirrors.
- Deferred to `uiai-engine-dfm`: evaluate persistent user setting for showing/hiding the UIAI widget by default, instead of command-only clearing.

### 2. MCP access to core features

- Completed: MCP bridge exposes browser/session/search plus non-browser `uiai_health` and `uiai_status` core tools.
- Completed: MCP smoke coverage verifies `uiai_health` and `uiai_status` calls.
- Completed: read-only critique metadata tools are exposed through MCP/Pi (`critique_models`, `critique_dimensions`; Pi `uiai_critique_*`).
- Completed: MCP bridge now routes existing frame helpers (`frame_catalog`, `frame_render`) with loopback-public/remote-auth parity.
- Completed: deeper non-browser API expose/omit rationale documented in `docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md` for reference, admin/usage, memory, workflow, intelligence, training, captcha, and related route families.
- Completed: client reconnect requirements are documented for cached MCP metadata and loaded bridge code.

### 3. Agent web surfing

- Search → open → read → snapshot → diagnostics workflow is now advertised and smoke-tested.
- Deferred to `uiai-engine-1if`: add optional `open_result` helper only if repeated agent workflows show the separate search/open/read sequence is too verbose.
- Deferred to `uiai-engine-r0d`: evaluate search result caching/rate limiting to reduce provider cost and protect Brave quota beyond the documented provider-account quota boundary.

### 4. Focusa integration

- Completed: scoped UIAI browser diagnostic session with `focusa_scope` ingested via `focusa_browser_diagnostics_intake`.
- Completed: successful search/browser/release proof linked to canonical Focusa evidence handles and Workpoint evidence after live deploy.
- Completed: API responses and docs specify `uiai-search:<provider>:<query-hash>:<rank>` handles so search evidence can cite selected result URL/title/snippet without raw SERP blobs.
- Completed: screenshot/share evidence packets echo `focusa_scope` when provided.

### 5. Portability

- Completed: installer/smoke/docs mention `BRAVE_SEARCH_API_KEY`, `UIAI_ENGINE_URL`, `UIAI_API_KEY`, `UIAI_BEARER_TOKEN`, and MCP config location; auth values stay redacted.
- Completed: degraded-mode check shows `/api/search/providers` reports `configured=false`, `status=degraded`, and `degraded_reason=missing_key` when Brave is missing.
- Keep provider-specific secrets out of repo and public docs.

### 6. Security, performance, and stability

- Completed: remote-auth positive test covers `/api/search*`, `/api/errors*`, `/api/media/frame*`, `/api/session*`, and `/api/screenshot*` with API key and Bearer credentials.
- Completed: provider timeout/quota behavior is documented; future work remains optional caching/rate-limit controls beyond the current 12s HTTP timeout.
- Deferred to `uiai-engine-jh1`: plan bounded response truncation/redaction if future providers return richer metadata.
- Completed: live release browser soak reviewed repeated browser workflow behavior (`/tmp/uiai-browser-flakiness-soak-5m.json`, 96/96 passed over 300s).

### 7. Public and related documentation

- README and Session API are updated for search, `/uiai off`, Focusa evidence handles, MCP reconnect behavior, auth matrices, deployment env-file redaction, non-browser exposure inventory, and interoperability matrix discoverability.
- Add a dedicated search API section if the route gains more providers or parameters.
- Update MCP/Pi install smoke docs after adding broader non-browser MCP tools.

## Recommended next slice

Next stability slice: review remaining P1 epics for closeout criteria, then choose the next concrete bead that adds proof rather than only inventory.
