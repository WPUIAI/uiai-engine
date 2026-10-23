import { DEFAULT_ENGINE_URL } from "$lib/engine-client";

export const ENGINE_HINTS_KEY = "uiai.cockpit.engine_hints.v1";
export interface EngineConnection { baseUrl: string; status: "connected" | "unavailable"; latencyMs?: number; service?: string; browserless?: boolean; source: "loopback" | "saved" | "environment"; }

function normalize(value: string): string | undefined {
  try { const url = new URL(value.trim()); if (!/^https?:$/.test(url.protocol)) return; url.pathname = ""; url.search = ""; url.hash = ""; return url.toString().replace(/\/$/, ""); } catch { return; }
}

export function readEngineHints(): string[] {
  if (typeof window === "undefined") return [];
  try { const values = JSON.parse(localStorage.getItem(ENGINE_HINTS_KEY) || "[]"); return Array.isArray(values) ? values.map(String).map(normalize).filter((v): v is string => Boolean(v)).slice(0, 12) : []; } catch { return []; }
}

export function saveEngineHints(values: string[]): string[] {
  const normalized = [...new Set(values.map(normalize).filter((v): v is string => Boolean(v)))].slice(0, 12);
  if (typeof window !== "undefined") localStorage.setItem(ENGINE_HINTS_KEY, JSON.stringify(normalized));
  return normalized;
}

async function probe(baseUrl: string, source: EngineConnection["source"]): Promise<EngineConnection> {
  const controller = new AbortController(); const started = performance.now(); const timer = window.setTimeout(() => controller.abort(), 1800);
  try {
    const response = await fetch(`${baseUrl}/health`, { cache: "no-store", signal: controller.signal });
    const body = await response.json().catch(() => ({})) as Record<string, unknown>;
    if (!response.ok || body.service !== "uiai-engine" || body.status !== "healthy") throw new Error("not UIAI Engine");
    return { baseUrl, source, status: "connected", latencyMs: Math.round(performance.now() - started), service: String(body.service), browserless: body.browserless === true };
  } catch { return { baseUrl, source, status: "unavailable" }; } finally { window.clearTimeout(timer); }
}

export async function discoverEngines(): Promise<EngineConnection[]> {
  if (typeof window === "undefined") return [];
  const candidates = new Map<string, EngineConnection["source"]>();
  candidates.set(DEFAULT_ENGINE_URL, "loopback"); candidates.set("http://localhost:7456", "loopback");
  for (const value of readEngineHints()) candidates.set(value, "saved");
  const environment = normalize(String((window as unknown as Record<string, unknown>).__UIAI_ENGINE_URL__ || ""));
  if (environment) candidates.set(environment, "environment");
  const results = await Promise.all([...candidates].map(([url, source]) => probe(url, source)));
  return results.sort((a, b) => Number(b.status === "connected") - Number(a.status === "connected") || (a.latencyMs || 9999) - (b.latencyMs || 9999));
}

export function selectBestEngine(connections: EngineConnection[]): EngineConnection | undefined {
  return connections.filter((item) => item.status === "connected").sort((a, b) => (a.latencyMs || 9999) - (b.latencyMs || 9999))[0];
}
