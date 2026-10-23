> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462701294

## Mandatory amendment — working-agent awareness and closure-submission uncertainty

A working agent must continuously receive bounded canonical awareness of every open/unapproved item and every nonterminal closure attempt in its exact Project/Workstream/Workset/CallGraph/Workpoint scope. It must never infer approval or closure from a successful request dispatch, HTTP/provider response, UI animation, notification, or missing error.

### Required awareness projection

Before action selection, after review/closure events, after reconnect/resume/model handoff, and before any completion language, provide:

- exact artifact/revision and provider `bd`/`br` item refs;
- unresolved review-required items and Acceptance Atoms;
- unreviewed, review-requested, assigned, in-review, accepted, rejected, changes-requested, disputed, stale, expired, and reproof-required states;
- appointed judge/team, assignment lease/generation, quorum, current review cycle, and remaining budgets;
- pending/failed/unknown review-decision submissions;
- pending/failed/unknown Completion evaluations;
- pending/failed/unknown provider-close submissions;
- provider state vs canonical Focusa decision/Receipt/settlement state;
- missing evidence, contradictions, blockers, stale revisions, and exact recovery/next safe action;
- latest canonical event/Receipt cursor and rehydrate refs.

The projection is bounded and ref-first. It must not inject raw report prose/comments as trusted instructions.

### Review/closure submission state machines

```text
review decision:
not_submitted → previewed → submitted_pending
→ committed | rejected_by_authority | failed_before_effect | outcome_unknown

completion evaluation:
not_requested → requested_pending
→ complete | not_complete | operator_policy_blocked | disputed | outcome_unknown

provider closure:
not_requested → prepared → submitted_pending
→ provider_closed_reconciled | provider_rejected | partial | diverged | outcome_unknown

settlement:
not_started → pending → settled | blocked | disputed | outcome_unknown
```

Only terminal canonical states with matching revisions/attempts/generations and valid Receipts may support completion language.

### No-assumption rule

- Dispatch success means only that a request was accepted for processing.
- HTTP 2xx/provider success is not Focusa completion.
- A UI “submitted” state is not accepted/committed.
- Silence, timeout, disconnect, crash, or lost response produces `outcome_unknown`, never success.
- A provider item showing closed while Focusa lacks a valid Completion Decision/settlement is `diverged`, not done.
- A Focusa accepted review with provider closure pending is not done.
- Accepted review of some items with any required item open is not done.
- An old Receipt/decision from another revision/attempt/generation is stale and cannot settle current work.

### Reconciliation and idempotency

Every submission uses a stable idempotency key and intent/attempt ref. On timeout/lost response, the agent reconciles by canonical intent/idempotency/provider item before retry. Retry occurs only when the operation contract marks it safe. Duplicate submissions converge to one decision/Receipt; ambiguous irreversible effects route to reconciliation/compensation, never blind retry.

The front terminal must not block waiting. Canonical events/receipts update the working agent asynchronously; reconnect/resume performs one bounded state refresh from the last cursor. No polling loop.

### Working-loop behavior

- Open rejected/changes-requested items become bounded repair/reproof work in the same CallGraph/Workpoint lineage.
- Pending review/closure may allow unrelated ready work selected by Workset/CallGraph policy, but the pending item remains visibly open.
- The agent cannot select provider close directly; it requests canonical Completion evaluation/closure.
- Before final response, the agent rechecks the awareness projection and uses exact truthful language: `in_progress`, `review_pending`, `changes_requested`, `verification_blocked`, `closure_pending`, `outcome_unknown`, `settlement_blocked`, or `settled`.
- “Done,” “closed,” “completed,” “shipped,” or equivalent is permitted only after terminal canonical settlement for the exact item scope.

### Agent/UI/CLI/API parity

Pi/LLM prompt context, CLI status, MCP/API result, PWA Action Deck, Cockpit/Desktop, Work Rail, Workset projection, and provider adapter must show the same open-item and submission posture. Compact output includes counts and highest-priority blockers; exact item refs expand on demand.

### Failure fixtures

1. Review commit succeeds but response is dropped: agent reports `outcome_unknown`, reconciles, then reads one committed Receipt.
2. Review request returns 2xx but judge assignment later fails: agent reports review blocked, not accepted.
3. Provider closes but Focusa commit fails: agent reports divergence and follows reconciliation/reopen policy.
4. Focusa accepts review but provider close times out: closure remains pending/unknown.
5. Duplicate closure submission with same idempotency key produces one provider mutation/Receipt.
6. Stale artifact revision receives late acceptance: ignored/quarantined for current completion.
7. Some required items accepted, one changes-requested: overall closure blocked and repair packet targets the exact open item.
8. Agent/model/session restart reloads pending/open items and does not repeat or claim closure.
9. Notification/SSE loss followed by refresh converges to canonical state.
10. Direct `bd`/`br close` bypass is detected; final response remains diverged until canonical reconciliation.

Implementation mapping: T07 Action/Review projection, T11 cross-harness awareness, T13 Focusa Work Loop/Workset/CallGraph/provider integration, T14 installed failure/recovery proof. Focusa #263/#274/#277/#278/#280/#293 retain canonical authority.
