package evidenceartifact

import (
	"sort"
	"strings"
)

// Normalize returns a deterministic deep copy without silently truncating data.
// Validation remains responsible for rejecting oversized or invalid values.
func Normalize(in Manifest) Manifest {
	out := in
	out.Schema = strings.TrimSpace(in.Schema)
	out.ArtifactID = strings.TrimSpace(in.ArtifactID)
	out.Title = strings.TrimSpace(in.Title)
	out.Summary = strings.TrimSpace(in.Summary)
	out.Kinds = normalizeSet(in.Kinds)
	out.CapturedAt = strings.TrimSpace(in.CapturedAt)
	out.CreatedAt = strings.TrimSpace(in.CreatedAt)
	out.Scope = normalizeScope(in.Scope)
	out.Authority = normalizeAuthority(in.Authority)
	out.Claims = normalizeClaims(in.Claims)
	out.Assets = normalizeAssets(in.Assets)
	out.Provenance = normalizeProvenance(in.Provenance)
	out.Verification = normalizeVerification(in.Verification)
	out.Security = normalizeSecurity(in.Security)
	out.ReceiptRefs = normalizeSet(in.ReceiptRefs)
	out.Policy = normalizePolicy(in.Policy)
	out.Integrity = normalizeIntegrity(in.Integrity)
	out.Links = normalizeLinks(in.Links)
	return out
}

func normalizeScope(in Scope) Scope {
	in.Project.ProjectRef = strings.TrimSpace(in.Project.ProjectRef)
	in.Project.Fingerprint = strings.TrimSpace(in.Project.Fingerprint)
	in.Project.WorkingSubpathRef = strings.TrimSpace(in.Project.WorkingSubpathRef)
	in.Project.State = BindingState(strings.TrimSpace(string(in.Project.State)))
	in.Workstream.WorkstreamRef = strings.TrimSpace(in.Workstream.WorkstreamRef)
	in.Workstream.State = BindingState(strings.TrimSpace(string(in.Workstream.State)))
	in.Workset.WorksetRef = strings.TrimSpace(in.Workset.WorksetRef)
	in.Workset.Digest = strings.ToLower(strings.TrimSpace(in.Workset.Digest))
	in.Workset.MembershipRef = strings.TrimSpace(in.Workset.MembershipRef)
	in.Workset.RequirementRefs = normalizeSet(in.Workset.RequirementRefs)
	in.Workset.DispositionRefs = normalizeSet(in.Workset.DispositionRefs)
	in.Workset.State = BindingState(strings.TrimSpace(string(in.Workset.State)))
	in.CallGraph.DefinitionRef = strings.TrimSpace(in.CallGraph.DefinitionRef)
	in.CallGraph.RunRef = strings.TrimSpace(in.CallGraph.RunRef)
	in.CallGraph.FrameRef = strings.TrimSpace(in.CallGraph.FrameRef)
	in.CallGraph.NodeRef = strings.TrimSpace(in.CallGraph.NodeRef)
	in.CallGraph.ItemRef = strings.TrimSpace(in.CallGraph.ItemRef)
	in.CallGraph.PathRef = strings.TrimSpace(in.CallGraph.PathRef)
	in.CallGraph.ParentFrameRef = strings.TrimSpace(in.CallGraph.ParentFrameRef)
	in.CallGraph.JoinRef = strings.TrimSpace(in.CallGraph.JoinRef)
	in.CallGraph.CompensationRef = strings.TrimSpace(in.CallGraph.CompensationRef)
	in.CallGraph.State = BindingState(strings.TrimSpace(string(in.CallGraph.State)))
	in.Workpoint.WorkpointRef = strings.TrimSpace(in.Workpoint.WorkpointRef)
	in.Workpoint.CheckpointRef = strings.TrimSpace(in.Workpoint.CheckpointRef)
	in.Workpoint.CurrentActionIntentRef = strings.TrimSpace(in.Workpoint.CurrentActionIntentRef)
	in.Workpoint.State = BindingState(strings.TrimSpace(string(in.Workpoint.State)))
	in.Autonomy = normalizeAutonomy(in.Autonomy)
	in.WorkItems = normalizeWorkItems(in.WorkItems)
	in.TrajectoryRef = strings.TrimSpace(in.TrajectoryRef)
	in.AssignmentRefs = normalizeSet(in.AssignmentRefs)
	in.OperationRefs = normalizeSet(in.OperationRefs)
	in.OntologyRefs = normalizeSet(in.OntologyRefs)
	in.RehydrateRefs = normalizeSet(in.RehydrateRefs)
	return in
}

