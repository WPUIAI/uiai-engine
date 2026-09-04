package evidencejudge

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

const (
	InformationSetSchema = "uiai.evidence_judge_information_set.v1"

	MaxInformationSelectionRefs = 512
	MaxInformationOmissions     = 512
	MaxInformationSetBytes      = 1 << 20
	MaxInformationRefRunes      = 512
)

var (
	ErrInformationSetInvalid       = errors.New("evidence judge information set invalid")
	ErrInformationSelectionInvalid = errors.New("evidence judge information selection invalid")
	ErrInformationSourceMissing    = errors.New("evidence judge information source missing")
	ErrInformationOmissionInvalid  = errors.New("evidence judge information omission invalid")
	ErrInformationScopeMismatch    = errors.New("evidence judge information scope mismatch")
)

type OmissionReason string

const (
	OmissionMissing     OmissionReason = "missing"
	OmissionRedacted    OmissionReason = "redacted"
	OmissionStale       OmissionReason = "stale"
	OmissionUnsupported OmissionReason = "unsupported"
	OmissionUnavailable OmissionReason = "unavailable"
	OmissionBlocked     OmissionReason = "blocked"
)

type SourceSelection struct {
	SourceRef string   `json:"source_ref"`
	AtomRefs  []string `json:"atom_refs"`
	Required  bool     `json:"required"`
}

type InformationSelectionRequest struct {
	InformationSetID       string            `json:"information_set_id"`
	ArtifactRef            string            `json:"artifact_ref"`
	WorkItemRef            string            `json:"work_item_ref"`
	AcceptanceAtomRefs     []string          `json:"acceptance_atom_refs"`
	Sources                []SourceSelection `json:"sources"`
	PolicyRef              string            `json:"policy_ref"`
	RubricRef              string            `json:"rubric_ref"`
	BlindingProfileRef     string            `json:"blinding_profile_ref,omitempty"`
	ContradictionPolicyRef string            `json:"contradiction_policy_ref"`
	CreatedAt              time.Time         `json:"created_at"`
	ExpiresAt              time.Time         `json:"expires_at"`
}

type InformationOmission struct {
	SourceRef  string         `json:"source_ref"`
	ReasonCode OmissionReason `json:"reason_code"`
	Required   bool           `json:"required"`
}

type InformationSet struct {
	Schema                   string                `json:"schema"`
	InformationSetID         string                `json:"information_set_id"`
	InformationSetSHA256     string                `json:"information_set_sha256,omitempty"`
	Artifact                 ArtifactBinding       `json:"artifact"`
	AcceptanceAtomRefs       []string              `json:"acceptance_atom_refs"`
	Sources                  []EvidenceSource      `json:"sources"`
	Citations                []Citation            `json:"citations"`
	Omissions                []InformationOmission `json:"omissions,omitempty"`
	PolicyRef                string                `json:"policy_ref"`
	RubricRef                string                `json:"rubric_ref"`
	BlindingProfileRef       string                `json:"blinding_profile_ref,omitempty"`
	ContradictionPolicyRef   string                `json:"contradiction_policy_ref"`
	EvidenceInstructionClass string                `json:"evidence_instruction_class"`
	CreatedAt                time.Time             `json:"created_at"`
	ExpiresAt                time.Time             `json:"expires_at"`
}

type JudgeViewPolicy struct {
	CompletionContractRef      string
	CompletionContractRevision string
	Policy                     JudgePolicy
}

type inventorySource struct {
	source       EvidenceSource
	locator      CitationLocator
	allowedAtoms map[string]struct{}
	unavailable  OmissionReason
}

func BuildInformationSet(manifest evidenceartifact.Manifest, request InformationSelectionRequest) (InformationSet, error) {
	if err := evidenceartifact.Validate(manifest); err != nil {
		return InformationSet{}, fmt.Errorf("%w: artifact", ErrInformationSetInvalid)
	}
	if err := validateSelectionRequest(request); err != nil {
		return InformationSet{}, err
	}
	if manifest.ArtifactID != request.ArtifactRef {
		return InformationSet{}, ErrInformationScopeMismatch
	}
	manifestSHA, err := evidenceartifact.ComputeManifestSHA256(manifest)
	if err != nil || !validInformationSHA256(manifest.Integrity.BundleSHA256) {
		return InformationSet{}, fmt.Errorf("%w: artifact integrity", ErrInformationSetInvalid)
	}
	binding, err := artifactBinding(manifest, request.WorkItemRef, manifestSHA)
	if err != nil {
		return InformationSet{}, err
	}
	inventory, err := buildSourceInventory(manifest)
	if err != nil {
		return InformationSet{}, err
	}
	atomSet := informationStringSet(request.AcceptanceAtomRefs)
	set := InformationSet{
		Schema:                   InformationSetSchema,
		InformationSetID:         request.InformationSetID,
		Artifact:                 binding,
		AcceptanceAtomRefs:       sortedCopy(request.AcceptanceAtomRefs),
		PolicyRef:                request.PolicyRef,
		RubricRef:                request.RubricRef,
		BlindingProfileRef:       request.BlindingProfileRef,
		ContradictionPolicyRef:   request.ContradictionPolicyRef,
		EvidenceInstructionClass: "untrusted_evidence_data",
		CreatedAt:                request.CreatedAt.UTC(),
		ExpiresAt:                request.ExpiresAt.UTC(),
	}
	for _, selection := range request.Sources {
		item, ok := inventory[selection.SourceRef]
		reason := OmissionMissing
		if ok {
			reason = item.unavailable
		}
		if !ok || reason != "" {
			if selection.Required {
				return InformationSet{}, fmt.Errorf("%w: required source", ErrInformationSourceMissing)
			}
			set.Omissions = append(set.Omissions, InformationOmission{SourceRef: selection.SourceRef, ReasonCode: reason, Required: false})
			continue
		}
		for _, atomRef := range selection.AtomRefs {
			if _, exists := atomSet[atomRef]; !exists {
				return InformationSet{}, ErrInformationSelectionInvalid
			}
			if len(item.allowedAtoms) > 0 {
				if _, allowed := item.allowedAtoms[atomRef]; !allowed {
					return InformationSet{}, ErrInformationSelectionInvalid
				}
			}
		}
		atoms := sortedCopy(selection.AtomRefs)
		citationID, err := informationCitationID(item.source, atoms)
		if err != nil {
			return InformationSet{}, err
		}
		set.Sources = append(set.Sources, item.source)
		set.Citations = append(set.Citations, Citation{
			Schema:        JudgeCitationSchema,
			CitationID:    citationID,
			SourceRef:     item.source.SourceRef,
			SourceSHA256:  item.source.SHA256,
			Modality:      item.source.Modality,
			Locator:       item.locator,
			SupportsAtoms: atoms,
		})
	}
	set = normalizeInformationSet(set)
	if len(set.Sources) == 0 {
		return InformationSet{}, ErrInformationSourceMissing
	}
	digest, err := ComputeInformationSetSHA256(set)
	if err != nil {
		return InformationSet{}, err
	}
	set.InformationSetSHA256 = digest
	if err := ValidateInformationSet(set); err != nil {
		return InformationSet{}, err
	}
	return set, nil
}

