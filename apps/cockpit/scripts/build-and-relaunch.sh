#!/usr/bin/env bash
# Build Cockpit and relaunch the native browser UI from this terminal.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

npm run check
npm run build
npm exec tauri -- build --no-bundle

pkill -x uaiengine-cockpit 2>/dev/null || true
nohup "$ROOT/src-tauri/target/release/uaiengine-cockpit" >/tmp/uiai-cockpit-gui.log 2>&1 &
echo "Cockpit relaunched from terminal: $!"
