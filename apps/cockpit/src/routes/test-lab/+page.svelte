<script lang="ts">
  import "$lib/ui/screen.css";
  import { engineClient, type BrowserHealth, type EngineHealth } from "$lib/engine-client";

  type Check = { id: string; name: string; detail: string; state: "idle" | "running" | "passed" | "failed"; result?: string };
  let checks: Check[] = [
    { id: "engine", name: "Engine health", detail: "GET /health", state: "idle" },
    { id: "browser", name: "Browser readiness", detail: "GET /api/health/browser", state: "idle" },
    { id: "sessions", name: "Session inventory", detail: "GET /api/session/", state: "idle" },
  ];
  let running = false;

  async function runChecks() {
    running = true;
    checks = checks.map((check) => ({ ...check, state: "running", result: undefined }));
    const results = new Map<string, { state: "passed" | "failed"; result: string }>();
    try { const result: EngineHealth = await engineClient.health(); results.set("engine", { state: result.status === "healthy" ? "passed" : "failed", result: result.status }); } catch (error) { results.set("engine", { state: "failed", result: error instanceof Error ? error.message : "Request failed" }); }
    try { const result: BrowserHealth = await engineClient.browserHealth(); results.set("browser", { state: result.browser_alive ? "passed" : "failed", result: String(result.browser_state || result.status || "not ready") }); } catch (error) { results.set("browser", { state: "failed", result: error instanceof Error ? error.message : "Request failed" }); }
    try { const result = await engineClient.sessions(); results.set("sessions", { state: "passed", result: `${result.length} session${result.length === 1 ? "" : "s"} returned` }); } catch (error) { results.set("sessions", { state: "failed", result: error instanceof Error ? error.message : "Request failed" }); }
    checks = checks.map((check) => ({ ...check, ...(results.get(check.id) || { state: "failed", result: "No result" }) }));
    running = false;
  }
</script>

<svelte:head><title>Test Lab · UIAI Engine Cockpit</title></svelte:head>
<div class="screen"><div class="screen-header"><div><p class="screen-kicker">Verify</p><h1>Test Lab</h1><p class="screen-lede">Run checks against the connected engine. Results come from real API calls.</p></div><button class="screen-button primary" type="button" onclick={runChecks} disabled={running}>{running ? "Running checks…" : "Run health checks"}</button></div><section class="screen-card" style="margin-bottom: 18px"><div class="form-section"><h2>Engine verification</h2><p>No simulated results. Every check below calls the local UIAI Engine and records the response.</p></div><div class="data-list">{#each checks as check}<div class="data-row"><span class="activity-icon" data-tone={check.state === "passed" ? "green" : "blue"}>{check.state === "passed" ? "✓" : check.state === "failed" ? "!" : "⌁"}</span><div class="data-row-main"><strong>{check.name}</strong><span>{check.detail}{check.result ? ` · ${check.result}` : ""}</span></div><span class:success={check.state === "passed"} class:warn={check.state === "failed"} class="badge">{check.state}</span></div>{/each}</div></section><section class="screen-grid"><article class="screen-card screen-stat"><strong>{checks.filter((check) => check.state === "passed").length}/{checks.length}</strong><span>checks passed from the engine</span></article><article class="screen-card screen-stat"><strong>{checks.filter((check) => check.state === "failed").length}</strong><span>real request failures</span></article><article class="screen-card screen-stat"><strong>API</strong><span>source of truth</span></article></section></div>
