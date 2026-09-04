package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/storage"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func screenshotEvidenceRef(data []byte) string {
	sum := sha256.Sum256(data)
	return "uiai-screenshot:sha256:" + hex.EncodeToString(sum[:])[:16]
}

func routeFocusaScopeStatus(scope *vision.FocusaScope) string {
	if scope == nil {
		return string(focusapacket.ScopeMissing)
	}
	if scope.ProjectRoot != "" && scope.ContinuityID != "" {
		return string(focusapacket.ScopePresent)
	}
	return string(focusapacket.ScopePartial)
}

func screenshotFocusaMetadata(targetURL, artifactRef, format string, bytes int, scope *vision.FocusaScope) map[string]any {
	summary := "UIAI screenshot captured"
	if bytes > 0 {
		summary += " bytes=" + strconv.Itoa(bytes)
	}
	evidence := map[string]any{
		"target_ref":          "browser:" + focusapacket.SanitizeURL(targetURL),
		"result":              summary,
		"summary":             summary,
		"evidence_ref":        artifactRef,
		"artifact_ref":        artifactRef,
		"preferred_tool":      "focusa_evidence_capture",
		"next_tools":          []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		"focusa_scope_status": routeFocusaScopeStatus(scope),
		"mime_type":           "image/" + format,
		"bytes":               bytes,
	}
	if scope != nil {
		evidence["focusa_scope"] = scope
	}
	return evidence
}

