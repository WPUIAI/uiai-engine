# Device Frame Integration (Long-Term)

Goal: replace ad-hoc generated frames with versioned, high-quality upstream frame assets (GitHub/API) for consistent landing and marketing outputs.

## Short answer
Yes — we can integrate GitHub device-frame sources into UIAI API cleanly.

Best long-term path:
1. Vendor approved frame packs into repo-managed asset store.
2. Normalize to a single internal manifest format (`frame_id`, viewport safe area, output sizes).
3. Expose stable API endpoint for rendering screenshots into those frames.
4. Add update/sync job with license checks + visual regression snapshots.

---

## Candidate upstream sources

### Open source (self-host, predictable)
- PommePlate: https://github.com/ephread/PommePlate
- iOS device SVG templates: https://github.com/neogeek/ios-device-svg-templates
- deviceframe (CLI-style workflow): https://github.com/c0bra/deviceframe

### API providers (fastest quality, paid/managed)
- Mockuuups API: https://mockuuups.studio/api/
- Shots.so: https://shots.so/

Recommendation:
- Primary baseline: **Open-source vendored frames** (control + no runtime vendor dependency).
- Optional premium tier: provider API fallback for campaign-grade variants.

---

## Proposed architecture

## 1) Asset layout (versioned)

```text
internal/media/deviceframes/
  manifest.json
  vendor/
    pommeplate/
      <commit>/
        macbook-pro-16.svg
        iphone-15-pro.svg
        LICENSE
    ios-device-svg-templates/
      <commit>/
        iphone-15-pro.svg
        LICENSE
  normalized/
    macbook-pro-16-dark/
      frame.png
      mask.png
      meta.json
    iphone-15-pro-black/
      frame.png
      mask.png
      meta.json
```

`meta.json` defines safe area and target aspect:

```json
{
  "frame_id": "macbook-pro-16-dark",
  "source": "pommeplate",
  "source_ref": "<commit>",
  "license": "MIT",
  "safe_area": { "x": 212, "y": 118, "w": 1980, "h": 1238 },
  "output": { "width": 2600, "height": 1600 },
  "mask": "mask.png",
  "frame": "frame.png"
}
```

---

## 2) Render pipeline

Input:
- screenshot image (from `/api/screenshot` or `/api/session/{id}/screenshot`)
- `frame_id`
- fit mode (`cover|contain`)
- output format (`webp|png|jpeg`)

Steps:
1. Load frame metadata.
2. Fit screenshot into safe area.
3. Apply mask/rounded corners.
4. Composite under/over frame layers.
5. Export 1x/2x variants.
6. Cache by hash (`frame_id + input_hash + params`).

---

## 3) New API contract

Endpoint (proposed): `POST /api/media/frame/render`

Request:

```json
{
  "frameId": "iphone-15-pro-black",
  "imageBase64": "...",
  "fit": "cover",
  "format": "webp",
  "quality": 92,
  "scale": 2
}
```

Response:

```json
{
  "frameId": "iphone-15-pro-black",
  "format": "webp",
  "width": 1200,
  "height": 2360,
  "imageBase64": "...",
  "cacheHit": false,
  "source": "pommeplate@<commit>"
}
```

---

## 4) Sync/update workflow

Scripted sync (manual + cron-ready):
- pull upstream repos to temp
- copy approved assets only
- verify license allowlist
- generate/update manifest
- render golden previews
- require visual diff approval before activation

No auto-publish to production without review.

---

## 5) License policy (required)

Allowlist only:
- MIT
- Apache-2.0
- CC0

Blocklist by default:
- unknown license
- non-commercial restrictions
- attribution-required unless legal approves display path

Each frame stores:
- source URL
- commit hash
- license text snapshot
- import timestamp

---

## 6) Quality gates

Per frame before activation:
- Safe area alignment test (pixel bounds check)
- Multi-resolution render check (1x/2x)
- Contrast/no halo checks
- Device proportion checks (phone viewport not stretched)
- Real screenshot test set (Wire, WINS logged-in)

---

## 7) Operational plan

Phase 1 (now):
- integrate 3 stable frames: MacBook, iPhone portrait, iPhone landscape
- migrate landing generation to `frame/render` path

Phase 2:
- add frame catalog endpoint (`GET /api/media/frame/catalog`)
- add per-frame style variants (silver/space-black)

Phase 3:
- optional external provider fallback (Mockuuups) when `premium=true`

---

## 8) Security + reliability

- sanitize SVG on ingest (no scripts/external refs)
- pre-render to PNG layers for runtime speed
- no live network pull at request time
- cache output in `data/media/frames-cache`
- hard timeout + memory cap per render

---

## 9) Migration from current state

Current landing uses generated JPG frame composites.
Migration:
1. Keep current images as fallback.
2. Build new frame renderer.
3. Regenerate hero/architecture assets with stable frame IDs.
4. Swap URLs in WP only after QA snapshots pass.

---

## 10) Acceptance criteria

- Device frames render proportionally on desktop/mobile.
- WINS logged-in screenshot appears fully loaded and readable.
- The Wire capture shows useful app context (not empty/broken slice).
- Output files are retina-ready and non-pixelated.
- Rebuilds are deterministic from manifest + source commit.

---

## Related documentation

- Project overview and media route map: [README](../README.md)
- Screenshot/session capture sources for frame rendering: [Session API](SESSION_API.md)
- Browser reliability checks for screenshot artifacts: [Browser Reliability Runbook](BROWSER_RELIABILITY_RUNBOOK.md)
- Screenshot/share parity caveats: [Full API Parity Evaluation](FULL_API_PARITY_EVALUATION_AND_RETIREMENT_INVENTORY_2026-03-07.md)
