# UIAI Engine

UIAI Engine is the **proof browser for Focusa-powered agents** and the agent-compatible browser/intelligence backend for WPUIAI. It gives agents a safer browser surface, source-to-proof research, diagnostics-first debugging, visual checks, and clean evidence handoff to Pi, MCP clients, Focusa, WPUIAI, and remote agents. UIAI owns browser/search/session/media/diagnostics execution and stable evidence handles; Focusa owns cognitive continuity: ProjectIdentity, Workpoints, Trajectory, evidence linkage, predictions, metacog, and recovery.

## Clearest benefits

Start here. These are the main reasons to use UIAI Engine as the proof browser for Focusa-powered agents:

| Benefit | What you get |
|---|---|
| **Agents discover features fast** | Agent cards, tool search, tool graph, docs metadata, OpenAI/MCP schemas, Pi cards, MCP cards, and CLI discovery. |
| **Public sources become usable proof** | Source-to-Markdown converts URLs into clean Markdown, records, metadata, diagnostics, Focusa evidence refs, and WPUIAI research cards/reports. |
| **Agents can browse real sites reliably** | Persistent browser sessions keep state across reads, snapshots, @ref actions, forms, navigation, cookies, auth state, and diagnostics. |
| **Debugging starts from evidence** | Console errors, exceptions, failed requests, structured error envelopes, and `uiai_errors` reduce screenshot-only guessing. |
| **Research flows hand off cleanly** | Search → browse → read/snapshot → diagnostics → redacted Focusa/Pi packet → evidence capture. |
| **Focusa-powered agents get a proof browser** | UIAI sees and proves what happened in the browser; Focusa remembers the project goal, Workpoint, evidence, prediction, metacog lesson, and next action. |
| **Cohort access has guardrails** | Hardened auth, redaction, URL safety, public-target smokes, diagnostics, and release proof make browser access safer to share with real users. |
| **FPV co-pilot is the roadmap** | Planned PWA share links will let operators watch and steer live agent browser sessions from mobile before native app work. |
| **Visual QA has artifacts** | Screenshots, device frames, shares, browser reliability checks, and release proof scripts produce reviewable output. |
| **WPUIAI gets product-ready outputs** | Research cards, proof reports, critique/design/reference support, workflow support, and usage/accounting surfaces. |

Primary entry points: `/api/tools/agent-card`, `/api/tools/search`, `/api/tools/graph`, `/api/tools/docs`, `/api/markdown`, `/api/session/*`, `/api/agent/research-packet`, Pi `pi_uiai_agent_card`, Pi `uiai_source_to_markdown`, MCP `source_to_markdown`, and `scripts/uiai`.

Legacy parity, old route retirement, and compatibility inventories are maintenance references below the primary product/agent benefits.

## Public browser cohort positioning

Use UIAI Engine when Focusa-powered agents need a browser they can explain, prove, and hand off. The cohort value proposition is simple:

1. **See the web now:** persistent browser sessions, screenshots, snapshots, @ref actions, reads, forms, cookies, auth state, and diagnostics.
2. **Turn activity into proof:** useful reads, searches, diagnostics, screenshots, and packets get stable evidence handles instead of raw transcript blobs.
3. **Keep agents on track with Focusa:** Focusa binds browser evidence to project identity, Workpoints, trajectory, predictions, metacog lessons, and next actions.
4. **Share safely:** remote/browser access requires auth where appropriate, private/internal targets are blocked by default, and `url_not_allowed` is captured as policy evidence.
5. **Ship product output:** Source-to-Markdown and research packets produce WPUIAI research cards/reports plus Focusa-ready capture arguments.
6. **Plan live co-pilot oversight:** the FPV/PWA specs describe the next cohort-facing leap: a mobile share link where an operator can watch, annotate, and steer an agent browser session without installing an app.

For users, this means less “the agent clicked around somewhere” and more “the agent inspected these sources, captured this evidence, diagnosed these failures, and Focusa knows what to do next.” The current product already supports browser proof and Focusa handoff; the planned FPV/PWA layer adds live operator oversight on top of that foundation.

## What this engine does

UIAI Engine serves as a single local or remote API surface for:

