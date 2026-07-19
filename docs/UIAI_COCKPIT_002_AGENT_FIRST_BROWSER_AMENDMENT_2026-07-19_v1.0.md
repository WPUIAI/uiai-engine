# UIAI Cockpit Agent-First Browser Amendment

**Document number:** `UIAI-COCKPIT-002`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding numbered decision:** `UIAI-COCKPIT-001`  
**Status:** Proposed normative amendment  
**Version:** 1.0  
**Date:** 2026-07-19  
**Amends:** [`UIAI-COCKPIT-000`](./UIAI_COCKPIT_000_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md)  
**Companion authority:** Focusa Spec 135 series (`135` through `135K`) and the Spec 135 Current Authoritative Delivery Contract  
**Primary implementation home:** `WPUIAI/uiai-engine`  
**Machine-readable companion:** [`UIAI-COCKPIT-002-C01`](./contracts/UIAI_COCKPIT_002_C01_AGENT_FIRST_BROWSER_CONTRACT_LEDGER_v1.yaml)

---

## 0. Amendment decision

UIAI Engine Cockpit SHALL adopt one **agent-first browser exchange and execution layer** that makes browser work machine-readable, compact for ordinary agent loops, expandable by stable reference, observation-versioned, stale-safe, provenance-aware, policy-bound, predicate-verifiable, settlement-capable, and reconstructable through Evidence and Receipts.

This amendment adds no new top-level workspace, canonical mission store, event history, Evidence store, procedure authority, browser runtime, or task authority.

```text
Focusa
  Mission Kernel · Workpoint · authority · operation registry · predicates
  verification policy · Evidence linkage · Receipts · settlement · memory

UIAI Engine
  browser contexts/targets · observations · actions · diagnostics · artifacts
  runtime verification · settlement observation · browser proof · Eval

UIAI Cockpit
  operator experience · oversight · approval · inspection · takeover · review
```

UIAI SHALL consume Focusa's immutable generated contract bundle by commit SHA or release digest. UIAI MUST NOT hand-maintain duplicate Focusa DTOs, operation metadata, permission rules, or action registries when generated contracts can represent them.

### 0.1 Numbering and stable requirement identity

This amendment is `UIAI-COCKPIT-002`. Its machine-readable companion is `UIAI-COCKPIT-002-C01`. Requirement IDs in the companion ledger resolve fully as `UIAI-COCKPIT-002/<local-requirement-id>`. Future machine-readable companions increment `C02`, `C03`, and so on without renumbering stable requirements.

---

# 1. Agent-first objective

The default agent interaction SHALL be:

```text
discover outcome capability
→ receive compact scoped observation
→ propose typed action
→ pass deterministic authority and influence checks
→ execute against the observed version
→ receive compact state delta
→ verify explicit predicate
→ wait for settlement when required
→ expand Evidence, provenance, diagnostics, or trace only on demand
```

The optimization objective is:

> **Lowest total cost per correctly authorized, externally verified, evidence-supported, and—where required—settled outcome.**

Raw snapshot size, tool-call count, or model-token count SHALL NOT be treated as sufficient success metrics by themselves.

---

# 2. Ownership boundaries

## 2.1 Focusa owns

- canonical Mission, Workpoint, Trajectory, task, authority, capability and permission state;
- the generated Focusa Operation Registry;
- Completion Predicates and Verification Policies;
- Evidence linkage, Receipts, contradiction handling, completion and settlement;
- durable procedural-memory promotion, scope, retrieval, demotion, and revocation;
- canonical event history and replay.

## 2.2 UIAI Engine owns

- browser process, session, context/container, target/tab, page and frame execution;
- browser observations, runtime references and stale-state detection;
- DOM, native accessibility, visual and bounded network perception;
- browser actions and origin-bound structured browser actuators;
- screenshots, recordings, diagnostics, diffs and browser artifacts;
- browser verification probes and leased settlement watchers;
- browser execution capsules and low-level causality artifacts;
- UIAI Engine Eval.

## 2.3 Cockpit owns presentation and intent

Cockpit may render, request, approve, steer, pause, stop, inspect, compare, recapture, reverify, share and export. It MUST NOT become canonical mission truth, action authority, settlement authority, or durable browser memory.

