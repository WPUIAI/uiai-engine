package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func TestRenderProjectionArchiveIsSafeDeterministicAndBound(t *testing.T) {
	request, projection, manifest, source := archiveFixture(t)
	first, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("archive render is not deterministic")
	}
	if first.Manifest.ArchivePosture != ArchiveSafe || len(first.Manifest.ArchiveEntries) != 2 {
		t.Fatalf("archive manifest = %#v", first.Manifest)
	}
	reader, err := zip.NewReader(bytes.NewReader(first.Output), int64(len(first.Output)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 2 || reader.File[0].Name != "assets/layout.png" || reader.File[1].Name != "evidence/selection.json" {
		t.Fatalf("archive files = %#v", reader.File)
	}
	body, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source.Body) {
		t.Fatal("asset bytes changed")
	}
	selectionStream, err := reader.File[1].Open()
	if err != nil {
		t.Fatal(err)
	}
	selectionBytes, err := io.ReadAll(selectionStream)
	selectionStream.Close()
	if err != nil {
		t.Fatal(err)
	}
	var selection struct {
		Schema           string               `json:"schema"`
		RequestSHA256    string               `json:"request_sha256"`
		Scope            ScopeBinding         `json:"scope"`
		ArtifactRevision uint64               `json:"artifact_revision"`
		Licenses         []LicenseAttestation `json:"licenses"`
		OmissionRefs     []string             `json:"omission_refs"`
	}
	if err := json.Unmarshal(selectionBytes, &selection); err != nil {
		t.Fatal(err)
	}
	requestDigest, _ := DigestRequest(request)
	if selection.Schema != "uiai.evidence_archive_selection.v1" || selection.RequestSHA256 != requestDigest ||
		selection.Scope != request.Scope || selection.ArtifactRevision != request.ArtifactRevision ||
		!reflect.DeepEqual(selection.Licenses, manifest.Licenses) || !reflect.DeepEqual(selection.OmissionRefs, request.OmissionRefs) {
		t.Fatalf("embedded selection ledger = %#v", selection)
	}
	if err := ValidateManifest(first.Manifest, request); err != nil {
		t.Fatal(err)
	}
}

func TestRenderProjectionArchiveRejectsTraversalTamperAndDuplicates(t *testing.T) {
	request, projection, manifest, source := archiveFixture(t)
	for _, badPath := range []string{"../escape", "/absolute", "assets/../escape", "assets\\escape", ".hidden"} {
		bad := source
		bad.Path = badPath
		if _, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{bad}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt); !errors.Is(err, ErrDerivativeUnsafeArchive) {
			t.Fatalf("path %q error = %v", badPath, err)
		}
	}
	bad := source
	bad.Body = []byte("tampered")
	if _, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{bad}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt); err == nil {
		t.Fatal("tampered asset accepted")
	}
	if _, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source, source}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt); err == nil {
		t.Fatal("duplicate asset accepted")
	}
	extraLicenses := append([]LicenseAttestation(nil), manifest.Licenses...)
	extra := manifest.Licenses[0]
	extra.AssetRef = "asset:unselected"
	extra.LicenseRef = "license:unselected"
	extraLicenses = append(extraLicenses, extra)
	if _, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source}, manifest.Renderer, manifest.ViewerMatrix, extraLicenses, "receipt:archive", manifest.CreatedAt); !errors.Is(err, ErrDerivativeLicenseMissing) {
		t.Fatalf("unselected license error = %v", err)
	}
}

func TestRenderProjectionArchiveRejectsFalseRenderingProfile(t *testing.T) {
	request, projection, manifest, source := archiveFixture(t)
	request.Rendering.ProfileRef = "rendering:unimplemented"
	if _, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("false archive profile accepted: %v", err)
	}
}

func archiveFixture(t *testing.T) (DerivativeRequest, evidencepwa.Projection, DerivativeManifest, ArchiveAssetSource) {
	t.Helper()
	request, projection, manifest := portableFixture(t)
	body := []byte("deterministic archive asset\n")
	sum := sha256.Sum256(body)
	projection.Assets = projection.Assets[:1]
	projection.Assets[0].SHA256 = hex.EncodeToString(sum[:])
	projection.Assets[0].Bytes = uint64(len(body))
	for index := range projection.Citations {
		if projection.Citations[index].SourceRef == projection.Assets[0].AssetID {
			projection.Citations[index].SHA256 = projection.Assets[0].SHA256
		}
	}
	request.DerivativeType = DerivativeArchive
	request.Rendering = ArchiveStoreRenderingProfile()
	request.AssetRefs = []string{projection.Assets[0].AssetID}
	request.OmissionRefs = DerivativeOmissionRefs(request, projection)
	request.RequiredEvidenceRefs = selectedEvidenceRefs(request)
	digest, err := evidencepwa.DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectionSHA256 = digest
	manifest.Licenses = manifest.Licenses[:1]
	manifest.Licenses[0].AssetRef = projection.Assets[0].AssetID
	return request, projection, manifest, ArchiveAssetSource{AssetRef: projection.Assets[0].AssetID, Path: "assets/layout.png", MIME: projection.Assets[0].MIME, Body: body}
}
