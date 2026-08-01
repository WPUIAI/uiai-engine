# UIAI Engine Endpoint Authentication and Entitlement Matrix

Last reviewed against current main: 2026-08-01

> **Critical status:** This document records current as-built authentication behavior and the required Spec-152 target. Current loopback-public routes and tier-derived feature behavior are legacy migration facts, not approved Evaluation entitlement. New evaluator/customer distribution is blocked until every execution/mutation route has independent authority-issued entitlement enforcement.

Source of current auth truth: `internal/auth/auth.go`.  
Source of route mounts: `internal/server/server.go`.  
Current feature helper: `internal/license/entitlements.go`.  
Normative target: `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md` and the protected-worker addendum.

## Authentication versus entitlement

Authentication answers **who/what is calling**. Entitlement answers **whether this product, feature, node, time, and limit permit execution**.

A request may be authenticated and still denied by entitlement.

The following never create entitlement by themselves:

- loopback source address;
- `UIAI_LOCAL_API_TOKEN` or `UIAI_LOCAL_API_TOKENS`;
- API key, extension token, Pi/MCP/Cockpit token, webhook secret, or Focusa pairing token;
- source checkout or successful build;
- health/browser readiness;
- a tier string reconstructed by the client;
- an editable local file.

## Current authentication modes

| Mode | Current meaning | Target entitlement rule |
| --- | --- | --- |
| public | reachable without caller credentials | recovery/public metadata only; no execution side effect |
| loopback-public remote-auth | current code permits unauthenticated loopback for selected tools | must still load/verify authority-issued lease and feature/limits before execution |
| authenticated | caller credential required | authentication only; product/feature gate follows |
| service-token | handler-specific service credential | token scopes must derive from verified service/product authority |
| handler-auth | handler validates its own token | handler must also enforce product/feature/time/node/limit state |
| public tokenized | possession of signed/opaque share token | token scope cannot escalate to general engine execution |
| recovery | health/license/diagnostic/export/uninstall-safe surface | intentionally available without active execution entitlement |

## Required middleware order

```text
request safety / body and rate limits
→ caller authentication
→ canonical signed-lease or child-token verification
→ uiai-engine product grant
→ route feature grant
→ node/time/sequence/offline-window validation
→ limit reservation
→ handler or protected worker
→ usage commit/release
→ redacted observability
```

No expensive browser page, model call, media job, persistent session, share-control channel, or mutable storage transaction starts before the entitlement decision and required reservation.

## Route matrix

| Endpoint group | Current auth behavior | Required target feature/posture | Migration status |
| --- | --- | --- | --- |
| `/`, `/dashboard` | public | bounded product/version/onboarding shell only; no private state | review public payload |
| `/health`, `/api/health`, `/api/health/*`, `/api/status`, `/api/metrics/browser` | public | recovery-safe operational readiness plus redacted entitlement state | retain public/recovery |
| `/api/license/status|start|poll|activate|refresh|doctor` | not yet canonical | recovery-safe, rate-limited authority flow | must implement |
| `/api/tools`, `/api/tools/*` | public metadata | may list locked tools with `license_feature`/limit metadata; invocation remains gated | audit metadata leakage |
| critique/UI-reverse model or operation metadata | public | public only if no private prompts/routing/customer data; otherwise gated | route review |
| `/api/copilot/health`, `/api/intelligence/health` | public | readiness only; no product result | retain after payload audit |
| `/api/screenshot`, `/api/screenshot/*` | loopback-public remote-auth | `uiai.screenshot.capture`; signed Evaluation/paid feature and limits | release blocker |
| `/api/session`, `/api/session/*` | loopback-public remote-auth | `uiai.session.execute`; concurrency/duration/idle limits | release blocker |
| `/api/search`, `/api/search/*` | loopback-public remote-auth | `uiai.search.execute`; signed feature and usage policy | release blocker |
| `/api/markdown`, `/api/markdown/*` | loopback-public remote-auth | `uiai.markdown.capture`; signed feature/limit | release blocker |
| `/api/agent/research-packet` | loopback-public remote-auth | `uiai.agent.research_packet`; gate if it invokes/exposes entitled work | release blocker |
| `/api/media/frame`, `/api/media/frame/*` | loopback-public remote-auth | public catalog may remain metadata; render/execution requires `uiai.media.frame` | split route/read-write |
| `/api/errors`, `/api/errors/*` | loopback-public remote-auth | recovery-safe redacted diagnostics only; no private request bodies/results | payload audit |
| `/api/2fa/code` | authenticated | sensitive helper; explicit feature and deployment policy, disabled by default | gate/review |
| `/api/share/*`, `/v/{token}` | mixed public/auth | viewing only through scoped signed expiring token; creation/mutation requires `uiai.share.create` | split read/write |
| `/m/{token}` status/screenshot/control | public tokenized | signed short-lived audience/resource scope; control separately granted/audited; no escalation | redesign token contract |
| `/api/media/jobs`, `/api/media/status/*` | public read | only opaque caller-owned job status; no enumeration/customer output leakage | ownership/token audit |
| media production routes | authenticated + some feature middleware | `uiai.media.produce`, limit reservation, protected worker where selected | extend gate |
| `/api/extension/verify`, `/api/extension/token` | handler-auth | issuance requires verified parent lease; token scopes no broader and no hard-coded `pro` | release blocker |
| `/api/memory/*` | public/legacy compatibility | public metadata none; reads/writes require explicit data/product feature and subject scope | release blocker |
| `/api/usage/*` | public/legacy compatibility | caller/lease-scoped usage only; no cross-account enumeration | release blocker |
| `/api/workflow/*` | public/legacy compatibility | mutating/triggering routes require explicit workflow feature | release blocker |
| `/api/training/*` | service-token | service auth plus product/feature scope; test and production roots separated | gate/review |
| `/api/intelligence/*` except health | handler-auth | explicit intelligence feature, limits, protected worker where selected | release blocker |
| `/api/admin/*` | authenticated | admin identity and separately authorized role/product operation | harden |
| `/api/reference/*` | authenticated + feature middleware | `uiai.reference.access`, usage reservation | verify coverage |
| critique/section/layout/style/copilot execution | authenticated + feature middleware | explicit signed product/features/limits; remove tier-derived blanket grants | release blocker |
| `/api/intake/*` | authenticated | read/write/execute split; mutation feature required | route review |
| design-system/content-map/block-recipes/comparison | authenticated + feature middleware | explicit signed features and usage reservation | verify coverage |
| `/api/captcha/*` | authenticated | sensitive execution feature, resource/session scope, limits | release blocker |
| `/api/migration/*` | authenticated + feature middleware | `uiai.migration`, explicit target/scope and destructive confirmation | verify/harden |
| `/api/events` | authenticated | lease/client-scoped entitled event stream; recovery stream excludes private execution | release blocker |
| `/vision/*` | authenticated | `uiai.vision.interactive`, protected worker/session token | release blocker |

