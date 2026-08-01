# UIAI Engine Protected Worker and Feature Capsule Addendum

**Status:** Proposed — companion to the mandatory entitlement/onboarding specification  
**Created:** 2026-08-01  
**Shared architecture:** `Startempire-Wire/focusa/docs/152a-protected-distribution-private-feature-capsules-and-anti-tamper-spec.md`

## Decision

UIAI Engine should not rely on public Go middleware as the only barrier around commercially valuable execution. Selected crown-jewel route implementations must move behind independently signed protected workers or hosted services distributed from private source.

Changing `RequireFeature` or authentication middleware in the public gateway must not produce the protected worker, decrypt its feature capsule, mint an accepted operation token, or reconstruct missing proprietary behavior.

## Recommended split

### Public gateway

Retain publicly reviewable:

- health/version and license recovery;
- signed-lease and child-token verification contracts;
- public-safe tool metadata;
- configuration and diagnostics;
- local data formats and operator-owned artifact access;
- strict IPC schemas;
- bounded tokenized viewers that cannot escalate;
- recovery/export/uninstall behavior.

### Protected worker

Move selected valuable implementations to a private worker distribution, prioritizing:

- persistent browser execution and control;
- premium critique/reverse/layout/style/copilot operations;
- proprietary prompts, model routing, ranking, and transformation policy;
- premium media generation;
- sensitive multi-browser/pool orchestration;
- commercial remote API and managed-service functions;
- other code whose presence in the public tree would make a license patch sufficient.

The exact split remains private and may evolve by release.

## IPC contract

The gateway sends a narrow signed operation capability containing:

- audience and worker id;
- feature key;
- parent lease id and sequence/digest;
- account/node/client identifiers in opaque form;
- request/job/session id;
- not-before and short expiry;
- allowed action and resource scope;
- limit reservation id;
- idempotency key;
- nonce/token id.

The worker independently verifies the operation capability and compatible signed lease state. It rejects arbitrary commands, shell execution, unscoped URLs/resources, stale/replayed tokens, wrong audience, wrong node, missing reservation, and unsupported contract versions.

## Capsule delivery

Official UIAI workers and sensitive assets ship as signed feature capsules. The authority delivers a node-bound content-key envelope only when the signed lease grants `uiai-engine` and the required feature set.

Where practical:

- device keys are hardware/OS protected;
- capsules remain encrypted at rest;
- plaintext extraction is ephemeral and permission-restricted;
- worker/capsule versions are atomic and rollback-safe;
- copied capsules cannot unwrap on a different registered node;
- modified or unofficial gateway/worker identities are ineligible for protected updates or keys;
- a safe reduced-assurance fallback remains explicit for unsupported hardware.

Encryption is a cost raiser, not a claim that native code cannot be recovered from memory.

## Route migration priority

1. Add universal entitlement middleware and route coverage ledger.
2. Establish recovery-only gateway operation.
3. Create protected worker IPC and synthetic worker fixture.
4. Move one bounded high-value route family end-to-end.
5. Add node-bound capsule delivery and operation-token verification.
6. Move persistent browser/session execution where product UX remains acceptable.
7. Move proprietary AI policy and premium operations.
8. Add adversarial bypass tests before expanding the split.

## Local-first boundary

Protected distribution must not turn UIAI Engine into covert cloud execution. Browser state, screenshots, source content, prompts, and user artifacts remain local unless a specific hosted feature clearly requires and obtains consent for bounded transfer.

## Development-agent boundary

General repository agents may work on public gateway contracts and synthetic workers. Access to private workers, authority signing, production data, and module-key delivery uses separate credentials and audited tools. No general coding agent can mint a production-trusted lease or protected capsule.

## Required proof

- public gateway patched to allow does not execute absent protected routes;
- direct IPC without a signed operation capability fails;
- replayed or broadened capabilities fail;
- copied capsule fails on another node in bound modes;
- wrong-product, expired, revoked, stale-sequence, and exhausted-limit cases fail before browser/model allocation;
- gateway/worker/capsule downgrade and mixed versions fail safely;
- recovery, local artifact access, and uninstall remain available;
- release logs and support bundles redact leases, keys, tokens, and capsule material.
