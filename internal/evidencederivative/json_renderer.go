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
)

const maxJSONProjectionBytes = 16 * 1024 * 1024

var ErrProjectionMismatch = errors.New("evidence derivative projection mismatch")

type RenderedDerivative struct {
	Output   []byte
	Manifest DerivativeManifest
}

func RenderCanonicalJSON(request DerivativeRequest, projection []byte, renderer RendererIdentity, matrix ViewerMatrix, licenses []LicenseAttestation, receiptRef string, createdAt time.Time) (RenderedDerivative, error) {
	if err := ValidateRequest(request); err != nil {
		return RenderedDerivative{}, err
	}
	if request.DerivativeType != DerivativeJSON || len(projection) == 0 || len(projection) > maxJSONProjectionBytes {
		return RenderedDerivative{}, ErrDerivativeContractInvalid
	}
	projectionDigest := sha256.Sum256(projection)
	if hex.EncodeToString(projectionDigest[:]) != request.ProjectionSHA256 {
		return RenderedDerivative{}, ErrProjectionMismatch
	}
	canonical, err := canonicalJSON(projection)
	if err != nil {
		return RenderedDerivative{}, err
	}
	viewerMatrix := cloneViewerMatrix(matrix)
	licenseSet := append([]LicenseAttestation(nil), licenses...)
	sort.Slice(licenseSet, func(i, j int) bool { return licenseSet[i].AssetRef < licenseSet[j].AssetRef })
	derivativeID, err := DerivativeID(request, renderer, viewerMatrix, licenseSet)
	if err != nil {
		return RenderedDerivative{}, err
	}
	requestDigest, err := DigestRequest(request)
	if err != nil {
		return RenderedDerivative{}, err
	}
	outputDigest := sha256.Sum256(canonical)
	manifest := DerivativeManifest{
		Schema: ManifestSchema, DerivativeID: derivativeID, RequestRef: request.RequestID, RequestSHA256: requestDigest,
		ArtifactRef: request.ArtifactRef, ArtifactSHA256: request.ArtifactSHA256,
		ProjectionRef: request.ProjectionRef, ProjectionSHA256: request.ProjectionSHA256,
		OutputRef:    "derivatives/" + strings.TrimPrefix(derivativeID, "derivative:") + ".json",
		OutputSHA256: hex.EncodeToString(outputDigest[:]), OutputBytes: uint64(len(canonical)), OutputMIME: "application/json",
		Renderer: renderer, Rendering: request.Rendering,
		AccessibilityTarget: request.AccessibilityTarget, AccessibilityPosture: ConformanceNotClaimed,
		ArchivePosture: ArchiveNotApplicable, ViewerMatrix: viewerMatrix, Licenses: licenseSet,
		OmissionRefs: append([]string(nil), request.OmissionRefs...), ReceiptRef: receiptRef, CreatedAt: createdAt.UTC(),
	}
	if err := ValidateManifest(manifest, request); err != nil {
		return RenderedDerivative{}, err
	}
	return RenderedDerivative{Output: append([]byte(nil), canonical...), Manifest: manifest}, nil
}

func canonicalJSON(body []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, ErrDerivativeContractInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, ErrDerivativeContractInvalid
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, ErrDerivativeContractInvalid
	}
	return append(canonical, '\n'), nil
}

func cloneViewerMatrix(matrix ViewerMatrix) ViewerMatrix {
	matrix.Entries = append([]ViewerEntry(nil), matrix.Entries...)
	for index := range matrix.Entries {
		matrix.Entries[index].EvidenceRefs = append([]string(nil), matrix.Entries[index].EvidenceRefs...)
	}
	sort.Slice(matrix.Entries, func(i, j int) bool { return matrix.Entries[i].Client < matrix.Entries[j].Client })
	return matrix
}
