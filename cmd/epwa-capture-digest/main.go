package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func main() {
	runs := flag.Int("runs", 30, "number of assembly runs (1..1000)")
	codeRef := flag.String("code-ref", "", "code reference recorded in the report")
	output := flag.String("output", "", "write canonical report JSON to this path")
	verifyAgainst := flag.String("verify-against", "", "verify output matches a prior report and exit")
	flag.Parse()

	if *verifyAgainst != "" {
		body, err := os.ReadFile(*verifyAgainst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read prior report: %v\n", err)
			os.Exit(1)
		}
		var prior evidenceartifact.CaptureDigestReport
		if err := json.Unmarshal(body, &prior); err != nil {
			fmt.Fprintf(os.Stderr, "decode prior report: %v\n", err)
			os.Exit(1)
		}
		if err := evidenceartifact.VerifyCaptureDigestReport(prior, *runs); err != nil {
			fmt.Fprintf(os.Stderr, "verify-against: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("verified against %s (runs=%d)\n", prior.DigestSHA256[:16], *runs)
		return
	}

	report, err := evidenceartifact.RunCaptureDigest(*runs, *codeRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture digest: %v\n", err)
		os.Exit(1)
	}
	if err := report.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "report invalid: %v\n", err)
		os.Exit(1)
	}
	canonical, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal report: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, canonical, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("capture digest written: %s (runs=%d digest=%s all_identical=%v)\n",
		*output, report.Runs, report.DigestSHA256[:16], report.AllIdentical)
}
