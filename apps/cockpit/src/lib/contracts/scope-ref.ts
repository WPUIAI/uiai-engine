// §3.2, §3.3, §3.38 H2 — every Focusa/SaaS card carries a typed ScopeRef.

export type ScopeRole =
  | "owner"
  | "assistant"
  | "observer"
  | "reviewer"
  | "ci_runner"
  | "support";

export type ScopeAuthorityState =
  | "verified"
  | "missing"
  | "stale"
  | "conflict"
  | "read_only";

export interface ScopeRef {
  project_root_key?: string;
  project_label?: string;
  workstream_key?: string;
  continuity_id?: string;
  thread_id?: string;
  session_id?: string;
  cloud_node_id?: string;
  machine_id?: string;
  daemon_endpoint?: string;
  role?: ScopeRole;
  authority_state: ScopeAuthorityState;
}
