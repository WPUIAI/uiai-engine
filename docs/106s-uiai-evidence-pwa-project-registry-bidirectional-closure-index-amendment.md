# 106s — UIAI Evidence PWA Project Registry, Bidirectional Closure Index Amendment

Status: normative amendment to `106`, `106a`–`106r`  
Primary implementation lane: T08-C Registry  
Scale/degradation dependency: T08-F  
Surface joins: CG-12, CG-13, CG-16, CG-21, CG-22  
Authority: UIAI indexes immutable artifact projections; Focusa alone owns Acceptance Atom verification, Completion Decision, provider closure, reopen, and settlement.

## 1. Purpose

A single Evidence Artifact detail article is necessary but not the primary operating surface. Each Project requires a compact, high-throughput Evidence Registry that can index and navigate thousands to hundreds of thousands of immutable artifacts while preserving exact Project → Workstream → Workset → CallGraph → Workpoint → Work Item → Acceptance Atom → Completion Case lineage.

The canonical `/evidence/` route opens the project-scoped registry. Selecting a row opens the existing Overview / Evidence / Timeline / Inspect / Developer detail article. The registry and detail view are two projections over the same immutable artifact identities; neither becomes a mutable completion truth store.

## 2. Storage and authority model

### 2.1 Immutable source, rebuildable project index

- Immutable manifests, bundles, assets, inspection receipts, verification artifacts, and canonical Receipts remain source records.
- Every Project has one versioned local registry database, logically named `evidence-index.sqlite3` unless a deployment supplies an equivalent adapter.
- The database is a derived index and relationship projection. It must be completely rebuildable from immutable manifests and canonical Focusa linkage/Receipt feeds.
- Index loss or corruption never deletes or rewrites canonical Evidence. The UI enters `degraded | rebuilding | stale_index | corrupt_index` explicitly.
- SQLite deployments use WAL mode, bounded busy timeout, transactional migrations, foreign keys, integrity checks, and atomic checkpoint/backup policy. Shared/server deployments may use another engine only behind the same versioned contract and conformance tests.
- No private filesystem path, credential, browser cookie, unrestricted provider payload, or raw chain-of-thought enters a public or portable index.

### 2.2 Minimum logical schema

The versioned schema MUST represent:

1. `artifact` — artifact ref, revision, manifest/bundle digests, title/descriptor, modality, availability, access, interaction, redaction, freshness, capture time, byte/media summaries, supersession/tombstone posture.
2. `scope_binding` — exact Project descriptor/ref/fingerprint, Workstream, Workset, CallGraph run/frame/attempt, Workpoint revision, Completion Contract revision, and source authority.
3. `work_item_binding` — every canonical T01 provider Work Item snapshot: provider surface, exact ID/ref, item type (task/epic/etc.), title, policy-safe description or description ref/digest, revision/digest, captured status, parent/dependency/blocker refs, Acceptance Atom/evidence/review requirement refs, and closure posture. An epic is a typed Work Item, never a second mutable hierarchy.
4. `artifact_edge` — one canonical typed edge record with source ref, target ref, relation type, source/target revisions, provenance Receipt, and observed time.
5. `acceptance_binding` — Acceptance Atom ref/revision, required evidence dimensions, bound artifact revisions, verification state, verifier class/independence, decision/Receipt refs, freshness, and stale/reopen posture.
6. `closure_binding` — Completion Case/Contract ref, Work Item ref, required Atom counts, satisfied/failed/blocked/stale counts, closure eligibility projection, canonical Completion Decision ref, provider-close Receipt ref, reopen ref, and settlement posture.
7. `collection` and `collection_member` — Project, Workstream, Workset/release, Workpoint, Acceptance Atom, Completion Case, incident, run, or user-saved grouping without changing artifact truth.
8. `tag_binding` — namespaced tags with provenance and policy; imported tags are untrusted display metadata until locally accepted.
9. `search_document` — normalized bounded searchable text and typed facets, including policy-safe Work Item type/title/description and relationship refs; large diagnostics/transcripts/media remain ref-first.
10. `index_meta` — schema revision, source cursor, rebuild cursor, last integrity check, stale reason, migration Receipt, and index digest.

### 2.3 Bidirectional relationship invariant

- Artifact ↔ Work Item, Artifact ↔ Acceptance Atom, Artifact ↔ Completion Case, Artifact ↔ Artifact, and Artifact ↔ Receipt navigation is mandatory in both directions.
- A relationship is recorded once as a canonical typed edge and exposed through forward and reverse queries; clients MUST NOT maintain two independently mutable copies.
- Every edge binds exact endpoint revisions/digests where the source contract provides them.
- Missing, stale, mismatched, superseded, or quarantined endpoint revisions are visible states and cannot silently resolve to a newer object.
- Deleting an index row cannot delete an immutable artifact. Reference-aware retention/GC operates only through canonical lifecycle policy and Receipts.

