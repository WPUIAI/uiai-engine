package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

func proofRouter(t *testing.T) chi.Router {
	t.Helper()
	router := chi.NewRouter()
	MountProofPublishRoutes(router, &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}})
	return router
}

// #106 acceptance 1: a multi-breakpoint report publishes through the EPWA
// contract and the durable artifact outlives the session — the surface is
// session-independent by construction, so there is nothing to close first.
func TestProofPublishComposesDurableReport(t *testing.T) {
	router := proofRouter(t)
	body := `{
		"title": "Homepage responsive proof",
		"target_url": "https://focusa.dev/",
		"engine_version": "engine-v2.0.57",
		"viewports": [
			{"width": 1440, "height": 900, "label": "desktop", "screenshot_url": "https://e.example.test/s1", "screenshot_ref": "uiai-artifact:sha256:1111111111111111111111111111111111111111111111111111111111111111"},
			{"width": 768, "height": 1024, "label": "tablet", "screenshot_url": "https://e.example.test/s2"},
			{"width": 390, "height": 844, "label": "mobile", "screenshot_url": "https://e.example.test/s3"}
		],
		"assertions": [
			{"name": "hero visible", "expected": "visible", "actual": "visible", "passed": true},
			{"name": "menu opens on tap", "expected": "scrollY changes", "actual": "scrollY 0 to 48", "passed": true}
		],
		"diagnostics": {"console_errors": 0, "exceptions": 0, "failed_requests": 1, "http_4xx": 0, "http_5xx": 0},
		"interaction_notes": ["tap hamburger at 390px", "menu expanded; scrollY changed"]
	}`
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest("POST", "/publish", strings.NewReader(body)))
	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.Code != http.StatusOK && resp.Code != http.StatusAccepted {
		t.Fatalf("publish failed: status=%d body=%.200s", resp.Code, resp.Body.String())
	}
	ref, _ := result["artifact_ref"].(string)
	if ref == "" {
		t.Fatal("publish returned no artifact_ref — the durable handle is the point")
	}
	if result["viewport_count"].(float64) != 3 {
		t.Fatalf("viewport count wrong: %v", result["viewport_count"])
	}
	asserts := result["assertions"].(map[string]any)
	if asserts["passed"].(float64) != 2 || asserts["failed"].(float64) != 0 {
		t.Fatalf("assertion tally wrong: %v", asserts)
	}
}

// #106 acceptance 3: secrets cannot appear — caller text passes the
// secrettext scrubber before assembly, and a secret-looking assertion value
// surfaces as the marker, never verbatim.
func TestProofPublishRedactsCallerText(t *testing.T) {
	router := proofRouter(t)
	body := `{
		"title": "auth flow proof",
		"viewports": [{"width": 390, "height": 844, "screenshot_url": "https://e.example.test/a1"}],
		"assertions": [
			{"name": "api key echoes", "expected": "hidden", "actual": "sk-ant-api03-ABCDEFGH1234567890abcdefgh", "passed": false}
		],
		"interaction_notes": ["token sk-ant-api03-ABCDEFGH1234567890abcdefgh appeared in console"]
	}`
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest("POST", "/publish", strings.NewReader(body)))
	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusOK && resp.Code != http.StatusAccepted {
		t.Fatalf("publish failed: status=%d", resp.Code)
	}
	raw := resp.Body.String()
	if strings.Contains(raw, "sk-ant-api03-ABCDEFGH1234567890abcdefgh") {
		t.Fatal("secret-looking value survived redaction into the publish response")
	}
}

// Bounds and validation: title required, viewports bounded, screenshot
// links must be http(s), and over-long inputs are refused, never truncated
// into a misleading report.
func TestProofPublishValidationBounds(t *testing.T) {
	router := proofRouter(t)
	cases := []struct {
		name string
		body string
		code string
	}{
		{"missing title", `{"viewports":[{"width":1,"height":1}]}`, "proof_title_required"},
		{"no viewports", `{"title":"x"}`, "proof_viewports_bounded"},
		{"bad screenshot url", `{"title":"x","viewports":[{"width":1,"height":1,"screenshot_url":"javascript:alert(1)"}]}`, "proof_screenshot_url_invalid"},
	}
	for _, tc := range cases {
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest("POST", "/publish", strings.NewReader(tc.body)))
		if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), tc.code) {
			t.Fatalf("%s: status=%d body=%.150s", tc.name, resp.Code, resp.Body.String())
		}
	}
}
