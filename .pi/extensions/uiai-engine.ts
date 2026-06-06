import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { keyHint } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import { Type } from "typebox";

const DEFAULT_ENGINE_URL = "http://localhost:7456";
const REQUEST_TIMEOUT_MS = Number(process.env.UIAI_PI_TIMEOUT_MS || 30000);
const UIAI_WIDGET_STATE_ENTRY = "uiai-widget-visibility";
const RESEARCH_DIAGNOSTICS_PACKET_SCHEMA = "uiai.focusa_research_diagnostics_packet.v1";
const MAX_PACKET_BYTES = 8 * 1024;
const MAX_CAPTURE_SUMMARY_CHARS = 500;
const MAX_GOAL_CHARS = 240;
const MAX_NEXT_ACTION_CHARS = 240;
const MAX_DIAGNOSTICS_SUMMARY_BYTES = 2 * 1024;
const MAX_ARGS_PREVIEW_BYTES = 2 * 1024;
const MAX_CAPTURES = 8;
const MAX_TARGET_REFS = 16;
const MAX_EVIDENCE_REFS = 32;
const MAX_ACTIVE_OBJECT_HINTS = 16;

type PacketMode = "research" | "diagnose" | "proof";
type ScopeStatus = "present" | "missing" | "partial" | "mismatch_candidate";

type FocusaScope = {
	project_root?: string;
	continuity_id?: string;
	workpoint_id?: string;
	evidence_ref?: string;
};

type PacketCapture = {
	type: "search" | "source_markdown" | "read" | "diagnostics" | "error" | string;
	evidence_ref: string;
	target_ref: string;
	title?: string;
	summary: string;
};

type PacketDiagnosticsSummary = {
	console_errors: number;
	failed_requests: number;
	top_findings: string[];
};

type PacketActiveObjectHint = {
	kind: "url" | "endpoint" | "selector" | "source" | "search_result" | string;
	hint: string;
};

type ResearchDiagnosticsPacket = {
	schema: typeof RESEARCH_DIAGNOSTICS_PACKET_SCHEMA;
	mode: PacketMode;
	goal: string;
	scope_status: ScopeStatus;
	focusa_scope?: FocusaScope;
	target_refs: string[];
	evidence_refs: string[];
	captures: PacketCapture[];
	diagnostics_summary?: PacketDiagnosticsSummary;
	active_object_hints?: PacketActiveObjectHint[];
	recommended_focusa: {
		preferred_tool: "focusa_evidence_capture" | "focusa_browser_diagnostics_intake" | string;
		fallback_tool?: string;
		args_preview?: Record<string, any>;
		next_tools?: string[];
	};
	recommended_next_action: string;
	render: {
		summary_line: string;
		expandable_json_ref?: string;
	};
	headless_next_action: string;
	cleanup?: {
		session_id?: string;
		recommended_action: "close_when_done" | "keep_for_followup" | "none" | string;
		tool?: "uiai_browser_close" | string;
	};
};

function engineUrl(): string {
	return (process.env.UIAI_ENGINE_URL || DEFAULT_ENGINE_URL).replace(/\/$/, "");
}

function authHeaders(): Record<string, string> {
	const headers: Record<string, string> = {};
	if (process.env.UIAI_API_KEY) headers["X-API-Key"] = process.env.UIAI_API_KEY;
	if (process.env.UIAI_BEARER_TOKEN) headers.Authorization = `Bearer ${process.env.UIAI_BEARER_TOKEN}`;
	return headers;
}

async function callEngine(path: string, init?: RequestInit): Promise<any> {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
	try {
		const res = await fetch(`${engineUrl()}${path}`, {
			...init,
			signal: controller.signal,
			headers: { "Content-Type": "application/json", ...authHeaders(), ...(init?.headers || {}) },
		});
		const text = await res.text();
		let data: any = text;
		try {
			data = text ? JSON.parse(text) : {};
		} catch {
			// Keep raw text for non-JSON responses.
		}
		if (!res.ok) {
			throw new Error(formatEngineError(data, res.status, path));
		}
		return data;
	} finally {
		clearTimeout(timeout);
	}
}

function formatEngineError(data: any, status: number, path: string) {
	if (typeof data === "string") return `UIAI ${status} ${path}: ${data}`;
	const message = data?.message || data?.error || "request failed";
	const parts = [`UIAI ${status}`, path];
	if (data?.error_id) parts.push(`id=${data.error_id}`);
	if (data?.error_class) parts.push(`class=${data.error_class}`);
	let out = `${parts.join(" ")}: ${message}`;
	if (data?.suggested_next_action) out += `\nNext: ${data.suggested_next_action}`;
	if (data?.diagnostics) out += `\nDiagnostics: run uiai_errors or GET ${data.diagnostics}`;
	return out;
}

function textResult(data: any, details: Record<string, any> = {}) {
	return {
		content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }],
		details,
	};
}

function withoutScreenshot(data: any) {
	const summary = { ...data };
	delete summary.screenshot;
	return summary;
}

function cleanBody(body: Record<string, any>) {
	return Object.fromEntries(Object.entries(body).filter(([, value]) => value !== undefined));
}

function post(path: string, body: Record<string, any>) {
	return callEngine(path, { method: "POST", body: JSON.stringify(cleanBody(body)) });
}

