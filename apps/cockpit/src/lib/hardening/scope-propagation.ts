// H2 ScopeRef propagation — inherits parent project scope or DefaultHostScope for browser
import type { ScopeRef } from "../contracts/scope-ref";
export const DEFAULT_HOST: ScopeRef = { project_root_key:"/tmp/loopback", authority_state:"verified" as const };
export function propagateScope(parent: ScopeRef | null): ScopeRef { return parent ?? DEFAULT_HOST; }
