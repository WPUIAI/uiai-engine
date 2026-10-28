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
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./integrations/pi/uiai-engine.packet-builder.test.ts'
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

Cockpit dependency changes require a successful clean `npm ci` on OVH before
`npm audit --audit-level=high`, `npm run check`, unit tests, the mocked backend
smoke and `npm run build:frontend`. An updated existing `node_modules` tree is not
proof that the committed lockfile supports a clean install; validate the lockfile
before committing even when typecheck, tests and build already passed. Critical/high findings block readiness; disclose remaining moderate/low
findings rather than describing a nonzero audit as clean. The committed same-major
Zod override retains the patched 4.6.6 floor for Remotion transitive dependencies;
never use forced audit downgrades or silence peer conflicts to obtain green gates.

### Go toolchain and vulnerability gate

`go.mod` owns the Go baseline (1.26.10); GitHub builds and the Windows installer
derive that version rather than maintaining a separate version pin. Windows still
verifies the archive against its independently published SHA-256. OVH verification
uses the approved side-by-side Go-managed toolchain (`GOTOOLCHAIN=go1.26.10`),
retaining `/opt/go1.26.7` for rollback without changing global service configuration.

Build `govulncheck` with that same compiler; an older scanner's source loader may
reject newer Go syntax. CI runs the pinned `golang.org/x/vuln/cmd/govulncheck@v1.3.0`
in ordinary text mode so findings fail the job. Audit both source and the exact
candidate executable; a JSON-mode exit0 alone does not mean there are no findings.
Module-only or binary-symbol findings are not proof of exploitability or a source
call graph. Failed package loading is an incomplete audit, never a clean result.

