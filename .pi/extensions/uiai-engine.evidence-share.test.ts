import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";

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
