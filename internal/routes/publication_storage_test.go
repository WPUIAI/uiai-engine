package routes

import (
	"fmt"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No routes test, opt-in browser test, or inherited service environment may
// accidentally publish fixture artifacts into an installed engine's store.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "uiai-routes-evidence-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot isolate route test evidence storage:", err)
		os.Exit(1)
	}
	store := root
	if os.Getenv("UIAI_TEST_BLOCK_DEFAULT_EVIDENCE_STORE") == "1" {
		store, err = blockedPublicationStore(root)
	}
	if err == nil {
		err = os.Setenv("UIAI_SCREENSHOT_DIR", store)
	}
	code := 1
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot configure isolated route test evidence storage:", err)
	} else {
		code = m.Run()
	}
	// Only the exact newly-created test-owned root is removed; never live data.
	if err := os.RemoveAll(root); err != nil {
		fmt.Fprintln(os.Stderr, "cannot clean route test evidence storage:", err)
		code = 1
	}
	os.Exit(code)
}

func blockedPublicationStore(root string) (string, error) {
	path := filepath.Join(root, "default-store-is-not-a-directory")
	return path, os.WriteFile(path, []byte("fixture"), 0600)
}

// Successful-publication fixtures own an explicit store and prove that success
// does not depend on writable global/default engine storage.
func isolateEvidencePublicationStorage(t *testing.T, cfg *config.Config) {
	t.Helper()
	cfg.Storage.DataDir = t.TempDir()
	blocked, err := blockedPublicationStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("UIAI_SCREENSHOT_DIR", blocked)
}

func TestRoutesDefaultEvidenceStoreIsAlwaysTestOwned(t *testing.T) {
	path := screenshotStoreDir()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	root := path
	if os.Getenv("UIAI_TEST_BLOCK_DEFAULT_EVIDENCE_STORE") == "1" {
		if st.IsDir() {
			t.Fatal("default-store sentinel is writable directory")
		}
		root = filepath.Dir(path)
	} else if !st.IsDir() {
		t.Fatal("isolated default store is not a directory")
	}
	if !strings.HasPrefix(filepath.Base(root), "uiai-routes-evidence-") {
		t.Fatal("inherited or installed default evidence store escaped isolation")
	}
}
