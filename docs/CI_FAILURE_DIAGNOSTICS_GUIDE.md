# UIAI CI Failure Diagnostics Guide

Use this guide when GitHub Actions fails for UIAI Engine, especially the `Browser Reliability` workflow.

## 1. Identify the failing run

```bash
cd /home/wpuiai/uiai-engine
gh run list --branch main --limit 10 --json databaseId,headSha,status,conclusion,workflowName,displayTitle,createdAt,url \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle) \(.url)"'
```

For the current head:

```bash
HEAD_SHA=$(git rev-parse HEAD)
gh run list --branch main --limit 20 --json databaseId,headSha,status,conclusion,workflowName,url \
  | jq -r --arg sha "$HEAD_SHA" '.[] | select(.headSha==$sha)'
```

Watch an active run:

```bash
gh run watch <run-id> --exit-status
```

## 2. Summarize failed logs and artifacts

Use the helper first when available:

```bash
scripts/ci-log-summary.py <run-id>
scripts/ci-log-summary.py --latest-failed --branch main
scripts/ci-log-summary.py <run-id> --json > /tmp/uiai-ci-summary.json
```

It runs `gh run view --log-failed`, downloads artifacts to `/tmp/uiai-ci-artifacts-<run-id>`, redacts likely secrets, classifies common UIAI failure causes, and prints the next recommended fix. Offline fixture/debug mode:

```bash
scripts/ci-log-summary.py --log-file /tmp/failed.log --artifact-dir /tmp/uiai-ci-artifacts --json
```

## 3. Read failed step logs manually

```bash
gh run view <run-id> --log-failed | sed -n '1,320p'
```

Look for:

- failed step name (`Go tests`, `Focusa packet drift check`, `Diagnostics stress 40/40`, `Mixed browser soak`)
- exit code
- stack trace or script output
- whether the script printed engine/site logs inline

If failed logs only show a timeout or `curl` error, continue to artifact download before guessing.

## 4. Download and inspect artifacts

```bash
RUN_ID=<run-id>
OUT=/tmp/uiai-ci-artifacts-$RUN_ID
rm -rf "$OUT"
mkdir -p "$OUT"
gh run download "$RUN_ID" -D "$OUT" || true
find "$OUT" -type f -maxdepth 5 -print
```

Print bounded artifact logs:

```bash
for f in $(find "$OUT" -type f | sort); do
  echo "--- $f"
  sed -n '1,220p' "$f"
done
```

Common artifact files:

```text
uiai-diag-stress-engine.log
uiai-diag-stress-site.log
diagnostics-4x10.json
uiai-soak-engine.log
uiai-soak-site.log
browser-flakiness-soak.json
```

## 5. Browser Reliability workflow anatomy

Workflow file:

```text
.github/workflows/browser-reliability.yml
```

Main steps:

1. `go test ./...`
2. `scripts/check-tool-parity.sh`
3. `scripts/check-focusa-packet-drift.sh`
3. `SESSIONS=4 ROUNDS=10 scripts/stress-browser-diagnostics.sh`
4. `scripts/soak-browser-flakiness.sh`
5. artifact upload

Local reproduction:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/release-service-smoke.sh --check-only'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 OUT=/tmp/uiai-local-diagnostics-4x10.json scripts/stress-browser-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && DURATION_SECONDS=30 CONCURRENCY=2 OUT=/tmp/uiai-local-soak.json scripts/soak-browser-flakiness.sh'
```

For full local release reliability:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

## 6. Failure classes and fixes

### A. VPS-only path in CI temp config

Symptoms:

```text
Failed to create data dir /home/wpuiai/uiai-engine/data: permission denied
cannot create log dir /var/log/uiai: permission denied
```

Cause:

- A script copied `config.yaml` into a temp CI config but did not rewrite VPS-only paths.

Fix:

- Rewrite `data_dir`, `share_dir`, device frame `script_dir`, captcha `health_file`, captcha stats `log_file`, and logging `file` to temp paths before starting the isolated engine.
- Existing scripts should use `$TMPDIR/data`, `$TMPDIR/shares`, `$TMPDIR/device-templates`, and temp log files.

Local proof:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 scripts/stress-browser-diagnostics.sh'
```

