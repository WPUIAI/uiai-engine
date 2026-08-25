# UIAI Cockpit Operational Constitution Amendment

**Document number:** `UIAI-COCKPIT-003`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding amendment:** `UIAI-COCKPIT-002`  
**Status:** Proposed normative amendment  
**Version:** 1.0  
**Date:** 2026-08-01  
**Primary implementation home:** `WPUIAI/uiai-engine`  
**Machine-readable companion:** [`UIAI-COCKPIT-003-C01`](./contracts/UIAI_COCKPIT_003_C01_OPERATIONAL_CONTRACT_LEDGER_v1.yaml)

---

## 0. Amendment decision

UIAI SHALL add one operational constitution governing cross-plane commits, ambiguity, crash recovery, resource ownership, admission control, runtime truth, identity and delegation, human intervention, protocol compatibility, evidence storage, and independently verifiable proof.

This amendment closes the gap between feature contracts and runtime behavior under disagreement, scarcity, partial failure, restart, version skew, and multi-agent contention.

```text
Focusa
  canonical mission intent · authority · budgets · predicates · settlement

UIAI Engine
  bounded execution attempts · runtime resources · artifacts · observations
  attempt journal · ambiguity classification · execution receipts

Cockpit
  operator decisions · intervention inbox · recovery · inspection · takeover
```

No component may infer success from the absence of an error. No retry may repeat an ambiguous consequential action before reconciliation.

---

# 1. Cross-plane operation protocol

Every consequential operation SHALL use a durable operation identity and the following state model:

```text
PROPOSED
→ AUTHORIZED
→ ADMITTED
→ STARTED
→ EXTERNAL_RESULT_OBSERVED
→ ARTIFACTS_COMMITTED
→ RECEIPT_COMMITTED
→ ACKNOWLEDGED_BY_FOCUSA
→ VERIFIED
→ SETTLEMENT_PENDING
→ SETTLED
```

Exceptional states:

```text
BLOCKED
CANCELLED_BEFORE_START
AMBIGUOUS_EXTERNAL_RESULT
RECONCILIATION_REQUIRED
COMPENSATION_REQUIRED
DEAD_LETTERED
SUPERSEDED
REVOKED
```

## 1.1 Stable operation identity

A consequential attempt SHALL carry:

- `operation_instance_id` — stable across transport retries;
- `attempt_id` — unique per execution attempt;
- `idempotency_key` — scoped to operation, target, account, and business intent;
- `mission_ref`, `workpoint_ref`, `task_ref`, and `action_proposal_ref`;
- `principal_ref` and `delegation_chain_ref`;
- `capability_grant_ref` and its effective version;
- `correlation_id` and `causation_id`;
- expected preconditions, completion predicates, and reconciliation strategy.

## 1.2 Commit law

UIAI MUST NOT begin a consequential side effect until the authorization reference and operation identity are durably available.

After an external side effect may have occurred, failure to persist a local Receipt SHALL produce `AMBIGUOUS_EXTERNAL_RESULT`, not `failed` or `not_executed`.

Focusa SHALL acknowledge the exact Receipt version it incorporated. UIAI retains unacknowledged Receipts for replay until acknowledgement or explicit operator resolution.

## 1.3 Recovery classes

Every operation declares one recovery class:

| Class | Meaning |
|---|---|
| `read_restartable` | Safe to restart from a fresh observation. |
| `retry_safe` | Exact idempotency key makes repetition safe. |
| `reconcile_before_retry` | External state must be checked before repetition. |
| `compensatable` | A defined compensating action exists. |
| `human_adjudication` | Operator judgment is required. |
| `irreversible_ambiguous` | No automatic retry or compensation is allowed. |

## 1.4 Attempt journal

UIAI SHALL maintain an append-only attempt journal containing stage transitions, runtime ownership, external target, actuator, pre/postconditions, artifact refs, receipt refs, errors, cleanup, and reconciliation status.

The journal is runtime authority for attempt recovery; it is not a second Mission or Workpoint store.

---

# 2. Resource Governor

UIAI SHALL implement one Resource Governor for all scarce execution resources:

- browser processes, contexts, pages, and persistent sessions;
- screenshot and visual workers;
- test runners and devices;
- document, OCR, conversion, and signing workers;
- media workers;
- AI provider calls and credits;
- network routes and proxy/IP capacity;
- disk, artifact retention, and bandwidth;
- operator-attention requests.

## 2.1 Admission contract

Every admitted job receives:

- `resource_lease_ref`;
- owner principal, mission, Workpoint, and project scope;
- requested and granted capacity;
- priority and fairness class;
- lease TTL and renewal rules;
- queue position and estimated eligibility where calculable;
- preemption, parking, and reclaim posture;
- budget reservation;
- exact blocked and recovery reasons.

## 2.2 Fairness and contention

The governor SHALL support:

- per-principal, project, tenant, mission, and profile quotas;
- bounded queues;
- starvation prevention;
- priority without permanent starvation of lower classes;
- expired-lease reclaim;
- voluntary release;
- parking and resumption;
- safe eviction only when policy permits;
- overload shedding before process failure.

## 2.3 Browser Session Broker

Persistent browser sessions SHALL expose owner, scope, lease, priority, last use, parkability, and parked-state refs. Pool-full responses SHALL return actionable choices rather than a generic capacity error.

