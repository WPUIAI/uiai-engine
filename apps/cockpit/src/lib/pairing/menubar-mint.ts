/** UIAI-COCKPIT-005 T005-06.04 — Distinct Cockpit token minting (Path B, docs/164 workstream-aware). */
import { parseMenubarCrossReferenceRequest, type MenubarCrossReferenceRequestV1 } from "./menubar-cross-reference";

const opaque = /^[A-Za-z0-9._~:-]{1,256}$/;
const forbidden = /token|secret|password|authorization|private.?key/i;

function isRecord(v: unknown): v is Record<string, unknown> {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

export interface DaemonMenubarProofV1 {
  schema: "focusa.daemon_menubar_proof.v1";
  verified_device_id: string;
  daemon_url: string;
  verified_at: string;
  scopes: string[];
}

export interface DistinctCockpitMintResultV1 {
  schema: "focusa.distinct_cockpit_mint.v1";
  cockpit_device_id: string;
  menubar_device_id: string;
  daemon_url: string;
  token_handle: string;
  scopes: string[];
  distinct: true;
}

function assertOpaque(v: string, label: string) {
  if (!opaque.test(v)) throw new Error(`${label} is not opaque`);
}
function assertTs(v: string, label: string) {
  if (!Number.isFinite(Date.parse(v))) throw new Error(`${label} is not a timestamp`);
}
function assertUrl(v: string, label: string): string {
  const u = new URL(v);
  if (!/^https?:$/.test(u.protocol) || u.username || u.password || u.hash) throw new Error(`${label} is not an allowed URL`);
  return v.replace(/\/$/, "");
}
function rejectSecretShape(obj: Record<string, unknown>, label: string) {
  for (const k of Object.keys(obj)) if (forbidden.test(k)) throw new Error(`${label} contains forbidden secret field ${k}`);
}

export function parseDaemonMenubarProof(value: unknown): DaemonMenubarProofV1 {
  if (!isRecord(value)) throw new Error("daemon_proof must be an object");
  rejectSecretShape(value, "daemon_proof");
  const allowed = ["schema", "verified_device_id", "daemon_url", "verified_at", "scopes"];
  for (const k of Object.keys(value)) if (!allowed.includes(k)) throw new Error(`daemon_proof contains unknown field ${k}`);
  if (value.schema !== "focusa.daemon_menubar_proof.v1") throw new Error("daemon_proof schema mismatch");
  const verified_device_id = String(value.verified_device_id ?? "");
  const daemon_url = String(value.daemon_url ?? "");
  const verified_at = String(value.verified_at ?? "");
  assertOpaque(verified_device_id, "daemon_proof.verified_device_id");
  assertUrl(daemon_url, "daemon_proof.daemon_url");
  assertTs(verified_at, "daemon_proof.verified_at");
  if (!Array.isArray(value.scopes) || value.scopes.length === 0) throw new Error("daemon_proof.scopes is required");
  for (const s of value.scopes as unknown[]) if (typeof s !== "string" || !["read","write"].includes(s)) throw new Error("daemon_proof.scopes invalid");
  return { schema: "focusa.daemon_menubar_proof.v1", verified_device_id, daemon_url, verified_at, scopes: value.scopes as string[] };
}

/**
 * Mint a distinct Cockpit device + token handle from a validated Menubar cross-reference
 * and a daemon proof that the Menubar device is verified. Never copies Menubar token.
 * Caller supplies a fresh cockpit nonce/device seed for generation; daemon mints server-side,
 * but this pure function validates distinctness and shapes token_handle.
 */
export function mintDistinctCockpitDevice(input: {
  crossRef: unknown;
  daemonProof: unknown;
  cockpit_device_id: string;
  token_handle: string;
}): DistinctCockpitMintResultV1 {
  const cross = parseMenubarCrossReferenceRequest(input.crossRef);
  const proof = parseDaemonMenubarProof(input.daemonProof);
  // mint_input is trusted caller shape: allow only token_handle handle reference (not secret value).
  // Reject unexpected secret-bearing keys without flagging the legitimate token_handle handle name.
  {
    const raw = input as unknown as Record<string, unknown>;
    for (const k of Object.keys(raw)) {
      if (["crossRef", "daemonProof", "cockpit_device_id", "token_handle"].includes(k)) continue;
      if (forbidden.test(k)) throw new Error(`mint_input contains forbidden secret field ${k}`);
    }
  }

  // Cross-reference must match proof: same verified device + same daemon origin
  if (proof.verified_device_id !== cross.device_id) throw new Error("daemon proof device mismatch");
  if (new URL(proof.daemon_url).origin !== new URL(cross.daemon_url).origin) throw new Error("daemon proof origin mismatch");

  // Cockpit identity must be fresh and distinct from Menubar device
  assertOpaque(input.cockpit_device_id, "cockpit_device_id");
  assertOpaque(input.token_handle, "token_handle");
  if (input.cockpit_device_id === cross.device_id) throw new Error("cockpit device must be distinct from menubar device");
  if (forbidden.test(input.token_handle)) throw new Error("token_handle contains forbidden secret shape");
  // Proof itself must not carry a token/secret — never copy Menubar credential
  // (already enforced by rejectSecretShape on proof)

  return {
    schema: "focusa.distinct_cockpit_mint.v1",
    cockpit_device_id: input.cockpit_device_id,
    menubar_device_id: cross.device_id,
    daemon_url: cross.daemon_url,
    token_handle: input.token_handle,
    scopes: proof.scopes,
    distinct: true,
  };
}
