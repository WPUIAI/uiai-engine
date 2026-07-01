// §3.15 Phase 0 card manifest.
//   Every focusa_* card below is mapped to a real Spec 90 contract.
//   Cards without a Spec 90 contract yet are flagged contract_ref=null.

import type { CardManifest } from "../contracts/card-manifest";

export const phase0Cards: CardManifest[] = [
  {
    card_id: "uiai.health",
    label: "UIAI Engine Health",
    product_surface: "uiai_engine",
    authority_plane: "browser_execution",
    normative_source: "UIAI Engine /api/health/browser",
    contract_ref: null,
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["uiai.health.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "none",
    visual_priority: "phase0",
    notes: "adapter_only",
  },
  {
    card_id: "uiai.diagnostics",
    label: "Browser Diagnostics",
    product_surface: "uiai_engine",
    authority_plane: "browser_execution",
    normative_source: "UIAI Engine /api/session/{id}/diagnostics",
    contract_ref: null,
    required_scope: "session",
    side_effect_class: "read",
    capabilities: ["uiai.session.diagnostics.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
    notes: "adapter_only",
  },
  {
    card_id: "focusa.project_identity",
    label: "Project Identity",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 104 + Spec 90 contract focusa_project_identity",
    contract_ref: "focusa_project_identity",
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["focusa.project.identity.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.project_card",
    label: "Project Card",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_project_card",
    contract_ref: "focusa_project_card",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.project.card.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.workpoint_resume",
    label: "Workpoint Resume",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_workpoint_resume",
    contract_ref: "focusa_workpoint_resume",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.workpoint.resume.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.trajectory_view",
    label: "Trajectory View",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_trajectory_view",
    contract_ref: "focusa_trajectory_view",
    required_scope: "workstream",
    side_effect_class: "read",
    capabilities: ["focusa.trajectory.view.read"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.tool_doctor",
    label: "Tool Doctor",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_tool_doctor",
    contract_ref: "focusa_tool_doctor",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.tool.doctor.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.dxux_requirement",
    label: "DXUX Requirement",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_dxux_requirement",
    contract_ref: "focusa_dxux_requirement",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.dxux.requirement.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.work_loop_status",
    label: "Work-loop Status",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source:
      "Spec 90 contract focusa_work_loop_status (parity_status=domain)",
    contract_ref: "focusa_work_loop_status",
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["focusa.workloop.status.read"],
    offline_behavior: "works",
    receipt_behavior: "none",
    visual_priority: "phase0",
    parity_status: "domain",
  },
  {
    card_id: "focusa.device_pair_status",
    label: "Device Pair Status",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_device_pair_status + Spec 53",
    contract_ref: "focusa_device_pair_status",
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["focusa.device.pair.status.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
  },
  {
    card_id: "focusa.evidence_link",
    label: "Capture Evidence",
    product_surface: "focusa_local",
    authority_plane: "local_node",
    normative_source: "Spec 90 contract focusa_workpoint_link_evidence",
    contract_ref: "focusa_workpoint_link_evidence",
    required_scope: "workstream",
    side_effect_class: "local_write",
    capabilities: ["focusa.evidence.link.write"],
    offline_behavior: "read_only",
    receipt_behavior: "local_receipt",
    visual_priority: "phase0",
    notes: "Phase 0's one write card.",
  },
  {
    card_id: "cloud.node_status",
    label: "Cloud Node Status",
    product_surface: "focusa_cloud",
    authority_plane: "cloud_control_plane",
    normative_source: "Spec 115 §9.2 Node Registry",
    contract_ref: null,
    required_scope: "node",
    side_effect_class: "read",
    capabilities: ["node.health.read", "node.version.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "cloud_receipt",
    visual_priority: "phase0",
    notes: "cloud-only",
  },
  {
    card_id: "cloud.device_pairing",
    label: "Device Pairing",
    product_surface: "focusa_cloud",
    authority_plane: "cloud_control_plane",
    normative_source: "Spec 53 + Spec 115 §9.3",
    contract_ref: null,
    required_scope: "node",
    side_effect_class: "cloud_write",
    capabilities: ["device.pair", "device.repair", "device.revoke"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "cloud_receipt",
    visual_priority: "phase0",
    notes: "cloud-only",
  },
  {
    card_id: "ai_api.health_usage",
    label: "AI API Health & Usage",
    product_surface: "ai_api",
    authority_plane: "hosted_ai",
    normative_source: "AI API health/usage endpoints (hosted)",
    contract_ref: null,
    required_scope: "none",
    side_effect_class: "read",
    capabilities: ["ai_api.health.read", "ai_api.usage.read"],
    offline_behavior: "blocked_with_reason",
    receipt_behavior: "none",
    visual_priority: "phase0",
    notes: "hosted",
  },
];

export type Phase0CardId = (typeof phase0Cards)[number]["card_id"];

/** Spec 90 contract coverage matrix for Phase 0. */
export const phase0ContractCoverage: Array<{
  card_id: string;
  contract_ref: string | null;
  parity_status?: string;
  notes: string;
}> = phase0Cards.map((c) => ({
  card_id: c.card_id,
  contract_ref: c.contract_ref ?? null,
  parity_status: c.parity_status,
  notes: c.notes ?? (c.contract_ref ? "core read" : "adapter_only"),
}));

export function validateCardManifest(
  manifest: CardManifest[],
): { ok: true } | { ok: false; errors: string[] } {
  const errors: string[] = [];
  for (const card of manifest) {
    if (!card.card_id) errors.push(`${card.label}: missing card_id`);
    if (!card.capabilities || card.capabilities.length === 0) {
      errors.push(`${card.card_id}: missing capabilities`);
    }
    if (
      card.product_surface === "focusa_local" &&
      !card.contract_ref &&
      !card.notes?.includes("adapter_only")
    ) {
      errors.push(
        `${card.card_id}: focusa_local card without Spec 90 contract_ref (must be adapter_only or similar)`,
      );
    }
  }
  return errors.length === 0 ? { ok: true } : { ok: false, errors };
}