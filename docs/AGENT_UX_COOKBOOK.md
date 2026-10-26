# UIAI Agent UX Cookbook

This cookbook gives agents and operators repeatable UIAI workflows with clear intent, tools, evidence handles, Focusa handoff, and cleanup.

## Recipe 1: Search → open → read

Use when an agent needs web content and should avoid raw SERP blobs.

CLI helper for a selected search result:

```bash
scripts/uiai-open-result.sh --query "UIAI Engine browser agents" --index 1 --out /tmp/uiai-open-result.json
jq '{ok,selected_url,session_id,evidence_refs,cleanup}' /tmp/uiai-open-result.json
```

The helper also accepts `--search-json <file>` when a prior search response already exists, validates invalid indices, returns Focusa-ready metadata, and recommends `close_when_done`.

### Pi

Evidence-producing calls automatically compile and yield a durable HTTPS EPWA link when the producer returns a ready delivery. No separate export/share command is needed. Supply the existing verified `focusa_scope` (project, workstream, workset, CallGraph, Workpoint, work item and continuity refs); the adapter forwards it per request without account-specific defaults or invented bindings. Missing scope or publication remains an explicit delivery failure.

The collapsed result shows the evidence page and portable-copy URLs. Raw-only responses, missing required capture envelopes, invalid HTTPS links and non-ready deliveries are rejected; committed artifact/session/job reconciliation handles are retained. Dispatch acknowledgement is not completed delivery. This adapter surfaces returned terminal evidence; it does not manufacture evidence for unrelated background jobs.

Use `scripts/install-agent-integrations.sh` to install the adapter and its shared `uiai/epwa-contract.mjs` dependency together. `UIAI_PI_EXTENSION_DEST` and `UIAI_MCP_CONFIG_DEST` select arbitrary installation paths. Pi and MCP use the same validator. Reload the affected Pi session or start a fresh normal session after approved installation; source changes alone do not activate it. A legacy engine without EPWA publication must be upgraded/configured through the approved delivery path, not accommodated by silently dropping evidence.

```text
uiai_search query="<topic>" limit=3
uiai_browser_open url="<selected result url>" focusa_scope={<existing verified scope refs>}
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_close session_id="<sid>"
```

### MCP

```text
browser_search query="<topic>" limit=3
browser_open url="<selected result url>" focusa_scope={...}
browser_read session_id="<sid>" max_chars=2000 include_links=true
browser_close session_id="<sid>"
```

### CLI/HTTP

```bash
scripts/uiai --json tools search search
curl -s -X POST "$UIAI_ENGINE_URL/api/search" -H 'Content-Type: application/json' -d '{"query":"<topic>","limit":3}'
```

Evidence handles:

- `uiai-search:<provider>:<query-hash>:<rank>`
- `uiai-browser:session=<id>:read:<seq>`

Cleanup:

- Always close the browser session when the page is no longer needed.


## Recipe: Source → Markdown / JSONL

Use when an agent needs a public webpage, social thread, GitHub issue/PR, Reddit thread, YouTube transcript, or similar source converted into clean, cited, agent-ready Markdown or JSONL.

Generic webpage Markdown is implemented; source-specific adapters and JSONL chunking remain roadmap items in [`SOURCE_TO_MARKDOWN_AGENT_SPEC.md`](SOURCE_TO_MARKDOWN_AGENT_SPEC.md).

Implemented MVP surfaces:

```text
uiai_source_to_markdown url="<public-url>"
source_to_markdown url="<public-url>"
scripts/uiai markdown <public-url> --max-chars 4000
scripts/uiai session read <sid> --format markdown --selector main
```

Evidence handles:

- `uiai-source-markdown:sha256:<prefix>`
- source-specific record refs for future JSONL chunks

Rules:

- Public-only by default.
- Return structured `source_blocked` / `auth_required` failures instead of inventing content.
- Preserve public tokenizer type labels such as `html.ErrorToken:` and `html.TextToken:`; their following `return`/`if` keywords are not secret aliases. Bare token labels and explicit credential-role assignments, including short actual values, still receive redaction.
- URL redaction preserves Markdown link/image/inline-code boundaries and adjacent citations, including balanced URL path parentheses and IPv6 brackets. Credentials, userinfo, fragments and nested OAuth redirect values still pass through the canonical safeguards; surrounding URL delimiters remain outside the redacted URL span.
- Markdown text-node boundaries retain normalized spaces around inline emphasis, links, code and spans; block edges still trim. This is not a full CommonMark or browser-layout fidelity guarantee.
- After opening its owned session, one-shot conversion gives initially empty text a five-second, caller-cancellable read budget. It reuses canonical redacted extraction, never retries mutations, and returns immediately for nonempty text. Still-empty output returns HTTP422 `source_content_unavailable` with a persistent-session/diagnostics next step, not an empty successful snapshot. Error replies report owned-session cleanup. Already-cancelled callers do not open a page; read cancellation does not poison the shared page lifetime. Navigation retains its existing separate budget; nonempty text is not a semantic-completeness or all-JavaScript-ready guarantee.
- Preserve Focusa-ready `target_ref`, `evidence_ref`, `summary`, and next tools.

