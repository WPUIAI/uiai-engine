#!/usr/bin/env bash
# Non-destructive MCP reconnect guidance for stale tools/list metadata.
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
CHECK=0
usage(){ cat <<'USAGE'
Usage: scripts/uiai-mcp-reconnect-help.sh [--check]

Prints safe reconnect steps for stale MCP tools/list metadata.
Does not kill processes or mutate MCP clients.

Options:
  --check  Also run local bridge syntax and MCP route parity checks.
USAGE
}
while [[ $# -gt 0 ]]; do
  case "$1" in
    --check) CHECK=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
cd "$ROOT_DIR"
cat <<EOF
UIAI MCP reconnect helper
repo: $ROOT_DIR
bridge: $ROOT_DIR/mcp/browser-session-mcp.mjs
engine_url: ${UIAI_ENGINE_URL:-http://127.0.0.1:7456}

Symptoms this fixes:
- tools/list misses a new tool or still shows a removed/renamed tool
- tool schema/description looks stale
- tools/list advertises a tool but tools/call uses old bridge code
- Pi sessions using pi-mcp-adapter still show old metadata

Safe reconnect steps:
1. Save current agent work/checkpoint if needed.
2. Run: node --check mcp/browser-session-mcp.mjs && scripts/smoke-mcp-tool-routes.sh
3. Disconnect/reconnect the MCP client or restart the client session that launched the stdio bridge.
4. Reload/reopen Pi sessions using pi-mcp-adapter.
5. Ask the client to list/search MCP tools again.
6. If still stale, verify MCP config points at this bridge path and current UIAI_ENGINE_URL.

No processes were killed by this helper.
Docs: docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md
EOF
if [[ "$CHECK" == "1" ]]; then
  node --check mcp/browser-session-mcp.mjs
  scripts/smoke-mcp-tool-routes.sh
fi
