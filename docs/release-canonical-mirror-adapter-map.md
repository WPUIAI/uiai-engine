# Core Canonical → Cockpit / Engine Mirror — Adapter Map (NO COMPROMISE)

**Authority:** `Startempire-Wire/focusa` `docs/current/RELEASE_RULES_2026-08-19.md` + `.github/workflows/release.yml` + `scripts/local-release-preflight.sh` + `scripts/stamp-menubar-version.py` is the **single source of truth**.

Cockpit and Engine **mirror it uncompromisingly** — same lifecycle, same failure-closed gates, same vocabulary. Stack differences are *adapters*, not relaxations. When Core improves, mirrors inherit automatically via shared script surface.

## Lifecycle (identical everywhere)

```
1. Version surfaces stamping (single writer, never hand-edit)
2. local preflight [--strict] (blocking, fails closed, ONE command ONE result)
3. pre-push hook (PREFLIGHT_FAST=1, <30s, blocks stale)
4. Commit + push main (deterministic CI proof)
5. Candidate-SHA receipt wait (exact SHA, no polling)
6. Tag (immutable, script-created)
7. Release workflow (identical gate order, waits deterministically)
8. Verify isLatest / assets / SHA256SUMS / journal
```

Vocabulary stays strict: **Release**=stable Latest 30+ assets, **Dev release**=vX.Y.Z-dev prerelease full matrix, **Tag ≠ Release**, **No partial**.

## What is 1:1 (no adapter — verbatim)

| Core gate | Mirror | Why identical |
|-----------|--------|---------------|
| `open-issue-release-gate` (label `release-gate:*` blocks tag) | same job, filter `cockpit-v*` / `engine-v*` | same durability |
| `pull-request-release-gate` (queue + tag inclusion `git merge-base --is-ancestor`) | same | same |
| `version-policy` via `scripts/next-version.py` | same `next-version.py` ported or shared | same monotonicity |
| `tag-ci-proof` wait loop 120×10s for stamped SHA CI | same | same determinism |
| `require_success WithWait` for slow matrix | Cockpit waits for `Spec132` equivalent? Engine waits for `go test` matrix | same pattern |
| `final-release-gap-gate` probe | Cockpit/EC use adapted gap probe | same closure |
| `checksums` SHA256 + sigstore + manifest + intelligence packet + `isLatest` verify | same (SHA256SUMS.txt + cosign) | same trust |
| Script-owned notes `generate-release-notes.py` | same file, repo-aware branch already in script | same |
| Journal `plan/progress/benchmark` via `canonical-release-journal.py` | same, project_id `uiai-engine` | same inheritance |

## Adapters (stack-catered, not weakened)

