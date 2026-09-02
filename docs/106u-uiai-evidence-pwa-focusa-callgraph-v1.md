# EPWA Focusa CallGraph v1 Runtime Projection

## Purpose

This artifact projects the canonical 32-node EPWA completion graph into the current `focusa.callgraph.v1` schema required by Spec 155 validation. It preserves the canonical node identities and all 55 dependency edges.

It does **not** authorize dispatch, independent verification, settlement, or closure.

## Sources

- Canonical graph: `docs/106q-uiai-evidence-pwa-completion-callgraph.json`
- Canonical graph SHA-256: `d4d0cf8c1d9ac3f08652c1de7d985199697b6fc5b7c60d39a7859e065e3957fb`
- Runtime projection: `docs/106u-uiai-evidence-pwa-focusa-callgraph-v1.json`
- Projection SHA-256: `8c908b58da1ca2a1a6de4a2a8e7f7abf31efa91a9eb63ec1303c39e1b3ab99c9`
- Validator source: Focusa `origin/main` commit `14eea0336`
- Exact validator module SHA-256: `13c35bf853cb3711c204505362b13f5a195b34b021d858c79d4ea534566fddac`

## Verification

1. `python3 scripts/audit-epwa-callgraph-v1.py`
   - PASS: 32 ordered unique frames.
   - PASS: 55 dependency edges exactly match the canonical graph.
   - PASS: `CG-01` is the sole entry frame.
   - PASS: CG-01 retains baseline-only acceptance; CG-02–CG-32 require independent verification.
   - PASS: every frame has explicit acceptance atoms.
2. Thirty repeated projection audits preserved the projection hash above.
3. The exact pure validator module from Focusa `crates/focusa-core/src/callgraph.rs` returned:

```json
{
  "valid": true,
  "issues": []
}
```

## Remaining canonical gate

The installed daemon remains `0.9.178-dev` and returns HTTP 404 for both `/v1/callgraphs/validate` and the durable silent-session approval preview route. Therefore:

- canonical daemon validation is not complete;
- durable verifier approval is not available;
- the graph has not been dispatched;
- CG-02 has not been independently verified or settled.

After canonical runtime delivery, submit this exact projection to the delivered validation route, create durable approval, and start verifier session `01a060a9-779f-7c22-accb-4f03f16c8b41`.
