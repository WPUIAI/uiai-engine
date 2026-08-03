import type { ScopeRef } from "./scope-ref";

export const DESKTOP_PRESENTATION_SCHEMA_IDS = [
  "uiai.browser_runtime_manifest.v1",
  "uiai.desktop_presentation_request.v1",
  "uiai.desktop_presentation_receipt.v1",
  "uiai.desktop_presentation_status.v1",
  "uiai.app_handoff_intent.v1",
  "uiai.app_handoff_receipt.v1",
  "focusa.app.manifest.v2",
] as const;

export type DesktopPresentationSchemaId = (typeof DESKTOP_PRESENTATION_SCHEMA_IDS)[number];
export type PresentationMode = "full" | "pip" | "focus_existing";
export type PresentationReason = "operator_request" | "takeover_required" | "policy_confirmation" | "failure_recovery" | "workflow";
export type HandoffScheme = "focusa" | "cockpit";

export type PresentationStatus =
  | "requested"
  | "resolving_session"
  | "resolving_cockpit"
  | "launching"
  | "attaching"
  | "visible"
  | "focused"
  | "already_visible"
  | "blocked"
  | "unavailable"
  | "failed"
  | "blocked_scope"
  | "session_missing"
  | "cockpit_missing"
  | "incompatible"
  | "attach_failed"
  | "desktop_unavailable"
  | "expired"
  | "cancelled";

export interface ClientRef {
  client_type: "pi" | "cockpit" | "menubar" | "api" | "mcp" | "cli";
  client_id: string;
}

export interface BrowserRuntimeManifestV1 {
  schema: "uiai.browser_runtime_manifest.v1";
  runtime_id: string;
  engine: "chromium";
  version: string;
  cdp_protocol: string;
  platform: string;
  arch: string;
  executable_relpath: string;
  sha256: string;
  signed: boolean;
  source: "uiai-release";
  built_at: string;
}

export interface DesktopPresentationRequestV1 {
  schema: "uiai.desktop_presentation_request.v1";
  mode: PresentationMode;
  reason: PresentationReason;
  scope_ref?: ScopeRef;
  requested_by: ClientRef;
  focus: boolean;
  expires_in_ms: number;
  idempotency_key: string;
}

export interface DesktopPresentationReceiptV1 {
  schema: "uiai.desktop_presentation_receipt.v1";
  presentation_id: string;
  session_id: string;
  status: PresentationStatus;
  cockpit_instance_id?: string;
  handoff_ref?: string;
  reason_code?: string;
  created_at: string;
  expires_at: string;
}

export interface DesktopPresentationStatusV1 {
  schema: "uiai.desktop_presentation_status.v1";
  presentation_id: string;
  session_id: string;
  status: PresentationStatus;
  reason_code?: string;
  observed_at: string;
}

export interface AppHandoffIntentV1 {
  schema: "uiai.app_handoff_intent.v1";
  intent_id: string;
  scheme: HandoffScheme;
  route: string;
  target_ref: string;
  requested_by: ClientRef;
  protocol_version: "1";
  created_at: string;
  expires_at: string;
}

export interface AppHandoffReceiptV1 {
  schema: "uiai.app_handoff_receipt.v1";
  intent_id: string;
  status: "opened" | "focused" | "blocked" | "unavailable" | "failed";
  target_app: "focusa-menubar" | "uaiengine-cockpit";
  resolved_ref?: string;
  reason_code?: string;
  observed_at: string;
}

export interface FocusaAppManifestV2 {
  schema: "focusa.app.manifest.v2";
  app: "focusa-menubar" | "uaiengine-cockpit";
  version: string;
  channel: "stable" | "preview" | "dev";
  protocols: Record<string, string>;
  capabilities: string[];
}

export interface DesktopContractFixtureBundle {
  runtime_manifest: BrowserRuntimeManifestV1;
  presentation_request: DesktopPresentationRequestV1;
  presentation_receipt: DesktopPresentationReceiptV1;
  presentation_status: DesktopPresentationStatusV1;
  handoff_intent: AppHandoffIntentV1;
  handoff_receipt: AppHandoffReceiptV1;
  app_manifest: FocusaAppManifestV2;
}

