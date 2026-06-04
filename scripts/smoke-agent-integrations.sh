#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
AUTH_ARGS=()
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  AUTH_ARGS=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq

say(){ printf "%s\n" "$*"; }
fetch(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "$@"; }
fetch_auth(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$@"; }

say "UIAI agent integration smoke"
say "engine_url=$ENGINE_URL"

fetch "$ENGINE_URL/api/health" | jq -e '.status == "healthy" or .status == "ok"' >/dev/null
fetch "$ENGINE_URL/api/tools/agent-card" | jq -e '.service == "uiai-engine"' >/dev/null
fetch "$ENGINE_URL/api/tools/graph" | jq -e '.focusa_integration.preferred_focusa_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/graph" | jq -e '.focusa_integration.evidence_refs[] | select(. == "uiai-search:<provider>:<query-hash>:<rank>")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=read" | jq -e '.tools[] | select(.name == "browser_read")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=search" | jq -e '.tools[] | select(.name == "browser_search")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_open") | .related_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_search")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_health")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_status")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "critique_models")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "critique_dimensions")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "frame_catalog")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_errors")' >/dev/null
fetch "$ENGINE_URL/api/media/frame/catalog" | jq -e '.count > 0' >/dev/null
fetch "$ENGINE_URL/api/errors?limit=1" | jq -e 'has("events") and has("redaction")' >/dev/null
fetch_auth "$ENGINE_URL/api/search/providers" | jq -e '.providers[] | select(.id == "brave") | has("configured") and has("cache_ttl_seconds")' >/dev/null
fetch_auth -X POST "$ENGINE_URL/api/search" -H "Content-Type: application/json" -d '{"query":"UIAI Engine browser agents","limit":1}' | jq -e '.provider == "brave" and .count >= 1 and has("cached") and has("cache_ttl_seconds") and (.results[0].evidence_ref | startswith("uiai-search:brave:")) and .results[0].rank == 1' >/dev/null
node --check "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/mcp/browser-session-mcp.mjs" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-mcp-tool-routes.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-mcp-structured-failure.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-focusa-error-evidence.sh" >/dev/null
CLI="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/uiai"
bash -n "$CLI"
"$CLI" --json status | jq -e '.type == "status"' >/dev/null
"$CLI" --json errors --limit 1 | jq -e 'has("events")' >/dev/null
"$CLI" --json tools mcp | jq -e '.tools[] | select(.name == "uiai_errors")' >/dev/null

"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-pi-extension-registration.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-pi-rendering.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-pi-uiai-off.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-failed-network-diagnostics.sh" >/dev/null
"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/smoke-browser-error-regressions.sh" >/dev/null

say "agent integration smoke ok"
