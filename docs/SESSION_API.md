# Browser Session API — LLM Vision Tool

Persistent browser sessions let agents open a page once, read/snapshot it, act through selectors and `@ref`s, capture screenshots, inspect diagnostics, and close the session without repeated navigation.

> **Mandatory entitlement warning:** Current code still allows selected unauthenticated loopback session/search/screenshot paths. The examples below describe API shape for authorized development/migration only. They do not grant Evaluation or approve customer exposure. Final production use requires caller authentication plus an authority-issued signed `uiai-engine` lease or a narrower verified child token, the required feature, node/time/sequence validation, and any limit reservation before browser allocation.

## Required request authority

The target request pipeline is:

```text
request safety/rate limit
→ caller authentication
→ signed lease or child-token verification
→ uiai-engine product grant
→ route feature grant
→ node/time/sequence/offline checks
→ concurrency/usage reservation
→ session handler or protected worker
→ usage commit/release
→ redacted observability
```

Loopback, `UIAI_LOCAL_API_TOKEN`, extension/Pi/MCP/Cockpit tokens, Focusa pairing, source checkout, or browser health never create product permission.

Representative feature keys:

- `uiai.session.execute`
- `uiai.screenshot.capture`
- `uiai.search.execute`
- `uiai.markdown.capture`
- `uiai.browser.diagnostics`
- `uiai.share.create`
- `uiai.fpv.control`

Exact grants/limits come from the signed authority policy.

## Session lifecycle

| Method | Route | Purpose | Target authority |
| --- | --- | --- | --- |
| `GET` | `/api/session` | list caller/lease-scoped sessions | authenticated + entitled; no cross-account enumeration |
| `POST` | `/api/session` | open/navigate a persistent session | `uiai.session.execute` + concurrency reservation |
| `GET` | `/api/session/{id}` | session metadata | owner/client/session scope |
| `DELETE` | `/api/session/{id}` | close session | owner/client/session scope; idempotent |
| `POST` | `/api/session/{id}/navigate` | navigate | session feature/scope |
| `GET` | `/api/session/{id}/read` | canonical bounded text/Markdown read | session feature/scope |
| `POST` | `/api/session/{id}/read` | supported compatibility transport; JSON ReadOptions | same authority and delivery rules as GET |
| `GET` | `/api/session/{id}/snapshot` | accessibility tree and `@ref`s | session feature/scope |
| `POST` | `/api/session/{id}/click` | click CSS selector or `@ref` | action scope, confirmation policy where required |
| `POST` | `/api/session/{id}/fill` | fill input | action scope |
| `POST` | `/api/session/{id}/type` | type text | action scope |
| `POST` | `/api/session/{id}/press` | keypress | action scope |
| `POST` | `/api/session/{id}/scroll` | scroll | action scope |
| `POST` | `/api/session/{id}/screenshot` | capture current page (artifact-ref default: metadata + artifact_url, no inline pixels) | screenshot feature/limit |
| `GET` | `/api/session/{id}/read` | compact page text (query: selector, max_chars (Unicode code points, not bytes), include_links, include_images, format, mode) | session feature/scope |
| `GET` | `/api/session/{id}/diagnostics` | console/exceptions/network/failures | diagnostics feature and redaction |
| `POST` | `/api/session/{id}/eval` | run bounded JavaScript | sensitive feature; may be excluded from Evaluation |
| `POST` | `/api/session/{id}/css` | inject CSS | sensitive action scope |
| `GET/POST` | session cookie/auth routes | save/load state (load accepts flat or origin-keyed storage; returns applied counts, fails loudly on unknown shapes) | explicit sensitive feature, encrypted storage, no logs |

GET `/read` uses query parameters; POST `/read` accepts the equivalent JSON
options and remains supported for existing Pi/MCP/CLI clients without a removal
schedule. Both are browser-read-only, capped at 30,000 characters, and use the
same redaction and evidence-delivery policy (including HTTP 202 when delivery is
not ready). Neither method grants permission to mutate a page. Read counters,
idle leases and evidence metadata may advance; these are operational bookkeeping,
not browser actuation. Invalid query option types or malformed JSON return 400.

The exact current route list remains in code/tool discovery and must be generated into the endpoint-feature coverage ledger before release.

## Authentication state restoration

`POST /api/session/{id}/auth/load` accepts the state returned by `/auth/save`.
Storage fields contain string values only. For example:

```json
{"url":"https://example.com/","cookies":[],"localStorage":{"probe":"example"},"sessionStorage":{}}
```

