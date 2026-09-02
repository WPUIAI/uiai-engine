package evidencejudge

import (
	"reflect"
	"sort"
	"strings"
	"time"
)

func BuildCalibrationResult(corpus AdversarialCorpus, evaluations []CaseEvaluation, capability CalibrationCapability, now time.Time) (CalibrationResult, error) {
	if err := ValidateAdversarialCorpus(corpus); err != nil || !validCalibrationCapability(capability, corpus, now) || now.IsZero() {
		return CalibrationResult{}, ErrCalibrationResultInvalid
	}
	byID := make(map[string]CaseEvaluation, len(evaluations))
	for _, evaluation := range evaluations {
		if _, duplicate := byID[evaluation.CaseID]; duplicate || evaluation.CorpusRevision != corpus.CorpusRevision {
			return CalibrationResult{}, ErrCalibrationResultInvalid
		}
		byID[evaluation.CaseID] = evaluation
	}
	if len(byID) != len(corpus.Cases) {
		return CalibrationResult{}, ErrCalibrationResultInvalid
	}
	expiresAt := now.UTC().Add(time.Duration(corpus.CalibrationValidityMS) * time.Millisecond)
	if capability.ValidUntil.Before(expiresAt) {
		expiresAt = capability.ValidUntil.UTC()
	}
	result := CalibrationResult{Schema: CalibrationResultSchema, CorpusRef: corpus.CorpusRef, CorpusRevision: corpus.CorpusRevision, CorpusSHA256: corpus.CorpusSHA256,
		PolicyRef: corpus.PolicyRef, PolicyRevision: corpus.PolicyRevision, RubricRef: corpus.RubricRef, HarnessRef: corpus.HarnessRef,
		CapabilityDigest: capability.CapabilityDigest, CapabilityClass: capability.CapabilityClass, CapabilityRank: capability.CapabilityRank,
		JudgeIdentityRef: capability.JudgeIdentityRef, ProviderRef: capability.ProviderRef, ModelRef: capability.ModelRef,
		Threshold: corpus.PassThreshold, Total: uint32(len(corpus.Cases)), EvaluatedAt: now.UTC(), ExpiresAt: expiresAt}
	mutation, atoms, modalities := map[string]*CalibrationCount{}, map[string]*CalibrationCount{}, map[string]*CalibrationCount{}
	citations := map[string]struct{}{}
	for _, c := range corpus.Cases {
		evaluation, ok := byID[c.CaseID]
		if !ok || validateCaseEvaluation(c, evaluation) != nil {
			return CalibrationResult{}, ErrCalibrationResultInvalid
		}
		if c.HighConsequence {
			result.HighConsequenceCases++
			if evaluation.Passed {
				result.HighConsequencePassed++
			}
		}
		if evaluation.Passed {
			result.Passed++
		} else {
			result.Failed++
			result.OmissionRefs = append(result.OmissionRefs, c.CaseID)
		}
		incrementCalibrationCount(mutation, string(c.MutationClass), evaluation.Passed)
		for _, atom := range c.Expected.Atoms {
			incrementCalibrationCount(atoms, atom.AtomRef, evaluation.Passed)
		}
		for _, modality := range c.Expected.Modalities {
			incrementCalibrationCount(modalities, string(modality.Modality), evaluation.Passed)
		}
		for _, ref := range c.Expected.CitationRefs {
			citations[ref] = struct{}{}
		}
	}
	result.MutationCounts, result.AtomCounts, result.ModalityCounts = flattenCalibrationCounts(mutation), flattenCalibrationCounts(atoms), flattenCalibrationCounts(modalities)
	for ref := range citations {
		result.CitationRefs = append(result.CitationRefs, ref)
	}
	sort.Strings(result.CitationRefs)
	sort.Strings(result.OmissionRefs)
	if ratioAtLeast(result.Passed, result.Total, corpus.PassThreshold) && result.HighConsequencePassed == result.HighConsequenceCases {
		result.Status = CalibrationPassed
	} else {
		result.Status = CalibrationFailed
		if result.HighConsequencePassed != result.HighConsequenceCases {
			result.Uncertainty = "high_consequence_calibration_unmet"
		} else {
			result.Uncertainty = "calibration_threshold_unmet"
		}
	}
	result.CalibrationID = "calibration:" + shortAdversarialDigest(corpus.CorpusSHA256+"\x00"+capability.CapabilityDigest+"\x00"+
		now.UTC().Format(time.RFC3339Nano)+"\x00"+strings.Join(result.OmissionRefs, "\x00"))
	digest, err := computeCalibrationResultSHA256(result)
	if err != nil {
		return CalibrationResult{}, err
	}
	result.CalibrationSHA256 = digest
	if err := ValidateCalibrationResult(result); err != nil {
		return CalibrationResult{}, err
	}
	if result.Status != CalibrationPassed {
		return result, ErrCalibrationThresholdUnmet
	}
	return result, nil
}

