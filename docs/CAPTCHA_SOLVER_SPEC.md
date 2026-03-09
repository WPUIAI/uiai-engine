# Self-Hosted Captcha Solver — Engine Specification

**Created:** 2026-03-09  
**Updated:** 2026-03-09 (v2 — deep research pass)  
**Status:** Plan  
**Owner:** uiai-engine  
**Depends on:** Session API, AI Provider, Vision Pool

## Problem

Wire Pitch distributes press releases to 10+ surfaces. Many require:
1. **Account registration** — one-time, but gated by captcha
2. **Press release submission** — repeated, often gated by captcha
3. **Email verification** — sometimes includes captcha on confirmation page

If the engine can't solve captchas, the entire automated distribution service is dead.
Manual registration is a workaround for the first category but doesn't scale.
Submission captchas appear on every single press release, so they **must** be solved
programmatically or the service has zero value.

## Architecture

Self-hosted, open-source, no external captcha-farm dependencies.

```
┌──────────────────────────────────────────────────────────────────┐
│                        uiai-engine                               │
│                                                                  │
│  ┌─────────────┐   ┌───────────────────┐   ┌─────────────────┐  │
│  │ Session API  │──▶│  Captcha Solver    │──▶│  AI Provider    │  │
│  │ (Rod browser)│   │  /api/captcha/*    │   │  (VLM backend)  │  │
│  │              │   │                   │   │                 │  │
│  │ • screenshot │   │ • detect type     │   │ • OpenRouter    │  │
│  │ • eval (DOM) │   │ • extract image   │   │   → Gemini Flash│  │
│  │ • click      │   │ • solve via VLM   │   │   → Claude      │  │
│  │ • fill       │   │ • inject answer   │   │ • MiniMax       │  │
│  │ • snapshot   │   │ • verify success  │   │ • Ollama (local)│  │
│  └─────────────┘   │ • retry loop      │   │ • Future: any   │  │
│                     └───────────────────┘   └─────────────────┘  │
│                              │                                   │
│                     ┌────────▼────────┐                          │
│                     │  Local Fallback  │                          │
│                     │ • Tesseract OCR  │                          │
│                     │ • ddddocr (ONNX) │                          │
│                     │ • Whisper STT    │                          │
│                     │ • CV2 preprocess │                          │
│                     └─────────────────┘                          │
└──────────────────────────────────────────────────────────────────┘
```

**Key principle:** The solver is a session-aware action. It operates on a live browser
session, not on detached images. This means it can interact with the DOM (click
checkboxes, extract tiles, read challenge text, click verify, detect success).

## Research Sources

Best practices synthesized from deep review of these open-source implementations:

