package evidenceshare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

const (
	GenericArtifactSchema   = "uiai.epwa_generic_artifact.v1"
	GenericProjectionSchema = "uiai.epwa_generic_projection.v1"
	GenericTruthNotice      = "This EPWA record proves only the captured artifact bytes and declared scope. It does not establish review, verification, completion, provider closure, settlement, or legal admissibility."
	maxGenericBytes         = 64 << 20
)

type GenericInput struct {
	ArtifactRef       string
	Revision          uint64
	Title             string
	Kind              string
	MediaType         string
	Extension         string
	Payload           []byte
	SourceRef         string
	CapturedAt        time.Time
	Scope             Scope
	ParentArtifactRef string
	ChildArtifactRefs []string
}

type GenericManifest struct {
	Schema            string    `json:"schema"`
	ArtifactRef       string    `json:"artifact_ref"`
	Revision          uint64    `json:"revision"`
	Title             string    `json:"title"`
	Kind              string    `json:"kind"`
	MediaType         string    `json:"media_type"`
	AssetRef          string    `json:"asset_ref"`
	AssetSHA256       string    `json:"asset_sha256"`
	Bytes             int       `json:"bytes"`
	SourceRef         string    `json:"source_ref,omitempty"`
	CapturedAt        time.Time `json:"captured_at"`
	Availability      string    `json:"availability"`
	Access            string    `json:"access"`
	Interaction       string    `json:"interaction"`
	Scope             Scope     `json:"scope"`
	ProjectionRef     string    `json:"projection_ref,omitempty"`
	ParentArtifactRef string    `json:"parent_artifact_ref,omitempty"`
	ChildArtifactRefs []string  `json:"child_artifact_refs,omitempty"`
	TruthNotice       string    `json:"truth_notice"`
}

type GenericProjection struct {
	Schema            string    `json:"schema"`
	ProjectionID      string    `json:"projection_id"`
	ArtifactRef       string    `json:"artifact_ref"`
	ArtifactSHA256    string    `json:"artifact_sha256"`
	Revision          uint64    `json:"revision"`
	Kind              string    `json:"kind"`
	MediaType         string    `json:"media_type"`
	AssetRef          string    `json:"asset_ref"`
	Scope             Scope     `json:"scope"`
	Availability      string    `json:"availability"`
	Access            string    `json:"access"`
	ParentArtifactRef string    `json:"parent_artifact_ref,omitempty"`
	ChildArtifactRefs []string  `json:"child_artifact_refs,omitempty"`
	ProjectedAt       time.Time `json:"projected_at"`
	TruthNotice       string    `json:"truth_notice"`
}

