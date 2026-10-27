package routes

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// A process boundary bounds a broken ancestor walk without leaving a stuck
// goroutine or changing the parent test process's working directory.
func TestFPVRegistryPathTerminatesOutsideModule(t *testing.T) {
	dir := t.TempDir()
	for parent := dir; ; {
		if _, err := os.Stat(filepath.Join(parent, "go.mod")); err == nil {
			t.Skip("test temporary directory has an enclosing module")
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFPVRegistryPathOutsideModuleChild$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "UIAI_FPV_PATH_TEST_CHILD=1")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal("registry lookup did not terminate at filesystem root")
	}
	if err != nil {
		t.Fatalf("isolated registry lookup failed: %v\n%s", err, output)
	}
}

func TestFPVRegistryPathOutsideModuleChild(t *testing.T) {
	if os.Getenv("UIAI_FPV_PATH_TEST_CHILD") != "1" {
		return
	}
	want := filepath.Join("data", "fpv-shares.json")
	if got := fpvRegistryPath(); got != want {
		t.Fatalf("fallback path=%q, want %q", got, want)
	}
}

func TestFPVRegistryPathFindsNearestModule(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(filepath.Join(nested, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(nested, "child"))
	if got, want := fpvRegistryPath(), filepath.Join(root, "data", "fpv-shares.json"); got != want {
		t.Fatalf("enclosing module path=%q, want %q", got, want)
	}
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module nested\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, want := fpvRegistryPath(), filepath.Join(nested, "data", "fpv-shares.json"); got != want {
		t.Fatalf("nearest module path=%q, want %q", got, want)
	}
}