## Recipe 2: Persistent browser loop with @refs

Use when the agent must interact with a web UI repeatedly.

```text
uiai_browser_open url="<url>"
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_click session_id="<sid>" selector="@e3"
uiai_browser_fill session_id="<sid>" selector="@e4" text="..."
uiai_browser_press session_id="<sid>" key="Enter"
uiai_browser_read session_id="<sid>" max_chars=2000
uiai_browser_close session_id="<sid>"
```

Rules:

- Prefer snapshot `@ref` selectors over guessed CSS.
- Read page text when content matters more than pixels.
- On any failed action or unexpected navigation, run diagnostics before patching or retrying.

Evidence handles:

- snapshot/read/session metadata in response `focusa` fields
- browser error events as `uiai-error:<error_id>` when failures occur

## Recipe 3: Diagnostics-first debugging

Use when the page is blank, broken, flaky, or action output is surprising.

```text
uiai_browser_diagnostics session_id="<sid>" limit=100
uiai_errors limit=20 source="browser_session"
```

HTTP:

```bash
curl -s "$UIAI_ENGINE_URL/api/session/$SID/diagnostics?limit=100" | jq '{summary,console,exceptions,failed_requests,focusa}'
curl -s "$UIAI_ENGINE_URL/api/errors?source=browser_session&limit=20" | jq .
```

If diagnostics show relevant console/errors/network failures, compose a packet or intake directly:

```text
focusa_browser_diagnostics_intake diagnostics=<diagnostics-json> target_ref=<target> project_root=<root> continuity_id=<continuity>
```

Rules:

- Do not rely on screenshots alone for broken pages.
- Preserve bounded diagnostic summaries and evidence refs, not raw full logs.
- Clear diagnostics only after evidence is captured or when starting a clean repro.

## Recipe 4: Proof packet capture

Render packet/diagnostics artifacts for human reports:

```bash
scripts/render-diagnostics-artifact.py /tmp/uiai-research-packet.json
```


Use when an agent needs a bounded, redacted proof bundle for Focusa or a release report.

Steps:

1. Run search/read/snapshot/diagnostics or screenshot/share actions.
2. Compose a packet.
3. Capture/link evidence in Focusa.
4. Close the session.

Pi:

```text
uiai_focusa_packet_compose goal="<proof goal>" mode="proof" responses=[search,read,snapshot,diagnostics] focusa_scope={...} cleanup_session_id="<sid>"
```

One-command CLI:

```bash
scripts/uiai research packet --url https://example.com --goal "Proof packet" --out /tmp/uiai-research-packet.json
```

Manual CLI composer:

```bash
scripts/uiai --json packet compose /tmp/uiai-packet-request.json | tee /tmp/uiai-packet.json
```

HTTP:

```bash
curl -s -X POST "$UIAI_ENGINE_URL/api/agent/research-packet" \
  -H 'Content-Type: application/json' \
  --data @/tmp/uiai-packet-request.json | jq .
```

Focusa evidence capture:

```text
focusa_evidence_capture(
  target_ref=packet.recommended_focusa.args_preview.target_ref,
  result=packet.recommended_focusa.args_preview.result,
  evidence_ref=packet.recommended_focusa.args_preview.evidence_ref,
  project_root=<root>,
  continuity_id=<continuity>
)
```

Evidence handles:

- packet `evidence_refs[]`
- packet `recommended_focusa.args_preview.evidence_ref`
- `epwa_delivery.delivery_id`, durable HTTPS `artifact_url`, and portable `portable_url`; never a local path

## Recipe 5: Visual QA with screenshot/share artifacts

Use when pixels matter: layout, visual regressions, frame renders, or human review.

One-shot screenshot:

```text
uiai_screenshot url="<url>" width=1280 height=800 format="jpeg" quality=70
```

Session screenshot:

```text
uiai_browser_screenshot session_id="<sid>" fullPage=true output="file"
```

Every screenshot response must be a ready `uiai.artifact_result.v2` carrying identity-bound `uiai.epwa_delivery.v1`, a durable HTTPS `artifact_url`, and a portable `portable_url`. Compatibility `output` flags are input hints only; raw bytes, base64, local paths, and URL-only results are never successful delivery.

Share artifact:

```bash
curl -s -X POST "$UIAI_ENGINE_URL/api/share" \
  -H 'Content-Type: application/json' \
  -H 'X-UIAI-Project-Ref: <project-ref>' \
  -H 'X-UIAI-Workstream-Ref: <workstream-ref>' \
  -H 'X-UIAI-Workset-Ref: <workset-ref>' \
  -H 'X-UIAI-CallGraph-Ref: <callgraph-ref>' \
  -H 'X-UIAI-Workpoint-Ref: <workpoint-ref>' \
  -H 'X-UIAI-Work-Item-Ref: <work-item-ref>' \
  -H 'X-UIAI-Continuity-Ref: <continuity-ref>' \
  -H 'X-UIAI-Work-Items: <projected-work-items-json>' \
  -d @/tmp/share-request.json
```

