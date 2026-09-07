package evidencepwa

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

type WorkItemDescriptionState string

type WorkItemRevisionState string

const (
	WorkItemDescriptionVisible     WorkItemDescriptionState = "visible"
	WorkItemDescriptionRedacted    WorkItemDescriptionState = "redacted"
	WorkItemDescriptionUnavailable WorkItemDescriptionState = "unavailable"
	WorkItemDescriptionBlocked     WorkItemDescriptionState = "blocked"

	WorkItemRevisionCurrent WorkItemRevisionState = "current"
	WorkItemRevisionStale   WorkItemRevisionState = "stale"
	WorkItemRevisionUnknown WorkItemRevisionState = "unknown"
	WorkItemRevisionBlocked WorkItemRevisionState = "blocked"
)

type WorkItemAuthorityState struct {
	AcceptanceAtomRefs      []string `json:"acceptance_atom_refs,omitempty"`
	EvidenceRequirementRefs []string `json:"evidence_requirement_refs,omitempty"`
	ReviewRequirementRefs   []string `json:"review_requirement_refs,omitempty"`
	CompletionContractRef   string   `json:"completion_contract_ref,omitempty"`
	CompletionCaseRef       string   `json:"completion_case_ref,omitempty"`
	CompletionDecisionRef   string   `json:"completion_decision_ref,omitempty"`
	ProviderCloseReceiptRef string   `json:"provider_close_receipt_ref,omitempty"`
	ReopenRef               string   `json:"reopen_ref,omitempty"`
	SettlementPosture       string   `json:"settlement_posture,omitempty"`
}

type WorkItemProjection struct {
	ProviderSurface   string                   `json:"provider_surface"`
	WorkItemRef       string                   `json:"work_item_ref"`
	ItemID            string                   `json:"item_id"`
	ItemType          string                   `json:"item_type"`
	Title             string                   `json:"title"`
	Description       string                   `json:"description,omitempty"`
	DescriptionRef    string                   `json:"description_ref,omitempty"`
	DescriptionSHA256 string                   `json:"description_sha256,omitempty"`
	DescriptionState  WorkItemDescriptionState `json:"description_state"`
	Revision          string                   `json:"revision"`
	Digest            string                   `json:"digest"`
	RevisionState     WorkItemRevisionState    `json:"revision_state"`
	StatusAtCapture   string                   `json:"status_at_capture"`
	ParentRefs        []string                 `json:"parent_refs,omitempty"`
	DependencyRefs    []string                 `json:"dependency_refs,omitempty"`
	BlockerRefs       []string                 `json:"blocker_refs,omitempty"`
	ClosurePosture    string                   `json:"closure_posture"`
	Authority         WorkItemAuthorityState   `json:"authority"`
}

type WorkItemProjectionPolicy struct {
	WorkItemRef             string
	DescriptionState        WorkItemDescriptionState
	RevisionState           WorkItemRevisionState
	CompletionContractRef   string
	CompletionCaseRef       string
	CompletionDecisionRef   string
	ProviderCloseReceiptRef string
	ReopenRef               string
	SettlementPosture       string
}

func ProjectWorkItems(bindings []evidenceartifact.WorkItemBinding, policies []WorkItemProjectionPolicy) ([]WorkItemProjection, error) {
	if len(bindings) == 0 || len(bindings) > MaxRefs || len(policies) != len(bindings) {
		return nil, ErrProjectionBindingMismatch
	}
	byRef := make(map[string]WorkItemProjectionPolicy, len(policies))
	for _, policy := range policies {
		if !validWorkItemRef(policy.WorkItemRef) || !validDescriptionState(policy.DescriptionState) ||
			!validRevisionState(policy.RevisionState) {
			return nil, ErrProjectionBindingMismatch
		}
		if _, exists := byRef[policy.WorkItemRef]; exists {
			return nil, ErrProjectionBindingMismatch
		}
		byRef[policy.WorkItemRef] = policy
	}

	projected := make([]WorkItemProjection, 0, len(bindings))
	for _, binding := range bindings {
		policy, exists := byRef[binding.WorkItemRef]
		if !exists {
			return nil, ErrProjectionBindingMismatch
		}
		item := WorkItemProjection{
			ProviderSurface: binding.ProviderSurface, WorkItemRef: binding.WorkItemRef,
			ItemID: binding.ItemID, ItemType: binding.ItemType, Title: binding.Title,
			DescriptionRef: binding.DescriptionRef, DescriptionSHA256: binding.DescriptionSHA256,
			DescriptionState: policy.DescriptionState, Revision: binding.Revision, Digest: binding.Digest,
			RevisionState: policy.RevisionState, StatusAtCapture: binding.StatusAtCapture,
			ParentRefs: cloneWorkItemRefs(binding.ParentRefs), DependencyRefs: cloneWorkItemRefs(binding.DependencyRefs),
			BlockerRefs: cloneWorkItemRefs(binding.BlockerRefs), ClosurePosture: binding.ClosurePosture,
			Authority: WorkItemAuthorityState{
				AcceptanceAtomRefs:      cloneWorkItemRefs(binding.AcceptanceAtomRefs),
				EvidenceRequirementRefs: cloneWorkItemRefs(binding.EvidenceRequirementRefs),
				ReviewRequirementRefs:   cloneWorkItemRefs(binding.ReviewRequirementRefs),
				CompletionContractRef:   policy.CompletionContractRef, CompletionCaseRef: policy.CompletionCaseRef,
				CompletionDecisionRef: policy.CompletionDecisionRef, ProviderCloseReceiptRef: policy.ProviderCloseReceiptRef,
				ReopenRef: policy.ReopenRef, SettlementPosture: policy.SettlementPosture,
			},
		}
		if policy.DescriptionState == WorkItemDescriptionVisible {
			item.Description = binding.Description
		}
		projected = append(projected, item)
	}
	if err := validateWorkItemProjections(projected); err != nil {
		return nil, err
	}
	return projected, nil
}

