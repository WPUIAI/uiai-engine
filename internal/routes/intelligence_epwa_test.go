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

func TestRuntimeArtifactsRequireEPWADeliveryAndRawRouteIsGone(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	r := chi.NewRouter()
	r.Route("/intelligence", func(r chi.Router) { MountIntelligenceReal(r, cfg, nil) })

	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/intelligence/wasm/run-1", strings.NewReader("\x00asm\x01\x00\x00\x00"))
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["schema"] != "uiai.binary_artifact_result.v2" || body["delivery_state"] != "ready" || body["artifact_url"] == "" || body["portable_url"] == "" {
		t.Fatalf("runtime artifact escaped EPWA delivery: %#v", body)
	}
	delivery := body["epwa_delivery"].(map[string]any)
	if delivery["producer"] != "artifact.runtime" {
		t.Fatalf("producer=%#v", delivery["producer"])
	}

	rawReq := httptest.NewRequest(http.MethodGet, "/intelligence/wasm/run-1", nil)
	rawRes := httptest.NewRecorder()
	r.ServeHTTP(rawRes, rawReq)
	if rawRes.Code != http.StatusGone || !strings.Contains(rawRes.Body.String(), "legacy_raw_artifact_removed") {
		t.Fatalf("raw route status=%d body=%s", rawRes.Code, rawRes.Body.String())
	}
}
