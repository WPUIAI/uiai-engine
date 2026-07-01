// §3.3 — multiple Focusa daemons on Mac + VPS, sync + thread ownership.

export type NodeTransport =
  | "loopback"
  | "ssh"
  | "byo_tunnel"
  | "focusa_relay"
  | "cloud_only";

export type SyncState =
  | "unknown"
  | "current"
  | "backlog"
  | "conflict"
  | "offline";

export interface NodeRef {
  cloud_node_id?: string;
  machine_id: string;
  display_name: string;
  endpoint: string;
  transport: NodeTransport;
  health: import("./api-plane").HealthState;
  sync_state?: SyncState;
  authority_role?: import("./scope-ref").ScopeRole;
  version?: string;
}