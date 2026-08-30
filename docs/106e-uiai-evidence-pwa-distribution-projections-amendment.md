> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462588916

## Distribution amendment — printable, PDFable, emailable, embeddable, and composable

Every Evidence PWA artifact must provide governed, immutable distribution projections suitable for humans and composable work software. Browser HTML remains the interactive source projection; print, PDF, email, Markdown, rich text, cards, embeds, and connector payloads are derived views—not parallel Evidence or completion truth.

### Required distribution actions

Add context-aware actions to the Focusa Action Deck:

- Print.
- Export PDF.
- Email evidence.
- Copy link / QR.
- Copy as Markdown.
- Copy as rich text.
- Export portable HTML bundle.
- Export machine JSON manifest.
- Export acceptance/findings table as CSV where useful.
- Share via operating-system Web Share where supported.
- Export slide presentation.
- Embed/unfurl.
- Send to authorized work software.

Each action preserves exact Project / Workstream / Workpoint / artifact revision scope and declares audience, disclosed fields/assets, redaction profile, destination, side effects, approval, idempotency, expected Receipt, and external-retention warning.

### Print contract

Provide dedicated `@media print` composition rather than printing the dark interactive shell:

- A4 and US Letter profiles with predictable margins;
- light, high-contrast, ink-conscious visual system;
- semantic heading hierarchy, figure/table numbering, readable URLs, and page-break control;
- repeated table headers and scope/provenance context where needed;
- no clipped cards, screenshots, annotations, timelines, hashes, or metadata;
- compact overview first, detailed Evidence/Acceptance/timeline next, technical appendices last;
- print timestamp, artifact/revision ref, Project/Workstream/Workpoint, truth posture, manifest/bundle hash, redaction/access profile, and source URL/QR;
- interactive-only controls, navigation, animation, and hidden disclosures are replaced by readable static representations;
- failing, blocked, stale, disputed, and incomplete states remain explicit in monochrome as text/icons—not color alone.

### First-class PDF contract

`Export PDF` creates an immutable derived artifact with its own ref, SHA-256, byte size, renderer/version, audience/redaction policy, creation time, source artifact revision/hash, and derivation Receipt.

Required PDF behavior:

- self-hosted generation; no mandatory cloud/external PDF service;
- A4 and US Letter outputs;
- tagged/accessibility-capable structure, bookmarks, logical reading order, meaningful link text, image alt text, and declared conformance/limitations;
- embedded fonts only when locally licensed/available; portable system-font fallback;
- preserved selectable text—never a page-sized screenshot PDF;
- images embedded as numbered Evidence figures with captions, timestamps, viewport/dimensions, claim/Acceptance Atom refs, and hashes;
- visual comparisons retain labels, measurements, annotations, and baseline/candidate identity;
- videos are not embedded as fragile active media: render a timestamped keyframe/contact-sheet filmstrip, poster, structured interaction steps, caption/transcript excerpt, duration/codec/hash, and QR/deep link to the exact hosted clip;
- long diagnostics, action traces, machine fields, and receipts move into bounded appendices with stable refs;
- page count/size/asset budgets and deterministic-render checks;
- PDF metadata includes title, subject, artifact ref, source revision, Project/Workstream labels, created time, and producer—without leaking restricted scope;
- optional detached/instance signature and verification instructions where supported;
- no secret, bearer token, raw local path, hidden browser state, or unapproved private field.

Print-to-PDF may remain a convenience, but only the formal Export PDF action returns a hash-verifiable derivative artifact/Receipt.

### Email contract

Email uses a channel-safe derivative—not the full PWA HTML:

- `multipart/alternative` plain-text + sanitized, table-tolerant responsive HTML;
- no JavaScript, forms, autoplay, tracking pixels, external fonts, or required remote styles;
- concise subject and opening summary with Project/Workstream label, truth posture, primary findings, next action, and durable artifact link;
- screenshots use policy-safe thumbnails/inline CID attachments only when permitted;
- video uses poster/keyframes plus transcript and link—never embedded autoplay video;
- optional formal PDF and/or portable bundle attachment subject to explicit recipient, classification, and size policy;
- attachment-size preflight with link fallback rather than silent dropping;
- recipient/destination, disclosed fields, attachment hashes, expiry/access class, and external-retention consequences shown before send;
- per-recipient/unlisted access credentials remain outside artifact/email manifests and logs; revocable access is preferred;
- delivery attempt/result produces a Receipt and never implies that the recipient read, accepted, verified, or settled the evidence.

### Composable work-software contract