| Core surface | Focusa adapter | Cockpit adapter | Engine adapter | Inheritance |
|--------------|----------------|-----------------|----------------|-------------|
| **Stamp surfaces** | `stamp-menubar-version.py` 16 files + Cargo.lock + agent-card + manifest | `stamp-uiai-version.py --cockpit-only` (package.json, tauri.conf, Cargo.toml/Cargo.lock, version.ts, manifest) | `stamp-uiai-version.py --engine-only` (cmd/uiai-engine/main.go + manifest) | Single writer each track; manifest recompute + source_commit + generated_at identical logic; future Core stamp changes ported via shared `regen_manifest()` |
| **Verify surfaces** | `verify-version-surfaces.py vX.Y.Z` | `verify-uiai-version-surfaces.py cockpit-vX.Y.Z` | same `engine-vX.Y.Z` | Same mismatch detection |
| **Preflight FAST** | `local-release-preflight.sh` PREFLIGHT_FAST=1: Windows lint + surfaces + parity + manifest ancestor + fmt | `local-uiai-preflight.sh` PREFLIGHT_FAST=1: same but `go fmt` + `npm check` + manifest ancestor (already mirrors) | same | Shared gate shape; improvement = update both scripts from Core template |
| **Preflight STRICT** | + gap gate + `FOCUSA_TEST_MODE=1` spec gates | + `go test ./...` + `check-docs-completeness` + `check-tool-parity` + cockpit `npm check` (already) | same, no cockpit check | Strict always runs before tag |
| **Build matrix** | `rust-release` 7 targets + `tauri-build` | `signed-macos-build x86_64+aarch64` (already dual-arch merge via jq) + `linux-binary` | `go build linux/amd64 CGO_ENABLED=0 -trimpath -ldflags version/buildTime` | Matrix stays deterministic; Core improvement (e.g., new target) propagates as new matrix entry |
| **Signing** | `TAURI_SIGNING_PRIVATE_KEY` mandatory | same (already mandatory) | cosign for engine binary (already) + sigstore via checksums job | No unsigned path |
| **Manifest** | `docs/contracts/spec141/generated-capability-v2/distribution-manifest.json` schema `focusa.distribution_manifest.v1` | `docs/distribution-manifest.json` schema `uiai.distribution_manifest.v1` | shared with cockpit (same file, versioned per track) | Same freshness rule: STRICT requires HEAD/parent, FAST allows ancestor |
| **CI** | `ci.yml` 5 jobs: commit-msg, rust, menubar, release-automation-static, spec-gates | `ci` via cockpit-release `checks` + engine-release `go test` → to be unified into `.github/workflows/ci.yml` for uiai-engine with same 5-job shape | same | Azure mirror resilient `rg` install etc inherited |
| **Pre-push** | `.git/hooks/pre-push` runs `validate-commit-messages` + `local-release-preflight PREFLIGHT_FAST=1` | `scripts/git-hooks-pre-push` already exists in uiai-engine (to be wired as hook) | same | Hook is generated, not hand-written |

## Inheritance mechanism (how Core improvement auto-flows)

1. Core scripts are canonical: `generate-release-notes.py` already repo-aware (`is_uiai`, `is_cockpit`); mirror reuses it verbatim — no fork.
2. `stamp-*` and `local-*-preflight.sh` share `regen_manifest()` and lint logic from Core template — diff is file list only. Update Core template → patch both stamp/preflight files (3-line file-list change).
3. Workflow jobs `open-issue`, `pull-request`, `tag-ci-proof`, `checksums` are copied verbatim with tag-prefix filter (`v*` vs `cockpit-v*` vs `engine-v*`). Future Core job change = same copy.
4. Journal: same `scripts/canonical-release-journal.py` with `project_id` param. No custom journal.

## Non-negotiables (NO COMPROMISE)

- No `--no-verify`, no hand-edit manifest, no partial release without written operator approval.
- FAST preflight <30s blocks push; STRICT blocks tag. Both fail closed.
- Every tag waits deterministically for its exact SHA CI; no `gh run rerun --failed` guessing.
- Script-owned notes only; agent never writes body.
- Signing mandatory; unsigned build fails.

## Current delta (what this GO implements)

- [ ] Wire `uiai-engine/.github/workflows/cockpit-release.yml`: add `open-issue-release-gate`, `pull-request-release-gate`, `version-policy`, `tag-ci-proof`+`require_success` wait, tighten to Core order.
- [ ] Wire `uiai-engine/.github/workflows/engine-release.yml`: same gates, engine tag prefix.
- [ ] Harden `scripts/stamp-uiai-version.py` manifest recompute to always sha256 (currently skips missing artifacts) → match Core strict recompute.
- [ ] Harden `scripts/local-uiai-preflight.sh` freshness to STRICT HEAD/parent rule (currently only ancestry) → match Core.
- [ ] Ensure `scripts/git-hooks-pre-push` is installed as `.git/hooks/pre-push` (PREFLIGHT_FAST).
- [ ] Verify `scripts/generate-release-notes.py` remains single writer (already true).

Result: releases become frictionless — `bash scripts/create-cockpit-dev-release-tag.sh --push` and `engine` equivalent behave exactly like `bash scripts/create-dev-release-tag.sh --push`: push, wait deterministically, publish, verify, done.
