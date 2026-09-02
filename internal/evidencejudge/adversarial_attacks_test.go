package evidencejudge

import (
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
