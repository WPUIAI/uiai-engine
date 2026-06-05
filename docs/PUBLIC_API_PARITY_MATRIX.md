# UIAI Public API Parity Matrix

Purpose: map public/operator/agent-facing UIAI HTTP route families to Pi, MCP, CLI, auth, evidence, and smoke coverage so routes do not silently drift across surfaces.

## Source of truth

- HTTP route mounts: `internal/server/server.go`.
- Auth modes: `internal/auth/auth.go` and [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md).
- Pi tools: `.pi/extensions/uiai-engine.ts` plus `scripts/smoke-pi-extension-registration.sh`.
- MCP tools/routes: `mcp/browser-session-mcp.mjs` plus `scripts/smoke-mcp-tool-routes.sh`.
- CLI wrapper: `scripts/uiai`.
- Public API details: [Session API](SESSION_API.md), [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md), [Agent UX Cookbook](AGENT_UX_COOKBOOK.md).

## Exposure states

- **Full agent parity**: HTTP, Pi, MCP, and CLI/docs/smoke coverage exist for the intended workflow.
- **HTTP + Pi/MCP**: HTTP plus Pi/MCP tool coverage exist; CLI may be smoke/install/discovery only.
- **HTTP/docs only**: route is documented, but generic agent tools intentionally omit it.
- **HTTP internal/guarded**: route exists for plugin/admin/service use; not public agent surface.
- **Deferred**: route family needs workflow, auth, redaction, cost, or rollback proof before agent exposure.

## Public and agent-facing route families

| HTTP route family | Auth mode | Pi exposure | MCP exposure | CLI exposure | Evidence / handles | Smoke proof | Parity state | Notes / next gate |
|---|---|---|---|---|---|---|---|---|
| `/health`, `/api/health`, `/api/status` | public | `uiai_health`, `uiai_status` | `uiai_health`, `uiai_status` | `scripts/uiai health`, `scripts/uiai status` | service health/status JSON | `scripts/smoke-agent-integrations.sh`, `go test ./...` | Full agent parity | `/api/health/browser` adds browser pressure/agent summaries. |
| `/api/metrics/browser` | public | surfaced through `uiai_health`/browser health docs | via health/status route planning | status/health docs only | browser pressure summary | `scripts/smoke-agent-integrations.sh` graph/health checks | HTTP/docs only | Keep as read-only diagnostics; no separate tool unless repeated demand. |
| `/api/tools/*` | public discovery | `pi_uiai_agent_card`, `pi_uiai_tool_search`, `pi_uiai_tool_graph` | `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph` | `scripts/uiai tools ...` | tool graph workflow refs | `scripts/smoke-agent-integrations.sh` | Full agent parity | Generated tool defs remain authoritative. |
| `/api/search`, `/api/search/providers` | loopback-public remote-auth | `uiai_search` | `browser_search` | HTTP examples; packet smoke uses search | `uiai-search:<provider>:<query-hash>:<rank>` | `scripts/smoke-agent-integrations.sh`, `scripts/smoke-focusa-packet.sh` | Full agent parity | Provider metadata redacts keys and reports degraded status/cache TTL. |
| `/api/session/*` browser open/actions/read/snapshot/diagnostics | loopback-public remote-auth | `uiai_browser_*` | `browser_*` | `scripts/uiai session ...` for core open/read/diagnostics/close | `uiai-browser:*`, `uiai-diagnostics:*`, `uiai-error:*` | `scripts/smoke-agent-integrations.sh`, `scripts/smoke-browser-error-regressions.sh`, Browser Reliability workflow | Full agent parity for core browser workflows | CLI intentionally covers core loop; Pi/MCP cover broad action set. |
| `/api/screenshot`, `/api/screenshot/*` | loopback-public remote-auth | `uiai_screenshot` | `screenshot`/browser screenshot helpers | HTTP examples/smokes | `uiai-screenshot:sha256:<prefix>` | browser reliability scripts, agent integrations | HTTP + Pi/MCP | CLI one-shot can stay HTTP example until demand. |
| `/api/share/*`, `/v/{token}` | public share view; create/update auth per route | packet builder recognizes share artifacts | MCP graph/docs reference share evidence | HTTP examples | `uiai-share:<share_id>` | packet tests recognize share captures | HTTP/docs only | Keep transcript raw blobs out; expose share write tools only with workflow proof. |
| `/api/errors` | loopback-public remote-auth | `uiai_errors` | `uiai_errors` | HTTP examples; `scripts/uiai errors` if available | `uiai-error:<error_id>` | `scripts/smoke-agent-integrations.sh`, `scripts/smoke-mcp-structured-failure.sh` | Full agent parity | Redaction and bounded event shape required. |
| `/api/agent/research-packet` | loopback-public remote-auth | `uiai_focusa_packet_build` | `uiai_focusa_packet_compose` | `scripts/uiai packet compose`, `scripts/uiai smoke packet` | packet `evidence_refs[]`, Focusa args preview | `scripts/smoke-focusa-packet.sh`, packet builder tests, drift check | Full agent parity | Packet is a proposal/evidence bundle; Focusa decides durable writes. |
| `/api/media/frame/catalog`, `/api/media/frame/render` | loopback-public remote-auth | `uiai_frame_catalog`, `uiai_frame_render` | `frame_catalog`, `frame_render` | HTTP examples | rendered frame artifact refs | `scripts/smoke-agent-integrations.sh`, MCP route smoke | HTTP + Pi/MCP | Used for visual QA; keep base64 out of transcript. |
| `/api/critique/models`, `/api/critique/dimensions` | public metadata | `uiai_critique_models`, `uiai_critique_dimensions` | `critique_models`, `critique_dimensions` | HTTP examples | read-only model/dimension metadata | `scripts/smoke-agent-integrations.sh` | HTTP + Pi/MCP | Metadata only; paid critique execution remains omitted from generic agent tools. |

