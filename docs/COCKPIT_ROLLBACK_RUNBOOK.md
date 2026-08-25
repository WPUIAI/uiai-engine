# Cockpit Rollback Runbook — focusa.cockpit.bundle_manifest.v1

## Trigger
- Failed `Deploy Cockpit` (`release_tag` mismatch, health timeout, audit schema fail) or operator-initiated.
- Auto-retry caps at 1 (cockpit-auto-retry-deploy.yml); rollback is explicit.

## Procedure
1. Identify last healthy tag: `git tag --list 'cockpit-v*' --sort=-v:refname | head -1` and `gh release view <tag> --json tagName`.
2. Redeploy: `gh workflow run "Deploy Cockpit" --ref main -f release_tag=<healthy-tag> -f asset_suffix=x86_64-unknown-linux-musl`
3. Require CI gate: `deploy-cockpit.yml` blocks until `ci.yml` success for `tag_sha` (github-script check).
4. Audit: row appended to `release-proof/cockpit/audit.jsonl` with `event=failure|addition`, linked_run=<previous_run,retry_run>.
5. Verify: `health_url` 200 + `plutil -lint` on bundle + `audit-schema.py validate`.

## Rollback Window
- Keep 5 backups (`DEPLOY_BACKUP_KEEP`), re-install from `release-metadata.json` `focusa.cockpit.release.v1`.

## Dry-run
`gh workflow run "Deploy Cockpit" --ref main -f release_tag=<tag> -f dry_run=true`
