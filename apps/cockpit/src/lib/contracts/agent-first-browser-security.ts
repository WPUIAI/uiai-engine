import type { AgentScopeRef, Freshness, RuntimeRef } from "./agent-first-browser";

export type VerificationOutcome = "passed" | "failed" | "inconclusive" | "stale" | "blocked" | "settlement_pending";
export type PolicyDecision = "allowed" | "blocked" | "requires_review";

export interface BrowserVerificationRequest {
  schema: "uiai.focusa_browser_verification_request.v1";
  verification_request_id: string;
  mission_ref?: string; workpoint_ref?: string; completion_predicate_ref?: string; verification_policy_ref?: string;
  scope_ref?: AgentScopeRef; runtime_ref?: RuntimeRef;
  structured_conditions: unknown[]; permitted_channels: string[]; required_evidence_kinds: string[];
  independence_requirement?: string;
  watch: { mode: "once" | "until_change" | "until_verified" | "until_deadline"; deadline?: string; maximum_observations?: number; maximum_browser_minutes?: number; minimum_interval?: number; settlement_window?: number };
}

export interface BrowserVerificationResult {
  schema: "uiai.focusa_browser_verification_result.v1";
  verification_result_id: string; verification_request_id: string;
  completion_predicate_ref?: string; verification_policy_ref?: string;
  outcome: VerificationOutcome; channel_results: unknown[]; observation_refs: string[];
  artifact_refs: string[]; evidence_candidate_refs: string[]; external_confirmation_refs: string[]; contradiction_refs: string[];
  freshness?: Freshness; watch_state_ref?: string; uncertainty?: Record<string, unknown>; cost?: Record<string, unknown>; duration?: number;
}

export interface BrowserContentProvenance {
  schema: "uiai.browser_content_provenance.v1";
  content_ref: string; observation_ref?: string;
  source: { origin?: string; frame_id?: string; target_id?: string; backend_node_id?: string; ax_node_id?: string };
  channel?: string; visibility?: string;
  trust: { class: "untrusted_browser_data" | "trusted_control"; instruction_like: boolean; untrusted_content: boolean };
  data: { classifications: string[]; permitted_egress: string[] };
  integrity?: { hash?: string; captured_at?: string };
}

export interface BrowserActionInfluenceManifest {
  schema: "uiai.browser_action_influence_manifest.v1";
  action_proposal_ref: string; trusted_intent_refs: string[]; trusted_policy_refs: string[];
  untrusted_content_refs: string[]; data_egress_refs: string[];
  influence_analysis: Record<string, unknown>;
  policy: { result: PolicyDecision; rule_refs: string[]; violations: string[] };
  manifest_ref?: string;
}

export interface BrowserExecutionCapsule {
  schema: "uiai.browser_execution_capsule.v1";
  capsule_id: string;
  focusa: Record<string, string | undefined>; worker?: Record<string, unknown>; runtime?: RuntimeRef;
  environment?: Record<string, unknown>; timeline?: Record<string, unknown>; observations: string[];
  execution: Record<string, unknown>; proof: string[]; cleanup?: Record<string, unknown>; integrity?: Record<string, unknown>;
}

export interface OriginToolCandidate {
  schema: "uiai.origin_tool_candidate.v1";
  candidate_id: string; runtime: RuntimeRef;
  origin: { top_level_origin: string; registration_origin?: string; cross_origin_frame?: boolean };
  tool: { name: string; description?: string; input_schema: Record<string, unknown>; annotations?: Record<string, unknown>; registration_stack_ref?: string; manifest_hash?: string };
  classification: { side_effect_class?: string; risk_class?: string; required_data_classes: string[]; possible_destinations: string[] };
  lifecycle: { status: "discovered" | "authorized" | "changed" | "removed" | "revoked" | "expired"; valid_for_observation_ref?: string; expires_at?: string };
}