## 3. Task, Acceptance, and closure semantics

### 3.1 Required chain

Formal closure eligibility projects this chain:

`Project → Workstream → Workset → CallGraph run/frame/attempt → Workpoint revision → provider Work Item revision → Completion Contract revision → Acceptance Atom revision → accepted verification artifact revision → Completion Decision → provider-close Receipt → settlement posture`

### 3.2 Closure gate

- A Work Item can display `eligible_for_closure` only when every required Acceptance Atom is satisfied by a current, scope-matched, integrity-valid Evidence Artifact revision and all reviewer independence/quorum/revalidation requirements pass.
- An accepted Evidence Artifact is an input to completion; it never closes a task by itself.
- UIAI may index and display the closure projection but cannot write acceptance, completion, provider-close, reopen, or settlement truth.
- Only Focusa Completion Authority may commit Completion Decision, initiate eligible provider closure, reconcile provider state, and emit canonical Receipts.
- Changed Workset, CallGraph attempt, Workpoint, Work Item, Completion Contract, Acceptance Atom, artifact, review policy, runtime/deployment identity, or provider state invalidates stale closure projections and triggers refresh/reproof/reopen policy.
- The registry must support reverse questions efficiently: “Which artifacts prove this task/Atom?”, “Which tasks/Atoms depend on this artifact revision?”, and “What currently blocks closure?”

## 4. Compact registry UX

### 4.1 Default project list

`/evidence/` opens a dense project-scoped table/list, not a card gallery. Required structure:

- Project/Workstream scope breadcrumb and explicit access posture.
- One-line title/descriptor plus compact immutable ID.
- Configurable columns: state, modality, Work Item, Acceptance coverage, closure posture, verification, freshness, captured time, size, owner/source, Workpoint, CallGraph attempt, tags.
- Density modes targeting approximately 32 px compact, 40 px comfortable, and 48 px accessible rows.
- Sticky header, keyboard row navigation, visible focus, column resize/reorder/show-hide, deterministic sorting, and saved project views.
- Detail opens without losing registry query, cursor, selection, or scroll state.
- Mobile uses a compact semantic row layout with the same fields progressively disclosed; it does not render a desktop table wider than the viewport.

### 4.2 Search and filters

- `/` focuses search. Search is local/index-backed, cancellable, typo-tolerant where configured, and never scans binary assets synchronously.
- Required facets: Project, Workstream, Workset, Workpoint, Work Item, Acceptance Atom, Completion Case, modality, availability, access, verification, closure posture, freshness/staleness, time range, tags, source, and warning state.
- Exact ref/digest lookup bypasses fuzzy ranking and returns deterministic identity matches.
- Search results expose match reason and do not imply acceptance or completion.
- Stable keyset cursors use deterministic sort keys plus artifact identity; deep lists MUST NOT depend on unbounded offset pagination.
- Every response reports total posture (`exact | estimated | omitted`), active filters, cursor, page size, index revision, stale state, and omitted fields.

### 4.3 Bulk operations

Allowed bulk operations are typed and scope-bounded: select, tag proposal, collection membership, export, archive request, retention request, reproof request, compare, and authorized share/access-policy request.

- Bulk operations require preview, affected-count/scope summary, capability refresh, confirmation where policy requires it, idempotency key, per-item result, partial-effect truth, reconciliation, and Receipts.
- Bulk acceptance, Completion Decision, provider closure, reopen, settlement, destructive retention, or access widening cannot be inferred from row selection.
- No bulk operation may cross Project/authority boundaries silently.
- Static/public/offline projections are read-only and expose no mutation authority.

## 5. Performance and scale

### 5.1 Required budgets

Conformance fixtures cover 10,000 and 100,000 artifact rows with representative edges and Acceptance bindings.

On the reference local deployment after warm-up:

- exact ref lookup p95 ≤ 25 ms;
- scoped first-page list p95 ≤ 100 ms;
- indexed text/facet search p95 ≤ 150 ms;
- reverse Artifact ↔ Work Item/Atom lookup p95 ≤ 100 ms;
- initial semantic registry shell interactive ≤ 1 s excluding unavailable remote media;
- UI maintains responsive input and scroll without rendering more than a bounded viewport overscan.

Budgets must be reported with fixture size, hardware class, cold/warm state, index revision, and LowMem posture. A slower environment degrades truthfully; it does not fake compliance.

