#!/usr/bin/env bash
# Sibling of Focusa scripts/release.sh — cockpit lane.
# Builds Tauri cockpit (+ engine if requested), writes release-metadata.json + checksums.
set -euo pipefail
cd "$(dirname "$0")/.."
MODE="--cockpit-only"
SKIP_BUILD=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --engine-only) MODE="--engine-only"; shift;;
    --cockpit-only) MODE="--cockpit-only"; shift;;
    --all) MODE="--all"; shift;;
    --skip-build) SKIP_BUILD=1; shift;;
    --dry-run) echo "Dry-run: would build $MODE, write release-metadata.json, checksums"; exit 0;;
    -h|--help) echo "Usage: $0 [--cockpit-only|--engine-only|--all] [--skip-build] [--dry-run]"; exit 0;;
    *) echo "unknown $1" >&2; exit 2;;
  esac
done

if [[ "$SKIP_BUILD" == "0" ]]; then
  if [[ "$MODE" == "--cockpit-only" || "$MODE" == "--all" ]]; then
    echo "Building cockpit frontend..."
    (cd apps/cockpit && npm run build 2>&1 | tail -n 20)
  fi
  if [[ "$MODE" == "--engine-only" || "$MODE" == "--all" ]]; then
    echo "Building engine binary..."
    go build -o /tmp/uiai-engine-build ./cmd/uiai-engine && ls -lh /tmp/uiai-engine-build | awk '{print $9,$5}'
  fi
fi

# Write release-metadata.json (focusa.cockpit.release.v1 style) and checksums
VERSION=$(grep '"version"' apps/cockpit/package.json | head -1 | sed -E 's/.*: "([^"]+)".*/\1/')
META="release-metadata.json"
cat > "$META" <<JSON
{
  "schema": "focusa.cockpit.release.v1",
  "version": "$VERSION",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "source_commit": "$(git rev-parse HEAD 2>/dev/null || echo unknown)",
  "artifacts": []
}
JSON
echo "Wrote $META: $VERSION"
sha256sum "$META" > "$META.sha256" 2>/dev/null || shasum -a 256 "$META" > "$META.sha256"
cat "$META.sha256"
echo "✓ cockpit-release complete ($MODE) — $META + checksums"
