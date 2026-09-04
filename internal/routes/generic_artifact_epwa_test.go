package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
)

func TestJSONArtifactPayloadIsWithheldUntilEPWADeliveryIsReady(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	payload := map[string]any{"schema": "uiai.test_report.v1", "report_rows": []string{"row-1"}}

	blockedReq := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/report", nil)
	blockedRec := httptest.NewRecorder()
	writeJSONArtifactEPWA(blockedRec, blockedReq, cfg, evidenceshare.Scope{}, "report:test", "Test report", "generated_report", payload, http.StatusOK)
	if blockedRec.Code != http.StatusAccepted {
		t.Fatalf("blocked status=%d body=%s", blockedRec.Code, blockedRec.Body.String())
	}
	var blocked map[string]any
	if err := json.Unmarshal(blockedRec.Body.Bytes(), &blocked); err != nil {
		t.Fatal(err)
	}
	if blocked["delivery_state"] != "blocked" || blocked["epwa_delivery"] == nil {
		t.Fatalf("missing fail-closed delivery: %#v", blocked)
	}
	if blocked["report_rows"] != nil || blocked["artifact_url"] != nil || blocked["portable_url"] != nil {
		t.Fatalf("blocked response exposed artifact payload or URL: %#v", blocked)
	}

	readyReq := httptest.NewRequest(http.MethodPost, "https://evidence.example/report", nil)
	readyRec := httptest.NewRecorder()
	writeJSONArtifactEPWA(readyRec, readyReq, cfg, completeEvidenceScope(), "report:test", "Test report", "generated_report", payload, http.StatusOK)
	if readyRec.Code != http.StatusOK {
		t.Fatalf("ready status=%d body=%s", readyRec.Code, readyRec.Body.String())
	}
	var ready map[string]any
	if err := json.Unmarshal(readyRec.Body.Bytes(), &ready); err != nil {
		t.Fatal(err)
	}
	if ready["delivery_state"] != "ready" || ready["epwa_delivery"] == nil || ready["artifact_url"] == nil || ready["portable_url"] == nil {
		t.Fatalf("ready response missing EPWA delivery: %#v", ready)
	}
	if ready["report_rows"] == nil {
		t.Fatalf("ready response omitted report payload: %#v", ready)
	}
}
