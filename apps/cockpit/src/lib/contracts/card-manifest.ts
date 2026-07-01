// §3.8 CardManifest — every browser card declares this shape.

import type {
  ApiPlane,
  AuthorityPlane,
  HealthState,
  SideEffectClass,
} from "./api-plane";

export type CardManifest =
  | {
      card_id: string;
      label: string;
      product_surface: ApiPlane | "wirebot";
      authority_plane: AuthorityPlane;
      normative_source: string;
      contract_ref?: string | null; // null = adapter_only / cloud-only / hosted
      required_scope:
        | "none"
        | "project"
        | "workstream"
        | "thread"
        | "session"
        | "node"
        | "team";
      side_effect_class: SideEffectClass;
      capabilities: string[];
      offline_behavior: "works" | "read_only" | "hidden" | "blocked_with_reason";
      receipt_behavior:
        | "none"
        | "local_receipt"
        | "cloud_receipt"
        | "proof_receipt";
      visual_priority: "phase0" | "phase1" | "later";
      parity_status?: "full" | "domain" | "pi_only" | "local_only" | "degraded_known";
      notes?: string;
    };