## 4. Commit small batches

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short && git diff --stat'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git add <files> && git commit -m "type: concise subject"'
```

Commit guidance:

- Keep docs-only, feature, and CI-fix commits separate when practical.
- Do not add ignored `.beads/` unless the repository intentionally tracks a bead export.
- Run `bd update <id> --notes "..."` and `bd close <id> --reason "..."` before or after the commit when work is complete.

## 5. Build and stage the immutable candidate

Build/test/package on the approved OVH runner, not the production control host.
`.github/workflows/engine-release.yml` resolves the exact immutable release tag,
checks its version surfaces through `scripts/verify-uiai-version-surfaces.py`,
and waits for CI on that checked-out source SHA. Record the binary SHA-256.

The workflow creates or resumes a **draft**, never an already-published release.
A metadata-read failure is not a missing release. Dry-run dispatch does not upload
a release, and build-only dispatch leaves its candidate unpublished.

## 6. Deploy, verify, then publish

Use `scripts/deploy-engine-ovh.sh` with the exact approved artifact and named OVH
workers. Its idle checks, backups, rollback, health, and running-executable digest
checks replace direct rebuild/restart commands. Do not rebuild between hashing,
deploying and verifying; do not change services or public routing by implication.

A release requires its mandatory scoped browser/evidence smoke to pass.
`RUN_BROWSER_SMOKE=0` may describe an explicitly authorized diagnostic candidate,
not successful release acceptance. On failure retain the draft and diagnose;
never publish simply because the build or ordinary health check succeeded.

Only a successful non-dry-run deployment admits the publication job. It downloads
the draft binary and requires an exact match to the tested/deployed digest before
promotion. Published releases are not overwritten; stable promotion sets Latest.

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


## Idle and account-owned worker rollout

The canonical OVH deployment checks `/api/session` and `/api/health/browser`
before replacing the artifact and before each restart. Busy, missing or unreadable
counts block that action. For the two named workers, explicitly set
`REMOTE_IDLE_BASE_URLS="http://127.0.0.1:7456 http://127.0.0.1:7460"`
when `REMOTE_EXTRA_SERVICES=uiai-engine-ovh-w2.service` is selected. Never add an
unrelated service to a rollout. Home-directory backup/install writes run as the
owning account; backups, health readiness and running-process hashes remain
mandatory. Sealed evidence readiness is reported separately from API health.


The registered startup warmer is derived from `scripts/warm-browser-session.sh`;
`UPDATE_WARMUP_HELPER=1` installs that exact hashed artifact to the existing
`/usr/local/sbin/uiai-ovh-warm-session.sh` hook while preserving rollback. No unit
configuration is changed. Both201/202 and malformed deliveries with an exact
created-session header are cleaned up; absent identity remains explicitly unknown.
Warm-up uses `system:uiai-warmup` as provenance, never an authorization grant.


Browser Reliability binds its payload to the existing `self-hosted/ovh/uiai-engine`
runner. A lightweight hosted matrix sibling checks only its own run's job metadata
with Actions-read permission; a120-second admission timeout fails explicitly and
GitHub's native matrix fail-fast cancels the still-queued sibling. It never calls
foreign-run cancellation or needs Actions-write/runner-administration credentials.
Missing Chrome fails explicitly without installing packages. Per-run output paths
and required artifact uploads prevent stale or absent JSON from becoming proof.

## Cockpit over-the-air release gate

The Cockpit workflow stages both signed Mac architectures and Linux artifacts in
a draft before promotion. Platform versions must match; missing bundles or
signatures fail closed. The mutable channel uploads bundles/signatures first and
`latest.json` last, so a failed upload cannot advertise unavailable artifacts.
Release-shell regression checks execute the real workflow commands against an
isolated GitHub/filesystem fixture; they are not actual publication or native OTA.

Engine releases and signed Cockpit desktop updates are separate delivery lanes.
Stable tags publish as non-prereleases; `-dev` tags remain prereleases. Parent
Engine stable releases advance the repository's GitHub Latest. Cockpit keeps its
own `cockpit-latest` updater feed and must not displace the parent Engine Latest;
its immutable stable release is still explicitly non-prerelease.
Native qualification can use the existing public-repository hosted macOS runners:
manual `Cockpit CI` dispatch with `qualify_ota=true` runs Intel and Apple Silicon
canaries, without tags or publication. It builds the actual signed candidate with
production configuration, then a declared prior-source/version baseline in the
same worktree. Only that baseline uses a private loopback feed; the existing trust
key is unchanged and a corrupted-signature control must fail before a valid
automatic upgrade/relaunch and preservation/owned-fixture-rollback checks pass.
Receipts identify source revisions, versions and actual candidate bytes. This is
not a claim about an unidentified historical public binary or notarization;
manual qualification artifacts are not automatically published.

The same hosted fixture may capture its exact app window before upgrade, after
candidate activation and after owned rollback. The receipt binds each PNG to a
currently running fixture PID/window and digest; no full-desktop fallback or OS
permission change is allowed. Preserve both the early candidate frame and a later
bounded-settlement frame; an early blank view is not silently discarded or called
ready. An unavailable capture stays explicitly unproven
and does not manufacture visual acceptance or erase a separately valid native
updater receipt. Inspect the PNG pixels before claiming a version/toast is visible;
a captured file or matched hash alone is not that verification.

The release workflow calls that same native qualification workflow with the exact
candidate SHA and version. It never rebuilds separate Mac packages for publication.
Candidate metadata, DMG, updater bundle and signature are frozen before the baseline
checkout. Only successful native checks upload the release artifacts. Before any
staging, `scripts/verify-native-ota-artifact.py` binds both platform receipts to the
source SHA, version, production feed, exact archive and native-running executable
hashes, signature bytes and frozen DMG metadata. Missing, foreign or changed artifacts
fail closed; the draft and mutable feed remain unchanged.

Before claiming an existing-user Cockpit rollout is ready:

- Inspect the actual updater feed and assets for every advertised platform; an
  Intel Mac entry alone does not establish Apple Silicon or other-platform support.
- On named, authorized native devices, verify the existing-version → signed
  download → installation → relaunch → running-version path and data preservation.
  Check failure/retry behavior and the supported rollback path as well.
- A stored `installed_relaunching` receipt is only a pending installation intent.
  Cockpit queries native `getVersion()` before announcing activation, preserves
  pending receipts when the old process is still running or verification fails,
  and never removes a newer receipt during an overlapping check.
  A manifest check does not verify the downloaded artifact signature, and the
  updater's `Finished` download event is not installation or activation proof.
  Download/install/relaunch progress stays informational; only a matching native
  running-version confirmation produces the successful activation notification.
- Mocked updater tests, frontend builds and cross-compilation are not native OTA
  receipts. Public evidence-delivery failures remain separate release blockers.
- Use the actual authorized release channel, not probe releases or paid CI used
  merely to obtain a test receipt; never upgrade unnamed customer devices.
