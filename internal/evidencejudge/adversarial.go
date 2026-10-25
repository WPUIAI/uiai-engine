package evidencejudge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

func ValidateAdversarialCorpus(corpus AdversarialCorpus) error {
	if err := validateAdversarialCorpusShape(corpus); err != nil || !validAssignmentSHA256(corpus.CorpusSHA256) {
		return ErrAdversarialCorpusInvalid
	}
	digest, err := computeAdversarialCorpusSHA256Unchecked(corpus)
	if err != nil || digest != corpus.CorpusSHA256 {
		return ErrAdversarialCorpusInvalid
	}
	return nil
}

func ValidateAdversarialCase(c AdversarialCase) error {
	if c.Schema != AdversarialCaseSchema || !safeAdversarialRef(c.CaseID) || !safeAdversarialRef(c.CorpusRevision) ||
		!safeAdversarialRef(c.PolicyRef) || !safeAdversarialRef(c.PolicyRevision) || !safeAdversarialRef(c.RubricRef) ||
		!safeAdversarialRef(c.RequiredModelRef) || !safeAdversarialRef(c.RequiredCapabilityClass) ||
		!validAssignmentSHA256(c.BaseViewSHA256) || !validAssignmentSHA256(c.BaseRequestSHA256) || !validAssignmentSHA256(c.BaseResultSHA256) ||
		!validMutationClass(c.MutationClass) || c.DeterministicSeed == 0 || !safeAdversarialRef(c.FixtureLicenseRef) ||
		len(c.ProvenanceRefs) == 0 || badAdversarialRefs(c.ProvenanceRefs) || !utf8.ValidString(c.UntrustedEvidenceData) ||
		utf8.RuneCountInString(c.UntrustedEvidenceData) > MaxAdversarialPayloadRunes || unsafeAdversarialText(c.UntrustedEvidenceData) ||
		validateProtected(c.Protected) != nil || validateExpectation(c.Expected) != nil {
		return ErrAdversarialCaseInvalid
	}
	if c.HighConsequence && (c.Expected.MinimumQuorum < 2 || c.Expected.MinimumCapabilityRank == 0) {
		return ErrAdversarialCaseInvalid
	}
	return nil
}

func CanonicalAdversarialCorpusBytes(corpus AdversarialCorpus) ([]byte, error) {
	if err := ValidateAdversarialCorpus(corpus); err != nil {
		return nil, err
	}
	corpus = cloneAdversarialCorpus(corpus)
	normalizeAdversarialCorpus(&corpus)
	return json.Marshal(corpus)
}

func ComputeAdversarialCorpusSHA256(corpus AdversarialCorpus) (string, error) {
	if err := validateAdversarialCorpusShape(corpus); err != nil {
		return "", err
	}
	return computeAdversarialCorpusSHA256Unchecked(corpus)
}

func VerifyAdversarialCorpusSHA256(corpus AdversarialCorpus, expected string) error {
	if err := ValidateAdversarialCorpus(corpus); err != nil || expected != corpus.CorpusSHA256 {
		return ErrAdversarialCorpusInvalid
	}
	return nil
}

func ParseAdversarialCorpus(body []byte) (AdversarialCorpus, error) {
	if len(body) == 0 || len(body) > MaxContractBytes {
		return AdversarialCorpus{}, ErrAdversarialCorpusInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var corpus AdversarialCorpus
	if err := decoder.Decode(&corpus); err != nil {
		return AdversarialCorpus{}, ErrAdversarialCorpusInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return AdversarialCorpus{}, ErrAdversarialCorpusInvalid
	}
	if err := ValidateAdversarialCorpus(corpus); err != nil {
		return AdversarialCorpus{}, err
	}
	return corpus, nil
}

func EvaluateAdversarialCase(c AdversarialCase, observed ObservedAdversarialOutcome) (CaseEvaluation, error) {
	eval := CaseEvaluation{CaseID: c.CaseID, CorpusRevision: c.CorpusRevision, MutationClass: c.MutationClass, HighConsequence: c.HighConsequence}
	for _, atom := range c.Expected.Atoms {
		eval.AtomRefs = append(eval.AtomRefs, atom.AtomRef)
	}
	for _, modality := range c.Expected.Modalities {
		eval.Modalities = append(eval.Modalities, modality.Modality)
	}
	eval.CitationRefs = append([]string(nil), c.Expected.CitationRefs...)
	sort.Strings(eval.AtomRefs)
	sort.Slice(eval.Modalities, func(i, j int) bool { return eval.Modalities[i] < eval.Modalities[j] })
	sort.Strings(eval.CitationRefs)
	if err := ValidateAdversarialCase(c); err != nil {
		eval.FailureCode = "case_invalid"
		return eval, err
	}
	mismatch := func(code string) (CaseEvaluation, error) {
		eval.FailureCode = code
		return eval, ErrAdversarialExpectationMismatch
	}
	if observed.CaseID != c.CaseID || observed.CorpusRevision != c.CorpusRevision || observed.MutationClass != c.MutationClass {
		return mismatch("identity_mismatch")
	}
	if !validOutcome(observed.Outcome) || !validVerdict(observed.Verdict) || (observed.ErrorCode != "" && !validJudgeErrorCode(observed.ErrorCode)) {
		return mismatch("observed_invalid")
	}
	if observed.Outcome != c.Expected.Outcome || observed.Verdict != c.Expected.Verdict || observed.ErrorCode != c.Expected.ErrorCode || observed.OutcomeKnown != c.Expected.OutcomeKnown {
		return mismatch("expected_outcome_mismatch")
	}
	if isFramingMutation(c.MutationClass) && c.Expected.Outcome != OutcomeVerified && observed.Outcome == OutcomeVerified {
		return mismatch("verdict_improved_by_framing")
	}
	if !reflect.DeepEqual(normalizedProtected(observed.Protected), normalizedProtected(c.Protected)) {
		return mismatch("protected_invariant_changed")
	}
	if !reflect.DeepEqual(sortedAtoms(observed.Atoms), sortedAtoms(c.Expected.Atoms)) {
		return mismatch("atom_mismatch")
	}
	if !reflect.DeepEqual(sortedModalities(observed.Modalities), sortedModalities(c.Expected.Modalities)) {
		return mismatch("modality_mismatch")
	}
	if !reflect.DeepEqual(sortedStrings(observed.CitationRefs), sortedStrings(c.Expected.CitationRefs)) {
		return mismatch("citation_mismatch")
	}
	if observed.Quorum < c.Expected.MinimumQuorum {
		return mismatch("quorum_reduced")
	}
	if observed.CapabilityClass != c.Expected.MinimumCapabilityClass || observed.CapabilityRank < c.Expected.MinimumCapabilityRank {
		return mismatch("capability_reduced")
	}
	eval.Passed = true
	return eval, nil
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
