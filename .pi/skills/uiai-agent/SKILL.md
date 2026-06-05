---
name: uiai-agent
description: Main UIAI Engine agent workflow: discovery, search/open/read, browser @refs, diagnostics, Focusa packet proof, parity checks, and release-safe cleanup.
---
# UIAI Agent Skill

Use this skill when working in `/home/wpuiai/uiai-engine` on agent-facing UIAI workflows, Pi tools, MCP tools, CLI wrapper behavior, browser sessions, search, diagnostics, Focusa proof packets, or public docs.

## Start here

1. Verify repo context and state:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch && bd ready | sed -n "1,120p"'
```

2. Read the compact docs for the task:

- `README.md` documentation map.
- `docs/UIAI_FOR_AGENTS_QUICKSTART.md` for shortest end-to-end agent path.
- `docs/AGENT_UX_COOKBOOK.md` for repeatable recipes.
- `docs/PUBLIC_API_PARITY_MATRIX.md` for HTTP/Pi/MCP/CLI/auth/smoke parity.
- `docs/ENDPOINT_AUTH_MATRIX.md` for auth boundaries.

3. Use discovery before loading large schemas:

```text
pi_uiai_agent_card
pi_uiai_tool_search q="diagnostics"
pi_uiai_tool_graph
uiai_health
```

MCP equivalents:

```text
uiai_agent_card
uiai_tool_search
uiai_tool_graph
uiai_health
```

CLI equivalents:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai status && scripts/uiai health && scripts/uiai tools search diagnostics'
```

## Default agent web workflow

Use this sequence for research or page inspection:

```text
uiai_search query="<query>" limit=3
uiai_browser_open url="<selected url>" focusa_scope={project_root,continuity_id,evidence_ref}
# CLI helper: scripts/uiai-open-result.sh --query "<query>" --index 1 --out /tmp/uiai-open-result.json
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_diagnostics session_id="<sid>" limit=100   # if blank/broken/flaky/failed action
uiai_browser_close session_id="<sid>"
```

Rules:

- Prefer `browser_read` for content.
- Prefer `browser_snapshot` `@ref` selectors for action targets.
- Run diagnostics after failed clicks/waits/evals, blank screenshots, unexpected navigation, CORS/API suspicion, or console-error clues.
- Close sessions when done.
- Never put raw base64, full page dumps, secrets, cookies, or token values into docs, bead notes, Focusa state, or final reports.

## Focusa packet workflow

Use when the outcome needs durable proof or handoff context.

1. Collect bounded UIAI responses: search/read/snapshot/diagnostics/errors/screenshot/share.
2. Compose packet, either with executable `/uiai` shortcuts or tool calls:

```text
/uiai research <query>
/uiai proof <url>
/uiai diagnose <session_id>
```

```text
uiai_focusa_packet_build goal="<goal>" mode="proof" responses=[...] focusa_scope={project_root,continuity_id,evidence_ref} cleanup_session_id="<sid>"
```

HTTP/CLI alternatives:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai research packet --url https://example.com --goal "Proof packet" --out /tmp/uiai-research-packet.json'
# Or manual compose from a prepared request:
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai packet compose /tmp/uiai-packet-request.json | tee /tmp/uiai-packet.json'
```

3. Pass `recommended_focusa.args_preview` to the appropriate Focusa tool:

- `focusa_evidence_capture` for search/read/snapshot/screenshot/share proof.
- `focusa_browser_diagnostics_intake` for diagnostics/failure envelopes.
- `focusa_predict_record` for bounded next-action/risk forecasts.

UIAI packets are proposals/evidence bundles. Focusa remains the durable Workpoint/evidence/prediction/metacog authority.

## Parity and smoke gates

Run relevant gates after changing tool schemas, docs, route exposure, packet fields, browser behavior, or MCP/Pi code:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
```

Browser reliability gates when browser/session/error behavior changes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-browser-error-regressions.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-failed-network-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

## Docs completeness gate

When adding or changing an agent-facing route/tool/skill/smoke, run:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
```

The gate checks known agent surfaces for README, Session API, quickstart/cookbook, parity matrix, auth matrix, release checklist, skills, smokes, and CI workflow coverage. Fix failures by updating the named file/needle instead of weakening the gate unless the canonical surface truly changed.

## Release-safe proof loop

For release/deploy/push/watch-CI, use:

- `docs/RELEASE_DEPLOY_RUNBOOK.md`
- `docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md`
- `docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md`

Minimum local proof before claiming agent-surface completion:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh && scripts/smoke-mcp-tool-routes.sh'
```

## Auth and environment reminders

- Loopback callers may access browser/session/search/packet/error/frame routes without remote tokens.
- Remote callers need `UIAI_API_KEY` or `UIAI_BEARER_TOKEN` for loopback-public remote-auth route families.
- Discovery endpoints are public and safe for low-context bootstrap.
- Do not print secret values. Reference env var names and config paths only.

## When to use sibling skills

- `/skill:vision` — detailed browser/screenshot/session API patterns.
- `/skill:uiai-focusa-packet` — packet-specific proof/handoff playbook.
- `/skill:uiai-release` — release/deploy/push/CI workflow.
- `/skill:uiai-mcp` — MCP bridge setup, reconnect, route parity, and tool routing workflow.
- `/skill:uiai-remote-auth` — remote/tunnel auth boundaries, env vars, and credential-safe examples.
- `/skill:uiai-docs-maintenance` — docs source-of-truth, parity, drift, release proof, and bead closure workflow.
- `/skill:uiai-ci-debug` — GitHub Actions failure diagnostics.
- `/skill:uiai-browser-debug` — browser failure triage.

## Final report format

Include:

- Task/bead completed.
- Files changed.
- Proof commands and results.
- Evidence handles or artifact paths.
- Prediction/known risk if relevant.
- Next bounded gap from `bd ready` or trajectory.
