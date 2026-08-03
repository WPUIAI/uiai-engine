<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { DEFAULT_ENGINE_URL, engineClient } from "$lib/engine-client";
  import { discoverFocusaDaemons, focusaDaemonSummary, readSavedFocusaDaemonHints, saveFocusaDaemonHints } from "$lib/focusa-daemon-discovery";

  let saved = false;
  let testing = false;
  let connectionResult = "";
  let engineUrl = DEFAULT_ENGINE_URL;
  let focusaDaemonHints = "";
  let projectRoot = "";
  let continuityId = "";
  let workpointId = "";
  let responseProfile = "agent_compact";
  let liveUpdates = true;
  let confirmMutations = true;
  let reducedMotion = false;

  function saveSettings() {
    window.localStorage.setItem("uiai.engine.url", engineUrl.trim() || DEFAULT_ENGINE_URL);
    saveFocusaDaemonHints(focusaDaemonHints.split(/[\n,]/).map((value) => value.trim()).filter(Boolean));
    window.localStorage.setItem("uiai.scope.project_root", projectRoot.trim());
    window.localStorage.setItem("uiai.scope.continuity_id", continuityId.trim());
    window.localStorage.setItem("uiai.scope.workpoint_id", workpointId.trim());
    window.localStorage.setItem("uiai.response.profile", responseProfile);
    window.localStorage.setItem("uiai.live-updates", String(liveUpdates));
    window.localStorage.setItem("uiai.confirm-mutations", String(confirmMutations));
    window.localStorage.setItem("uiai.reduced-motion", String(reducedMotion));
    saved = true;
    window.setTimeout(() => (saved = false), 2200);
  }

  async function testConnection() {
    saveSettings();
    testing = true;
    connectionResult = "";
    try { const [result, daemons] = await Promise.all([engineClient.health(), discoverFocusaDaemons()]); connectionResult = `${result.status} · ${result.service || "UIAI Engine"} · ${focusaDaemonSummary(daemons)}`; } catch (cause) { connectionResult = cause instanceof Error ? cause.message : "Connection failed"; } finally { testing = false; }
  }

  onMount(() => {
    engineUrl = window.localStorage.getItem("uiai.engine.url") || DEFAULT_ENGINE_URL;
    focusaDaemonHints = readSavedFocusaDaemonHints().join("\n");
    projectRoot = window.localStorage.getItem("uiai.scope.project_root") || "";
    continuityId = window.localStorage.getItem("uiai.scope.continuity_id") || "";
    workpointId = window.localStorage.getItem("uiai.scope.workpoint_id") || "";
    responseProfile = window.localStorage.getItem("uiai.response.profile") || responseProfile;
    liveUpdates = window.localStorage.getItem("uiai.live-updates") !== "false";
    confirmMutations = window.localStorage.getItem("uiai.confirm-mutations") !== "false";
    reducedMotion = window.localStorage.getItem("uiai.reduced-motion") === "true";
  });
</script>

<svelte:head><title>Settings · UIAI Engine Cockpit</title></svelte:head>
<div class="screen">
  <div class="screen-header"><div><p class="screen-kicker">Configure</p><h1>Settings</h1><p class="screen-lede">Tune the connected engine and the amount of context this Cockpit keeps visible.</p></div><div class="screen-actions"><button class="screen-button" type="button" onclick={testConnection} disabled={testing}>{testing ? "Testing…" : "Test connection"}</button><button class="screen-button primary" type="button" onclick={saveSettings}>Save changes</button></div></div>
  {#if saved || connectionResult}<div class="selection-banner" style="margin: -14px 0 18px; padding: 11px 14px; border: 1px solid color-mix(in srgb, var(--color-success) 25%, var(--color-border)); border-radius: 9px; color: var(--color-success); background: color-mix(in srgb, var(--color-success) 7%, transparent); font-size: 12px">{saved ? "Settings saved locally." : "Connection result:"} {connectionResult}</div>{/if}
  <section class="screen-card form-grid"><div class="form-section"><h2>Engine connection</h2><p>The local engine remains the execution boundary for browser work.</p><label class="form-field">Engine URL<input bind:value={engineUrl} /></label><label class="form-field">Remote Focusa daemon hints<textarea bind:value={focusaDaemonHints} rows="3" placeholder="http://focusa-vps.tailnet.ts.net:8787"></textarea><span>Optional saved VPS/Tailscale URLs. Cockpit always probes loopback and Bonjour too; discovery never grants authority.</span></label><p class="scope-note">Scope is required for governed Workpoint continuity. Leave fields empty only for an explicitly local, unscoped session.</p><label class="form-field">Project root<input bind:value={projectRoot} placeholder="/path/to/project" /></label><label class="form-field">Continuity ID<input bind:value={continuityId} placeholder="Focusa continuity id" /></label><label class="form-field">Workpoint ID<input bind:value={workpointId} placeholder="Focusa Workpoint id" /></label><label class="form-field" style="margin-top: 16px">Response profile<select bind:value={responseProfile}><option value="agent_compact">Agent compact</option><option value="agent_standard">Agent standard</option><option value="evidence_grade">Evidence grade</option></select></label></div><div class="form-section"><h2>Workspace behavior</h2><div class="toggle-row"><div><strong>Live updates</strong><span>Keep run and engine status moving without manual refresh.</span></div><input type="checkbox" bind:checked={liveUpdates} aria-label="Live updates" /></div><div class="toggle-row"><div><strong>Confirm consequential actions</strong><span>Require an explicit confirmation before browser mutations.</span></div><input type="checkbox" bind:checked={confirmMutations} aria-label="Confirm consequential actions" /></div><div class="toggle-row"><div><strong>Reduce motion</strong><span>Respect a calmer presentation for longer work sessions.</span></div><input type="checkbox" bind:checked={reducedMotion} aria-label="Reduce motion" /></div></div></section>
</div>