func ValidateInformationSet(set InformationSet) error {
	if err := validateInformationSetShape(set); err != nil {
		return err
	}
	if set.InformationSetSHA256 != "" {
		digest, err := computeInformationSetSHA256Unchecked(set)
		if err != nil || digest != set.InformationSetSHA256 {
			return ErrInformationSetMismatch
		}
	}
	return nil
}

func CanonicalInformationSetBytes(set InformationSet) ([]byte, error) {
	if err := validateInformationSetShape(set); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeInformationSet(set))
}

func ComputeInformationSetSHA256(set InformationSet) (string, error) {
	if err := validateInformationSetShape(set); err != nil {
		return "", err
	}
	return computeInformationSetSHA256Unchecked(set)
}

func VerifyInformationSetSHA256(set InformationSet) error {
	if !validInformationSHA256(set.InformationSetSHA256) {
		return ErrInformationSetMismatch
	}
	digest, err := computeInformationSetSHA256Unchecked(set)
	if err != nil || digest != set.InformationSetSHA256 {
		return ErrInformationSetMismatch
	}
	return nil
}

func BuildJudgeView(set InformationSet, policy JudgeViewPolicy, atoms []AcceptanceAtom, modalities []ModalityRequirement) (JudgeView, error) {
	if err := ValidateInformationSet(set); err != nil {
		return JudgeView{}, err
	}
	if set.InformationSetSHA256 == "" {
		return JudgeView{}, ErrInformationSetMismatch
	}
	if blank(policy.CompletionContractRef) || blank(policy.CompletionContractRevision) ||
		policy.Policy.PolicyRef != set.PolicyRef || policy.Policy.RubricRef != set.RubricRef ||
		policy.Policy.BlindingProfileRef != set.BlindingProfileRef ||
		policy.Policy.ContradictionPolicyRef != set.ContradictionPolicyRef {
		return JudgeView{}, ErrInformationSetMismatch
	}
	atoms = normalizeAcceptanceAtoms(atoms)
	modalities = normalizeModalityRequirements(modalities)
	if !sameAtomRefs(atoms, set.AcceptanceAtomRefs) {
		return JudgeView{}, ErrInformationSetMismatch
	}
	viewMaterial := struct {
		InformationSetSHA256 string
		Policy               JudgeViewPolicy
		Atoms                []AcceptanceAtom
		Modalities           []ModalityRequirement
	}{set.InformationSetSHA256, policy, atoms, modalities}
	viewDigest, err := informationDigest(viewMaterial)
	if err != nil {
		return JudgeView{}, err
	}
	omissions := make([]Omission, len(set.Omissions))
	for i, omission := range set.Omissions {
		omissions[i] = Omission{Ref: omission.SourceRef, ReasonCode: string(omission.ReasonCode)}
	}
	judgePolicy := policy.Policy
	judgePolicy.ForbiddenAssumptions = sortedCopy(judgePolicy.ForbiddenAssumptions)
	view := JudgeView{
		Schema:                     JudgeViewSchema,
		ViewID:                     "judge-view:sha256:" + viewDigest,
		Artifact:                   set.Artifact,
		CompletionContractRef:      policy.CompletionContractRef,
		CompletionContractRevision: policy.CompletionContractRevision,
		AcceptanceAtoms:            atoms,
		AllowedVerdicts: []Verdict{
			VerdictBlocked, VerdictDisputed, VerdictInsufficientEvidence, VerdictRebutted, VerdictSupported,
		},
		InformationSetRef:    set.InformationSetID,
		InformationSetSHA256: set.InformationSetSHA256,
		Sources:              append([]EvidenceSource(nil), set.Sources...),
		Citations:            append([]Citation(nil), set.Citations...),
		Modalities:           modalities,
		Omissions:            omissions,
		Policy:               judgePolicy,
		CreatedAt:            set.CreatedAt,
		ExpiresAt:            set.ExpiresAt,
	}
	if err := ValidateJudgeView(view); err != nil {
		return JudgeView{}, err
	}
	return view, nil
}
