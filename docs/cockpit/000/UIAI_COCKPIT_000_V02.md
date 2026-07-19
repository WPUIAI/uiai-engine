Healthy details stay quiet. The interface becomes more explicit when:

- scope is missing, stale, or conflicting;
- the selected node is not authoritative;
- capacity is saturated;
- an action requires approval;
- a cloud call will send data;
- an output invalidates a signature;
- a test or document operation fails;
- evidence has not yet been captured;
- a result is provisional, surrogate, or degraded.

## 3.6 Technical language with human-first ordering

Preferred ordering:

```text
Human meaning
Technical classification
Recovery action
Raw details
```

Example:

```text
Couldn’t click Save because another panel covers it.
selector_obscured · browser execution
Scroll the element into view or inspect the current snapshot.
[Show technical details]
```

## 3.7 Consistency over cleverness

The same patterns must work across browser sessions, tests, documents, research, and media:

- open as a work object;
- show content in the center;
- inspect on the right;
- show work state in Activity;
- produce artifacts;
- request approval consistently;
- link evidence consistently;
- preserve scope consistently.

---

# 4. Governed mission and outcome execution model

This section integrates the best-practice agentic-browser architecture into the Cockpit without collapsing UI, cognition, policy, execution, verification, and memory into one application process.

The governing thesis is:

> **The Cockpit is the Mission Experience for a governed intent-to-outcome system. A successful tool call, click, form submission, test step, file transformation, or agent claim is not completion. Completion requires explicit predicates, sufficient evidence, and—when asynchronous—settlement.**

## 4.1 Required architectural separation

The unified system maps to these logical layers:

```text
Mission Experience
  UIAI Cockpit · FPV PWA · Pi guided UX
        ↓
Mission Kernel
  Focusa ProjectIdentity · Mission/Workpoint · Trajectory · constraints · continuity
        ↓
Orchestrator
  explicit task graphs · workers · leases · locks · budgets · retries · escalation
        ↓
Authority Kernel               Action Router
  deterministic policy          safest eligible actuator
        ↓                              ↓
UIAI execution planes · API/MCP/connectors · human takeover
        ↓
Evidence · independent verification · outcome settlement
        ↓
Scoped memory and procedural learning in Focusa
```

Ownership rules:

| Concern | Canonical owner | Cockpit responsibility |
|---|---|---|
| Mission/project truth | Focusa Mission Kernel | Display, edit through explicit contracts, request amendments, never shadow it. |
| Task graph, leases, locks, retries, budgets | Focusa or a dedicated orchestrator using canonical Focusa scope | Visualize, request, pause, cancel, and inspect; never keep a hidden browser-only plan as truth. |
| Deterministic authority | ScopeGuard, AuthorityGuard, Focusa policy, UIAI action-boundary enforcement | Explain grants and denials; collect informed approvals. |
| Browser/document/test/media execution | UIAI Engine and registered runners | Start, observe, steer, cancel, and render artifacts. |
| API/MCP/native connector execution | Registered adapter/tool provider behind the same authority checks | Show route, destination, data disclosure, fallback, and receipt. |
| Verification | Deterministic checks, independent verifier, external query, or human review | Present predicate-by-predicate results and contradictions. |
| Settlement | Focusa Mission Kernel or explicit receipt/settlement service | Distinguish provisional completion from settled completion. |
| Durable evidence and memory | UIAI artifacts plus Focusa evidence/memory linkage | Render, compare, redact, attach, correct, delete, and export by policy. |

Integrated UX MUST NOT imply integrated authority. A visually seamless transition between Focusa, UIAI, a runner, an API, or a human takeover does not permit one plane to impersonate another.

## 4.2 Mission Deck contract

Overview evolves into the **Mission Deck** whenever a verified mission or Workpoint is active. It is not a chat transcript and not a dashboard of unrelated metrics.

The Mission Deck MUST make the following discoverable:

- desired external outcome;
- current mission lifecycle state;
- current trajectory and meaningful Workpoints;
- active workers, sessions, jobs, and lease holders;
- pending decisions, contradictions, and clarifications;
- currently granted authority and expiration;
- remaining financial, token, time, retry, browser, and human-attention budgets where applicable;
- current browser, document, test, API, or application context;
- required and collected evidence;
- uncertainty, missing evidence, and verifier disagreement;
- pause, stop, takeover, revoke, amend, fork, handoff, and resume controls;
- Completion Contract status and settlement status.

