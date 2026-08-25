import { describe, expect, it } from "vitest";
import { AGENT_FIRST_BROWSER_SCHEMA_IDS, assertAgentFirstBrowserSchemaId, isAgentFirstBrowserSchemaId } from "../../src/lib/contracts/agent-first-browser-registry";

describe("agent-first browser schema registry", () => {
  it("contains the ten C01 schemas", () => {
    expect(AGENT_FIRST_BROWSER_SCHEMA_IDS).toHaveLength(10);
    expect(new Set(AGENT_FIRST_BROWSER_SCHEMA_IDS).size).toBe(10);
  });

  it("fails closed for unknown schema ids", () => {
    expect(isAgentFirstBrowserSchemaId("uiai.unknown.v1")).toBe(false);
    expect(() => assertAgentFirstBrowserSchemaId("uiai.unknown.v1")).toThrow("Unsupported");
  });
});