func settingMap(values map[string]any, group string) map[string]any {
	m, _ := values[group].(map[string]any)
	return m
}
func settingString(values map[string]any, group, key string) string {
	v, _ := settingMap(values, group)[key].(string)
	return v
}
func settingInt(values map[string]any, group, key string) int {
	switch v := settingMap(values, group)[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
func scopeProject(scope *vision.FocusaScope) string {
	if scope == nil {
		return ""
	}
	return scope.ProjectRoot
}
func scopeWorkstream(scope *vision.FocusaScope) string {
	if scope == nil {
		return ""
	}
	return scope.DerivedWorkstreamKey()
}

func MountScreenshotReal(r chi.Router, cfg *config.Config, pool vision.PoolSource, usage *storage.UsageStore) {
	settingsStore, _ := evidenceshare.NewSettingsStore(cfg.Storage.DataDir)
	if settingsStore != nil {
		mountEvidenceShareSettings(r, settingsStore)
	}
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL           string               `json:"url"`
			Width         int                  `json:"width"`
			Height        int                  `json:"height"`
			FullPage      bool                 `json:"fullPage"`
			Format        string               `json:"format"`
			Quality       int                  `json:"quality"`
			WaitFor       string               `json:"waitFor"`
			Delay         int                  `json:"delay"`
			Cookies       string               `json:"cookies"` // "name=value; name2=value2"
			Timeout       int                  `json:"timeout"` // overall timeout in seconds (default: 30)
			NoCache       bool                 `json:"nocache"` // skip cache, always take fresh screenshot
			FocusaScope   *vision.FocusaScope  `json:"focusa_scope"`
			EvidenceScope *evidenceshare.Scope `json:"evidence_scope"`
			Inline        *bool                `json:"inline"` // C-010-09: default false → artifact_ref only
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		if pool == nil {
			writeJSON(w, 503, map[string]string{"error": "vision pool not initialized"})
			return
		}
		if settingsStore != nil {
			effective := settingsStore.Effective(evidenceshare.SettingsScope{ProjectRef: scopeProject(body.FocusaScope), WorkstreamRef: scopeWorkstream(body.FocusaScope)})
			if body.Format == "" {
				body.Format = settingString(effective.Values, "image", "format")
			}
			if body.Quality == 0 {
				body.Quality = settingInt(effective.Values, "image", "quality")
			}
		}

		result, err := pool.Screenshot(vision.ScreenshotOpts{
			URL:      body.URL,
			Width:    body.Width,
			Height:   body.Height,
			FullPage: body.FullPage,
			Format:   body.Format,
			Quality:  body.Quality,
			WaitFor:  body.WaitFor,
			Delay:    body.Delay,
			Cookies:  body.Cookies,
			Timeout:  body.Timeout,
			NoCache:  body.NoCache,
		})
		if err != nil {
			if errors.Is(err, vision.ErrQueueFull) {
				w.Header().Set("Retry-After", "10")
				writeJSON(w, 429, map[string]string{"error": "too many requests — screenshot queue full, retry after 10s"})
				return
			}
			if errors.Is(err, vision.ErrQueueTimeout) {
				writeJSON(w, 408, map[string]string{"error": "request timed out waiting in screenshot queue"})
				return
			}
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		// Record usage
		if usage != nil {
			usage.Record(storage.UsageRecord{
				Type:      "screenshot",
				Status:    "success",
				CostUSD:   0.005, // flat per-screenshot cost
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			})
		}

		artifactRef := screenshotEvidenceRef(result.Data)
		focusaEvidence := screenshotFocusaMetadata(body.URL, artifactRef, result.Format, len(result.Data), body.FocusaScope)
		if body.FocusaScope != nil {
			focusaEvidence["focusa_scope"] = body.FocusaScope
		}
		capturedAt := time.Now().UTC()
		shareInput := evidenceshare.Input{Screenshot: result.Data, Format: result.Format, Width: result.Width, Height: result.Height, SourceURL: body.URL, CapturedAt: capturedAt, DurationMS: result.Duration.Milliseconds()}
		if body.EvidenceScope != nil {
			shareInput.Scope = *body.EvidenceScope
		} else if body.FocusaScope != nil {
			shareInput.Scope = evidenceshare.Scope{
				ProjectRef: body.FocusaScope.ProjectRef, WorkstreamRef: body.FocusaScope.WorkstreamRef,
				WorksetRef: body.FocusaScope.WorksetRef, CallGraphRef: body.FocusaScope.CallGraphRef,
				WorkpointRef: body.FocusaScope.WorkpointID, WorkItemRef: body.FocusaScope.WorkItemRef,
				WorkItems: body.FocusaScope.WorkItems, ContinuityRef: body.FocusaScope.ContinuityID,
			}
		}
		delivery, err := publishScreenshotEPWA(req, cfg, shareInput)
		if err != nil {
			digest := sha256.Sum256(result.Data)
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, artifactRef, hex.EncodeToString(digest[:]), "reconcile:screenshot-epwa-publication")
			return
		}
		response := map[string]any{
			"schema": "uiai.screenshot_result.v2", "width": result.Width, "height": result.Height,
			"format": result.Format, "size": len(result.Data), "duration_ms": result.Duration.Milliseconds(),
			"artifact_ref": delivery.Artifact.ArtifactRef, "delivery_state": delivery.State,
			"epwa_delivery": delivery, "focusa_evidence": focusaEvidence, "focusa": focusaEvidence,
		}
		if delivery.State == epwadelivery.StateReady {
			response["artifact_url"] = delivery.EPWA.RecordURL
			response["portable_url"] = delivery.EPWA.PortableURL
		}
		if body.Inline != nil && *body.Inline {
			response["inline_posture"] = "withheld_by_mandatory_epwa_delivery"
		}
		if result.DOMReport != "" {
			response["dom_report_posture"] = "withheld_pending_epwa_projection"
		}
		status := http.StatusCreated
		if delivery.State != epwadelivery.StateReady {
			status = http.StatusAccepted
		}
		writeJSON(w, status, response)
	})

	mountScreenshotArtifact(r) // C-010-09 retrieval surface
	mountEvidenceShare(r, cfg)
	mountEPWADelivery(r, cfg)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		if pool == nil {
			writeJSON(w, 503, map[string]string{"status": "unavailable", "error": "vision pool not initialized"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"status": "healthy",
			"pool":   pool.Stats(),
		})
	})
}

// screenshotArtifactDir resolves the durable store for C-010-09 artifacts.
func screenshotStoreDir() string {
	dir := os.Getenv("UIAI_SCREENSHOT_DIR")
	if dir == "" {
		dir = "/home/wpuiai/uiai-engine/data/screenshots"
	}
	return dir
}

// mountScreenshotArtifact preserves the legacy route only as an explicit fail-closed tombstone.
func mountScreenshotArtifact(r chi.Router) {
	r.Get("/artifact/{sha}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusGone, map[string]any{
			"schema": "uiai.epwa_delivery_error.v1", "code": "legacy_raw_artifact_removed",
			"artifact_ref": chi.URLParam(req, "sha"),
			"message":      "raw screenshots are available only through an EPWA viewer and portable package",
		})
	})
}

