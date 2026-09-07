package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLegacyScreenshotArtifactRouteIsRetired(t *testing.T) {
	router := chi.NewRouter()
	mountScreenshotArtifact(router)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/deadbeefdeadbeef", nil))
	if recorder.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "legacy_raw_artifact_removed") {
		t.Fatalf("missing retired-route code: %s", recorder.Body.String())
	}
}
