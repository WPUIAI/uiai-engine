package evidenceaction

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

const maxPreviewTTL = 10 * time.Minute

var ErrNonceUnavailable = errors.New("action preview nonce unavailable")

type NonceSource func() (string, error)

type PreviewRuntime struct {
	mu           sync.Mutex
	now          func() time.Time
	nonce        NonceSource
	usedNonces   map[string]struct{}
	issuedNonces map[string]string
}

func NewPreviewRuntime(now func() time.Time, nonce NonceSource) *PreviewRuntime {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if nonce == nil {
		nonce = secureNonce
	}
	return &PreviewRuntime{now: now, nonce: nonce, usedNonces: map[string]struct{}{}, issuedNonces: map[string]string{}}
}

func (runtime *PreviewRuntime) BuildRegisteredPreview(proposal ActionProposal, registry *OperationRegistry, effects []ExpectedEffect, risk RiskClass, ttl time.Duration) (ActionPreview, error) {
	if registry == nil {
		return ActionPreview{}, ErrCapabilityStale
	}
	approval, err := registry.Resolve(proposal, runtime.now().UTC())
	if err != nil {
		return ActionPreview{}, err
	}
	return runtime.BuildPreview(proposal, approval, effects, risk, ttl)
}

func (runtime *PreviewRuntime) BuildPreview(proposal ActionProposal, approval OperationApproval, effects []ExpectedEffect, risk RiskClass, ttl time.Duration) (ActionPreview, error) {
	now := runtime.now().UTC()
	if err := ValidateActionProposalAgainst(proposal, proposal.Scope, proposal.ArtifactRef, proposal.ArtifactSHA256, proposal.TargetAcceptanceAtomRefs, approval, now); err != nil {
		return ActionPreview{}, err
	}
	if ttl <= 0 || ttl > maxPreviewTTL {
		ttl = maxPreviewTTL
	}
	expiresAt := now.Add(ttl)
	if proposal.ExpiresAt.Before(expiresAt) {
		expiresAt = proposal.ExpiresAt
	}
	if approval.ExpiresAt.Before(expiresAt) {
		expiresAt = approval.ExpiresAt
	}
	if !now.Before(expiresAt) {
		return ActionPreview{}, ErrActionExpired
	}
	nonce, err := runtime.reserveNonce()
	if err != nil {
		return ActionPreview{}, err
	}
	bound := false
	defer func() {
		if !bound {
			runtime.releaseNonce(nonce)
		}
	}()
	proposalDigest, err := DigestActionProposal(proposal)
	if err != nil {
		return ActionPreview{}, err
	}
	expectedEffects := append([]ExpectedEffect(nil), effects...)
	sort.Slice(expectedEffects, func(i, j int) bool { return expectedEffects[i].EffectRef < expectedEffects[j].EffectRef })
	preview := ActionPreview{
		Schema: ActionPreviewSchema, PreviewID: previewID(proposalDigest, nonce),
		ProposalRef: proposal.ProposalID, ProposalSHA256: proposalDigest, NormalizedRequestSHA256: proposalDigest,
		CapabilitySnapshotRef: proposal.CapabilitySnapshotRef, CapabilitySnapshotSHA256: proposal.CapabilitySnapshotSHA256,
		TargetRefs: append([]string(nil), proposal.TargetAcceptanceAtomRefs...), ExpectedEffects: expectedEffects,
		ExpectedStateVersion: proposal.ExpectedStateVersion, Risk: risk,
		ConfirmationRequired: confirmationRequired(proposal, risk), ExpiresAt: expiresAt, AntiReplayNonce: nonce,
	}
	if err := ValidateActionPreviewAgainst(preview, proposal, now); err != nil {
		return ActionPreview{}, err
	}
	previewDigest, err := DigestActionPreview(preview)
	if err != nil {
		return ActionPreview{}, err
	}
	runtime.bindNonce(nonce, previewDigest)
	bound = true
	return preview, nil
}

func (runtime *PreviewRuntime) ValidateIssued(preview ActionPreview) error {
	previewDigest, err := DigestActionPreview(preview)
	if err != nil {
		return err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.issuedNonces[preview.AntiReplayNonce] != previewDigest {
		return ErrPreviewMismatch
	}
	return nil
}

func (runtime *PreviewRuntime) ConsumeConfirmation(confirmation ActionConfirmation, preview ActionPreview, actorRef string) error {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	now := runtime.now().UTC()
	previewDigest, err := DigestActionPreview(preview)
	if err != nil || runtime.issuedNonces[confirmation.Nonce] != previewDigest {
		return ErrPreviewMismatch
	}
	if err := ValidateConfirmationAgainst(confirmation, preview, actorRef, runtime.usedNonces, now); err != nil {
		return err
	}
	if preview.ConfirmationRequired {
		runtime.usedNonces[confirmation.Nonce] = struct{}{}
	}
	return nil
}

func (runtime *PreviewRuntime) reserveNonce() (string, error) {
	for attempt := 0; attempt < 4; attempt++ {
		nonce, err := runtime.nonce()
		if err != nil || nonce == "" {
			return "", ErrNonceUnavailable
		}
		runtime.mu.Lock()
		_, issued := runtime.issuedNonces[nonce]
		_, used := runtime.usedNonces[nonce]
		if !issued && !used {
			runtime.issuedNonces[nonce] = ""
			runtime.mu.Unlock()
			return nonce, nil
		}
		runtime.mu.Unlock()
	}
	return "", ErrNonceUnavailable
}

func (runtime *PreviewRuntime) bindNonce(nonce, previewDigest string) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.issuedNonces[nonce] = previewDigest
}

func (runtime *PreviewRuntime) releaseNonce(nonce string) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.issuedNonces[nonce] == "" {
		delete(runtime.issuedNonces, nonce)
	}
}

func confirmationRequired(proposal ActionProposal, risk RiskClass) bool {
	return proposal.HumanReviewMandated || proposal.SourceTrust == SourceImportedUntrusted ||
		proposal.AutonomousEligibility != AutonomousEligible || proposal.SideEffect != EffectReadOnly || risk != RiskLow
}

func secureNonce() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "nonce:" + hex.EncodeToString(value), nil
}

func previewID(proposalDigest, nonce string) string {
	digest := sha256.Sum256([]byte(proposalDigest + "\x00" + nonce))
	return "preview:" + hex.EncodeToString(digest[:])
}
