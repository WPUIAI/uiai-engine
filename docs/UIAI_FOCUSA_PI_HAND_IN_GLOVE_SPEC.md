# UIAI × Focusa × Pi Hand-in-Glove Iterative Spec

**Status:** draft iterative product/engineering spec  
**Date:** 2026-06-04  
**HLT alignment:** make UIAI Engine the agent-compatible browser/intelligence platform that Pi can operate fluently and Focusa can use as evidence, Workpoint, trajectory, prediction, and metacognition substrate.  
**Primary repos inspected:**

- UIAI Engine: `/home/wpuiai/uiai-engine`
- Focusa core/Pi extension: `/home/wirebot/focusa`
- Pi docs: `README.md` plus every file in `docs/*.md` under the installed Pi package.
- Focusa docs/code: `docs/52-pi-extension-contract.md`, `docs/current/PI_EXTENSION_AND_SKILLS_GUIDE.md`, `docs/current/UIAI_BROWSER_DIAGNOSTICS_FOCUSA_INTEGRATION_SPEC.md`, `docs/current/FOCUSA_TOOL_CHOREOGRAPHY_MAP.md`, `docs/current/FOCUSA_TOOL_CONTRACT_REGISTRY.md`, `apps/pi-extension/src/{tools,state,turns,session,commands,awareness,polish}.ts`.

**Schema decision companion:** [`RESEARCH_DIAGNOSTICS_PACKET_SCHEMA_DECISION_MATRIX.md`](RESEARCH_DIAGNOSTICS_PACKET_SCHEMA_DECISION_MATRIX.md)

## 1. Thesis

UIAI is most valuable with Focusa and Pi when it behaves as the **agent browser + research + diagnostics + proof engine**:

1. **Pi** is the fast operator/agent harness: it chooses UIAI tools, interacts with web pages, displays compact results, and keeps the UX thin.
2. **UIAI Engine** owns browser/search/session/media/diagnostics execution and emits bounded, stable evidence handles.
3. **Focusa** owns cognitive continuity: ProjectIdentity, Workpoint, Trajectory, evidence linkage, active-object hints, predictions, metacog lessons, and post-compaction recovery.

The winning product shape is not “more tools.” It is a deterministic route from user intent to browser/research action to bounded proof to Focusa continuity.

### 1.1 Refined STG for this iteration

**STG:** turn UIAI's existing search/browser/diagnostics surfaces and Focusa's existing ProjectIdentity/Trajectory/Workpoint/evidence/prediction/metacog tools into a Pi-native guided workflow that produces bounded ResearchDiagnosticsPackets, renders compactly in Pi, survives compaction through Workpoints, and remains usable through Pi RPC/JSON/MCP/CLI without TUI-only assumptions.

This STG deliberately avoids a big-bang autonomous endpoint build. The implemented iteration adds metadata and packet composition where the source-backed systems already have affordance:

1. Pi extension commands/renderers orchestrate the UX.
2. UIAI Engine emits stable evidence handles and Focusa-ready metadata.
3. Focusa core keeps authority over continuity, evidence, trajectory, predictions, metacog, and route recovery.
4. MCP/CLI/RPC reuse the same packet schema rather than receiving a second-class format.

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
- `focusa_browser_diagnostics_intake` is the preferred UIAI diagnostics wrapper: it consumes existing UIAI diagnostics/error envelopes, captures bounded evidence, infers scope, emits active-object hints, records prediction candidates, and optionally captures metacog lessons; it does not call UIAI or open browser targets.

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

Minimum viable implementation includes both a Pi extension workflow menu that chains existing tools and a first-class HTTP composer at `/api/agent/research-packet` for Pi/MCP/CLI parity. The HTTP endpoint composes only from caller-supplied UIAI responses; it does not open paid or mutating browser actions itself.

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

### 4.5 Pi-docs-informed Focusa/Pi deep integration pass

This pass tightens the original STG from “add a packet/workflow” to “make Pi, UIAI, and Focusa cooperate through the native extension/session/compaction model that Pi already documents.”

#### Source-backed observations

