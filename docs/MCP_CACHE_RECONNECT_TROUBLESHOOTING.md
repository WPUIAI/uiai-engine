# MCP Cache and Reconnect Troubleshooting

UIAI exposes MCP tools through the Node stdio bridge at `mcp/browser-session-mcp.mjs`. MCP clients often cache `tools/list` metadata for the lifetime of the stdio process or client session. After UIAI route, schema, or bridge code changes, reconnect before trusting the advertised MCP surface.

## Symptoms

- `tools/list` misses a newly added tool.
- `tools/list` still advertises a removed or renamed tool.
- A tool is advertised but `tools/call` returns an unknown-tool or no-route failure.
- A fixed `tools/call` handler still behaves like the old code.
- Tool schema or argument descriptions look stale in Pi, Claude Desktop, or another MCP client.
- Pi sessions using `pi-mcp-adapter` keep old metadata after code/docs changed.

## Why it happens

There are three cache layers:

1. **Go engine metadata**: `/api/tools/mcp` returns generated MCP tool definitions.
2. **Node MCP bridge process**: `tools/list` fetches `/api/tools/mcp`, bridge-normalizes core tools, and caches the merged result in memory.
3. **MCP client session**: many clients call `tools/list` once per stdio process/session and reuse that metadata.

This means adding/removing/renaming tools, changing schemas, or fixing `tools/call` routes requires a fresh MCP bridge/client process. `tools/call` route fixes also require reconnect because the running Node stdio bridge keeps old JavaScript loaded until it exits.

## Quick proof commands

Run from the repo root:

```bash
node --check mcp/browser-session-mcp.mjs
scripts/smoke-mcp-tool-routes.sh
scripts/smoke-mcp-structured-failure.sh
scripts/check-tool-parity.sh
```

`smoke-mcp-tool-routes.sh` proves every advertised MCP tool has a bridge `tools/call` route. It catches advertised-but-unrouted drift after add/remove/rename work.

## Reconnect procedure

### UIAI bridge used by generic MCP clients

1. Stop or disconnect the MCP client session that launched `mcp/browser-session-mcp.mjs`.
2. Start a fresh session so the stdio bridge launches a new Node process.
3. Ask the client to reload/list MCP tools.
4. Re-run the route parity smoke locally if the client still looks stale:

```bash
scripts/smoke-mcp-tool-routes.sh
```

If the stale tool persists after reconnect, inspect the client-side MCP config and verify it points at the current repo path:

```bash
cat mcp/mcp.json
node --check mcp/browser-session-mcp.mjs
```

### Pi sessions using `pi-mcp-adapter`

1. Exit or reload the Pi session that loaded the adapter.
2. Reopen Pi from the UIAI repo root.
3. Reconnect the MCP server through the Pi MCP tool or adapter command.
4. Re-run tool discovery; do not rely on the previous session's tool list.

If a Pi session still shows old tools, start a clean Pi session after confirming the repo-local bridge path and rerun:

```bash
scripts/smoke-agent-integrations.sh
scripts/smoke-mcp-tool-routes.sh
```

### Local service changes

If route code or `/api/tools/mcp` generation changed, rebuild/restart the engine before reconnecting clients:

```bash
go test ./...
scripts/check-tool-parity.sh
# service-managed deploys only:
sudo systemctl restart uiai-engine.service
```

Then reconnect MCP clients so both the engine metadata and stdio bridge code are fresh.

## Add/remove/rename checklist

When adding, removing, or renaming an MCP tool:

1. Update Go tool metadata / HTTP routes as needed.
2. Update `mcp/browser-session-mcp.mjs` `tools/call` routing.
3. Update Pi extension parity if the tool is mirrored.
4. Update docs and examples.
5. Run:

```bash
node --check mcp/browser-session-mcp.mjs
scripts/check-tool-parity.sh
scripts/smoke-mcp-tool-routes.sh
scripts/smoke-mcp-structured-failure.sh
scripts/smoke-pi-extension-registration.sh
```

6. Restart/reconnect MCP clients and reload Pi adapter sessions.

## Call-route fix checklist

When `tools/list` is correct but calls fail:

1. Confirm the advertised name exists in `/api/tools/mcp`.
2. Add or fix the matching `tools/call` branch in `mcp/browser-session-mcp.mjs`.
3. Run `node --check mcp/browser-session-mcp.mjs`.
4. Run `scripts/smoke-mcp-tool-routes.sh`.
5. Reconnect the MCP client because the running Node bridge still has old code.

## Reporting stale MCP issues

Include:

- Client name/version if known.
- Whether a fresh stdio bridge process was started.
- Output from `scripts/smoke-mcp-tool-routes.sh`.
- Tool name and whether the issue is stale `tools/list`, wrong schema, or broken `tools/call`.
- Current `UIAI_ENGINE_URL` and bridge path, with secrets redacted.

## Related docs

- [Session API](SESSION_API.md)
- [Public API Parity Matrix](PUBLIC_API_PARITY_MATRIX.md)
- [Agent UX Cookbook](AGENT_UX_COOKBOOK.md)
- [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md)
