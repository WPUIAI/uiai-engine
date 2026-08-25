<script lang="ts">
  import { onMount } from "svelte";
  import { invoke } from "@tauri-apps/api/core";

  type Profile = Record<string, any>;
  type BrowserSettings = {
    default_profile: string;
    profiles: Record<string, Profile>;
    domain_rules?: Array<Record<string, any>>;
  };

  let configPath = "config.yaml";
  let browser: BrowserSettings = { default_profile: "detect", profiles: {} };
  let selectedProfile = "detect";
  let status = "Loading browser profiles…";
  let error = "";
  let busy = false;

  const modes = ["detect", "no_detect", "operator", "research", "auto"];
  const engines = ["system_chromium", "chromium", "camoufox"];
  const networkRoutes = ["direct", "local_ip_pool", "named_proxy", "tailscale_exit", "operator_route"];
  const challengePolicies = ["disabled", "detect", "assist", "solve", "solve_and_retry"];

  $: profileNames = Object.keys(browser.profiles ?? {});
  $: profile = browser.profiles?.[selectedProfile] ?? {};

  onMount(async () => {
    await loadDefaults();
  });

  async function loadDefaults() {
    busy = true;
    error = "";
    try {
      browser = await invoke<BrowserSettings>("browser_profiles_default");
      selectedProfile = browser.default_profile;
      status = "Default profile set loaded.";
    } catch (cause) {
      error = String(cause);
      status = "Tauri profile bridge is unavailable in this preview.";
    } finally {
      busy = false;
    }
  }

  async function loadConfig() {
    busy = true;
    error = "";
    try {
      browser = await invoke<BrowserSettings>("browser_profiles_load", { path: configPath });
      selectedProfile = browser.profiles[browser.default_profile] ? browser.default_profile : Object.keys(browser.profiles)[0];
      status = `Loaded ${profileNames.length} profiles from ${configPath}.`;
    } catch (cause) {
      error = String(cause);
      status = "Load failed.";
    } finally {
      busy = false;
    }
  }

  async function validateConfig() {
    busy = true;
    error = "";
    try {
      await invoke("browser_profiles_validate", { browser });
      status = `${profileNames.length} browser profiles validated.`;
    } catch (cause) {
      error = String(cause);
      status = "Validation failed.";
    } finally {
      busy = false;
    }
  }

  async function saveConfig() {
    busy = true;
    error = "";
    try {
      await invoke("browser_profiles_save", { path: configPath, browser });
      status = `Saved browser profiles to ${configPath}; previous file preserved as .bak.`;
    } catch (cause) {
      error = String(cause);
      status = "Save failed.";
    } finally {
      busy = false;
    }
  }

  function setProfileValue(key: string, value: any) {
    browser.profiles[selectedProfile] = { ...profile, [key]: value };
    browser = { ...browser, profiles: { ...browser.profiles } };
  }

  function setNested(section: string, key: string, value: any) {
    browser.profiles[selectedProfile] = {
      ...profile,
      [section]: { ...(profile[section] ?? {}), [key]: value }
    };
    browser = { ...browser, profiles: { ...browser.profiles } };
  }

  function cloneProfile() {
    const base = selectedProfile || "profile";
    let index = 2;
    let name = `${base}_${index}`;
    while (browser.profiles[name]) {
      index += 1;
      name = `${base}_${index}`;
    }
    browser.profiles[name] = JSON.parse(JSON.stringify(profile));
    browser.profiles[name].label = `${profile.label ?? base} ${index}`;
    browser = { ...browser, profiles: { ...browser.profiles } };
    selectedProfile = name;
    status = `Cloned ${base} as ${name}.`;
  }

  function deleteProfile() {
    if (selectedProfile === browser.default_profile || profileNames.length <= 1) return;
    const next = { ...browser.profiles };
    delete next[selectedProfile];
    browser = { ...browser, profiles: next };
    selectedProfile = Object.keys(next)[0];
    status = "Profile removed from the editor. Save to persist.";
  }
</script>