| Project | Approach | Key Insights |
|---------|----------|--------------|
| [aydinnyunus/ai-captcha-bypass](https://github.com/aydinnyunus/ai-captcha-bypass) | VLM (GPT-4o / Gemini) for text, reCAPTCHA, puzzle, audio | Per-tile classification with ThreadPoolExecutor; "Act as blind person assistant" prompt pattern; separate instruction-extraction step |
| [njraladdin/recaptcha-v2-solver](https://github.com/njraladdin/recaptcha-v2-solver) | Visual (Gemini), Audio (wit.ai/Speech-to-Text), 2Captcha | Full grid screenshot → JSON coordinate response (not per-tile); `CaptchaWatcher` polling pattern; dynamic vs static challenge handling; puppeteer-extra-plugin-stealth |
| [ecthros/uncaptcha2](https://github.com/ecthros/uncaptcha2) | Audio challenge → Speech-to-Text API | 90% accuracy on reCAPTCHA audio with single STT call; Google now blocks audio for automation-detected sessions |
| [dessant/buster](https://github.com/dessant/buster) | Browser extension + speech recognition for audio | Most popular open-source solver (1M+ users); client app for mouse simulation |
| [kerlomz/captcha_trainer](https://github.com/kerlomz/captcha_trainer) | CNN5/DenseNet + BiLSTM/LSTM + CTC | Industrial-grade training framework; handles sticky/overlapping/warped/noisy text captchas; requires GPU + training data |
| [ZYSzys/awesome-captcha](https://github.com/ZYSzys/awesome-captcha) | Curated list | Comprehensive index of crack tools, ML solvers, and generation libraries |

### Key Best Practices Extracted

**1. VLM Prompt Engineering (from ai-captcha-bypass)**
- Text captcha: "Act as a blind person assistant. Read the text from the image and give me only the text answer." — role-framing increases accuracy
- reCAPTCHA instructions: Separate VLM call just to extract the target object from the blue instruction bar, then per-tile classification calls
- Per-tile approach: `ask_if_tile_contains_object(tile_image, object_name)` with "respond only with 'true' if you are certain, otherwise 'false'" — high-precision prompt
- Temperature 0 for classification, temperature 1 for text reading

**2. Full Grid Analysis (from recaptcha-v2-solver)**
- Alternative to per-tile: send **entire grid screenshot** to VLM with grid coordinate system
- VLM returns JSON `{"[1,1]": {"has_match": false}, "[1,2]": {"has_match": true}, ...}`
- Prompt includes grid layout reference and explicit instruction:
  "If a tile already appears selected, do not mark it as 'has_match': true"
- Gemini config: `temperature: 0.1, topP: 0.95, topK: 40`
- **This avoids N API calls per round** — single call analyzes all tiles

**3. Challenge State Management (from recaptcha-v2-solver)**
- `CaptchaWatcher` pattern: continuous polling for challenge state changes
- Detect `rc-imageselect-dynamic-selected` class for tiles in transition
- Track `currentImageUrls` to detect when challenge images actually change
- Handle "Click verify once there are none left" (dynamic) vs standard challenges separately
- Dynamic challenges: up to 4 iterations of tile replacement before verify
- Standard challenges: single round of tile selection then verify

**4. Anti-Detection (from recaptcha-v2-solver)**
- `puppeteer-extra-plugin-stealth` — patches Chrome automation detection
- `Object.defineProperty(navigator, 'webdriver', { get: () => undefined })`
- Random user-agent rotation from curated list
- Random delays between tile clicks: `500 + Math.random() * 500 ms`
- Random delay before verify: `500 + Math.random() * 500 ms`
- `--disable-blink-features=AutomationControlled`
- Profile rotation: random Chrome profile per session

**5. Audio Fallback Strategy (from uncaptcha2 + buster)**
- Download audio challenge MP3 from reCAPTCHA
- Single call to Speech-to-Text API achieves ~90% accuracy
- Google blocks audio for automation-detected sessions → opportunistic, not primary
- Our advantage: Whisper STT running locally at `localhost:8115` — zero-cost, no API key
- Clean transcription: `re.sub(r'[^a-zA-Z0-9]', '', response)` — strip all non-alphanumeric

**6. Success Detection (from both implementations)**
- Token check: `document.querySelector('textarea[name="g-recaptcha-response"]').value`
- Verify button disabled after success
- `Promise.race([waitForToken, waitForNewChallenge, timeout])` pattern
- Challenge frame URL contains `api2/bframe`; checkbox frame contains `api2/anchor`

**7. Error Recovery (from ai-captcha-bypass)**
- Track previously clicked tile indices across rounds (avoid re-clicking)
- Reset clicked set when new object type appears
- Reset clicked set when ≥3 tiles were clicked previously (assume new grid)
- Maximum 5 challenge attempts before giving up
- Reload button (`#recaptcha-reload-button`) to get fresh challenge if format invalid

## Captcha Types

### Type 1: Text/Image Captcha

Sites: OpenPR, PRLog, some smaller directories.

**Flow:**
1. Caller provides `sessionID` and a CSS/JS selector for the captcha image
2. Engine extracts the image from the DOM (canvas toDataURL or img.src)
3. Image is preprocessed (upscale, denoise, threshold — configurable per site)
4. Image is sent to VLM with a text-extraction prompt
5. VLM returns the text
6. Engine fills the answer field and optionally submits

**Preprocessing pipeline (Go, using existing imaging deps):**
- Upscale 2–4x (nearest-neighbor for pixel art, bicubic for photos)
- Convert to grayscale
- Apply adaptive threshold or fixed threshold (configurable)
- Morphological open/close to remove grid lines (configurable kernel)
- Connected component filtering: remove components below area threshold
  or with extreme aspect ratios (line fragments)
- Output: cleaned PNG at original resolution

**VLM prompt (from research — "blind assistant" framing works best):**
```
Act as a blind person assistant. Read the text from the image and give me
only the text answer. If there is no text, give me an empty string.
The text may be distorted with noise, grid lines, or warping — read through
the distortion and return only the characters you can identify.
```

For verification-code style captchas (uppercase alphanumeric):
```
This image contains a verification code. Read ONLY the characters shown.
Return only uppercase letters and digits, no spaces or punctuation.
If unsure about a character, make your best guess.
```

**Fallback chain:**
1. VLM (OpenRouter → Gemini Flash) — primary, ~2s, temperature 1.0
2. VLM (OpenRouter → Claude Sonnet) — if Gemini rate-limited
3. Tesseract with preprocessing variants — if VLM unavailable, ~500ms
4. ddddocr ONNX model — if Tesseract too noisy, ~200ms

**Multi-attempt strategy:**
- On wrong answer, the captcha image refreshes
- Re-extract new image, re-solve, retry
- Max attempts configurable (default: 5)
- Delay between attempts: `1s + random(0-1s)` (avoid triggering rate limits)

### Type 2: reCAPTCHA v2

Sites: PR.com, 24-7 Press Release, IssueWire, many others.

**Two solving strategies (configurable):**

#### Strategy A: Full Grid Screenshot (recommended — fewer API calls)
Based on njraladdin/recaptcha-v2-solver pattern.

1. Take screenshot of entire challenge area (`.rc-imageselect-challenge`)
2. Send single image to VLM with grid coordinate prompt
3. VLM returns JSON with per-tile match boolean
4. Click matching tiles
5. Click verify
6. Handle new challenge rounds

**VLM prompt (from research — JSON coordinate response):**
```
For each tile in the grid, check if it contains a VISIBLE -- {TARGET_OBJECT} --.
Only mark a tile as "has_match": true if you are CERTAIN the object appears in
that specific tile. It should be clear and visible to a normal human.
If a tile already appears selected, do not mark it as "has_match": true.
Think carefully before marking a tile — you will fail if you mark an incorrect tile.

Grid layout (row,column coordinates):
{GRID_LAYOUT}

Respond ONLY with a JSON object:
{"[1,1]": {"has_match": false}, "[1,2]": {"has_match": true}, ...}
```

Grid layouts:
- 3×3: `Row 1: [1,1] [1,2] [1,3] / Row 2: [2,1] [2,2] [2,3] / Row 3: [3,1] [3,2] [3,3]`
- 4×4: `Row 1: [1,1]...[1,4] / ... / Row 4: [4,1]...[4,4]`

VLM config: `temperature: 0.1, topP: 0.95, topK: 40`

#### Strategy B: Per-Tile Classification (higher accuracy, more API calls)
Based on ai-captcha-bypass pattern.

1. Extract each tile image individually (screenshot per tile or canvas extract)
2. Send each tile to VLM in parallel: "Does this image contain {target}?"
3. Collect boolean responses
4. Click positive tiles
5. Verify

**VLM prompt (per tile):**
```
Does this image clearly contain a '{target_object}' or a recognizable part
of a '{target_object}'? Respond only with 'true' if you are certain.
If you are unsure or cannot tell confidently, respond only with 'false'.
```

#### Multi-Model Voting (applicable to both strategies)
- Send same image to 2 models (Gemini Flash + Claude Sonnet)
- Strategy A: parse both JSON responses, take intersection of positive tiles
- Strategy B: for each tile, both models must agree to mark positive
- Raises accuracy from ~65% to ~85%+ per round

#### reCAPTCHA Solve Flow (shared)
```
1. Locate reCAPTCHA iframe: document.querySelector("iframe[title='reCAPTCHA']")
2. Access iframe contentDocument
3. Click checkbox: .recaptcha-checkbox-border
4. Wait: race(token_appears, challenge_opens, timeout_3s)
5. If token → done (sometimes checkbox click is sufficient)
6. If challenge:
   a. Locate challenge iframe: iframe[title*='challenge']
   b. Extract target from .rc-imageselect-desc strong
   c. Detect grid type: .rc-imageselect-table-33 (3×3) or .rc-imageselect-table-44 (4×4)
   d. Detect dynamic: text includes "Click verify once there are none left"
   e. Screenshot/extract tiles
   f. VLM classify (Strategy A or B)
   g. Click matching tiles with random delays (200-800ms between clicks)
   h. Wait 500-1000ms
   i. Click #recaptcha-verify-button
   j. Wait: race(token, new_challenge, timeout_5s)
   k. If new challenge → repeat from (b), max 4 rounds
   l. If timeout → check if verify button disabled (= success)
7. Extract token: textarea[name="g-recaptcha-response"]
```

#### Dynamic Challenge Handling
When challenge says "Click verify once there are none left":
- Tiles fade out and new tiles appear after clicking
- Must loop: screenshot → classify → click → wait for new tiles → repeat
- Max 4 dynamic iterations before clicking verify
- Track which tiles were already clicked (by index, not image)

**Audio fallback (opportunistic):**
1. Click audio icon in challenge iframe (`.rc-button-audio`)
2. If audio loads (not blocked):
   a. Extract audio source URL from `<audio>` element
   b. Download MP3
   c. POST to local Whisper STT (`http://localhost:8115/transcribe`)
   d. Clean transcription: strip all non-alphanumeric
   e. Type into `#audio-response` input
   f. Click verify
3. If Google blocks audio ("automated queries" error) → fall back to visual

**Anti-detection measures (from research):**
- Randomized delays between all actions (200-800ms)
- Random delay before verify click (500-1000ms)
- Navigator.webdriver patched to undefined (Rod may already do this)
- User-agent from curated list (not default headless UA)
- No instant teleport clicks — use small random offsets
- Session cookies from prior authenticated browsing reduce challenge difficulty

### Type 3: Cloudflare Turnstile

Sites: Product Hunt (blocks auth), potentially others.

**Status:** Deferred. Turnstile fingerprints the browser environment deeply.
Requires undetected-chromium patches. Not solvable with VLM approach.

### Type 4: Slider/Puzzle Captcha

Sites: Some Chinese platforms, GeeTest.

**Flow (from ai-captcha-bypass research):**
1. Screenshot the puzzle
2. VLM calculates horizontal pixel distance from slider to slot
3. Simulate drag with human-like acceleration curve
4. Screenshot result, VLM checks alignment
5. Micro-correct if needed

**Status:** Deferred. Not needed for current press distribution surfaces.
Implementation pattern is documented in `ai-captcha-bypass/puzzle_solver.py`.

## API Design

### Endpoint: `POST /api/session/{id}/captcha/solve`

Solves a captcha within an active browser session.

**Request:**
```json
{
  "type": "text|recaptcha_v2|auto",
  "config": {
    "image_selector": "img.captcha",
    "answer_selector": "input[name=captcha]",
    "submit_selector": "#submit-btn",
    "auto_submit": false,
    "max_attempts": 5,
    "preprocessing": {
      "upscale": 3,
      "threshold": 100,
      "morphology_kernel": 3,
      "component_min_area": 30,
      "component_max_aspect": 8
    },
    "strategy": "full_grid"
  },
  "provider": "openrouter",
  "model": "google/gemini-2.0-flash-001",
  "voting": {
    "enabled": true,
    "models": [
      "google/gemini-2.0-flash-001",
      "anthropic/claude-3.5-sonnet"
    ]
  }
}
```

**`type: "auto"`** — Engine inspects the page DOM to auto-detect:
- reCAPTCHA iframe present → `recaptcha_v2`
- Captcha image matching `config.image_selector` → `text`

**Response (success):**
```json
{
  "solved": true,
  "type": "text",
  "answer": "m0ded",
  "attempts": 2,
  "method": "vlm:openrouter/gemini-2.0-flash-001",
  "duration_ms": 3200,
  "token": null
}
```

**Response (reCAPTCHA success):**
```json
{
  "solved": true,
  "type": "recaptcha_v2",
  "answer": null,
  "attempts": 1,
  "rounds": 3,
  "method": "vlm:full_grid:openrouter/gemini-2.0-flash-001",
  "duration_ms": 12500,
  "token": "03AGdBq24PBgq..."
}
```

**Response (failure):**
```json
{
  "solved": false,
  "type": "text",
  "answer": null,
  "attempts": 5,
  "method": "vlm:openrouter/gemini-2.0-flash-001",
  "duration_ms": 15000,
  "error": "max attempts exceeded",
  "last_answer": "m0oed",
  "debug": {
    "all_answers": ["mobed", "m0oed", "moded", "m0ded", "m0oed"],
    "preprocessing": "upscale3x_thresh100_morph3"
  }
}
```

### Endpoint: `POST /api/captcha/solve-image`

Stateless image-only solve (no browser session required).
For callers that already have the captcha image and just need the text.

**Request:**
```json
{
  "image_base64": "/9j/4AAQSkZJR...",
  "image_type": "image/png",
  "hint": "5 lowercase alphanumeric characters",
  "preprocessing": {
    "upscale": 3,
    "threshold": 100
  },
  "provider": "openrouter",
  "model": "google/gemini-2.0-flash-001"
}
```

**Response:**
```json
{
  "text": "m0ded",
  "confidence": "high",
  "method": "vlm:openrouter/gemini-2.0-flash-001",
  "alternatives": ["moded", "m0oed"],
  "duration_ms": 2100
}
```

### Endpoint: `GET /api/captcha/status`

Returns solver capabilities and health.

**Response:**
```json
{
  "available_types": ["text", "recaptcha_v2"],
  "backends": {
    "vlm": {
      "available": true,
      "provider": "openrouter",
      "models": ["google/gemini-2.0-flash-001", "anthropic/claude-3.5-sonnet"]
    },
    "tesseract": { "available": true, "version": "4.1.1" },
    "ddddocr": { "available": true },
    "whisper": { "available": true, "endpoint": "http://localhost:8115" }
  },
  "stats": {
    "total_attempts": 142,
    "total_solved": 118,
    "success_rate": 0.831,
    "by_type": {
      "text": { "attempts": 89, "solved": 78, "rate": 0.876 },
      "recaptcha_v2": { "attempts": 53, "solved": 40, "rate": 0.755 }
    }
  }
}
```

## Implementation Plan

### Phase 1: Text Captcha Solver (unblocks PRLog, OpenPR)

**Files to create/modify:**

| File | Action | Purpose |
|------|--------|---------|
| `internal/captcha/solver.go` | Create | Core solver logic, type detection, retry loop |
| `internal/captcha/text_solver.go` | Create | Text captcha: extract → preprocess → VLM → answer |
| `internal/captcha/preprocess.go` | Create | Image preprocessing pipeline (Go imaging) |
| `internal/captcha/types.go` | Create | Request/response types, config structs |
| `internal/captcha/stats.go` | Create | Solve attempt stats tracking |
| `internal/routes/captcha.go` | Create | HTTP handlers for `/api/captcha/*` and `/api/session/{id}/captcha/*` |
| `internal/server/server.go` | Modify | Mount captcha routes |
| `config.yaml` | Modify | Add `captcha:` config section |

**Dependencies (Go):**
- `github.com/disintegration/imaging` — resize, grayscale
- No external captcha-farm SDK

**Steps:**
1. Define types and config (`types.go`)
2. Implement preprocessing pipeline (`preprocess.go`)
3. Implement text solver with VLM + fallback chain (`text_solver.go`)
4. Implement session-aware solve action (`solver.go`)
5. Implement stats tracking (`stats.go`)
6. Mount HTTP routes (`captcha.go`, `server.go`)
7. Add config section (`config.yaml`)
8. Test against saved PRLog captcha samples (`/tmp/prlog-captcha*.png`)
9. Test against live OpenPR verification captcha
10. Document results (update this spec)

**Estimated effort:** 4-6 hours

### Phase 2: reCAPTCHA v2 Solver (unblocks PR.com, 24-7PR, IssueWire)

**Files to create/modify:**

| File | Action | Purpose |
|------|--------|---------|
| `internal/captcha/recaptcha_solver.go` | Create | reCAPTCHA v2: full grid screenshot strategy + per-tile fallback |
| `internal/captcha/recaptcha_dom.go` | Create | DOM interaction helpers: iframe access, checkbox, tiles, verify |
| `internal/captcha/voting.go` | Create | Multi-model voting for classification accuracy |
| `internal/captcha/solver.go` | Modify | Add reCAPTCHA dispatch |

**Steps:**
1. Implement DOM helpers for reCAPTCHA iframe navigation (eval calls via Rod)
2. Implement full-grid screenshot strategy (Strategy A)
3. Implement per-tile classification strategy (Strategy B)
4. Implement multi-model voting
5. Implement multi-round retry loop with dynamic challenge detection
6. Implement success detection (token extraction, verify button state)
7. Add randomized timing between actions
8. Test against PR.com (site key: `6LcOef8SAAAAANHRO0asSp6bjrMFbIT105J3b2ow`)
9. Test against 24-7 Press Release
10. Measure and log accuracy per round per strategy

**Estimated effort:** 6-8 hours

### Phase 3: Audio Fallback + Anti-Detection (hardening)

**Files to create/modify:**

| File | Action | Purpose |
|------|--------|---------|
| `internal/captcha/audio_solver.go` | Create | Audio download → Whisper STT → transcribe |
| `internal/captcha/stealth.go` | Create | Randomized delays, navigator patches, UA rotation |
| `internal/captcha/solver.go` | Modify | Add audio fallback, stealth config |

**Steps:**
1. Implement audio challenge detection (`rc-button-audio`)
2. Implement audio download from reCAPTCHA challenge
3. Integrate with Whisper STT at `localhost:8115`
4. Clean transcription (strip non-alphanumeric, from uncaptcha2 research)
5. Implement navigator.webdriver patch via Rod eval
6. Add user-agent rotation from curated list
7. Add random timing jitter to all DOM interactions
8. Test audio path on reCAPTCHA demo page
9. Measure detection rate with and without stealth measures

**Whisper integration:**
```go
// POST audio to local Whisper STT
resp, err := http.Post("http://localhost:8115/transcribe", 
    "audio/mpeg", audioReader)
// Clean: strip all non-alphanumeric
answer := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(text, "")
```

**Estimated effort:** 3-4 hours

### Phase 4: Per-Site Profiles + Operational Excellence

**Files to create/modify:**

| File | Action | Purpose |
|------|--------|---------|
| `internal/captcha/profiles.go` | Create | Site-specific captcha configurations |
| `config.yaml` | Modify | Add per-site captcha profiles |

**Site profiles** define per-surface captcha handling:

```yaml
captcha:
  enabled: true
  default_provider: "openrouter"
  default_model: "google/gemini-2.0-flash-001"
  text:
    max_attempts: 5
    retry_delay: "1s"
    prompt_template: "blind_assistant"
    fallback_chain: ["vlm", "tesseract", "ddddocr"]
  recaptcha:
    strategy: "full_grid"
    max_rounds: 4
    max_attempts: 4
    action_delay_min_ms: 200
    action_delay_max_ms: 800
    verify_delay_min_ms: 500
    verify_delay_max_ms: 1000
    dynamic_max_iterations: 4
    audio_fallback: true
    whisper_url: "http://localhost:8115"
  voting:
    enabled: true
    models:
      - "google/gemini-2.0-flash-001"
      - "anthropic/claude-3.5-sonnet"
    temperature: 0.1
  stealth:
    patch_webdriver: true
    random_user_agent: true
    user_agents:
      - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36"
      - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36"
      - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36"
  stats:
    enabled: true
    log_file: "/var/log/uiai/captcha-stats.jsonl"
  profiles:
    prlog:
      type: text
      image_selector: "img[src^='data:image']"
      answer_selector: "input[name=captcha_hash]"
      preprocessing:
        upscale: 3
        threshold: 100
        morphology_kernel: 3
        component_min_area: 30
      hint: "5 lowercase alphanumeric characters with crosshatch grid overlay"
      prompt_template: "blind_assistant"
    openpr:
      type: text
      image_selector: ".verification-image img"
      answer_selector: "input[name=verification]"
      preprocessing:
        upscale: 2
        threshold: 120
      hint: "uppercase alphanumeric verification code"
      prompt_template: "verification_code"
    prcom:
      type: recaptcha_v2
      site_key: "6LcOef8SAAAAANHRO0asSp6bjrMFbIT105J3b2ow"
      strategy: "full_grid"
      voting: true
    "247pressrelease":
      type: recaptcha_v2
      strategy: "full_grid"
      voting: true
```

**Estimated effort:** 2-3 hours

## VLM Provider Strategy

**Primary:** OpenRouter → `google/gemini-2.0-flash-001`
- Proved working for both text OCR and image classification
- ~2-3s per call
- Rate limit: ~4-5 rapid calls before 429 → needs backoff
- Best for full-grid reCAPTCHA analysis (single call for all tiles)

**Secondary:** OpenRouter → `anthropic/claude-3.5-sonnet`
- Proved working for image classification
- Used for multi-model voting (intersection of answers)
- Slightly more verbose but accurate

**VLM parameters (from research):**
- Text captcha: temperature 1.0, maxTokens 256
- reCAPTCHA classification: temperature 0.1, topP 0.95, topK 40, maxTokens 500
- Instruction extraction: temperature 0, maxTokens 50

**Future/local:** Ollama → any vision model
- No GPU on current server (14GB RAM, VPS)
- When GPU available: `llava:34b`, `moondream2`, or `minicpm-v`
- Config: add `ollama` provider with local URL

**Fallback chain for text captchas:**
1. VLM (OpenRouter/Gemini Flash) — best accuracy, ~2s
2. VLM (OpenRouter/Claude Sonnet) — if Gemini rate-limited
3. Tesseract + preprocessing — if all VLM fails, ~500ms
4. ddddocr (ONNX) — if Tesseract too noisy, ~200ms

## Cost Analysis

| Operation | Cost | Notes |
|-----------|------|-------|
| Text captcha (1 VLM call) | ~$0.0004 | $0.40/1K images via OpenRouter |
| Text captcha with retry (avg 2 calls) | ~$0.0008 | Most solve in 1-2 attempts |
| reCAPTCHA full-grid (1 call per round, avg 3 rounds) | ~$0.0012 | 3 × $0.0004 |
| reCAPTCHA with voting (2 models × 3 rounds) | ~$0.0024 | 6 VLM calls |
| Audio fallback via Whisper | $0.00 | Local, no API cost |
| Tesseract/ddddocr fallback | $0.00 | Local inference |
| **Per press release (1 captcha)** | **~$0.001-$0.003** | Negligible |
| **1000 press releases** | **~$1-3** | Less than a coffee |

## Security & Ethics

1. **No captcha-farm dependency** — all solving is AI/local, no human labor marketplace
2. **Own accounts only** — solver is used for our own press release submissions,
   not for scraping or abuse
3. **Rate limiting** — built-in randomized delays prevent hammering target sites
4. **API key isolation** — uses the same AI provider keys as the rest of the engine
5. **Audit logging** — every solve attempt logged: timestamp, type, result, duration, method
6. **Fail-open policy** — if solver fails, adapter reports honestly; no fake states

## Relationship to Existing Code

### Session API
The captcha solver is a **session action**, like `fill`, `click`, or `eval`.
It operates within an existing session and uses the session's browser page.
The session must already be navigated to the page containing the captcha.

### AI Provider (`internal/ai/provider.go`)
The solver calls `ai.Provider.Complete()` with `ImageBase64` set — the same
path used by `reference.Analyzer.callVision()`. No new AI plumbing needed.

### Reference Analyzer (`internal/reference/analyzer.go`)
The existing `ExtractText()` method already does VLM-based text extraction
with Tesseract fallback. The text captcha solver reuses this pattern but
with captcha-specific prompts and preprocessing.

### Wire Pitch Adapters (PHP callers)
Adapters call the captcha solver through the UIAI client:
```php
// In adapter distribute() method, after navigating to form
$result = $this->uiai->solve_captcha($sid, [
    'type'            => 'auto',
    'profile'         => 'prlog',
    'auto_submit'     => false,
]);
if (!$result['solved']) {
    return ['success' => false, 'error' => 'Captcha solve failed: ' . $result['error']];
}
// Continue with form submission...
```

### OpenPR Adapter (existing)
The OpenPR adapter currently has hand-coded captcha/verification handling
in `fill_verification_text()`. After Phase 1 ships, this should be refactored
to call the captcha solver endpoint — single code path for all captchas.

## Testing Strategy

### Offline Tests (saved samples)
- 10+ saved PRLog captcha images (`/tmp/prlog-captcha*.png`)
- Saved OpenPR verification images
- Target: >80% accuracy on text captchas with VLM
- Benchmark each provider/model combination

### Live Tests
- PRLog registration (after IP unblock)
- OpenPR submission verification step
- PR.com reCAPTCHA (site key known)
- 24-7 Press Release reCAPTCHA
- reCAPTCHA demo page (`https://www.google.com/recaptcha/api2/demo`)

### Benchmark Script
```bash
# Text captcha accuracy on saved samples
for f in /tmp/prlog-captcha*.png; do
  curl -s -X POST http://localhost:7456/api/captcha/solve-image \
    -H "Content-Type: application/json" \
    -d "{\"image_base64\":\"$(base64 -w0 $f)\",\"image_type\":\"image/png\",\"hint\":\"5 lowercase alphanumeric\"}"
  echo ""
done
```

## Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Text captcha first-attempt accuracy | >80% | Tested on 10+ saved samples |
| Text captcha solve rate (5 attempts) | >95% | Logged per-surface |
| reCAPTCHA v2 solve rate (4 attempts) | >70% | Logged per-surface |
| Average text solve time | <5s | Logged |
| Average reCAPTCHA solve time | <30s | Logged |
| Zero external service dependencies | No captcha-farm API keys | Architecture review |
| All adapters use single path | No hand-coded captcha logic in adapters | Code review |

## Open Questions

1. **Ollama on this server?** 14GB RAM, no GPU. Running a VLM locally would be
   slow (~30-60s per inference) and consume memory. Defer until GPU available.

2. **ddddocr as Go subprocess vs microservice?** Options:
   a. Shell out to Python script (simplest, ~500ms overhead) — chosen for now
   b. Wrap in HTTP microservice (clean separation) — if latency matters
   c. Port ONNX model to Go via `onnxruntime-go` — if critical path

3. **Full-grid vs per-tile for reCAPTCHA?** Research shows full-grid is better:
   fewer API calls, VLM sees spatial context, single-call per round.
   Per-tile kept as fallback if full-grid accuracy is insufficient.

4. **IP rotation for blocked sites?** PRLog blocked our IP after ~20 attempts.
   Options: Cloudflare Worker proxy, rotating residential proxies, or wait.
   Deferred to per-site profile config.

5. **Rod vs undetected-chromium?** Current Rod setup may trigger automation detection.
   The `puppeteer-extra-plugin-stealth` patches (from recaptcha-v2-solver) need
   Rod equivalents. Some patches (navigator.webdriver) can be done via eval.
   Full stealth may require Rod launcher flags research.

## Appendix A: Proved Capabilities (from prior work)

| Capability | Status | Evidence |
|-----------|--------|----------|
| reCAPTCHA iframe DOM access | ✅ Proved | contentDocument across iframe boundary |
| reCAPTCHA checkbox click | ✅ Proved | `.recaptcha-checkbox-border.click()` |
| reCAPTCHA tile image extraction | ✅ Proved | Canvas toDataURL as base64 |
| reCAPTCHA challenge info extraction | ✅ Proved | `.rc-imageselect-desc strong` |
| reCAPTCHA tile click | ✅ Proved | Individual tile click |
| reCAPTCHA verify button | ✅ Proved | `#recaptcha-verify-button` |
| reCAPTCHA success detection | ✅ Proved | `aria-checked="true"` |
| VLM tile classification | ✅ Proved | Gemini Flash ~2-3s/call |
| Multi-model VLM | ✅ Proved | Gemini + Claude via OpenRouter |
| Text captcha OCR via VLM | ✅ Proved | OpenPR verification solved |
| Tesseract preprocessing | ✅ Proved | Multiple threshold/PSM variants |
| ddddocr ONNX solving | ✅ Proved | Installed, functional |
| Canvas extraction at 4x | ✅ Proved | Higher resolution improves OCR |
| Connected component filtering | ✅ Proved | Removes grid lines |
| reCAPTCHA audio challenge | ❌ Blocked | Google detects automation |
| Whisper STT | ✅ Running | `localhost:8115`, model: base |

## Appendix B: PRLog Captcha Characteristics

- **Size:** 180×60 pixels
- **Characters:** 5 lowercase alphanumeric
- **Distortion:** Heavy crosshatch grid overlay (diagonal intersecting lines)
- **Background:** Noisy gradient
- **Font:** Variable width, slight rotation per character
- **Color:** Dark characters on lighter background
- **Samples:** `/tmp/prlog-captcha*.png`, `/tmp/prlog-cap*.png`
- **Cleaned samples:** `/tmp/prlog-clean-*.png`, `/tmp/prlog-v2-clean.png`

## Appendix C: reCAPTCHA v2 Site Keys (Known)

| Site | Site Key | Notes |
|------|----------|-------|
| PR.com | `6LcOef8SAAAAANHRO0asSp6bjrMFbIT105J3b2ow` | Registration form |
| 24-7 Press Release | TBD | Registration form |
| IssueWire | TBD | Registration + PayPal |
| Google Demo | `6Le-wvkSVVABCPBMRTvw0Q4Muexq1bi0DJwx_mJ-` | Test page |

## Appendix D: Reference Implementation Links

- **Full grid reCAPTCHA solver:** https://github.com/njraladdin/recaptcha-v2-solver/blob/main/lib/generateCaptchaTokensWithVisual.js
- **VLM prompt library:** https://github.com/aydinnyunus/ai-captcha-bypass/blob/main/ai_utils.py
- **Audio reCAPTCHA bypass:** https://github.com/ecthros/uncaptcha2
- **Stealth plugin patterns:** https://github.com/nicknsy/puppeteer-extra-plugin-stealth (various detection patches)
- **Awesome captcha (index):** https://github.com/ZYSzys/awesome-captcha
- **CNN+LSTM captcha trainer:** https://github.com/kerlomz/captcha_trainer (for custom model training if VLM insufficient)

## Revision History

| Date | Change |
|------|--------|
| 2026-03-09 | Initial spec from PRLog/PR.com sprint findings |
| 2026-03-09 | v2: Deep research pass — ai-captcha-bypass prompts, recaptcha-v2-solver grid strategy, uncaptcha2 audio insights, CaptchaWatcher patterns, anti-detection best practices, Whisper STT integration at localhost:8115 |