func AssembleGeneric(root string, input GenericInput) (Result, error) {
	if strings.TrimSpace(root) == "" || len(input.Payload) == 0 || len(input.Payload) > maxGenericBytes || input.CapturedAt.IsZero() {
		return Result{}, ErrInvalidInput
	}
	extension, ok := normalizeGenericExtension(input.Extension)
	if !ok || !validGenericText(input.MediaType, 128) || !validGenericText(input.Kind, 80) || !validGenericText(input.Title, 256) {
		return Result{}, ErrInvalidInput
	}
	outputSHA := digestBytes(input.Payload)
	artifactRef := strings.TrimSpace(input.ArtifactRef)
	if artifactRef == "" {
		artifactRef = "uiai-artifact:sha256:" + outputSHA
	}
	if !validGenericText(artifactRef, 512) || strings.ContainsAny(artifactRef, "\r\n\x00") {
		return Result{}, ErrInvalidInput
	}
	revision := input.Revision
	if revision == 0 {
		revision = 1
	}
	availability := "blocked"
	projectionRef := ""
	if input.Scope.Complete() {
		availability = "ready"
		projectionRef = "./projection.json"
	}
	assetRef := "./payload." + extension
	manifest := GenericManifest{
		Schema: GenericArtifactSchema, ArtifactRef: artifactRef, Revision: revision, Title: strings.TrimSpace(input.Title),
		Kind: strings.TrimSpace(input.Kind), MediaType: strings.TrimSpace(input.MediaType), AssetRef: assetRef,
		AssetSHA256: outputSHA, Bytes: len(input.Payload), SourceRef: strings.TrimSpace(input.SourceRef), CapturedAt: input.CapturedAt.UTC(),
		Availability: availability, Access: "public_safe_read_only", Interaction: "read_only", Scope: input.Scope,
		ProjectionRef: projectionRef, ParentArtifactRef: strings.TrimSpace(input.ParentArtifactRef),
		ChildArtifactRefs: normalizedGenericRefs(input.ChildArtifactRefs), TruthNotice: GenericTruthNotice,
	}
	manifestBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, err
	}
	manifestBody = append(manifestBody, '\n')
	manifestSHA := digestBytes(manifestBody)
	writes := map[string][]byte{
		"artifact.json":                    manifestBody,
		strings.TrimPrefix(assetRef, "./"): append([]byte(nil), input.Payload...),
	}
	projectionSHA := ""
	projectionID := ""
	if input.Scope.Complete() {
		projectionSeed, err := json.Marshal(struct {
			ManifestSHA256 string
			Scope          Scope
			Parent         string
			Children       []string
		}{manifestSHA, input.Scope, manifest.ParentArtifactRef, manifest.ChildArtifactRefs})
		if err != nil {
			return Result{}, err
		}
		projectionID = "uiai-evidence-projection:sha256:" + digestBytes(projectionSeed)
		projection := GenericProjection{
			Schema: GenericProjectionSchema, ProjectionID: projectionID, ArtifactRef: artifactRef, ArtifactSHA256: outputSHA,
			Revision: revision, Kind: manifest.Kind, MediaType: manifest.MediaType, AssetRef: assetRef, Scope: input.Scope,
			Availability: availability, Access: manifest.Access, ParentArtifactRef: manifest.ParentArtifactRef,
			ChildArtifactRefs: manifest.ChildArtifactRefs, ProjectedAt: input.CapturedAt.UTC(), TruthNotice: GenericTruthNotice,
		}
		projectionBody, err := json.MarshalIndent(projection, "", "  ")
		if err != nil {
			return Result{}, err
		}
		projectionBody = append(projectionBody, '\n')
		projectionSHA = digestBytes(projectionBody)
		writes["projection.json"] = projectionBody
	}
	if err := addShellAssets(writes); err != nil {
		return Result{}, err
	}
	identity := sha256.Sum256([]byte(manifestSHA + "\n" + embeddedAssetsSHA256()))
	packageID := hex.EncodeToString(identity[:])
	finalDir := filepath.Join(root, packageID)
	result := func(packageSHA string) Result {
		return Result{
			PackageID: packageID, ArtifactRef: artifactRef, ArtifactSHA256: outputSHA, ManifestSHA256: manifestSHA,
			ProjectionRef: projectionID, ProjectionSHA256: projectionSHA, OutputSHA256: outputSHA, PackageSHA256: packageSHA,
			RelativePath: "./" + packageID + "/", PortableRelativePath: "./" + packageID + "/portable.zip", Directory: finalDir,
		}
	}
	if _, err := os.Stat(finalDir); err == nil {
		if err := verifyPackageFiles(finalDir, writes); err != nil {
			return Result{}, err
		}
		packageSHA, err := EnsurePortableArchive(root, packageID)
		if err != nil {
			return Result{}, err
		}
		return result(packageSHA), nil
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return Result{}, err
	}
	staging, err := os.MkdirTemp(root, ".generic-share-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	if err := writePackageFiles(staging, writes); err != nil {
		return Result{}, err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			return AssembleGeneric(root, input)
		}
		return Result{}, err
	}
	packageSHA, err := EnsurePortableArchive(root, packageID)
	if err != nil {
		return Result{}, err
	}
	return result(packageSHA), nil
}

func ValidateGenericPackage(directory, packageID string) (GenericManifest, GenericProjection, error) {
	var manifest GenericManifest
	body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
	if err != nil || json.Unmarshal(body, &manifest) != nil || manifest.Schema != GenericArtifactSchema {
		return GenericManifest{}, GenericProjection{}, ErrInvalidInput
	}
	if manifest.Availability != "ready" || !manifest.Scope.Complete() || manifest.AssetRef == "" || !safePackagePath(strings.TrimPrefix(manifest.AssetRef, "./")) {
		return GenericManifest{}, GenericProjection{}, ErrInvalidInput
	}
	payload, err := os.ReadFile(filepath.Join(directory, strings.TrimPrefix(manifest.AssetRef, "./")))
	if err != nil || len(payload) != manifest.Bytes || digestBytes(payload) != manifest.AssetSHA256 {
		return GenericManifest{}, GenericProjection{}, ErrInvalidInput
	}
	var projection GenericProjection
	projectionBody, err := os.ReadFile(filepath.Join(directory, "projection.json"))
	if err != nil || json.Unmarshal(projectionBody, &projection) != nil || projection.Schema != GenericProjectionSchema || projection.ArtifactRef != manifest.ArtifactRef || projection.ArtifactSHA256 != manifest.AssetSHA256 || !reflect.DeepEqual(projection.Scope, manifest.Scope) || projection.Availability != "ready" {
		return GenericManifest{}, GenericProjection{}, ErrInvalidInput
	}
	manifestBody, _ := json.MarshalIndent(manifest, "", "  ")
	manifestBody = append(manifestBody, '\n')
	identity := sha256.Sum256([]byte(digestBytes(manifestBody) + "\n" + embeddedAssetsSHA256()))
	if hex.EncodeToString(identity[:]) != packageID {
		return GenericManifest{}, GenericProjection{}, ErrInvalidInput
	}
	return manifest, projection, nil
}

func normalizeGenericExtension(value string) (string, bool) {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
	allowed := map[string]bool{
		"json": true, "txt": true, "md": true, "html": true, "csv": true, "pdf": true, "zip": true,
		"png": true, "jpeg": true, "jpg": true, "gif": true, "webp": true, "mp4": true, "webm": true, "mp3": true, "wav": true, "vtt": true,
		"wasm": true, "js": true,
	}
	return value, allowed[value]
}

func normalizedGenericRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" || strings.ContainsAny(ref, "\r\n\x00") {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func validGenericText(value string, max int) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= max && !strings.ContainsAny(value, "\r\n\x00")
}
