package evidencederivative

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

const maxArchiveEntries = 4096
const maxArchiveExpandedBytes = 256 * 1024 * 1024
const maxArchiveCompressionRatio = 100

func VerifyArchive(output []byte, manifest DerivativeManifest) error {
	if manifest.ArchivePosture != ArchiveSafe || len(output) == 0 || len(manifest.ArchiveEntries) == 0 || len(manifest.ArchiveEntries) > maxArchiveEntries {
		return ErrDerivativeUnsafeArchive
	}
	if err := validateArchive(manifest.ArchivePosture, manifest.ArchiveEntries); err != nil {
		return err
	}
	outputDigest := sha256.Sum256(output)
	if hex.EncodeToString(outputDigest[:]) != manifest.OutputSHA256 || uint64(len(output)) != manifest.OutputBytes {
		return ErrDerivativeIdentityMismatch
	}
	reader, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	if err != nil || len(reader.File) != len(manifest.ArchiveEntries) || len(reader.File) > maxArchiveEntries {
		return ErrDerivativeUnsafeArchive
	}
	expected := make(map[string]ArchiveEntry, len(manifest.ArchiveEntries))
	for _, entry := range manifest.ArchiveEntries {
		if !safeArchivePath(entry.Path) || entry.Link || entry.Bytes == 0 {
			return ErrDerivativeUnsafeArchive
		}
		if _, duplicate := expected[entry.Path]; duplicate {
			return ErrDerivativeUnsafeArchive
		}
		expected[entry.Path] = entry
	}
	seen := make(map[string]struct{}, len(reader.File))
	var expanded uint64
	for _, file := range reader.File {
		entry, found := expected[file.Name]
		if !found || !safeArchivePath(file.Name) || file.FileInfo().IsDir() || file.Mode()&os.ModeSymlink != 0 ||
			(file.Method != zip.Store && file.Method != zip.Deflate) || file.Flags&1 != 0 {
			return ErrDerivativeUnsafeArchive
		}
		if _, duplicate := seen[file.Name]; duplicate {
			return ErrDerivativeUnsafeArchive
		}
		seen[file.Name] = struct{}{}
		if file.UncompressedSize64 != entry.Bytes || file.UncompressedSize64 == 0 || file.UncompressedSize64 > maxArchiveExpandedBytes {
			return ErrDerivativeUnsafeArchive
		}
		expanded += file.UncompressedSize64
		if expanded > maxArchiveExpandedBytes || file.CompressedSize64 == 0 ||
			(file.UncompressedSize64 > file.CompressedSize64 && file.UncompressedSize64/file.CompressedSize64 > maxArchiveCompressionRatio) {
			return ErrDerivativeUnsafeArchive
		}
		stream, err := file.Open()
		if err != nil {
			return ErrDerivativeUnsafeArchive
		}
		hash := sha256.New()
		written, copyErr := io.Copy(hash, io.LimitReader(stream, int64(entry.Bytes)+1))
		closeErr := stream.Close()
		if copyErr != nil || closeErr != nil || written != int64(entry.Bytes) || hex.EncodeToString(hash.Sum(nil)) != entry.SHA256 {
			return ErrDerivativeUnsafeArchive
		}
	}
	return nil
}
