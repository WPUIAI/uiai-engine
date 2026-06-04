# UIAI × Focusa × Pi Hand-in-Glove Iterative Spec

**Status:** draft iterative product/engineering spec  
**Date:** 2026-06-04  
**HLT alignment:** make UIAI Engine the agent-compatible browser/intelligence platform that Pi can operate fluently and Focusa can use as evidence, Workpoint, trajectory, prediction, and metacognition substrate.  
**Primary repos inspected:**

- UIAI Engine: `/home/wpuiai/uiai-engine`
- Focusa core/Pi extension: `/home/wirebot/focusa`

## 1. Thesis

UIAI is most valuable with Focusa and Pi when it behaves as the **agent browser + research + diagnostics + proof engine**:

1. **Pi** is the fast operator/agent harness: it chooses UIAI tools, interacts with web pages, displays compact results, and keeps the UX thin.
2. **UIAI Engine** owns browser/search/session/media/diagnostics execution and emits bounded, stable evidence handles.
3. **Focusa** owns cognitive continuity: ProjectIdentity, Workpoint, Trajectory, evidence linkage, active-object hints, predictions, metacog lessons, and post-compaction recovery.

The winning product shape is not “more tools.” It is a deterministic route from user intent to browser/research action to bounded proof to Focusa continuity.

## 2. Current verified baseline

### UIAI baseline

- Persistent browser sessions, screenshots, reads, snapshots, actions, diagnostics, search, tool discovery, Pi extension, MCP bridge, CLI wrapper, and evidence handles are implemented/documented.
- `browser_open` accepts `focusa_scope`; diagnostics/session errors echo scope.
- Evidence handles exist for diagnostics, errors, search results, browser read/snapshot, screenshots, and shares.
- `/api/tools/graph` advertises related tools and Focusa handoff routes.
- Pi extension exposes compact-by-default UIAI tools and `/uiai` command surface.
- MCP bridge mirrors browser/session/search/core metadata tools.

### Focusa baseline

- Focusa Pi integration spec defines **Focusa as single cognitive authority** and Pi extension as thin UX glue.
- Focusa Pi contract allows Pi to emit typed proposals, action intents, evidence observations, failures, blockers, scratch records, and decision candidates; Pi must not become parallel memory.
- Focusa tools expose ProjectIdentity, Workpoint checkpoint/resume/evidence, Trajectory, active object resolution, diagnostics intake, prediction, metacog, traversal, and tool doctor flows.
- `focusa_browser_diagnostics_intake` is the preferred UIAI diagnostics wrapper: it captures bounded evidence, infers scope, emits active-object hints, records prediction candidates, and optionally captures metacog lessons.

## 3. Product target

### 3.1 One-command agent research + diagnostics packet

A Pi/MCP workflow should be able to do this in one guided route:

```text
intent -> project/workpoint scope -> search/open/read/snapshot -> diagnose -> evidence packet -> Focusa link -> prediction -> next action
```

Output shape should be a **ResearchDiagnosticsPacket**:

```json
{
  "schema": "uiai.focusa_research_diagnostics_packet.v1",
  "goal": "bounded user-visible goal",
  "focusa_scope": {
    "project_root": "/path/to/project",
    "continuity_id": "focusa-cont-...",
    "workpoint_id": "019..."
  },
  "target_refs": ["browser:https://example.test", "endpoint:/api/items"],
  "evidence_refs": [
    "uiai-search:brave:<query-hash>:1",
    "uiai-browser:session=<id>:read:<seq>",
    "uiai-diagnostics:session=<id>:seq=<seq>"
  ],
  "summary": "bounded result summary for Focusa evidence capture",
  "diagnostics_summary": {
    "console_errors": 0,
    "failed_requests": 0,
    "top_findings": []
  },
  "recommended_focusa_tools": [
    "focusa_browser_diagnostics_intake",
    "focusa_evidence_capture",
    "focusa_active_object_resolve",
    "focusa_predict_record"
  ],
  "recommended_next_action": "exact next browser/source/proof step"
}
```

### 3.2 User-facing Pi command concept

Add a guided Pi command or tool family, not necessarily a huge new engine primitive at first:

