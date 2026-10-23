/** UIAI-COCKPIT-005 T005-06.05 — Denial / mismatch / absence fallback (Path B → Path A). */
import { evaluateAutoAddEligibility, type AutoAddEligibility } from "./auto-add-eligibility";
import { mintDistinctCockpitDevice, type DaemonMenubarProofV1 } from "./menubar-mint";
import { parseMenubarCrossReferenceRequest } from "./menubar-cross-reference";

export type FallbackReason =
  | "denied_by_policy"
  | "denied_by_operator"
  | "mismatch"
  | "absent";

export interface MenubarFallbackDecisionV1 {
  schema: "focusa.menubar_fallback_decision.v1";
  fallbackToPathA: boolean;
  reason: FallbackReason;
  eligibility: AutoAddEligibility;
  authorityMutated: false;
  partialProfileCreated: false;
  message: string;
}

function fallback(eligibility: AutoAddEligibility, reason: FallbackReason, message: string): MenubarFallbackDecisionV1 {
  return {
    schema: "focusa.menubar_fallback_decision.v1",
    fallbackToPathA: true,
    reason,
    eligibility,
    authorityMutated: false,
    partialProfileCreated: false,
    message,
  };
}

/**
 * Evaluate Path B outcome. On denial/mismatch/absence, returns a fallback decision
 * that truthfully routes to Path A with zero authority mutation or partial profile.
 * Caller must NOT have mutated store before this decision; mint is attempted only
 * when eligibility and proof both validate.
 */
export function evaluateMenubarFallback(input: {
  same_machine: boolean;
  same_user: boolean;
  policy_enabled: boolean;
  dismissed: boolean;
  selected_daemon_url: string;
  hint: unknown;
  manifest: unknown;
  crossRef: unknown; // may be absent (null)
  daemonProof: unknown; // may be absent (null)
  cockpit_device_id?: string;
  token_handle?: string;
}): MenubarFallbackDecisionV1 | { schema: "focusa.distinct_cockpit_mint.v1"; [k: string]: unknown } {
  const eligibility = evaluateAutoAddEligibility({
    same_machine: input.same_machine,
    same_user: input.same_user,
    policy_enabled: input.policy_enabled,
    dismissed: input.dismissed,
    selected_daemon_url: input.selected_daemon_url,
    hint: input.hint,
    manifest: input.manifest,
  });

  if (!eligibility.eligible) {
    const reason: FallbackReason = eligibility.reason === "dismissed" ? "denied_by_operator" : eligibility.reason === "policy_disabled" ? "denied_by_policy" : eligibility.reason === "different_machine" || eligibility.reason === "different_user" ? "absent" : eligibility.reason === "daemon_mismatch" ? "mismatch" : "absent";
    return fallback(eligibility, reason, `Path B ineligible (${eligibility.reason}) — fallback to Path A`);
  }

  // Absence: no cross-ref or no proof
  if (input.crossRef == null || input.daemonProof == null) {
    return fallback(eligibility, "absent", "Menubar proof absent — fallback to Path A");
  }

  // Validate shapes without mutating authority; mismatch maps to fallback
  try {
    parseMenubarCrossReferenceRequest(input.crossRef);
  } catch (e) {
    return fallback(eligibility, "mismatch", `Cross-reference invalid (${String(e).slice(0, 120)}) — fallback to Path A`);
  }

  // Attempt mint as pure validation (no side-effects yet). The real mint writes to keychain
  // only after this gate; here we validate distinctness/origin without mutating.
  // If validation throws mismatch/forbidden, map to fallback without partial profile.
  if (input.cockpit_device_id && input.token_handle) {
    try {
      const minted = mintDistinctCockpitDevice({
        crossRef: input.crossRef,
        daemonProof: input.daemonProof,
        cockpit_device_id: input.cockpit_device_id,
        token_handle: input.token_handle,
      });
      // Mint success — return minted result directly (caller will persist). No fallback.
      return minted as unknown as { schema: "focusa.distinct_cockpit_mint.v1"; [k: string]: unknown };
    } catch (e) {
      const msg = String(e);
      if (/mismatch|origin mismatch|device mismatch/i.test(msg)) return fallback(eligibility, "mismatch", `${msg} — fallback to Path A`);
      if (/forbidden|secret/i.test(msg)) return fallback(eligibility, "mismatch", `${msg} — fallback to Path A`);
      // Denied / other validation also falls back truthfully without partial mutation
      if (/distinct/i.test(msg)) return fallback(eligibility, "mismatch", `${msg} — fallback to Path A`);
      return fallback(eligibility, "absent", `${msg} — fallback to Path A`);
    }
  }

  // No mint attempted — treat proof validation separately (e.g. caller will mint later)
  // If we reached here, eligibility passed and shapes parsed; defer to caller.
  return fallback(eligibility, "absent", "Menubar proof present but mint deferred — no authority mutation; Path A available");
}

export function isFallbackDecision(v: unknown): v is MenubarFallbackDecisionV1 {
  return !!v && typeof v === "object" && (v as Record<string, unknown>).schema === "focusa.menubar_fallback_decision.v1";
}