---

# 3. Shared Agent Exchange Protocol

All new browser capabilities SHALL use one shared exchange pattern instead of five unrelated tool families.

## 3.1 Registry split

**Focusa Operation Registry** describes outcome-level operations, exact scope, capabilities, permissions, preview/commit posture, idempotency, concurrency, Evidence and Receipt requirements.

**UIAI Capability Registry** describes browser-native execution capabilities, supported actuators, risk posture, required context, evidence quality, cost posture, compatibility and implementation status.

An `ActuatorRef` binds a Focusa operation or Action Proposal to an eligible UIAI capability. UIAI SHALL NOT copy a Focusa operation into its own canonical operation registry.

## 3.2 Lazy discovery

```text
compact agent card
→ outcome/capability search
→ bounded ranked summaries
→ inspect one selected schema
→ execute
```

Ordinary agent context MUST NOT preload the complete UIAI catalog, every Focusa operation, every procedure, every WebMCP tool, or every detailed schema.

## 3.3 Common result envelope

```yaml
schema: uiai.agent_result.v1
status: ok | blocked | stale | inconclusive | pending | failed | requires_operator | resync_required
failure_class:
summary:
canonical: false
degraded: false
scope_ref:
runtime_ref:
payload:
  kind:
  compact:
  payload_ref:
artifact_refs: []
evidence_candidate_refs: []
receipt_ref:
execution_capsule_ref:
freshness:
uncertainty:
retry:
recovery:
next_actions: []
usage:
correlation_id:
causation_id:
```

Specialized payloads remain typed, but common status, recovery, Evidence, usage and causality fields MUST NOT be redundantly redefined.

## 3.4 Response profiles

Every request MAY declare:

```text
agent_compact
agent_standard
evidence_grade
developer_full
```

Every result MUST return:

```yaml
requested_profile:
effective_profile:
minimum_policy_profile:
profile_upgrade_reasons: []
```

Policy may upgrade the effective profile. A caller MUST NOT reduce the result below the minimum required by risk class, Capability Grant, Completion Predicate, evidence requirement, failure condition or operator-control state.

- `agent_compact`: status, material changes, freshness, uncertainty, refs, next actions and usage.
- `agent_standard`: bounded preconditions, selected observation context, verification-channel summary and compact provenance.
- `evidence_grade`: immutable before/after refs, predicate linkage, integrity, detailed lineage and capsule linkage.
- `developer_full`: full redacted diagnostics, protocol details, traces and expandable raw envelopes.

## 3.5 Agent client capability profile

```yaml
schema: uiai.agent_client_capability_profile.v1
client_id:
supported_schema_versions: []
supports:
  structured_content:
  artifact_handles:
  semantic_deltas:
  image_content_items:
  streaming_updates:
  schema_refs:
  continuation_refs:
budgets:
  max_tool_result_tokens:
  max_inline_nodes:
  max_inline_evidence_refs:
  max_tool_schemas_per_discovery:
  max_image_items:
preferences:
  response_profile:
  preferred_representation:
  expand_details_on_failure_only:
```

UIAI SHALL select the smallest truthful representation supported by the client.

---

# 4. Versioned Browser Observation and Action Consistency

UIAI SHALL replace snapshot-local action assumptions with versioned observations and observation-bound actions.

## 4.1 Observation identity

```yaml
schema: uiai.browser_observation.v1
observation_id:
parent_observation_id:
observation_sequence:
runtime:
  uiai_session_id:
  browser_context_id:
  target_id:
document:
  document_id:
  navigation_id:
  url:
  origin:
  lifecycle_state:
projection:
  representation: []
  projection_policy:
  query_hash:
  budget:
frames: []
global_state:
element_refs: []
semantic_hash:
redacted_projection_hash:
captured_at:
freshness:
```

## 4.2 Element reference model

A reference MUST keep separate:

1. runtime identity — context, target, document, navigation, frame, native AX node and backend DOM identity where available;
2. locator candidates — stable ID/test attribute, role/name, labels, structural locator and visual anchor;
3. state fingerprint — role, name, value, state, visibility, bounding region, associated error/warning and relevant ancestor context.