### B. Browser pool starvation under CI

Symptoms:

```text
TimeoutError: timed out
POST /api/session 201 2m4.038s
[vision] queued request (depth=..., active=2, max=2)
```

Cause:

- Stress concurrency exceeds the isolated engine browser pool size on a slower GitHub runner.

Fix:

- Match temp `vision_pool_size`, `vision.pool_size`, and `vision.max_pool` to `SESSIONS` for diagnostics stress, or lower `SESSIONS`.
- Print failed result details before exit so the next failure is visible in step logs.

Local proof:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 OUT=/tmp/uiai-local-diagnostics-4x10-pool4.json scripts/stress-browser-diagnostics.sh'
```

### C. Missing immediate logs

Symptoms:

```text
curl: (7) Failed to connect to 127.0.0.1 port 7468
Process completed with exit code 7
```

Cause:

- Startup failed, but the script did not print engine/site logs before exiting.

Fix:

- Add a bounded log dump around the health check failure:

```bash
echo "--- engine log ---" >&2
sed -n '1,220p' /tmp/<engine-log>.log >&2 || true
echo "--- site log ---" >&2
sed -n '1,120p' /tmp/<site-log>.log >&2 || true
exit 7
```

### D. Packet drift failure

Symptoms:

```text
Focusa packet drift check failed:
<file>: missing <needle>
```

Cause:

- A packet surface, doc, tool, or smoke was updated without updating the drift expectations or related public docs.

Fix:

- Update the missing file or, if the check is stale, update `scripts/check-focusa-packet-drift.sh` with the new canonical surface.
- Re-run:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
```

### E. MCP advertised-but-unrouted tool

Symptoms:

```text
mcp tool route parity failed
advertised-but-unrouted: <tool>
```

Cause:

- `/api/tools/mcp` or MCP bridge metadata advertises a tool that `mcp/browser-session-mcp.mjs` does not route in `tools/call`.

Fix:

- Add the call route in `mcp/browser-session-mcp.mjs` or remove/rename stale metadata.
- Validate:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs && scripts/smoke-mcp-tool-routes.sh'
```

### F. Pi extension registration failure

Symptoms:

```text
Pi extension missing required tools: <tool>
Pi extension missing MCP mirrors: <tool>
```

Cause:

- A tool was added to HTTP/MCP without a Pi mirror, or the static smoke's required list needs a legitimate update.

Fix:

- Add the Pi tool under `.pi/extensions/uiai-engine.ts` or document a deliberate omission.
- Re-run:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
```

### G. Go test failure

Symptoms:

```text
FAIL github.com/WPUIAI/uiai-engine/internal/...
```

Fix:

- Reproduce locally with the exact package first:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./internal/routes -run TestName -count=1 -v'
```

Then run full suite:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
```

## 7. CI logging requirements for scripts

Every CI script that starts a temp engine should:

- write engine logs to a deterministic `/tmp/uiai-*-engine.log`
- write site logs to a deterministic `/tmp/uiai-*-site.log`
- rewrite VPS-only config paths to temp paths
- wait for `/health`
- print bounded engine/site logs before startup-exit
- write a JSON report when possible
- upload artifacts through the workflow

Do not rely only on uploaded artifacts. The failed step should include enough log context to diagnose startup failures immediately.

## 8. Fix and push loop

```bash
# after patching
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 scripts/stress-browser-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short && git add <files> && git commit -m "fix: concise CI fix"'
cd /home/wpuiai/uiai-engine
gh auth setup-git
git push origin HEAD:main
gh run list --branch main --limit 5 --json databaseId,headSha,status,conclusion,workflowName,displayTitle \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle)"'
```

Watch the new run until green or repeat diagnostics.

## Related docs

- [Release Deploy Runbook](RELEASE_DEPLOY_RUNBOOK.md)
- [Browser Reliability Runbook](BROWSER_RELIABILITY_RUNBOOK.md)
- [Agent Surface Release Proof Checklist](AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md)
- [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md)
