package routes

import (
	"embed"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi_embed.json
//go:embed schemas_embed/*.schema.json
var openapiEmbedFS embed.FS

// MountOpenAPI exposes GET /api/openapi.json and GET /api/schema/{id} — MR-P0-03/MR-P0-04.
func MountOpenAPI(r chi.Router) {
	r.Get("/openapi.json", HandleOpenAPI)
	r.Get("/schema/{id}", HandleSchema)
}

// HandleOpenAPI serves the embedded OpenAPI 3.1 document — GET /api/openapi.json
func HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := openapiEmbedFS.ReadFile("openapi_embed.json")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "openapi not embedded"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

// HandleSchema serves one JSON Schema by $id — GET /api/schema/{id} (e.g. uiai.status.v1)
func HandleSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing schema id"})
		return
	}
	safe := filepath.Base(id)
	if safe != id || strings.Contains(safe, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid schema id"})
		return
	}
	// Try embedded schemas first (deterministic, no filesystem dep)
	if data, err := openapiEmbedFS.ReadFile("schemas_embed/" + safe + ".schema.json"); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(data)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]any{
		"error":            "schema not found: " + safe + "; see /api/openapi.json components/schemas or contracts/schemas/manifest.json",
		"schema":           safe,
		"openapi":          "/api/openapi.json",
		"manifest":         "contracts/schemas/manifest.json",
		"suggested_action": "fetch /api/openapi.json and resolve #/components/schemas/" + strings.ReplaceAll(safe, ".", "_"),
	})
}