Runtime identifiers are document-scoped and MUST NOT be described as durable across document replacement.

## 4.3 Observation-bound action

```yaml
schema: uiai.browser_action_request.v2
action_id:
focusa_action_proposal_ref:
capability_grant_ref:
runtime_ref:
expected_observation:
  observation_id:
  document_id:
  navigation_id:
  frame_id:
target:
  element_ref:
  expected_fingerprint:
  fallback_locators: []
preconditions: []
action:
  kind:
  parameters:
response_policy:
```

UIAI MUST revalidate at the action boundary. Navigation, document replacement, frame replacement, ref invalidation, changed preconditions, obstruction or materially changed identity SHALL fail closed with `stale` or `resync_required`.

## 4.4 Semantic delta states

```text
full
delta
unchanged
baseline_unknown
baseline_expired
document_replaced
frame_topology_changed
projection_changed
resync_required
```

A delta MUST identify its baseline. Relevance projections MUST preserve labels, descriptions, form/fieldset context, table headers, active modal, focus, validation errors, obstruction and consequence warnings.

---

# 5. Focusa-Directed Verification Probes and Settlement Watchers

UIAI SHALL execute browser-native verification requests compiled from Focusa Completion Predicates and Verification Policies. UIAI verifies browser-observable external state; Focusa evaluates sufficiency, contradictions, completion and settlement.

## 5.1 Verification request

```yaml
schema: uiai.focusa_browser_verification_request.v1
verification_request_id:
mission_ref:
workpoint_ref:
completion_predicate_ref:
verification_policy_ref:
scope_ref:
runtime_ref:
structured_conditions: []
permitted_channels: []
required_evidence_kinds: []
independence_requirement:
watch:
  mode: once | until_change | until_verified | until_deadline
  deadline:
  maximum_observations:
  maximum_browser_minutes:
  minimum_interval:
  settlement_window:
```

Supported channels may include native accessibility, DOM, visual, navigation, bounded network confirmation, download state, dialog state, external confirmation identifiers and approved human review.

## 5.2 Verification result

```yaml
schema: uiai.focusa_browser_verification_result.v1
verification_result_id:
verification_request_id:
completion_predicate_ref:
verification_policy_ref:
outcome: passed | failed | inconclusive | stale | blocked | settlement_pending
channel_results: []
observation_refs: []
artifact_refs: []
evidence_candidate_refs: []
external_confirmation_refs: []
contradiction_refs: []
freshness:
watch_state_ref:
uncertainty:
cost:
duration:
```

## 5.3 Settlement watcher

A watcher SHALL be server-side, bounded, leased, read-only by default, predicate-scoped, event-driven where possible, budget-limited, observable and cancellable. The model MUST NOT be required to poll repeatedly. A watcher supplies candidate verification to Focusa; it does not settle the mission.

---

# 6. Browser Content Provenance and Influence Firewall

UIAI SHALL separate trusted control from untrusted browser data and enforce the separation outside the model.

Trusted control may include Focusa Mission/Workpoint refs, Capability Grants, Action Proposals, Completion Predicates, approved operator instructions, registered Focusa operations and UIAI policy.

Untrusted data includes page text, hidden DOM, accessibility labels, images/OCR, search results, ads, frames, tool descriptions/outputs, WebMCP metadata/results, downloads and remote responses.

Untrusted content may inform an action but MUST NOT expand authority, origin access, data disclosure, financial limits, Evidence suppression, credential access, procedure promotion or completion state.

```yaml
schema: uiai.browser_content_provenance.v1
content_ref:
observation_ref:
source:
  origin:
  frame_id:
  target_id:
  backend_node_id:
  ax_node_id:
channel:
visibility:
trust:
  class:
  instruction_like:
  untrusted_content:
data:
  classifications: []
  permitted_egress: []
integrity:
  source_hash:
  captured_at:
```

Provenance SHOULD use shared dictionaries, ranges, chunks or handles rather than repeating full metadata.

Before a consequential action, UIAI SHALL retain:

