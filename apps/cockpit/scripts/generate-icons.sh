#!/usr/bin/env bash
# Generate UIAI Engine Cockpit icons from a minimal solid-color PNG.
# Produces icons/32x32.png, 128x128.png, icon.icns, icon.ico via the
# @tauri-apps/cli icon command. The source PNG is written in pure
# Python stdlib so no PIL or imagemagick is required on the runner.
set -euo pipefail

SELF="$(readlink -f "$0")"
SCRIPT_DIR="$(cd "$(dirname "$SELF")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ICON_DIR="$REPO_ROOT/apps/cockpit/src-tauri/icons"
SRC_PNG="$ICON_DIR/app-icon.png"

mkdir -p "$ICON_DIR"

python3 - "$SRC_PNG" <<'PY'
import struct, sys, zlib
out = sys.argv[1]
def chunk(t, d):
    return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
ihdr = struct.pack(">IIBBBBB", 512, 512, 8, 2, 0, 0, 0)
row = b"\x00" + bytes([0x33, 0x66, 0x99]) * 512
raw = row * 512
idat = zlib.compress(raw, 9)
png = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", ihdr) + chunk(b"IDAT", idat) + chunk(b"IEND", b"")
open(out, "wb").write(png)
print(f"wrote {out} ({len(png)} bytes)")
PY

cd "$REPO_ROOT/apps/cockpit"
npx --yes @tauri-apps/cli@latest icon "$SRC_PNG" --output "$ICON_DIR"
echo "[generate-icons] done; icons in $ICON_DIR"
ls "$ICON_DIR"