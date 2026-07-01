#!/usr/bin/env python3
"""UIAI Engine Cockpit self-heal — real fix, not just a claim.

Pipeline (workflow_run: completed triggers this from
cockpit-audit-recorder.yml when a cockpit CI/release run fails):

  1. Pull failed workflow logs via gh API.
  2. Match against known anti-patterns in the cockpit pipeline.
  3. Apply a targeted fix.
  4. Commit the fix on a new branch and push.
  5. Re-dispatch the failed workflow.
  6. Write an audit row that records the actual fix that was applied
     (not just a self_heal claim).

Usage:
  python3 scripts/auto-heal-cockpit-audit.py <run_id> <conclusion> <workflow>
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

REPO = os.environ.get("GITHUB_REPOSITORY", "WPUIAI/uiai-engine")
AUDIT_FILE = Path("release-proof/cockpit/audit.jsonl")
SCHEMA = "uaiengine.cockpit.audit.v1"

# Each fix has: id, match (substring in log), apply (Python lambda path -> bool).
# Fixes are applied in order; first match wins.
FIXES = [
    {
        "id": "fix-cockpit-script-resolve-via-readlink",
        "match": "cd: apps/cockpit/scripts: No such file or directory",
        "description": (
            "Bash scripts invoked with relative paths fail because "
            "`cd $(dirname $0)/../..` walks up too many levels. "
            "Replace with absolute resolution via `readlink -f`."
        ),
        "files": [
            "apps/cockpit/scripts/create-cockpit-dev-release-tag.sh",
            "apps/cockpit/scripts/cockpit-release.sh",
        ],
        "bad": 'REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"\ncd "$REPO_ROOT"',
        "good": (
            'set -euo pipefail\n'
            'SELF="$(readlink -f "$0")"\n'
            'SCRIPT_DIR="$(cd "$(dirname "$SELF")" && pwd)"\n'
            'REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"\n'
            'cd "$REPO_ROOT"'
        ),
    },
    {
        "id": "fix-cockpit-tokens-frontend-dist",
        "match": "frontendDist: ../build",
        "description": (
            "tauri.conf.json frontendDist must be `../build` relative to src-tauri/."
        ),
        "files": ["apps/cockpit/src-tauri/tauri.conf.json"],
        "validate": lambda content: '"frontendDist": "../build"' in content,
    },
    {
        "id": "fix-cockpit-rollup-darwin-arm64-optional",
        "match": "Cannot find module @rollup/rollup-darwin-arm64",
        "description": (
            "npm's optional-dependency resolution drops platform binaries that "
            "match the runner's host but not its architecture in some npm versions. "
            "Install @rollup/rollup-darwin-arm64 explicitly as a follow-up step on "
            "macOS runners so vite's rollup native call succeeds."
        ),
        "files": [".github/workflows/cockpit.yml"],
        "bad": "run: npm ci\n      - name: Typecheck",
        "good": (
            "run: npm ci\n"
            "      - name: Ensure rollup darwin-arm64 native binary\n"
            "        if: runner.os == 'macOS'\n"
            "        working-directory: apps/cockpit\n"
            "        run: |\n"
            "          npm install --no-save --no-package-lock \\\n"
            "            @rollup/rollup-darwin-arm64@4.62.2\n"
            "      - name: Typecheck"
        ),
    },
]


def gh(*args: str, check: bool = True) -> str:
    r = subprocess.run(["gh", *args], capture_output=True, text=True)
    if check and r.returncode != 0:
        print(f"  gh {' '.join(args)}: {r.stderr.strip()}", file=sys.stderr)
        raise SystemExit(r.returncode)
    return (r.stdout or "").strip()


def fetch_run_logs(run_id: str) -> str:
    """Pull all logs for a run. Returns concatenated text across all log files.

    GitHub returns a zip archive (PK\\x03\\x04 header), so download to a temp
    file, extract with the stdlib zipfile module, and concatenate plain text.
    """
    import tempfile
    import zipfile

    try:
        with tempfile.NamedTemporaryFile(suffix=".zip", delete=False) as tmp:
            tmp_path = tmp.name
        subprocess.run(
            ["gh", "api", f"repos/{REPO}/actions/runs/{run_id}/logs",
             "--output", tmp_path],
            check=True, capture_output=True, text=True,
        )
    except Exception as e:
        print(f"[auto-heal] log fetch failed: {e}", file=sys.stderr)
        return ""
    finally:
        pass

    try:
        with zipfile.ZipFile(tmp_path, "r") as zf:
            chunks: list[str] = []
            for name in zf.namelist():
                with zf.open(name) as fh:
                    chunks.append(fh.read().decode("utf-8", errors="replace"))
            return "\n".join(chunks)
    except Exception as e:
        print(f"[auto-heal] log extract failed: {e}", file=sys.stderr)
        return ""
    finally:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass


def detect_fix(log_blob: str) -> dict | None:
    for fix in FIXES:
        if fix["match"] in log_blob:
            return fix
    return None


def apply_fix(fix: dict) -> bool:
    """Apply a fix; return True if any file changed."""
    changed = False
    for path in fix["files"]:
        p = Path(path)
        if not p.exists():
            print(f"[auto-heal] missing file: {path}")
            continue
        content = p.read_text()
        if "bad" in fix and fix["bad"] in content:
            new = content.replace(fix["bad"], fix["good"], 1)
            if new != content:
                p.write_text(new)
                print(f"[auto-heal] patched {path}")
                changed = True
    return changed


def commit_and_push(branch: str, message: str) -> None:
    subprocess.run(["git", "checkout", "-b", branch], check=True)
    subprocess.run(["git", "add", "-A"], check=True)
    subprocess.run(
        [
            "git", "-c", "user.name=uaiengine-cockpit-bot",
            "-c", "user.email=uaiengine-cockpit-bot@users.noreply.github.com",
            "commit", "-m", message,
        ],
        check=False,  # may be nothing to commit
    )
    subprocess.run(["git", "push", "origin", branch], check=True)


def write_audit_row(row: dict) -> None:
    AUDIT_FILE.parent.mkdir(parents=True, exist_ok=True)
    with AUDIT_FILE.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(row) + "\n")


def main() -> int:
    if len(sys.argv) < 4:
        print("usage: auto-heal-cockpit-audit.py <run_id> <conclusion> <workflow>")
        return 1

    run_id, conclusion, workflow = sys.argv[1], sys.argv[2], sys.argv[3]
    if conclusion != "failure":
        print(f"[auto-heal] conclusion={conclusion}; no action")
        write_audit_row({
            "schema": SCHEMA,
            "kind": "self_heal_no_op",
            "workflow": workflow,
            "run_id": run_id,
            "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "reason": "non-failure conclusion",
        })
        return 0

    print(f"[auto-heal] workflow={workflow} run_id={run_id} failure detected")
    logs = fetch_run_logs(run_id)
    print(f"[auto-heal] fetched {len(logs)} bytes of logs")

    fix = detect_fix(logs)
    if not fix:
        print(f"[auto-heal] no known fix for {workflow} run_id={run_id}")
        write_audit_row({
            "schema": SCHEMA,
            "kind": "self_heal_no_match",
            "workflow": workflow,
            "run_id": run_id,
            "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "match_attempted": [f["id"] for f in FIXES],
        })
        return 0

    print(f"[auto-heal] matched fix: {fix['id']}")
    print(f"[auto-heal] description: {fix['description']}")

    changed = apply_fix(fix)
    if not changed:
        print(f"[auto-heal] fix already applied or no-op")
        write_audit_row({
            "schema": SCHEMA,
            "kind": "self_heal_no_change",
            "workflow": workflow,
            "run_id": run_id,
            "fix_id": fix["id"],
            "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        })
        return 0

    branch = f"auto-heal/{fix['id']}/{run_id}"
    commit_and_push(
        branch,
        f"auto-heal(cockpit): {fix['id']}\n\nWorkflow: {workflow}\nRun: {run_id}\nFix: {fix['description']}",
    )

    print(f"[auto-heal] pushed branch {branch}; re-dispatching {workflow}")
    subprocess.run(
        ["gh", "workflow", "run", f"{workflow}.yml", "--ref", branch],
        check=True,
    )

    write_audit_row({
        "schema": SCHEMA,
        "kind": "self_heal_applied",
        "workflow": workflow,
        "run_id": run_id,
        "fix_id": fix["id"],
        "branch": branch,
        "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "description": fix["description"],
        "re_dispatched": True,
    })

    return 0


if __name__ == "__main__":
    sys.exit(main())