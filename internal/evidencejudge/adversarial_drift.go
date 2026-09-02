package evidencejudge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

func CompareCalibrationResults(baseline, candidate CalibrationResult, policy CalibrationDriftPolicy) (DriftReport, error) {
	if ValidateCalibrationResult(baseline) != nil || ValidateCalibrationResult(candidate) != nil || !validDriftPolicy(policy) {
		return DriftReport{}, ErrDriftReportInvalid
	}
	report := DriftReport{Schema: DriftReportSchema, BaselineRef: baseline.CalibrationID, BaselineSHA256: baseline.CalibrationSHA256, CandidateRef: candidate.CalibrationID,
		CandidateSHA256: candidate.CalibrationSHA256, CorpusRef: baseline.CorpusRef, CorpusRevision: baseline.CorpusRevision, PolicyRef: policy.PolicyRef,
		PolicyRevision: policy.PolicyRevision, Status: DriftWithinThreshold, Comparable: true, EvaluatedAt: policy.EvaluatedAt.UTC()}
	report.BaselinePassPPM, report.CandidatePassPPM = passPPM(baseline.Passed, baseline.Total), passPPM(candidate.Passed, candidate.Total)
	if report.BaselinePassPPM > report.CandidatePassPPM {
		report.DropPPM = report.BaselinePassPPM - report.CandidatePassPPM
	}
	block := func(reason string) {
		report.Status = DriftBlocked
		report.Comparable = false
		report.ReasonCodes = append(report.ReasonCodes, reason)
	}
	if baseline.CorpusRef != candidate.CorpusRef || baseline.CorpusRevision != candidate.CorpusRevision || baseline.CorpusSHA256 != candidate.CorpusSHA256 {
		block("corpus_identity_incompatible")
	}
	report.ModelRevisionChanged = baseline.ModelRef != candidate.ModelRef
	report.ProviderChanged = baseline.ProviderRef != candidate.ProviderRef
	report.PolicyRevisionChanged = baseline.PolicyRevision != candidate.PolicyRevision
	report.CapabilityDowngraded = candidate.CapabilityRank < baseline.CapabilityRank
	classDrop, classCountsCompatible := calibrationClassDropPPM(baseline.MutationCounts, candidate.MutationCounts)
	report.ClassDropPPM = classDrop
	if !classCountsCompatible {
		block("mutation_counts_incompatible")
	}
	if candidate.HighConsequencePassed < baseline.HighConsequencePassed {
		block("high_consequence_regression")
	}
	if report.ModelRevisionChanged && !policy.AllowModelRevision {
		block("model_revision_incompatible")
	}
	if report.ProviderChanged && !policy.AllowProviderChange {
		block("provider_incompatible")
	}
	if report.PolicyRevisionChanged && !policy.AllowPolicyRevision {
		block("policy_revision_incompatible")
	}
	if report.CapabilityDowngraded && baseline.HighConsequenceCases > 0 {
		block("high_consequence_capability_downgrade")
	}
	if candidate.Status != CalibrationPassed || !policy.EvaluatedAt.Before(candidate.ExpiresAt) {
		block("candidate_not_current_and_passing")
	}
	if report.Status != DriftBlocked {
		if report.DropPPM >= policy.BlockDropPPM || report.ClassDropPPM >= policy.BlockDropPPM {
			report.Status = DriftBlocked
			if report.ClassDropPPM >= policy.BlockDropPPM {
				report.ReasonCodes = append(report.ReasonCodes, "mutation_class_drop_blocked")
			}
			if report.DropPPM >= policy.BlockDropPPM {
				report.ReasonCodes = append(report.ReasonCodes, "pass_rate_drop_blocked")
			}
		} else if report.DropPPM >= policy.WarningDropPPM || report.ClassDropPPM >= policy.WarningDropPPM ||
			report.ModelRevisionChanged || report.ProviderChanged || report.PolicyRevisionChanged {
			report.Status = DriftWarning
			report.ReasonCodes = append(report.ReasonCodes, "drift_warning")
		}
	}
	sort.Strings(report.ReasonCodes)
	report.DriftReportID = "drift:" + shortAdversarialDigest(baseline.CalibrationSHA256+candidate.CalibrationSHA256+policy.EvaluatedAt.UTC().Format(time.RFC3339Nano))
	digest, err := computeDriftReportSHA256(report)
	if err != nil {
		return DriftReport{}, err
	}
	report.DriftReportSHA256 = digest
	if err := ValidateDriftReport(report); err != nil {
		return DriftReport{}, err
	}
	return report, nil
}

