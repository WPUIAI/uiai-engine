// UIAI-COCKPIT-002-C01 — portable agent-first browser contracts.
// These wire types mirror the normative amendment; execution authority remains in UIAI/Focusa.

export type AgentStatus = "ok" | "blocked" | "stale" | "inconclusive" | "pending" | "failed" | "requires_operator" | "resync_required";
export type ResponseProfile = "agent_compact" | "agent_standard" | "evidence_grade" | "developer_full";
export type SemanticDeltaState = "full" | "delta" | "unchanged" | "baseline_unknown" | "baseline_expired" | "document_replaced" | "frame_topology_changed" | "projection_changed" | "resync_required";

export interface AgentScopeRef { project_root?: string; continuity_id?: string; session_id?: string; }
export interface Freshness { captured_at?: string; age_ms?: number; expires_at?: string; state?: "fresh" | "stale" | "unknown"; }
export interface Usage { result_tokens?: number; inline_nodes?: number; image_items?: number; elapsed_ms?: number; }
export interface RetryPolicy { retryable?: boolean; after_ms?: number; reason?: string; }

export interface AgentResult<TPayload = unknown> {
  schema: "uiai.agent_result.v1";
  status: AgentStatus;
  failure_class?: string;
  summary?: string;
  canonical: boolean;
  degraded: boolean;
  scope_ref?: AgentScopeRef;
  runtime_ref?: string;
  payload?: { kind: string; compact?: string; value?: TPayload; payload_ref?: string };
  artifact_refs: string[];
  evidence_candidate_refs: string[];
  receipt_ref?: string;
  execution_capsule_ref?: string;
  freshness?: Freshness;
  uncertainty?: Record<string, unknown>;
  retry?: RetryPolicy;
  recovery?: string;
  next_actions: string[];
  usage?: Usage;
  correlation_id?: string;
  causation_id?: string;
  requested_profile: ResponseProfile;
  effective_profile: ResponseProfile;
  minimum_policy_profile: ResponseProfile;
  profile_upgrade_reasons: string[];
}

export interface AgentClientCapabilityProfile {
  schema: "uiai.agent_client_capability_profile.v1";
  client_id: string;
  supported_schema_versions: string[];
  supports: { structured_content: boolean; artifact_handles: boolean; semantic_deltas: boolean; image_content_items: boolean; streaming_updates: boolean; schema_refs: boolean; continuation_refs: boolean };
  budgets: { max_tool_result_tokens?: number; max_inline_nodes?: number; max_inline_evidence_refs?: number; max_tool_schemas_per_discovery?: number; max_image_items?: number };
  preferences: { response_profile: ResponseProfile; preferred_representation?: string; expand_details_on_failure_only?: boolean };
}

export interface RuntimeRef { uiai_session_id: string; browser_context_id?: string; target_id?: string; document_id?: string; navigation_id?: string; frame_id?: string; }
export interface LocatorCandidate { kind: "stable_id" | "test_attribute" | "role_name" | "label" | "structural" | "visual_anchor"; value: string; confidence?: number; }
export interface ElementFingerprint { role?: string; name?: string; value?: string; state?: Record<string, unknown>; visible?: boolean; bounds?: { x: number; y: number; width: number; height: number }; ancestor_context?: string[]; }
export interface ElementRef { runtime: RuntimeRef; locators: LocatorCandidate[]; fingerprint: ElementFingerprint; }

export interface BrowserObservation {
  schema: "uiai.browser_observation.v1";
  observation_id: string;
  parent_observation_id?: string;
  observation_sequence: number;
  runtime: RuntimeRef;
  document: { document_id: string; navigation_id?: string; url: string; origin: string; lifecycle_state?: string };
  projection: { representation: unknown[]; projection_policy?: string; query_hash?: string; budget?: Record<string, number> };
  frames: unknown[];
  global_state: Record<string, unknown>;
  element_refs: ElementRef[];
  semantic_hash: string;
  redacted_projection_hash?: string;
  captured_at: string;
  freshness: Freshness;
}

export interface BrowserActionRequest {
  schema: "uiai.browser_action_request.v2";
  action_id: string;
  focusa_action_proposal_ref?: string;
  capability_grant_ref?: string;
  runtime_ref: RuntimeRef;
  expected_observation: { observation_id: string; document_id: string; navigation_id?: string; frame_id?: string };
  target: { element_ref: ElementRef; expected_fingerprint: ElementFingerprint; fallback_locators: LocatorCandidate[] };
  preconditions: string[];
  action: { kind: string; parameters?: Record<string, unknown> };
  response_policy: ResponseProfile;
}