const opaqueRefPattern = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/;
const sha256Pattern = /^[a-f0-9]{64}$/;
const modes = new Set<PresentationMode>(["full", "pip", "focus_existing"]);
const reasons = new Set<PresentationReason>(["operator_request", "takeover_required", "policy_confirmation", "failure_recovery", "workflow"]);
const statuses = new Set<PresentationStatus>([
  "requested", "resolving_session", "resolving_cockpit", "launching", "attaching", "visible", "focused", "already_visible",
  "blocked", "unavailable", "failed", "blocked_scope", "session_missing", "cockpit_missing", "incompatible", "attach_failed",
  "desktop_unavailable", "expired", "cancelled",
]);
const handoffRoutes: Record<HandoffScheme, Set<string>> = {
  focusa: new Set(["mission", "card", "workpoint", "connect"]),
  cockpit: new Set(["live/session", "focusa", "evidence", "settings/pairing"]),
};
const clientTypes = new Set(["pi", "cockpit", "menubar", "api", "mcp", "cli"]);
const authorityStates = new Set(["verified", "missing", "stale", "conflict", "read_only"]);

function record(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`${label} must be an object`);
  return value as Record<string, unknown>;
}

function exactKeys(value: Record<string, unknown>, allowed: readonly string[], label: string): void {
  const set = new Set(allowed);
  const unknown = Object.keys(value).filter((key) => !set.has(key));
  if (unknown.length) throw new Error(`${label} contains unsupported fields: ${unknown.join(", ")}`);
}

function requiredString(value: Record<string, unknown>, key: string, label: string): string {
  const current = value[key];
  if (typeof current !== "string" || !current) throw new Error(`${label}.${key} is required`);
  return current;
}

function assertRFC3339(value: string, label: string): void {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/.test(value) || Number.isNaN(Date.parse(value))) {
    throw new Error(`${label} must be RFC3339 UTC`);
  }
}

