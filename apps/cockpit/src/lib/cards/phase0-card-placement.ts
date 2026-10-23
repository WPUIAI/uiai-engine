export interface Phase0CardPlacement {
  surface: string;
  href: string;
  role: "summary" | "inspector" | "recovery" | "action" | "catalog";
}

export const phase0CardPlacements: Record<string, readonly Phase0CardPlacement[]> = {
  "uiai.health": [
    { surface: "Overview · System posture", href: "/", role: "summary" },
    { surface: "Nodes & Services", href: "/nodes-services", role: "summary" },
    { surface: "Capabilities", href: "/capabilities?object=uiai.health", role: "catalog" },
  ],
  "uiai.diagnostics": [
    { surface: "Live Inspector", href: "/live?inspector=diagnostics", role: "inspector" },
    { surface: "Test Lab Inspector", href: "/test-lab?inspector=diagnostics", role: "inspector" },
    { surface: "Activity", href: "/activity?view=diagnostics", role: "summary" },
    { surface: "Capabilities", href: "/capabilities?object=uiai.diagnostics", role: "catalog" },
  ],
  "focusa.project_identity": [
    { surface: "Project step in Focusa authority ladder", href: "/settings?section=scope", role: "summary" },
    { surface: "Overview", href: "/", role: "summary" },
    { surface: "Inspector", href: "/?inspector=project-identity", role: "inspector" },
  ],
  "focusa.project_card": [
    { surface: "Overview", href: "/", role: "summary" },
    { surface: "Scope Inspector", href: "/?inspector=scope", role: "inspector" },
  ],
  "focusa.workpoint_resume": [
    { surface: "Resume Workpoint", href: "/", role: "action" },
    { surface: "Overview · Continue", href: "/", role: "summary" },
  ],
  "focusa.trajectory_view": [
    { surface: "Overview", href: "/", role: "summary" },
    { surface: "Inspector", href: "/?inspector=trajectory", role: "inspector" },
  ],
  "focusa.tool_doctor": [
    { surface: "Nodes & Services", href: "/nodes-services?view=recovery", role: "recovery" },
    { surface: "Contextual recovery", href: "/help?topic=tool-readiness", role: "recovery" },
  ],
  "focusa.dxux_requirement": [
    { surface: "Contextual recovery", href: "/help?topic=recovery", role: "recovery" },
    { surface: "Help", href: "/help", role: "recovery" },
    { surface: "Capabilities", href: "/capabilities?object=focusa.dxux_requirement", role: "catalog" },
  ],
  "focusa.work_loop_status": [
    { surface: "Overview · Active now", href: "/", role: "summary" },
    { surface: "Activity", href: "/activity?view=work-loop", role: "summary" },
    { surface: "Nodes & Services", href: "/nodes-services?view=work-loop", role: "summary" },
  ],
  "focusa.device_pair_status": [
    { surface: "Nodes & Services · Pairing & Devices", href: "/nodes-services?view=pairing", role: "summary" },
  ],
  "focusa.evidence_link": [
    { surface: "Capture Evidence", href: "/evidence?action=capture", role: "action" },
    { surface: "Evidence", href: "/evidence", role: "summary" },
  ],
  "cloud.node_status": [
    { surface: "Nodes & Services", href: "/nodes-services?view=cloud", role: "summary" },
  ],
  "cloud.device_pairing": [
    { surface: "Nodes & Services · Pairing & Devices", href: "/nodes-services?view=pairing", role: "action" },
  ],
  "ai_api.health_usage": [
    { surface: "Nodes & Services · AI API", href: "/nodes-services?view=ai-api", role: "summary" },
    { surface: "Paid-action approval", href: "/activity?view=approvals", role: "action" },
  ],
};

export function placementsForCard(cardId: string): readonly Phase0CardPlacement[] {
  return phase0CardPlacements[cardId] || [];
}

export function overviewCardIds(): string[] {
  return Object.entries(phase0CardPlacements)
    .filter(([, placements]) => placements.some((placement) => placement.href === "/" && placement.role === "summary"))
    .map(([cardId]) => cardId);
}
