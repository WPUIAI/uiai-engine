#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq

say(){ printf "%s\n" "$*"; }
fetch(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "$@"; }

say "UIAI agent integration smoke"
say "engine_url=$ENGINE_URL"

fetch "$ENGINE_URL/api/health" | jq -e '.status == "healthy" or .status == "ok"' >/dev/null
fetch "$ENGINE_URL/api/tools/agent-card" | jq -e '.service == "uiai-engine"' >/dev/null
fetch "$ENGINE_URL/api/tools/graph" | jq -e '.focusa_integration.preferred_focusa_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=read" | jq -e '.tools[] | select(.name == "browser_read")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_open") | .related_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
node --check "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/mcp/browser-session-mcp.mjs" >/dev/null

say "agent integration smoke ok"