The broker SHALL distinguish persistent-session occupancy from transactional page pressure.

---

# 3. Operational truth model

Health and capacity fields SHALL use separate semantics:

| Signal | Question |
|---|---|
| `liveness` | Is the process alive? |
| `readiness` | Can this capability accept a request now? |
| `capacity` | How much work can be accepted now? |
| `degradation` | What quality, latency, or fallback reduction exists? |
| `recent_pressure` | What occurred in a bounded recent window? |
| `historical_slo` | What occurred over a declared historical window? |
| `dependency_state` | Which required dependency is unavailable? |
| `freshness` | When and how was this status measured? |

Cumulative counters MUST NOT latch current readiness. A healthy generic endpoint MUST NOT conceal an unavailable required capability.

Every status object SHALL include source, measured time, validity window, scope, current-versus-historical classification, and recommended caller behavior.

---

# 4. Crash and restart recovery

On startup, UIAI SHALL reconcile:

- attempts admitted but not started;
- started attempts without terminal results;
- possible external success without a committed Receipt;
- partially written artifacts;
- abandoned browser contexts and sessions;
- orphaned resource and credential leases;
- active settlement watchers;
- incomplete document transformations;
- interrupted report freeze/share operations.

Runtime configuration SHALL be fingerprinted and persisted with process identity. Cockpit SHALL show the effective running configuration, its source path, digest, loaded time, and differences from the persistent restart candidate.

Temporary or missing configuration and log paths SHALL produce an explicit restart-risk state.

---

# 5. Principal and delegation model

All work SHALL identify a canonical `PrincipalRef`:

```yaml
principal_ref:
  principal_id:
  kind: human | agent | model | harness | service | node | device
  tenant_ref:
  organization_ref:
  display_name:
  authenticated_by:
```

Delegated work SHALL carry a bounded `DelegationChain` identifying requester, delegator, approving human or policy, executing harness, model, node, and external account context.

Revocation SHALL propagate to pending admission, active leases, credential grants, follow-up actions, and settlement watchers according to operation risk and external ambiguity.

---

# 6. Human Intervention Protocol

Cockpit SHALL provide a first-class Decision and Intervention Inbox for:

- approvals and capability grants;
- authentication and MFA barriers;
- CAPTCHA or challenge escalation;
- contradictory or insufficient evidence;
- budget increases;
- ambiguous external results;
- takeover requests;
- failed settlement;
- expiring leases and grants;
- incidents and dead letters.

Every item SHALL state why a human is required, deadline, consequence of no action, recommended action, reversibility, evidence summary, and resumable context.

Authentication takeover SHALL support secret-shielded operator input, recording suppression, fresh post-handoff observation, and an explicit resulting authority/session handle.

---

# 7. Protocol compatibility and long-running work

Every shared schema SHALL declare supported min/max versions and compatibility posture.

Runtime negotiation SHALL cover mixed Focusa, UIAI, Cockpit, Cloud, and node versions. Long-running missions, leases, parked sessions, Receipts, and frozen reports SHALL pin the contract versions required for deterministic continuation.

Breaking changes require migration adapters, canary proof, rollback support, and deprecation telemetry.

---

# 8. Evidence storage constitution

Evidence and artifact storage SHALL provide:

- content-addressed identity and integrity checks;
- encryption and tenant/project isolation;
- immutable originals and derived lineage;
- deduplication;
- pinned retention and legal hold;
- dependency-aware deletion;
- garbage collection and storage-pressure behavior;
- backup, restore, and corruption detection;
- portable evidence bundles;
- redacted derivatives that never overwrite originals.

Deletion SHALL never leave a Receipt silently pointing to unavailable proof without an explicit tombstone and policy reason.

---

# 9. Independent proof verification

UIAI SHALL provide a standalone proof-verification contract suitable for CLI and public-safe viewer implementations.

A verifier SHALL resolve:

- manifest and schema version;
- signer identity and authorization scope;
- signature and revocation state;
- artifact presence and integrity;
- evidence lineage;
- verified, inferred, provisional, and settled claims;
- time and freshness posture;
- missing or contradictory sources.

Cockpit-rendered reports are not the only means of validating a proof bundle.

---

# 10. Implementation phases

```text
Phase 0  Operation identities, attempt journal, recovery classes
Phase 1  Resource Governor and Browser Session Broker
Phase 2  Truthful health/readiness/capacity contracts
Phase 3  Restart reconciliation and effective-config provenance
Phase 4  Principal/delegation model and Intervention Inbox
Phase 5  Protocol negotiation and evidence-storage constitution
Phase 6  Standalone proof verifier and conformance release gates
```

# 11. Acceptance conditions

This amendment is complete only when:

1. A crash after an external side effect cannot be reported as a clean failure.
2. An ambiguous action cannot be retried until reconciliation policy permits it.
3. Every scarce runtime resource has an owner, lease, queue, and reclaim posture.
4. Current readiness cannot be permanently poisoned by historical counters.
5. Restart recovery identifies orphaned attempts, resources, watchers, and credentials.
6. Operator decisions appear in one actionable inbox.
7. Mixed-version nodes negotiate or fail with an exact compatibility reason.
8. Evidence bundles can be independently validated outside Cockpit.
