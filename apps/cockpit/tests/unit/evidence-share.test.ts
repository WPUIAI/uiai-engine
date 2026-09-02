import { describe, expect, it } from "vitest";
import { humanBytes, packetMatchesWorkpoint, sourceHost, type EvidenceShareManifest } from "../../src/lib/evidence-share";

const manifest = { scope: { workpoint_ref: "workpoint:homepage", continuity_ref: "focusa-dev-homepage-main" } } as EvidenceShareManifest;

describe("Evidence Share progressive disclosure", () => {
  it("renders bounded human source labels without leaking credentials", () => {
    expect(sourceHost("https://user:secret@focusa.dev/path?token=secret")).toBe("focusa.dev");
    expect(sourceHost("not a URL")).toBe("Source unavailable");
    expect(sourceHost()).toBe("Source not disclosed");
  });
  it("matches only exact Workpoint scope when a filter is active", () => {
    expect(packetMatchesWorkpoint(manifest, "workpoint:homepage")).toBe(true);
    expect(packetMatchesWorkpoint(manifest, "workpoint:other")).toBe(false);
    expect(packetMatchesWorkpoint(undefined, "workpoint:homepage")).toBe(false);
    expect(packetMatchesWorkpoint(manifest)).toBe(true);
  });
  it("formats media sizes without false precision or invalid values", () => {
    expect(humanBytes(337199)).toContain("337");
    expect(humanBytes(-1)).toBe("Unknown size");
  });
});
