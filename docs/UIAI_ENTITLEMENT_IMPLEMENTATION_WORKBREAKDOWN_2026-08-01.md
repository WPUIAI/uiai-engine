# UIAI Engine Entitlement Implementation Work Breakdown

**Status:** P0 implementation plan  
**Tracks:** `WPUIAI/uiai-engine#5`  
**Authority dependency:** `WPUIAI/wpuiai#1`  
**Focusa bundle dependency:** `Startempire-Wire/focusa#119`

## WP0 — Inventory and fixture isolation

- Inventory every route, middleware exception, caller credential, token issuer, browser allocation, model/provider call, job/session/share/control mutation, and public payload.
- Generate the initial endpoint-method-handler-auth-feature-limit-worker ledger.
- Separate test trust roots/fixture statuses from production builds.

Acceptance: no unclassified execution/mutation route and production fixture rejection.

## WP1 — Entitlement context and lease verifier

- Remove entitlement derivation from `auth.Identity`.
- Implement canonical signed lease verification shared with Focusa/authority golden vectors.
- Validate `uiai-engine` product, status, node, sequence, key id, time/offline/expiry, explicit features, and limits.
- Persist only verified lease/refresh metadata; no raw commercial key in normal runtime.

Acceptance: forged, edited, wrong-product, stale, expired, revoked, unknown-schema/key tests.

## WP2 — Authentication-only identities

Refactor:

- local API token => authenticated local caller only;
- extension/Pi/MCP/Cockpit token => explicit signed scopes only;
- webhook/service token => narrow service role and product operation;
- bearer/API key => caller identity plus separate entitlement lookup;
- remove hard-coded `internal`/`pro` authorization.

Acceptance: authenticated caller without lease is denied execution.

## WP3 — Universal gate before allocation

Install ordered middleware:

```text
request safety
→ authentication
→ lease/child token
→ product
→ feature
→ node/time/sequence
→ limit reservation
→ handler/protected worker
→ usage commit/release
```

Gate session, screenshot, search, Markdown, research packet, diagnostics with private data, captcha, analysis, media, workflow, memory, intelligence, training, migration, events, vision, share creation/control, extension issuance, and admin operations.

Acceptance: no browser/model/job/storage allocation before successful decision.

## WP4 — Route splitting and ownership

- Split public metadata from execution/mutation handlers.
- Scope session/job/usage/error/share/event reads to caller/lease/resource.
- Redesign FPV viewer/control tokens for no escalation.
- Audit health/tools/models payloads for private prompts, keys, customer state, or callable side effects.

Acceptance: enumeration and share-escalation negative tests.

## WP5 — Limit reservations

- Implement concurrency, duration/idle, per-period, absolute Evaluation, and feature-preview buckets.
- Reserve atomically; bind to request/session/job and idempotency key.
- Commit actual use/release failures according to signed policy.
- Reconcile tamper-evident receipts without customer content.

Acceptance: parallel race tests cannot exceed limits; retries are idempotent.

## WP6 — Standalone onboarding/recovery

- Recovery-mode status and device-code start/poll/activate/refresh/doctor routes.
- Verified authority origin and account/email flow.
- Register node and verify signed lease.
- Configure caller authentication separately.
- Bounded first browser/proof walkthrough.

Acceptance: no lease => recovery-only; Evaluation/paid activation without reinstall.

## WP7 — Focusa bundle child tokens

- Verify explicit UIAI product in parent lease.
- Accept audience-bound token carrying parent id/sequence/digest, node/client, exact features/limits, short expiry, nonce/token id.
- Reject pairing/project/local tokens as commercial credentials.
- Propagate revoke/expire/refund/replace.

Acceptance: child token cannot exceed/outlive parent and replay/wrong-audience/stale-parent fail.

## WP8 — Protected worker migration

- Define narrow IPC and worker identity.
- Move one bounded high-value family to private signed worker/capsule.
- Require operation capability and node-bound key envelope.
- Preserve public gateway recovery and operator-owned artifacts.

Acceptance: patched gateway alone cannot execute feature; direct/replay/copy/substitution/downgrade tests.

## WP9 — Interface/docs parity

- Generate license feature/limit metadata for HTTP, Pi, MCP, CLI, Cockpit, and OpenAI schemas.
- Use the same stable denial envelope.
- Update route matrix/docs in the same change.
- Keep `scripts/check-license-doc-consistency.py` green.

Acceptance: machine ledger equals route mounts/tool schemas/docs.

## WP10 — Migration and release

- Exchange legitimate current keys for signed leases.
- Eliminate local `uiai_eval_` and anonymous loopback Evaluation.
- Preserve local artifacts and recovery after expiry.
- Run standalone/bundle, proxy-loopback, refund/revoke, outage/offline, node and limit E2E.
- Reject test roots/fixture status in release artifacts.

Do not claim evaluator/customer readiness until authority, UIAI, and Focusa bundle receipts reconcile.
