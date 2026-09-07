package intelligence

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/go-chi/chi/v5"
)

func newEPWATestLayer(t *testing.T, state epwadelivery.State) (*Layer, http.Handler) {
	t.Helper()
	layer := NewLayer(&config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}, nil)
	layer.svcToken = "service-test-token"
	layer.SetArtifactPublisher(func(_ *http.Request, artifactRef, _, _, _, _ string, payload []byte) (epwadelivery.Delivery, error) {
		delivery := epwadelivery.Delivery{
			Schema: "uiai.epwa_delivery.v1", State: state,
			Artifact: epwadelivery.ArtifactBinding{ArtifactRef: artifactRef},
		}
		if state == epwadelivery.StateReady {
			delivery.EPWA.RecordURL = "https://evidence.example/share/package/"
			delivery.EPWA.PortableURL = "https://evidence.example/share/package/portable.zip"
		}
		if len(payload) == 0 {
			t.Fatal("publisher received empty artifact")
		}
		return delivery, nil
	})
	router := chi.NewRouter()
	layer.Mount(router)
	return layer, router
}

func serviceRequest(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer service-test-token")
	return req
}

func TestIntelligenceRawRuntimeArtifactsAreRetired(t *testing.T) {
	_, router := newEPWATestLayer(t, epwadelivery.StateReady)
	for _, path := range []string{"/wasm/run-1", "/js/run-1"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusGone {
			t.Fatalf("GET %s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("legacy_raw_artifact_removed")) {
			t.Fatalf("GET %s missing retired-route code: %s", path, rec.Body.String())
		}
	}
}

func TestIntelligenceUploadWithholdsCompatibilityArtifactUntilEPWAReady(t *testing.T) {
	layer, router := newEPWATestLayer(t, epwadelivery.StatePendingReconcile)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, serviceRequest(http.MethodPost, "/wasm/run-pending", []byte("\x00asm\x01\x00\x00\x00")))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if layer.store.HasArtifact("run-pending", "docfind_bg.wasm") {
		t.Fatal("compatibility artifact was stored before ready EPWA delivery")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["delivery_state"] != string(epwadelivery.StatePendingReconcile) || body["epwa_delivery"] == nil {
		t.Fatalf("unexpected delivery envelope: %#v", body)
	}
	if _, ok := body["artifact_url"]; ok {
		t.Fatalf("pending delivery exposed artifact_url: %#v", body)
	}
}

func TestIntelligenceUploadStoresOnlyAfterReadyEPWADelivery(t *testing.T) {
	layer, router := newEPWATestLayer(t, epwadelivery.StateReady)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, serviceRequest(http.MethodPost, "/js/run-ready", []byte("export const ready = true;")))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !layer.store.HasArtifact("run-ready", "docfind.js") {
		t.Fatal("ready EPWA delivery did not permit compatibility artifact storage")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["artifact_url"] != "https://evidence.example/share/package/" || body["portable_url"] == nil {
		t.Fatalf("ready response missing EPWA URLs: %#v", body)
	}
}

func TestIntelligenceIndexUploadIsAtomicAcrossEPWADeliveries(t *testing.T) {
	layer, router := newEPWATestLayer(t, epwadelivery.StatePendingReconcile)
	body := []byte(`{"runId":"run-index","documents":[{"id":"doc-1","title":"Doc","content":"body"}]}`)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, serviceRequest(http.MethodPost, "/index/upload", body))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	docs, err := layer.store.ReadDocuments("run-index")
	if err != nil {
		t.Fatal(err)
	}
	if docs != nil {
		t.Fatalf("documents stored before all EPWA deliveries were ready: docs=%v", docs)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("withheld_until_all_epwa_deliveries_ready")) {
		t.Fatalf("missing atomic withholding posture: %s", rec.Body.String())
	}
}
