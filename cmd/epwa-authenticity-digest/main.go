package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func main() {
	runs := flag.Int("runs", 30, "number of clause-battery runs (1..1000)")
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
		var prior evidenceartifact.AuthenticityDigestReport
		if err := json.Unmarshal(body, &prior); err != nil {
			fmt.Fprintf(os.Stderr, "decode prior report: %v\n", err)
			os.Exit(1)
		}
		if err := evidenceartifact.VerifyAuthenticityDigestReport(prior, *runs); err != nil {
			fmt.Fprintf(os.Stderr, "verify-against: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("verified against %s (runs=%d)\n", prior.DigestSHA256[:16], *runs)
		return
	}

	report, err := evidenceartifact.RunAuthenticityDigest(*runs, *codeRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "authenticity digest: %v\n", err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, encoded, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(string(encoded))
}