| Source | Observed capability | Hand-in-glove implication |
|---|---|---|
| Pi docs: extensions/custom tools | Extensions can register tools, commands, providers, events, custom renderers, UI, autocomplete, settings/status widgets, and compaction/tree hooks; mutating custom tools should use Pi's file mutation queue. | Focusa Pi plugin should remain a thin extension that registers ergonomic tools/commands and never stores parallel memory; UIAI should expose stable packet handles and metadata rather than asking the extension to reconstruct durable state. |
| Pi docs: sessions/compaction/session-format | Pi sessions are tree-structured JSONL; compaction and branch summaries preserve structured context, file operations, and extension hook state. | Focusa WorkpointResumePacket should be the canonical continuity object injected before/after compaction; UIAI packet refs should be evidence handles inside Workpoint, not raw transcript blobs. |
| Pi docs: TUI/custom rendering | Tool result renderers, widgets, status lines, overlays, SelectList/SettingsList, and compact/expanded output are first-class. | `/uiai research` and `/uiai diagnose` should render one compact cockpit line plus expandable packet JSON; Focusa state should show only next route/evidence status, not full diagnostics. |
| Pi docs: RPC/JSON/SDK | Pi can be embedded or controlled headlessly, with extension UI degraded in RPC and JSON event streams for tool/message lifecycle. | Packet workflows must work without TUI-only assumptions: HTTP/MCP/CLI/RPC paths need the same stable schema, while Pi TUI only improves selection/display. |
| Pi docs: packages/skills/prompts | Extensions, skills, prompts, themes can ship as packages; skills use progressive disclosure and `/skill:name`. | Focusa-facing UIAI behavior should be packaged as extension + skills + docs, with skill descriptions routing agents to diagnostics/packet flows before raw screenshots or guesses. |
| Focusa contract docs | Pi may emit proposals, action intents, observations, failures, blockers, scratch, and decision candidates, but not canonical ontology writes or long-lived memory. | UIAI packet output must be a typed proposal/evidence observation; Focusa decides Workpoint/evidence/prediction/metacog writes. |
| Focusa tool choreography/registry | 63 tools span project identity, trajectory, Workpoint, traversal/reflexes, evidence, prediction, metacog, work-loop, diagnostics, resource mode, and session transfer; `tool_result_v1` carries recovery and next tools. | UIAI responses should include ready-to-call Focusa argument metadata and `next_tools`; Pi should follow Focusa's graph instead of inventing route logic. |
| Focusa Pi extension code | Current plugin tracks current ask, project switch ledger, attention recall, tool-output flood, trajectory slice, Workpoint packet, visible recap, widgets/status, before-agent context injection, input/turn/tool-result hooks, and project-root safety. | New UIAI workflows should hook into existing state: pass `focusa_scope`, update no parallel memory, use Focusa tool envelopes, and reduce prompt/tool-output flood by emitting handles. |
| Focusa UIAI diagnostics guide | `focusa_browser_diagnostics_intake` already turns UIAI diagnostics into bounded evidence, active-object hints, prediction context, and optional metacog. | The packet MVP should reuse this wrapper for diagnostics and add a broader packet path only for search/read/snapshot bundles. |

#### Integration design conclusions

1. **Pi command first, Focusa authority always:** implement guided `/uiai research|diagnose|proof` as Pi workflows that call UIAI and recommend Focusa tool calls; they should not mutate Focusa except through explicit Focusa tools.
2. **Packet as evidence proposal:** `ResearchDiagnosticsPacket` is a typed evidence/action proposal with `focusa` metadata, not a new memory store.
3. **Use Pi session mechanics instead of fighting them:** packet refs, Workpoint ids, ProjectIdentity, and next action should survive compaction through `focusa_workpoint_checkpoint/resume`, not hidden extension globals.
4. **Render compact, expand on demand:** Pi TUI should show `goal`, `target_ref`, `evidence_ref_count`, `diagnostics_status`, `preferred_focusa_tool`, and `next`; expanded JSON remains available.
5. **Headless parity from day one:** UIAI packet schema must be usable from Pi TUI, Pi RPC/JSON, MCP, and CLI with the same bounded handles.
6. **Focusa tool graph as route policy:** use `focusa_project_identity -> trajectory_view -> workpoint_resume/checkpoint -> evidence/intake -> active_object -> prediction/metacog` as default choreography; operator steering still wins.
7. **Resource safety:** avoid full lineage/ontology/diagnostics payloads in packet workflows; Focusa `tool_doctor` reads only UIAI browser health/metrics (`/api/health/browser`, `/api/metrics/browser`) for pressure, while `traverse`/`resource_mode` manage Focusa-side load.

