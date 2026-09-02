package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const maxArchiveSourceBytes = 64 * 1024 * 1024

type ArchiveAssetSource struct {
	AssetRef string
	Path     string
	MIME     string
	Body     []byte
}

func RenderProjectionArchive(request DerivativeRequest, projection evidencepwa.Projection, sources []ArchiveAssetSource, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeArchive {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	if len(sources) != len(selection.assets) {
		return RenderedDerivative{}, ErrDerivativeSelectionIncomplete
	}
	selectedAssets := make(map[string]evidencepwa.Asset, len(selection.assets))
	for _, asset := range selection.assets {
		selectedAssets[asset.AssetID] = asset
	}
	type file struct {
		path string
		mime string
		body []byte
	}
	files := make([]file, 0, len(sources)+1)
	seenPaths := map[string]struct{}{"evidence/selection.json": {}}
	seenAssets := make(map[string]struct{}, len(sources))
	totalBytes := 0
	for _, source := range sources {
		asset, found := selectedAssets[source.AssetRef]
		if !found || !safeArchivePath(source.Path) || source.MIME != asset.MIME || len(source.Body) == 0 || uint64(len(source.Body)) != asset.Bytes {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		if _, duplicate := seenAssets[source.AssetRef]; duplicate {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		seenAssets[source.AssetRef] = struct{}{}
		if _, duplicate := seenPaths[source.Path]; duplicate {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		seenPaths[source.Path] = struct{}{}
		sum := sha256.Sum256(source.Body)
		if hex.EncodeToString(sum[:]) != asset.SHA256 {
			return RenderedDerivative{}, ErrProjectionMismatch
		}
		totalBytes += len(source.Body)
		if totalBytes > maxArchiveSourceBytes {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		files = append(files, file{path: source.Path, mime: source.MIME, body: append([]byte(nil), source.Body...)})
	}
	selectionBody, err := json.Marshal(struct {
		ProjectionRef    string                 `json:"projection_ref"`
		ProjectionSHA256 string                 `json:"projection_sha256"`
		ArtifactRef      string                 `json:"artifact_ref"`
		ArtifactSHA256   string                 `json:"artifact_sha256"`
		Claims           []evidencepwa.Claim    `json:"claims"`
		Assets           []evidencepwa.Asset    `json:"assets"`
		Citations        []evidencepwa.Citation `json:"citations"`
		OmissionRefs     []string               `json:"omission_refs,omitempty"`
	}{request.ProjectionRef, request.ProjectionSHA256, request.ArtifactRef, request.ArtifactSHA256, selection.claims, selection.assets, selection.citations, append([]string(nil), request.OmissionRefs...)})
	if err != nil {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selectionBody = append(selectionBody, '\n')
	files = append(files, file{path: "evidence/selection.json", mime: "application/json", body: selectionBody})
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entries := make([]ArchiveEntry, 0, len(files))
	for _, candidate := range files {
		header := &zip.FileHeader{Name: candidate.path, Method: zip.Store}
		header.SetMode(0o644)
		header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		destination, createErr := writer.CreateHeader(header)
		if createErr != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		if _, writeErr := destination.Write(candidate.body); writeErr != nil {
			return RenderedDerivative{}, ErrDerivativeUnsafeArchive
		}
		sum := sha256.Sum256(candidate.body)
		entries = append(entries, ArchiveEntry{Path: candidate.path, SHA256: hex.EncodeToString(sum[:]), MIME: candidate.mime, Bytes: uint64(len(candidate.body)), Link: false})
	}
	if err := writer.Close(); err != nil {
		return RenderedDerivative{}, ErrDerivativeUnsafeArchive
	}
	return buildRenderedDerivativeWithArchive(request, archive.Bytes(), "zip", "application/zip", renderer, matrix, licenses, receiptRef, createdAt, ArchiveSafe, entries)
}

func safeArchivePath(value string) bool {
	return relativeRef(value) && path.Clean(value) == value && !strings.HasPrefix(value, ".") && !strings.Contains(value, ":")
}
