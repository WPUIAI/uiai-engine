import { requireCapabilityEntitlement } from "./contracts/entitlement";

export const DEFAULT_ENGINE_URL = "http://127.0.0.1:7456";

export interface EngineHealth { status: string; service?: string; uptime?: number; browserless?: boolean; }
export interface BrowserHealth { status?: string; browser_state?: string; browser_alive?: boolean; active_pages?: number; max_pages?: number; available_pages?: number; pressure?: Record<string, unknown>; [key: string]: unknown; }
export interface BrowserSession { id: string; url: string; title?: string; width?: number; height?: number; created_at?: string; last_used?: string; nav_count?: number; snap_count?: number; }
export interface EvidenceScope {
  project_root?: string; project_ref?: string; workstream_ref?: string; workset_ref?: string; callgraph_ref?: string;
  workpoint_id?: string; workpoint_ref?: string; work_item_ref?: string; continuity_id?: string; continuity_ref?: string;
  workstream_key?: string; work_items?: Record<string, unknown>[];
}
export interface EPWADelivery {
  schema: "uiai.epwa_delivery.v1"; delivery_id: string; revision: number;
  artifact: { artifact_ref: string; revision: number; manifest_sha256: string; output_sha256: string };
  epwa: { package_id: string; package_ref: string; package_sha256: string; projection_ref?: string; projection_sha256?: string; record_url: string; portable_url: string; access: string };
  scope: { posture: string; [key: string]: unknown }; state: string; recovery_ref?: string;
}
export interface ArtifactDeliveryResult {
  schema: string; artifact_ref: string; delivery_state: string; epwa_delivery: EPWADelivery;
  artifact_url?: string; portable_url?: string; raw_output_posture?: string; inline_posture?: string; [key: string]: unknown;
}
export interface ReadyArtifactDeliveryResult extends ArtifactDeliveryResult { artifact_url: string; portable_url: string; }
export interface ScreenshotResult extends ArtifactDeliveryResult { format?: string; width?: number; height?: number; size?: number; url?: string; title?: string; duration_ms?: number; }
export interface SearchResult { title: string; url: string; description?: string; source?: string; age?: string; rank?: number; evidence_ref?: string; }

export class EngineRequestError extends Error {
  constructor(public readonly status: number, message: string) { super(message); this.name = "EngineRequestError"; }
}

export class EngineDeliveryError extends Error {
  constructor(public readonly delivery: Partial<ArtifactDeliveryResult>, message: string) { super(message); this.name = "EngineDeliveryError"; }
}

export function engineUrl(): string {
  if (typeof window === "undefined") return DEFAULT_ENGINE_URL;
  return window.localStorage.getItem("uiai.engine.url") || DEFAULT_ENGINE_URL;
}

export function selectEngineUrl(baseUrl: string): void {
  if (typeof window !== "undefined") window.localStorage.setItem("uiai.engine.url", baseUrl.replace(/\/$/, ""));
}

export function savedScope(): EvidenceScope {
  if (typeof window === "undefined") return {};
  const get = (key: string) => window.localStorage.getItem(`uiai.scope.${key}`) || undefined;
  const project_root = get("project_root");
  const continuity_id = get("continuity_id");
  const workpoint_id = get("workpoint_id");
  const stored = get("workstream_key");
  let work_items: Record<string, unknown>[] | undefined;
  try {
    const parsed = JSON.parse(get("work_items") || "null");
    if (Array.isArray(parsed)) work_items = parsed;
  } catch { /* malformed local scope remains incomplete and fails closed */ }
  return {
    project_root, project_ref: get("project_ref") || project_root, workstream_ref: get("workstream_ref"), workset_ref: get("workset_ref"),
    callgraph_ref: get("callgraph_ref"), workpoint_id, workpoint_ref: get("workpoint_ref") || workpoint_id,
    work_item_ref: get("work_item_ref"), continuity_id, continuity_ref: get("continuity_ref") || continuity_id,
    workstream_key: stored ?? (project_root && continuity_id ? `${project_root.replace(/\/+$/, "")}::${continuity_id}` : undefined), work_items,
  };
}

export function evidenceScopeHeaders(scope: EvidenceScope): Headers {
  const headers = new Headers();
  const values: Record<string, string | undefined> = {
    "X-UIAI-Project-Ref": scope.project_ref || scope.project_root,
    "X-UIAI-Workstream-Ref": scope.workstream_ref,
    "X-UIAI-Workset-Ref": scope.workset_ref,
    "X-UIAI-CallGraph-Ref": scope.callgraph_ref,
    "X-UIAI-Workpoint-Ref": scope.workpoint_ref || scope.workpoint_id,
    "X-UIAI-Work-Item-Ref": scope.work_item_ref,
    "X-UIAI-Continuity-Ref": scope.continuity_ref || scope.continuity_id,
  };
  for (const [name, value] of Object.entries(values)) if (value?.trim()) headers.set(name, value.trim());
  if (scope.work_items?.length) headers.set("X-UIAI-Work-Items", JSON.stringify(scope.work_items));
  return headers;
}

