---
name: uiai-remote-auth
description: UIAI authenticated remote-use workflow: safe env vars, loopback-public vs remote-auth boundaries, CLI/Pi/MCP credentials, and examples for browser/search/screenshot/packet/frame calls.
---
# UIAI Remote Auth Skill

Use this skill when configuring, documenting, testing, or debugging UIAI Engine access from non-loopback agents, tunnels, remote Pi sessions, MCP clients, or scripts.

## Start here

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && sed -n "1,90p" docs/ENDPOINT_AUTH_MATRIX.md'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && sed -n "1,220p" docs/REMOTE_AUTH_EXAMPLES.md'
```

Canonical docs:

- `docs/ENDPOINT_AUTH_MATRIX.md` — auth-mode source of truth.
- `docs/REMOTE_AUTH_EXAMPLES.md` — safe curl, CLI, Pi, and MCP examples.
- `docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md` — reconnect after MCP env/tool changes.
- `docs/UIAI_FOR_AGENTS_QUICKSTART.md` — short operator path.

## Security rules

- Never paste real keys, bearer tokens, cookies, authorization headers, or webhook secrets into docs, bead notes, Focusa state, screenshots, packet examples, or final reports.
- Use env var names and placeholders only.
- Prefer loopback engine URLs for local agent work: `http://127.0.0.1:7456`.
- Browser target URLs are separate from engine URLs: private/internal targets are blocked by default unless `vision.allow_private_urls: true` is explicitly configured for local/dev.
- Remote/tunnel callers must send credentials for loopback-public remote-auth routes.
- Credentials do not bypass the private/internal URL safety policy.
- Public discovery endpoints are metadata/readiness only; do not infer that browser/session/search surfaces are public remotely.

## Auth boundary quick map

| Route family | Local loopback | Remote/tunnel |
|---|---:|---:|
| `/api/health`, `/api/status`, `/api/tools`, `/api/tools/mcp`, `/api/metrics/browser` | public | public |
| `/api/session`, `/api/screenshot`, `/api/search`, `/api/errors`, `/api/agent/research-packet`, `/api/media/frame` | no credential required | credential required |
| mutating/admin/AI pipeline routes | credential required | credential required |

Accepted client env vars:

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_API_KEY_VALUE"
# or:
export UIAI_BEARER_TOKEN="REDACTED_BEARER_VALUE"
```

`UIAI_API_KEY` sends `X-API-Key`. `UIAI_BEARER_TOKEN` sends `Authorization: Bearer ...`.

## CLI workflow

`scripts/uiai` reads `UIAI_ENGINE_URL`, `UIAI_API_KEY`, and `UIAI_BEARER_TOKEN`.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai status'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai health'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai tools'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai --json session open https://example.com'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai --json session read SESSION_ID'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai research packet --url https://example.com --goal "Remote auth proof" --out /tmp/uiai-research-packet.json'
```

## Curl workflow

Use a helper array so secrets are not printed in command output:

```bash
ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
AUTH_HEADER=()
if [ -n "${UIAI_API_KEY:-}" ]; then
  AUTH_HEADER=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [ -n "${UIAI_BEARER_TOKEN:-}" ]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi
```

Examples:

```bash
curl -s "$ENGINE_URL/api/tools/mcp" | jq '.tools | length'
curl -s "$ENGINE_URL/api/search" "${AUTH_HEADER[@]}" -H 'Content-Type: application/json' -d '{"query":"UIAI browser agents","numResults":3}' | jq
curl -s "$ENGINE_URL/api/screenshot" "${AUTH_HEADER[@]}" -H 'Content-Type: application/json' -d '{"url":"https://example.com","width":1280,"height":800}' | jq
curl -s "$ENGINE_URL/api/agent/research-packet" "${AUTH_HEADER[@]}" -H 'Content-Type: application/json' --data @/tmp/uiai-packet-request.json | jq
curl -s "$ENGINE_URL/api/media/frame/catalog" "${AUTH_HEADER[@]}" | jq
```

## Pi workflow

Start Pi after exporting env vars:

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_API_KEY_VALUE"
pi
```

Use:

```text
pi_uiai_agent_card
uiai_health
uiai_browser_open url="https://example.com"
uiai_browser_read session_id="SESSION_ID"
uiai_browser_diagnostics session_id="SESSION_ID"
uiai_focusa_packet_compose mode="proof" goal="Remote auth proof" responses=[...]
uiai_screenshot url="https://example.com"
uiai_frame_catalog
```

## MCP workflow

Install/merge config with env vars set:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/install-agent-integrations.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
```

After changing credentials or `UIAI_ENGINE_URL`, reconnect the MCP client and reload Pi sessions using `pi-mcp-adapter`.

## Validation

Local loopback proof:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./internal/auth'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
```

Agent surface proof:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
```

Remote scenario proof checklist:

- `UIAI_ENGINE_URL` points to the remote/tunnel endpoint.
- Exactly one credential style is configured unless deliberately testing precedence.
- Public discovery calls work without credentials.
- Loopback-public route families return `401` remotely without credentials and succeed with `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`.
- MCP/Pi sessions were restarted after env var changes.

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| Remote browser/search call returns `401` | missing remote credential | Set `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`; rerun smoke. |
| Browser/session call returns `url_not_allowed` | target URL is private/internal under hardened policy | Use a public target URL or switch only an explicit local/dev profile to `vision.allow_private_urls: true`. |
| Public discovery works but session/search fails | route is loopback-public remote-auth, not public | Use auth header for remote call. |
| Pi/MCP still uses old URL/token | session cached env/stdio process | Restart Pi/MCP client and reconnect bridge. |
| Auth value appears in logs/docs | unsafe reporting | Redact immediately; replace with env var names and rotate if exposed. |
| Local loopback unexpectedly needs auth | auth matrix drift or loopback detection issue | Run `go test ./internal/auth` and inspect `internal/auth/auth.go`. |

## Final report

Include:

- Auth mode and route family tested.
- Credential style used by name only, never value.
- Loopback vs remote distinction.
- Proof commands and pass/fail result.
- Reconnect instruction if Pi/MCP env changed.