#### Required plugin/core upgrades discovered by the pass

| Upgrade | Component | Why it matters |
|---|---|---|
| Packet workflow command surface | Pi extension | Pi docs make commands/workflows the right UX layer; UIAI execution stays engine-owned. |
| Packet result renderer | Pi extension | One-line cockpit output prevents tool-output flood and respects Focusa visible-output boundaries. |
| Focusa-ready metadata in UIAI results | UIAI Engine | Eliminates manual argument reconstruction for `focusa_evidence_capture`, diagnostics intake, active-object resolve, and prediction. |
| Workpoint-aware packet checkpoint hints | Focusa Pi extension + Focusa core | Keeps packet refs canonical across compaction/model switch/fork. |
| Broader packet intake decision | Focusa core | Diagnostics intake exists; search/read/snapshot packet bundles may need either a new intake tool or stricter `focusa_evidence_capture` conventions. |
| Headless schema parity | UIAI + Pi extension + MCP/CLI | RPC/JSON modes cannot rely on TUI custom UI; packet schema must carry all next-step data. |
| Tool graph route weights for UIAI packets | Focusa core | Choreography registry can rank UIAI packet flows as evidence/proof routes instead of ad hoc tool choice. |

#### Non-negotiable constraints from this pass

- Focusa remains the single cognitive authority; Pi and UIAI emit typed, bounded proposals and evidence handles.
- Focusa does not bypass UIAI auth, redaction, or URL safety; UIAI `url_not_allowed` is captured as policy evidence, not treated as a Focusa failure.
- UIAI packet workflows must never inline raw screenshots, raw HAR, auth headers, cookies, or large page bodies.
- The Pi extension must not create long-lived local packet memory beyond Pi session/Workpoint handles and explicit Focusa state.
- Guided workflows must degrade cleanly in Pi RPC/JSON/SDK contexts where TUI dialogs/custom components are no-ops or protocol-mediated.
- Existing uncommitted Focusa plugin/core changes must be treated as other-agent work until owned explicitly.

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

**Goal:** compose a ResearchDiagnosticsPacket from existing UIAI calls and Focusa-ready response metadata.

Deliverables:

- Add a lightweight packet builder in the Pi extension and a composer-only HTTP endpoint backed by `/api/tools/graph` workflow metadata.
- For existing search/read/snapshot/diagnostics/error/screenshot/share responses, include:
  - `focusa.target_ref`
  - `focusa.evidence_ref`
  - `focusa.preferred_tool`
  - `focusa.summary`
  - `focusa.next_tools`
  - `recommended_next_action`
  - `focusa_scope_status`
- Add packet fields for Pi/RPC parity: `render.summary_line`, `expandable_json_ref`, `cleanup`, and `headless_next_action`.
- Add smoke proof using a harmless page.

Acceptance:

- Pi can produce a packet from search/open/read/diagnostics without raw blobs.
- Focusa can ingest packet summary via `focusa_evidence_capture` or diagnostics intake.
- Packet output is useful in Pi TUI and still complete in Pi RPC/JSON/MCP/CLI contexts.

### Iteration 2 — Guided Pi workflows

**Goal:** make Pi feel like an integrated UIAI+Focusa cockpit.

Deliverables:

- `/uiai research` workflow menu: query -> search -> select/open -> read/snapshot -> diagnostics when needed -> packet.
- `/uiai diagnose` workflow menu: open/reuse session -> diagnostics -> `focusa_browser_diagnostics_intake` hint or dry-run payload.
- `/uiai proof` workflow menu: URL/session/evidence refs -> verification read/diagnostics -> packet -> Workpoint checkpoint suggestion.
- Compact result renderer shows: `evidence_ref`, `target_ref`, `scope_status`, `next`, `Focusa tool`; expanded JSON remains available.
- Headless behavior emits the same packet and `headless_next_action` without requiring TUI selections.

Acceptance:

- Static Pi extension registration test covers new commands/tools.
- Runtime smoke verifies menu route against local engine.
- Existing `/uiai off/on` behavior remains green.
- Focusa plugin visible-output boundaries remain green: no raw Focusa slice, raw diagnostics, or unbounded packet text in the compact view.

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

