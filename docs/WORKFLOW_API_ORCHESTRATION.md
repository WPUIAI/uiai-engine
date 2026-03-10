# Workflow API Remote Orchestration

**Updated:** 2026-03-06

## Architecture

The parent WordPress plugin (WPUIAI) orchestrates site builds locally. When a cloud
license is active, the **Capability Router** delegates AI-intensive operations to the
Go engine (uiai-engine) via HTTP.

```
┌──────────────────────┐        ┌───────────────────────────┐
│  WordPress Plugin    │        │  Go Engine (uiai-engine)   │
│                      │  HTTP  │                            │
│  Capability Router  ─┼───────►│  /api/critique             │
│  Workflow Enforcer   │        │  /api/reference/analyze    │
│  Mimic Intelligence  │        │  /api/intake               │
│  Design Critic       │        │  /api/vision/*             │
│  Cloud API helper    │        │  /api/screenshot           │
│                      │        │  /api/share                │
└──────────────────────┘        └───────────────────────────┘
         │                                │
    WordPress DB              Cloudflare Tunnel
    + local files             (ai.wpuiai.com)
```

## Endpoint Map

### Critical Path (Capability Router)

| Capability | Cloud Endpoint | Plugin Caller | Fallback |
|------------|---------------|---------------|----------|
| `critique` | `POST /api/critique` | `call_critique()` | `Design_Critic::critique()` |
| `ui_reverse` | `POST /api/reference/analyze` | `call_ui_reverse()` | local LLM |
| `section_detect` | `POST /api/critique` (mode=section) | `call_section_detect()` | local LLM |
| `layout_compare` | `POST /api/critique` (mode=layout) | `call_layout_compare()` | local LLM |
| `style_enhance` | `POST /api/critique` (mode=style) | `call_style_enhance()` | local LLM |
| `copilot` | `POST /api/intake` | `call_copilot()` | local LLM |

### Captcha Solver (Wire Pitch / Press Distribution)

| Capability | Endpoint | Purpose |
|------------|----------|---------|
| `solve_image` | `POST /api/captcha/solve-image` | Stateless text captcha solve from base64 image |
| `solve_session` | `POST /api/session/{id}/captcha/solve` | Session-scoped solve (text + reCAPTCHA v2) |
| `solve_proxied` | `POST /api/captcha/solve-proxied` | One-shot proxied browser solve with form fill |
| `captcha_status` | `GET /api/captcha/status` | Backend availability and solve stats |
| `pool_status` | `GET /api/captcha/pool` | Per-IP health, success rates, cooldown state |
| `pool_add` | `POST /api/captcha/pool/add` | Add IP at runtime (no restart) |
| `pool_remove` | `POST /api/captcha/pool/remove` | Remove IP at runtime |

IP pool uses weighted rotation with per-IP health tracking, auto-cooldown on flag detection, and concurrent solve limits. IPs at $1/mo each.

See [`CAPTCHA_SOLVER_SPEC.md`](CAPTCHA_SOLVER_SPEC.md) for full API reference, accuracy data, and scale model.

### Media Frame Pipeline (GitHub-vendored device frames)

| Capability | Cloud Endpoint | Purpose |
|------------|---------------|---------|
| `frame_catalog` | `GET /api/media/frame/catalog` | List approved frame IDs + metadata |
| `frame_render` | `POST /api/media/frame/render` | Composite screenshot into selected frame |

Example render request:

```json
POST /api/media/frame/render
{
  "frameId": "macbook-pro-16-silver",
  "imageBase64": "...",
  "fit": "cover",
  "format": "jpeg",
  "quality": 90,
  "scale": 1
}
```

### Critique Endpoint Contract

**Request:**
```json
POST /api/critique
{
  "websiteUrl": "https://example.com",
  "model": "claude-sonnet-4-20250514",
  "provider": "anthropic",
  "pageType": "homepage",
  "critiqueMode": "mimic",
  "referenceUrl": "https://reference.com",
  "designTokens": {
    "colors": {"primary": "#1a365d"},
    "typography": {"heading_font": "Inter"},
    "spacing": {"base": "16px"}
  },
  "imageBase64": "...",
  "imageType": "image/webp"
}
```

**Response (success):**
```json
{
  "success": true,
  "scores": {
    "layout": 7,
    "typography": 8,
    "color_palette": 6,
    "visual_hierarchy": 7,
    "consistency": 8,
    "accessibility": 5,
    "emotional_impact": 7,
    "overall_quality": 6.9
  },
  "critique": [
    {
      "dimension": "layout",
      "score": 7,
      "observations": ["..."],
      "suggestions": ["..."]
    }
  ],
  "priority_fixes": [
    {
      "fix": "Increase body text contrast",
      "dimension": "accessibility",
      "impact": "high",
      "css": "body { color: #1a1a1a; }"
    }
  ],
  "summary": "..."
}
```

**Response (parse failure):**
```json
{
  "success": false,
  "parse_error": "unexpected EOF at position 1234",
  "content": "raw LLM text..."
}
```