async function runGuidedPacketWorkflow(mode: PacketMode, input: string, ctx: any) {
	const focusa_scope = {
		project_root: "/home/wpuiai/uiai-engine",
		continuity_id: "focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc",
		evidence_ref: `pi-uiai-${mode}-packet`,
	};
	const responses: any[] = [];
	let selectedUrl = input.trim();
	let sessionId = "";
	let sessionClosed = false;
	if (mode === "research") {
		const search = await post("/api/search", { query: input, limit: 1 });
		responses.push(search);
		selectedUrl = search?.results?.[0]?.url || "";
		if (!selectedUrl) throw new Error("UIAI research search returned no selectable URL");
	}
	if (mode === "diagnose") {
		const diagnostics = await callEngine(`/api/session/${encodeURIComponent(input.trim())}/diagnostics?limit=100`);
		const packet = await post("/api/agent/research-packet", {
			mode,
			goal: `Diagnose UIAI browser session ${input.trim()}`,
			responses: [diagnostics],
			focusa_scope,
			recommended_next_action: "Pass recommended_focusa.args_preview to focusa_browser_diagnostics_intake if scope is canonical.",
		});
		ctx.ui.setEditorText(`UIAI diagnose packet ready. Preferred Focusa tool: ${packet?.recommended_focusa?.preferred_tool || "unknown"}\n\n${JSON.stringify(packet?.recommended_focusa?.args_preview || packet, null, 2)}`);
		ctx.ui.notify("UIAI diagnose packet composed", "info");
		return;
	}
	try {
		const opened = await post("/api/session", { url: selectedUrl, width: 1280, height: 800, focusa_scope });
		sessionId = opened?.session?.id || opened?.id || "";
		if (!sessionId) throw new Error("UIAI browser session id missing");
		responses.push(
			await post(`/api/session/${sessionId}/read`, { max_chars: 2000, include_links: true }),
			await post(`/api/session/${sessionId}/snapshot`, { interactive: true, compact: true }),
			await callEngine(`/api/session/${sessionId}/diagnostics?limit=100`),
		);
		const packet = await post("/api/agent/research-packet", {
			mode,
			goal: mode === "research" ? `Research packet for ${input}` : `Proof packet for ${selectedUrl}`,
			responses,
			focusa_scope,
			recommended_next_action: "Capture packet evidence with Focusa, then continue from the Workpoint.",
			cleanup_session_id: sessionId,
		});
		try {
			await callEngine(`/api/session/${sessionId}`, { method: "DELETE" });
			sessionClosed = true;
		} catch {
			sessionClosed = false;
		}
		ctx.ui.setEditorText(`UIAI ${mode} packet ready for ${selectedUrl}\nSession closed: ${sessionClosed}\nPreferred Focusa tool: ${packet?.recommended_focusa?.preferred_tool || "unknown"}\n\n${JSON.stringify(packet?.recommended_focusa?.args_preview || packet, null, 2)}`);
		ctx.ui.notify(`UIAI ${mode} packet composed`, "info");
	} finally {
		if (sessionId && !sessionClosed) {
			try { await callEngine(`/api/session/${sessionId}`, { method: "DELETE" }); } catch { /* best-effort cleanup */ }
		}
	}
}

function truncateText(value: any, maxChars: number): string {
	const text = String(value ?? "").trim();
	if (maxChars <= 0) return "";
	const chars = Array.from(text);
	if (chars.length <= maxChars) return text;
	return `${chars.slice(0, Math.max(0, maxChars - 1)).join("")}…`;
}

function packetByteSize(value: any): number {
	return new TextEncoder().encode(JSON.stringify(value)).length;
}

function isSecretQueryKey(key: string): boolean {
	const lower = key.toLowerCase();
	return ["token", "key", "secret", "auth", "session", "password", "passwd", "code", "sig", "signature", "jwt", "credential", "authorization", "cookie", "api_key", "apikey", "access_token", "refresh_token"].some((part) => lower.includes(part));
}

function sanitizeUrl(raw: string): string {
	const value = String(raw || "").trim();
	if (!value) return "";
	try {
		const parsed = new URL(value);
		parsed.hash = "";
		for (const key of Array.from(parsed.searchParams.keys())) {
			if (isSecretQueryKey(key)) parsed.searchParams.set(key, "REDACTED");
		}
		return truncateText(parsed.toString(), 2048);
	} catch {
		return truncateText(value, 2048);
	}
}

function sanitizeTargetRef(ref: any): string {
	const value = String(ref || "").trim();
	if (!value) return "";
	if (value.startsWith("browser:http://") || value.startsWith("browser:https://")) return `browser:${sanitizeUrl(value.slice("browser:".length))}`;
	if (value.startsWith("source-markdown:http://") || value.startsWith("source-markdown:https://")) return `source-markdown:${sanitizeUrl(value.slice("source-markdown:".length))}`;
	if (value.startsWith("http://") || value.startsWith("https://")) return sanitizeUrl(value);
	return truncateText(value, 500);
}

function boundedStrings(values: any[], maxItems: number, maxChars = 500): string[] {
	return (Array.isArray(values) ? values : []).slice(0, maxItems).map((value) => truncateText(value, maxChars)).filter(Boolean);
}

function scopeStatus(scope: any): ScopeStatus {
	if (!scope || typeof scope !== "object") return "missing";
	if (scope.project_root && scope.continuity_id) return "present";
	if (scope.project_root || scope.continuity_id || scope.workpoint_id || scope.evidence_ref) return "partial";
	return "missing";
}

function primaryFocusaTool(captures: PacketCapture[]): string {
	return captures.some((capture) => capture.type === "diagnostics" || capture.type === "error") ? "focusa_browser_diagnostics_intake" : "focusa_evidence_capture";
}

