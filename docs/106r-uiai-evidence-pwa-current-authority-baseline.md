# 106r — UIAI Evidence PWA Current Authority Baseline

**CallGraph node:** CG-01 — Freeze current authority baseline. **Captured:** 2026-08-30 UTC. **Parent:** Issue #133 — EPWA Completion CallGraph. This baseline records authority and compatibility; it does not accept an implementation, review, completion, provider-close, or settlement result.

## Canonical compatibility floor

- UIAI `origin/main`: `6003418` — canonical EPWA 106 spec family merged over T01–T05.
- T01 — immutable artifact: `f9e73de`.
- T02 — crash-safe store/index: `7bad720`.
- T03 — hostile-content inspection: `b7c0af4`.
- T04 — capture/coverage integrity: `48f4bbd`.
- T05 — identity/time/custody: `e9736e2`.
- Completion graph definition candidate: `99da981`; graph ID `callgraph:epwa-completion-v1`; 32 nodes; 55 acyclic dependency edges.

A consumer may not assume a contract newer than its declared base. Stacked candidates remain unavailable to `main` consumers until their exact ancestry is adopted.

## Candidate inventory

| ID + descriptor | Head | Base | State at capture | Authority posture |
|---|---|---|---|---|
| PR #121 — Evidence Judge contracts | `2a7a16f` | `main` | open, mergeable | implementation candidate only |
| PR #122 — Evidence PWA projection contract | `52d6d52` | `main` | open, mergeable | implementation candidate only |
| PR #123 — Item-scoped Evidence Action contracts | `e4c004a` | PR #121 branch | open, mergeable | implementation candidate only |
| PR #125 — Deterministic evidence derivative contracts | `c515a22` | PR #122 branch | open, mergeable | implementation candidate only |
| PR #128 — Automatic Screenshot Evidence Share Packets | `202572c` | PR #122 branch | open, mergeable | runtime candidate; settings authority included |
| PR #131 — Cockpit Evidence Share board/settings | `77f1e01` | PR #128 branch | open; mergeability pending | consumer candidate only |
| PR #132 — Canonical Evidence Share Settings | `287c69a` | PR #128 branch | merged into stack | not merged to `main` |
| PR #134 — EPWA Completion CallGraph | `99da981` | `main` | open, mergeable | planning/task authority candidate |

## Cross-surface inventory

- UIAI packet package/API/Pi tools: PR #128; Issue #126.
- Cockpit board/settings: PR #131; Issues #127 and #129.
- Canonical settings scope: Issue #130; implementation incorporated into PR #128 stack.
- Workforce Chrome viewer/settings: Focusa Issues #437/#439; tested local candidate `dbdbffad5`; publication blocked by Focusa Issue #442 release-manifest pre-push defect.
- Desktop Canvas/settings: Focusa Issues #438/#440; PR #441 contains connected settings but not the complete Canvas Evidence object.
- Veragensia preview: commits `4b10c7a` and `03aaf1f`; `https://os.focusa.dev/evidence/` is a public read-only visual fixture, explicitly noncanonical and not yet restart-safe or API-connected.

## Open task authority

- Issue #106 — Evidence PWA authority remains open.
- Issue #133 — EPWA Completion CallGraph is the execution parent.
- Nodes CG-02 through CG-32 remain non-terminal until their individual evidence and join conditions pass.
- No current PR satisfies its own independent join merely by being open, mergeable, tested, or published.

## Immutable authority boundaries

1. UIAI owns immutable evidence bytes, manifests, typed refs/digests, packet serving, and proof inputs.
2. Focusa owns mutable execution, review, reconciliation, Completion Authority, provider synchronization, and settlement.
3. Review approval is not completion. HTTP/provider success is not settlement. Artifact existence is not verification.
4. Public/static/offline copies are read-only. Public Veragensia is a credential-free demo trust class.
5. Unknown outcome reconciles against authoritative state before retry.
6. IDs are always accompanied by human-readable descriptors.

## Dirty-state and ownership exclusions

- Original `/home/wpuiai/uiai-engine` checkout is stale/dirty and read-only for this CallGraph; isolated worktrees own mutations.
- Original `/home/wirebot/focusa` checkout is detached/conflict-dirty and excluded.
- Hook-generated mutation to `cmd/uiai-browser-profile/main.go` is unrelated, restored after every push, and excluded from all EPWA candidates.
- `.beads/issues.jsonl`, release-owned generated manifests, Wordfence files, data/usage, node_modules, release proof, and concurrent agent work are excluded unless an exact later node authorizes them.
- No recovery code, nonrenewable credential, public-demo credential, or private operator path may enter evidence or automation.

## CG-01 done condition and evidence

- Canonical specs, commits, candidates, issues, surfaces, authority boundaries, compatibility floor, and dirty-state exclusions are recorded above.
- Evidence: committed hash of this document; PR #134; graph definition digest and validator output (`32 nodes`, `55 edges`, no duplicates/dangling dependencies/cycles).
- Independent CG-02 work must consume this committed baseline; it may not consume uncommitted worktree state.
