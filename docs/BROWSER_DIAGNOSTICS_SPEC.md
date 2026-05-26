# Browser Diagnostics Spec — Console, Exceptions, Network, and Flake Evidence

**Status:** implemented baseline for local UIAI Engine session API.  
**Scope:** `localhost:7456` browser sessions backed by the existing Rod/Chrome pool.  
**Companion Focusa spec:** `/home/wirebot/focusa/docs/current/UIAI_BROWSER_DIAGNOSTICS_FOCUSA_INTEGRATION_SPEC.md`.

## 1. Problem

The Browser Session API is strong for visual QA and interaction. The implemented diagnostics baseline now exposes bounded Chrome DevTools-style console, exception, and network streams as first-class session API data.

Current implemented surfaces include screenshots, snapshot/@ref, DOM, eval, cookies/auth, navigation, form actions, captcha integration, `browser_diagnostics`, and `browser_diagnostics_clear`. Full HAR, trace export, source-map mapping, and raw body/header capture remain out of scope for the baseline.

This closes the main flake-debugging gap: agents can inspect browser evidence instead of relying on screenshots alone.

## 2. Goals

- Expose console logs, warnings, errors, and JS exceptions per browser session.
- Expose bounded network request/response/failure data per browser session.
- Improve flake diagnosis without adding Playwright/Puppeteer or a second browser runtime.
- Keep diagnostics lightweight enough for agent loops.
- Return model-friendly, redacted, bounded JSON suitable for Focusa evidence ingestion.

## 3. Non-goals

- Full Chrome DevTools UI parity.
- Unbounded HAR capture.
- Raw secret/header/body exfiltration by default.
- Replacing screenshots, snapshot/@ref, or existing UIAI interaction tools.
- Adding a new browser dependency.

## 4. Current baseline

Implemented session routes in `internal/routes/session.go`:

- `GET /api/session`
- `POST /api/session`
- `GET /api/session/{id}`
- `DELETE /api/session/{id}`
- `POST /api/session/{id}/screenshot`
- `POST /api/session/{id}/navigate`
- `POST /api/session/{id}/scroll`
- `POST /api/session/{id}/click`
- `POST /api/session/{id}/hover`
- `POST /api/session/{id}/type`
- `POST /api/session/{id}/eval`
- `POST /api/session/{id}/resize`
- `POST /api/session/{id}/css`
- `POST /api/session/{id}/fill`
- `POST /api/session/{id}/select`
- `POST /api/session/{id}/press`
- `POST /api/session/{id}/back`
- `POST /api/session/{id}/forward`
- `POST /api/session/{id}/text`
- `POST /api/session/{id}/cookies`
- `POST /api/session/{id}/auth/save`
- `POST /api/session/{id}/auth/load`
- `GET/POST /api/session/{id}/snapshot`
- `GET /api/session/{id}/dom`
- `POST /api/session/{id}/wait`

Current session state in `internal/vision/session.go` stores identity, URL/title, viewport, counters, Rod page, timer, mutex, @ref map, diagnostics recorder, and diagnostics cancel function. The recorder implementation lives in `internal/vision/diagnostics.go`.

## 5. Implemented API

### `GET /api/session/{id}/diagnostics`

Return bounded diagnostic data for the active session. This endpoint MUST NOT take a screenshot by default.

Query parameters:

| Param | Default | Description |
|---|---:|---|
| `limit` | `100` | Max events per category, capped by server max. |
| `level` | `all` | `all`, `error`, `warning`, `info`. |
| `failed_only` | `false` | Return only failed/4xx/5xx network events in the `network` list. |

Not implemented in the baseline: `since_seq`, header capture, body capture, HAR export, and trace export.

Example response:

```json
{
  "session_id": "abc12345",
  "url": "https://example.test/app",
  "title": "Example App",
  "seq": 42,
  "generated_at": "2026-05-26T22:00:00Z",
  "console": [
    {
      "seq": 12,
      "ts": "2026-05-26T22:00:01Z",
      "level": "error",
      "text": "Uncaught TypeError: Cannot read properties of undefined",
      "args_preview": ["Uncaught TypeError: Cannot read properties of undefined"],
      "url": "https://example.test/assets/app.js",
      "line": 88,
      "column": 14
    }
  ],
  "exceptions": [
    {
      "seq": 13,
      "ts": "2026-05-26T22:00:01Z",
      "text": "Cannot read properties of undefined",
      "url": "https://example.test/assets/app.js",
      "line": 88,
      "column": 14,
      "stack_preview": "TypeError: Cannot read properties of undefined\n    at renderWidget ..."
    }
  ],
  "network": [
    {
      "seq": 21,
      "request_id": "1234.5",
      "method": "GET",
      "url": "https://example.test/api/items",
      "resource_type": "fetch",
      "status": 500,
      "mime_type": "application/json",
      "duration_ms": 84,
      "failed": true,
      "failure_reason": "HTTP 500"
    }
  ],
  "failed_requests": [
    {
      "seq": 21,
      "method": "GET",
      "url": "https://example.test/api/items",
      "status": 500,
      "failure_reason": "HTTP 500"
    }
  ],
  "summary": {
    "console_errors": 1,
    "exceptions": 1,
    "requests": 18,
    "failed_requests": 1,
    "http_4xx": 0,
    "http_5xx": 1
  }
}
```

