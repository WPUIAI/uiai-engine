# UIAI Source-to-Markdown Agent Spec

**Status:** draft product/engineering spec  
**Date:** 2026-06-05  
**Project:** UIAI Engine  
**HLT alignment:** make UIAI Engine the agent-compatible browser/intelligence platform for Pi, MCP, Focusa, WPUIAI, portability, safety, and public documentation.  
**Core idea:** agents should be able to retrieve public webpages, social threads, docs, issues, videos, and other public sources as clean, bounded, cited, agent-ready Markdown or JSONL without needing to know UIAI internals.

## 1. Thesis

UIAI should expose **Source-to-Markdown** as an agent-first evidence surface:

> Turn a public source URL into clean Markdown, structured JSONL, source metadata, diagnostics, and stable evidence handles.

This is not just a clone of webpage-to-Markdown tools. UIAI can combine:

1. **Browser-rendered capture** from persistent sessions.
2. **Source-specific adapters** for networks like X, Reddit, GitHub, Hacker News, YouTube, and public webpages.
3. **Agent discovery** through `/api/tools`, `/api/tools/search`, `/api/tools/graph`, Pi, MCP, CLI, and docs.
4. **Focusa evidence** via bounded summaries and stable refs.
5. **WPUIAI product surface** via research cards, proof reports, and content/intelligence workflows.

The winning shape is not “scrape everything.” It is **permission-safe public source capture with clean failure reasons, proof, and agent-readable outputs**.

## 2. Source-backed references

Observed reference behaviors:

- `supermemoryai/markdowner`: converts any website into LLM-ready Markdown; supports detailed response, crawl subpages, LLM filtering, text/JSON responses, and self-hosting.
- `xmdbot.com`: converts X/Twitter articles, tweets, and threads to Markdown; markets `.md`, `.jsonl`, raw output, public gallery/archive links, API/CLI/MCP positioning, and reply-bot workflow.

UIAI should borrow the useful product concepts while staying broader and more agent-operable:

- Generic webpage Markdown.
- Source adapters beyond X.
- JSONL for RAG/agent memory.
- Diagnostics when blocked or degraded.
- Stable evidence refs and Focusa handoff.
- Pi/MCP/CLI/API parity.

## 3. Product name candidates

| Name | Fit |
|---|---|
| **Source-to-Markdown** | Plain engineering name. |
| **UIAI MD** | Short product/tool name. |
| **Source-to-Memory** | Stronger when paired with Focusa and RAG. |
| **Web Memory Capture** | Marketable for agents/knowledge workflows. |
| **Page-to-Proof Markdown** | Best when evidence/reporting is emphasized. |

Recommended internal name: **Source-to-Markdown**.  
Recommended product phrase: **Web Memory Capture**.

## 4. User value

Agents and operators need source content in a predictable format:

- Claude/ChatGPT prompt context.
- Pi session research.
- Focusa evidence and Workpoint memory.
- WPUIAI research cards and client proof reports.
- RAG/vector ingestion.
- Obsidian/Notion/dev docs.
- Archive-before-deletion workflows.
- Competitor/content analysis.

Value proposition:

> One URL in. Clean Markdown, JSONL, metadata, diagnostics, and evidence out.

## 5. Non-goals and boundaries

### Non-goals for MVP

- No credential scraping.
- No bypassing paywalls or private content boundaries.
- No automatic form submission.
- No raw cookie/header/token capture in output.
- No unbounded crawl.
- No LLM filtering by default.
- No guarantee that bot-blocked networks will always work.

### Required boundary language

- Public-only by default.
- Authenticated capture must use explicit user-provided browser/session/auth profile later, with revocable scope.
- Blocked sources return structured diagnostics and next options, not invented content.
- Network-specific adapters are best-effort unless the site provides a public API or reliable public markup.

## 6. API design

### 6.1 Existing surface extension: browser read Markdown

Extend current session read endpoint:

```http
POST /api/session/{id}/read
```

Request:

```json
{
  "format": "markdown",
  "selector": "main",
  "mode": "main_content",
  "max_chars": 30000,
  "include_links": true,
  "include_images": false,
  "include_metadata": true
}
```

