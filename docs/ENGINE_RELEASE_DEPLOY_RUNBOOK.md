# UIAI Engine parent release + OVH worker deploy runbook

Status: active runbook
Scope: parent UIAI Engine worker release, distinct from Cockpit child release

## Relationship to Cockpit

Cockpit is a child application under `apps/cockpit` and continues to use the existing Cockpit workflows (`cockpit-release.yml`, `deploy-cockpit.yml`, and related auto-retry/audit workflows). The parent UIAI Engine worker has its own release/deploy workflow:

- `.github/workflows/engine-release.yml`
- `scripts/deploy-engine-ovh.sh`

## Release tags

Parent engine release tags must start with one of:

- `engine-v*`
- `uiai-engine-v*`

Cockpit release tags remain `cockpit-v*`.

## What parent release does

On a parent engine tag push, `UIAI Engine Release + OVH Deploy`:

1. Checks out the repo.
2. Runs `go test ./...`.
3. Builds `./cmd/uiai-engine` as `uiai-engine-linux-amd64`.
4. Embeds version/build time through `-ldflags`.
5. Creates or updates the GitHub Release with:
   - `uiai-engine-linux-amd64`
   - `uiai-engine-linux-amd64.sha256`
6. Deploys the same artifact to the OVH UIAI worker service.
7. Verifies hash, service health, and local browser smoke on OVH.

## OVH deploy target

Default target values:

- Host: configured by GitHub secret `OVH_UIAI_SSH_HOST`.
- User: root by default in the workflow SSH config.
- Install root: `/home/wpuiai/uiai-engine`.
- Service: `uiai-engine-ovh.service`.
- Health URL: `http://127.0.0.1:7456/v1/health`.

Optional repository/environment variables:

- `OVH_UIAI_INSTALL_ROOT`
- `OVH_UIAI_SERVICE_NAME`
- `OVH_UIAI_HEALTH_URL`

Required GitHub secrets:

- `OVH_UIAI_SSH_KEY`
- `OVH_UIAI_SSH_HOST`

Optional GitHub secrets:

- `OVH_UIAI_SSH_USER`
- `OVH_UIAI_SSH_PORT`

Do not commit keys or tokens.

## Manual dry run

```bash
ASSET_PATH=dist/uiai-engine-linux-amd64 \
REMOTE_HOST=vps-d09121de \
REMOTE_USER=root \
DRY_RUN=1 \
scripts/deploy-engine-ovh.sh
```

## Manual deploy/redeploy

```bash
ASSET_PATH=dist/uiai-engine-linux-amd64 \
REMOTE_HOST=vps-d09121de \
REMOTE_USER=root \
RELEASE_TAG=engine-vX.Y.Z \
scripts/deploy-engine-ovh.sh
```

## Proof checks

The deploy script verifies:

- Remote install root exists.
- Target service exists.
- Uploaded binary SHA256 matches local artifact.
- Previous binary is backed up under `backups/`.
- `systemctl restart uiai-engine-ovh.service` succeeds.
- Installed binary hash is printed.
- Protected local health returns HTTP `200` or expected protected `401`.
- Local OVH browser smoke opens `https://example.com`, confirms screenshot presence, and closes the session.

## Public domains

This workflow **does not** rewire public hostnames. It deploys the operable OVH worker only. Public route cutover for `ai.wpuiai.com` / `fpv.wpuiai.com` remains a separate W14a proof window.

## Rollback

On OVH, prior binaries are backed up under:

```text
/home/wpuiai/uiai-engine/backups/
```

Rollback pattern:

```bash
ssh ovh 'cp -a /home/wpuiai/uiai-engine/backups/uiai-engine.TAG.TIMESTAMP /home/wpuiai/uiai-engine/uiai-engine && systemctl restart uiai-engine-ovh.service'
```

Then rerun local health and browser smoke.