Frame render:

```text
uiai_frame_catalog
uiai_frame_render frameId="<frame>" imageBase64="<trusted-input-image>"
```

`imageBase64` is a bounded frame-render input, not evidence delivery. The rendered output must still return the ready EPWA envelope and portable package.

Evidence handles:

- `uiai-screenshot:sha256:<prefix>`
- `uiai-share:<share_id>`

Rules:

- Keep raw base64 out of transcript and Focusa state.
- Store or cite stable artifact refs and bounded summaries.
- Run diagnostics if the screenshot is blank or visually broken.

## Recipe 6: Release proof loop

Use when changes are ready to ship.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./integrations/pi/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh && scripts/smoke-agent-integrations.sh && scripts/smoke-mcp-tool-routes.sh'
```

Rebuild/restart and live smoke:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && cp ./uiai-engine ./uiai-engine.bak.$(date +%Y%m%d-%H%M%S) && go build -o ./uiai-engine ./cmd/uiai-engine'
systemctl restart uiai-engine.service
curl -fsS http://127.0.0.1:7456/health | jq .
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh && scripts/smoke-agent-integrations.sh'
```

Push and CI:

```bash
cd /home/wpuiai/uiai-engine
gh auth setup-git
git push origin HEAD:main
gh run watch <run-id> --exit-status
```

If CI fails, use [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md).

Evidence handle format for final report:

```text
gh-run:<run-id>:green+git:<sha>+service:uiai-engine:healthy
```

## Recipe 7: Remote authenticated agent use

Use when the agent is outside loopback.

```bash
export UIAI_ENGINE_URL="https://<authenticated-tunnel>"
export UIAI_API_KEY="..."       # or UIAI_BEARER_TOKEN
# See REMOTE_AUTH_EXAMPLES.md for copy/paste-safe curl, CLI, Pi, and MCP examples.
scripts/uiai health
scripts/uiai tools graph
```

Rules:

- Discovery endpoints are public; browser/session/search/packet route families require auth remotely.
- Never paste token values into docs, logs, or reports.
- Prefer environment variables and approved secret storage.

The CLI and optional-auth smoke scripts source `scripts/uiai-http-auth.sh`;
keep that companion file alongside them. API-key authentication takes precedence
when both credential variables are configured; neither variable means no auth
header. The helper uses positional arguments, not empty optional arrays. Evidence
scope values reject C0/DEL characters before transport while preserving printable
header values. Run `python3 -m unittest discover -s tests -p test_cli_optional_auth.py -v`
for loopback fixtures; set `UIAI_TEST_BASH=/bin/bash` on macOS to exercise its
system shell. Linux Bash or compatibility flags alone are not Bash 3.2 proof.

## Recipe 8: MCP tool freshness after changes

