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

# #312: capture routes must document their request contracts. /api/screenshot
# rejects unknown fields with reason_code unknown_field (#311), so the published
# spec must make a correct call constructible instead of forcing clients to
# guess field names that the engine would silently or loudly ignore.
capture_request_bodies = {
    "/api/screenshot": {
        "required": True,
        "content": {"application/json": {
            "schema": {
                "type": "object",
                "additionalProperties": False,
                "required": ["url"],
                "properties": {
                    "url": {"type": "string", "description": "Absolute http(s) URL to capture; required"},
                    "width": {"type": "integer", "minimum": 1, "description": "Viewport width; omitted uses the engine default viewport"},
                    "height": {"type": "integer", "minimum": 1, "description": "Viewport height; omitted uses the engine default viewport"},
                    "fullPage": {"type": "boolean", "description": "Capture the full scrollable page instead of the viewport"},
                    "format": {"type": "string", "enum": ["png", "jpg", "jpeg", "webp"]},
                    "quality": {"type": "integer", "minimum": 1, "maximum": 100},
                    "waitFor": {"type": "string", "description": "CSS selector to wait for before capture"},
                    "delay": {"type": "integer", "minimum": 0, "description": "Extra settle delay in milliseconds"},
                    "cookies": {"type": "string", "description": "Cookie header string: name=value; name2=value2"},
                    "timeout": {"type": "integer", "minimum": 1, "description": "Overall capture timeout in seconds (default 30)"},
                    "nocache": {"type": "boolean", "description": "Skip the capture cache and force a fresh page load"},
                    "inline": {"type": "boolean", "description": "Deprecated posture hint; raw output stays withheld by mandatory EPWA delivery"},
                    "focusa_scope": {"type": "object", "description": "Focusa evidence scope binding (project/workstream/workset/callgraph/workpoint/work item/continuity refs)"},
                    "evidence_scope": {"type": "object", "description": "Direct evidence scope binding; equivalent to focusa_scope for EPWA publication gating"},
                },
            },
            "example": {
                "url": "https://example.com/",
                "width": 390,
                "height": 844,
                "format": "png",
                "fullPage": False,
                "focusa_scope": {
                    "project_ref": "project:example",
                    "workpoint_ref": "workpoint:example",
                    "work_item_ref": "work-item:example",
                    "continuity_id": "example-continuity",
                },
            },
        }},
    },
    "/api/screenshot/compare": {
        "required": True,
        "content": {"application/json": {
            "schema": {
                "type": "object",
                "additionalProperties": False,
                "properties": {
                    "url1": {"type": "string", "description": "First page URL (or supply base64_1)"},
                    "url2": {"type": "string", "description": "Second page URL (or supply base64_2)"},
                    "base64_1": {"type": "string", "description": "First image as base64; ignored when url1 is present"},
                    "base64_2": {"type": "string", "description": "Second image as base64; ignored when url2 is present"},
                    "width": {"type": "integer", "minimum": 1, "description": "Capture viewport width (default 1280)"},
                    "height": {"type": "integer", "minimum": 1, "description": "Capture viewport height (default 800)"},
                    "model": {"type": "string", "description": "Comparison model; default from engine AI config"},
                    "provider": {"type": "string", "description": "Comparison provider; default from engine AI config"},
                },
            },
            "example": {"url1": "https://example.com/", "url2": "https://example.org/", "width": 1280, "height": 800},
        }},
    },
    "/api/session/{sessionID}/screenshot": {
        "content": {"application/json": {
            "schema": {
                "type": "object",
                "additionalProperties": False,
                "properties": {
                    "format": {"type": "string", "enum": ["png", "jpg", "jpeg", "webp"]},
                    "quality": {"type": "integer", "minimum": 1, "maximum": 100},
                    "fullPage": {"type": "boolean", "description": "Capture the full scrollable page instead of the viewport"},
                    "output": {"type": "string", "description": "Delivery output mode for the captured visual"},
                    "focusa_scope": {"type": "object", "description": "Focusa evidence scope binding applied to this capture"},
                },
            },
        }},
    },
}
for path, request_body in capture_request_bodies.items():
    openapi["paths"][path]["post"]["requestBody"] = request_body

capture_path_parameters = {
    "/api/session/{sessionID}/screenshot": [{"name": "sessionID", "in": "path", "required": True, "schema": {"type": "string"}}],
    "/api/share/{id}/screenshot": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}, "description": "Existing share id; the share URL is re-captured at 1280x800 jpeg with no request body"}],
}
for path, parameters in capture_path_parameters.items():
    openapi["paths"][path]["post"]["parameters"] = parameters

# Browser reading is GET-canonical; retain POST for existing tool transports.
read_operation = dict(openapi["paths"]["/api/session/{sessionID}/read"]["post"])
read_operation["summary"] = "Read bounded page text with mandatory EPWA delivery (canonical GET)"
read_operation["parameters"] = [
    {"name": "sessionID", "in": "path", "required": True, "schema": {"type": "string"}},
    {"name": "selector", "in": "query", "schema": {"type": "string"}},
    {"name": "max_chars", "in": "query", "schema": {"type": "integer", "maximum": 30000, "default": 8000}},
    {"name": "include_links", "in": "query", "schema": {"type": "boolean"}},
    {"name": "include_images", "in": "query", "schema": {"type": "boolean"}},
    {"name": "format", "in": "query", "schema": {"type": "string", "enum": ["text", "markdown"]}},
    {"name": "mode", "in": "query", "schema": {"type": "string", "enum": ["main_content", "full"]}},
]
openapi["paths"]["/api/session/{sessionID}/read"]["get"] = read_operation
openapi["paths"]["/api/session/{sessionID}/read"]["post"]["description"] = "Supported compatibility transport for Pi, MCP and CLI; JSON ReadOptions use the same bounded, read-only implementation and delivery policy as GET."

# A committed artifact may fail before a real package binding exists. Preserve
# commit/recovery truth in a documented error variant rather than inventing hashes.
commit_operation = openapi["paths"]["/api/evidence/artifacts/commit"]["post"]
commit_operation["responses"] = dict(artifact_responses)
commit_operation["responses"]["202"] = {
    "description": "Artifact committed; delivery withheld or package publication failed",
    "content": {"application/json": {"schema": {"oneOf": [
        {"$ref": "#/components/schemas/uiai_artifact_delivery_envelope_v2"},
        {"$ref": "#/components/schemas/uiai_evidence_artifact_commit_delivery_error_v1"},
    ]}}},
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
