package evidenceshare

import (
	"errors"
	"testing"
)

func TestSettingsInheritancePersistenceAndConflict(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSettingsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	global, err := store.Update(SettingsScope{}, 0, map[string]any{"presentation": map[string]any{"theme": "dark"}})
	if err != nil {
		t.Fatal(err)
	}
	if global.Values["presentation"].(map[string]any)["theme"] != "dark" {
		t.Fatal("global override missing")
	}
	project, err := store.Update(SettingsScope{ProjectRef: "project:homepage"}, 0, map[string]any{"image": map[string]any{"quality": 91}})
	if err != nil {
		t.Fatal(err)
	}
	if project.Values["presentation"].(map[string]any)["theme"] != "dark" {
		t.Fatal("inherited theme missing")
	}
	if project.Values["image"].(map[string]any)["quality"] != 91 {
		t.Fatal("project quality missing")
	}
	if _, err := store.Update(SettingsScope{ProjectRef: "project:homepage"}, 0, map[string]any{"image": map[string]any{"quality": 90}}); !errors.Is(err, ErrSettingsConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	reloaded, err := NewSettingsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Effective(SettingsScope{ProjectRef: "project:homepage"}).Revision != 1 {
		t.Fatal("revision did not persist")
	}
}

func TestSettingsRejectUnknownTopLevel(t *testing.T) {
	store, err := NewSettingsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Preview(SettingsScope{}, map[string]any{"secrets": map[string]any{"token": "x"}}); !errors.Is(err, ErrSettingsInvalid) {
		t.Fatalf("expected invalid settings, got %v", err)
	}
}
