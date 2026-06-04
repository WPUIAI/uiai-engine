#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-30}"
BRIDGE="$ROOT_DIR/mcp/browser-session-mcp.mjs"
AUTH_ARGS=()
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  AUTH_ARGS=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq
need node

OPEN_JSON="$(curl -fsS --max-time "$TIMEOUT_SECONDS" -X POST "${AUTH_ARGS[@]}" "$ENGINE_URL/api/session" -H 'Content-Type: application/json' -d '{"url":"https://example.com","width":800,"height":600}')"
SID="$(printf '%s' "$OPEN_JSON" | jq -r '.session.id // .session_id // .id')"
if [[ -z "$SID" || "$SID" == "null" ]]; then
  echo "could not open smoke session" >&2
  exit 1
fi
cleanup(){ curl -fsS --max-time 5 -X DELETE "${AUTH_ARGS[@]}" "$ENGINE_URL/api/session/$SID" >/dev/null 2>&1 || true; }
trap cleanup EXIT

OUT="$(mktemp)"
trap 'cleanup; rm -f "$OUT"' EXIT
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"browser_click\",\"arguments\":{\"session_id\":\"$SID\",\"selector\":\"#uiai-smoke-definitely-missing\"}}}" \
| timeout "$TIMEOUT_SECONDS" node "$BRIDGE" > "$OUT"

jq -e 'select(.id == 2) | .result.isError == true' "$OUT" >/dev/null
TEXT="$(jq -r 'select(.id == 2) | .result.content[]? | select(.type == "text") | .text' "$OUT")"
for needle in 'Error 500:' 'id=' 'class=selector_not_found' 'Next:' 'snapshot or /diagnostics'; do
  if [[ "$TEXT" != *"$needle"* ]]; then
    echo "MCP structured failure text missing: $needle" >&2
    echo "$TEXT" >&2
    exit 1
  fi
done

echo "mcp structured failure ok: session=$SID class=selector_not_found"
