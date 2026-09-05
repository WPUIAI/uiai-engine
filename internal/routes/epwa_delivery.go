package routes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

var ErrEPWAHTTPSUnavailable = errors.New("canonical EPWA HTTPS base URL unavailable")

type epwaPublishError struct {
	Schema         string                 `json:"schema"`
	State          epwadelivery.State     `json:"state"`
	ArtifactRef    string                 `json:"artifact_ref,omitempty"`
	ArtifactSHA256 string                 `json:"artifact_sha256,omitempty"`
	RecoveryRef    string                 `json:"recovery_ref"`
	Error          epwaPublishErrorDetail `json:"error"`
}

type epwaPublishErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func publishScreenshotEPWA(req *http.Request, cfg *config.Config, input evidenceshare.Input, producers ...epwadelivery.ProducerID) (epwadelivery.Delivery, error) {
	producer := epwadelivery.ProducerScreenshot
	if len(producers) > 0 {
		producer = producers[0]
	}
	if !epwadelivery.ValidProducer(producer) {
		return epwadelivery.Delivery{}, errors.New("unregistered EPWA producer")
	}
	share, err := evidenceshare.Assemble(evidenceShareDir(cfg), input)
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	state := epwadelivery.StateReady
	scopePosture := epwadelivery.ScopeComplete
	access := epwadelivery.AccessPublicSafe
	recoveryRef, recordURL, portableURL := "", "", ""
	if !input.Scope.Complete() || share.ProjectionRef == "" {
		state, scopePosture, access = epwadelivery.StateBlocked, epwadelivery.ScopeBlocked, epwadelivery.AccessRedacted
		recoveryRef = "reconcile:epwa-scope-required"
	} else if base, baseErr := canonicalEPWABase(req); baseErr != nil {
		state = epwadelivery.StatePendingReconcile
		recoveryRef = "reconcile:epwa-https-required"
	} else {
		recordURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/"}).String()
		portableURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/portable.zip"}).String()
	}
	createdAt := input.CapturedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	observedAt := time.Now().UTC()
	if observedAt.Before(createdAt) {
		observedAt = createdAt
	}
	delivery, err := epwadelivery.New(epwadelivery.Input{
		Producer: producer,
		Artifact: epwadelivery.ArtifactBinding{ArtifactRef: share.ArtifactRef, Revision: 1, ManifestSHA256: share.ManifestSHA256, OutputSHA256: share.OutputSHA256},
		EPWA: epwadelivery.EPWABinding{
			PackageID: share.PackageID, ProjectionRef: share.ProjectionRef, ProjectionSHA256: share.ProjectionSHA256,
			PackageRef: "uiai-epwa-package:sha256:" + share.PackageSHA256, PackageSHA256: share.PackageSHA256,
			RecordURL: recordURL, PortableURL: portableURL, Access: access,
		},
		Scope: epwadelivery.ScopeBinding{
			Posture: scopePosture, ProjectRef: input.Scope.ProjectRef, WorkstreamRef: input.Scope.WorkstreamRef,
			WorksetRef: input.Scope.WorksetRef, CallGraphRef: input.Scope.CallGraphRef,
			WorkpointRef: input.Scope.WorkpointRef, WorkItemRef: input.Scope.WorkItemRef, ContinuityRef: input.Scope.ContinuityRef,
		},
		State: state, IdempotencyKey: "capture:" + share.ArtifactRef,
		CreatedAt: createdAt, ObservedAt: observedAt, RecoveryRef: recoveryRef,
	})
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	return epwadelivery.Record(epwaDeliveryRoot(cfg), delivery)
}

func publishGenericEPWA(req *http.Request, cfg *config.Config, input evidenceshare.GenericInput) (epwadelivery.Delivery, error) {
	base, _ := canonicalEPWABase(req)
	return publishGenericEPWAAtBase(cfg, input, base)
}

