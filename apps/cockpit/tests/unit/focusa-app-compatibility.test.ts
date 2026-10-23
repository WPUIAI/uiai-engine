import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { negotiateFocusaPairingCompatibility } from "../../src/lib/contracts/focusa-app-compatibility";

const valid = JSON.parse(readFileSync(
  new URL("../../../../tests/fixtures/desktop-presentation/focusa-app-manifest.valid.json", import.meta.url),
  "utf8",
));

describe("Focusa sibling pairing compatibility", () => {
  it("accepts the exact supported pairing protocol set", () => {
    expect(negotiateFocusaPairingCompatibility(valid).compatibility).toEqual({
      schema: "focusa.pairing_compatibility_result.v1",
      compatible: true,
      path_b_available: true,
      recovery_action: "continue_path_b",
      reason: "compatible",
    });
  });

  it("fails closed for absent or unsupported protocol versions", () => {
    const absent = structuredClone(valid);
    delete absent.protocols.bridge;
    expect(negotiateFocusaPairingCompatibility(absent).compatibility).toMatchObject({
      compatible: false,
      path_b_available: false,
      recovery_action: "use_path_a",
      reason: "missing_protocol",
      protocol: "bridge",
    });

    const future = structuredClone(valid);
    future.protocols.pairing = "2";
    expect(negotiateFocusaPairingCompatibility(future).compatibility).toMatchObject({
      compatible: false,
      reason: "unsupported_protocol_version",
      protocol: "pairing",
      expected_version: "1",
      observed_version: "2",
    });
  });

  it("fails closed when Path B authority capability is absent", () => {
    const noInheritance = structuredClone(valid);
    noInheritance.capabilities = noInheritance.capabilities.filter((value: string) => value !== "pair.inherited");
    expect(negotiateFocusaPairingCompatibility(noInheritance).compatibility).toMatchObject({
      compatible: false,
      reason: "missing_capability",
      capability: "pair.inherited",
    });
  });

  it("rejects unknown manifest fields before compatibility negotiation", () => {
    expect(() => negotiateFocusaPairingCompatibility({ ...valid, token: "must-not-cross" })).toThrow("unsupported fields");
  });
});