func normalizeAutonomy(in AutonomyBinding) AutonomyBinding {
	in.Mode = strings.TrimSpace(in.Mode)
	in.PolicyRef = strings.TrimSpace(in.PolicyRef)
	in.WorkLoopRef = strings.TrimSpace(in.WorkLoopRef)
	in.RunRef = strings.TrimSpace(in.RunRef)
	in.RunStatus = strings.TrimSpace(in.RunStatus)
	in.AgentTeamPlanRef = strings.TrimSpace(in.AgentTeamPlanRef)
	in.ExecutorAssignmentRef = strings.TrimSpace(in.ExecutorAssignmentRef)
	in.VerifierAssignmentRefs = normalizeSet(in.VerifierAssignmentRefs)
	in.ArbitratorAssignmentRefs = normalizeSet(in.ArbitratorAssignmentRefs)
	in.CapabilityDigestRefs = normalizeSet(in.CapabilityDigestRefs)
	in.BudgetPolicyRef = strings.TrimSpace(in.BudgetPolicyRef)
	in.ResourcePolicyRef = strings.TrimSpace(in.ResourcePolicyRef)
	in.RetryPolicyRef = strings.TrimSpace(in.RetryPolicyRef)
	in.FailoverPolicyRef = strings.TrimSpace(in.FailoverPolicyRef)
	in.CooldownPolicyRef = strings.TrimSpace(in.CooldownPolicyRef)
	in.CircuitBreakerPolicyRef = strings.TrimSpace(in.CircuitBreakerPolicyRef)
	in.ReviewPostureRef = strings.TrimSpace(in.ReviewPostureRef)
	in.ClosurePostureRef = strings.TrimSpace(in.ClosurePostureRef)
	in.EventCursorRef = strings.TrimSpace(in.EventCursorRef)
	in.ContinuationRefs = normalizeSet(in.ContinuationRefs)
	return in
}

func normalizeWorkItems(in []WorkItemBinding) []WorkItemBinding {
	out := append([]WorkItemBinding(nil), in...)
	for i := range out {
		item := &out[i]
		item.ProviderSurface = strings.TrimSpace(item.ProviderSurface)
		item.WorkItemRef = strings.TrimSpace(item.WorkItemRef)
		item.ItemID = strings.TrimSpace(item.ItemID)
		item.ItemType = strings.TrimSpace(item.ItemType)
		// Provider title and description remain exact frozen source metadata.
		item.DescriptionRef = strings.TrimSpace(item.DescriptionRef)
		item.DescriptionSHA256 = strings.ToLower(strings.TrimSpace(item.DescriptionSHA256))
		item.Revision = strings.TrimSpace(item.Revision)
		item.Digest = strings.ToLower(strings.TrimSpace(item.Digest))
		item.StatusAtCapture = strings.TrimSpace(item.StatusAtCapture)
		item.ParentRefs = normalizeSet(item.ParentRefs)
		item.DependencyRefs = normalizeSet(item.DependencyRefs)
		item.BlockerRefs = normalizeSet(item.BlockerRefs)
		item.AcceptanceAtomRefs = normalizeSet(item.AcceptanceAtomRefs)
		item.EvidenceRequirementRefs = normalizeSet(item.EvidenceRequirementRefs)
		item.ReviewRequirementRefs = normalizeSet(item.ReviewRequirementRefs)
		item.ClosurePosture = strings.TrimSpace(item.ClosurePosture)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WorkItemRef == out[j].WorkItemRef {
			return out[i].ItemID < out[j].ItemID
		}
		return out[i].WorkItemRef < out[j].WorkItemRef
	})
	return out
}

func normalizeAuthority(in Authority) Authority {
	in.ProducerRef = strings.TrimSpace(in.ProducerRef)
	in.SourceAuthorityRef = strings.TrimSpace(in.SourceAuthorityRef)
	in.EvidenceAuthorityRef = strings.TrimSpace(in.EvidenceAuthorityRef)
	in.CompletionAuthorityRef = strings.TrimSpace(in.CompletionAuthorityRef)
	in.ReviewerPolicyRef = strings.TrimSpace(in.ReviewerPolicyRef)
	in.Posture = Posture(strings.TrimSpace(string(in.Posture)))
	return in
}

func normalizeClaims(in []Claim) []Claim {
	out := append([]Claim(nil), in...)
	for i := range out {
		out[i].ClaimID = strings.TrimSpace(out[i].ClaimID)
		out[i].Summary = strings.TrimSpace(out[i].Summary)
		out[i].Status = ClaimStatus(strings.TrimSpace(string(out[i].Status)))
		out[i].AcceptanceAtomRefs = normalizeSet(out[i].AcceptanceAtomRefs)
		out[i].EvidenceRefs = normalizeSet(out[i].EvidenceRefs)
		out[i].ReviewRequirementRefs = normalizeSet(out[i].ReviewRequirementRefs)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ClaimID < out[j].ClaimID })
	return out
}

