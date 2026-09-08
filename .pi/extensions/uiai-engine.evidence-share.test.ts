import { afterEach, describe, expect, mock, test } from "bun:test";
import { readFileSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const source = readFileSync(new URL("./uiai-engine.ts", import.meta.url), "utf8");

describe("Evidence Share Packet Pi toolset", () => {
	test("registers progressive-disclosure tools and canonical endpoints", () => {
		for (const value of [
			'uiai_screenshot',
			'uiai_evidence_share_list',
			'uiai_evidence_share_inspect',
			'uiai_evidence_share_verify',
			'uiai_evidence_share_resolve',
			'/api/screenshot/share',
			'/api/screenshot/share/{id}/verify',
		]) expect(source).toContain(value);
	});

	test("promotes human descriptor and clickable URL before technical refs", () => {
		const screenshotBlock = source.slice(source.indexOf('name: "uiai_screenshot"'), source.indexOf('name: "uiai_frame_catalog"'));
		expect(screenshotBlock.indexOf('descriptor: "Screenshot Evidence Share Packet"')).toBeGreaterThan(-1);
		expect(screenshotBlock.indexOf("artifact_url: data.artifact_url")).toBeGreaterThan(-1);
		expect(screenshotBlock.indexOf("artifact_url: data.artifact_url")).toBeLessThan(screenshotBlock.indexOf("artifact_ref: data.artifact_ref"));
		expect(screenshotBlock).toContain("beautiful portable Evidence Share Packet");
	});

	test("requires exact packet identity for inspect verify and resolve", () => {
		expect(source).toContain('pattern: "^[a-f0-9]{64}$"');
		expect(source).toContain('descriptor: "Screenshot Evidence Share Packet"');
	});
});

// Framework UI is mocked; the real adapter and shared delivery code execute.
// Reuse an installed TypeBox when the standalone Go checkout has no JS dependencies.
const actualTypebox = await import(process.env.UIAI_TEST_TYPEBOX || "typebox");
mock.module("typebox", () => actualTypebox);
mock.module("@earendil-works/pi-coding-agent", () => ({ keyHint: () => "expand" }));
mock.module("@earendil-works/pi-tui", () => ({ Text: class { constructor(public text: string) {} } }));
const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });
const fixture = (base = "https://evidence.example/customer/") => ({
  artifact_url: base + "record/", portable_url: base + "record/portable.zip",
  epwa_delivery: { schema: "uiai.epwa_delivery.v1", state: "ready", artifact: { artifact_ref: "artifact:test" },
    epwa: { record_url: base + "record/", portable_url: base + "record/portable.zip" } },
});
async function tools() {
  const registered = new Map<string, any>();
  const extension = (await import("./uiai-engine.ts")).default;
  extension({ on() {}, registerCommand() {}, registerTool(tool: any) { registered.set(tool.name, tool); } } as any);
  return registered;
}
test("native screenshot automatically returns and visibly renders its HTTPS evidence", async () => {
  globalThis.fetch = (async () => Response.json(fixture())) as any;
  const tool = (await tools()).get("uiai_browser_screenshot");
  const result = await tool.execute("capture", { session_id: "fixture" });
  expect(JSON.parse(result.content[0].text).artifact_url).toBe(fixture().artifact_url);
  const view = tool.renderResult(result, { expanded: false }, { fg: (_: string, value: string) => value });
  expect(view.text).toContain(fixture().artifact_url);
  expect(view.text).toContain("Evidence ready");
});
test("native screenshot refuses raw, missing and pending delivery instead of silently stripping", async () => {
  const tool = (await tools()).get("uiai_browser_screenshot");
  for (const response of [{ screenshot: "private-bytes" }, { width: 1 },
    { epwa_delivery: { state: "pending_reconcile" }, recovery_ref: "reconcile:test" }]) {
    globalThis.fetch = (async () => Response.json(response)) as any;
    await expect(tool.execute("capture", { session_id: "fixture" })).rejects.toThrow("EPWA delivery unavailable");
  }
});
test("concurrent native requests forward their own exact scope without account defaults", async () => {
  const seen: string[] = [];
  globalThis.fetch = (async (_url: any, options: any) => {
    const headers = new Headers(options.headers);
    seen.push(headers.get("X-UIAI-Project-Ref") || "missing");
    await new Promise(resolve => setTimeout(resolve, 1));
    return Response.json(fixture());
  }) as any;
  const tool = (await tools()).get("uiai_browser_screenshot");
  expect(tool.parameters.properties.focusa_scope).toBeDefined();
  await Promise.all(["project:a", "project:b"].map(project_ref => tool.execute("capture", {
    session_id: project_ref, focusa_scope: { project_ref },
  })));
  await tool.execute("capture", { session_id: "unscoped" });
  expect(seen).toEqual(["project:a", "project:b", "missing"]);
  expect(source).not.toContain("/home/wpuiai");
  expect(source).not.toContain("focusa-cont-uiai-engine-");
});
test("HTTP publication failure includes the committed artifact reconciliation handle", async () => {
  globalThis.fetch = (async () => Response.json({ error: { message: "publication pending" },
    recovery_ref: "reconcile:test", artifact_ref: "artifact:test" }, { status: 503 })) as any;
  const tool = (await tools()).get("uiai_browser_screenshot");
  await expect(tool.execute("capture", { session_id: "fixture" })).rejects.toThrow("reconcile:test");
});

test("collapsed rendering preserves thrown delivery failures rather than showing ok", async () => {
  const tool = (await tools()).get("uiai_browser_screenshot");
  const colors: string[] = [];
  const rendered = tool.renderResult({ content: [{ type: "text", text: "EPWA delivery unavailable" }] },
    { expanded: false }, { fg: (color: string, text: string) => { colors.push(color); return text; } }, { isError: true });
  expect(rendered.text).toContain("EPWA delivery unavailable");
  expect(rendered.text).not.toContain("UIAI ok");
  expect(colors).toContain("error");
});

test("canonical installer ships a self-contained adapter at an arbitrary location", async () => {
  const directory = mkdtempSync(join(tmpdir(), "epwa-install-"));
  try {
    const target = join(directory, "arbitrary location", "extensions", "evidence.ts");
    const installer = fileURLToPath(new URL("../../scripts/install-agent-integrations.sh", import.meta.url));
    const processResult = Bun.spawnSync(["bash", installer], { env: {
      ...process.env, HOME: directory, UIAI_PI_EXTENSION_DEST: target,
      UIAI_MCP_CONFIG_DEST: join(directory, "mcp.json"), UIAI_API_KEY: "", UIAI_BEARER_TOKEN: "", DRY_RUN: "0",
    } });
    expect(processResult.exitCode).toBe(0);
    expect(readFileSync(join(dirname(target), "uiai", "epwa-contract.mjs"), "utf8"))
      .toBe(readFileSync(new URL("./uiai/epwa-contract.mjs", import.meta.url), "utf8"));
    const registered = new Map<string, any>();
    const extension = (await import(pathToFileURL(target).href)).default;
    extension({ on() {}, registerCommand() {}, registerTool(tool: any) { registered.set(tool.name, tool); } });
    globalThis.fetch = (async () => Response.json(fixture("https://relocated.example/installation/"))) as any;
    const result = await registered.get("uiai_screenshot").execute("capture", { url: "https://example.com/" });
    expect(JSON.parse(result.content[0].text).artifact_url).toBe("https://relocated.example/installation/record/");
  } finally {
    rmSync(directory, { recursive: true }); // Only this test's newly created fixture tree.
  }
});
