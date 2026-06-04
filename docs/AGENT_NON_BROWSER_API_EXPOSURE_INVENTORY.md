# Agent Non-Browser API Exposure Inventory

Purpose: document deeper non-browser API families that are intentionally exposed or omitted from Pi/MCP/CLI agent surfaces. This prevents accidental drift between HTTP route presence and agent-facing tool availability.

## Exposure principles

- Public/agent tool exposure is not automatic just because an HTTP route exists.
- Paid, mutating, admin, migration, memory, training, and handler-auth routes require a concrete agent workflow, auth boundary, redaction review, and smoke proof before Pi/MCP/CLI exposure.
- Public docs cite environment variable names and external env-file paths only; no literal API keys, bearer tokens, webhook secrets, cookies, or provider credentials.

## Current inventory

| API family | HTTP routes | Agent exposure state | Reason / next gate |
|---|---|---|---|
| Tool discovery | `/api/tools/*` | Exposed in Pi/MCP/CLI-adjacent docs | Low-context public metadata; smoke-tested through agent card/search/graph/MCP endpoints. |
| Browser/session | `/api/session/*`, `/api/screenshot`, `/api/share/*` | Exposed in Pi/MCP | Core browser-agent workflow; diagnostics/evidence handles and auth boundaries are smoke-tested. |
| Search providers | `/api/search*` | Exposed in Pi/MCP | Provider-neutral search is low-context enough for agents; results include bounded `uiai-search:<provider>:<query-hash>:<rank>` evidence refs. |
| Media frame helpers | `/api/media/frame/*` | Exposed in Pi/MCP | Read/render helper for screenshots; remote callers authenticate; smoke-tested. |
| Error events | `/api/errors` | Exposed in Pi/MCP | Bounded/redacted diagnostic surface; smoke-tested with `uiai-error:*` evidence handles. |
| Reference analysis | `/api/reference/analyze` | Intentionally omitted from Pi/MCP for now | Can spend AI/credits and needs model/input contract proof; expose only after auth, cost, and output-shape smokes. |
| Critique/UI reverse/section/layout/style/copilot | `/api/critique`, `/api/ui-reverse`, `/api/section-detect`, `/api/layout-compare`, `/api/style-enhance`, `/api/copilot/*` | Mostly omitted from Pi/MCP except metadata helpers | Paid AI routes; current agent need is model/dimension discovery, not full mutation/cost-incurring calls. |
| Admin/usage | `/api/admin/*`, `/api/usage/*` | Intentionally omitted from Pi/MCP | Admin reads/writes and usage data need stricter auth/redaction and least-privilege read-only design before exposure. |
| Memory | `/api/memory/*` | Intentionally omitted from Pi/MCP | User memory is sensitive and legacy-compatible; expose only with per-user scoping and redaction proof. |
| Workflow/intake/migration | `/api/workflow/*`, `/api/intake/*`, `/api/migration/*` | Intentionally omitted from Pi/MCP | Workflow and import routes are mutating; require concrete operator workflow, auth proof, and rollback/diagnostics handling. |
| Intelligence | `/api/intelligence/*` | Health only through general status; other routes omitted | Index/upload/search/embed/artifact routes need handler-auth review and artifact redaction before agent exposure. |
| Training | `/api/training/*` | Intentionally omitted from Pi/MCP | Service-token surface with model/dataset mutation; not safe for generic agent tools without dedicated service-token wiring and tests. |
| Captcha/IP pool | `/api/captcha/*` | Intentionally omitted from Pi/MCP | Operationally sensitive proxy/IP-pool mutation; keep HTTP-only until an explicit operator workflow and safety gates exist. |

## Add-exposure gate

Before adding any omitted family to Pi, MCP, or CLI surfaces:

1. Document the exact workflow and least-privilege route subset.
2. Confirm auth mode in `docs/ENDPOINT_AUTH_MATRIX.md`.
3. Add redaction tests for errors/logs/diagnostics when request or response data can contain secrets or user data.
4. Add route parity smoke for the new Pi/MCP/CLI tool.
5. Update README, Session API, interoperability spec, release proof checklist, and this inventory.
6. Live deploy and run the agent-surface release proof checklist.
