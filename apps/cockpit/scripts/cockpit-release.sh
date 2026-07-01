#!/usr/bin/env bash
# cockpit-release.sh — invoked by deploy-cockpit.yml on the self-hosted
# macOS runner. Stamps version, builds the web bundle, runs Tauri build,
# and emits release-metadata.json + checksums into release-proof/cockpit/.

set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APPS_COCKPIT="${REPO_ROOT}/apps/cockpit"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STAMP="$SCRIPT_DIR/stamp-cockpit-version/stamp-cockpit-version"
TAG="${TAG:-cockpit-v0.1.0}"

cd "${APPS_COCKPIT}"

VERSION="${TAG#cockpit-v}"
VERSION="${VERSION%-dev}"

echo "== stamping ${VERSION} =="
"$STAMP" "$VERSION"

echo "== npm ci =="
npm ci

echo "== typecheck =="
npm run check

echo "== web build =="
npm run build

echo "== tauri build =="
npm run tauri build -- --bundles app

mkdir -p release-proof/cockpit
METADATA="release-proof/cockpit/${TAG}-metadata.json"
CHECKSUMS="release-proof/cockpit/${TAG}-checksums.txt"
: > "$CHECKSUMS"
APP_PATH="$(find src-tauri/target/release/bundle/macos -name '*.app' | head -1 || true)"
DMG_PATH="$(find src-tauri/target/release/bundle/dmg -name '*.dmg' 2>/dev/null | head -1 || true)"

ARTIFACTS_JSON="[]"
if [ -n "$APP_PATH" ]; then
  APP_SHA=$(shasum -a 256 "$APP_PATH" | awk '{print $1}')
  ARTIFACTS_JSON=$(echo "$ARTIFACTS_JSON" | jq --arg n "$(basename "$APP_PATH")" --arg s "$APP_SHA" '. + [{name:$n, platform:"macos-aarch64", sha256:$s}]')
  (cd "$(dirname "$APP_PATH")" && shasum -a 256 "$(basename "$APP_PATH")") >> "$CHECKSUMS"
fi
if [ -n "$DMG_PATH" ]; then
  DMG_SHA=$(shasum -a 256 "$DMG_PATH" | awk '{print $1}')
  ARTIFACTS_JSON=$(echo "$ARTIFACTS_JSON" | jq --arg n "$(basename "$DMG_PATH")" --arg s "$DMG_SHA" '. + [{name:$n, platform:"macos-aarch64", sha256:$s}]')
  (cd "$(dirname "$DMG_PATH")" && shasum -a 256 "$(basename "$DMG_PATH")") >> "$CHECKSUMS"
fi

CHANNEL="stable"
[[ "$VERSION" == *-dev ]] && CHANNEL="dev"
[[ "$VERSION" == *-preview ]] && CHANNEL="preview"

echo "$ARTIFACTS_JSON" | jq --arg v "$VERSION" --arg c "$CHANNEL" --arg t "$TAG" '{
  schema: "focusa.cockpit.release.v1",
  app: "focusa-cockpit",
  version: $v,
  channel: $c,
  tag: $t,
  signed: false,
  notarized: false,
  artifacts: .
}' > "$METADATA"

echo "wrote $METADATA + $CHECKSUMS"