```text
/uiai research <query-or-url>
/uiai diagnose <url>
/uiai proof <url-or-session>
```

Minimum viable implementation can be a Pi extension workflow menu that chains existing tools and emits a packet. Later, UIAI HTTP can add a first-class `/api/agent/research-packet` endpoint if repeated use proves the packet shape.

## 4. Hand-in-glove contracts

### 4.1 Authority split

| Concern | Owner | Contract |
|---|---|---|
| Browser/search/session execution | UIAI | Run actions, return bounded output, stable evidence refs, diagnostics, redaction. |
| Cognitive continuity | Focusa | ProjectIdentity, Workpoint, Trajectory, evidence, predictions, metacog, resume packets. |
| Operator UX/tool selection | Pi | Compact tools, workflow menus, command hints, expandable JSON, no parallel memory. |
| Proof | Shared | UIAI generates handles; Focusa captures/links/evaluates; Pi displays route and next action. |

### 4.2 Scope contract

Every UIAI workflow that might become Focusa evidence should accept or infer:

```json
{
  "focusa_scope": {
    "project_root": "/path/to/project",
    "continuity_id": "focusa-cont-...",
    "workpoint_id": "019...",
    "evidence_ref": "optional caller seed"
  }
}
```

Rules:

- UIAI echoes scope in session info, diagnostics, errors, screenshots/shares, and future packet outputs.
- Focusa tools use `project_root + continuity_id` as authority; UIAI must treat scope as metadata, not authority.
- Pi should prompt/route from canonical Focusa WorkpointResumePacket when available.
- If scope is absent, UIAI still works, but packet output should say `scope_status=missing` and recommend `focusa_workpoint_resume` or `focusa_project_identity`.

### 4.3 Evidence contract

All UIAI agent outputs should prefer stable handles over blobs:

| Flow | Handle |
|---|---|
| Search result | `uiai-search:<provider>:<query-hash>:<rank>` |
| Browser read | `uiai-browser:session=<id>:read:<seq>` |
| Browser snapshot | `uiai-browser:session=<id>:snapshot:<seq>` |
| Diagnostics | `uiai-diagnostics:session=<id>:seq=<seq>` |
| Error | `uiai-error:<error_id>` |
| Screenshot | `uiai-screenshot:sha256:<prefix>` |
| Share | `uiai-share:<share_id>` |

Rules:

- Handle summaries must be bounded and redacted.
- Large page text, screenshots, HAR-like data, and raw SERP blobs stay out of Focusa by default.
- UIAI should return `preferred_focusa_tool` and `target_ref` with evidence-bearing results.

### 4.4 Tool choreography contract

Recommended default route for Pi:

```text
focusa_workpoint_resume or focusa_project_identity
  -> pi_uiai_agent_card / pi_uiai_tool_graph
  -> uiai_search or uiai_browser_open
  -> uiai_browser_read and/or uiai_browser_snapshot
  -> uiai_browser_diagnostics on uncertainty/failure
  -> focusa_browser_diagnostics_intake or focusa_evidence_capture
  -> focusa_active_object_resolve
  -> focusa_predict_record before patch/action
  -> verify with UIAI/tests
  -> focusa_predict_evaluate + metacog capture if reusable
  -> focusa_workpoint_checkpoint
```

Recommended route for UIAI failure:

```text
UIAI structured error envelope
  -> uiai_errors
  -> browser_diagnostics if session-related
  -> focusa_browser_diagnostics_intake
  -> active-object resolve
  -> prediction + fix/proof
```

## 5. Iterative implementation plan

### Iteration 0 — Spec alignment and invariants

**Goal:** make the contract explicit before adding features.

Deliverables:

- This spec.
- README/Session API link to this spec.
- Tool graph metadata names packet direction and Focusa handoff expectations.

Acceptance:

- Docs cite Focusa authority split and UIAI evidence-handle contract.
- No new paid/mutating surface exposed.

### Iteration 1 — Packet shape without new browser execution

**Goal:** compose a ResearchDiagnosticsPacket from existing UIAI calls.

Deliverables:

- Add a lightweight packet builder in Pi extension or `/api/tools/graph` workflow metadata.
- For existing search/read/diagnostics responses, include:
  - `target_ref`
  - `preferred_focusa_tool`
  - `recommended_next_action`
  - `focusa_scope_status`
