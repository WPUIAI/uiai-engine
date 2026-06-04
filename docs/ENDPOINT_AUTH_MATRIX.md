# UIAI Engine Endpoint Auth Matrix

Last updated: 2026-06-04

This matrix is the security contract for Go engine route families. Source of truth for enforcement is `internal/auth/auth.go`; route mounts are in `internal/server/server.go`.

## Auth modes

| Mode | Meaning | Accepted credentials |
|---|---|---|
| public | Safe read-only metadata/UI, reachable without credentials. If credentials are present, identity may be attached opportunistically. | Optional: `X-License-Key`, `X-API-Key`, `X-Extension-Token`, `Authorization: Bearer`, `X-Webhook-Secret` |
| loopback-public remote-auth | Local agent/tool calls may be unauthenticated from loopback; non-loopback callers must authenticate. | Loopback IP or any accepted credential |
| authenticated | Global middleware requires credentials before route handler runs. | `X-License-Key`, `X-API-Key`, `Authorization: Bearer`, `X-Extension-Token`, or `X-Webhook-Secret` |
| service-token | Route handler has its own service-token auth in addition to middleware exception. | Handler-specific token such as `X-API-Token`/Bearer for training |
| handler-auth | Middleware allows route family, but handler implements its own auth or token semantics. | Handler-specific |

Local-token env support: `UIAI_LOCAL_API_TOKEN` or comma-separated `UIAI_LOCAL_API_TOKENS` authenticate `X-API-Key`, `Authorization: Bearer`, and `X-License-Key` as internal local calls.

## Matrix

| Endpoint group | Mode | Evidence | Notes |
|---|---|---|---|
| `/`, `/dashboard` | public | `internal/auth/auth.go` public list; `internal/server/server.go` root/dashboard routes | Service info and local dashboard shell. |
| `/health`, `/api/health`, `/api/health/*`, `/api/status`, `/api/metrics/browser` | public | `internal/auth/auth.go` public list | Operational status/readiness only. |
| `/api/tools`, `/api/tools/*` | public | `internal/auth/auth.go` `strings.HasPrefix(p, "/api/tools")` | Agent discovery metadata; no secrets. |
| `/api/critique/models`, `/api/critique/dimensions` | public | `internal/auth/auth.go` public list | Metadata only; `/api/critique` write path remains authenticated. |
| `/api/ui-reverse/models`, `/api/ui-reverse/operations` | public | `internal/auth/auth.go` public list | Metadata only. |
| `/api/copilot/health`, `/api/intelligence/health` | public | `internal/auth/auth.go` public list | Health/readiness only. |
| `/api/screenshot`, `/api/screenshot/*` | loopback-public remote-auth | `isLoopbackToolPath`; `internal/auth/auth_test.go` pattern | Local visual tool surface; remote callers authenticate. |
| `/api/session`, `/api/session/*` | loopback-public remote-auth | `isLoopbackToolPath` | Persistent browser automation; remote callers authenticate. |
| `/api/search`, `/api/search/*` | loopback-public remote-auth | `isLoopbackToolPath`; `TestSearchToolPathLoopbackPublicRemoteAuth` | Provider-neutral search; remote callers authenticate. |
| `/api/media/frame`, `/api/media/frame/*` | loopback-public remote-auth | `isLoopbackToolPath`; `TestMediaFrameToolPathLoopbackPublicRemoteAuth` | Frame catalog/render agent helpers. |
| `/api/errors`, `/api/errors/*` | loopback-public remote-auth | `isLoopbackToolPath`; `TestErrorsToolPathLoopbackPublicRemoteAuth` | Bounded redacted troubleshooting events. |
| `/api/share/*`, `/v/{token}` | public | `internal/auth/auth.go` share exception; server viewer route | Share viewing is public by design. Review mutating share routes before expanding. |
| `/api/media/jobs`, `/api/media/status/*` | public read | `internal/auth/auth.go` media read exceptions | Read-only polling/listing. Media production remains authenticated. |
| `/api/media/produce` and non-frame/non-status media writes | authenticated | falls through middleware | Potentially paid/mutating media generation. |
| `/api/extension/verify`, `/api/extension/token` | handler-auth | `internal/auth/auth.go` exception; `internal/routes/extension.go` | Handler validates extension token behavior. Other extension routes fall through unless explicitly excepted. |
| `/api/memory/*`, `/api/usage/*`, `/api/workflow/*` | public/legacy-compatible | `internal/auth/auth.go` Bun compatibility exceptions | Compatibility surface; audit before remote exposure. |
| `/api/training/*` | service-token | `internal/auth/auth.go` training exception; `internal/routes/training.go requireAuth` | Requires `X-API-Token` or Bearer matching training auth env. |
| `/api/intelligence/*` except `/health` | handler-auth | `internal/auth/auth.go` intelligence exception; `internal/routes/intelligence.go` | Per-handler auth/validation semantics. |
| `/api/admin/*` | authenticated | falls through middleware | Admin reads/writes. |
| `/api/reference/*` | authenticated | falls through middleware | Reference analysis can spend AI/credits. |
| `/api/critique`, `/api/section-detect`, `/api/layout-compare`, `/api/style-enhance`, `/api/copilot/*`, `/api/intake/*` | authenticated | falls through middleware | AI/analysis operations. |
| `/api/design-system`, `/api/content-map`, `/api/block-recipes`, `/api/comparison` | authenticated | falls through middleware | Pipeline operations. |
| `/api/captcha/*` | authenticated | falls through middleware | Solver/pool operations are sensitive/mutating. |
| `/api/migration/*`, `/api/events`, `/vision/*` | authenticated | falls through middleware | Migration, SSE, and interactive vision surfaces. |

## Update rules for new route families

1. Add the route family to this matrix in the same change that adds or changes route mounts.
2. Choose the narrowest auth mode. Use `loopback-public remote-auth` only for local agent helper surfaces that are safe for unauthenticated loopback use.
3. If a route is added to `isLoopbackToolPath`, add or update an auth test proving loopback allowed and remote unauthenticated denied.
4. If a route is added to the middleware public exception list, document why public exposure is safe and whether credentials are still opportunistically accepted.
5. If a route has handler-specific auth, document the token/header name and add handler-level positive/negative tests when practical.
6. All auth failures and error logs must preserve redaction guarantees: no `Authorization`, cookies, API keys, bearer tokens, webhook secrets, request bodies, query secrets, or fragments.

## Current verification

- `go test ./internal/auth` covers loopback/remote boundaries for search, errors, media frame, loopback detection, and local-token auth.
- `scripts/smoke-agent-integrations.sh` exercises public discovery, loopback tool surfaces, MCP/Pi parity, and browser error smoke.