Response keeps existing compatibility by preserving `text`:

```json
{
  "schema": "uiai.browser_read.v2",
  "url": "https://example.com/article",
  "title": "Example Article",
  "description": "...",
  "selector": "main",
  "format": "markdown",
  "mode": "main_content",
  "text": "# Example Article\n\nClean Markdown...",
  "chars": 12000,
  "truncated": false,
  "headings": [],
  "links": [],
  "metadata": {
    "source_type": "webpage",
    "canonical_url": "https://example.com/article",
    "site_name": "Example",
    "captured_at": "2026-06-05T00:00:00Z"
  },
  "focusa": {
    "target_ref": "browser:https://example.com/article",
    "evidence_ref": "uiai-browser:session=abc:read:2",
    "preferred_tool": "focusa_evidence_capture",
    "summary": "Read Markdown from Example Article",
    "next_tools": ["focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"],
    "focusa_scope_status": "present"
  }
}
```

### 6.2 New one-shot endpoint

```http
POST /api/markdown
```

Request:

```json
{
  "url": "https://example.com/article",
  "source": "auto",
  "format": "markdown",
  "mode": "main_content",
  "max_chars": 30000,
  "include_metadata": true,
  "include_links": true,
  "include_images": false,
  "include_diagnostics": true,
  "archive": false,
  "focusa_scope": {
    "project_root": "/home/wpuiai/uiai-engine",
    "continuity_id": "focusa-cont-example"
  }
}
```

Response:

```json
{
  "schema": "uiai.source_markdown.v1",
  "ok": true,
  "source": {
    "type": "webpage",
    "adapter": "webpage_browser",
    "url": "https://example.com/article",
    "canonical_url": "https://example.com/article",
    "title": "Example Article",
    "author": null,
    "published_at": null
  },
  "format": "markdown",
  "markdown": "# Example Article\n\n...",
  "jsonl": null,
  "stats": {
    "chars": 12000,
    "tokens_estimate": 3000,
    "links": 12,
    "images": 0,
    "truncated": false,
    "duration_ms": 450
  },
  "evidence": {
    "evidence_ref": "uiai-md:sha256:abcdef123456",
    "capture_refs": ["uiai-browser:session=abc:read:1"],
    "diagnostics_ref": "uiai-diagnostics:session=abc:seq=4"
  },
  "diagnostics_summary": {
    "console_errors": 0,
    "failed_requests": 0,
    "blocked": false,
    "top_findings": []
  },
  "focusa": {
    "target_ref": "source-md:https://example.com/article",
    "evidence_ref": "uiai-md:sha256:abcdef123456",
    "preferred_tool": "focusa_evidence_capture",
    "summary": "Converted Example Article to Markdown; 3000 estimated tokens; diagnostics clean.",
    "next_tools": ["focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"]
  },
  "cleanup": {
    "session_closed": true
  }
}
```

### 6.3 Plain text response option

For simple clients:

```http
GET /api/markdown?url=https://example.com&format=markdown
Accept: text/plain
```

Returns only Markdown when safe. JSON remains default for agents because it carries metadata/evidence.

## 7. Source adapter architecture

### 7.1 Adapter interface

Conceptual Go interface:

```go
type SourceMarkdownAdapter interface {
    ID() string
    Match(url string) MatchResult
    Capture(ctx context.Context, req SourceMarkdownRequest, browser BrowserRunner) (*SourceMarkdownResult, error)
}
```

Adapter result must include:

- Source type.
- Canonical URL.
- Markdown.
- Optional JSONL records.
- Metadata.
- Diagnostics/failure class.
- Evidence refs.

### 7.2 Adapter selection

`source=auto` chooses by URL:

