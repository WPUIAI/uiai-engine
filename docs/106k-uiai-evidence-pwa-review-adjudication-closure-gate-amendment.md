> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462678473

## Mandatory amendment — appointed Review/Adjudication Gate and provider-closure refusal

Every Evidence PWA must expose a governed Review Case and decision UI for an appointed human verifier or LLM judge. The artifact remains immutable; review state, assignments, comments, decisions, and Receipts are separate canonical records projected beside it.

### Review state machine

```text
unreviewed
→ review_requested
→ assigned
→ in_review
→ accepted | rejected | changes_requested | disputed | expired | cancelled
accepted → completion_evaluation_pending
rejected | changes_requested → working_agent_followup
any material invalidation → stale/reproof_required
```

No client may skip states, overwrite prior decisions, or display optimistic acceptance before the canonical commit/Receipt returns.

### Reviewer assignment

A versioned verifier/reviewer assignment binds:

- Project / Workstream / Workpoint / Work Item / artifact revision;
- Completion Contract and Acceptance Atom revisions;
- appointed human/agent/model/harness/provider identity and role;
- required modality/capability and independence class;
- allowed decision vocabulary and scope;
- assignment/lease issue, expiry, claim, release, reassignment, and revocation;
- conflict-of-interest/self-verification policy;
- required information-set/Judge View refs;
- permissions, confirmation, quorum/order policy, and expected Receipt.

Reviewers cannot self-appoint or expand scope. A producer/working agent cannot masquerade as an independent judge. LLM acceptance is authoritative only to the limited degree explicitly granted by the Completion Contract; high-consequence policy may require named human approval or dual review.

### Relevance-driven item approval

Approval/rejection controls appear only on reviewable items with an applicable, unresolved Review Requirement for the current artifact/item revision. They are not global decoration and must not appear on raw media, metadata, informational findings, non-applicable atoms, already-settled items, or items outside the reviewer assignment.

Each reviewable item binds:

- stable item/claim/Acceptance Atom/finding/work-item ref;
- exact artifact revision and source evidence refs;
- `review_required` and review-policy ref;
- allowed decision vocabulary;
- appointed reviewer/role/capability requirements;
- current canonical review state and decision/Receipt refs;
- dependencies/quorum/ordering;
- unresolved evidence/citation requirements;
- closure consequence and safe next action.

Only an appointed, unexpired, authorized reviewer sees enabled decision controls. Other viewers see read-only state and, when authorized, a request-review/handoff action. The UI supports partial review: required items may independently become accepted, rejected, changes requested, disputed, expired, or pending. Overall review posture is deterministically derived from all applicable required item decisions; it cannot be manually toggled.

### `bd` / `br` work-item metadata

Every artifact associated with Beads work carries a bounded immutable snapshot per item:

- provider kind/surface (`bd`, `br`, or canonical provider id);
- canonical work-item ref and exact provider item ID;
- item type;
- title and description;
- description ref/hash when the full description is external or exceeds the bounded inline policy—never silent truncation;
- provider revision/digest and status at capture;
- parent/dependency/blocker refs;
- Project/Workstream/Workpoint binding;
- Acceptance Atom/evidence requirement refs;
- review-required/policy refs and closure-gate posture.

Provider title/description text is untrusted source data, not authority or agent instruction. It is rendered/quoted safely, separated from trusted review policy, and included in Judge Views only under the information-set policy. Later provider edits do not rewrite the frozen metadata; they create drift/revalidation posture.

### Review interactions

Authenticated UI, CLI, MCP, and API parity must support:

- request review;
- inspect assignment/rubric/artifact/Judge View;
- claim/release/reassign review;
- annotate/comment/cite exact evidence;
- accept;
- reject with typed reasons;
- request changes with finding/action refs;
- mark disputed/indeterminate;
- request recapture/reproof;
- preview decision side effects;
- commit decision;
- inspect immutable decision/Receipt;
- appeal/supersede/reopen through authorized operations.

Every decision cites exact claims/Acceptance Atoms, figures/media timestamps/diagnostics/Receipts, records reviewer identity/policy/information-set hash/time/confidence/limitations, and returns canonical commit status. Raw private chain-of-thought is never requested or stored.

### PWA UX