Progressive disclosure applies:

- **Glance:** outcome, lifecycle, strongest next action, blocking decision, completion posture.
- **Work:** Workpoints, active work, relevant evidence, approvals, remaining budget.
- **Inspect:** grants, predicates, workers, task graph, contradictions, receipts.
- **Configure:** policies, routing preferences, escalation, retention, verifier requirements.
- **Developer:** canonical envelopes, action/receipt IDs, route scoring, raw events, hashes.

The Mission Deck SHOULD report meaningful progress such as “4 of 6 completion predicates verified” or “submission accepted; external settlement pending.” It MUST NOT lead with meaningless activity such as “thinking,” “working,” or “step 14.”

## 4.3 Versioned Mission Contract

Before consequential autonomous execution, an ambiguous request MUST be compiled into or linked to a versioned Mission Contract owned by the Mission Kernel.

A Mission Contract SHOULD contain:

```ts
interface MissionContract {
  mission_id: string;
  version: number;
  objective: string;
  constraints: Constraint[];
  completion_predicates: CompletionPredicate[];
  evidence_requirements: EvidenceRequirement[];
  authority_grants: CapabilityGrant[];
  risk_policy: RiskPolicy;
  budget: BudgetPolicy;
  failure_policy: FailurePolicy;
  escalation_policy: EscalationPolicy;
  data_policy: DataPolicy;
  created_at: string;
  expires_at?: string;
}
```

Requirements:

1. The objective identifies an external outcome, not merely an internal action.
2. Constraints are machine-evaluable where practical.
3. Ambiguities affecting authority, spending, legal status, identity, external communication, sensitive data, or irreversible outcomes are surfaced rather than silently guessed.
4. Low-risk defaults identify their policy or preference source.
5. Amendments create a new version and preserve prior versions.
6. Cockpit may draft and visualize a contract but MUST NOT maintain a competing canonical copy.
7. Existing Focusa Workpoint and Trajectory contracts remain usable while richer Mission Contract fields are introduced additively.

## 4.4 Completion Contract and false-done prevention

Every consequential mission MUST define what counts as complete.

A Completion Contract contains predicates and acceptable evidence, for example:

```text
Predicate: Report delivered to the approved recipient
Required evidence:
- generated document hash;
- sent-message record;
- recipient and subject match;
- no delivery failure event within the settlement window.
```

The system MUST NOT declare completion merely because:

- a button was clicked;
- a form was submitted;
- a page navigated;
- no error appeared;
- a confirmation-looking screen appeared;
- an agent or model expressed confidence;
- a file was generated locally but not delivered;
- a test runner reached its final step;
- a remote system acknowledged receipt but has not settled the result.

The Cockpit MUST show completion predicate status as **verified**, **failed**, **contradicted**, **missing evidence**, **not evaluated**, or **settlement pending**.

## 4.5 Mission, task, and action lifecycle

The canonical backend may use richer internal states, but Cockpit must preserve the following semantic distinctions.

### Mission lifecycle

```text
DRAFT → COMPILING → NEEDS_CLARIFICATION → READY_FOR_APPROVAL
→ AUTHORIZED → PLANNED → EXECUTING → WAITING_EXTERNAL
→ VERIFYING → PROVISIONALLY_COMPLETE → SETTLED
```

Exceptional states:

```text
PAUSED · BLOCKED · EXPIRED · CANCELED · FAILED
PARTIALLY_COMPLETED · DISPUTED · ROLLED_BACK
```

### Task lifecycle

```text
PENDING → READY → LEASED → RUNNING → ACTION_ATTEMPTED
→ RESULT_OBSERVED → POSTCONDITION_CHECKED → VERIFIED → COMPLETED
```

Failure classes:

```text
RETRYABLE_FAILURE · AUTHORIZATION_FAILURE · SEMANTIC_FAILURE
EXTERNAL_FAILURE · IRREVERSIBLE_OR_UNKNOWN_FAILURE · DEAD_LETTER
```

### Consequential action lifecycle

```text
PROPOSED → AUTHORIZED → STARTED → SUBMITTED → ACCEPTED → VERIFIED → SETTLED
```

These states are not synonyms:

- **Submitted:** UIAI or another actuator sent the request.
- **Accepted:** the target acknowledged it.
- **Verified:** evidence supports the intended postcondition.
- **Settled:** expected external confirmation windows and reconciliation checks have completed.

Cockpit MUST NOT flatten all four into `Complete`.

## 4.6 Deterministic Authority Kernel and Capability Grants

The reasoning model may propose an action. It MUST NOT be the sole authority deciding whether that action is allowed.

A Capability Grant SHOULD identify:

```ts
interface CapabilityGrant {
  grant_id: string;
  principal_id: string;
  mission_id: string;
  action_classes: string[];
  allowed_origins?: string[];
  allowed_resources?: string[];
  permitted_data_classes?: DataClass[];
  maximum_financial_value?: number;
  maximum_uses?: number;
  may_delegate: boolean;
  valid_from: string;
  expires_at: string;
  required_evidence?: string[];
  approval_level: string;
}
```

Guardrails:

- grants are scoped to a principal and mission;
- grants expire and can be revoked;
- authority is re-evaluated at the actual action boundary;
- page content, tool descriptions, model statements, or prior approval claims cannot expand a grant;
- fallback routing cannot silently expand origin, data, cost, use count, or delegation rights;
- every consequential receipt records the grant that authorized it;
- delegation requires explicit permission and cannot exceed the delegator’s authority;
- revoked or expired authority fails closed.

### Risk classes

| Class | Meaning | Typical examples | Default posture |
|---|---|---|---|
| **R0** | Observation only | Read, inspect, render, validate | Execute when scoped and authorized. |
| **R1** | Reversible local change | Draft, local derivative, sort/filter | Immediate or lightweight preview. |
| **R2** | Reversible remote change | Save setting, create remote draft | Scoped grant; clear receipt. |
| **R3** | External communication | Send message, submit support request | Explicit destination/data preview. |
| **R4** | Account or identity change | Password/profile/permission changes | Strong identity and explicit approval. |
| **R5** | Financial, legal, public, destructive, or irreversible | Purchase, sign/file, publish, delete, final redaction | Narrow one-purpose grant, independent verification, minimal retries, human approval. |

Approvals SHOULD issue a scoped capability lease rather than a vague acknowledgment. The UI MUST communicate consequence, destination, disclosed data, financial ceiling, reversibility, duration, reuse, and required evidence.

## 4.7 Action Proposal

Before a consequential action, the executing worker SHOULD emit a machine-readable Action Proposal containing:

- mission, Workpoint, task, worker, and principal identity;
- intended action and target;
- purpose and expected external effect;
- data that will be disclosed and destination origin;
- proposed capability grant;
- preconditions and expected postconditions;
- risk class and reversibility;
- idempotency strategy;
- proposed actuator and fallback policy;
- expected evidence and settlement behavior.

Cockpit renders the proposal as a calm human preview. Developer Mode exposes the full envelope.

## 4.8 Action Router and actuator-neutral UX

The system MUST NOT assume that browser clicking is always the safest or most reliable execution method.

The Action Router SHOULD evaluate eligible mechanisms in this general order:

1. verified native connector;
2. authenticated API;
3. authenticated MCP tool;
4. brokered origin-bound structured browser tool;
5. deterministic DOM or accessibility action;
6. visual computer use;
7. human takeover.

Selection factors include reliability, security, reversibility, authentication scope, evidence quality, cost, latency, site policy, freshness, semantic explicitness, availability, and observed failure rate.

Requirements:

- the chosen actuator is visible in Inspect level and recorded in the receipt;
- user or organizational policy can prohibit an actuator;
- structured tools remain untrusted origin-supplied interfaces;
- a fallback is explicit and cannot increase authority or disclosure;
- Cockpit workspaces organize by outcome and object, not by actuator;
- manual browsing remains available without an active mission.

## 4.9 Action Receipt and typed Evidence Items

Every consequential action MUST produce a machine-readable Action Receipt. Replay video may supplement it but does not replace it.

A receipt SHOULD include:

- mission, Workpoint, task, action, worker, model/runtime, and capability-grant IDs;
- timestamp, target origin/service, and actuator;
- bounded/redacted input summary;
- relevant before state;
- action performed and immediate result;
- relevant after state;
- evidence references;
- network/API/external confirmation identifiers;
- verification and settlement status;
- cost, duration, retry count, uncertainty, and failure status.

