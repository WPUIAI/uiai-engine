# UIAI Engine

## Agent integration highlights

- Project-local Pi extension: `.pi/extensions/uiai-engine.ts` registers compact UIAI tools for Pi sessions.
- MCP bridge: `mcp/browser-session-mcp.mjs` exposes browser/session tools plus `uiai_agent_card` and `uiai_tool_search`.
- Agent web surfing: persistent sessions now include `/api/session/{id}/read` / `browser_read` for bounded page text extraction.
- Focusa handoff: `browser_open` accepts `focusa_scope`; diagnostics and evidence flows can preserve project/workpoint scope. `/api/tools/graph` exposes Focusa-aware related-tool routes.
- Portability: set `UIAI_ENGINE_URL`, `UIAI_PI_TIMEOUT_MS`, or `UIAI_MCP_TIMEOUT_MS` for remote/tunnel deployments. Remote browser/session API callers must authenticate; loopback remains frictionless for local agents.

See `docs/SESSION_API.md` and `docs/BROWSER_DIAGNOSTICS_SPEC.md` for public API details.

## Quick setup for agents

```bash
# Optional preview
DRY_RUN=1 scripts/install-agent-integrations.sh

# Install Pi extension + MCP config
scripts/install-agent-integrations.sh

# Smoke-check discovery/graph/MCP bridge
scripts/smoke-agent-integrations.sh
```

Set `UIAI_ENGINE_URL` for remote/tunnel targets.