export function assertOpaqueRef(value: string, label = "opaque ref"): string {
  if (!opaqueRefPattern.test(value) || value.includes("://") || /[\\/?#@]/.test(value)) {
    throw new Error(`${label} must be a 1-160 character opaque identifier without URL, path, query, fragment, or authority data`);
  }
  return value;
}

function parseClientRef(value: unknown): ClientRef {
  const item = record(value, "requested_by");
  exactKeys(item, ["client_type", "client_id"], "requested_by");
  const clientType = requiredString(item, "client_type", "requested_by");
  if (!clientTypes.has(clientType)) throw new Error(`unsupported client_type: ${clientType}`);
  return { client_type: clientType as ClientRef["client_type"], client_id: assertOpaqueRef(requiredString(item, "client_id", "requested_by"), "client_id") };
}

function parseScopeRef(value: unknown): ScopeRef {
  const item = record(value, "scope_ref");
  exactKeys(item, ["project_root_key", "project_label", "workstream_key", "continuity_id", "thread_id", "session_id", "cloud_node_id", "machine_id", "daemon_endpoint", "role", "authority_state"], "scope_ref");
  const authorityState = requiredString(item, "authority_state", "scope_ref");
  if (!authorityStates.has(authorityState)) throw new Error(`unsupported authority_state: ${authorityState}`);
  for (const key of ["project_root_key", "workstream_key", "continuity_id", "thread_id", "session_id"] as const) {
    const current = item[key];
    if (current !== undefined) assertOpaqueRef(String(current), `scope_ref.${key}`);
  }
  return item as unknown as ScopeRef;
}

export function parseBrowserRuntimeManifest(value: unknown): BrowserRuntimeManifestV1 {
  const item = record(value, "runtime_manifest");
  exactKeys(item, ["schema", "runtime_id", "engine", "version", "cdp_protocol", "platform", "arch", "executable_relpath", "sha256", "signed", "source", "built_at"], "runtime_manifest");
  if (item.schema !== "uiai.browser_runtime_manifest.v1") throw new Error("unsupported runtime manifest schema");
  assertOpaqueRef(requiredString(item, "runtime_id", "runtime_manifest"), "runtime_id");
  if (item.engine !== "chromium" || item.source !== "uiai-release") throw new Error("runtime engine/source mismatch");
  for (const key of ["version", "cdp_protocol", "platform", "arch"] as const) requiredString(item, key, "runtime_manifest");
  const relpath = requiredString(item, "executable_relpath", "runtime_manifest");
  if (relpath.startsWith("/") || relpath.includes("..")) throw new Error("executable_relpath must remain relative");
  if (!sha256Pattern.test(requiredString(item, "sha256", "runtime_manifest"))) throw new Error("invalid runtime sha256");
  if (typeof item.signed !== "boolean") throw new Error("runtime signed flag is required");
  assertRFC3339(requiredString(item, "built_at", "runtime_manifest"), "built_at");
  return item as unknown as BrowserRuntimeManifestV1;
}

export function parseDesktopPresentationRequest(value: unknown): DesktopPresentationRequestV1 {
  const item = record(value, "presentation_request");
  exactKeys(item, ["schema", "mode", "reason", "scope_ref", "requested_by", "focus", "expires_in_ms", "idempotency_key"], "presentation_request");
  if (item.schema !== "uiai.desktop_presentation_request.v1") throw new Error("unsupported presentation request schema");
  if (!modes.has(item.mode as PresentationMode)) throw new Error("unsupported presentation mode");
  if (!reasons.has(item.reason as PresentationReason)) throw new Error("unsupported presentation reason");
  if (item.scope_ref !== undefined) parseScopeRef(item.scope_ref);
  parseClientRef(item.requested_by);
  if (typeof item.focus !== "boolean") throw new Error("focus must be boolean");
  if (!Number.isInteger(item.expires_in_ms) || Number(item.expires_in_ms) < 1000 || Number(item.expires_in_ms) > 300000) throw new Error("expires_in_ms out of range");
  assertOpaqueRef(requiredString(item, "idempotency_key", "presentation_request"), "idempotency_key");
  return item as unknown as DesktopPresentationRequestV1;
}

export function parseDesktopPresentationReceipt(value: unknown): DesktopPresentationReceiptV1 {
  const item = record(value, "presentation_receipt");
  exactKeys(item, ["schema", "presentation_id", "session_id", "status", "cockpit_instance_id", "handoff_ref", "reason_code", "created_at", "expires_at"], "presentation_receipt");
  if (item.schema !== "uiai.desktop_presentation_receipt.v1") throw new Error("unsupported presentation receipt schema");
  for (const key of ["presentation_id", "session_id", "cockpit_instance_id", "handoff_ref"] as const) {
    if (item[key] !== undefined) assertOpaqueRef(String(item[key]), key);
  }
  if (!statuses.has(item.status as PresentationStatus)) throw new Error("unsupported presentation status");
  assertRFC3339(requiredString(item, "created_at", "presentation_receipt"), "created_at");
  assertRFC3339(requiredString(item, "expires_at", "presentation_receipt"), "expires_at");
  return item as unknown as DesktopPresentationReceiptV1;
}

export function parseDesktopPresentationStatus(value: unknown): DesktopPresentationStatusV1 {
  const item = record(value, "presentation_status");
  exactKeys(item, ["schema", "presentation_id", "session_id", "status", "reason_code", "observed_at"], "presentation_status");
  if (item.schema !== "uiai.desktop_presentation_status.v1") throw new Error("unsupported presentation status schema");
  assertOpaqueRef(requiredString(item, "presentation_id", "presentation_status"), "presentation_id");
  assertOpaqueRef(requiredString(item, "session_id", "presentation_status"), "session_id");
  if (!statuses.has(item.status as PresentationStatus)) throw new Error("unsupported presentation status");
  assertRFC3339(requiredString(item, "observed_at", "presentation_status"), "observed_at");
  return item as unknown as DesktopPresentationStatusV1;
}

export function parseAppHandoffIntent(value: unknown): AppHandoffIntentV1 {
  const item = record(value, "handoff_intent");
  exactKeys(item, ["schema", "intent_id", "scheme", "route", "target_ref", "requested_by", "protocol_version", "created_at", "expires_at"], "handoff_intent");
  if (item.schema !== "uiai.app_handoff_intent.v1") throw new Error("unsupported handoff intent schema");
  assertOpaqueRef(requiredString(item, "intent_id", "handoff_intent"), "intent_id");
  const scheme = requiredString(item, "scheme", "handoff_intent") as HandoffScheme;
  if (!(scheme in handoffRoutes)) throw new Error(`unsupported handoff scheme: ${scheme}`);
  const route = requiredString(item, "route", "handoff_intent");
  if (!handoffRoutes[scheme].has(route)) throw new Error(`unsupported ${scheme} handoff route: ${route}`);
  assertOpaqueRef(requiredString(item, "target_ref", "handoff_intent"), "target_ref");
  parseClientRef(item.requested_by);
  if (item.protocol_version !== "1") throw new Error("unsupported handoff protocol version");
  const createdAt = requiredString(item, "created_at", "handoff_intent");
  const expiresAt = requiredString(item, "expires_at", "handoff_intent");
  assertRFC3339(createdAt, "created_at");
  assertRFC3339(expiresAt, "expires_at");
  const ttl = Date.parse(expiresAt) - Date.parse(createdAt);
  if (ttl <= 0 || ttl > 300_000) throw new Error("handoff TTL must be between 1ms and five minutes");
  return item as unknown as AppHandoffIntentV1;
}

export function parseAppHandoffReceipt(value: unknown): AppHandoffReceiptV1 {
  const item = record(value, "handoff_receipt");
  exactKeys(item, ["schema", "intent_id", "status", "target_app", "resolved_ref", "reason_code", "observed_at"], "handoff_receipt");
  if (item.schema !== "uiai.app_handoff_receipt.v1") throw new Error("unsupported handoff receipt schema");
  assertOpaqueRef(requiredString(item, "intent_id", "handoff_receipt"), "intent_id");
  if (item.resolved_ref !== undefined) assertOpaqueRef(String(item.resolved_ref), "resolved_ref");
  if (!["opened", "focused", "blocked", "unavailable", "failed"].includes(String(item.status))) throw new Error("unsupported handoff status");
  if (!["focusa-menubar", "uaiengine-cockpit"].includes(String(item.target_app))) throw new Error("unsupported target app");
  assertRFC3339(requiredString(item, "observed_at", "handoff_receipt"), "observed_at");
  return item as unknown as AppHandoffReceiptV1;
}

export function parseFocusaAppManifest(value: unknown): FocusaAppManifestV2 {
  const item = record(value, "app_manifest");
  exactKeys(item, ["schema", "app", "version", "channel", "protocols", "capabilities"], "app_manifest");
  if (item.schema !== "focusa.app.manifest.v2") throw new Error("unsupported app manifest schema");
  if (!["focusa-menubar", "uaiengine-cockpit"].includes(String(item.app))) throw new Error("unsupported app manifest owner");
  requiredString(item, "version", "app_manifest");
  if (!["stable", "preview", "dev"].includes(String(item.channel))) throw new Error("unsupported app channel");
  const protocols = record(item.protocols, "app_manifest.protocols");
  for (const protocol of ["focusa_deep_link", "cockpit_deep_link", "desktop_presentation", "fpv"]) requiredString(protocols, protocol, "app_manifest.protocols");
  if (!Array.isArray(item.capabilities) || item.capabilities.length === 0 || item.capabilities.some((entry) => typeof entry !== "string" || !entry)) throw new Error("app capabilities are required");
  return item as unknown as FocusaAppManifestV2;
}

export function parseDesktopContractFixtureBundle(value: unknown): DesktopContractFixtureBundle {
  const item = record(value, "fixture_bundle");
  exactKeys(item, ["runtime_manifest", "presentation_request", "presentation_receipt", "presentation_status", "handoff_intent", "handoff_receipt", "app_manifest"], "fixture_bundle");
  return {
    runtime_manifest: parseBrowserRuntimeManifest(item.runtime_manifest),
    presentation_request: parseDesktopPresentationRequest(item.presentation_request),
    presentation_receipt: parseDesktopPresentationReceipt(item.presentation_receipt),
    presentation_status: parseDesktopPresentationStatus(item.presentation_status),
    handoff_intent: parseAppHandoffIntent(item.handoff_intent),
    handoff_receipt: parseAppHandoffReceipt(item.handoff_receipt),
    app_manifest: parseFocusaAppManifest(item.app_manifest),
  };
}
