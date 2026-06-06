---
name: uiai-docs-maintenance
description: UIAI docs maintenance workflow: update public docs, README, skills, parity matrices, drift checks, release proof docs, and beads whenever UIAI tools/workflows change.
---
# UIAI Docs Maintenance Skill

Use this skill when a UIAI Engine change affects public docs, README links, Pi/MCP/CLI/HTTP tool surfaces, auth modes, Focusa packets, browser workflows, release proof, skills, or smoke gates.

## Start here

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && git status --short --branch && bd ready | sed -n "1,120p"'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-skills.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
```

Read the smallest relevant source-of-truth docs:

- `README.md` documentation map.
- `docs/SKILL_SOURCE_OF_TRUTH_POLICY.md` for repo-local skill ownership.
- `docs/PUBLIC_API_PARITY_MATRIX.md` for HTTP/Pi/MCP/CLI parity.
- `docs/ENDPOINT_AUTH_MATRIX.md` for auth boundaries.
- `docs/SESSION_API.md` for browser/session/API docs.
- `docs/UIAI_FOR_AGENTS_QUICKSTART.md` and `docs/AGENT_UX_COOKBOOK.md` for agent workflows.
- `docs/RELEASE_DEPLOY_RUNBOOK.md` and `docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md` for release proof.

## Source-of-truth rules

- Code/schema behavior first; docs describe verified behavior, not guesses.
- Repo-local skills under `.pi/skills/` are source of truth for UIAI-specific skills.
- Global `~/.pi/skills/` copies are convenience installs only.
- README should link major public docs and versioned agent artifacts.
- Parity matrix and smoke gates must change with exposed tool/route behavior.
- Bead notes should summarize what changed and which proof commands passed.
- Never paste secrets, raw base64, raw browser logs, cookies, full page dumps, or token values into docs.

## Changed surface → required docs/gates

| Changed surface | Docs to update | Gates/proof |
|---|---|---|
| HTTP route or API schema | `docs/SESSION_API.md`, `docs/PUBLIC_API_PARITY_MATRIX.md`, README if public | `go test ./...`, `scripts/check-tool-parity.sh` |
| Auth mode or route exposure | `docs/ENDPOINT_AUTH_MATRIX.md`, `docs/REMOTE_AUTH_EXAMPLES.md`, quickstart/cookbook if agent-facing | `go test ./internal/auth`, `scripts/smoke-agent-integrations.sh` |
| Pi extension tool/command | README, quickstart/cookbook, parity matrix, relevant `.pi/skills/*` | `scripts/smoke-pi-extension-registration.sh`, `scripts/check-tool-parity.sh` |
| MCP tool/route | `docs/MCP_CACHE_RECONNECT_TROUBLESHOOTING.md`, parity matrix, relevant skills | `node --check mcp/browser-session-mcp.mjs`, `scripts/smoke-mcp-tool-routes.sh` |
| Focusa packet field/mode/routing | `docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`, `docs/FOCUSA_PACKET_EXAMPLES_GALLERY.md`, packet skill, Session API | `scripts/check-focusa-packet-drift.sh`, `scripts/check-focusa-packet-examples.py`, packet builder tests |
| CLI behavior | README CLI section, quickstart/cookbook, parity matrix | `bash -n scripts/uiai`, `scripts/uiai smoke agent`, `scripts/check-tool-parity.sh` |
| Browser/session workflow | `docs/SESSION_API.md`, quickstart/cookbook, vision skill, browser debug skill | browser error/regression smokes, `make browser-reliability` when relevant |
| URL safety/private target policy | README, Session API, quickstart, release runbook/checklist, release/browser-debug/remote-auth/vision skills | release-service smoke, private URL negative proof, CI reliability smoke with temp config override |
| Release/deploy workflow | release runbook, proof checklist, release skill | release-service smoke, CI watch proof |
| Repo-local skill | README skill list, `docs/SKILL_SOURCE_OF_TRUTH_POLICY.md`, sibling skill references | docs completeness, relevant workflow gates |

## Docs maintenance workflow

1. Identify the canonical behavior in code, tests, or live smoke output.
2. Update the narrowest docs first.
3. Update README documentation map or skill list if a new public doc/skill exists.
4. Update parity matrix/checkers if an exposed surface changed.
5. Update release proof docs when completion criteria changed.
6. Update bead notes with files changed and proof commands.
7. Commit a small conventional commit.

## Drift and completeness gates

Run after docs-affecting work:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-docs-completeness.py'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-tool-parity.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-examples.py'
```

Add targeted gates based on the changed surface:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-mcp-tool-routes.sh'
```

## Beads-to-docs workflow

When closing a docs or skill bead:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd update <id> --notes "Changed docs/files; proof commands passed; any known caveat."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd close <id> --reason "Completed: <short outcome>."'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bd sync'
```

Do not use bead notes as the only durable docs. If behavior affects other agents or public usage, update repository docs/skills in the same change.

## Release-proof report checklist

Include:

- Task/bead id.
- Public docs changed.
- Skills changed.
- Parity/drift gates run.
- Tests/smokes run.
- Evidence artifacts or handles.
- Known caveats and next ready bead.

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| Docs gate fails | missing required public surface needle | Update named doc with real behavior; avoid weakening gate unless canonical surface changed. |
| Parity gate fails | HTTP/Pi/MCP/CLI surfaces drifted | Update tool route/schema or docs consistently. |
| Packet drift fails | schema/docs/examples/skills out of sync | Update packet docs, fixtures, and drift needles together. |
| Skill says stale workflow | repo-local skill not updated with code/doc change | Edit `.pi/skills/<name>/SKILL.md`, then optionally sync global copy. |
| README misses new doc | documentation map not updated | Add concise map row near related docs. |

## Final answer shape

Status: docs/skill maintenance complete or in progress.
Next action: next bead or validation step.
Blocker: only if a gate remains red or a risky action needs operator approval.
