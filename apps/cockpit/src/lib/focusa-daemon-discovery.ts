import { discoverViaBonjour } from "$lib/bridge/tauri";

export const FOCUSA_DAEMON_HINTS_KEY = "uiai.cockpit.focusa_daemon_hints.v1";

export type FocusaDaemonSource = "loopback" | "bonjour" | "saved" | "environment";
export type FocusaDaemonLocation = "local" | "remote";

export interface FocusaDaemonConnection {
  baseUrl: string;
  source: FocusaDaemonSource;
  location: FocusaDaemonLocation;
  status: "connected" | "unavailable";
  latencyMs?: number;
  version?: string;
  nodeId?: string;
  machineId?: string;
  displayName?: string;
  paired?: boolean;
}

function normalizeBaseUrl(value: string): string | undefined {
  try {
    const parsed = new URL(value.trim());
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return undefined;
    parsed.pathname = "";
    parsed.search = "";
    parsed.hash = "";
    return parsed.toString().replace(/\/$/, "");
  } catch {
    return undefined;
  }
}

export function readSavedFocusaDaemonHints(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const parsed = JSON.parse(window.localStorage.getItem(FOCUSA_DAEMON_HINTS_KEY) || "[]");
    return Array.isArray(parsed) ? parsed.flatMap((value) => typeof value === "string" ? [normalizeBaseUrl(value)] : []).filter((value): value is string => Boolean(value)).slice(0, 6) : [];
  } catch {
    return [];
  }
}

export function saveFocusaDaemonHints(values: string[]): string[] {
  const normalized = [...new Set(values.flatMap((value) => normalizeBaseUrl(value) || []))].slice(0, 6);
  if (typeof window !== "undefined") window.localStorage.setItem(FOCUSA_DAEMON_HINTS_KEY, JSON.stringify(normalized));
  return normalized;
}

async function probe(baseUrl: string, source: FocusaDaemonSource): Promise<FocusaDaemonConnection> {
  const hostname = new URL(baseUrl).hostname;
  const location: FocusaDaemonLocation = hostname === "127.0.0.1" || hostname === "localhost" || hostname.endsWith(".local") ? "local" : "remote";
  const started = performance.now();
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 1800);
  try {
    const response = await fetch(`${baseUrl}/v1/health`, { signal: controller.signal, cache: "no-store" });
    if (!response.ok) throw new Error(`health_${response.status}`);
    const body = await response.json().catch(() => ({})) as Record<string, unknown>;
    const node = body.node && typeof body.node === "object" ? body.node as Record<string, unknown> : body;
    const nodeId = typeof node.node_id === "string" ? node.node_id : typeof node.instance_id === "string" ? node.instance_id : undefined;
    const machineId = typeof node.machine_id === "string" ? node.machine_id : undefined;
    const displayName = typeof node.display_name === "string" ? node.display_name : typeof node.host === "string" ? node.host : location === "local" ? "Local Focusa" : "Remote Focusa";
    const pairingStatus = body.pairing && typeof body.pairing === "object" ? (body.pairing as Record<string, unknown>).status : undefined;
    const paired = body.paired === true || node.paired === true || pairingStatus === "paired";
    return { baseUrl, source, location, status: "connected", latencyMs: Math.round(performance.now() - started), version: typeof body.version === "string" ? body.version : undefined, nodeId, machineId, displayName, paired };
  } catch {
    return { baseUrl, source, location, status: "unavailable" };
  } finally {
    window.clearTimeout(timeout);
  }
}

/** Discover every truthful candidate; discovery never grants scope or mutation authority. */
export async function discoverFocusaDaemons(): Promise<FocusaDaemonConnection[]> {
  const candidates = new Map<string, FocusaDaemonSource>();
  candidates.set("http://127.0.0.1:8787", "loopback");
  for (const value of readSavedFocusaDaemonHints()) candidates.set(value, "saved");
  const runtimeEnv = (import.meta as unknown as { env?: Record<string, string | undefined> }).env;
  const environmentHint = normalizeBaseUrl(runtimeEnv?.VITE_FOCUSA_DAEMON_URL || "");
  if (environmentHint) candidates.set(environmentHint, "environment");
  const bonjour = await discoverViaBonjour(1);
  const bonjourUrl = bonjour?.url ? normalizeBaseUrl(bonjour.url) : undefined;
  if (bonjourUrl) candidates.set(bonjourUrl, "bonjour");
  return Promise.all([...candidates].map(([url, source]) => probe(url, source)));
}

export function focusaDaemonSummary(connections: FocusaDaemonConnection[]): string {
  const connected = connections.filter((item) => item.status === "connected");
  const local = connected.some((item) => item.location === "local");
  const remote = connected.some((item) => item.location === "remote");
  if (local && remote) return "Focusa · Local + VPS";
  if (local) return "Focusa · Local";
  if (remote) return "Focusa · VPS";
  return connections.length ? "Focusa · Unavailable" : "Focusa · Discovering";
}
