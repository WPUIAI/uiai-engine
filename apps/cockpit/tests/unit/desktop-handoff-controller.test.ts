import { afterEach, describe, expect, it, vi } from "vitest";
import { createFocusaHandoff, openInFocusa } from "../../src/lib/controllers/desktop-handoff-controller";

afterEach(() => vi.unstubAllGlobals());

describe("Cockpit Focusa handoff controller", () => {
  it("creates a short-lived token-free protocol-v1 intent", () => {
    vi.stubGlobal("crypto", { randomUUID: () => "12345678-1234-1234-1234-123456789abc" });
    const intent = createFocusaHandoff("workpoint", "wp_01", new Date("2026-08-03T00:00:00Z"));
    expect(intent.scheme).toBe("focusa");
    expect(intent.route).toBe("workpoint");
    expect(intent.target_ref).toBe("wp_01");
    expect(intent.protocol_version).toBe("1");
    expect(intent.expires_at).toBe("2026-08-03T00:02:00.000Z");
    expect(JSON.stringify(intent)).not.toMatch(/token|secret|authorization/i);
  });

  it("returns truthful recovery when no native sibling-app launcher exists", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "12345678-1234-1234-1234-123456789abc" });
    vi.stubGlobal("window", {});
    const receipt = await openInFocusa("workpoint", "wp_01");
    expect(receipt.status).toBe("unavailable");
    expect(receipt.reason_code).toBe("native_shell_required");
    expect(receipt.target_app).toBe("focusa-menubar");
  });
});
