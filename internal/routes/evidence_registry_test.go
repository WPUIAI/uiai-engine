package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

func TestEvidenceRegistryReadRoutes(t *testing.T) {
	ctx := context.Background()
	manager, err := evidenceregistry.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store, err := manager.EnsureProject(ctx, "project:uiai-engine")
	if err != nil {
		t.Fatal(err)
	}
	manifest := routeGoldenManifest(t)
	_, err = store.Index(ctx, evidenceregistry.IndexInput{
		Manifest:              manifest,
		ManifestSHA256:        strings.Repeat("d", 64),
		CompletionCaseRef:     "completion-case:t01",
		CompletionContractRef: "completion-contract:t01",
		SettlementPosture:     "open",
		ObservedAt:            time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
		Acceptances:           []evidenceregistry.AcceptanceBinding{{AcceptanceAtomRef: "atom:types", Revision: "1", State: evidenceregistry.AcceptanceAccepted, VerifierClass: "independent", Fresh: true, ScopeMatched: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bound := manifest.Scope.WorkItems[0]
	_, err = store.ReplaceProviderGraph(ctx, evidenceregistry.ProviderGraphInput{
		Project: evidenceregistry.ProjectProjection{ProjectRef: "project:uiai-engine", ProjectID: "uiai-engine", DisplayName: "UIAI Engine", Fingerprint: "project-fingerprint:test", ScopeSafety: "safe", SourceSchema: "focusa.project_dashboard.v1", ObservedAt: time.Date(2026, 9, 2, 12, 1, 0, 0, time.UTC)},
		Items:   []evidenceregistry.ProviderWorkItem{{ProjectRef: "project:uiai-engine", WorkItemRef: bound.WorkItemRef, ProviderSurface: bound.ProviderSurface, ItemID: bound.ItemID, ItemType: bound.ItemType, Title: bound.Title, Description: bound.Description, Status: bound.StatusAtCapture, Revision: bound.Revision, Digest: bound.Digest, DependencyRefs: []string{"work-item:focusa-prereq"}, SourceAuthority: "provider:br", BindingState: "focusa_binding_pending"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	router.Route("/api/evidence/registry", func(r chi.Router) { MountEvidenceRegistry(r, manager) })
	project := url.QueryEscape("project:uiai-engine")
	cases := []struct {
		path string
		want string
	}{
		{"/api/evidence/registry/sync-status", `"freshness":"unavailable"`},
		{"/api/evidence/registry/projects", `"display_name":"UIAI Engine"`},
		{"/api/evidence/registry/?project_ref=" + project + "&q=golden%20tests", `"artifact_ref":"artifact:epwa-001"`},
		{"/api/evidence/registry/provider-work-items?project_ref=" + project, `"binding_state":"artifact_bound"`},
		{"/api/evidence/registry/provider-edges?project_ref=" + project + "&object_ref=" + url.QueryEscape(bound.WorkItemRef) + "&direction=forward", `"relation":"depends_on"`},
		{"/api/evidence/registry/status?project_ref=" + project, `"artifact_count":1`},
		{"/api/evidence/registry/resolve?project_ref=" + project + "&artifact_ref=artifact:epwa-001&revision=1", `"first_work_item_title":"Implement artifact contract"`},
		{"/api/evidence/registry/work-items?project_ref=" + project + "&artifact_ref=artifact:epwa-001&revision=1", `"description":"Run contract and golden tests."`},
		{"/api/evidence/registry/edges?project_ref=" + project + "&object_ref=artifact:epwa-001&direction=forward&relation=artifact_work_item", `"target_ref":"work-item:focusa-a1"`},
		{"/api/evidence/registry/closure?project_ref=" + project + "&work_item_ref=work-item:focusa-a1&completion_case_ref=completion-case:t01", `"posture":"eligible_for_closure"`},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), tc.want) {
			t.Fatalf("GET %s status=%d body=%s want=%s", tc.path, recorder.Code, recorder.Body.String(), tc.want)
		}
	}

	for _, path := range []string{
		"/api/evidence/registry/",
		"/api/evidence/registry/?project_ref=project:unknown",
		"/api/evidence/registry/resolve?project_ref=" + project + "&artifact_ref=artifact:epwa-001&revision=bad",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusBadRequest && recorder.Code != http.StatusNotFound {
			t.Fatalf("GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), registryErrorSchema) {
			t.Fatalf("GET %s missing typed error: %s", path, recorder.Body.String())
		}
	}
}

func TestEvidenceRegistryEventStreamConsumer(t *testing.T) {
	manager, err := evidenceregistry.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	router := chi.NewRouter()
	router.Route("/api/evidence/registry", func(r chi.Router) { MountEvidenceRegistry(r, manager) })
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	responseCh := make(chan *http.Response, 1)
	errorCh := make(chan error, 1)
	go func() {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/evidence/registry/events", nil)
		response, requestErr := http.DefaultClient.Do(req)
		if requestErr != nil {
			errorCh <- requestErr
			return
		}
		responseCh <- response
	}()
	trigger := time.NewTicker(10 * time.Millisecond)
	defer trigger.Stop()
	var response *http.Response
	for response == nil {
		select {
		case response = <-responseCh:
		case requestErr := <-errorCh:
			t.Fatal(requestErr)
		case <-trigger.C:
			_, _ = manager.SyncAndPublish(ctx, evidenceregistry.FocusaSyncConfig{}, "consumer_test")
		case <-ctx.Done():
			t.Fatal("event stream did not open")
		}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("status=%d content-type=%q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	scanner := bufio.NewScanner(response.Body)
	found := false
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data:") {
			found = strings.Contains(scanner.Text(), `"status":"degraded"`) && strings.Contains(scanner.Text(), `"freshness":"stale"`)
			break
		}
	}
	if !found {
		t.Fatal("typed degraded registry event was not streamed")
	}
}

func routeGoldenManifest(t *testing.T) evidenceartifact.Manifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "evidenceartifact", "testdata", "manifest.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest evidenceartifact.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}
