export type SidebarGroup = "work" | "create" | "prove" | "system";
export type WorkspaceState = "current" | "planned" | "guarded";

export interface WorkspaceManifestEntry {
  id: string;
  label: string;
  group: SidebarGroup;
  order: number;
  route: string;
  icon: string;
  emphasis: "primary" | "quiet";
  state: WorkspaceState;
  description: string;
  subsections: string[];
}

export const sidebarGroups: Array<{ id: SidebarGroup; label: string }> = [
  { id: "work", label: "Work" },
  { id: "create", label: "Create" },
  { id: "prove", label: "Prove" },
  { id: "system", label: "System" },
];

export const workspaceManifest: WorkspaceManifestEntry[] = [
  { id: "overview", label: "Overview", group: "work", order: 10, route: "/", icon: "⌂", emphasis: "primary", state: "current", description: "Mission Deck and current work.", subsections: [] },
  { id: "live", label: "Live", group: "work", order: 20, route: "/live", icon: "◉", emphasis: "primary", state: "current", description: "Sessions, shares, and recordings.", subsections: ["Sessions", "Recordings", "Shares"] },
  { id: "test_lab", label: "Test Lab", group: "work", order: 30, route: "/test-lab", icon: "✓", emphasis: "primary", state: "current", description: "Flows, runs, baselines, and runners.", subsections: ["Flows", "Runs", "Baselines", "Environments", "Runners"] },
  { id: "documents", label: "Documents", group: "work", order: 40, route: "/documents", icon: "▤", emphasis: "primary", state: "current", description: "Bounded source capture and documents.", subsections: ["Inbox", "Recent", "Pinned", "Reports"] },
  { id: "research", label: "Research", group: "work", order: 50, route: "/research", icon: "⌕", emphasis: "quiet", state: "planned", description: "Search, captures, collections, and packets.", subsections: ["Search", "Captures", "Collections", "Packets"] },
  { id: "studio", label: "Studio", group: "create", order: 10, route: "/studio", icon: "▧", emphasis: "quiet", state: "current", description: "Capture and inspect visual surfaces.", subsections: ["Capture", "Compare", "Analyze", "Design", "Produce"] },
  { id: "automations", label: "Automations", group: "create", order: 20, route: "/automations", icon: "↻", emphasis: "quiet", state: "planned", description: "Recipes, runs, intake, and schedules.", subsections: ["Recipes", "Runs", "Intake", "Migration", "Templates"] },
  { id: "evidence", label: "Evidence", group: "prove", order: 10, route: "/evidence", icon: "◇", emphasis: "primary", state: "current", description: "Evidence and review outputs.", subsections: ["Current Workpoint", "Recent", "Needs capture", "Needs review", "Verified", "Reports"] },
  { id: "activity", label: "Activity", group: "prove", order: 20, route: "/activity", icon: "≋", emphasis: "quiet", state: "planned", description: "Now, approvals, and history.", subsections: ["Now", "Approvals", "History"] },
  { id: "nodes_services", label: "Nodes & Services", group: "system", order: 10, route: "/nodes-services", icon: "⌘", emphasis: "quiet", state: "planned", description: "Connected nodes and protected workers.", subsections: ["UIAI Engine", "Focusa Local", "Focusa Cloud", "AI API", "Pairing & Devices", "Capacity", "Sync", "Updates"] },
  { id: "capabilities", label: "Capabilities", group: "system", order: 20, route: "/capabilities", icon: "✦", emphasis: "quiet", state: "planned", description: "Capability catalog and access state.", subsections: [] },
];

export const footerDestinations = [
  { id: "settings", label: "Settings", route: "/settings", icon: "⚙" },
  { id: "help", label: "Help", route: "/help", icon: "?" },
];

export function workspaceForPath(pathname: string): WorkspaceManifestEntry | undefined {
  return workspaceManifest.find((item) => pathname === item.route || (item.id === "live" && pathname === "/runs"));
}
