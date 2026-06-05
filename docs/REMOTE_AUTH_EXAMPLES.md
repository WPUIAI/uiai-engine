# Remote Auth Examples

These examples show safe local loopback and remote authenticated calls for UIAI Engine agent surfaces. They use placeholders only. Never paste real tokens into docs, chats, screenshots, or packet examples.

## Auth modes to know

See [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md) for the canonical route table.

- **public**: safe discovery/readiness metadata. Remote callers may call without credentials.
- **loopback-public remote-auth**: local loopback callers may be unauthenticated; remote/tunnel callers must authenticate.
- **authenticated**: credentials required even locally.

Public discovery is not the same as loopback-public remote-auth. `/api/tools`, `/api/tools/mcp`, `/api/health`, `/api/status`, and `/api/metrics/browser` are public metadata/readiness surfaces. Browser/session, screenshot, search, research-packet, errors, and frame helper routes are local-agent surfaces: unauthenticated on loopback, authenticated over remote/tunnel access.

## Safe environment setup

Use one credential style. Values below are placeholders.

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_API_KEY_VALUE"          # sent as X-API-Key
# or:
export UIAI_BEARER_TOKEN="REDACTED_BEARER_VALUE"      # sent as Authorization: Bearer ...
```

Server-side local-token deployments may configure `UIAI_LOCAL_API_TOKEN` or comma-separated `UIAI_LOCAL_API_TOKENS`; clients still send `X-API-Key`, `X-License-Key`, or `Authorization: Bearer ...`.

## Curl helper

```bash
ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
AUTH_HEADER=()
if [ -n "${UIAI_API_KEY:-}" ]; then
  AUTH_HEADER=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [ -n "${UIAI_BEARER_TOKEN:-}" ]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi
```

## Public discovery examples

These are public metadata/readiness calls. Credentials are optional.

```bash
curl -s "$ENGINE_URL/api/health" | jq
curl -s "$ENGINE_URL/api/status" | jq
curl -s "$ENGINE_URL/api/tools" | jq '.tools | length'
curl -s "$ENGINE_URL/api/tools/mcp" | jq '.tools | length'
curl -s "$ENGINE_URL/api/tools/search?q=diagnostics" | jq
curl -s "$ENGINE_URL/api/search/providers" | jq
curl -s "$ENGINE_URL/api/metrics/browser" | jq '.agent_pressure'
```

## Browser/session examples

Loopback unauthenticated:

```bash
curl -s http://127.0.0.1:7456/api/session   -H 'Content-Type: application/json'   -d '{"url":"https://example.com","focusa_scope":{"project_root":"/home/wpuiai/uiai-engine"}}' | jq
```

Remote authenticated:

```bash
curl -s "$ENGINE_URL/api/session"   "${AUTH_HEADER[@]}"   -H 'Content-Type: application/json'   -d '{"url":"https://example.com","focusa_scope":{"project_root":"/home/wpuiai/uiai-engine"}}' | jq
```

Read/diagnostics/close after replacing `SESSION_ID`:

```bash
curl -s "$ENGINE_URL/api/session/SESSION_ID/read" "${AUTH_HEADER[@]}" | jq
curl -s "$ENGINE_URL/api/session/SESSION_ID/diagnostics" "${AUTH_HEADER[@]}" | jq
curl -s -X DELETE "$ENGINE_URL/api/session/SESSION_ID" "${AUTH_HEADER[@]}" | jq
```

## Screenshot examples

```bash
curl -s "$ENGINE_URL/api/screenshot"   "${AUTH_HEADER[@]}"   -H 'Content-Type: application/json'   -d '{"url":"https://example.com","width":1280,"height":800,"format":"jpeg","quality":80}' | jq '{ok,width,height,format,evidence_ref}'
```

## Search examples

```bash
curl -s "$ENGINE_URL/api/search"   "${AUTH_HEADER[@]}"   -H 'Content-Type: application/json'   -d '{"query":"UIAI browser agents","numResults":3}' | jq '{ok,provider,cached,results:[.results[] | {rank,title,url,evidence_ref}]}'
```

## Focusa research packet example

```bash
curl -s "$ENGINE_URL/api/agent/research-packet"   "${AUTH_HEADER[@]}"   -H 'Content-Type: application/json'   -d '{
    "mode":"research",
    "goal":"Summarize selected UIAI browser-agent result",
    "search_response":{"results":[{"rank":1,"title":"Example","url":"https://example.com","snippet":"Bounded snippet","evidence_ref":"uiai-search:brave:example:1"}]},
    "focusa_scope":{"project_root":"/home/wpuiai/uiai-engine","continuity_id":"focusa-cont-example"}
  }' | jq '{schema:.packet.schema,mode:.packet.mode,preferred_tool:.packet.recommended_focusa.preferred_tool,args_preview:.packet.recommended_focusa.args_preview}'
