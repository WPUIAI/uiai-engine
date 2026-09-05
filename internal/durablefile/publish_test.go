package durablefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationAndReplacement(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "published")
	for _, text := range []string{"first", "replacement"} {
		source := filepath.Join(root, "staged")
		f, err := os.Create(source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.WriteString(text); err != nil {
			t.Fatal(err)
		}
		if err = f.Sync(); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		if err = Rename(source, target); err != nil {
			t.Fatal(err)
		}
		if err = SyncDirectory(root); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(target)
		if err != nil || string(body) != text {
			t.Fatalf("body=%q err=%v", body, err)
		}
	}
	if err := SyncDirectory(target); err == nil {
		t.Fatal("regular file accepted as directory")
	}
	if err := SyncDirectory(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing directory accepted")
	}
	if err := Rename(filepath.Join(root, "missing"), target); err == nil {
		t.Fatal("missing source accepted")
	}
}

func TestDirectoryPublication(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "stage")
	target := filepath.Join(root, "store")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Rename(source, target); err != nil {
		t.Fatal(err)
	}
	if err := SyncDirectory(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory publication: %v", err)
	}
}
