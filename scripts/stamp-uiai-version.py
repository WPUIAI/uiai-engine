#!/usr/bin/env python3
"""Stamp all UIAI Engine release-version surfaces from one release tag.

Dual-track aware: handles engine-v* and cockpit-v* tags, or bare version.
Single writer for UIAI version surfaces + distribution-manifest.

Usage:
  scripts/stamp-uiai-version.py v2.0.1
  scripts/stamp-uiai-version.py engine-v2.0.1
  scripts/stamp-uiai-version.py cockpit-v0.1.5
  scripts/stamp-uiai-version.py 2.0.1 --cockpit-only
"""
from __future__ import annotations
import json
import re
import sys
import subprocess
import hashlib
import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^(?:engine-v|uiai-engine-v|cockpit-v|v)?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$")

def parse_version(raw: str) -> str:
    m = VERSION_RE.match(raw.strip())
    if not m:
        raise SystemExit(f"Invalid UIAI release tag/version: {raw!r} (expected [engine-|cockpit-|v]X.Y.Z[-dev])")
    return m.group(1)

def replace_json_version(path: str, version: str):
    p = ROOT / path
    data = json.loads(p.read_text(encoding="utf-8"))
    data["version"] = version
    if path.endswith("package-lock.json"):
        pkgs = data.get("packages")
        if isinstance(pkgs, dict) and "" in pkgs and isinstance(pkgs[""], dict):
            pkgs[""]["version"] = version
    p.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

def replace_cargo_version(path: str, version: str):
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    nxt, cnt = re.subn(r'(?m)^version\s*=\s*"[^"]+"', f'version = "{version}"', text, count=1)
    if cnt != 1:
        raise SystemExit(f"Expected one version in {path}")
    p.write_text(nxt, encoding="utf-8")

def replace_go_version(path: str, version: str):
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    # matches: version   = "2.0.0-dev"
    nxt, cnt = re.subn(r'version\s*=\s*"[^"]+"', f'version   = "{version}"', text, count=1)
    if cnt != 1:
        raise SystemExit(f"Expected one Go version var in {path}")
    p.write_text(nxt, encoding="utf-8")

def regen_manifest(version: str):
    """Regenerate docs/distribution-manifest.json — single writer, never hand-edited."""
    mp = ROOT / "docs/distribution-manifest.json"
    if not mp.exists():
        # create minimal manifest if missing
        data = {
            "schema": "uiai.distribution_manifest.v1",
            "release_version": version,
            "artifacts": {},
            "compatibility_status": "compatible",
        }
    else:
        data = json.loads(mp.read_text(encoding="utf-8"))
        data["release_version"] = version
        # Recompute sha256 for every listed artifact — fail closed if missing (mirror Core strictness).
        # No stale preservation: every artifact must exist at stamp time; CI will verify.
        new_artifacts = {}
        for rel, old_hash in list(data.get("artifacts", {}).items()):
            ap = ROOT / rel
            if not ap.exists() or not ap.is_file():
                raise SystemExit(f"Missing artifact for manifest (fail-closed): {rel}")
            digest = "sha256:" + hashlib.sha256(ap.read_bytes()).hexdigest()
            new_artifacts[rel] = digest
        data["artifacts"] = new_artifacts
        data.setdefault("schema", "uiai.distribution_manifest.v1")
        data.setdefault("compatibility_status", "compatible")
    head = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"], cwd=str(ROOT)).decode().strip()
    data["source_commit"] = head
    data["generated_at"] = datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")
    mp.parent.mkdir(parents=True, exist_ok=True)
    mp.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

def main() -> int:
    if len(sys.argv) < 2:
        raise SystemExit("Usage: scripts/stamp-uiai-version.py <tag-or-version> [--cockpit-only|--engine-only]")
    raw = sys.argv[1]
    mode = sys.argv[2] if len(sys.argv) > 2 else "auto"
    version = parse_version(raw)

    # auto-detect track from tag prefix when mode=auto
    if mode == "auto":
        if raw.startswith("cockpit-v"):
            mode = "--cockpit-only"
        elif raw.startswith("engine-v") or raw.startswith("uiai-engine-v"):
            mode = "--engine-only"
        else:
            mode = "all"
    engine_only = mode == "--engine-only"
    cockpit_only = mode == "--cockpit-only"

    if not cockpit_only:
        # Go engine surface
        replace_go_version("cmd/uiai-engine/main.go", version)
    if not engine_only:
        # Cockpit surfaces
        replace_json_version("apps/cockpit/package.json", version)
        replace_json_version("apps/cockpit/src-tauri/tauri.conf.json", version)
        replace_cargo_version("apps/cockpit/src-tauri/Cargo.toml", version)
        # Cargo.lock cockpit entry — name uaiengine-cockpit
        try:
            # update lock entry similar to focusa
            lp = ROOT / "apps/cockpit/src-tauri/Cargo.lock"
            if lp.exists():
                lines = lp.read_text(encoding="utf-8").splitlines(keepends=True)
                cur = None
                out = []
                updated = False
                for line in lines:
                    if line.strip() == "[[package]]":
                        cur = None
                    m = re.match(r'^name\s*=\s*"([^"]+)"', line)
                    if m:
                        cur = m.group(1)
                    if cur == "uaiengine-cockpit" and re.match(r'^version\s*=\s*"[^"]+"', line) and not updated:
                        out.append(f'version = "{version}"\n')
                        updated = True
                        continue
                    out.append(line)
                if updated:
                    lp.write_text("".join(out), encoding="utf-8")
        except Exception as e:
            print(f"warning: Cargo.lock update skipped: {e}", file=sys.stderr)

    # stamp file for preflight
    stamp = ROOT / "docs/current/.release-version-stamp"
    if not cockpit_only and not engine_only:
        stamp.parent.mkdir(parents=True, exist_ok=True)
        stamp.write_text(version + "\n", encoding="utf-8")
    elif cockpit_only:
        # cockpit stamp
        cp = ROOT / "apps/cockpit/.release-version-stamp"
        cp.write_text(version + "\n", encoding="utf-8")
    else:
        stamp.parent.mkdir(parents=True, exist_ok=True)
        stamp.write_text(version + "\n", encoding="utf-8")

    regen_manifest(version)
    print(f"Stamped UIAI version {version} (mode={mode})")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
