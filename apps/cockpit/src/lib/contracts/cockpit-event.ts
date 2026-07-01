// §3.8 CockpitEvent — append-only event stream.

import type { ApiPlane } from "./api-plane";
import type { NodeRef } from "./node-ref";
import type { ScopeRef } from "./scope-ref";

export type CockpitEventKind =
  | "health_changed"
  | "scope_changed"
  | "node_changed"
  | "pairing_changed"
  | "card_started"
  | "card_completed"
  | "card_failed"
  | "sync_changed"
  | "proof_changed"
  | "auto_pair_observed_via_menubar"
  | "pairing_observed_via_menubar"
  | "pairing_refreshed"
  | "pairing_repaired"
  | "pairing_revoked"
  | "pairing_via_replicated_flow"
  | "pairing_via_auto_add";

export interface CockpitEvent {
  event_id: string;
  at: string;
  plane: ApiPlane | "cockpit";
  kind: CockpitEventKind;
  scope?: ScopeRef;
  node?: NodeRef;
  summary: string;
  evidence_ref?: string;
  receipt_ref?: string;
}