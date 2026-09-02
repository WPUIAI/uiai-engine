#!/usr/bin/env python3
"""Generate docs/api/openapi.json from contracts/schemas/manifest.json + route inventory."""
import json, pathlib
ROOT = pathlib.Path(__file__).resolve().parent.parent
MANIFEST = ROOT / "contracts/schemas/manifest.json"
OUT = ROOT / "docs/api/openapi.json"
manifest = json.loads(MANIFEST.read_text())
schemas = {s["id"]: s for s in manifest["schemas"]}

def ref_schema(id):
    return {"$ref": f"../contracts/schemas/{id.replace('.', '_').replace('-', '_')}.schema.json"}  # placeholder, actual served via /api/schema/{id}

# Build components/schemas by inlining $id pointers
components_schemas = {}
for sid, entry in schemas.items():
    p = ROOT / entry["src"]
    sch = json.loads(p.read_text())
    # store under sanitized key for OpenAPI components
    key = sid.replace(".", "_").replace("-", "_")
    components_schemas[key] = sch

openapi = {
    "openapi": "3.1.0",
    "info": {"title": "UIAI Engine API", "version": "2.0.0-dev", "description": "Machine-readable Engine API — every JSON payload carries schema: uiai.*.v1"},
    "servers": [{"url": "http://localhost:3000", "description": "local engine"}],
    "paths": {
        "/api/status": {
            "get": {
                "summary": "Engine status (machine-readable)",
                "responses": {"200": {"description": "status", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/uiai_status_v1"}}}}}
            }
        },
        "/api/health/browser": {
            "get": {
                "summary": "Agent pressure telemetry (advisory, noncanonical)",
                "responses": {"200": {"description": "agent_pressure", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/uiai_agent_pressure_v1"}}}}}
            }
        },
        "/api/openapi.json": {
            "get": {
                "summary": "OpenAPI 3.1 document for this engine build",
                "responses": {"200": {"description": "openapi", "content": {"application/json": {"schema": {"type": "object"}}}}}
            }
        },
        "/api/schema/{id}": {
            "get": {
                "summary": "Serve one JSON Schema by $id (e.g. uiai.status.v1)",
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {"200": {"description": "schema", "content": {"application/json": {"schema": {"type": "object"}}}}, "404": {"description": "not found"}}
            }
        },
        "/api/session": {
            "post": {"summary": "Create browser session", "responses": {"200": {"description": "ok"}}},
            "get": {"summary": "List sessions", "responses": {"200": {"description": "ok"}}}
        },
        "/api/session/{id}/read": {
            "get": {
                "summary": "Read bounded page text or Markdown",
                "parameters": [
                    {"name": "id", "in": "path", "required": True, "schema": {"type": "string"}},
                    {"name": "selector", "in": "query", "schema": {"type": "string"}},
                    {"name": "max_chars", "in": "query", "schema": {"type": "integer", "minimum": 0}},
                    {"name": "include_links", "in": "query", "schema": {"type": "boolean"}},
                    {"name": "format", "in": "query", "schema": {"type": "string"}},
                    {"name": "mode", "in": "query", "schema": {"type": "string"}},
                    {"name": "include_images", "in": "query", "schema": {"type": "boolean"}},
                ],
                "responses": {"200": {"description": "bounded read result"}, "400": {"description": "invalid query"}, "404": {"description": "session not found"}},
            },
            "post": {
                "summary": "Read bounded page text or Markdown (legacy compatibility)",
                "deprecated": True,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {"200": {"description": "bounded read result"}, "404": {"description": "session not found"}},
            },
        },
        "/api/errors": {
            "get": {"summary": "Bounded redacted error history", "responses": {"200": {"description": "ok"}}}
        },
    },
    "components": {"schemas": components_schemas}
}
OUT.parent.mkdir(parents=True, exist_ok=True)
OUT.write_text(json.dumps(openapi, indent=2) + "\n")
# MR-P0-03/04: keep Go embed in sync — binary serves via //go:embed
embed_out = ROOT / "internal/routes/openapi_embed.json"
embed_out.write_text(json.dumps(openapi, indent=2) + "\n")
# also copy schemas into embed dir for GET /api/schema/{id}
import shutil
emb_schemas = ROOT / "internal/routes/schemas_embed"
emb_schemas.mkdir(parents=True, exist_ok=True)
for s in manifest["schemas"]:
    src = ROOT / s["src"]
    dst = emb_schemas / (s["id"] + ".schema.json")
    shutil.copy(src, dst)
print(f"wrote {OUT} with {len(components_schemas)} schemas -> synced embed {embed_out} + {emb_schemas}")