The plugin handles `parse_error` by attempting local JSON repair via `Design_Critic::parse_critique_response()`.

### Reference Analysis Endpoint

**Request:**
```json
POST /api/reference/analyze
{
  "imageBase64": "...",
  "imageType": "image/webp",
  "provider": "openrouter",
  "model": "openai/gpt-4o-mini-2024-07-18",
  "pass": "full"
}
```

**Passes:**
- `analyze` — Overall design analysis
- `components` — Component identification
- `tokens` — Design token extraction (colors, fonts, spacing)
- `spacing` — Spacing system analysis
- `text` — OCR-style visible text extraction from an image
- `full` — All 4 analysis passes sequentially

### OCR / Verification Extraction

The `text` pass is intended for reading visible human-readable text from screenshots or cropped regions, including verification strings and short captcha-style tokens on third-party forms.

Example:

```json
POST /api/reference/analyze
{
  "imageBase64": "...",
  "imageType": "image/jpeg",
  "pass": "text"
}
```

#### Route behavior

`POST /api/reference/analyze` intentionally treats OCR a little differently from the other reference-analysis passes:

- if the caller omits `provider` / `model`, the route clears them before calling `ExtractText()`
- that lets the OCR layer itself choose the best runtime path instead of inheriting an unrelated generic default from another caller
- callers may still force a provider/model explicitly, but should avoid doing so unless they truly need to override runtime policy

#### Current OCR stack

Current runtime behavior for `pass: "text"` is:

1. prompt asks for **plain text only**
2. if the image looks like a verification challenge / captcha, the prompt asks for **only the uppercase alphanumeric token**
3. when provider/model are omitted, OCR prefers the runtime MiniMax path instead of a hardcoded OpenRouter caller route
4. MiniMax image OCR is resolved in two stages:
   - **first:** MiniMax Coding Plan VLM / MCP path at `POST /v1/coding_plan/vlm`
   - **fallback:** raw MiniMax text API example path at `POST /v1/text/chatcompletion_v2`
5. if MiniMax OCR fails or returns empty text, `uiai-engine` now falls back to **local Tesseract**
6. local fallback preprocesses the crop with ImageMagick `convert`, runs several variants + PSM modes, normalizes to uppercase alphanumerics, and picks the best-scoring short token

#### MiniMax findings that matter

- Official MiniMax OpenAPI docs at `https://platform.minimax.io/docs/api-reference/text/api/openapi.json` document multimodal/image requests on `POST /v1/text/chatcompletion_v2`
- Official image example uses model `MiniMax-Text-01` with `messages[].content[]` blocks of type `text` + `image_url`
- MiniMax Coding Plan image understanding is separately exposed through `POST /v1/coding_plan/vlm`
- raw MiniMax text/image responses may return HTTP 200 while still embedding a real failure in `base_resp.status_code`
- callers and providers must treat non-zero `base_resp.status_code` as a failure even on HTTP 200
- on this host, raw `MiniMax-Text-01` paygo-style image calls were observed returning `base_resp.status_code=1008` / `status_msg="insufficient balance"`, while the Coding Plan VLM path worked

#### Local OCR fallback

Local fallback currently depends on:

- `tesseract`
- ImageMagick `convert`

Local fallback is currently **MiniMax-path fallback**, not a generic all-provider OCR chain.
That means:

- MiniMax fail/empty → try local Tesseract
- explicit non-MiniMax provider failures are not automatically rerouted through local OCR here

#### Example response

```json
{
  "success": true,
  "data": {
    "text": "13FQM1",
    "warnings": [
      "OCR defaulted provider to MiniMax to avoid caller-side hardcoded OpenRouter routing.",
      "MiniMax OCR switched away from the generic M2.5 text route; Coding Plan keys may use coding-plan VLM, while generic text API image examples use MiniMax-Text-01."
    ]
  }
}
```

#### Notes / caller policy

- `text` returns plain extracted text in `data.text`; any runtime notes appear in `data.warnings`
- do not hardcode OCR to a specific provider/model in callers when runtime defaults exist
- current host policy: prefer subscribed providers first, with MiniMax as the OCR-default path
- OpenRouter is intentionally excluded from automatic OCR fallback on this host
- captcha-style verification remains best-effort; if extraction is ambiguous, caller should fall back to human assist instead of blindly submitting
- best results still come from **tight crops** around the verification image rather than full-page screenshots
- **For dedicated captcha solving** (text captchas, reCAPTCHA v2), see [`CAPTCHA_SOLVER_SPEC.md`](CAPTCHA_SOLVER_SPEC.md) — **implemented and live** as `POST /api/session/{id}/captcha/solve` and `POST /api/captcha/solve-image`. Uses multi-model VLM voting (Gemini Flash), multi-pass preprocessing, proxy-rotated browsers, and per-site profiles. ~80% first-attempt accuracy on text captchas, >99% with retries

### Vision Interactive Endpoints