<section class="settings" aria-labelledby="browser-profile-heading">
  <header class="section-header">
    <div>
      <p class="eyebrow">Settings</p>
      <h2 id="browser-profile-heading">Browser Profiles</h2>
      <p class="muted">Choose and configure detect, no-detect, operator, and research browser postures in the Go core.</p>
    </div>
    <span class="status-pill" data-error={Boolean(error)}>{error ? "Needs attention" : "Configured"}</span>
  </header>

  <div class="toolbar">
    <label class="path-field">
      <span>Engine config</span>
      <input bind:value={configPath} aria-label="Engine configuration path" />
    </label>
    <button on:click={loadConfig} disabled={busy}>Load</button>
    <button on:click={validateConfig} disabled={busy}>Validate</button>
    <button class="primary" on:click={saveConfig} disabled={busy}>Save</button>
  </div>

  <div class="status-line" aria-live="polite">
    <span>{status}</span>
    {#if error}<code>{error}</code>{/if}
  </div>

  <div class="profile-layout">
    <aside aria-label="Browser profiles">
      <label class="default-profile">
        <span>Default</span>
        <select bind:value={browser.default_profile}>
          {#each profileNames as name}<option value={name}>{name}</option>{/each}
        </select>
      </label>

      <nav>
        {#each profileNames as name}
          <button
            class:active={selectedProfile === name}
            on:click={() => (selectedProfile = name)}
            aria-current={selectedProfile === name ? "page" : undefined}
          >
            <strong>{browser.profiles[name]?.label ?? name}</strong>
            <small>{browser.profiles[name]?.mode ?? `extends ${browser.profiles[name]?.extends ?? "profile"}`}</small>
          </button>
        {/each}
      </nav>

      <div class="profile-actions">
        <button on:click={cloneProfile}>Clone</button>
        <button on:click={deleteProfile} disabled={selectedProfile === browser.default_profile || profileNames.length <= 1}>Delete</button>
      </div>
    </aside>

    <div class="editor">
      <div class="field-grid">
        <label>
          <span>Label</span>
          <input value={profile.label ?? ""} on:input={(event) => setProfileValue("label", event.currentTarget.value)} />
        </label>
        <label>
          <span>Extends</span>
          <select value={profile.extends ?? ""} on:change={(event) => setProfileValue("extends", event.currentTarget.value)}>
            <option value="">None</option>
            {#each profileNames.filter((name) => name !== selectedProfile) as name}<option value={name}>{name}</option>{/each}
          </select>
        </label>
        <label>
          <span>Mode</span>
          <select value={profile.mode ?? "detect"} on:change={(event) => setProfileValue("mode", event.currentTarget.value)}>
            {#each modes as mode}<option value={mode}>{mode}</option>{/each}
          </select>
        </label>
        <label>
          <span>Engine</span>
          <select value={profile.engine ?? "system_chromium"} on:change={(event) => setProfileValue("engine", event.currentTarget.value)}>
            {#each engines as engine}<option value={engine}>{engine}</option>{/each}
          </select>
        </label>
      </div>

      <fieldset>
        <legend>Launch and operator context</legend>
        <div class="field-grid">
          <label class="toggle"><input type="checkbox" checked={profile.headless ?? true} on:change={(event) => setProfileValue("headless", event.currentTarget.checked)} /><span>Headless</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.launch?.persistent_context ?? false} on:change={(event) => setNested("launch", "persistent_context", event.currentTarget.checked)} /><span>Persistent context</span></label>
          <label class="wide"><span>Executable path</span><input value={profile.launch?.executable_path ?? ""} on:input={(event) => setNested("launch", "executable_path", event.currentTarget.value)} /></label>
          <label class="wide"><span>User-data directory</span><input value={profile.launch?.user_data_dir ?? ""} on:input={(event) => setNested("launch", "user_data_dir", event.currentTarget.value)} /></label>
        </div>
      </fieldset>

      <fieldset>
        <legend>Identity and anti-detect</legend>
        <div class="toggle-grid">
          <label class="toggle"><input type="checkbox" checked={profile.identity?.patch_webdriver ?? false} on:change={(event) => setNested("identity", "patch_webdriver", event.currentTarget.checked)} /><span>Patch WebDriver</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.identity?.disable_automation_controlled ?? false} on:change={(event) => setNested("identity", "disable_automation_controlled", event.currentTarget.checked)} /><span>Disable AutomationControlled</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.identity?.patch_chrome_object ?? false} on:change={(event) => setNested("identity", "patch_chrome_object", event.currentTarget.checked)} /><span>Patch Chrome object</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.identity?.patch_plugins ?? false} on:change={(event) => setNested("identity", "patch_plugins", event.currentTarget.checked)} /><span>Patch plugins</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.identity?.patch_languages ?? false} on:change={(event) => setNested("identity", "patch_languages", event.currentTarget.checked)} /><span>Patch languages</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.identity?.random_user_agent ?? false} on:change={(event) => setNested("identity", "random_user_agent", event.currentTarget.checked)} /><span>Random user agent</span></label>
        </div>
        <div class="field-grid compact">
          <label><span>Locale</span><input value={profile.identity?.locale ?? ""} on:input={(event) => setNested("identity", "locale", event.currentTarget.value)} /></label>
          <label><span>Timezone</span><input value={profile.identity?.timezone ?? ""} on:input={(event) => setNested("identity", "timezone", event.currentTarget.value)} /></label>
          <label class="wide"><span>User agent</span><input value={profile.identity?.user_agent ?? ""} on:input={(event) => setNested("identity", "user_agent", event.currentTarget.value)} /></label>
        </div>
      </fieldset>

      <fieldset>
        <legend>Network identity</legend>
        <div class="field-grid">
          <label><span>Route</span><select value={profile.network?.route ?? "direct"} on:change={(event) => setNested("network", "route", event.currentTarget.value)}>{#each networkRoutes as route}<option value={route}>{route}</option>{/each}</select></label>
          <label><span>Route reference</span><input value={profile.network?.route_ref ?? ""} on:input={(event) => setNested("network", "route_ref", event.currentTarget.value)} /></label>
          <label><span>Proxy server</span><input value={profile.network?.proxy_server ?? ""} on:input={(event) => setNested("network", "proxy_server", event.currentTarget.value)} /></label>
          <label><span>WebRTC mode</span><input value={profile.network?.webrtc_mode ?? ""} on:input={(event) => setNested("network", "webrtc_mode", event.currentTarget.value)} /></label>
          <label class="toggle"><input type="checkbox" checked={profile.network?.geo_consistency ?? false} on:change={(event) => setNested("network", "geo_consistency", event.currentTarget.checked)} /><span>Geo consistency</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.network?.sticky ?? false} on:change={(event) => setNested("network", "sticky", event.currentTarget.checked)} /><span>Sticky route</span></label>
        </div>
      </fieldset>

      <fieldset>
        <legend>Challenge handling</legend>
        <div class="field-grid">
          <label><span>Policy</span><select value={profile.challenge?.policy ?? "detect"} on:change={(event) => setNested("challenge", "policy", event.currentTarget.value)}>{#each challengePolicies as policy}<option value={policy}>{policy}</option>{/each}</select></label>
          <label><span>Maximum attempts</span><input type="number" min="0" max="20" value={profile.challenge?.max_attempts ?? 1} on:input={(event) => setNested("challenge", "max_attempts", Number(event.currentTarget.value))} /></label>
          <label class="toggle"><input type="checkbox" checked={profile.challenge?.route_rotation ?? false} on:change={(event) => setNested("challenge", "route_rotation", event.currentTarget.checked)} /><span>Rotate route on retry</span></label>
          <label class="toggle"><input type="checkbox" checked={profile.challenge?.operator_escalation ?? false} on:change={(event) => setNested("challenge", "operator_escalation", event.currentTarget.checked)} /><span>Operator escalation</span></label>
        </div>
      </fieldset>

      <div class="profile-summary">
        <strong>{selectedProfile}</strong>
        <span>{profile.mode ?? "inherited"}</span>
        <span>{profile.engine ?? "inherited engine"}</span>
        <span>{profile.network?.route ?? "inherited route"}</span>
        <span>{profile.challenge?.policy ?? "inherited challenge policy"}</span>
      </div>
    </div>
  </div>
</section>

<style>
  .settings { margin-top: var(--space-6); }
  .section-header { display: flex; justify-content: space-between; gap: var(--space-4); align-items: flex-start; }
  .section-header h2 { margin: 0; font-size: 24px; }
  .eyebrow { margin: 0 0 4px; text-transform: uppercase; letter-spacing: .08em; font-size: 11px; color: var(--color-text-muted); }
  .muted { color: var(--color-text-muted); margin: 6px 0 0; max-width: 760px; }
  .status-pill { border: 1px solid var(--color-success); color: var(--color-success); border-radius: var(--radius-pill); padding: 4px 10px; font-size: 12px; }
  .status-pill[data-error="true"] { border-color: var(--color-warn); color: var(--color-warn); }
  .toolbar { display: flex; align-items: end; gap: var(--space-2); margin: var(--space-4) 0 var(--space-2); }
  .toolbar button, .profile-actions button { min-height: 34px; }
  .path-field { flex: 1; }
  label > span { display: block; font-size: 12px; color: var(--color-text-muted); margin-bottom: 5px; }
  input, select, button { font: inherit; color: inherit; }
  input, select { width: 100%; box-sizing: border-box; border: 1px solid var(--color-border); border-radius: 7px; background: var(--color-surface); padding: 8px 10px; }
  button { border: 1px solid var(--color-border); border-radius: 7px; background: var(--color-surface); padding: 7px 12px; cursor: pointer; }
  button:disabled { opacity: .5; cursor: default; }
  button.primary { border-color: var(--color-accent); color: var(--color-accent); }
  .status-line { min-height: 22px; display: flex; gap: var(--space-3); align-items: center; color: var(--color-text-muted); font-size: 12px; }
  .status-line code { color: var(--color-warn); }
  .profile-layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: var(--space-4); margin-top: var(--space-3); }
  aside { border: 1px solid var(--color-border); background: var(--color-surface); border-radius: var(--radius-card); padding: var(--space-3); align-self: start; }
  .default-profile { display: block; margin-bottom: var(--space-3); }
  nav { display: grid; gap: 6px; }
  nav button { text-align: left; display: grid; gap: 2px; }
  nav button.active { border-color: var(--color-accent); background: var(--color-surface-elevated); }
  nav small { color: var(--color-text-muted); }
  .profile-actions { display: flex; gap: 6px; margin-top: var(--space-3); }
  .editor { display: grid; gap: var(--space-3); }
  fieldset { margin: 0; border: 1px solid var(--color-border); background: var(--color-surface); border-radius: var(--radius-card); padding: var(--space-4); }
  legend { font-weight: 650; padding: 0 7px; }
  .field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-3); }
  .field-grid.compact { margin-top: var(--space-3); }
  .wide { grid-column: 1 / -1; }
  .toggle-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--space-2); }
  label.toggle { display: flex; gap: 8px; align-items: center; border: 1px solid var(--color-border); border-radius: 7px; padding: 8px; }
  label.toggle input { width: auto; }
  label.toggle span { display: inline; margin: 0; color: inherit; }
  .profile-summary { display: flex; flex-wrap: wrap; gap: 8px; padding: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-card); background: var(--color-surface-elevated); }
  .profile-summary span { font-size: 12px; border: 1px solid var(--color-border); border-radius: var(--radius-pill); padding: 3px 8px; }
  @media (max-width: 900px) {
    .profile-layout { grid-template-columns: 1fr; }
    .toggle-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  }
  @media (max-width: 620px) {
    .toolbar { align-items: stretch; flex-wrap: wrap; }
    .path-field { flex-basis: 100%; }
    .field-grid, .toggle-grid { grid-template-columns: 1fr; }
    .wide { grid-column: auto; }
  }
</style>
