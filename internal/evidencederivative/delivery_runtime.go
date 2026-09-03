package evidencederivative

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var ErrDeliveryConflict = errors.New("evidence derivative delivery idempotency conflict")
var ErrDeliveryRetryBlocked = errors.New("evidence derivative delivery retry blocked")

type DeliveryCommand struct {
	DerivativeRef, DerivativeSHA256, DestinationRef, IdempotencyKey string
	Payload                                                         []byte
	EmailPolicy                                                     *EmailDeliveryPolicy
}
type ProviderDeliveryResult struct {
	State              DeliveryState
	ProviderReceiptRef string
	EvidenceRefs       []string
	ObservedAt         time.Time
}
type DeliveryTransport interface {
	Send(context.Context, DeliveryCommand) (ProviderDeliveryResult, error)
}
type deliveryRecord struct {
	command DeliveryCommand
	receipt DeliveryReceipt
}
type DeliveryRuntime struct {
	mu      sync.Mutex
	records map[string]deliveryRecord
}

func NewDeliveryRuntime() *DeliveryRuntime {
	return &DeliveryRuntime{records: map[string]deliveryRecord{}}
}
func (r *DeliveryRuntime) Deliver(ctx context.Context, c DeliveryCommand, t DeliveryTransport) (DeliveryReceipt, error) {
	if t == nil || c.DerivativeRef == "" || c.DestinationRef == "" || c.IdempotencyKey == "" || len(c.Payload) == 0 || c.EmailPolicy == nil {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	if err := ValidateEmailDeliveryPolicy(*c.EmailPolicy); err != nil || uint64(len(c.Payload)) > c.EmailPolicy.MaxMessageBytes {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	policyDigest, err := DigestEmailDeliveryPolicy(*c.EmailPolicy)
	if err != nil {
		return DeliveryReceipt{}, err
	}
	sum := sha256.Sum256(c.Payload)
	if hex.EncodeToString(sum[:]) != c.DerivativeSHA256 {
		return DeliveryReceipt{}, ErrDerivativeIdentityMismatch
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if prior, ok := r.records[c.IdempotencyKey]; ok {
		if !sameCommand(prior.command, c) {
			return DeliveryReceipt{}, ErrDeliveryConflict
		}
		cached := cloneReceipt(prior.receipt)
		if cached.State == DeliveryOutcomeUnknown {
			return cached, ErrDeliveryRetryBlocked
		}
		return cached, nil
	}
	result, sendErr := t.Send(ctx, c)
	if result.ObservedAt.IsZero() {
		result.ObservedAt = time.Now().UTC()
	}
	validProviderState := result.State == DeliveryAccepted || result.State == DeliveryRejected || result.State == DeliveryBlocked
	validAccepted := result.State != DeliveryAccepted || (result.ProviderReceiptRef != "" && len(result.EvidenceRefs) > 0)
	uncertain := sendErr != nil || result.State == DeliveryOutcomeUnknown || !validProviderState || !validAccepted
	state := result.State
	if uncertain {
		state = DeliveryOutcomeUnknown
	}
	receipt := DeliveryReceipt{Schema: DeliverySchema, DeliveryID: deliveryID(c), DerivativeRef: c.DerivativeRef, DerivativeSHA256: c.DerivativeSHA256, DestinationRef: c.DestinationRef, IdempotencyKey: c.IdempotencyKey, PolicyRef: c.EmailPolicy.PolicyRef, PolicySHA256: policyDigest, State: state, ProviderReceiptRef: result.ProviderReceiptRef, EvidenceRefs: append([]string(nil), result.EvidenceRefs...), ObservedAt: result.ObservedAt.UTC()}
	if uncertain {
		receipt.ProviderReceiptRef = ""
		receipt.EvidenceRefs = []string{deliveryOutcomeUnknownEvidence(result, sendErr)}
	} else if state == DeliveryAccepted {
		v := receipt.ObservedAt
		receipt.AcceptedAt = &v
	}
	receipt.RetryPermitted = false
	validationErr := ValidateDelivery(receipt)
	if validationErr != nil && !errors.Is(validationErr, ErrDeliveryOutcomeUnknown) {
		receipt.State = DeliveryOutcomeUnknown
		receipt.ProviderReceiptRef = ""
		receipt.AcceptedAt = nil
		receipt.DeliveredAt = nil
		receipt.EvidenceRefs = []string{deliveryOutcomeUnknownEvidence(result, validationErr)}
		receipt.ReconciliationRefs = nil
		receipt.RetryPermitted = false
		uncertain = true
	}
	r.records[c.IdempotencyKey] = deliveryRecord{cloneCommand(c), cloneReceipt(receipt)}
	if uncertain {
		return cloneReceipt(receipt), ErrDeliveryRetryBlocked
	}
	return cloneReceipt(receipt), nil
}
func (r *DeliveryRuntime) Reconcile(key string, state DeliveryState, providerRef string, evidence []string, at time.Time) (DeliveryReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[key]
	if !ok || at.IsZero() || len(evidence) == 0 || blank(providerRef) {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	if state != DeliveryDelivered && state != DeliveryBounced && state != DeliveryRejected {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	if record.receipt.State == state && record.receipt.ProviderReceiptRef == providerRef &&
		record.receipt.ObservedAt.Equal(at.UTC()) && equalStrings(record.receipt.ReconciliationRefs, evidence) {
		return cloneReceipt(record.receipt), nil
	}
	if (record.receipt.State != DeliveryAccepted && record.receipt.State != DeliveryOutcomeUnknown) || at.UTC().Before(record.receipt.ObservedAt) {
		return DeliveryReceipt{}, ErrDeliveryConflict
	}
	receipt := record.receipt
	receipt.State = state
	receipt.ProviderReceiptRef = providerRef
	receipt.ReconciliationRefs = append([]string(nil), evidence...)
	receipt.EvidenceRefs = append(receipt.EvidenceRefs, evidence...)
	receipt.ObservedAt = at.UTC()
	receipt.RetryPermitted = false
	if state == DeliveryDelivered {
		v := receipt.ObservedAt
		if receipt.AcceptedAt == nil {
			receipt.AcceptedAt = &v
		}
		receipt.DeliveredAt = &v
		receipt.RetryPermitted = false
	}
	if e := ValidateDelivery(receipt); e != nil {
		return DeliveryReceipt{}, e
	}
	record.receipt = cloneReceipt(receipt)
	r.records[key] = record
	return cloneReceipt(receipt), nil
}
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func deliveryOutcomeUnknownEvidence(result ProviderDeliveryResult, err error) string {
	digest := sha256.New()
	for _, value := range []string{string(result.State), result.ProviderReceiptRef, result.ObservedAt.UTC().Format(time.RFC3339Nano)} {
		digest.Write([]byte(value))
		digest.Write([]byte{0})
	}
	for _, value := range result.EvidenceRefs {
		digest.Write([]byte(value))
		digest.Write([]byte{0})
	}
	if err != nil {
		digest.Write([]byte(err.Error()))
	}
	return "evidence:delivery-outcome-unknown:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func deliveryID(command DeliveryCommand) string {
	digest := sha256.New()
	policyDigest, _ := emailPolicyDigest(command.EmailPolicy)
	for _, value := range []string{command.DerivativeRef, command.DerivativeSHA256, command.DestinationRef, command.IdempotencyKey, policyDigest} {
		digest.Write([]byte(value))
		digest.Write([]byte{0})
	}
	return "delivery:" + hex.EncodeToString(digest.Sum(nil)[:16])
}

func sameCommand(a, b DeliveryCommand) bool {
	aPolicy, aErr := emailPolicyDigest(a.EmailPolicy)
	bPolicy, bErr := emailPolicyDigest(b.EmailPolicy)
	return aErr == nil && bErr == nil && aPolicy == bPolicy && a.DerivativeRef == b.DerivativeRef && a.DerivativeSHA256 == b.DerivativeSHA256 && a.DestinationRef == b.DestinationRef && a.IdempotencyKey == b.IdempotencyKey && string(a.Payload) == string(b.Payload)
}
func emailPolicyDigest(policy *EmailDeliveryPolicy) (string, error) {
	if policy == nil {
		return "", ErrDerivativeContractInvalid
	}
	return DigestEmailDeliveryPolicy(*policy)
}
func cloneCommand(c DeliveryCommand) DeliveryCommand {
	c.Payload = append([]byte(nil), c.Payload...)
	if c.EmailPolicy != nil {
		policy := *c.EmailPolicy
		c.EmailPolicy = &policy
	}
	return c
}
func cloneReceipt(v DeliveryReceipt) DeliveryReceipt {
	v.EvidenceRefs = append([]string(nil), v.EvidenceRefs...)
	v.ReconciliationRefs = append([]string(nil), v.ReconciliationRefs...)
	if v.AcceptedAt != nil {
		value := *v.AcceptedAt
		v.AcceptedAt = &value
	}
	if v.DeliveredAt != nil {
		value := *v.DeliveredAt
		v.DeliveredAt = &value
	}
	return v
}
