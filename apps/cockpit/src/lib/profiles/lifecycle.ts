/**
 * T005-07.04 — Profile health and lifecycle
 * Active / unavailable / expired / revoked / conflict transitions are evidence-backed.
 */

import { loadProfileMetadata, saveProfileMetadata, type DaemonProfileMetadata } from "./metadata";
import { __setProfileExpired, __setProfileRevoked, __setProfileLocked } from "./resolver";

export type ProfileHealthStatus = "active" | "unavailable" | "expired" | "revoked" | "conflict";

export interface LifecycleTransition {
  profileId: string;
  from: ProfileHealthStatus;
  to: ProfileHealthStatus;
  reason: string;
  evidenceRef: string;
  at: string; // RFC3339
}

const transitions: LifecycleTransition[] = [];

const ALLOWED: Record<ProfileHealthStatus, ProfileHealthStatus[]> = {
  active: ["unavailable", "expired", "revoked", "conflict"],
  unavailable: ["active", "expired", "revoked", "conflict"],
  expired: ["revoked", "conflict"],
  revoked: ["conflict"],
  conflict: ["active", "revoked"],
};

function nowRfc3339(): string {
  return new Date().toISOString();
}

function assertTransition(from: ProfileHealthStatus, to: ProfileHealthStatus): void {
  if (from === to) return;
  const allowed = ALLOWED[from] ?? [];
  if (!allowed.includes(to)) {
    throw new Error(`lifecycle transition ${from} -> ${to} not allowed`);
  }
}

function statusFromMetadata(m: DaemonProfileMetadata): ProfileHealthStatus {
  // Map stored metadata source/active flag to health. Default active.
  // If explicit lifecycle sets expired/revoked/conflict via resolver flags, reflect that.
  // For now, derive from in-memory flags would require cross-check, but we store status in metadata extension via scopes trick.
  // Simplify: if metadata has custom field _status (test injection) else active.
  const raw = (m as unknown as Record<string, unknown>)._status as string | undefined;
  if (raw && ["active", "unavailable", "expired", "revoked", "conflict"].includes(raw)) return raw as ProfileHealthStatus;
  return "active";
}

export async function getProfileHealth(profileId: string): Promise<ProfileHealthStatus> {
  const m = await loadProfileMetadata(profileId);
  if (!m) throw new Error(`profile ${profileId} not found`);
  return statusFromMetadata(m);
}

export async function transitionProfileHealth(
  profileId: string,
  to: ProfileHealthStatus,
  evidenceRef: string,
  reason: string,
): Promise<LifecycleTransition> {
  if (!evidenceRef || !reason) throw new Error("evidenceRef and reason required");
  const m = await loadProfileMetadata(profileId);
  if (!m) throw new Error(`profile ${profileId} not found`);
  const from = statusFromMetadata(m);
  assertTransition(from, to);

  // Persist status as extension field (non-secret) for audit
  const nextMeta = { ...(m as unknown as Record<string, unknown>), _status: to, lastSeenAt: nowRfc3339() } as unknown as DaemonProfileMetadata;
  await saveProfileMetadata(nextMeta);

  // Sync resolver flags for typed unavailable states
  __setProfileExpired(profileId, to === "expired");
  __setProfileRevoked(profileId, to === "revoked");
  __setProfileLocked(profileId, to === "unavailable");

  const t: LifecycleTransition = {
    profileId,
    from,
    to,
    reason,
    evidenceRef,
    at: nowRfc3339(),
  };
  transitions.push(t);
  return t;
}

export function listLifecycleTransitions(profileId?: string): LifecycleTransition[] {
  if (profileId) return transitions.filter((t) => t.profileId === profileId);
  return [...transitions];
}

export function __resetLifecycleForTests(): void {
  transitions.length = 0;
}
