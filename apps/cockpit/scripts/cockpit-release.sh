#!/usr/bin/env bash
# cockpit-release.sh — invoked by deploy-cockpit.yml on the self-hosted
# macOS runner. Stamps version, builds the web bundle, runs Tauri build,
# and emits release-metadata.json + checksums into release-proof/cockpit/.

set -euo pipefail
SELF="$(readlink -f "$0")"
SCRIPT_DIR="$(cd "$(dirname "$SELF")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
APPS_COCKPIT="${REPO_ROOT}/apps/cockpit"
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
: "${TAURI_SIGNING_PRIVATE_KEY:?TAURI_SIGNING_PRIVATE_KEY must be configured for updater releases}"
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="${TAURI_SIGNING_PRIVATE_KEY_PASSWORD:-}"
TARGET_TRIPLE="${TARGET_TRIPLE:-$(rustc -vV | awk '/^host:/ {print $2}')}"
npm run tauri build -- --target "$TARGET_TRIPLE"

BUNDLE_DIR="src-tauri/target/${TARGET_TRIPLE}/release/bundle"
[ -d "$BUNDLE_DIR" ] || BUNDLE_DIR="src-tauri/target/release/bundle"
test -d "$BUNDLE_DIR" || { echo "missing bundle directory for $TARGET_TRIPLE" >&2; exit 1; }
mkdir -p release-proof/cockpit
METADATA="release-proof/cockpit/${TAG}-metadata.json"
CHECKSUMS="release-proof/cockpit/${TAG}-checksums.txt"
: > "$CHECKSUMS"
APP_PATH="$(find "$BUNDLE_DIR/macos" -name '*.app' | head -1 || true)"
DMG_PATH="$(find "$BUNDLE_DIR/dmg" -name '*.dmg' 2>/dev/null | head -1 || true)"
UPDATER_TARBALL="$(find "$BUNDLE_DIR" \( -name '*.app.tar.gz' -o -name '*.AppImage.tar.gz' \) | head -1 || true)"
UPDATER_SIG="${UPDATER_TARBALL}.sig"
if [ -z "$UPDATER_TARBALL" ] || [ ! -f "$UPDATER_SIG" ]; then
  echo "missing signed updater artifact" >&2
  exit 1
fi

PLATFORM="darwin-x86_64"
[[ "$TARGET_TRIPLE" == aarch64-* ]] && PLATFORM="darwin-aarch64"
ARTIFACTS_JSON="[]"
if [ -n "$APP_PATH" ]; then
  APP_SHA=$(find "$APP_PATH" -type f -print0 | xargs -0 shasum -a 256 | sort | shasum -a 256 | awk '{print $1}')
  ARTIFACTS_JSON=$(echo "$ARTIFACTS_JSON" | jq --arg n "$(basename "$APP_PATH")" --arg s "$APP_SHA" --arg p "$PLATFORM" '. + [{name:$n, platform:$p, sha256:$s}]')
  printf '%s  %s\n' "$APP_SHA" "$(basename "$APP_PATH")" >> "$CHECKSUMS"
fi
if [ -n "$DMG_PATH" ]; then
  DMG_SHA=$(shasum -a 256 "$DMG_PATH" | awk '{print $1}')
  ARTIFACTS_JSON=$(echo "$ARTIFACTS_JSON" | jq --arg n "$(basename "$DMG_PATH")" --arg s "$DMG_SHA" --arg p "$PLATFORM" '. + [{name:$n, platform:$p, sha256:$s}]')
  (cd "$(dirname "$DMG_PATH")" && shasum -a 256 "$(basename "$DMG_PATH")") >> "$CHECKSUMS"
fi

CHANNEL="stable"
[[ "$VERSION" == *-dev ]] && CHANNEL="dev"
[[ "$VERSION" == *-preview ]] && CHANNEL="preview"

echo "$ARTIFACTS_JSON" | jq --arg v "$VERSION" --arg c "$CHANNEL" --arg t "$TAG" --argjson notarized "${NOTARIZED:-false}" '{
  schema: "uaiengine.cockpit.release.v1",
  app: "uaiengine-cockpit",
  version: $v,
  channel: $c,
  tag: $t,
  signed: true,
  notarized: $notarized,
  artifacts: .
}' > "$METADATA"

UPDATER_DIR="release-proof/cockpit/updater"
mkdir -p "$UPDATER_DIR"
cp "$UPDATER_TARBALL" "$UPDATER_DIR/"
cp "$UPDATER_SIG" "$UPDATER_DIR/"
SIG=$(cat "$UPDATER_SIG")
if [ -n "${UPDATER_ASSET_BASE_URL:-}" ]; then
  URL="${UPDATER_ASSET_BASE_URL%/}/$(basename "$UPDATER_TARBALL")"
else
  URL="file://$(pwd)/$UPDATER_DIR/$(basename "$UPDATER_TARBALL")"
fi
jq -n --arg v "$VERSION" --arg n "UIAI Engine Cockpit $VERSION" --arg s "$SIG" --arg u "$URL" --arg p "$PLATFORM" '{version:$v,notes:$n,pub_date:(now|todate),platforms:{($p):{signature:$s,url:$u}}}' > "${UPDATER_DIR}/latest.json"

echo "wrote $METADATA + $CHECKSUMS + ${UPDATER_DIR}/latest.json"