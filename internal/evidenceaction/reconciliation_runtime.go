package evidenceaction

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"
)

type ReconciliationObservation struct {
	InspectionRef  string
	EffectOccurred bool
	RetrySafe      bool
}

func ReconcileActionResult(result ActionResult, observations []ReconciliationObservation, reconciledAt time.Time) (ActionReconciliation, error) {
	if err := ValidateActionResult(result); err != nil {
		return ActionReconciliation{}, err
	}
	if result.Status != StatusOutcomeUnknown && result.Status != StatusPartiallyApplied {
		return ActionReconciliation{}, ErrReconciliationRequired
	}
	if len(observations) == 0 {
		return ActionReconciliation{}, ErrReconciliationRequired
	}
	ordered := append([]ReconciliationObservation(nil), observations...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].InspectionRef < ordered[j].InspectionRef })
	refs := make([]string, 0, len(ordered))
	effectOccurred := ordered[0].EffectOccurred
	consistent := true
	retrySafe := result.Status == StatusOutcomeUnknown && !effectOccurred
	for _, observation := range ordered {
		refs = append(refs, observation.InspectionRef)
		if observation.EffectOccurred != effectOccurred {
			consistent = false
		}
		if !observation.RetrySafe || observation.EffectOccurred {
			retrySafe = false
		}
	}
	state := ReconciliationConsistent
	if !consistent {
		state = ReconciliationConflict
		retrySafe = false
	}
	resultDigest, err := DigestActionResult(result)
	if err != nil {
		return ActionReconciliation{}, err
	}
	reason := ""
	if retrySafe {
		reason = "authoritative_no_effect"
	}
	reconciliation := ActionReconciliation{
		Schema:           ActionReconciliationSchema,
		ReconciliationID: reconciliationID(resultDigest, refs, state, effectOccurred, retrySafe),
		ResultRef:        result.ResultID, ResultSHA256: resultDigest, IdempotencyKey: result.IdempotencyKey,
		AuthoritativeInspectionRefs: refs, State: state, RetryPermitted: retrySafe,
		RetryReasonCode: reason, ReconciledAt: reconciledAt.UTC(),
	}
	if err := ValidateReconciliation(reconciliation, result); err != nil {
		return ActionReconciliation{}, err
	}
	return reconciliation, nil
}

func reconciliationID(resultDigest string, refs []string, state ReconciliationState, effectOccurred, retry bool) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(resultDigest))
	for _, ref := range refs {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(ref))
	}
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(state))
	if effectOccurred {
		_, _ = hash.Write([]byte{2})
	}
	if retry {
		_, _ = hash.Write([]byte{1})
	}
	return "reconciliation:" + hex.EncodeToString(hash.Sum(nil))
}
