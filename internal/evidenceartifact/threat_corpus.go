package evidenceartifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ThreatCorpusSchema     = "uiai.threat_corpus.v1"
	ThreatScanReportSchema = "uiai.threat_scan_report.v1"
)

// Threat defense states are typed and fail closed: uncovered classes remain
// visible in every scan report instead of being silently claimed as defended.
type ThreatDefenseState string

const (
	DefenseDefended           ThreatDefenseState = "defended"
	DefenseUncoveredHardening ThreatDefenseState = "uncovered_hardening"
)

type ThreatCorpusEntry struct {
	Class           string             `json:"class"`
	ID              string             `json:"id"`
	Description     string             `json:"description"`
	MediaType       string             `json:"media_type"`
	Fixture         string             `json:"fixture"`
	FixtureSHA256   string             `json:"fixture_sha256"`
	ExpectedOutcome string             `json:"expected_outcome"`
	DefenseState    ThreatDefenseState `json:"defense_state"`
	AccessClass     AccessClass        `json:"access_class,omitempty"`
	RedactionState  RedactionState     `json:"redaction_state,omitempty"`
}

type ThreatCorpus struct {
	Schema    string              `json:"schema"`
	PolicyRef string              `json:"policy_ref"`
	Entries   []ThreatCorpusEntry `json:"entries"`

	// FileSHA256 binds the report to the committed corpus.json artifact bytes,
	// not an in-memory re-marshal. It is empty for structs not loaded from disk.
	FileSHA256 string `json:"-"`
}

// ThreatScanReport is the deterministic, hash-bound producer artifact for the
// CG-04 join: one observed result per corpus entry, no timestamps, sorted
// order. The committed canonical report is re-verified by tests so drift
// fails closed.
type ThreatScanReport struct {
	Schema       string            `json:"schema"`
	CorpusSHA256 string            `json:"corpus_sha256"`
	CodeRef      string            `json:"code_ref"`
	Entries      []ThreatScanEntry `json:"entries"`
	UncoveredIDs []string          `json:"uncovered_ids"`
	Summary      ThreatScanSummary `json:"summary"`
}

type ThreatScanEntry struct {
	ID               string             `json:"id"`
	Class            string             `json:"class"`
	MediaType        string             `json:"media_type"`
	FixtureSHA256    string             `json:"fixture_sha256"`
	ExpectedOutcome  string             `json:"expected_outcome"`
	DefenseState     ThreatDefenseState `json:"defense_state"`
	ObservedOutcome  string             `json:"observed_outcome"`
	ObservedStatus   InspectionStatus   `json:"observed_status,omitempty"`
	ObservedFindings []string           `json:"observed_findings,omitempty"`
	Match            bool               `json:"match"`
}

type ThreatScanSummary struct {
	Total      int `json:"total"`
	Matched    int `json:"matched"`
	Uncovered  int `json:"uncovered"`
	Unexpected int `json:"unexpected"`
}

func LoadThreatCorpus(path string) (ThreatCorpus, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ThreatCorpus{}, fmt.Errorf("read threat corpus: %w", err)
	}
	var corpus ThreatCorpus
	if err := json.Unmarshal(body, &corpus); err != nil {
		return ThreatCorpus{}, fmt.Errorf("decode threat corpus: %w", err)
	}
	if err := corpus.Validate(); err != nil {
		return ThreatCorpus{}, err
	}
	corpus.FileSHA256 = textSHA256(string(body))
	return corpus, nil
}

