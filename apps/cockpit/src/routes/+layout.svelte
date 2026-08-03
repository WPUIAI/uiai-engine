<script lang="ts">
  import { onMount, tick } from "svelte";
  import "../lib/ui/design-tokens.css";
  import "../lib/ui/sidebar-primitives.css";
  import { installCockpitDeepLinkRouting } from "$lib/desktop-deep-link";
  import { discoverEngines, selectBestEngine } from "$lib/engine-discovery";
  import { engineClient, savedScope, selectEngineUrl } from "$lib/engine-client";
  import { discoverFocusaDaemons, focusaDaemonSummary, readSavedFocusaDaemonHints, saveFocusaDaemonHints, type FocusaDaemonConnection } from "$lib/focusa-daemon-discovery";
  import { projectBindingRequiresReconciliation, readFocusaProjectBindings, rememberFocusaProjectBinding, selectFocusaProjectBinding, verifyFocusaProject, type FocusaProjectBinding } from "$lib/focusa-projects";
  import { phase0Cards } from "$lib/cards/phase0-card-manifest";
  import { entitlementFromHost, installEntitlementProjection, type CanonicalEntitlementProjection } from "$lib/contracts/entitlement";
  import { parseCockpitWorkpointResume, workpointResumeFromHost, type CockpitWorkpointResume } from "$lib/contracts/workpoint-resume";
  import { buildCommandIndex, filterCommandIndex } from "$lib/navigation/command-index";
  import ResumeWorkpointButton from "$lib/ui/sidebar/ResumeWorkpointButton.svelte";
  import SidebarIcon from "$lib/ui/SidebarIcon.svelte";
  import { directionForLocale, message } from "$lib/ui/messages";
  import { reportCompletedCockpitUpdate, runCockpitUpdate, startAutomaticCockpitUpdate, type CockpitUpdateResult } from "$lib/updater";
  import ToastHost from "$lib/ui/ToastHost.svelte";
  import { APP_CHANNEL, APP_VERSION } from "$lib/version";
  import { footerDestinations, sidebarGroups, workspaceForPath, workspaceManifest, type SidebarGroup } from "$lib/navigation/sidebar-manifest";
  import { applyWorkspaceDrop, createSidebarDndAdapter } from "$lib/navigation/sidebar-dnd";
  import { defaultSidebarPreferences, readSidebarPreferences, resetSidebarPreferences, saveSidebarPreferences, type SidebarMode, type SidebarPreferencesV1 } from "$lib/navigation/sidebar-preferences";

  let preferences: SidebarPreferencesV1 = defaultSidebarPreferences();
  let activeWorkspaceId = "overview";
  let commandOpen = false;
  let commandQuery = "";
  let commandActiveIndex = 0;
  let commandInput: HTMLInputElement | null = null;
  let activityOpen = false;
  let contextOpen = false;
  let workspaceMenuOpen = false;
  let userMenuOpen = false;
  let constrainedViewport = false;
  let draggingWorkspaceId = "";
  let sidebarAnnouncement = "";
  let resizeStart: { x: number; width: number } | null = null;
  let engineStatus = "checking";
  let focusaConnections: FocusaDaemonConnection[] = [];
  let projectBindings: FocusaProjectBinding[] = [];
  let selectedDaemonBaseUrl = "";
  let projectRootDraft = "";
  let continuityDraft = "";
  let remoteNodeDraft = "";
  let remoteNodeMessage = "";
  let projectConnectState: "idle" | "verifying" | "connected" | "error" = "idle";
  let projectConnectMessage = "";
  let scope: ReturnType<typeof savedScope> = {};
  let update: CockpitUpdateResult = { phase: "checking", message: "Checking for signed updates…" };
  let resumeState: CockpitWorkpointResume | null = null;
  let entitlement: CanonicalEntitlementProjection | null = null;
  $: projectConnected = Boolean(scope.project_root);
  $: scopeConnected = Boolean(scope.project_root && scope.continuity_id);
  $: projectLabel = projectConnected ? scope.project_root?.split("/").pop() || "Project connected" : "Select a project";
  $: trajectoryLabel = scopeConnected ? "Not projected" : projectConnected ? "Workstream required" : "Project required";
  $: worksetLabel = scopeConnected ? "Trajectory required" : projectConnected ? "Workstream required" : "Project required";
  $: taskGraphLabel = scopeConnected ? "Workset required" : projectConnected ? "Workstream required" : "Project required";
  $: taskLabel = resumeState?.status === "resumable" ? "Workpoint ready" : scopeConnected ? "TaskGraph required" : projectConnected ? "Workstream required" : "Project required";

  $: activeWorkspace = workspaceManifest.find((item) => item.id === activeWorkspaceId) || workspaceManifest[0];
  $: activeNav = activeWorkspace?.label || "Overview";
  $: resolvedWorkspaceManifest = workspaceManifest.map((item) => {
    const placement = preferences.workspace_placements.find((entry) => entry.workspace_id === item.id);
    return placement ? { ...item, group: placement.display_group, order: placement.order } : item;
  });
  $: groupedWorkspaces = sidebarGroups.map((group) => ({ ...group, items: resolvedWorkspaceManifest.filter((item) => item.group === group.id && !preferences.hidden_workspace_ids.includes(item.id)).sort((a, b) => a.order - b.order) }));
  $: favoriteWorkspaces = preferences.pinned_refs.filter((ref) => ref.kind === "workspace").sort((a, b) => a.order - b.order).map((ref) => workspaceManifest.find((item) => item.id === ref.workspace_id)).filter((item): item is (typeof workspaceManifest)[number] => Boolean(item));
  $: commands = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, resumeState);
  $: resumePresentationState = resumeState?.status !== "resumable" || commands.some((command) => command.kind === "resume") ? resumeState : null;
  $: filteredCommands = filterCommandIndex(commands, commandQuery);

  onMount(() => {
    preferences = readSidebarPreferences();
    const locale = navigator.language || "en";
    document.documentElement.lang = locale;
    document.documentElement.dir = directionForLocale(locale);
    scope = savedScope();
    projectBindings = readFocusaProjectBindings();
    projectRootDraft = scope.project_root || "";
    continuityDraft = scope.continuity_id || "";
    resumeState = workpointResumeFromHost();
    entitlement = entitlementFromHost();
    const onResumeContract = (event: Event) => {
      try {
        resumeState = parseCockpitWorkpointResume((event as CustomEvent<unknown>).detail);
      } catch {
        resumeState = null;
      }
    };
    window.addEventListener("uiai:workpoint-resume", onResumeContract);
    const onEntitlementContract = (event: Event) => {
      entitlement = installEntitlementProjection((event as CustomEvent<unknown>).detail);
    };
    window.addEventListener("uiai:entitlement-state", onEntitlementContract);
    const setActiveFromPath = () => { activeWorkspaceId = workspaceForPath(window.location.pathname)?.id || (window.location.pathname === "/settings" ? "settings" : "overview"); };
    setActiveFromPath();
    const onKeyDown = (event: KeyboardEvent) => {
      const typing = event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement || event.target instanceof HTMLSelectElement;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); openCommandPalette(); }
      if ((event.metaKey || event.ctrlKey) && event.key === ",") { event.preventDefault(); navigateTo("/settings"); }
      if (!typing && event.key === "?") { event.preventDefault(); navigateTo("/help"); }
      if ((event.metaKey || event.ctrlKey) && /^[1-9]$/.test(event.key)) {
        const favoriteId = preferences.pinned_refs.filter((ref) => ref.kind === "workspace")[Number(event.key) - 1]?.workspace_id;
        const favorite = workspaceManifest.find((workspace) => workspace.id === favoriteId);
        if (favorite) { event.preventDefault(); navigateTo(favorite.route); }
      }
      if (event.key === "Escape") { commandOpen = false; workspaceMenuOpen = false; userMenuOpen = false; contextOpen = false; activityOpen = false; }
      if (event.key === "[" && !event.metaKey && !event.ctrlKey) { event.preventDefault(); setSidebarMode(preferences.mode === "expanded" ? "compact" : "expanded"); }
    };
    window.addEventListener("keydown", onKeyDown);
    const updateViewport = () => { constrainedViewport = window.innerWidth <= 780; };
    updateViewport();
    window.addEventListener("resize", updateViewport);
    const completionToastTimer = window.setTimeout(reportCompletedCockpitUpdate, 250);
    const stop = startAutomaticCockpitUpdate((result) => (update = result));
    const refreshEngineStatus = () => void engineClient.health().then((result) => (engineStatus = result.status)).catch(() => (engineStatus = "unavailable"));
    const refreshEngineDiscovery = () => void discoverEngines().then((connections) => { const selected = selectBestEngine(connections); if (selected) { selectEngineUrl(selected.baseUrl); engineStatus = "healthy"; } else engineStatus = "unavailable"; });
    refreshEngineDiscovery();
    const engineTimer = window.setInterval(refreshEngineStatus, 5000);
    const engineDiscoveryTimer = window.setInterval(refreshEngineDiscovery, 15_000);
    const refreshFocusaDaemons = () => void discoverFocusaDaemons().then((connections) => { focusaConnections = connections; if (!selectedDaemonBaseUrl) selectedDaemonBaseUrl = connections.find((item) => item.status === "connected" && item.location === "local")?.baseUrl || connections.find((item) => item.status === "connected")?.baseUrl || ""; });
    refreshFocusaDaemons();
    const focusaTimer = window.setInterval(refreshFocusaDaemons, 10_000);
    let stopDeepLinkRouting: () => void = () => {};
    void installCockpitDeepLinkRouting(navigateTo).then((stopRouting) => (stopDeepLinkRouting = stopRouting));
    return () => { window.removeEventListener("keydown", onKeyDown); window.removeEventListener("resize", updateViewport); window.removeEventListener("uiai:workpoint-resume", onResumeContract); window.removeEventListener("uiai:entitlement-state", onEntitlementContract); window.clearTimeout(completionToastTimer); window.clearInterval(engineTimer); window.clearInterval(engineDiscoveryTimer); window.clearInterval(focusaTimer); stopDeepLinkRouting(); stop(); };
  });

  function persist(next: SidebarPreferencesV1) { preferences = next; saveSidebarPreferences(next); }
  const sidebarDnd = createSidebarDndAdapter(async (target, source) => { persist(applyWorkspaceDrop(preferences, workspaceManifest, source, target)); draggingWorkspaceId = ""; sidebarAnnouncement = message("sidebar.moved", { workspace: source, group: target.display_group }); });
  function beginWorkspaceDrag(id: string) { draggingWorkspaceId = id; sidebarDnd.beginDrag(id); }
  function beginWorkspacePointerDrag(id: string, event: PointerEvent) { if (event.pointerType === "touch") { event.preventDefault(); beginWorkspaceDrag(id); } }
  function pointerDropWorkspace(item: { id: string; group: SidebarGroup; order: number }, event: PointerEvent) { if (event.pointerType === "touch" && draggingWorkspaceId) { event.preventDefault(); dropWorkspace({ workspace_id: item.id, display_group: item.group, order: item.order }); } }
  function cancelWorkspaceDrag() { draggingWorkspaceId = ""; sidebarDnd.cancelDrag(); }
  function dropWorkspace(target: { workspace_id: string; display_group: SidebarGroup; order: number }) { void sidebarDnd.commitDrop(target).catch(() => { draggingWorkspaceId = ""; sidebarDnd.cancelDrag(); }); }
  function handleWorkspaceNavKeydown(event: KeyboardEvent) {
    const current = (event.target as HTMLElement).closest<HTMLElement>("[data-workspace-row]");
    if (!current) return;
    const sidebar = current.closest<HTMLElement>(".sidebar");
    const rows = [...(sidebar?.querySelectorAll<HTMLElement>("[data-workspace-row]") || [])];
    const index = rows.indexOf(current);
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      rows[(index + (event.key === "ArrowDown" ? 1 : -1) + rows.length) % rows.length]?.focus();
    }
    const group = current.dataset.workspaceGroup as SidebarGroup | undefined;
    if (group && event.key === "ArrowLeft" && !preferences.collapsed_groups.includes(group)) { event.preventDefault(); toggleGroup(group); }
    if (group && event.key === "ArrowRight" && preferences.collapsed_groups.includes(group)) { event.preventDefault(); toggleGroup(group); }
    if (event.shiftKey && event.key === "F10") {
      event.preventDefault();
      const id = current.dataset.workspaceId;
      if (id) selectWorkspace(id);
      workspaceMenuOpen = true;
      sidebarAnnouncement = message("sidebar.commands_opened", { workspace: current.textContent?.trim() || "workspace" });
    }
  }
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
  function openCommandPalette() { commandOpen = true; commandActiveIndex = 0; void tick().then(() => commandInput?.focus()); }
  function updateCommandQuery() { commandActiveIndex = 0; }
  async function revealActiveCommand() { await tick(); document.querySelector<HTMLElement>(".command-list [data-command-active='true']")?.scrollIntoView({ block: "nearest" }); }
  function handleCommandKeydown(event: KeyboardEvent) {
    if (!filteredCommands.length) return;
    if (event.key === "ArrowDown") { event.preventDefault(); commandActiveIndex = (commandActiveIndex + 1) % filteredCommands.length; void revealActiveCommand(); }
    if (event.key === "ArrowUp") { event.preventDefault(); commandActiveIndex = (commandActiveIndex - 1 + filteredCommands.length) % filteredCommands.length; void revealActiveCommand(); }
    if (event.key === "Home") { event.preventDefault(); commandActiveIndex = 0; void revealActiveCommand(); }
    if (event.key === "End") { event.preventDefault(); commandActiveIndex = filteredCommands.length - 1; void revealActiveCommand(); }
    if (event.key === "Enter") { event.preventDefault(); navigateTo(filteredCommands[commandActiveIndex]?.href || filteredCommands[0].href); }
  }
  function navigateTo(href: string) { commandOpen = false; commandQuery = ""; commandActiveIndex = 0; workspaceMenuOpen = false; window.location.href = href; }
  function activateProjectBinding(binding: FocusaProjectBinding) {
    if (binding.status !== "verified") { projectConnectState = "error"; projectConnectMessage = "This project binding is not verified by its Focusa node."; return; }
    selectFocusaProjectBinding(binding);
    scope = savedScope();
    projectRootDraft = binding.projectRoot;
    continuityDraft = binding.continuityId || "";
    selectedDaemonBaseUrl = binding.daemonBaseUrl;
    projectConnectState = "connected";
    projectConnectMessage = projectBindingRequiresReconciliation(binding, projectBindings) ? "Project selected on this node. Another node has the same project and still requires explicit reconciliation." : "Project authority selected.";
  }
  async function addFocusaNode() {
    remoteNodeMessage = "Discovering node…";
    const saved = saveFocusaDaemonHints([...readSavedFocusaDaemonHints(), remoteNodeDraft]);
    if (!saved.some((value) => value === remoteNodeDraft.replace(/\/$/, ""))) { remoteNodeMessage = "Enter a valid http(s) Focusa daemon URL."; return; }
    focusaConnections = await discoverFocusaDaemons();
    const added = focusaConnections.find((item) => item.baseUrl === remoteNodeDraft.replace(/\/$/, ""));
    if (added?.status === "connected") { selectedDaemonBaseUrl = added.baseUrl; remoteNodeDraft = ""; remoteNodeMessage = added.location === "remote" && !added.paired ? "Remote node detected. Pairing and project reconciliation remain explicit." : "Focusa node connected."; }
    else remoteNodeMessage = "Node saved but is not currently reachable.";
  }
  async function connectProject() {
    const daemon = focusaConnections.find((item) => item.baseUrl === selectedDaemonBaseUrl);
    if (!daemon) { projectConnectState = "error"; projectConnectMessage = "Select a connected Focusa node."; return; }
    projectConnectState = "verifying";
    projectConnectMessage = `Verifying with ${daemon.displayName || daemon.baseUrl}…`;
    try {
      const binding = await verifyFocusaProject(daemon, projectRootDraft, continuityDraft);
      projectBindings = rememberFocusaProjectBinding(binding);
      if (binding.status !== "verified") { projectConnectState = "error"; projectConnectMessage = "Focusa returned an unverified project identity; scope remains unchanged."; return; }
      activateProjectBinding(binding);
    } catch (error) {
      projectConnectState = "error";
      projectConnectMessage = error instanceof Error ? error.message : "Project verification failed.";
    }
  }
  function selectWorkspace(id: string) { activeWorkspaceId = id; }
  function checkForUpdate() { void runCockpitUpdate({ install: true, reporter: (result) => (update = result) }); }
