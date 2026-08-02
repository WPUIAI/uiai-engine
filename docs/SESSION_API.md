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
| `GET` | `/api/session/{id}/read` | bounded text/Markdown read | session feature/scope |
| `GET` | `/api/session/{id}/snapshot` | accessibility tree and `@ref`s | session feature/scope |
| `POST` | `/api/session/{id}/click` | click CSS selector or `@ref` | action scope, confirmation policy where required |
| `POST` | `/api/session/{id}/fill` | fill input | action scope |
| `POST` | `/api/session/{id}/type` | type text | action scope |
| `POST` | `/api/session/{id}/press` | keypress | action scope |
| `POST` | `/api/session/{id}/scroll` | scroll | action scope |
| `POST` | `/api/session/{id}/screenshot` | capture current page | screenshot feature/limit |
| `GET` | `/api/session/{id}/diagnostics` | console/exceptions/network/failures | diagnostics feature and redaction |
| `POST` | `/api/session/{id}/eval` | run bounded JavaScript | sensitive feature; may be excluded from Evaluation |
| `POST` | `/api/session/{id}/css` | inject CSS | sensitive action scope |
| `GET/POST` | session cookie/auth routes | save/load state | explicit sensitive feature, encrypted storage, no logs |

The exact current route list remains in code/tool discovery and must be generated into the endpoint-feature coverage ledger before release.

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

## Packet parity status

The browser research and diagnostics handoff is represented by `uiai.focusa_research_diagnostics_packet.v1`. The `POST /api/agent/research-packet` route and the Pi/extension `uiai_focusa_packet_build`/`uiai_focusa_packet_compose` surfaces preserve bounded evidence, redacted diagnostics, Focusa scope, and an exact next action. Packet intake remains advisory: it never grants UIAI entitlement or replaces Focusa Workpoint authority. This is proposal-only evidence handoff.

## Search and source workflows

Search is available at `/api/search` and returns provider-scoped results; callers may pass `provider=\` when selecting a provider. The keyless public fallback/second provider is bounded and must preserve redaction and stable result handles. A provider-qualified request may use `provider="wikipedia"` when the Wikipedia OpenSearch fallback is selected.

## Browser sessions and diagnostics

The browser sessions/actions/diagnostics surface includes `/api/errors`, `uiai_errors`, and `browser_diagnostics`. CLI, Pi, and MCP callers should retain a bounded session diagnostics handle after uncertain or failed actions.

## CLI, Pi, and MCP research packet

The shared CLI/Pi/MCP surface uses the `research packet` and `session diagnostics` flows so callers can compose the same bounded packet without creating a second authority.

## FPV and shares

Session creation may return an FPV share in the current code. Final production behavior requires:

- share creation as an entitled operation;
- signed/opaque, short-lived token;
- audience, session, action, and resource scope;
- `controls` separately granted;
- replay/revocation/expiry handling;
- no escalation from `/m/{token}` into normal `/api/*` execution;
- audit/redaction of operator actions;
- no public session enumeration.

A viewer may remain public by possession of a narrowly scoped token. Control is never implied by view access.

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
