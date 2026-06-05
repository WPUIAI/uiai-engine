# UIAI Engine Interoperability Quality Spec

**Date:** 2026-06-04  
**Scope:** UIAI Engine interoperability quality across WordPress plugin, Go engine, Pi plugin, MCP bridge, Focusa, browser/session diagnostics, CLI, search providers, auth/security, and public docs.  
**HLT alignment:** Improve UIAI Engine as an agent-compatible browser/intelligence platform for Pi, MCP, Focusa, portability, security, performance, stability, and public documentation.

## 1. Purpose

UIAI Engine now has several agent/operator-facing surfaces that share the same underlying browser, intelligence, diagnostics, search, and media capabilities. Quality risk increases when these surfaces drift: an endpoint may work over HTTP but be missing from MCP, a Pi tool may hide useful error context, a WordPress UI may display an opaque failure, or CLI workflows may require hand-written curl commands.

This spec defines interoperability contracts, implementation tracks, acceptance gates, and proof requirements so each component can be improved without creating surface drift.

## 2. Non-goals

- No provider secrets in docs, logs, issue descriptions, diagnostics, or error events.
- No broad exposure of paid/mutating APIs until a concrete agent workflow, auth boundary, and smoke proof exist.
- No replacement of existing HTTP APIs; this spec standardizes parity and routing around current routes.
- No assumption that loopback-public routes are safe for remote callers; remote access must remain authenticated.

## 3. Components and primary surfaces

| Component | Primary surface | Current role | Main quality risk |
|---|---|---|---|
| Go engine HTTP API | `/api/*` | Canonical capability implementation. | Endpoint behavior drifts from agent/plugin wrappers. |
| WordPress plugin | PHP/UI/REST consumer | Product UI and customer workflow. | Opaque errors, auth mismatch, route parity gaps. |
| Pi plugin | `.pi/extensions/uiai-engine.ts` | Direct Pi tool surface and compact UI. | Missing tools, stale extension reload, poor rendering. |
| MCP bridge | `mcp/browser-session-mcp.mjs` | Stdio bridge for MCP clients. | Advertised tools missing call routes; metadata cache confusion. |
| Focusa | `focusa_scope`, diagnostics intake | Evidence/workpoint linkage. | Scope loss or raw blob evidence instead of stable handles. |
| Browser/session API | `/api/session/*` | Persistent browser automation. | Failures not actionable; diagnostics not linked to errors. |
| CLI/scripts | `scripts/*`, future `scripts/uiai` | Operator/agent shell workflows. | Fragmented scripts/curl; unstable output modes/exit codes. |
| Search providers | `/api/search*` | Provider-neutral discovery. | Provider config/rate/degraded behavior not uniform. |
| Auth/security | middleware + env | Loopback/remote boundary. | Endpoint groups accidentally public or unusable remotely. |
| Docs/smokes | README, `docs/*`, scripts | Onboarding and proof. | Docs claim support without smoke proof. |

## 4. Global interoperability contracts

### 4.1 Surface parity contract

Every capability exposed to agents should have an explicit parity record:

1. **HTTP route** exists and is documented.
2. **Pi tool** exists when Pi users need the capability.
3. **MCP tool** exists when MCP users need the capability.
4. **CLI command** exists when shell operators/agents need repeatable access.
5. **Smoke proof** verifies at least one happy path and one meaningful failure/degraded path where applicable.
6. **Docs** name auth requirements, env vars, output shape, and recovery path.

If a capability intentionally lacks Pi/MCP/CLI exposure, the spec, gap inventory, or [`AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`](AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md) must state why.

### 4.2 Error envelope contract

Agent-facing failures should prefer structured error envelopes:

```json
{
  "error": "human-readable short failure",
  "message": "same or clearer message",
  "error_id": "uiai-error-123",
  "error_class": "selector_not_found",
  "status": 500,
  "suggested_next_action": "Call snapshot or diagnostics, then retry with a current @ref.",
  "diagnostics": "/api/errors?limit=20&source=browser_session",
  "details": {}
}
```

