# ResearchDiagnosticsPacket Schema Decision Matrix

**Status:** planning-only decision artifact; no implementation authorized yet.  
**Date:** 2026-06-04  
**Parent spec:** [`UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`](UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md)  
**HLT alignment:** make UIAI Engine the agent browser/research/diagnostics/proof engine for Pi, with Focusa as cognitive authority.

## 1. Decision goal

Decide the smallest safe packet schema and first response surfaces before building UIAI, Pi, or Focusa changes.

The packet must answer:

```text
What did UIAI inspect, what evidence handles prove it, what should Focusa capture, and what should Pi do next?
```

## 2. Non-build constraints

- No new paid or mutating actions.
- No automatic form submission/click automation inside packet creation.
- No raw SERP blobs, screenshots, HARs, cookies, auth headers, request bodies, or unbounded page text.
- No Pi-side durable memory or parallel cognitive authority.
- No Focusa writes unless the operator/tool flow explicitly calls Focusa evidence/prediction/metacog tools.
- UIAI treats `focusa_scope` as metadata only; Focusa authority remains `project_root + continuity_id`.

## 3. Candidate packet schema

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
    "evidence_ref": "optional seed"
  },
  "target_refs": [],
  "evidence_refs": [],
  "captures": [],
  "diagnostics_summary": {},
  "active_object_hints": [],
  "recommended_focusa": {
    "preferred_tool": "focusa_evidence_capture",
    "fallback_tool": "focusa_browser_diagnostics_intake",
    "args_preview": {}
  },
  "recommended_next_action": "exact next step",
  "cleanup": {
    "session_id": "optional",
    "recommended_action": "close_when_done|keep_for_followup|none",
    "tool": "uiai_browser_close"
  }
}
```

## 4. Field decision matrix

| Field | Source | Owner | Max size | Redaction rule | Focusa target | Build now? | Decision |
|---|---|---|---:|---|---|---|---|
| `schema` | static | UIAI | 64 chars | none | all | Yes | Required. |
| `mode` | caller/workflow | Pi or UIAI | enum | none | routing context | Yes | Required: `research`, `diagnose`, `proof`. |
| `goal` | operator query/url/task | Pi | 240 chars | strip secrets/query tokens | evidence result | Yes | Required for packet; absent becomes generated URL/query summary. |
| `scope_status` | focusa_scope presence/shape | UIAI | enum | no path rewriting | Focusa safety | Yes | Required to avoid silent unscoped capture. |
| `focusa_scope` | request/session | UIAI echo | bounded object | no secrets; project_root/continuity/workpoint only | Workpoint/evidence | Yes | Optional but echoed when present. |
| `target_refs` | URL/search/session/endpoint | UIAI | 16 refs | strip fragments; redact secret query keys | `target_ref` | Yes | Required; primary target first. |
| `evidence_refs` | UIAI handles | UIAI | 32 refs | handles only | evidence refs | Yes | Required when evidence exists. |
| `captures` | selected search/read/snapshot/diagnostics summaries | UIAI/Pi | 8 items, 500 chars each | bounded text only | evidence result | Yes, minimal | Include type, handle, title/url/path, summary. |
| `diagnostics_summary` | browser diagnostics/errors | UIAI | 2KB | no bodies/headers/cookies | diagnostics intake | Yes for diagnose/proof | Include counts + top bounded findings. |
| `active_object_hints` | URL/path/selector/source/endpoint | UIAI | 16 hints | path-only for failed URLs when possible | active object resolve | Yes | Normalize hints but mark candidate. |
| `recommended_focusa.preferred_tool` | mode + evidence type | UIAI/Pi | enum/tool name | none | Focusa tool routing | Yes | Usually `focusa_evidence_capture`; diagnostics can prefer intake. |
| `recommended_focusa.args_preview` | packet summary | UIAI/Pi | 2KB | no raw blobs | Focusa tool args | Yes, preview only | Preview, not automatic write. |
| `recommended_next_action` | workflow route | UIAI/Pi | 240 chars | none | Pi UX | Yes | Required. |
| `cleanup` | browser session lifecycle | UIAI/Pi | small object | session id only | Pi cleanup | Yes | Required if session involved. |

## 5. First response surfaces

Recommended first surfaces, in order:

| Surface | Why first | Packet contribution | Risk | Decision |
|---|---|---|---|---|
| `/api/search` | already has evidence refs/cache/redaction | search capture, selected result handles | low | First. |
| `/api/session/{id}/read` | core research output; no screenshot blob | read capture + target ref | medium due page text size | First with strict max summary. |
| `/api/session/{id}/diagnostics` | existing Focusa intake path | diagnostics summary + hints | low/medium | First. |
| `/api/errors` | engine/browser failures are actionable | error capture + diagnostics route | low | First or second. |
| `/api/session/{id}/snapshot` | action refs useful for next step | candidate selector hints | medium due tree size | Second. |
| `/api/screenshot` and `/api/share/*` | already have focusa_evidence | artifact capture | low | Second; existing evidence object may suffice. |

## 6. Packet composition location

| Option | Pros | Cons | Decision |
|---|---|---|---|
| Pi-composed from existing tools | no new endpoint; proves workflow fast | harder to share with MCP/CLI; Pi can grow workflow logic | Best first proof. |
| UIAI HTTP endpoint `/api/agent/research-packet` | reusable by Pi/MCP/CLI; canonical packet shape | endpoint can over-automate if it gathers data itself | Implemented as a composer only: callers pass existing UIAI responses; endpoint normalizes/redacts/bounds the packet. |
| Focusa intake composes packet | closest to Workpoint/evidence | violates UIAI execution ownership; can blur authority | No for execution; Focusa only ingests/captures. |

## 7. Focusa intake decision

Initial decision: **no new Focusa tool yet**.

Use existing tools:

- `focusa_evidence_capture` for research/read/search packet summaries.
- `focusa_browser_diagnostics_intake` for diagnostics/failure envelopes.
- `focusa_active_object_resolve` for packet hints.
- `focusa_predict_record` before patches or risky next actions.

New Focusa tool only if repeated packets need one-call evidence + hints + prediction behavior beyond diagnostics.

## 8. Proof path before implementation

Use a harmless local/public page and no paid/mutating calls:

```text
uiai_search query="UIAI Engine browser agents" limit=1
uiai_browser_open selected URL with optional focusa_scope
uiai_browser_read max_chars<=2000
uiai_browser_diagnostics failed_only=false
compose packet JSON locally
focusa_evidence_capture attach_to_workpoint=false or scoped Workpoint when canonical
```

Acceptance:

- Packet under 8KB.
- No raw screenshot/base64.
- No secrets or auth headers.
- Evidence handles present.
- `args_preview` can be copied directly into Focusa tool call.
- Cleanup recommendation explicit.

## 9. Build/no-build decision checklist

Build may start only when all are true:

- [ ] Packet schema accepted.
- [ ] First response surfaces accepted.
- [ ] Max sizes accepted.
- [ ] Redaction rules accepted.
- [ ] Focusa intake path accepted.
- [ ] Proof path accepted.
- [ ] Project-card crosswire handling defined or excluded from MVP.

Until then: planning/spec only.

## 10. Recommended next planning question

Should Iteration 1 add metadata to **all first surfaces** (`search`, `read`, `diagnostics`, `errors`) or start with only `search + read + diagnostics`?
