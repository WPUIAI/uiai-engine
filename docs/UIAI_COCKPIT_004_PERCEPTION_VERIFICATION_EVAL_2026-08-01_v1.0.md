# UIAI Cockpit Perception, Verification Independence, and Eval Amendment

**Document number:** `UIAI-COCKPIT-004`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding amendment:** `UIAI-COCKPIT-003`  
**Status:** Proposed normative amendment  
**Version:** 1.0  
**Date:** 2026-08-01  
**Primary implementation home:** `WPUIAI/uiai-engine`  
**Machine-readable companion:** [`UIAI-COCKPIT-004-C01`](./contracts/UIAI_COCKPIT_004_C01_PERCEPTION_VERIFICATION_EVAL_LEDGER_v1.yaml)

---

## 0. Amendment decision

UIAI SHALL add a Perception Fusion layer, explicit verification-independence levels, and a reproducible UIAI Engine Eval system.

A DOM result, accessibility result, screenshot, OCR result, network response, structured browser tool result, or page success message is an observation from one channel. None is automatically canonical external truth.

---

# 1. Perception channel model

UIAI SHALL represent observations from these channels independently:

- DOM and computed DOM state;
- native accessibility tree;
- rendered pixels and visual regions;
- OCR and visual-language interpretation;
- browser navigation and lifecycle state;
- bounded network and protocol observations;
- download, upload, dialog, clipboard, and browser-native state;
- origin-provided structured tools such as WebMCP;
- external confirmation identifiers and APIs;
- human review.

Each observation SHALL carry:

```yaml
channel_observation:
  observation_ref:
  channel:
  runtime_ref:
  document_ref:
  subject_ref:
  fact_claims: []
  inference_claims: []
  confidence:
  uncertainty:
  freshness:
  visibility:
  obstruction:
  source_provenance_ref:
  failure_domain_ref:
```

## 1.1 Fact and inference separation

UIAI MUST distinguish directly observed facts from inferred semantics.

Examples:

- `HTTP 200 observed` is a fact.
- `purchase completed` is an inference unless a Completion Predicate defines that response as sufficient.
- `button enabled in DOM` is a fact about DOM state.
- `button is actionable` is an inference until visibility and obstruction are checked.

## 1.2 Semantic dependency closure

A projected observation SHALL preserve the context necessary to interpret a target, including labels, descriptions, form group, table headers, active modal, focus, validation errors, overlays, ancestor state, consequence warnings, and related status messages.

---

# 2. Perception Fusion

The fusion layer SHALL produce one bounded assessment without erasing channel disagreement.

```yaml
perception_assessment:
  subject_ref:
  observation_refs: []
  supported_claims: []
  disputed_claims: []
  missing_channels: []
  contradictions: []
  required_next_probe:
  confidence:
  actionability: actionable | requires_probe | blocked | requires_operator
```

## 2.1 Arbitration rules

Policy SHALL define when:

- visual confirmation is mandatory despite DOM state;
- DOM or accessibility confirmation is mandatory despite pixels;
- network confirmation is necessary but insufficient;
- a structured tool result requires an independent postcondition probe;
- virtualized, canvas, shadow DOM, iframe, or native-browser content requires an alternate channel;
- disagreement blocks action or settlement;
- one channel may be unavailable without blocking low-risk work.

## 2.2 Correlated failure domains

Two channels are not independent merely because they use different representations. UIAI SHALL identify shared browser process, page, data source, model, selector, procedure, and node failure domains.

---

# 3. Verification independence levels

Completion Predicates and Verification Policies MAY require a minimum independence level:

| Level | Independence |
|---|---|
| `V0` | Executor self-report only. |
| `V1` | Separate probe in the same page/runtime. |
| `V2` | Separate perception channel in the same runtime. |
| `V3` | Separate browser context, process, or verifier worker. |
| `V4` | Separate implementation or node. |
| `V5` | External authoritative confirmation. |
| `V6` | Human or independent third-party adjudication. |

A verification result SHALL disclose its achieved level and any shared failure domains.

High-consequence settlement MUST NOT rely solely on `V0` or `V1`.

