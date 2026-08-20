/**
 * T005-07.01 — Non-secret profile metadata repository
 * Atomically persists daemon profile metadata WITHOUT token material.
 * Tokens remain in native secure credential storage (Keychain/Secret Service/Credential Manager)
 * via Tauri secure-store; this repository never sees or stores them.
 *
 * Persistence: Tauri LazyStore "cockpit-profiles.json" (atomic file replace on save)
 * Headless/test fallback: in-memory Map with same API, no filesystem writes.
 */

const STORE_FILE = "cockpit-profiles.json";
const STORE_KEY = "profiles";

const TOKEN_LIKE_KEYS = /token|secret|authorization|api[_-]?key|password|pairing[_-]?code|bearer/i;

export interface DaemonProfileMetadata {
  profileId: string; // opaque, 1..160, no URL delimiters
  daemonId: string;
  endpoint: string; // validated bounded URL, no secret query
  displayName?: string;
  pairedAt: string; // RFC3339 UTC
  lastSeenAt?: string; // RFC3339 UTC
  scopes?: string[];
  source: "pairing" | "menubar" | "manual";
  active?: boolean;
}

const OPAQUE = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/;
const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/;

function assertOpaque(value: string, label: string): void {
  if (!OPAQUE.test(value) || value.includes("://") || /[\\/?#@]/.test(value)) {
    throw new Error(`${label} must be opaque 1..160 without URL delimiters`);
  }
}

function assertRfc3339(value: string, label: string): void {
  if (!RFC3339.test(value) || Number.isNaN(Date.parse(value))) {
    throw new Error(`${label} must be RFC3339 UTC`);
  }
}

function assertNoTokenMaterial(obj: unknown, label = "profile metadata"): void {
  if (!obj || typeof obj !== "object") return;
  for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
    if (TOKEN_LIKE_KEYS.test(k)) {
      throw new Error(`${label} must not contain token material: ${k}`);
    }
    if (typeof v === "string" && v.length >= 16 && /[A-Za-z0-9_\-]{20,}/.test(v) && TOKEN_LIKE_KEYS.test(label)) {
      throw new Error(`${label} must not contain token-shaped values`);
    }
    if (v && typeof v === "object") assertNoTokenMaterial(v, `${label}.${k}`);
  }
}

export function validateProfileMetadata(input: unknown): DaemonProfileMetadata {
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("profile metadata must be an object");
  const m = input as Record<string, unknown>;
  assertNoTokenMaterial(m, "profile metadata");

  const profileId = String(m.profileId ?? "");
  const daemonId = String(m.daemonId ?? "");
  const endpoint = String(m.endpoint ?? "");
  const pairedAt = String(m.pairedAt ?? "");
  const source = String(m.source ?? "");

  assertOpaque(profileId, "profileId");
  assertOpaque(daemonId, "daemonId");
  if (!endpoint || endpoint.includes("?") || endpoint.includes("#") || /\s/.test(endpoint)) {
    throw new Error("endpoint must be bounded URL without query/fragment");
  }
  try {
    const u = new URL(endpoint);
    if (u.protocol !== "http:" && u.protocol !== "https:") throw new Error("endpoint must be http(s)");
  } catch {
    throw new Error("endpoint must be valid http(s) URL");
  }
  assertRfc3339(pairedAt, "pairedAt");
  if (m.lastSeenAt !== undefined) assertRfc3339(String(m.lastSeenAt), "lastSeenAt");
  if (m.displayName !== undefined && typeof m.displayName !== "string") throw new Error("displayName must be string");
  if (m.scopes !== undefined) {
    if (!Array.isArray(m.scopes) || m.scopes.some((s) => typeof s !== "string" || !s)) throw new Error("scopes must be string[]");
  }
  if (!["pairing", "menubar", "manual"].includes(source)) throw new Error("source must be pairing|menubar|manual");

  return m as unknown as DaemonProfileMetadata;
}

type StoreShape = Record<string, DaemonProfileMetadata>;

// In-memory fallback for headless/CI
let memory: StoreShape = {};
let memoryDirty = false;

let tauriStore: unknown | null = null;
let tauriStoreTried = false;

async function getTauriStore(): Promise<unknown | null> {
  if (typeof window === "undefined") return null;
  if (tauriStore) return tauriStore;
  if (tauriStoreTried) return null;
  tauriStoreTried = true;
  try {
    const mod: unknown = await import("@tauri-apps/plugin-store").catch(() => null) as unknown;
    const ms = mod as { LazyStore?: new (path: string) => unknown };
    if (!ms?.LazyStore) return null;
    tauriStore = new ms.LazyStore(STORE_FILE);
    return tauriStore;
  } catch {
    return null;
  }
}

async function loadAll(): Promise<StoreShape> {
  const store = (await getTauriStore()) as { get?: (k: string) => Promise<unknown> } | null;
  if (store?.get) {
    try {
      const v = (await store.get(STORE_KEY)) as StoreShape | null;
      if (v && typeof v === "object" && !Array.isArray(v)) {
        // validate each entry, reject token material
        for (const entry of Object.values(v)) {
          validateProfileMetadata(entry);
        }
        memory = { ...v };
        return { ...v };
      }
    } catch {
      // fall through to memory on parse error
    }
  }
  return { ...memory };
}

async function persistAll(next: StoreShape): Promise<void> {
  // validate before persist — fail closed on token material
  for (const entry of Object.values(next)) {
    validateProfileMetadata(entry);
  }
  const store = (await getTauriStore()) as { set?: (k: string, v: unknown) => Promise<void>; save?: () => Promise<void> } | null;
  if (store?.set && store?.save) {
    await store.set(STORE_KEY, next);
    await store.save(); // Tauri store does atomic file replace
    memory = { ...next };
    return;
  }
  // headless fallback — atomic in-memory replace
  memory = { ...next };
  memoryDirty = true;
}

export async function saveProfileMetadata(meta: DaemonProfileMetadata): Promise<DaemonProfileMetadata> {
  const validated = validateProfileMetadata(meta);
  const all = await loadAll();
  const next = { ...all, [validated.profileId]: validated };
  await persistAll(next);
  return validated;
}

export async function loadProfileMetadata(profileId: string): Promise<DaemonProfileMetadata | null> {
  assertOpaque(profileId, "profileId");
  const all = await loadAll();
  return all[profileId] ?? null;
}

export async function listProfileMetadata(): Promise<DaemonProfileMetadata[]> {
  const all = await loadAll();
  return Object.values(all);
}

export async function removeProfileMetadata(profileId: string): Promise<boolean> {
  assertOpaque(profileId, "profileId");
  const all = await loadAll();
  if (!(profileId in all)) return false;
  const { [profileId]: _, ...rest } = all;
  await persistAll(rest as StoreShape);
  return true;
}

export function __resetMemoryForTests(): void {
  memory = {};
  memoryDirty = false;
}

export function __getMemorySnapshot(): StoreShape {
  return { ...memory };
}
