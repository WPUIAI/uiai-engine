package evidencejudge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAdversarialGoldenRoundTripAndDeterminism(t *testing.T) {
	corpusPath := filepath.Join("testdata", "adversarial-corpus.golden.json")
	resultsPath := filepath.Join("testdata", "adversarial-results.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		writeAdversarialGoldens(t, corpusPath, resultsPath)
	}
	corpusBody, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := ParseAdversarialCorpus(corpusBody)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyAdversarialCorpusSHA256(corpus, corpus.CorpusSHA256); err != nil {
		t.Fatal(err)
	}
	canonical, err := CanonicalAdversarialCorpusBytes(corpus)
	if err != nil {
		t.Fatal(err)
	}
	parsedAgain, err := ParseAdversarialCorpus(canonical)
	if err != nil || !reflect.DeepEqual(corpus, parsedAgain) {
		t.Fatalf("canonical round trip drift: %v", err)
	}

	var golden adversarialGoldenResults
	body, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := strictJSON(body, &golden); err != nil {
		t.Fatal(err)
	}
	actual := evaluateCorpus(t, corpus)
	if !reflect.DeepEqual(actual, golden.Evaluations) {
		t.Fatal("evaluation golden drift")
	}
	baseline, err := BuildCalibrationResult(corpus, actual, testCalibrationCapability(), testAdversarialNow())
	if err != nil || !reflect.DeepEqual(baseline, golden.Baseline) {
		t.Fatalf("baseline golden drift: %v", err)
	}
	candidateEvaluations := cloneJSON(t, actual)
	candidateEvaluations[0].Passed = false
	candidateEvaluations[0].FailureCode = "synthetic_threshold_probe"
	candidate, err := BuildCalibrationResult(corpus, candidateEvaluations, testCalibrationCapability(), testAdversarialNow())
	if err != nil || !reflect.DeepEqual(candidate, golden.Candidate) {
		t.Fatalf("candidate golden drift: %v", err)
	}
	drift, err := CompareCalibrationResults(baseline, candidate, testDriftPolicy())
	if err != nil || !reflect.DeepEqual(drift, golden.Drift) {
		t.Fatalf("drift golden drift: %v", err)
	}

	original := cloneJSON(t, corpus)
	for run := 0; run < 30; run++ {
		permuted := cloneJSON(t, corpus)
		switch run % 6 {
		case 1:
			reverseCases(permuted.Cases)
		case 2:
			rotateCases(permuted.Cases, 3)
		case 3:
			for i := range permuted.Cases {
				reverseAtoms(permuted.Cases[i].Expected.Atoms)
				reverseModalities(permuted.Cases[i].Expected.Modalities)
			}
		case 4:
			reverseCases(permuted.Cases)
			for i := range permuted.Cases {
				reverseStrings(permuted.Cases[i].Expected.CitationRefs)
				reverseVerdicts(permuted.Cases[i].Protected.AllowedVerdicts)
			}
		case 5:
			rotateCases(permuted.Cases, 7)
			for i := range permuted.Cases {
				reverseAtoms(permuted.Cases[i].Expected.Atoms)
				reverseModalities(permuted.Cases[i].Expected.Modalities)
				reverseStrings(permuted.Cases[i].Expected.CitationRefs)
				reverseVerdicts(permuted.Cases[i].Protected.AllowedVerdicts)
			}
		}
		permutedBefore := cloneJSON(t, permuted)
		digest, err := ComputeAdversarialCorpusSHA256(permuted)
		if err != nil || digest != corpus.CorpusSHA256 {
			t.Fatalf("run %d corpus permutation drift: %v %s", run, err, digest)
		}
		if !reflect.DeepEqual(permuted, permutedBefore) {
			t.Fatalf("run %d digest mutated unsorted caller input", run)
		}
		result, err := BuildCalibrationResult(corpus, actual, testCalibrationCapability(), testAdversarialNow())
		if err != nil || result.CalibrationSHA256 != baseline.CalibrationSHA256 {
			t.Fatalf("run %d calibration drift: %v", run, err)
		}
	}
	if !reflect.DeepEqual(corpus, original) {
		t.Fatal("public calls mutated corpus")
	}
}

type adversarialGoldenResults struct {
	Schema      string            `json:"schema"`
	Evaluations []CaseEvaluation  `json:"evaluations"`
	Baseline    CalibrationResult `json:"baseline"`
	Candidate   CalibrationResult `json:"candidate"`
	Drift       DriftReport       `json:"drift"`
}

func reverseCases(v []AdversarialCase) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func rotateCases(v []AdversarialCase, offset int) {
	if len(v) == 0 {
		return
	}
	offset %= len(v)
	rotated := append(append([]AdversarialCase(nil), v[offset:]...), v[:offset]...)
	copy(v, rotated)
}
func reverseAtoms(v []AdversarialAtomExpectation) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func reverseModalities(v []AdversarialModalityExpectation) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func reverseStrings(v []string) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func reverseVerdicts(v []Verdict) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}

func writeAdversarialGoldens(t *testing.T, corpusPath, resultsPath string) {
	t.Helper()
	corpus := testAdversarialCorpus(t)
	evaluations := evaluateCorpus(t, corpus)
	baseline, err := BuildCalibrationResult(corpus, evaluations, testCalibrationCapability(), testAdversarialNow())
	if err != nil {
		t.Fatal(err)
	}
	candidateEvaluations := cloneJSON(t, evaluations)
	candidateEvaluations[0].Passed = false
	candidateEvaluations[0].FailureCode = "synthetic_threshold_probe"
	candidate, err := BuildCalibrationResult(corpus, candidateEvaluations, testCalibrationCapability(), testAdversarialNow())
	if err != nil {
		t.Fatal(err)
	}
	drift, err := CompareCalibrationResults(baseline, candidate, testDriftPolicy())
	if err != nil {
		t.Fatal(err)
	}
	writePrettyJSON(t, corpusPath, corpus)
	writePrettyJSON(t, resultsPath, adversarialGoldenResults{Schema: "uiai.evidence_judge_adversarial_results_fixture.v1", Evaluations: evaluations, Baseline: baseline, Candidate: candidate, Drift: drift})
}
func writePrettyJSON(t *testing.T, path string, value any) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