function captureFromResponse(response: any): PacketCapture | undefined {
	const focusa = response?.focusa;
	if (!focusa || typeof focusa !== "object") return undefined;
	let type = "search";
	if (String(focusa.evidence_ref || "").includes("diagnostics")) type = "diagnostics";
	else if (String(focusa.evidence_ref || "").includes("error")) type = "error";
	else if (String(focusa.evidence_ref || "").includes("uiai-source-markdown")) type = "source_markdown";
	else if (String(focusa.evidence_ref || "").includes(":read:")) type = "read";
	else if (String(focusa.evidence_ref || "").includes(":snapshot:")) type = "snapshot";
	else if (String(focusa.evidence_ref || "").includes("uiai-screenshot:")) type = "screenshot";
	else if (String(focusa.evidence_ref || "").includes("uiai-share:")) type = "share";
	return {
		type,
		evidence_ref: truncateText(focusa.evidence_ref || "", 240),
		target_ref: sanitizeTargetRef(focusa.target_ref || response?.url || ""),
		title: truncateText(response?.title || response?.results?.[0]?.title || "", 200) || undefined,
		summary: truncateText(focusa.summary || response?.message || response?.description || "UIAI evidence capture", MAX_CAPTURE_SUMMARY_CHARS),
	};
}

function diagnosticsSummaryFromResponses(responses: any[]): PacketDiagnosticsSummary | undefined {
	const findings: string[] = [];
	let consoleErrors = 0;
	let failedRequests = 0;
	for (const response of responses) {
		const summary = response?.summary || response?.diagnostics_summary || response?.details?.diagnostics_summary;
		if (!summary) continue;
		consoleErrors += Number(summary.console_errors || 0);
		failedRequests += Number(summary.failed_requests || 0);
		if (Number(summary.exceptions || 0) > 0) findings.push(`exceptions=${summary.exceptions}`);
		if (Number(summary.http_4xx || 0) > 0) findings.push(`http_4xx=${summary.http_4xx}`);
		if (Number(summary.http_5xx || 0) > 0) findings.push(`http_5xx=${summary.http_5xx}`);
	}
	if (!consoleErrors && !failedRequests && findings.length === 0) return undefined;
	let out: PacketDiagnosticsSummary = { console_errors: consoleErrors, failed_requests: failedRequests, top_findings: boundedStrings(findings, 12, 240) };
	while (packetByteSize(out) > MAX_DIAGNOSTICS_SUMMARY_BYTES && out.top_findings.length > 0) out.top_findings.pop();
	return out;
}

function sanitizeArgsPreview(args: Record<string, any>): Record<string, any> | undefined {
	const out: Record<string, any> = {};
	for (const [key, value] of Object.entries(args || {})) {
		if (isSecretQueryKey(key)) continue;
		if (typeof value === "string") out[truncateText(key, 80)] = key.includes("ref") || value.startsWith("http") || value.startsWith("browser:http") ? sanitizeTargetRef(value) : truncateText(value, 500);
		else if (typeof value === "boolean" || typeof value === "number") out[truncateText(key, 80)] = value;
		else if (Array.isArray(value)) out[truncateText(key, 80)] = boundedStrings(value, 16, 240);
		else if (value && typeof value === "object") out[truncateText(key, 80)] = sanitizeArgsPreview(value as Record<string, any>);
	}
	while (packetByteSize(out) > MAX_ARGS_PREVIEW_BYTES) {
		const first = Object.keys(out)[0];
		if (!first) break;
		delete out[first];
	}
	return Object.keys(out).length ? out : undefined;
}

export function buildResearchDiagnosticsPacket(input: {
	mode?: string;
	goal: string;
	responses: any[];
	focusa_scope?: FocusaScope;
	recommended_next_action?: string;
	cleanup_session_id?: string;
	expandable_json_ref?: string;
}): ResearchDiagnosticsPacket {
	const mode: PacketMode = input.mode === "diagnose" || input.mode === "proof" ? input.mode : "research";
	const captures = input.responses.map(captureFromResponse).filter(Boolean).slice(0, MAX_CAPTURES) as PacketCapture[];
	const targetRefs = boundedStrings(captures.map((capture) => capture.target_ref), MAX_TARGET_REFS).map(sanitizeTargetRef);
	const evidenceRefs = boundedStrings(captures.map((capture) => capture.evidence_ref), MAX_EVIDENCE_REFS, 240);
	const preferredTool = primaryFocusaTool(captures);
	const primaryCapture = captures[0];
	const recommendedNext = truncateText(input.recommended_next_action || (mode === "diagnose" ? "Call focusa_browser_diagnostics_intake with args_preview or inspect failed requests." : "Call focusa_evidence_capture with args_preview, then resolve active object if needed."), MAX_NEXT_ACTION_CHARS);
	let packet: ResearchDiagnosticsPacket = {
		schema: RESEARCH_DIAGNOSTICS_PACKET_SCHEMA,
		mode,
		goal: truncateText(input.goal, MAX_GOAL_CHARS),
		scope_status: scopeStatus(input.focusa_scope),
		focusa_scope: input.focusa_scope,
		target_refs: targetRefs,
		evidence_refs: evidenceRefs,
		captures,
		diagnostics_summary: diagnosticsSummaryFromResponses(input.responses),
		active_object_hints: targetRefs.slice(0, MAX_ACTIVE_OBJECT_HINTS).map((hint) => ({ kind: hint.startsWith("browser:") ? "url" : "source", hint })),
		recommended_focusa: {
			preferred_tool: preferredTool,
			fallback_tool: "focusa_evidence_capture",
			args_preview: sanitizeArgsPreview({
				target_ref: primaryCapture?.target_ref || targetRefs[0] || "uiai:packet",
				result: primaryCapture?.summary || truncateText(input.goal, MAX_CAPTURE_SUMMARY_CHARS),
				evidence_ref: primaryCapture?.evidence_ref || evidenceRefs[0] || "uiai-packet:manual",
				attach_to_workpoint: false,
			}),
			next_tools: preferredTool === "focusa_browser_diagnostics_intake" ? ["focusa_browser_diagnostics_intake", "focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"] : ["focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"],
		},
		recommended_next_action: recommendedNext,
		render: {
			summary_line: `UIAI packet evidence=${evidenceRefs.length} target=${targetRefs[0] || "none"} scope=${scopeStatus(input.focusa_scope)} tool=${preferredTool} next=${recommendedNext}`,
			expandable_json_ref: truncateText(input.expandable_json_ref || "", 240) || undefined,
		},
		headless_next_action: recommendedNext,
		cleanup: input.cleanup_session_id ? { session_id: truncateText(input.cleanup_session_id, 120), recommended_action: "close_when_done", tool: "uiai_browser_close" } : undefined,
	};
	while (packetByteSize(packet) > MAX_PACKET_BYTES) {
		if (packet.diagnostics_summary?.top_findings?.length) packet.diagnostics_summary.top_findings.pop();
		else if (packet.active_object_hints?.length) packet.active_object_hints.pop();
		else if (packet.captures.length) packet.captures.pop();
		else if (packet.evidence_refs.length) packet.evidence_refs.pop();
		else if (packet.target_refs.length) packet.target_refs.pop();
		else { delete packet.recommended_focusa.args_preview; break; }
	}
	return packet;
}