### 7.3 Build-ready decisions for Iteration 1

This section resolves the previous no-build questions. Iteration 1 may start when these choices are accepted by the implementation owner.

1. **First response surfaces:** start with `search`, browser `read`, browser `snapshot`, browser `diagnostics`, structured `errors`, `screenshot`, and `share` to Iteration 1.5/2 unless they are already returned by the chosen proof path.
2. **Packet composition location:** compose in the Pi extension for local Pi workflows and expose `POST /api/agent/research-packet` for HTTP/MCP/CLI parity. The endpoint accepts existing UIAI responses and returns the same bounded packet schema.
3. **Focusa intake path:** do not add a new Focusa packet intake tool for Iteration 1. Route research/read/search summaries through `focusa_evidence_capture`; route diagnostics/failure envelopes through `focusa_browser_diagnostics_intake`; use `focusa_active_object_resolve` and `focusa_predict_record` as explicit follow-up tools.
4. **Packet size budget:** packet JSON should stay under 8 KB by default. Individual capture summaries max 500 chars; `goal` and `recommended_next_action` max 240 chars; `diagnostics_summary` max 2 KB; `args_preview` max 2 KB; max 8 captures, 16 target refs, 32 evidence refs, and 16 active-object hints.
5. **Stable schema:** use `uiai.focusa_research_diagnostics_packet.v1` with required fields `schema`, `mode`, `goal`, `scope_status`, `target_refs`, `evidence_refs`, `captures`, `recommended_focusa`, `recommended_next_action`, and `cleanup` when a browser session exists.
6. **Proof path:** use harmless search/open/read/diagnostics only; no paid provider requirement, no form submission, no auto-click except opening a selected URL. Proof passes when packet has handles, redaction, Focusa args preview, and explicit cleanup.
7. **Project-card crosswire handling:** exclude project-card automation from Iteration 1. If card/trajectory/project signals conflict, Pi must prefer `focusa_project_verify`, `focusa_project_identity`, `focusa_trajectory_view`, and canonical WorkpointResumePacket over project-card text.

#### 7.3.1 Iteration 1 packet schema

```json
{
  "schema": "uiai.focusa_research_diagnostics_packet.v1",
  "mode": "research|diagnose|proof",
  "goal": "bounded user-visible goal",
  "scope_status": "present|missing|partial|mismatch_candidate",
  "focusa_scope": {
    "project_root": "/path/to/project",
    "continuity_id": "focusa-cont-...",
    "workpoint_id": "019...",
    "evidence_ref": "optional caller seed"
  },
  "target_refs": ["browser:https://example.test"],
  "evidence_refs": ["uiai-search:brave:<query-hash>:1"],
  "captures": [
    {
      "type": "search|read|diagnostics|error",
      "evidence_ref": "stable UIAI handle",
      "target_ref": "browser:https://example.test",
      "title": "bounded title",
      "summary": "<=500 chars, redacted"
    }
  ],
  "diagnostics_summary": {
    "console_errors": 0,
    "failed_requests": 0,
    "top_findings": []
  },
  "active_object_hints": [
    { "kind": "url|endpoint|selector|source|search_result", "hint": "bounded candidate hint" }
  ],
  "recommended_focusa": {
    "preferred_tool": "focusa_evidence_capture|focusa_browser_diagnostics_intake",
    "fallback_tool": "focusa_evidence_capture",
    "args_preview": {
      "target_ref": "primary target_ref",
      "result": "bounded packet summary",
      "evidence_ref": "primary evidence_ref",
      "attach_to_workpoint": false
    },
    "next_tools": ["focusa_active_object_resolve", "focusa_predict_record"]
  },
  "recommended_next_action": "exact next browser/source/proof step",
  "render": {
    "summary_line": "UIAI packet evidence=3 target=browser:https://... next=...",
    "expandable_json_ref": "local temp/ref when available"
  },
  "headless_next_action": "same next action without TUI assumptions",
  "cleanup": {
    "session_id": "optional",
    "recommended_action": "close_when_done|keep_for_followup|none",
    "tool": "uiai_browser_close"
  }
}
```

#### 7.3.2 Iteration 1 redaction rules