func ValidateDriftReport(report DriftReport) error {
	if report.Schema != DriftReportSchema || !safeAdversarialRef(report.DriftReportID) || !validAssignmentSHA256(report.DriftReportSHA256) ||
		!safeAdversarialRef(report.BaselineRef) || !validAssignmentSHA256(report.BaselineSHA256) || !safeAdversarialRef(report.CandidateRef) ||
		!validAssignmentSHA256(report.CandidateSHA256) || !safeAdversarialRef(report.CorpusRef) || !safeAdversarialRef(report.CorpusRevision) ||
		!safeAdversarialRef(report.PolicyRef) || !safeAdversarialRef(report.PolicyRevision) || !validDriftStatus(report.Status) ||
		report.BaselinePassPPM > 1_000_000 || report.CandidatePassPPM > 1_000_000 || report.DropPPM > 1_000_000 || report.ClassDropPPM > 1_000_000 || report.EvaluatedAt.IsZero() ||
		badAdversarialRefs(report.ReasonCodes) || (report.Status == DriftBlocked && len(report.ReasonCodes) == 0) {
		return ErrDriftReportInvalid
	}
	digest, err := computeDriftReportSHA256(report)
	if err != nil || digest != report.DriftReportSHA256 {
		return ErrDriftReportInvalid
	}
	return nil
}

func validateAdversarialCorpusShape(c AdversarialCorpus) error {
	if c.Schema != AdversarialCorpusSchema || !safeAdversarialRef(c.CorpusRef) || !safeAdversarialRef(c.CorpusRevision) ||
		!safeAdversarialRef(c.PolicyRef) || !safeAdversarialRef(c.PolicyRevision) || !safeAdversarialRef(c.RubricRef) || !safeAdversarialRef(c.HarnessRef) ||
		!safeAdversarialRef(c.RequiredModelRef) || !safeAdversarialRef(c.RequiredCapabilityClass) || !validRational(c.PassThreshold) ||
		c.CalibrationValidityMS == 0 || c.CalibrationValidityMS > MaxCalibrationValidityMS || !safeAdversarialRef(c.FixtureLicenseRef) ||
		len(c.ProvenanceRefs) == 0 || badAdversarialRefs(c.ProvenanceRefs) || len(c.Cases) == 0 || len(c.Cases) > MaxAdversarialCases {
		return ErrAdversarialCorpusInvalid
	}
	seen := map[string]struct{}{}
	for _, item := range c.Cases {
		if ValidateAdversarialCase(item) != nil || item.CorpusRevision != c.CorpusRevision || item.PolicyRef != c.PolicyRef || item.PolicyRevision != c.PolicyRevision ||
			item.RubricRef != c.RubricRef || item.RequiredModelRef != c.RequiredModelRef || item.RequiredCapabilityClass != c.RequiredCapabilityClass {
			return ErrAdversarialCorpusInvalid
		}
		if _, duplicate := seen[item.CaseID]; duplicate {
			return ErrAdversarialCorpusInvalid
		}
		seen[item.CaseID] = struct{}{}
	}
	body, err := json.Marshal(c)
	if err != nil || len(body) > MaxContractBytes {
		return ErrAdversarialCorpusInvalid
	}
	return nil
}

func computeAdversarialCorpusSHA256Unchecked(c AdversarialCorpus) (string, error) {
	c = cloneAdversarialCorpus(c)
	c.CorpusSHA256 = ""
	normalizeAdversarialCorpus(&c)
	return adversarialDigest(c)
}

func computeDriftReportSHA256(r DriftReport) (string, error) {
	r.ReasonCodes = append([]string(nil), r.ReasonCodes...)
	r.DriftReportSHA256 = ""
	sort.Strings(r.ReasonCodes)
	return adversarialDigest(r)
}

