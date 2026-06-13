#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXT="$ROOT_DIR/.pi/extensions/uiai-fpv-steer.ts"
[ -f "$EXT" ] || { echo "missing $EXT" >&2; exit 1; }
python3 - <<'PY' "$EXT"
from pathlib import Path
import sys
src = Path(sys.argv[1]).read_text()
required = [
    'pi.sendUserMessage',
    'deliverAs: "steer"',
    'pi.registerCommand("fpv-watch"',
    'pi.registerCommand("fpv-unwatch"',
    'pi.registerCommand("fpv-watch-status"',
    'pi.on("session_start"',
    '/api/fpv/events?since_seq=',
    'deliverAs: "steer"',
    '/m/${encodeURIComponent(token)}/status',
]
missing = [item for item in required if item not in src]
if missing:
    raise SystemExit(f"missing FPV steer extension patterns: {missing}")
print("pi fpv steer extension smoke ok")
PY
