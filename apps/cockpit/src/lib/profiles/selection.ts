/**
 * T005-07.05 — Active profile selection
 * Selection is explicit, durable, reversible, and cannot cross-bind credentials.
 */

import { loadProfileMetadata, type DaemonProfileMetadata } from "./metadata";
import { getProfileHealth } from "./lifecycle";

const ACTIVE_KEY = "activeProfileId";
const ACTIVE_FILE = "cockpit-active-profile.json";

let memoryActive: string | null = null;
let tauriStore: unknown | null = null;
let tauriStoreTried = false;

async function getTauriStore(): Promise<unknown | null> {
  if (typeof window === "undefined") return null;
  if (tauriStore) return tauriStore;
  if (tauriStoreTried) return null;
  tauriStoreTried = true;
  try {
    const mod: unknown = (await import("@tauri-apps/plugin-store").catch(() => null)) as unknown;
    const ms = mod as { LazyStore?: new (p: string) => unknown };
    if (!ms?.LazyStore) return null;
    tauriStore = new ms.LazyStore(ACTIVE_FILE);
    return tauriStore;
  } catch {
    return null;
  }
}

export async function getActiveProfileId(): Promise<string | null> {
  const store = (await getTauriStore()) as { get?: (k: string) => Promise<unknown> } | null;
  if (store?.get) {
    try {
      const v = (await store.get(ACTIVE_KEY)) as string | null;
      if (typeof v === "string" && v) return v;
    } catch {}
  }
  return memoryActive;
}

export async function getActiveProfile(): Promise<DaemonProfileMetadata | null> {
  const id = await getActiveProfileId();
  if (!id) return null;
  return (await loadProfileMetadata(id)) ?? null;
}

export async function setActiveProfileId(profileId: string): Promise<DaemonProfileMetadata> {
  if (!profileId || typeof profileId !== "string") throw new Error("profileId required");
  const meta = await loadProfileMetadata(profileId);
  if (!meta) throw new Error(`profile ${profileId} not found`);

  const health = await getProfileHealth(profileId);
  if (health !== "active") {
    throw new Error(`cannot select profile ${profileId} with health ${health}`);
  }

  const store = (await getTauriStore()) as { set?: (k: string, v: unknown) => Promise<void>; save?: () => Promise<void> } | null;
  if (store?.set && store?.save) {
    await store.set(ACTIVE_KEY, profileId);
    await store.save();
  }
  memoryActive = profileId;
  return meta;
}

export async function clearActiveProfile(): Promise<void> {
  const store = (await getTauriStore()) as { delete?: (k: string) => Promise<void>; save?: () => Promise<void> } | null;
  if (store?.delete && store?.save) {
    try {
      await store.delete(ACTIVE_KEY);
      await store.save();
    } catch {}
  }
  memoryActive = null;
}

export function __resetSelectionForTests(): void {
  memoryActive = null;
}
