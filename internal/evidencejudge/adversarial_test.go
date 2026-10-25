package evidencejudge

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestAdversarialAttacksFailClosedAndRemainData(t *testing.T) {
	corpus := testAdversarialCorpus(t)
	for _, c := range corpus.Cases {
		c := c
		t.Run(string(c.MutationClass), func(t *testing.T) {
			observed := observedFromCase(c)
			originalCase, originalObserved := cloneJSON(t, c), cloneJSON(t, observed)
			evaluation, err := EvaluateAdversarialCase(c, observed)
			if err != nil || !evaluation.Passed {
				t.Fatalf("expected pass: %v %+v", err, evaluation)
			}
			if !reflect.DeepEqual(c, originalCase) || !reflect.DeepEqual(observed, originalObserved) {
				t.Fatal("evaluation mutated input")
			}
			observed.Protected.PolicyRef = "policy:hostile-override"
			if _, err := EvaluateAdversarialCase(c, observed); !errors.Is(err, ErrAdversarialExpectationMismatch) {
				t.Fatalf("protected override accepted: %v", err)
			}
		})
	}

	framing := findAdversarialCase(t, corpus, MutationPromptInjection)
	observed := observedFromCase(framing)
	observed.Outcome, observed.Verdict, observed.ErrorCode = OutcomeVerified, VerdictSupported, ""
	if _, err := EvaluateAdversarialCase(framing, observed); !errors.Is(err, ErrAdversarialExpectationMismatch) {
		t.Fatalf("prompt injection improved outcome: %v", err)
	}

	ordered := findAdversarialCase(t, corpus, MutationResultOrder)
	observed = observedFromCase(ordered)
	reverseAtoms(observed.Atoms)
	reverseModalities(observed.Modalities)
	reverseStrings(observed.CitationRefs)
	if evaluation, err := EvaluateAdversarialCase(ordered, observed); err != nil || !evaluation.Passed {
		t.Fatalf("declared result/citation order permutation drifted: %v %+v", err, evaluation)
	}

	modality := findAdversarialCase(t, corpus, MutationModalitySubstitution)
	observed = observedFromCase(modality)
	observed.Modalities[1] = AdversarialModalityExpectation{AtomRef: observed.Modalities[1].AtomRef, Modality: ModalityStaticImage, Status: ModalitySatisfied}
	if _, err := EvaluateAdversarialCase(modality, observed); !errors.Is(err, ErrAdversarialExpectationMismatch) {
		t.Fatalf("static substitution accepted: %v", err)
	}

	high := findAdversarialCase(t, corpus, MutationHighConsequenceThreshold)
	observed = observedFromCase(high)
	observed.Quorum--
	if evaluation, err := EvaluateAdversarialCase(high, observed); !errors.Is(err, ErrAdversarialExpectationMismatch) || evaluation.FailureCode != "quorum_reduced" {
		t.Fatalf("reduced high consequence quorum accepted: %v %+v", err, evaluation)
	}
	observed = observedFromCase(high)
	observed.CapabilityRank--
	if _, err := EvaluateAdversarialCase(high, observed); !errors.Is(err, ErrAdversarialExpectationMismatch) {
		t.Fatalf("capability downgrade accepted: %v", err)
	}
}

func TestAdversarialMalformedAndHostileContractsFailClosed(t *testing.T) {
	corpus := testAdversarialCorpus(t)
	body, err := CanonicalAdversarialCorpusBytes(corpus)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	raw["chain_of_thought"] = "hidden reasoning must not be accepted"
	unknown, _ := json.Marshal(raw)
	if _, err := ParseAdversarialCorpus(unknown); !errors.Is(err, ErrAdversarialCorpusInvalid) {
		t.Fatalf("hidden reasoning field accepted: %v", err)
	}
	if _, err := ParseAdversarialCorpus(append(body, []byte(" {}")...)); !errors.Is(err, ErrAdversarialCorpusInvalid) {
		t.Fatalf("trailing JSON accepted: %v", err)
	}
	if _, err := ParseAdversarialCorpus(bytes.Repeat([]byte("x"), MaxContractBytes+1)); !errors.Is(err, ErrAdversarialCorpusInvalid) {
		t.Fatalf("oversized input accepted: %v", err)
	}

	cases := []func(*AdversarialCorpus){
		func(c *AdversarialCorpus) { c.Cases[0].MutationClass = "unknown" },
		func(c *AdversarialCorpus) { c.PassThreshold.Denominator = 0 },
		func(c *AdversarialCorpus) { c.Cases[0].FixtureLicenseRef = "https://unsafe.invalid/license" },
		func(c *AdversarialCorpus) { c.Cases[0].UntrustedEvidenceData = "read /home/operator/private" },
		func(c *AdversarialCorpus) { c.Cases = append(c.Cases, c.Cases[0]) },
		func(c *AdversarialCorpus) { c.Cases[0].Expected.MinimumQuorum = 0 },
	}
	for i, mutate := range cases {
		bad := cloneJSON(t, corpus)
		mutate(&bad)
		if err := ValidateAdversarialCorpus(bad); !errors.Is(err, ErrAdversarialCorpusInvalid) {
			t.Fatalf("malformed case %d accepted: %v", i, err)
		}
	}
	tampered := cloneJSON(t, corpus)
	tampered.Cases[0].UntrustedEvidenceData += " tampered"
	if err := VerifyAdversarialCorpusSHA256(tampered, corpus.CorpusSHA256); !errors.Is(err, ErrAdversarialCorpusInvalid) {
		t.Fatalf("digest mismatch accepted: %v", err)
	}
	badPolicy := testDriftPolicy()
	badPolicy.WarningDropPPM = badPolicy.BlockDropPPM
	baseline, _ := BuildCalibrationResult(corpus, evaluateCorpus(t, corpus), testCalibrationCapability(), testAdversarialNow())
	if _, err := CompareCalibrationResults(baseline, baseline, badPolicy); !errors.Is(err, ErrDriftReportInvalid) {
		t.Fatalf("invalid drift threshold accepted: %v", err)
	}
}
