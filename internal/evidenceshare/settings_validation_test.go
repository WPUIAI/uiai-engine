package evidenceshare

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"testing"
)

func TestSettingsValidateCanonicalFieldsAndValues(t *testing.T) {
	if err := validatePatch(DefaultSettings()); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := validatePatch(decoded); err != nil {
		t.Fatal(err)
	}
	for _, patch := range []map[string]any{
		{"image": map[string]any{"quality": "82"}},
		{"image": map[string]any{"quality": -1}},
		{"image": map[string]any{"quality": 101}},
		{"image": map[string]any{"quality": 1.5}},
		{"image": map[string]any{"quality": math.NaN()}},
		{"image": map[string]any{"format": "svg"}},
		{"image": map[string]any{"max_width": 0}},
		{"enablement": map[string]any{"enabled": "true"}},
		{"enablement": map[string]any{"read_only": false}},
		{"access": map[string]any{"unrecognized_field": "not persisted"}},
		{"storage": map[string]any{"location_template": "../../outside"}},
		{"storage": map[string]any{"location_template": "/absolute"}},
		{"storage": map[string]any{"location_template": "{unknown}"}},
		{"storage": map[string]any{"quota_headroom_percent": 101}},
	} {
		if err := validatePatch(patch); !errors.Is(err, ErrSettingsInvalid) {
			t.Fatalf("invalid patch accepted: %v", patch)
		}
	}
}

func TestSettingsScopeWireCompatibility(t *testing.T) {
	want := SettingsScope{ProjectRef: "project:one", WorkstreamRef: "workstream:one"}
	for _, body := range []string{`{"project_ref":"project:one","workstream_ref":"workstream:one"}`, `{"ProjectRef":"project:one","WorkstreamRef":"workstream:one"}`} {
		var got SettingsScope
		if err := json.Unmarshal([]byte(body), &got); err != nil || got != want {
			t.Fatalf("scope changed: %#v, %v", got, err)
		}
	}
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]string
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["project_ref"] != want.ProjectRef || wire["workstream_ref"] != want.WorkstreamRef || len(wire) != 2 {
		t.Fatalf("wrong consumer fields: %s", body)
	}
	for _, body := range []string{`null`, `{"project_ref":null}`, `{"project":"wrong-field"}`, `{"project_ref":"one","ProjectRef":"two"}`} {
		var scope SettingsScope
		if err := json.Unmarshal([]byte(body), &scope); err == nil {
			t.Fatalf("ambiguous scope accepted: %s", body)
		}
	}
}

func TestSettingsResetRevisionAndFailedPersistence(t *testing.T) {
	store, err := NewSettingsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := SettingsScope{ProjectRef: "project:one"}
	patch := map[string]any{"image": map[string]any{"quality": 61}}
	if _, err := store.Update(scope, 0, patch); err != nil {
		t.Fatal(err)
	}
	original := store.path
	store.path = t.TempDir() // Publication onto a directory must fail.
	if _, err := store.Update(scope, 1, map[string]any{"image": map[string]any{"quality": 62}}); err == nil {
		t.Fatal("publication failure accepted")
	}
	if err := store.Reset(scope, 1); err == nil {
		t.Fatal("reset publication failure accepted")
	}
	result := store.Effective(scope)
	if result.Revision != 1 || result.Values["image"].(map[string]any)["quality"] != 61 {
		t.Fatal("failed publication mutated live settings")
	}
	store.path = original
	if err := store.Reset(scope, 1); err != nil {
		t.Fatal(err)
	}
	if store.Effective(scope).Revision != 2 {
		t.Fatal("reset reused an old revision")
	}
	if _, err := store.Update(scope, 1, patch); !errors.Is(err, ErrSettingsConflict) {
		t.Fatal("stale writer accepted after reset")
	}
	if _, err := store.Update(SettingsScope{WorkstreamRef: "orphan"}, 0, patch); !errors.Is(err, ErrSettingsInvalid) {
		t.Fatal("orphan workstream accepted")
	}
}

func TestMalformedSettingsDocumentsFailClosed(t *testing.T) {
	for _, body := range []string{`{}`, `{"schema":"uiai.evidence_share_settings.v1","records":[{"values":{"image":{"quality":61}}}]}`, `{"schema":"uiai.evidence_share_settings.v1","records":[{"scope":{},"values":{"image":{"quality":"bad"}}}]}`} {
		dir := t.TempDir()
		store, err := NewSettingsStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(store.path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewSettingsStore(dir); !errors.Is(err, ErrSettingsInvalid) {
			t.Fatalf("malformed settings accepted: %s", body)
		}
	}
}

func TestSettingsLegacyRecordsKeepTheirScope(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSettingsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"schema":"uiai.evidence_share_settings.v1","records":[{"scope":{"ProjectRef":"project:legacy","WorkstreamRef":"workstream:legacy"},"revision":1,"values":{"image":{"quality":61}}}]}`
	if err := os.WriteFile(store.path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewSettingsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	result := reopened.Effective(SettingsScope{ProjectRef: "project:legacy", WorkstreamRef: "workstream:legacy"})
	if result.Revision != 1 || result.Values["image"].(map[string]any)["quality"] != float64(61) {
		t.Fatal("legacy scoped override lost")
	}
	if reopened.Effective(SettingsScope{}).Values["image"].(map[string]any)["quality"] != 82 {
		t.Fatal("legacy override became global")
	}
}
