import { beforeAll, describe, expect, mock, test } from "bun:test";

beforeAll(() => {
	mock.module("@earendil-works/pi-coding-agent", () => ({ keyHint: () => "expand" }));
	mock.module("@earendil-works/pi-tui", () => ({ Text: class Text { constructor(public text: string) {} } }));
	mock.module("typebox", () => ({ Type: { Object: (x: any) => x, String: (x?: any) => x || {}, Optional: (x: any) => x, Number: (x?: any) => x || {}, Boolean: (x?: any) => x || {}, Any: (x?: any) => x || {}, Array: (x: any) => [x] } }));
});

describe("buildResearchDiagnosticsPacket", () => {
	test("builds research packet from Focusa metadata and redacts secrets", async () => {
		const { buildResearchDiagnosticsPacket } = await import("./uiai-engine");
		const packet = buildResearchDiagnosticsPacket({
			goal: "research UIAI",
			responses: [{
				focusa: {
					target_ref: "browser:https://example.com/page?token=secret&ok=1#frag",
					evidence_ref: "uiai-search:brave:abc:1",
					summary: "Search returned one result",
				},
				results: [{ title: "Example" }],
			}],
		});
		expect(packet.schema).toBe("uiai.focusa_research_diagnostics_packet.v1");
		expect(packet.mode).toBe("research");
		expect(packet.scope_status).toBe("missing");
		expect(packet.captures).toHaveLength(1);
		expect(packet.recommended_focusa.preferred_tool).toBe("focusa_evidence_capture");
		expect(packet.headless_next_action).toBeTruthy();
		expect(packet.render.summary_line).toContain("scope=missing");
		expect(packet.render.summary_line).toContain("tool=focusa_evidence_capture");
		const encoded = JSON.stringify(packet);
		expect(encoded.length).toBeLessThanOrEqual(8 * 1024);
		expect(encoded).not.toContain("secret");
		expect(encoded).not.toContain("#frag");
		expect(encoded).toContain("token=REDACTED");
	});

	test("builds diagnostics packet with cleanup and present scope", async () => {
		const { buildResearchDiagnosticsPacket } = await import("./uiai-engine");
		const packet = buildResearchDiagnosticsPacket({
			mode: "diagnose",
			goal: "diagnose page",
			focusa_scope: { project_root: "/repo", continuity_id: "cont" },
			cleanup_session_id: "sess1",
			responses: [{
				focusa: {
					target_ref: "browser:https://example.com/app",
					evidence_ref: "uiai-diagnostics:session=sess1:seq=2",
					summary: "Diagnostics failed_requests=1",
				},
				summary: { console_errors: 1, failed_requests: 1, exceptions: 1 },
			}],
		});
		expect(packet.mode).toBe("diagnose");
		expect(packet.scope_status).toBe("present");
		expect(packet.diagnostics_summary?.console_errors).toBe(1);
		expect(packet.recommended_focusa.preferred_tool).toBe("focusa_browser_diagnostics_intake");
		expect(packet.cleanup?.tool).toBe("uiai_browser_close");
	});

	test("enforces budget on oversized input", async () => {
		const { buildResearchDiagnosticsPacket } = await import("./uiai-engine");
		const responses = Array.from({ length: 40 }, (_, i) => ({ focusa: { target_ref: `browser:https://example.com/${i}`, evidence_ref: `uiai-browser:session=s:read:${i}`, summary: "x".repeat(1000) } }));
		const packet = buildResearchDiagnosticsPacket({ goal: "x".repeat(1000), responses });
		expect(JSON.stringify(packet).length).toBeLessThanOrEqual(8 * 1024);
		expect(packet.captures.length).toBeLessThanOrEqual(8);
	});
});


test("buildResearchDiagnosticsPacket recognizes snapshot packet capture", async () => {
	const { buildResearchDiagnosticsPacket } = await import("./uiai-engine");
	const packet = buildResearchDiagnosticsPacket({
		goal: "snapshot proof",
		responses: [{ focusa: { target_ref: "browser:https://example.com", evidence_ref: "uiai-browser:session=s:snapshot:1", summary: "Snapshot 2 refs" } }],
	});
	expect(packet.captures[0].type).toBe("snapshot");
	expect(packet.recommended_focusa.preferred_tool).toBe("focusa_evidence_capture");
});
