# UIAI Engine Licensing

UIAI Engine is commercially valuable infrastructure and the proof browser for Focusa-powered agents. The repository is source-available under the [Business Source License 1.1](../LICENSE) with a commercial licensing path.

This page is an operational summary, not legal advice. The repository `LICENSE` controls source-use rights. The mandatory entitlement/onboarding specification controls official runtime activation, feature execution, and Evaluation issuance.

## Two boundaries that must not be conflated

### 1. Source-use license

The BSL and Additional Use Grant define what a recipient may legally do with repository source and versions subject to those terms.

### 2. Product entitlement

Every official UIAI Engine runtime, including Evaluation, must receive an authority-issued signed entitlement before browser, analysis, media, migration, remote-control, or other execution-capable features run.

Source visibility, a successful build, loopback origin, missing identity, local API token, extension token, Focusa pairing token, health response, or tier string is not a product entitlement.

The current Go implementation still contains legacy loopback Evaluation and tier-derived feature behavior. New evaluator/customer distribution is release-blocked until the mandatory-entitlement specification is implemented.

## License model

UIAI Engine uses:

- **Business Source License 1.1:** source-available license for repository versions.
- **Additional Use Grant:** legal permission for the uses stated in `LICENSE`; it does not synthesize a signed product lease.
- **Authority-issued Evaluation:** verified account/email, time-bounded signed product/features/limits, node/concurrency controls, and data-preserving expiry.
- **Founders Forge grant:** authority-issued product/features tied to eligible participation; participation labels alone do not unlock runtime.
- **Commercial license:** required for commercial production, hosted/SaaS, resale, paid service, commercial browser automation, commercial agent infrastructure, or paid product-feature use outside the BSL terms.
- **Developer license:** authority-issued private development entitlement; never inferred from repository access or environment variables.
- **Change License:** the license identified in `LICENSE` on the applicable Change Date for that version.

## Evaluation requirements

Evaluation is a real license class, not absence of a license.

Required flow:

```text
standalone UIAI or Focusa bundle onboarding
→ verified authority origin/device code
→ verified account/email and current terms
→ optional promotional consent recorded separately
→ Evaluation license and node activation
→ signed uiai-engine product/features/limits lease
→ independent engine verification
→ bounded walkthrough
```

An Evaluation lease must specify:

- `uiai-engine` product grant;
- explicit features;
- absolute expiry;
- refresh/offline window;
- node and concurrency constraints;
- signed limit policy/version;
- lease id, sequence, authority key id, and signature;
- recovery/manage-license actions.

After expiry/revocation, UIAI Engine preserves safe diagnostics, license recovery, operator-owned data/artifacts, export where applicable, and uninstall guidance. It denies new execution.

## What requires a commercial or other explicit product grant

A separate applicable grant is required for:

- running UIAI Engine as part of a paid product or paid feature;
- hosted service, managed service, API service, or SaaS;
- resale, sublicensing, paid distribution, or commercial automation;
- commercial production use outside the Additional Use Grant;
- team/multi-node, remote-service, product-embedding, protected-worker, or premium features not present in Evaluation;
- protected workers/capsules distributed from private source.

Removing or obscuring license notices remains subject to the repository license and commercial terms.

## What the public repository provides

The public repository may provide:

- public gateway and recovery surfaces;
- stable request/response/IPC schemas;
- signed-lease and child-token verification contracts;
- health, diagnostics, configuration, and public-safe metadata;
- tests and development fixtures under a separate test trust root;
- operator-owned artifact formats and recovery behavior.

Selected crown-jewel implementations may move to privately built, signed, encrypted feature capsules or hosted services. Patching public middleware must not create those missing capabilities.

## Authentication is not entitlement

- `UIAI_LOCAL_API_TOKEN` authenticates a local caller only.
- Extension/Pi/MCP/Cockpit tokens must contain explicit scopes derived from a verified parent lease.
- Focusa child tokens must be short-lived, audience-bound, feature-limited, node-bound, and no broader than the parent lease.
- Loopback changes network-authentication risk only; it never grants Evaluation.
- A healthy gateway or browser pool never proves product permission.

## Open-source status

UIAI Engine is source-available under BSL 1.1. It is not open source until the Change License takes effect for the applicable version.

## Current implementation status

Current code and the pre-Spec-152 endpoint matrix document legacy behavior for migration and testing. They must not be interpreted as the target entitlement contract.

Truthful current claim:

```text
route/auth and feature-middleware foundations exist;
mandatory authority-issued entitlement and universal execution coverage remain release blockers
```

## Canonical references

- `docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`
- `docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`
- Focusa Spec 152 and Spec 150A for bundled onboarding
- `docs/ENDPOINT_AUTH_MATRIX.md` for current as-built auth behavior and migration warning

## Contact

Use the published UIAI/Focusa commercial, support, or license-management channel for commercial licensing, Evaluation, partnership terms, cohort access, reissue, refund, or node-management questions.