- Add smoke proof using a harmless page.

Acceptance:

- Pi can produce a packet from search/open/read/diagnostics without raw blobs.
- Focusa can ingest packet summary via `focusa_evidence_capture` or diagnostics intake.

### Iteration 2 — Guided Pi workflows

**Goal:** make Pi feel like an integrated UIAI+Focusa cockpit.

Deliverables:

- `/uiai research` workflow menu: query -> search -> open selected -> read -> packet.
- `/uiai diagnose` workflow menu: open/reuse session -> diagnostics -> Focusa intake hint.
- Compact result renderer shows: `evidence_ref`, `target_ref`, `next`, `Focusa tool`.

Acceptance:

- Static Pi extension registration test covers new commands/tools.
- Runtime smoke verifies menu route against local engine.
- Existing `/uiai off/on` behavior remains green.

### Iteration 3 — Focusa-aware UIAI HTTP packet endpoint

**Goal:** promote packet from Pi composition to engine capability if Iteration 1/2 proves value.

Candidate endpoint:

```text
POST /api/agent/research-packet
```

Request:

```json
{
  "query": "optional search query",
  "url": "optional direct URL",
  "mode": "research|diagnose|proof",
  "max_results": 3,
  "max_read_chars": 4000,
  "focusa_scope": {}
}
```

Response: `uiai.focusa_research_diagnostics_packet.v1`.

Constraints:

- Do not automatically click/submit forms in this endpoint.
- Do not call paid/mutating APIs.
- Bound all text and diagnostics.
- Use search cache/redaction contracts.

Acceptance:

- HTTP, Pi, MCP, CLI parity entries exist or intentional omissions documented.
- Smoke covers degraded provider, successful read, diagnostics/no-failure path, and redaction.

### Iteration 4 — Active-object and prediction handoff upgrades

**Goal:** make Focusa better at turning UIAI evidence into source/action hypotheses.

Deliverables:

- Packet includes normalized hints:
  - URL hostname/path.
  - failed endpoint paths.
  - selector/ref labels.
  - JS source locations when available.
  - title/snippet/source for selected search result.
- UIAI graph recommends `focusa_active_object_resolve` with a compact hint string.
- Pi workflow can optionally call `focusa_predict_record` before code changes.

Acceptance:

- Focusa active-object resolution receives useful target hints without raw diagnostics.
- Prediction/evaluation loop appears in release/task report.

### Iteration 5 — Reliability and observability

**Goal:** make the combined system robust under long agent runs.

Deliverables:

- UIAI health/metrics include packet/search/browser pressure summaries.
- Focusa `tool_doctor`/project card can surface UIAI browser pressure and recommend narrowing.
- Add alerts or dashboard hooks for browser queue, search failures, cache hit rate, diagnostics error classes.

Acceptance:

- Long-running Pi/Focusa workflow has bounded cleanup: close sessions, clear diagnostics only when safe, checkpoint Workpoint.
- Observability catches provider degradation and browser pool pressure before agent failure loops.

## 6. Design details

### 6.1 Pi result rendering

Pi output should prioritize one-line usefulness:

```text
UIAI research packet evidence=3 target=browser:https://... next="capture Focusa evidence then verify source"
```

Expanded JSON remains available. Error compact line should include:

```text
UIAI 500 /api/session/<id>/click class=selector_not_found id=uiai-error-... -> run browser_snapshot or diagnostics
```

### 6.2 UIAI tool graph additions

`/api/tools/graph` should eventually include:

```json
{
  "packet_workflows": [
    {
      "name": "focusa_research_packet",
      "steps": ["focusa_workpoint_resume", "uiai_search", "uiai_browser_open", "uiai_browser_read", "uiai_browser_diagnostics", "focusa_evidence_capture"]
    }
  ],
  "packet_schema": "uiai.focusa_research_diagnostics_packet.v1"
}
```

### 6.3 Focusa integration metadata per UIAI response

Evidence-bearing responses should include optional metadata:

