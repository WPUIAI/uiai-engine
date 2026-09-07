#!/usr/bin/env python3
"""Verify UIAI Engine version surfaces are aligned.

Usage:
  scripts/verify-uiai-version-surfaces.py v2.0.1
  scripts/verify-uiai-version-surfaces.py engine-v2.0.1
"""
import json, re, sys
from pathlib import Path
ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^(?:engine-v|uiai-engine-v|cockpit-v|v)?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$")

def parse_version(raw: str) -> str:
    m = VERSION_RE.match(raw.strip())
    if not m:
        raise SystemExit(f"Invalid version/tag: {raw!r}")
    return m.group(1)

def read_json_version(path: str) -> str:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))["version"]

def read_cargo_version(path: str) -> str:
    for line in (ROOT / path).read_text(encoding="utf-8").splitlines():
        if line.strip().startswith("version"):
            return line.split('"')[1]
    raise SystemExit(f"version not found: {path}")

def read_go_version(path: str) -> str:
    text = (ROOT / path).read_text(encoding="utf-8")
    m = re.search(r'version\s*=\s*"([^"]+)"', text)
    if not m:
        raise SystemExit(f"Go version not found: {path}")
    return m.group(1)

def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("Usage: scripts/verify-uiai-version-surfaces.py <tag-or-version>")
    raw = sys.argv[1]
    expected = parse_version(raw)
    is_cockpit = raw.startswith("cockpit-v")
    is_engine = raw.startswith("engine-v") or raw.startswith("uiai-engine-v")
    if is_engine:
        checks = [("cmd/uiai-engine/main.go", read_go_version("cmd/uiai-engine/main.go"))]
    elif is_cockpit:
        checks = [
            ("apps/cockpit/package.json", read_json_version("apps/cockpit/package.json")),
            ("apps/cockpit/src-tauri/tauri.conf.json", read_json_version("apps/cockpit/src-tauri/tauri.conf.json")),
            ("apps/cockpit/src-tauri/Cargo.toml", read_cargo_version("apps/cockpit/src-tauri/Cargo.toml")),
        ]
    else:
        checks = [
            ("cmd/uiai-engine/main.go", read_go_version("cmd/uiai-engine/main.go")),
            ("apps/cockpit/package.json", read_json_version("apps/cockpit/package.json")),
            ("apps/cockpit/src-tauri/tauri.conf.json", read_json_version("apps/cockpit/src-tauri/tauri.conf.json")),
            ("apps/cockpit/src-tauri/Cargo.toml", read_cargo_version("apps/cockpit/src-tauri/Cargo.toml")),
        ]
    mismatches = [(label, actual) for label, actual in checks if actual != expected]
    # The distribution manifest is dual-track. Legacy v1 manifests fall back to release_version.
    mp = ROOT / "docs/distribution-manifest.json"
    if mp.exists():
        manifest = json.loads(mp.read_text(encoding="utf-8"))
        if is_engine:
            manifest_checks = [("engine_version", manifest.get("engine_version", manifest.get("release_version", "")))]
        elif is_cockpit:
            manifest_checks = [("cockpit_version", manifest.get("cockpit_version", manifest.get("release_version", "")))]
        else:
            manifest_checks = [
                ("engine_version", manifest.get("engine_version", manifest.get("release_version", ""))),
                ("cockpit_version", manifest.get("cockpit_version", manifest.get("release_version", ""))),
            ]
        for key, actual in manifest_checks:
            label = f"docs/distribution-manifest.json::{key}"
            checks.append((label, actual))
            if actual != expected:
                mismatches.append((label, actual))
    if mismatches:
        print(f"Version surface mismatch; expected {expected}:", file=sys.stderr)
        for label, actual in mismatches:
            print(f"  - {label}: {actual}", file=sys.stderr)
        return 1
    print(f"All UIAI version surfaces match {expected}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
