#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
PI_EXT_SRC="$ROOT_DIR/.pi/extensions/uiai-engine.ts"
PI_EXT_DEST="${UIAI_PI_EXTENSION_DEST:-$HOME/.pi/agent/extensions/uiai-engine.ts}"
MCP_CONFIG_DEST="${UIAI_MCP_CONFIG_DEST:-$HOME/.pi/agent/mcp.json}"
MCP_SERVER_NAME="${UIAI_MCP_SERVER_NAME:-uiai-browser}"
DRY_RUN="${DRY_RUN:-0}"

say(){ printf "%s\n" "$*"; }
run(){ if [[ "$DRY_RUN" == "1" ]]; then printf "DRY_RUN "; printf "%q " "$@"; printf "\n"; else "$@"; fi; }

say "UIAI agent integration installer"
say "root=$ROOT_DIR"
say "engine_url=$ENGINE_URL"

if [[ ! -f "$PI_EXT_SRC" ]]; then
  say "missing Pi extension source: $PI_EXT_SRC" >&2
  exit 1
fi

run mkdir -p "$(dirname "$PI_EXT_DEST")"
run cp "$PI_EXT_SRC" "$PI_EXT_DEST"
say "Pi extension installed: $PI_EXT_DEST"

run mkdir -p "$(dirname "$MCP_CONFIG_DEST")"
if [[ "$DRY_RUN" == "1" ]]; then
  say "DRY_RUN would update MCP config: $MCP_CONFIG_DEST"
elif command -v jq >/dev/null 2>&1; then
  tmp="$(mktemp)"
  if [[ -f "$MCP_CONFIG_DEST" ]]; then
    cp "$MCP_CONFIG_DEST" "$tmp"
  else
    printf '{"mcpServers":{}}' > "$tmp"
  fi
  jq --arg name "$MCP_SERVER_NAME" --arg script "$ROOT_DIR/mcp/browser-session-mcp.mjs" --arg url "$ENGINE_URL" \
    '.mcpServers[$name] = {"command":"node","args":[$script],"env":{"UIAI_ENGINE_URL":$url},"lifecycle":"lazy","idleTimeout":60,"description":"UIAI Engine browser/session tools with Focusa-aware graph metadata"}' \
    "$tmp" > "$tmp.out"
  mv "$tmp.out" "$MCP_CONFIG_DEST"
  rm -f "$tmp"
  say "MCP config updated: $MCP_CONFIG_DEST server=$MCP_SERVER_NAME"
else
  say "jq not found; writing standalone MCP config to $MCP_CONFIG_DEST.uiai-example"
  cat > "$MCP_CONFIG_DEST.uiai-example" <<JSON
{
  "mcpServers": {
    "$MCP_SERVER_NAME": {
      "command": "node",
      "args": ["$ROOT_DIR/mcp/browser-session-mcp.mjs"],
      "env": { "UIAI_ENGINE_URL": "$ENGINE_URL" },
      "lifecycle": "lazy",
      "idleTimeout": 60,
      "description": "UIAI Engine browser/session tools with Focusa-aware graph metadata"
    }
  }
}
JSON
fi

say "Next: run scripts/smoke-agent-integrations.sh"
