# UIAI Engine Mandatory Entitlement and Onboarding Enforcement

**Status:** Proposed — release-blocking for evaluator/customer distribution  
**Created:** 2026-08-01  
**Related Focusa contract:** `Startempire-Wire/focusa/docs/152-mandatory-authority-licensing-evaluation-entitlements-and-unified-onboarding-spec.md`

## 1. Decision

Every UIAI Engine execution must be backed by a valid authority-issued entitlement, including evaluation use.

Loopback origin, a local API token, an extension token, a source checkout, or a missing identity must never create an evaluation or commercial entitlement. These may authenticate a caller or reduce network exposure, but product permission comes only from a verified signed lease or a short-lived child token derived from one.

UIAI Engine must be independently enforceable even when installed through Focusa. Focusa owns the shared onboarding experience; UIAI Engine verifies its own product grant, feature scope, time bounds, and limits before executing browser or analysis work.

## 2. Current-state audit

The current code establishes useful middleware and feature names, but the effective policy is fail-open for important local paths.

| Current surface | Observed behavior | Required change |
| --- | --- | --- |
| `internal/license/entitlements.go` | `nil` identity becomes Evaluation; loopback receives an eval allowlist; every non-evaluation tier receives the same broad `tierFeatures()` set | Require explicit authority claims; no identity means recovery-only; remove tier-derived blanket grants |
| `internal/auth/auth.go` | Loopback callers may use screenshot/session/search/markdown/research and related APIs without credentials; public/legacy exceptions include workflow, memory, usage, tools, and other routes | Separate authentication from entitlement and gate every execution/mutation path after auth |
| Local token handling | `UIAI_LOCAL_API_TOKEN` produces tier `internal` | Treat it only as caller authentication; it must carry no product features by itself |
| Extension token handling | Valid extension token is assigned hard-coded tier `pro` | Derive product/features/expiry/scope from the parent lease; never synthesize a paid tier |
| Authority validation | Auth consumes mainly `valid`, `license_id`, `tier`, and status while entitlement code reconstructs features locally | Consume a signed lease or explicit signed claims including product, features, limits, status, expiry, node, and sequence |
| `internal/server/server.go` | Feature middleware covers several analysis families, but browser sessions, screenshot, search, markdown, agent packets, vision, captcha, events, workflow, and other valuable paths are not uniformly entitlement-gated | Add a universal execution gate and a generated endpoint-to-feature ledger |
| `/health` | Hard-codes `dev_mode: false` and exposes no canonical license posture | Report a safe entitlement state summary and provide dedicated license status/recovery routes |
| Onboarding | No UIAI-specific first-run/license flow exists | Add a standalone flow and a Focusa-brokered bundle flow |

The present implementation also conflates network locality with product licensing. A reverse proxy or container network may make a remote caller appear loopback unless proxy trust is configured precisely. Even perfect client-IP detection would not make loopback an entitlement source.

## 3. Runtime state machine

UIAI Engine uses the same canonical states as Focusa:

| State | Execution posture |
| --- | --- |
| `unactivated` | Health, license recovery, safe diagnostics, and uninstall guidance only |
| `active_evaluation` | Explicit signed evaluation feature allowlist and limits |
| `active_paid` | Explicit signed product/features/limits |
| `offline_grace` | Previously verified signed scope continues until its signed offline deadline |
| `expired` | No browser/analysis execution; data and safe diagnostics preserved |
| `revoked` | No execution; refresh/manage-license path available |
| `invalid` | No execution; signature/schema/product/node/time diagnostic available |

The process may start in recovery mode so the operator can repair licensing without losing access to diagnostics. It must not silently fall back to evaluation.

## 4. Identity and entitlement separation

`auth.Identity` answers **who or what is calling**. It must not be the source of product policy.

Recommended separation:

```go
type Identity struct {
    SubjectID  string
    ClientID   string
    UserID     string
    AuthMethod string
    TokenID    string
}

type EntitlementContext struct {
    LeaseID          string
    LicenseID        string
    LicenseClass     string
    Status           string
    AllowedProducts  map[string]bool
    Features         map[string]bool
    Limits           map[string]Limit
    NodeID            string
    IssuedAt          time.Time
    RefreshAfter      time.Time
    OfflineValidUntil time.Time
    ExpiresAt         *time.Time
    Sequence          uint64
}
```

Rules:

1. Authentication can succeed while entitlement fails.
2. `UIAI_LOCAL_API_TOKEN` authenticates a trusted local caller but grants no feature.
3. Extension, Pi, MCP, Cockpit, and Focusa child tokens carry explicit scopes derived from a verified parent lease.
4. A token cannot outlive the parent lease or its offline window.
5. Product `uiai-engine` must be present in `AllowedProducts`.
6. Unknown features, tiers, token types, schemas, and states fail closed.
7. Commercial-use posture is a signed claim/contract result, not inferred from a tier string.

## 5. Signed lease and child-token model

UIAI Engine accepts either:

- a locally persisted, authority-signed UIAI Engine entitlement lease; or
- a short-lived child token issued by the local Focusa license broker from a verified bundle lease.

