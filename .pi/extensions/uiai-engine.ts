import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { keyHint } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import { Type } from "typebox";

const DEFAULT_ENGINE_URL = "http://localhost:7456";
const REQUEST_TIMEOUT_MS = Number(process.env.UIAI_PI_TIMEOUT_MS || 30000);
const UIAI_WIDGET_STATE_ENTRY = "uiai-widget-visibility";

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
		description: "Open a persistent UIAI browser session. Prefer read/snapshot @refs and diagnostics-first debugging for reliable web surfing.",
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
		description: "Read compact page or region text for web surfing without taking a screenshot. Use after open/navigate when content matters more than pixels.",
		parameters: Type.Object({
			session_id: Type.String({ description: "UIAI browser session id" }),
			selector: Type.Optional(Type.String({ description: "Optional CSS selector or @ref region" })),
			max_chars: Type.Optional(Type.Number({ description: "Max text characters, capped by engine", default: 8000 })),
			include_links: Type.Optional(Type.Boolean({ description: "Include visible links" })),
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
		description: "One-shot screenshot: navigate, capture, forget. Use sessions for multi-step browsing.",
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
			const action = String(args || "").trim().toLowerCase();
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
					"Open browser session": "Use uiai_browser_open with a URL, then uiai_browser_read, uiai_browser_snapshot, and action tools.",
					"Run diagnostics": "Use uiai_browser_diagnostics with a session_id after any failed or surprising browser action.",
					"One-shot screenshot": "Use uiai_screenshot with a URL for a one-off capture.",
					"Show tool graph": "Use pi_uiai_tool_graph to inspect UIAI workflows and Focusa handoff routes.",
				};
				if (choice && prompts[choice]) ctx.ui.setEditorText(prompts[choice]);
			} catch (err) {
				ctx.ui.notify(`UIAI unavailable: ${err instanceof Error ? err.message : String(err)}`, "warning");
			}
		},
	});
}
