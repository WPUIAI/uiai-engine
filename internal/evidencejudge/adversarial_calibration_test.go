package evidencejudge

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCalibrationThresholdCompletenessAndDrift(t *testing.T) {
	corpus := testAdversarialCorpus(t)
	evaluations := evaluateCorpus(t, corpus)
	capability := testCalibrationCapability()
	originalCorpus, originalEvaluations, originalCapability := cloneJSON(t, corpus), cloneJSON(t, evaluations), cloneJSON(t, capability)
	baseline, err := BuildCalibrationResult(corpus, evaluations, capability, testAdversarialNow())
	if err != nil || baseline.Status != CalibrationPassed || baseline.Passed != 17 || len(baseline.MutationCounts) != 17 {
		t.Fatalf("baseline calibration: %v %+v", err, baseline)
	}

	exact := cloneJSON(t, evaluations)
	exact[0].Passed = false
	exact[0].FailureCode = "threshold_probe"
	candidate, err := BuildCalibrationResult(corpus, exact, capability, testAdversarialNow())
	if err != nil || candidate.Status != CalibrationPassed || candidate.Passed != 16 {
		t.Fatalf("exact threshold did not pass: %v %+v", err, candidate)
	}
	alternate := cloneJSON(t, evaluations)
	alternate[1].Passed = false
	alternate[1].FailureCode = "threshold_probe"
	alternateResult, err := BuildCalibrationResult(corpus, alternate, capability, testAdversarialNow())
	if err != nil || alternateResult.CalibrationID == candidate.CalibrationID {
		t.Fatalf("distinct failed cases shared calibration identity: %v %+v", err, alternateResult)
	}
	below := cloneJSON(t, exact)
	below[1].Passed = false
	below[1].FailureCode = "threshold_probe"
	failed, err := BuildCalibrationResult(corpus, below, capability, testAdversarialNow())
	if !errors.Is(err, ErrCalibrationThresholdUnmet) || failed.Status != CalibrationFailed || failed.Passed != 15 {
		t.Fatalf("below threshold did not fail exactly: %v %+v", err, failed)
	}

	if _, err := BuildCalibrationResult(corpus, evaluations[:len(evaluations)-1], capability, testAdversarialNow()); !errors.Is(err, ErrCalibrationResultInvalid) {
		t.Fatalf("missing case accepted: %v", err)
	}
	duplicate := append(cloneJSON(t, evaluations), evaluations[0])
	if _, err := BuildCalibrationResult(corpus, duplicate, capability, testAdversarialNow()); !errors.Is(err, ErrCalibrationResultInvalid) {
		t.Fatalf("duplicate case accepted: %v", err)
	}
	unexpected := cloneJSON(t, evaluations)
	unexpected[0].CaseID = "case:unexpected"
	if _, err := BuildCalibrationResult(corpus, unexpected, capability, testAdversarialNow()); !errors.Is(err, ErrCalibrationResultInvalid) {
		t.Fatalf("unexpected case accepted: %v", err)
	}
	expired := capability
	expired.ValidUntil = testAdversarialNow()
	if _, err := BuildCalibrationResult(corpus, evaluations, expired, testAdversarialNow()); !errors.Is(err, ErrCalibrationResultInvalid) {
		t.Fatalf("expired capability accepted: %v", err)
	}
	bounded := capability
	bounded.ValidUntil = testAdversarialNow().Add(time.Hour)
	boundedResult, err := BuildCalibrationResult(corpus, evaluations, bounded, testAdversarialNow())
	if err != nil || !boundedResult.ExpiresAt.Equal(bounded.ValidUntil) {
		t.Fatalf("calibration outlived capability: %v %+v", err, boundedResult)
	}
	forged := cloneJSON(t, evaluations)
	forged[0].AtomRefs = nil
	if _, err := BuildCalibrationResult(corpus, forged, capability, testAdversarialNow()); !errors.Is(err, ErrCalibrationResultInvalid) {
		t.Fatalf("forged evaluation projection accepted: %v", err)
	}
	highFailure := cloneJSON(t, evaluations)
	highFailure[len(highFailure)-1].Passed = false
	highFailure[len(highFailure)-1].FailureCode = "high_consequence_probe"
	highResult, err := BuildCalibrationResult(corpus, highFailure, capability, testAdversarialNow())
	if !errors.Is(err, ErrCalibrationThresholdUnmet) || highResult.Status != CalibrationFailed ||
		highResult.HighConsequencePassed == highResult.HighConsequenceCases {
		t.Fatalf("high-consequence failure passed aggregate threshold: %v %+v", err, highResult)
	}

	report, err := CompareCalibrationResults(baseline, candidate, testDriftPolicy())
	if err != nil || report.Status != DriftBlocked || !report.Comparable || report.DropPPM != 58_824 || report.ClassDropPPM != 1_000_000 {
		t.Fatalf("per-class drift mismatch: %v %+v", err, report)
	}
	blocked, err := CompareCalibrationResults(baseline, failed, testDriftPolicy())
	if err != nil || blocked.Status != DriftBlocked {
		t.Fatalf("failed candidate not blocked: %v %+v", err, blocked)
	}
	modelDrift := rehashCalibration(t, baseline, func(r *CalibrationResult) { r.ModelRef = "model:revision-2" })
	blocked, err = CompareCalibrationResults(baseline, modelDrift, testDriftPolicy())
	if err != nil || blocked.Status != DriftBlocked || !blocked.ModelRevisionChanged {
		t.Fatalf("model drift not blocked: %v %+v", err, blocked)
	}
	allowedPolicy := testDriftPolicy()
	allowedPolicy.AllowModelRevision = true
	warning, err := CompareCalibrationResults(baseline, modelDrift, allowedPolicy)
	if err != nil || warning.Status != DriftWarning {
		t.Fatalf("allowed model revision not explicit warning: %v %+v", err, warning)
	}
	corpusDrift := rehashCalibration(t, baseline, func(r *CalibrationResult) { r.CorpusRevision = "corpus:other" })
	blocked, err = CompareCalibrationResults(baseline, corpusDrift, testDriftPolicy())
	if err != nil || blocked.Status != DriftBlocked || blocked.Comparable {
		t.Fatalf("incompatible corpus silently compared: %v %+v", err, blocked)
	}
	downgrade := rehashCalibration(t, baseline, func(r *CalibrationResult) { r.CapabilityRank = baseline.CapabilityRank - 1 })
	blocked, err = CompareCalibrationResults(baseline, downgrade, testDriftPolicy())
	if err != nil || blocked.Status != DriftBlocked || !blocked.CapabilityDowngraded {
		t.Fatalf("high consequence downgrade accepted: %v %+v", err, blocked)
	}

	if !reflect.DeepEqual(corpus, originalCorpus) || !reflect.DeepEqual(evaluations, originalEvaluations) || !reflect.DeepEqual(capability, originalCapability) {
		t.Fatal("calibration mutated inputs")
	}
}