Use after adding/removing/renaming tools or changing MCP call routes.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs && scripts/smoke-mcp-tool-routes.sh'
```

Then restart/reconnect MCP clients and reload Pi sessions that use the MCP adapter. For cache-layer symptoms and exact reconnect steps, see [MCP Cache and Reconnect Troubleshooting](MCP_CACHE_RECONNECT_TROUBLESHOOTING.md).

Symptoms of stale MCP metadata:

- a new tool works in HTTP/Pi but not MCP
- `tools/list` misses a new tool
- a removed tool is still visible
- `tools/call` fails for an advertised tool

## Cross-cutting rules

- Prefer `browser_read` for content and `browser_snapshot` for action targets.
- Prefer `browser_diagnostics` before patching broken browser behavior.
- Prefer stable evidence handles over raw blobs.
- Close sessions when done.
- Use Focusa tools for durable evidence, Workpoint, prediction, and metacog writes; UIAI packets are proposals, not Focusa memory.
- Focusa intake records UIAI diagnostics/errors but cannot bypass UIAI URL safety; treat `url_not_allowed` as hardened private-target policy evidence.

## Related docs

- [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md)
- [Session API](SESSION_API.md)
- [Browser Diagnostics Spec](BROWSER_DIAGNOSTICS_SPEC.md)
- [Release Deploy Runbook](RELEASE_DEPLOY_RUNBOOK.md)
- [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md)
- [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md)


### Viewer-aware FPV collaboration

Poll the authenticated `/api/fpv/events?session_id=<owned-session>&since_seq=<cursor>`
for `fpv_viewer_joined` / `fpv_viewer_left`. Events carry a non-authorizing hashed
`share_ref`, never the capability token or user agent. The read-only
`/m/{capability}/events` exposes only that share's cursor/events, and `/status`
includes `viewer_state` both directly and in its context. `viewer_count` counts
active stream connections, not authenticated people; cookie-deduplicated
`viewer_join_count`, `viewer_since`, and `last_viewer_at` describe accepted page or
status requests. Polls/refreshes with the share-path cookie do not re-announce arrival.
Stream disconnect releases capacity and emits left; denied/closed/expired shares
cannot announce arrivals. These signals are advisory, not proof of rendered pixels,
operator approval, authenticated identity, or consent. No outbound webhook is sent.
Existing explicit-share, publication and control-approval policy remains unchanged.

Native/MCP capture and research-packet callers may supply the complete typed
`focusa_scope`, including project/workstream/workset/CallGraph/workpoint/work-item
refs and work-item metadata. Adapters forward exact caller fields without filling
missing authority refs. Scope presence is transport metadata, not a verified grant
or sealed/publication success. Source adapter tests do not prove an already-running
Pi/MCP process has reloaded or a real requesting channel received an artifact.

For a standalone MCP installation, run the canonical integration installer with
`UIAI_MCP_BUNDLE_ROOT` set to the verified installation directory. It preserves
relative imports by installing the MCP entry, contract bridge, canonical contract,
and shared response decoder together. `UIAI_INSTALL_PI=0` limits that delivery to
MCP assets/configuration without adding or replacing any Pi extension. Back up the
existing installation first; reconnect MCP and verify the actual discovered schema
and consumer response. A staged configuration is not an activated configuration.

Tool discovery treats an exact qualified identifier such as `browser_screenshot`
as a single-tool lookup (case-insensitive, surrounding whitespace ignored). Broad
keywords such as `screenshot` retain ranked discovery; full inventories remain at
`/api/tools/openai` and `/api/tools/mcp`.

AI credential-settings refresh inherits the completion caller deadline and
cancellation, rejects malformed endpoints without a panic, and caps complete
responses at 1 MiB. Transport/read failures expose bounded value-free errors, not
URLs, headers, or provider bodies. Rejected or partial responses never populate
the cache. Cached AI settings are reused for at most 60 seconds; shorter configured
TTLs still win, nonpositive TTLs disable reuse, and backward clock jumps expire the
snapshot. Local `Provider.InvalidateCredentialCache()` clears cached keys and model
metadata and fences older in-flight responses, including environment-fallback
responses. Cancellation is checked before publishing a refreshed snapshot.

This is local cache/transport hardening, not a broker revocation endpoint or lease
contract. No broker event subscription or credential-reference binding is added;
existing environment fallback is unchanged. Already-started provider requests are
not cancelled. Cached key storage uses owned mutable bytes, overwritten on local
invalidation, replacement, and on-demand expiry before refresh (even if refresh
fails). Inference takes a detached copy while the cache lock is held and clears
that mutable copy on return; the owned credential-settings response buffer is
also cleared. Formatting (including numeric verbs), JSON and text serialization
remain masked. This is best-effort buffer hygiene, not a whole-heap wipe: immutable
strings, environment/configuration, decoded JSON and HTTP/runtime copies can remain.
There is no background idle-cache eraser or upstream revoke/event implementation.
Full broker revocation timing and memory-wipe acceptance remain open under #86
(credential revocation and memory hygiene); this change grants no authority to
retrieve/use new provider credentials.

Credential-settings and inference redirects stay within the configured origin
(scheme/host/port); credentials and prompt/image bodies are not forwarded to a
foreign origin, a changed scheme, or a redirect URL containing user information.
All inference dialects reuse the canonical `httpboundary.OriginBoundClient` copy,
not a second validator or a mutation of the shared client. Same-origin redirects
retain existing caller policy, transport/timeout, and the default ten-request bound.
MiniMax vision derives its route only from the caller-resolved API base, retaining
custom origins and ordinary path prefixes/trailing slashes instead of silently
substituting the official host. The default official route and existing text
fallback remain unchanged. General initial endpoint validation and query/fragment
base-URL support remain outside these changes.

AI upstream HTTP/application failure responses expose a provider label and numeric
code, never the upstream application message. Retry classification is carried
separately from display text and survives error wrapping: HTTP429/5xx can use the
existing configured AI failover chain, while HTTP401/403 do not trigger failover.
Application failures retain legacy retry signals without retaining or logging the
untrusted message. This adds no provider, credential, setup probe or fallback
configuration; positive failure/failover tests use synthetic upstream fixtures.
Inference request construction is checked before attaching credentials: malformed
URLs return a safe, non-retryable error rather than a panic. One shared send
boundary projects transport errors into value-free failure classification; it
retains cancellation/deadline `errors.Is` identity as flags, not raw URL/error
causes. Legacy retry signals and configured failover remain intact, and canceled
completion does not start another provider request. A shared response decoder
rejects read failures, invalid JSON/types, and missing textual content before
publishing completion. Parse/content failures are non-retryable; transient read
failures retain existing retry/context classification. Anthropic text blocks are
collected after non-text blocks (legacy type-omitted text remains supported),
without rewriting successful text whitespace; MiniMax VLM retains its existing
trim behavior. Owned decoding buffers are wiped best-effort, not whole-heap copies.
Response-size admission and general initial endpoint policy remain separate.

### Configured search fallback

Unnamed search requests may use an operator-approved ordered fallback list:

```yaml
search:
  default_provider: brave
  fallback_providers: [wikipedia]
