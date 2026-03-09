#!/usr/bin/env bash
set -euo pipefail

# Sync approved device frame assets from upstream GitHub repos
# into UIAI internal media store. Manual run; safe for CI later.

ROOT="/home/wpuiai/uiai-engine"
TMP="$(mktemp -d)"
OUT="$ROOT/internal/media/deviceframes/vendor"
MANIFEST="$ROOT/internal/media/deviceframes/manifest.json"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$OUT"

echo "[1/5] clone upstream repos"

git clone --depth=1 https://github.com/ephread/PommePlate "$TMP/pommeplate"
git clone --depth=1 https://github.com/neogeek/ios-device-svg-templates "$TMP/ios-device-svg-templates"

echo "[2/5] capture commit refs"
POMME_REF="$(git -C "$TMP/pommeplate" rev-parse HEAD)"
IOS_REF="$(git -C "$TMP/ios-device-svg-templates" rev-parse HEAD)"

echo "[3/5] copy approved assets"
mkdir -p "$OUT/pommeplate/$POMME_REF" "$OUT/ios-device-svg-templates/$IOS_REF"

# NOTE: tighten globs after final frame shortlist
find "$TMP/pommeplate" -type f \( -name '*.svg' -o -name 'LICENSE*' \) -maxdepth 3 -print0 |
  xargs -0 -I{} cp "{}" "$OUT/pommeplate/$POMME_REF/" || true

find "$TMP/ios-device-svg-templates" -type f \( -name '*.svg' -o -name 'LICENSE*' \) -maxdepth 3 -print0 |
  xargs -0 -I{} cp "{}" "$OUT/ios-device-svg-templates/$IOS_REF/" || true

echo "[4/5] write manifest skeleton"
mkdir -p "$(dirname "$MANIFEST")"
cat > "$MANIFEST" <<JSON
{
  "version": 1,
  "generated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "sources": [
    {
      "name": "pommeplate",
      "repo": "https://github.com/ephread/PommePlate",
      "commit": "$POMME_REF",
      "license_status": "REVIEW_REQUIRED"
    },
    {
      "name": "ios-device-svg-templates",
      "repo": "https://github.com/neogeek/ios-device-svg-templates",
      "commit": "$IOS_REF",
      "license_status": "REVIEW_REQUIRED"
    }
  ],
  "frames": []
}
JSON

echo "[5/5] done"
echo "Vendor assets: $OUT"
echo "Manifest: $MANIFEST"
echo "Next: curate frame list + safe_area metadata + PNG layer generation"
