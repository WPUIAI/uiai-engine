package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestSessionScreenshotOutputIsOnlyHTTPSPortableEPWA(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	snap := &vision.SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(append([]byte("\x89PNG\r\n\x1a\n"), []byte("session evidence")...)),
		Format:     "png", Size: 24, Width: 375, Height: 812, URL: "https://example.test", Title: "Example", Duration: 12,
	}
	scope := completeEvidenceScope()
	sess := &vision.Session{FocusaScope: &vision.FocusaScope{
		ProjectRef: scope.ProjectRef, WorkstreamRef: scope.WorkstreamRef, WorksetRef: scope.WorksetRef,
		CallGraphRef: scope.CallGraphRef, WorkpointID: scope.WorkpointRef, WorkItemRef: scope.WorkItemRef,
		WorkItems: scope.WorkItems, ContinuityID: scope.ContinuityRef,
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://engine.example/api/session/example/screenshot", nil)
	writeScreenshotOutput(recorder, request, cfg, sess, snap, "file")
	if recorder.Code != http.StatusOK {
		t.Fatalf("session delivery status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["screenshot"] != nil || response["artifact_path"] != nil || response["delivery_state"] != "ready" || response["raw_output_posture"] != "withheld_by_mandatory_epwa_delivery" {
		t.Fatalf("session raw-only output escaped: %#v", response)
	}
	for _, key := range []string{"artifact_url", "portable_url"} {
		value, _ := response[key].(string)
		if !strings.HasPrefix(value, "https://engine.example/api/screenshot/share/") {
			t.Fatalf("%s is not an HTTPS EPWA link: %q", key, value)
		}
	}
}