## Current-code contradictions to remove

1. `FromIdentity(nil)` creates Evaluation.
2. `EvalAllowedFeatures` grants execution based on loopback.
3. `tierFeatures()` grants a broad static feature set from a tier string.
4. `UIAI_LOCAL_API_TOKEN` maps to tier `internal` rather than authentication-only identity.
5. Extension token validation creates hard-coded tier `pro`.
6. Auth middleware public/legacy exceptions allow valuable execution without a signed product grant.
7. Route mounts apply feature middleware to selected families only.
8. Health/status do not report one canonical entitlement state.

These remain accurate observations until code migration; they are not permitted target behavior.

## Required entitlement context

At minimum, a verified context contains:

```text
lease_id
license_id
license_class
status
allowed_products
features
limits/policy_version
node_id
issued_at/not_before/refresh_after/offline_valid_until/expires_at
sequence
authority_key_id
signature/payload digest
subject/client/token scope
```

Friendly tier labels are presentation only. Unknown status, schema, product, feature, token type, or sequence fails closed.

## Child-token rules

Focusa/extension/Pi/MCP/Cockpit child tokens must be:

- signed or authenticated by the approved local broker trust;
- audience-bound to exact engine/worker;
- product and feature scoped;
- node and client scoped;
- short-lived and no longer than parent lease/offline validity;
- bound to parent lease id/sequence/digest;
- optionally bound to request/session/job/limit reservation;
- replay protected with token id/nonce;
- invalidated/rejected when parent state advances to revoke/expire/replace.

## Update rules for route changes

1. Add/update the route in this matrix in the same change.
2. Add it to the generated endpoint-feature coverage ledger.
3. Classify public/recovery/authentication/entitlement/limit/protected-worker posture separately.
4. Prove that loopback does not bypass entitlement.
5. Prove reverse-proxy client-IP handling cannot turn remote traffic into license permission.
6. Add positive/negative tests for missing, wrong-product, expired, revoked, stale-sequence, feature-missing, and limit-reached cases.
7. Ensure checks occur before expensive allocation or mutation.
8. Preserve redaction of authorization, cookies, keys, tokens, codes, customer identity, request bodies, and sensitive URLs.
9. Public tokenized routes must prove non-escalation.
10. Test trust roots/fixtures must be impossible in production artifacts.

## Required verification

Current auth tests remain useful as migration fixtures, but final release proof additionally requires:

- missing identity and missing lease => recovery-only;
- authenticated local token without lease => denied execution;
- loopback without lease => denied execution;
- reverse-proxy remote cannot inherit loopback permission;
- extension token cannot synthesize `pro`;
- wrong product/feature/status/sequence/time/node => denied;
- concurrent limit reservation is atomic;
- child token cannot exceed/outlive parent;
- protected worker rejects direct/replayed/wrong-audience operations;
- public/share/status routes cannot enumerate or expose private work;
- standalone and Focusa-brokered onboarding produce the same canonical state;
- expiry preserves diagnostics/data/export/uninstall behavior.

## Canonical references

- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
- `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`
- `docs/LICENSING.md`
- Focusa Spec 152 and Spec 150A
