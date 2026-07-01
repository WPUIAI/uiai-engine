// §3.8 CockpitError — every adapter failure surfaces this, never an exception.

import type { ApiPlane } from "./api-plane";

export type CockpitErrorCode =
  | "adapter_offline"
  | "adapter_auth_missing"
  | "adapter_unreachable"
  | "scope_missing"
  | "scope_stale"
  | "scope_conflict"
  | "thread_role_blocked"
  | "untrusted_local_caller"
  | "keychain_locked"
  | "keychain_empty"
  | "keychain_access_denied"
  | "missing_credential"
  | "ratelimit"
  | "publish_blocked"
  | "uiai_unreachable"
  | "focusa_crash_mid_write"
  | "cloud_timeout_mid_receipt"
  | "ai_api_rate_limited"
  | "proof_publish_blocked"
  | "proof_publish_aborted";

export type CockpitRecoveryAction =
  | "retry"
  | "open_health"
  | "select_scope"
  | "select_node"
  | "open_pair"
  | "repair_pairing"
  | "open_consent"
  | "open_keychain_settings"
  | "open_logs"
  | "wait"
  | "refresh_from_menubar";

export interface CockpitError {
  code: CockpitErrorCode | string;
  plane: ApiPlane | "cockpit";
  human_message: string; // i18n key
  technical_detail?: string;
  retry_strategy?: string;
  recovery_action?: CockpitRecoveryAction;
  correlation_id?: string;
}
