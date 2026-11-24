package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

)

// #298: defaults must never assume one Linux host. Screenshot artifact store
// resolves relative to the engine working directory, not /home/wpuiai.
func TestScreenshotStoreDirHasNoHostFallback(t *testing.T) {
	t.Setenv("UIAI_SCREENSHOT_DIR", "")
	got := screenshotStoreDir()
	if filepath.IsAbs(got) || strings.Contains(got, "/home/") {
		t.Fatalf("screenshot store default %q must be a relative portable path", got)
	}
	if got != filepath.Join("data", "screenshots") {
		t.Fatalf("screenshot store default %q, want data/screenshots", got)
	}
	t.Setenv("UIAI_SCREENSHOT_DIR", "/tmp/uiai-298-override")
	if got := screenshotStoreDir(); got != "/tmp/uiai-298-override" {
		t.Fatalf("explicit UIAI_SCREENSHOT_DIR override ignored: %q", got)
	}
}

// #298: FPV project context must resolve from the working directory, never a
// hard-coded checkout path.
func TestFPVProjectRootIsWorkingDirectory(t *testing.T) {
	root := fpvProjectRoot()
	wd, err := os.Getwd()
	if err != nil {
		t.Skip("no working directory available")
	}
	if root != wd {
		t.Fatalf("project root %q must resolve from the working directory %q", root, wd)
	}
	ctx := fpvContext(nil)
	project := ctx["project"].(map[string]any)
	if project["root"] != wd {
		t.Fatalf("fpv context project root %v does not match working directory %q", project["root"], wd)
	}
}