The raw commercial license key is used only to activate/exchange with the authority. It is not forwarded on every UIAI request and is not stored in normal logs or request traces.

A child token should contain or reference:

- issuer and audience (`uiai-engine`);
- lease/license identifiers;
- subject/client identity;
- node/instance identifier;
- exact feature scopes;
- limit bucket identifiers;
- issued/not-before/expiry timestamps;
- parent lease sequence/digest;
- token id for replay/revocation tracking;
- signature or authenticated local-channel proof.

The engine validates both caller authentication and entitlement before route execution.

## 6. Evaluation posture

Evaluation is authority-issued, time-bounded, node/concurrency bounded, and feature-limited. Exact commercial caps and anti-abuse policy remain private and arrive as signed `features` and `limits`.

The public evaluation contract must:

- prove a complete but bounded browser-to-evidence journey;
- allow only an explicit evaluation allowlist;
- expose remaining time and metered capacity without exposing anti-abuse internals;
- make locked capabilities discoverable through metadata and Cockpit cards;
- return stable `license_required`, `evaluation_limit_reached`, or `evaluation_expired` errors with a manage/purchase action;
- preserve local artifacts and diagnostics after expiry;
- prohibit remote API service, hosted/multi-tenant use, team/multi-node operation, redistribution, and embedding unless explicitly signed.

There is no perpetual loopback evaluation mode.

## 7. Route enforcement model

### 7.1 Public/recovery routes

The unauthenticated or recovery-safe set should be narrow:

- `/` — bounded product/version metadata;
- `/health`, `/api/health`, `/api/health/*` — operational health without secrets;
- `/api/license/status` — redacted state and recovery action;
- `/api/license/start` — device-code/evaluation/activation start with rate limiting;
- `/api/license/poll` — one-time device-code polling;
- `/api/license/activate` — activation exchange where supported;
- `/api/license/refresh` — authenticated refresh;
- `/api/license/doctor` — redacted diagnostics;
- `/api/tools` metadata may remain publicly discoverable if it contains no private implementation or callable bypass;
- tokenized public viewers may remain public only when the token contract is separately signed, scoped, expiring, and non-escalating.

### 7.2 Entitlement-required routes

Every route that starts, controls, observes privileged state from, or spends resources on an execution requires an entitlement. This includes, at minimum:

- browser sessions and interactive vision;
- screenshot and comparison execution;
- search and source-to-Markdown execution;
- research packet generation when it invokes or exposes entitled work;
- captcha operations;
- critique, reverse engineering, section detection, layout comparison, style enhancement, and copilot;
- media production and frame execution;
- intake/workflow operations that mutate or trigger work;
- memory writes and commercial retrieval surfaces;
- intelligence/training execution;
- design system, content map, block recipes, comparison, and migration;
- SSE/event streams containing entitled execution state;
- share creation, FPV control, and remote-control surfaces;
- extension token issuance;
- administrative operations beyond license recovery.

Read-only metadata is not automatically safe. Route-by-route review must determine whether it leaks customer data, paid results, prompts, model configuration, or execution state.

### 7.3 Universal middleware order

Recommended order:

```text
request safety / limits
→ caller authentication
→ load and verify canonical lease
→ product grant check
→ route feature check
→ limit reservation
→ handler execution
→ usage/receipt commit or rollback
→ redacted observability
```

Entitlement checks must occur before expensive browser allocation, model calls, media jobs, persistent session creation, or mutating storage.

## 8. Replace tier inference with explicit claims

Remove `tierFeatures()` as an authorization source.

`FromIdentity` should not create entitlements. Instead, a lease verifier returns a validated entitlement context. Feature resolution is direct:

```go
func FeatureEnabled(ent *EntitlementContext, feature string) bool {
    return ent != nil &&
        ent.Status == "active" &&
        ent.AllowedProducts["uiai-engine"] &&
        ent.Features[feature] &&
        ent.TimeValid() &&
        ent.NodeValid()
}
```

Any friendly tier label is presentation only. It cannot grant a capability.

## 9. Limit enforcement

The signed lease references versioned limit buckets. UIAI Engine must support:

- concurrent-resource limits;
- session duration/idle limits;
- per-period and absolute evaluation counters;
- feature-specific preview limits;
- node/instance limits;
- optional authority reconciliation.

Local enforcement must be transactional:

1. reserve capacity before work;
2. attach reservation to request/session/job id;
3. commit actual usage on success;
4. release or classify failed attempts according to signed policy;
5. persist tamper-evident local receipts;
6. reconcile with the authority when online;
7. never extend absolute evaluation expiry from local activity.

The public repository may implement generic counters and receipt verification. Private policy decides exact caps and abuse handling.

## 10. Standalone onboarding

When installed without Focusa, UIAI Engine requires a high-level first-run flow:

