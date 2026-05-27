# Browser Reliability Runbook

Current local runbook for UIAI browser/session reliability gates.

## Goals

- Catch diagnostics/session regressions before release.
- Keep browser QA fast enough for agent loops.
- Prefer direct browser actions over fragile long async eval chains.
- Preserve proof as JSON/log artifacts.

## Commands

```bash
make browser-stress
make browser-soak
make browser-reliability
make release-browser-reliability
```

Defaults:

- `browser-stress`: `SESSIONS=4`, `ROUNDS=10`, output `/tmp/uiai-browser-diagnostics-4x10.json`.
- `browser-soak`: `DURATION_SECONDS=30`, `CONCURRENCY=2`, output `/tmp/uiai-browser-flakiness-soak.json`.
- `release-browser-reliability`: `DURATION_SECONDS=300`, `CONCURRENCY=2`, output `/tmp/uiai-browser-flakiness-soak-5m.json`.

Override example:

```bash
DURATION_SECONDS=600 CONCURRENCY=3 OUT=/tmp/uiai-soak-10m.json make browser-soak
```

## CI gate

`.github/workflows/browser-reliability.yml` runs on pull requests, pushes to `master`/`main`, and manual dispatch.

Gate steps:

1. `go test ./...`
2. diagnostics stress `40/40`
3. mixed browser soak
4. upload JSON reports and engine/site logs

Manual inputs:

- `soak_seconds`
- `concurrency`

## Async eval reliability rule

Use the smallest tool that matches the task:

- `/api/session/{id}/eval`: short synchronous DOM reads only.
- `/api/session/{id}/eval_async`: small bounded awaits with `timeout_ms` default `5000`, max `15000`.
- Direct session actions: multi-step UI workflows (`snapshot`, `click`, `type`, `wait`, `diagnostics`).

Avoid hiding long browser workflows inside one Promise. If a long eval flakes or a Promise handle appears stale/collected, split into direct actions and read diagnostics before changing app code.

## Diagnostics workflow

During browser troubleshooting:

1. Open/reuse session.
2. Reproduce issue with direct actions.
3. Read `GET /api/session/{id}/diagnostics` or tool `browser_diagnostics`.
4. Classify: console error, JS exception, failed request, selector not found, timeout, page crash, browser unavailable.
5. Patch only after diagnostics or DOM/snapshot evidence supports the cause.
6. Re-run diagnostics and soak/stress proof.

## Artifact handles

Typical local artifacts:

- `/tmp/uiai-browser-diagnostics-4x10.json`
- `/tmp/uiai-browser-flakiness-soak.json`
- `/tmp/uiai-browser-flakiness-soak-5m.json`
- `/tmp/uiai-diag-stress-engine.log`
- `/tmp/uiai-diag-stress-site.log`
- `/tmp/uiai-soak-engine.log`
- `/tmp/uiai-soak-site.log`

## Acceptance

- `go test ./...` passes.
- diagnostics stress reports `ok=true`, `failed=0`.
- soak reports `ok=true`, `failed=0` or only expected negative-test failure classes.
- diagnostics buffers remain bounded and secret-redacted.
- browser action failures return typed error classes plus diagnostics summary when available.
