# Public read, private control — UIAI exposure posture (2026-11-02)

Owner decision: **anyone may read published evidence; nobody but the operator and
authorized agents may write or execute.** This note records the exact configuration so
it can be restored, audited, or rolled back without rediscovering it.

## Why

`docs/REMOTE_AUTH_EXAMPLES.md` ("Remote/tunnel deployment") requires TLS, explicit
trusted-proxy/client-IP configuration, caller authentication, entitlement enforcement,
origin/CORS rules, limits, session ownership, and — verbatim — *"no proxy path that
turns remote traffic into loopback permission."* The engine binds `127.0.0.1` and has no
auth middleware, so publishing its control API through a tunnel handed the whole world
loopback authority: an unauthenticated caller could list live sessions and drive the
pool browser. Evidence records themselves are `access: public_safe` and are meant to be
readable (`uiai.epwa_delivery.v1`).

## Posture

| Surface | Reachable from | Route |
| --- | --- | --- |
| Evidence record pages, assets, portable zip (`/e/*`) | the public internet | Cloudflare tunnel → `localhost:7456` |
| Evidence share read API (`/api/screenshot/share/*`) | the public internet | same |
| FPV viewer (`/m/*` on `fpv.wpuiai.com`) | the public internet (tokenized) | same |
| Control API (sessions, capture, publish, search, tools) | loopback on the KH host only | SSH tunnel `127.0.0.1:7458` → OVH `127.0.0.1:7456` |
| Anything else on `ai.wpuiai.com` / `fpv.wpuiai.com` | nowhere | `http_status:404` |

### 1. Tunnel ingress — `/etc/cloudflared/wpuiai.yml`

Only read paths are routed. Any hostname/path not listed falls through to the
catch-all `http_status:404`, so an `/api/session` request from the internet is answered
by the tunnel itself and never reaches the engine.

```yaml
  - hostname: ai.wpuiai.com
    path: /e/*
    service: http://localhost:7456
  - hostname: ai.wpuiai.com
    path: /api/screenshot/share/*
    service: http://localhost:7456
  - hostname: ai.wpuiai.com
    service: http_status:404
  - hostname: fpv.wpuiai.com
    path: /m/*
    service: http://localhost:7456
  - hostname: fpv.wpuiai.com
    service: http_status:404
```

### 2. Public origin for record URLs — engine drop-in

Publishing now happens over a private path, so the request can no longer express the
public origin. `canonicalEPWABase` also requires `https`. The contract's own escape
hatch (`docs/106r`, §2 item 1: *optional operator override*) supplies it:

```ini
# /etc/systemd/system/uiai-engine-ovh.service.d/epwa-public-base.conf
# /etc/systemd/system/uiai-engine-ovh-w2.service.d/epwa-public-base.conf
[Service]
Environment=UIAI_EPWA_PUBLIC_BASE_URL=https://ai.wpuiai.com
```

### 3. Private control path — `/etc/systemd/system/uiai-control-tunnel.service`

```ini
[Service]
ExecStart=/usr/bin/ssh -o BatchMode=yes -o StrictHostKeyChecking=yes -o ConnectTimeout=10 \
  -o ExitOnForwardFailure=yes -o ServerAliveInterval=15 -o ServerAliveCountMax=3 -N \
  -L 127.0.0.1:7458:127.0.0.1:7456 ovh
Restart=always
```

Loopback-only bind, so no firewall opening and no public listener. Agent tooling points
at `http://127.0.0.1:7458` (`UIAI_ENGINE_URL` in the Pi MCP config).

## Verification (observed 2026-11-02, from the OVH host — not the tunnel host, no credentials)

```
public record page      https://ai.wpuiai.com/e/e351c3d22b4d/        200
public image bytes      .../e/e351c3d22b4d/screenshot.png           200  72985 bytes
public portable zip     .../e/e351c3d22b4d.zip                      200
public POST /api/screenshot (write attempt)                         404
public GET  /api/session    (read of live sessions)                 404
private publish         POST http://127.0.0.1:7458/api/screenshot   state=ready,
                        scope=complete, record https://ai.wpuiai.com/e/e351c3d22b4d/
private tool surface    GET  http://127.0.0.1:7458/api/tools/mcp     42 tools
```

## Rollback

- Restore the tunnel routes from `/etc/cloudflared/wpuiai.yml.bak-<timestamp>` and
  `systemctl restart cloudflared-wpuiai`.
- `systemctl disable --now uiai-control-tunnel` and point tooling back at a reachable
  engine URL.
- Remove the two `epwa-public-base.conf` drop-ins, `systemctl daemon-reload`, restart
  both engine units.

Rolling back re-exposes the unauthged control API and therefore re-opens the gap in
`docs/REMOTE_AUTH_EXAMPLES.md`. Do not do it without an owner decision.