Rules:

- `error_id` must map to `/api/errors` when an event is recorded.
- `error_class` must be stable enough for agents and tests.
- `suggested_next_action` must say what to do next, not just restate the error.
- `details` must be redacted and bounded.
- Pi/MCP/CLI should display id/class/next action by default and full JSON only on expansion/`--json`.

### 4.3 Auth boundary contract

Canonical endpoint auth mode table and update rules: [`ENDPOINT_AUTH_MATRIX.md`](ENDPOINT_AUTH_MATRIX.md).


Endpoint groups use one of these auth modes:

| Mode | Meaning | Examples |
|---|---|---|
| public | Safe public metadata/read. | `/health`, `/api/status`, `/api/tools/*`, critique metadata. |
| loopback-public remote-auth | Local agents can call without auth; remote callers authenticate. | browser/session, screenshot, search, media frame, errors. |
| authenticated | Always requires API/license/local token unless handler has its own auth. | paid/mutating AI operations, admin writes. |
| service-token | Internal service route with explicit service token. | training/service-style routes. |

Each endpoint group must have regression coverage for its boundary when feasible.

### 4.4 Redaction contract

Never store or print:

- Authorization headers.
- Cookies.
- API keys, bearer tokens, license keys, webhook secrets.
- Raw request bodies unless explicitly safe and bounded.
- URL query strings/fragments in stored error events.

### 4.5 Smoke proof contract

A feature is not considered interoperable until proof exists in at least one stable handle:

- `go test ./...`
- `node --check mcp/browser-session-mcp.mjs`
- `scripts/smoke-agent-integrations.sh`
- targeted MCP JSON-RPC call
- targeted curl/JQ check
- Focusa evidence capture when relevant

## 5. Interoperability matrix

| Track | HTTP | Pi | MCP | CLI | Focusa | Smoke/proof |
|---|---|---|---|---|---|---|
| Status/health | `/api/status`, `/api/health/browser` | `uiai_status`, `uiai_health` | `uiai_status`, `uiai_health` | `uiai status`, `uiai health` planned | advisory evidence | health/status smoke |
| Errors | `/api/errors` | `uiai_errors` | `uiai_errors` | `uiai errors` planned | error evidence candidate | forced harmless error + MCP proof |
| Browser sessions | `/api/session/*` | `uiai_browser_*` | `browser_*` | `uiai session *` planned | `focusa_scope` echo | browser open/read/diag smoke |
| Search | `/api/search*` | `uiai_search` | `browser_search` | `uiai search` optional/planned | `uiai-search:<provider>:<query-hash>:<rank>` result evidence refs | Brave provider + evidence-ref smoke |
| Tool discovery | `/api/tools/*` | `pi_uiai_tool_*` | `uiai_tool_*` | `uiai tools *` planned | graph includes Focusa routing | tool search/MCP smoke |
| Critique metadata | `/api/critique/models`, `/dimensions` | `uiai_critique_*` | `critique_*` | optional tools route | none | metadata smoke |
| Frame helpers | `/api/media/frame/*` | `uiai_frame_*` | `frame_*` | optional media route | screenshot evidence | catalog/render smoke |
| WordPress product flows | plugin REST/UI | indirect | indirect | limited | optional | parity tests needed |
| Reference analysis | `/api/reference/analyze` | intentionally omitted | intentionally omitted | intentionally omitted | optional | see [`AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`](AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md) |
| Admin usage reads | `/api/admin/*`, `/api/usage/*` | intentionally omitted | intentionally omitted | intentionally omitted | optional | see [`AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`](AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md) |

## 6. Implementation tracks

### Track A — WordPress plugin ↔ Go engine parity

Current route contract inventory: [`WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md`](WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md).


**Problem:** Product users experience UIAI through WordPress, but many recent improvements landed in Go/Pi/MCP. The plugin must surface the same contracts: structured errors, auth status, and route parity.