### `POST /api/session/{id}/diagnostics/clear`

Clear all session diagnostic buffers and reset sequence counters for that session.

Response:

```json
{ "status": "cleared", "session_id": "abc12345" }
```

### Failure-response enrichment

When an existing action fails, the response SHOULD include a small diagnostic summary when available:

```json
{
  "error": "selector_not_found",
  "url": "https://example.test/app",
  "diagnostics_summary": {
    "console_errors": 1,
    "exceptions": 0,
    "failed_requests": 2,
    "last_error": "GET /api/items returned 500"
  }
}
```

## 6. Implemented MCP/OpenAI tool names

Add to `/api/tools` and MCP bridge metadata:

- `browser_diagnostics` — read current session diagnostics.
- `browser_diagnostics_clear` — clear session diagnostic buffers.

Tool parameters for `browser_diagnostics`:

```json
{
  "session_id": "abc12345",
  "limit": 100,
  "level": "all",
  "failed_only": false,
  "include_headers": false
}
```

## 7. Rod/CDP event sources

Attach listeners when a session is opened or wrapped:

- `Runtime.consoleAPICalled` → console buffer.
- `Runtime.exceptionThrown` → exception buffer.
- `Network.requestWillBeSent` → request start buffer/map.
- `Network.responseReceived` → response status/timing classification.
- `Network.loadingFailed` → failed request classification.

Enable the required domains for session pages:

- `Runtime.enable`
- `Log.enable` if used for browser log entries.
- `Network.enable`

## 8. Data model

Add bounded buffers to `vision.Session` or a dedicated embedded diagnostics recorder:

```go
type DiagnosticRecorder struct {
    mu sync.Mutex
    seq uint64
    console RingBuffer[ConsoleEvent]
    exceptions RingBuffer[ExceptionEvent]
    network RingBuffer[NetworkEvent]
    requests map[proto.NetworkRequestID]*NetworkEvent
}
```

Default caps:

- console: 200 events
- exceptions: 100 events
- network: 300 events
- failed_requests view: derived from network buffer

Buffers MUST be per-session and cleaned when the session closes or expires.

## 9. Redaction and safety

Default redaction rules:

- Never return `Cookie`, `Authorization`, `Proxy-Authorization`, `Set-Cookie`, API keys, bearer tokens, passwords, or known secret-looking values.
- Query strings may be preserved for local debugging, but secret-like query keys MUST be redacted.
- Request/response bodies are omitted by default.
- Body capture, if added later, must be separately gated and size-limited.

## 10. Performance and flake-hardening requirements

Diagnostics should improve reliability, not add instability.

Requirements:

- Diagnostics read path MUST NOT force a screenshot.
- Event handlers MUST use bounded append-only buffers and avoid blocking Rod event loops.
- Session actions should use explicit timeouts and typed error classes:
  - `timeout`
  - `selector_not_found`
  - `navigation_failed`
  - `page_crashed`
  - `browser_unavailable`
  - `diagnostics_unavailable`
- Page health checks should distinguish stale page, closed page, browser crash, and selector failure.
- Agent loops should prefer session reuse and diagnostics reads over repeated full navigation.
- Default screenshots for agent loops should prefer JPEG with modest quality unless full-fidelity proof is needed.

## 11. Focusa integration contract

UIAI diagnostics are evidence, not authority. Focusa should ingest bounded diagnostic snapshots through its existing evidence/prediction/Workpoint flow:

1. Agent opens or reuses a UIAI browser session.
2. Agent reproduces the page issue.
3. Agent reads `browser_diagnostics`.
4. Agent captures a stable evidence reference in Focusa.
5. Focusa active object resolution maps URL/stack/API routes to likely project files.
6. Focusa prediction records the likely cause/fix path.
7. Fix verification captures a second diagnostics snapshot proving console/network clean or improved.

See companion spec: `/home/wirebot/focusa/docs/current/UIAI_BROWSER_DIAGNOSTICS_FOCUSA_INTEGRATION_SPEC.md`.

## 12. Acceptance checks

- `GET /api/session/{id}/diagnostics` returns console/error/network arrays without taking a screenshot.
- A page with `console.error("x")` records one console error.
- A page that throws an uncaught JS exception records one exception with source location when available.
- A page that requests a missing URL records one failed request.
- Existing screenshot/snapshot/click/type flows still pass.
- `/api/tools` lists `browser_diagnostics` and `browser_diagnostics_clear`.
- Redaction test proves auth/cookie headers are not returned.
- Focusa docs cross-reference this spec and define evidence ingestion shape.