function renderUiaiWidget(ctx: any) {
	ctx.ui.setWidget("uiai-engine", [
		"UIAI Engine",
		`Base: ${engineUrl()}`,
		"Tools: agent_card/search/graph + full browser session, screenshot, frame catalog/render",
	]);
}

function latestWidgetVisibility(ctx: any): boolean | undefined {
	const entries = ctx.sessionManager?.getEntries?.() || [];
	for (let i = entries.length - 1; i >= 0; i--) {
		const entry = entries[i];
		if (entry?.type === "custom" && entry?.customType === UIAI_WIDGET_STATE_ENTRY && typeof entry?.data?.visible === "boolean") {
			return entry.data.visible;
		}
	}
	return undefined;
}

function compactSummary(data: any, details: Record<string, any> = {}) {
	const endpoint = details.endpoint ? `${details.endpoint}` : "UIAI";
	if (data?.error || data?.error_class) {
		const id = data.error_id ? ` id=${data.error_id}` : "";
		const next = data.suggested_next_action ? ` → ${data.suggested_next_action}` : "";
		return `${endpoint} error${data.error_class ? `:${data.error_class}` : ""}${id} ${data.message || data.error || ""}${next}`.trim();
	}
	if (data?.schema === RESEARCH_DIAGNOSTICS_PACKET_SCHEMA) {
		return data?.render?.summary_line || `UIAI packet evidence=${data?.evidence_refs?.length ?? 0}`;
	}
	if (Array.isArray(data?.events)) {
		return `${endpoint} ${data.count ?? data.events.length} error events stored=${data.stored_count ?? "?"}`;
	}
	if (data?.session_id || data?.id) {
		const id = data.session_id || data.id;
		const url = data.url ? ` ${data.url}` : "";
		return `${endpoint} ${id}${url}`;
	}
	if (data?.summary) {
		return `${endpoint} summary ${JSON.stringify(data.summary)}`;
	}
	if (typeof data?.count === "number") return `${endpoint} count=${data.count}`;
	if (data?.status) return `${endpoint} status=${data.status}`;
	return `${endpoint} ok`;
}

function compactRenderResult(result: any, { expanded, isPartial }: { expanded?: boolean; isPartial?: boolean }, theme: any) {
	if (isPartial) return new Text(theme.fg("warning", "UIAI running…"), 0, 0);
	const textContent = result?.content?.find?.((c: any) => c.type === "text");
	const raw = textContent?.type === "text" ? textContent.text : "";
	let data: any = raw;
	try { data = raw ? JSON.parse(raw) : {}; } catch { /* keep raw */ }
	const details = result?.details || {};
	const isError = Boolean(data?.error || data?.error_class || result?.isError);
	let line = isError ? theme.fg("error", compactSummary(data, details)) : theme.fg("success", compactSummary(data, details));
	if (!expanded) {
		line += theme.fg("muted", ` (${keyHint("app.tools.expand", "expand")})`);
		return new Text(line, 0, 0);
	}
	const body = typeof data === "string" ? data : JSON.stringify(data, null, 2);
	return new Text(`${line}\n${theme.fg("toolOutput", body)}`, 0, 0);
}