## Plugin/service/admin route families

| HTTP route family | Auth mode | Agent exposure | Reason / next gate |
|---|---|---|---|
| `/api/critique`, `/api/ui-reverse`, `/api/section-detect`, `/api/layout-compare`, `/api/style-enhance`, `/api/copilot/*` | authenticated or public metadata subroutes | Metadata exposed; paid/mutating execution omitted | Requires cost controls, output-shape smokes, structured error UI/log preservation, and operator workflow before Pi/MCP tools. |
| `/api/reference/analyze` | authenticated | omitted | Can spend AI/credits and needs model/input contract proof before agent exposure. |
| `/api/design-system`, `/api/content-map`, `/api/block-recipes`, `/api/comparison` | authenticated | omitted | Mutating/paid pipeline routes; expose only with explicit agent workflow, cost guard, and fallback/rollback proof. |
| `/api/intake`, `/api/workflow`, `/api/migration` | public-prefix/handler rules vary; mutating | omitted | Import/workflow/migration actions need rollback, auth, redaction, and operator confirmation policy. |
| `/api/admin/*`, `/api/usage/*` | public-prefix/handler rules vary | omitted | Admin/usage data needs least-privilege read-only design and redaction before exposure. |
| `/api/memory/*` | public-prefix/legacy-compatible | omitted | User memory is sensitive; expose only with per-user scoping and redaction proof. |
| `/api/intelligence/*` | handler-auth | health/status only through general health | Index/upload/search/embed/artifact routes need handler-auth review and artifact redaction before tools. |
| `/api/training/*` | service-token | omitted | Service-token dataset/model mutation is unsafe for generic agent tools without dedicated token wiring and tests. |
| `/api/captcha/*` | operationally sensitive | omitted | Proxy/IP-pool state and captcha solving are operational controls; keep HTTP-only until explicit operator workflow. |
| `/api/media/produce`, `/api/media/status/*`, `/api/media/jobs` | authenticated submit; public read/status | omitted except frame helpers | Production jobs can be long-running/costly; expose only with job lifecycle, artifact, and cancellation proof. |
| `/api/extension/*` | public verify/token rules | omitted | WordPress extension token lifecycle, not a generic agent tool. |
| `/api/events` | public SSE | omitted | Observability stream; expose only after bounded summary tool is defined. |
| `/dashboard` | public UI | omitted | Human dashboard, not tool surface. |

## Parity gates before adding a new public agent tool

1. Auth mode documented in [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md).
2. HTTP route has bounded/redacted success and structured failure behavior.
3. Pi tool added or deliberate omission documented.
4. MCP tool metadata and `tools/call` route added or deliberate omission documented.
5. CLI route added only if it improves operator workflow; otherwise link HTTP/Pi/MCP examples.
6. Smoke proof added or updated:
   - `scripts/smoke-pi-extension-registration.sh`
   - `scripts/smoke-mcp-tool-routes.sh`
   - `scripts/smoke-agent-integrations.sh`
   - route-specific smoke when failure behavior matters
7. Docs updated: README map, Session API, Agent UX Cookbook or Quickstart, release proof checklist when relevant.
8. Evidence handle shape documented if the route creates proof artifacts.

## Current known gaps

- CLI is intentionally lighter than Pi/MCP for browser actions; it covers discovery, core session/read/diagnostics/close, packet compose/smoke, installs, and release smokes.
- Share write/update workflows are documented as artifact refs but not generic Pi/MCP write tools.
- Paid AI execution routes are intentionally not generic Pi/MCP tools; metadata helper exposure is the current least-risk slice.
- Admin, memory, intelligence, training, captcha, migration, and media production route families remain HTTP/plugin/service surfaces pending explicit operator workflows.

## Verification commands

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
```

## Related docs

- [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md)
- [Agent Non-Browser API Exposure Inventory](AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md)
- [WordPress Plugin ↔ Go Engine Route Parity Matrix](WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md)
- [Full API Parity Evaluation and Retirement Inventory](FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)
- [Session API](SESSION_API.md)
- [Agent Surface Release Proof Checklist](AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md)