```json
{
  "focusa": {
    "target_ref": "browser:https://example.test",
    "evidence_ref": "uiai-browser:session=abc:read:2",
    "preferred_tool": "focusa_evidence_capture",
    "next_tools": ["focusa_active_object_resolve", "focusa_predict_record"],
    "summary": "bounded capture summary"
  }
}
```

This is more useful than only listing `related_tools` because it gives Focusa-ready tool arguments.

### 6.4 Security/redaction invariants

- No provider keys, bearer tokens, cookies, license keys, webhook secrets, raw auth headers, or raw request bodies in packets.
- Strip URL fragments and redact secret-like query keys.
- Remote access remains authenticated for loopback-public routes.
- Paid/mutating API families remain omitted from Pi/MCP until a concrete workflow, auth boundary, cost policy, redaction, and smoke proof exist.

### 6.5 Cleanup invariants

Every guided workflow must say whether a browser session remains open.

Packet field:

```json
{
  "cleanup": {
    "session_id": "abc",
    "recommended_action": "close_when_done|keep_for_followup",
    "tool": "uiai_browser_close"
  }
}
```

## 7. Second-pass audit gap analysis

This section captures the audit-only pass performed after the initial spec. It is intentionally a **certainty gate**: do not build from this list until the owning component, acceptance proof, and no-build constraints are clear.

### 7.1 High-confidence gaps

1. **ResearchDiagnosticsPacket missing:** current tools expose handles and graph routes, but no single packet bundles goal, scope, target refs, evidence refs, diagnostic summary, next Focusa tool, and cleanup.
2. **Ready Focusa arguments missing in UIAI response metadata:** `/api/tools/graph` names preferred Focusa tools, but most evidence-bearing responses do not include ready-to-call `target_ref`, `evidence_ref`, `preferred_tool`, `summary`, and `next_tools` metadata.
3. **Pi workflow still manual:** 37 UIAI Pi tools are present, but research/diagnose/proof are not first-class guided Pi flows.
4. **Focusa packet intake broader than diagnostics is absent:** `focusa_browser_diagnostics_intake` is strong for diagnostics/failure envelopes, but there is no broader intake path for search/read/snapshot/error bundles.
5. **Active-object hints are shallow:** diagnostics and search outputs can produce better compact hints for URL path, failed endpoint, selector label/ref, JS source location, and selected search-result source.
6. **Observability can deepen:** browser health/metrics and smoke/soak proof exist, but browser/search/cache/error trend summaries are not yet surfaced as long-term paired Focusa/Pi health guidance.
7. **Focusa project-card crosswire risk:** a project-card read returned stale-looking unrelated HLG text while `project_verify` and `trajectory_view` were correct; use trajectory + Workpoint + repo evidence as authority until card context is checked.
8. **WordPress parity remains outside the packet path:** training service-token mismatch, structured-error preservation, and screenshot health route naming are still product-facing gaps.
9. **Non-browser API exposure remains gated:** reference/admin/memory/workflow/intelligence/training/captcha routes still require workflow, auth, cost, redaction, and smoke proof before Pi/MCP/CLI exposure.
10. **Search/provider future gaps remain:** Brave is the only implemented provider; explicit local provider rate limiting and future-provider parity are not yet proven.
11. **Docs/proof automation missing:** README, Session API, gap inventory, tool graph, Pi/MCP/CLI surfaces are updated manually; no CI drift check enforces alignment.
12. **Focusa/Pi hook maturity is relevant:** Focusa specs call out richer Pi lifecycle/provider/tool hooks; those may matter before automatic packet/proof capture during long Pi runs.

### 7.2 Certainty matrix

