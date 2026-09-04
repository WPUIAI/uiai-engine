package evidenceshare

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func TestAssembleArtifactCreatesDeterministicPortableEPWA(t *testing.T) {
	manifest, assets := artifactFixture(t)
	root := t.TempDir()
	first, err := AssembleArtifact(root, ArtifactInput{Manifest: manifest, Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	second, err := AssembleArtifact(root, ArtifactInput{Manifest: manifest, Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	if first.PackageID != second.PackageID || first.PackageSHA256 != second.PackageSHA256 || first.ProjectionSHA256 == "" {
		t.Fatalf("artifact package is not deterministic: first=%#v second=%#v", first, second)
	}
	if got, err := os.ReadFile(filepath.Join(first.Directory, manifest.Assets[0].Path)); err != nil || string(got) != string(assets[manifest.Assets[0].AssetID]) {
		t.Fatalf("nested evidence asset mismatch: got=%q err=%v", got, err)
	}
	index, err := os.ReadFile(filepath.Join(first.Directory, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(index, "generic-record.js") || !containsText(index, `data-default-view="record"`) {
		t.Fatalf("portable viewer is not wired for generic artifacts: %s", index)
	}
	archive, err := zip.OpenReader(filepath.Join(root, first.PackageID+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	foundNested := false
	for _, entry := range archive.File {
		if entry.Name == manifest.Assets[0].Path {
			foundNested = true
		}
	}
	if !foundNested {
		t.Fatalf("portable archive omitted nested evidence asset: %#v", archive.File)
	}
}

func TestAssembleArtifactRejectsAssetDriftAndShellCollision(t *testing.T) {
	manifest, assets := artifactFixture(t)
	assets[manifest.Assets[0].AssetID] = []byte("changed")
	if _, err := AssembleArtifact(t.TempDir(), ArtifactInput{Manifest: manifest, Assets: assets}); err == nil {
		t.Fatal("asset digest drift was accepted")
	}
	manifest, assets = artifactFixture(t)
	manifest.Assets[0].Path = "index.html"
	manifest.Integrity.ManifestSHA256 = ""
	manifest.Integrity.ManifestSHA256, _ = evidenceartifact.ComputeManifestSHA256(manifest)
	if _, err := AssembleArtifact(t.TempDir(), ArtifactInput{Manifest: manifest, Assets: assets}); err == nil {
		t.Fatal("shell collision was accepted")
	}
}

func artifactFixture(t *testing.T) (evidenceartifact.Manifest, map[string][]byte) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "evidenceartifact", "testdata", "manifest.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest evidenceartifact.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	data := []byte("{\"result\":\"portable\"}\n")
	manifest.Assets[0].Path = "assets/proof.json"
	manifest.Assets[0].SHA256 = digestBytes(data)
	manifest.Assets[0].ByteSize = int64(len(data))
	manifest.Integrity.BundleSHA256 = digestBytes([]byte("artifact-bundle"))
	manifest.Integrity.ManifestSHA256 = ""
	manifest.Integrity.ManifestSHA256, err = evidenceartifact.ComputeManifestSHA256(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, map[string][]byte{manifest.Assets[0].AssetID: data}
}

func containsText(body []byte, text string) bool {
	for index := 0; index+len(text) <= len(body); index++ {
		if string(body[index:index+len(text)]) == text {
			return true
		}
	}
	return false
}
