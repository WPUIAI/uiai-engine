// Secure secret storage for pairing tokens / cloud tokens / AI dev keys.
// Tauri OS keychain / secure store only — never localStorage for secrets.
// localStorage remains allowed only for non-secret metadata + preview.
let memory = new Map<string, string>();
let tauriStore: any = null;

async function getTauriStore(): Promise<any | null> {
  if (tauriStore) return tauriStore;
  try {
    const mod: any = await import("@tauri-apps/plugin-store").catch(() => null);
    if (!mod?.LazyStore) return null;
    tauriStore = new mod.LazyStore("cockpit-secrets.json");
    return tauriStore;
  } catch {
    return null;
  }
}

export async function saveSecret(key: string, value: string): Promise<void> {
  const store = await getTauriStore();
  if (store) {
    await store.set(key, value);
    await store.save();
    return;
  }
  memory.set(key, value);
}

export async function loadSecret(key: string): Promise<string | null> {
  const store: any = await getTauriStore();
  if (store) {
    const v = await store.get(key);
    return (v as string | null) ?? null;
  }
  return memory.get(key) ?? null;
}

export async function clearSecret(key: string): Promise<void> {
  const store = await getTauriStore();
  if (store) {
    await store.delete(key);
    await store.save();
    return;
  }
  memory.delete(key);
}

export function redactedPreview(value: string | null): string {
  if (!value) return "";
  if (value.length <= 4) return "••••";
  return value.slice(0, 2) + "••••" + value.slice(-2);
}

export function assertNoSecretsInLocalStorage(): string[] {
  const leaks: string[] = [];
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i)!;
      const v = localStorage.getItem(k) || "";
      if (/token|secret|authorization|api_key|pairing.*code/i.test(k) && v.length > 8) {
        if (/[A-Za-z0-9_\-]{16,}/.test(v)) leaks.push(k);
      }
    }
  } catch {}
  return leaks;
}
