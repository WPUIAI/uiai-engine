// Command epwa-threat-scan executes the committed threat corpus through the
// built-in evidence inspector and writes the deterministic scan report used
// as the canonical CG-04 producer artifact. Any mismatch between observed
// and expected outcomes exits nonzero — the report is never allowed to
// silently describe a corpus it did not actually scan.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func main() {
	corpusPath := flag.String("corpus", filepath.Join("internal", "evidenceartifact", "testdata", "threat-corpus", "corpus.json"), "path to corpus.json")
	fixturesDir := flag.String("fixtures", filepath.Join("internal", "evidenceartifact", "testdata", "threat-corpus"), "fixture base directory")
	codeRef := flag.String("code-ref", "", "provenance ref of the producing commit (informational)")
	committedPath := flag.String("verify-against", "", "canonical report to verify against; run fails on any drift")
	output := flag.String("output", "", "write the report JSON here")
	flag.Parse()

	corpus, err := evidenceartifact.LoadThreatCorpus(*corpusPath)
	if err != nil {
		fail(err)
	}
	inspector := evidenceartifact.NewBuiltinInspector()
	report, err := evidenceartifact.RunThreatScan(corpus, *fixturesDir, inspector, *codeRef)
	if err != nil {
		fail(err)
	}
	if err := report.Validate(); err != nil {
		fail(err)
	}
	if *committedPath != "" {
		body, readErr := os.ReadFile(*committedPath)
		if readErr != nil {
			fail(readErr)
		}
		var committed evidenceartifact.ThreatScanReport
		if decodeErr := json.Unmarshal(body, &committed); decodeErr != nil {
			fail(decodeErr)
		}
		if verifyErr := evidenceartifact.VerifyThreatScanReport(report, committed); verifyErr != nil {
			fmt.Fprintln(os.Stderr, "drift against canonical report:", verifyErr)
			os.Exit(2)
		}
		fmt.Println("canonical scan report verified: all corpus entries match")
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err)
	}
	body = append(body, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(body)
		return
	}
	if err := os.WriteFile(*output, body, 0o600); err != nil {
		fail(err)
	}
	fmt.Fprintf(os.Stderr, "scan report written: %s (entries=%d matched=%d uncovered=%d unexpected=%d)\n", *output, report.Summary.Total, report.Summary.Matched, report.Summary.Uncovered, report.Summary.Unexpected)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "epwa-threat-scan:", err)
	os.Exit(1)
}