func normalizeAssets(in []Asset) []Asset {
	out := append([]Asset(nil), in...)
	for i := range out {
		out[i].AssetID = strings.TrimSpace(out[i].AssetID)
		out[i].Kind = strings.TrimSpace(out[i].Kind)
		out[i].MediaType = strings.ToLower(strings.TrimSpace(out[i].MediaType))
		out[i].Path = strings.TrimSpace(out[i].Path)
		out[i].SHA256 = strings.ToLower(strings.TrimSpace(out[i].SHA256))
		out[i].CapturedAt = strings.TrimSpace(out[i].CapturedAt)
		out[i].SourceRef = strings.TrimSpace(out[i].SourceRef)
		out[i].ClaimRefs = normalizeSet(out[i].ClaimRefs)
		out[i].VerificationClass = VerificationClass(strings.TrimSpace(string(out[i].VerificationClass)))
		out[i].RedactionState = RedactionState(strings.TrimSpace(string(out[i].RedactionState)))
		out[i].AltText = strings.TrimSpace(out[i].AltText)
		out[i].TranscriptRef = strings.TrimSpace(out[i].TranscriptRef)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].AssetID < out[j].AssetID })
	return out
}

func normalizeProvenance(in Provenance) Provenance {
	in.SourceRefs = normalizeSet(in.SourceRefs)
	in.EnvironmentRefs = normalizeSet(in.EnvironmentRefs)
	in.OmissionRefs = normalizeSet(in.OmissionRefs)
	in.Custody = append([]CustodyEvent(nil), in.Custody...)
	for i := range in.Custody {
		in.Custody[i].EventID = strings.TrimSpace(in.Custody[i].EventID)
		in.Custody[i].Action = strings.TrimSpace(in.Custody[i].Action)
		in.Custody[i].ActorRef = strings.TrimSpace(in.Custody[i].ActorRef)
		in.Custody[i].InstanceRef = strings.TrimSpace(in.Custody[i].InstanceRef)
		in.Custody[i].InputRefs = normalizeSet(in.Custody[i].InputRefs)
		in.Custody[i].OutputRefs = normalizeSet(in.Custody[i].OutputRefs)
		in.Custody[i].OccurredAt = strings.TrimSpace(in.Custody[i].OccurredAt)
	}
	return in
}

func normalizeVerification(in Verification) Verification {
	in.Status = VerificationStatus(strings.TrimSpace(string(in.Status)))
	in.ReviewCaseRef = strings.TrimSpace(in.ReviewCaseRef)
	in.VerifierRefs = normalizeSet(in.VerifierRefs)
	in.JudgeResultRefs = normalizeSet(in.JudgeResultRefs)
	in.DecisionRefs = normalizeSet(in.DecisionRefs)
	in.InformationSetHash = strings.ToLower(strings.TrimSpace(in.InformationSetHash))
	return in
}

func normalizeSecurity(in Security) Security {
	in.PolicyRef = strings.TrimSpace(in.PolicyRef)
	in.InspectionReceiptRefs = normalizeSet(in.InspectionReceiptRefs)
	in.SanitizationRefs = normalizeSet(in.SanitizationRefs)
	in.RedactionRefs = normalizeSet(in.RedactionRefs)
	return in
}

func normalizePolicy(in Policy) Policy {
	in.AccessClass = AccessClass(strings.TrimSpace(string(in.AccessClass)))
	in.RedactionState = RedactionState(strings.TrimSpace(string(in.RedactionState)))
	in.Audience = strings.TrimSpace(in.Audience)
	in.RetentionClass = RetentionClass(strings.TrimSpace(string(in.RetentionClass)))
	in.ExpiresAt = strings.TrimSpace(in.ExpiresAt)
	in.PolicyRefs = normalizeSet(in.PolicyRefs)
	return in
}

func normalizeIntegrity(in Integrity) Integrity {
	in.Algorithm = strings.ToLower(strings.TrimSpace(in.Algorithm))
	in.ManifestSHA256 = strings.ToLower(strings.TrimSpace(in.ManifestSHA256))
	in.BundleSHA256 = strings.ToLower(strings.TrimSpace(in.BundleSHA256))
	return in
}

func normalizeLinks(in Links) Links {
	in.PWAPath = strings.TrimSpace(in.PWAPath)
	in.ManifestPath = strings.TrimSpace(in.ManifestPath)
	in.RelatedRefs = normalizeSet(in.RelatedRefs)
	in.SupersedesRef = strings.TrimSpace(in.SupersedesRef)
	return in
}

func normalizeSet(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	write := 0
	for _, value := range out {
		if write == 0 || value != out[write-1] {
			out[write] = value
			write++
		}
	}
	return out[:write]
}