export async function engineRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 120_000);
  const headers = new Headers(init?.headers);
  if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  let response: Response;
  try {
    response = await fetch(`${engineUrl()}${path}`, { ...init, signal: controller.signal, headers });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === "AbortError") throw new EngineRequestError(408, "The engine request timed out while starting or reading the browser.");
    throw cause;
  } finally {
    window.clearTimeout(timeout);
  }
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new EngineRequestError(response.status, body?.error?.message || body?.error || `Engine request failed (${response.status})`);
  return body as T;
}

export function validateArtifactDelivery<T extends ArtifactDeliveryResult>(body: T): T & ReadyArtifactDeliveryResult {
  const forbidden = ["screenshot", "imageBase64", "image_base64", "artifact_path", "result_path", "result_url"];
  if (forbidden.some((key) => Object.prototype.hasOwnProperty.call(body, key))) throw new EngineDeliveryError(body, "Engine returned a forbidden raw artifact field.");
  const delivery = body.epwa_delivery;
  if (!delivery || delivery.schema !== "uiai.epwa_delivery.v1" || delivery.artifact?.artifact_ref !== body.artifact_ref || delivery.state !== body.delivery_state) {
    throw new EngineDeliveryError(body, "Engine returned an invalid EPWA delivery binding.");
  }
  if (delivery.state !== "ready") throw new EngineDeliveryError(body, delivery.recovery_ref || "EPWA delivery is not ready.");
  if (!body.artifact_url?.startsWith("https://") || !body.portable_url?.startsWith("https://") || body.artifact_url !== delivery.epwa.record_url || body.portable_url !== delivery.epwa.portable_url) {
    throw new EngineDeliveryError(body, "Engine delivery is missing canonical HTTPS EPWA URLs.");
  }
  return body as T & ReadyArtifactDeliveryResult;
}

export async function artifactRequest<T extends ArtifactDeliveryResult>(path: string, init?: RequestInit, scope = savedScope()): Promise<T & ReadyArtifactDeliveryResult> {
  const headers = new Headers(init?.headers);
  evidenceScopeHeaders(scope).forEach((value, name) => headers.set(name, value));
  return validateArtifactDelivery(await engineRequest<T>(path, { ...init, headers }));
}

export const engineClient = {
  health: () => engineRequest<EngineHealth>("/health"),
  browserHealth: () => engineRequest<BrowserHealth>("/api/health/browser"),
  sessions: async () => (await engineRequest<{ sessions: BrowserSession[]; count: number; max: number }>("/api/session/")).sessions,
  openSession: (url: string, scope = savedScope()) => {
    requireCapabilityEntitlement("uiai.browser.session.create");
    return artifactRequest<ArtifactDeliveryResult & { session: BrowserSession; size: { width: number; height: number }; fpv_share?: unknown }>("/api/session/", { method: "POST", body: JSON.stringify({ url, width: 1280, height: 800, project_root: scope.project_root, continuity_id: scope.continuity_id, workpoint_id: scope.workpoint_id, workstream_key: scope.workstream_key }) }, scope);
  },
  closeSession: (id: string) => {
    requireCapabilityEntitlement("uiai.browser.session.control");
    return engineRequest<{ status: string; id: string }>(`/api/session/${encodeURIComponent(id)}`, { method: "DELETE" });
  },
  diagnostics: (id: string) => artifactRequest<ArtifactDeliveryResult & { diagnostics?: Record<string, unknown> }>(`/api/session/${encodeURIComponent(id)}/diagnostics`),
  navigate: (id: string, url: string) => {
    requireCapabilityEntitlement("uiai.browser.session.control");
    return artifactRequest<ScreenshotResult>(`/api/session/${encodeURIComponent(id)}/navigate`, { method: "POST", body: JSON.stringify({ url }) });
  },
  screenshot: (id: string) => {
    requireCapabilityEntitlement("uiai.browser.screenshot.execute");
    return artifactRequest<ScreenshotResult>(`/api/session/${encodeURIComponent(id)}/screenshot`, { method: "POST", body: JSON.stringify({ format: "jpeg", quality: 72 }) });
  },
  sourceMarkdown: (url: string) => {
    requireCapabilityEntitlement("uiai.source.markdown.execute");
    return artifactRequest<ArtifactDeliveryResult & { markdown?: string; title?: string; source?: Record<string, unknown>; focusa?: Record<string, unknown> }>("/api/markdown/", { method: "POST", body: JSON.stringify({ url, mode: "main_content", format: "markdown", include_links: true }) });
  },
  search: (query: string) => {
    requireCapabilityEntitlement("uiai.search.execute");
    return artifactRequest<ArtifactDeliveryResult & { results: SearchResult[]; provider?: string; focusa?: Record<string, unknown> }>(`/api/search/?q=${encodeURIComponent(query)}&limit=10`);
  },
  screenshotUrl: (url: string) => {
    requireCapabilityEntitlement("uiai.browser.screenshot.execute");
    return artifactRequest<ScreenshotResult & { focusa?: Record<string, unknown> }>("/api/screenshot/", { method: "POST", body: JSON.stringify({ url, width: 1440, height: 900, format: "jpeg", quality: 78 }) });
  },
};
