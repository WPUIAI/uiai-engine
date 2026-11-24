package deviceframes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #298: manifest discovery must not fall back to a hard-coded checkout path;
// it resolves relative to the working directory plus an explicit operator
// override, and reports absence instead of inventing a location.
func TestDiscoverManifestPathHasNoHostFallback(t *testing.T) {
	t.Setenv("UIAI_DEVICEFRAMES_MANIFEST", "")
	wd, err := os.Getwd()
	if err != nil {
		t.Skip("no working directory available")
	}
	tmp := t.TempDir()
	prev, _ := os.Getwd()
	if chErr := os.Chdir(tmp); chErr != nil {
		t.Skip("cannot chdir")
	}
	defer func() { _ = os.Chdir(prev) }()
	got := discoverManifestPath()
	if got == "" || strings.HasPrefix(got, "/home/") || !filepath.IsAbs(got) && got == "" {
		t.Fatalf("manifest discovery returned %q from an empty temp dir; want the relative candidate", got)
	}
	if got != "internal/media/deviceframes/manifest.json" {
		t.Fatalf("manifest discovery returned %q, want the relative candidate", got)
	}
	_ = wd
}
