---
name: uiai-ci-debug
description: UIAI GitHub Actions and Browser Reliability CI debugging: identify failed runs, inspect logs/artifacts, classify failures, reproduce locally, patch, push, and watch reruns.
---
# UIAI CI Debug Skill

Use this skill when a UIAI Engine GitHub Actions run fails or when preparing a CI-oriented fix.

## Source docs

- `docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md` — focused CI failure-class guide.
- `docs/BROWSER_RELIABILITY_RUNBOOK.md` — browser stress/soak gates.
- `docs/RELEASE_DEPLOY_RUNBOOK.md` — push/watch/proof loop.
- `.github/workflows/browser-reliability.yml` — current reliability workflow.

## 1. Identify failing run

```bash
cd /home/wpuiai/uiai-engine
gh run list --branch main --limit 10 --json databaseId,headSha,status,conclusion,workflowName,displayTitle,createdAt,url \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle) \(.url)"'
```

For current head only:

```bash
HEAD_SHA=$(git rev-parse HEAD)
gh run list --branch main --limit 20 --json databaseId,headSha,status,conclusion,workflowName,url \
  | jq -r --arg sha "$HEAD_SHA" '.[] | select(.headSha==$sha)'
```

Watch active run:

```bash
gh run watch <run-id> --exit-status
```

## 2. Read failed logs first

```bash
gh run view <run-id> --log-failed | sed -n '1,320p'
```

Capture:

- workflow name
- failed job/step
- exit code
- first concrete error line
- whether bounded engine/site logs appeared inline

Do not guess from a timeout alone. Download artifacts next.

## 3. Download artifacts

```bash
RUN_ID=<run-id>
OUT=/tmp/uiai-ci-artifacts-$RUN_ID
rm -rf "$OUT"
mkdir -p "$OUT"
gh run download "$RUN_ID" -D "$OUT" || true
find "$OUT" -type f -maxdepth 5 -print
```

Bounded print:

```bash
for f in $(find "$OUT" -type f | sort); do
  echo "--- $f"
  sed -n '1,220p' "$f"
done
```

Common Browser Reliability artifacts:

- `uiai-diag-stress-engine.log`
- `uiai-diag-stress-site.log`
- `diagnostics-4x10.json`
- `uiai-soak-engine.log`
- `uiai-soak-site.log`
- `browser-flakiness-soak.json`

## 4. Classify common failures

| Failure | Symptom | Fix direction |
|---|---|---|
| VPS-only CI path | permission denied for `/home/wpuiai/...` or `/var/log/uiai...` | Rewrite temp config paths in smoke/stress script. |
| Browser pool starvation | session opens take minutes, queue depth high, timeout | Match temp pool size to concurrency or reduce concurrency. |
| Missing immediate logs | failed step only shows curl/timeout | Print bounded engine/site logs before startup exit. |
| Packet drift | `Focusa packet drift check failed` | Update paired schema/docs/tools/smokes or drift script. |
| MCP advertised-but-unrouted | route parity smoke names tool | Add `tools/call` route or remove stale metadata. |
| Pi registration drift | required Pi tool/MCP mirror missing | Add Pi tool/mirror or update deliberate omission docs/smoke. |
| Go test failure | `FAIL github.com/WPUIAI/uiai-engine/...` | Reproduce exact package/test with `-count=1 -v`. |

## 5. Local reproduction commands

Core gates:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
```

Browser Reliability reproduction:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 OUT=/tmp/uiai-local-diagnostics-4x10.json scripts/stress-browser-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && DURATION_SECONDS=30 CONCURRENCY=2 OUT=/tmp/uiai-local-soak.json scripts/soak-browser-flakiness.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

Exact Go test repro:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./internal/routes -run TestName -count=1 -v'
```

## 6. Patch expectations for CI scripts

CI scripts that start temp engines should:

- use deterministic `/tmp/uiai-*-engine.log` and `/tmp/uiai-*-site.log`
- rewrite VPS-only paths to temp dirs/files
- wait for `/health`
- print bounded engine/site logs on startup failure
- write JSON reports
- exit nonzero only after printing actionable context
- avoid raw secrets and raw base64 in logs/artifacts

## 7. Commit, push, watch rerun

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short && git add <files> && git commit -m "fix: concise CI fix"'
cd /home/wpuiai/uiai-engine
gh auth setup-git
git push origin HEAD:main
gh run list --branch main --limit 5 --json databaseId,headSha,status,conclusion,workflowName,displayTitle \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle)"'
gh run watch <new-run-id> --exit-status
```

## 8. Report format

Include:

- Run id and workflow.
- Failed step and failure class.
- Artifact/log handles inspected.
- Root cause and patch files.
- Local reproduction/gates run.
- Rerun id/status.
- Next risk if not fully closed.
