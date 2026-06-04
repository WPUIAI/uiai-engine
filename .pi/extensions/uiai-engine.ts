import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

const DEFAULT_ENGINE_URL = "http://localhost:7456";
const REQUEST_TIMEOUT_MS = Number(process.env.UIAI_PI_TIMEOUT_MS || 30000);

function engineUrl(): string {
	return (process.env.UIAI_ENGINE_URL || DEFAULT_ENGINE_URL).replace(/\/$/, "");
}

async function callEngine(path: string, init?: RequestInit): Promise<any> {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
	try {
		const res = await fetch(`${engineUrl()}${path}`, {
			...init,
			signal: controller.signal,
			headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
		});
		const text = await res.text();
		let data: any = text;
		try {
			data = text ? JSON.parse(text) : {};
		} catch {
			// Keep raw text for non-JSON responses.
		}
		if (!res.ok) {
			throw new Error(`UIAI ${res.status}: ${typeof data === "string" ? data : data?.error || data?.message || JSON.stringify(data)}`);
		}
		return data;
	} finally {
		clearTimeout(timeout);
	}
}

function textResult(data: any, details: Record<string, any> = {}) {
	return {
		content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }],
		details,
	};
}

export default function uiaiEngineExtension(pi: ExtensionAPI) {
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
		parameters: Type.Object({
			q: Type.String({ description: "Keyword or phrase to search" }),
		}),
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
		name: "uiai_health",
		label: "UIAI Browser Health",
		description: "Check UIAI browser health/readiness and pressure before remote or long browser workflows.",
		parameters: Type.Object({}),
		async execute() {
			return textResult(await callEngine("/api/health/browser"), { endpoint: "/api/health/browser" });
		},
	});

	pi.registerTool({
		name: "uiai_browser_open",
		label: "UIAI Browser Open",
		description: "Open a persistent UIAI browser session. Prefer snapshot @refs and diagnostics-first debugging for reliable web surfing.",
		parameters: Type.Object({
			url: Type.String({ description: "URL to open" }),
			width: Type.Optional(Type.Number({ description: "Viewport width", default: 1280 })),
			height: Type.Optional(Type.Number({ description: "Viewport height", default: 800 })),
			focusa_scope: Type.Optional(Type.Any({ description: "Optional Focusa scope object echoed through diagnostics/evidence" })),
		}),
		async execute(_toolCallId, params) {
			const data = await callEngine("/api/session", {
				method: "POST",
				body: JSON.stringify(params),
			});
			const summary = { ...data };
			delete summary.screenshot;
			return textResult(summary, { endpoint: "/api/session", has_screenshot: Boolean(data.screenshot) });
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
			return textResult(await callEngine(`/api/session/${session_id}/snapshot`, {
				method: "POST",
				body: JSON.stringify(body),
			}), { endpoint: "/api/session/{id}/snapshot" });
		},
	});


	pi.registerTool({
		name: "uiai_browser_read",
		label: "UIAI Browser Read",
		description: "Read compact page or region text for web surfing without taking a screenshot. Use after open/navigate when content matters more than pixels.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			selector: Type.Optional(Type.String({ description: "Optional CSS selector or @ref region" })),
			max_chars: Type.Optional(Type.Number({ description: "Max text characters, capped by engine", default: 8000 })),
			include_links: Type.Optional(Type.Boolean({ description: "Include visible links" })),
		}),
		async execute(_toolCallId, params) {
			const { session_id, ...body } = params;
			return textResult(await callEngine(`/api/session/${session_id}/read`, {
				method: "POST",
				body: JSON.stringify(body),
			}), { endpoint: "/api/session/{id}/read" });
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
		name: "uiai_browser_close",
		label: "UIAI Browser Close",
		description: "Close a UIAI browser session and free its page.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
		}),
		async execute(_toolCallId, params) {
			return textResult(await callEngine(`/api/session/${params.session_id}`, { method: "DELETE" }), { endpoint: "/api/session/{id}" });
		},
	});

	pi.registerCommand("uiai", {
		description: "Show UIAI Engine agent bootstrap card",
		handler: async (_args, ctx) => {
			try {
				const card = await callEngine("/api/tools/agent-card");
				ctx.ui.notify(`UIAI ready: ${card.purpose || "agent card loaded"}`, "info");
				ctx.ui.setWidget("uiai-engine", [
					"UIAI Engine",
					`Base: ${engineUrl()}`,
					"Use tools: pi_uiai_agent_card, pi_uiai_tool_graph, pi_uiai_tool_search, uiai_browser_open, uiai_browser_read, uiai_browser_diagnostics",
				]);
			} catch (err) {
				ctx.ui.notify(`UIAI unavailable: ${err instanceof Error ? err.message : String(err)}`, "warning");
			}
		},
	});
}