### 5.2 Rendering and resource policy

- Virtualized rows, bounded overscan, cursor paging, lazy thumbnails, intersection-based media loading, and cancellation are mandatory for large sets.
- Thumbnail/poster derivatives are browser-native, digest-bound, metadata-stripped, and optional; list usability never depends on full media decode.
- LowMem mode disables nonessential thumbnails/background work, reduces page size/overscan, preserves semantic rows, and reports omitted media.
- Offline mode serves the last verified bounded index snapshot with snapshot time/cursor and explicit staleness; it queues no acceptance/closure mutation.
- Corrupt/stale index recovery is resumable and rebuilds from immutable sources without making artifacts disappear silently.

## 6. API and projection contract

Required bounded operations, later generated through T11, include:

- `registry.list`
- `registry.search`
- `registry.facets`
- `registry.resolve`
- `registry.edges.forward`
- `registry.edges.reverse`
- `registry.collections.list`
- `registry.closure_projection`
- `registry.rebuild.status`
- typed bulk-operation preview/commit/status where authorized

REST/OpenAPI/CLI/MCP/Pi/Cockpit/Chrome/Desktop consumers use the same schema, stable cursors, field projection, ETags/digests, typed errors, and response profiles (`summary`, `agent_standard`, explicit `forensic`). Binary assets remain ref-first.

## 7. Security and privacy

- List/search authorization is independent of knowledge of an artifact URL.
- Public-safe registries contain only explicitly public-safe projections and resist enumeration, scraping, identifier guessing, and cross-project leakage.
- Private/unlisted artifact existence, counts, facets, tags, titles, and relationship edges do not leak through public totals or timing-sensitive error differences.
- Search text is prompt-injection-separated untrusted data. Rendering uses safe text/DOM APIs and strict CSP.
- Imported indexes and action metadata are untrusted; local policy regenerates eligible operations.

## 8. CallGraph refinement

- **T08-C Registry:** project DB schema/migrations, immutable indexer/rebuild, typed bidirectional edge graph, list/search/filter/facets/collections, compact responsive table, stable cursors, and exact lookup.
- **T08-F Scale/degradation:** 10k/100k fixtures, virtualization, lazy media, LowMem, offline snapshot, stale/corrupt/rebuild states, and performance evidence.
- **CG-12:** cannot pass on a single-artifact detail page; requires the project registry plus detail navigation and closure projections.
- **CG-13:** independent accessibility/security/performance verifier reviews table semantics, keyboard/bulk flows, authorization, CSP, index recovery, and scale budgets.
- **CG-16:** canonical packet API includes list/search/inspect/verify/resolve/edges/closure projection and restart/rebuild proof.
- **CG-21:** `/evidence/` durable mount defaults to registry; public fixture and private live databases remain separate trust classes.
- **CG-22:** all consumers prove the same artifact/task/Atom/closure edges and index revision.

## 9. Acceptance

1. `/evidence/` renders a compact project registry; one row opens the exact immutable detail article and back navigation restores list state.
2. 10k and 100k artifact fixtures remain searchable, filterable, sortable, and keyboard-usable within declared budgets.
3. Exact Project/Workstream/Workset/CallGraph/Workpoint/Work Item/Acceptance/Completion bindings are queryable forward and reverse; real task/epic type, title, policy-safe description/ref/digest, revision, parents, dependencies, blockers, requirements, and Acceptance refs survive index/rebuild/render parity.
4. A task with missing, rejected, stale, mismatched, or unverified required Evidence cannot display closure eligibility.
5. Accepted Evidence alone cannot write Completion or provider closure.
6. A superseded artifact revision invalidates dependent closure projection and exposes reproof/reopen requirements.
7. Index deletion/corruption followed by rebuild yields the same immutable artifact identities and edge digests.
8. Public-safe list/search proves no private/unlisted count, facet, title, tag, edge, or timing leakage.
9. Bulk operations prove preview/confirm/idempotency/per-item partial results and cannot cross scope/authority.
10. 375/768/1024/1440 pixel inspection shows world-class compact hierarchy, no page overflow, readable media, visible states, and no card-grid bloat.
11. WCAG 2.2 AA, localization/RTL, reduced motion, screen-reader table/list semantics, focus restoration, and LowMem/offline states pass.
12. Independent consumer and authority reviews pass before CG-13 closure.

## 10. Rollback

Disable the derived registry/index service and return `/evidence/` to direct immutable artifact resolution. Preserve all immutable artifacts, Receipts, and accepted schema migrations. Never roll back by rewriting artifact identities, dropping canonical Evidence, or granting UIAI completion authority.