func ValidateCalibrationResult(result CalibrationResult) error {
	thresholdMet := ratioAtLeast(result.Passed, result.Total, result.Threshold)
	highConsequenceMet := result.HighConsequencePassed == result.HighConsequenceCases
	statusValid := result.Status == CalibrationPassed && thresholdMet && highConsequenceMet && result.Uncertainty == "" ||
		result.Status == CalibrationFailed && !highConsequenceMet && result.Uncertainty == "high_consequence_calibration_unmet" ||
		result.Status == CalibrationFailed && highConsequenceMet && !thresholdMet && result.Uncertainty == "calibration_threshold_unmet"
	if result.Schema != CalibrationResultSchema || !safeAdversarialRef(result.CalibrationID) || !validAssignmentSHA256(result.CalibrationSHA256) ||
		!safeAdversarialRef(result.CorpusRef) || !safeAdversarialRef(result.CorpusRevision) || !validAssignmentSHA256(result.CorpusSHA256) ||
		!safeAdversarialRef(result.PolicyRef) || !safeAdversarialRef(result.PolicyRevision) || !safeAdversarialRef(result.RubricRef) || !safeAdversarialRef(result.HarnessRef) ||
		!validAssignmentSHA256(result.CapabilityDigest) || !safeAdversarialRef(result.CapabilityClass) || result.CapabilityRank == 0 ||
		!safeAdversarialRef(result.JudgeIdentityRef) || !safeAdversarialRef(result.ProviderRef) || !safeAdversarialRef(result.ModelRef) ||
		!validRational(result.Threshold) || result.Total == 0 || result.Passed > result.Total || result.Failed != result.Total-result.Passed ||
		result.HighConsequenceCases > result.Total || result.HighConsequencePassed > result.HighConsequenceCases || !statusValid ||
		result.EvaluatedAt.IsZero() || !result.ExpiresAt.After(result.EvaluatedAt) ||
		result.ExpiresAt.Sub(result.EvaluatedAt) > time.Duration(MaxCalibrationValidityMS)*time.Millisecond ||
		badCalibrationCounts(result.MutationCounts) || badCalibrationCounts(result.AtomCounts) || badCalibrationCounts(result.ModalityCounts) ||
		!calibrationCountsMatchResult(result) || badAdversarialRefs(result.CitationRefs) || badAdversarialRefs(result.OmissionRefs) ||
		uint32(len(result.OmissionRefs)) != result.Failed {
		return ErrCalibrationResultInvalid
	}
	digest, err := computeCalibrationResultSHA256(result)
	if err != nil || digest != result.CalibrationSHA256 {
		return ErrCalibrationResultInvalid
	}
	return nil
}

func computeCalibrationResultSHA256(r CalibrationResult) (string, error) {
	r = cloneCalibrationResult(r)
	r.CalibrationSHA256 = ""
	normalizeCalibrationResult(&r)
	return adversarialDigest(r)
}