| URL/source | Adapter | Priority |
|---|---|---:|
| `x.com`, `twitter.com` | `x_public` | 90 |
| `reddit.com`, `old.reddit.com` | `reddit_public` | 80 |
| `github.com/.../issues`, `/pull`, `/discussions` | `github_public` | 80 |
| `news.ycombinator.com/item` | `hackernews_public` | 75 |
| `youtube.com/watch`, `youtu.be` | `youtube_public` | 70 |
| public Facebook post URL | `facebook_public_best_effort` | 50 |
| LinkedIn public post/article URL | `linkedin_public_best_effort` | 45 |
| everything else | `webpage_browser` | 10 |

### 7.3 Adapter output formats

Each adapter should support:

- `markdown`: one Markdown document.
- `json`: structured object.
- `jsonl`: one record per post/comment/tweet/message/section.
- `raw_text`: bounded plain text fallback.

## 8. Initial adapter roadmap

### 8.1 MVP adapters

#### `webpage_browser`

- Browser-render page.
- Select `article`, `main`, `[role=main]`, or body fallback.
- Strip scripts/styles/nav/footer/header/iframes by default.
- Convert cleaned HTML to Markdown.
- Preserve headings, links, image alt text optionally.
- Works with current session infrastructure.

#### `github_public`

Targets:

- Issues.
- Pull requests.
- Discussions.
- README/blob pages later.

Output:

```md
---
source: github_issue
repo: owner/repo
issue: 123
state: open
evidence_ref: uiai-md:sha256:...
---

# Issue title

## Description
...

## Comments

### user · 2026-06-05
...
```

Why early: reliable public structure, high agent/dev value.

#### `reddit_public`

Targets:

- Public posts and comment threads.

Output:

```md
---
source: reddit
subreddit: r/example
author: u/example
comments: 42
evidence_ref: uiai-md:sha256:...
---

# Post title

Post body...

## Top comments

### u/commenter · 238 points
Comment text...
```

Why early: strong market/research value and public threads are structured.

### 8.2 V2 adapters

#### `x_public`

Targets:

- Public tweet.
- Thread.
- Article.

Output:

```md
---
source: x_thread
author: "@user"
url: https://x.com/user/status/...
tweets: 12
evidence_ref: uiai-md:sha256:...
---

# X Thread by @user

## Tweet 1
...

## Tweet 2
...
```

Boundary: X blocking is common. If blocked, return `blocked=true`, failure class, diagnostics, and next options.

#### `youtube_public`

Targets:

- Video metadata.
- Transcript when publicly available.
- Chapters.
- Description links.

Output:

```md
---
source: youtube
channel: Example Channel
video_id: abc123
transcript_available: true
---

# Video title

## Description
...

## Transcript
...
```

#### `hackernews_public`

Targets:

- Public HN item threads.

Why: easy, useful for agents, public HTML.

### 8.3 Best-effort adapters

#### `facebook_public_best_effort`

- Public posts only.
- No login-gated or private groups.
- Fail cleanly if blocked.
- No credential scraping.

#### `linkedin_public_best_effort`

- Public posts/articles only.
- Fail cleanly if blocked.

## 9. Markdown quality contract

Markdown should be:

- LLM-ready.
- Stable across runs where content unchanged.
- Bounded by `max_chars` / `max_tokens_estimate`.
- Cited to source URL.
- Metadata-rich enough for RAG.
- Free of raw nav/sidebar/cookie noise where possible.
- Safe: no secrets, cookies, auth headers, or raw hidden fields.

Required front matter when `include_metadata=true`:

```yaml
---
source_type: webpage|x_thread|reddit_post|github_issue|youtube_video|...
source_url: https://...
canonical_url: https://...
title: ...
author: ...
published_at: ...
captured_at: ...
evidence_ref: uiai-md:sha256:...
tokens_estimate: 1234
truncated: false
---
```

## 10. JSONL contract

Use JSONL for RAG/agent memory:

```jsonl
{"schema":"uiai.source_markdown_record.v1","source_type":"x_thread","record_type":"post","index":1,"author":"@user","text":"...","url":"...","evidence_ref":"uiai-md:sha256:...#1"}
{"schema":"uiai.source_markdown_record.v1","source_type":"x_thread","record_type":"post","index":2,"author":"@user","text":"...","url":"...","evidence_ref":"uiai-md:sha256:...#2"}
```

Record fields:

| Field | Required | Notes |
|---|---:|---|
| `schema` | yes | `uiai.source_markdown_record.v1` |
| `source_type` | yes | adapter source type |
| `record_type` | yes | post/comment/tweet/section/transcript_segment |
| `index` | yes | stable order |
| `author` | no | public author handle/name |
| `text` | yes | bounded text |
| `url` | yes | source or canonical sub-URL |
| `created_at` | no | source timestamp if public |
| `evidence_ref` | yes | record-specific or parent evidence |
| `parent_index` | no | comments/replies |

## 11. Agent discovery requirements

This feature must be discoverable through every agent surface.

### 11.1 HTTP tool discovery

Add tool definitions to:

- `GET /api/tools`
- `GET /api/tools/openai`
- `GET /api/tools/mcp`
- `GET /api/tools/graph`
- `GET /api/tools/search?q=markdown`
- `GET /api/tools/agent-card`
- `GET /api/tools/docs`

Tool names:

| Surface | Tool |
|---|---|
| HTTP | `/api/markdown`, `/api/session/{id}/read format=markdown` |
| Pi | `uiai_source_to_markdown`, `uiai_browser_read format=markdown` |
| MCP | `source_to_markdown`, `browser_read format=markdown` |
| CLI | `scripts/uiai md <url>`, `scripts/uiai session read <sid> --format markdown` |
| WPUIAI | Research card/report action later |

### 11.2 Tool search keywords

`/api/tools/search` should match:

- markdown
- md
- source to markdown
- webpage markdown
- twitter markdown
- x to md
- reddit markdown
- github issue markdown
- youtube transcript
- rag
- jsonl
- archive
- memory capture
- source capture

### 11.3 Tool graph relations

Add graph edges:

```text
source_to_markdown -> focusa_evidence_capture
source_to_markdown -> uiai_focusa_packet_compose
source_to_markdown -> browser_diagnostics
source_to_markdown -> browser_open
source_to_markdown -> browser_read
source_to_markdown -> browser_close
source_to_markdown -> focusa_active_object_resolve
source_to_markdown -> focusa_predict_record
```

## 12. Pi extension design

### 12.1 Tools

Add Pi tool:

```ts
uiai_source_to_markdown({
  url: string,
  source?: "auto" | "webpage" | "x" | "reddit" | "github" | "youtube" | "hackernews" | "facebook_public" | "linkedin_public",
  format?: "markdown" | "json" | "jsonl",
  max_chars?: number,
  include_metadata?: boolean,
  include_diagnostics?: boolean,
  focusa_scope?: object
})
```

Extend `uiai_browser_read` params:

```ts
format?: "text" | "markdown"
mode?: "main_content" | "full" | "selector"
include_images?: boolean
```

### 12.2 Commands

Add command variants:

```text
/uiai md <url>
/uiai md --jsonl <url>
/uiai md --source reddit <url>
/uiai research <query> --md
/uiai proof <url> --md
```

Compact render:

```text
UIAI MD source=reddit chars=8421 tokens≈2105 evidence=uiai-md:sha256:abc next="capture Focusa evidence"
```

Expanded JSON remains available.

## 13. MCP design

Add MCP tool:

```json
{
  "name": "source_to_markdown",
  "description": "Convert a public URL/source into clean agent-ready Markdown or JSONL with metadata, diagnostics, and evidence refs.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "url": { "type": "string" },
      "source": { "type": "string", "default": "auto" },
      "format": { "type": "string", "default": "markdown" },
      "max_chars": { "type": "integer", "default": 30000 },
      "include_metadata": { "type": "boolean", "default": true },
      "include_diagnostics": { "type": "boolean", "default": true },
      "focusa_scope": { "type": "object" }
    },
    "required": ["url"]
  }
}
```

MCP bridge should also pass `format` through to `browser_read`.

## 14. CLI design

Add:

```bash
scripts/uiai md https://example.com
scripts/uiai md https://x.com/user/status/123 --source x --format jsonl
scripts/uiai md https://reddit.com/r/foo/comments/bar --max-chars 50000 --out /tmp/source.md
scripts/uiai session read SID --format markdown --selector main
```