```

An explicit request `provider` is always pinned. An empty fallback list preserves
legacy single-provider behavior; merely having another credential does not consent
to disclosing a query to another provider. Disabled, unimplemented and repeated
alternates are skipped. Credentials alone never make an adapter ready.

The selected provider owns result ranks, evidence references and cache entries.
If every attempt fails, the final provider's HTTP status and Retry-After survive.
Caller cancellation/deadlines reach upstream requests; redirects cannot expand the
configured origin. Retiring a provider prevents serving its old cached results.

Use `GET /api/search/providers` for installed adapters and current admission.
Issue90's remaining adapters, quota accounting and complete acceptance stay open;
the following sections describe implemented slices and their limits.
No production fallback list is enabled by a code deployment alone.

### Historical provider quota observations

The existing search runtime can retain one latest valid numeric Brave header
observation per provider slot. `/api/search/providers` and browser health expose
it as provider `quota`; the same projection appears in agent-card/setup
`quota_observations`. Other providers or missing/malformed headers stay unknown.
Global `remaining_quota_status` deliberately remains `unknown`: an as-of response
is neither a verified account balance nor permission to spend.

`uiai.search_quota_observation.v1` is `advisory_only`, scoped to
`engine_runtime_provider_slot`, with `account_binding:not_verified` and
`current_remaining_status:unknown`. Its windows contain `limit_as_observed`,
`window_seconds`, `remaining_as_observed`, `reset_after_seconds_as_observed` and
`unlimited_as_observed` (documented zero limit). `received_at` is local header
receipt time, not a provider billing timestamp. After five minutes or clock
uncertainty the projection is stale; `reported_reset_elapsed` never refills a
counter or proves quota restoration. Counts remain explicitly historical.

Only the documented coherent Brave `X-RateLimit-Limit`, `-Policy`, `-Remaining`
and `-Reset` contract is supported: at most eight aligned windows, bounded
numeric/header sizes, no arbitrary header/body/query/credential strings retained.
The execution owner supplies the internal provider binding; request URLs cannot
mint it. Earlier-started responses cannot overwrite newer attempts' observations.
Cache hits, inventory/card reads and health never query providers, refresh the
observation timestamp or simulate billing. This is not a durable account ledger,
credential-bound balance, automatic admission rule, authorization or full issue90
completion. Validation uses loopback fixtures; no paid provider probe is needed.

Protocol: https://api-dashboard.search.brave.com/documentation/guides/rate-limiting
(reviewed2026-10-26, public source `uiai-source-markdown:sha256:7e8c59a2eaf6f8e7`).

### Keyless DuckDuckGo HTML search

`provider: "ddg"` uses `https://html.duckduckgo.com/html/` without credentials,
browser sessions, result-link fetching, pagination or challenge solving. Existing
configured providers retain implicit selection priority; only otherwise
unconfigured installations gain the public keyless web default. Explicit defaults,
disables, pinned requests and approved fallback/research disclosure remain controlling.
Wikipedia remains encyclopedia search, not a general-web substitute.

The adapter uses the Go HTML parser (`golang.org/x/net/html`, x/net v0.58.0,
compatible with the existing Go1.25 baseline), sharing the bounded2MiB/12s,
origin-confined transport, normalization, citations, cache and circuit owner.
Tracking links are decoded locally, not followed; ads and unsafe citation URLs
are omitted. A provider challenge (including HTTP202) is a non-retryable failure
with `respect_provider_challenge` guidance, never a false empty result. Unknown
markup fails closed; only an explicit no-results marker permits an empty success.

Each engine runtime admits at most one uncached DDG request every15s. Concurrent
callers receive local429 with Retry-After and safe `wait_then_retry` guidance;
no upstream status or upstream failure count is fabricated. No queue, background
probe or automatic retry exists. Cache hits do not consume a request slot.
Discovery reports this local admission delay but never performs a provider probe
or claims known quota/availability. A ready configuration does not guarantee that
DDG will serve a given network; challenge failures cannot establish clean-install
usability. Neither additional query disclosure nor private infrastructure is enabled.

Protocol sources: DuckDuckGo's official non-JavaScript help and
`https://html.duckduckgo.com/robots.txt` (Allow:/ reviewed2026-10-26); this is not a
provider SLA. Module v0.59+ needs a Go-baseline upgrade outside this dependency
addition. The v0.58 parser differs from v0.61 only by an equivalent min() refactor;
five OSV advisories at v0.58 affect the unimported x/net/http2 package, not html/atom.
This package/reachability check is not a whole-project vulnerability-clean claim.

Tavily uses its documented `POST https://api.tavily.com/search` contract and
header-only bearer authentication. Supply an operator-owned credential reference
through `search.providers.tavily` or `TAVILY_API_KEY`; configuration does not
prove provider entitlement or available credits. Basic search is explicit;
automatic parameter upgrades, generated answers, raw content and images are off.
All adapters share bounded 2 MiB complete-JSON transport and value-free errors.
Explicit credential sources cannot fall back silently to a different env key.
Protocol source: https://docs.tavily.com/documentation/api-reference/endpoint/search
(reviewed2026-10-25; local synthetic protocol tests, no paid provider probe).