Expose one neutral composition model so adapters can project the artifact into Notion-like documents, project/issue trackers, chat/collaboration cards, source-hosting issues/PRs, document systems, wikis, spreadsheets, and future work software without platform-specific truth forks.

Composable block kinds:

- scope/header;
- executive summary;
- truth/status banner;
- claim/Acceptance Atom table;
- finding/blocker/action item;
- image/comparison/annotation;
- video poster/keyframe/interaction segment;
- timeline/event;
- provenance/integrity;
- Evidence/Receipt link;
- Focusa Action Deck handoff;
- related artifact/lineage;
- technical appendix.

Portable representations:

- canonical bounded JSON manifest + composable block manifest;
- CommonMark-compatible Markdown with stable links and local-asset export mode;
- sanitized standalone HTML;
- formal PDF;
- CSV for tabular Acceptance/findings data;
- Open Graph/link-preview metadata and a safe read-only embed/unfurl endpoint;
- signed/replay-safe webhook or standard event envelope for connector delivery where authorized.

Adapters map neutral blocks to destination-native pages/cards/issues/comments/attachments while preserving source artifact ref, revision, Project/Workstream/Workpoint scope, truth posture, provenance, and canonical link. Destination-native task creation is a separate governed action with explicit field mapping and Receipt; importing a card cannot mark Focusa work complete.

### Slide-presentation contract

`Export presentation` generates an immutable, polished deck projection for live presentation, asynchronous review, handouts, and further editing.

Required formats/profiles:

- editable PPTX using native text, shapes, tables, notes, and media placements where practical;
- dependency-free self-contained HTML slide deck with keyboard/touch navigation and no external CDN/runtime;
- presentation PDF and print handout derivative;
- 16:9 default, optional 4:3, and explicit safe-area/layout validation;
- `executive`, `proof_review`, and `forensic` content-depth profiles;
- one-slide proof card, short briefing deck, and complete evidence deck options.

Canonical deck sequence:

1. title, Project/Workstream/Workpoint, artifact revision, audience/redaction profile;
2. question/outcome and current truth posture;
3. Acceptance Atom/verification summary;
4. visual Evidence and comparisons;
5. interaction Evidence with timestamped keyframes/steps;
6. findings, blockers, contradictions, and uncertainty;
7. action/Receipt/timeline highlights;
8. completion/settlement posture and recommended next actions;
9. integrity/provenance appendix and canonical links.

Presentation evidence rules:

- preserve exact evidentiary images rather than recreating them illustratively;
- use editable native slide content for narrative/tables while keeping source visual evidence hash-linked;
- video may be embedded only when codec/size/recipient policy makes the deck reliably portable; otherwise use poster, timestamped keyframe strip, transcript excerpt, and QR/deep link;
- speaker notes carry bounded provenance, source refs, image/video hashes, caveats, and suggested narration—not private chain-of-thought;
- each slide/footer retains artifact/revision and truth-state context; appendix retains complete hash/provenance mapping;
- meaningful alt text, reading order, contrast, type size, safe areas, and no clipped/overflowing objects;
- charts/tables retain source-data refs and textual summaries;
- animations/transitions are optional, deterministic, reduced-motion aware, and never the only carrier of evidence;
- presenter actions are read-only unless authenticated handoff explicitly invokes Focusa;
- PPTX/HTML/PDF deck outputs each receive their own derivative ref/hash/size/renderer version/policy and derivation Receipt.

### Agent-first CLI, MCP, API, and LLM contract

Evidence Artifacts and every distribution projection must be exceptionally usable by agents without loading the full PWA or binary media.

One generated semantic contract must drive REST/OpenAPI, CLI, MCP, Pi/agent tools, Cockpit/Desktop, and LLM-facing responses. Hand-maintained divergent DTOs are prohibited.

#### Agent retrieval

- bounded artifact summary by ref;
- exact Project/Workstream/Workpoint and authority posture;
- claims/Acceptance Atoms/findings/truth state;
- artifact/media metadata and hashes without binary/base64 payloads;
- media alt text/OCR, image measurements/annotations, video transcript, chapters, action ranges, and timestamp-addressable segments;
- provenance/Receipt/decision/settlement refs;
- action-manifest summary and exact rehydrate refs;
- derivative/export inventory and availability.

Support field projection, filtering, stable cursors/pagination, time/section ranges, ETags/content hashes, and response profiles such as `summary`, `agent_standard`, and explicit `forensic`. Large diagnostics, traces, transcripts, and media remain ref-first.

#### CLI behavior