**Spec requirements:**

1. Inventory plugin calls to UIAI Engine routes.
2. For each call, record:
   - route
   - method
   - request schema
   - response schema
   - auth mechanism
   - expected error envelope
   - UI display behavior
3. Plugin UI should display at least:
   - concise message
   - `error_id` if present
   - `error_class` if present
   - suggested next action when user-actionable
4. Plugin logs may store `error_id` and class, not secrets.
5. Plugin retry behavior must be explicit per route: no blind retries for paid/mutating operations.

**Acceptance criteria:**

- A parity table exists for critique/reference/media/workflow/session-adjacent plugin calls.
- At least one plugin-facing flow shows structured engine errors with `error_id` in UI/log proof.
- Auth failures show a configuration/action message, not raw validation internals.

**Suggested beads:**

- `WP parity: inventory plugin route contracts`
- `WP parity: surface structured engine errors in plugin UI`
- `WP parity: align auth and retry behavior`

### Track B — Pi plugin interoperability

**Problem:** Pi is a high-value local agent surface. It must stay compact, complete, and reload-safe.

**Spec requirements:**

1. Pi tools mirror selected HTTP/MCP capabilities.
2. Result rendering:
   - compact by default
   - expands with `Ctrl+O`
   - errors show id/class/status/next action
   - full JSON available when expanded
3. Runtime/reload behavior documented:
   - project-local extension path
   - `/reload` required after extension edits in existing sessions
   - current sessions may retain old tool definitions
4. Pi commands:
   - `/uiai` workflow menu
   - `/uiai off` card clear
   - future CLI link hints as needed
5. Static/runtime test strategy should exist before large Pi tool additions.

**Acceptance criteria:**

- Pi tool list is generated or verified against desired surface.
- Compact render verified manually or by extension harness.
- Error formatting proof exists for a browser failure.
- `/uiai off` behavior verified in a freshly reloaded session.

**Suggested beads:**

- `Pi: add static/runtime extension registration test`
- `Pi: verify compact/expanded rendering across UIAI tools`
- `Pi: verify /uiai off in reloaded session`

### Track C — MCP bridge interoperability

**Problem:** MCP clients depend on `tools/list` and `tools/call`; stale metadata or missing call routes break agents.

**Spec requirements:**

1. Every MCP-advertised tool must have a call route in `toolsCall()`.
2. Bridge core tools may fallback-normalize stale engine metadata, but docs must explain client reconnect/cache behavior.
3. Error formatting must include message, id, class, status, next action, and diagnostics route when present.
4. MCP smoke must include:
   - `tools/list`
   - at least one call per tool family
   - at least one structured failure call
5. MCP schemas must stay within MCP/OpenAI-compatible JSON Schema subsets.

**Acceptance criteria:**

- Automated smoke detects advertised-but-unrouted tools.
- MCP bridge syntax check passes.
- Structured failure call proof exists.
- Reconnect/cache guidance is documented.

**Suggested beads:**

- `MCP: add advertised-tool call-route parity smoke`
- `MCP: document metadata cache and reconnect behavior`
- `MCP: add structured failure smoke`

### Track D — Focusa evidence interoperability

**Problem:** Focusa needs stable handles and scope; agents should not paste raw browser logs or SERP blobs into memory.

**Spec requirements:**

1. `browser_open` accepts `focusa_scope`.
2. Diagnostics and error envelopes echo scope when present.
3. Stable evidence refs should be generated/used:
   - `uiai-diagnostics:session=<id>:seq=<seq>`
   - `uiai-error:<error_id>`
   - `uiai-screenshot:sha256:<prefix>`
   - `uiai-search:<provider>:<query-hash>:<rank>` for search results
4. Docs should show Focusa workflow:
   - open with scope
   - read/snapshot
   - diagnostics on failure
   - `uiai_errors` for engine failure
   - `focusa_browser_diagnostics_intake` or `focusa_evidence_capture`
