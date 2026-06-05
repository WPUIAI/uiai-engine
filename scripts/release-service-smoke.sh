#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
SERVICE="${UIAI_SERVICE_NAME:-uiai-engine.service}"
HEALTH_URL="${UIAI_HEALTH_URL:-$ENGINE_URL/health}"
MODE="live"
SKIP_BUILD=0
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-30}"
usage(){ cat <<'EOF'
UIAI release service smoke bundle

Usage:
  scripts/release-service-smoke.sh [--dry-run|--check-only] [--skip-build]

Modes:
  --dry-run      Print commands/proof plan only; no build, restart, or smokes.
  --check-only   Do not build or restart; wait for current health and run smokes.
  --skip-build   In live mode, restart and smoke without rebuilding binary.

Live mode requires root for systemctl restart. Repo commands run as current user; invoke as wpuiai for build/smokes or use release runbook boundaries.
EOF
}
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) MODE="dry-run"; shift ;;
    --check-only) MODE="check-only"; shift ;;
    --skip-build) SKIP_BUILD=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done
cmds=(
  "cd $ROOT_DIR"
  "go build -o ./uiai-engine ./cmd/uiai-engine"
  "systemctl restart $SERVICE"
  "curl -fsS $HEALTH_URL"
  "scripts/smoke-focusa-packet.sh"
  "scripts/smoke-agent-integrations.sh"
  "scripts/smoke-mcp-tool-routes.sh"
  "scripts/smoke-pi-extension-registration.sh"
  "scripts/check-docs-completeness.py"
  "scripts/check-tool-parity.sh"
)
if [[ "$MODE" == "dry-run" ]]; then
  echo "release smoke dry-run: service=$SERVICE health=$HEALTH_URL root=$ROOT_DIR"
  printf 'plan:\n'
  printf -- '- %s\n' "${cmds[@]}"
  exit 0
fi
cd "$ROOT_DIR"
proof=()
run(){ echo "+ $*"; "$@"; }
if [[ "$MODE" == "live" ]]; then
  if [[ "$EUID" -ne 0 ]]; then
    echo "live mode requires root for systemctl restart; use --check-only for current service smoke" >&2
    exit 4
  fi
  if [[ "$SKIP_BUILD" -eq 0 ]]; then
    ts="$(date +%Y%m%d-%H%M%S)"
    [[ -f ./uiai-engine ]] && cp ./uiai-engine "./uiai-engine.bak.$ts"
    run go build -o ./uiai-engine ./cmd/uiai-engine
    proof+=("build:./uiai-engine")
  fi
  run systemctl restart "$SERVICE"
  proof+=("service:$SERVICE:restarted")
fi
for i in $(seq 1 "$TIMEOUT_SECONDS"); do
  if curl -fsS "$HEALTH_URL" >/tmp/uiai-release-smoke-health.json; then
    proof+=("health:$HEALTH_URL:ok")
    break
  fi
  sleep 1
done
if [[ ! -s /tmp/uiai-release-smoke-health.json ]]; then
  echo "health failed: $HEALTH_URL" >&2
  if command -v systemctl >/dev/null 2>&1; then systemctl status "$SERVICE" --no-pager -l | sed -n '1,80p' >&2 || true; fi
  exit 1
fi
run scripts/smoke-focusa-packet.sh; proof+=("smoke:focusa-packet:ok:/tmp/uiai-focusa-packet-smoke.json")
run scripts/smoke-agent-integrations.sh; proof+=("smoke:agent-integrations:ok")
run scripts/smoke-mcp-tool-routes.sh; proof+=("smoke:mcp-tool-routes:ok")
run scripts/smoke-pi-extension-registration.sh; proof+=("smoke:pi-extension-registration:ok")
run scripts/check-docs-completeness.py >/tmp/uiai-docs-completeness-release-smoke.json; proof+=("smoke:docs-completeness:ok:/tmp/uiai-docs-completeness-release-smoke.json")
run scripts/check-tool-parity.sh >/tmp/uiai-tool-parity-release-smoke.json; proof+=("smoke:tool-parity:ok:/tmp/uiai-tool-parity-release-smoke.json")
sha="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo "release smoke ok: mode=$MODE service=$SERVICE git=$sha health=$HEALTH_URL"
printf 'proof:\n'
printf -- '- %s\n' "${proof[@]}"
