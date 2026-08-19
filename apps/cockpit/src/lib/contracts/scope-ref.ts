// §3.2, §3.3, §3.38 H2 — every Focusa/SaaS card carries a typed ScopeRef.
// WorkstreamKey = ProjectRootKey::ContinuityId (docs/164, WorkstreamRoot resolution_key).
import { assertWorkstreamKeyMatchesScope, deriveWorkstreamKeyFromScope } from "./workstream";
export { assertWorkstreamKeyMatchesScope, deriveWorkstreamKeyFromScope, workstreamKey, parseWorkstreamKey } from "./workstream";

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

export function scopeWorkstreamKey(scope: ScopeRef): string | undefined {
  return deriveWorkstreamKeyFromScope({ project_root_key: scope.project_root_key, workstream_key: scope.workstream_key, continuity_id: scope.continuity_id });
}

export function assertScopeWorkstream(scope: ScopeRef): void {
  assertWorkstreamKeyMatchesScope({ project_root_key: scope.project_root_key, workstream_key: scope.workstream_key, continuity_id: scope.continuity_id });
}
