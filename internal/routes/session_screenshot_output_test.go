package routes

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestSaveScreenshotArtifact(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Vision: config.VisionConfig{ShareDir: dir}}
	snap := &vision.SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString([]byte("fake image")),
		Format:     "png",
		Size:       len("fake image"),
		Width:      10,
		Height:     20,
	}
	name, path, err := saveScreenshotArtifact(cfg, "session/with spaces", snap)
	if err != nil {
		t.Fatalf("saveScreenshotArtifact returned error: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(dir, "session-screenshots") {
		t.Fatalf("unexpected artifact dir: %s", path)
	}
	if filepath.Base(path) != name {
		t.Fatalf("name/path mismatch: name=%s path=%s", name, path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(got) != "fake image" {
		t.Fatalf("unexpected artifact content: %q", got)
	}
}

func TestScreenshotArtifactPathRejectsTraversal(t *testing.T) {
	cfg := &config.Config{Vision: config.VisionConfig{ShareDir: t.TempDir()}}
	if _, ok := screenshotArtifactPath(cfg, "../secret.png"); ok {
		t.Fatal("expected traversal artifact name to be rejected")
	}
	if path, ok := screenshotArtifactPath(cfg, "safe.png"); !ok || filepath.Base(path) != "safe.png" {
		t.Fatalf("expected safe artifact path, got path=%q ok=%v", path, ok)
	}
}
