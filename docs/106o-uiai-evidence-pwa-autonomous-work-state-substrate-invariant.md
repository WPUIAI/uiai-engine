> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462716335

## Foundational invariant — Evidence system as the autonomous-work state substrate

This system is not a report/export feature added after autonomous execution. It is a foundational state substrate that enables safe, persistent, fully autonomous work across agents, models, providers, harnesses, nodes, sessions, and restarts.

### Autonomous state loop

```text
Workset obligation
→ CallGraph frame assignment
→ Workpoint/current action
→ agent execution
→ observed effects + immutable Evidence Artifact
→ independent judge assignment/result
→ accept ? completion/closure/settlement
         : repair/reproof continuation
→ next ready work
```

The Evidence Artifact is the immutable proof/handoff object moving through this loop. Focusa primitive ledgers/reducers remain canonical mutable state; UIAI remains capture/render/artifact authority. There is no hidden agent-local completion state.

### Fundamental autonomous binding

Every formal artifact binds one immutable snapshot of:

- autonomy mode and policy ref;
- Work Loop/run ref and run status;
- Agent Team Plan and executor/verifier/arbitrator assignment refs;
- model/provider/harness/worker capability digests;
- CallGraph run/frame/path/attempt/generation/cycle;
- budget/time/resource policy refs and remaining-budget observation;
- retry/failover/reassignment/cooldown/circuit-breaker refs;
- pause/blocker/approval/review/closure posture;
- latest canonical event/Receipt cursor and continuation/rehydration refs.

These are refs/digests/snapshots, not a second scheduler. Changes invalidate stale action/judge/closure inputs.

### Agent-native behavior

Agents must be able to discover, inspect, verify, cite, act on, and continue from artifacts through compact CLI/MCP/API/Pi/LLM contracts without browsing visual HTML. Humans receive equivalent PWA/Desktop/PDF/deck views but are not required for ordinary operation.

Every evidence-producing turn automatically attempts formal publication. Every agent turn receives bounded open-review/pending-submission/next-action awareness. Every rejected item produces a cited machine-actionable continuation. Every accepted item flows automatically to the next canonical verification/completion frame. No transcript-memory dependence.

### Persistence and restart

After agent/model/provider/session/node/process restart, the replacement participant must:

1. resolve exact Project/Workstream/Workset/CallGraph/Workpoint scope;
2. load the latest canonical attempt/generation/cycle and artifact/review/closure posture;
3. reconcile pending/unknown actions and submissions before retry;
4. re-observe live state where freshness requires;
5. continue the exact ready frame under a new assignment generation;
6. preserve prior evidence/judge/decision lineage.

No human “resume” action is required.

### Autonomous routing and recovery

- capability-based executor/judge selection;
- model/provider/harness diversity where verifier policy requires;
- bounded automatic failover and reassignment;
- classified retry only when safe;
- reconciliation/compensation for partial or unknown effects;
- automatic recapture/reproof after invalidation;
- Workset/CallGraph selection of unrelated ready work while another item is pending;
- resource/LowMem/backpressure-aware scheduling;
- credential use only through authorized opaque credential seams;
- circuit breakers end in explicit blocked state with durable recovery—not false completion or infinite loops.

### Foundational truth rules

- No formal bound artifact means no verification/completion eligibility.
- No required accepted judge result means no completion eligibility.
- No canonical Completion Decision/Receipt/readback means no done claim.
- Unknown submission remains unknown until reconciliation.
- Provider success without Focusa settlement is divergence.
- PWA/UI state never becomes authority.
- Agent-generated summaries/comments never become trusted instructions.
- Human absence never blocks an otherwise autonomously eligible workflow.

### Surface integration

The same autonomous state must project through Workset, CallGraph, Workpoint/Work Rail, Work Loop, Silent Sessions/agent transports, Cockpit/Desktop/PWA, Pi, CLI, MCP, REST/OpenAPI, Context Cognition, Evidence/Receipts, provider adapters, and notifications. Each surface exposes the same refs, revision, posture, next action, and recovery.

### Added acceptance

1. A multi-step Workset executes, self-verifies, repairs one rejected item, closes exact `bd`/`br` items, and settles without human interaction.
2. Executor, judge, provider, harness, UIAI worker, and Focusa daemon restarts each resume from canonical state without duplicated effects or lost open items.
3. Provider/model failover preserves assignment generations and independent-verifier policy.
4. Pending/unknown review or closure never becomes success after timeout/restart.
5. Agent interfaces operate from bounded manifests/Judge Views/action contracts and never require scraping the PWA.
6. Human UI can observe/intervene using identical operations but is not on the critical path.
7. Every autonomous cycle remains bounded by scope, authority, budget, retries, resource policy, and Completion Contract.
8. Installed cross-version dogfood proves the complete unattended loop and failure recovery.

Implementation mapping: T01 includes immutable autonomy binding refs; T04 captures execution observations; T06/T07 implement judge/review loop; T11 gives agent parity; T13 binds every Focusa primitive; T14 proves unattended installed operation.