func validateCaseEvaluation(c AdversarialCase, evaluation CaseEvaluation) error {
	atomRefs := make([]string, 0, len(c.Expected.Atoms))
	for _, atom := range c.Expected.Atoms {
		atomRefs = append(atomRefs, atom.AtomRef)
	}
	modalities := make([]Modality, 0, len(c.Expected.Modalities))
	for _, modality := range c.Expected.Modalities {
		modalities = append(modalities, modality.Modality)
	}
	sort.Strings(atomRefs)
	sort.Slice(modalities, func(i, j int) bool { return modalities[i] < modalities[j] })
	citations := sortedStrings(c.Expected.CitationRefs)
	failureValid := evaluation.Passed && evaluation.FailureCode == "" ||
		!evaluation.Passed && safeAdversarialRef(evaluation.FailureCode)
	if evaluation.CaseID != c.CaseID || evaluation.CorpusRevision != c.CorpusRevision || evaluation.MutationClass != c.MutationClass ||
		evaluation.HighConsequence != c.HighConsequence || !failureValid || !reflect.DeepEqual(evaluation.AtomRefs, atomRefs) ||
		!reflect.DeepEqual(evaluation.Modalities, modalities) || !reflect.DeepEqual(evaluation.CitationRefs, citations) {
		return ErrCalibrationResultInvalid
	}
	return nil
}

func validRational(r RationalThreshold) bool {
	return r.Denominator > 0 && r.Numerator > 0 && r.Numerator <= r.Denominator && r.Denominator <= 1_000_000
}

func ratioAtLeast(passed, total uint32, t RationalThreshold) bool {
	return uint64(passed)*uint64(t.Denominator) >= uint64(total)*uint64(t.Numerator)
}

func passPPM(passed, total uint32) uint32 {
	if total == 0 {
		return 0
	}
	return uint32(uint64(passed) * 1_000_000 / uint64(total))
}

func validCalibrationCapability(c CalibrationCapability, corpus AdversarialCorpus, now time.Time) bool {
	return validAssignmentSHA256(c.CapabilityDigest) && c.CapabilityClass == corpus.RequiredCapabilityClass && c.CapabilityRank > 0 && safeAdversarialRef(c.JudgeIdentityRef) && safeAdversarialRef(c.ProviderRef) && c.ModelRef == corpus.RequiredModelRef && c.HarnessRef == corpus.HarnessRef && contains(c.PolicyRefs, corpus.PolicyRef) && !now.Before(c.ValidFrom) && now.Before(c.ValidUntil)
}

func incrementCalibrationCount(counts map[string]*CalibrationCount, key string, passed bool) {
	c := counts[key]
	if c == nil {
		c = &CalibrationCount{Key: key}
		counts[key] = c
	}
	c.Total++
	if passed {
		c.Passed++
	} else {
		c.Failed++
	}
}

func flattenCalibrationCounts(m map[string]*CalibrationCount) []CalibrationCount {
	out := make([]CalibrationCount, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func badCalibrationCounts(values []CalibrationCount) bool {
	if len(values) == 0 {
		return true
	}
	seen := map[string]struct{}{}
	for _, v := range values {
		if !safeAdversarialRef(v.Key) || v.Total == 0 || v.Passed > v.Total || v.Failed != v.Total-v.Passed {
			return true
		}
		if _, ok := seen[v.Key]; ok {
			return true
		}
		seen[v.Key] = struct{}{}
	}
	return false
}

func calibrationCountsMatchResult(result CalibrationResult) bool {
	var mutationTotal, mutationPassed, mutationFailed uint64
	for _, count := range result.MutationCounts {
		mutationTotal += uint64(count.Total)
		mutationPassed += uint64(count.Passed)
		mutationFailed += uint64(count.Failed)
	}
	if mutationTotal != uint64(result.Total) || mutationPassed != uint64(result.Passed) || mutationFailed != uint64(result.Failed) {
		return false
	}
	for _, counts := range [][]CalibrationCount{result.AtomCounts, result.ModalityCounts} {
		for _, count := range counts {
			if count.Total > result.Total || count.Passed > result.Passed || count.Failed > result.Failed {
				return false
			}
		}
	}
	return true
}
