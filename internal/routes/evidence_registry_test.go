package routes

import (
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

	router := chi.NewRouter()
	router.Route("/api/evidence/registry", func(r chi.Router) { MountEvidenceRegistry(r, manager) })
	project := url.QueryEscape("project:uiai-engine")
	cases := []struct {
		path string
		want string
	}{
		{"/api/evidence/registry/?project_ref=" + project + "&q=golden%20tests", `"artifact_ref":"artifact:epwa-001"`},
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
