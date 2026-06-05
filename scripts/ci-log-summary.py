#!/usr/bin/env python3
"""Summarize failed UIAI GitHub Actions logs/artifacts with bounded, redacted output."""
from __future__ import annotations
import argparse, json, os, re, shutil, subprocess, sys
from pathlib import Path

SECRET_PATTERNS = [
    re.compile(r'(?i)(authorization:\s*bearer\s+)[^\s]+'),
    re.compile(r'(?i)(x-api-key\s*[:=]\s*)[^\s]+'),
    re.compile(r'(?i)(api[_-]?key\s*[:=]\s*)[^\s]+'),
    re.compile(r'(?i)(token\s*[:=]\s*)[^\s]+'),
    re.compile(r'(?i)(secret\s*[:=]\s*)[^\s]+'),
    re.compile(r'(?i)(password\s*[:=]\s*)[^\s]+'),
]
CLASSIFIERS = [
    ("vps_only_path", [r'permission denied.*(/home/wpuiai|/var/log/uiai)', r'failed to create.*(/home/wpuiai|/var/log/uiai)'], "Rewrite CI temp config paths to temp data/share/device/log locations before starting isolated engine."),
    ("browser_pool_starvation", [r'queued request', r'timed out', r'POST /api/session .*m\d|POST /api/session .*\d{3,}\.\d+s', r'max[_ ]?pool|pool.*active'], "Match temp browser pool size to stress concurrency or lower concurrency; print failed result details before exit."),
    ("startup_health_failure", [r'Failed to connect to 127\.0\.0\.1', r'health unavailable', r'curl: \(7\)', r'connection refused'], "Print bounded engine/site logs before health-check exit; inspect temp config and startup panic."),
    ("packet_drift", [r'Focusa packet drift check failed', r'packet drift'], "Update paired packet schema/docs/tools/smokes or the drift checker expectations."),
    ("mcp_route_parity", [r'mcp tool route parity failed', r'advertised-but-unrouted', r'missing MCP routes'], "Add MCP tools/call route or remove stale advertised metadata; rerun smoke-mcp-tool-routes."),
    ("pi_registration_drift", [r'Pi extension missing', r'pi extension registration'], "Add/update Pi tool mirror, command phrase, or smoke expectation."),
    ("go_test_failure", [r'^FAIL\s+github\.com/WPUIAI/uiai-engine', r'--- FAIL:'], "Reproduce exact package/test with go test -run ... -count=1 -v, then full go test ./...."),
    ("browser_action_error", [r'selector_not_found|eval_failed|wait_timeout|failed_requests|browser_session'], "Inspect browser diagnostics/errors; capture Focusa diagnostics intake if actionable."),
]
INTERESTING_FILES = re.compile(r'(?i)(engine|site|diagnostics|soak|flakiness|browser|uiai|log|json)')

def redact(text: str) -> str:
    out = text
    for pat in SECRET_PATTERNS:
        out = pat.sub(lambda m: m.group(1) + "REDACTED", out)
    out = re.sub(r'([?&](?:token|key|secret|signature|sig|password|credential|auth)=[^\s&#]+)', lambda m: m.group(0).split('=')[0] + '=REDACTED', out, flags=re.I)
    return out

