package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

const publicRegistrySchemaV1 = "uiai.public_evidence_registry.v1"

var publicRegistrySSESlots = make(chan struct{}, 64)

type publicProjectProjection struct {
	ProjectRef    string    `json:"project_ref"`
	ProjectID     string    `json:"project_id"`
	DisplayName   string    `json:"display_name"`
	WorkspaceKind string    `json:"workspace_kind,omitempty"`
	ScopeSafety   string    `json:"scope_safety"`
	ObservedAt    time.Time `json:"observed_at"`
}

func MountPublicEvidenceRegistry(r chi.Router, manager *evidenceregistry.Manager, projectRefs []string) {
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, req)
		})
	})
	allowed := make(map[string]struct{}, len(projectRefs))
	allowedRefs := make([]string, 0, len(projectRefs))
	for _, ref := range projectRefs {
		if ref = strings.TrimSpace(ref); ref != "" {
			if _, exists := allowed[ref]; !exists {
				allowedRefs = append(allowedRefs, ref)
			}
			allowed[ref] = struct{}{}
		}
	}
	isAllowed := func(ref string) bool {
		_, ok := allowed[strings.TrimSpace(ref)]
		return ok
	}

	r.Get("/projects", func(w http.ResponseWriter, req *http.Request) {
		projects, err := manager.ProjectProjections(req.Context())
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		projectByRef := make(map[string]evidenceregistry.ProjectProjection, len(projects))
		for _, project := range projects {
			projectByRef[project.ProjectRef] = project
		}
		public := make([]publicProjectProjection, 0, len(allowedRefs))
		for _, projectRef := range allowedRefs {
			if project, ok := projectByRef[projectRef]; ok {
				public = append(public, publicProjectProjection{ProjectRef: project.ProjectRef, ProjectID: project.ProjectID, DisplayName: project.DisplayName, WorkspaceKind: project.WorkspaceKind, ScopeSafety: project.ScopeSafety, ObservedAt: project.ObservedAt})
				continue
			}
			store, err := manager.Project(req.Context(), projectRef)
			if err != nil {
				continue
			}
			page, err := store.List(req.Context(), evidenceregistry.Query{ProjectRef: projectRef, Access: "public_safe", PageSize: 1})
			if err != nil || len(page.Rows) == 0 {
				continue
			}
			public = append(public, publicProjectProjection{ProjectRef: projectRef, ProjectID: projectRef, DisplayName: projectRef, ScopeSafety: "artifact_public_safe", ObservedAt: page.ObservedAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": publicRegistrySchemaV1, "interaction": "read_only", "projects": public})
	})

	r.Get("/artifacts", func(w http.ResponseWriter, req *http.Request) {
		projectRef := strings.TrimSpace(req.URL.Query().Get("project_ref"))
		if !isAllowed(projectRef) {
			writeRegistryError(w, evidenceregistry.ErrNotFound)
			return
		}
		pageSize, ok := registryUint(w, req, "page_size", 50)
		if !ok {
			return
		}
		store, err := manager.Project(req.Context(), projectRef)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		page, err := store.List(req.Context(), evidenceregistry.Query{ProjectRef: projectRef, Text: req.URL.Query().Get("q"), WorkItemRef: req.URL.Query().Get("work_item_ref"), WorkItemType: req.URL.Query().Get("work_item_type"), Verification: req.URL.Query().Get("verification"), Access: "public_safe", Closure: req.URL.Query().Get("closure"), Cursor: req.URL.Query().Get("cursor"), PageSize: uint32(pageSize)})
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		for i := range page.Rows {
			if !strings.HasPrefix(page.Rows[i].PWAPath, "/") || strings.Contains(page.Rows[i].PWAPath, "..") || strings.Contains(page.Rows[i].PWAPath, "://") {
				page.Rows[i].PWAPath = ""
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": "uiai.public_evidence_artifacts.v1", "project_ref": projectRef, "interaction": "read_only", "rows": page.Rows, "next_cursor": page.NextCursor, "page_size": page.PageSize, "index_revision": page.IndexRevision, "index_state": page.IndexState, "observed_at": page.ObservedAt})
	})

	r.Get("/work-items", func(w http.ResponseWriter, req *http.Request) {
		projectRef := strings.TrimSpace(req.URL.Query().Get("project_ref"))
		if !isAllowed(projectRef) {
			writeRegistryError(w, evidenceregistry.ErrNotFound)
			return
		}
		limit, ok := registryUint(w, req, "limit", 100)
		if !ok {
			return
		}
		store, err := manager.Project(req.Context(), projectRef)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		page, err := store.ListProviderWorkItems(req.Context(), evidenceregistry.ProviderWorkItemQuery{ProjectRef: projectRef, Text: req.URL.Query().Get("q"), Status: req.URL.Query().Get("status"), ItemType: req.URL.Query().Get("item_type"), Limit: uint32(limit), Cursor: req.URL.Query().Get("cursor")})
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		for i := range page.WorkItems {
			page.WorkItems[i].ExternalRef = ""
		}
		writeJSON(w, http.StatusOK, page)
	})

	r.Get("/edges", func(w http.ResponseWriter, req *http.Request) {
		projectRef := strings.TrimSpace(req.URL.Query().Get("project_ref"))
		if !isAllowed(projectRef) {
			writeRegistryError(w, evidenceregistry.ErrNotFound)
			return
		}
		limit, ok := registryUint(w, req, "limit", 100)
		if !ok {
			return
		}
		store, err := manager.Project(req.Context(), projectRef)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		edges, err := store.ProviderWorkItemEdges(req.Context(), req.URL.Query().Get("object_ref"), evidenceregistry.EdgeDirection(req.URL.Query().Get("direction")), req.URL.Query().Get("relation"), uint32(limit))
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": "uiai.public_evidence_work_item_edges.v1", "project_ref": projectRef, "interaction": "read_only", "edges": edges})
	})

	r.Get("/sync-status", func(w http.ResponseWriter, _ *http.Request) {
		status := manager.RegistrySyncStatus()
		writeJSON(w, http.StatusOK, map[string]any{"schema": "uiai.public_evidence_sync_status.v1", "status": status.Status, "freshness": status.Freshness, "last_attempt_at": status.LastAttemptAt, "last_success_at": status.LastSuccessAt})
	})

	r.Get("/events", func(w http.ResponseWriter, req *http.Request) {
		select {
		case publicRegistrySSESlots <- struct{}{}:
			defer func() { <-publicRegistrySSESlots }()
		default:
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"schema": "uiai.public_evidence_registry_error.v1", "error": map[string]string{"code": "subscriber_limit", "message": "public registry event capacity reached"}})
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeRegistryError(w, evidenceregistry.ErrIndexUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		events := manager.RegistryEvents(req.Context())
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-req.Context().Done():
				return
			case event, open := <-events:
				if !open {
					return
				}
				event = filterPublicRegistryEvent(event, isAllowed)
				body, err := json.Marshal(event)
				if err != nil {
					return
				}
				_, _ = fmt.Fprintf(w, "id: %s\nevent: registry_revision\ndata: %s\n\n", event.EventID, body)
				flusher.Flush()
			case <-heartbeat.C:
				_, _ = fmt.Fprint(w, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	})
}

func filterPublicRegistryEvent(event evidenceregistry.RegistryEvent, isAllowed func(string) bool) evidenceregistry.RegistryEvent {
	filtered := make([]evidenceregistry.ProviderGraphResult, 0, len(event.Results))
	var items uint64
	for _, result := range event.Results {
		if isAllowed(result.ProjectRef) {
			filtered = append(filtered, result)
			items += result.Items
		}
	}
	event.Results, event.Items, event.Projects = filtered, items, uint64(len(filtered))
	event.ChangedProjects = 0
	for _, result := range filtered {
		if result.Changed {
			event.ChangedProjects++
		}
	}
	return event
}
