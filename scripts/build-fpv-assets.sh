#!/usr/bin/env bash
set -euo pipefail
SRC_DIR=${SRC_DIR:-web/fpv}
OUT_DIR=${OUT_DIR:-web/fpv/dist}
mkdir -p "$OUT_DIR"
for file in fpv.css fpv.js; do
  [ -f "$SRC_DIR/$file" ] || { echo "missing $SRC_DIR/$file" >&2; exit 1; }
  if grep -Eiq "https?://|unpkg|jsdelivr|cdnjs|cdn" "$SRC_DIR/$file"; then
    echo "external runtime dependency found in $SRC_DIR/$file" >&2
    exit 1
  fi
  cp "$SRC_DIR/$file" "$OUT_DIR/$file"
done
if grep -Eiq "https?://.*(unpkg|jsdelivr|cdnjs|cdn)|<script[^>]+https?://" "$SRC_DIR/index.html"; then
  echo "external runtime dependency found in $SRC_DIR/index.html" >&2
  exit 1
fi
sha_css=$(sha256sum "$OUT_DIR/fpv.css" | awk '{print $1}')
sha_js=$(sha256sum "$OUT_DIR/fpv.js" | awk '{print $1}')
cat > "$OUT_DIR/fpv-assets.json" <<JSON
{
  "schema": "uiai.fpv_assets.v1",
  "source_dir": "$SRC_DIR",
  "assets": {
    "fpv.css": { "sha256": "$sha_css" },
    "fpv.js": { "sha256": "$sha_js" }
  },
  "external_runtime_dependencies": false
}
JSON
echo "fpv assets built: $OUT_DIR/fpv.css $OUT_DIR/fpv.js $OUT_DIR/fpv-assets.json"
