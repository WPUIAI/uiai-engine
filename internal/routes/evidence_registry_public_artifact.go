package routes

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/go-chi/chi/v5"
)

const publicArtifactDetailSchemaV1 = "uiai.public_evidence_artifact_detail.v1"

type publicArtifactAsset struct {
	AssetID           string                             `json:"asset_id"`
	Kind              string                             `json:"kind"`
	MediaType         string                             `json:"media_type"`
	SHA256            string                             `json:"sha256"`
	ByteSize          int64                              `json:"byte_size"`
	Width             int                                `json:"width,omitempty"`
	Height            int                                `json:"height,omitempty"`
	DurationMS        int64                              `json:"duration_ms,omitempty"`
	CapturedAt        string                             `json:"captured_at,omitempty"`
	SourceRef         string                             `json:"source_ref"`
	ClaimRefs         []string                           `json:"claim_refs"`
	VerificationClass evidenceartifact.VerificationClass `json:"verification_class"`
	RedactionState    evidenceartifact.RedactionState    `json:"redaction_state"`
	AltText           string                             `json:"alt_text,omitempty"`
	TranscriptRef     string                             `json:"transcript_ref,omitempty"`
	Href              string                             `json:"href"`
}

func mountPublicArtifactReads(r chi.Router, artifacts *evidenceartifact.Store, isAllowed func(string) bool) {
	r.Get("/artifacts/{artifact_ref}/revisions/{revision}", func(w http.ResponseWriter, req *http.Request) {
		manifest, entry, ok := readPublicArtifact(req, artifacts, isAllowed)
		if !ok {
			writePublicArtifactNotFound(w)
			return
		}
		base := publicArtifactBasePath(manifest.ArtifactID, manifest.Revision)
		assets := make([]publicArtifactAsset, 0, len(manifest.Assets))
		for _, asset := range manifest.Assets {
			if asset.RedactionState != evidenceartifact.RedactionPublicSafe {
				continue
			}
			assets = append(assets, publicArtifactAsset{
				AssetID: asset.AssetID, Kind: asset.Kind, MediaType: asset.MediaType, SHA256: asset.SHA256,
				ByteSize: asset.ByteSize, Width: asset.Width, Height: asset.Height, DurationMS: asset.DurationMS,
				CapturedAt: asset.CapturedAt, SourceRef: asset.SourceRef, ClaimRefs: asset.ClaimRefs,
				VerificationClass: asset.VerificationClass, RedactionState: asset.RedactionState,
				AltText: asset.AltText, TranscriptRef: asset.TranscriptRef,
				Href: base + "/assets/" + asset.SHA256,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"schema": publicArtifactDetailSchemaV1, "interaction": "read_only",
			"artifact_id": manifest.ArtifactID, "revision": manifest.Revision,
			"manifest_sha256": entry.ManifestSHA256, "manifest": manifest, "assets": assets,
			"detail_path": base, "manifest_path": base + "/manifest",
		})
	})

	r.Get("/artifacts/{artifact_ref}/revisions/{revision}/manifest", func(w http.ResponseWriter, req *http.Request) {
		manifest, entry, ok := readPublicArtifact(req, artifacts, isAllowed)
		if !ok {
			writePublicArtifactNotFound(w)
			return
		}
		w.Header().Set("ETag", `"sha256:`+entry.ManifestSHA256+`"`)
		writeJSON(w, http.StatusOK, manifest)
	})

	serveAsset := func(w http.ResponseWriter, req *http.Request) {
		manifest, _, ok := readPublicArtifact(req, artifacts, isAllowed)
		if !ok {
			writePublicArtifactNotFound(w)
			return
		}
		assetSHA256 := chi.URLParam(req, "asset_sha256")
		var declared *evidenceartifact.Asset
		for i := range manifest.Assets {
			if manifest.Assets[i].SHA256 == assetSHA256 && manifest.Assets[i].RedactionState == evidenceartifact.RedactionPublicSafe {
				declared = &manifest.Assets[i]
				break
			}
		}
		if declared == nil {
			writePublicArtifactNotFound(w)
			return
		}
		file, record, err := artifacts.OpenAsset(declared.SHA256)
		if err != nil || record.SHA256 != declared.SHA256 || record.ByteSize != declared.ByteSize || record.MediaType != declared.MediaType {
			writePublicArtifactNotFound(w)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", record.MediaType)
		w.Header().Set("Content-Length", strconv.FormatInt(record.ByteSize, 10))
		w.Header().Set("Cache-Control", "public, immutable, max-age=31536000")
		w.Header().Set("ETag", `"sha256:`+record.SHA256+`"`)
		if req.Method != http.MethodHead {
			_, _ = io.Copy(w, file)
		}
	}
	r.Get("/artifacts/{artifact_ref}/revisions/{revision}/assets/{asset_sha256}", serveAsset)
	r.Head("/artifacts/{artifact_ref}/revisions/{revision}/assets/{asset_sha256}", serveAsset)
}

func readPublicArtifact(req *http.Request, artifacts *evidenceartifact.Store, isAllowed func(string) bool) (evidenceartifact.Manifest, evidenceartifact.Entry, bool) {
	if artifacts == nil {
		return evidenceartifact.Manifest{}, evidenceartifact.Entry{}, false
	}
	revision, err := strconv.ParseUint(chi.URLParam(req, "revision"), 10, 64)
	if err != nil || revision == 0 {
		return evidenceartifact.Manifest{}, evidenceartifact.Entry{}, false
	}
	manifest, entry, err := artifacts.GetManifest(chi.URLParam(req, "artifact_ref"), revision)
	if err != nil || !isAllowed(manifest.Scope.Project.ProjectRef) || manifest.Policy.AccessClass != evidenceartifact.AccessPublicSafe || manifest.Policy.RedactionState != evidenceartifact.RedactionPublicSafe {
		return evidenceartifact.Manifest{}, evidenceartifact.Entry{}, false
	}
	return manifest, entry, true
}

func publicArtifactBasePath(artifactID string, revision uint64) string {
	return "/api/evidence/registry/public/artifacts/" + url.PathEscape(artifactID) + "/revisions/" + strconv.FormatUint(revision, 10)
}

func writePublicArtifactNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"schema": "uiai.public_evidence_registry_error.v1", "error": map[string]string{"code": "not_found", "message": "public evidence artifact not found"}})
}