func publishGenericEPWAAtBase(cfg *config.Config, input evidenceshare.GenericInput, base *url.URL) (epwadelivery.Delivery, error) {
	share, err := evidenceshare.AssembleGeneric(evidenceShareDir(cfg), input)
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	state, scopePosture, access := epwadelivery.StatePendingReconcile, epwadelivery.ScopeComplete, epwadelivery.AccessPublicSafe
	recoveryRef := "reconcile:epwa-https-required"
	recordURL, portableURL := "", ""
	if !input.Scope.Complete() {
		state, scopePosture, access = epwadelivery.StateBlocked, epwadelivery.ScopeBlocked, epwadelivery.AccessRedacted
		recoveryRef = "reconcile:epwa-scope-required"
	} else if base != nil && base.Scheme == "https" && base.Host != "" {
		state, recoveryRef = epwadelivery.StateReady, ""
		recordURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/"}).String()
		portableURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/portable.zip"}).String()
	}
	createdAt := input.CapturedAt.UTC()
	revision := input.Revision
	if revision == 0 {
		revision = 1
	}
	observedAt := time.Now().UTC()
	if observedAt.Before(createdAt) {
		observedAt = createdAt
	}
	delivery, err := epwadelivery.New(epwadelivery.Input{
		Producer: epwadelivery.ProducerForArtifactKind(input.Kind),
		Artifact: epwadelivery.ArtifactBinding{ArtifactRef: share.ArtifactRef, Revision: revision, ManifestSHA256: share.ManifestSHA256, OutputSHA256: share.OutputSHA256},
		EPWA: epwadelivery.EPWABinding{
			PackageID: share.PackageID, ProjectionRef: share.ProjectionRef, ProjectionSHA256: share.ProjectionSHA256,
			PackageRef: "uiai-epwa-package:sha256:" + share.PackageSHA256, PackageSHA256: share.PackageSHA256,
			RecordURL: recordURL, PortableURL: portableURL, Access: access,
		},
		Scope: epwadelivery.ScopeBinding{
			Posture: scopePosture, ProjectRef: input.Scope.ProjectRef, WorkstreamRef: input.Scope.WorkstreamRef,
			WorksetRef: input.Scope.WorksetRef, CallGraphRef: input.Scope.CallGraphRef, WorkpointRef: input.Scope.WorkpointRef,
			WorkItemRef: input.Scope.WorkItemRef, ContinuityRef: input.Scope.ContinuityRef,
		},
		State: state, IdempotencyKey: "artifact:" + share.ArtifactRef + ":" + share.ManifestSHA256,
		CreatedAt: createdAt, ObservedAt: observedAt, RecoveryRef: recoveryRef,
	})
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	return epwadelivery.Record(epwaDeliveryRoot(cfg), delivery)
}

