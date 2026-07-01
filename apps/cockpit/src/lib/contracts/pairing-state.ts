// §3.8 PairingState — mirrors menubar FirstRunWizard state machine.

import type { NodeRef } from "./node-ref";

export type PairingDeviceType =
  | "desktop_cockpit"
  | "browser_session"
  | "mac_app"
  | "pi_session"
  | "mcp_client";

export type PairingTokenState =
  | "none"
  | "pending"
  | "active"
  | "expired"
  | "revoked"
  | "repair_required";

export interface PairingState {
  device_id?: string;
  device_name: string;
  device_type: PairingDeviceType;
  token_state: PairingTokenState;
  granted_scopes: string[];
  mutation_grant: boolean;
  node?: NodeRef;
  repair_action?: "re_pair" | "rotate_token" | "revoke" | "open_connect";
  daemon_url?: string;
  expires_at?: string;
}
