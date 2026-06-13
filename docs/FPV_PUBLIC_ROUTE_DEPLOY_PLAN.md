# FPV Public Route Deployment Plan

Last updated: 2026-06-12

## Current status

External/mobile FPV validation is blocked by deployment topology, not by the FPV PWA implementation:

- `uiai-engine` listens on `127.0.0.1:7456` only.
- Public `https://wpuiai.com/m/not-a-real-token` currently returns the WordPress 404 page.
- Local FPV routes already pass smoke tests:
  - `POST /api/fpv/share`
  - `GET /m/{token}`
  - `GET /m/{token}/status`
  - `GET /m/{token}/screenshot.jpg`
  - `POST /m/{token}/control` for shares created with `controls=true`

## Goal

Expose the FPV PWA to an operator phone/browser without exposing the full local agent surface.

## Recommended public route shape

Use a narrow reverse proxy from the public host to the loopback engine:

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

### Option A — LiteSpeed/Apache reverse-proxy path on `wpuiai.com`

Map only `/m/` to `127.0.0.1:7456`.

Required checks before enabling:

1. Confirm cPanel/LiteSpeed proxy syntax for this host.
2. Preserve WordPress routes outside `/m/`.
3. Set no-cache/no-store for `/m/*` responses if supported at proxy layer.
4. Verify invalid token still returns engine JSON 404 or safe PWA error, not WordPress.
5. Verify `POST /m/{token}/control` is not blocked by WAF when `controls=true` is intentionally used.

### Option B — Dedicated subdomain

Create a dedicated hostname, e.g. `fpv.wpuiai.com`, and proxy all requests on that hostname to `127.0.0.1:7456`, but allow only `/m/*` at the proxy layer.

This is cleaner than path-sharing with WordPress and reduces route collision risk.

### Option C — Temporary tunnel for validation only

Use a short-lived operator-approved tunnel that exposes only `/m/*` for a single validation run. Do not expose `/api/session*` or broad engine routes through the tunnel.

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
3. Open the public `/m/{token}` URL from phone/non-local browser.
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
