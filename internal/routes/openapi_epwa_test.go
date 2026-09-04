package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestOpenAPIAdvertisesMandatoryEPWADeliveryEnvelope(t *testing.T) {
	router := chi.NewRouter()
	router.Route("/api", MountOpenAPI)

	schemaRec := httptest.NewRecorder()
	router.ServeHTTP(schemaRec, httptest.NewRequest(http.MethodGet, "/api/schema/uiai.epwa_delivery.v1", nil))
	if schemaRec.Code != http.StatusOK {
		t.Fatalf("delivery schema status=%d body=%s", schemaRec.Code, schemaRec.Body.String())
	}
	var deliverySchema map[string]any
	if err := json.Unmarshal(schemaRec.Body.Bytes(), &deliverySchema); err != nil {
		t.Fatal(err)
	}
	if deliverySchema["title"] != "UIAI EPWA Delivery" {
		t.Fatalf("unexpected delivery schema: %#v", deliverySchema)
	}

	openAPIRec := httptest.NewRecorder()
	router.ServeHTTP(openAPIRec, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if openAPIRec.Code != http.StatusOK {
		t.Fatalf("openapi status=%d body=%s", openAPIRec.Code, openAPIRec.Body.String())
	}
	var document struct {
		Paths      map[string]map[string]map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(openAPIRec.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	for _, component := range []string{"uiai_epwa_delivery_v1", "uiai_artifact_delivery_envelope_v2"} {
		if document.Components.Schemas[component] == nil {
			t.Fatalf("OpenAPI missing component %s", component)
		}
	}
	envelope := document.Components.Schemas["uiai_artifact_delivery_envelope_v2"].(map[string]any)
	properties := envelope["properties"].(map[string]any)
	deliveryRef := properties["epwa_delivery"].(map[string]any)["$ref"]
	if deliveryRef != "#/components/schemas/uiai_epwa_delivery_v1" {
		t.Fatalf("artifact envelope does not resolve the in-document EPWA contract: %v", deliveryRef)
	}
	artifactPaths := []string{
		"/api/screenshot", "/api/screenshot/compare", "/api/session/{sessionID}/screenshot",
		"/api/session/{sessionID}/snapshot", "/api/session/{sessionID}/dom", "/api/session/{sessionID}/read",
		"/api/session/{sessionID}/diagnostics", "/api/search", "/api/markdown", "/api/agent/research-packet",
		"/api/critique", "/api/media/frame/render", "/api/evidence/artifacts/commit",
		"/api/share/{id}/screenshot", "/api/fpv/share", "/api/intelligence/index/upload",
		"/api/intelligence/wasm/{runId}", "/api/intelligence/js/{runId}",
	}
	for _, path := range artifactPaths {
		operations := document.Paths[path]
		if operations == nil {
			t.Fatalf("OpenAPI missing artifact path %s", path)
		}
		for method, operation := range operations {
			responses, ok := operation["responses"].(map[string]any)
			if !ok {
				t.Fatalf("%s %s missing responses", method, path)
			}
			accepted, ok := responses["202"].(map[string]any)
			if !ok {
				t.Fatalf("%s %s missing fail-closed HTTP 202 response", method, path)
			}
			content := accepted["content"].(map[string]any)["application/json"].(map[string]any)
			schema := content["schema"].(map[string]any)
			if schema["$ref"] != "#/components/schemas/uiai_artifact_delivery_envelope_v2" {
				t.Fatalf("%s %s uses wrong delivery schema: %#v", method, path, schema)
			}
		}
	}
}