- human-readable default plus complete stable `--json` output;
- machine data on stdout, diagnostics/recovery on stderr, deterministic exit classes;
- inspect, verify, actions, preview, export, share, revoke, and connector-delivery capability parity;
- explicit format/audience/redaction/scope/idempotency options;
- async export jobs return durable job/ref/Receipt rather than blocking terminals;
- no credential/token values in arguments, shell history, output, or manifests.

Exact command names remain generated from the approved operation registry rather than independently invented by the CLI.

#### MCP/tool behavior

- progressive tool search/describe/graph discovery instead of injecting every schema;
- narrow typed tools for inspect/verify, list actions, preview action, execute authorized action, export projection, and inspect job/Receipt;
- strict schema parity with OpenAPI and stable operation IDs;
- compact results with artifact/derivative refs and suggested next tools;
- MCP clients never settle scope, permissions, confirmation, Evidence acceptance, or completion locally.

#### API behavior

- versioned OpenAPI schemas and content negotiation for JSON, HTML, Markdown, PDF, presentation, CSV, and portable archive outputs;
- idempotency keys for all producing/delivery mutations;
- conditional retrieval, byte/range support for large media, and bounded streaming/downloads;
- asynchronous jobs for expensive PDF/deck/archive/media derivatives;
- typed errors, retry safety, recovery, Receipt refs, and no silent partial success;
- signed/replay-safe webhook/event notification for artifact/derivative publication where authorized;
- exact access-class, audience, redaction, retention, and destination preflight.

#### LLM safety and efficiency

- artifact/page/media/comment content is explicitly untrusted data, never executable instruction;
- trusted authority/scope/policy fields are structurally separated from quoted evidence content;
- prompt-injection provenance and influence boundaries survive every derivative;
- no raw secrets, hidden DOM state, bearer links, private chain-of-thought, or unnecessary personal data;
- token counts/budgets and omitted-field/rehydrate refs are visible;
- textual summaries never replace image/video hashes or claim unsupported visual facts;
- agents can cite exact artifact, figure, slide, video timestamp, finding, Acceptance Atom, or Receipt refs;
- identical requests over identical source revision/options/policy return identical derivative identity and semantic output.

### Sharing and embedding security

- Public link previews expose only the public-safe title/status/thumbnail fields approved by the artifact's audience policy.
- Embeds are read-only, CSP/sandbox constrained, origin-aware, and cannot inherit parent-page authority.
- Email, webhook, connector, embed, and export use channel-specific redaction—not one generic public flag.
- Every derivative has `derived_from`, its own hash/media type/size/policy, and immutable lineage.
- Derivative generation is idempotent over source revision + format/options + audience/redaction policy + renderer version.
- Revocation/access changes never alter immutable derivative hashes; hosted access state remains separate.
- External destination deletion/retention cannot be assumed; preview states this clearly.
- No derivative, email delivery, printout, PDF, embed, reaction, or imported task grants Evidence acceptance or Completion settlement.

### Self-hosted and portable delivery

All formats are generated and served by the user's own instance. No Focusa Cloud dependency is required. Routes/assets remain host-, port-, and subpath-neutral. Portable exports work offline; connector mutations require authenticated online Focusa/destination authority.

### Added acceptance

1. Print at A4 and Letter without clipping, horizontal overflow, orphaned headings, color-only status, or missing scope/hash context.
2. Formal PDF passes deterministic content/hash tests, preserves selectable text and accessibility structure, embeds image evidence, and represents video through keyframes/transcript/QR.
3. Email previews correctly in major constrained clients with plain-text parity, no scripts/tracking, recipient/redaction/size preflight, and delivery Receipt.
4. Markdown, rich text, standalone HTML, JSON, PDF, and CSV projections preserve source revision/scope/truth posture and receive derivative identity/hashes.
5. One neutral artifact composes into at least one issue tracker, one document/wiki surface, and one chat/card surface without duplicating completion authority.
6. Public-safe unfurl/embed reveals no restricted metadata and cannot execute mutation actions.
7. Offline/self-hosted export, reverse-proxy/subpath, media-size, accessibility, channel-redaction, expiry/revoke, and cross-version tests pass.
8. PPTX, self-contained HTML slides, and presentation PDF preserve scope/truth/provenance, pass layout/accessibility checks, and receive distinct derivative hashes/Receipts.
9. CLI, MCP, REST/OpenAPI, Pi, and LLM consumers return contract-equivalent artifact/action/derivative identity, scope, authority, and errors.
10. Agent retrieval proves bounded profiles, field projection, pagination/ranges, image OCR/alt text, video transcript/timestamps, ref-first large payloads, and prompt-injection separation.