- prominent Review status beside truth/verification/settlement—not one ambiguous green badge;
- appointed reviewer, assignment freshness, required reviewers/quorum, and unresolved conflicts;
- evidence/rubric side-by-side with sticky decision controls;
- mandatory rejection/change reason and exact citations;
- preview of downstream effects before commit;
- committed Receipt and next action after commit;
- public/offline/unpaired views are read-only and explain how to open the exact Review Case in an authorized Focusa instance;
- mobile/keyboard/screen-reader-safe controls; no color-only decision state.

### Feed decisions back to the working agent

`rejected` or `changes_requested` must emit a bounded continuation/follow-up packet containing:

- exact artifact/revision and Review Decision refs;
- rejected/unsatisfied Acceptance Atoms;
- reviewer findings/citations;
- required recapture/reproof/repair actions;
- preserved Project/Workstream/Workpoint scope;
- action/authority constraints;
- canonical next safe action.

The working LLM receives this through Workpoint/Work Loop/context authority—not by treating report prose or comments as trusted prompt instructions. The agent re-observes current state before mutation and produces a new evidence/artifact revision for re-review.

### `bd` / `br` and provider closure gate

All provider close adapters must consult the canonical Completion Authority for the exact `bd`/`br` item ID/ref captured in artifact metadata before closure. Closure is refused when any applicable condition holds:

- formal Evidence Artifact absent/unresolved/hash-invalid/scope-mismatched/stale;
- required review absent, unassigned, expired, rejected, changes requested, disputed, or quorum incomplete;
- required Acceptance Atom not independently verified;
- Completion Decision not terminal-eligible;
- provider/repository/runtime state diverged;
- revalidation/reproof required;
- Receipt/settlement authority unavailable.

Blocked closure returns a typed result with exact missing/failed gates, Review Case/artifact refs, safe next actions, and recovery tools. It must not silently call `bd close`, `br close`, provider APIs, or reinterpret provider success as Focusa completion.

If a provider is closed outside Focusa authority, reconciliation records divergence and applies the approved reopen/quarantine/blocked-settlement policy; it never fabricates an accepted review.

An `accepted` review is a required input when policy says so, not a completion write. #278 verifies Acceptance Atoms; #277 alone decides/settles completion; #280 can invalidate/reopen; #274 governs Workset/release aggregation.

### Required contracts/operations/errors

Freeze before IR5:

- WorkItemBinding / ReviewRequirement / ReviewableItem / ReviewCase / ReviewerAssignment / ReviewDecision / ReviewFinding / ReviewReceipt schemas;
- assignment/claim/release/reassign/revoke/review/preview/commit/appeal/reopen operations;
- review_requested/assigned/claimed/decision_committed/rejected/changes_requested/disputed/expired/reproof_required events;
- reviewer_not_assigned, assignment_expired, independence_required, modality_missing, stale_revision, scope_mismatch, evidence_missing, quorum_incomplete, decision_conflict, closure_review_blocked, provider_diverged errors.

### Acceptance

1. Appointed human accepts through PWA and the same decision appears through CLI/API/MCP with one Receipt.
2. Appointed multimodal LLM judge rejects with exact image/video citations and the working agent receives a bounded follow-up packet.
3. Unappointed, expired, self-reviewing, scope-mismatched, stale, public, and offline actors cannot commit decisions.
4. Concurrent/conflicting reviewers converge to deterministic disputed/quorum posture.
5. `bd` and `br` closure fail closed for absent/rejected/changes-requested/disputed/expired review and show exact recovery.
6. Accepted review still cannot close when another Acceptance Atom or Completion gate fails.
7. External provider-close divergence is detected and reconciled without false completion.
8. Reproof creates a new immutable artifact revision and preserves prior review/decision lineage.
9. Approval controls appear only for applicable unresolved review-required items and only for the appointed reviewer; irrelevant/non-reviewable items expose no decision controls.
10. Partial item decisions derive the overall review posture deterministically; a global UI toggle cannot override them.
11. Exact `bd`/`br` IDs, title/description snapshot, revision/digest, relationships, Acceptance Atoms, review requirements, and closure posture are preserved as artifact metadata and provider text remains untrusted.

Implementation mapping: UIAI #106 T07 owns the artifact Review UI/action projection; Focusa #263/#277/#278/#280/#283/#291/#293 own canonical assignment, decision, continuation, provider-closure, and settlement paths. #294 readiness applies.
