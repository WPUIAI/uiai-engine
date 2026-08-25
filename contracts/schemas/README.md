# Contracts Schemas — Single Source of Truth (MR-P0-01)

Machine-readable JSON Schema 2020-12 for every Cockpit/Engine wire payload.
TS types in `apps/cockpit/src/lib/contracts/*.ts` and Go `map[string]any` handlers must validate against these schemas.

- `$id` = `schema:` value emitted on wire (e.g. `uiai.desktop_presentation_request.v1`)
- Generated from TS `interface` + Go structs; `ajv` validates fixtures + live payloads.
- Distribution: `distribution-manifest.json` `schemas:[{id,sha256,src}]`, served at `GET /api/schema/:id` (MR-P0-04).
