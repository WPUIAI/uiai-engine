---
name: uiai-mcp
description: UIAI MCP bridge workflow: setup, discovery, tools/list cache refresh, reconnect, route parity, new tool routing, and validation of mcp/browser-session-mcp.mjs.
---
# UIAI MCP Skill

Use this skill when setting up, validating, debugging, or changing UIAI Engine MCP access through `mcp/browser-session-mcp.mjs`, `/api/tools/mcp`, or MCP client integrations.

## Start here

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch && bd ready | sed -n "1,120p"'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
```

Read:

- `docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md`
- `docs/PUBLIC_API_PARITY_MATRIX.md`
- `docs/SESSION_API.md` MCP section
- `docs/UIAI_FOR_AGENTS_QUICKSTART.md`
- `docs/REMOTE_AUTH_EXAMPLES.md` for remote/tunnel auth

## Canonical MCP surfaces

- Bridge: `mcp/browser-session-mcp.mjs`
- Bridge config example: `mcp/mcp.json`
- Engine metadata: `GET /api/tools/mcp`
- Tool graph/discovery: `uiai_tool_graph`, `uiai_tool_search`, `uiai_agent_card`
- Focusa packet composer: `uiai_focusa_packet_compose`
- Route parity smoke: `scripts/smoke-mcp-tool-routes.sh`
- Structured failure smoke: `scripts/smoke-mcp-structured-failure.sh`
- Integration smoke: `scripts/smoke-agent-integrations.sh`

## Setup

Install or refresh MCP config from repo root:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/install-agent-integrations.sh'
```

Manual config shape:

```json
{
  "mcpServers": {
    "uiai-engine": {
      "command": "node",
      "args": ["/home/wpuiai/uiai-engine/mcp/browser-session-mcp.mjs"],
      "env": {
        "UIAI_ENGINE_URL": "http://127.0.0.1:7456"
      }
    }
  }
}
```

Remote/tunnel clients may set `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`; never store literal secrets in docs, bead notes, Focusa state, or examples.

## Discovery workflow

Use discovery before loading full schemas:

```text
uiai_agent_card
uiai_tool_search q="diagnostics"
uiai_tool_graph
uiai_health
```

CLI equivalents:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai tools mcp'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai tools graph'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && curl -s http://127.0.0.1:7456/api/tools/mcp | jq ".tools | length"'
```

## tools/list caching and reconnect

MCP freshness is process-scoped:

1. The Go engine serves `/api/tools/mcp`.
2. The Node bridge fetches and caches merged `tools/list` metadata in memory.
3. MCP clients often cache `tools/list` for the session.

After adding/removing/renaming tools, changing schemas, or fixing `tools/call` routes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs && scripts/smoke-mcp-tool-routes.sh'
```

Then restart/reconnect the MCP client and reload Pi sessions using `pi-mcp-adapter`. A running stdio bridge keeps old JavaScript and old `tools/list` until restarted.

## Adding or changing an MCP tool

1. Update engine metadata and route docs if the HTTP/Pi/MCP surface changes.
2. Update `mcp/browser-session-mcp.mjs` `tools/call` routing.
3. Keep Pi parity aligned if the tool is mirrored.
4. Update `docs/PUBLIC_API_PARITY_MATRIX.md`, `docs/SESSION_API.md`, quickstart/cookbook, and relevant skills.
5. Run proof gates:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-structured-failure.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
```

6. Reconnect MCP clients.

## Focusa packet MCP route

`uiai_focusa_packet_compose` mirrors `POST /api/agent/research-packet` and CLI `scripts/uiai packet compose`.

Use it when MCP-collected search/read/snapshot/diagnostics/error/screenshot/share responses need bounded Focusa handoff proof:

```text
uiai_focusa_packet_compose mode="diagnose" goal="Explain browser failure" responses=[...] focusa_scope={project_root,continuity_id,evidence_ref}
```

Then pass `packet.recommended_focusa.args_preview` to the named Focusa tool. Diagnostics usually route to `focusa_browser_diagnostics_intake`; research/proof usually route to `focusa_evidence_capture`.

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| `tools/list` misses a new tool | client or stdio bridge cache | Run route parity smoke, then reconnect MCP client. |
| Tool advertised but call fails | missing `tools/call` branch | Fix `mcp/browser-session-mcp.mjs`, run `smoke-mcp-tool-routes.sh`, reconnect. |
| Schema looks old | cached metadata | Restart bridge/client and reload Pi MCP adapter session. |
| Remote `401` | loopback-public route called over remote without credentials | Set `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`; see `docs/REMOTE_AUTH_EXAMPLES.md`. |
| Structured failure lacks next action | bridge error envelope drift | Run `scripts/smoke-mcp-structured-failure.sh` and update failure text. |

## Final report

Include:

- MCP tool or route changed.
- `node --check` result.
- `scripts/check-tool-parity.sh` result.
- `scripts/smoke-mcp-tool-routes.sh` result.
- Reconnect/reload instruction given if metadata/schema changed.
- Any Focusa packet handoff evidence when `uiai_focusa_packet_compose` was involved.
