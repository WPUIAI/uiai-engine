export interface ScopeContext {
  project_root?: string;
  continuity_id?: string;
  workpoint_id?: string;
}

export interface ScopeGuardDecision {
  allowed: boolean;
  status: "connected" | "missing" | "partial";
  message: string;
  recovery: string;
}

export function inspectScope(scope: ScopeContext): ScopeGuardDecision {
  if (scope.project_root && scope.continuity_id) return { allowed: true, status: "connected", message: "Scoped browser mutation is allowed.", recovery: "" };
  if (scope.project_root || scope.continuity_id) return { allowed: false, status: "partial", message: "The project scope is incomplete.", recovery: "Configure both project root and continuity ID in Settings before changing browser state." };
  return { allowed: false, status: "missing", message: "The browser mutation is unscoped.", recovery: "Connect a project root and continuity ID in Settings before changing browser state." };
}

export function requireScopedMutation(scope: ScopeContext, operation: string): void {
  const decision = inspectScope(scope);
  if (!decision.allowed) throw new Error(`${operation} blocked: ${decision.message} ${decision.recovery}`);
}
