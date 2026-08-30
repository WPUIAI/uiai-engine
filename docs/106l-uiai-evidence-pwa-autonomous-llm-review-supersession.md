> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462687469

## Superseding steering — human review optional; full autonomous LLM approval path required

This comment supersedes any prior wording that makes a human reviewer, dual human control, or human-in-the-loop step an architectural prerequisite.

### Core invariant

Evidence review, acceptance/rejection, rework, reverification, Completion evaluation, and eligible `bd`/`br` closure must support end-to-end autonomous execution with no human wait state.

Humans remain optional authorized participants: they may inspect, comment, intervene, steer, dispute, or override where policy permits, but ordinary or high-rigor proof execution does not depend on their availability.

### Autonomous review loop

```text
working agent executes scoped work
→ publishes immutable Evidence Artifact
→ reviewer policy appoints independent LLM judge/team
→ judge claims expiring assignment
→ judge retrieves exact Judge View and required source media
→ item-scoped accept | reject | changes_requested | disputed/indeterminate
→ rejected items produce bounded cited continuation packet
→ working agent repairs/recaptures/reproofs
→ new immutable artifact revision
→ automatic re-review
→ Completion Authority evaluates all item decisions/Atoms/revalidation
→ provider closure allowed or refused
```

No LLM may self-appoint, silently broaden scope, alter the rubric, or write Completion/settlement state directly.

### Robust autonomous controls

- executor/verifier role separation;
- independent assignment generated from approved verifier policy;
- exact Project/Workstream/Workpoint/provider-item/artifact revision binding;
- model/provider/harness identity and capability digest;
- required modality checks before assignment;
- calibrated judge-policy/model thresholds and golden regression gates;
- optional multi-model/provider diversity and anti-correlation policy;
- quorum/ordering/tie/disagreement rules;
- blinded evaluation profiles that withhold producer verdict/persuasive summaries where appropriate;
- exact immutable information-set hash and evidence citations;
- confidence/uncertainty/omission/contradiction reporting;
- bounded retry/reassignment/failover ladder by classified failure;
- maximum review/rework cycles, token/media/spend/time budgets, and cooldown;
- circuit breakers for repeated rejection, disagreement, drift, unavailable modality, provider outage, or suspected prompt injection;
- additional independent judge/arbitrator routing for disputed cases;
- deterministic blocked/indeterminate posture when policy cannot safely settle;
- action leases and exact allowed vocabulary for any autonomous recapture/repair;
- no raw chain-of-thought; bounded rationale and typed findings only;
- immutable judge result, assignment, action, rework, and Completion Receipts;
- watchdog revalidation and automatic reopening when source/runtime/policy changes.

### Item-scoped autonomy

Approval controls and judge operations remain visible/active only for applicable unresolved review-required items. Autonomous judges may accept some items and reject others. Overall review posture is derived from all required item decisions and policy; no blanket approval switch.

Exact `bd`/`br` IDs/descriptions/revisions and Review Requirement refs remain in artifact metadata. Rejection packets cite those exact provider items and feed directly into the working agent's bounded Workpoint/Work Loop continuation.

### Human-optional UI

The PWA/Desktop still offers authorized human decision controls, but they are optional projections of the same operations. UI states clearly distinguish:

- autonomous review queued/running;
- appointed judge(s), capability, independence, budget, and lease;
- accepted/rejected/changes/disputed item states;
- current rework cycle and next automatic action;
- circuit-breaker/blocked reason;
- optional human intervention available—not required.

Public/offline artifacts remain read-only.

### Externally mandated human authority

If a law, contract, provider, or explicitly configured operator policy genuinely requires a human signature, the system must declare that operation `autonomous_ineligible`/blocked at admission. It must not fake human approval, weaken the rule, or claim full autonomous eligibility for that operation. This is an external applicability boundary, not a default Focusa architecture dependency.

### Autonomous closure gate

Required accepted LLM review can satisfy the reviewer leg when the Completion Contract authorizes that verifier class. Acceptance remains only one input. Completion Authority still checks Evidence integrity/scope/freshness, every Acceptance Atom, reviewer independence/quorum, revalidation, provider state, Receipts, and settlement.

`bd`/`br` closure proceeds automatically only after the canonical Completion Decision authorizes the exact item. Rejection, missing proof, dispute, exhausted budget, circuit breaker, or unresolved contradiction blocks closure and routes the next safe autonomous recovery—never false success.

### Added acceptance

1. A working LLM and independently appointed multimodal judge complete publish → reject → repair → republish → accept → canonical provider-close without human action.
2. Judge outage triggers bounded provider/model failover; no human wait state is introduced.
3. Conflicting judges route through deterministic arbitration/quorum and remain disputed if unresolved.
4. Repeated rejection obeys cycle/spend/time limits and ends blocked with exact evidence—not infinite loops.
5. Self-review, collusion/correlated verifier policy, modality mismatch, stale assignment, prompt injection, and information-set drift fail closed.
6. Optional human intervention uses identical operations/Receipts and does not create a second approval system.
7. An externally human-mandated operation is rejected as autonomously ineligible rather than bypassed.

Implementation mapping: T06 owns autonomous Judge View/runtime; T07 owns item-scoped Review UI/actions; T13 owns Focusa continuation/Completion/provider closure; T14 proves no-human end-to-end dogfood.
