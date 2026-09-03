package evidencederivative

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const maxJSONProjectionBytes = 16 * 1024 * 1024

var ErrProjectionMismatch = errors.New("evidence derivative projection mismatch")

type RenderedDerivative struct {
	Output   []byte
	Manifest DerivativeManifest
}

func RenderCanonicalJSON(request DerivativeRequest, projection []byte, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if len(projection) == 0 || len(projection) > maxJSONProjectionBytes {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(projection))
	decoder.DisallowUnknownFields()
	var typed evidencepwa.Projection
	if err := decoder.Decode(&typed); err != nil {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	return RenderProjectionJSON(request, typed, renderer, matrix, licenses, receiptRef, createdAt)
}

func RenderProjectionJSON(request DerivativeRequest, projection evidencepwa.Projection, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if request.DerivativeType != DerivativeJSON || !renderingProfileMatches(request.Rendering, PortableDataRenderingProfile()) {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	selection, err := selectProjection(request, projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	output := struct {
		Schema              string                        `json:"schema"`
		ProjectionRef       string                        `json:"projection_ref"`
		ProjectionSHA256    string                        `json:"projection_sha256"`
		Artifact            evidencepwa.ArtifactBinding   `json:"artifact"`
		Title               string                        `json:"title"`
		Summary             string                        `json:"summary"`
		Availability        evidencepwa.AvailabilityState `json:"availability"`
		Access              evidencepwa.AccessPosture     `json:"access"`
		Redaction           evidencepwa.Redaction         `json:"redaction"`
		FreshnessObservedAt time.Time                     `json:"freshness_observed_at"`
		Claims              []evidencepwa.Claim           `json:"claims"`
		Assets              []evidencepwa.Asset           `json:"assets"`
		Citations           []evidencepwa.Citation        `json:"citations"`
		OmissionRefs        []string                      `json:"omission_refs"`
	}{
		Schema: "uiai.evidence_derivative_json.v1", ProjectionRef: request.ProjectionRef,
		ProjectionSHA256: request.ProjectionSHA256, Artifact: projection.Artifact,
		Title: projection.Title, Summary: projection.Summary, Availability: projection.Availability,
		Access: projection.Access, Redaction: projection.Redaction, FreshnessObservedAt: projection.FreshnessObservedAt,
		Claims: selection.claims, Assets: selection.assets, Citations: selection.citations,
		OmissionRefs: append([]string(nil), request.OmissionRefs...),
	}
	canonical, err := json.Marshal(output)
	if err != nil {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	canonical = append(canonical, '\n')
	return buildRenderedDerivative(request, canonical, "json", "application/json", renderer, matrix, licenses, receiptRef, createdAt)
}

func buildRenderedDerivative(request DerivativeRequest, output []byte, extension, mime string, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	return buildRenderedDerivativeWithArchive(request, output, extension, mime, renderer, matrix, licenses, receiptRef, createdAt, ArchiveNotApplicable, nil)
}

func buildRenderedDerivativeWithArchive(request DerivativeRequest, output []byte, extension, mime string, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time, archivePosture ArchivePosture, archiveEntries []ArchiveEntry) (RenderedDerivative, error) {
	if len(output) == 0 || extension == "" || strings.ContainsAny(extension, "/\\") || mime == "" {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	viewerMatrix := cloneViewerMatrix(matrix)
	licenseSet := append([]LicenseAttestation(nil), licenses...)
	sort.Slice(licenseSet, func(i, j int) bool {
		if licenseSet[i].AssetRef == licenseSet[j].AssetRef {
			return licenseSet[i].LicenseRef < licenseSet[j].LicenseRef
		}
		return licenseSet[i].AssetRef < licenseSet[j].AssetRef
	})
	derivativeID, err := DerivativeID(request, renderer, viewerMatrix, licenseSet)
	if err != nil {
		return RenderedDerivative{}, err
	}
	requestDigest, err := DigestRequest(request)
	if err != nil {
		return RenderedDerivative{}, err
	}
	outputDigest := sha256.Sum256(output)
	manifest := DerivativeManifest{
		Schema: ManifestSchema, DerivativeID: derivativeID, RequestRef: request.RequestID, RequestSHA256: requestDigest,
		ArtifactRef: request.ArtifactRef, ArtifactSHA256: request.ArtifactSHA256,
		ProjectionRef: request.ProjectionRef, ProjectionSHA256: request.ProjectionSHA256,
		OutputRef:    "derivatives/" + strings.TrimPrefix(derivativeID, "derivative:") + "." + extension,
		OutputSHA256: hex.EncodeToString(outputDigest[:]), OutputBytes: uint64(len(output)), OutputMIME: mime,
		Renderer: renderer, Rendering: request.Rendering,
		AccessibilityTarget: request.AccessibilityTarget, AccessibilityPosture: ConformanceNotClaimed,
		ArchivePosture: archivePosture, ArchiveEntries: append([]ArchiveEntry(nil), archiveEntries...), ViewerMatrix: viewerMatrix, Licenses: licenseSet,
		OmissionRefs: append([]string(nil), request.OmissionRefs...), ReceiptRef: receiptRef, CreatedAt: createdAt.UTC(),
	}
	if err := ValidateManifest(manifest, request); err != nil {
		return RenderedDerivative{}, err
	}
	return RenderedDerivative{Output: append([]byte(nil), output...), Manifest: manifest}, nil
}

func cloneViewerMatrix(matrix ViewerMatrix) ViewerMatrix {
	matrix.Entries = append([]ViewerEntry(nil), matrix.Entries...)
	for index := range matrix.Entries {
		matrix.Entries[index].EvidenceRefs = append([]string(nil), matrix.Entries[index].EvidenceRefs...)
	}
	sort.Slice(matrix.Entries, func(i, j int) bool {
		if matrix.Entries[i].Client == matrix.Entries[j].Client {
			return matrix.Entries[i].Version < matrix.Entries[j].Version
		}
		return matrix.Entries[i].Client < matrix.Entries[j].Client
	})
	return matrix
}
