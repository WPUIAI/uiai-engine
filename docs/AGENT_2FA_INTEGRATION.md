# Agent 2FA Integration

UIAI Engine provides a portable, optional 2FA helper so agents can complete operator-approved browser logins when a site asks for a TOTP code.

The design goal is portability:

- no hard dependency on Aegis, Bitwarden, rbw, or this server;
- native TOTP profiles work anywhere with an `otpauth://totp/...` URL or base32 secret supplied through env/rbw/config expansion;
- Aegis Authenticator vault support is an optional command adapter using [`aegis-rs`](https://github.com/Granddave/aegis-rs), which is built on [`aegis-vault-utils`](https://github.com/Granddave/aegis-vault-utils/);
- UIAI never logs TOTP secrets and returns only the short-lived code plus expiry metadata.

> Note: the originally discussed `Granddave/aegis-rsn` URL was not reachable during implementation. The public Granddave projects found are `aegis-rs` and `aegis-vault-utils`.

## HTTP API

```http
POST /api/2fa/code
Content-Type: application/json

{
  "profile": "github",
  "issuer": "GitHub",
  "name": "agent@example.com"
}
```

Response:

```json
{
  "profile": "github",
  "provider": "aegis-rs",
  "code": "123456",
  "expires_in": 17,
  "period": 30,
  "digits": 6,
  "issuer": "GitHub",
  "name": "agent@example.com"
}
```

## Pi tool

Agents can use the Pi tool:

```text
uiai_2fa_code profile="github" issuer="GitHub" name="agent@example.com"
```

Use this only when the browser workflow reaches a legitimate 2FA prompt and the operator/project has configured the profile.

## Native portable TOTP profile

Use this when you have an `otpauth://` URI or base32 TOTP secret from any authenticator. Combine with env vars or rbw config references so secrets are not committed.

```yaml
two_factor:
  enabled: true
  profiles:
    github:
      provider: totp
      otpauth_url: "${rbw:GitHub TOTP:otpauth}"
```

Or:

```yaml
two_factor:
  enabled: true
  profiles:
    github:
      provider: totp
      secret: "${GITHUB_TOTP_SECRET}"
      issuer: "GitHub"
      name: "agent@example.com"
      digits: 6
      period: 30
      algorithm: SHA1
```

## Optional Aegis vault profile

Local VPS note: this server currently has `Granddave/aegis-rs` v0.4.1 installed as `/usr/local/bin/aegis`; portable configs should use `command: "aegis"` or an explicit deployment-specific path.


Use this when the operator keeps 2FA entries in an exported Aegis Authenticator backup vault and has installed `aegis` CLI from `Granddave/aegis-rs` for the UIAI service user.

```yaml
two_factor:
  enabled: true
  profiles:
    github-aegis:
      provider: aegis-rs
      command: "aegis"   # optional; defaults to aegis in PATH; set /usr/local/bin/aegis on this VPS if desired
      vault_file: "${AEGIS_VAULT_FILE}"
      password: "${rbw:Aegis Vault Password}"
      issuer: "GitHub"
      name: "agent@example.com"
```

UIAI invokes the command without a shell:

```bash
aegis --json --issuer GitHub --name agent@example.com --password <password> <vault_file>
```

A password file is also supported:

```yaml
password_file: "${AEGIS_PASSWORD_FILE}"
```

## Security and portability rules

- Keep `two_factor.enabled: false` unless this deployment needs agent-assisted 2FA.
- Prefer native TOTP for maximum portability; use Aegis only where the `aegis` CLI from `Granddave/aegis-rs` is installed and maintained by the operator.
- Do not commit vault files, TOTP secrets, passwords, recovery codes, or generated OTP codes.
- Do not paste generated codes into Focusa state, docs, issue comments, or transcripts beyond the immediate browser login step.
- Use env vars or `rbw` references for secrets; see [`RBW_SECRET_INTEGRATION.md`](RBW_SECRET_INTEGRATION.md).
- Aegis support shells out to the CLI instead of linking GPL code into UIAI Engine.

## Tests

Native TOTP generation is tested with RFC 6238 vectors. Aegis command integration is tested with a fake `aegis` binary so CI does not need a real Aegis vault or real secrets.
