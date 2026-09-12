> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462492569
> Contract base: docs/106a-uiai-evidence-pwa-portability-responsive-contract-amendment.md
> Status: agent-first contract + human guide for shipped EPWA delivery (issue #106 workstream, 106r)

# EPWA delivery — agent-first contract, zero-config guarantee, human guide

**EPWA = Evidence Package Web Archive.** One call publishes a durable,
self-contained evidence package: a machine-readable delivery record plus a
portable `.zip` that renders offline on any host, any domain, forever — with
**zero mandatory configuration**.

---

## 0. Sixty-second summary (humans first, because the product is agent-first)

- You publish evidence once. You get back a **record** (JSON, the durable
  truth) and a **portable URL** (a `.zip` — the whole package, downloadable).
- The `.zip` is fully self-contained: relative paths only, embedded viewer,
  service worker, manifest, icon. **No UIAI hostname inside. Nothing to
  configure.** Unzip it anywhere — double-click `index.html`, or drop it on
  any static file server, any domain — and it works.
- If publishing cannot complete, you get a machine-readable **error record**
  (never silence): state `pending_reconcile`, a `recovery_ref` pointing at the
  exact reconcile path, and `retryable: true`.
- Every package carries a **truth notice**: artifact existence and EPWA
  delivery do not establish review, verification, completion, provider
  closure, settlement, or legal admissibility.

---

## 1. Agent contract (machine-first — this is the primary interface)

### 1.1 Endpoints (all mounted under `/api/screenshot`)

| Method | Path | Purpose | Notes |
|---|---|---|---|
| `POST` | `/api/screenshot` | Capture + publish in one call | Body: `{"url": ..., "width": ..., "height": ..., ...}`. Returns `uiai.session_visual_result.v2` with `epwa_delivery`. |
| `GET` | `/api/screenshot/health` | Liveness + delivery posture | Returns `status`, `service`, `uptime`. |
| `GET` | `/api/screenshot/artifact/{sha}` | Fetch raw artifact by digest | Content-addressed. |
| `GET` | `/api/screenshot/share` | List published packets | Newest first; each entry: `packet_id`, `artifact_url`, `portable_url`, `availability: "ready"`. |
| `GET` | `/api/screenshot/share/{id}` | **Delivery record / viewer (negotiated)** | Browsers (`Accept: text/html`) get the EPWA viewer **webpage**; agents (`Accept: application/json` or default `*/*`) get the durable JSON record. Same URL, both audiences. |
| `GET` | `/api/screenshot/share/{id}/portable.zip` | **Portable package** | Immutable, `ETag: "sha256:<digest>"`, `Cache-Control: public, max-age=31536000, immutable`, `Content-Disposition: attachment; filename="<id>.epwa.zip"`. |
| `GET` | `/api/screenshot/share/{id}/verify` | **Integrity verification** | Deterministic digest re-computation; returns `valid` + `issues[]`. |
| `GET` | `/api/screenshot/share/{id}/*` | Package assets (viewer shell, JS, manifest, icon) | Same-origin relative serving. |

Session-flow routes (`POST .../sessions`, `POST .../sessions/{id}/screenshot`,
navigate/type/hover/etc.) publish through the same EPWA path via
`writeSessionSnapshot` and embed `artifact_url` + `portable_url` in the
response when delivery state is `ready`.

### 1.2 Success response schema — `uiai.session_visual_result.v2`

```json
{
  "schema": "uiai.session_visual_result.v2",
  "width": 1440, "height": 900, "format": "png", "size": 123456,
  "url": "https://example.com/", "title": "Example",
  "duration_ms": 421,
  "artifact_ref": "sha256:<artifact-digest>",
  "delivery_state": "ready",
  "artifact_url": "https://<your-host>/api/screenshot/share/<packet-id>",
  "portable_url": "https://<your-host>/api/screenshot/share/<packet-id>/portable.zip",
  "epwa_delivery": {
    "schema": "uiai.epwa_delivery.v1",
    "delivery_id": "...", "revision": 1, "producer": "...",
    "artifact": { "artifact_ref": "sha256:...", "sha256": "..." },
    "epwa": {
      "record_url": "https://<your-host>/api/screenshot/share/<packet-id>",
      "portable_url": "https://<your-host>/api/screenshot/share/<packet-id>/portable.zip",
      "access": "public_safe"
    },
    "scope": { "state": "complete" },
    "state": "ready",
    "idempotency_key": "...",
    "created_at": "...", "observed_at": "...",
    "truth_notice": "Artifact existence and EPWA delivery do not establish review, verification, completion, provider closure, settlement, or legal admissibility."
  }
}
```

Agent rules:
- `artifact_url` and `portable_url` are **absolute working HTTPS URLs** at
  publish time — derived dynamically, never hard-coded (see §2).
- The **relative canonical path is always preserved** in the package:
  `api/screenshot/share/{id}` and `api/screenshot/share/{id}/portable.zip`.
- `delivery_state: "ready"` ⇒ both URLs are live and fetchable. Any other
  state ⇒ treat as not-yet-delivered and consult `recovery_ref`.

### 1.3 Error contract — never silent, always recoverable

```json
{
  "schema": "uiai.epwa_delivery_error.v1",
  "state": "pending_reconcile",
  "artifact_ref": "", "artifact_sha256": "",
  "recovery_ref": "reconcile:session-snapshot-epwa-publication",
  "error": { "code": "epwa_publication_failed", "message": "...", "retryable": true }
}
```

Agent rules:
- HTTP `503` + `pending_reconcile` = **retryable by contract**. Back off and
  retry; do not treat as a hard failure.
- `recovery_ref` is the single machine entry point: `reconcile:*` for
  republication paths, `configure:UIAI_EPWA_PUBLIC_BASE_URL` only when the
  operator has explicitly chosen to pin an origin override.
- The record is still durable on error: `artifact_ref` / `artifact_sha256`
  are populated whenever the artifact exists, so nothing is lost.

### 1.4 Verification endpoint — deterministic integrity

`GET /api/screenshot/share/{id}/verify` →

```json
{ "packet_id": "<id>", "descriptor": "EPWA evidence package", "valid": true, "issues": [] }
```

Checks performed (by package schema):
- `uiai.evidence.share.v1`: manifest parses; screenshot file present; file
  SHA-256 equals `manifest.ScreenshotSHA256`.
- `uiai.evidence.artifact.manifest.v1` / generic: every entry in
  `descriptor.Assets` re-read from disk; byte size and SHA-256 must match
  exactly (`asset_digest_mismatch:<asset_id>` on any drift).
- Invalid/unknown `packet_id` ⇒ `invalid_packet_id`; missing public package ⇒
  `package_not_public_ready`.

---

## 2. Zero-config guarantee (out of the box — no stupid configs)

Origin resolution is **fully dynamic**, evaluated per request, in this order:

1. `UIAI_EPWA_PUBLIC_BASE_URL` env — optional operator override. Only needed
   if you deliberately want to pin an origin that the request itself cannot
   express (e.g. a different public CDN domain). **Never required.**
2. Standard proxy headers — `X-Forwarded-Proto` (`http`/`https`) +
   `X-Forwarded-Host` (first hop wins). Any conforming reverse proxy works
   with no engine-side configuration.
3. The request itself — request scheme + `Host` header; TLS connections
   detected via `req.TLS`.

Validation: parsed origin must have `http` or `https` scheme, a host, no
userinfo, no query, no fragment. Path is normalized to end with `/`.

**Works unchanged on:** localhost, LAN, tailnet, reverse proxy (nginx/caddy/
traefik), tunnel (cloudflared/ngrok), and public remote deployments. No
hard-coded UIAI hostname, port, CDN, or deployment path anywhere in the
serving path.

### 106a acceptance mapping

| 106a clause | Where satisfied |
|---|---|
| No hard-coded UIAI hostname/port/CDN/path | `canonicalEPWABase` — dynamic resolution only; no literal host anywhere in publish/serve |
| Same-origin/relative routes in PWA/manifest/JSON/image assets | Package assets and manifest are relative (`./...`); share routes serve same-origin |
| Absolute operator-facing `artifact_url` from trusted request/proxy origin, relative canonical path preserved | §1.2 — absolute URLs derived per request; relative path preserved in package |
| Storage derives only from configured UIAI data/artifact directories | `epwaDeliveryRoot` → `cfg.Storage.DataDir` → `screenshotStoreDir()` |
| Works unchanged on localhost, LAN, tailnet, reverse proxy, tunnel, public remote | §2 resolution chain (http accepted; headers honored; TLS detected) |
| Dependency-free renderer; no external JS/CSS requirement | Package shell assets are embedded; no external fetches |

---

## 3. Human walkthrough — five commands

```bash
# 1. Publish (capture + package in one call)
curl -sS -X POST https://<your-host>/api/screenshot \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","width":1440,"height":900}'
# → note .epwa_delivery.epwa.record_url and .epwa_delivery.epwa.portable_url

# 2. Read the durable record
curl -sS https://<your-host>/api/screenshot/share/<packet-id> | jq .state

# 3. Download the portable package
curl -sSLO https://<your-host>/api/screenshot/share/<packet-id>/portable.zip

# 4. Verify integrity
curl -sS https://<your-host>/api/screenshot/share/<packet-id>/verify | jq .

# 5. Use it anywhere — no config, no domain lock-in
unzip <packet-id>.epwa.zip -d evidence && open evidence/index.html
# …or: cd evidence && python3 -m http.server 8000   # any static server, any host
```

---

## 4. Package contents (what is inside the `.zip`)

- `artifact.json` — the durable machine-readable record (manifest, digests,
  capture metadata, access class, truth notice).
- The captured screenshot (content-addressed; digest recorded in manifest).
- `index.html` + shell assets — dependency-free viewer: `styles.css`,
  `work-items.js`, `locale.js`, `generic-record.js`, `pwa.js`, `app.js`.
- `manifest.webmanifest`, `icon.svg`, `sw.js` — installable PWA shell with
  offline service worker.
- All paths are **same-origin relative** (`./...`) — the package never
  references the publishing host, so it renders identically offline, on
  another domain, or behind any future deployment.

---

## 5. Guarantees and non-guarantees (truth notice)

- **Guaranteed:** durable record; portable, self-contained, offline-rendering
  package; deterministic integrity verification; machine-readable error
  contract with retryable state; zero mandatory configuration; no domain
  lock-in — any self-hosted user can use the EPWA without our domain.
- **Not established by artifact existence or EPWA delivery:** review,
  verification, completion, provider closure, settlement, or legal
  admissibility. The truth notice ships inside every record and every
  package; agents must surface it, humans should read it.
