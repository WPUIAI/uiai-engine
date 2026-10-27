package evidenceshare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// screenshotPackageID is the existing producer identity, shared with recognition
// of this producer's unsealed records. It is not a public-readiness decision.
func screenshotPackageID(screenshotSHA string, input Input) (string, error) {
	scopeBody, err := json.Marshal(input.Scope)
	if err != nil {
		return "", err
	}
	value := fmt.Sprintf("%s\n%d\n%d\n%s\n%s\n%s\n%s", screenshotSHA, input.Width, input.Height, sanitizeURL(input.SourceURL), input.CapturedAt.UTC().Format(time.RFC3339Nano), embeddedAssetsSHA256(), scopeBody)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:]), nil
}

// VerifyScreenshotArtifactBytes recognizes current-producer bytes without
// publishing them. Scope/projection/access readiness stays in the public gate;
// older public packages are deliberately not revalidated against current assets.
func VerifyScreenshotArtifactBytes(directory, id string) (Manifest, error) {
	var manifest Manifest
	if len(id) != 64 {
		return manifest, ErrInvalidInput
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() {
		return manifest, ErrInvalidInput
	}
	info, err = os.Lstat(filepath.Join(directory, "artifact.json"))
	if err != nil || !info.Mode().IsRegular() {
		return manifest, ErrInvalidInput
	}
	body, err := os.ReadFile(filepath.Join(directory, "artifact.json"))
	if err != nil || json.Unmarshal(body, &manifest) != nil {
		return Manifest{}, ErrInvalidInput
	}
	format, mime, ok := normalizedFormat(manifest.Format)
	if !ok || manifest.Schema != Schema || manifest.ArtifactSHA256 != id || manifest.ArtifactRef != "uiai-evidence-share:sha256:"+id || manifest.Format != format || manifest.MIME != mime || manifest.ScreenshotRef != "./screenshot."+format || manifest.Bytes <= 0 || manifest.Width <= 0 || manifest.Height <= 0 || manifest.CapturedAt.IsZero() {
		return Manifest{}, ErrInvalidInput
	}
	want, err := screenshotPackageID(manifest.ScreenshotSHA256, Input{Width: manifest.Width, Height: manifest.Height, SourceURL: manifest.SourceURL, CapturedAt: manifest.CapturedAt, Scope: manifest.Scope})
	if err != nil || want != id {
		return Manifest{}, ErrInvalidInput
	}
	path := filepath.Join(directory, "screenshot."+format)
	info, err = os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(manifest.Bytes) {
		return Manifest{}, ErrInvalidInput
	}
	pixels, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, ErrInvalidInput
	}
	digest := sha256.Sum256(pixels)
	if hex.EncodeToString(digest[:]) != manifest.ScreenshotSHA256 {
		return Manifest{}, ErrInvalidInput
	}
	return manifest, nil
}