1. Start in recovery mode.
2. Open local onboarding URL or print device-code instructions.
3. Choose Evaluate, Activate existing license, or Manage/Purchase.
4. Verify email/account and accept current terms at the authority.
5. Receive and verify a signed `uiai-engine` lease.
6. Register the local engine instance/node.
7. Configure local caller authentication separately.
8. Run health, Chromium/browser, provider, storage, and entitlement checks.
9. Execute one bounded evaluation walkthrough.
10. Display license state, remaining evaluation capacity, locked features, support, and data paths.

The engine must not ask the user to paste a raw key into a web page served by an untrusted remote origin. Browser-based activation should use a one-time device code and a verified authority origin.

## 11. Focusa-brokered bundle onboarding

When installed with Focusa:

```text
Focusa verifies bundle lease
→ Focusa registers/locates UIAI Engine instance
→ Focusa requests or mints scoped child token
→ UIAI Engine verifies audience, signature, parent digest, node, features, time, limits
→ UIAI Engine returns canonical license/health posture
→ Focusa runs bounded proof walkthrough
```

Requirements:

- one evaluator account/email flow, not duplicate signup;
- one authority record may grant both products;
- each product still validates its own grant;
- UIAI Engine never trusts “Focusa is installed” as entitlement;
- Focusa pairing/device tokens cannot be reused as UIAI commercial tokens;
- child-token scopes are narrower than or equal to the parent lease;
- revocation/expiry propagates within a bounded interval;
- status surfaces show the same lease id/sequence and compatible feature digest.

## 12. Cockpit and agent UX

Cockpit, CLI wrappers, Pi/MCP adapters, and agents need one consistent locked-feature response:

```json
{
  "error": "license_required",
  "state": "active_evaluation",
  "product": "uiai-engine",
  "feature": "uiai.analysis.critique",
  "message": "This capability is not included in the current evaluation entitlement.",
  "manage_url": "https://install.focusa.dev/license",
  "retryable": false,
  "remaining": null
}
```

For a metered limit, return the limit bucket, safe remaining value, reset/expiry time, and next action. Never include raw keys, customer records, authority internals, or anti-abuse explanations.

Agent tool discovery may list locked tools, but invocation must fail before side effects. Descriptors should include `license_feature`, evaluation availability, and whether the operation consumes a limit unit.

## 13. Source and packaging boundary

BSL source availability is not a license bypass, but visible client checks can be patched. Strengthen protection by keeping these private/server-side:

- license issuance, signing, evaluator eligibility, anti-abuse, revocation, billing/refund synchronization, and customer records;
- proprietary premium prompts, model-routing policy, assets, or algorithms selected for private distribution;
- hosted orchestration and managed-service internals;
- admin and operational tooling.

Do not rely on a gitignored folder inside the public repository as the durable protection mechanism. Use private repositories/packages, signed release inputs, or server-side APIs with stable public interfaces.

Keep public the verification code, schemas, feature names, stable errors, recovery behavior, and integration contracts needed for transparent client operation.

## 14. Migration plan

1. Add lease verifier, canonical entitlement state, and recovery routes.
2. Stop deriving entitlement from `auth.Identity` tier.
3. Remove unauthenticated loopback evaluation.
4. Make local and extension tokens authentication-only unless accompanied by signed scopes.
5. Add universal route/feature middleware and generated coverage ledger.
6. Add signed limits and usage reservations.
7. Implement standalone device-code onboarding.
8. Implement Focusa child-token integration.
9. Migrate legitimate existing keys to authority-signed leases.
10. Give existing unlicensed evaluators a short activation migration window; do not preserve perpetual free execution.
11. Update `docs/LICENSING.md`, `docs/ENDPOINT_AUTH_MATRIX.md`, quickstarts, examples, and deployment runbooks.
12. Release only after bypass and reverse-proxy tests pass.

## 15. Required tests

- missing identity and missing lease produce recovery-only, not evaluation;
- loopback does not bypass entitlement;
- a reverse-proxied remote request cannot inherit loopback permission;
- local API token authenticates but grants no feature by itself;
- extension token has only signed parent scopes and cannot synthesize `pro`;
- unknown tier/feature/status fails closed;
- edited local lease is rejected;
- wrong-product lease is rejected;
- expired/revoked/offline-window behavior is deterministic;
- child token cannot outlive or exceed parent lease;
- limit reservation is atomic under concurrent requests;
- expensive resources are not allocated before entitlement succeeds;
- every execution/mutation route is present in the generated endpoint-feature ledger;
- public metadata routes contain no private prompts, customer data, or callable side effects;
- share/FPV tokens cannot escalate into normal engine execution;
- raw keys and bearer tokens are absent from logs and observability;
- standalone and bundled onboarding reach the same canonical status;
- data and safe diagnostics remain available after expiry.

## 16. Definition of done

UIAI Engine licensing is closed only when:

```text
verified evaluator/customer identity
+ authority-issued signed uiai-engine grant
+ independent product verification
+ explicit route feature scope
+ transactional limit enforcement
+ bounded refresh/revocation propagation
+ no loopback/local-token entitlement bypass
+ standalone and Focusa-brokered onboarding
+ consistent locked-feature upsell UX
+ data-preserving recovery mode
```

Existing feature middleware is a foundation, not completion, until every execution path participates in this chain.
