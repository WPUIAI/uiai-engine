package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"
)

func TestVerifyArchiveAcceptsBoundRendererOutput(t *testing.T) {
	request, projection, manifest, source := archiveFixture(t)
	rendered, err := RenderProjectionArchive(request, projection, []ArchiveAssetSource{source}, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:archive", manifest.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyArchive(rendered.Output, rendered.Manifest); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), rendered.Output...)
	tampered[len(tampered)/2] ^= 1
	if err := VerifyArchive(tampered, rendered.Manifest); !errors.Is(err, ErrDerivativeIdentityMismatch) {
		t.Fatalf("tampered output error = %v", err)
	}
	for _, wrapped := range [][]byte{
		append([]byte("hidden-prefix"), rendered.Output...),
		append(append([]byte(nil), rendered.Output...), []byte("hidden-suffix")...),
	} {
		wrappedManifest := rendered.Manifest
		wrappedDigest := sha256.Sum256(wrapped)
		wrappedManifest.OutputSHA256 = hex.EncodeToString(wrappedDigest[:])
		wrappedManifest.OutputBytes = uint64(len(wrapped))
		if err := VerifyArchive(wrapped, wrappedManifest); !errors.Is(err, ErrDerivativeUnsafeArchive) {
			t.Fatalf("polyglot archive error = %v", err)
		}
	}
	badManifest := rendered.Manifest
	badManifest.ArchiveEntries = append([]ArchiveEntry(nil), rendered.Manifest.ArchiveEntries...)
	badManifest.ArchiveEntries[0].SHA256 = string(bytes.Repeat([]byte{'f'}, 64))
	if err := VerifyArchive(rendered.Output, badManifest); !errors.Is(err, ErrDerivativeUnsafeArchive) {
		t.Fatalf("entry hash error = %v", err)
	}
}

func TestVerifyArchiveRejectsTraversalLinksAndCompressionBombs(t *testing.T) {
	body := []byte("payload")
	for _, test := range []struct {
		name    string
		path    string
		symlink bool
		method  uint16
		body    []byte
	}{
		{name: "traversal", path: "../escape", method: zip.Store, body: body},
		{name: "symlink", path: "evidence/link", symlink: true, method: zip.Store, body: body},
		{name: "compression_bomb", path: "evidence/bomb", method: zip.Deflate, body: bytes.Repeat([]byte{0}, 2*1024*1024)},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := testArchive(t, test.path, test.symlink, test.method, test.body)
			bodyDigest := sha256.Sum256(test.body)
			outputDigest := sha256.Sum256(output)
			manifest := DerivativeManifest{
				ArchivePosture: ArchiveSafe,
				ArchiveEntries: []ArchiveEntry{{Path: test.path, SHA256: hex.EncodeToString(bodyDigest[:]), MIME: "application/octet-stream", Bytes: uint64(len(test.body))}},
				OutputSHA256:   hex.EncodeToString(outputDigest[:]), OutputBytes: uint64(len(output)),
			}
			if err := VerifyArchive(output, manifest); !errors.Is(err, ErrDerivativeUnsafeArchive) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func testArchive(t *testing.T, name string, symlink bool, method uint16, body []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	header := &zip.FileHeader{Name: name, Method: method}
	if symlink {
		header.SetMode(os.ModeSymlink | 0o777)
	}
	header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
