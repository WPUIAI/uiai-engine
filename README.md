# UIAI Engine

UIAI Engine is the Go backend for WPUIAI cloud features, visual/browser automation, persistent agent browsing, AI-powered design analysis, workflow orchestration, usage/credit accounting, and agent tool discovery. It is the successor/rewrite path for the older `WPUIAI/ai-api` Bun/PHP/vision-daemon stack, with expanded Go-native browser sessions, MCP/Pi integrations, [Focusa](https://github.com/Startempire-Wire/focusa) evidence handoff, captcha support, media jobs, training/data APIs, and operational tooling.

## What this engine does

UIAI Engine serves as a single local or remote API surface for:

- **WPUIAI plugin cloud calls:** critique, UI reverse/reference analysis, section detection, layout comparison, style enhancement, copilot chat, intake, workflow orchestration, and usage reporting.
- **Visual automation:** one-shot screenshots, persistent browser sessions, DOM snapshots, page text extraction, click/type/fill/select/press actions, CSS injection, viewport changes, cookies/auth save-load, diagnostics, and shareable artifacts.
- **Agent integrations:** Pi extension tools, MCP browser-session bridge, OpenAI/MCP tool schemas, compact agent cards, provider-neutral web search, tool search, tool graph metadata, and [Focusa](https://github.com/Startempire-Wire/focusa)-aware evidence routes.
- **Design/build pipelines:** design-system extraction, content mapping, block recipes, five-way comparison, migration helpers, and SSE events.
- **Media and device output:** device-frame catalog/rendering, media job production, screenshot compare, share viewers, and artifact handles.
- **Reliability and safety:** browser pool metrics, diagnostics without screenshots, URL allow/deny rules, loopback-public browser APIs, authenticated remote exposure, rate limits, credit deduction, and secret-safe deployment conventions.
- **Training/intelligence surfaces:** indexing, search/embed helpers, dataset/job/eval/model-registry endpoints, and code artifact upload/download routes.

## Relationship to `WPUIAI/ai-api`

`uiai-engine` is not a direct in-place continuation of the older `ai-api` codebase. It is a Go rewrite/successor that preserves many product routes while adding larger browser/session and agent-integration capabilities.

Shared/successor route families include `/api/critique`, `/api/ui-reverse`, `/api/section-detect`, `/api/layout-compare`, `/api/style-enhance`, `/api/copilot`, `/api/intake`, `/api/workflow`, `/api/usage`, `/api/extension`, `/api/memory`, `/api/admin`, `/api/intelligence`, `/api/training`, `/api/screenshot`, and `/api/share`.

Parity is intentionally documented instead of assumed:

- Full retirement/readiness inventory: [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)
- UI Reverse parity gaps: [`docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`](docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md)
- Workflow/cloud orchestration map: [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md)

Do not remove old API dependency paths solely because a Go route exists. The parity inventory requires caller compatibility, behavior parity, auth parity, response-shape parity, fallback/repair parity, and proof before retirement.

## Runtime architecture

Core runtime:

- Language/runtime: Go.
- HTTP router: Chi.
- Browser automation: Rod-backed shared vision pool.
- AI providers: Anthropic, OpenAI, OpenRouter, Fireworks, Kimi, Minimax provider configuration.
- Storage: local JSON/state files under configured `storage.data_dir` plus usage/job stores.
- Deployment: local binary, systemd service files under [`deploy/`](deploy/), and GitHub Actions browser reliability workflow.
- Agent bridge: project Pi extension and Node MCP bridge.

Main entry points:

- Go server: [`cmd/uiai-engine/main.go`](cmd/uiai-engine/main.go)
- Route mounting: [`internal/server/server.go`](internal/server/server.go)
- Config model: [`internal/config/config.go`](internal/config/config.go)
- Default config: [`config.yaml`](config.yaml)
- MCP bridge: [`mcp/browser-session-mcp.mjs`](mcp/browser-session-mcp.mjs)
- Pi extension: [`.pi/extensions/uiai-engine.ts`](.pi/extensions/uiai-engine.ts)
- Search flow: call `uiai_search`/`browser_search`, open a selected result with `browser_open`, then use `browser_read` and diagnostics as needed.

## Documentation map

Start here by task:

| Need | Read |
|---|---|
| HLT ledger and current trajectory | [`docs/HLT_LEDGER.md`](docs/HLT_LEDGER.md) |
| Browser/session API, agent tools, MCP formats, security boundaries, portability helpers | [`docs/SESSION_API.md`](docs/SESSION_API.md) |
| Diagnostics contract: console/errors/network/failed requests/[Focusa](https://github.com/Startempire-Wire/focusa) evidence refs | [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md) |
| Browser reliability gates, soak/stress commands, CI expectations | [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md) |
| Captcha solver architecture, proxy/IP pool, preprocessing, status APIs | [`docs/CAPTCHA_SOLVER_SPEC.md`](docs/CAPTCHA_SOLVER_SPEC.md) |
| Device frame catalog/rendering and media integration | [`docs/DEVICE_FRAME_INTEGRATION.md`](docs/DEVICE_FRAME_INTEGRATION.md) |
| Go-vs-old-API parity/retirement matrix | [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md) |
| UI Reverse/reference analyzer gaps | [`docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`](docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md) |
| WPUIAI workflow/cloud orchestration and route/caller mapping | [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md) |

Related implementation anchors:

| Feature area | Code |
|---|---|
| Server mount/root/status/auth boundaries | [`internal/server/server.go`](internal/server/server.go), [`internal/auth/auth.go`](internal/auth/auth.go) |
| Critique | [`internal/routes/critique.go`](internal/routes/critique.go) |
| UI reverse/section/layout/style/copilot | [`internal/routes/ai_routes.go`](internal/routes/ai_routes.go) |
| Reference analysis | [`internal/routes/reference.go`](internal/routes/reference.go), [`internal/reference/`](internal/reference/) |
| Browser sessions | [`internal/routes/session.go`](internal/routes/session.go), [`internal/vision/session.go`](internal/vision/session.go), [`internal/vision/session_actions.go`](internal/vision/session_actions.go) |
| Browser pool/screenshot/vision | [`internal/vision/pool.go`](internal/vision/pool.go), [`internal/routes/screenshot.go`](internal/routes/screenshot.go), [`internal/routes/vision_interactive.go`](internal/routes/vision_interactive.go) |
| Diagnostics/snapshots | [`internal/vision/diagnostics.go`](internal/vision/diagnostics.go), [`internal/vision/snapshot.go`](internal/vision/snapshot.go) |
| Tool discovery | [`internal/routes/tools.go`](internal/routes/tools.go), [`mcp/browser-session-mcp.mjs`](mcp/browser-session-mcp.mjs), [`.pi/extensions/uiai-engine.ts`](.pi/extensions/uiai-engine.ts) |
| Captcha | [`internal/routes/captcha.go`](internal/routes/captcha.go), [`internal/captcha/`](internal/captcha/) |
| Media/device frames | [`internal/routes/media.go`](internal/routes/media.go), [`internal/routes/media_frame.go`](internal/routes/media_frame.go), [`internal/media/`](internal/media/) |
| Workflow/intake/migration | [`internal/routes/workflow.go`](internal/routes/workflow.go), [`internal/routes/intake.go`](internal/routes/intake.go), [`internal/routes/migration.go`](internal/routes/migration.go) |
| Intelligence/training | [`internal/intelligence/`](internal/intelligence/), [`internal/routes/training.go`](internal/routes/training.go) |
| Admin/usage/rate limits | [`internal/routes/admin.go`](internal/routes/admin.go), [`internal/routes/usage.go`](internal/routes/usage.go), [`internal/credits/`](internal/credits/), [`internal/ratelimit/`](internal/ratelimit/) |

## API surface overview

### Root, health, status, and metrics

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/` | Service metadata and route map. |
| `GET` | `/health` | Compatibility health check for older PHP/API monitors. |
| `GET` | `/api/health` | Health status. |
| `GET` | `/api/health/providers` | Provider health/config status. |
| `GET` | `/api/health/browser` | Browser pool health. |
| `GET` | `/api/metrics/browser` | Browser queue/latency/pool metrics. |
| `GET` | `/api/status` | Engine runtime status. |

See also [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md) for browser-health gates.

### AI and design analysis

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/critique` | UICrit-style design critique with provider/model selection, credit accounting, and structured scores. |
| `GET` | `/api/critique/models` | Supported critique models/providers. |
| `GET` | `/api/critique/dimensions` | Critique dimensions. |
| `POST` | `/api/ui-reverse` | Legacy-compatible UI reverse operation surface. |
| `GET` | `/api/ui-reverse/models` | UI reverse model metadata. |
| `GET` | `/api/ui-reverse/operations` | UI reverse operation metadata. |
| `POST` | `/api/section-detect` | Identify page sections from screenshots/content. |
| `POST` | `/api/layout-compare` | Compare layout screenshots and describe differences. |
| `POST` | `/api/style-enhance` | Generate style/CSS improvement suggestions. |
| `POST` | `/api/copilot/chat` | Copilot chat endpoint. |
| `GET` | `/api/copilot/health` | Copilot route health. |
| `POST` | `/api/reference/analyze` | Go reference analyzer path used by newer UI Reverse cloud flows. |

Cross-links: [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md), [`docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`](docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md), [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)

### Browser sessions for agents

Persistent sessions keep a browser page alive across calls. They are optimized for LLM/agent workflows where repeated screenshots/navigation would be expensive and flaky.

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/api/session` | List active sessions. |
| `POST` | `/api/session` | Open a session and navigate to a URL. |
| `GET` | `/api/session/{id}` | Session metadata. |
| `DELETE` | `/api/session/{id}` | Close session. |
| `POST` | `/api/session/{id}/screenshot` | Instant screenshot of current page. |
| `POST` | `/api/session/{id}/navigate` | Navigate existing session. |
| `POST` | `/api/session/{id}/scroll` | Scroll and screenshot. |
| `POST` | `/api/session/{id}/click` | Click CSS selector or `@ref`. |
| `POST` | `/api/session/{id}/hover` | Hover CSS selector or `@ref`. |
| `POST` | `/api/session/{id}/type` | Type into an element. |
| `POST` | `/api/session/{id}/fill` | Fill input value. |
| `POST` | `/api/session/{id}/select` | Select option. |
| `POST` | `/api/session/{id}/press` | Press keyboard key. |
| `POST` | `/api/session/{id}/eval` | Run short sync JavaScript. |
| `POST` | `/api/session/{id}/eval_async` | Run bounded async JavaScript. |
| `POST` | `/api/session/{id}/resize` | Change viewport. |
| `POST` | `/api/session/{id}/css` | Inject CSS. |
| `POST` | `/api/session/{id}/back` | Browser history back. |
| `POST` | `/api/session/{id}/forward` | Browser history forward. |
| `POST` | `/api/session/{id}/text` | Extract element text. |
| `POST` | `/api/session/{id}/read` | Compact page/region text extraction. |
| `POST` | `/api/session/{id}/cookies` | Get/set/clear cookies. |
| `POST` | `/api/session/{id}/auth/save` | Save auth/cookie state. |
| `POST` | `/api/session/{id}/auth/load` | Load auth/cookie state. |
| `GET`/`POST` | `/api/session/{id}/snapshot` | DOM snapshot with stable element refs. |
| `GET` | `/api/session/{id}/diagnostics` | Console/errors/exceptions/network summary without screenshot. |
| `POST` | `/api/session/{id}/diagnostics/clear` | Clear diagnostic buffers. |
| `GET` | `/api/session/{id}/dom` | DOM overview. |
| `POST` | `/api/session/{id}/wait` | Wait for selector/text. |
| `POST` | `/api/session/{id}/captcha/solve` | Session-scoped captcha solve. |

Cross-links: [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md), [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md), [`docs/CAPTCHA_SOLVER_SPEC.md`](docs/CAPTCHA_SOLVER_SPEC.md)

### One-shot screenshot, share, and vision interactive APIs

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/screenshot` | Transactional navigate/capture screenshot. |
| `GET` | `/api/screenshot/health` | Screenshot subsystem health. |
| `POST` | `/api/screenshot/compare` | Screenshot comparison. |
| `GET` | `/api/share` | Share metadata/list endpoint. |
| `POST` | `/api/share/create` | Create share artifact. |
| `POST` | `/api/share/multi` | Multi-viewport share artifact. |
| `GET` | `/api/share/{id}` | Fetch share metadata/artifact. |
| `DELETE` | `/api/share/{id}` | Delete share. |
| `POST` | `/api/share/{id}/screenshot` | Capture screenshot for share. |
| `GET` | `/v/{token}` | Public share viewer. |
| `GET` | `/vision/state` | Vision pool/page state. |
| `POST` | `/vision/capture` | Capture current page. |
| `GET`/`POST` | `/vision/look` | Navigate/look at URL or target. |
| `POST` | `/vision/inject` | Inject CSS. |
| `GET` | `/vision/el` | Element screenshot. |
| `GET`/`POST` | `/vision/multi` | Multi-viewport capture. |
| `GET` | `/vision/diff` | Visual diff. |
| `GET` | `/vision/viewport` | View/change viewport metadata. |
| `GET` | `/vision/analyze` | DOM/page analysis. |
| `POST` | `/vision/critique` | Capture and critique bundle. |
| `POST` | `/vision/regression` | Regression capture/check. |

Cross-links: [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md), [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)

### Agent tool discovery, MCP, Pi, and [Focusa](https://github.com/Startempire-Wire/focusa) handoff

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/api/tools` | Full OpenAI-style tool definitions. |
| `GET` | `/api/tools/openai` | OpenAI function-calling schema. |
| `GET` | `/api/tools/mcp` | MCP tool definitions. |
| `GET` | `/api/tools/agent-card` | Compact bootstrap card for agents. |
| `GET` | `/api/tools/graph` | Tool relationship graph, workflows, [Focusa](https://github.com/Startempire-Wire/focusa) integration metadata. |
| `GET` | `/api/tools/search?q=...` | Low-context search for relevant tools. |
| `GET`/`POST` | `/api/search` | Provider-neutral web search for browser agents; Brave is the default provider. |
| `GET` | `/api/search/providers` | Search provider metadata/configuration status. |

Agent surfaces:

- Pi extension: [`.pi/extensions/uiai-engine.ts`](.pi/extensions/uiai-engine.ts)
- MCP bridge: [`mcp/browser-session-mcp.mjs`](mcp/browser-session-mcp.mjs)
- MCP config example: [`mcp/mcp.json`](mcp/mcp.json)
- Installer: [`scripts/install-agent-integrations.sh`](scripts/install-agent-integrations.sh)
- Smoke test: [`scripts/smoke-agent-integrations.sh`](scripts/smoke-agent-integrations.sh)

[Focusa](https://github.com/Startempire-Wire/focusa) integration highlights:

- `browser_open` accepts `focusa_scope` metadata.
- Diagnostics/screenshot/share responses can include `focusa_evidence` handles.
- `/api/tools/graph` advertises related [Focusa](https://github.com/Startempire-Wire/focusa) handoff tools such as diagnostics intake and evidence capture.
- Browser diagnostics are intended to be linked through [Focusa](https://github.com/Startempire-Wire/focusa) rather than pasted as raw transcript blobs.

Cross-links: [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md), [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md)

### Workflow orchestration and intake

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/intake/analyze` | Analyze intake payload. |
| `POST` | `/api/intake/submit` | Submit intake data. |
| `GET` | `/api/intake/status/{id}` | Intake job status. |
| `POST` | `/api/workflow/sites` | Register/provision workflow site. |
| `GET` | `/api/workflow/sites` | List workflow sites. |
| `GET` | `/api/workflow/sites/{id}` | Site metadata. |
| `DELETE` | `/api/workflow/sites/{id}` | Remove site record. |
| `GET` | `/api/workflow/sites/{id}/status` | Site workflow status. |
| `POST` | `/api/workflow/sites/{id}/ping` | Ping/probe site. |
| `POST` | `/api/workflow/sites/{id}/workflow/create-run` | Create workflow run. |
| `POST` | `/api/workflow/sites/{id}/workflow/execute` | Execute workflow step/action. |
| `POST` | `/api/workflow/sites/{id}/workflow/{action}` | Start/run/complete/skip/set-active-run style actions. |
| `GET` | `/api/workflow/runs/{runId}` | Run status. |
| `GET` | `/api/workflow/runs` | List runs. |
| `GET` | `/api/workflow/templates` | List templates. |
| `GET` | `/api/workflow/templates/{id}` | Template details. |
| `POST` | `/api/workflow/execute` | Execute workflow operation. |
| `POST` | `/api/workflow/run` | Run workflow operation. |
| `GET` | `/api/workflow/status/{runId}` | Workflow run status. |
| `GET` | `/api/events` | SSE real-time event stream. |

Cross-link: [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md)

### Pipeline, comparison, and migration APIs

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/design-system` | Extract/generate design system data. |
| `POST` | `/api/content-map` | Generate content maps. |
| `POST` | `/api/block-recipes` | Generate block recipes. |
| `POST` | `/api/comparison` | Five-way comparison/pipeline analysis. |
| `POST` | `/api/migration/import/usage` | Import usage data. |
| `POST` | `/api/migration/import/memory` | Import memory data. |
| `POST` | `/api/migration/import/shares` | Import share data. |
| `GET` | `/api/migration/status` | Migration status. |

Implementation anchors: [`internal/routes/pipeline.go`](internal/routes/pipeline.go), [`internal/comparison/`](internal/comparison/), [`internal/routes/migration.go`](internal/routes/migration.go)

### Memory, extension auth, usage, admin

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/extension/token` | Issue extension token. |
| `GET` | `/api/extension/verify` | Verify extension token. |
| `DELETE` | `/api/extension/token` | Revoke extension token. |
| `GET` | `/api/extension/rate-limits` | Extension rate limits. |
| `GET` | `/api/memory/{userId}` | User memory. |
| `GET` | `/api/memory/{userId}/stats` | Memory stats. |
| `GET` | `/api/memory/{userId}/conversation` | Conversation history. |
| `POST` | `/api/memory/{userId}/conversation` | Append conversation message. |
| `DELETE` | `/api/memory/{userId}/conversation` | Clear conversation. |
| `GET` | `/api/memory/{userId}/preferences` | User preferences. |
| `PUT` | `/api/memory/{userId}/preferences` | Update preferences. |
| `GET` | `/api/memory/{userId}/context` | User context. |
| `PUT` | `/api/memory/{userId}/context` | Update context. |
| `DELETE` | `/api/memory/{userId}` / `/api/memory/{userId}/all` | Delete user memory. |
| `GET` | `/api/usage/critique` | Critique usage stats. |
| `GET` | `/api/usage/ui-reverse` | UI Reverse usage stats. |
| `GET` | `/api/usage/all` | Aggregate usage stats. |
| `GET` | `/api/admin/dashboard` | Admin dashboard JSON. |
| `GET` | `/api/admin/services` | Service state. |
| `GET` | `/api/admin/resources` | Resource stats. |
| `GET` | `/api/admin/usage*` | Usage summaries/aggregates/breakdowns. |
| `GET`/`POST`/`DELETE` | `/api/admin/keys*` | API key management. |
| `GET` | `/api/admin/tokens` | Token listing. |
| `GET` | `/api/admin/rate-limits*` | Rate-limit introspection. |
| `GET` | `/api/admin/config` | Redacted config summary. |
| `GET` | `/dashboard` | Dashboard HTML. |

Cross-links: [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md)

### Media and device frames

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/media/produce` | Create media job. |
| `GET` | `/api/media/status/{jobID}` | Media job status. |
| `GET` | `/api/media/jobs` | List media jobs. |
| `GET` | `/api/media/frame/catalog` | Device frame catalog. |
| `POST` | `/api/media/frame/render` | Render screenshot into selected device frame. |

Cross-link: [`docs/DEVICE_FRAME_INTEGRATION.md`](docs/DEVICE_FRAME_INTEGRATION.md)

### Captcha solver

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/captcha/solve-image` | Stateless image/text captcha solve. |
| `POST` | `/api/captcha/solve-proxied` | Proxy/browser-assisted captcha solve. |
| `GET` | `/api/captcha/status` | Solver status/capabilities. |
| `GET` | `/api/captcha/pool` | Proxy/IP pool status. |
| `POST` | `/api/captcha/pool/add` | Add IP/proxy to pool. |
| `POST` | `/api/captcha/pool/remove` | Remove IP/proxy from pool. |
| `POST` | `/api/session/{id}/captcha/solve` | Solve captcha in an existing browser session. |

Cross-link: [`docs/CAPTCHA_SOLVER_SPEC.md`](docs/CAPTCHA_SOLVER_SPEC.md)

### Intelligence and training

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/api/intelligence/health` | Intelligence subsystem health. |
| `POST` | `/api/intelligence/index/trigger` | Trigger index run. |
| `POST` | `/api/intelligence/index/upload` | Upload index artifact/content. |
| `GET` | `/api/intelligence/index/{runId}` | Index status. |
| `GET` | `/api/intelligence/documents/{runId}` | Indexed documents. |
| `POST` | `/api/intelligence/search` | Search indexed content. |
| `POST` | `/api/intelligence/embed` | Embedding helper. |
| `GET`/`POST` | `/api/intelligence/wasm/{runId}` | Serve/upload WASM artifact. |
| `GET`/`POST` | `/api/intelligence/js/{runId}` | Serve/upload JS artifact. |
| `POST`/`GET` | `/api/training/jobs` | Create/list training jobs. |
| `GET` | `/api/training/jobs/{id}` | Training job status. |
| `POST` | `/api/training/jobs/{id}/cancel` | Cancel training job. |
| `POST`/`GET` | `/api/training/evals` | Create/list evaluations. |
| `GET` | `/api/training/evals/{id}` | Evaluation status. |
| `GET`/`POST` | `/api/training/registry/models*` | Model registry read/write. |
| `POST` | `/api/training/registry/promote` | Promote model. |
| `GET` | `/api/training/registry/audit` | Registry audit. |
| `POST`/`GET` | `/api/training/datasets` | Create/list datasets. |
| `GET` | `/api/training/datasets/{id}` | Dataset metadata. |
| `GET` | `/api/training/datasets/{id}/shards` | Dataset shards. |
| `GET` | `/api/training/datasets/shards/{id}` | Shard metadata. |
| `POST` | `/api/training/datasets/signed-url` | Upload signed URL. |
| `POST` | `/api/training/datasets/confirm-upload` | Confirm dataset upload. |
| `POST` | `/api/training/teacher/label` | Teacher labeling endpoint. |
| `GET` | `/api/training/eval-config/{engineId}` / `/api/training/eval-configs` | Eval config metadata. |

Implementation anchors: [`internal/intelligence/`](internal/intelligence/), [`internal/routes/training.go`](internal/routes/training.go)


### Engine/browser error tracking

- `GET /api/errors?limit=20&source=&class=` returns bounded, redacted engine/browser error events.
- Browser/session failures and recovered panics return structured envelopes: `error_id`, `error_class`, `message`, `suggested_next_action`, `diagnostics`, and redacted `details`.
- Captures HTTP 4xx/5xx, recovered panics, and rich browser-session action failures with session diagnostics summaries.
- Query strings, fragments, auth headers, cookies, request bodies, and secret-like context keys are not stored.
- Pi/MCP error text surfaces the id/class/next action and points agents to `uiai_errors`; UIAI Pi tool results render compact by default and expand with `Ctrl+O` (`app.tools.expand`).

## Agent integration highlights

- Project-local Pi extension: [`.pi/extensions/uiai-engine.ts`](.pi/extensions/uiai-engine.ts) registers a full Pi-facing mirror of the MCP/browser tool surface: agent card/tool search/graph, provider web search, browser sessions/actions/diagnostics, one-shot screenshots, and frame catalog/render helpers. `/uiai off` clears the UIAI widget.
- MCP bridge: [`mcp/browser-session-mcp.mjs`](mcp/browser-session-mcp.mjs) exposes browser/session tools plus `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph`, `uiai_health`, `uiai_status`, `uiai_errors`, `critique_models`, `critique_dimensions`, `browser_search`, `browser_read`, `frame_catalog`, and `frame_render`; `tools/list` normalizes these core tools even when a running engine returns stale metadata.
- Agent web surfing: persistent sessions include `/api/session/{id}/read` / `browser_read` for bounded page text extraction plus @ref actions for navigation and forms.
- Diagnostics-first debugging: `browser_diagnostics` exposes console/errors/network/failed requests without forcing screenshots.
- [Focusa](https://github.com/Startempire-Wire/focusa) handoff: `browser_open` accepts `focusa_scope`; diagnostics and evidence flows preserve project/workpoint scope; `/api/tools/graph` exposes [Focusa](https://github.com/Startempire-Wire/focusa)-aware related-tool routes.
- Portability: set `UIAI_ENGINE_URL`, `UIAI_PI_TIMEOUT_MS`, or `UIAI_MCP_TIMEOUT_MS` for remote/tunnel deployments.
- Security: remote browser/session API callers must authenticate; loopback remains frictionless for local agents.

See [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md), and [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md).

## Quick setup for agents

```bash
# Optional preview
DRY_RUN=1 scripts/install-agent-integrations.sh

# Install Pi extension + MCP config
scripts/install-agent-integrations.sh

# Smoke-check discovery/graph/MCP bridge
scripts/smoke-agent-integrations.sh
```

Remote/tunnel setup:

```bash
export UIAI_ENGINE_URL="https://your-authenticated-engine.example"
export UIAI_API_KEY="..."              # optional; enables authenticated routes such as media/frame helpers
export UIAI_BEARER_TOKEN="..."         # optional alternative to UIAI_API_KEY for Pi/MCP calls
# Server-side only: set BRAVE_SEARCH_API_KEY in the uiai-engine service environment to enable /api/search.
export UIAI_PI_TIMEOUT_MS=30000
export UIAI_MCP_TIMEOUT_MS=60000
scripts/install-agent-integrations.sh
scripts/smoke-agent-integrations.sh
```

## Build, test, and run

```bash
# Run all Go tests
go test ./...

# Check MCP bridge syntax
node --check mcp/browser-session-mcp.mjs

# Build binary
go build ./cmd/uiai-engine

# Run with config
./uiai-engine -config config.yaml
```

Useful smoke/reliability scripts:

```bash
scripts/smoke-agent-integrations.sh
scripts/stress-browser-diagnostics.sh
scripts/soak-browser-flakiness.sh
scripts/sync-device-frames.sh
scripts/trim-runtime-logs.sh
```

Cross-links: [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md), [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/DEVICE_FRAME_INTEGRATION.md`](docs/DEVICE_FRAME_INTEGRATION.md)

## Configuration

Default config lives in [`config.yaml`](config.yaml). Important sections:

| Section | Purpose |
|---|---|
| `server` | Host, port, vision pool size, timeouts. |
| `wordpress` | WPUIAI site URL, REST namespace, webhook secret env reference, cache TTL. |
| `ai.providers` | Provider API base URLs and provider-specific metadata. |
| `vision` | Browser pool size, idle timeout, screenshot quality, share dir, private URL allowance. |
| `media` | Device/template rendering script path and job timeout. |
| `credits` | Cost table by operation. |
| `rate_limits` | Tier-based hourly/daily limits. |
| `captcha` | Solver/provider/proxy/stealth/stats settings. |
| `storage` | Data dir and usage file. |
| `logging` | Log level/file. |
| `cors` | Allowed origins/methods/headers. |

Security notes:

- Secrets should be referenced through environment variables, not committed literal values.
- `vision.allow_private_urls: true` is appropriate for local/dev; remote deployment should review URL safety rules in [`docs/SESSION_API.md`](docs/SESSION_API.md).
- Browser/session, screenshot, and provider search APIs are loopback-public only; remote callers must authenticate. The Pi extension can send `UIAI_API_KEY` or `UIAI_BEARER_TOKEN` for authenticated remote/media helpers. Local VPS deployments may configure an eternal env-backed `UIAI_LOCAL_API_TOKEN` accepted as `X-API-Key`, `X-License-Key`, or `Authorization: Bearer ...`.

## Security and exposure model

- `/api/tools*` discovery is intentionally public and low-context.
- `/api/session*`, `/api/screenshot*`, and `/api/search*` are unauthenticated only for loopback callers.
- Remote browser/session/screenshot callers require normal UIAI auth headers; Pi extension callers can set `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`. The local VPS eternal token is configured server-side with `UIAI_LOCAL_API_TOKEN` or comma-separated `UIAI_LOCAL_API_TOKENS`.
- Browser navigation accepts `http://` and `https://` only.
- `file://`, `data:`, `ftp://`, and similar schemes are blocked.
- Private/internal hosts are blocked unless `vision.allow_private_urls: true` is configured.
- Error responses classify blocked URLs as `url_not_allowed` and guide agents to policy-safe targets.

Cross-link: [`docs/SESSION_API.md#security--remote-exposure-boundaries`](docs/SESSION_API.md#security--remote-exposure-boundaries)

## Operational files

| Path | Purpose |
|---|---|
| [`deploy/uiai-engine.service`](deploy/uiai-engine.service) | systemd service unit. |
| [`deploy/uiai-log-trim.service`](deploy/uiai-log-trim.service) | log trimming service. |
| [`deploy/uiai-log-trim.timer`](deploy/uiai-log-trim.timer) | log trimming timer. |
| [`.github/workflows/browser-reliability.yml`](.github/workflows/browser-reliability.yml) | browser reliability CI. |
| [`scripts/trim-runtime-logs.sh`](scripts/trim-runtime-logs.sh) | runtime log cleanup. |
| [`scripts/stress-browser-diagnostics.sh`](scripts/stress-browser-diagnostics.sh) | diagnostics stress run. |
| [`scripts/soak-browser-flakiness.sh`](scripts/soak-browser-flakiness.sh) | browser flake soak. |

## Current parity/readiness warnings

The Go engine is more developed than the old AI API in many areas, but parity remains intentionally tracked:

- Screenshot/share/viewer: partial; not universally safe to retire old paths without proof.
- UI Reverse/reference analysis: implemented but not fully parity-certified.
- Section detect/layout compare/style enhance/copilot/intake/workflow/intelligence/training: route presence does not equal full caller/contract parity.

Before deleting or disabling old API dependency code, read:

1. [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)
2. [`docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`](docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md)
3. [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md)

## Recommended contribution workflow

```bash
git status --short --branch --untracked-files=all
go test ./...
node --check mcp/browser-session-mcp.mjs
go build ./cmd/uiai-engine
scripts/smoke-agent-integrations.sh
```

For browser-affecting changes, also run or consult:

```bash
scripts/stress-browser-diagnostics.sh
scripts/soak-browser-flakiness.sh
```

When changing API behavior, update the relevant doc linked above and add/adjust route tests under [`internal/routes/`](internal/routes/) or subsystem tests under [`internal/`](internal/).