| Gap | Owner | Certainty | Build decision gate | Acceptance proof |
|---|---|---:|---|---|
| ResearchDiagnosticsPacket | UIAI first, Pi wrapper second | High | Exact schema accepted; no automatic paid/mutating actions | Harmless search/open/read/diagnostics packet smoke; bounded/redacted JSON |
| Ready Focusa args in responses | UIAI | High | Decide field names and target responses before code | Unit/API smoke shows `focusa.target_ref/evidence_ref/preferred_tool/summary` |
| Guided `/uiai research` / `/uiai diagnose` | Pi | Medium-high | Packet metadata exists or workflow can compose it without hidden state | Pi registration/runtime smoke; compact result includes evidence and next Focusa tool |
| Broader Focusa packet intake | Focusa | Medium | Determine whether existing evidence/diagnostics tools are sufficient | Dry-run intake or evidence capture accepts packet summary without raw blobs |
| Active-object hints | UIAI + Focusa | High | Define normalized hint fields and redaction rules | Focusa active-object resolve receives URL/endpoint/selector/source hints |
| Observability summaries | UIAI + Focusa doctor/project-card | Medium | Define which metrics predict agent failure loops | Health/metrics expose pressure; Focusa doctor summarizes `uiai_browser` and trend risk |
| Project-card crosswire risk | Focusa | Medium-high | Diagnose card HLG/context source before trusting card route for UIAI planning | Project card aligns with verified project/trajectory or marks advisory mismatch |
| WordPress parity | WP plugin + UIAI | High | Separate product-flow plan; not part of packet MVP | Plugin preserves structured errors; service-token route either wired or disabled |
| Non-browser exposure | UIAI/Pi/MCP | High | Concrete workflow + auth/cost/redaction proof per family | Exposure inventory updated; auth/redaction/smoke passes |
| Provider expansion/rate limiting | UIAI | Medium | Evidence of provider pressure or second provider need | Degraded/cache/redaction/evidence contract passes for provider N |
| Docs drift automation | UIAI CI | Medium | Choose drift assertions that avoid brittle docs tests | CI fails when surface changes lack README/Session API/gap inventory updates |
| Pi hook maturity | Focusa/Pi | Medium | Decide if automatic packet capture needs lifecycle hooks | Hook telemetry proves capture points without raw message/tool blobs |

### 7.3 No-build constraints before Iteration 1

Before implementation starts, answer these:

1. Which existing responses get Focusa metadata first: search, browser read, diagnostics, errors, screenshot/share, or all evidence-bearing responses?
2. Is the packet composed in Pi first, or returned by a UIAI HTTP endpoint first?
3. Does Focusa need a new packet intake tool, or should packet summaries route through `focusa_evidence_capture` and `focusa_browser_diagnostics_intake`?
4. What is the exact stable packet schema and maximum size budget?
5. Which proof path demonstrates value without paid/mutating calls?
6. How should stale/cross-project project-card signals be detected and reported?

## 8. Acceptance matrix

| Iteration | Must prove |
|---|---|
| 0 | Spec linked; authority/evidence contracts documented; second-pass certainty matrix recorded. |
| 1 | Packet can be assembled from existing search/open/read/diagnostics without raw blobs and with ready Focusa argument metadata. |
| 2 | Pi guided workflow creates packet and recommends Focusa capture without becoming parallel memory. |
| 3 | Optional HTTP packet endpoint has auth/redaction/degraded smokes and does not perform paid/mutating actions. |
| 4 | Packet hints improve active-object/prediction flow with URL/endpoint/selector/source evidence. |
| 5 | Health/metrics expose enough pressure to prevent browser/search/cache/error failure loops. |

## 9. Proposed next beads

1. `Decide: ResearchDiagnosticsPacket schema and response surfaces`.
2. `Packet: add Focusa metadata object to browser_read/search/diagnostics responses`.
3. `Smoke: compose packet from existing tools without new endpoint`.
4. `Pi: add /uiai research and /uiai diagnose guided workflows`.
5. `Focusa: evaluate whether packet needs a dedicated intake tool`.
6. `Graph: expose packet workflow schema and ready Focusa argument hints`.
7. `Observe: add browser/search/cache pressure summary for Focusa tool doctor/project card`.
8. `CI: add docs/surface drift check after packet metadata lands`.

## 10. Decision rule

Prefer the smallest iteration that reduces agent friction while preserving the Focusa authority split:

- If a feature only improves display, implement in Pi.
- If it creates durable browser/search/diagnostics evidence, implement in UIAI.
- If it changes memory, Workpoint, trajectory, prediction, or lessons, implement in Focusa.
- If it crosses boundaries, expose stable handles and ready-to-call arguments, not raw blobs or hidden writes.
- If a gap lacks owner, schema, proof, or safety boundary, keep it in audit/spec mode instead of building.