func validateWorkItemScope(scope ScopeBinding, items []WorkItemProjection) error {
	if len(items) == 0 {
		return nil
	}
	if err := validateWorkItemProjections(items); err != nil {
		return err
	}
	for _, item := range items {
		if item.WorkItemRef == scope.WorkItemRef {
			return nil
		}
	}
	return ErrProjectionBindingMismatch
}

func validateWorkItemProjections(items []WorkItemProjection) error {
	if len(items) == 0 || len(items) > MaxRefs {
		return ErrProjectionTooLarge
	}
	refs := make(map[string]struct{}, len(items))
	providerIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !validWorkItemToken(item.ProviderSurface, 80) || !validWorkItemRef(item.WorkItemRef) ||
			!validWorkItemToken(item.ItemID, 256) || !validWorkItemToken(item.ItemType, 80) ||
			!validWorkItemText(item.Title, MaxTitleRunes) || !validWorkItemToken(item.Revision, 256) ||
			!validSHA256(item.Digest) || !validRevisionState(item.RevisionState) ||
			!validWorkItemToken(item.StatusAtCapture, 80) || !validWorkItemToken(item.ClosurePosture, 80) {
			return ErrProjectionBindingMismatch
		}
		if _, duplicate := refs[item.WorkItemRef]; duplicate {
			return ErrProjectionBindingMismatch
		}
		refs[item.WorkItemRef] = struct{}{}
		providerID := item.ProviderSurface + "\x00" + item.ItemID
		if _, duplicate := providerIDs[providerID]; duplicate {
			return ErrProjectionBindingMismatch
		}
		providerIDs[providerID] = struct{}{}
		if !validWorkItemDescription(item) || !validWorkItemRefLists(item) || !validWorkItemAuthority(item.Authority) {
			return ErrProjectionBindingMismatch
		}
	}
	return nil
}

func validWorkItemDescription(item WorkItemProjection) bool {
	if !validDescriptionState(item.DescriptionState) {
		return false
	}
	switch item.DescriptionState {
	case WorkItemDescriptionVisible:
		if !validWorkItemText(item.Description, MaxSummaryRunes) ||
			(item.DescriptionRef != "" && !validWorkItemRef(item.DescriptionRef)) {
			return false
		}
		if item.DescriptionSHA256 == "" {
			return true
		}
		digest := sha256.Sum256([]byte(item.Description))
		return validSHA256(item.DescriptionSHA256) && hex.EncodeToString(digest[:]) == item.DescriptionSHA256
	case WorkItemDescriptionRedacted:
		return item.Description == "" && validWorkItemRef(item.DescriptionRef) && validSHA256(item.DescriptionSHA256)
	case WorkItemDescriptionUnavailable, WorkItemDescriptionBlocked:
		return item.Description == "" && validOptionalRefDigest(item.DescriptionRef, item.DescriptionSHA256)
	default:
		return false
	}
}

func validWorkItemRefLists(item WorkItemProjection) bool {
	for _, values := range [][]string{item.ParentRefs, item.DependencyRefs, item.BlockerRefs} {
		if !validWorkItemRefs(values) {
			return false
		}
	}
	return true
}

func validWorkItemAuthority(authority WorkItemAuthorityState) bool {
	for _, values := range [][]string{authority.AcceptanceAtomRefs, authority.EvidenceRequirementRefs, authority.ReviewRequirementRefs} {
		if !validWorkItemRefs(values) {
			return false
		}
	}
	for _, ref := range []string{authority.CompletionContractRef, authority.CompletionCaseRef, authority.CompletionDecisionRef,
		authority.ProviderCloseReceiptRef, authority.ReopenRef} {
		if ref != "" && !validWorkItemRef(ref) {
			return false
		}
	}
	return authority.SettlementPosture == "" || validWorkItemToken(authority.SettlementPosture, 80)
}

func validWorkItemRefs(values []string) bool {
	if len(values) > MaxRefs {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validWorkItemRef(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validOptionalRefDigest(ref, digest string) bool {
	if ref == "" && digest == "" {
		return true
	}
	return validWorkItemRef(ref) && validSHA256(digest)
}

func validWorkItemRef(value string) bool {
	return !blank(value) && len(value) <= 4096 && !hasPrivatePath(value) &&
		!strings.ContainsAny(value, "\r\n\x00")
}

func validWorkItemToken(value string, max int) bool {
	return !blank(value) && len(value) <= max && !hasPrivatePath(value) &&
		!strings.ContainsAny(value, "\r\n\x00")
}

func validWorkItemText(value string, maxRunes int) bool {
	return !blank(value) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maxRunes &&
		!hasPrivatePath(value) && !strings.ContainsRune(value, '\x00')
}

func validDescriptionState(state WorkItemDescriptionState) bool {
	return state == WorkItemDescriptionVisible || state == WorkItemDescriptionRedacted ||
		state == WorkItemDescriptionUnavailable || state == WorkItemDescriptionBlocked
}

func validRevisionState(state WorkItemRevisionState) bool {
	return state == WorkItemRevisionCurrent || state == WorkItemRevisionStale ||
		state == WorkItemRevisionUnknown || state == WorkItemRevisionBlocked
}

func cloneWorkItemRefs(values []string) []string {
	return append([]string(nil), values...)
}