func publishStoredArtifactEPWA(req *http.Request, cfg *config.Config, artifacts *evidenceartifact.Store, artifactID string, revision uint64, idempotencyKey string) (epwadelivery.Delivery, error) {
	manifest, _, err := artifacts.GetManifest(artifactID, revision)
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	assetData := make(map[string][]byte, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		file, record, err := artifacts.OpenAsset(asset.SHA256)
		if err != nil {
			return epwadelivery.Delivery{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, record.ByteSize+1))
		closeErr := file.Close()
		if readErr != nil {
			return epwadelivery.Delivery{}, readErr
		}
		if closeErr != nil {
			return epwadelivery.Delivery{}, closeErr
		}
		if int64(len(data)) != record.ByteSize {
			return epwadelivery.Delivery{}, evidenceartifact.ErrAssetMismatch
		}
		assetData[asset.AssetID] = data
	}
	share, err := evidenceshare.AssembleArtifact(evidenceShareDir(cfg), evidenceshare.ArtifactInput{Manifest: manifest, Assets: assetData})
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	state := epwadelivery.StateBlocked
	access := epwadelivery.AccessPrivate
	scopePosture := epwadelivery.ScopeComplete
	recoveryRef := "reconcile:epwa-access-required"
	recordURL, portableURL := "", ""
	if strings.TrimSpace(manifest.Scope.ContinuityRef) == "" {
		scopePosture, access, recoveryRef = epwadelivery.ScopeBlocked, epwadelivery.AccessRedacted, "reconcile:epwa-scope-required"
	}
	if manifest.Policy.AccessClass == evidenceartifact.AccessUnlisted {
		access = epwadelivery.AccessUnlisted
	}
	if manifest.Policy.RedactionState == evidenceartifact.RedactionBlocked || artifactHasBlockedAssets(manifest) {
		state, access, recoveryRef = epwadelivery.StateRedacted, epwadelivery.AccessRedacted, "reconcile:epwa-redaction-required"
	} else if scopePosture == epwadelivery.ScopeComplete && manifest.Policy.AccessClass == evidenceartifact.AccessPublicSafe && manifest.Policy.RedactionState == evidenceartifact.RedactionPublicSafe {
		access = epwadelivery.AccessPublicSafe
		if base, baseErr := canonicalEPWABase(req); baseErr != nil {
			state, recoveryRef = epwadelivery.StatePendingReconcile, "reconcile:epwa-https-required"
		} else {
			state, recoveryRef = epwadelivery.StateReady, ""
			recordURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/"}).String()
			portableURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + share.PackageID + "/portable.zip"}).String()
		}
	}
	createdAt, err := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	observedAt := time.Now().UTC()
	if observedAt.Before(createdAt) {
		observedAt = createdAt
	}
	delivery, err := epwadelivery.New(epwadelivery.Input{
		Producer: epwadelivery.ProducerArtifactCommit,
		Artifact: epwadelivery.ArtifactBinding{ArtifactRef: manifest.ArtifactID, Revision: manifest.Revision, ManifestSHA256: manifest.Integrity.ManifestSHA256, OutputSHA256: manifest.Integrity.BundleSHA256},
		EPWA: epwadelivery.EPWABinding{
			PackageID: share.PackageID, ProjectionRef: share.ProjectionRef, ProjectionSHA256: share.ProjectionSHA256,
			PackageRef: "uiai-epwa-package:sha256:" + share.PackageSHA256, PackageSHA256: share.PackageSHA256,
			RecordURL: recordURL, PortableURL: portableURL, Access: access,
		},
		Scope: epwadelivery.ScopeBinding{
			Posture: scopePosture, ProjectRef: manifest.Scope.Project.ProjectRef, WorkstreamRef: manifest.Scope.Workstream.WorkstreamRef,
			WorksetRef: manifest.Scope.Workset.WorksetRef, CallGraphRef: manifest.Scope.CallGraph.RunRef,
			WorkpointRef: manifest.Scope.Workpoint.WorkpointRef, WorkItemRef: manifest.Scope.WorkItems[0].WorkItemRef,
			ContinuityRef: manifest.Scope.ContinuityRef,
		},
		State: state, IdempotencyKey: idempotencyKey, CreatedAt: createdAt, ObservedAt: observedAt, RecoveryRef: recoveryRef,
	})
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	recorded, recordErr := epwadelivery.Record(epwaDeliveryRoot(cfg), delivery)
	if recordErr == nil {
		return recorded, nil
	}
	delivery.State = epwadelivery.StatePendingReconcile
	delivery.EPWA.RecordURL = ""
	delivery.EPWA.PortableURL = ""
	delivery.RecoveryRef = "reconcile:epwa-delivery-record"
	delivery.ObservedAt = time.Now().UTC()
	if validationErr := epwadelivery.Validate(delivery); validationErr != nil {
		return epwadelivery.Delivery{}, errors.Join(recordErr, validationErr)
	}
	return delivery, recordErr
}

func artifactHasBlockedAssets(manifest evidenceartifact.Manifest) bool {
	for _, asset := range manifest.Assets {
		if asset.RedactionState == evidenceartifact.RedactionBlocked {
			return true
		}
	}
	return false
}

func epwaDeliveryRoot(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Storage.DataDir) != "" {
		return cfg.Storage.DataDir
	}
	return screenshotStoreDir()
}

func writeSessionSnapshot(w http.ResponseWriter, req *http.Request, cfg *config.Config, sess *vision.Session, snap *vision.SnapResult, successStatus int, extra map[string]any) {
	delivery, err := publishSessionSnapshotEPWA(req, cfg, sess, snap, epwadelivery.ProducerSessionVisual)
	if err != nil {
		writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, "", "", "reconcile:session-snapshot-epwa-publication")
		return
	}
	response := map[string]any{
		"schema": "uiai.session_visual_result.v2", "width": snap.Width, "height": snap.Height,
		"format": snap.Format, "size": snap.Size, "url": snap.URL, "title": snap.Title,
		"duration_ms": snap.Duration, "artifact_ref": delivery.Artifact.ArtifactRef,
		"delivery_state": delivery.State, "epwa_delivery": delivery,
	}
	if delivery.State == epwadelivery.StateReady {
		response["artifact_url"] = delivery.EPWA.RecordURL
		response["portable_url"] = delivery.EPWA.PortableURL
	}
	if snap.DOM != nil {
		response["dom_posture"] = "withheld_pending_epwa_projection"
	}
	for key, value := range extra {
		if key != "screenshot" && key != "artifact_path" && key != "artifact_url" && key != "portable_url" && key != "epwa_delivery" {
			response[key] = value
		}
	}
	if delivery.State != epwadelivery.StateReady {
		successStatus = http.StatusAccepted
	}
	writeJSON(w, successStatus, response)
}

