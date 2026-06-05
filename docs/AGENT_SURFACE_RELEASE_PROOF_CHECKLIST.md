# Agent Surface Release Proof Checklist

Use this checklist before claiming a UIAI Engine release that changes HTTP, Pi, MCP, CLI, browser/session, search, Focusa, auth, or docs surfaces.

## Required proof commands

Run from `/home/wpuiai/uiai-engine` unless noted. For live release claims, build/deploy the current commit to `uiai-engine.service` first, wait for `/health`, then run localhost proof against the live service. Use [Release Deploy Runbook](RELEASE_DEPLOY_RUNBOOK.md) for the full rebuild, restart, push, CI-watch, failure-triage, and proof loop.

```bash
go test ./...
node --check mcp/browser-session-mcp.mjs
scripts/smoke-agent-integrations.sh
scripts/check-tool-parity.sh
scripts/smoke-mcp-tool-routes.sh
scripts/smoke-mcp-structured-failure.sh
scripts/smoke-pi-extension-registration.sh
scripts/smoke-pi-rendering.sh
scripts/smoke-pi-uiai-off.sh
scripts/smoke-browser-error-regressions.sh
scripts/smoke-failed-network-diagnostics.sh
scripts/smoke-focusa-error-evidence.sh
```

Optional pre-release soak:

```bash
make release-browser-reliability
```

## Surface checklist

| Surface | Required proof | Evidence to cite |
|---|---|---|
| HTTP discovery/tools | `/api/tools/agent-card`, `/api/tools/search`, `/api/tools/graph`, `/api/tools/mcp` pass through `scripts/smoke-agent-integrations.sh` | smoke output + commit hash |
| Browser/session | open/read/diagnostics/close and browser error regression smokes pass | `/tmp/uiai-browser-error-regressions.json` when generated |
| Failed network diagnostics | `scripts/smoke-failed-network-diagnostics.sh` passes | `/tmp/uiai-failed-network-diagnostics.json` |
| MCP bridge | `node --check`, route parity smoke, structured failure smoke pass | `scripts/smoke-mcp-tool-routes.sh`, `scripts/smoke-mcp-structured-failure.sh` |
| Pi extension | registration, rendering, and `/uiai off` smokes pass | `scripts/smoke-pi-extension-registration.sh`, `scripts/smoke-pi-rendering.sh`, `scripts/smoke-pi-uiai-off.sh` |
| CLI | `scripts/uiai` smoke paths in `scripts/smoke-agent-integrations.sh` pass and `scripts/check-tool-parity.sh` sees required commands | CLI JSON/status/errors/tools/research packet proof |
| Search/providers | provider metadata and search smoke pass; missing-key degraded-mode test passes; search results include `uiai-search:<provider>:<query-hash>:<rank>` refs | `go test ./internal/routes`, `scripts/smoke-agent-integrations.sh` |
| Auth/security | loopback/remote negative and remote positive tests pass; redaction tests pass | `go test ./internal/auth ./internal/observability ./internal/vision` |
| Focusa evidence | diagnostics/error evidence handles and smoke pass | `/tmp/uiai-error-focusa-evidence.json` plus docs handle refs |
| WordPress plugin parity | Existing plugin route/auth/error docs updated; plugin changes committed separately if needed | `docs/WORDPRESS_PLUGIN_ROUTE_PARITY_MATRIX.md`; WPUIAI commit hash if plugin changed |
| Public docs | README, Session API, gap inventory, and matrices updated when surface behavior changes | doc commit hash |

## Release notes template

```md
## UIAI Engine agent-surface proof

- Commit: <hash>
- HTTP/Pi/MCP/CLI/browser/search/auth/Focusa gate: passed
- Proof commands: <list or CI job URL>
- Evidence refs: <uiai-error:* / uiai-diagnostics:* / artifact paths>
- Known gaps/deferred work: <bounded list>
```

## Update rules

- Add a checklist row when a new agent-facing surface or smoke family is added.
- Browser reliability scripts must rewrite VPS-only config paths (`data_dir`, `share_dir`, device template `script_dir`) into temp directories before CI starts an isolated engine.
- CI startup failures must print bounded engine/site logs before exiting; artifact upload is useful, but logs should be visible in the failing step. Use `scripts/ci-log-summary.py <run-id>` to classify failures and artifact excerpts.
- Keep proof handles bounded: cite commit hashes, artifact paths, and stable `uiai-*` handles instead of raw logs/base64 blobs.
- Keep deployment docs secret-safe: reference env file paths and variable names only, never literal provider/API/token values.
- If a gate is intentionally skipped, record why in `docs/AGENT_COMPATIBILITY_GAP_INVENTORY_2026-06-04.md` before release.