Exit codes:

| Code | Meaning |
|---:|---|
| 0 | success |
| 2 | invalid args |
| 4 | source blocked/auth required |
| 5 | capture failed |
| 6 | adapter unsupported |

## 15. Focusa integration

Every successful Source-to-Markdown result should include:

```json
{
  "focusa": {
    "target_ref": "source-md:<sanitized-url>",
    "evidence_ref": "uiai-md:sha256:<prefix>",
    "preferred_tool": "focusa_evidence_capture",
    "summary": "Converted <source> to Markdown; <tokens> estimated tokens; <diagnostics status>.",
    "next_tools": ["focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"],
    "focusa_scope_status": "present|partial|missing"
  }
}
```

Failure result should recommend:

- `browser_diagnostics` if session exists.
- `focusa_browser_diagnostics_intake` for diagnostics/failure envelopes.
- `focusa_evidence_capture` only if there is useful bounded proof.

## 16. WPUIAI integration ideas

WPUIAI should not expose raw internals. It can package Source-to-Markdown as product workflows:

### Research card

Implemented engine response contract:

```json
{
  "wpuiai": {
    "research_card": {
      "schema": "wpui.source_markdown_research_card.v1",
      "source_url": "https://example.com/source",
      "source_type": "webpage|github_issue|reddit_thread|x_thread|...",
      "title": "Source title",
      "markdown_excerpt": "bounded excerpt",
      "evidence_ref": "uiai-source-markdown:sha256:<prefix>",
      "captured_at": "2026-06-05T00:00:00Z",
      "suggested_uses": ["content source", "competitor proof", "reference", "FAQ seed", "SEO seed"],
      "metadata": {}
    },
    "report": {
      "schema": "wpui.source_markdown_report.v1",
      "summary": "Captured source as Markdown for WPUIAI research/card workflows.",
      "evidence_ref": "uiai-source-markdown:sha256:<prefix>"
    }
  }
}
```

WordPress admin workflow:

```text
Paste URL -> Capture Markdown -> Save Research Card
```

Card fields:

- Source URL.
- Source type.
- Markdown excerpt.
- JSONL/record artifact refs.
- Evidence ref.
- Captured at.
- Suggested WPUIAI use: content source, competitor proof, reference, FAQ seed, SEO seed.

### Blueprint input

Allow Source-to-Markdown output as a content source for:

- Google Drive alternative.
- Competitor/reference analysis.
- Client interview import.
- Article-to-landing-page conversion.

### Client report

Show:

- “Sources captured.”
- “What was used.”
- “Evidence refs.”
- “Content transformed into page sections.”

## 17. Safety and compliance

Required protections:

- URL allow/deny rules reused from browser/session auth policy.
- Remote callers authenticated unless route is explicitly public-safe.
- SSRF protections.
- Max pages and max bytes.
- Max duration.
- Robots/legal posture documented: public capture only, caller responsible for rights; UIAI should avoid evasion claims.
- No storing private content by default.
- Redact auth tokens, query secrets, cookies, emails where configured.
- No output of raw hidden form values.

## 18. Caching and archive

MVP:

- No public archive by default.
- Optional short-lived cache keyed by URL + adapter + format + mode.
- Evidence hash derived from normalized Markdown + canonical URL + timestamp or stable content hash.

V2:

- Optional `archive=true` creates a share artifact.
- Share artifacts should be explicit and authenticated/permissioned for non-public deployments.
- Public gallery is not default; avoid accidental content publication.

## 19. Failure classes

Structured failures:

| Class | Meaning | Recommended next |
|---|---|---|
| `source_blocked` | Bot/login/geofence blocked | Use browser diagnostics; try authenticated session if allowed. |
| `unsupported_source` | No adapter and generic failed | Use generic webpage mode or add adapter. |
| `selector_not_found` | Requested selector missing | Use browser snapshot/read without selector. |
| `content_empty` | Page loaded but no usable text | Diagnostics, screenshot, or alternate source. |
| `network_timeout` | Timed out | Retry bounded; check diagnostics. |
| `rate_limited` | Source/provider rate limit | Backoff/cache. |
| `auth_required` | Source requires login | Explicit scoped auth profile needed. |
| `too_large` | Output exceeded caps | Lower crawl/max pages or use JSONL chunks. |

