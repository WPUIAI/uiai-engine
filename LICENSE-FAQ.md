# UIAI Engine License FAQ

## Is UIAI Engine open source?

No. UIAI Engine is source-available under the Business Source License terms in `LICENSE`. It becomes available under the named Change License only for a version when that version reaches its Change Date.

## Can I inspect, clone, or build the source?

Use of the repository source is governed by `LICENSE` and its Additional Use Grant. Source visibility, cloning, and a successful build do not themselves create an official UIAI Engine product entitlement, commercial rights, protected-worker access, support, or official updates.

## Can I evaluate UIAI Engine?

Yes, through an **authority-issued Evaluation license** under the mandatory entitlement architecture.

The required official flow is:

```text
verified account/email and current terms
→ explicit uiai-engine Evaluation product/features/limits
→ node registration
→ authority-signed lease
→ independent engine verification
→ bounded evaluation workflow
```

Evaluation is time-, node-, concurrency-, feature-, and usage-bounded. Missing identity or license state is recovery-only, not Evaluation.

The current repository still contains legacy code that can infer Evaluation from missing/loopback state. That behavior is release-blocked and must not be used to onboard new evaluators.

## Can I generate a local evaluation key?

No approved production/evaluator flow uses locally generated `uiai_eval_` keys or any other self-issued key.

A local API token, extension token, Pi/MCP/Cockpit token, Focusa pairing token, loopback address, tier string, source checkout, or health response authenticates or describes a surface; it does not create a product grant.

Test fixtures may use a separate test trust root in isolated development tests. Production artifacts must reject those fixtures and test roots.

## Do loopback browser/session/search/Markdown routes remain free without entitlement?

No, not in the target production model. Loopback affects network authentication risk only. Every execution-capable route must still verify an authority-issued signed `uiai-engine` lease or a narrower child token, product/feature/time/node/sequence status, and applicable limits.

Current loopback-public behavior is an implementation migration issue documented in `docs/ENDPOINT_AUTH_MATRIX.md`.

## What is the Additional Use Grant?

The Additional Use Grant defines legal source-use permission under `LICENSE`. It is distinct from official product activation.

It does not:

- mint an Evaluation lease;
- create protected private workers or capsules;
- grant hosted/commercial/team/embedding rights;
- bypass route entitlement enforcement;
- grant support, official updates, or authority services to a modified fork.

## When do I need a commercial license?

A commercial license is required when the repository license or Additional Use Grant does not authorize the intended use, including applicable production, company/team, paid service, hosted/SaaS, resale, redistribution, embedding, managed automation, or commercial proof-browser use.

The commercial agreement must explicitly grant the intended products, features, nodes/seats, update channel, support, hosted/embedding/redistribution rights, and protected components.

## What does a commercial license include?

Depending on the agreement:

- single-operator commercial use;
- team or enterprise use;
- hosted or managed-service rights;
- redistribution/resale/embedding rights;
- private/on-prem/air-gapped deployment;
- protected workers or feature capsules;
- support and update terms;
- node/seat limits;
- offline and renewal policy.

A friendly tier label is presentation only. Runtime permission comes from explicit signed product and feature grants.

## What about Founders Forge?

Eligible Founders Forge participants may receive authority-issued UIAI Engine grants under the applicable program agreement. Participation labels alone do not unlock runtime features.

## What happens when Evaluation expires or a license is revoked?

UIAI Engine enters recovery-only posture. It preserves safe health/license diagnostics, operator-owned data/artifacts, applicable export, recovery, and uninstall guidance. It denies new browser, analysis, media, migration, remote-control, and other execution.

License failure must not delete or encrypt operator data.

## Can Focusa grant UIAI access?

Only when the parent Focusa/bundle license explicitly includes `uiai-engine`. Focusa may broker a short-lived, audience-bound child token no broader or longer-lived than the parent grant. UIAI independently verifies product, feature, parent lease sequence/digest, node, expiry, and limits.

A Focusa pairing or project-scope token is not a UIAI commercial token.

## Can I fork UIAI Engine and remove the checks?

Forking remains governed by `LICENSE`; it does not remove legal restrictions. Modified forks are unsupported and cannot obtain official protected workers, node-bound capsule keys, signed operation capabilities, authority services, or official updates merely by returning an “allowed” result.

Client-side controls are cost escalation, not a claim of unbreakable DRM.

## How do I get or manage a license?

Use the official UIAI/Focusa license-management, purchase, or support channel published at the install/license site. Do not send raw keys, tokens, device codes, customer records, or secrets through public issues or chat.

## Canonical references

- `LICENSE`
- `docs/LICENSING.md`
- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
- `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`
- `docs/ENDPOINT_AUTH_MATRIX.md`
- `docs/UIAI_ENTITLEMENT_SUPERSESSION_MATRIX_2026-08-01.yaml`
