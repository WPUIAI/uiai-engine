package evidenceshare

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (scope SettingsScope) Validate() error {
	if scope.WorkstreamRef != "" && scope.ProjectRef == "" {
		return fmt.Errorf("%w: workstream scope requires a project", ErrSettingsInvalid)
	}
	for _, ref := range []string{scope.ProjectRef, scope.WorkstreamRef} {
		if len(ref) > 4096 || strings.ContainsAny(ref, "\x00\r\n") || strings.TrimSpace(ref) != ref {
			return fmt.Errorf("%w: malformed scope reference", ErrSettingsInvalid)
		}
	}
	return nil
}

func (record *SettingsRecord) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if _, exists := fields["scope"]; !exists {
		return fmt.Errorf("%w: settings record requires explicit scope", ErrSettingsInvalid)
	}
	type wireRecord SettingsRecord
	var value wireRecord
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*record = SettingsRecord(value)
	return nil
}

// Older settings documents used Go field names. Accept those on read without
// turning existing project/workstream overrides into global settings; writes use
// the canonical snake-case wire fields. Ambiguous migrations fail closed.
func (scope *SettingsScope) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("%w: null settings scope", ErrSettingsInvalid)
	}
	for name, value := range fields {
		switch name {
		case "project_ref", "workstream_ref", "ProjectRef", "WorkstreamRef":
		default:
			return fmt.Errorf("%w: unknown settings scope field %s", ErrSettingsInvalid, name)
		}
		if strings.TrimSpace(string(value)) == "null" {
			return fmt.Errorf("%w: null settings scope field", ErrSettingsInvalid)
		}
	}
	var wire struct {
		Project          *string `json:"project_ref"`
		Workstream       *string `json:"workstream_ref"`
		LegacyProject    *string `json:"ProjectRef"`
		LegacyWorkstream *string `json:"WorkstreamRef"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.Project != nil && wire.LegacyProject != nil && *wire.Project != *wire.LegacyProject {
		return fmt.Errorf("%w: conflicting project scope fields", ErrSettingsInvalid)
	}
	if wire.Workstream != nil && wire.LegacyWorkstream != nil && *wire.Workstream != *wire.LegacyWorkstream {
		return fmt.Errorf("%w: conflicting workstream scope fields", ErrSettingsInvalid)
	}
	*scope = SettingsScope{}
	if wire.LegacyProject != nil {
		scope.ProjectRef = *wire.LegacyProject
	}
	if wire.LegacyWorkstream != nil {
		scope.WorkstreamRef = *wire.LegacyWorkstream
	}
	if wire.Project != nil {
		scope.ProjectRef = *wire.Project
	}
	if wire.Workstream != nil {
		scope.WorkstreamRef = *wire.Workstream
	}
	return scope.Validate()
}
