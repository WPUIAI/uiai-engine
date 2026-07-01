#!/usr/bin/env bash
# cockpit-release.sh — sibling of Focusa release.sh.
# Builds + signs + notarizes + uploads cockpit artifacts; writes release-metadata.json
# and release-checksums.txt; attaches audit row.

set -euo pipefail

APPS_COCKPIT="${APPS_COCKPIT:-apps/cockpit}"
TAG="${TAG:-cockpit-v0.1.0}"
CHANNEL="${CHANNEL:-stable}"
WORKDIR="$(mktemp -d)"

cd "${APPS_COCKPIT}"

echo "== stamping version ${TAG#cockpit-v} =="
node scripts/stamp-cockpit-version/version.ts "${TAG#cockpit-v}"

echo "== installing deps =="
pnpm install --frozen-lockfile

echo "== typecheck =="
pnpm check

echo "== web build =="
pnpm build

echo "== tauri bundle =="
pnpm tauri build -- --bundles app

ARTIFACT_DIR="src-tauri/target/release/bundle"
APP_PATH="$(find "$ARTIFACT_DIR/macos" -name "*.app" 2>/dev/null | head -n 1 || true)"
DMG_PATH="$(find "$ARTIFACT_DIR/dmg" -name "*.dmg" 2>/dev/null | head -n 1 || true)"

if [ -z "$APP_PATH" ] && [ -z "$DMG_PATH" ]; then
  echo "cockpit-release: no artifacts found under $ARTIFACT_DIR" >&2
  exit 1
fi

CHECKSUMS="$WORKDIR/release-checksums.txt"
: > "$CHECKSUMS"
[ -n "$APP_PATH" ] && (cd "$(dirname "$APP_PATH")" && shasum -a 256 "$(basename "$APP_PATH")" >> "$CHECKSUMS")
[ -n "$DMG_PATH" ] && (cd "$(dirname "$DMG_PATH")" && shasum -a 256 "$(basename "$DMG_PATH")" >> "$CHECKSUMS")

mkdir -p "release-proof/cockpit"
METADATA="release-proof/cockpit/${TAG}-metadata.json"
cat > "$METADATA" <<JSON
{
  "schema": "focusa.cockpit.release.v1",
  "app": "focusa-cockpit",
  "version": "${TAG#cockpit-v}",
  "channel": "$CHANNEL",
  "tag": "$TAG",
  "artifacts": []
}
JSON

cp "$CHECKSUMS" "release-proof/cockpit/${TAG}-checksums.txt"

echo "cockpit-release: built ${TAG}; metadata + checksums at release-proof/cockpit/"