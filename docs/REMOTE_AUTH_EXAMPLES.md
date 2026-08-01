# Remote Authentication Examples

These examples show caller authentication for UIAI Engine. They use placeholders only.

> **Authentication is not entitlement:** A valid API/local/extension/Bearer token proves who or what is calling; it does not grant the `uiai-engine` product, Evaluation, commercial status, route features, or usage limits. Current loopback-public behavior is legacy code and is not the target production contract.

## Required production authority

Execution-capable routes require both:

```text
caller authentication
+ authority-issued signed lease or narrower verified child token
```

The entitlement check additionally validates product, feature, node, time, lease sequence/digest, status, offline window, and limits.

See:

- `docs/ENDPOINT_AUTH_MATRIX.md`
- `docs/LICENSING.md`
- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`

## Public/recovery metadata

Only bounded health, version, tool metadata, and future license recovery routes may be called without execution entitlement.

```bash
ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
curl -fsS "$ENGINE_URL/health" | jq .
curl -fsS "$ENGINE_URL/api/tools/agent-card" | jq .
curl -fsS "$ENGINE_URL/api/tools/search?q=diagnostics" | jq .
```

A public metadata response is never proof that browser execution is licensed.

## Safe scoped-token setup

Use one short-lived credential style. Values are placeholders.

```bash
export UIAI_ENGINE_URL="https://uiai.example.invalid"
export UIAI_API_KEY="REDACTED_SHORT_LIVED_API_KEY"
# or
export UIAI_BEARER_TOKEN="REDACTED_SHORT_LIVED_BEARER"
```

Do not use a raw commercial license key for ordinary route calls. Activation exchanges it for a signed lease; callers use narrower access/child tokens.

Server-side `UIAI_LOCAL_API_TOKEN(S)` may authenticate local callers during migration, but must not produce tier `internal`, product grants, or features by itself.

## Curl helper

```bash
ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"
AUTH_HEADER=()
if [ -n "${UIAI_API_KEY:-}" ]; then
  AUTH_HEADER=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [ -n "${UIAI_BEARER_TOKEN:-}" ]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
else
  echo "A scoped authentication token is required for this example" >&2
  exit 4
fi
```

## Entitled session example

This assumes the token is bound to a valid signed parent lease and includes `uiai.session.execute`.

```bash
SID=$(curl -fsS -X POST "$ENGINE_URL/api/session" \
  "${AUTH_HEADER[@]}" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"url":"https://example.com"}' \
  | jq -r '.session.id')

curl -fsS "$ENGINE_URL/api/session/$SID/read" "${AUTH_HEADER[@]}" | jq .
curl -fsS "$ENGINE_URL/api/session/$SID/diagnostics" "${AUTH_HEADER[@]}" | jq .
curl -fsS -X DELETE "$ENGINE_URL/api/session/$SID" "${AUTH_HEADER[@]}" | jq .
```

If current code accepts this call without entitlement on loopback, that is a known release blocker—not a supported Evaluation path.

## Pi, MCP, CLI, and Focusa child tokens

Tokens issued to these callers must be:

- short-lived;
- audience bound;
- product and feature scoped;
- node/client scoped;
- tied to parent lease id/sequence/digest;
- no broader or longer-lived than the parent lease;
- replay protected;
- redacted from logs, docs, packets, screenshots, Workpoints, and Evidence.

A Focusa pairing token or project scope cannot be reused as a UIAI commercial token.

## Remote/tunnel deployment

Remote exposure requires:

- TLS;
- explicit trusted proxy/client-IP configuration;
- caller authentication;
- canonical entitlement enforcement;
- origin/CORS restrictions;
- URL safety and private-target policy;
- rate/concurrency/body limits;
- session/share ownership;
- secret-safe logging;
- no proxy path that turns remote traffic into loopback permission.

Do not expose the full API publicly merely because `/m/*` tokenized viewers are intended for public access.

## Share/FPV token example

A viewer token is not an API credential. It must be bounded to one session/share and allowed actions.

```bash
curl -fsS "$ENGINE_URL/m/REDACTED_VIEW_TOKEN/status" | jq .
```

Control must be separately granted and audited. Viewer/control tokens cannot create sessions, call unrelated routes, mint extension tokens, or access other users' work.

## Error expectations

Authentication failure:

```json
{"error":"authentication_required"}
```

Authenticated but not entitled:

```json
{
  "error":"license_required",
  "product":"uiai-engine",
  "feature":"uiai.session.execute",
  "retryable":false
}
```

Other stable entitlement errors include wrong product, expired, revoked, stale sequence, feature missing, node limit, concurrency/Evaluation limit, and authority unavailable outside signed offline grace.

## Secret handling

Never place real values in:

- repository files;
- chat or issue comments;
- command history where avoidable;
- screenshots;
- Focusa Evidence/Workpoints;
- logs, traces, query strings, or error envelopes;
- public FPV/share payloads.

Prefer device-code activation, OS-protected storage, short-lived tokens, and environment injection from an approved secret store.
