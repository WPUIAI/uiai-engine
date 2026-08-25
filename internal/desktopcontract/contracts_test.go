package desktopcontract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const fixtureDir = "../../tests/fixtures/desktop-presentation"

func decodeStrict[T any](t *testing.T, name string) T {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return value
}

func TestSharedFixtureBundleValidates(t *testing.T) {
	bundle := decodeStrict[FixtureBundle](t, "valid-contracts.json")
	if err := ValidateFixtureBundle(bundle); err != nil {
		t.Fatal(err)
	}
	if got, want := len(SchemaIDs), 7; got != want {
		t.Fatalf("schema count=%d want=%d", got, want)
	}

	menubarManifest := decodeStrict[AppManifest](t, "focusa-app-manifest.valid.json")
	if err := ValidateAppManifest(menubarManifest); err != nil {
		t.Fatalf("Focusa Menubar manifest: %v", err)
	}
	if menubarManifest.App != "focusa-menubar" {
		t.Fatalf("manifest app=%q want focusa-menubar", menubarManifest.App)
	}
}

func TestHandoffInvalidFixturesFailClosed(t *testing.T) {
	t.Run("unknown field rejects secret", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(fixtureDir, "handoff-secret.invalid.json"))
		if err != nil {
			t.Fatal(err)
		}
		var intent AppHandoffIntent
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&intent); err == nil {
			t.Fatal("expected secret-bearing unknown field to be rejected")
		}
	})

	for _, name := range []string{
		"handoff-raw-path.invalid.json",
		"handoff-private-url.invalid.json",
		"handoff-unknown-route.invalid.json",
	} {
		t.Run(name, func(t *testing.T) {
			intent := decodeStrict[AppHandoffIntent](t, name)
			if err := ValidateHandoffIntent(intent); err == nil {
				t.Fatalf("expected %s to fail validation", name)
			}
		})
	}
}

func TestOpaqueRefRejectsPathsURLsQueriesAndFragments(t *testing.T) {
	for _, value := range []string{
		"/tmp/private",
		"https://example.com",
		"session?token=secret",
		"session#fragment",
		`C:\\Users\\operator`,
		"contains space",
	} {
		if err := ValidateOpaqueRef(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
