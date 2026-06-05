# UIAI Skill Source-of-Truth Policy

UIAI Engine project-specific Pi skills are versioned project artifacts. Their canonical source lives in this repository under `.pi/skills/`.

## Canonical location

| Skill | Canonical file | Purpose |
|---|---|---|
| `uiai-agent` | `.pi/skills/uiai-agent/SKILL.md` | Main UIAI agent workflow: discovery, browser/search/read, diagnostics, Focusa packet handoff, parity gates, release-safe cleanup. |
| `uiai-focusa-packet` | `.pi/skills/uiai-focusa-packet/SKILL.md` | Focusa research diagnostics packet playbook: compose, route `args_preview`, validate redaction/budget, cleanup. |
| `uiai-release` | `.pi/skills/uiai-release/SKILL.md` | Release/deploy/push/CI proof workflow and service smoke bundle. |
| `uiai-mcp` | `.pi/skills/uiai-mcp/SKILL.md` | MCP bridge setup, discovery, cache/reconnect, route parity, and tool routing workflow. |
| `uiai-remote-auth` | `.pi/skills/uiai-remote-auth/SKILL.md` | Authenticated remote/tunnel setup, safe env vars, and loopback-public boundary workflow. |
| `uiai-ci-debug` | `.pi/skills/uiai-ci-debug/SKILL.md` | GitHub Actions log/artifact failure diagnostics and local reproduction. |
| `uiai-browser-debug` | `.pi/skills/uiai-browser-debug/SKILL.md` | Diagnostics-first browser/session debugging and Focusa intake. |
| `vision` | `.pi/skills/vision/SKILL.md` | Browser/vision/session API reference workflow for screenshots, @refs, diagnostics, and packets. |

## Global copies are convenience installs

Global skills under `~/.pi/skills/` are convenience copies only. They are not source of truth for UIAI project behavior.

Rules:

- Edit repo-local `.pi/skills/<name>/SKILL.md` first.
- Commit repo-local skill changes with the related code/docs/smoke changes.
- Copy repo-local skills to `~/.pi/skills/` only after the repo version is correct.
- Do not copy a stale global skill into the repo unless manually reconciling it against current code/docs.
- Treat differences between global and repo-local skills as drift until reviewed.

## When a skill belongs in the repo

Add or update a repo-local skill when the workflow is UIAI-specific and one or more are true:

- It references UIAI routes, scripts, docs, services, evidence handles, or repo paths.
- It depends on project-specific auth, Focusa packet, MCP, Pi extension, CLI, CI, or release behavior.
- It should travel with the repository for other agents or future Pi sessions.
- It is required by `scripts/check-docs-completeness.py` or the release proof checklist.

Keep a skill global-only when it is generic and does not reference UIAI project internals.

## Update workflow

1. Edit the repo-local skill.
2. Update paired docs and smokes when behavior changed.
3. Run gates:

```bash
scripts/check-docs-completeness.py
scripts/check-tool-parity.sh
scripts/check-focusa-packet-drift.sh
scripts/smoke-pi-extension-registration.sh
```

4. Commit repo-local changes.
5. Optionally sync to global convenience copies.

## Optional sync commands

From the UIAI repo root:

```bash
mkdir -p ~/.pi/skills
cp -a .pi/skills/uiai-agent ~/.pi/skills/
cp -a .pi/skills/uiai-focusa-packet ~/.pi/skills/
cp -a .pi/skills/uiai-release ~/.pi/skills/
cp -a .pi/skills/uiai-mcp ~/.pi/skills/
cp -a .pi/skills/uiai-remote-auth ~/.pi/skills/
cp -a .pi/skills/uiai-ci-debug ~/.pi/skills/
cp -a .pi/skills/uiai-browser-debug ~/.pi/skills/
cp -a .pi/skills/vision ~/.pi/skills/
```

Before overwriting global copies, inspect local operator customizations if any:

```bash
diff -ru ~/.pi/skills/uiai-agent .pi/skills/uiai-agent || true
```

If the global copy has useful local additions, port those additions into the repo-local skill deliberately, then sync outward.

## Release proof expectation

A release that changes repo-local skills should cite:

- changed `.pi/skills/*/SKILL.md` paths,
- related docs and scripts,
- `scripts/check-docs-completeness.py`,
- `scripts/smoke-pi-extension-registration.sh`,
- any workflow-specific proof artifact.

## Related docs

- [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md)
- [Agent UX Cookbook](AGENT_UX_COOKBOOK.md)
- [Agent Experience Roadmap Implementation Summary](AGENT_EXPERIENCE_ROADMAP_IMPLEMENTATION_SUMMARY.md)
- [Agent Surface Release Proof Checklist](AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md)
