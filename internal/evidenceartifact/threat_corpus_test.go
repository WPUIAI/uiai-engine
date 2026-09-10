package evidenceartifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestThreatCorpusContract verifies the committed corpus is typed, complete,
// and hash-bound to its fixtures. Any drift in the corpus or fixtures fails
// the producer test before an independent reviewer ever consumes it.
func TestThreatCorpusContract(t *testing.T) {
	corpus, err := LoadThreatCorpus(filepath.Join("testdata", "threat-corpus", "corpus.json"))
	if err != nil {
		t.Fatalf("LoadThreatCorpus() error = %v", err)
	}
	if len(corpus.Entries) < 20 {
		t.Fatalf("corpus entries = %d, want >= 20", len(corpus.Entries))
	}
	uncovered := 0
	seen := map[string]bool{}
	for _, entry := range corpus.Entries {
		if seen[entry.ID] {
			t.Fatalf("duplicate entry %s", entry.ID)
		}
		seen[entry.ID] = true
		body, err := os.ReadFile(filepath.Join("testdata", "threat-corpus", entry.Fixture))
		if err != nil {
			t.Fatalf("fixture %s: %v", entry.ID, err)
		}
		if textSHA256(string(body)) != entry.FixtureSHA256 {
			t.Fatalf("fixture hash drift for %s", entry.ID)
		}
		if entry.DefenseState == DefenseUncoveredHardening {
			uncovered++
		}
	}
	if uncovered == 0 {
		t.Fatal("corpus must document uncovered hardening classes")
	}
}

// TestThreatScanReportCurrent is the drift guard: it re-runs the full scan
// through the built-in inspector and requires the result to equal the
// committed canonical scan report byte-for-byte. A sanitizer change or
// fixture edit must be accompanied by an explicitly regenerated report.
func TestThreatScanReportCurrent(t *testing.T) {
	corpus, err := LoadThreatCorpus(filepath.Join("testdata", "threat-corpus", "corpus.json"))
	if err != nil {
		t.Fatalf("LoadThreatCorpus() error = %v", err)
	}
	canonical, err := os.ReadFile(filepath.Join("testdata", "threat-corpus", "scan-report.json"))
	if err != nil {
		t.Fatalf("read canonical scan report: %v", err)
	}
	var committed ThreatScanReport
	if err := json.Unmarshal(canonical, &committed); err != nil {
		t.Fatalf("decode canonical scan report: %v", err)
	}
	if err := committed.Validate(); err != nil {
		t.Fatalf("canonical scan report invalid: %v", err)
	}
	inspector := NewBuiltinInspector()
	report, err := RunThreatScan(corpus, filepath.Join("testdata", "threat-corpus"), inspector, "ci-drift-guard")
	if err != nil {
		t.Fatalf("RunThreatScan() error = %v", err)
	}
	if report.CorpusSHA256 != committed.CorpusSHA256 {
		t.Fatalf("corpus hash drift: live %s canonical %s", report.CorpusSHA256, committed.CorpusSHA256)
	}
	if report.Summary != committed.Summary {
		t.Fatalf("summary drift: live %+v canonical %+v", report.Summary, committed.Summary)
	}
	for index, entry := range report.Entries {
		if index >= len(committed.Entries) {
			t.Fatalf("live report has more entries than canonical")
		}
		committedEntry := committed.Entries[index]
		if entry.ID != committedEntry.ID || entry.ObservedOutcome != committedEntry.ObservedOutcome || entry.Match != committedEntry.Match || !equalStrings(entry.ObservedFindings, committedEntry.ObservedFindings) {
			t.Fatalf("entry %s drift: live %#v canonical %#v", entry.ID, entry, committedEntry)
		}
	}
	for _, entry := range committed.Entries {
		if !entry.Match {
			t.Fatalf("canonical report contains unmatched entry %s", entry.ID)
		}
	}
}
