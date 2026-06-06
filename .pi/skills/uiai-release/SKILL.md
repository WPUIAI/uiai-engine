---
name: uiai-release
description: UIAI Engine release workflow: preflight, local gates, rebuild, restart, live smoke, push, GitHub Actions watch, CI triage, proof capture, and bead closure.
---
# UIAI Release Skill

Use this skill when preparing, deploying, pushing, or proving a UIAI Engine release that changes routes, Pi/MCP/CLI tools, browser/session behavior, Focusa packet/evidence flows, auth, docs, or repo-local skills.

## Safety boundaries

- Project root: `/home/wpuiai/uiai-engine`.
- Run repo file operations as `wpuiai`.
- Use root/authorized shell only for `systemctl` and GitHub CLI auth/push operations when required.
- Never print literal secrets, API keys, bearer tokens, webhook secrets, cookies, or env-file values.
- Reference secret-adjacent paths/variable names only.
- If root accidentally writes in `/home/wpuiai`, run `fix-user-perms wpuiai` immediately.

## Source docs

- `docs/RELEASE_DEPLOY_RUNBOOK.md` — full release/deploy/push/watch-CI loop.
- `docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md` — agent-surface proof gates.
- `docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md` — failed GitHub Actions triage.
- `docs/PUBLIC_API_PARITY_MATRIX.md` — HTTP/Pi/MCP/CLI/auth parity map.

## 1. Preflight

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd ready | sed -n "1,120p"'
```

Expected:

- Changes are understood and owned.
- Active bead is in progress or a release bead exists.
- Branch is the intended release branch.

## 2. Local gates

Run the relevant set before release claims:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-examples.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-skills.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-open-result.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-render-diagnostics-artifact.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai-mcp-reconnect-help.sh --check'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
```

Browser/session/error behavior changes also require reliability gates. These isolated scripts start temporary engines and enable private URLs only inside their temp configs; they do not weaken the live `vision.allow_private_urls: false` default.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-browser-error-regressions.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-failed-network-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

Only when deliberately testing a live local/dev engine configured with `vision.allow_private_urls: true`, opt into private localhost smokes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && UIAI_ALLOW_PRIVATE_SMOKES=1 scripts/release-service-smoke.sh --check-only'
```

Packet behavior changes also require:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai smoke packet'
```

## 3. Commit small batches

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short && git diff --stat'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git add <files> && git commit -m "type: concise subject"'
```

Keep docs, features, and CI fixes separate when practical.

## 4. Rebuild binary

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && cp ./uiai-engine ./uiai-engine.bak.$(date +%Y%m%d-%H%M%S)'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go build -o ./uiai-engine ./cmd/uiai-engine'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && ./uiai-engine --version'
```

## 5. Restart service

```bash
systemctl status uiai-engine.service --no-pager -l | sed -n '1,100p'
systemctl restart uiai-engine.service
for i in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:7456/health >/tmp/uiai-health.json; then
    cat /tmp/uiai-health.json | jq .
    break
  fi
  sleep 1
done
systemctl status uiai-engine.service --no-pager -l | sed -n '1,100p'
```

If health fails:

```bash
journalctl -u uiai-engine.service -n 120 --no-pager
```

Fix source/config/unit issue, rebuild, restart, and repeat health.

## 6. Live service proof

Bundled proof against the current running service. Default proof is compatible with hardened `vision.allow_private_urls: false`: public browser targets are used and private localhost browser smokes are skipped.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/release-service-smoke.sh --check-only'
```

Live restart/bundle mode requires root/systemctl boundary:

```bash
cd /home/wpuiai/uiai-engine && scripts/release-service-smoke.sh --skip-build
```

Manual proof:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
```

Expected:

- `focusa packet smoke ok`.
- `agent integration smoke ok` plus the hardened-default skip note for private localhost browser smokes unless `UIAI_ALLOW_PRIVATE_SMOKES=1` is set.
- `mcp tool route parity ok`.
- `pi skill smoke ok`.
- `open result smoke ok`.
- `diagnostics artifact render smoke ok`.
- `pi extension registration ok`.

## 7. Push and watch GitHub Actions

Use authorized account/shell for GitHub CLI if `wpuiai` has no credentials:

```bash
cd /home/wpuiai/uiai-engine
gh auth status
gh auth setup-git
git push origin HEAD:main
```

Find and watch runs:

```bash
gh run list --branch main --limit 10 --json databaseId,headSha,status,conclusion,workflowName,displayTitle,url \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle) \(.url)"'

gh run watch <run-id> --exit-status
```

If CI fails, load `/skill:uiai-ci-debug` when available or follow `docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md`:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/ci-log-summary.py <run-id>'
gh run view <run-id> --log-failed | sed -n '1,320p'
OUT=/tmp/uiai-ci-artifacts-<run-id>
rm -rf "$OUT" && mkdir -p "$OUT"
gh run download <run-id> -D "$OUT" || true
find "$OUT" -type f -maxdepth 5 -print
```

## 8. Proof capture and close

When CI is green:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd update <id> --notes "Release proof: service healthy; local gates green; GitHub Actions run <run-id> green."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd close <id> --reason "Completed: release/deploy/push/CI proof green."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd sync'
```

Focusa evidence handle:

```text
focusa_evidence_capture(
  target_ref="uiai-engine release",
  result="service healthy, local gates green, GitHub Actions run <run-id> green",
  evidence_ref="gh-run:<run-id>:green+git:<sha>+service:uiai-engine:healthy",
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

## Final release report

Include:

- Commit SHA(s) and branch.
- Service health proof.
- Local gates run and results.
- Live smoke results.
- GitHub Actions run id/status.
- Evidence handle.
- Bead(s) closed.
- Remaining gap from `bd ready`.
