package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func TestEvidenceShareSettingsRoutesPreviewUpdateConflict(t *testing.T) {
	store, err := evidenceshare.NewSettingsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	mountEvidenceShareSettings(r, store)
	preview := httptest.NewRecorder()
	r.ServeHTTP(preview, httptest.NewRequest(http.MethodPost, "/settings/preview", strings.NewReader(`{"project_ref":"project:homepage","values":{"presentation":{"theme":"dark"}}}`)))
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	update := httptest.NewRecorder()
	r.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"project_ref":"project:homepage","expected_revision":0,"values":{"presentation":{"theme":"dark"}}}`)))
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	conflict := httptest.NewRecorder()
	r.ServeHTTP(conflict, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"project_ref":"project:homepage","expected_revision":0,"values":{"presentation":{"theme":"light"}}}`)))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(update.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schema"] != evidenceshare.SettingsSchema {
		t.Fatalf("schema=%v", body["schema"])
	}
}
