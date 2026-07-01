// §3.5, §3.6 — three product surfaces plus wirebot slot.

export type OperatingProfile = "local_only" | "cloud_profile";

export type ApiPlane =
  | "uiai_engine"
  | "focusa_local"
  | "focusa_cloud"
  | "ai_api"
  | "wirebot"; // deferred

export type AuthorityPlane =
  | "browser_execution"
  | "local_node"
  | "cloud_control_plane"
  | "hosted_ai";

export type HealthState =
  | "unknown"
  | "ok"
  | "degraded"
  | "offline"
  | "blocked";

export type SideEffectClass =
  | "read"
  | "local_write"
  | "cloud_write"
  | "code_capsule"
  | "proof_publish"
  | "benchmark_run";