func (c ThreatCorpus) Validate() error {
	if c.Schema != ThreatCorpusSchema {
		return fmt.Errorf("threat corpus schema: %w", ErrUnsafeContent)
	}
	if c.PolicyRef != StrictSecurityPolicyV1 {
		return fmt.Errorf("threat corpus policy: %w", ErrInspectionRequired)
	}
	if len(c.Entries) == 0 {
		return fmt.Errorf("threat corpus is empty: %w", ErrUnsafeContent)
	}
	seen := make(map[string]bool, len(c.Entries))
	for index, entry := range c.Entries {
		if entry.ID == "" || seen[entry.ID] {
			return fmt.Errorf("threat corpus entry %d identity: %w", index, ErrUnsafeContent)
		}
		seen[entry.ID] = true
		if entry.Fixture == "" || !validSHA256(entry.FixtureSHA256) {
			return fmt.Errorf("threat corpus entry %s fixture binding: %w", entry.ID, ErrUnsafeContent)
		}
		if entry.ExpectedOutcome == "" {
			return fmt.Errorf("threat corpus entry %s expected outcome: %w", entry.ID, ErrUnsafeContent)
		}
		switch entry.AccessClass {
		case AccessPrivateTeam, AccessPublicSafe, AccessLocal, AccessLAN, AccessTailnet, AccessUnlisted, "":
		default:
			return fmt.Errorf("threat corpus entry %s access class: %w", entry.ID, ErrUnsafeContent)
		}
		switch entry.RedactionState {
		case RedactionNone, RedactionRedacted, RedactionBlocked, RedactionPublicSafe, "":
		default:
			return fmt.Errorf("threat corpus entry %s redaction state: %w", entry.ID, ErrUnsafeContent)
		}
		if entry.AccessClass == "" {
			entry.AccessClass = AccessPrivateTeam
		}
		if entry.RedactionState == "" {
			entry.RedactionState = RedactionNone
		}
		if entry.DefenseState != DefenseDefended && entry.DefenseState != DefenseUncoveredHardening {
			return fmt.Errorf("threat corpus entry %s defense state %q: %w", entry.ID, entry.DefenseState, ErrUnsafeContent)
		}
		if entry.DefenseState == DefenseUncoveredHardening && !strings.HasPrefix(entry.ExpectedOutcome, "accepted") && !strings.HasPrefix(entry.ExpectedOutcome, "passed") {
			return fmt.Errorf("threat corpus entry %s uncovered defense must expect acceptance: %w", entry.ID, ErrUnsafeContent)
		}
	}
	return nil
}

// RunThreatScan executes every corpus entry through the built-in inspector
// against the staged fixture directory and produces the deterministic
// report. Fixture bytes are re-hashed before inspection; any drift fails the
// entry instead of scanning stale bytes.
// VerifyThreatScanReport compares a freshly scanned report against the
// committed canonical report. Only observable outcomes are compared — code
// refs are provenance metadata, not outcomes.
func VerifyThreatScanReport(live, committed ThreatScanReport) error {
	if committed.Schema != ThreatScanReportSchema {
		return fmt.Errorf("canonical report schema: %w", ErrUnsafeContent)
	}
	if live.CorpusSHA256 != committed.CorpusSHA256 {
		return fmt.Errorf("corpus hash drift: live %s canonical %s", live.CorpusSHA256, committed.CorpusSHA256)
	}
	if live.Summary != committed.Summary {
		return fmt.Errorf("summary drift: live %+v canonical %+v", live.Summary, committed.Summary)
	}
	if len(live.Entries) != len(committed.Entries) {
		return fmt.Errorf("entry count drift: live %d canonical %d", len(live.Entries), len(committed.Entries))
	}
	for index := range committed.Entries {
		want := committed.Entries[index]
		got := live.Entries[index]
		if got.ID != want.ID || got.ObservedOutcome != want.ObservedOutcome || got.Match != want.Match || !equalStrings(got.ObservedFindings, want.ObservedFindings) || got.ObservedStatus != want.ObservedStatus {
			return fmt.Errorf("entry %s drift: live %#v canonical %#v", want.ID, got, want)
		}
	}
	return nil
}

