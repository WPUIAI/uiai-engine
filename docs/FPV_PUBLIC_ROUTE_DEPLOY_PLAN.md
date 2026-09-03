# FPV Public Route Deployment Plan

Last updated: 2026-09-03

## Current status

The Cloudflare FPV front door is deployed. Security correction candidate `fix/147-private-session-open` is source-only until canonical release, installation, and browser proof; the installed public runtime must not be assumed to have these controls yet:

- `fpv.wpuiai.com` is a proxied Cloudflare CNAME to the existing `wpuiai` Cloudflare Tunnel.
- `/etc/cloudflared/wpuiai.yml` routes only `fpv.wpuiai.com/m/*` to `http://localhost:7456`.
- `fpv.wpuiai.com` paths outside `/m/*` return Cloudflare Tunnel `404`, so `/api/*` is not exposed through the FPV hostname.
- `uiai-engine` still listens on `127.0.0.1:7456` only.
- Public `https://wpuiai.com/m/*` is not the supported FPV route; use `https://fpv.wpuiai.com/m/{token}`.
- FPV routes pass local and Cloudflare validation:
  - `POST /api/fpv/share`
  - `GET /m/{token}`
  - `GET /m/{token}/status`
  - `GET /m/{token}/screenshot.jpg`
  - `POST /m/{token}/control` is fail-closed; public control-share issuance is disabled until a separate governed confirmation contract is implemented

## Goal

Expose the FPV PWA to an operator phone/browser without exposing the full local agent surface.

## Deployed public route shape

Use the narrow Cloudflare Tunnel ingress from `fpv.wpuiai.com` to the loopback engine:

| Public route | Upstream | Auth posture | Notes |
|---|---|---|---|
| `GET /m/{token}` | `http://127.0.0.1:7456/m/{token}` | public tokenized | PWA shell. Invalid/expired tokens return 404. |
| `GET /m/{token}/status` | `http://127.0.0.1:7456/m/{token}/status` | public tokenized | Read-only session metadata/diagnostics summary. |
| `GET /m/{token}/screenshot.jpg` | `http://127.0.0.1:7456/m/{token}/screenshot.jpg` | public tokenized | No-store JPEG poll frame. |
| `POST /m/{token}/control` | `http://127.0.0.1:7456/m/{token}/control` | disabled | New and legacy public capabilities are read-only. Governed control confirmation is not implemented and must fail closed. |

Keep these routes private or authenticated:

| Route | Required posture | Reason |
|---|---|---|
| `/api/fpv/share` | loopback-public remote-auth | Explicit share creation binds one live session and exact non-sensitive origin. It must never be called by session creation and cannot issue control capabilities. |
| `/api/session*` | loopback-public remote-auth | Full browser control surface. |
| `/api/screenshot*`, `/api/markdown*`, `/api/search*`, `/api/agent/*` | loopback-public remote-auth | Agent surfaces; do not expose anonymously. |

## Safe deployment options

### Implemented option — Cloudflare Tunnel dedicated hostname

`fpv.wpuiai.com` is routed through Cloudflare Tunnel with path-gated ingress:

```yaml
- hostname: fpv.wpuiai.com
  path: /m/*
  service: http://localhost:7456
- hostname: fpv.wpuiai.com
  service: http_status:404
```

This is cleaner than path-sharing with WordPress and reduces route collision risk. Do not add a broad `fpv.wpuiai.com -> localhost:7456` rule without a path gate.

## Validation checklist

After proxy is enabled:

1. Create a local browser session:
   ```bash
   curl -fsS -X POST http://127.0.0.1:7456/api/session \
     -H 'Content-Type: application/json' \
     -d '{"url":"https://example.com","width":390,"height":844}'
   ```
2. Create a read-only FPV share:
   ```bash
   curl -fsS -X POST http://127.0.0.1:7456/api/fpv/share \
     -H 'Content-Type: application/json' \
     -d '{"session_id":"SESSION_ID","expected_origin":"https://example.com","explicit_consent_ref":"consent:operator:SHARE_ID","expires_minutes":5,"max_views":1,"one_time":true}'
   ```
3. Open the public `https://fpv.wpuiai.com/m/{token}` URL from phone/non-local browser.
4. Confirm:
   - PWA page loads.
   - Screenshot refreshes.
   - Status shows expected URL/title.
   - Read-only share rejects `POST /control` with 403.
5. Confirm a `controls=true` request is denied and no capability is persisted.
6. Confirm auth, privacy, payment, and health origins are denied before token creation.
7. Confirm the share has a maximum 60-minute TTL, bounded views, a 192-bit URL-safe capability, and exact origin/session binding.
8. Close the session and confirm every derived share is atomically revoked; persistence failure must be reported rather than presented as successful revocation.

## Acceptance evidence

Capture these refs in the validation bead:

- Public URL tested, with token redacted.
- Browser/mobile read proof for `/m/{token}`.
- Diagnostics proof for failed requests/errors.
- Read-only control rejection proof.
- Control-issuance denial proof.
- Sensitive-origin denial matrix.
- Concurrent max-view enforcement and token-redaction proof.
- Session-close revocation plus durable-registry failure-injection proof.
- Engine health after validation.

## Safety boundary

Do not proxy the full engine to the public internet. The FPV public surface is intentionally tokenized `/m/*`; all creator and agent surfaces remain loopback/authenticated. Browser/session creation is private by default. Public sharing is a separate explicit operation, sensitive origins are non-shareable, and public controls remain disabled until governed confirmation exists.
