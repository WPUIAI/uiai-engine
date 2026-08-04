// §17 — bridge between SvelteKit and Tauri commands.
// On web (no Tauri), commands gracefully no-op (so the SPA renders without crashing).

import { isTauri } from "@tauri-apps/api/core";

export interface BridgeStartResult {
  callbackUrl?: string;
}

export interface BridgeCompletionPayload {
  protocol: "focusa-connect-v1";
  role: "mac_completion_payload";
  mac_completion_payload: string;
}

export interface BonjourDiscovery {
  url: string;
  host: string;
  port: number;
}

export interface FocusaSiblingManifest {
  schema: "focusa.app.manifest.v2";
  app: string;
  version: string;
  channel: string;
  protocols: Record<string, string>;
  capabilities: string[];
}

export async function startBridgeCallback(nonce: string): Promise<BridgeStartResult> {
  if (!isTauri()) return {};
  try {
    const invoke = (await import("@tauri-apps/api/core")).invoke;
    const url = await invoke<string | null>("focusa_start_bridge_callback", { nonce });
    return { callbackUrl: url ?? undefined };
  } catch (err) {
    console.error("startBridgeCallback failed:", err);
    return {};
  }
}

export async function takeBridgeCompletion(nonce: string): Promise<string | null> {
  if (!isTauri()) return null;
  try {
    const invoke = (await import("@tauri-apps/api/core")).invoke;
    return await invoke<string | null>("focusa_take_bridge_completion", { nonce });
  } catch (err) {
    console.error("takeBridgeCompletion failed:", err);
    return null;
  }
}

export async function clearBridge(nonce: string): Promise<void> {
  if (!isTauri()) return;
  try {
    const invoke = (await import("@tauri-apps/api/core")).invoke;
    await invoke("focusa_clear_bridge", { nonce });
  } catch (err) {
    console.error("clearBridge failed:", err);
  }
}

export async function cockpitManifestEndpoint(): Promise<string | null> {
  if (!isTauri()) return null;
  const invoke = (await import("@tauri-apps/api/core")).invoke;
  return invoke<string>("cockpit_focusa_manifest_endpoint");
}

export async function fetchFocusaSiblingManifest(endpoint: string): Promise<FocusaSiblingManifest | null> {
  if (!isTauri()) return null;
  try {
    const invoke = (await import("@tauri-apps/api/core")).invoke;
    return await invoke<FocusaSiblingManifest>("cockpit_fetch_focusa_manifest", { endpoint });
  } catch (err) {
    console.error("fetchFocusaSiblingManifest failed:", err);
    return null;
  }
}

export async function discoverViaBonjour(timeoutSecs = 2): Promise<BonjourDiscovery | null> {
  if (!isTauri()) return null;
  try {
    const invoke = (await import("@tauri-apps/api/core")).invoke;
    return await invoke<BonjourDiscovery | null>("focusa_discover_via_bonjour", {
      timeoutSecs,
    });
  } catch (err) {
    console.error("discoverViaBonjour failed:", err);
    return null;
  }
}