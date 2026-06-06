# WPUIAI Research Card Integration Contract

Purpose: define how the WordPress plugin should save/display UIAI Engine Source-to-Markdown product objects without exposing raw engine internals.

Status: implemented across engine contract plus plugin save/display flow in `/home/wpuiai/public_html/wp-content/plugins/wpuiai` (`uiai-833`).

## Scope

- Engine endpoint: `POST /api/markdown`
- Engine objects:
  - `wpuiai.research_card` with schema `wpui.source_markdown_research_card.v1`
  - `wpuiai.report` with schema `wpui.source_markdown_report.v1`
- Evidence refs:
  - primary: `uiai-source-markdown:sha256:<prefix>`
  - record refs: `uiai-source-markdown:sha256:<prefix>#record=N`
  - chunk refs: `uiai.source_markdown_chunk.v1` objects when structured records exist

## Plugin workflow

```text
URL input -> call UIAI /api/markdown -> validate schemas -> save research card -> display bounded excerpt/report -> allow source reuse in WPUIAI workflows
```

## Request contract

The plugin should call:

```json
{
  "url": "https://example.com/source",
  "mode": "main_content",
  "format": "json",
  "max_chars": 30000,
  "include_links": true,
  "include_images": false
}
```

Use `format=jsonl` / JSONL only when the UI needs record/chunk lines for adapter-supported structured sources. The plugin should never store raw diagnostics, raw browser session state, cookies, auth headers, or full unbounded page bodies.

## Response fields the plugin may persist

Persist only bounded, product-facing fields:

| Field | Source | Required | Notes |
|---|---|---:|---|
| `schema` | top-level | yes | Must be `uiai.source_markdown.v1`. |
| `url` | top-level | yes | Sanitized canonical/source URL. |
| `title` | top-level | no | Display title. |
| `metadata.source_type` | top-level metadata | yes | E.g. `webpage`, `github_issue`, `reddit_thread`, `x_thread`. |
| `metadata.adapter` | top-level metadata | yes | Adapter used by engine. |
| `focusa.evidence_ref` | top-level focusa | yes | Stable proof handle. |
| `wpuiai.research_card` | product object | yes | Primary card payload. |
| `wpuiai.report` | product object | yes | Summary/report payload. |
| `records[].evidence_ref` | top-level records | no | Persist refs/counts; avoid raw bulky record bodies unless explicitly requested. |
| `chunks[]` | top-level chunks | no | Store chunk refs/indexes for structured adapters. |
| `duration_ms`, `chars`, `truncated` | top-level | no | Operational display/debug only. |

## Minimum storage shape

Recommended plugin DB/storage record:

```json
{
  "schema": "wpui.saved_research_card.v1",
  "source_url": "https://example.com/source",
  "source_type": "webpage",
  "adapter": "webpage_browser",
  "title": "Source title",
  "markdown_excerpt": "bounded excerpt from research_card",
  "evidence_ref": "uiai-source-markdown:sha256:<prefix>",
  "record_refs": ["uiai-source-markdown:sha256:<prefix>#record=1"],
  "chunk_refs": ["uiai-source-markdown:sha256:<prefix>#record=1"],
  "suggested_uses": ["content source", "competitor proof"],
  "captured_at": "2026-06-05T00:00:00Z",
  "engine_schema": "uiai.source_markdown.v1",
  "engine_version": "optional build/version if present"
}
```

## Admin display contract

The plugin UI should show:

- title/source URL/source type/adapter,
- bounded excerpt, not full Markdown by default,
- evidence ref with copy action,
- record/chunk count and expandable refs,
- suggested uses,
- capture time,
- degraded/blocked/truncated indicators from metadata.

The UI should not show raw JSON unless an advanced/debug affordance is explicitly opened.

## Reuse contract

Saved cards may become inputs for:

- content/FAQ/SEO seed workflows,
- competitor/reference analysis,
- blueprint/content map generation,
- client/report summaries,
- Focusa evidence capture by an agent.

When reused, pass the saved `evidence_ref`, `source_url`, `source_type`, and bounded excerpt/record refs. Avoid passing raw full Markdown into unrelated prompts unless the user explicitly asks for full-source context.

## Error contract

On engine failure, preserve structured error fields when present:

- `error_id`
- `error_class`
- `message` / `error`
- `suggested_next_action`
- `diagnostics`

The plugin should display a concise message and keep raw response bodies out of normal UI.

## Security and privacy

- Remote UIAI calls must authenticate according to `docs/ENDPOINT_AUTH_MATRIX.md` and `docs/REMOTE_AUTH_EXAMPLES.md`.
- Never persist provider keys, bearer tokens, cookies, auth headers, raw request bodies, or raw diagnostics.
- Treat `focusa` metadata as evidence proposal metadata; Focusa durable capture happens only through Focusa tools.

## Acceptance checklist for plugin implementation

- [x] Capture form calls `/api/markdown` and validates top-level + `wpuiai` schemas.
- [x] Saved card record uses `wpui.saved_research_card.v1` as private `wpuiai_research` post metadata.
- [x] Admin capture page and `wpuiai_research` private post list/detail expose bounded saved card fields.
- [x] Structured engine failures are reduced to concise admin messages without raw body persistence.
- [x] Plugin test `tests/unit/test-source-markdown-research-cards.php` covers successful save shape, bounded/truncated excerpt, record/chunk refs, and missing schema rejection.
- [x] Proof cites UIAI `scripts/smoke-source-markdown-e2e.sh` plus plugin-side `php tests/unit/test-source-markdown-research-cards.php` and WP-CLI load smoke.