Exa uses its documented `POST https://api.exa.ai/search` contract with `x-api-key`
header authentication, `type: auto`, bounded `numResults` and highlights only.
Set operator-owned `search.providers.exa` credentials or `EXA_API_KEY`; no key or
fallback configuration is enabled by installation. Highlights, URLs and dates
use the existing normalized/redacted result fields. Generated answers, entities,
subpages and full-page extraction are not requested.
Protocol source: https://exa.ai/docs/reference/search (reviewed2026-10-25;
synthetic protocol/HTTP consumer tests only, no real provider query or fee).

### Search-provider cooldowns

Each mounted search router owns bounded, in-memory circuit state for its installed
providers; it is neither a credential store nor a cross-instance quota ledger.
Two consecutive upstream429/408/5xx responses open a cooldown:30s initially,
exponential backoff capped at5min. A valid upstream `Retry-After` is a minimum,
including a future HTTP date. Caller cancellation/deadlines and non-transient
responses do not open circuits. After cooldown, only one caller-triggered probe
is admitted; no background/provider probes are generated. A successful probe
resets the circuit. Older in-flight completions cannot clear a newer circuit.

Approved fallback continues past cooling providers; an explicitly pinned
provider never discloses the query to an alternate. If its circuit is cooling,
the pinned call returns503 `search_provider_circuit_open`, `retryable:true` and
`Retry-After`. Cached hits can still serve without contacting the upstream.
`GET /api/search/providers` adds per-provider `circuit` state, failure count,
retry delay and probe-in-flight status. Configured-but-cooling providers remain
`configured:true`, but are degraded and omitted from `ready_providers` until a
recovery probe can be attempted. Aggregate browser/metrics pressure now shares that same runtime inventory;
remaining credits still report unknown.
Circuit state resets with that router/runtime, not with a durable quota ledger.

Proof uses fake clocks, concurrent probe admission and local synthetic HTTP
providers (no actual paid provider calls). Issue90 remains open for other
adapters, richer results/fusion, quota/health integration and zero-key acceptance.

### Operator-configured SearxNG (no client API key)

`search.providers.searxng.api_url` must name an explicitly approved full search
endpoint, for example `https://search.operator.example/search`. No implicit
public instance or default endpoint is selected. HTTPS is required except for
loopback local/dev endpoints; user info, endpoint query credentials and fragments
are rejected. A missing/invalid endpoint is not advertised ready. Explicitly
disabled instances remain disabled.

The adapter posts form-encoded `q` and `format=json` to keep query text out of
request URLs. The instance must enable JSON in `settings.yml`; upstream403 is an
honest non-retryable provider error, not a successful empty result. Normalized
results use title/URL/content/publishedDate and the shared bounded, origin-confined
transport/redactor. No credentials are sent by this adapter. Add `searxng` to the
approved fallback list only when query disclosure to that instance is intended.
The instance may itself forward queries to external engines.

Protocol: https://docs.searxng.org/dev/search_api.html and
https://docs.searxng.org/dev/result_types/main/mainresult.html (reviewed2026-10-25).
Synthetic local-provider proof is not a deployed SearxNG service or out-of-box
zero-key web-search acceptance. No service installation or production configuration
is implied by shipping the adapter; issue90 remains open.

### Bounded search-result cache

The default query/result cache TTL is one hour, as specified in issue90.
`UIAI_SEARCH_CACHE_TTL_SECONDS` still overrides it;0 disables caching. The shared
result cache holds at most512 entries. Writes prune expired entries, and a new
entry at capacity evicts the earliest-expiring entry (deterministic key tie-break).
Updating an existing key does not evict another entry. Cache results remain
cloned; provider/query/limit identity and existing disabled-provider checks remain.
Search pressure reports `cache_max_entries` and prunes expired entries before
reporting occupancy. This bounds entry count, not an exact measured byte quota or
per-account credit ledger. No duplicate result-cache store was introduced.

### Compatible normalized search-result fields

`uiai.search_results.v1` adds `snippet` (same normalized/redacted text as legacy
`description`) and canonical `provider` id alongside legacy display `source`.
`published_at` is present only for a source-supplied RFC3339 timestamp or date;
relative `age` strings are preserved without inventing publication dates/timezones.
A finite provider-supplied `score` is optional; absent is not fabricated as0 and
scores are not calibrated/comparable between providers. Tavily/Exa/SearxNG can
supply scores; no new provider retrieval or full-page content was enabled.
Legacy fields remain available. Score values are copied at normalization and
cache boundaries so caller mutation cannot change cached results. `content_md`
remains absent because full-content retrieval is intentionally disabled. This
compatible result extension is not multi-provider fusion or provider acceptance.

### Search health uses the actual search runtime