```yaml
schema: uiai.browser_action_influence_manifest.v1
action_proposal_ref:
trusted_intent_refs: []
trusted_policy_refs: []
untrusted_content_refs: []
data_egress_refs: []
influence_analysis:
policy:
  result: allowed | blocked | requires_review
  rule_refs: []
  violations: []
manifest_ref:
```

No raw private chain-of-thought is required or permitted.

---

# 7. Attested Browser Execution Capsule and Causality Trace

Every consequential browser execution SHALL produce a stable capsule reference linking Focusa intent and authority to UIAI runtime execution and resulting proof.

```yaml
schema: uiai.browser_execution_capsule.v1
capsule_id:
focusa:
  mission_ref:
  workpoint_ref:
  task_ref:
  action_proposal_ref:
  capability_grant_ref:
  operation_id:
  receipt_ref:
  correlation_id:
  causation_id:
worker:
runtime:
environment:
timeline:
observations:
execution:
proof:
cleanup:
integrity:
```

The full capsule SHALL be stored as an artifact and returned by reference by default. The compact result SHOULD expose only capsule ref, action and verification status, artifact refs, Receipt, duration, cost and cleanup status.

Where applicable, the capsule records UIAI/browser versions, session/context/target, isolation class, execution profile digest, environment/emulation, opaque auth handle, observations, actuator and fallbacks, retries, verification refs, context park/disposal, credential-lease revocation, hash and optional signature.

Trace metadata MUST NOT contain secrets, hidden values or unnecessary PII and MUST NOT be injected into unrelated third-party origins by default.

---

# 8. Origin-Bound WebMCP Actuator Broker

UIAI MAY discover and invoke supported origin-provided structured browser tools through an experimental origin-bound broker.

A discovered tool is an ephemeral actuator candidate. It is not a Focusa operation, trusted capability, durable procedure or canonical tool.

```yaml
schema: uiai.origin_tool_candidate.v1
candidate_id:
runtime:
  uiai_session_id:
  browser_context_id:
  target_id:
  document_id:
  navigation_id:
  frame_id:
  backend_node_id:
origin:
  top_level_origin:
  registration_origin:
  cross_origin_frame:
tool:
  name:
  description:
  input_schema:
  annotations:
  registration_stack_ref:
  manifest_hash:
classification:
  side_effect_class:
  risk_class:
  required_data_classes: []
  possible_destinations: []
lifecycle:
  status: discovered | authorized | changed | removed | revoked | expired
  valid_for_observation_ref:
  expires_at:
```

Broker laws:

- discovery does not grant authority;
- metadata, schemas and outputs are untrusted;
- candidates are bound to document, navigation, frame, origin and observation;
- navigation, removal or schema/hash change invalidates authorization;
- inputs are minimum necessary;
- returned success requires postcondition verification;
- fallback MUST NOT expand authority or disclosure;
- discovery is lazy and ranked;
- complete page tool catalogs MUST NOT flood agent context;
- unsupported browsers degrade to approved DOM/accessibility/visual/human routes.

---

# 9. Token and context efficiency laws

1. Agent-facing responses MUST default to compact structured projections.
2. Large observations, screenshots, recordings, diagnostics, traces, provenance and Evidence MUST remain behind stable handles unless policy requires inline content.
3. UIAI MUST NOT repeat full Focusa scope, runtime identity, artifact metadata or Evidence metadata when refs resolve them.
4. Every expandable object MUST have a typed bounded retrieval operation.
5. Observation deltas MUST identify a baseline and support explicit resynchronization.
6. Settlement watchers MUST execute server-side rather than through model polling.
7. WebMCP discovery MUST be lazy, query-scoped and ranked.
8. Execution capsules MUST be ref-first.
9. Provenance MUST use dictionaries, ranges, chunks or handles.
10. UIAI MUST select the minimum perception channels sufficient for reliability, security and evidence.
11. Policy may upgrade the response profile above the caller request.
12. Every result MUST contain an exact next action or truthful terminal state.
13. Bounded text and collections MUST expose truncation, completeness and continuation.
14. Usage MUST distinguish schema tokens, result tokens, model input/output, images/vision, browser time, retries, verification and Evidence cost.
15. Efficiency SHALL be measured as cost per verified and settled outcome.

---

# 10. Agent work-experience laws

