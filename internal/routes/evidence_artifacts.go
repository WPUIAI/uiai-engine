package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

const maxEvidenceCommitBody = 10 << 20

func MountEvidenceArtifacts(r chi.Router, cfg *config.Config, artifacts *evidenceartifact.Store, registry *evidenceregistry.Manager) {
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
		response := map[string]any{"schema": "uiai.evidence_artifact_commit_result.v2", "artifact_ref": commit.ArtifactID, "commit": commit, "registry_state": "projection_pending"}
		delivery, deliveryErr := publishStoredArtifactEPWA(req, cfg, artifacts, commit.ArtifactID, commit.Revision, "artifact:"+commit.CommitID)
		if deliveryErr != nil {
			response["epwa_delivery_error"] = epwaPublishError{
				Schema: "uiai.epwa_delivery_error.v1", State: epwadelivery.StatePendingReconcile,
				ArtifactRef: commit.ArtifactID, ArtifactSHA256: commit.ManifestSHA256,
				RecoveryRef: "reconcile:evidence-artifact-epwa-publication",
				Error:       epwaPublishErrorDetail{Code: "epwa_publication_failed", Message: deliveryErr.Error(), Retryable: true},
			}
			if delivery.Schema == epwadelivery.Schema {
				response["delivery_state"] = delivery.State
				response["epwa_delivery"] = delivery
			} else {
				response["schema"] = "uiai.evidence_artifact_commit_delivery_error.v1"
			}
		} else {
			response["delivery_state"] = delivery.State
			response["epwa_delivery"] = delivery
			if delivery.State == epwadelivery.StateReady {
				response["artifact_url"] = delivery.EPWA.RecordURL
				response["portable_url"] = delivery.EPWA.PortableURL
			}
		}
		if project, registryErr := registry.EnsureProject(req.Context(), manifest.Scope.Project.ProjectRef); registryErr != nil {
			response["registry_error"] = registryErr.Error()
		} else if indexed, indexErr := project.Index(req.Context(), evidenceregistry.IndexInput{Manifest: manifest, ManifestSHA256: commit.ManifestSHA256}); indexErr != nil {
			response["registry_error"] = indexErr.Error()
		} else {
			response["registry_state"] = "projected"
			response["registry"] = indexed
		}
		status := http.StatusAccepted
		if deliveryErr == nil && delivery.State == epwadelivery.StateReady && response["registry_state"] == "projected" {
			status = http.StatusCreated
		}
		writeJSON(w, status, response)
	})

	r.Post("/rebuild", func(w http.ResponseWriter, req *http.Request) {
		result, err := registry.RebuildFromArtifactStore(req.Context(), artifacts)
		if err != nil {
			writeRegistryError(w, err)
			return
		}
		cursor, _ := strconv.Atoi(req.URL.Query().Get("epwa_cursor"))
		if cursor < 0 {
			cursor = 0
		}
		limit, _ := strconv.Atoi(req.URL.Query().Get("epwa_limit"))
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		entries := artifacts.List()
		if cursor > len(entries) {
			cursor = len(entries)
		}
		end := cursor + limit
		if end > len(entries) {
			end = len(entries)
		}
		deliveries := make([]epwadelivery.Delivery, 0, end-cursor)
		failures := make([]map[string]any, 0)
		allReady := true
		for _, entry := range entries[cursor:end] {
			delivery, deliveryErr := publishStoredArtifactEPWA(req, cfg, artifacts, entry.ArtifactID, entry.Revision, "artifact:"+entry.CommitID)
			if deliveryErr != nil {
				allReady = false
				failures = append(failures, map[string]any{"artifact_ref": entry.ArtifactID, "revision": entry.Revision, "error": deliveryErr.Error(), "recovery_ref": "reconcile:evidence-artifact-epwa-publication"})
				continue
			}
			deliveries = append(deliveries, delivery)
			if delivery.State != epwadelivery.StateReady {
				allReady = false
			}
		}
		backfill := map[string]any{"schema": "uiai.epwa_backfill_result.v1", "cursor": cursor, "processed": end - cursor, "total": len(entries), "deliveries": deliveries, "failures": failures}
		if end < len(entries) {
			backfill["next_cursor"] = end
			allReady = false
		}
		status := http.StatusAccepted
		if allReady {
			status = http.StatusOK
		}
		writeJSON(w, status, map[string]any{"schema": "uiai.evidence_artifact_rebuild_result.v2", "registry": result, "epwa_backfill": backfill})
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
