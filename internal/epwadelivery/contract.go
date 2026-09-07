package epwadelivery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalidContract = errors.New("EPWA delivery contract invalid")
	ErrDeliveryBlocked = errors.New("EPWA delivery is not ready")
)

func New(input Input) (Delivery, error) {
	result := Delivery{
		Schema: Schema, Revision: 1, Producer: input.Producer, Artifact: input.Artifact, EPWA: input.EPWA, Scope: input.Scope,
		State: input.State, IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		CreatedAt: input.CreatedAt.UTC(), ObservedAt: input.ObservedAt.UTC(),
		RecoveryRef: strings.TrimSpace(input.RecoveryRef), TruthNotice: TruthNotice,
	}
	id, err := deliveryID(result)
	if err != nil {
		return Delivery{}, err
	}
	result.DeliveryID = id
	if err := Validate(result); err != nil {
		return Delivery{}, err
	}
	return result, nil
}

func Validate(result Delivery) error {
	if result.Schema != Schema || result.Revision == 0 || !ValidProducer(result.Producer) || result.TruthNotice != TruthNotice || !validRef(result.IdempotencyKey) || result.CreatedAt.IsZero() || result.ObservedAt.IsZero() || result.ObservedAt.Before(result.CreatedAt) {
		return ErrInvalidContract
	}
	if !validRef(result.Artifact.ArtifactRef) || result.Artifact.Revision == 0 || !validSHA256(result.Artifact.ManifestSHA256) || !validSHA256(result.Artifact.OutputSHA256) {
		return ErrInvalidContract
	}
	if !validSHA256(result.EPWA.PackageID) || !strings.HasPrefix(result.EPWA.PackageRef, "uiai-epwa-package:sha256:") || !validSHA256(strings.TrimPrefix(result.EPWA.PackageRef, "uiai-epwa-package:sha256:")) || !validSHA256(result.EPWA.PackageSHA256) {
		return ErrInvalidContract
	}
	if result.EPWA.PackageRef != "uiai-epwa-package:sha256:"+result.EPWA.PackageSHA256 || !validAccess(result.EPWA.Access) {
		return ErrInvalidContract
	}
	if !validState(result.State) || !validScope(result.Scope) {
		return ErrInvalidContract
	}
	if result.State == StateReady {
		if result.Scope.Posture != ScopeComplete || result.EPWA.Access != AccessPublicSafe ||
			!strings.HasPrefix(result.EPWA.ProjectionRef, "uiai-evidence-projection:sha256:") || !validSHA256(result.EPWA.ProjectionSHA256) ||
			!validHTTPS(result.EPWA.RecordURL) || !validHTTPS(result.EPWA.PortableURL) || !strings.HasSuffix(mustURLPath(result.EPWA.PortableURL), ".zip") || result.RecoveryRef != "" {
			return ErrInvalidContract
		}
	} else {
		if !validRef(result.RecoveryRef) || result.EPWA.RecordURL != "" || result.EPWA.PortableURL != "" {
			return ErrInvalidContract
		}
	}
	want, err := deliveryID(result)
	if err != nil || result.DeliveryID != want {
		return ErrInvalidContract
	}
	return nil
}

func RequireReady(result Delivery) error {
	if err := Validate(result); err != nil {
		return err
	}
	if result.State != StateReady {
		return fmt.Errorf("%w: state=%s recovery=%s", ErrDeliveryBlocked, result.State, result.RecoveryRef)
	}
	return nil
}

func deliveryID(result Delivery) (string, error) {
	identityEPWA := result.EPWA
	identityEPWA.RecordURL = ""
	identityEPWA.PortableURL = ""
	body, err := json.Marshal(struct {
		Schema         string          `json:"schema"`
		Producer       ProducerID      `json:"producer"`
		Artifact       ArtifactBinding `json:"artifact"`
		EPWA           EPWABinding     `json:"epwa"`
		Scope          ScopeBinding    `json:"scope"`
		IdempotencyKey string          `json:"idempotency_key"`
		TruthNotice    string          `json:"truth_notice"`
	}{result.Schema, result.Producer, result.Artifact, identityEPWA, result.Scope, result.IdempotencyKey, result.TruthNotice})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return "uiai-epwa-delivery:sha256:" + hex.EncodeToString(digest[:]), nil
}

func validScope(scope ScopeBinding) bool {
	if scope.Posture != ScopeComplete && scope.Posture != ScopeBlocked {
		return false
	}
	values := []string{scope.ProjectRef, scope.WorkstreamRef, scope.WorksetRef, scope.CallGraphRef, scope.WorkpointRef, scope.WorkItemRef, scope.ContinuityRef}
	if scope.Posture == ScopeComplete {
		for _, value := range values {
			if !validRef(value) {
				return false
			}
		}
		return true
	}
	for _, value := range values {
		if value != "" && !validRef(value) {
			return false
		}
	}
	return true
}

func validState(value State) bool {
	switch value {
	case StateReady, StateBlocked, StatePendingReconcile, StateUnavailable, StateCorrupt, StateStale, StateRedacted, StateDegraded:
		return true
	default:
		return false
	}
}

func validAccess(value AccessPosture) bool {
	return value == AccessPublicSafe || value == AccessPrivate || value == AccessUnlisted || value == AccessRedacted
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validRef(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= 512 && !strings.ContainsAny(value, "\r\n\x00")
}

func validHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}

func mustURLPath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return parsed.Path
}