```

## Media/frame examples

Frame catalog:

```bash
curl -s "$ENGINE_URL/api/media/frame/catalog" "${AUTH_HEADER[@]}" | jq
```

Frame render uses a placeholder image payload. Replace with a bounded test image, not a secret screenshot.

```bash
curl -s "$ENGINE_URL/api/media/frame/render"   "${AUTH_HEADER[@]}"   -H 'Content-Type: application/json'   -d '{"frameId":"iphone-15-pro","imageBase64":"REDACTED_TEST_IMAGE_BASE64","fit":"cover","format":"png"}' | jq '{ok,frameId,width,height,evidence_ref}'
```

## scripts/uiai examples

`scripts/uiai` reads `UIAI_ENGINE_URL`, `UIAI_API_KEY`, and `UIAI_BEARER_TOKEN` automatically.

```bash
scripts/uiai status
scripts/uiai health
scripts/uiai tools
scripts/uiai --json session open https://example.com
scripts/uiai --json session read SESSION_ID
scripts/uiai --json session diagnostics SESSION_ID
scripts/uiai research packet --url https://example.com --goal "Remote auth proof" --out /tmp/uiai-research-packet.json
scripts/uiai smoke agent
scripts/uiai smoke browser
scripts/uiai smoke packet
```

## Pi extension examples

Set env before starting Pi from the repo root:

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_API_KEY_VALUE"
# or export UIAI_BEARER_TOKEN="REDACTED_BEARER_VALUE"
pi
```

Then use Pi tools such as `pi_uiai_agent_card`, `uiai_health`, `uiai_browser_open`, `uiai_browser_read`, `uiai_browser_diagnostics`, `uiai_focusa_packet_compose`, `uiai_screenshot`, `uiai_frame_catalog`, and `uiai_frame_render`.

## MCP bridge examples

Install/merge MCP config with placeholder credentials:

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_API_KEY_VALUE"
scripts/install-agent-integrations.sh
```

Manual MCP config shape:

```json
{
  "mcpServers": {
    "uiai-engine": {
      "command": "node",
      "args": ["/home/wpuiai/uiai-engine/mcp/browser-session-mcp.mjs"],
      "env": {
        "UIAI_ENGINE_URL": "https://uiai.example.invalid",
        "UIAI_API_KEY": "REDACTED_API_KEY_VALUE"
      }
    }
  }
}
```

After credential or tool schema changes, reconnect the MCP client; see [MCP Cache and Reconnect Troubleshooting](MCP_CACHE_RECONNECT_TROUBLESHOOTING.md).

## Verification

```bash
go test ./internal/auth
scripts/smoke-agent-integrations.sh
scripts/check-tool-parity.sh
scripts/smoke-pi-extension-registration.sh
scripts/smoke-mcp-tool-routes.sh
```

Expected proof:

- Remote-auth tests cover search, errors, media frame, session, screenshot, and research packet route families.
- Public discovery remains unauthenticated metadata/readiness.
- Pi/MCP/CLI helpers send either `X-API-Key` or `Authorization: Bearer ...` without exposing values.