The agent SHOULD reason in outcome-level Focusa operations, not raw browser primitives.

Preferred:

```text
verify order settlement
submit approved support request
capture evidence for predicate
```

Not preferred:

```text
choose among dozens of raw CDP, DOM, WebMCP and watcher tools
```

Every result SHALL classify recovery:

```text
stale observation → reobserve and replan
lost baseline → full resync
expired credential lease → request reauthorization
origin-tool manifest changed → invalidate and reroute
ambiguous side effect → reconcile before retry
verification inconclusive → invoke another permitted channel
context incompatible after restart → block restore or enter explicit degraded mode
```

The agent MUST NOT need to understand CDP node IDs, browser process IDs, trace internals, cache mechanics or event-store implementation for ordinary work.

---

# 11. Cockpit integration amendments

## 11.1 Amendment to Section 4 — Governed mission and outcome execution

Add observation-bound actions, browser verification requests/results, leased settlement watchers, browser-content influence policy, execution-capsule refs and origin-tool posture.

## 11.2 Amendment to Section 7.2 — Live

Add Inspector views for current observation/freshness, stale refs, selected actuator, untrusted-content/egress posture, watcher state, execution capsule, context attestation and cleanup.

## 11.3 Amendment to Section 7.3 — Test Lab

UIAI Engine Eval SHALL prove observation identity, revalidation, deltas/resync, predicate probes, settlement watchers, influence firewall, capsule reproduction and WebMCP behavior.

## 11.4 Amendment to Section 7.7 — Automations

Recipes may call outcome-level operations and verification capabilities. Execution traces may produce candidates, but Focusa retains procedural-memory promotion and authority.

## 11.5 Amendment to Section 7.8 — Evidence and Review Reports

Consume observation refs, verification results, influence manifests, execution capsules, watcher state, exact event ranges, immutable artifacts and hashes. Report review remains distinct from settlement or execution authorization.

## 11.6 Amendment to Section 13 — Contracts

Add this amendment's schemas to generated contract/parity systems. UIAI-native schemas remain UIAI-owned. Focusa refs come from Focusa's immutable generated bundle.

## 11.7 Amendment to Section 17 — Security

Add deterministic instruction/data separation, parameter-source provenance, origin/data egress enforcement and malicious DOM/ARIA/visual/tool fixtures.

## 11.8 Amendment to Section 18 — Performance

Add response-profile accounting, perception-channel selection, observation retention, watcher budgets, discovery cost and verified-outcome economics.

---

# 12. Implementation sequence

## 12.1 Phase 0 — Generated contracts and parity

Schemas, requirement IDs, generated clients, UIAI Capability Registry, Focusa bundle handshake, compact result and expansion operations.

## 12.2 Phase 1 — Observation truth

Native AX/DOM adapters, document/navigation/frame identity, versioned observations, reference model, observation-bound actions, stale rejection and resync.

## 12.3 Phase 2 — Agent exchange efficiency

Response profiles, client capability profiles, lazy discovery, compact results, handles, continuation and usage accounting.

## 12.4 Phase 3 — Security and attribution

Content provenance, influence firewall, data-class/egress enforcement, execution capsules, causality and cleanup attestation.

## 12.5 Phase 4 — Runtime verification

Predicate compiler, probes, contradictory-channel handling, settlement watchers and Focusa Evidence/Receipt handoff.

## 12.6 Phase 5 — Experimental structured actuators

WebMCP detection, origin/document/schema binding, lazy ranking, invocation, output tainting, postcondition verification and fallback.

No phase removes any requirement. Each remains in the machine-readable closure graph until verified.

---

# 13. UIAI Engine Eval matrix

Required scenarios:

- navigation after observation and before action;
- document and iframe replacement;
- inserted/reordered elements;
- modal or obstruction change;
- lost delta and resync;
- isolated authenticated contexts;
- context disposal and forbidden reuse;
- restart with compatible/incompatible profile digest;
- hidden DOM, ARIA and visual/OCR injection;
- tool-description and tool-output injection;
- unrelated data request and egress denial;
- confirmation-looking page without proof;
- conflicting DOM/visual/network evidence;
- asynchronous success followed by settlement failure;
- watcher budget exhaustion;
- ambiguous side effect and reconciliation;
- origin tool added, changed, removed or moved across navigation;
- cross-origin frame registration;
- DOM fallback without authority expansion;
- capsule and Review Report source-manifest reproduction.

