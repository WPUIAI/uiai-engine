# UIAI-COCKPIT-003 rollout and support

Status: production shell enabled; no migration flag remains.

## Single-shell boundary

The sole Cockpit shell is `apps/cockpit/src/routes/+layout.svelte`, backed by the versioned workspace manifest. The former primitive navigator, scope-chip strip, Phase 0 card-grid home, and process ribbon are not production paths. Phase 0 card contracts remain as data and are mapped by `phase0-card-placement.ts`.

## Settings migration

`sidebar-preferences.ts` reads the current versioned preference envelope, migrates legacy values through bounded defaults, and preserves reset/recovery behavior. Unknown, malformed, or future-version values fail to defaults rather than creating a parallel shell.

## Support diagnostics

1. Run `npm --prefix apps/cockpit run cockpit:gate`.
2. Inspect Context Control for project/continuity authority.
3. Inspect `/nodes-services?view=uiai-engine` for entitlement, protected-worker, capsule, engine, and browser posture.
4. Inspect `/capabilities` for manifest and signed capability metadata.
5. Preserve `/evidence` and local artifacts during recovery-only operation.
6. Use `/help` for keyboard and recovery routes.

## Release evidence

The gate proves type diagnostics, unit/contract/route/visual-state/performance tests, manifest/backend smoke, screenshot baselines, and production build. GitHub's existing Cockpit Web build invokes the blocking test/smoke build script.