5. Evidence payloads must be bounded summaries, not raw blobs.

**Acceptance criteria:**

- Focusa handoff example includes error and diagnostics evidence.
- A smoke/test demonstrates `focusa_scope` through diagnostics/error envelope.
- Docs state how to cite UIAI evidence handles.

**Suggested beads:**

- `Focusa: document search/browser/error evidence handles`
- `Focusa: smoke uiai_error evidence intake path`
- `Focusa: propagate scope to additional relevant endpoints`

### Track E — Browser/session diagnostics and errors

**Problem:** Browser agents fail for stale selectors, JS exceptions, navigation failures, network errors, timeouts, and page crashes. All must be classified and recoverable.

**Spec requirements:**

1. Browser action failures classify into stable classes:
   - `selector_not_found`
   - `timeout`
   - `navigation_failed`
   - `page_unavailable`
   - `url_not_allowed`
   - `screenshot_failed`
   - `click_failed`
   - `eval_failed`
   - `action_failed`
2. Each class has a suggested next action.
3. Error envelope includes diagnostics summary and relevant bounded details.
4. Diagnostics endpoint captures:
   - console errors/warnings
   - JS exceptions
   - network failed requests
   - HTTP 4xx/5xx counts
5. Tests should cover common failures with stable output.

**Acceptance criteria:**

- Regression tests for missing selector, stale session, URL blocked, timeout/wait failure, JS eval failure.
- `/api/errors` stores each failure with redacted details.
- Pi/MCP/CLI show class and next action.

**Suggested beads:**

- `Browser errors: add common failure regression tests`
- `Browser errors: tune classes from real agent failures`
- `Browser diagnostics: add failed-network proof workflow`

### Track F — CLI ↔ HTTP API interoperability

**Problem:** Operators and shell agents currently use a mix of binary flags, scripts, and curl. This is fragmented.

**Spec requirements:**

1. Add lightweight `scripts/uiai` wrapper before full Go CLI.
2. Support commands:
   - `uiai status`
   - `uiai health`
   - `uiai errors [--limit N] [--source S] [--class C] [--json]`
   - `uiai tools [search <q>|mcp|openai|graph|agent-card]`
   - `uiai session open/read/snapshot/diagnostics/close`
   - `uiai smoke agent|browser|packet`
   - `uiai packet compose <json-file|->`
   - `uiai install-agent-integrations`
3. Output modes:
   - compact default
   - `--json`
   - `--pretty` where useful
4. Stable exit codes:
   - `0` success
   - `1` API/tool failure
   - `2` usage error
   - `3` dependency missing
   - `4` auth/config error
5. Env vars align with Pi/MCP:
   - `UIAI_ENGINE_URL`
   - `UIAI_API_KEY`
   - `UIAI_BEARER_TOKEN`
   - `UIAI_SMOKE_TIMEOUT_SECONDS`

**Acceptance criteria:**

- Existing CLI beads under `uiai-engine-dl0` are implemented or explicitly deferred.
- `scripts/uiai --help` documents commands.
- CLI smoke covers status, errors, tools search, and a browser session open/close.

**Existing beads:**

- `uiai-engine-dl0` — CLI epic
- `uiai-engine-dl0.1` — wrapper entrypoint
- `uiai-engine-dl0.2` — errors subcommand
- `uiai-engine-dl0.3` — browser session subcommands
- `uiai-engine-dl0.4` — output modes and exit codes
- `uiai-engine-dl0.5` — tool discovery subcommands
- `uiai-engine-dl0.6` — smoke/install wrappers
- `uiai-engine-dl0.7` — docs/completion

### Track G — Search/provider interoperability

**Problem:** Brave search is implemented, but future providers and degraded states need uniform behavior.

**Spec requirements:**

1. Provider-neutral contract:
   - query
   - provider
   - limit
   - results
   - provider metadata
   - configured/degraded status
