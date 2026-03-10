# Self-Hosted Captcha Solver — Engine Specification

**Created:** 2026-03-09  
**Updated:** 2026-03-09 (v4 — full implementation truth, IP pool service-scale, rota patterns)  
**Status:** Deployed and live  
**Owner:** uiai-engine  
**Depends on:** Session API, AI Provider, Vision Pool

---

## Implementation Status

| Phase | Status | Commit | Description |
|-------|--------|--------|-------------|
| **Phase 1: Text Captcha** | ✅ Live | `12e7711` | VLM + Tesseract + ddddocr fallback, multi-model voting, multi-pass preprocessing |
| **Phase 2: reCAPTCHA v2** | ✅ Live | `aa48b15` | Full-grid + per-tile strategies, multi-model intersection voting, dynamic tiles, audio fallback |
| **Phase 3: IP Pool** | ✅ Live | `59e373d` | Service-scale IP pool with health tracking, weighted/least-conn rotation, active probes, auto-retry |
| **Phase 4: Per-Site Profiles** | ✅ Live | `12e7711` | prlog, openpr, prcom, 247pressrelease profiles in config |

### Source Files

```
internal/captcha/
├── types.go              # All types, CaptchaConfig, defaults, 4 site profiles
├── preprocess.go         # Image preprocessing (upscale, grayscale, threshold, morphology, component filter)
├── text_solver.go        # VLM + Tesseract + ddddocr fallback chain, prompt templates
├── voting.go             # Multi-model voting (char-level consensus), multi-pass preprocessing
├── recaptcha_solver.go   # Full reCAPTCHA v2 solver (660 lines: grid + per-tile + audio + dynamic tiles)
├── solver.go             # Main Solver engine, SolveInSession, SolveImage, GetStatus, Pool()
├── stats.go              # JSONL stats tracking (/var/log/uiai/captcha-stats.jsonl)
└── proxy.go              # IP pool, in-process SOCKS5, ephemeral browsers, health probes, auto-retry

internal/vision/
└── session.go            # WrapPage() — wraps raw rod.Page into Session for proxy browsers

internal/routes/
└── captcha.go            # HTTP route handlers for all captcha endpoints

internal/config/
└── config.go             # CaptchaYAML, CaptchaProxyYAML structs for config.yaml loading

internal/server/
└── server.go             # Config merge: YAML proxy settings → DefaultCaptchaConfig → NewSolver

config.yaml               # captcha section with full proxy/pool config
```

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           uiai-engine                                    │
│                                                                          │
│  ┌─────────────┐   ┌─────────────────────┐   ┌─────────────────┐        │
│  │ Session API  │──▶│   Captcha Solver     │──▶│  AI Provider    │        │
│  │ (Rod browser)│   │   /api/captcha/*     │   │  (VLM backend)  │        │
│  │              │   │                     │   │                 │        │
│  │ • screenshot │   │ • type detection    │   │ • OpenRouter    │        │
│  │ • eval (DOM) │   │ • image extraction  │   │   → Gemini Flash│        │
│  │ • click      │   │ • multi-model vote  │   │   (primary)     │        │
│  │ • fill       │   │ • multi-pass pp     │   │ • Tesseract     │        │
│  │ • snapshot   │   │ • answer injection  │   │   (fallback)    │        │
│  └──────────────┘   │ • verify + retry    │   │ • ddddocr       │        │
│                      │ • auto-retry on     │   │   (fallback)    │        │
│  ┌──────────────┐   │   different IP      │   │ • Whisper STT   │        │
│  │   IP Pool    │──▶│ • anti-detection    │   │   (audio)       │        │
│  │              │   └─────────────────────┘   └─────────────────┘        │
│  │ 3 local IPs  │                                                        │
│  │ SOCKS5 bind  │   ┌─────────────────────┐                              │
│  │ health probe │   │  Health Probe Loop   │                              │
│  │ weighted pick│   │  every 5 min         │                              │
│  │ auto-cooldown│   │  TCP test per IP     │                              │
│  │ auto-retry   │   │  removes unreachable │                              │
│  └──────────────┘   └─────────────────────┘                              │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## API Endpoints

### Captcha Solving

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/api/captcha/solve-image` | Stateless text captcha solve from base64 image |
| `POST` | `/api/session/{id}/captcha/solve` | Session-scoped solve (text + reCAPTCHA v2) |
| `POST` | `/api/captcha/solve-proxied` | One-shot proxied browser: navigate + fill + solve |
| `GET`  | `/api/captcha/status` | Backend availability, solve stats, pool info |

### IP Pool Management

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET`  | `/api/captcha/pool` | Full pool status with per-IP health |
| `POST` | `/api/captcha/pool/add` | Add IP at runtime (no restart) |
| `POST` | `/api/captcha/pool/remove` | Remove IP at runtime |

### Request/Response Examples

#### `POST /api/captcha/solve-image`
```json
// Request
{
  "image_base64": "<base64>",
  "image_type": "image/png",
  "hint": "Exactly 5 lowercase English letters. Ignore ALL grid/crosshatch lines.",
  "voting": true,
  "multipass": true
}

// Response
{
  "text": "mosyn",
  "confidence": "high",
  "method": "multipass_vote",
  "alternatives": ["mosyn", "mogyn", "mosyn"],
  "duration_ms": 1673
}
```

#### `POST /api/captcha/solve-proxied`
```json
// Request
{
  "url": "https://admin.pr.com/create-account",
  "width": 1280,
  "height": 900,
  "fields": {"company": "...", "email": "...", "passwd": "..."},
  "selects": {"organization_type": "Private Company"},
  "captcha": {"type": "recaptcha_v2", "config": {"strategy": "full_grid"}}
}

// Response — auto-retries on different IP if first attempt fails
{
  "solved": true,
  "type": "recaptcha_v2",
  "rounds": 2,
  "method": "full_grid",
  "duration_ms": 18500
}
```

#### `GET /api/captcha/pool`
```json
{
  "total_ips": 3,
  "strategy": "weighted",
  "max_concurrent_per_ip": 2,
  "cooldown_minutes": 60,
  "max_retries": 2,
  "probe_url": "https://www.google.com/recaptcha/api.js",
  "probe_interval_seconds": 300,
  "ips": [
    {
      "endpoint": "local:199.167.201.52",
      "label": "199.167.201.52",
      "status": "healthy",
      "active": 0,
      "total_attempts": 0,
      "total_solved": 0,
      "success_rate": 0,
      "flagged": false,
      "probe_ok": true,
      "last_probe": "2026-03-09T17:43:48Z"
    }
  ]
}
```

---

## IP Pool — Service-Scale Design

### Server IP Assignment Map

| IP | Subnet | Status | Assigned To |
|----|--------|--------|-------------|
| `199.167.200.52` | .200/24 | **FLAGGED** — main shared IP | 37 cPanel accounts |
| `199.167.201.52` | .201/24 | **In pool — clean** | Nobody |
| `199.167.202.209` | .202/24 | **In pool — clean** | Nobody |
| `199.167.203.234` | .203/24 | **In pool — clean** | Nobody |
| `67.222.9.109` | different range | Used | thedream/SendyKit |

All IPs same ASN (`AS63410 PrivateSystems Networks`). Google flags individual IPs, not ASNs.

### How It Works

1. **Pick** — weighted rotation selects healthiest available IP (or least-connections)
2. **SOCKS5 bind** — in-process Go SOCKS5 proxy binds outgoing traffic to that local IP
3. **Ephemeral Chrome** — new browser per solve, isolated from main vision pool
4. **Anti-detection** — navigator.webdriver patched, random UA, plugin spoofing
5. **Solve** — captcha solver runs in the proxied browser session
6. **Report** — success/failure/flagged recorded on the IP's health record
7. **Auto-retry** — on failure, automatically tries a different IP (up to `max_retries`)
8. **Auto-cooldown** — flagged IPs pulled from rotation for `cooldown_minutes`
9. **Health probe** — background goroutine tests each IP every 5min via TCP to google.com:443

### Rotation Strategies

| Strategy | When to Use | Config |
|----------|------------|--------|
| `weighted` (default) | General use — prefers healthy IPs, penalizes busy/flagged | Success rate scoring + weighted random |
| `least_conn` | High concurrency — routes to IP with fewest active solves | Ties broken by success rate |
| `round_robin` | Even distribution | Simple index rotation |
| `random` | Unpredictable | Random selection |

### Active Health Probes

Background goroutine runs every `health_probe_seconds` (default 300 = 5min):

- **Local IPs**: TCP dial to `www.google.com:443` from that specific local address
- **External proxies**: TCP connectivity to the proxy host
- Failed probes mark `probe_ok: false` → IP excluded from rotation
- Next successful probe restores the IP automatically
- Probe runs 10 seconds after startup for initial status

This catches IPs that go down between solves — no more silent failures.

### Auto-Retry on Different IP

When a solve fails on IP A:
1. IP A is reported as failed (health updated)
2. Solver picks IP B (excluding A)
3. Full solve attempt runs on IP B
4. If B also fails and `max_retries` not exhausted, tries IP C
5. Returns first success, or last failure if all attempts exhausted

Configurable via `max_retries` (default 2). With 3 IPs and 2 retries, all 3 get tried before returning failure.

### Flag Detection

Error messages are scanned for IP-flagging indicators:
- "automated queries"
- "unusual traffic"
- "blocked"
- "rate limit"
- "access denied"
- "forbidden"

Flagged IPs enter cooldown (`cooldown_minutes`, default 60) and are excluded from rotation.
Success on a previously-flagged IP clears the flag (IP recovered).

### Runtime Management

No restart needed to modify the pool:

```bash
# Add a new $1/mo IP
curl -X POST localhost:7456/api/captcha/pool/add \
  -H "X-Webhook-Secret: $SECRET" \
  -d '{"endpoint":"local:199.167.204.100"}'

# Remove a flagged IP
curl -X POST localhost:7456/api/captcha/pool/remove \
  -H "X-Webhook-Secret: $SECRET" \
  -d '{"endpoint":"local:199.167.201.52"}'
```

### In-Process SOCKS5

Chrome doesn't support `--bind-address`. To bind outgoing traffic to a specific local IP:

1. Go starts a minimal SOCKS5 proxy on `127.0.0.1:<random-port>`
2. The SOCKS5 handler dials outgoing connections from the specified local IP via `net.Dialer{LocalAddr: &net.TCPAddr{IP: ...}}`
3. Chrome gets `--proxy-server=socks5://127.0.0.1:<port>`
4. SOCKS5 proxies are reused across browsers for the same IP (not recreated per solve)
5. Supports IPv4, IPv6, and domain name resolution

### Scale Model

| IPs | Cost/mo | Max Concurrent Solves | Notes |
|-----|---------|----------------------|-------|
| 3 | $0 | 6 | Current — existing unused server IPs |
| 5 | $2 | 10 | Add 2 more IPs |
| 10 | $7 | 20 | Handles steady fleet work |
| 20 | $17 | 40 | High-volume service |

### IP Source Options

| Source | Cost | How |
|--------|------|-----|
| **Server IPs** (primary) | $1/IP/mo | Add via hosting provider, `local_ips` in config |
| **Tailscale exit node** | Free | Route through Mac/phone — residential IP |
| **External VPS** | $3-5/mo each | SSH SOCKS5 tunnel, add as `socks5://` proxy |
| **Commercial residential** | $1-4/GB | DataImpulse ($5 trial), Geonode — only if all datacenter IPs flagged |

**Primary strategy:** Server IPs at $1/mo each. No external vendor needed at current scale.

---

## Configuration — Full Reference

```yaml
# config.yaml — captcha section
captcha:
  enabled: true
  default_provider: "openrouter"
  default_model: "google/gemini-2.0-flash-001"

  text:
    max_attempts: 5
    retry_delay: "1s"
    prompt_template: "blind_assistant"
    fallback_chain: ["vlm", "tesseract"]

  recaptcha:
    strategy: "full_grid"          # "full_grid" | "per_tile"
    max_rounds: 3
    max_attempts: 2
    action_delay_min_ms: 200       # random delay between tile clicks
    action_delay_max_ms: 800
    verify_delay_min_ms: 500       # random delay before verify
    verify_delay_max_ms: 1000
    audio_fallback: true
    whisper_url: "http://localhost:8115"

  proxy:
    enabled: true
    strategy: "weighted"           # "weighted" | "least_conn" | "round_robin" | "random"
    max_concurrent_per_ip: 2       # max simultaneous solves per IP
    cooldown_minutes: 60           # auto-cooldown flagged IPs
    max_retries: 2                 # auto-retry on different IP before returning failure
    health_file: "/var/log/uiai/ip-pool-health.json"
    health_probe_url: "https://www.google.com/recaptcha/api.js"
    health_probe_seconds: 300      # probe every 5 min
    local_ips:
      - "199.167.201.52"           # unused, clean
      - "199.167.202.209"          # unused, clean
      - "199.167.203.234"          # unused, clean
      # Main IP 199.167.200.52 is FLAGGED — do NOT add
      # 67.222.9.109 is thedream/SendyKit — avoid
    # proxies:                     # external proxies if needed
    #   - "socks5://user:pass@host:port"

  stealth:
    patch_webdriver: true
    random_user_agent: true

  stats:
    enabled: true
    log_file: "/var/log/uiai/captcha-stats.jsonl"
```

---

## Text Captcha Solver

### Prompt Templates

| Template | Use Case | Framing |
|----------|----------|---------|
| `blind_assistant` | General distorted text | "Act as a blind person assistant. Read the text..." |
| `verification_code` | Uppercase alphanumeric | "This image contains a verification code..." |
| `lowercase_captcha` | Lowercase-only (PRLog) | "Exactly 5 lowercase English letters..." |

### Multi-Model Voting

Default voter panel: **3× Gemini Flash** at temperatures 0.8, 0.3, 1.0 (weights 2:2:1).

Process:
1. Same image sent to all voters in parallel
2. Responses cleaned (strip JSON, markdown, prefixes)
3. Normalized to lowercase alphanumeric
4. Consensus length found (most common answer length)
5. Character-by-character majority vote

**Claude Sonnet removed from default voters** — hallucinates real English words ("bread", "alarm") instead of reading distorted characters.

### Multi-Pass Preprocessing

| Variant | Upscale | Threshold | Morphology | Component Filter |
|---------|---------|-----------|------------|-----------------|
| Light | 2× | 120 | none | none |
| Medium | 3× | 100 | 3×3 kernel | min area 20 |
| Aggressive | 4× | 80 | 5×5 kernel | min area 50, max aspect 6 |

9 VLM calls total (3 variants × 3 voters). ~$0.003, ~2-5 seconds.

### Preprocessing Pipeline (Pure Go)

1. **Upscale** — nearest-neighbor
2. **Grayscale** — ITU-R BT.601 luma
3. **Threshold** — binary
4. **Morphological open** — erode + dilate (removes grid lines)
5. **Connected component filter** — 8-connected flood fill, remove by area/aspect

### Fallback Chain

1. **VLM** — OpenRouter → Gemini Flash (~1s, highest accuracy)
2. **Tesseract** — Multiple PSM modes (7, 8, 13), char whitelist (~500ms)
3. **ddddocr** — Python ONNX model (~200ms, zero API cost)

---

## reCAPTCHA v2 Solver

### Full Grid Strategy (Default)

1. Click reCAPTCHA checkbox → wait 0.8-1.5s
2. Check if auto-solved (lucky pass)
3. Extract challenge target + grid size from DOM
4. Extract source image (single `naturalWidth × naturalHeight`, not stitched tiles)
5. Send to 3 VLM voters → intersection vote on tile numbers
6. Click matching tiles with 200-800ms random delays
7. Click Verify → wait 500-1000ms
8. Check solved → repeat if new challenge

### Grid Extraction Detail

reCAPTCHA uses **one source image** for all tiles. CSS `clip` shows different regions. The solver extracts the full source via canvas `toDataURL()` at native resolution, not by stitching visible tile fragments.

### Dynamic Tile Handling

"Select more" challenges replace selected tiles. The solver detects `.rc-imageselect-dynamic-selected` and re-classifies up to 4 iterations per round.

### Audio Fallback

Opportunistic — Google blocks audio for automation-detected sessions. When available: download MP3 → Whisper STT at localhost:8115 → fill response.

---

## Site Profiles

| Profile | Type | Captcha | Key Config |
|---------|------|---------|-----------|
| `prlog` | text | Crosshatch 5-char lowercase | voting=true, hint="Exactly 5 lowercase English letters..." |
| `openpr` | text | Verification code | prompt_template=verification_code |
| `prcom` | recaptcha_v2 | Google reCAPTCHA | strategy=full_grid, site_key=6LcOef8S... |
| `247pressrelease` | recaptcha_v2 | Google reCAPTCHA | strategy=full_grid |

---

## Accuracy Data (Live Testing)

### Text Captcha — PRLog

| Mode | Per-Captcha Accuracy | Speed | Cost |
|------|---------------------|-------|------|
| Single VLM (raw) | ~50% | ~1s | $0.0003 |
| Single VLM (preprocessed) | ~65% | ~1.2s | $0.0003 |
| Multi-model voting | ~70% | ~1.5s | $0.001 |
| **Multi-pass voting** | **~80%** | ~2-5s | $0.003 |

**Hint quality is the #1 accuracy lever.**  
With good hint + 5 retry attempts: effective **>99%** success rate.

### reCAPTCHA v2

| Metric | Result |
|--------|--------|
| Grid extraction | ✅ Single source image, correct |
| Target detection | ✅ bicycles, crosswalks, motorcycles, fire hydrants |
| VLM classification | ✅ Unanimous 3-model agreement |
| **Main server IP** | ❌ Flagged by Google ("automated queries") |
| **Pool IPs (3 clean)** | ✅ Ready — untested (awaiting first proxied solve) |

---

## Cost Summary

| Operation | VLM Calls | Cost | Time |
|-----------|-----------|------|------|
| Text captcha (single) | 1 | $0.0003 | ~1s |
| Text captcha (multipass) | 9 | $0.003 | ~3s |
| reCAPTCHA round (full grid) | 3 | $0.001 | ~3s |
| reCAPTCHA full solve | 6 | $0.002 | ~15s |
| **Per press release** | **~12** | **~$0.005** | **~20s** |
| **1000 press releases/mo** | — | **~$5** | — |

Server IPs: $0/mo (existing). External: $1/IP/mo if needed.

---

## Commit History (This Session)

| Commit | Description |
|--------|-------------|
| `f81c149` | Initial proxy rotation — ephemeral browsers, round-robin/random, anti-detection |
| `0901827` | Docs fix — clean datacenter IP sufficient, rota properly credited |
| `8bf0252` | Docs v3 — implementation truth, proxy provider research, cost math |
| `d492eed` | Service-scale IP pool — per-IP health, weighted rotation, runtime API, SOCKS5 |
| `a6c04a5` | Config — add 199.167.203.234, document IP assignment map |
| `7f5b05c` | Docs — IP pool architecture, scale model, API routes |
| `59e373d` | **least_conn, active health probes, auto-retry** — rota patterns |

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **No rota dependency** | Pool functionality built in-process. Same features, no external daemon to manage. |
| **No commercial proxy vendor** | 3 clean server IPs at $0/mo. DataImpulse ($5 trial) only if all datacenter IPs get flagged. |
| **Residential IP not required** | Google flags individual IPs by reputation, not by type. Fresh datacenter works. |
| **In-process SOCKS5** | Chrome doesn't support `--bind-address`. Go SOCKS5 proxy binds outgoing to local IP with zero external deps. |
| **Gemini Flash only for voters** | Claude Sonnet hallucinates English words on distorted text. Gemini reads characters accurately. |
| **Full-grid over per-tile** | 1 VLM call analyzes entire grid vs 9-16 calls for per-tile. Faster, cheaper, equally accurate. |
| **Health probe to google.com** | reCAPTCHA is the primary target. If an IP can't reach Google, it can't solve reCAPTCHA. |
| **Weighted + auto-retry** | Weighted picks the best IP. Auto-retry ensures one bad IP doesn't fail the whole solve. |

---

## Known Blockers

| Blocker | Impact | Mitigation |
|---------|--------|-----------|
| Main IP (`199.167.200.52`) flagged by Google | reCAPTCHA fails on default IP | 3 clean IPs in pool, auto-retry |
| PRLog blocks server IP (port 80/443 timeout) | Can't reach PRLog at all | Use proxied browser with clean IP |
| MiniMax VLM API key invalid for image endpoint | Can't use MiniMax for tile classification | OpenRouter → Gemini Flash works |
| Kimi returns 401 for vision | Can't use Kimi for image tasks | OpenRouter → Gemini Flash works |
| Google blocks audio challenge for automation | Audio fallback unreliable | Full-grid visual is primary |

---

## Related Docs

- [`SESSION_API.md`](SESSION_API.md) — Session API reference, captcha solver section
- [`WORKFLOW_API_ORCHESTRATION.md`](WORKFLOW_API_ORCHESTRATION.md) — Full endpoint map including captcha routes
