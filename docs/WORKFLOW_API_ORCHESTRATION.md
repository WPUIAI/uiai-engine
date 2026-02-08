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

## Authentication

All endpoints require a valid license key in the `Authorization` header:
```
Authorization: Bearer <license_key>
```

The Go engine validates this against the WordPress license API.

## Error Handling

1. **5xx from Go engine** → Plugin retries next API base (failover)
2. **Timeout (>60s)** → Plugin falls back to local
3. **parse_error in response** → Plugin attempts local JSON repair
4. **Cloud degraded** → `WPUIAI_Cloud_Status` records failure; after N failures, auto-fallback to local

## Cloud Status Tracking

The `WPUIAI_Cloud_Status` singleton tracks:
- Last success/failure timestamps
- Consecutive failure count
- Circuit breaker state (open/closed/half-open)

When circuit is open, all requests fall back to local without attempting cloud.

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
