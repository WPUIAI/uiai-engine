export const ENTITLEMENT_PROJECTION_SCHEMA = "uiai.entitlement_projection.v1" as const;
export const ENTITLEMENT_DENIAL_SCHEMA = "uiai.entitlement_denial.v1" as const;

export type EntitlementState = "unactivated" | "active_evaluation" | "active_paid" | "offline_grace" | "expired" | "revoked" | "invalid";
export type CapabilityEntitlementStatus = "allowed" | "locked" | "limit_reached";

export interface CapabilityEntitlement {
  capability_id: string;
  status: CapabilityEntitlementStatus;
  remaining?: number;
  limit_bucket?: string;
  reset_at?: string;
}

export interface ProtectedWorkerStatus {
  worker_status: "unavailable" | "ready" | "locked" | "incompatible" | "degraded";
  capsule_status: "missing" | "encrypted" | "verified" | "mounted" | "rollback";
  version?: string;
  compatibility?: string;
}

export interface EntitlementRecoveryAction {
  kind: "manage_license" | "purchase" | "refresh" | "doctor";
  label: string;
  href: string;
}

export interface CanonicalEntitlementProjection {
  schema: typeof ENTITLEMENT_PROJECTION_SCHEMA;
  source: "uiai_authority" | "focusa_license_broker";
  verified: true;
  product: "uiai-engine";
  state: EntitlementState;
  capabilities: CapabilityEntitlement[];
  recovery_actions: EntitlementRecoveryAction[];
  protected_worker: ProtectedWorkerStatus;
  local_artifacts: { access: "preserved"; evidence: "preserved" };
  observed_at: string;
}

export interface EntitlementDenial {
  schema: typeof ENTITLEMENT_DENIAL_SCHEMA;
  code: "license_required" | "evaluation_limit_reached" | "evaluation_expired";
  capability_id: string;
  message: string;
  recovery: EntitlementRecoveryAction;
  retryable: boolean;
  local_artifacts: "preserved";
}

function record(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`${label} must be an object`);
  return value as Record<string, unknown>;
}

function exactKeys(value: Record<string, unknown>, keys: readonly string[], label: string): void {
  const allowed = new Set(keys);
  const unsupported = Object.keys(value).filter((key) => !allowed.has(key));
  if (unsupported.length) throw new Error(`${label} contains unsupported fields: ${unsupported.join(", ")}`);
}

function text(value: Record<string, unknown>, key: string, label: string): string {
  const current = value[key];
  if (typeof current !== "string" || !current.trim()) throw new Error(`${label}.${key} is required`);
  return current;
}

function localHref(value: string, label: string): string {
  if (!value.startsWith("/") || value.startsWith("//") || value.includes("\\") || /[\u0000-\u001f]/.test(value)) throw new Error(`${label} must be a local Cockpit path`);
  return value;
}

function opaque(value: string, label: string): string {
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/.test(value)) throw new Error(`${label} must be opaque`);
  return value;
}

function timestamp(value: string, label: string): string {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/.test(value) || Number.isNaN(Date.parse(value))) throw new Error(`${label} must be RFC3339 UTC`);
  return value;
}

