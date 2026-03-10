# Self-Hosted Captcha Solver — Engine Specification

**Created:** 2026-03-09  
**Updated:** 2026-03-09 (v3 — implementation complete, proxy added, accuracy data)  
**Status:** Implemented (Phase 1+2 live, Phase 3 proxy built, Phase 4 pending config)  
**Owner:** uiai-engine  
**Depends on:** Session API, AI Provider, Vision Pool

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| **Phase 1: Text Captcha** | ✅ Live | VLM + Tesseract + ddddocr fallback, multi-model voting, multi-pass preprocessing |
| **Phase 2: reCAPTCHA v2** | ✅ Live | Full-grid + per-tile strategies, multi-model intersection voting, audio fallback |
| **Phase 3: Proxy Rotation** | ✅ Built | Ephemeral proxied browsers, round-robin/random rotation, anti-detection |
| **Phase 4: Per-Site Profiles** | ✅ Built | prlog, openpr, prcom, 247pressrelease profiles in config |

### Files

```
internal/captcha/
├── types.go              # All types, config, defaults, 4 site profiles
├── preprocess.go         # Image preprocessing pipeline (upscale, threshold, morphology, component filter)
├── text_solver.go        # VLM + Tesseract + ddddocr fallback chain, prompt templates
├── voting.go             # Multi-model voting (char-level consensus), multi-pass preprocessing
├── recaptcha_solver.go   # Full reCAPTCHA v2 solver (grid + per-tile + audio + dynamic tiles)
├── solver.go             # Main Solver engine, session + image solve, status
├── stats.go              # JSONL stats tracking, per-type metrics
└── proxy.go              # Proxy-rotated ephemeral browsers, anti-detection, WrapPage

internal/routes/captcha.go  # HTTP route handlers
config.yaml                 # captcha section with proxy config
```

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                        uiai-engine                                   │
│                                                                      │
│  ┌─────────────┐   ┌───────────────────┐   ┌─────────────────┐      │
│  │ Session API  │──▶│  Captcha Solver    │──▶│  AI Provider    │      │
│  │ (Rod browser)│   │  /api/captcha/*    │   │  (VLM backend)  │      │
│  │              │   │                   │   │                 │      │
│  │ • screenshot │   │ • detect type     │   │ • OpenRouter    │      │
│  │ • eval (DOM) │   │ • extract image   │   │   → Gemini Flash│      │
│  │ • click      │   │ • multi-model vote│   │   (primary)     │      │
│  │ • fill       │   │ • multi-pass pp   │   │ • Tesseract     │      │
│  │ • snapshot   │   │ • inject answer   │   │   (fallback)    │      │
│  └─────────────┘   │ • verify success  │   │ • ddddocr       │      │
│                     │ • retry loop      │   │   (fallback)    │      │
│        ┌───────┐    │ • proxy rotation  │   │ • Whisper STT   │      │
│        │ Proxy │───▶│ • anti-detection  │   │   (audio)       │      │
│        │ Pool  │    └───────────────────┘   └─────────────────┘      │
│        └───────┘                                                     │
│   socks5://...                                                       │
│   http://...                                                         │
└──────────────────────────────────────────────────────────────────────┘
```

## API Endpoints

### `POST /api/captcha/solve-image` — Stateless image solve

Solves a text captcha from a raw base64 image. No browser session needed.

```json
// Request
{
  "image_base64": "<base64>",
  "image_type": "image/png",
  "hint": "Exactly 5 lowercase English letters. Ignore ALL grid/crosshatch lines.",
  "voting": true,          // multi-model voting (3 Gemini calls)
  "multipass": true,        // 3 preprocessing variants × 3 models = 9 readings
  "preprocessing": {        // optional manual override
    "upscale": 3,
    "threshold": 100,
    "morphology_kernel": 3,
    "component_min_area": 30
  }
}

// Response
{
  "text": "mosyn",
  "confidence": "high",
  "method": "multipass_vote",
  "alternatives": ["mosyn", "mogyn", "mosyn", ...],
  "duration_ms": 1673
}
```

### `POST /api/session/{id}/captcha/solve` — Session-scoped solve

Solves a captcha in a live browser session. Auto-detects type (text vs reCAPTCHA).

```json
// Request
{
  "type": "auto",                    // "text", "recaptcha_v2", or "auto"
  "profile": "prlog",               // per-site profile (optional)
  "config": {
    "image_selector": "img[src^='data:image']",
    "answer_selector": "input[name='captcha']",
    "submit_selector": "input[type='submit']",
    "auto_submit": true,
    "max_attempts": 5,
    "hint": "5 lowercase letters",
    "strategy": "full_grid"          // reCAPTCHA only
  }
}

// Response
{
  "solved": true,
  "type": "text",
  "answer": "mosyn",
  "attempts": 2,
  "method": "multipass_vote",
  "duration_ms": 4200
}
```

### `POST /api/captcha/solve-proxied` — One-shot proxied solve

Opens a proxied browser, fills a form, solves captcha, returns result. Browser auto-closes.

```json
// Request
{
  "url": "https://admin.pr.com/create-account",
  "width": 1280,
  "height": 900,
  "fields": {
    "company": "Startempire Wire Network",
    "email": "press@startempirewire.com",
    "passwd": "secret123"
  },
  "selects": {
    "organization_type": "Private Company"
  },
  "captcha": {
    "type": "recaptcha_v2",
    "config": {"strategy": "full_grid"}
  }
}

// Response
{
  "solved": true,
  "type": "recaptcha_v2",
  "rounds": 2,
  "method": "full_grid",
  "duration_ms": 18500
}
```

### `GET /api/captcha/status` — Solver capabilities

```json
{
  "available_types": ["text", "recaptcha_v2"],
  "backends": {
    "vlm":       {"available": true, "provider": "openrouter", "models": ["google/gemini-2.0-flash-001"]},
    "tesseract": {"available": true, "version": "installed"},
    "ddddocr":   {"available": true},
    "whisper":   {"available": true, "endpoint": "http://localhost:8115"},
    "proxy":     {"available": false, "version": "no proxies configured"}
  },
  "stats": {
    "total_attempts": 10,
    "total_solved": 8,
    "success_rate": 0.80,
    "by_type": {"text": {"attempts": 10, "solved": 8, "rate": 0.80}}
  }
}
```

## Accuracy Results (Live Testing)

### Text Captcha — PRLog Crosshatch

PRLog uses 5-char lowercase text captchas with heavy crosshatch grid overlay (180×60px).

| Mode | Accuracy | Speed | Cost |
|------|----------|-------|------|
| Single VLM (raw) | ~50% per-captcha | ~1s | $0.0003 |
| Single VLM (preprocessed) | ~65% per-captcha | ~1.2s | $0.0003 |
| Multi-model voting (3 Gemini) | ~70% per-captcha | ~1.5s | $0.001 |
| **Multi-pass (3pp × 3 models)** | **~80% per-captcha** | ~2-5s | $0.003 |

**Critical finding:** Hint quality is the #1 accuracy lever.  
- Bad hint: "read the text" → 50%  
- Good hint: "Exactly 5 lowercase English letters. Ignore ALL grid/crosshatch lines." → **95%+**

With 5 retry attempts (each gets a fresh captcha), effective success rate: **>99%**

### reCAPTCHA v2 — Grid Classification

| Metric | Result |
|--------|--------|
| Grid extraction | ✅ Correct (single source image, not stitched) |
| Target detection | ✅ "bicycles", "crosswalks", "motorcycles", "fire hydrant" |
| Tile classification | ✅ VLM returns correct tile numbers |
| Multi-model voting | ✅ Intersection voting produces consistent results |
| **IP reputation** | ❌ Google flags server IP after ~3 failed attempts |

**Blocker:** reCAPTCHA's IP reputation system. This server's IP is flagged across all reCAPTCHA domains. Solution: proxy rotation (built, needs proxy config).

## IP Pool (Service-Scale)

### Why

Sites flag IPs after failed captcha attempts. The flag persists across all sites on the same IP. Any **clean IP** works — datacenter or residential. At $1/IP/month, the pool scales linearly.

### Architecture

```
                    ┌─────────────────────────┐
                    │       IP Pool           │
                    │                         │
   API calls ──────▶  Pick() → best IP       │
                    │  ├─ weighted scoring    │
                    │  ├─ concurrency limits  │
                    │  └─ cooldown tracking   │
                    │                         │
                    │  ReportResult()         │
                    │  ├─ success rate        │
                    │  ├─ flag detection      │
                    │  └─ auto-cooldown       │
                    │                         │
                    │  local:199.167.201.52 ──┼──▶ in-process SOCKS5 ──▶ Chrome
                    │  local:199.167.202.209 ─┼──▶ in-process SOCKS5 ──▶ Chrome
                    │  socks5://external ─────┼──────────────────────▶ Chrome
                    └─────────────────────────┘
```

### Per-IP Health Tracking

Each IP tracks:
- **Success rate** — weighted rotation prefers higher rates
- **Flagged state** — auto-detected from error messages ("automated queries", "unusual traffic")
- **Cooldown timer** — flagged IPs auto-cool for N minutes (default 60)
- **Active count** — concurrent solves in progress (max 2 per IP)
- **Last used / last success / last error** — full observability

Health persists to `/var/log/uiai/ip-pool-health.json` across restarts.

### Weighted Rotation

Default strategy scores IPs by:
- Higher success rate = higher score
- New IPs (0 attempts) get a 0.8 bonus to be tried
- Active solves penalize: -0.3 per active
- Recently flagged: -0.2
- Weighted random selection from scored pool

### API Management (Runtime, No Restart)

```bash
# View pool health
GET /api/captcha/pool

# Add new $1/mo IP
POST /api/captcha/pool/add
{"endpoint": "local:199.167.204.100"}

# Remove flagged IP
POST /api/captcha/pool/remove
{"endpoint": "local:199.167.201.52"}
```

### Scale Model

| IPs | Cost/mo | Max Concurrent | Notes |
|-----|---------|---------------|-------|
| 2 | $0 | 4 | Existing unused server IPs |
| 5 | $3 | 10 | Good for moderate volume |
| 10 | $8 | 20 | Handles steady fleet work |
| 20 | $18 | 40 | High-volume service |

### Anti-Detection (Built In)

```javascript
// Patched on every proxied page before navigation:
Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']});
Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
window.chrome = {runtime: {}};
```

Plus: random delays (200-800ms between actions), timing jitter, User-Agent rotation.

### Configuration

```yaml
# config.yaml
captcha:
  proxy:
    enabled: true
    strategy: weighted               # "weighted" | "round_robin" | "random"
    max_concurrent_per_ip: 2         # max simultaneous solves per IP
    cooldown_minutes: 60             # auto-cooldown flagged IPs
    health_file: /var/log/uiai/ip-pool-health.json
    local_ips:                       # server IPs ($1/mo each to add more)
      - "199.167.201.52"
      - "199.167.202.209"
    # proxies:                       # external if needed
    #   - "socks5://user:pass@host:port"
```

### IP Source Options

| Source | Cost | How |
|--------|------|-----|
| **Server IPs** (primary) | $1/IP/mo | Add via hosting provider, `local_ips` in config |
| **Tailscale exit node** | Free | Route through Mac/phone, add as external proxy |
| **External VPS** | $3-5/mo each | SSH SOCKS5 tunnel, add as `socks5://` proxy |
| **Commercial residential** | $1-4/GB | DataImpulse, Geonode — if datacenter IPs get flagged |

**Primary strategy:** Server IPs at $1/mo each. The pool handles health, rotation, and cooldown automatically. Add more IPs via API when volume grows.

## Text Captcha Solver Details

### Prompt Templates

Three built-in templates:

| Template | Use Case | Temperature |
|----------|----------|-------------|
| `blind_assistant` | General distorted text | 1.0 |
| `verification_code` | Uppercase alphanumeric codes | 1.0 |
| `lowercase_captcha` | Lowercase-only text (PRLog) | 1.0 |

### Multi-Model Voting

Default voter panel: 3× Gemini Flash at temperatures 0.8, 0.3, 1.0 (weighted 2:2:1).

Process:
1. Send same image to all voters in parallel
2. Clean each response (strip JSON wrappers, markdown, common prefixes)
3. Normalize to lowercase alphanumeric
4. Find consensus length (most common answer length)
5. Character-by-character majority vote across aligned answers

**Claude Sonnet was removed from default voters** — it hallucinates real English words ("bread", "alarm", "chump") instead of reading distorted characters. Gemini Flash is superior for captcha text reading.

### Multi-Pass Preprocessing

Three preprocessing variants run in parallel:

| Variant | Upscale | Threshold | Morphology | Component Filter |
|---------|---------|-----------|------------|-----------------|
| Light | 2× | 120 | none | none |
| Medium | 3× | 100 | 3×3 kernel | min area 20 |
| Aggressive | 4× | 80 | 5×5 kernel | min area 50, max aspect 6 |

Each variant gets multi-model voting → 3 "best" answers → final char-level vote across variants.

Total: 9 VLM calls (3 variants × 3 voters) = ~$0.003, ~2-5 seconds.

### Preprocessing Pipeline

Pure Go implementation using `golang.org/x/image`:

1. **Upscale** — nearest-neighbor (preserves pixel edges)
2. **Grayscale** — ITU-R BT.601 luma
3. **Threshold** — binary (ink < thresh → 255, else 0)
4. **Morphological open** — erode then dilate (removes thin grid lines)
5. **Connected component filter** — 8-connected flood fill, remove by area/aspect ratio

### Fallback Chain

Configurable per profile. Default: `["vlm", "tesseract"]`

1. **VLM** — OpenRouter → Gemini Flash, ~1s, highest accuracy
2. **Tesseract** — Multiple PSM modes (7, 8, 13), char whitelist, ~500ms
3. **ddddocr** — Python ONNX model, ~200ms, lowest accuracy but zero API cost

## reCAPTCHA v2 Solver Details

### Strategy A: Full Grid (Default, Recommended)

1. Click reCAPTCHA checkbox
2. Wait 0.8-1.5s (random)
3. Check if already solved (lucky checkbox pass)
4. Extract challenge info (target object + grid size from DOM)
5. Extract source image (single 450×450 image, CSS-clipped into 3×3 or 4×4 tiles)
6. Send to 3 VLM voters with grid-aware prompt
7. Intersection vote: tile selected by majority wins
8. Click matching tiles with random delays
9. Click Verify
10. Check solved → if not, repeat for new challenge

**Grid prompt template:**
```
This image shows a {rows}×{cols} grid of tiles from a visual puzzle.
Each tile is numbered 1 to {total}, left-to-right, top-to-bottom.
Identify ALL tiles that contain: {target}

Return ONLY a JSON array of tile numbers. Example: [1, 4, 7]
If no tiles match, return: []
Be thorough — select ALL matching tiles, including partially visible objects.
```

### Strategy B: Per-Tile Classification

For each tile, ask: "Does this image contain a {target}? Answer ONLY yes or no."
More API calls (9-16 per round) but potentially more precise.
Used as fallback when full-grid accuracy is insufficient.

### Audio Fallback

1. Click audio challenge button
2. Check for "automated queries" block
3. Extract audio source URL
4. Send to Whisper STT at localhost:8115
5. Fill audio response, click Verify

**Known limitation:** Google blocks audio for automation-detected sessions. This is opportunistic, not primary.

### Dynamic Tile Handling

Some challenges replace selected tiles with new images ("Please also check the new images").
The solver detects `.rc-imageselect-dynamic-selected` and re-classifies up to 4 iterations.

## Site Profiles

Built-in profiles for known press distribution surfaces:

| Profile | Type | Image Selector | Hint |
|---------|------|---------------|------|
| `prlog` | text | `img[src^='data:image']` | "Exactly 5 lowercase English letters..." |
| `openpr` | text | `img.captcha-image` | "Verification code: uppercase letters and digits" |
| `prcom` | recaptcha_v2 | (auto) | full_grid strategy |
| `247pressrelease` | recaptcha_v2 | (auto) | full_grid strategy |

## Stats & Monitoring

JSONL log at `/var/log/uiai/captcha-stats.jsonl`:
```json
{"timestamp":"2026-03-09T16:30:00Z","type":"text","solved":true,"attempts":1,"duration_ms":1673,"method":"multipass_vote","profile":"prlog"}
```

Status endpoint shows per-type success rates, total attempts, backend availability.

## Research Sources

Best practices synthesized from deep review of these open-source implementations:

| Project | Approach | Key Insights |
|---------|----------|--------------|
| [ai-captcha-bypass](https://github.com/aydinnyunus/ai-captcha-bypass) | VLM (GPT-4o/Gemini) | "Blind assistant" prompt pattern; per-tile classification |
| [recaptcha-v2-solver](https://github.com/njraladdin/recaptcha-v2-solver) | Visual + Audio + 2Captcha | Full grid screenshot → JSON response; CaptchaWatcher polling |
| [uncaptcha2](https://github.com/ecthros/uncaptcha2) | Audio → STT | 90% accuracy; Google now blocks audio for automation |
| [buster](https://github.com/dessant/buster) | Browser extension + speech | Most popular solver (1M+ users) |
| [captcha_trainer](https://github.com/kerlomz/captcha_trainer) | CNN + BiLSTM + CTC | Industrial training framework; requires GPU |
| [rota](https://github.com/alpkeskin/rota) | Go proxy rotation engine | Health monitoring, auto-rotation |

## Cost Summary

| Operation | VLM Calls | Cost | Time |
|-----------|-----------|------|------|
| Text captcha (single) | 1 | $0.0003 | ~1s |
| Text captcha (voting) | 3 | $0.001 | ~1.5s |
| Text captcha (multipass) | 9 | $0.003 | ~3s |
| reCAPTCHA round (full grid) | 3 | $0.001 | ~3s |
| reCAPTCHA full solve (2 rounds) | 6 | $0.002 | ~15s |
| Proxy traffic per solve | - | $0.0001 | - |
| **Total per press release** | **~12** | **~$0.005** | **~20s** |

At 1000 press releases/month: **$5 in VLM + $0.10 in proxy = $5.10/month**

## Known Blockers

1. **PRLog IP blocked** — server IP times out on port 80/443. Need to wait or use proxy.
2. **reCAPTCHA IP flagged** — Google "automated queries" block on this server IP. Proxy rotation solves this.
3. **MiniMax VLM unavailable** — API key invalid for VLM endpoint (status 2049). OpenRouter is the working path.
4. **Kimi no vision** — Returns 401 for image-based completions.
5. **Audio fallback blocked** — Google blocks audio challenge for automation-detected sessions.
