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
    if sid == "uiai.artifact_delivery_envelope.v2":
        sch["properties"]["epwa_delivery"]["$ref"] = "#/components/schemas/uiai_epwa_delivery_v1"
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
        "/api/errors": {
            "get": {"summary": "Bounded redacted error history", "responses": {"200": {"description": "ok"}}}
        },
    },
    "components": {"schemas": components_schemas}
}

artifact_response = {
    "description": "EPWA delivery envelope; HTTP 202 means raw output is withheld pending reconciliation",
    "content": {"application/json": {"schema": {"$ref": "#/components/schemas/uiai_artifact_delivery_envelope_v2"}}},
}
artifact_responses = {"200": artifact_response, "201": artifact_response, "202": artifact_response}
artifact_routes = {
    "/api/screenshot": ("post", "Capture screenshot with mandatory EPWA delivery"),
    "/api/screenshot/compare": ("post", "Compare screenshots with mandatory EPWA report delivery"),
    "/api/session": ("post", "Create browser session with mandatory initial visual EPWA delivery"),
    "/api/session/{sessionID}/screenshot": ("post", "Capture session visual with mandatory EPWA delivery"),
    "/api/session/{sessionID}/snapshot": ("get", "Capture accessibility snapshot with mandatory EPWA delivery"),
    "/api/session/{sessionID}/dom": ("get", "Capture DOM snapshot with mandatory EPWA delivery"),
    "/api/session/{sessionID}/read": ("post", "Capture source snapshot with mandatory EPWA delivery"),
    "/api/session/{sessionID}/diagnostics": ("get", "Capture diagnostics bundle with mandatory EPWA delivery"),
    "/api/search": ("post", "Produce search report with mandatory EPWA delivery"),
    "/api/markdown": ("post", "Produce source-to-Markdown artifact with mandatory EPWA delivery"),
    "/api/agent/research-packet": ("post", "Produce bounded research packet with mandatory EPWA delivery"),
    "/api/critique": ("post", "Produce critique report with mandatory EPWA delivery"),
    "/api/media/frame/render": ("post", "Render framed media with mandatory EPWA delivery"),
    "/api/evidence/artifacts/commit": ("post", "Commit immutable artifact with mandatory EPWA delivery"),
    "/api/share/{id}/screenshot": ("post", "Capture share screenshot with mandatory EPWA delivery"),
    "/api/fpv/share": ("post", "Create operational FPV mirror with mandatory EPWA evidence snapshot"),
    "/api/intelligence/index/upload": ("post", "Upload index artifacts with mandatory EPWA delivery for every artifact"),
    "/api/intelligence/wasm/{runId}": ("post", "Upload WASM with mandatory EPWA delivery"),
    "/api/intelligence/js/{runId}": ("post", "Upload JavaScript with mandatory EPWA delivery"),
}
for path, (method, summary) in artifact_routes.items():
    openapi["paths"].setdefault(path, {})[method] = {"summary": summary, "responses": artifact_responses}

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