export function parseEntitlementProjection(value: unknown): CanonicalEntitlementProjection {
  const item = record(value, "entitlement");
  exactKeys(item, ["schema", "source", "verified", "product", "state", "capabilities", "recovery_actions", "protected_worker", "local_artifacts", "observed_at"], "entitlement");
  if (item.schema !== ENTITLEMENT_PROJECTION_SCHEMA || !["uiai_authority", "focusa_license_broker"].includes(String(item.source)) || item.verified !== true || item.product !== "uiai-engine") throw new Error("entitlement authority is not canonical");
  const state = text(item, "state", "entitlement") as EntitlementState;
  if (!["unactivated", "active_evaluation", "active_paid", "offline_grace", "expired", "revoked", "invalid"].includes(state)) throw new Error("unsupported entitlement state");
  if (!Array.isArray(item.capabilities) || !Array.isArray(item.recovery_actions)) throw new Error("entitlement capabilities and recovery actions are required");

  const capabilities = item.capabilities.map((value, index): CapabilityEntitlement => {
    const capability = record(value, `capabilities[${index}]`);
    exactKeys(capability, ["capability_id", "status", "remaining", "limit_bucket", "reset_at"], `capabilities[${index}]`);
    const status = text(capability, "status", `capabilities[${index}]`) as CapabilityEntitlementStatus;
    if (!["allowed", "locked", "limit_reached"].includes(status)) throw new Error("unsupported capability entitlement status");
    const result: CapabilityEntitlement = { capability_id: opaque(text(capability, "capability_id", `capabilities[${index}]`), "capability_id"), status };
    if (capability.remaining !== undefined) {
      if (!Number.isInteger(capability.remaining) || Number(capability.remaining) < 0) throw new Error("remaining must be a non-negative integer");
      result.remaining = Number(capability.remaining);
    }
    if (capability.limit_bucket !== undefined) result.limit_bucket = opaque(text(capability, "limit_bucket", `capabilities[${index}]`), "limit_bucket");
    if (capability.reset_at !== undefined) result.reset_at = timestamp(text(capability, "reset_at", `capabilities[${index}]`), "reset_at");
    return result;
  });

  const recoveryActions = item.recovery_actions.map((value, index): EntitlementRecoveryAction => {
    const action = record(value, `recovery_actions[${index}]`);
    exactKeys(action, ["kind", "label", "href"], `recovery_actions[${index}]`);
    const kind = text(action, "kind", `recovery_actions[${index}]`) as EntitlementRecoveryAction["kind"];
    if (!["manage_license", "purchase", "refresh", "doctor"].includes(kind)) throw new Error("unsupported recovery action");
    return { kind, label: text(action, "label", `recovery_actions[${index}]`), href: localHref(text(action, "href", `recovery_actions[${index}]`), "recovery href") };
  });

  const worker = record(item.protected_worker, "protected_worker");
  exactKeys(worker, ["worker_status", "capsule_status", "version", "compatibility"], "protected_worker");
  const workerStatus = text(worker, "worker_status", "protected_worker") as ProtectedWorkerStatus["worker_status"];
  const capsuleStatus = text(worker, "capsule_status", "protected_worker") as ProtectedWorkerStatus["capsule_status"];
  if (!["unavailable", "ready", "locked", "incompatible", "degraded"].includes(workerStatus) || !["missing", "encrypted", "verified", "mounted", "rollback"].includes(capsuleStatus)) throw new Error("unsupported protected worker or capsule status");
  const protectedWorker: ProtectedWorkerStatus = { worker_status: workerStatus, capsule_status: capsuleStatus };
  if (worker.version !== undefined) protectedWorker.version = text(worker, "version", "protected_worker");
  if (worker.compatibility !== undefined) protectedWorker.compatibility = text(worker, "compatibility", "protected_worker");

  const artifacts = record(item.local_artifacts, "local_artifacts");
  exactKeys(artifacts, ["access", "evidence"], "local_artifacts");
  if (artifacts.access !== "preserved" || artifacts.evidence !== "preserved") throw new Error("canonical entitlement must preserve local artifacts and Evidence");

  return {
    schema: ENTITLEMENT_PROJECTION_SCHEMA,
    source: item.source as CanonicalEntitlementProjection["source"], verified: true, product: "uiai-engine", state,
    capabilities, recovery_actions: recoveryActions, protected_worker: protectedWorker,
    local_artifacts: { access: "preserved", evidence: "preserved" },
    observed_at: timestamp(text(item, "observed_at", "entitlement"), "observed_at"),
  };
}

let currentEntitlement: CanonicalEntitlementProjection | null = null;

export function installEntitlementProjection(value: unknown): CanonicalEntitlementProjection | null {
  try { currentEntitlement = parseEntitlementProjection(value); }
  catch { currentEntitlement = null; }
  return currentEntitlement;
}

export function currentEntitlementProjection(): CanonicalEntitlementProjection | null {
  return currentEntitlement;
}

export function entitlementFromHost(): CanonicalEntitlementProjection | null {
  if (typeof window === "undefined") return null;
  const candidate = window.__UIAI_COCKPIT_CONTRACTS__?.entitlement;
  return candidate === undefined ? null : installEntitlementProjection(candidate);
}

function fallbackRecovery(state: CanonicalEntitlementProjection | null): EntitlementRecoveryAction {
  return state?.recovery_actions[0] || { kind: "manage_license", label: "Manage entitlement", href: "/nodes-services?view=uiai-engine" };
}

export function requireCapabilityEntitlement(capabilityId: string): void {
  const entitlement = currentEntitlement;
  const feature = entitlement?.capabilities.find((capability) => capability.capability_id === capabilityId);
  const active = entitlement && ["active_evaluation", "active_paid", "offline_grace"].includes(entitlement.state);
  if (active && feature?.status === "allowed") return;
  const code: EntitlementDenial["code"] = entitlement?.state === "expired"
    ? "evaluation_expired"
    : feature?.status === "limit_reached" ? "evaluation_limit_reached" : "license_required";
  const denial: EntitlementDenial = {
    schema: ENTITLEMENT_DENIAL_SCHEMA, code, capability_id: capabilityId,
    message: code === "evaluation_limit_reached" ? "The signed capability limit has been reached." : code === "evaluation_expired" ? "The signed evaluation has expired." : "A verified UIAI entitlement is required.",
    recovery: fallbackRecovery(entitlement), retryable: code !== "license_required", local_artifacts: "preserved",
  };
  throw new EntitlementDeniedError(denial);
}

export class EntitlementDeniedError extends Error {
  constructor(public readonly denial: EntitlementDenial) {
    super(denial.message);
    this.name = "EntitlementDeniedError";
  }
}