- **Agent integrations:** Pi extension tools, MCP browser-session bridge, OpenAI/MCP tool schemas, compact agent cards, provider-neutral web search, tool search, tool graph metadata, and [Focusa](https://github.com/Startempire-Wire/focusa)-aware evidence routes that make UIAI the proof browser inside Focusa-powered workflows.
- **Visual automation:** one-shot screenshots, persistent browser sessions, DOM snapshots, page text extraction, click/type/fill/select/press actions, CSS injection, viewport changes, cookies/auth save-load, diagnostics, and shareable artifacts.
- **WPUIAI plugin cloud calls:** critique, UI reverse/reference analysis, section detection, layout comparison, style enhancement, copilot chat, intake, workflow orchestration, and usage reporting.
- **Design/build pipelines:** design-system extraction, content mapping, block recipes, five-way comparison, migration helpers, and SSE events.
- **Media and device output:** device-frame catalog/rendering, media job production, screenshot compare, share viewers, and artifact handles.
- **Reliability and safety:** browser pool metrics, diagnostics without screenshots, URL allow/deny rules, loopback-public browser APIs, authenticated remote exposure, rate limits, credit deduction, and secret-safe deployment conventions.
- **Training/intelligence surfaces:** indexing, search/embed helpers, dataset/job/eval/model-registry endpoints, and code artifact upload/download routes.

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
- Search flow: call `uiai_search`/`browser_search`, open a selected result with `browser_open` or `scripts/uiai-open-result.sh`, then use `browser_read` and diagnostics as needed.

## Documentation map

Start here by task:

| Need | Read |
|---|---|
| HLT ledger and current trajectory | [`docs/HLT_LEDGER.md`](docs/HLT_LEDGER.md) |
| Agent discovery index: `/api/tools/agent-card`, `pi_uiai_agent_card`, cards, search, graph, docs endpoint, Pi/MCP/CLI discovery, gaps | [`docs/AGENT_DISCOVERY_INDEX.md`](docs/AGENT_DISCOVERY_INDEX.md) |
| Agent quickstart: Pi, MCP, CLI, HTTP, browser workflow, Focusa packet handoff | [`docs/UIAI_FOR_AGENTS_QUICKSTART.md`](docs/UIAI_FOR_AGENTS_QUICKSTART.md) |
| Agent UX cookbook: search/read, @refs, diagnostics, packets, visual QA, release proof | [`docs/AGENT_UX_COOKBOOK.md`](docs/AGENT_UX_COOKBOOK.md) |
| Draft Source-to-Markdown spec: webpage/social/docs/issues/videos → Markdown/JSONL with agent discovery and Focusa evidence | [`docs/SOURCE_TO_MARKDOWN_AGENT_SPEC.md`](docs/SOURCE_TO_MARKDOWN_AGENT_SPEC.md) |
| Completed agent-experience roadmap implementation summary | [`docs/AGENT_EXPERIENCE_ROADMAP_IMPLEMENTATION_SUMMARY.md`](docs/AGENT_EXPERIENCE_ROADMAP_IMPLEMENTATION_SUMMARY.md) |
| Repo-local Pi skill source-of-truth policy | [`docs/SKILL_SOURCE_OF_TRUTH_POLICY.md`](docs/SKILL_SOURCE_OF_TRUTH_POLICY.md) |
| Focusa packet examples gallery | [`docs/FOCUSA_PACKET_EXAMPLES_GALLERY.md`](docs/FOCUSA_PACKET_EXAMPLES_GALLERY.md) |
| MCP cache and reconnect troubleshooting | [`docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md`](docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md) |
| Remote auth examples for curl, CLI, Pi, and MCP | [`docs/REMOTE_AUTH_EXAMPLES.md`](docs/REMOTE_AUTH_EXAMPLES.md) |
| Browser/session API, agent tools, MCP formats, security boundaries, portability helpers | [`docs/SESSION_API.md`](docs/SESSION_API.md) |
| UIAI × Focusa × Pi hand-in-glove research/diagnostics/evidence packet spec | [`docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`](docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md) |
| Agent FPV Co-Pilot spec for live operator oversight of browser sessions | [`docs/UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`](docs/UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md) |
| Agent FPV PWA fast-path for cohort-friendly mobile share links | [`docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`](docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md) |
| Browser UX/DX feedback and recommendation conflict inventory | [`docs/UIAI_BROWSER_UX_DX_FEEDBACK_2026-06-09.md`](docs/UIAI_BROWSER_UX_DX_FEEDBACK_2026-06-09.md), [`docs/UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md`](docs/UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md) |
| Cross-surface interoperability contracts, acceptance matrix, and proof gates | [`docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md`](docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md) |
| Agent-surface release proof checklist | [`docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md`](docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md) |
| Release/deploy/push/watch-CI runbook | [`docs/RELEASE_DEPLOY_RUNBOOK.md`](docs/RELEASE_DEPLOY_RUNBOOK.md) |
| GitHub Actions / Browser Reliability failure diagnostics | [`docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md`](docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md) |
| Public API parity matrix across HTTP, Pi, MCP, CLI, auth, and smokes | [`docs/PUBLIC_API_PARITY_MATRIX.md`](docs/PUBLIC_API_PARITY_MATRIX.md) |
| Non-browser agent API exposure inventory | [`docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`](docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md) |
| Endpoint auth mode matrix and route update rules | [`docs/ENDPOINT_AUTH_MATRIX.md`](docs/ENDPOINT_AUTH_MATRIX.md) |
| Existing WordPress plugin ↔ Go engine route parity matrix | [`docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md`](docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md) |
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
| `GET` | `/api/tools/docs` | Lightweight docs/examples metadata for agents. |
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
| `GET` | `/api/captcha/status` | Authenticated solver status/capabilities. |
| `GET` | `/api/captcha/pool` | Authenticated proxy/IP pool status. |
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

- Project-local Pi extension: [`.pi/extensions/uiai-engine.ts`](.pi/extensions/uiai-engine.ts) registers a full Pi-facing mirror of the MCP/browser tool surface: agent card/tool search/graph, provider web search, Source-to-Markdown, browser sessions/actions/diagnostics, one-shot screenshots, and frame catalog/render helpers. `/uiai research <query>`, `/uiai proof <url>`, and `/uiai diagnose <session_id>` execute guided packet workflows and insert Focusa args previews; `/uiai off` persists hidden widget state for the session; `/uiai on`/`show`/`enable` restores and persists the widget.
- Project-local Pi skills: [`.pi/skills/uiai-agent/SKILL.md`](.pi/skills/uiai-agent/SKILL.md) is the main UIAI agent workflow skill; [`.pi/skills/uiai-focusa-packet/SKILL.md`](.pi/skills/uiai-focusa-packet/SKILL.md) is the Focusa packet proof/handoff playbook; [`.pi/skills/uiai-release/SKILL.md`](.pi/skills/uiai-release/SKILL.md) is the release/deploy/push/CI proof workflow; [`.pi/skills/uiai-mcp/SKILL.md`](.pi/skills/uiai-mcp/SKILL.md) is the MCP setup/reconnect/route-parity workflow; [`.pi/skills/uiai-remote-auth/SKILL.md`](.pi/skills/uiai-remote-auth/SKILL.md) is the authenticated remote/tunnel workflow; [`.pi/skills/uiai-docs-maintenance/SKILL.md`](.pi/skills/uiai-docs-maintenance/SKILL.md) is the docs/parity/drift maintenance workflow; [`.pi/skills/uiai-ci-debug/SKILL.md`](.pi/skills/uiai-ci-debug/SKILL.md) is the GitHub Actions failure diagnostics workflow; [`.pi/skills/uiai-browser-debug/SKILL.md`](.pi/skills/uiai-browser-debug/SKILL.md) is the diagnostics-first browser failure workflow; [`.pi/skills/vision/SKILL.md`](.pi/skills/vision/SKILL.md) is the browser/vision workflow skill. Global copies under `~/.pi/skills/` are convenience installs only and should be refreshed from repo copies when UIAI workflows change.
- MCP bridge: [`mcp/browser-session-mcp.mjs`](mcp/browser-session-mcp.mjs) exposes browser/session tools plus `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph`, `uiai_health`, `uiai_status`, `uiai_errors`, `uiai_focusa_packet_compose`, `critique_models`, `critique_dimensions`, `source_to_markdown`, `browser_search`, `browser_read`, `frame_catalog`, and `frame_render`; `tools/list` normalizes these core tools even when a running engine returns stale metadata. MCP clients commonly cache `tools/list` for the lifetime of the stdio process, so after adding/removing/renaming tools restart/reconnect the MCP server/client (and reload Pi sessions that use `pi-mcp-adapter`) before expecting fresh tool metadata.
- Agent web surfing: persistent sessions include `/api/session/{id}/read` / `browser_read` for bounded text or Markdown extraction plus @ref actions for navigation and forms; `POST /api/markdown` provides one-shot public URL to `uiai.source_markdown.v1` with diagnostics, Focusa-ready evidence, and `wpuiai.research_card` / `wpuiai.report` productization objects.
- Search provider behavior: `/api/search/providers` reports configured/degraded readiness and `cache_ttl_seconds` without secrets; Brave is the default provider and Wikipedia OpenSearch is a keyless public second provider. Provider calls use a bounded 12s timeout, successful results are cached in memory for `UIAI_SEARCH_CACHE_TTL_SECONDS` (default `60`, `0` disables), result fields are bounded/redacted before agent exposure, and upstream quota/rate limits remain provider-account concerns.
- Diagnostics-first debugging: `browser_diagnostics` exposes console/errors/network/failed requests without forcing screenshots.
- [Focusa](https://github.com/Startempire-Wire/focusa) handoff: `browser_open` accepts `focusa_scope`; diagnostics and evidence flows preserve project/workpoint scope; Focusa intake consumes UIAI outputs without bypassing UIAI auth/redaction/URL safety; `/api/tools/graph` exposes [Focusa](https://github.com/Startempire-Wire/focusa)-aware related-tool routes. Stable evidence handles are documented for diagnostics, errors, search results, browser reads/snapshots, screenshots, and shares. UIAI composes `uiai.focusa_research_diagnostics_packet.v1` through the Pi-local `uiai_focusa_packet_build` tool and the HTTP/MCP/CLI composer at `POST /api/agent/research-packet` / `uiai_focusa_packet_compose` / `scripts/uiai packet compose <json-file|->`; `/api/tools/graph` advertises the workflow and metadata surfaces. The iterative hand-in-glove roadmap is in [`docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`](docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md).
- Portability: set `UIAI_ENGINE_URL`, `UIAI_PI_TIMEOUT_MS`, or `UIAI_MCP_TIMEOUT_MS` for remote/tunnel deployments.
- Security: remote browser/session API callers must authenticate; loopback remains frictionless for local agents.

See [`docs/SESSION_API.md`](docs/SESSION_API.md), [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md), and [`docs/BROWSER_RELIABILITY_RUNBOOK.md`](docs/BROWSER_RELIABILITY_RUNBOOK.md).
- Interoperability quality spec: [`docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md`](docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md) covers WordPress, Pi, MCP, Focusa, browser diagnostics/errors, CLI, search/providers, auth/security, and proof gates.

## Quick setup for agents

Start with the full agent path in [`docs/UIAI_FOR_AGENTS_QUICKSTART.md`](docs/UIAI_FOR_AGENTS_QUICKSTART.md), then install local integrations:

```bash
# Optional preview
DRY_RUN=1 scripts/install-agent-integrations.sh
scripts/install-pi-skills.sh --dry-run

# Install Pi extension + MCP config + convenience global skill copies
scripts/install-agent-integrations.sh
scripts/install-pi-skills.sh --apply

# Smoke-check skills/discovery/graph/MCP bridge
scripts/smoke-pi-skills.sh
scripts/smoke-open-result.sh
scripts/smoke-render-diagnostics-artifact.sh
scripts/smoke-agent-integrations.sh
scripts/uiai-mcp-reconnect-help.sh --check
scripts/smoke-mcp-tool-routes.sh
scripts/smoke-mcp-structured-failure.sh
scripts/smoke-pi-extension-registration.sh
scripts/smoke-pi-rendering.sh
scripts/smoke-pi-uiai-off.sh
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
scripts/install-pi-skills.sh --apply
scripts/smoke-agent-integrations.sh
```


### CLI wrapper

`scripts/uiai` is the lightweight unified CLI surface for operator/agent shell workflows. It wraps the HTTP API and existing smoke/install scripts before a full Go subcommand CLI is justified.

Examples:

```bash
scripts/uiai status
scripts/uiai health
scripts/uiai errors --limit 10 --source browser_session
scripts/uiai tools search diagnostics
scripts/uiai tools docs
scripts/uiai-open-result.sh --query "UIAI Engine browser agents" --index 1 --out /tmp/uiai-open-result.json
SID=$(scripts/uiai --json session open https://example.com | jq -r '.session.id')
scripts/uiai session read "$SID" --max-chars 1000
scripts/uiai session diagnostics "$SID"
scripts/render-diagnostics-artifact.py /tmp/uiai-focusa-packet-smoke.json
scripts/uiai session close "$SID"
scripts/uiai smoke agent
scripts/uiai smoke packet
```

Output modes: `--compact` (default), `--json`, `--pretty`.

`browser_eval_async` returns a structured `eval_failed` error envelope for JS/runtime errors instead of a successful text payload; its next action points agents to browser_diagnostics console/exceptions and bounded direct-action retries.
 The Pi extension also uses compact-by-default rendering with Ctrl+O expansion to full JSON for representative success/error tool results. Exit codes: `0` success, `1` API/tool failure, `2` usage error, `3` missing dependency, `4` auth/config error. Auth/env vars match Pi/MCP: `UIAI_ENGINE_URL`, `UIAI_API_KEY`, `UIAI_BEARER_TOKEN`, `UIAI_CLI_TIMEOUT_SECONDS`.

## Interoperability and security matrices

Use [`docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md`](docs/UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md) as the top-level matrix for WordPress, Pi, MCP, Focusa, browser diagnostics/errors, CLI, search/providers, auth/security, and proof gates.

Endpoint auth modes and update rules live in [`docs/ENDPOINT_AUTH_MATRIX.md`](docs/ENDPOINT_AUTH_MATRIX.md). Existing WordPress plugin route parity is inventoried in [`docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md`](docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md).

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
scripts/smoke-browser-error-regressions.sh
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

- Secrets should be referenced through environment variables or an external systemd environment file such as `/etc/wpuiai/ai-api.env`, not committed literal values.
- `vision.allow_private_urls: false` is the hardened default and blocks private/internal navigation. Set it to `true` only in explicit local/dev profiles, then run private localhost smokes with `UIAI_ALLOW_PRIVATE_SMOKES=1`.
- Browser/session, screenshot, provider search, and `/api/agent/research-packet` APIs are loopback-public only; remote callers must authenticate. The Pi extension can send `UIAI_API_KEY` or `UIAI_BEARER_TOKEN` for authenticated remote/media helpers. Local VPS deployments may configure an eternal env-backed `UIAI_LOCAL_API_TOKEN` accepted as `X-API-Key`, `X-License-Key`, or `Authorization: Bearer ...`.

## Security and exposure model

- `/api/tools*` discovery is intentionally public and low-context.
- Non-browser paid/mutating/admin/training/memory route families are not automatically agent-exposed; current expose/omit rationale lives in [`docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`](docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md).
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
| [`deploy/uiai-engine.service`](deploy/uiai-engine.service) | secret-safe systemd service unit template aligned with browser/session resource needs. |
| [`deploy/uiai-log-trim.service`](deploy/uiai-log-trim.service) | log trimming service. |
| [`deploy/uiai-log-trim.timer`](deploy/uiai-log-trim.timer) | log trimming timer. |
| [`.github/workflows/browser-reliability.yml`](.github/workflows/browser-reliability.yml) | browser reliability CI. |
| [`scripts/trim-runtime-logs.sh`](scripts/trim-runtime-logs.sh) | runtime log cleanup. |
| [`scripts/stress-browser-diagnostics.sh`](scripts/stress-browser-diagnostics.sh) | diagnostics stress run. |
| [`scripts/soak-browser-flakiness.sh`](scripts/soak-browser-flakiness.sh) | browser flake soak. |

## Maintenance reference: legacy/parity notes

These docs are intentionally lower-priority for new readers. They exist for maintainers deciding whether old dependency paths can be retired or whether plugin callers need compatibility proof:

1. [`docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md`](docs/FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)
2. [`docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`](docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md)
3. [`docs/WORKFLOW_API_ORCHESTRATION.md`](docs/WORKFLOW_API_ORCHESTRATION.md)

For public/product positioning, lead with agent discovery, Source-to-Markdown, persistent browser workflows, diagnostics, Focusa/Pi/MCP handoff, and WPUIAI research-card/report outputs.

## Recommended contribution workflow

```bash
git status --short --branch --untracked-files=all
go test ./...
node --check mcp/browser-session-mcp.mjs
go build ./cmd/uiai-engine
scripts/smoke-agent-integrations.sh
# Optional local/dev-only private localhost browser regressions:
# UIAI_ALLOW_PRIVATE_SMOKES=1 scripts/smoke-agent-integrations.sh
```

For browser-affecting changes, also run or consult:

```bash
scripts/stress-browser-diagnostics.sh
scripts/soak-browser-flakiness.sh
```

When changing API behavior, update the relevant doc linked above and add/adjust route tests under [`internal/routes/`](internal/routes/) or subsystem tests under [`internal/`](internal/).