func publishSessionSnapshotEPWA(req *http.Request, cfg *config.Config, sess *vision.Session, snap *vision.SnapResult, producers ...epwadelivery.ProducerID) (epwadelivery.Delivery, error) {
	data, err := base64.StdEncoding.DecodeString(snap.Screenshot)
	if err != nil {
		return epwadelivery.Delivery{}, fmt.Errorf("decode screenshot: %w", err)
	}
	return publishScreenshotEPWA(req, cfg, evidenceshare.Input{
		Screenshot: data, Format: snap.Format, Width: snap.Width, Height: snap.Height,
		SourceURL: snap.URL, CapturedAt: time.Now().UTC(), DurationMS: snap.Duration,
		Scope: evidenceScopeFromSession(sess),
	}, producers...)
}

func evidenceScopeFromSession(sess *vision.Session) evidenceshare.Scope {
	if sess == nil {
		return evidenceshare.Scope{}
	}
	return evidenceScopeFromFocusa(sess.FocusaScope)
}

func evidenceScopeFromRequest(req *http.Request) evidenceshare.Scope {
	if req == nil {
		return evidenceshare.Scope{}
	}
	scope := evidenceshare.Scope{
		ProjectRef: req.Header.Get("X-UIAI-Project-Ref"), WorkstreamRef: req.Header.Get("X-UIAI-Workstream-Ref"),
		WorksetRef: req.Header.Get("X-UIAI-Workset-Ref"), CallGraphRef: req.Header.Get("X-UIAI-CallGraph-Ref"),
		WorkpointRef: req.Header.Get("X-UIAI-Workpoint-Ref"), WorkItemRef: req.Header.Get("X-UIAI-Work-Item-Ref"),
		ContinuityRef: req.Header.Get("X-UIAI-Continuity-Ref"),
	}
	if raw := strings.TrimSpace(req.Header.Get("X-UIAI-Work-Items")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &scope.WorkItems)
	}
	return scope
}

func publishLegacyVisualEPWA(req *http.Request, cfg *config.Config, pixels []byte, format string, width, height int, sourceURL string, producers ...epwadelivery.ProducerID) (epwadelivery.Delivery, error) {
	if width <= 0 || height <= 0 {
		dimensions, _, err := image.DecodeConfig(bytes.NewReader(pixels))
		if err != nil {
			return epwadelivery.Delivery{}, err
		}
		width, height = dimensions.Width, dimensions.Height
	}
	return publishScreenshotEPWA(req, cfg, evidenceshare.Input{
		Screenshot: pixels, Format: format, Width: width, Height: height,
		SourceURL: sourceURL, CapturedAt: time.Now().UTC(), Scope: evidenceScopeFromRequest(req),
	}, producers...)
}

func writeLegacyVisualEPWA(w http.ResponseWriter, req *http.Request, cfg *config.Config, pixels []byte, format string, width, height int, sourceURL string, extra map[string]any, producers ...epwadelivery.ProducerID) {
	delivery, err := publishLegacyVisualEPWA(req, cfg, pixels, format, width, height, sourceURL, producers...)
	if err != nil {
		writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, screenshotEvidenceRef(pixels), "", "reconcile:legacy-visual-epwa-publication")
		return
	}
	status := http.StatusAccepted
	if delivery.State == epwadelivery.StateReady {
		status = http.StatusCreated
	}
	response := map[string]any{
		"schema": "uiai.visual_artifact_result.v2", "artifact_ref": delivery.Artifact.ArtifactRef,
		"delivery_state": delivery.State, "epwa_delivery": delivery,
		"inline_posture": "withheld_by_mandatory_epwa_delivery",
	}
	for key, value := range extra {
		if key != "screenshot" && key != "imageBase64" && key != "artifact_path" {
			response[key] = value
		}
	}
	if delivery.State == epwadelivery.StateReady {
		response["artifact_url"] = delivery.EPWA.RecordURL
		response["portable_url"] = delivery.EPWA.PortableURL
	}
	writeJSON(w, status, response)
}