2. `/api/search/providers` must report provider readiness without exposing secrets.
3. Missing provider key must return configured/degraded state consistently across HTTP/Pi/MCP/CLI.
4. Provider timeout/rate behavior must be bounded.
5. Search result evidence should use deterministic `uiai-search:<provider>:<query-hash>:<rank>` handles instead of raw SERP blobs when linked to Focusa.

**Acceptance criteria:**

- Degraded-mode smoke for missing Brave key.
- Provider timeout/rate settings documented.
- Search output truncation/redaction reviewed for future richer providers.

**Suggested beads:**

- `Search: add missing-key degraded-mode smoke`
- `Search: document provider timeout and quota behavior`
- `Search: design evidence handle for SERP results`

### Track H — Security/auth interoperability

**Problem:** Every new surface must preserve loopback convenience without remote exposure.

**Spec requirements:**

1. Maintain endpoint auth matrix.
2. Add auth boundary tests for every loopback-public route family:
   - session
   - screenshot
   - search
   - media frame
   - errors
   - future CLI-relevant groups
3. Remote authenticated callers should work with either:
   - `X-API-Key`
   - `Authorization: Bearer`
   - approved local token where configured
4. Docs must tell remote users how to set auth env vars.
5. Error/log systems must not leak auth data.

**Acceptance criteria:**

- Auth tests cover positive loopback and negative remote without credentials.
- Remote credential path is smoke-tested where safe.
- Docs include the endpoint auth mode table.

**Suggested beads:**

- `Security: maintain endpoint auth mode matrix`
- `Security: add remote-auth positive smoke`
- `Security: audit error/log redaction across route families`

### Track I — Docs, public proof, and release quality

**Problem:** Feature claims need proof handles and consistent docs.

**Spec requirements:**

1. Docs must include:
   - route contract
   - Pi/MCP/CLI surfaces
   - auth mode
   - error behavior
   - smoke proof
2. Gap inventory should be updated after each shipped slice.
3. README should remain high-level; detailed contracts live in docs.
4. Release proof should include commands run and commit IDs.

**Acceptance criteria:**

- Every interoperability slice updates docs and gap inventory.
- End-of-task report includes proof handles.
- Public docs do not expose secrets or environment-specific sensitive values.

**Suggested beads:**

- `Docs: add interoperability matrix to README links`
- `Docs: keep gap inventory current after each slice`
- `Docs: add release proof checklist for agent surfaces`

## 7. Recommended implementation order

1. **CLI wrapper epic (`uiai-engine-dl0`)** — closes immediate operator surface fragmentation.
2. **Browser error regression tests** — validates recent error-tracking work against real failure classes.
3. **MCP parity smoke** — prevents advertised-but-unrouted tool regressions.
4. **Focusa evidence docs/smoke** — turns UIAI diagnostics/errors into stable Workpoint evidence.
5. **WordPress plugin parity inventory** — aligns product UI with engine contracts.
6. **Search degraded-mode smoke** — hardens provider portability.
7. **Auth mode matrix** — keeps loopback/remote boundaries explicit as surfaces expand.

## 8. Implementation checklist template

Use this template for each new interoperability slice:

```markdown
## Slice: <name>

- [ ] HTTP contract documented
- [ ] Pi tool added or intentionally omitted
- [ ] MCP tool added or intentionally omitted
- [ ] CLI command added or intentionally omitted
- [ ] Auth mode recorded
- [ ] Structured error behavior verified
- [ ] Redaction reviewed
- [ ] Smoke proof added
- [ ] Gap inventory updated
- [ ] Focusa evidence path considered
- [ ] README/docs links updated
```

## 9. Acceptance for this spec

This spec is complete when:

- It covers all requested interoperability areas.
- It names concrete contracts, acceptance gates, and proof requirements.
- It maps the existing CLI bead epic and proposes follow-up beads for non-CLI tracks.
- It can be used by another agent to implement the next slice without re-deriving the overall plan.
