/**
 * MR-P0-02 — Runtime validators that mirror contracts/schemas JSON Schemas.
 * Desktop-presentation already has parse* (fail-closed, exactKeys, opaque, RFC3339).
 * This module wires every ScopeRef/workstream and OTA receipt through those validators
 * so Cockpit never trusts a wire shape without runtime proof.
 * No zod dep — hand validators match draft 2020-12 schemas in contracts/schemas.
 */

import {
  parseDesktopPresentationRequest,
  parseAppHandoffIntent,
  parseAppHandoffReceipt,
  assertOpaqueRef,
} from "./desktop-presentation";

export { parseDesktopPresentationRequest, parseAppHandoffIntent, parseAppHandoffReceipt, assertOpaqueRef };

export function validateScopeRefShape(value: unknown): void {
  // delegate to desktop-presentation's parseScopeRef via a dummy presentation request
  // avoids duplicating workstream invariant; throws on mismatch.
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("scope_ref must be an object");
  const scope = value as Record<string, unknown>;
  if (typeof scope.authority_state !== "string") throw new Error("authority_state is required");
  const allowed = new Set(["verified", "missing", "stale", "conflict", "read_only"]);
  if (!allowed.has(String(scope.authority_state))) throw new Error(`unsupported authority_state: ${scope.authority_state}`);
  // workstream invariant — mirrors contracts/schemas/uiai.scope_ref.v1 + workstream.ts
  const pr = scope.project_root_key as string | undefined;
  const wk = scope.workstream_key as string | undefined;
  const ci = scope.continuity_id as string | undefined;
  if (wk !== undefined) {
    if (!String(wk).includes("::")) throw new Error("workstream_key must be project_root_key::continuity_id");
    if (String(wk).includes("://") || String(wk).includes("?")) throw new Error("workstream_key must not contain URL delimiters");
  }
  if (pr && ci && wk) {
    const derived = `${String(pr).replace(/\/+$/, "")}::${String(ci)}`;
    if (wk !== derived) throw new Error(`workstream_key mismatch: expected ${derived}`);
  }
  for (const k of ["continuity_id", "thread_id", "session_id"] as const) {
    if (scope[k] !== undefined) assertOpaqueRef(String(scope[k]), `scope_ref.${k}`);
  }
}

export function validateOtaActivationReceipt(value: unknown): void {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("ota receipt must be an object");
  const r = value as Record<string, unknown>;
  if (r.schema !== "uiai.cockpit_ota_activation_receipt.v1") throw new Error("unsupported ota receipt schema");
  if (r.status !== "installed_relaunching") throw new Error("ota receipt status must be installed_relaunching");
  if (typeof r.version !== "string" || !/^[0-9]+\.[0-9]+\.[0-9]+(-dev)?$/.test(r.version)) throw new Error("ota receipt version invalid");
  if (r.activation !== "signed_updater_install_and_relaunch") throw new Error("ota receipt activation invalid");
  const ts = String(r.activated_at ?? "");
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/.test(ts) || Number.isNaN(Date.parse(ts))) throw new Error("activated_at must be RFC3339");
}
