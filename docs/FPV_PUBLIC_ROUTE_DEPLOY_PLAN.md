# FPV Public Route Deployment Plan

Last updated: 2026-06-12

## Current status

Cloudflare FPV front door is deployed:

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
  - `POST /m/{token}/control` for shares created with `controls=true`

## Goal

Expose the FPV PWA to an operator phone/browser without exposing the full local agent surface.

## Deployed public route shape

Use the narrow Cloudflare Tunnel ingress from `fpv.wpuiai.com` to the loopback engine:

| Public route | Upstream | Auth posture | Notes |
|---|---|---|---|
| `GET /m/{token}` | `http://127.0.0.1:7456/m/{token}` | public tokenized | PWA shell. Invalid/expired tokens return 404. |
| `GET /m/{token}/status` | `http://127.0.0.1:7456/m/{token}/status` | public tokenized | Read-only session metadata/diagnostics summary. |
| `GET /m/{token}/screenshot.jpg` | `http://127.0.0.1:7456/m/{token}/screenshot.jpg` | public tokenized | No-store JPEG poll frame. |
| `POST /m/{token}/control` | `http://127.0.0.1:7456/m/{token}/control` | public tokenized + controls flag | Only works when share was created with `controls=true`; audited server-side. |

Keep these routes private or authenticated:

| Route | Required posture | Reason |
|---|---|---|
| `/api/fpv/share` | loopback-public remote-auth | Share creation binds a live browser session and can enable controls. |
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
     -d '{"session_id":"SESSION_ID","expires_minutes":5}'
   ```
3. Open the public `https://fpv.wpuiai.com/m/{token}` URL from phone/non-local browser.
4. Confirm:
   - PWA page loads.
   - Screenshot refreshes.
   - Status shows expected URL/title.
   - Read-only share rejects `POST /control` with 403.
5. Create a `controls=true` share and confirm:
   - `message` action audits successfully.
   - A safe selector click or press action audits successfully.
   - Status shows `audit_count` incrementing.
6. Close the session and confirm public status returns `410 session closed`.

## Acceptance evidence

Capture these refs in the validation bead:

- Public URL tested, with token redacted.
- Browser/mobile read proof for `/m/{token}`.
- Diagnostics proof for failed requests/errors.
- Read-only control rejection proof.
- `controls=true` audit proof.
- Engine health after validation.

## Safety boundary

Do not proxy the full engine to the public internet. The FPV public surface is intentionally tokenized `/m/*`; all creator and agent surfaces remain loopback/authenticated.