func writeJSONArtifactEPWA(w http.ResponseWriter, req *http.Request, cfg *config.Config, scope evidenceshare.Scope, sourceRef, title, kind string, payload any, successStatus int, childArtifactRefs ...string) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeEPWAPublishError(w, http.StatusServiceUnavailable, "artifact_serialization_failed", err, "", "", "reconcile:json-artifact-serialization")
		return
	}
	body = append(body, '\n')
	delivery, err := publishGenericEPWA(req, cfg, evidenceshare.GenericInput{
		Title: title, Kind: kind, MediaType: "application/json", Extension: "json", Payload: body,
		SourceRef: sourceRef, CapturedAt: time.Now().UTC(), Scope: scope, ChildArtifactRefs: childArtifactRefs,
	})
	if err != nil {
		writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, "", "", "reconcile:json-artifact-epwa-publication")
		return
	}
	status := http.StatusAccepted
	response := map[string]any{
		"schema": "uiai.artifact_result.v2", "artifact_ref": delivery.Artifact.ArtifactRef,
		"delivery_state": delivery.State, "epwa_delivery": delivery, "artifact_payload_posture": "withheld_until_epwa_ready",
	}
	if delivery.State == epwadelivery.StateReady {
		status = successStatus
		response["artifact_payload_posture"] = "included_with_epwa_delivery"
		var object map[string]any
		if json.Unmarshal(body, &object) == nil && object != nil {
			for key, value := range object {
				if key == "schema" {
					response["artifact_schema"] = value
					continue
				}
				if _, reserved := response[key]; !reserved {
					response[key] = value
				}
			}
		} else {
			response["artifact"] = payload
		}
		response["artifact_url"] = delivery.EPWA.RecordURL
		response["portable_url"] = delivery.EPWA.PortableURL
	}
	writeJSON(w, status, response)
}

// PublishGenericArtifactEPWA binds a nonvisual artifact to the shared delivery contract.
// It is exported for server-owned packages that cannot import routes without a cycle.
func PublishGenericArtifactEPWA(req *http.Request, cfg *config.Config, artifactRef, title, kind, mediaType, extension string, payload []byte) (epwadelivery.Delivery, error) {
	return publishGenericEPWA(req, cfg, evidenceshare.GenericInput{
		ArtifactRef: artifactRef, Revision: 1, Title: title, Kind: kind, MediaType: mediaType,
		Extension: extension, Payload: payload, CapturedAt: time.Now().UTC(), Scope: evidenceScopeFromRequest(req),
	})
}

func writeBinaryArtifactEPWA(w http.ResponseWriter, req *http.Request, cfg *config.Config, artifactRef, title, mediaType, extension string, payload []byte, successStatus int) {
	delivery, err := PublishGenericArtifactEPWA(req, cfg, artifactRef, title, "runtime_binary", mediaType, extension, payload)
	if err != nil {
		writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, screenshotEvidenceRef(payload), "", "reconcile:binary-artifact-epwa-publication")
		return
	}
	status := http.StatusAccepted
	response := map[string]any{
		"schema": "uiai.binary_artifact_result.v2", "artifact_ref": delivery.Artifact.ArtifactRef,
		"delivery_state": delivery.State, "epwa_delivery": delivery, "raw_output_posture": "withheld_by_mandatory_epwa_delivery",
	}
	if delivery.State == epwadelivery.StateReady {
		status = successStatus
		response["artifact_url"] = delivery.EPWA.RecordURL
		response["portable_url"] = delivery.EPWA.PortableURL
	}
	writeJSON(w, status, response)
}

func evidenceScopeFromFocusa(scope *vision.FocusaScope) evidenceshare.Scope {
	if scope == nil {
		return evidenceshare.Scope{}
	}
	return evidenceshare.Scope{
		ProjectRef: scope.ProjectRef, WorkstreamRef: scope.WorkstreamRef, WorksetRef: scope.WorksetRef,
		CallGraphRef: scope.CallGraphRef, WorkpointRef: scope.WorkpointID, WorkItemRef: scope.WorkItemRef,
		WorkItems: scope.WorkItems, ContinuityRef: scope.ContinuityID,
	}
}

