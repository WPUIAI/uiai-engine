<script lang="ts">
  import { onMount } from "svelte";
  import "../lib/ui/design-tokens.css";
  import "../lib/ui/sidebar-primitives.css";
  import { engineClient, savedScope } from "$lib/engine-client";
  import { phase0Cards } from "$lib/cards/phase0-card-manifest";
  import { parseCockpitWorkpointResume, workpointResumeFromHost, type CockpitWorkpointResume } from "$lib/contracts/workpoint-resume";
  import { buildCommandIndex, filterCommandIndex } from "$lib/navigation/command-index";
  import ResumeWorkpointButton from "$lib/ui/sidebar/ResumeWorkpointButton.svelte";
  import { runCockpitUpdate, startAutomaticCockpitUpdate, type CockpitUpdateResult } from "$lib/updater";
  import { footerDestinations, sidebarGroups, workspaceForPath, workspaceManifest, type SidebarGroup } from "$lib/navigation/sidebar-manifest";
  import { applyWorkspaceDrop, createSidebarDndAdapter } from "$lib/navigation/sidebar-dnd";
  import { defaultSidebarPreferences, readSidebarPreferences, resetSidebarPreferences, saveSidebarPreferences, type SidebarMode, type SidebarPreferencesV1 } from "$lib/navigation/sidebar-preferences";

  let preferences: SidebarPreferencesV1 = defaultSidebarPreferences();
  let activeWorkspaceId = "overview";
  let commandOpen = false;
  let commandQuery = "";
  let activityOpen = false;
  let contextOpen = false;
  let workspaceMenuOpen = false;
  let constrainedViewport = false;
  let overlayOpen = true;
  let draggingWorkspaceId = "";
  let resizeStart: { x: number; width: number } | null = null;
  let engineStatus = "checking";
  let scope: ReturnType<typeof savedScope> = {};
  let update: CockpitUpdateResult = { phase: "checking", message: "Checking for signed updates…" };
  let resumeState: CockpitWorkpointResume | null = null;
  $: scopeConnected = Boolean(scope.project_root && scope.continuity_id);
  $: scopeText = scopeConnected ? `${scope.project_root?.split("/").pop() || "Project"} · ${scope.workpoint_id ? "Workpoint connected" : "Continuity connected"}` : "Scope not connected";

  $: activeWorkspace = workspaceManifest.find((item) => item.id === activeWorkspaceId) || workspaceManifest[0];
  $: activeNav = activeWorkspace?.label || "Overview";
  $: groupedWorkspaces = sidebarGroups.map((group) => ({ ...group, items: workspaceManifest.filter((item) => item.group === group.id && !preferences.hidden_workspace_ids.includes(item.id)).sort((a, b) => a.order - b.order) }));
  $: commands = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, resumeState);
  $: resumePresentationState = resumeState?.status !== "resumable" || commands.some((command) => command.kind === "resume") ? resumeState : null;
  $: filteredCommands = filterCommandIndex(commands, commandQuery);

  onMount(() => {
    preferences = readSidebarPreferences();
    scope = savedScope();
    resumeState = workpointResumeFromHost();
    const onResumeContract = (event: Event) => {
      try {
        resumeState = parseCockpitWorkpointResume((event as CustomEvent<unknown>).detail);
      } catch {
        resumeState = null;
      }
    };
    window.addEventListener("uiai:workpoint-resume", onResumeContract);
    const setActiveFromPath = () => { activeWorkspaceId = workspaceForPath(window.location.pathname)?.id || (window.location.pathname === "/settings" ? "settings" : "overview"); };
    setActiveFromPath();
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); commandOpen = true; }
      if (event.key === "Escape") { commandOpen = false; workspaceMenuOpen = false; contextOpen = false; activityOpen = false; if (constrainedViewport) overlayOpen = false; }
      if (event.key === "[" && !event.metaKey && !event.ctrlKey) { event.preventDefault(); setSidebarMode(preferences.mode === "expanded" ? "compact" : "expanded"); }
    };
    window.addEventListener("keydown", onKeyDown);
    const updateViewport = () => { constrainedViewport = window.innerWidth <= 780; if (!constrainedViewport) overlayOpen = true; };
    updateViewport();
    window.addEventListener("resize", updateViewport);
    const stop = startAutomaticCockpitUpdate((result) => (update = result));
    const refreshEngineStatus = () => void engineClient.health().then((result) => (engineStatus = result.status)).catch(() => (engineStatus = "unavailable"));
    refreshEngineStatus();
    const engineTimer = window.setInterval(refreshEngineStatus, 5000);
    return () => { window.removeEventListener("keydown", onKeyDown); window.removeEventListener("resize", updateViewport); window.removeEventListener("uiai:workpoint-resume", onResumeContract); window.clearInterval(engineTimer); stop(); };
  });

  function persist(next: SidebarPreferencesV1) { preferences = next; saveSidebarPreferences(next); }
  const sidebarDnd = createSidebarDndAdapter(async (target, source) => { persist(applyWorkspaceDrop(preferences, workspaceManifest, source, target)); draggingWorkspaceId = ""; });
  function beginWorkspaceDrag(id: string) { draggingWorkspaceId = id; sidebarDnd.beginDrag(id); }
  function beginWorkspacePointerDrag(id: string, event: PointerEvent) { if (event.pointerType === "touch") { event.preventDefault(); beginWorkspaceDrag(id); } }
  function pointerDropWorkspace(item: { id: string; group: SidebarGroup; order: number }, event: PointerEvent) { if (event.pointerType === "touch" && draggingWorkspaceId) { event.preventDefault(); dropWorkspace({ workspace_id: item.id, display_group: item.group, order: item.order }); } }
  function cancelWorkspaceDrag() { draggingWorkspaceId = ""; sidebarDnd.cancelDrag(); }
  function dropWorkspace(target: { workspace_id: string; display_group: SidebarGroup; order: number }) { void sidebarDnd.commitDrop(target).catch(() => { draggingWorkspaceId = ""; sidebarDnd.cancelDrag(); }); }
  function keyboardMoveWorkspace(id: string, direction: -1 | 1) {
    const current = workspaceManifest.find((item) => item.id === id);
    if (!current) return;
    const peers = workspaceManifest.filter((item) => item.group === current.group && !preferences.hidden_workspace_ids.includes(item.id)).sort((a, b) => a.order - b.order);
    const target = peers[peers.findIndex((item) => item.id === id) + direction];
    if (!target) return;
    sidebarDnd.beginDrag(id);
    dropWorkspace({ workspace_id: target.id, display_group: target.group, order: target.order });
  }
  function setSidebarMode(mode: SidebarMode) { persist({ ...preferences, mode }); workspaceMenuOpen = false; }
  function toggleGroup(group: SidebarGroup) { persist({ ...preferences, collapsed_groups: preferences.collapsed_groups.includes(group) ? preferences.collapsed_groups.filter((item) => item !== group) : [...preferences.collapsed_groups, group] }); }
  function setLayoutMode(layout_mode: SidebarPreferencesV1["layout_mode"]) { persist({ ...preferences, layout_mode }); }
  function resetLayout() { persist(resetSidebarPreferences()); workspaceMenuOpen = false; }
  function setAllGroups(collapsed: boolean) { persist({ ...preferences, collapsed_groups: collapsed ? sidebarGroups.map((group) => group.id) : [] }); }
  function showHiddenWorkspaces() { persist({ ...preferences, hidden_workspace_ids: [] }); }
  function hideActiveWorkspace() {
    if (activeWorkspaceId === "overview") return;
    persist({ ...preferences, hidden_workspace_ids: [...new Set([...preferences.hidden_workspace_ids, activeWorkspaceId])] });
    navigateTo("/");
  }
  function beginResize(event: PointerEvent) {
    if (preferences.mode !== "expanded") return;
    event.preventDefault();
    resizeStart = { x: event.clientX, width: preferences.width_px };
    window.addEventListener("pointermove", resizeSidebar);
    window.addEventListener("pointerup", endResize, { once: true });
  }
  function resizeSidebar(event: PointerEvent) {
    if (!resizeStart) return;
    preferences = { ...preferences, width_px: Math.min(320, Math.max(208, resizeStart.width + event.clientX - resizeStart.x)) };
  }
  function endResize() {
    if (!resizeStart) return;
    persist({ ...preferences, layout_mode: "custom" });
    resizeStart = null;
    window.removeEventListener("pointermove", resizeSidebar);
  }
  function navigateTo(href: string) { commandOpen = false; commandQuery = ""; workspaceMenuOpen = false; window.location.href = href; }
  function selectWorkspace(id: string) { activeWorkspaceId = id; if (constrainedViewport) overlayOpen = false; }
  function toggleOverlay() { overlayOpen = !overlayOpen; }
  function checkForUpdate() { void runCockpitUpdate({ install: true, reporter: (result) => (update = result) }); }