One search runtime is created per engine route mount and passed to both search
and browser-health/metrics routes. The provider endpoint and aggregate pressure
reuse one inventory projection, including per-provider cooldown. With an injected
runtime, `readiness_basis: observed_circuits` and `default_request_ready` reflect
whether the actual default or an explicitly approved fallback can serve now.
A ready but unapproved alternate does not conceal a cooling primary. Half-open
means one recovery request is eligible, not that the provider has recovered.
Legacy callers without runtime injection are labeled `configuration_only`.
`remaining_quota_status: unknown` is not a fabricated account-credit balance;
Cockpit credit-ledger integration and issue90's other requirements remain open.

### Search result identity and deduplication

Fresh results are deduplicated before dense citation ranking and caching. Exact
normalized HTTP(S) URLs share identity; tracking-only URL variants are merged
only when titles have at least0.8 token-set similarity. Only `utm_*`, `gclid` and
`fbclid` are ignored for that comparison. Meaningful query parameters, different
hosts and unrelated titles are retained. The first original citation, text and
source metadata are preserved, not rewritten to the comparison URL.
Malformed/non-HTTP(S)/userinfo citation URLs are omitted. Dedupe never fetches
additional results or spends another provider request to fill a limit; output may
therefore contain fewer items. Cached and fresh results share citation ranks.
This is single-response deduplication, not multi-provider query/rank fusion;
issue90 remains open for that and its other acceptance requirements.

### Explicit HTTP research fusion (opt-in)

`POST /api/search` can accept `providers:["brave","wikipedia"]` instead of
`provider`. Two or three distinct installed providers are required. Every member
must also be the operator's resolved default or named in the approved fallback
configuration. Unsupported/unapproved/disabled members and a simultaneous pinned
`provider` are rejected before traffic. Merely possessing another key never
authorizes fanout. Ordinary requests and explicit single-provider pins do not fan
out. HTTP, installed MCP and fresh native Pi source support this field; actual native integration installation and a current-process native reload remain separate before activation can be claimed.

At most three provider calls run concurrently under one12s overall context, using
exactly the existing per-provider cache/circuit/normalization path. Successes are
merged using reciprocal-rank fusion with constant60; citation dedupe uses the same
identity owner as ordinary results. `fusion_score` is distinct from the original
provider-native `score`. Each merged result keeps its first original citation and
lists `sources` with each contributing provider, original evidence ref and rank.
The result `rank` is fused order, not a rewritten original evidence reference.

Top-level `providers` lists successes, `provider_errors` reports value-free lane
failures and `fusion_method` is `rrf60`; failures never masquerade as a full-success
provider set. All-failed is a non-success response; all429 remains429. `cached`
is true only when every requested lane succeeded from cache. No second fusion
cache or implicit refill/probe requests exist. This does not grant paid-provider
use, key acquisition or production fallback configuration; tests use local fakes.

MCP `browser_search` forwards explicit `providers` unchanged, including conflicting
`provider` inputs so the engine rejects them rather than silently changing query
routing. Omission remains omission. The engine tool catalog and MCP fallback
schema expose2..3 unique research IDs. Installing/reconnecting the MCP bridge is
separate from a native Pi extension reload; no native schema activation is implied.

The native `uiai_search` source schema also exposes the optional bounded research
list, forwarding it unchanged through the existing scope/HTTP boundary. Fresh
module/import and installer-fixture tests prove source behavior; editing the
source or reloading an unrelated MCP process does not reload a running Pi native
extension. Do not claim current native activation without that separate proof.


### Agent-first search setup (no Cockpit prerequisite)

Start with the existing agent card (`pi_uiai_agent_card` / MCP
`uiai_agent_card`, HTTP `GET /api/tools/agent-card`). Its `setup` object is the
same `uiai.search_setup.v1` projection returned by `GET /api/search/providers`;
searching existing tool discovery for `setup` finds the card. No new client tool
registration or native schema reload is needed for this additive response.
The existing CLI consumes it too: `scripts/uiai --json tools agent-card`;
`scripts/uiai --json tools search setup` discovers the same entry point.

- Inspect `ready_providers_by_capability`: `encyclopedia_search` (Wikipedia) is
  not `web_search`. A fresh installation with no credentials/endpoints honestly
  reports no general-web provider, while still permitting encyclopedia search.
- Follow `provider_setup.<id>.next_action`: use selected provider, connect an
  existing credential reference or approved endpoint, repair that existing
  reference/endpoint, respect an explicit disable, choose a supported adapter,
  or wait for the shared circuit's retry delay. Never replace credentials just
  because a circuit is cooling or a credential file is unreadable.
- Credential fields name existing secret environment variables or credential
  files, not values to paste into chat. Explicit provider configuration wins
  even when empty; conventional environment fallback applies only when that
  provider has no explicit configuration entry. Actual private references,
  endpoint URLs and values are not returned by discovery.