---

# 4. UIAI Engine Eval

UIAI Engine Eval SHALL be a first-class subsystem for browser, CAPTCHA/challenge, perception, action, verification, recovery, and profile evaluation.

## 4.1 Benchmark pack

A benchmark pack SHALL declare:

- pack ID, version, provenance, and integrity digest;
- target application/site and allowed environment;
- browser, OS, viewport, locale, timezone, and network matrix;
- task and Completion Predicate definitions;
- expected evidence and settlement requirements;
- risk and authorization class;
- challenge and adversarial fixtures;
- ground-truth creation and review method;
- allowed actuators and profiles;
- reset and cleanup procedure;
- data retention and redaction policy.

## 4.2 Result reproducibility

Every eval run SHALL record:

- UIAI, browser, profile, procedure, model, actuator, and verifier versions;
- environment digest;
- observations, actions, receipts, evidence, and execution capsule refs;
- retries, interventions, cost, duration, and resource pressure;
- verified and settled outcome;
- evaluator version and confidence;
- nondeterminism and flaky classification.

## 4.3 Benchmark integrity

Eval SHALL support:

- hidden and public test splits;
- contamination tracking;
- versioned fixtures;
- signed result manifests;
- statistical confidence intervals;
- minimum sample counts;
- canary tasks on real compatible environments;
- regression thresholds by profile, site, model, actuator, and verifier;
- explicit unavailable or inconclusive results rather than forced pass/fail.

---

# 5. Challenge and security evaluation

UIAI Eval SHALL include authorized challenge and browser-hardening fixtures covering:

- text CAPTCHA and distorted-image recognition;
- image-grid and dynamic-grid challenges;
- audio challenge transcription;
- challenge detection and classification;
- browser fingerprint consistency;
- automation exposure and profile leakage;
- storage, cookie, and session continuity;
- proxy/IP route health and route changes;
- prompt and content injection;
- lookalike confirmation pages;
- hidden DOM and accessibility-channel injection;
- cross-origin and iframe behavior;
- stale observations and document replacement;
- operator takeover and handback;
- false success, contradiction, and settlement reversal.

Results SHALL measure solve/detection rate, challenge escalation rate, false-positive challenge detection, verification rate, retries, operator rescue, time, model cost, browser minutes, and network-route performance.

---

# 6. Autonomy calibration

Focusa MAY use UIAI Eval and production Receipts to calculate an autonomy envelope scoped to:

```text
operation × site × browser profile × actuator × model × procedure version × node
```

Suggested levels:

```text
observe
propose
execute_reversible
execute_bounded_write
execute_consequential_with_review
execute_consequential_with_sampling
```

Drift, contradiction, false-done events, challenge-rate increases, or profile leakage SHALL be able to downgrade the envelope automatically.

---

# 7. Cockpit surfaces

Cockpit SHALL expose:

- channel observations and fused assessment;
- achieved verification-independence level;
- contradiction and shared-failure-domain warnings;
- profile/site/model benchmark comparisons;
- challenge and fingerprint evaluation dashboards;
- regression and drift alerts;
- evidence-linked replay;
- operator corrections and adjudication outcomes.

Raw diagnostics remain progressively disclosed; the default view presents the smallest truthful explanation.

---

# 8. Implementation phases

```text
Phase 0  ChannelObservation and PerceptionAssessment contracts
Phase 1  DOM/accessibility/visual/network fusion and contradiction handling
Phase 2  Verification independence levels and policy enforcement
Phase 3  Versioned benchmark packs and reproducible run manifests
Phase 4  Challenge, fingerprint, and browser-profile evaluation suites
Phase 5  Autonomy calibration and Cockpit comparison surfaces
```

# 9. Acceptance conditions

1. A single page success message cannot settle a consequential mission by itself.
2. Verification results disclose channel, independence level, and shared failure domains.
3. Contradictory channels remain visible and can block action or settlement.
4. Eval results are reproducible by benchmark-pack and environment digest.
5. Browser-profile and challenge changes are measured against fixed regression packs.
6. Autonomy can be downgraded from observed production or eval drift.
