package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

const registryErrorSchema = "uiai.evidence_registry_error.v1"

type registryProvider interface {
	Project(req *http.Request, projectRef string) (*evidenceregistry.Store, error)
}

type evidenceRegistryManager struct{ manager *evidenceregistry.Manager }

func (p evidenceRegistryManager) Project(req *http.Request, projectRef string) (*evidenceregistry.Store, error) {
	return p.manager.Project(req.Context(), projectRef)
}

func MountEvidenceRegistry(r chi.Router, manager *evidenceregistry.Manager) {
	mountEvidenceRegistry(r, evidenceRegistryManager{manager: manager})
}

func mountEvidenceRegistry(r chi.Router, provider registryProvider) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		pageSize, ok := registryUint(w, req, "page_size", 0)
		if !ok {
			return
		}
		query := evidenceregistry.Query{
			ProjectRef:        req.URL.Query().Get("project_ref"),
			Text:              req.URL.Query().Get("q"),
			WorkstreamRef:     req.URL.Query().Get("workstream_ref"),
			WorksetRef:        req.URL.Query().Get("workset_ref"),
			WorkpointRef:      req.URL.Query().Get("workpoint_ref"),
			WorkItemRef:       req.URL.Query().Get("work_item_ref"),
			WorkItemType:      req.URL.Query().Get("work_item_type"),
			AcceptanceAtomRef: req.URL.Query().Get("acceptance_atom_ref"),
			Verification:      req.URL.Query().Get("verification"),
			Access:            req.URL.Query().Get("access"),
			Closure:           req.URL.Query().Get("closure"),
			Cursor:            req.URL.Query().Get("cursor"),
			PageSize:          uint32(pageSize),
		}
		page, err := store.List(req.Context(), query)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, page)
	})

	r.Get("/status", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		status, err := store.Status(req.Context())
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	})

	r.Get("/resolve", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		revision, ok := registryUint(w, req, "revision", 0)
		if !ok || revision == 0 {
			if ok {
				writeRegistryError(w, evidenceregistry.ErrInputInvalid)
			}
			return
		}
		row, err := store.Resolve(req.Context(), req.URL.Query().Get("artifact_ref"), revision)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	r.Get("/work-items", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		revision, ok := registryUint(w, req, "revision", 0)
		if !ok || revision == 0 {
			if ok {
				writeRegistryError(w, evidenceregistry.ErrInputInvalid)
			}
			return
		}
		items, err := store.WorkItems(req.Context(), req.URL.Query().Get("artifact_ref"), revision)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": evidenceregistry.RegistrySchemaV1, "project_ref": req.URL.Query().Get("project_ref"), "work_items": items})
	})

	r.Get("/edges", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		limit, ok := registryUint(w, req, "limit", 50)
		if !ok {
			return
		}
		edges, err := store.Edges(req.Context(), req.URL.Query().Get("object_ref"), evidenceregistry.EdgeDirection(req.URL.Query().Get("direction")), evidenceregistry.RelationType(req.URL.Query().Get("relation")), uint32(limit))
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": evidenceregistry.EdgeSchemaV1, "project_ref": req.URL.Query().Get("project_ref"), "edges": edges})
	})

	r.Get("/closure", func(w http.ResponseWriter, req *http.Request) {
		store, ok := registryStore(w, req, provider)
		if !ok {
			return
		}
		projection, err := store.Closure(req.Context(), req.URL.Query().Get("work_item_ref"), req.URL.Query().Get("completion_case_ref"))
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, projection)
	})
}

func registryStore(w http.ResponseWriter, req *http.Request, provider registryProvider) (*evidenceregistry.Store, bool) {
	projectRef := strings.TrimSpace(req.URL.Query().Get("project_ref"))
	if projectRef == "" {
		writeRegistryError(w, evidenceregistry.ErrInputInvalid)
		return nil, false
	}
	store, err := provider.Project(req, projectRef)
	if err != nil {
		writeRegistryError(w, err)
		return nil, false
	}
	return store, true
}

func registryUint(w http.ResponseWriter, req *http.Request, key string, fallback uint64) (uint64, bool) {
	value := strings.TrimSpace(req.URL.Query().Get(key))
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		writeRegistryError(w, evidenceregistry.ErrInputInvalid)
		return 0, false
	}
	return parsed, true
}

func writeRegistryError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "index_unavailable"
	switch {
	case errors.Is(err, evidenceregistry.ErrInputInvalid), errors.Is(err, evidenceregistry.ErrCursorInvalid), errors.Is(err, evidenceregistry.ErrConfig):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, evidenceregistry.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, evidenceregistry.ErrIndexCorrupt):
		status, code = http.StatusServiceUnavailable, "corrupt_index"
	case errors.Is(err, evidenceregistry.ErrIndexUnavailable):
		status, code = http.StatusServiceUnavailable, "index_unavailable"
	}
	writeJSON(w, status, map[string]string{"schema": registryErrorSchema, "error": code})
}
