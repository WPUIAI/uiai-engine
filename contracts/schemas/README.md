# Contracts Schemas — Single Source of Truth (MR-P0-01)

Machine-readable JSON Schema 2020-12 for every Cockpit/Engine wire payload.
TS types in `apps/cockpit/src/lib/contracts/*.ts` and Go `map[string]any` handlers must validate against these schemas.

- Manifest `id` is the short registry/component and `/api/schema/{id}` lookup key (e.g. `uiai.epwa_delivery.v1`).
- Optional manifest `schema_id` is the canonical JSON Schema `$id`; it defaults to `id` for legacy schemas. EPWA schemas use explicit HTTPS canonical IDs while retaining their short wire discriminators.
- The machine-readability gate validates both identities: schema `$id` must equal `schema_id` (or the legacy `id`), and an explicit distinct canonical ID requires the schema discriminator constants/enums to match manifest `wire_schema_ids` (default `[id]`). This supports envelope schemas whose accepted runtime result discriminators differ from their registry key. Neither URL identity nor runtime compatibility is weakened.
- OpenAPI components and embedded schema filenames retain the short lookup keys; relative `$ref` resolution is independent of those wire discriminators.
- Generated from TS `interface` + Go structs; `ajv` validates fixtures + live payloads.
- Distribution: `distribution-manifest.json` `schemas:[{id,sha256,src}]`, served at `GET /api/schema/:id` (MR-P0-04).
