package evidenceshare

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAssembleGenericCreatesDeterministicPortableRecord(t *testing.T) {
	root := t.TempDir()
	input := GenericInput{
		ArtifactRef: "report:focusa-homepage-v2.1", Revision: 1, Title: "Homepage verification report",
		Kind: "generated_report", MediaType: "application/json", Extension: "json", Payload: []byte("{\"verified\":true}\n"),
		SourceRef: "https://focusa.dev/", CapturedAt: time.Date(2026, 9, 4, 1, 2, 3, 0, time.UTC), Scope: completeScope(),
		ParentArtifactRef: "artifact:homepage", ChildArtifactRefs: []string{"artifact:desktop", "artifact:mobile"},
	}
	first, err := AssembleGeneric(root, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := AssembleGeneric(root, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.PackageID != second.PackageID || first.PackageSHA256 != second.PackageSHA256 || first.ManifestSHA256 != second.ManifestSHA256 || first.ProjectionSHA256 != second.ProjectionSHA256 {
		t.Fatalf("generic package is not deterministic: first=%#v second=%#v", first, second)
	}
	manifest, projection, err := ValidateGenericPackage(first.Directory, first.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ArtifactRef != input.ArtifactRef || projection.ParentArtifactRef != input.ParentArtifactRef || len(projection.ChildArtifactRefs) != 2 {
		t.Fatalf("generic relationship projection drifted: manifest=%#v projection=%#v", manifest, projection)
	}
	archiveBody, err := os.ReadFile(filepath.Join(root, first.PackageID+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveBody), int64(len(archiveBody)))
	if err != nil {
		t.Fatal(err)
	}
	foundPayload := false
	for _, file := range archive.File {
		if file.Name == "payload.json" {
			foundPayload = true
		}
	}
	if !foundPayload {
		t.Fatalf("portable package omitted generic payload: %#v", archive.File)
	}
}

func TestAssembleGenericBlocksIncompleteScopeAndRejectsTampering(t *testing.T) {
	root := t.TempDir()
	input := GenericInput{Title: "Diagnostics", Kind: "diagnostics", MediaType: "application/json", Extension: "json", Payload: []byte("{}\n"), CapturedAt: time.Now().UTC()}
	blocked, err := AssembleGeneric(root, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateGenericPackage(blocked.Directory, blocked.PackageID); err == nil {
		t.Fatal("incomplete-scope generic package was accepted as public ready")
	}
	input.Scope = completeScope()
	ready, err := AssembleGeneric(root, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ready.Directory, "artifact.json"), []byte("{}\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateGenericPackage(ready.Directory, ready.PackageID); err == nil {
		t.Fatal("tampered generic package was accepted")
	}
}
