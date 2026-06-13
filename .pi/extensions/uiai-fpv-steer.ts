import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";

const DEFAULT_ENGINE_URL = "http://127.0.0.1:7456";
const POLL_MS = Number(process.env.UIAI_FPV_STEER_POLL_MS || 1200);

type WatchState = {
	token: string;
	lastAuditCount: number;
	timer?: ReturnType<typeof setInterval>;
	ctx?: ExtensionContext;
};

function engineUrl(): string {
	return (process.env.UIAI_ENGINE_URL || DEFAULT_ENGINE_URL).replace(/\/$/, "");
}

function auditText(event: any): string {
	return String(event?.message || event?.text || event?.note || "").trim();
}

async function fetchStatus(token: string): Promise<any> {
	const res = await fetch(`${engineUrl()}/m/${encodeURIComponent(token)}/status`, { cache: "no-store" });
	const text = await res.text();
	const data = text ? JSON.parse(text) : {};
	if (!res.ok) throw new Error(data?.error || `FPV status ${res.status}`);
	return data;
}

function steerPrompt(token: string, event: any, text: string): string {
	return [
		`FPV_OPERATOR_STEER token=${token}`,
		`action=${event?.action || "message"}`,
		`verbatim=${JSON.stringify(text)}`,
		"Acknowledge this FPV operator note immediately, then follow any steering instruction if safe.",
	].join("\n");
}

export default function uiaiFpvSteerExtension(pi: ExtensionAPI) {
	const watches = new Map<string, WatchState>();

	function stop(token: string): boolean {
		const state = watches.get(token);
		if (!state) return false;
		if (state.timer) clearInterval(state.timer);
		watches.delete(token);
		return true;
	}

	async function poll(state: WatchState) {
		const status = await fetchStatus(state.token);
		const audit = Array.isArray(status.audit) ? status.audit : [];
		if (state.lastAuditCount === 0) state.lastAuditCount = audit.length;
		const fresh = audit.slice(state.lastAuditCount);
		state.lastAuditCount = audit.length;
		for (const event of fresh) {
			const text = auditText(event);
			if (!text) continue;
			state.ctx?.ui.notify(`FPV note: ${text}`, "info");
			if (state.ctx?.isIdle()) {
				pi.sendUserMessage(steerPrompt(state.token, event, text));
			} else {
				pi.sendUserMessage(steerPrompt(state.token, event, text), { deliverAs: "steer" });
			}
		}
	}

	async function start(token: string, ctx: ExtensionContext) {
		if (!token) throw new Error("FPV token required");
		stop(token);
		const status = await fetchStatus(token);
		const audit = Array.isArray(status.audit) ? status.audit : [];
		const state: WatchState = { token, lastAuditCount: audit.length, ctx };
		state.timer = setInterval(() => {
			poll(state).catch((err) => ctx.ui.notify(`FPV watch ${token} failed: ${err.message}`, "warning"));
		}, POLL_MS);
		watches.set(token, state);
		ctx.ui.notify(`Watching FPV ${token} for operator notes`, "info");
	}

	pi.registerCommand("fpv-watch", {
		description: "Watch an FPV token for audited notes and inject them as steering prompts",
		handler: async (args, ctx) => {
			const token = args.trim();
			if (!token) {
				ctx.ui.notify("Usage: /fpv-watch <token>", "warning");
				return;
			}
			await start(token, ctx);
		},
	});

	pi.registerCommand("fpv-unwatch", {
		description: "Stop watching an FPV token, or all tokens with /fpv-unwatch all",
		handler: async (args, ctx) => {
			const token = args.trim();
			if (token === "all") {
				for (const key of [...watches.keys()]) stop(key);
				ctx.ui.notify("Stopped all FPV watches", "info");
				return;
			}
			if (!token || !stop(token)) ctx.ui.notify("No matching FPV watch", "warning");
			else ctx.ui.notify(`Stopped FPV watch ${token}`, "info");
		},
	});

	pi.registerCommand("fpv-watch-status", {
		description: "Show active FPV note watches",
		handler: async (_args, ctx) => {
			const tokens = [...watches.keys()];
			ctx.ui.notify(tokens.length ? `FPV watches: ${tokens.join(", ")}` : "No active FPV watches", "info");
		},
	});

	pi.on("session_end", async () => {
		for (const key of [...watches.keys()]) stop(key);
	});
}
