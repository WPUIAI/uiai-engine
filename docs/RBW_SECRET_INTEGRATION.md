# rbw Secret Integration

UIAI Engine config supports optional [`rbw`](https://github.com/doy/rbw) references for secrets that should not live directly in `config.yaml`, docs, examples, or Focusa state. Environment variables remain the most portable default; `rbw` is an optional adapter for deployments that already use a local Bitwarden vault.

This is a generic config-loader integration. It is not tied to one provider key, one endpoint, one host, or one operating system account.

## Supported syntax

Use normal environment references for the most portable deployments:

```yaml
wordpress:
  webhook_secret: "${WEBHOOK_SECRET}"
```

Use `rbw` references only when the operator has provisioned rbw for the service account and wants the secret to come from the local Bitwarden vault:

```yaml
wordpress:
  webhook_secret: "${rbw:UIAI Webhook Secret}"

ai:
  providers:
    openrouter:
      api_key: "${rbw:UIAI OpenRouter:api key}"
```

Forms:

- `${rbw:item}` → `rbw get item`
- `${rbw:item:field}` → `rbw get --field field item`

Set `UIAI_RBW_BIN` when the service user cannot find `rbw` in `PATH`:

```bash
export UIAI_RBW_BIN=/usr/local/bin/rbw
```

## Runtime behavior

- `rbw` is optional and only invoked when a config value explicitly uses an `${rbw:...}` reference.
- Missing or failing `rbw` references fail config loading with a clear error; deployments that need maximum portability should use `${ENV_VAR}` references instead.
- Secret values are expanded in memory for the config consumer; UIAI code must not log expanded config secrets.
- Use the service account's own `rbw` configuration/unlock state. Do not rely on a root-only `rbw` binary or vault for production services.

## Security notes

- Never commit literal API keys, bearer tokens, webhook secrets, license keys, cookies, or passwords.
- Prefer environment variables for portable deployments and `rbw` for operator-managed local/server vaults.
- Keep `UIAI_RBW_BIN` and rbw vault setup in systemd environment files or host provisioning, not in public docs with host-specific secret paths.
- Focusa captures evidence handles and summaries, not secret values.

## Verification

Config expansion is covered by `internal/config/secrets_test.go` using a fake `rbw` binary so CI never needs real vault access.