func mountEPWADelivery(r chi.Router, cfg *config.Config) {
	r.Get("/delivery/{sha256}", func(w http.ResponseWriter, req *http.Request) {
		deliveryID, ok := deliveryIDFromParam(chi.URLParam(req, "sha256"))
		if !ok {
			http.NotFound(w, req)
			return
		}
		delivery, err := epwadelivery.Load(epwaDeliveryRoot(cfg), deliveryID)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		writeJSON(w, http.StatusOK, delivery)
	})
	r.Post("/delivery/{sha256}/reconcile", func(w http.ResponseWriter, req *http.Request) {
		deliveryID, ok := deliveryIDFromParam(chi.URLParam(req, "sha256"))
		if !ok {
			http.NotFound(w, req)
			return
		}
		current, err := epwadelivery.Load(epwaDeliveryRoot(cfg), deliveryID)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		updated, err := reconcileEPWADelivery(req, cfg, current)
		if err != nil {
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_reconciliation_failed", err, current.Artifact.ArtifactRef, current.Artifact.OutputSHA256, current.RecoveryRef)
			return
		}
		status := http.StatusAccepted
		if updated.State == epwadelivery.StateReady {
			status = http.StatusOK
		} else if updated.State == epwadelivery.StateCorrupt {
			status = http.StatusConflict
		}
		writeJSON(w, status, updated)
	})
}

func deliveryIDFromParam(value string) (string, bool) {
	if !validShareID(value) {
		return "", false
	}
	return "uiai-epwa-delivery:sha256:" + value, true
}

func reconcileEPWADelivery(req *http.Request, cfg *config.Config, current epwadelivery.Delivery) (epwadelivery.Delivery, error) {
	if current.Scope.Posture != epwadelivery.ScopeComplete || current.EPWA.Access != epwadelivery.AccessPublicSafe {
		return current, nil
	}
	next := current
	next.ObservedAt = time.Now().UTC()
	directory := filepath.Join(evidenceShareDir(cfg), current.EPWA.PackageID)
	if _, err := loadPublicPackage(directory, current.EPWA.PackageID); err != nil {
		next.State, next.EPWA.RecordURL, next.EPWA.PortableURL = epwadelivery.StateCorrupt, "", ""
		next.RecoveryRef = "reconcile:epwa-package-corrupt"
		return epwadelivery.Record(epwaDeliveryRoot(cfg), next)
	}
	packageSHA, err := evidenceshare.EnsurePortableArchive(evidenceShareDir(cfg), current.EPWA.PackageID)
	if err != nil || packageSHA != current.EPWA.PackageSHA256 {
		next.State, next.EPWA.RecordURL, next.EPWA.PortableURL = epwadelivery.StateCorrupt, "", ""
		next.RecoveryRef = "reconcile:epwa-package-corrupt"
		return epwadelivery.Record(epwaDeliveryRoot(cfg), next)
	}
	if current.State == epwadelivery.StateReady {
		return current, nil
	}
	base, err := canonicalEPWABase(req)
	if err != nil {
		return current, nil
	}
	next.State, next.RecoveryRef = epwadelivery.StateReady, ""
	next.EPWA.RecordURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + current.EPWA.PackageID + "/"}).String()
	next.EPWA.PortableURL = base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + current.EPWA.PackageID + "/portable.zip"}).String()
	return epwadelivery.Record(epwaDeliveryRoot(cfg), next)
}

func canonicalEPWABase(req *http.Request) (*url.URL, error) {
	raw := strings.TrimSpace(os.Getenv("UIAI_EPWA_PUBLIC_BASE_URL"))
	if raw == "" && req != nil && req.URL != nil && req.URL.Scheme == "https" && req.URL.Host != "" {
		raw = "https://" + req.URL.Host + "/"
	}
	base, err := url.Parse(raw)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, ErrEPWAHTTPSUnavailable
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	return base, nil
}

func writeEPWAPublishError(w http.ResponseWriter, status int, code string, err error, artifactRef, artifactSHA, recoveryRef string) {
	writeJSON(w, status, epwaPublishError{
		Schema: "uiai.epwa_delivery_error.v1", State: epwadelivery.StatePendingReconcile,
		ArtifactRef: artifactRef, ArtifactSHA256: artifactSHA, RecoveryRef: recoveryRef,
		Error: epwaPublishErrorDetail{Code: code, Message: err.Error(), Retryable: true},
	})
}