- Strip URL fragments.
- Redact secret-like query keys: `token`, `key`, `secret`, `auth`, `session`, `password`, `code`, `sig`, `signature`, `jwt`.
- Never include cookies, auth headers, raw request/response bodies, raw screenshots/base64, HAR-like dumps, or unbounded page text.
- Prefer URL paths over full failed URLs when the host is not needed for diagnosis.
- Mark `active_object_hints` as candidates; Focusa/agent must verify them against source/tests before treating them as canonical.

#### 7.3.3 Iteration 1 smoke proof

```text
focusa_workpoint_resume or focusa_project_identity when scoped
uiai_search query="UIAI Engine browser agents" limit=1
uiai_browser_open selected URL with optional focusa_scope
uiai_browser_read max_chars<=2000
uiai_browser_diagnostics failed_only=false
compose uiai.focusa_research_diagnostics_packet.v1 locally in Pi extension
focusa_evidence_capture attach_to_workpoint=false unless canonical Workpoint scope is present
uiai_browser_close when cleanup.recommended_action=close_when_done
```

Smoke acceptance:

- Packet is under 8 KB.
- Evidence handles are present for search/read/diagnostics when those steps run.
- Compact Pi render is one line plus expandable JSON/ref.
- `recommended_focusa.args_preview` can be copied into the named Focusa tool without manual reconstruction.
- No raw screenshots, cookies, auth headers, request bodies, or unbounded page text appear.
- Cleanup recommendation is explicit.

Smoke proof recorded 2026-06-04 with `scripts/smoke-focusa-packet.sh` against a source-built local engine on port 7466: output `/tmp/uiai-focusa-packet-smoke.json`, packet size 2922 bytes, captures `search`, `read`, and `diagnostics`, `search_ran=true`, cleanup explicit.

### 7.4 Focusa packet intake friction gate

Iteration 1 does **not** add a new Focusa packet intake tool. Existing tools remain sufficient when all of these are true:

- `recommended_focusa.args_preview` can be copied directly into `focusa_evidence_capture` or `focusa_browser_diagnostics_intake` without manually reconstructing `target_ref`, `result`, or `evidence_ref`.
- Packet captures are <=8 and packet JSON is under 8 KB after redaction.
- Diagnostics/failure packets have `preferred_tool=focusa_browser_diagnostics_intake`; research/read packets have `preferred_tool=focusa_evidence_capture`.
- `active_object_hints` are candidates only and can route to `focusa_active_object_resolve` without becoming canonical.
- Smoke proof does not require hidden Focusa writes or a parallel packet memory store.

A dedicated Focusa packet intake tool becomes justified only if at least two of these happen repeatedly during smoke/guided workflow use:

1. Agents must manually split one packet into multiple Focusa calls to preserve evidence.
2. `args_preview` loses required context for Workpoint/evidence/prediction linkage.
3. Diagnostics plus search/read bundles need atomic capture semantics.
4. Active-object/prediction follow-up is consistently skipped because the route is too manual.
5. Packet summaries regularly exceed 8 KB after normal trimming.

Current evaluation from `/tmp/uiai-focusa-packet-smoke.json` (2026-06-04): no new intake tool needed; packet size was 2922 bytes, `args_preview` was present, search/read/snapshot/diagnostics captures were included, and existing `focusa_evidence_capture` / `focusa_browser_diagnostics_intake` routing was enough for the proof artifact.

## 8. Acceptance matrix

| Iteration | Must prove |
|---|---|
| 0 | Spec linked; authority/evidence contracts documented; second-pass certainty matrix recorded. |
| 1 | Packet can be assembled from existing search/open/read/diagnostics without raw blobs and with ready Focusa argument metadata. |
| 2 | Pi guided workflow creates packet and recommends Focusa capture without becoming parallel memory. |
| 3 | Optional HTTP packet endpoint has auth/redaction/degraded smokes and does not perform paid/mutating actions. |
| 4 | Packet hints improve active-object/prediction flow with URL/endpoint/selector/source evidence. |
| 5 | Health/metrics expose enough pressure to prevent browser/search/cache/error failure loops. |

## 9. Implementation bead decomposition

Root epic: `uiai-engine-3ds` — Implement UIAI Focusa Pi hand-in-glove packet workflow.

### 9.1 Iteration 1 core packet path

