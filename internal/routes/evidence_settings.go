package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func mountEvidenceShareSettings(r chi.Router, store *evidenceshare.SettingsStore) {
	r.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, store.Effective(settingsScope(req)))
	})
	r.Post("/settings/preview", func(w http.ResponseWriter, req *http.Request) {
		var body settingsMutation
		if !decodeSettingsMutation(w, req, &body) {
			return
		}
		result, err := store.Preview(body.Scope(), body.Values)
		if err != nil {
			writeSettingsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	r.Put("/settings", func(w http.ResponseWriter, req *http.Request) {
		var body settingsMutation
		if !decodeSettingsMutation(w, req, &body) {
			return
		}
		result, err := store.Update(body.Scope(), body.ExpectedRevision, body.Values)
		if err != nil {
			writeSettingsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	r.Delete("/settings", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			ProjectRef       string `json:"project_ref"`
			WorkstreamRef    string `json:"workstream_ref"`
			ExpectedRevision uint64 `json:"expected_revision"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if err := store.Reset(evidenceshare.SettingsScope{ProjectRef: body.ProjectRef, WorkstreamRef: body.WorkstreamRef}, body.ExpectedRevision); err != nil {
			writeSettingsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reset": true, "scope": body.ProjectRef})
	})
}

type settingsMutation struct {
	ProjectRef       string         `json:"project_ref"`
	WorkstreamRef    string         `json:"workstream_ref"`
	ExpectedRevision uint64         `json:"expected_revision"`
	Values           map[string]any `json:"values"`
}

func (m settingsMutation) Scope() evidenceshare.SettingsScope {
	return evidenceshare.SettingsScope{ProjectRef: m.ProjectRef, WorkstreamRef: m.WorkstreamRef}
}
func settingsScope(req *http.Request) evidenceshare.SettingsScope {
	return evidenceshare.SettingsScope{ProjectRef: req.URL.Query().Get("project_ref"), WorkstreamRef: req.URL.Query().Get("workstream_ref")}
}
func decodeSettingsMutation(w http.ResponseWriter, req *http.Request, body *settingsMutation) bool {
	if err := json.NewDecoder(req.Body).Decode(body); err != nil || body.Values == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "values object required"})
		return false
	}
	return true
}
func writeSettingsError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "invalid_settings"
	if errors.Is(err, evidenceshare.ErrSettingsConflict) {
		status = http.StatusConflict
		code = "revision_conflict"
	}
	writeJSON(w, status, map[string]string{"error": code, "message": strings.TrimSpace(err.Error())})
}
