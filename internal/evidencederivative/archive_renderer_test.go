package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	request.AssetRefs = []string{projection.Assets[0].AssetID}
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
