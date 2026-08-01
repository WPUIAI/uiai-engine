# UIAI for Agents Quickstart

UIAI Engine is the proof browser for Focusa-powered agents: browser sessions, research, diagnostics, screenshots, and stable evidence handles. Focusa owns project identity, Workpoints, Trajectory, evidence linkage, prediction, metacognition, and recovery.

> **Release boundary:** Current UIAI code still exposes selected loopback-public execution. That is legacy development behavior, not authority-issued Evaluation. New evaluator/customer use is blocked until signed product/feature/limit enforcement is universal. Agents must not treat a successful local call, local token, extension token, or health result as a license.

## 1. Read the authority contracts

Before invoking execution-capable tools in licensing/onboarding work, read:

1. `docs/LICENSING.md`
2. `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
3. `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`
4. `docs/ENDPOINT_AUTH_MATRIX.md`
5. Focusa Spec 152 and Spec 150A when using bundle onboarding.

## 2. Start with recovery-safe discovery

Public/recovery-safe calls may inspect version, health, and tool metadata. They do not authorize execution.

```bash
export UIAI_ENGINE_URL="${UIAI_ENGINE_URL:-http://127.0.0.1:7456}"

scripts/uiai status
scripts/uiai health
scripts/uiai tools search diagnostics
scripts/uiai tools graph --json

curl -fsS "$UIAI_ENGINE_URL/health" | jq .
curl -fsS "$UIAI_ENGINE_URL/api/tools/agent-card" | jq .
curl -fsS "$UIAI_ENGINE_URL/api/tools/docs" | jq .
curl -fsS "$UIAI_ENGINE_URL/api/tools/search?q=diagnostics" | jq .
```

Tool metadata must eventually expose product-qualified `license_feature` and limit information. A listed tool may be locked.

## 3. Establish caller and entitlement separately

Target production calls require:

```text
caller authentication
+ signed uiai-engine lease or narrower child token
+ product grant
+ route feature
+ node/time/sequence validity
+ limit reservation when applicable
```

Use a short-lived scoped token:

```bash
export UIAI_BEARER_TOKEN="<short-lived-scoped-token>"
```

Do not place a raw commercial license key in normal browser calls. A license/activation credential exchanges for a signed lease; Focusa or the local broker issues narrower operation/access tokens.

Until device-code and signed-lease onboarding ships, use only synthetic/test trust roots or existing authorized internal development environments. Do not onboard external evaluators through the current loopback bypass.

## 4. Entitled browser workflow

```text
verify authority
→ browser open with optional Focusa scope
→ read or snapshot
→ act through selectors/@refs
→ diagnostics after failure/uncertainty
→ capture bounded evidence or research packet
→ close
→ commit/release usage reservation
```

Representative authorized-development calls:

```bash
AUTH=(-H "Authorization: Bearer $UIAI_BEARER_TOKEN")

SID=$(curl -fsS -X POST "$UIAI_ENGINE_URL/api/session" \
  "${AUTH[@]}" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"url":"https://example.com","focusa_scope":{"project_root":"/safe/project","continuity_id":"demo"}}' \
  | jq -r '.session.id')

curl -fsS "$UIAI_ENGINE_URL/api/session/$SID/read" "${AUTH[@]}" | jq .
curl -fsS "$UIAI_ENGINE_URL/api/session/$SID/snapshot" "${AUTH[@]}" | jq .
curl -fsS "$UIAI_ENGINE_URL/api/session/$SID/diagnostics" "${AUTH[@]}" | jq .
curl -fsS -X DELETE "$UIAI_ENGINE_URL/api/session/$SID" "${AUTH[@]}" | jq .
```

These examples describe API shape. Current middleware may accept less; do not rely on that weaker behavior.

## 5. Diagnostics-first behavior

After a failed/blank screenshot, click/navigation issue, JavaScript problem, CORS/API suspicion, console clue, or provider failure:

1. call browser diagnostics;
2. capture bounded console/exception/network/failure summary;
3. retain stable `uiai-*` evidence handles;
4. pass only bounded/redacted evidence into Focusa;
5. never paste cookies, auth headers, request bodies, keys/tokens, or raw customer data.

## 6. Search and Source-to-Markdown

Use provider metadata first, then search/capture under the applicable signed feature and limit policy. Selected results should carry bounded title/snippet/URL and stable evidence refs. Strip fragments and redact secret query values.

```bash
curl -fsS "$UIAI_ENGINE_URL/api/search/providers" "${AUTH[@]}" | jq .
curl -fsS -X POST "$UIAI_ENGINE_URL/api/search" \
  "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d '{"query":"official public source"}' | jq .
```

## 7. Focusa handoff

UIAI outputs are evidence proposals. Focusa intake must verify exact project/Workpoint scope and must not infer that UIAI was licensed merely because an output exists.

Preferred flow:

```text
UIAI stable evidence handle
→ bounded summary/metadata
→ focusa_browser_diagnostics_intake or focusa_evidence_capture
→ Workpoint evidence link
```

Focusa bundle onboarding may issue a child token only from a valid parent UIAI grant. UIAI independently verifies its audience, parent lease id/sequence/digest, node, feature, expiry, and limits.

## 8. Pi and MCP

Project-local Pi tools and MCP schemas are discovery/caller surfaces. They must:

- accept scoped tokens;
- project canonical entitlement errors;
- deny before side effects;
- preserve Focusa scope separately from UIAI entitlement;
- restart/reconnect after schema changes;
- never hard-code `pro`, `internal`, or Evaluation from token type.

Useful skills:

```text
/skill:uiai-agent
/skill:uiai-focusa-packet
/skill:uiai-release
/skill:uiai-mcp
/skill:uiai-remote-auth
/skill:uiai-docs-maintenance
/skill:uiai-ci-debug
/skill:uiai-browser-debug
/skill:vision
```

## 9. FPV and shares

Share creation and control are entitled operations. A viewer token may provide bounded public access, but must be short-lived, session/resource/action scoped, audited, revocable, and unable to escalate into general API execution.

## 10. Recovery behavior

Without a valid execution entitlement:

Allowed:

- health/version;
- license onboarding/status/refresh/doctor when implemented;
- bounded redacted diagnostics;
- operator-owned data location/export where applicable;
- uninstall guidance.

Denied:

- new sessions/screenshots/search/Markdown/analysis/media/control;
- extension token issuance that synthesizes product scope;
- protected-worker/capsule operations;
- mutation or expensive allocation.

## 11. Stable denial examples

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

Other required states include wrong product, expired, revoked, stale sequence, node limit, feature missing, Evaluation limit, concurrency limit, and authority unavailable outside signed offline grace.

## 12. Completion checklist

- Caller authentication is explicit.
- Canonical entitlement/product/feature/limit is explicit.
- No raw key/token/secret enters output or Evidence.
- Browser resources are allocated only after the gate.
- Focusa scope is exact and independently verified.
- Stable evidence refs are used.
- Session/job/share is closed or has a bounded lifetime.
- Usage reservation is committed/released.
- Failures preserve recovery and operator data.
