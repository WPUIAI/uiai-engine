export type SidebarFixtureId = "expanded" | "compact" | "overlay" | "empty-context" | "missing-scope" | "blocked-entitlement";

export interface SidebarFixture {
  id: SidebarFixtureId;
  mode: "expanded" | "compact";
  overlay: boolean;
  scope: "connected" | "empty" | "missing";
  entitlement: "available" | "blocked";
  expected: string;
}

/** Test-only shell states; these are not product data or default navigation values. */
export const sidebarFixtures: SidebarFixture[] = [
  { id: "expanded", mode: "expanded", overlay: false, scope: "connected", entitlement: "available", expected: "labels, groups, and footer are visible" },
  { id: "compact", mode: "compact", overlay: false, scope: "connected", entitlement: "available", expected: "icon rail preserves selected workspace and keyboard access" },
  { id: "overlay", mode: "expanded", overlay: true, scope: "connected", entitlement: "available", expected: "focus enters overlay and Escape returns focus" },
  { id: "empty-context", mode: "expanded", overlay: false, scope: "empty", entitlement: "available", expected: "no Workpoint or object is fabricated" },
  { id: "missing-scope", mode: "expanded", overlay: false, scope: "missing", entitlement: "available", expected: "scope recovery remains explicit" },
  { id: "blocked-entitlement", mode: "expanded", overlay: false, scope: "connected", entitlement: "blocked", expected: "capability is visible but execution is blocked before allocation" },
];
