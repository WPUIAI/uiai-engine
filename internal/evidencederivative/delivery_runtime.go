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
	if t == nil || c.DerivativeRef == "" || c.DestinationRef == "" || c.IdempotencyKey == "" || len(c.Payload) == 0 {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
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
		return cloneReceipt(prior.receipt), nil
	}
	result, err := t.Send(ctx, c)
	if result.ObservedAt.IsZero() {
		result.ObservedAt = time.Now().UTC()
	}
	state := result.State
	if err != nil {
		state = DeliveryOutcomeUnknown
	}
	if state != DeliveryAccepted && state != DeliveryRejected && state != DeliveryBlocked && state != DeliveryOutcomeUnknown {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	receipt := DeliveryReceipt{Schema: DeliverySchema, DeliveryID: "delivery:" + hex.EncodeToString(sum[:16]), DerivativeRef: c.DerivativeRef, DerivativeSHA256: c.DerivativeSHA256, DestinationRef: c.DestinationRef, IdempotencyKey: c.IdempotencyKey, State: state, ProviderReceiptRef: result.ProviderReceiptRef, EvidenceRefs: append([]string(nil), result.EvidenceRefs...), ObservedAt: result.ObservedAt.UTC()}
	if state == DeliveryAccepted {
		v := receipt.ObservedAt
		receipt.AcceptedAt = &v
	}
	receipt.RetryPermitted = false
	if e := ValidateDelivery(receipt); e != nil && !(state == DeliveryOutcomeUnknown && errors.Is(e, ErrDeliveryOutcomeUnknown)) {
		return DeliveryReceipt{}, e
	}
	r.records[c.IdempotencyKey] = deliveryRecord{cloneCommand(c), cloneReceipt(receipt)}
	if err != nil {
		return cloneReceipt(receipt), ErrDeliveryRetryBlocked
	}
	return cloneReceipt(receipt), nil
}
func (r *DeliveryRuntime) Reconcile(key string, state DeliveryState, providerRef string, evidence []string, at time.Time) (DeliveryReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[key]
	if !ok || at.IsZero() || len(evidence) == 0 {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	if state != DeliveryDelivered && state != DeliveryBounced && state != DeliveryRejected {
		return DeliveryReceipt{}, ErrDerivativeContractInvalid
	}
	receipt := record.receipt
	receipt.State = state
	receipt.ProviderReceiptRef = providerRef
	receipt.ReconciliationRefs = append([]string(nil), evidence...)
	receipt.EvidenceRefs = append(receipt.EvidenceRefs, evidence...)
	receipt.ObservedAt = at.UTC()
	receipt.RetryPermitted = state == DeliveryBounced || state == DeliveryRejected
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
func sameCommand(a, b DeliveryCommand) bool {
	return a.DerivativeRef == b.DerivativeRef && a.DerivativeSHA256 == b.DerivativeSHA256 && a.DestinationRef == b.DestinationRef && a.IdempotencyKey == b.IdempotencyKey && string(a.Payload) == string(b.Payload)
}
func cloneCommand(c DeliveryCommand) DeliveryCommand {
	c.Payload = append([]byte(nil), c.Payload...)
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