Origin-keyed storage is also accepted:

```json
{"url":"https://example.com/","localStorage":{"https://example.com/":{"probe":"example"},"https://other.example":{"unrelated":"not-applied"}}}
```

Only the target page's origin is restored; values from different websites are
never merged. An optional trailing slash on an origin is accepted. Invalid
storage shapes or non-string values return HTTP 400 with `invalid_auth_state`
and the affected field name, before cookies or navigation are changed.
Navigation, origin changes, and storage-write failures return errors rather than
`loaded`. A runtime failure may follow partial cookie/storage application; it is
not a transactional rollback. Success reports `cookies_applied`,
`localstorage_keys`, `sessionstorage_keys`, and the accepted storage shapes.
Treat saved state as sensitive; never log it or put real credentials in examples.

## Authorized-development example

This example assumes a synthetic/test trust root or a valid signed entitlement and an authentication token. It is not an anonymous Evaluation flow.

```bash
export UIAI_ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
export UIAI_BEARER_TOKEN="<short-lived-scoped-token>"

SID=$(curl -fsS -X POST "$UIAI_ENGINE_URL/api/session" \
  -H "Authorization: Bearer $UIAI_BEARER_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"url":"https://example.com","width":1280,"height":800}' \
  | jq -r '.session.id')

curl -fsS "$UIAI_ENGINE_URL/api/session/$SID/snapshot" \
  -H "Authorization: Bearer $UIAI_BEARER_TOKEN" \
  | jq '{tree,stats}'

curl -fsS -X POST "$UIAI_ENGINE_URL/api/session/$SID/click" \
  -H "Authorization: Bearer $UIAI_BEARER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector":"@e1"}'

curl -fsS "$UIAI_ENGINE_URL/api/session/$SID/diagnostics" \
  -H "Authorization: Bearer $UIAI_BEARER_TOKEN"

curl -fsS -X DELETE "$UIAI_ENGINE_URL/api/session/$SID" \
  -H "Authorization: Bearer $UIAI_BEARER_TOKEN"
```

Do not put raw commercial license keys in headers for normal runtime calls. Activation credentials exchange for a signed lease; Focusa/clients use scoped child/access tokens.

## Search and Source-to-Markdown

Current provider-neutral search and Markdown capture remain useful, but final production execution requires product/feature/limit checks even from loopback.

Provider readiness endpoints may remain public only if they expose bounded configuration status without keys, quotas, account identity, private queries, or cached customer results.

Search/Markdown results must continue to:

- strip URL fragments;
- redact secret-like query values;
- bound titles/snippets/records;
- expose stable evidence refs rather than raw transcript-sized payloads;
- scope cache/results to safe public inputs and caller/lease where needed.

## Diagnostics and Evidence

Diagnostics include bounded:

- console logs;
- JavaScript exceptions;
- network requests and failures;
- browser/session action failures;
- structured engine error ids/classes/next actions.

Never store or return authorization headers, cookies, request bodies, raw keys/tokens, secret query values, unmasked customer identity, or cross-account session data.

Preferred stable handles:

| Flow | Handle |
| --- | --- |
| diagnostics | `uiai-diagnostics:session=<id>:seq=<seq>` |
| engine error | `uiai-error:<error_id>` |
| search result | `uiai-search:<provider>:<query-hash>:<rank>` |
| Source-to-Markdown | `uiai-source-markdown:sha256:<prefix>` |
| browser read/snapshot | `uiai-browser:session=<id>:read|snapshot:<seq>` |
| screenshot/share | `uiai-screenshot:sha256:<prefix>` / `uiai-share:<id>` |

Focusa ingestion must preserve exact project/Workpoint scope and independently verify the UIAI product/child token. A Focusa Evidence link cannot retroactively authorize an unlicensed UIAI action.

Artifact inspection (`GET /api/screenshot/artifact/{ref}`) accepts a typed
`uiai-artifact:sha256:` reference or its bare digest. Screenshot packet refs
(`uiai-evidence-share:sha256:`) are also accepted; packet-id inspection at
`GET /api/screenshot/share/{id}` recognizes current-producer unsealed screenshots.
The native inspector accepts either typed reference without caller-side prefix
rewriting. An intact unsealed browser artifact or screenshot returns HTTP202 with `delivery_state: blocked` and a reconciliation hint,
not the retired raw-image error. This metadata-only response contains no private
payload, scope, viewer or download links. Only a fully verified public package
returns ready links; unknown or corrupted artifacts remain unavailable.