</script>

<div class="app-shell">
  <aside class:compact={preferences.mode === "compact" || constrainedViewport} class="sidebar" aria-label="Primary navigation" style={`--sidebar-width: ${preferences.width_px}px`}>
    <div class="brand"><div class="brand-identity"><div class="brand-mark" aria-hidden="true">✦</div><div class="brand-copy"><strong>UIAI</strong><span>Engine Cockpit</span></div></div><button class="sidebar-mode-button" type="button" aria-label={preferences.mode === "compact" || constrainedViewport ? "Expand sidebar" : "Collapse sidebar"} title={preferences.mode === "compact" || constrainedViewport ? "Expand sidebar" : "Collapse sidebar"} onclick={() => setSidebarMode(preferences.mode === "compact" ? "expanded" : "compact")}><svg aria-hidden="true" viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"></rect><path d="M9 4v16"></path>{#if preferences.mode === "compact" || constrainedViewport}<path d="m13 9 3 3-3 3"></path>{:else}<path d="m16 9-3 3 3 3"></path>{/if}</svg></button></div>
    <div class="sidebar-primary-actions"><button class="sidebar-action project-control" class:scope-missing={!projectConnected} type="button" aria-label={`Project: ${projectLabel}`} aria-expanded={contextOpen} onclick={() => (contextOpen = !contextOpen)} title={projectConnected ? "Inspect project authority" : "Select a project"}><span class="sidebar-action-icon"><SidebarIcon name="context" /></span><span class="sidebar-action-copy"><strong>Project</strong><small>{projectLabel}</small></span><span class="sidebar-action-chevron" aria-hidden="true">›</span></button><div class="authority-ladder" aria-label="Focusa authority ladder"><div class="authority-step"><span>Trajectory</span><small>{trajectoryLabel}</small></div><div class="authority-step"><span>Workset</span><small>{worksetLabel}</small></div><div class="authority-step"><span>TaskGraph</span><small>{taskGraphLabel}</small></div><div class="authority-step"><span>Tasks</span><small>{taskLabel}</small></div></div><button class="sidebar-action" type="button" aria-label="Find or do" title="Find or do" onclick={openCommandPalette} aria-keyshortcuts="Meta+K"><span class="sidebar-action-icon"><SidebarIcon name="search" /></span><span class="sidebar-action-copy"><strong>{message("shell.find_do")}</strong></span><kbd>⌘K</kbd></button></div>
    {#if contextOpen}<div class="context-panel sidebar-context-panel project-selector" role="dialog" aria-label="Project and Focusa authority"><div class="project-selector-header"><div><p class="screen-kicker">Project authority</p><strong>{projectConnected ? projectLabel : "Choose a project"}</strong></div><button type="button" aria-label="Close project selector" onclick={() => (contextOpen = false)}>×</button></div><section><p class="project-selector-label">Focusa nodes</p><div class="daemon-choice-list">{#each focusaConnections as daemon}<button class:selected={selectedDaemonBaseUrl === daemon.baseUrl} type="button" disabled={daemon.status !== "connected"} onclick={() => (selectedDaemonBaseUrl = daemon.baseUrl)}><span><i class:healthy={daemon.status === "connected"}></i>{daemon.displayName || (daemon.location === "local" ? "Local Focusa" : "Remote Focusa")}</span><small>{daemon.status === "connected" ? `${daemon.location}${daemon.location === "remote" ? daemon.paired ? " · paired" : " · unpaired" : ""} · ${daemon.latencyMs || 0}ms` : "Unavailable"}</small></button>{/each}</div><details class="add-node-disclosure"><summary>Add another Focusa node</summary><div><input bind:value={remoteNodeDraft} aria-label="Focusa daemon URL" placeholder="https://focusa-host:8787" autocomplete="off" /><button type="button" disabled={!remoteNodeDraft.trim()} onclick={addFocusaNode}>Discover</button></div>{#if remoteNodeMessage}<p aria-live="polite">{remoteNodeMessage}</p>{/if}</details>{#if focusaConnections.filter((item) => item.status === "connected").length > 1}<p class="multiplex-note">Multiple nodes detected. Their project authority remains separate until pairing and explicit reconciliation.</p>{/if}</section>{#if projectBindings.length}<section><p class="project-selector-label">Known projects</p><div class="project-binding-list">{#each projectBindings as binding}<button type="button" class:active={scope.project_root === binding.projectRoot && selectedDaemonBaseUrl === binding.daemonBaseUrl} onclick={() => activateProjectBinding(binding)}><span><strong>{binding.canonicalName}</strong><small>{binding.daemonLocation} · {binding.status}</small></span>{#if projectBindingRequiresReconciliation(binding, projectBindings)}<em>Reconcile</em>{:else}<span aria-hidden="true">›</span>{/if}</button>{/each}</div></section>{/if}<section><p class="project-selector-label">Connect project</p><label>Project folder<input bind:value={projectRootDraft} placeholder="/absolute/project/folder" autocomplete="off" /></label><label>Workstream / continuity ID <span>optional</span><input bind:value={continuityDraft} placeholder="Choose after project verification" autocomplete="off" /></label><button class="project-connect-button" type="button" disabled={projectConnectState === "verifying" || !selectedDaemonBaseUrl || !projectRootDraft.trim()} onclick={connectProject}>{projectConnectState === "verifying" ? "Verifying…" : "Verify and connect"}</button>{#if projectConnectMessage}<p class:error={projectConnectState === "error"} class="project-connect-message" aria-live="polite">{projectConnectMessage}</p>{/if}</section>{#if projectConnected}<section class="project-authority-summary"><p class="project-selector-label">Authority projection</p><span>Project (ScopeRef) · {scope.project_root}</span><span>Trajectory Ladder · {trajectoryLabel}</span><span>Workset · {worksetLabel}</span><span>TaskGraph · {taskGraphLabel}</span><span>Individual Tasks · {taskLabel}</span></section>{/if}</div>{/if}
    {#if resumePresentationState}<ResumeWorkpointButton state={resumePresentationState} />{/if}
    <div class="sidebar-heading"><span>Workspaces</span><button class="workspace-menu-button sidebar-row" type="button" aria-label="Open workspace layout menu" aria-expanded={workspaceMenuOpen} onclick={() => (workspaceMenuOpen = !workspaceMenuOpen)}>•••</button></div>
    {#if workspaceMenuOpen}<div class="workspace-menu sidebar-popover" role="menu" aria-label="Workspace layout"><div class="menu-title">Layout · {preferences.layout_mode}</div><button role="menuitemradio" aria-checked={preferences.layout_mode === "recommended"} type="button" onclick={() => setLayoutMode("recommended")}>Recommended <span>Manifest order</span></button><button role="menuitemradio" aria-checked={preferences.layout_mode === "custom"} type="button" onclick={() => setLayoutMode("custom")}>Custom <span>Local preference</span></button><div class="menu-title">Sidebar</div><button role="menuitemradio" aria-checked={preferences.mode === "expanded"} type="button" onclick={() => setSidebarMode("expanded")}>Expanded <span>⌘[</span></button><button role="menuitemradio" aria-checked={preferences.mode === "compact"} type="button" onclick={() => setSidebarMode("compact")}>Compact <span>Icon rail · minimum</span></button><div class="menu-title">Visibility</div>{#if preferences.hidden_workspace_ids.length}<button role="menuitem" type="button" onclick={showHiddenWorkspaces}>Show hidden workspaces <span>{preferences.hidden_workspace_ids.length}</span></button>{/if}<button role="menuitem" type="button" disabled={activeWorkspaceId === "overview"} onclick={hideActiveWorkspace}>Hide current workspace <span>{activeWorkspaceId === "overview" ? "Overview fixed" : "Local only"}</span></button>{#if preferences.mode === "expanded" && !constrainedViewport}<div class="menu-title">Groups</div><button role="menuitem" type="button" onclick={() => setAllGroups(false)}>Expand all <span>⌄</span></button><button role="menuitem" type="button" onclick={() => setAllGroups(true)}>Collapse all <span>›</span></button>{/if}<button class="menu-reset" role="menuitem" type="button" onclick={resetLayout}>Reset sidebar layout</button></div>{/if}
    <div class="workspace-groups">
      {#each groupedWorkspaces as group}
        <section class="workspace-group" aria-label={group.label}>
          {#if preferences.mode === "expanded" && !constrainedViewport}<button class="group-heading sidebar-group-header" type="button" aria-label={`${group.label} group`} title={group.label} aria-expanded={!preferences.collapsed_groups.includes(group.id)} onclick={() => toggleGroup(group.id)}><span>{group.label}</span><span aria-hidden="true">{preferences.collapsed_groups.includes(group.id) ? "›" : "⌄"}</span></button>{/if}
          {#if preferences.mode === "compact" || constrainedViewport || !preferences.collapsed_groups.includes(group.id)}<nav aria-label={`${group.label} workspaces`}> {#each group.items as item}<a class:active={activeWorkspaceId === item.id} class:quiet={item.emphasis === "quiet"} class="nav-item sidebar-row" class:dragging={draggingWorkspaceId === item.id} data-selected={activeWorkspaceId === item.id} data-workspace-row data-workspace-id={item.id} data-workspace-group={item.group} draggable="true" aria-keyshortcuts="Alt+ArrowUp Alt+ArrowDown" href={item.route} aria-label={item.label} title={item.label} aria-current={activeWorkspaceId === item.id ? "page" : undefined} onclick={() => selectWorkspace(item.id)} onpointerdown={(event) => beginWorkspacePointerDrag(item.id, event)} onpointerup={(event) => pointerDropWorkspace(item, event)} ondragstart={() => beginWorkspaceDrag(item.id)} ondragover={(event) => event.preventDefault()} ondrop={() => dropWorkspace({ workspace_id: item.id, display_group: item.group, order: item.order })} ondragend={cancelWorkspaceDrag} onkeydown={(event) => { handleWorkspaceNavKeydown(event); if (event.altKey && event.key === "ArrowUp") { event.preventDefault(); keyboardMoveWorkspace(item.id, -1); } if (event.altKey && event.key === "ArrowDown") { event.preventDefault(); keyboardMoveWorkspace(item.id, 1); } }}><span class="nav-icon sidebar-row__icon"><SidebarIcon name={item.id} /></span><span class="nav-label sidebar-row__label">{item.label}</span>{#if item.state !== "current"}<span class="nav-state sidebar-row__state">{item.state}</span>{/if}{#if activeWorkspaceId === item.id}<span class="active-dot sidebar-row__active-bar" aria-hidden="true"></span>{/if}</a>{/each}</nav>{/if}
        </section>
      {/each}
      {#if favoriteWorkspaces.length}<section class="workspace-group favorites" aria-label="Favorites"><div class="group-heading favorites-heading"><span>Favorites</span></div><nav aria-label="Favorite workspaces">{#each favoriteWorkspaces as item}<a class:active={activeWorkspaceId === item.id} class="nav-item sidebar-row" href={item.route} aria-label={`Favorite: ${item.label}`} title={item.label} aria-current={activeWorkspaceId === item.id ? "page" : undefined} onclick={() => selectWorkspace(item.id)}><span class="nav-icon sidebar-row__icon"><SidebarIcon name={item.id} /></span><span class="nav-label sidebar-row__label">{item.label}</span></a>{/each}</nav></section>{/if}
    </div>
    {#if preferences.mode === "expanded"}<button class="resize-handle sidebar-resize-handle" type="button" aria-label="Resize sidebar" onpointerdown={beginResize}></button>{/if}
    <p class="sr-only" aria-live="polite" aria-atomic="true">{sidebarAnnouncement}</p>
    <div class="sidebar-bottom"><div class="user-menu-region">{#if userMenuOpen}<div class="user-menu-popover" role="menu" aria-label="Account and Cockpit menu"><div class="account-popover-header"><span class="user-avatar"><SidebarIcon name="user" /></span><div><strong>Account not connected</strong><small>Local Cockpit access</small></div></div><div class="account-menu-section"><p>Runtime</p><div class="account-status-row"><span><i class:healthy={engineStatus === "healthy"}></i>Engine</span><strong>{engineStatus}</strong></div><div class="account-status-row"><span><i class:healthy={focusaConnections.some((item) => item.status === "connected")}></i>Focusa</span><strong>{focusaDaemonSummary(focusaConnections).replace("Focusa · ", "")}</strong></div></div><div class="account-menu-section"><p>Build</p><div class="account-status-row"><span>Version</span><strong>v{APP_VERSION}</strong></div><div class="account-status-row"><span>Channel</span><strong>{APP_CHANNEL}</strong></div></div><div class="account-menu-section account-menu-links"><p>Preferences</p><a role="menuitem" href="/settings" onclick={() => (userMenuOpen = false)}><span class="footer-icon"><SidebarIcon name="settings" /></span><span>Settings</span><span aria-hidden="true">›</span></a><a role="menuitem" href="/help" onclick={() => (userMenuOpen = false)}><span class="footer-icon"><SidebarIcon name="help" /></span><span>Help</span><span aria-hidden="true">›</span></a></div></div>{/if}<button class="user-menu-button" type="button" aria-expanded={userMenuOpen} aria-label="Open user menu" onclick={() => (userMenuOpen = !userMenuOpen)}><span class="user-avatar"><SidebarIcon name="user" /></span><span class="user-menu-label"><strong>Sign in</strong><small>Account and preferences</small></span><span class="user-menu-chevron" aria-hidden="true">›</span></button></div></div>
  </aside>

  <main class="main-shell">
    <header class="topbar"><div class="breadcrumbs"><span>{message("shell.workspace")}</span><span class="slash">/</span><strong>{activeNav}</strong></div><div class="top-actions"><button class="update-button" type="button" onclick={checkForUpdate} title={update.message}><span class="update-dot" data-phase={update.phase}></span>{update.phase === "current" ? "Up to date" : update.phase === "available" ? "Update ready" : "Updates"}</button></div></header>
    {#if !entitlement || !["active_evaluation", "active_paid", "offline_grace"].includes(entitlement.state)}<div class="entitlement-banner" role="status"><div><strong>{message("shell.recovery_only")}</strong><span>{entitlement ? `Entitlement is ${entitlement.state}. Local artifacts and Evidence remain available.` : "No canonical UIAI entitlement state is connected. Execution allocation is blocked."}</span></div><a href={entitlement?.recovery_actions[0]?.href || "/nodes-services?view=uiai-engine"}>{entitlement?.recovery_actions[0]?.label || message("shell.manage_entitlement")} →</a></div>{/if}
    <section class="content"><slot /></section>
    <div class:expanded={activityOpen} class="activity-bar" aria-live="polite"><button class="activity-toggle" type="button" onclick={() => (activityOpen = !activityOpen)}><span class="activity-live-dot" data-phase={update.phase}></span><strong>Activity</strong><span>{activityOpen ? "Close" : update.phase === "available" ? "Update ready" : "View signals"}</span><span class="activity-chevron">{activityOpen ? "⌃" : "⌄"}</span></button>{#if activityOpen}<div class="activity-panel"><div><strong>Update state</strong><span>{update.message}</span></div><button class="text-action" type="button" onclick={checkForUpdate}>Check again</button></div>{/if}</div>
  </main>
</div>
<ToastHost />
{#if commandOpen}<div class="command-backdrop" role="presentation" onclick={(event) => event.target === event.currentTarget && (commandOpen = false)}><dialog open class="command-palette" aria-label="Command palette"><div class="command-heading"><span>{message("shell.find_do")}</span><span class="command-live"><i></i>Live</span><kbd>Esc</kbd></div><input bind:this={commandInput} bind:value={commandQuery} aria-label="Search commands" aria-controls="command-results" aria-activedescendant={filteredCommands[commandActiveIndex] ? `command-${commandActiveIndex}` : undefined} placeholder={message("shell.search_placeholder")} oninput={updateCommandQuery} onkeydown={handleCommandKeydown} /> <div class="command-result-status" aria-live="polite"><span>{filteredCommands.length} {filteredCommands.length === 1 ? "result" : "results"}</span><span>↑↓ Navigate · ↵ Open</span></div>{#if filteredCommands.length}<div id="command-results" class="command-list" role="listbox" aria-label="Command results">{#each filteredCommands as command, index}<button id={`command-${index}`} type="button" role="option" aria-selected={commandActiveIndex === index} class:active={commandActiveIndex === index} data-command-active={commandActiveIndex === index} onmouseenter={() => (commandActiveIndex = index)} onclick={() => navigateTo(command.href)}><span>{command.label}</span><small>{command.hint}</small></button>{/each}</div>{:else}<p class="command-empty">No commands in the current project authority.</p>{/if}</dialog></div>{/if}

<style>
  .app-shell { min-height: 100vh; display: flex; background: var(--color-bg); }
  .sidebar { position: sticky; top: 0; z-index: 10; width: var(--sidebar-width); min-width: var(--sidebar-width-min); max-width: var(--sidebar-width-max); height: 100vh; box-sizing: border-box; flex: 0 0 var(--sidebar-width); display: flex; flex-direction: column; overflow: visible; padding: 18px 12px 12px; background: var(--sidebar-surface); border-right: 1px solid var(--sidebar-divider); transition: width var(--sidebar-transition), flex-basis var(--sidebar-transition); }
  .sidebar.compact { width: var(--sidebar-width-compact); min-width: var(--sidebar-width-compact); flex-basis: var(--sidebar-width-compact); padding-inline: 8px; }
  .brand { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 0 8px 18px 12px; } .brand-identity { min-width: 0; display: flex; align-items: center; gap: 10px; } .sidebar-mode-button { flex: 0 0 auto; width: 28px; height: 28px; display: grid; place-items: center; border: 0; border-radius: 7px; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 17px; cursor: pointer; } .sidebar-mode-button:hover { color: var(--color-text); background: var(--sidebar-hover-surface); } .sidebar-mode-button svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.6; } .sidebar-primary-actions { position: relative; display: grid; gap: 3px; padding: 0 4px 12px; } .sidebar-action { width: 100%; min-height: 34px; display: flex; align-items: center; gap: 10px; padding: 6px 8px; border: 0; border-radius: 8px; color: var(--color-text-muted); background: transparent; font: inherit; text-align: left; cursor: pointer; } .sidebar-action:hover { color: var(--color-text); background: var(--sidebar-hover-surface); } .sidebar-action.scope-missing .sidebar-action-icon { color: var(--color-warn); } .authority-ladder { position: relative; display: grid; gap: 2px; margin: -2px 8px 5px 36px; padding-left: 10px; border-left: 1px solid var(--color-border); } .authority-step { min-width: 0; display: flex; align-items: baseline; justify-content: space-between; gap: 8px; min-height: 20px; color: var(--color-text-muted); } .authority-step > span { font-size: 11px; font-weight: 500; } .authority-step > small { overflow: hidden; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; } .sidebar-action-icon { flex: 0 0 auto; width: 18px; color: var(--color-accent); text-align: center; font-size: 17px; } .sidebar-action-copy { min-width: 0; display: grid; gap: 1px; } .sidebar-action-copy strong { font-size: 13px; font-weight: 500; } .sidebar-action-copy small { overflow: hidden; color: var(--color-text-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; } .sidebar-action-chevron { margin-left: auto; } .sidebar-action kbd { margin-left: auto; padding: 2px 4px; border: 1px solid var(--color-border); border-radius: 4px; color: var(--color-text-muted); background: transparent; font-size: 9px; } .sidebar-context-panel { top: 62px; right: auto; left: calc(100% - 8px); z-index: 40; max-width: calc(100vw - var(--sidebar-width) - 24px); } .sidebar-context-panel span { overflow: visible; text-overflow: clip; white-space: normal; } .project-selector { top: 48px; width: min(390px, calc(100vw - var(--sidebar-width) - 24px)); max-width: none; max-height: calc(100vh - 64px); display: block; overflow-x: hidden; overflow-y: auto; padding: 10px; scrollbar-width: thin; } .project-selector-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; padding: 4px 5px 9px; } .project-selector-header p { margin: 0 0 4px; } .project-selector-header strong { display: block; max-width: 290px; font-size: 14px; } .project-selector-header > button { width: 26px; height: 26px; border: 0; border-radius: 6px; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 18px; cursor: pointer; } .project-selector-header > button:hover { color: var(--color-text); background: var(--sidebar-hover-surface); } .project-selector section { display: grid; gap: 7px; padding: 9px 5px; border-top: 1px solid var(--color-border); } .project-selector-label { margin: 0; color: var(--color-text-muted); font-size: 9px; font-weight: 600; letter-spacing: .06em; text-transform: uppercase; } .daemon-choice-list, .project-binding-list { display: grid; gap: 3px; } .daemon-choice-list button, .project-binding-list button { width: 100%; min-height: 38px; display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 6px 8px; border: 1px solid transparent; border-radius: 7px; color: var(--color-text); background: transparent; font: inherit; text-align: left; cursor: pointer; } .daemon-choice-list button:hover, .project-binding-list button:hover, .daemon-choice-list button.selected, .project-binding-list button.active { background: var(--sidebar-hover-surface); border-color: var(--color-border); } .daemon-choice-list button:disabled { cursor: default; opacity: .58; } .daemon-choice-list button > span { display: inline-flex; align-items: center; gap: 8px; font-size: 11px; } .daemon-choice-list i { width: 7px; height: 7px; border-radius: 50%; background: var(--color-warn); } .daemon-choice-list i.healthy { background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 12%, transparent); } .daemon-choice-list small, .project-binding-list small { color: var(--color-text-muted); font-size: 9px; } .project-binding-list button > span:first-child { min-width: 0; display: grid; gap: 2px; } .project-binding-list strong { overflow: hidden; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; } .project-binding-list em { color: var(--color-warn); font-size: 9px; font-style: normal; } .add-node-disclosure { margin-top: 2px; } .add-node-disclosure summary { padding: 5px 3px; color: var(--color-accent); font-size: 10px; cursor: pointer; } .add-node-disclosure > div { display: grid; grid-template-columns: 1fr auto; gap: 6px; margin-top: 4px; } .add-node-disclosure > div > button { padding: 0 10px; border: 1px solid var(--color-border); border-radius: 7px; color: var(--color-text); background: var(--color-surface); font: inherit; font-size: 10px; cursor: pointer; } .add-node-disclosure > div > button:disabled { opacity: .5; cursor: default; } .add-node-disclosure > p { margin: 6px 2px 0; color: var(--color-text-muted); font-size: 9px; line-height: 1.4; } .multiplex-note { margin: 1px 3px 0; padding: 7px 8px; border-radius: 6px; color: var(--color-text-muted); background: color-mix(in srgb, var(--color-warn) 8%, transparent); font-size: 9px; line-height: 1.4; } .project-selector label { display: grid; gap: 4px; color: var(--color-text-muted); font-size: 10px; } .project-selector label > span { margin-left: 4px; font-size: 9px; } .project-selector input { width: 100%; height: 32px; padding: 0 9px; border: 1px solid var(--color-border); border-radius: 7px; outline: 0; color: var(--color-text); background: var(--color-surface); font: inherit; font-size: 11px; } .project-selector input:focus { border-color: color-mix(in srgb, var(--color-accent) 65%, var(--color-border)); } .project-connect-button { min-height: 32px; border: 0; border-radius: 7px; color: white; background: var(--color-accent); font: inherit; font-size: 11px; font-weight: 600; cursor: pointer; } .project-connect-button:disabled { cursor: default; opacity: .5; } .project-connect-message { margin: 0; padding: 6px 8px; border-radius: 6px; color: var(--color-success); background: color-mix(in srgb, var(--color-success) 8%, transparent); font-size: 9px; line-height: 1.4; } .project-connect-message.error { color: var(--color-error); background: color-mix(in srgb, var(--color-error) 8%, transparent); } .project-authority-summary > span { color: var(--color-text-muted); font-size: 9px; } .brand-mark { width: 30px; height: 30px; display: grid; place-items: center; color: white; background: linear-gradient(135deg, #6b7cff, #9a5cff); border-radius: 9px; box-shadow: 0 5px 14px rgba(112, 92, 255, .28); font-size: 17px; } .brand-copy strong, .brand-copy span { display: block; letter-spacing: -.01em; } .brand strong { font-size: 14px; line-height: 16px; } .brand span { color: var(--color-text-muted); font-size: 11px; line-height: 15px; }
  .sidebar-heading { display: flex; align-items: center; justify-content: space-between; padding: 0 8px 8px 12px; color: var(--color-text-muted); font-size: 12px; font-weight: 600; } .workspace-menu-button { border: 0; color: var(--color-text-muted); background: transparent; font: inherit; letter-spacing: .12em; cursor: pointer; } .workspace-menu-button:hover { color: var(--color-text); }
  .workspace-groups { flex: 1 1 auto; min-width: 0; min-height: 0; overflow-x: hidden; overflow-y: auto; padding-bottom: 12px; scrollbar-gutter: stable; scrollbar-width: thin; mask-image: linear-gradient(to bottom, #000 0, #000 calc(100% - 12px), transparent 100%); } .workspace-group { margin-bottom: 8px; } .group-heading { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 5px 12px; border: 0; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 12px; font-weight: 600; text-align: left; cursor: pointer; } .group-heading:hover { color: var(--color-text); }
  nav { display: grid; gap: 3px; } .nav-item { position: relative; width: 100%; min-height: var(--sidebar-row-height); display: flex; align-items: center; gap: 10px; padding: 8px 12px; border: 0; border-radius: 8px; color: var(--color-text-muted); background: transparent; font: inherit; font-size: 13px; text-align: left; text-decoration: none; cursor: pointer; transition: color var(--motion-fast) ease-out, background var(--motion-fast) ease-out, transform var(--motion-fast) ease-out; } .nav-item:hover { color: var(--color-text); background: var(--sidebar-hover-surface); } .nav-item:active, .update-button:active { transform: scale(.98); } .nav-item.active { color: var(--color-text); background: var(--sidebar-selected-surface); font-weight: 600; } .nav-item.active::before { content: ""; position: absolute; inset-block: 7px; inset-inline-start: 0; width: var(--sidebar-active-bar-width); border-radius: var(--sidebar-active-bar-width); background: var(--color-accent); } .nav-item.quiet { font-size: 12px; } .nav-item.dragging { opacity: .58; box-shadow: var(--sidebar-drag-elevation); } .nav-icon, .footer-icon { flex: 0 0 auto; width: 18px; height: 18px; color: currentColor; } .nav-label { min-width: 0; } .nav-state { flex: 0 0 auto; margin-left: auto; color: var(--color-text-muted); font-size: 11px; } .active-dot { width: 4px; height: 4px; margin-left: auto; border-radius: 50%; background: var(--color-accent); }
  .workspace-menu { position: absolute; top: 76px; left: calc(100% - 8px); z-index: 30; width: clamp(212px, 24vw, 260px); max-width: calc(100vw - var(--sidebar-width) - 24px); max-height: calc(100vh - 92px); box-sizing: border-box; overflow-x: hidden; overflow-y: auto; padding: 7px; border: 1px solid var(--color-border-strong); border-radius: 9px; background: var(--color-surface-elevated); box-shadow: var(--shadow-popover); } .menu-title { padding: 7px 9px 4px; color: var(--color-text-muted); font-size: 10px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; } .workspace-menu button { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 8px 9px; border: 0; border-radius: 6px; color: var(--color-text); background: transparent; font: inherit; font-size: 12px; text-align: left; cursor: pointer; } .workspace-menu button:hover { background: color-mix(in srgb, var(--color-accent) 9%, transparent); } .workspace-menu button span { color: var(--color-text-muted); font-size: 10px; } .workspace-menu .menu-reset { margin-top: 6px; border-top: 1px solid var(--color-border); border-radius: 0; color: var(--color-accent); }
  .resize-handle { position: absolute; top: 0; right: -3px; bottom: 0; z-index: 2; width: 6px; border: 0; background: transparent; cursor: col-resize; } .resize-handle:hover, .resize-handle:focus-visible { background: var(--sidebar-resize-handle); outline: none; }
  .sidebar-bottom { position: relative; margin-top: auto; padding: 8px 0 0; border-top: 1px solid var(--color-border); } .user-menu-region { position: relative; } .user-menu-button { width: 100%; min-height: 44px; display: flex; align-items: center; gap: 9px; padding: 6px 8px; border: 0; border-radius: 8px; color: var(--color-text); background: transparent; font: inherit; text-align: left; cursor: pointer; } .user-menu-button:hover { background: var(--sidebar-hover-surface); } .user-avatar { flex: 0 0 auto; width: 25px; height: 25px; display: grid; place-items: center; border: 1px solid var(--color-border-strong); border-radius: 50%; padding: 4px; color: var(--color-accent); font-size: 15px; } .user-menu-label { min-width: 0; display: grid; gap: 1px; } .user-menu-label strong { font-size: 13px; font-weight: 500; } .user-menu-label small { overflow: hidden; color: var(--color-text-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; } .user-menu-chevron { margin-left: auto; color: var(--color-text-muted); } .user-menu-popover { position: absolute; left: calc(100% + 12px); bottom: 0; z-index: 40; width: min(280px, calc(100vw - var(--sidebar-width) - 24px)); display: grid; gap: 0; overflow: hidden; padding: 8px; border: 1px solid var(--color-border-strong); border-radius: 12px; color: var(--color-text); background: var(--color-surface-elevated); box-shadow: var(--shadow-popover); } .account-popover-header { display: flex; align-items: center; gap: 10px; padding: 8px 8px 10px; } .account-popover-header > div { min-width: 0; display: grid; gap: 2px; } .account-popover-header strong { font-size: 13px; font-weight: 600; } .account-popover-header small { color: var(--color-text-muted); font-size: 10px; } .account-menu-section { display: grid; gap: 2px; padding: 8px 6px; border-top: 1px solid var(--color-border); } .account-menu-section > p { margin: 0 0 4px; color: var(--color-text-muted); font-size: 9px; font-weight: 600; letter-spacing: .06em; text-transform: uppercase; } .account-status-row { min-height: 26px; display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--color-text-muted); font-size: 11px; } .account-status-row > span { display: inline-flex; align-items: center; gap: 8px; } .account-status-row strong { overflow: hidden; color: var(--color-text); font-size: 11px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; } .account-status-row i { width: 7px; height: 7px; border-radius: 50%; background: var(--color-warn); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-warn) 12%, transparent); } .account-status-row i.healthy { background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 12%, transparent); } .account-menu-links a { min-height: 32px; display: grid; grid-template-columns: 18px 1fr auto; align-items: center; gap: 9px; padding: 6px 7px; border-radius: 7px; color: var(--color-text); font-size: 12px; text-decoration: none; } .account-menu-links a:hover { background: var(--sidebar-hover-surface); } .account-menu-links a > span:last-child { color: var(--color-text-muted); } .update-dot { width: 7px; height: 7px; display: inline-block; border-radius: 50%; background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 14%, transparent); }   .sidebar.compact .brand { justify-content: center; padding-inline: 0; } .sidebar.compact .brand-identity { display: none; } .sidebar.compact .sidebar-heading > span, .sidebar.compact .group-heading span:first-child, .sidebar.compact .nav-label, .sidebar.compact .nav-state, .sidebar.compact .user-menu-label, .sidebar.compact .user-menu-chevron { display: none; } .sidebar.compact .user-menu-button { justify-content: center; padding-inline: 4px; } .sidebar.compact .sidebar-primary-actions { padding-inline: 0; } .sidebar.compact .sidebar-action { justify-content: center; padding-inline: 6px; } .sidebar.compact .sidebar-action-copy, .sidebar.compact .sidebar-action-chevron, .sidebar.compact .sidebar-action kbd, .sidebar.compact .authority-ladder { display: none; } .sidebar.compact .sidebar-heading { justify-content: center; padding-inline: 0; } .sidebar.compact .workspace-group { margin-bottom: 3px; } .sidebar.compact .workspace-group + .workspace-group { margin-top: 5px; padding-top: 6px; border-top: 1px solid var(--color-border); } .sidebar.compact .workspace-menu-button { font-size: 12px; } .sidebar.compact .nav-item { justify-content: center; padding-inline: 7px; } .sidebar.compact .active-dot { position: absolute; right: 4px; }
  .main-shell { min-width: 0; flex: 1; } .topbar { height: 62px; display: flex; align-items: center; justify-content: space-between; padding: 0 32px; border-bottom: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-bg) 82%, transparent); backdrop-filter: blur(18px); } .breadcrumbs { display: flex; gap: 10px; align-items: center; color: var(--color-text-muted); font-size: 13px; } .breadcrumbs strong { color: var(--color-text); font-weight: 600; } .slash { color: var(--color-border); } .top-actions { position: relative; display: flex; align-items: center; gap: 10px; } kbd { padding: 2px 5px; border: 1px solid var(--color-border); border-radius: 4px; color: var(--color-text-muted); background: var(--color-surface-elevated); font: inherit; font-size: 10px; }
  .context-panel { position: absolute; top: 40px; right: 38px; z-index: 25; display: grid; gap: 6px; width: min(360px, calc(100vw - 32px)); padding: 14px; border: 1px solid var(--color-border); border-radius: var(--radius-card); color: var(--color-text-muted); background: var(--color-surface-elevated); box-shadow: var(--shadow-popover); font-size: 11px; } .context-panel strong { overflow: hidden; color: var(--color-text); text-overflow: ellipsis; white-space: nowrap; } .context-panel span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .update-button { display: inline-flex; align-items: center; gap: 7px; height: 30px; padding: 0 10px; border: 1px solid var(--color-border); border-radius: 7px; color: var(--color-text-muted); background: var(--color-surface); font: inherit; font-size: 11px; } .scope-missing { border-color: color-mix(in srgb, var(--color-warn) 35%, var(--color-border)); } .update-button { cursor: pointer; } .update-dot[data-phase="error"] { background: var(--color-error); } .update-dot[data-phase="available"], .update-dot[data-phase="downloading"], .update-dot[data-phase="installing"] { background: var(--color-accent); }   .content { max-width: 1180px; margin: 0 auto; padding: 48px 52px 88px; } .activity-bar { position: fixed; right: 18px; bottom: 16px; z-index: 5; width: min(360px, calc(100vw - 36px)); border: 1px solid var(--color-border); border-radius: 10px; background: color-mix(in srgb, var(--color-surface-elevated) 88%, transparent); box-shadow: var(--shadow-popover); backdrop-filter: blur(16px); } .activity-toggle { display: flex; align-items: center; gap: 8px; width: 100%; padding: 10px 12px; border: 0; color: var(--color-text); background: transparent; font: inherit; font-size: 12px; text-align: left; cursor: pointer; } .activity-toggle span:nth-of-type(2) { margin-left: auto; color: var(--color-text-muted); font-size: 11px; } .activity-chevron { color: var(--color-text-muted); } .activity-live-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-success); } .activity-live-dot[data-phase="available"] { background: var(--color-accent); } .activity-live-dot[data-phase="error"] { background: var(--color-error); } .activity-panel { display: flex; align-items: center; gap: 12px; padding: 12px; border-top: 1px solid var(--color-border); color: var(--color-text-muted); font-size: 11px; } .activity-panel div { min-width: 0; flex: 1; } .activity-panel strong, .activity-panel span { display: block; } .activity-panel strong { color: var(--color-text); font-size: 12px; } .text-action { border: 0; color: var(--color-accent); background: none; font: inherit; font-size: 11px; cursor: pointer; white-space: nowrap; }
  .command-backdrop { position: fixed; inset: 0; z-index: 50; display: grid; place-items: start center; overflow: hidden; padding: clamp(96px, 18vh, 180px) 16px 32px; background: rgba(0, 0, 0, .3); backdrop-filter: blur(8px); } .command-palette { position: relative; inset: auto; width: min(520px, calc(100vw - 32px)); max-height: min(620px, calc(82vh - 32px)); margin: 0; display: flex; flex-direction: column; overflow: hidden; padding: 0; border: 1px solid var(--color-border-strong); border-radius: 12px; color: var(--color-text); background: var(--color-surface-elevated); box-shadow: var(--shadow-overlay); } .command-heading { display: flex; align-items: center; gap: 9px; padding: 12px 14px 7px; color: var(--color-text-muted); font-size: 10px; } .command-heading > span:first-child { margin-right: auto; } .command-live { display: inline-flex; align-items: center; gap: 5px; color: var(--color-text-muted); } .command-live i { width: 6px; height: 6px; border-radius: 50%; background: var(--color-success); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 12%, transparent); } .command-palette > input { width: calc(100% - 28px); margin: 0 14px; padding: 9px 0 10px; border: 0; border-bottom: 1px solid var(--color-border); outline: 0; color: var(--color-text); background: transparent; font: inherit; font-size: 14px; } .command-palette > input:focus { border-bottom-color: color-mix(in srgb, var(--color-accent) 65%, var(--color-border)); } .command-result-status { display: flex; justify-content: space-between; gap: 12px; padding: 7px 14px 5px; color: var(--color-text-muted); font-size: 9px; } .command-list { min-height: 0; display: grid; gap: 2px; overflow-x: hidden; overflow-y: auto; overscroll-behavior: contain; padding: 4px 7px 8px; scroll-padding-block: 7px; scrollbar-gutter: stable; scrollbar-width: thin; } .command-list button { position: relative; min-height: 34px; display: flex; align-items: center; justify-content: space-between; gap: 18px; padding: 7px 10px; border: 0; border-radius: 7px; color: var(--color-text); background: transparent; font: inherit; font-size: 12px; text-align: left; cursor: pointer; transition: background var(--motion-fast) ease-out, color var(--motion-fast) ease-out; } .command-list button:hover, .command-list button.active { background: color-mix(in srgb, var(--color-accent) 14%, var(--color-surface-elevated)); } .command-list button.active::before { content: ""; position: absolute; inset-block: 8px; inset-inline-start: 0; width: 2px; border-radius: 2px; background: var(--color-accent); } .command-list small { overflow: hidden; color: var(--color-text-muted); font-size: 9.5px; text-overflow: ellipsis; white-space: nowrap; } .command-empty { margin: 10px 14px 18px; color: var(--color-text-muted); font-size: 12px; }
  @media (prefers-reduced-motion: reduce) { .sidebar { transition: none; } } @media (max-width: 780px) { .sidebar { width: var(--sidebar-width-compact); min-width: var(--sidebar-width-compact); flex-basis: var(--sidebar-width-compact); padding-inline: 8px; } .topbar { padding: 0 16px; } .content { padding: 26px 18px 44px; } }
  .entitlement-banner { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 9px 18px; border-bottom: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-warn) 9%, var(--color-surface)); color: var(--color-text); font-size: 12px; }
  .entitlement-banner div { display: grid; gap: 2px; }
  .entitlement-banner span { color: var(--color-text-muted); }
  .entitlement-banner a { flex: 0 0 auto; color: inherit; font-weight: 700; }
  .entitlement-banner a:focus-visible { outline: 3px solid var(--color-focus-ring); outline-offset: 2px; }
  @media (max-width: 640px) { .entitlement-banner { align-items: flex-start; flex-direction: column; } }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }
  @media (prefers-reduced-motion: reduce) { .app-shell *, .app-shell *::before, .app-shell *::after { scroll-behavior: auto !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; transition-duration: .01ms !important; } }
  :global([dir="rtl"]) .sidebar { border-right: 0; border-left: 1px solid var(--sidebar-divider); }
  :global([dir="rtl"]) .group-heading { text-align: right; }
  :global([dir="rtl"]) .active-dot { right: auto; left: 3px; }
  :global([dir="rtl"]) .context-panel { right: auto; left: 18px; }
  @media (forced-colors: active) { .nav-item.active, :global(.screen-button), .entitlement-banner { border: 1px solid CanvasText; } .activity-live-dot { forced-color-adjust: none; } }
</style>
