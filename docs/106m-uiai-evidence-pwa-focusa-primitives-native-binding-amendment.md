> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462695534

## Mandatory amendment — tight native binding to Focusa primitives

Evidence Artifacts/PWAs are first-class projections of the exact Focusa execution chain, not detached reports linked only by a loose project label.

### Ontological invariant

A formal **Focusa Evidence Artifact exists only as the immutable proof projection of a bound Focusa execution chain**. Project, Workstream, Workset membership, CallGraph run/frame/attempt, Workpoint revision, and work-item lineage are type-level identity—not optional integration metadata or a post-publication attachment.

An unbound screenshot, video, diagnostic packet, file, test output, or imported bundle is a `capture_candidate` only. It cannot be labeled a Focusa Evidence Artifact, exposed as closure-eligible, assigned to a canonical judge, or support `bd`/`br` closure until a governed intake/publication frame binds it to the required primitives.

Formal publication must be atomic with validated binding capture. If Focusa authority is unavailable, the response is `binding_blocked` with preserved local candidate refs and recovery; UIAI never fabricates the missing chain.

### Canonical relationship

```text
ProjectIdentity
→ Workstream / continuity
→ Trajectory / goal / waypoint context
→ Workset membership + requirement disposition
→ CallGraph definition/run/frame/path/attempt/generation
→ Workpoint revision/checkpoint/current action
→ provider Work Item (`bd`/`br` exact ID)
→ operation/action intent + observed effects
→ immutable Evidence Artifact
→ Focusa Evidence + Receipt linkage
→ independent judge CallGraph frame(s)
→ Acceptance Atom verification
→ Completion Decision
→ provider closure
→ Workset/release settlement
```

Each primitive keeps its own authority. UIAI stores immutable evidence bytes and safe typed refs/digests; it never creates a shadow mutable Workset, CallGraph, Workpoint, review, completion, or settlement store.

### Focusa binding envelope

Every artifact carries one bounded, revisioned binding snapshot:

- Project ref/identity fingerprint and working-subpath ref;
- Workstream/continuity ref;
- trajectory/MLG/STG/waypoint refs when applicable;
- Workset ref, revision/digest, membership ref, requirement/disposition refs;
- CallGraph definition ref/revision, run ref, frame/node/item/path refs, cycle, attempt, generation, parent/join/compensation refs;
- Workpoint ref, revision, checkpoint ref, current-action intent ref;
- exact provider Work Item bindings (`bd`/`br` IDs, descriptions, revision/digest, relationships, requirements);
- agent/team assignment, worker/model/harness/provider refs and assignment generation;
- operation/action-intent/effect/return/error refs;
- Completion Contract and Acceptance Atom refs/revisions;
- Evidence/Receipt/Review Case/Judge Result/Completion Decision/settlement refs;
- Context Cognition/active-object/Ontology refs where relevant;
- temporal/freshness/policy/capability/runtime-constitution refs;
- binding status, missing/applicable refs, revision digest, captured time, and rehydrate refs.

Use refs/digests, not copied mutable graphs or giant payloads. Public projections expose only policy-safe opaque refs/display labels.

### Binding and stale safety

Core identity bindings—Project, Workstream, Workset membership, CallGraph run/frame/attempt, Workpoint revision, and at least one work-item lineage—must be `matched` at formal publication. `missing | stale | mismatch` blocks formal publication and closure eligibility. `not_applicable` is forbidden for these core fields.

Auxiliary refs such as trajectory, specific ontology objects, external provider state, or optional temporal policy may declare `not_applicable` only when the governing contract explicitly allows it. Missing applicable auxiliary refs remain explicit and may degrade or block according to policy.

Any changed Workset revision, CallGraph generation/attempt, Workpoint revision, assignment lease, provider item revision, Completion Contract, Acceptance Atom, runtime/deployment identity, or review policy invalidates stale action/judge/closure inputs and triggers refresh/reproof/reopen policy.

### Autonomous execution through CallGraph

The no-human autonomous review loop must compile into ordinary governed CallGraph work:

```text
executor frame
→ evidence-publication frame
→ verifier/judge fanout frames
→ deterministic join/quorum frame
→ accepted ? completion-evaluation frame
           : repair/recapture frame → next cycle
→ provider-close frame
→ Workset settlement frame
```

- verifier frames have independent assignments/capability requirements;
- round/cycle/attempt/generation prevents stale judge results from settling a newer run;
- joins implement quorum/disagreement policy;
- repair loops are bounded by CallGraph cycle, budget, retry, cooldown, and circuit-breaker policy;
- partial/unknown effects route through reconciliation/compensation frames;
- no hidden PWA-local retry loop or agent recursion;
- human intervention, when chosen, is another authorized frame—not a required special plane.

### Workset integration

The artifact links to the exact Workset membership and requirement dispositions it can support. Publication alone does not change disposition. Canonical verification/Completion events may transition applicable requirements through the Workset reducer. Failed/missing/stale/disputed evidence keeps obligations open. Full release settles only when Workset completion and every required child settlement pass.

### Workpoint and working-agent continuation

Each rejected/changes-requested item emits a bounded continuation bound to the exact Workpoint revision/current action/provider item/CallGraph frame. The working agent resumes through Workpoint/Work Loop authority with reviewer citations and next safe action, re-observes current state, and produces a new attempt/artifact revision. It never follows raw PWA prose as instruction.

### PWA/Cockpit/agent navigation

Overview exposes Project → Workstream → Workset → CallGraph → Workpoint → Work Item breadcrumbs. Evidence, Timeline, Review, and Developer views permit authorized navigation to exact primitive projections. The Action Deck calls canonical Focusa operations for inspect/resume/reproof/reassign/continue/adjudicate/close; it never mutates primitive state itself.

CLI/MCP/API/Pi/LLM Judge Views return the same binding identity/revisions/digests and explicit expansion refs. A citation can target artifact, Workset requirement, CallGraph frame/attempt, Workpoint revision, provider item, Acceptance Atom, or Receipt precisely.

### Events and receipts

Artifact publication, review assignment/results, reproof, provider closure, and settlement emit canonical events/Receipts carrying exact binding refs. Lost responses/retries replay idempotently against the same run/frame/attempt/revision. Incompatible or orphaned events fail closed.

### Added acceptance

1. One artifact traces losslessly from Project/Workstream through Workset membership, CallGraph frame/attempt, Workpoint revision, exact `bd`/`br` item, evidence, judge result, Completion Decision, provider close, and Workset settlement.
2. A missing core primitive yields only `capture_candidate`/`binding_blocked`; no formal artifact ID, judge assignment, completion eligibility, or provider closure is fabricated.
3. UIAI contains no mutable shadow Workset/CallGraph/Workpoint/completion state.
4. Changing any applicable revision/generation invalidates stale review/action/closure and requires refresh/reproof.
5. Autonomous executor → judge fanout → quorum join → repair cycle → provider close runs entirely through CallGraph with bounded cycles and no human dependency.
6. Rejected review resumes the correct Workpoint/frame/item and cannot drift to another Workstream or generation.
7. Failed/partial/unknown effects reconcile or compensate before completion.
8. Public PWA exposes safe breadcrumbs but no private roots/topology; authorized views rehydrate exact primitives.
9. Workset/release settlement remains blocked while any required evidence/review/Completion child is open.
10. Imported evidence becomes formal only through a governed binding/intake frame preserving source provenance.

Implementation mapping: T01 stores the immutable bounded binding types; T04 captures runtime binding; T06/T07 use judge/review refs; T11 exposes parity; T13 performs canonical Focusa integration under #293/#294/#295; T14 proves installed end-to-end behavior.