func evidenceShareDir(cfg *config.Config) string {
	if dir := os.Getenv("UIAI_EVIDENCE_SHARE_DIR"); dir != "" {
		return dir
	}
	if cfg != nil && cfg.Storage.DataDir != "" {
		return filepath.Join(cfg.Storage.DataDir, "evidence-share")
	}
	return filepath.Join(screenshotStoreDir(), "evidence-share")
}

func requestArtifactURL(req *http.Request, artifactPath string) string {
	base, err := canonicalEPWABase(req)
	if err != nil {
		return ""
	}
	return base.ResolveReference(&url.URL{Path: strings.TrimPrefix(artifactPath, "/")}).String()
}

const evidenceShareCSP = "default-src 'self'; base-uri 'none'; object-src 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'none'; script-src 'self'; style-src 'self'; img-src 'self'; media-src 'self'; connect-src 'self'; worker-src 'self'; manifest-src 'self'"

type publicPackageDescriptor struct {
	Schema string
	Assets map[string]evidenceartifact.Asset
}

func loadPublicPackage(directory, id string) (publicPackageDescriptor, error) {
	body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
	if err != nil {
		return publicPackageDescriptor{}, err
	}
	var envelope struct {
		Schema string `json:"schema"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return publicPackageDescriptor{}, errors.New("invalid EPWA package manifest")
	}
	switch envelope.Schema {
	case evidenceshare.Schema:
		var manifest evidenceshare.Manifest
		if json.Unmarshal(body, &manifest) != nil || manifest.ArtifactSHA256 != id || manifest.Availability != "ready" || manifest.Access != "public_safe_read_only" || manifest.ProjectionRef == "" {
			return publicPackageDescriptor{}, errors.New("screenshot EPWA package is not public ready")
		}
		var projection evidencepwa.Projection
		projectionBody, err := os.ReadFile(filepath.Join(directory, strings.TrimPrefix(manifest.ProjectionRef, "./")))
		if err != nil || json.Unmarshal(projectionBody, &projection) != nil || evidencepwa.ValidateProjection(projection) != nil || projection.Access != evidencepwa.AccessPublicSafe || projection.Availability != evidencepwa.AvailabilityReady {
			return publicPackageDescriptor{}, errors.New("screenshot EPWA projection is not public ready")
		}
		return publicPackageDescriptor{Schema: envelope.Schema}, nil
	case evidenceartifact.SchemaManifestV1:
		var manifest evidenceartifact.Manifest
		if json.Unmarshal(body, &manifest) != nil || evidenceartifact.Validate(manifest) != nil || evidenceartifact.VerifyManifestSHA256(manifest) != nil || manifest.Policy.AccessClass != evidenceartifact.AccessPublicSafe || manifest.Policy.RedactionState != evidenceartifact.RedactionPublicSafe || strings.TrimSpace(manifest.Scope.ContinuityRef) == "" {
			return publicPackageDescriptor{}, errors.New("artifact EPWA package is not public ready")
		}
		wantID, err := evidenceshare.ArtifactPackageID(manifest)
		if err != nil || wantID != id {
			return publicPackageDescriptor{}, errors.New("artifact EPWA package identity mismatch")
		}
		assetByPath := make(map[string]evidenceartifact.Asset, len(manifest.Assets))
		for _, asset := range manifest.Assets {
			if asset.RedactionState == evidenceartifact.RedactionBlocked {
				return publicPackageDescriptor{}, errors.New("artifact EPWA package contains blocked assets")
			}
			assetByPath[asset.Path] = asset
		}
		var projection evidencepwa.Projection
		projectionBody, err := os.ReadFile(filepath.Join(directory, "projection.json"))
		if err != nil || json.Unmarshal(projectionBody, &projection) != nil || evidencepwa.ValidateProjection(projection) != nil || projection.Access != evidencepwa.AccessPublicSafe || projection.Availability != evidencepwa.AvailabilityReady || projection.Artifact.ManifestSHA256 != manifest.Integrity.ManifestSHA256 {
			return publicPackageDescriptor{}, errors.New("artifact EPWA projection is not public ready")
		}
		return publicPackageDescriptor{Schema: envelope.Schema, Assets: assetByPath}, nil
	case evidenceshare.GenericArtifactSchema:
		manifest, _, err := evidenceshare.ValidateGenericPackage(directory, id)
		if err != nil || manifest.Access != "public_safe_read_only" || manifest.Availability != "ready" {
			return publicPackageDescriptor{}, errors.New("generic EPWA package is not public ready")
		}
		assetPath := strings.TrimPrefix(manifest.AssetRef, "./")
		return publicPackageDescriptor{Schema: envelope.Schema, Assets: map[string]evidenceartifact.Asset{
			assetPath: {AssetID: "generic-payload", Path: assetPath, SHA256: manifest.AssetSHA256, ByteSize: int64(manifest.Bytes), MediaType: manifest.MediaType},
		}}, nil
	default:
		return publicPackageDescriptor{}, errors.New("unsupported EPWA package manifest")
	}
}

func safeShareAssetPath(value string) bool {
	return value != "" && !strings.ContainsAny(value, "\\\r\n\x00") && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func mountEvidenceShare(r chi.Router, cfg *config.Config) {
	r.Get("/share", func(w http.ResponseWriter, req *http.Request) {
		if _, err := canonicalEPWABase(req); err != nil {
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_https_unavailable", err, "", "", "configure:UIAI_EPWA_PUBLIC_BASE_URL")
			return
		}
		entries, err := os.ReadDir(evidenceShareDir(cfg))
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"packets": []any{}})
			return
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })
		packets := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() || !validShareID(entry.Name()) {
				continue
			}
			directory := filepath.Join(evidenceShareDir(cfg), entry.Name())
			descriptor, err := loadPublicPackage(directory, entry.Name())
			if err != nil {
				continue
			}
			body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
			if err != nil {
				continue
			}
			artifactPath := "/api/screenshot/share/" + entry.Name() + "/"
			packet := map[string]any{"packet_id": entry.Name(), "artifact_url": requestArtifactURL(req, artifactPath), "portable_url": requestArtifactURL(req, artifactPath+"portable.zip"), "availability": "ready"}
			switch descriptor.Schema {
			case evidenceshare.Schema:
				var manifest evidenceshare.Manifest
				if json.Unmarshal(body, &manifest) != nil {
					continue
				}
				packet["descriptor"], packet["artifact_ref"], packet["captured_at"] = "Screenshot evidence · "+manifest.CapturedAt.Format(time.RFC3339), manifest.ArtifactRef, manifest.CapturedAt
				packet["source_url"], packet["workpoint_ref"], packet["continuity_ref"] = manifest.SourceURL, manifest.Scope.WorkpointRef, manifest.Scope.ContinuityRef
			case evidenceartifact.SchemaManifestV1:
				var manifest evidenceartifact.Manifest
				if json.Unmarshal(body, &manifest) != nil {
					continue
				}
				packet["descriptor"], packet["artifact_ref"], packet["captured_at"] = manifest.Title, manifest.ArtifactID, manifest.CapturedAt
				packet["workpoint_ref"] = manifest.Scope.Workpoint.WorkpointRef
				packet["continuity_ref"] = manifest.Scope.ContinuityRef
			case evidenceshare.GenericArtifactSchema:
				var manifest evidenceshare.GenericManifest
				if json.Unmarshal(body, &manifest) != nil {
					continue
				}
				packet["descriptor"], packet["artifact_ref"], packet["captured_at"] = manifest.Title, manifest.ArtifactRef, manifest.CapturedAt
				packet["workpoint_ref"], packet["continuity_ref"] = manifest.Scope.WorkpointRef, manifest.Scope.ContinuityRef
			}
			packets = append(packets, packet)
			if len(packets) == 100 {
				break
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"packets": packets, "count": len(packets)})
	})
	r.Get("/share/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if !validShareID(id) {
			http.NotFound(w, req)
			return
		}
		directory := filepath.Join(evidenceShareDir(cfg), id)
		if _, err := loadPublicPackage(directory, id); err != nil {
			http.NotFound(w, req)
			return
		}
		body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
		if err != nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(body)
	})
	r.Get("/share/{id}/portable.zip", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if !validShareID(id) {
			http.NotFound(w, req)
			return
		}
		directory := filepath.Join(evidenceShareDir(cfg), id)
		if _, err := loadPublicPackage(directory, id); err != nil {
			http.NotFound(w, req)
			return
		}
		digest, err := evidenceshare.EnsurePortableArchive(evidenceShareDir(cfg), id)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		archivePath := filepath.Join(evidenceShareDir(cfg), id+".zip")
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.epwa.zip"`)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", `"sha256:`+digest+`"`)
		http.ServeFile(w, req, archivePath)
	})
	r.Get("/share/{id}/verify", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		issues := []string{}
		if !validShareID(id) {
			issues = append(issues, "invalid_packet_id")
		} else {
			directory := filepath.Join(evidenceShareDir(cfg), id)
			descriptor, err := loadPublicPackage(directory, id)
			if err != nil {
				issues = append(issues, "package_not_public_ready")
			} else {
				switch descriptor.Schema {
				case evidenceshare.Schema:
					var manifest evidenceshare.Manifest
					body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
					if err != nil || json.Unmarshal(body, &manifest) != nil {
						issues = append(issues, "manifest_unavailable")
					} else if shot, err := os.ReadFile(filepath.Join(directory, strings.TrimPrefix(manifest.ScreenshotRef, "./"))); err != nil {
						issues = append(issues, "screenshot_unavailable")
					} else if sum := sha256.Sum256(shot); hex.EncodeToString(sum[:]) != manifest.ScreenshotSHA256 {
						issues = append(issues, "screenshot_digest_mismatch")
					}
				case evidenceartifact.SchemaManifestV1, evidenceshare.GenericArtifactSchema:
					for assetPath, asset := range descriptor.Assets {
						data, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(assetPath)))
						if err != nil {
							issues = append(issues, "asset_unavailable:"+asset.AssetID)
							continue
						}
						sum := sha256.Sum256(data)
						if int64(len(data)) != asset.ByteSize || hex.EncodeToString(sum[:]) != asset.SHA256 {
							issues = append(issues, "asset_digest_mismatch:"+asset.AssetID)
						}
					}
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"packet_id": id, "descriptor": "EPWA evidence package", "valid": len(issues) == 0, "issues": issues})
	})
	r.Get("/share/{id}/*", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if !validShareID(id) {
			http.NotFound(w, req)
			return
		}
		directory := filepath.Join(evidenceShareDir(cfg), id)
		descriptor, err := loadPublicPackage(directory, id)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		name := chi.URLParam(req, "*")
		if name == "" {
			name = "index.html"
		}
		if !safeShareAssetPath(name) {
			http.NotFound(w, req)
			return
		}
		allowed := map[string]string{"index.html": "text/html; charset=utf-8", "styles.css": "text/css; charset=utf-8", "work-items.js": "application/javascript; charset=utf-8", "generic-record.js": "application/javascript; charset=utf-8", "pwa.js": "application/javascript; charset=utf-8", "app.js": "application/javascript; charset=utf-8", "manifest.webmanifest": "application/manifest+json", "icon.svg": "image/svg+xml", "sw.js": "application/javascript; charset=utf-8", "artifact.json": "application/json; charset=utf-8", "projection.json": "application/json; charset=utf-8", "inspection.json": "application/json; charset=utf-8", "screenshot.png": "image/png", "screenshot.jpg": "image/jpeg", "screenshot.webp": "image/webp"}
		mediaType, static := allowed[name]
		asset, artifactAsset := descriptor.Assets[name]
		if !static && !artifactAsset {
			http.NotFound(w, req)
			return
		}
		if artifactAsset {
			mediaType = asset.MediaType
			w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(name)+`"`)
			if strings.HasPrefix(strings.ToLower(mediaType), "text/html") || strings.HasPrefix(strings.ToLower(mediaType), "image/svg+xml") {
				w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
			}
		}
		data, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(name)))
		if err != nil {
			http.NotFound(w, req)
			return
		}
		if artifactAsset {
			sum := sha256.Sum256(data)
			if int64(len(data)) != asset.ByteSize || hex.EncodeToString(sum[:]) != asset.SHA256 {
				http.Error(w, "EPWA asset corrupt", http.StatusConflict)
				return
			}
		}
		w.Header().Set("Content-Type", mediaType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if name == "index.html" {
			w.Header().Set("Content-Security-Policy", evidenceShareCSP)
		}
		switch {
		case name == "sw.js":
			w.Header().Set("Cache-Control", "no-cache")
		case name == "artifact.json" || name == "projection.json" || name == "inspection.json" || strings.HasPrefix(name, "screenshot.") || artifactAsset:
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		_, _ = w.Write(data)
	})
}

func validShareID(id string) bool {
	if len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}
