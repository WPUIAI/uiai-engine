#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
PI_EXT_SRC="$ROOT_DIR/integrations/pi/uiai-engine.ts"
PI_EXT_DEST="${UIAI_PI_EXTENSION_DEST:-$HOME/.pi/agent/extensions/uiai-engine.ts}"
PI_CONTRACT_SRC="$ROOT_DIR/integrations/pi/uiai/epwa-contract.mjs"
PI_CONTRACT_DEST="$(dirname "$PI_EXT_DEST")/uiai/epwa-contract.mjs"
PI_RESPONSE_SRC="$ROOT_DIR/integrations/pi/uiai/engine-response.mjs"
PI_RESPONSE_DEST="$(dirname "$PI_EXT_DEST")/uiai/engine-response.mjs"
MCP_CONFIG_DEST="${UIAI_MCP_CONFIG_DEST:-$HOME/.pi/agent/mcp.json}"
MCP_SERVER_NAME="${UIAI_MCP_SERVER_NAME:-uiai-browser}"
DRY_RUN="${DRY_RUN:-0}"
INSTALL_PI="${UIAI_INSTALL_PI:-1}"
MCP_BUNDLE_ROOT="${UIAI_MCP_BUNDLE_ROOT:-}"
MCP_SCRIPT="$ROOT_DIR/mcp/browser-session-mcp.mjs"
MCP_FILES=(mcp/browser-session-mcp.mjs mcp/epwa-contract.mjs integrations/pi/uiai/engine-response.mjs integrations/pi/uiai/epwa-contract.mjs)

say(){ printf "%s\n" "$*"; }
run(){ if [[ "$DRY_RUN" == "1" ]]; then printf "DRY_RUN "; printf "%q " "$@"; printf "\n"; else "$@"; fi; }

if [[ -z "${UIAI_ENGINE_URL:-}" && -f "$MCP_CONFIG_DEST" ]] && command -v jq >/dev/null 2>&1; then
  ENGINE_URL="$(jq -r --arg name "$MCP_SERVER_NAME" --arg fallback "$ENGINE_URL" '.mcpServers[$name].env.UIAI_ENGINE_URL // $fallback' "$MCP_CONFIG_DEST")"
fi

say "UIAI agent integration installer"
say "root=$ROOT_DIR"
say "engine_url=explicit environment, preserved configuration, or localhost default (value not logged)"
if [[ -n "${UIAI_API_KEY:-}" || -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  say "auth_env=client credentials will be passed to MCP config (values redacted)"
fi

for source in "$PI_EXT_SRC" "$PI_CONTRACT_SRC" "$PI_RESPONSE_SRC" "${MCP_FILES[@]/#/$ROOT_DIR/}"; do
  if [[ ! -f "$source" ]]; then
    say "missing Pi runtime dependency: $source" >&2
    exit 1
  fi
done

if [[ "$INSTALL_PI" == "1" ]]; then
  run mkdir -p "$(dirname "$PI_EXT_DEST")"
  run mkdir -p "$(dirname "$PI_CONTRACT_DEST")"
  run cp "$PI_CONTRACT_SRC" "$PI_CONTRACT_DEST"
  run cp "$PI_RESPONSE_SRC" "$PI_RESPONSE_DEST"
  run cp "$PI_EXT_SRC" "$PI_EXT_DEST"
  say "Pi extension installed: $PI_EXT_DEST"
elif [[ "$INSTALL_PI" != "0" ]]; then
  say "UIAI_INSTALL_PI must be 0 or 1" >&2
  exit 1
fi

if [[ -n "$MCP_BUNDLE_ROOT" ]]; then
  if [[ "$MCP_BUNDLE_ROOT" != /* || "$MCP_BUNDLE_ROOT" == "$ROOT_DIR" || "$MCP_BUNDLE_ROOT" == "/" ]]; then
    say "MCP bundle must be an explicit absolute installation directory outside the source root" >&2
    exit 1
  fi
  for relative in "${MCP_FILES[@]}"; do
    run mkdir -p "$(dirname "$MCP_BUNDLE_ROOT/$relative")"
    run cp "$ROOT_DIR/$relative" "$MCP_BUNDLE_ROOT/$relative"
  done
  MCP_SCRIPT="$MCP_BUNDLE_ROOT/mcp/browser-session-mcp.mjs"
  say "MCP runtime bundle installed: $MCP_SCRIPT"
fi

run mkdir -p "$(dirname "$MCP_CONFIG_DEST")"
if [[ "$DRY_RUN" == "1" ]]; then
  say "DRY_RUN would update MCP config: $MCP_CONFIG_DEST"
elif command -v jq >/dev/null 2>&1; then
  tmp="$(mktemp)"
  # Same-directory private output keeps credentials private and replacement atomic.
  output="$(mktemp "${MCP_CONFIG_DEST}.XXXXXX")"
  trap 'rm -f "$tmp" "$output"' EXIT
  if [[ -f "$MCP_CONFIG_DEST" ]]; then
    cp "$MCP_CONFIG_DEST" "$tmp"
  else
    printf '{"mcpServers":{}}' > "$tmp"
  fi
  UIAI_INSTALL_ENGINE_URL="$ENGINE_URL" jq --arg name "$MCP_SERVER_NAME" --arg script "$MCP_SCRIPT" \
    '(env.UIAI_API_KEY // "") as $api_key | (env.UIAI_BEARER_TOKEN // "") as $bearer |
    .mcpServers[$name] = ((.mcpServers[$name] // {}) + {"command":"node","args":[$script],"env":((.mcpServers[$name].env // {}) + {"UIAI_ENGINE_URL":env.UIAI_INSTALL_ENGINE_URL} + (if $api_key != "" then {"UIAI_API_KEY":$api_key} else {} end) + (if $bearer != "" then {"UIAI_BEARER_TOKEN":$bearer} else {} end)),"lifecycle":"lazy","idleTimeout":60,"description":"UIAI Engine browser/session/search tools with Focusa-aware graph metadata"})' \
    "$tmp" > "$output"
  mv "$output" "$MCP_CONFIG_DEST"
  rm -f "$tmp"
  trap - EXIT
  say "MCP config updated: $MCP_CONFIG_DEST server=$MCP_SERVER_NAME"
else
  say "jq not found; writing standalone MCP config to $MCP_CONFIG_DEST.uiai-example"
  cat > "$MCP_CONFIG_DEST.uiai-example" <<JSON
{
  "mcpServers": {
    "$MCP_SERVER_NAME": {
      "command": "node",
      "args": ["$MCP_SCRIPT"],
      "env": { "UIAI_ENGINE_URL": "$ENGINE_URL" },
      "lifecycle": "lazy",
      "idleTimeout": 60,
      "description": "UIAI Engine browser/session/search tools with Focusa-aware graph metadata"
    }
  }
}
JSON
fi

say "Next: run scripts/smoke-agent-integrations.sh"
