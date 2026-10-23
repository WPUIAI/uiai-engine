import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import {
  DESKTOP_PRESENTATION_SCHEMA_IDS,
  assertOpaqueRef,
  parseAppHandoffIntent,
  parseDesktopContractFixtureBundle,
  parseFocusaAppManifest,
} from "../../src/lib/contracts/desktop-presentation";

const fixture = (name: string): unknown => JSON.parse(
  readFileSync(new URL(`../../../../tests/fixtures/desktop-presentation/${name}`, import.meta.url), "utf8"),
);

describe("desktop presentation contract parity", () => {
  it("parses the canonical shared fixture bundle", () => {
    const parsed = parseDesktopContractFixtureBundle(fixture("valid-contracts.json"));
    expect(parsed.runtime_manifest.schema).toBe("uiai.browser_runtime_manifest.v1");
    expect(parsed.presentation_request.mode).toBe("full");
    expect(parsed.handoff_intent.target_ref).toBe("browser-session:001");
    expect(parsed.app_manifest.protocols.desktop_presentation).toBe("1");
    expect(DESKTOP_PRESENTATION_SCHEMA_IDS).toHaveLength(7);

    const menubar = parseFocusaAppManifest(fixture("focusa-app-manifest.valid.json"));
    expect(menubar.app).toBe("focusa-menubar");
    expect(menubar.capabilities).toContain("cockpit.open");
  });

  it.each([
    "handoff-secret.invalid.json",
    "handoff-raw-path.invalid.json",
    "handoff-private-url.invalid.json",
    "handoff-unknown-route.invalid.json",
  ])("fails closed for %s", (name) => {
    expect(() => parseAppHandoffIntent(fixture(name))).toThrow();
  });

  it.each([
    "/tmp/private",
    "https://example.com",
    "session?token=secret",
    "session#fragment",
    "C:\\Users\\operator",
    "contains space",
  ])("rejects non-opaque ref %s", (value) => {
    expect(() => assertOpaqueRef(value)).toThrow();
  });
});
