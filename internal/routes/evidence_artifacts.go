package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

const maxEvidenceCommitBody = 10 << 20

func MountEvidenceArtifacts(r chi.Router, artifacts *evidenceartifact.Store, registry *evidenceregistry.Manager) {
	r.Post("/commit", func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, maxEvidenceCommitBody)
		if err := req.ParseMultipartForm(maxEvidenceCommitBody); err != nil {
			writeEvidenceArtifactError(w, http.StatusBadRequest, "invalid_multipart", err)
			return
		}
		defer req.MultipartForm.RemoveAll()
		decoder := json.NewDecoder(strings.NewReader(req.FormValue("manifest")))
		decoder.DisallowUnknownFields()
		var manifest evidenceartifact.Manifest
		if err := decoder.Decode(&manifest); err != nil {
			writeEvidenceArtifactError(w, http.StatusBadRequest, "invalid_manifest", err)
			return
		}
		if len(req.MultipartForm.File) != len(manifest.Assets) {
			writeEvidenceArtifactError(w, http.StatusBadRequest, "unexpected_assets", fmt.Errorf("manifest declares %d assets but request has %d asset fields", len(manifest.Assets), len(req.MultipartForm.File)))
			return
		}
		readers := make(map[string]io.Reader, len(manifest.Assets))
		files := make([]io.Closer, 0, len(manifest.Assets))
		defer func() {
			for _, file := range files {
				_ = file.Close()
			}
		}()
		for _, asset := range manifest.Assets {
			headers := req.MultipartForm.File[asset.AssetID]
			if len(headers) != 1 {
				writeEvidenceArtifactError(w, http.StatusBadRequest, "asset_cardinality", fmt.Errorf("asset %q requires exactly one part", asset.AssetID))
				return
			}
			file, err := headers[0].Open()
			if err != nil {
				writeEvidenceArtifactError(w, http.StatusBadRequest, "asset_unavailable", err)
				return
			}
			files = append(files, file)
			readers[asset.AssetID] = file
		}
		commit, err := artifacts.Commit(req.Context(), manifest, readers)
		if err != nil {
			writeEvidenceArtifactError(w, http.StatusUnprocessableEntity, "commit_rejected", err)
			return
		}
		project, err := registry.EnsureProject(req.Context(), manifest.Scope.Project.ProjectRef)
		if err != nil {
			writeJSON(w, http.StatusAccepted, map[string]any{"schema": "uiai.evidence_artifact_commit_result.v1", "commit": commit, "registry_state": "projection_pending", "registry_error": err.Error()})
			return
		}
		indexed, err := project.Index(req.Context(), evidenceregistry.IndexInput{Manifest: manifest, ManifestSHA256: commit.ManifestSHA256})
		if err != nil {
			writeJSON(w, http.StatusAccepted, map[string]any{"schema": "uiai.evidence_artifact_commit_result.v1", "commit": commit, "registry_state": "projection_pending", "registry_error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"schema": "uiai.evidence_artifact_commit_result.v1", "commit": commit, "registry": indexed})
	})

	r.Post("/rebuild", func(w http.ResponseWriter, req *http.Request) {
		result, err := registry.RebuildFromArtifactStore(req.Context(), artifacts)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Get("/manifest", func(w http.ResponseWriter, req *http.Request) {
		revision, err := strconv.ParseUint(req.URL.Query().Get("revision"), 10, 64)
		if err != nil || revision == 0 {
			writeEvidenceArtifactError(w, http.StatusBadRequest, "invalid_revision", evidenceartifact.ErrArtifactNotFound)
			return
		}
		manifest, entry, err := artifacts.GetManifest(req.URL.Query().Get("artifact_id"), revision)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, evidenceartifact.ErrArtifactNotFound) {
				status = http.StatusNotFound
			}
			writeEvidenceArtifactError(w, status, "manifest_unavailable", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schema": "uiai.evidence_artifact_manifest_read.v1", "manifest": manifest, "entry": entry})
	})

	r.Get("/assets/{sha256}", func(w http.ResponseWriter, req *http.Request) {
		file, record, err := artifacts.OpenAsset(chi.URLParam(req, "sha256"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, evidenceartifact.ErrAssetNotFound) {
				status = http.StatusNotFound
			}
			writeEvidenceArtifactError(w, status, "asset_unavailable", err)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", record.MediaType)
		w.Header().Set("Content-Length", strconv.FormatInt(record.ByteSize, 10))
		w.Header().Set("Cache-Control", "private, immutable, max-age=31536000")
		w.Header().Set("ETag", `"sha256:`+record.SHA256+`"`)
		_, _ = io.Copy(w, file)
	})
}

func writeEvidenceArtifactError(w http.ResponseWriter, status int, code string, err error) {
	writeJSON(w, status, map[string]any{"schema": "uiai.evidence_artifact_error.v1", "error": map[string]any{"code": code, "message": err.Error()}})
}
