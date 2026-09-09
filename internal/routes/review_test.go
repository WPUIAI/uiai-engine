package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceaction"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

type reviewFixture struct {
	router   chi.Router
	shareDir string
	recordID string
	identity *auth.Identity
}

func writeReviewManifest(t *testing.T, directory, recordID string) {
	t.Helper()
	manifest := map[string]any{
		"schema":            evidenceshare.Schema,
		"artifact_ref":      "artifact:" + recordID,
		"artifact_sha256":   "a1b2c3",
		"screenshot_ref":    "screenshot:" + recordID,
		"screenshot_sha256": "d4e5f6",
		"format":            "png",
		"mime":              "image/png",
		"bytes":             3,
		"width":             3,
		"height":            3,
		"source_url":        "https://engine.example/page",
		"availability":      "online",
		"access":            "public",
		"interaction":       "read_only",
		"truth_notice":      "bounded visual note",
		"scope": map[string]any{
			"project_ref":    "project:p",
			"workstream_ref": "workstream:w",
			"workset_ref":    "workset:s",
			"callgraph_ref":  "callgraph:g",
			"workpoint_ref":  "workpoint:wp",
			"work_item_ref":  "work-item:item",
			"work_items": []map[string]any{{
				"work_item_ref": "work-item:item",
				"item_id":       "item",
				"title":         "EPWA review item",
				"revision":      "1",
				"digest":        "digest",
				"authority": map[string]any{
					"acceptance_atom_refs":    []string{"atom:capture", "atom:verify"},
					"review_requirement_refs": []string{"review:req"},
					"reviewer_assignment_ref": "assignment:1",
					"reviewer_ref":            "user:reviewer",
				},
			}},
		},
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, recordID, "artifact.json"), body, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func newReviewFixture(t *testing.T) reviewFixture {
	t.Helper()
	root := t.TempDir()
	shareDir := filepath.Join(root, "evidence-share")
	reviewDir := filepath.Join(root, "epwa-review")
	recordID := "a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"
	if err := os.MkdirAll(filepath.Join(shareDir, recordID), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeReviewManifest(t, shareDir, recordID)
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", shareDir)
	t.Setenv("UIAI_EVIDENCE_REVIEW_DIR", reviewDir)
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: root}}
	router := chi.NewRouter()
	router.Route("/api/screenshot", func(r chi.Router) { mountEvidenceShare(r, cfg) })
	return reviewFixture{router: router, shareDir: shareDir, recordID: recordID, identity: &auth.Identity{UserID: "reviewer"}}
}

func reviewRequest(fixture reviewFixture, method, path string, body []byte, authenticated bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "https://engine.example/api/screenshot"+path, bytes.NewReader(body))
	if authenticated {
		request = request.WithContext(auth.ContextWithIdentity(request.Context(), fixture.identity))
	}
	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	return recorder
}

func TestReviewCaseUnknownRecordNotFound(t *testing.T) {
	fixture := newReviewFixture(t)
	response := reviewRequest(fixture, http.MethodGet, "/share/0000000000000000000000000000000000000000000000000000000000000000/review", nil, false)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body %s", response.Code, response.Body.String())
	}
}

func TestReviewCaseLoadsTypedProjection(t *testing.T) {
	fixture := newReviewFixture(t)
	response := reviewRequest(fixture, http.MethodGet, "/share/"+fixture.recordID+"/review", nil, false)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body %s", response.Code, response.Body.String())
	}
	var payload struct {
		Schema       string                             `json:"schema"`
		ReviewCase   evidenceaction.ReviewCase          `json:"review_case"`
		Dispositions []evidenceaction.ReviewDisposition `json:"dispositions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Schema != evidenceaction.ReviewCaseSchema || payload.ReviewCase.CaseRef == "" {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.ReviewCase.ReviewerAssignmentRef != "assignment:1" {
		t.Fatalf("assignment = %q", payload.ReviewCase.ReviewerAssignmentRef)
	}
	if len(payload.Dispositions) != 0 {
		t.Fatalf("fresh case must have zero dispositions, got %#v", payload.Dispositions)
	}
}

func reviewCaseRef(t *testing.T, fixture reviewFixture) string {
	t.Helper()
	response := reviewRequest(fixture, http.MethodGet, "/share/"+fixture.recordID+"/review", nil, false)
	if response.Code != http.StatusOK {
		t.Fatalf("review case status = %d", response.Code)
	}
	var payload struct {
		ReviewCase evidenceaction.ReviewCase `json:"review_case"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode review case: %v", err)
	}
	return payload.ReviewCase.CaseRef
}

func TestReviewDecisionRequiresAuthentication(t *testing.T) {
	fixture := newReviewFixture(t)
	response := reviewRequest(fixture, http.MethodPost, "/share/"+fixture.recordID+"/review/decision", []byte("{}"), false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if !strings.Contains(response.Body.String(), "reviewer_authentication_required") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestReviewDecisionBlocksUnassignedReviewer(t *testing.T) {
	fixture := newReviewFixture(t)
	fixture.identity = &auth.Identity{UserID: "someone-else"}
	response := reviewRequest(fixture, http.MethodPost, "/share/"+fixture.recordID+"/review/decision", []byte("{}"), true)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if !strings.Contains(response.Body.String(), "reviewer_not_assigned") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestReviewDecisionFailsClosedWithoutAuthority(t *testing.T) {
	fixture := newReviewFixture(t)
	body := map[string]any{
		"schema":                  evidenceaction.ReviewDecisionSchema,
		"case_ref":                reviewCaseRef(t, fixture),
		"artifact_ref":            "artifact:" + fixture.recordID,
		"artifact_sha256":         "a1b2c3",
		"work_item_ref":           "work-item:item",
		"decision":                evidenceaction.DecisionRejected,
		"reason":                  "proof withheld",
		"reviewer_assignment_ref": "assignment:1",
		"idempotency_key":         "review-1",
		"scope": map[string]any{
			"project_ref": "project:p", "workstream_ref": "workstream:w", "workset_ref": "workset:s",
			"callgraph_ref": "callgraph:g", "workpoint_ref": "workpoint:wp", "work_item_ref": "work-item:item",
		},
	}
	payload, _ := json.Marshal(body)
	response := reviewRequest(fixture, http.MethodPost, "/share/"+fixture.recordID+"/review/decision", payload, true)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "review_authority_unavailable") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestReviewDecisionApprovalRequiresProof(t *testing.T) {
	fixture := newReviewFixture(t)
	body := map[string]any{
		"schema":                  evidenceaction.ReviewDecisionSchema,
		"case_ref":                reviewCaseRef(t, fixture),
		"artifact_ref":            "artifact:" + fixture.recordID,
		"artifact_sha256":         "a1b2c3",
		"work_item_ref":           "work-item:item",
		"decision":                evidenceaction.DecisionApproved,
		"reviewer_assignment_ref": "assignment:1",
		"idempotency_key":         "review-2",
		"scope": map[string]any{
			"project_ref": "project:p", "workstream_ref": "workstream:w", "workset_ref": "workset:s",
			"callgraph_ref": "callgraph:g", "workpoint_ref": "workpoint:wp", "work_item_ref": "work-item:item",
		},
	}
	payload, _ := json.Marshal(body)
	response := reviewRequest(fixture, http.MethodPost, "/share/"+fixture.recordID+"/review/decision", payload, true)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "evidence_missing") {
		t.Fatalf("body = %s", response.Body.String())
	}
}
