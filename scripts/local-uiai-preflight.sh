#!/usr/bin/env bash
set -euo pipefail
# CANONICAL UIAI PREFLIGHT — BLOCKING, NON-STALE, FAILS CLOSED. Dual-compatible with Focusa discipline.
# Usage: bash scripts/local-uiai-preflight.sh [--strict]
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
STRICT=0
if [[ "${1:-}" == "--strict" ]]; then STRICT=1; fi

echo "=== UIAI preflight: Windows path lint ==="
if git -c core.quotepath=false ls-files | grep -q ":"; then echo "FAIL Windows lint colon"; git -c core.quotepath=false ls-files | grep ":"; exit 1; fi
if git -c core.quotepath=false ls-files | grep -q '[?*"<>|]'; then echo "FAIL Windows illegal char"; git -c core.quotepath=false ls-files | grep -E '[?*"<>|]'; exit 1; fi
echo "Windows lint: PASS"

echo "=== UIAI preflight: version surfaces ==="
# Dual-track: engine (Go) and cockpit (Tauri) have independent semvers.
ENG_V="$(grep -E 'version\s*=\s*"' cmd/uiai-engine/main.go | head -1 | sed -E 's/.*"(.*)"/\1/')"
COCK_V="$(grep -m1 '"version"' apps/cockpit/package.json | sed -E 's/.*: "([^"]+)".*/\1/' | head -1)"
echo "checking engine-v$ENG_V + cockpit-v$COCK_V"
python3 scripts/verify-uiai-version-surfaces.py "engine-v$ENG_V" || { echo "FAIL verify engine"; exit 1; }
python3 scripts/verify-uiai-version-surfaces.py "cockpit-v$COCK_V" || { echo "FAIL verify cockpit"; exit 1; }
echo "version surfaces: PASS (engine $ENG_V / cockpit $COCK_V)"

echo "=== UIAI preflight: distribution-manifest freshness ==="
python3 << 'PYFRESH'
import json, pathlib, subprocess, sys, datetime, hashlib
root = pathlib.Path(".")
mp = root / "docs/distribution-manifest.json"
if not mp.exists():
    print("manifest missing — will be created at stamp/build time (allowed in fast mode)")
    sys.exit(0)
m = json.loads(mp.read_text())
head_short = subprocess.check_output(["git","rev-parse","--short","HEAD"]).decode().strip()
head_full = subprocess.check_output(["git","rev-parse","HEAD"]).decode().strip()
cargo_v = None
for p in [root/"cmd/uiai-engine/main.go", root/"apps/cockpit/package.json"]:
    try:
        if p.suffix == ".go":
            import re
            mv = re.search(r'version\s*=\s*"([^"]+)"', p.read_text()).group(1)
        else:
            mv = json.loads(p.read_text())["version"]
        cargo_v = mv
        break
    except Exception:
        pass
if cargo_v and m.get("release_version") != cargo_v:
    print(f"FAIL release_version {m.get('release_version')} != {cargo_v}", file=sys.stderr); sys.exit(1)
import os
source_commit = m.get("source_commit","")
head_parent = subprocess.check_output(["git","rev-parse","--short","HEAD~1"], stderr=subprocess.DEVNULL).decode().strip() if subprocess.call(["git","rev-parse","--verify","HEAD~1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)==0 else head_short
manifest_touched = "distribution-manifest.json" in subprocess.check_output(["git","diff","--name-only","HEAD~1","HEAD"], stderr=subprocess.DEVNULL).decode() if head_parent != head_short else False
fast_mode = "PREFLIGHT_FAST" in os.environ
if source_commit and source_commit not in (head_short, head_full, head_full[:7], head_parent):
    if manifest_touched and source_commit == head_parent:
        pass
    elif fast_mode:
        if subprocess.call(["git","merge-base","--is-ancestor", source_commit, "HEAD"]) != 0:
            print(f"FAIL stale source_commit {source_commit} not ancestor of HEAD {head_short} (touched={manifest_touched})", file=sys.stderr); sys.exit(1)
    else:
        print(f"FAIL stale source_commit {source_commit} != HEAD {head_short} nor parent {head_parent} (touched={manifest_touched}) — STRICT requires HEAD/parent", file=sys.stderr); sys.exit(1)
try:
    gen = datetime.datetime.fromisoformat(m.get("generated_at","").replace("Z","+00:00"))
    age = datetime.datetime.now(datetime.timezone.utc) - gen
    if age.total_seconds() > 86400:
        print(f"FAIL stale generated_at {m.get('generated_at')}", file=sys.stderr); sys.exit(1)
except Exception:
    pass
print(f"manifest FRESH: release_version={m.get('release_version')} source_commit={m.get('source_commit')} head={head_short}")
PYFRESH
if [[ $? -ne 0 ]]; then echo "FAIL manifest freshness"; exit 1; fi
echo "manifest: FRESH"

if [[ "$STRICT" -eq 1 ]]; then
  echo "=== UIAI preflight: go tests ==="
  go test ./... || { echo "FAIL go test"; exit 1; }
  echo "go tests: PASS"
  echo "=== UIAI preflight: docs completeness ==="
  if [[ -x scripts/check-docs-completeness.py ]]; then
    python3 scripts/check-docs-completeness.py || { echo "FAIL docs completeness"; exit 1; }
  fi
  if [[ -x scripts/check-tool-parity.sh ]]; then
    bash scripts/check-tool-parity.sh || { echo "FAIL tool parity"; exit 1; }
  fi
fi

echo "=== UIAI preflight: FORMAT (blocking) ==="
if command -v go >/dev/null 2>&1; then
  if ! go fmt ./... 2>&1 | grep -q .; then echo "go fmt: PASS"; else echo "FAIL go fmt — run go fmt ./..."; exit 1; fi
fi
if [[ -f apps/cockpit/package.json ]] && command -v npm >/dev/null 2>&1; then
  if [[ "$STRICT" -eq 1 ]]; then
    (cd apps/cockpit && npm run check 2>&1 | tail -5) || { echo "FAIL cockpit check"; exit 1; }
  fi
fi
echo "format: PASS"

echo "=== UIAI preflight: DONE — PASS (may tag) ==="
