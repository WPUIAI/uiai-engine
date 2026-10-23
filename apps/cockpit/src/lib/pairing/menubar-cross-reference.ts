/** UIAI-COCKPIT-005 T005-06.03 — Menubar cross-reference request (proof_request). */
export interface MenubarCrossReferenceRequestV1 {
  schema: "focusa.menubar_cross_reference.v1";
  device_id: string;
  nonce: string;
  daemon_url: string;
  created_at: string;
}

const opaque = /^[A-Za-z0-9._~:-]{1,256}$/;
const forbidden = /token|secret|password|authorization|private.?key/i;

function obj(v: unknown, label: string): Record<string, unknown> {
  if (!v || typeof v !== "object" || Array.isArray(v)) throw new Error(`${label} must be an object`);
  return v as Record<string, unknown>;
}
function exact(item: Record<string, unknown>, allowed: string[], label: string) {
  for (const k of Object.keys(item)) if (!allowed.includes(k)) throw new Error(`${label} contains unknown field ${k}`);
}
function text(item: Record<string, unknown>, key: string, label: string): string {
  const v = item[key];
  if (typeof v !== "string" || !v) throw new Error(`${label}.${key} is required`);
  return v;
}
function ref(item: Record<string, unknown>, key: string, label: string): string {
  const v = text(item, key, label);
  if (!opaque.test(v)) throw new Error(`${label}.${key} is not opaque`);
  return v;
}
function ts(item: Record<string, unknown>, key: string, label: string): string {
  const v = text(item, key, label);
  if (!Number.isFinite(Date.parse(v))) throw new Error(`${label}.${key} is not a timestamp`);
  return v;
}
function url(item: Record<string, unknown>, key: string, label: string): string {
  const v = text(item, key, label);
  const u = new URL(v);
  if (!/^https?:$/.test(u.protocol) || u.username || u.password || u.hash) throw new Error(`${label}.${key} is not an allowed URL`);
  return v.replace(/\/$/, "");
}
function rejectSecretShape(item: Record<string, unknown>, label: string) {
  for (const k of Object.keys(item)) if (forbidden.test(k)) throw new Error(`${label} contains forbidden secret field`);
}

export function parseMenubarCrossReferenceRequest(value: unknown): MenubarCrossReferenceRequestV1 {
  const x = obj(value, "cross_ref");
  rejectSecretShape(x, "cross_ref");
  exact(x, ["schema", "device_id", "nonce", "daemon_url", "created_at"], "cross_ref");
  if (x.schema !== "focusa.menubar_cross_reference.v1") throw new Error("cross_ref schema mismatch");
  return {
    schema: "focusa.menubar_cross_reference.v1",
    device_id: ref(x, "device_id", "cross_ref"),
    nonce: ref(x, "nonce", "cross_ref"),
    daemon_url: url(x, "daemon_url", "cross_ref"),
    created_at: ts(x, "created_at", "cross_ref"),
  };
}

export function createMenubarCrossReferenceRequest(input: {
  device_id: string;
  nonce: string;
  daemon_url: string;
  created_at?: string;
}): MenubarCrossReferenceRequestV1 {
  return parseMenubarCrossReferenceRequest({
    schema: "focusa.menubar_cross_reference.v1",
    device_id: input.device_id,
    nonce: input.nonce,
    daemon_url: input.daemon_url,
    created_at: input.created_at ?? new Date().toISOString(),
  });
}