UIAI Engine Eval remains the exclusive browser proof path for Focusa. No Playwright runtime or fixture may be introduced into Focusa.

---

# 14. Metrics

Measure:

- verified and settled completion;
- false-done rate;
- stale-action rejection;
- duplicate side effects;
- unauthorized-action blocks;
- prompt-injection block rate;
- cross-context leakage;
- contradiction detection;
- baseline-loss recovery;
- watcher completion and expiry;
- human rescue/takeover;
- retries by failure class;
- schema/result tokens;
- images/vision calls;
- browser minutes;
- verification and Evidence cost;
- cost per verified and settled outcome;
- time to verification and settlement.

No unmeasured percentage token-reduction claim may appear as a normative or marketing claim.

---

# 15. Acceptance criteria

This amendment is accepted when:

1. All schemas and requirements exist in machine-readable ledgers with owner, dependencies, tasks, client surfaces, Eval scenarios, Evidence, Receipts, migration and closure state.
2. UIAI consumes Focusa generated contracts by immutable version and has no duplicate Focusa operation registry.
3. Agent discovery is lazy and outcome-oriented.
4. All semantic-ref actions are observation-bound and fail closed when stale.
5. Deltas identify baselines and recover through full resync.
6. Compact, standard, evidence and developer profiles work and policy can force upgrades.
7. Large artifacts, provenance and capsules are ref-first.
8. Browser content cannot grant authority or expand egress.
9. Verification probes produce predicate-linked results with channel evidence and contradictions.
10. Settlement watchers are bounded, leased, server-side and non-authoritative.
11. Consequential executions produce reconstructable capsule refs.
12. WebMCP tools remain origin/document/schema-bound untrusted candidates.
13. Fallback cannot expand authority or disclosure.
14. Cockpit exposes freshness, actuator, verification, influence, capsule and watcher state through progressive disclosure.
15. UIAI Engine Eval passes the scenario matrix.
16. Focusa Evidence and Receipts link UIAI refs without importing raw browser state.
17. Cost per verified and settled outcome is measured.
18. No duplicate store, runtime, event history, procedure authority or Playwright path is introduced.

---

# 16. Closure blockers

This amendment cannot close while:

- refs remain snapshot-local without observation binding;
- stale actions can execute against changed documents or frames;
- deltas lack baseline/resync;
- agents must poll settlement state;
- provenance/influence is duplicated unboundedly into context;
- browser content can expand authority, disclosure or tool access;
- consequential execution lacks a capsule or cleanup posture;
- verification trusts a confirmation-looking page or model confidence alone;
- contradictory evidence is discarded;
- WebMCP tools are trusted global tools;
- schema changes do not invalidate origin-tool authorization;
- agent context is flooded with catalogs or schemas;
- UIAI duplicates Focusa operation, permission, Evidence, Receipt, event or memory authority;
- Cockpit presents execution as verified or settled without proof;
- required Eval, Evidence, Receipt, migration, compatibility or cost proof is absent.

---

# 17. Normative source map

- `UIAI_COCKPIT_UNIFIED_PRODUCT_IA_UX_SPEC_2026-07-16_v0.5.md`
- `UIAI_COCKPIT_INTERACTIVE_REPORTS_INTEGRATION_DECISION_2026-07-16.md`
- Focusa `docs/135-series-current-manifest.md`
- Focusa `docs/135-focusa-professional-workspaces-and-crist-project-genesis-master-spec.md`
- Focusa `docs/135a-...` through `docs/135k-...`
- Focusa generated OpenAPI, JSON Schema, Operation Registry, compatibility lock and proof matrix
- UIAI Engine capability registry, Session API, Source-to-Markdown, artifact and Eval contracts

---

# 18. Final principle

> **UIAI should preserve full browser truth internally while presenting the smallest sufficient, freshness-aware, expandable machine-readable projection for the current agent decision. Focusa decides meaning, authority, completion and settlement; UIAI executes, observes, verifies and proves.**
