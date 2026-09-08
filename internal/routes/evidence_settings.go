package routes

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func mountEvidenceShareSettings(r chi.Router, store *evidenceshare.SettingsStore) {
	r.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
		scope := settingsScope(req)
		if err := scope.Validate(); err != nil {
			writeSettingsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, store.Effective(scope))
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
		if !decodeSettingsRequest(w, req, &body) {
			return
		}
		if err := store.Reset(evidenceshare.SettingsScope{ProjectRef: body.ProjectRef, WorkstreamRef: body.WorkstreamRef}, body.ExpectedRevision); err != nil {
			writeSettingsError(w, err)
			return
		}
		revision := body.ExpectedRevision
		if revision > 0 {
			revision++
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": "uiai.evidence_share_settings_reset.v1", "reset": true, "scope": evidenceshare.SettingsScope{ProjectRef: body.ProjectRef, WorkstreamRef: body.WorkstreamRef}, "revision": revision})
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
func decodeSettingsRequest(w http.ResponseWriter, req *http.Request, body any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var trailing any
	if err := decoder.Decode(body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid settings JSON or fields"})
		return false
	}
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "exactly one settings object required"})
		return false
	}
	return true
}

func decodeSettingsMutation(w http.ResponseWriter, req *http.Request, body *settingsMutation) bool {
	if !decodeSettingsRequest(w, req, body) {
		return false
	}
	if body.Values == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "values object required"})
		return false
	}
	return true
}
func writeSettingsUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "evidence_settings_unavailable", "message": "Canonical evidence settings are unavailable; capture is blocked rather than using fallback defaults."})
}

func writeSettingsError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "invalid_settings"
	message := strings.TrimSpace(err.Error())
	if errors.Is(err, evidenceshare.ErrSettingsConflict) {
		status = http.StatusConflict
		code = "revision_conflict"
	} else if !errors.Is(err, evidenceshare.ErrSettingsInvalid) {
		slog.Error("evidence settings publication failed", "error", err)
		status = http.StatusInternalServerError
		code = "settings_persistence_failed"
		message = "Settings publication failed; reload and inspect storage before retrying."
	}
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}