- Discovery is read-only: `configuration_write_supported:false`,
  `upstream_probes_performed:false`, `requires_gui:false`. It does not sign up,
  acquire keys, select public SearxNG instances, provision services, activate
  configuration, send test queries or assert a remaining balance. Apply approved
  configuration through the installation's supported deployment/reload path.
  Account creation, spending and changed query disclosure remain separate grants.
- Configured/admitted readiness is not external availability, account entitlement
  or a successful paid query. The card labels legacy standalone mounts
  `configuration_only`; engine mounts share the actual search circuit runtime.

This is the discover/detect/guide slice, not a claim of a complete automatic
configuration writer, account-acquisition workflow, general-web zero-setup
service, or Cockpit UI. Search issue #90 and key-acquisition issue #91 remain open.

An agent-style regression walkthrough covers connecting an existing environment
reference through the normal configuration loader and then requesting one local
synthetic provider search. It verifies that discovery sends no queries and that
neither configuration nor responses contain the credential value. It is not an
independent-model usability test or an actual account/paid-provider proof; no new
privileged configuration-writing API is needed for that existing-file workflow.


### Local endpoint validation and explicit-default intent

Readiness now applies the same endpoint resolver used by DDG, Brave, Exa, Tavily and
Wikipedia before advertising them as configured/ready. Defaults, allowed hosts,
override source and precedence live in the existing provider registry, not a
second setup-only table. No DNS lookup, account check or search query is sent.
An invalid endpoint yields `invalid_endpoint` / `repair_existing_endpoint` with
only safe field names; private URLs and credential-looking URL text stay out of
discovery and adapter errors.

API URLs require HTTPS except loopback development, a valid port, their existing
approved host policy, and no userinfo, query or fragment (including empty `?`/`#`
markers). Request parameters belong to the adapter protocol, not the base API
URL. SearxNG retains its explicit approved-instance policy and reuses the common
syntax validator. Brave keeps configuration-over-environment URL precedence;
Wikipedia keeps its legacy `UIAI_WIKIPEDIA_SEARCH_API_URL` endpoint override
rather than interpreting a previously unused provider configuration field.

An explicit `search.default_provider` remains the selected intent even when its
adapter is unsupported, disabled, missing a credential or misconfigured. It is
not silently replaced by another configured provider. Only explicitly approved
`search.fallback_providers` can receive its unnamed-request queries; an explicit
request `provider` stays pinned. With no configured default, the existing
first-ready selection remains unchanged. Provider discovery includes unsupported
default identifiers so agents can diagnose them rather than see a hidden swap.

Regression tests reproduce the previous false-ready endpoint, compare setup and
adapter rejection for all four adapters, fence unwanted outbound calls with
canceled contexts, preserve override precedence, verify zero discovery traffic,
and prove that a broken explicit default sends no query to an unapproved local
provider while an explicitly approved fallback works. These are synthetic/local
proofs, not real provider availability, quota or paid-query acceptance.


### Setup recovery directly in search errors

Search configuration and research-preflight errors carry structured `setup`
(a single recovery step, not the full card projection) and the existing
cross-client `suggested_next_action` string. Missing/unreadable references, invalid endpoints,
explicit disables and unsupported adapters reuse the same setup owner as the
card. Unsupported-provider suggestions list installed adapters, not unsupported
configuration/default identifiers. Cooling errors keep their HTTP status,
`Retry-After` and retryable flag, adding `wait_then_retry` guidance without probes.
When no provider can serve, the error points to capability-specific discovery
instead of assuming another key is always the remedy.

Native Pi, MCP and CLI already render `suggested_next_action`; this is a backend
response improvement, not a new client schema/tool or installation requirement.
A correctly configured provider's malformed response is not mislabeled as a
configuration problem. Guidance never grants permission, echoes private
references/URL values, creates accounts, changes configuration, or sends probes.

Research preflight uses the same formatter for conflicting search modes,
excessive or duplicate provider selections, unsupported adapters, unapproved
query disclosure, and explicit disables. It preserves the first rejection's
HTTP status and error code, adds the normalized failing provider where relevant,
and runs **zero provider requests**. Conflicts require choosing `provider` or
`providers`; research needs two to three distinct approved providers. Credential
possession is not research disclosure approval. `request_provider_disclosure_approval`
asks for that separate authorization or a narrower already-approved request;
it never changes defaults/fallbacks, enables a provider, or acquires an account.
Unsupported alternatives list installed adapters only. Validation order, valid
research execution and caching remain unchanged.

For valid research requests, `provider_errors[]` may additionally carry `setup`
and `suggested_next_action` when a lane has a known configuration problem or
circuit cooldown. Partial success still delivers the successful results and
per-lane recovery; it does not refill, retry or hide failures. If every lane
fails, the existing top-level `suggested_next_action` combines available recovery
steps with their provider names so current client error renderers retain them.
HTTP status/error codes and retry metadata are unchanged. Upstream responses,
cancellation and deadlines are not misdiagnosed as missing credentials, even
when local configuration is incomplete; an otherwise configured provider's
response failure likewise gets no invented setup diagnosis. No probe or
configuration/credential mutation is performed to generate guidance.
