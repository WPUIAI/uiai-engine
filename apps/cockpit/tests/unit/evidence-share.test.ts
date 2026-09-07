import { afterEach, describe, expect, it, vi } from "vitest";
import { engineClient } from "../../src/lib/engine-client";
afterEach(() => vi.unstubAllGlobals());
import { humanBytes, normalizeEvidenceShareManifest, packetMatchesWorkpoint, secureEvidenceURL, sourceHost, type EvidenceShareManifest } from "../../src/lib/evidence-share";

const manifest = { scope: { workpoint_ref: "workpoint:homepage", continuity_ref: "focusa-dev-homepage-main" } } as EvidenceShareManifest;

describe("Evidence Share progressive disclosure", () => {
  it("uses explicit settings scope and revision-bound update/reset operations", async () => {
    vi.stubGlobal("window", { localStorage: { getItem: () => null }, setTimeout, clearTimeout });
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ schema: "settings", scope: {}, revision: 3, sources: [], values: {}, reset: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await engineClient.evidenceShareSettings({ project_ref: "project:one", project_root: "/private/root", workstream_ref: "workstream:one", continuity_id: "continuity:other" });
    const url = new URL(String(fetchMock.mock.calls[0][0]));
    expect(url.searchParams.get("project_ref")).toBe("project:one");
    expect(url.searchParams.get("workstream_ref")).toBe("workstream:one");
    const mutation = { project_ref: "project:one", workstream_ref: "workstream:one", expected_revision: 3, values: { privacy: { redact: true } } };
    await engineClient.updateEvidenceShareSettings(mutation);
    expect(fetchMock.mock.calls[1][1]?.method).toBe("PUT");
    expect(JSON.parse(String(fetchMock.mock.calls[1][1]?.body))).toEqual(mutation);
    await engineClient.resetEvidenceShareSettings({ project_ref: "project:one", expected_revision: 3 });
    expect(fetchMock.mock.calls[2][1]?.method).toBe("DELETE");
  });
  it("accepts all current package contracts without exposing storage paths", () => {
    const digest = "a".repeat(64);
    const screenshot = normalizeEvidenceShareManifest({ schema: "uiai.screenshot_evidence_share.v1", artifact_ref: "artifact:shot", artifact_sha256: digest, format: "png", width: 800, height: 600 });
    expect(screenshot.format).toBe("png");
    expect(screenshot.width).toBe(800);
    const generic = normalizeEvidenceShareManifest({ schema: "uiai.epwa_generic_artifact.v1", artifact_ref: "artifact:report", asset_sha256: digest, media_type: "application/json", artifact_path: "/private/report", scope: { workpoint_ref: "wp:one" } });
    expect(generic.format).toBe("application/json");
    expect(generic.artifact_sha256).toBe(digest);
    expect(generic.digest_label).toBe("Payload SHA-256");
    expect(generic).not.toHaveProperty("artifact_path");
    const immutable = normalizeEvidenceShareManifest({ schema: "uiai.evidence_artifact_manifest.v1", artifact_id: "artifact:bundle", integrity: { manifest_sha256: digest }, scope: { workpoint: { workpoint_ref: "wp:one" } }, policy: { access_class: "public_safe" } });
    expect(immutable.artifact_ref).toBe("artifact:bundle");
    expect(immutable.access).toBe("public_safe");
    expect(immutable.digest_label).toBe("Manifest SHA-256");
    expect(packetMatchesWorkpoint(immutable, "wp:one")).toBe(true);
    expect(immutable.width).toBeUndefined();
    expect(immutable.truth_notice).toContain("does not establish");
    expect(() => normalizeEvidenceShareManifest({ schema: "unknown" })).toThrow("Unsupported");
  });
  it("only offers credential-free HTTPS package links", () => {
    expect(secureEvidenceURL("https://evidence.example/share/abc/")).toBe("https://evidence.example/share/abc/");
    for (const url of ["javascript:alert(1)", "data:text/html,hello", "http://example.com", "https://user:secret@example.com", undefined]) expect(secureEvidenceURL(url)).toBeUndefined();
  });
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
