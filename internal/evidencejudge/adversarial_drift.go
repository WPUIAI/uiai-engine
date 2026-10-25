package evidencejudge

import (
	"sort"
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

func computeDriftReportSHA256(r DriftReport) (string, error) {
	r.ReasonCodes = append([]string(nil), r.ReasonCodes...)
	r.DriftReportSHA256 = ""
	sort.Strings(r.ReasonCodes)
	return adversarialDigest(r)
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
