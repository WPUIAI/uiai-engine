/**
 * T005-07.02 — Bind profiles to native token handles
 * Missing / locked / expired / revoked credentials produce typed unavailable states.
 * Never returns secret values — only opaque handles. Headless fallback is deterministic.
 */

import { loadProfileMetadata, type DaemonProfileMetadata } from "./metadata";
import { loadSecret } from "../secure-store";

export type CredentialUnavailableKind = "missing" | "locked" | "expired" | "revoked" | "unavailable";

export type ProfileCredentialResolution =
  | { kind: "available"; profile: DaemonProfileMetadata; tokenHandle: string }
  | { kind: CredentialUnavailableKind; profile: DaemonProfileMetadata | null; reason: string };

const TOKEN_HANDLE_PREFIX = "profile-token:";

function tokenHandleFor(profileId: string): string {
  return `${TOKEN_HANDLE_PREFIX}${profileId}`;
}

// Test-controlled state for headless deterministic proofs
const locked = new Set<string>();
const expired = new Set<string>();
const revoked = new Set<string>();

export function __setProfileLocked(profileId: string, v: boolean): void {
  if (v) locked.add(profileId);
  else locked.delete(profileId);
}
export function __setProfileExpired(profileId: string, v: boolean): void {
  if (v) expired.add(profileId);
  else expired.delete(profileId);
}
export function __setProfileRevoked(profileId: string, v: boolean): void {
  if (v) revoked.add(profileId);
  else revoked.delete(profileId);
}
export function __resetResolverForTests(): void {
  locked.clear();
  expired.clear();
  revoked.clear();
}

/**
 * Resolve a profile's credential handle without exposing secret material.
 * - missing: no metadata or no secret
 * - locked: credential store reports locked (test-injected)
 * - expired: profile expired (test-injected)
 * - revoked: profile revoked (test-injected)
 * - unavailable: generic failure
 */
export async function resolveProfileCredential(profileId: string): Promise<ProfileCredentialResolution> {
  const profile = await loadProfileMetadata(profileId);
  if (!profile) {
    return { kind: "missing", profile: null, reason: "profile not found" };
  }

  // Typed unavailable states — checked before secret access, never leak secret
  if (revoked.has(profileId)) {
    return { kind: "revoked", profile, reason: "credential revoked" };
  }
  if (expired.has(profileId)) {
    return { kind: "expired", profile, reason: "credential expired" };
  }
  if (locked.has(profileId)) {
    return { kind: "locked", profile, reason: "credential store locked" };
  }

  // Native secure storage holds the actual secret; we only return the handle if present
  const handle = tokenHandleFor(profileId);
  try {
    const secret = await loadSecret(handle);
    if (secret === null || secret === "") {
      return { kind: "missing", profile, reason: "credential missing" };
    }
    // Do not return secret value — only handle
    return { kind: "available", profile, tokenHandle: handle };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (/locked/i.test(msg)) return { kind: "locked", profile, reason: msg };
    if (/revoked/i.test(msg)) return { kind: "revoked", profile, reason: msg };
    if (/expired/i.test(msg)) return { kind: "expired", profile, reason: msg };
    return { kind: "unavailable", profile, reason: msg || "credential unavailable" };
  }
}

export function __tokenHandleForTests(profileId: string): string {
  return tokenHandleFor(profileId);
}