// Validate enforces the report contract: typed schema, hash-bound corpus,
// complete entry list, and a summary consistent with the entries. CodeRef is
// provenance metadata — the executable binding is the corpus hash plus the
// CI drift guard re-running the scan at the current commit.
func (r ThreatScanReport) Validate() error {
	if r.Schema != ThreatScanReportSchema {
		return fmt.Errorf("threat scan report schema: %w", ErrUnsafeContent)
	}
	if !validSHA256(r.CorpusSHA256) {
		return fmt.Errorf("threat scan report corpus binding: %w", ErrUnsafeContent)
	}
	if strings.TrimSpace(r.CodeRef) == "" {
		return fmt.Errorf("threat scan report code ref: %w", ErrUnsafeContent)
	}
	if len(r.Entries) != r.Summary.Total || len(r.UncoveredIDs) != r.Summary.Uncovered {
		return fmt.Errorf("threat scan report summary mismatch: %w", ErrUnsafeContent)
	}
	uncovered := make(map[string]bool, len(r.UncoveredIDs))
	for _, id := range r.UncoveredIDs {
		uncovered[id] = true
	}
	matched := 0
	unexpected := 0
	for _, entry := range r.Entries {
		if uncovered[entry.ID] != (entry.DefenseState == DefenseUncoveredHardening) {
			return fmt.Errorf("threat scan report uncovered list mismatch for %s: %w", entry.ID, ErrUnsafeContent)
		}
		if entry.Match {
			matched++
		} else {
			unexpected++
		}
	}
	if matched != r.Summary.Matched || unexpected != r.Summary.Unexpected {
		return fmt.Errorf("threat scan report totals mismatch: %w", ErrUnsafeContent)
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func RunThreatScan(corpus ThreatCorpus, fixturesDir string, inspector AssetInspector, codeRef string) (ThreatScanReport, error) {
	if inspector == nil || strings.TrimSpace(codeRef) == "" {
		return ThreatScanReport{}, ErrInspectionUnavailable
	}
	if corpus.FileSHA256 == "" {
		return ThreatScanReport{}, fmt.Errorf("threat corpus must be loaded from its committed artifact: %w", ErrUnsafeContent)
	}
	entries := append([]ThreatCorpusEntry(nil), corpus.Entries...)
	sort.Slice(entries, func(left, right int) bool { return entries[left].ID < entries[right].ID })
	report := ThreatScanReport{
		Schema:       ThreatScanReportSchema,
		CorpusSHA256: corpus.FileSHA256,
		CodeRef:      codeRef,
		Entries:      make([]ThreatScanEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		record := scanCorpusEntry(entry, fixturesDir, inspector)
		if !record.Match {
			report.Summary.Unexpected++
		} else {
			report.Summary.Matched++
		}
		if entry.DefenseState == DefenseUncoveredHardening {
			report.UncoveredIDs = append(report.UncoveredIDs, entry.ID)
			report.Summary.Uncovered++
		}
		report.Summary.Total++
		report.Entries = append(report.Entries, record)
	}
	return report, nil
}

func scanCorpusEntry(entry ThreatCorpusEntry, fixturesDir string, inspector AssetInspector) ThreatScanEntry {
	record := ThreatScanEntry{
		ID: entry.ID, Class: entry.Class, MediaType: entry.MediaType,
		FixtureSHA256: entry.FixtureSHA256, ExpectedOutcome: entry.ExpectedOutcome,
		DefenseState: entry.DefenseState, ObservedOutcome: "scan_failed",
	}
	assetSHA, size, err := stageFixture(filepath.Join(fixturesDir, entry.Fixture), entry.FixtureSHA256)
	if err != nil {
		return record
	}
	path := filepath.Join(fixturesDir, entry.Fixture)
	request := InspectionRequest{
		Path:     path,
		Asset:    Asset{AssetID: entry.ID, MediaType: entry.MediaType, ByteSize: size, SHA256: assetSHA},
		Security: Security{PolicyRef: StrictSecurityPolicyV1},
		Policy:   Policy{AccessClass: entry.AccessClass, RedactionState: entry.RedactionState},
	}
	inspection, inspectErr := inspector.Inspect(context.Background(), request)
	switch {
	case inspectErr == nil:
		record.ObservedStatus = inspection.Status
		record.ObservedFindings = append([]string(nil), inspection.FindingCodes...)
		record.ObservedOutcome = string(inspection.Status)
		if entry.DefenseState == DefenseUncoveredHardening && (record.ObservedOutcome == "passed" || record.ObservedOutcome == "passed_with_findings") && entry.ExpectedOutcome == "accepted_uncovered" {
			record.ObservedOutcome = "accepted_uncovered"
		}
	case errors.Is(inspectErr, ErrSensitiveContent):
		record.ObservedOutcome = "rejected:sensitive_content"
	case errors.Is(inspectErr, ErrMediaTypeMismatch):
		record.ObservedOutcome = "rejected:mime_mismatch"
	case errors.Is(inspectErr, ErrSanitizerRequired):
		record.ObservedOutcome = "rejected:sanitizer_required"
	case errors.Is(inspectErr, ErrUnsafeContent):
		record.ObservedOutcome = "rejected:unsafe_media"
	default:
		record.ObservedOutcome = "scan_failed"
	}
	record.Match = record.ObservedOutcome == entry.ExpectedOutcome
	return record
}

func stageFixture(path, expectedSHA string) (string, int64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("read fixture: %w", err)
	}
	sum := textSHA256(string(body))
	if sum != expectedSHA {
		return "", 0, fmt.Errorf("fixture hash drift: %w", ErrInspectionFailed)
	}
	return sum, int64(len(body)), nil
}
