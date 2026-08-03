#!/usr/bin/env bash
# Start a short-lived read-only Cockpit visual preview.
# This exposes only the frontend. Engine-backed actions require a local UIAI Engine.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${COCKPIT_PREVIEW_PORT:-4173}"
LOG_DIR="${TMPDIR:-/tmp}/uiai-cockpit-preview"
mkdir -p "$LOG_DIR"
VITE_LOG="$LOG_DIR/vite.log"
TUNNEL_LOG="$LOG_DIR/cloudflared.log"
VITE_PID=""
TUNNEL_PID=""

cleanup() {
  [[ -z "$TUNNEL_PID" ]] || kill "$TUNNEL_PID" >/dev/null 2>&1 || true
  [[ -z "$VITE_PID" ]] || kill "$VITE_PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

command -v npm >/dev/null 2>&1 || { echo "npm is required" >&2; exit 1; }
command -v cloudflared >/dev/null 2>&1 || { echo "cloudflared is required for a share URL" >&2; exit 1; }

cd "$ROOT_DIR"
echo "Starting Cockpit frontend on port ${PORT}…"
VITE_ALLOWED_HOSTS="*" npm run dev -- --host 0.0.0.0 --port "$PORT" >"$VITE_LOG" 2>&1 &
VITE_PID=$!
for _ in $(seq 1 60); do
  if curl -fsS --max-time 2 "http://127.0.0.1:$PORT/" >/dev/null 2>&1; then break; fi
  kill -0 "$VITE_PID" >/dev/null 2>&1 || { cat "$VITE_LOG" >&2; exit 1; }
  sleep 0.5
done

cloudflared tunnel --url "http://127.0.0.1:$PORT" --no-autoupdate >"$TUNNEL_LOG" 2>&1 &
TUNNEL_PID=$!
for _ in $(seq 1 30); do
  PREVIEW_URL="$(grep -Eo 'https://[a-z0-9-]+\.trycloudflare\.com' "$TUNNEL_LOG" | head -1 || true)"
  [[ -n "$PREVIEW_URL" ]] && break
  kill -0 "$TUNNEL_PID" >/dev/null 2>&1 || { cat "$TUNNEL_LOG" >&2; exit 1; }
  sleep 1
done

[[ -n "${PREVIEW_URL:-}" ]] || { cat "$TUNNEL_LOG" >&2; exit 1; }
echo
echo "Cockpit preview: $PREVIEW_URL"
echo "Read-only frontend fallback; press Ctrl-C to stop the preview and tunnel."
echo "For engine-backed actions, configure Settings → Engine URL to a reachable UIAI Engine (loopback by default; use a protected Tailscale or named HTTPS route for remote users)."
echo
tail -f /dev/null &
wait "$!"