export default function uiaiEngineExtension(pi: ExtensionAPI) {
	pi.on("session_start", async (_event, ctx) => {
		if (latestWidgetVisibility(ctx) === true) renderUiaiWidget(ctx);
	});

	const registerTool = pi.registerTool.bind(pi);
	pi.registerTool = ((definition: any) => registerTool({
		...definition,
		renderResult: definition.renderResult || compactRenderResult,
	})) as typeof pi.registerTool;

	pi.registerTool({
		name: "pi_uiai_agent_card",
		label: "UIAI Agent Card",
		description: "Read compact UIAI Engine bootstrap guidance: discovery endpoints, workflows, health, MCP/OpenAI schema links, and diagnostics rules.",
		promptSnippet: "Use pi_uiai_agent_card before loading full UIAI schemas or when orienting browser/visual QA workflows.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/tools/agent-card"), { endpoint: "/api/tools/agent-card" });
		},
	});

	pi.registerTool({
		name: "pi_uiai_tool_search",
		label: "UIAI Tool Search",
		description: "Search UIAI tools by keyword without loading all schemas. Try diagnostics, screenshot, snapshot, click, fill, eval_async, console, network, visual failure.",
		parameters: Type.Object({ q: Type.String({ description: "Keyword or phrase to search" }) }),
		async execute(_toolCallId, params) {
			const q = encodeURIComponent(params.q);
			return textResult(await callEngine(`/api/tools/search?q=${q}`), { endpoint: "/api/tools/search", query: params.q });
		},
	});

	pi.registerTool({
		name: "pi_uiai_tool_graph",
		label: "UIAI Tool Graph",
		description: "Read UIAI tool relationship graph with workflow routes and Focusa integration metadata.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/tools/graph"), { endpoint: "/api/tools/graph" });
		},
	});

	pi.registerTool({
		name: "uiai_focusa_packet_build",
		label: "UIAI Focusa Packet Build",
		description: "Compose a bounded uiai.focusa_research_diagnostics_packet.v1 locally from existing UIAI search/source_markdown/read/diagnostics/error responses. Does not call Focusa or create durable memory.",
		parameters: Type.Object({
			goal: Type.String({ description: "Bounded user-visible goal for the packet" }),
			mode: Type.Optional(Type.String({ description: "research, diagnose, or proof", default: "research" })),
			responses: Type.Array(Type.Any({ description: "Existing UIAI responses containing focusa metadata" })),
			focusa_scope: Type.Optional(Type.Any({ description: "Optional Focusa scope to echo into the packet" })),
			recommended_next_action: Type.Optional(Type.String({ description: "Exact next browser/source/proof step" })),
			cleanup_session_id: Type.Optional(Type.String({ description: "Browser session id to recommend closing when done" })),
		}),
		async execute(_toolCallId, params) {
			return textResult(buildResearchDiagnosticsPacket(params), { endpoint: "pi:uiai_focusa_packet_build" });
		},
	});

	pi.registerTool({
		name: "uiai_focusa_packet_compose",
		label: "UIAI Focusa Packet Compose",
		description: "Compose a bounded uiai.focusa_research_diagnostics_packet.v1 through POST /api/agent/research-packet from existing UIAI responses. Use for HTTP/MCP/CLI parity with the engine composer.",
		parameters: Type.Object({
			goal: Type.String({ description: "Bounded user-visible goal for the packet" }),
			mode: Type.Optional(Type.String({ description: "research, diagnose, or proof", default: "research" })),
			responses: Type.Array(Type.Any({ description: "Existing UIAI responses containing focusa/focusa_evidence metadata" })),
			focusa_scope: Type.Optional(Type.Any({ description: "Optional Focusa scope to echo into the packet" })),
			recommended_next_action: Type.Optional(Type.String({ description: "Exact next browser/source/proof step" })),
			cleanup_session_id: Type.Optional(Type.String({ description: "Browser session id to recommend closing when done" })),
			expandable_json_ref: Type.Optional(Type.String({ description: "Optional external artifact/ref for larger JSON" })),
		}),
		async execute(_toolCallId, params) {
			return textResult(await post("/api/agent/research-packet", params), { endpoint: "/api/agent/research-packet" });
		},
	});

	pi.registerTool({
		name: "uiai_search",
		label: "UIAI Web Search",
		description: "Provider-neutral web search for browser agents. Returns result URLs/snippets; open selected URLs with uiai_browser_open, then uiai_browser_read.",
		parameters: Type.Object({
			query: Type.String({ description: "Search query" }),
			provider: Type.Optional(Type.String({ description: "Search provider id; default brave" })),
			limit: Type.Optional(Type.Number({ description: "Result limit, max 20", default: 5 })),
		}),
		async execute(_toolCallId, params) {
			return textResult(await post("/api/search", params), { endpoint: "/api/search" });
		},
	});

	pi.registerTool({
		name: "uiai_source_to_markdown",
		label: "UIAI Source to Markdown",
		description: "One-shot Source-to-Markdown conversion for public URLs. Returns uiai.source_markdown.v1 with Markdown, source metadata/adapters, optional JSONL records/chunks, diagnostics, Focusa-ready evidence, and auto-closes the temporary session.",
		parameters: Type.Object({
			url: Type.String({ description: "Public URL to convert" }),
			selector: Type.Optional(Type.String({ description: "Optional CSS selector region" })),
			max_chars: Type.Optional(Type.Number({ description: "Max Markdown characters, capped by engine", default: 30000 })),
			mode: Type.Optional(Type.String({ description: "Read mode: main_content or full", default: "main_content" })),
			format: Type.Optional(Type.String({ description: "Response format hint: json, markdown, or jsonl", default: "json" })),
			include_links: Type.Optional(Type.Boolean({ description: "Include visible link metadata", default: true })),
			include_images: Type.Optional(Type.Boolean({ description: "Include Markdown image tags", default: false })),
			focusa_scope: Type.Optional(Type.Any({ description: "Optional Focusa scope to echo into metadata" })),
		}),
		async execute(_toolCallId, params) {
			return textResult(await post("/api/markdown", params), { endpoint: "/api/markdown" });
		},
	});

	pi.registerTool({
		name: "uiai_health",
		label: "UIAI Browser Health",
		description: "Check UIAI browser health/readiness and pressure before remote or long browser workflows.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/health/browser"), { endpoint: "/api/health/browser" });
		},
	});

	pi.registerTool({
		name: "uiai_status",
		label: "UIAI Engine Status",
		description: "Read UIAI engine runtime status and service metadata.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/status"), { endpoint: "/api/status" });
		},
	});

	pi.registerTool({
		name: "uiai_critique_models",
		label: "UIAI Critique Models",
		description: "List supported critique models/providers. Read-only metadata; use before paid critique calls.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/critique/models"), { endpoint: "/api/critique/models" });
		},
	});

	pi.registerTool({
		name: "uiai_critique_dimensions",
		label: "UIAI Critique Dimensions",
		description: "List UI critique scoring dimensions. Read-only metadata for critique workflows.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/critique/dimensions"), { endpoint: "/api/critique/dimensions" });
		},
	});

	pi.registerTool({
		name: "uiai_errors",
		label: "UIAI Error Events",
		description: "Read bounded, redacted UIAI engine/browser error events after UIAI tool failures.",
		parameters: Type.Object({
			limit: Type.Optional(Type.Number({ description: "Recent event limit, max 500", default: 20 })),
			source: Type.Optional(Type.String({ description: "Optional source filter: http, panic, browser_session" })),
			class: Type.Optional(Type.String({ description: "Optional error class filter" })),
		}),
		async execute(_toolCallId, params) {
			const q = new URLSearchParams();
			if (params.limit !== undefined) q.set("limit", String(params.limit));
			if (params.source !== undefined) q.set("source", params.source);
			if (params.class !== undefined) q.set("class", params.class);
			return textResult(await callEngine(`/api/errors${q.toString() ? `?${q.toString()}` : ""}`), { endpoint: "/api/errors" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_open",
		label: "UIAI Browser Open",
		description: "Open a persistent UIAI browser session for public http/https targets. Private/internal targets require explicit local/dev allow_private_urls; prefer read/snapshot @refs and diagnostics-first debugging.",
		parameters: Type.Object({
			url: Type.String({ description: "URL to open" }),
			width: Type.Optional(Type.Number({ description: "Viewport width", default: 1280 })),
			height: Type.Optional(Type.Number({ description: "Viewport height", default: 800 })),
			focusa_scope: Type.Optional(Type.Any({ description: "Optional Focusa scope object echoed through diagnostics/evidence" })),
		}),
		async execute(_toolCallId, params) {
			const data = await post("/api/session", params);
			return textResult(withoutScreenshot(data), { endpoint: "/api/session", has_screenshot: Boolean(data.screenshot) });
		},
	});

	pi.registerTool({
		name: "uiai_browser_screenshot",
		label: "UIAI Browser Screenshot",
		description: "Capture the current session view without navigation. Use diagnostics if the page is blank or broken.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			format: Type.Optional(Type.String({ description: "Image format, jpeg or png" })),
			quality: Type.Optional(Type.Number({ description: "JPEG quality 1-100" })),
			fullPage: Type.Optional(Type.Boolean({ description: "Capture entire page" })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			const data = await post(`/api/session/${session_id}/screenshot`, body);
			return textResult(withoutScreenshot(data), { endpoint: "/api/session/{id}/screenshot", has_screenshot: Boolean(data.screenshot) });
		},
	});

	pi.registerTool({
		name: "uiai_browser_scroll",
		label: "UIAI Browser Scroll",
		description: "Scroll a session by delta or absolute position and return page state.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			deltaX: Type.Optional(Type.Number({ description: "Horizontal scroll pixels" })),
			deltaY: Type.Optional(Type.Number({ description: "Vertical scroll pixels", default: 600 })),
			x: Type.Optional(Type.Number({ description: "Absolute x position" })),
			y: Type.Optional(Type.Number({ description: "Absolute y position" })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			return textResult(withoutScreenshot(await post(`/api/session/${session_id}/scroll`, body)), { endpoint: "/api/session/{id}/scroll" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_snapshot",
		label: "UIAI Browser Snapshot",
		description: "Get accessibility tree with @ref selectors for reliable click/fill/press actions.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			interactive: Type.Optional(Type.Boolean({ description: "Only interactive elements" })),
			compact: Type.Optional(Type.Boolean({ description: "Compact empty structural nodes" })),
			max_depth: Type.Optional(Type.Number({ description: "Max tree depth" })),
			selector: Type.Optional(Type.String({ description: "Scope CSS selector" })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			return textResult(await post(`/api/session/${session_id}/snapshot`, body), { endpoint: "/api/session/{id}/snapshot" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_dom",
		label: "UIAI Browser DOM",
		description: "Get legacy structured DOM summary. Prefer uiai_browser_snapshot for @ref actions.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }) }),
		async execute(_toolCallId, params) {
			return textResult(await callEngine(`/api/session/${params.session_id}/dom`), { endpoint: "/api/session/{id}/dom" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_navigate",
		label: "UIAI Browser Navigate",
		description: "Navigate an existing UIAI browser session to a new URL. Use read/snapshot after navigation, diagnostics on failures.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), url: Type.String({ description: "Destination URL" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/navigate`, { url: params.url })), { endpoint: "/api/session/{id}/navigate" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_click",
		label: "UIAI Browser Click",
		description: "Click a CSS selector or @ref from uiai_browser_snapshot. Read diagnostics after unexpected UI or failed actions.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref, e.g. @e3" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/click`, { selector: params.selector })), { endpoint: "/api/session/{id}/click" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_hover",
		label: "UIAI Browser Hover",
		description: "Hover a CSS selector or @ref from uiai_browser_snapshot.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/hover`, { selector: params.selector })), { endpoint: "/api/session/{id}/hover" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_type",
		label: "UIAI Browser Type",
		description: "Type text into an input by selector or @ref. Use fill when replacing a value.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref" }), text: Type.String({ description: "Text to type" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/type`, { selector: params.selector, text: params.text })), { endpoint: "/api/session/{id}/type" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_fill",
		label: "UIAI Browser Fill",
		description: "Replace an input value using a CSS selector or @ref. Prefer this over type when setting form values.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref" }), text: Type.String({ description: "Text value to fill" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/fill`, { selector: params.selector, text: params.text })), { endpoint: "/api/session/{id}/fill" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_select",
		label: "UIAI Browser Select",
		description: "Select dropdown option values or visible text by selector or @ref.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref of select element" }), values: Type.Array(Type.String({ description: "Option value or text" })) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/select`, { selector: params.selector, values: params.values })), { endpoint: "/api/session/{id}/select" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_press",
		label: "UIAI Browser Press",
		description: "Press a keyboard key such as Enter, Tab, Escape, ArrowDown, or Backspace in the current session.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), key: Type.String({ description: "Keyboard key name" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/press`, { key: params.key })), { endpoint: "/api/session/{id}/press" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_back",
		label: "UIAI Browser Back",
		description: "Navigate browser history back in the current session.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/back`, {})), { endpoint: "/api/session/{id}/back" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_forward",
		label: "UIAI Browser Forward",
		description: "Navigate browser history forward in the current session.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/forward`, {})), { endpoint: "/api/session/{id}/forward" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_eval",
		label: "UIAI Browser Eval",
		description: "Execute short synchronous JavaScript on the page. Prefer direct actions for workflows; use diagnostics for errors.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), js: Type.String({ description: "Short JavaScript; use return for output" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/eval`, { js: params.js })), { endpoint: "/api/session/{id}/eval" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_eval_async",
		label: "UIAI Browser Eval Async",
		description: "Execute bounded async JavaScript with timeout_ms. Keep waits small; prefer direct actions for long workflows.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), js: Type.String({ description: "Async-capable JavaScript body" }), timeout_ms: Type.Optional(Type.Number({ description: "Timeout ms, max enforced by engine", default: 5000 })) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/eval_async`, { js: params.js, timeout_ms: params.timeout_ms })), { endpoint: "/api/session/{id}/eval_async" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_resize",
		label: "UIAI Browser Resize",
		description: "Resize viewport for responsive checks.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), width: Type.Number({ description: "Viewport width" }), height: Type.Number({ description: "Viewport height" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/resize`, { width: params.width, height: params.height })), { endpoint: "/api/session/{id}/resize" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_css",
		label: "UIAI Browser CSS",
		description: "Inject CSS into the active session to test visual changes live.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), css: Type.String({ description: "CSS rules to inject" }) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/css`, { css: params.css })), { endpoint: "/api/session/{id}/css" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_wait",
		label: "UIAI Browser Wait",
		description: "Wait for a selector before reading, snapshotting, clicking, or diagnosing a page.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector to wait for" }), timeout_ms: Type.Optional(Type.Number({ description: "Max wait time in ms", default: 5000 })) }),
		async execute(_toolCallId, params) {
			return textResult(withoutScreenshot(await post(`/api/session/${params.session_id}/wait`, { selector: params.selector, timeout_ms: params.timeout_ms })), { endpoint: "/api/session/{id}/wait" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_text",
		label: "UIAI Browser Text",
		description: "Get text content of one selector or @ref. Prefer read for broader page text.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }), selector: Type.String({ description: "CSS selector or @ref" }) }),
		async execute(_toolCallId, params) {
			return textResult(await post(`/api/session/${params.session_id}/text`, { selector: params.selector }), { endpoint: "/api/session/{id}/text" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_read",
		label: "UIAI Browser Read",
		description: "Read compact page or region text/Markdown for web surfing without taking a screenshot. Use after open/navigate when content matters more than pixels.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			selector: Type.Optional(Type.String({ description: "Optional CSS selector or @ref region" })),
			max_chars: Type.Optional(Type.Number({ description: "Max text characters, capped by engine", default: 8000 })),
			include_links: Type.Optional(Type.Boolean({ description: "Include visible links" })),
			format: Type.Optional(Type.String({ description: "Output format: text or markdown", default: "text" })),
			mode: Type.Optional(Type.String({ description: "Read mode: main_content or full", default: "main_content" })),
			include_images: Type.Optional(Type.Boolean({ description: "Include Markdown image tags when format=markdown", default: false })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			return textResult(await post(`/api/session/${session_id}/read`, body), { endpoint: "/api/session/{id}/read" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_cookies",
		label: "UIAI Browser Cookies",
		description: "Get, set, or clear session cookies.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			action: Type.Optional(Type.String({ description: "get, set, or clear" })),
			name: Type.Optional(Type.String({ description: "Cookie name" })),
			value: Type.Optional(Type.String({ description: "Cookie value for set" })),
			domain: Type.Optional(Type.String({ description: "Cookie domain" })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			return textResult(await post(`/api/session/${session_id}/cookies`, body), { endpoint: "/api/session/{id}/cookies" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_diagnostics",
		label: "UIAI Browser Diagnostics",
		description: "Read console errors, JS exceptions, failed requests, CORS/API clues, and visual failure diagnostics without taking a screenshot.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			limit: Type.Optional(Type.Number({ description: "Max events per category", default: 100 })),
			level: Type.Optional(Type.String({ description: "all, error, warning, info" })),
			failed_only: Type.Optional(Type.Boolean({ description: "Only failed network events" })),
		}),
		async execute(_toolCallId, params) {
			const q = new URLSearchParams();
			if (params.limit !== undefined) q.set("limit", String(params.limit));
			if (params.level !== undefined) q.set("level", params.level);
			if (params.failed_only !== undefined) q.set("failed_only", params.failed_only ? "true" : "false");
			return textResult(await callEngine(`/api/session/${params.session_id}/diagnostics${q.toString() ? `?${q.toString()}` : ""}`), { endpoint: "/api/session/{id}/diagnostics" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_diagnostics_clear",
		label: "UIAI Browser Diagnostics Clear",
		description: "Clear diagnostic buffers for a browser session.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }) }),
		async execute(_toolCallId, params) {
			return textResult(await post(`/api/session/${params.session_id}/diagnostics/clear`, {}), { endpoint: "/api/session/{id}/diagnostics/clear" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_close",
		label: "UIAI Browser Close",
		description: "Close a UIAI browser session and free its page.",
		parameters: Type.Object({ session_id: Type.String({ description: "UIAI browser session id" }) }),
		async execute(_toolCallId, params) {
			return textResult(await callEngine(`/api/session/${params.session_id}`, { method: "DELETE" }), { endpoint: "/api/session/{id}" });
		},
	});

	pi.registerTool({
		name: "uiai_screenshot",
		label: "UIAI Screenshot",
		description: "One-shot screenshot for public http/https targets: navigate, capture, forget. Private/internal targets require explicit local/dev allow_private_urls; use sessions for multi-step browsing.",
		parameters: Type.Object({
			url: Type.String({ description: "URL to screenshot" }),
			width: Type.Optional(Type.Number({ description: "Viewport width", default: 1280 })),
			height: Type.Optional(Type.Number({ description: "Viewport height", default: 800 })),
			format: Type.Optional(Type.String({ description: "Image format" })),
			quality: Type.Optional(Type.Number({ description: "JPEG quality" })),
			fullPage: Type.Optional(Type.Boolean({ description: "Full page capture" })),
		}),
		async execute(_toolCallId, params) {
			const data = await post("/api/screenshot", params);
			return textResult(withoutScreenshot(data), { endpoint: "/api/screenshot", has_screenshot: Boolean(data.screenshot) });
		},
	});

	pi.registerTool({
		name: "uiai_frame_catalog",
		label: "UIAI Frame Catalog",
		description: "List available device frames before rendering screenshots into device mockups.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/media/frame/catalog"), { endpoint: "/api/media/frame/catalog" });
		},
	});

	pi.registerTool({
		name: "uiai_frame_render",
		label: "UIAI Frame Render",
		description: "Render a base64 screenshot into a selected device frame. Use uiai_frame_catalog first.",
		parameters: Type.Object({
			frameId: Type.String({ description: "Frame ID from catalog" }),
			imageBase64: Type.String({ description: "Source screenshot base64" }),
			fit: Type.Optional(Type.String({ description: "cover or contain" })),
			format: Type.Optional(Type.String({ description: "png or jpeg" })),
			quality: Type.Optional(Type.Number({ description: "JPEG quality" })),
			scale: Type.Optional(Type.Number({ description: "Output scale" })),
		}),
		async execute(_toolCallId, params) {
			return textResult(await post("/api/media/frame/render", params), { endpoint: "/api/media/frame/render" });
		},
	});

	pi.registerCommand("uiai", {
		description: "Show UIAI Engine agent menu and bootstrap widget; use /uiai off to hide the widget",
		handler: async (args, ctx) => {
			const rawAction = String(args || "").trim();
			const executable = rawAction.match(/^(research|proof|diagnose)\s+(.+)$/i);
			if (executable) {
				renderUiaiWidget(ctx);
				pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: true });
				await runGuidedPacketWorkflow(executable[1].toLowerCase() as PacketMode, executable[2], ctx);
				return;
			}
			const action = rawAction.toLowerCase();
			if (["off", "hide", "clear", "disable"].includes(action)) {
				ctx.ui.setWidget("uiai-engine", undefined);
				pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: false });
				ctx.ui.notify("UIAI card hidden", "info");
				return;
			}
			if (["on", "show", "enable"].includes(action)) {
				renderUiaiWidget(ctx);
				pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: true });
				ctx.ui.notify("UIAI card shown", "info");
				return;
			}
			const guidedPrompts: Record<string, string> = {
				research: "Run /uiai research <query> to search, open the top result, read/snapshot/diagnose, compose a packet, close the session, and insert Focusa args_preview; or manually call uiai_search → open → read → diagnostics → uiai_focusa_packet_compose.",
				diagnose: "Run /uiai diagnose <session_id> to read browser diagnostics, compose mode=diagnose, and insert Focusa args_preview; or manually call uiai_browser_diagnostics then uiai_focusa_packet_compose.",
				proof: "Run /uiai proof <url> with a public http/https URL to open/read/snapshot/diagnose, compose mode=proof, close the session, and insert Focusa args_preview; private/internal targets need explicit local/dev allow_private_urls. Or manually call uiai_focusa_packet_compose with cleanup_session_id.",
			};
			if (guidedPrompts[action]) {
				renderUiaiWidget(ctx);
				pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: true });
				ctx.ui.setEditorText(guidedPrompts[action]);
				ctx.ui.notify(`UIAI ${action} workflow prompt inserted`, "info");
				return;
			}
			try {
				const card = await callEngine("/api/tools/agent-card");
				ctx.ui.notify(`UIAI ready: ${card.purpose || "agent card loaded"}`, "info");
				renderUiaiWidget(ctx);
				pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: true });
				const choice = await ctx.ui.select("UIAI action", [
					"Hide UIAI card",
					"Show agent card",
					"Search tools",
					"Open browser session",
					"Run research packet",
					"Run diagnostics packet",
					"Run proof packet",
					"Run diagnostics",
					"One-shot screenshot",
					"Show tool graph",
				]);
				if (choice === "Hide UIAI card") {
					ctx.ui.setWidget("uiai-engine", undefined);
					pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: false });
					ctx.ui.notify("UIAI card hidden", "info");
					return;
				}
				const prompts: Record<string, string> = {
					"Show agent card": "Use pi_uiai_agent_card to show the UIAI Engine bootstrap card.",
					"Search tools": "Use pi_uiai_tool_search with q=diagnostics, read, click, screenshot, or frame.",
					"Open browser session": "Use uiai_browser_open with a public http/https URL, then uiai_browser_read, uiai_browser_snapshot, and action tools. Private/internal targets need an explicit local/dev allow_private_urls profile.",
					"Run research packet": guidedPrompts.research,
					"Run diagnostics packet": guidedPrompts.diagnose,
					"Run proof packet": guidedPrompts.proof,
					"Run diagnostics": "Use uiai_browser_diagnostics with a session_id after any failed or surprising browser action.",
					"One-shot screenshot": "Use uiai_screenshot with a public http/https URL for a one-off capture; private/internal targets need explicit local/dev allow_private_urls.",
					"Show tool graph": "Use pi_uiai_tool_graph to inspect UIAI workflows and Focusa handoff routes.",
				};
				if (choice && prompts[choice]) ctx.ui.setEditorText(prompts[choice]);
			} catch (err) {
				ctx.ui.notify(`UIAI unavailable: ${err instanceof Error ? err.message : String(err)}`, "warning");
			}
		},
	});
}
