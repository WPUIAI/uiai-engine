// §3.8 ApiAdapter — each adapter exposes: discover, capabilities, call, error mapping.

import type { ApiPlane, HealthState } from "./api-plane";
import type { CockpitError } from "./cockpit-error";
import type { ScopeRef } from "./scope-ref";
import type { NodeRef } from "./node-ref";
import type { SideEffectClass } from "./api-plane";

export interface EndpointStatus {
  plane: ApiPlane;
  endpoint: string;
  version?: string;
  health: HealthState;
  capabilities: string[];
  auth_state: "none" | "paired" | "expired" | "revoked" | "missing";
  last_checked_at: string;
  human_status: string;
}

export interface AdapterRequest<TInput = unknown> {
  plane: ApiPlane;
  capability: string;
  input: TInput;
  scope?: ScopeRef;
  node?: NodeRef;
  side_effect: SideEffectClass;
  idempotency_key?: string;
}

export interface AdapterResult<TOutput = unknown> {
  ok: boolean;
  output?: TOutput;
  error?: CockpitError;
  evidence_ref?: string;
  receipt_ref?: string;
  redaction_state?: "none" | "redacted" | "blocked" | "public_safe";
}

export interface ApiAdapter {
  plane: ApiPlane;
  discover(): Promise<EndpointStatus>;
  capabilities(): Promise<string[]>;
  call<TInput, TOutput>(
    request: AdapterRequest<TInput>,
  ): Promise<AdapterResult<TOutput>>;
}