All under `/api/vision/`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/state` | GET | Current page state + DOM analysis |
| `/capture` | POST | Screenshot current view |
| `/look` | POST | Navigate to URL + screenshot |
| `/inject` | POST | Inject CSS + screenshot |
| `/el?sel=.class` | GET | Element-level screenshot |
| `/multi` | POST/GET | Desktop + tablet + mobile screenshots |
| `/diff` | GET | Visual diff against previous capture |
| `/viewport?w=&h=` | GET | Set viewport dimensions |
| `/analyze` | GET | DOM analysis only (no screenshot) |
| `/critique` | POST | Navigate + capture + DOM analysis bundle |
| `/regression` | POST | Before/after diff with threshold |

### Intake / Plan Enrichment

```
POST /api/intake
{
  "intake_data": {...},
  "plan": {...}
}
```

Returns enriched plan with navigation architecture, content suggestions, missing pages.

## Cloud is Opt-In Premium

**Cloud AI is NEVER required.** It's a premium upgrade option.

The user controls cloud via Settings → AI tab:
1. **Global toggle**: `uiai_use_ai_cloud` — master on/off
2. **Per-capability**: checkboxes for each AI operation
3. **Filter**: `wpuiai_cloud_capability_enabled` for programmatic control

If cloud is disabled (or not purchased), ALL AI runs locally with user's own API keys.
The plugin works fully without any cloud connection.

### Gate Order (is_cloud_available)

```
1. Capability exists in registry?     → No: local
2. User opted in? (global toggle)      → No: local
3. Capability enabled? (per-cap)       → No: local
4. Circuit breaker open? (degraded)    → Yes: local
5. API base URL configured?            → No: local
6. License key present?                → No: local
7. Cloud provider available?           → No: local
All pass → try cloud (with graceful degradation to local on failure)
```

## Authentication

Protected endpoints accept any of these auth headers:

```text
Authorization: Bearer <license_key>
X-License-Key: <license_key>
X-API-Key: <api_key>
X-Webhook-Secret: <internal_webhook_secret>
```

Notes:
- `X-Webhook-Secret` is the server-to-server path for trusted local callers on the same host.
- Public capability consumers should continue using license or API key auth.
- Public info endpoints like `/api/health`, `/api/ui-reverse/models`, and `/api/ui-reverse/operations` remain readable without auth.

## Graceful Degradation (CRITICAL)

**`cloud_first` NEVER means `cloud_only`.** Every cloud call must degrade gracefully.

### Per-Call Degradation

Every `call_*` method in the Capability Router follows this pattern:

```
1. is_cloud_available()?
   → NO (circuit breaker open, no license, not configured)
     → Skip cloud entirely, go direct to local
   → YES
     → Try cloud
       → Success? Return cloud result (cloud_path=true)
       → Fail? Record failure, log error, fall through to local
2. Local execution
   → Returns result with fallback_from_cloud=true if cloud was attempted
```

### Circuit Breaker (Cloud_Status)

| State | Behavior | Trigger |
|-------|----------|---------|
| **Healthy** | Try cloud first, fall back on failure | Default |
| **Degraded** | Skip cloud entirely (instant local) | 3+ failures in 10 minutes |
| **Recovery** | Clear degraded state, resume cloud | 1 successful cloud call |

Timeline during outage:
```
Call 1: Try cloud → fail (60s timeout) → local fallback [slow]
Call 2: Try cloud → fail (60s timeout) → local fallback [slow]
Call 3: Try cloud → fail → threshold hit → DEGRADED [slow]
Call 4+: Skip cloud → local immediately [fast, no 60s wait]
...
Recovery probe succeeds → HEALTHY → resume cloud
```

### Error Channels

1. **5xx from Go engine** → `Cloud_API::request()` retries next API base → returns WP_Error
2. **Timeout (>60s)** → WP_Error → capability router degrades to local
3. **parse_error in response** → Plugin attempts local JSON repair via `Design_Critic::parse_critique_response()`
4. **Circuit breaker open** → `is_cloud_available()` returns false → instant local (no network call)
5. **All paths logged** → `error_log("[WPUIAI] Cloud X failed, falling back to local")`

### Response Tagging

| Field | Meaning |
|-------|---------|
| `cloud_path: true` | Cloud AI processed this request |
| `fallback_from_cloud: true` | Cloud was attempted but failed; local handled it |
| Neither | Direct local execution (no cloud license or cloud not configured) |

## Callers

After uiai-791/797 fixes, ALL 8 critique callers route through Capability Router:

1. `class-workflow-enforcer.php` — `run_ai_critique()` (Pass 6)
2. `class-workflow-enforcer.php` — Step 21.2 verify critique
3. `class-workflow-enforcer.php` — Step 21.7 re-critique after fix
4. `class-copilot-autopilot.php` — `run_critique` step
5. `class-copilot-autopilot.php` — `verify_improvements` step
6. `class-copilot-executor.php` — critique execution
7. `class-copilot-batch.php` — `batch_critique()` loop
8. `class-copilot-mcp.php` — `tool_critique_page()` MCP tool