Screenshot consumers still reject raw payloads and unverified publication.
These failures preserve a typed `uiai.epwa_delivery_error.v1` envelope containing
`code: epwa_delivery_unavailable`, `delivery_state: blocked`, and only bounded
artifact/session/job/reconciliation handles. Native Pi throws its JSON encoding;
MCP returns the same object in `structuredContent` and JSON text with `isError: true`.
Neither path returns private payloads, scope, titles or publication URLs, and a
structured failure never becomes a successful evidence-delivery claim.

## Packet parity status

The browser research and diagnostics handoff is represented by `uiai.focusa_research_diagnostics_packet.v1`. The `POST /api/agent/research-packet` route and the Pi/extension `uiai_focusa_packet_build`/`uiai_focusa_packet_compose` surfaces preserve bounded evidence, redacted diagnostics, Focusa scope, and an exact next action. Packet intake remains advisory: it never grants UIAI entitlement or replaces Focusa Workpoint authority. This is proposal-only evidence handoff.

## Search and source workflows

Search is available at `/api/search` and returns provider-scoped results; callers may pass `provider=\` when selecting a provider. The keyless public fallback/second provider is bounded and must preserve redaction and stable result handles. A provider-qualified request may use `provider="wikipedia"` when the Wikipedia OpenSearch fallback is selected.

## Browser sessions and diagnostics

The browser sessions/actions/diagnostics surface includes `/api/errors`, `uiai_errors`, and `browser_diagnostics`. CLI, Pi, and MCP callers should retain a bounded session diagnostics handle after uncertain or failed actions.

## CLI, Pi, and MCP research packet

The shared CLI/Pi/MCP surface uses the `research packet` and `session diagnostics` flows so callers can compose the same bounded packet without creating a second authority.

## FPV and shares

Session creation returns no FPV share; shares are created only through the explicit share endpoints. Final production behavior requires:

- share creation as an entitled operation;
- signed/opaque, short-lived token;
- audience, session, action, and resource scope;
- `controls` separately granted;
- replay/revocation/expiry handling;
- no escalation from `/m/{token}` into normal `/api/*` execution;
- audit/redaction of operator actions;
- no public session enumeration.

A viewer may remain public by possession of a narrowly scoped token. Control is never implied by view access.

## Planned desktop presentation and app handoff

[`UIAI-COCKPIT-004`](UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md) governs the planned desktop-presentation API. It is not implemented by the current routes yet.

The planned `POST /api/session/{session_id}/present` operation requests that the single Cockpit desktop shell launch or focus and attach to the existing canonical Engine session through FPV. It does not create a second browser session or authorize takeover. `focusa://` remains owned by Focusa Menubar, `cockpit://` remains owned by Cockpit, and both schemes carry opaque handoff refs rather than tokens, raw project paths, private URLs, or page data.

Implementation status and dependency order are tracked under `uiai-engine-roadmap.12` and `T004-00..T004-12` in the local Beads graph.

## CLI, Pi, and MCP

`scripts/uiai`, the Pi extension, and MCP bridge are caller surfaces. They must all:

- accept scoped authentication/child tokens;
- project the same canonical entitlement errors;
- expose `license_feature` and limit metadata in discovery;
- deny before side effects;
- avoid persisting raw activation/license keys;
- restart/reconnect when tool schemas change;
- keep Focusa scope separate from UIAI entitlement.

## Error contract

`GET /api/health/browser` and `GET /api/metrics/browser` include an `agent_pressure` summary for long agent workflows: `uiai.agent_pressure.v1`, noncanonical operational telemetry classification, overall pressure, packet proposal authority, search provider/cache status, browser pool/queue/failure pressure, screenshot cache pressure, stored error pressure, bounded recommended actions, and a Focusa routing hint. Immediate browser pressure is derived only from current pool occupancy and queue depth; cumulative rejection, wait-time, and failure counters remain visible under `historical_pressure` without latching current admission readiness. Focusa/Pi should use this before long packet workflows or after browser/search/cache pressure symptoms instead of reading raw logs; pressure can narrow/block operational workflows but never becomes Focusa cognition truth.

Representative entitlement failures:

```json
{
  "error": "license_required",
  "state": "unactivated",
  "product": "uiai-engine",
  "feature": "uiai.session.execute",
  "message": "This operation requires an active UIAI Engine entitlement.",
  "manage_url": "https://install.focusa.dev/license",
  "retryable": false
}
```

Other stable families include wrong product, expired, revoked, stale sequence, feature missing, node limit, concurrency limit, Evaluation limit, offline deadline, and unsupported schema.

## Recovery posture

Without a valid execution entitlement, allow only bounded health, license start/status/poll/activate/refresh/doctor, safe redacted diagnostics, operator-owned data location/export where applicable, and uninstall guidance. Do not allocate a browser or create a fresh local Evaluation.

## Required tests

- loopback without lease denied before browser allocation;
- local API token without lease denied;
- reverse-proxy remote cannot inherit loopback permission;
- wrong product/feature/node/time/sequence denied;
- child token cannot exceed/outlive parent lease;
- concurrent session reservation is atomic;
- session ownership prevents enumeration/control;
- share view cannot escalate to control/API;
- errors/logs contain no secrets;
- expiry/revocation closes new execution but preserves recovery/data;
- standalone and Focusa-brokered onboarding yield the same canonical posture;
- protected worker rejects direct, replayed, wrong-audience, and downgraded calls.

## Canonical references

- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
- `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`
- `docs/ENDPOINT_AUTH_MATRIX.md`
- `docs/LICENSING.md`
- Focusa Spec 152 and Spec 150A

## OAuth URL telemetry

Session and snapshot JSON, session-info/FPV URLs, read metadata and session-open
logs use the canonical URL sanitizer. It removes userinfo and fragments, masks
OAuth code/state/assertion/token keys case-insensitively, and sanitizes bounded
nested redirect URLs (including an additional percent-encoding layer). Malformed
query encoding is withheld rather than copied through. Internal navigation URLs
are unchanged. These producer protections do not claim that retained historical
Pi/MCP transcripts have already been scrubbed.

## Legacy share-viewer security

`/v/{token}` uses a precompiled context-aware HTML template; stored metadata
never becomes trusted HTML. Only structurally valid HTTP(S) targets without
userinfo/control characters are accepted, and callback parameters are redacted
in public link/frame URLs. Frames are sandboxed with no script/form/origin
permissions. Responses use a restrictive CSP, no-store, no-referrer, nosniff,
frame denial and no-index/no-archive policy, including static 404/410 errors;
invalid stored targets return 400 without embedding them. Shared baseline
security headers also apply to Evidence PWA assets, whose existing CSP still
denies all frames. Source/browser regression proof is not installed-runtime
proof; release and consumer verification remain separate acceptance steps.

## Engine pool storage isolation

Every pooled page has a distinct non-default Chromium BrowserContext. This
covers the SessionManager path and multi-browser pools, not only profile-launcher
sessions. Cookies, local/session storage, IndexedDB and caches are isolated
between simultaneous leases. Release disposes the entire context (including
prior origins and popups) and replenishes capacity with a fresh blank context;
navigating to `about:blank` alone is not storage cleanup. Named profile adapters
retain their separate explicit profile policies. Each managed browser launch
also receives a private unique temporary profile, so MultiPool browsers in the
same engine process cannot collide on Chromium's profile lock. Launcher teardown
owns deletion of those temporary profiles.

### Input timeout recovery

After acquiring the session action lock, fill/type use a five-second child CDP
context instead of inheriting the 175 ms element-lookup context. Type propagates
selection/input errors rather than reporting success. Both input methods wait for
the actual DOM readOnly/disabled state before clearing or inserting text, preserving
read-only values on failure. The shared action snapshot
helper has a separate three-second deadline. Cancellation releases the action
lock without cancelling the session page context; diagnostics and explicit close
remain available. HTTP failures retain the session/action and error-event ID,
without recording the submitted input value. Browser-backed regressions exercise
a slow focus listener, forced timeout, diagnostic observations and close with the
same browser PID. Fixture diagnostics returned as unsealed HTTP 202 are not sealed
EPWA delivery; no provider-authentication or installed-runtime proof is implied.

### Click cancellation and uncertain effects

Click uses an independent five-second action context and preserves HTTP caller
cancellation through dispatch, settling and the bounded snapshot. Snapshot refs
still resolve through the existing session ref store. A failure after dispatch
may have taken effect: HTTP 409 pending_side_effect_unknown carries
side_effects_may_have_occurred and retry_requires_reobservation in details, with
retryable=false. Inspect diagnostics and a fresh snapshot before any authorized
retry; never automatically repeat the click. Explicit cancellation uses HTTP 408
operation_cancelled, also without automatic retry. These are operational error
postures, not control-lease grants or canonical Focusa pause/resume decisions.
An owned HTTP-202 modal fixture covers eval, async eval, snapshot, screenshot,
network diagnostics and a single ref-click without invoking a paid provider.

### Evaluation failures and cancellation

Eval and eval_async share one bounded evaluation/result/screenshot boundary.
JavaScript exceptions, navigation/context replacement, caller cancellation and
capture failures are errors, never successful result strings. A legitimate result
beginning with `error:` remains ordinary data. HTTP caller cancellation propagates
to the action, without cancelling the session's shared page context.

After dispatch, arbitrary JavaScript may have taken effect even when its await,
result or screenshot fails. Such failures use `pending_side_effect_unknown` with
`retryable=false` and explicit re-observation guidance. Cancelling a wait does not
roll back or guarantee termination of JavaScript already running in Chromium.
Errors expose a bounded `cause_class` (such as `javascript_exception`,
`execution_context_changed` or `deadline_exceeded`), not submitted scripts,
exception values or protocol parameters. The internal wrapped cause remains
available for typed classification. Both legacy Go methods remain supported.

Non-string JavaScript results retain JSON structure inside the existing result
string, rather than Go `map[...]` formatting. Credential-labelled arrays, objects
and scalar values are classified with their aliases by the shared redactor before
the result preview is truncated. A sensitive field beyond the preview budget must
still protect an earlier alias. Ordinary typed JSON data remains intact; this is
not a broker binding or a guarantee against arbitrary secret transformations.

### Open response recovery

Open responses include `X-UIAI-Created-Session-ID`, identifying only the newly
allocated session; it is not a credential or capability grant. Native Pi and MCP
use the same strict JSON/session decoder. Fetch decodes HTTP gzip once; malformed,
numeric-byte-map and missing-identity responses are errors, never spread as
numeric object properties. A rejected response closes its header-bound created
session (or a valid body identity on older servers); cleanup failure or unknown
identity remains explicit. Full transport loss before any identity arrives remains
unknown and must be reconciled, never guessed from another session's identity.
Closing an awaiting-auth session remains permitted as resource cleanup; it does
not resolve the challenge or grant resumed actuation.


## Browser content redaction

Browser text, snapshot names, DOM projections, console argument previews, action
titles, evaluation projections, research capture summaries, and persisted error
text share `internal/secrettext` before output/persistence and before truncation.
It covers bearer/credential assignments, opaque values next to API-key/token
labels, credential-labelled controls and aliases, and PEM private-key blocks.
Explicitly labelled recovery-material sections are withheld as a unit, never
parsed or enumerated; this grants no recovery-code automation authority.

Read responses expose `metadata.redaction` with occurrence count/classes and a
constant `[REDACTED_SECRET]` marker, never credential length/prefix. Ordinary
prose and explicit documentation placeholders remain readable. This boundary
is disclosure reduction, not credential-use permission or a memory-wiping
promise; it does not claim OCR/image redaction or replace private evidence and
authentication policy. Synthetic fixtures only are used for regression proof.


OAuth query/fragment values and nested redirect parameters use the canonical
URL sanitizer, including URLs embedded in free-form diagnostics, errors, packet
summaries and evaluation results. Sanitization precedes truncation/persistence;
raw navigation inputs are not rewritten. Existing private transcript/journal
history is not silently edited or represented as scrubbed by a runtime upgrade.


The explicit public FPV share endpoint is view-only. `controls: true` fails closed
with `fpv_control_approval_required`; self-asserted confirmation booleans do not
supply an operator grant. A separate canonical control-approval resolver remains
required before that stronger capability can be exposed. This does not claim the
remaining origin/consent/revocation work complete or weaken evidence publication.


FPV CDP streaming uses one session-owned Chrome pump with latest-frame fanout;
disconnecting one viewer cannot stop capture for another. `stream.cdp.mjpg?fps=8`
accepts1–30FPS, rejects malformed/out-of-range values, and bounds writes to2seconds.
No first frame within3seconds returns typed `fpv_stream_stalled` rather than a
successful empty stream. Static pages need not repaint continuously. Native
multipart `<img>` load events do not measure per-frame FPS or glass-to-glass delay;
the viewer reports Live/Not measured instead of invented numbers. Local conformance
measures mutation-to-received-decoded-JPEG separately from public-network latency.
The same binary embeds the canonical FPV template and generated asset bundle;
`bash scripts/build-fpv-assets.sh` is its sole generated-asset writer.

Native MJPEG readers receive the retained last JPEG on idle output ticks so decoders
can finalize a frame without waiting for another page repaint; this playback does
not assert new capture/freshness. Smooth selects8FPS, balanced4FPS, saver1FPS.