</script>

<div class="app-shell">
  {#if constrainedViewport && overlayOpen}<button class="sidebar-scrim" type="button" aria-label="Close sidebar" onclick={() => (overlayOpen = false)}></button>{/if}
  <aside class:compact={preferences.mode === "compact"} class:hidden={preferences.mode === "hidden"} class:overlay={constrainedViewport} class:overlay-open={overlayOpen} class="sidebar" aria-label="Primary navigation" style={`--sidebar-width: ${preferences.width_px}px`}>
    <div class="brand"><div class="brand-mark" aria-hidden="true">✦</div><div><strong>UIAI</strong><span>Engine Cockpit</span></div></div>
    {#if resumePresentationState}<ResumeWorkpointButton state={resumePresentationState} />{/if}
    <div class="sidebar-heading"><span>Workspaces</span><button class="workspace-menu-button sidebar-row" type="button" aria-label="Open workspace layout menu" aria-expanded={workspaceMenuOpen} onclick={() => (workspaceMenuOpen = !workspaceMenuOpen)}>•••</button></div>
    {#if workspaceMenuOpen}<div class="workspace-menu sidebar-popover" role="menu" aria-label="Workspace layout"><div class="menu-title">Layout · {preferences.layout_mode}</div><button role="menuitemradio" aria-checked={preferences.layout_mode === "recommended"} type="button" onclick={() => setLayoutMode("recommended")}>Recommended <span>Manifest order</span></button><button role="menuitemradio" aria-checked={preferences.layout_mode === "custom"} type="button" onclick={() => setLayoutMode("custom")}>Custom <span>Local preference</span></button><div class="menu-title">Sidebar</div><button role="menuitemradio" aria-checked={preferences.mode === "expanded"} type="button" onclick={() => setSidebarMode("expanded")}>Expanded <span>⌘[</span></button><button role="menuitemradio" aria-checked={preferences.mode === "compact"} type="button" onclick={() => setSidebarMode("compact")}>Compact <span>64px rail</span></button><button role="menuitemradio" aria-checked={preferences.mode === "hidden"} type="button" onclick={() => setSidebarMode("hidden")}>Hidden <span>⌘[ to restore</span></button><div class="menu-title">Visibility</div>{#if preferences.hidden_workspace_ids.length}<button role="menuitem" type="button" onclick={showHiddenWorkspaces}>Show hidden workspaces <span>{preferences.hidden_workspace_ids.length}</span></button>{/if}<button role="menuitem" type="button" disabled={activeWorkspaceId === "overview"} onclick={hideActiveWorkspace}>Hide current workspace <span>{activeWorkspaceId === "overview" ? "Overview fixed" : "Local only"}</span></button><div class="menu-title">Groups</div><button role="menuitem" type="button" onclick={() => setAllGroups(false)}>Expand all <span>⌄</span></button><button role="menuitem" type="button" onclick={() => setAllGroups(true)}>Collapse all <span>›</span></button><button class="menu-reset" role="menuitem" type="button" onclick={resetLayout}>Reset sidebar layout</button></div>{/if}
    <div class="workspace-groups">
      {#each groupedWorkspaces as group}
        <section class="workspace-group" aria-label={group.label}>
          <button class="group-heading sidebar-group-header" type="button" aria-expanded={!preferences.collapsed_groups.includes(group.id)} onclick={() => toggleGroup(group.id)}><span>{group.label}</span><span aria-hidden="true">{preferences.collapsed_groups.includes(group.id) ? "›" : "⌄"}</span></button>
          {#if !preferences.collapsed_groups.includes(group.id)}<nav aria-label={`${group.label} workspaces`}>{#each group.items as item}<a class:active={activeWorkspaceId === item.id} class:quiet={item.emphasis === "quiet"} class="nav-item sidebar-row" class:dragging={draggingWorkspaceId === item.id} data-selected={activeWorkspaceId === item.id} draggable="true" aria-keyshortcuts="Alt+ArrowUp Alt+ArrowDown" href={item.route} aria-current={activeWorkspaceId === item.id ? "page" : undefined} onclick={() => selectWorkspace(item.id)} onpointerdown={(event) => beginWorkspacePointerDrag(item.id, event)} onpointerup={(event) => pointerDropWorkspace(item, event)} ondragstart={() => beginWorkspaceDrag(item.id)} ondragover={(event) => event.preventDefault()} ondrop={() => dropWorkspace({ workspace_id: item.id, display_group: item.group, order: item.order })} ondragend={cancelWorkspaceDrag} onkeydown={(event) => { if (event.altKey && event.key === "ArrowUp") { event.preventDefault(); keyboardMoveWorkspace(item.id, -1); } if (event.altKey && event.key === "ArrowDown") { event.preventDefault(); keyboardMoveWorkspace(item.id, 1); } }}><span class="nav-icon sidebar-row__icon" aria-hidden="true">{item.icon}</span><span class="nav-label sidebar-row__label">{item.label}</span>{#if item.state !== "current"}<span class="nav-state sidebar-row__state">{item.state}</span>{/if}{#if activeWorkspaceId === item.id}<span class="active-dot sidebar-row__active-bar" aria-hidden="true"></span>{/if}</a>{/each}</nav>{/if}
        </section>
      {/each}
    </div>
    {#if preferences.mode === "expanded"}<button class="resize-handle sidebar-resize-handle" type="button" aria-label="Resize sidebar" onpointerdown={beginResize}></button>{/if}
    <div class="sidebar-bottom"><div class="connection"><span class:healthy={engineStatus === "healthy"} class="status-dot"></span><span>Engine {engineStatus}</span></div>{#each footerDestinations as item}<a class="settings-button" class:active={activeWorkspaceId === item.id} href={item.route} aria-current={activeWorkspaceId === item.id ? "page" : undefined} onclick={() => selectWorkspace(item.id)}><span aria-hidden="true">{item.icon}</span><span>{item.label}</span></a>{/each}</div>
  </aside>

  <main class="main-shell">
    <header class="topbar"><div class="breadcrumbs"><span>Workspace</span><span class="slash">/</span><strong>{activeNav}</strong></div><div class="top-actions">{#if constrainedViewport}<button class="sidebar-toggle" type="button" aria-label="Toggle sidebar" aria-expanded={overlayOpen} onclick={toggleOverlay}>☰</button>{/if}<button class="command-trigger" type="button" onclick={() => (commandOpen = true)}><span>Find or do</span><kbd>⌘K</kbd></button><button class="scope-pill" class:scope-missing={!scopeConnected} type="button" aria-expanded={contextOpen} onclick={() => (contextOpen = !contextOpen)} title={scopeConnected ? "Inspect current project scope" : "Connect a project and Workpoint scope"}><span class="status-dot"></span> {scopeText}</button>{#if contextOpen}<div class="context-panel" role="dialog" aria-label="Context Control"><p class="screen-kicker">Context Control</p>{#if scopeConnected}<strong>{scope.project_root}</strong><span>{scope.continuity_id}</span><span>{scope.workpoint_id || "No Workpoint connected"}</span>{:else}<strong>Scope is not connected</strong><span>Browser actions remain local and unscoped until a project root and continuity ID are configured.</span>{/if}<a href="/settings" onclick={() => (contextOpen = false)}>Open scope settings →</a></div>{/if}<button class="update-button" type="button" onclick={checkForUpdate} title={update.message}><span class="update-dot" data-phase={update.phase}></span>{update.phase === "current" ? "Up to date" : update.phase === "available" ? "Update ready" : "Updates"}</button><button class="avatar" type="button" aria-label="Operator profile is not connected" title="Operator profile is not connected">?</button></div></header>
    <section class="content"><slot /></section>
    <div class:expanded={activityOpen} class="activity-bar" aria-live="polite"><button class="activity-toggle" type="button" onclick={() => (activityOpen = !activityOpen)}><span class="activity-live-dot" data-phase={update.phase}></span><strong>Activity</strong><span>{activityOpen ? "Close" : update.phase === "available" ? "Update ready" : "View signals"}</span><span class="activity-chevron">{activityOpen ? "⌃" : "⌄"}</span></button>{#if activityOpen}<div class="activity-panel"><div><strong>Update state</strong><span>{update.message}</span></div><button class="text-action" type="button" onclick={checkForUpdate}>Check again</button></div>{/if}</div>
  </main>
</div>
{#if commandOpen}<div class="command-backdrop" role="presentation" onclick={(event) => event.target === event.currentTarget && (commandOpen = false)}><dialog open class="command-palette" aria-label="Command palette"><div class="command-heading"><span>Find or do</span><kbd>Esc</kbd></div><input bind:value={commandQuery} aria-label="Search commands" placeholder="Search workspace or capability…" />{#if filteredCommands.length}<div class="command-list">{#each filteredCommands as command}<button type="button" onclick={() => navigateTo(command.href)}><span>{command.label}</span><small>{command.hint}</small></button>{/each}</div>{:else}<p class="command-empty">No commands in the current scope.</p>{/if}</dialog></div>{/if}

<style>
  .app-shell { min-height: 100vh; display: flex; background: var(--color-bg); }
  .sidebar { position: relative; z-index: 10; width: var(--sidebar-width); min-width: var(--sidebar-width-min); max-width: var(--sidebar-width-max); flex: 0 0 var(--sidebar-width); display: flex; flex-direction: column; padding: 22px 12px 14px; background: var(--sidebar-surface); border-right: 1px solid var(--sidebar-divider); transition: width var(--sidebar-transition), flex-basis var(--sidebar-transition); }
  .sidebar.compact { width: var(--sidebar-width-compact); min-width: var(--sidebar-width-compact); flex-basis: var(--sidebar-width-compact); padding-inline: 8px; } .sidebar.hidden { display: none; }
  .brand { display: flex; align-items: center; gap: 10px; padding: 0 12px 24px; } .brand-mark { width: 30px; height: 30px; display: grid; place-items: center; color: white; background: linear-gradient(135deg, #6b7cff, #9a5cff); border-radius: 9px; box-shadow: 0 5px 14px rgba(112, 92, 255, .28); font-size: 17px; } .brand strong, .brand span { display: block; letter-spacing: -.01em; } .brand strong { font-size: 14px; line-height: 16px; } .brand span { color: var(--color-text-muted); font-size: 11px; line-height: 15px; }
  .sidebar-heading { display: flex; align-items: center; justify-content: space-between; padding: 0 8px 8px 12px; color: var(--color-text-muted); font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase; } .workspace-menu-button { border: 0; color: var(--color-text-muted); background: transparent; font: inherit; letter-spacing: .12em; cursor: pointer; } .workspace-menu-button:hover { color: var(--color-text); }
  .workspace-groups { min-height: 0; overflow: auto; } .workspace-group { margin-bottom: 8px; } .group-heading { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 5px 12px; border: 0; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 10px; font-weight: 700; letter-spacing: .08em; text-align: left; text-transform: uppercase; cursor: pointer; } .group-heading:hover { color: var(--color-text); }
  nav { display: grid; gap: 3px; } .nav-item, .settings-button { position: relative; width: 100%; min-height: var(--sidebar-row-height); display: flex; align-items: center; gap: 10px; padding: 8px 12px; border: 0; border-radius: 8px; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 13px; text-align: left; text-decoration: none; cursor: pointer; transition: color var(--motion-fast) ease-out, background var(--motion-fast) ease-out, transform var(--motion-fast) ease-out; } .nav-item:hover, .settings-button:hover { color: var(--color-text); background: var(--sidebar-hover-surface); } .nav-item:active, .settings-button:active, .update-button:active { transform: scale(.98); } .nav-item.active { color: var(--color-text); background: var(--sidebar-selected-surface); font-weight: 600; } .nav-item.quiet { font-size: 12px; } .nav-item.dragging { opacity: .58; box-shadow: var(--sidebar-drag-elevation); } .nav-icon { width: 18px; color: currentColor; text-align: center; font-size: 15px; } .nav-label { min-width: 0; } .nav-state { margin-left: auto; color: var(--color-text-muted); font-size: 9px; text-transform: uppercase; } .active-dot { width: 4px; height: 4px; margin-left: auto; border-radius: 50%; background: var(--color-accent); }
  .workspace-menu { position: absolute; top: 74px; right: 12px; z-index: 20; width: 235px; padding: 7px; border: 1px solid var(--color-border); border-radius: 9px; background: var(--color-surface-elevated); box-shadow: var(--shadow-popover); } .menu-title { padding: 7px 9px 4px; color: var(--color-text-muted); font-size: 10px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; } .workspace-menu button { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 8px 9px; border: 0; border-radius: 6px; color: var(--color-text); background: transparent; font: inherit; font-size: 12px; text-align: left; cursor: pointer; } .workspace-menu button:hover { background: color-mix(in srgb, var(--color-accent) 9%, transparent); } .workspace-menu button span { color: var(--color-text-muted); font-size: 10px; } .workspace-menu .menu-reset { margin-top: 6px; border-top: 1px solid var(--color-border); border-radius: 0; color: var(--color-accent); }
  .resize-handle { position: absolute; top: 0; right: -3px; bottom: 0; z-index: 2; width: 6px; border: 0; background: transparent; cursor: col-resize; } .resize-handle:hover, .resize-handle:focus-visible { background: var(--sidebar-resize-handle); outline: none; }
  .sidebar-bottom { margin-top: auto; display: grid; gap: 5px; padding: 14px 4px 0; border-top: 1px solid var(--color-border); } .connection { display: flex; align-items: center; gap: 8px; padding: 0 8px 7px; color: var(--color-text-muted); font-size: 11px; } .status-dot, .update-dot { width: 7px; height: 7px; display: inline-block; border-radius: 50%; background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 14%, transparent); } .connection .status-dot { background: var(--color-warn); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-warn) 14%, transparent); } .connection .status-dot.healthy { background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 14%, transparent); } .settings-button { padding: 7px 8px; }
  .sidebar.compact .brand { padding-inline: 7px; } .sidebar.compact .brand > div:last-child, .sidebar.compact .sidebar-heading > span, .sidebar.compact .group-heading span:first-child, .sidebar.compact .nav-label, .sidebar.compact .nav-state, .sidebar.compact .connection span:last-child, .sidebar.compact .settings-button span:last-child { display: none; } .sidebar.compact .sidebar-heading, .sidebar.compact .group-heading { justify-content: center; padding-inline: 0; } .sidebar.compact .workspace-menu-button { font-size: 12px; } .sidebar.compact .nav-item, .sidebar.compact .settings-button { justify-content: center; padding-inline: 7px; } .sidebar.compact .active-dot { position: absolute; right: 4px; }
  .main-shell { min-width: 0; flex: 1; } .topbar { height: 62px; display: flex; align-items: center; justify-content: space-between; padding: 0 32px; border-bottom: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-bg) 82%, transparent); backdrop-filter: blur(18px); } .breadcrumbs { display: flex; gap: 10px; align-items: center; color: var(--color-text-muted); font-size: 13px; } .breadcrumbs strong { color: var(--color-text); font-weight: 600; } .slash { color: var(--color-border); } .top-actions { position: relative; display: flex; align-items: center; gap: 10px; } .command-trigger { display: inline-flex; align-items: center; gap: 14px; height: 30px; padding: 0 9px 0 11px; border: 1px solid var(--color-border); border-radius: 7px; color: var(--color-text-muted); background: var(--color-surface); font: inherit; font-size: 11px; cursor: pointer; } .command-trigger:hover { color: var(--color-text); border-color: color-mix(in srgb, var(--color-accent) 45%, var(--color-border)); } kbd { padding: 2px 5px; border: 1px solid var(--color-border); border-radius: 4px; color: var(--color-text-muted); background: var(--color-surface-elevated); font: inherit; font-size: 10px; }
  .context-panel { position: absolute; top: 40px; right: 38px; z-index: 25; display: grid; gap: 6px; width: min(360px, calc(100vw - 32px)); padding: 14px; border: 1px solid var(--color-border); border-radius: var(--radius-card); color: var(--color-text-muted); background: var(--color-surface-elevated); box-shadow: var(--shadow-popover); font-size: 11px; } .context-panel strong { overflow: hidden; color: var(--color-text); text-overflow: ellipsis; white-space: nowrap; } .context-panel span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .context-panel a { margin-top: 5px; color: var(--color-accent); text-decoration: none; } .scope-pill, .update-button { display: inline-flex; align-items: center; gap: 7px; height: 30px; padding: 0 10px; border: 1px solid var(--color-border); border-radius: 7px; color: var(--color-text-muted); background: var(--color-surface); font: inherit; font-size: 11px; } .scope-pill { text-decoration: none; cursor: pointer; } .scope-missing { border-color: color-mix(in srgb, var(--color-warn) 35%, var(--color-border)); } .scope-missing .status-dot { background: var(--color-warn); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-warn) 14%, transparent); } .update-button { cursor: pointer; } .update-dot[data-phase="error"] { background: var(--color-error); } .update-dot[data-phase="available"], .update-dot[data-phase="downloading"], .update-dot[data-phase="installing"] { background: var(--color-accent); } .avatar { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 50%; color: #fff; background: #34343a; font-size: 10px; font-weight: 700; }
  .content { max-width: 1180px; margin: 0 auto; padding: 48px 52px 88px; } .activity-bar { position: fixed; right: 18px; bottom: 16px; z-index: 5; width: min(360px, calc(100vw - 36px)); border: 1px solid var(--color-border); border-radius: 10px; background: color-mix(in srgb, var(--color-surface-elevated) 88%, transparent); box-shadow: var(--shadow-popover); backdrop-filter: blur(16px); } .activity-toggle { display: flex; align-items: center; gap: 8px; width: 100%; padding: 10px 12px; border: 0; color: var(--color-text); background: transparent; font: inherit; font-size: 12px; text-align: left; cursor: pointer; } .activity-toggle span:nth-of-type(2) { margin-left: auto; color: var(--color-text-muted); font-size: 11px; } .activity-chevron { color: var(--color-text-muted); } .activity-live-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-success); } .activity-live-dot[data-phase="available"] { background: var(--color-accent); } .activity-live-dot[data-phase="error"] { background: var(--color-error); } .activity-panel { display: flex; align-items: center; gap: 12px; padding: 12px; border-top: 1px solid var(--color-border); color: var(--color-text-muted); font-size: 11px; } .activity-panel div { min-width: 0; flex: 1; } .activity-panel strong, .activity-panel span { display: block; } .activity-panel strong { color: var(--color-text); font-size: 12px; } .text-action { border: 0; color: var(--color-accent); background: none; font: inherit; font-size: 11px; cursor: pointer; white-space: nowrap; }
  .command-backdrop { position: fixed; inset: 0; z-index: 20; display: grid; place-items: start center; padding-top: 16vh; background: rgba(0, 0, 0, .2); backdrop-filter: blur(8px); } .command-palette { width: min(540px, calc(100vw - 32px)); overflow: hidden; border: 1px solid var(--color-border); border-radius: 12px; background: var(--color-surface-elevated); box-shadow: var(--shadow-overlay); } .command-heading { display: flex; align-items: center; justify-content: space-between; padding: 14px 16px 8px; color: var(--color-text-muted); font-size: 11px; } .command-palette > input { width: calc(100% - 32px); box-sizing: border-box; margin: 0 16px 10px; padding: 11px 0; border: 0; border-bottom: 1px solid var(--color-border); outline: 0; color: var(--color-text); background: transparent; font: inherit; font-size: 16px; } .command-list { display: grid; padding: 4px 8px 8px; } .command-list button { display: flex; align-items: center; justify-content: space-between; padding: 11px 9px; border: 0; border-radius: 7px; color: var(--color-text); background: transparent; font: inherit; font-size: 13px; text-align: left; cursor: pointer; } .command-list button:hover { background: color-mix(in srgb, var(--color-accent) 8%, transparent); } .command-list small { color: var(--color-text-muted); font-size: 11px; } .command-empty { margin: 12px 16px 20px; color: var(--color-text-muted); font-size: 13px; }
  .sidebar-toggle { display: none; width: 30px; height: 30px; border: 1px solid var(--color-border); border-radius: var(--radius-button); color: var(--color-text); background: var(--color-surface); cursor: pointer; }
  .sidebar-scrim { position: fixed; inset: 0; z-index: 9; border: 0; background: rgba(0, 0, 0, .22); backdrop-filter: blur(2px); }
  @media (prefers-reduced-motion: reduce) { .sidebar { transition: none; } } @media (max-width: 780px) { .sidebar-toggle { display: grid; place-items: center; } .sidebar.overlay { position: fixed; top: 0; bottom: 0; left: 0; width: var(--sidebar-width-expanded); min-width: var(--sidebar-width-expanded); flex-basis: var(--sidebar-width-expanded); transform: translateX(-105%); box-shadow: var(--shadow-overlay); transition: transform var(--sidebar-transition); } .sidebar.overlay.overlay-open { transform: translateX(0); } .sidebar.overlay:not(.compact) .brand > div:last-child, .sidebar.overlay:not(.compact) .sidebar-heading > span, .sidebar.overlay:not(.compact) .group-heading span:first-child, .sidebar.overlay:not(.compact) .nav-label, .sidebar.overlay:not(.compact) .nav-state, .sidebar.overlay:not(.compact) .connection span:last-child, .sidebar.overlay:not(.compact) .settings-button span:last-child { display: initial; } .sidebar { width: 64px; min-width: 64px; flex-basis: 64px; padding-inline: 8px; } .sidebar:not(.hidden) .brand > div:last-child, .sidebar .sidebar-heading > span, .sidebar .group-heading span:first-child, .sidebar .nav-label, .sidebar .nav-state, .sidebar .connection span:last-child, .sidebar .settings-button span:last-child { display: none; } .sidebar .sidebar-heading, .sidebar .group-heading { justify-content: center; padding-inline: 0; } .sidebar .nav-item, .sidebar .settings-button { justify-content: center; padding-inline: 7px; } .sidebar .active-dot { position: absolute; right: 4px; } .topbar { padding: 0 16px; } .scope-pill { display: none; } .content { padding: 30px 20px 48px; } }
</style>
