package evidenceshare

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidInput = errors.New("screenshot evidence share input invalid")
	ErrConflict     = errors.New("screenshot evidence share content conflict")
)

//go:embed assets/index.html assets/styles.css assets/app.js
var assets embed.FS

func Assemble(root string, input Input) (Result, error) {
	if strings.TrimSpace(root) == "" || len(input.Screenshot) == 0 || input.Width <= 0 || input.Height <= 0 || input.CapturedAt.IsZero() {
		return Result{}, ErrInvalidInput
	}
	format, mime, ok := normalizedFormat(input.Format)
	if !ok {
		return Result{}, ErrInvalidInput
	}
	screenshotDigest := sha256.Sum256(input.Screenshot)
	screenshotSHA := hex.EncodeToString(screenshotDigest[:])
	idInput := fmt.Sprintf("%s\n%d\n%d\n%s\n%s\n%s", screenshotSHA, input.Width, input.Height, sanitizeURL(input.SourceURL), input.CapturedAt.UTC().Format(time.RFC3339Nano), embeddedAssetsSHA256())
	idDigest := sha256.Sum256([]byte(idInput))
	id := hex.EncodeToString(idDigest[:])
	artifactRef := "uiai-evidence-share:sha256:" + id
	manifest := Manifest{Schema: Schema, ArtifactRef: artifactRef, ArtifactSHA256: id, ScreenshotRef: "./screenshot." + format, ScreenshotSHA256: screenshotSHA, Format: format, MIME: mime, Bytes: len(input.Screenshot), Width: input.Width, Height: input.Height, SourceURL: sanitizeURL(input.SourceURL), CapturedAt: input.CapturedAt.UTC(), DurationMS: input.DurationMS, Availability: "ready", Access: "portable_read_only", Interaction: "read_only", Scope: input.Scope, TruthNotice: "This screenshot proves only the captured visual state. It does not by itself prove review, completion, provider closure, or settlement."}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, err
	}
	body = append(body, '\n')
	manifestSHA := id
	finalDir := filepath.Join(root, id)
	if existing, err := os.ReadFile(filepath.Join(finalDir, "artifact.json")); err == nil {
		if string(existing) != string(body) {
			return Result{}, ErrConflict
		}
		return Result{ArtifactRef: artifactRef, ArtifactSHA256: manifestSHA, RelativePath: "./" + id + "/", Directory: finalDir}, nil
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return Result{}, err
	}
	staging, err := os.MkdirTemp(root, ".share-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	writes := map[string][]byte{"artifact.json": body, "screenshot." + format: input.Screenshot}
	for _, name := range []string{"index.html", "styles.css", "app.js"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return Result{}, err
		}
		writes[name] = data
	}
	for name, data := range writes {
		if err := os.WriteFile(filepath.Join(staging, name), data, 0o640); err != nil {
			return Result{}, err
		}
	}
	if err := os.Rename(staging, finalDir); err != nil {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			return Assemble(root, input)
		}
		return Result{}, err
	}
	return Result{ArtifactRef: artifactRef, ArtifactSHA256: manifestSHA, RelativePath: "./" + id + "/", Directory: finalDir}, nil
}

func embeddedAssetsSHA256() string {
	hash := sha256.New()
	for _, name := range []string{"index.html", "styles.css", "app.js"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return ""
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func normalizedFormat(value string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "png":
		return "png", "image/png", true
	case "jpg", "jpeg":
		return "jpg", "image/jpeg", true
	case "webp":
		return "webp", "image/webp", true
	default:
		return "", "", false
	}
}
func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}
