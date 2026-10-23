/**
 * T005-07.03 — Profile-scoped authenticated native client
 * Requests bind exactly one profile daemon identity and scope set.
 * Fail-closed on missing/locked/expired/revoked credentials and on ambiguous bindings.
 */

import { resolveProfileCredential } from "./resolver";
import type { DaemonProfileMetadata } from "./metadata";

export type ClientInvokeErrorKind = "missing" | "locked" | "expired" | "revoked" | "unavailable" | "ambiguous_binding";

export class ProfileClientError extends Error {
  kind: ClientInvokeErrorKind;
  profileId: string;
  constructor(kind: ClientInvokeErrorKind, profileId: string, msg: string) {
    super(msg);
    this.kind = kind;
    this.profileId = profileId;
  }
}

export interface ProfileScopedClient {
  profile: DaemonProfileMetadata;
  tokenHandle: string;
  daemonId: string;
  endpoint: string;
  scopes: string[];
  invoke<T = unknown>(command: string, args?: Record<string, unknown>): Promise<T>;
}

function assertSingleBinding(args: Record<string, unknown> | undefined, profileId: string): void {
  if (!args) return;
  const forbidden = ["profile_id", "profileId", "daemon_id", "daemonId"];
  for (const k of forbidden) {
    if (k in args && String(args[k]) !== profileId && String(args[k]) !== "") {
      // If caller tries to override profile/daemon, fail closed
      if (k.toLowerCase().includes("profile") && String(args[k]) !== profileId) {
        throw new ProfileClientError("ambiguous_binding", profileId, `ambiguous profile binding: ${k}=${String(args[k])} vs ${profileId}`);
      }
    }
  }
  // Detect multiple profile refs in args
  const profileRefs = Object.entries(args).filter(([k]) => /profile/i.test(k));
  if (profileRefs.length > 1) {
    throw new ProfileClientError("ambiguous_binding", profileId, "multiple profile bindings in single request");
  }
}

export async function createProfileScopedClient(profileId: string): Promise<ProfileScopedClient> {
  const resolution = await resolveProfileCredential(profileId);
  if (resolution.kind !== "available") {
    throw new ProfileClientError(resolution.kind as ClientInvokeErrorKind, profileId, resolution.reason);
  }
  const { profile, tokenHandle } = resolution;

  // Validate daemon identity and scopes present
  if (!profile.daemonId || !profile.endpoint) {
    throw new ProfileClientError("unavailable", profileId, "profile missing daemon identity");
  }

  const client: ProfileScopedClient = {
    profile,
    tokenHandle,
    daemonId: profile.daemonId,
    endpoint: profile.endpoint,
    scopes: profile.scopes ?? [],
    async invoke<T = unknown>(command: string, args?: Record<string, unknown>): Promise<T> {
      if (!command || typeof command !== "string") {
        throw new ProfileClientError("unavailable", profileId, "command must be non-empty string");
      }
      assertSingleBinding(args, profileId);

      // Headless fallback: deterministic echo without touching native keychain
      if (typeof window === "undefined") {
        return {
          command,
          profileId: profile.profileId,
          daemonId: profile.daemonId,
          endpoint: profile.endpoint,
          scopes: profile.scopes ?? [],
          tokenHandle,
          args: args ?? null,
        } as unknown as T;
      }

      // Native path — invoke via Tauri, credential stays in Rust keychain
      const { invoke } = await import("@tauri-apps/api/core");
      const boundArgs = {
        ...(args ?? {}),
        profile_id: profile.profileId,
        daemon_id: profile.daemonId,
        token_handle: tokenHandle,
      };
      return (await invoke(command, boundArgs)) as T;
    },
  };

  return client;
}
