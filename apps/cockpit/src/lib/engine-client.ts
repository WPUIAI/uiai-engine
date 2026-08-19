import { requireCapabilityEntitlement } from "./contracts/entitlement";

export const DEFAULT_ENGINE_URL = "http://127.0.0.1:7456";

export interface EngineHealth { status: string; service?: string; uptime?: number; browserless?: boolean; }
export interface BrowserHealth { status?: string; browser_state?: string; browser_alive?: boolean; active_pages?: number; max_pages?: number; available_pages?: number; pressure?: Record<string, unknown>; [key: string]: unknown; }
export interface BrowserSession { id: string; url: string; title?: string; width?: number; height?: number; created_at?: string; last_used?: string; nav_count?: number; snap_count?: number; }
export interface ScreenshotResult { screenshot?: string; format?: string; width?: number; height?: number; size?: number; url?: string; title?: string; duration_ms?: number; artifact_url?: string; }
export interface SearchResult { title: string; url: string; description?: string; source?: string; age?: string; rank?: number; evidence_ref?: string; }

export class EngineRequestError extends Error {
  constructor(public readonly status: number, message: string) { super(message); this.name = "EngineRequestError"; }
}

export function engineUrl(): string {
  if (typeof window === "undefined") return DEFAULT_ENGINE_URL;
  return window.localStorage.getItem("uiai.engine.url") || DEFAULT_ENGINE_URL;
}

export function selectEngineUrl(baseUrl: string): void {
  if (typeof window !== "undefined") window.localStorage.setItem("uiai.engine.url", baseUrl.replace(/\/$/, ""));
}

export function savedScope(): { project_root?: string; continuity_id?: string; workpoint_id?: string; workstream_key?: string } {
  if (typeof window === "undefined") return {};
  const project_root = window.localStorage.getItem("uiai.scope.project_root") || undefined;
  const continuity_id = window.localStorage.getItem("uiai.scope.continuity_id") || undefined;
  const workpoint_id = window.localStorage.getItem("uiai.scope.workpoint_id") || undefined;
  const stored = window.localStorage.getItem("uiai.scope.workstream_key") || undefined;
  const workstream_key = stored ?? (project_root && continuity_id ? `${project_root.replace(/\/+$/, "")}::${continuity_id}` : undefined);
  return { project_root, continuity_id, workpoint_id, workstream_key };
}

export async function engineRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 120_000);
  let response: Response;
  try {
    response = await fetch(`${engineUrl()}${path}`, { ...init, signal: controller.signal, headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) } });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === "AbortError") throw new EngineRequestError(408, "The engine request timed out while starting or reading the browser.");
    throw cause;
  } finally {
    window.clearTimeout(timeout);
  }
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new EngineRequestError(response.status, body?.error || `Engine request failed (${response.status})`);
  return body as T;
}

export const engineClient = {
  health: () => engineRequest<EngineHealth>("/health"),
  browserHealth: () => engineRequest<BrowserHealth>("/api/health/browser"),
  sessions: async () => (await engineRequest<{ sessions: BrowserSession[]; count: number; max: number }>("/api/session/")).sessions,
  openSession: (url: string, scope = savedScope()) => {
    requireCapabilityEntitlement("uiai.browser.session.create");
    return engineRequest<{ session: BrowserSession; screenshot: string; size: { width: number; height: number }; fpv_share?: unknown }>("/api/session/", { method: "POST", body: JSON.stringify({ url, width: 1280, height: 800, project_root: scope.project_root, continuity_id: scope.continuity_id, workpoint_id: scope.workpoint_id, workstream_key: (scope as Record<string,string|undefined>).workstream_key ?? (scope.project_root && scope.continuity_id ? `${scope.project_root.replace(/\/+$/, "")}::${scope.continuity_id}` : undefined) }) });
  },
  closeSession: (id: string) => {
    requireCapabilityEntitlement("uiai.browser.session.control");
    return engineRequest<{ status: string; id: string }>(`/api/session/${encodeURIComponent(id)}`, { method: "DELETE" });
  },
  diagnostics: (id: string) => engineRequest<Record<string, unknown>>(`/api/session/${encodeURIComponent(id)}/diagnostics`),
  navigate: (id: string, url: string) => {
    requireCapabilityEntitlement("uiai.browser.session.control");
    return engineRequest<ScreenshotResult>(`/api/session/${encodeURIComponent(id)}/navigate`, { method: "POST", body: JSON.stringify({ url }) });
  },
  screenshot: (id: string) => {
    requireCapabilityEntitlement("uiai.browser.screenshot.execute");
    return engineRequest<ScreenshotResult>(`/api/session/${encodeURIComponent(id)}/screenshot`, { method: "POST", body: JSON.stringify({ format: "jpeg", quality: 72 }) });
  },
  sourceMarkdown: (url: string) => {
    requireCapabilityEntitlement("uiai.source.markdown.execute");
    return engineRequest<{ markdown?: string; title?: string; source?: Record<string, unknown>; focusa?: Record<string, unknown> }>("/api/markdown/", { method: "POST", body: JSON.stringify({ url, mode: "main_content", format: "markdown", include_links: true }) });
  },
  search: (query: string) => {
    requireCapabilityEntitlement("uiai.search.execute");
    return engineRequest<{ results: SearchResult[]; provider?: string; focusa?: Record<string, unknown> }>(`/api/search/?q=${encodeURIComponent(query)}&limit=10`);
  },
  screenshotUrl: (url: string) => {
    requireCapabilityEntitlement("uiai.browser.screenshot.execute");
    return engineRequest<ScreenshotResult & { focusa?: Record<string, unknown> }>("/api/screenshot/", { method: "POST", body: JSON.stringify({ url, width: 1440, height: 900, format: "jpeg", quality: 78 }) });
  },
};