func adversarialDigest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil || len(b) > MaxContractBytes {
		return "", ErrAdversarialCorpusInvalid
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

func shortAdversarialDigest(s string) string {
	d := sha256.Sum256([]byte(s))
	return hex.EncodeToString(d[:16])
}

func cloneAdversarialCorpus(c AdversarialCorpus) AdversarialCorpus {
	c.ProvenanceRefs = append([]string(nil), c.ProvenanceRefs...)
	c.Cases = append([]AdversarialCase(nil), c.Cases...)
	for index := range c.Cases {
		item := &c.Cases[index]
		item.ProvenanceRefs = append([]string(nil), item.ProvenanceRefs...)
		item.Expected.Atoms = append([]AdversarialAtomExpectation(nil), item.Expected.Atoms...)
		item.Expected.Modalities = append([]AdversarialModalityExpectation(nil), item.Expected.Modalities...)
		item.Expected.CitationRefs = append([]string(nil), item.Expected.CitationRefs...)
		item.Protected.AllowedVerdicts = append([]Verdict(nil), item.Protected.AllowedVerdicts...)
	}
	return c
}

func cloneCalibrationResult(r CalibrationResult) CalibrationResult {
	r.MutationCounts = append([]CalibrationCount(nil), r.MutationCounts...)
	r.AtomCounts = append([]CalibrationCount(nil), r.AtomCounts...)
	r.ModalityCounts = append([]CalibrationCount(nil), r.ModalityCounts...)
	r.CitationRefs = append([]string(nil), r.CitationRefs...)
	r.OmissionRefs = append([]string(nil), r.OmissionRefs...)
	return r
}

func normalizeAdversarialCorpus(c *AdversarialCorpus) {
	sort.Strings(c.ProvenanceRefs)
	sort.Slice(c.Cases, func(i, j int) bool { return c.Cases[i].CaseID < c.Cases[j].CaseID })
	for i := range c.Cases {
		normalizeAdversarialCase(&c.Cases[i])
	}
}

func normalizeAdversarialCase(c *AdversarialCase) {
	sort.Strings(c.ProvenanceRefs)
	c.Expected.Atoms = sortedAtoms(c.Expected.Atoms)
	c.Expected.Modalities = sortedModalities(c.Expected.Modalities)
	c.Expected.CitationRefs = sortedStrings(c.Expected.CitationRefs)
	c.Protected = normalizedProtected(c.Protected)
}

func normalizeCalibrationResult(r *CalibrationResult) {
	sort.Slice(r.MutationCounts, func(i, j int) bool { return r.MutationCounts[i].Key < r.MutationCounts[j].Key })
	sort.Slice(r.AtomCounts, func(i, j int) bool { return r.AtomCounts[i].Key < r.AtomCounts[j].Key })
	sort.Slice(r.ModalityCounts, func(i, j int) bool { return r.ModalityCounts[i].Key < r.ModalityCounts[j].Key })
	sort.Strings(r.CitationRefs)
	sort.Strings(r.OmissionRefs)
}

func validateExpectation(e AdversarialExpectation) error {
	if !validOutcome(e.Outcome) || !validVerdict(e.Verdict) || (e.ErrorCode != "" && !validJudgeErrorCode(e.ErrorCode)) || len(e.Atoms) == 0 || len(e.Modalities) == 0 ||
		e.MinimumQuorum == 0 || !safeAdversarialRef(e.MinimumCapabilityClass) || e.MinimumCapabilityRank == 0 || badAdversarialRefs(e.CitationRefs) {
		return ErrAdversarialCaseInvalid
	}
	seenAtoms := map[string]struct{}{}
	for _, a := range e.Atoms {
		if !safeAdversarialRef(a.AtomRef) || !validVerdict(a.Verdict) {
			return ErrAdversarialCaseInvalid
		}
		if _, ok := seenAtoms[a.AtomRef]; ok {
			return ErrAdversarialCaseInvalid
		}
		seenAtoms[a.AtomRef] = struct{}{}
	}
	seenMods := map[string]struct{}{}
	for _, m := range e.Modalities {
		key := m.AtomRef + "\x00" + string(m.Modality)
		if !safeAdversarialRef(m.AtomRef) || !validModality(m.Modality) || !validModalityStatus(m.Status) {
			return ErrAdversarialCaseInvalid
		}
		if _, ok := seenMods[key]; ok {
			return ErrAdversarialCaseInvalid
		}
		seenMods[key] = struct{}{}
	}
	return nil
}

func validateProtected(p ProtectedJudgeInvariants) error {
	if !validAssignmentSHA256(p.ScopeSHA256) || !safeAdversarialRef(p.RubricRef) || !safeAdversarialRef(p.PolicyRef) || !safeAdversarialRef(p.PolicyRevision) || !validAssignmentSHA256(p.InformationSetSHA256) || !safeAdversarialRef(p.AssignmentRef) || !safeAdversarialRef(p.BudgetRef) || !safeAdversarialRef(p.AuthorityRef) || len(p.AllowedVerdicts) == 0 {
		return ErrAdversarialCaseInvalid
	}
	seen := map[Verdict]struct{}{}
	for _, v := range p.AllowedVerdicts {
		if !validVerdict(v) {
			return ErrAdversarialCaseInvalid
		}
		if _, ok := seen[v]; ok {
			return ErrAdversarialCaseInvalid
		}
		seen[v] = struct{}{}
	}
	return nil
}

func validMutationClass(m MutationClass) bool {
	switch m {
	case MutationPromptInjection, MutationPersuasiveSummary, MutationSourceOrder, MutationCitationOrder, MutationResultOrder, MutationMissingModality, MutationModalitySubstitution, MutationIdentityMismatch, MutationDigestMismatch, MutationStaleEvidence, MutationCitationEscape, MutationCorrelatedVerifier, MutationBudgetPressure, MutationProviderFailure, MutationModelRevision, MutationPolicyRevision, MutationHighConsequenceThreshold:
		return true
	}
	return false
}

func isFramingMutation(m MutationClass) bool {
	return m == MutationPromptInjection || m == MutationPersuasiveSummary || m == MutationSourceOrder || m == MutationCitationOrder || m == MutationResultOrder
}

func validDriftPolicy(p CalibrationDriftPolicy) bool {
	return safeAdversarialRef(p.PolicyRef) && safeAdversarialRef(p.PolicyRevision) && p.WarningDropPPM > 0 && p.WarningDropPPM < p.BlockDropPPM && p.BlockDropPPM <= 1_000_000 && !p.EvaluatedAt.IsZero()
}

func validDriftStatus(s DriftStatus) bool {
	return s == DriftWithinThreshold || s == DriftWarning || s == DriftBlocked
}

func calibrationClassDropPPM(baseline, candidate []CalibrationCount) (uint32, bool) {
	candidateByKey := make(map[string]CalibrationCount, len(candidate))
	for _, count := range candidate {
		candidateByKey[count.Key] = count
	}
	var maximum uint32
	for _, prior := range baseline {
		next, ok := candidateByKey[prior.Key]
		if !ok || next.Total != prior.Total {
			return 0, false
		}
		priorPPM, nextPPM := passPPM(prior.Passed, prior.Total), passPPM(next.Passed, next.Total)
		if priorPPM > nextPPM && priorPPM-nextPPM > maximum {
			maximum = priorPPM - nextPPM
		}
		delete(candidateByKey, prior.Key)
	}
	return maximum, len(candidateByKey) == 0
}

func normalizedProtected(p ProtectedJudgeInvariants) ProtectedJudgeInvariants {
	p.AllowedVerdicts = append([]Verdict(nil), p.AllowedVerdicts...)
	sort.Slice(p.AllowedVerdicts, func(i, j int) bool { return p.AllowedVerdicts[i] < p.AllowedVerdicts[j] })
	return p
}

func sortedAtoms(v []AdversarialAtomExpectation) []AdversarialAtomExpectation {
	out := append([]AdversarialAtomExpectation(nil), v...)
	sort.Slice(out, func(i, j int) bool { return out[i].AtomRef < out[j].AtomRef })
	return out
}

func sortedModalities(v []AdversarialModalityExpectation) []AdversarialModalityExpectation {
	out := append([]AdversarialModalityExpectation(nil), v...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].AtomRef == out[j].AtomRef {
			return out[i].Modality < out[j].Modality
		}
		return out[i].AtomRef < out[j].AtomRef
	})
	return out
}

func sortedStrings(v []string) []string {
	out := append([]string(nil), v...)
	sort.Strings(out)
	return out
}

func badAdversarialRefs(v []string) bool {
	seen := map[string]struct{}{}
	for _, s := range v {
		if !safeAdversarialRef(s) {
			return true
		}
		if _, ok := seen[s]; ok {
			return true
		}
		seen[s] = struct{}{}
	}
	return false
}

func safeAdversarialRef(s string) bool { return validAssignmentRef(s) && !unsafeAdversarialText(s) }

func unsafeAdversarialText(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "file://") || strings.Contains(lower, "/root/") || strings.Contains(lower, "/home/") || strings.Contains(lower, "c:\\users\\") || strings.Contains(lower, "../") || strings.ContainsAny(s, "\x00\r")
}
