# Workflow API Remote Orchestration

**Updated:** 2026-02-08

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
```
POST /api/reference/analyze?pass=full
{
  "imageBase64": "...",
  "imageType": "image/webp"
}
```

**Passes:**
- `analyze` — Overall design analysis
- `components` — Component identification
- `tokens` — Design token extraction (colors, fonts, spacing)
- `spacing` — Spacing system analysis
- `full` — All 4 passes sequentially

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

All endpoints require a valid license key in the `Authorization` header:
```
Authorization: Bearer <license_key>
```

The Go engine validates this against the WordPress license API.

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