def run(cmd: list[str], cwd: Path | None = None) -> str:
    p = subprocess.run(cmd, cwd=cwd, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if p.returncode != 0:
        raise SystemExit(f"command failed ({p.returncode}): {' '.join(cmd)}\n{redact(p.stderr or p.stdout)}")
    return p.stdout

def latest_failed(branch: str) -> str:
    data = run(["gh","run","list","--branch",branch,"--limit","20","--json","databaseId,status,conclusion"])
    for item in json.loads(data):
        if item.get("conclusion") == "failure":
            return str(item["databaseId"])
    raise SystemExit(f"no failed run found on branch {branch}")

def bounded_lines(text: str, limit: int) -> list[str]:
    lines = redact(text).splitlines()
    if len(lines) <= limit: return lines
    head = max(1, limit // 2)
    tail = max(1, limit - head - 1)
    return lines[:head] + [f"... omitted {len(lines)-head-tail} lines ..."] + lines[-tail:]

def classify(blob: str) -> list[dict]:
    found=[]
    for name, patterns, action in CLASSIFIERS:
        hits=[]
        for pat in patterns:
            m=re.search(pat, blob, re.I|re.M)
            if m: hits.append(redact(m.group(0))[:220])
        if hits:
            found.append({"class":name,"evidence":hits[:3],"recommended_action":action})
    return found or [{"class":"unknown","evidence":[],"recommended_action":"Inspect failed step logs and downloaded artifacts; add bounded log dumps if failure context is missing."}]

def read_artifacts(path: Path, max_files: int, max_lines: int) -> list[dict]:
    if not path.exists(): return []
    files=[p for p in path.rglob('*') if p.is_file() and INTERESTING_FILES.search(str(p))]
    out=[]
    for p in sorted(files)[:max_files]:
        try: text=p.read_text(errors='replace')
        except Exception as e: text=f"<unreadable: {e}>"
        out.append({"path":str(p),"lines":bounded_lines(text, max_lines)})
    return out

def main() -> int:
    ap=argparse.ArgumentParser(description="Summarize failed UIAI GitHub Actions logs and artifacts.")
    ap.add_argument("run_id", nargs="?", help="GitHub Actions run id. Defaults to latest failed run on branch.")
    ap.add_argument("--latest-failed", action="store_true", help="Use latest failed run on branch.")
    ap.add_argument("--branch", default="main")
    ap.add_argument("--artifact-dir", help="Existing or target artifact directory. Default /tmp/uiai-ci-artifacts-<run>.")
    ap.add_argument("--log-file", help="Offline mode: summarize this failed log file instead of gh run view.")
    ap.add_argument("--no-download", action="store_true", help="Do not run gh run download.")
    ap.add_argument("--json", action="store_true", help="Emit JSON only.")
    ap.add_argument("--max-lines", type=int, default=80)
    ap.add_argument("--max-files", type=int, default=12)
    args=ap.parse_args()
    run_id=args.run_id
    if args.log_file:
        failed_log=Path(args.log_file).read_text(errors='replace')
        run_id=run_id or "offline"
    else:
        if args.latest_failed or not run_id:
            if not shutil.which("gh"): raise SystemExit("gh required for latest/run mode; use --log-file for offline mode")
            run_id=latest_failed(args.branch)
        failed_log=run(["gh","run","view",run_id,"--log-failed"])
    artifact_dir=Path(args.artifact_dir or f"/tmp/uiai-ci-artifacts-{run_id}")
    if not args.log_file and not args.no_download:
        artifact_dir.mkdir(parents=True, exist_ok=True)
        subprocess.run(["gh","run","download",run_id,"-D",str(artifact_dir)], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    artifacts=read_artifacts(artifact_dir, args.max_files, args.max_lines)
    blob=redact(failed_log)+"\n"+"\n".join("\n".join(a["lines"]) for a in artifacts)
    report={
        "ok": False,
        "run_id": run_id,
        "artifact_dir": str(artifact_dir),
        "failed_log_excerpt": bounded_lines(failed_log, args.max_lines),
        "artifact_count": len(artifacts),
        "artifacts": artifacts,
        "classifications": classify(blob),
    }
    if args.json:
        print(json.dumps(report, indent=2))
    else:
        print(f"ci summary: run={run_id} artifact_dir={artifact_dir} artifacts={len(artifacts)}")
        for c in report["classifications"]:
            print(f"- class={c['class']}: {c['recommended_action']}")
            for ev in c.get("evidence",[]): print(f"  evidence: {ev}")
        print("--- failed log excerpt ---")
        print("\n".join(report["failed_log_excerpt"]))
        if artifacts:
            print("--- artifact excerpts ---")
            for a in artifacts:
                print(f"### {a['path']}")
                print("\n".join(a["lines"]))
    return 1 if report["classifications"][0]["class"] == "unknown" else 0
if __name__ == "__main__":
    raise SystemExit(main())