1. `uiai-engine-3ds.1` — Packet: define ResearchDiagnosticsPacket v1 types and redaction helpers.
2. `uiai-engine-3ds.2` — Packet metadata: add Focusa-ready metadata to first UIAI response surfaces.
   - `uiai-engine-3ds.2.1` — Metadata: add focusa object to search responses.
   - `uiai-engine-3ds.2.2` — Metadata: add focusa object to browser read responses.
   - `uiai-engine-3ds.2.3` — Metadata: align browser diagnostics focusa object with diagnostics intake.
   - `uiai-engine-3ds.2.4` — Metadata: add focusa object to structured UIAI errors.
   - `uiai-engine-3ds.2.5` — Test: first-surface focusa metadata contract.
3. `uiai-engine-3ds.3` — Pi: compose ResearchDiagnosticsPacket from existing UIAI tools.
   - `uiai-engine-3ds.3.1` — Pi packet builder: types, budget enforcement, and capture normalization.
   - `uiai-engine-3ds.3.2` — Pi packet builder: generate Focusa args_preview and next_tools.
   - `uiai-engine-3ds.3.3` — Pi packet renderer: compact line plus expandable JSON/ref.
   - `uiai-engine-3ds.3.4` — Pi packet output: RPC/JSON/headless parity.
   - `uiai-engine-3ds.3.5` — Pi packet state: prevent parallel durable memory.
4. `uiai-engine-3ds.4` — Smoke: prove Iteration 1 packet workflow safely.
   - `uiai-engine-3ds.4.1` — Smoke script: harmless search/open/read/diagnostics packet.
   - `uiai-engine-3ds.4.2` — Smoke evidence: capture proof artifact and Focusa args preview.
   - `uiai-engine-3ds.4.3` — Docs: update hand-in-glove spec with smoke outcome.

### 9.2 Guided workflow and parity path

5. `uiai-engine-3ds.5` — Pi: add /uiai research diagnose proof guided workflows.
   - `uiai-engine-3ds.5.1` — Pi workflow: /uiai research.
   - `uiai-engine-3ds.5.2` — Pi workflow: /uiai diagnose.
   - `uiai-engine-3ds.5.3` — Pi workflow: /uiai proof.
   - `uiai-engine-3ds.5.4` — Test: Pi workflow registration and /uiai on/off regression.
6. `uiai-engine-3ds.6` — Graph/MCP/CLI: expose packet workflow metadata and hints.
   - `uiai-engine-3ds.6.1` — Graph: add focusa_research_packet workflow metadata.
   - `uiai-engine-3ds.6.2` — MCP/CLI parity: document or expose packet workflow.

### 9.3 Follow-on safety, observability, and drift path

7. `uiai-engine-3ds.7` — Focusa: evaluate packet intake friction after Iteration 1.
   - `uiai-engine-3ds.7.1` — Focusa intake: define friction metrics and decision gate.
   - `uiai-engine-3ds.7.2` — Focusa intake: evaluate post-smoke packet captures.
8. `uiai-engine-3ds.8` — Observe: browser/search/cache pressure summary for Focusa/Pi.
   - `uiai-engine-3ds.8.1` — Metrics: add packet/search/browser/cache pressure summary.
   - `uiai-engine-3ds.8.2` — Focusa/Pi: surface UIAI pressure in tool doctor/project guidance.
9. `uiai-engine-3ds.9` — CI/docs: prevent packet surface drift.
   - `uiai-engine-3ds.9.1` — CI: add packet metadata surface drift check.
   - `uiai-engine-3ds.9.2` — Docs: update README/Session API/tool graph references after packet lands.

Dependency policy: schema/redaction blocks first-surface metadata and Pi packet builder; first-surface contract tests block smoke; smoke evidence blocks guided workflows, graph metadata, Focusa intake evaluation, observability, and docs/CI drift updates.

## 10. Decision rule

Prefer the smallest iteration that reduces agent friction while preserving the Focusa authority split:

- If a feature only improves display, implement in Pi.
- If it creates durable browser/search/diagnostics evidence, implement in UIAI.
- If it changes memory, Workpoint, trajectory, prediction, or lessons, implement in Focusa.
- If it crosses boundaries, expose stable handles and ready-to-call arguments, not raw blobs or hidden writes.
- If a gap lacks owner, schema, proof, or safety boundary, keep it in audit/spec mode instead of building.
