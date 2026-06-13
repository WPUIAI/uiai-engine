# UIAI Engine Release, Deploy, Push, and CI Runbook

Use this runbook before claiming a UIAI Engine release that changes HTTP routes, Pi tools, MCP routes, CLI commands, browser/session behavior, Focusa packet/evidence flows, auth, docs, or repo-local skills.

For public/mobile FPV PWA exposure, also follow [FPV Public Route Deployment Plan](FPV_PUBLIC_ROUTE_DEPLOY_PLAN.md); do not expose the full engine publicly.

## Scope and safety boundaries

- Project root: `/home/wpuiai/uiai-engine`.
- Run repository file operations as `wpuiai`.
- Run `systemctl` and GitHub CLI auth operations only from an authorized shell/account.
- Never print literal API keys, bearer tokens, webhook secrets, or provider keys.
- Treat `/home/wpuiai/uiai-engine/config.yaml` and systemd env files as secret-adjacent: reference variable names and paths, not values.
- If root accidentally writes into `/home/wpuiai`, immediately run `fix-user-perms wpuiai`.

## 1. Preflight repo and bead state

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd ready | sed -n "1,120p"'
```

Expected:

- Branch tracks `origin/main` or the intended release branch.
- Uncommitted changes are understood and owned.
- Active bead is in progress or release work has a release bead.

## 2. Documentation and drift checks

Run these after docs, tool schemas, skills, or packet fields change:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && PATTERN="does \\*\\*not\\*\\* yet expo""se|intentionally defer""red|Later, UIAI HT""TP|Defer .*research-pack""et" && rg -n "$PATTERN" README.md docs .pi mcp scripts || true'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
```

If a new agent-facing surface is added, also update:

- README documentation map and agent integration summary.
- `docs/SESSION_API.md`.
- `docs/ENDPOINT_AUTH_MATRIX.md` for auth mode.
- `docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md`.
- Relevant repo-local skills under `.pi/skills/`.
- MCP/Pi/CLI parity smokes.

## 3. Local gates

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-structured-failure.sh'
```

For browser reliability, Source-to-Markdown, and error evidence changes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-source-markdown-e2e.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-browser-error-regressions.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-failed-network-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

For packet changes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet-ci.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai smoke packet'
```

## 4. Commit small batches

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short && git diff --stat'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git add <files> && git commit -m "type: concise subject"'
```

Commit guidance:

- Keep docs-only, feature, and CI-fix commits separate when practical.
- Do not add ignored `.beads/` unless the repository intentionally tracks a bead export.
- Run `bd update <id> --notes "..."` and `bd close <id> --reason "..."` before or after the commit when work is complete.

## 5. Rebuild binary

Create a timestamped backup, then rebuild the live binary:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && cp ./uiai-engine ./uiai-engine.bak.$(date +%Y%m%d-%H%M%S)'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go build -o ./uiai-engine ./cmd/uiai-engine'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && ./uiai-engine --version'
```

If the binary path changes, update `deploy/uiai-engine.service` and the live systemd unit together.

## 6. Restart live service

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

Fix the source/config/unit issue, rebuild, restart, and repeat health verification.

## 7. Live service proof

Use the bundled smoke after rebuild/restart, or `--check-only` to prove the currently running service without restarting. The default profile is compatible with hardened `vision.allow_private_urls: false`: public browser targets are used, and private localhost browser regression smokes are skipped unless explicitly enabled.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/release-service-smoke.sh --check-only'
# Optional local/dev-only private localhost browser regressions:
# as-user wpuiai 'cd /home/wpuiai/uiai-engine && UIAI_ALLOW_PRIVATE_SMOKES=1 scripts/release-service-smoke.sh --check-only'
# live mode requires root/systemctl boundary:
# cd /home/wpuiai/uiai-engine && scripts/release-service-smoke.sh --skip-build
```

Manual live smokes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-agent-integrations.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
```

Expected outputs:

- Packet smoke prints `focusa packet smoke ok` and writes `/tmp/uiai-focusa-packet-smoke.json`.
- Agent integration smoke prints `agent integration smoke ok`; hardened default also prints that private localhost browser smokes were skipped unless `UIAI_ALLOW_PRIVATE_SMOKES=1` is set.
- MCP route smoke prints advertised/routed parity.
- Pi extension smoke prints tool/command/mirror counts.

## 8. Push with GitHub CLI

`wpuiai` may not have GitHub credentials. If the authorized GitHub CLI account is root, use it only for Git/GitHub operations; keep file edits owned by `wpuiai`.

```bash
cd /home/wpuiai/uiai-engine
gh auth status
gh auth setup-git
git push origin HEAD:main
```

If push fails with `could not read Username`, run `gh auth setup-git` from the authorized account and retry.

## 9. Watch GitHub Actions

```bash
cd /home/wpuiai/uiai-engine
gh run list --branch main --limit 10 --json databaseId,headSha,status,conclusion,workflowName,displayTitle,url \
  | jq -r '.[] | "\(.databaseId) \(.status)/\(.conclusion // "-") \(.workflowName) \(.headSha[0:7]) \(.displayTitle) \(.url)"'

gh run watch <run-id> --exit-status
```

If the run fails, inspect logs and artifacts before guessing. Use [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md) for the focused failure-class playbook:

```bash
scripts/ci-log-summary.py <run-id>
gh run view <run-id> --log-failed | sed -n '1,320p'
rm -rf /tmp/uiai-ci-artifacts-<run-id>
mkdir -p /tmp/uiai-ci-artifacts-<run-id>
gh run download <run-id> -D /tmp/uiai-ci-artifacts-<run-id> || true
find /tmp/uiai-ci-artifacts-<run-id> -type f -maxdepth 5 -print
```

Known Browser Reliability failure classes:

- **VPS-only path in CI temp config:** engine log mentions permission denied creating `/home/wpuiai/...` or `/var/log/uiai/...`. Fix temp config rewriting in the smoke/stress script.
- **Browser pool starvation:** one session open takes minutes while stress runs more concurrent sessions than the temp engine pool. Match temp pool size to stress concurrency or lower concurrency.
- **Missing immediate logs:** failed step only shows curl/timeout. Enhance the script to print bounded engine/site logs before exit.
- **Advertised-but-unrouted MCP tool:** `scripts/smoke-mcp-tool-routes.sh` identifies tools/list entries missing tools/call routes.
- **Packet drift:** `scripts/check-focusa-packet-drift.sh` identifies docs/tool/schema files missing schema or key terms.

Fix, run local reproduction, commit, push again, and watch the new run.

## 10. Record proof and close work

When CI is green:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd update <id> --notes "Release proof: service healthy; local gates green; GitHub Actions run <run-id> green."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd close <id> --reason "Completed: release/deploy/push/CI proof green."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd sync'
```

If Focusa is in use, capture a bounded evidence handle:

```text
focusa_evidence_capture(
  target_ref="uiai-engine release",
  result="service healthy, local gates green, GitHub Actions run <run-id> green",
  evidence_ref="gh-run:<run-id>:green+git:<sha>+service:uiai-engine:healthy",
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

## Related docs

- [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md)
- [Agent Surface Release Proof Checklist](AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md)
- [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md)
- [Browser Reliability Runbook](BROWSER_RELIABILITY_RUNBOOK.md)
- [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md)
- [Session API](SESSION_API.md)