## 20. Acceptance criteria

### MVP acceptance

- `/api/session/{id}/read` accepts `format=markdown` and returns Markdown in `text` plus `format=markdown`.
- `/api/markdown` converts a normal webpage into Markdown using one-shot browser open/read/close.
- Pi tool `uiai_source_to_markdown` exists.
- MCP tool `source_to_markdown` exists.
- CLI `scripts/uiai md <url>` exists.
- `/api/tools/search?q=markdown` returns Source-to-Markdown tools.
- `/api/tools/graph` includes Source-to-Markdown route relations.
- Focusa metadata includes `target_ref`, `evidence_ref`, `summary`, and next tools.
- Tests prove no raw cookies/scripts/hidden inputs in Markdown output.
- Failure smoke returns structured `source_blocked` or `content_empty` instead of fake content.

### Adapter acceptance

For each adapter:

- Public fixture URL test or local fixture.
- Markdown snapshot test.
- JSONL structure test when supported.
- Bounds/truncation test.
- Diagnostics/failure test.
- Tool discovery keyword test.

## 21. Implementation sequence

### Phase 0 — Spec and discovery

- This spec.
- Add README/doc links.
- Add beads for MVP and adapters.

### Phase 1 — Generic Markdown read

- Add `format` to `ReadOptions`.
- Convert cleaned HTML to Markdown.
- Preserve existing text mode behavior.
- Add tests.

### Phase 2 — One-shot `/api/markdown`

- Open browser session.
- Run read format markdown.
- Capture diagnostics summary if requested.
- Close session.
- Return evidence refs.

### Phase 3 — Agent surfaces

- Pi tool + command.
- MCP tool.
- CLI command.
- Tool graph/search/agent-card/docs updates.
- Focusa packet capture compatibility.

### Phase 4 — Adapters

Recommended order:

1. GitHub public issue/PR/discussion.
2. Reddit public post/thread.
3. Hacker News item.
4. X public thread/article best-effort.
5. YouTube transcript/metadata.
6. Facebook/LinkedIn public best-effort.

### Phase 5 — WPUIAI productization

- Implemented engine-side `wpuiai.research_card` and `wpuiai.report` response objects for `/api/markdown`.
- WordPress plugin UI/save integration remains a plugin-side follow-up.
- Blueprint content-source import.
- Client proof report integration.

## 22. Beads candidates

Create implementation beads after spec acceptance:

1. `Source-to-Markdown MVP: browser_read format=markdown`
2. `Source-to-Markdown one-shot /api/markdown endpoint`
3. `Source-to-Markdown Pi/MCP/CLI discovery surfaces`
4. `Source-to-Markdown Focusa evidence metadata and packet examples`
5. `GitHub public adapter`
6. `Reddit public adapter`
7. `X public adapter best-effort`
8. `WPUIAI research card integration`

## 23. Open questions

1. Should `/api/markdown` be loopback-public remote-auth like browser/search, or authenticated even on loopback for non-webpage adapters?
2. Should UIAI store Markdown artifacts locally, or only return them unless `archive=true`?
3. Should adapter output include screenshots by default, or only evidence handles and optional proof mode?
4. Which network has highest market priority after generic webpage: Reddit, X, GitHub, or YouTube?
5. Should LLM filtering be UIAI Engine native, or delegated to WPUIAI/AI Cloud provider routes?

## 24. Product positioning

Short:

> **UIAI MD turns public web sources into agent-ready Markdown with proof.**

Long:

> **Source-to-Markdown gives agents a safe, discoverable way to capture webpages, threads, posts, issues, and videos as clean Markdown or JSONL, with metadata, diagnostics, evidence refs, and Focusa/WPUIAI handoff.**

Market angle:

> Not just scrape-to-Markdown. Capture-to-memory, with proof.
