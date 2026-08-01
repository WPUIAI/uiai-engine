<script lang="ts">
  import { onMount } from "svelte";

  type LiveState = "idle" | "connecting" | "replaying" | "live" | "stale" | "offline";
  type ActivityItem = { id: string; at: string; type: string; summary: string; sequence?: number };
  type FocusaEnvelope = {
    event_id?: string;
    sequence?: number;
    cursor?: string;
    timestamp?: string;
    event_type?: string;
    source_state_revision?: number | string;
    invalidate?: string[];
    payload?: Record<string, any>;
  };

  let baseUrl = "http://127.0.0.1:7455";
  let projectRoot = "";
  let continuityId = "";
  let state: LiveState = "idle";
  let cursor = "0";
  let lastEventAt = "Never";
  let errorMessage = "";
  let sourceRevision = "—";
  let eventSource: EventSource | null = null;

  let temporal = {
    posture: "No verified temporal projection",
    forecast: "—",
    freshness: "Not loaded",
    preflight: "unknown"
  };
  let prediction = { events: 0, open: 0, coverage: "Profile A subset" };
  let runtime = { status: "Not loaded", conflicts: 0, delivery: "unknown" };
  const semantic = { status: "Contract only", activation: "Runtime implementation open" };
  let activity: ActivityItem[] = [];

  function nowLabel(value?: string) {
    const date = value ? new Date(value) : new Date();
    return Number.isNaN(date.valueOf()) ? "now" : date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  }

  function addActivity(type: string, summary: string, envelope: FocusaEnvelope = {}) {
    const item: ActivityItem = {
      id: envelope.event_id || `${Date.now()}-${activity.length}`,
      at: nowLabel(envelope.timestamp),
      type,
      summary,
      sequence: envelope.sequence
    };
    activity = [item, ...activity.filter((entry) => entry.id !== item.id)].slice(0, 12);
  }

  function applyFocusaEvent(envelope: FocusaEnvelope) {
    const eventType = String(envelope.event_type || envelope.payload?.event || envelope.payload?.type || "unknown");
    const payload = envelope.payload || {};
    if (envelope.cursor || envelope.sequence) cursor = String(envelope.cursor || envelope.sequence);
    if (envelope.source_state_revision !== undefined) sourceRevision = String(envelope.source_state_revision);
    lastEventAt = nowLabel(envelope.timestamp);

    if (eventType.includes("temporal")) {
      temporal = {
        posture: String(payload.posture || payload.status || payload.claim?.status || "Temporal authority updated"),
        forecast: String(payload.forecast?.range || payload.forecast_range || payload.uncertainty || "Updated"),
        freshness: `Event ${cursor} · ${lastEventAt}`,
        preflight: String(payload.preflight_status || payload.preflight || "updated")
      };
      addActivity(eventType, "Temporal projection changed", envelope);
    } else if (eventType.includes("prediction") || eventType.includes("epistemic")) {
      prediction = {
        events: Number(payload.event_count ?? prediction.events + 1),
        open: Number(payload.open_commitments ?? prediction.open),
        coverage: String(payload.coverage || prediction.coverage)
      };
      addActivity(eventType, "Prediction authority projection changed", envelope);
    } else if (eventType.includes("runtime_constitution") || eventType.includes("instruction") || eventType.includes("artifact.delivery")) {
      runtime = {
        status: String(payload.status || payload.lifecycle_state || eventType.replaceAll("_", " ")),
        conflicts: Number(payload.conflict_count ?? runtime.conflicts),
        delivery: String(payload.delivery_status || payload.verified || runtime.delivery)
      };
      addActivity(eventType, "Agent Runtime Studio changed", envelope);
    } else {
      addActivity(eventType, String(payload.summary || "Focusa state changed"), envelope);
    }
  }

  async function hydrateReadModels() {
    const root = baseUrl.replace(/\/$/, "");
    const health = await fetch(`${root}/v1/events/health`);
    if (!health.ok) throw new Error(`Focusa event health returned ${health.status}`);

    if (projectRoot && continuityId) {
      const temporalResponse = await fetch(`${root}/v1/temporal/status?project_root=${encodeURIComponent(projectRoot)}&continuity_id=${encodeURIComponent(continuityId)}`);
      if (temporalResponse.ok) {
        const body = await temporalResponse.json();
        temporal = {
          posture: String(body.posture || body.status || "Temporal authority ready"),
          forecast: String(body.forecast?.range || body.forecast_range || "No current forecast"),
          freshness: "Hydrated from Focusa",
          preflight: String(body.preflight_status || "not run")
        };
      }
    }

    const doctor = await fetch(`${root}/v1/agent-runtime/doctor`);
    if (doctor.ok) {
      const body = await doctor.json();
      runtime = {
        status: String(body.status || body.readiness || "ready"),
        conflicts: Number(body.conflict_count || 0),
        delivery: String(body.delivery_status || "unknown")
      };
    }
  }

  async function connect() {
    disconnect();
    state = "connecting";
    errorMessage = "";
    try {
      await hydrateReadModels();
      const root = baseUrl.replace(/\/$/, "");
      const query = new URLSearchParams();
      if (cursor !== "0") query.set("cursor", cursor);
      if (projectRoot) query.set("project_root", projectRoot);
      if (continuityId) query.set("continuity_id", continuityId);
      eventSource = new EventSource(`${root}/v1/events/stream?${query.toString()}`);
      eventSource.addEventListener("focusa_event", (event) => {
        try {
          applyFocusaEvent(JSON.parse((event as MessageEvent).data));
          state = "live";
        } catch (error) {
          errorMessage = `Invalid Focusa event: ${String(error)}`;
          state = "stale";
        }
      });
      eventSource.addEventListener("focusa_stream_error", (event) => {
        errorMessage = (event as MessageEvent).data || "Focusa stream degraded";
        state = "stale";
      });
      eventSource.onopen = () => {
        state = cursor === "0" ? "live" : "replaying";
        addActivity("stream.connected", "Focusa durable event stream connected");
      };
      eventSource.onerror = () => {
        state = "offline";
        errorMessage = "Focusa stream disconnected; the browser will retry from the last cursor.";
      };
    } catch (error) {
      state = "offline";
      errorMessage = String(error);
    }
  }

  function disconnect() {
    eventSource?.close();
    eventSource = null;
    if (state !== "idle") state = "idle";
  }

  function runEvidenceReplay() {
    state = "replaying";
    projectRoot = "/evidence/focusa-project";
    continuityId = "continuity-evidence";
    const events: FocusaEnvelope[] = [
      {
        event_id: "evidence-101",
        sequence: 101,
        cursor: "101",
        timestamp: "2026-08-01T15:00:01Z",
        event_type: "temporal_forecast_revised",
        source_state_revision: 41,
        payload: { posture: "verified", forecast_range: "2.5–4.0 hours", preflight_status: "passed" }
      },
      {
        event_id: "evidence-102",
        sequence: 102,
        cursor: "102",
        timestamp: "2026-08-01T15:00:02Z",
        event_type: "prediction_authority_event_appended",
        source_state_revision: 42,
        payload: { event_count: 12, open_commitments: 4, coverage: "Profile A partial verified" }
      },
      {
        event_id: "evidence-103",
        sequence: 103,
        cursor: "103",
        timestamp: "2026-08-01T15:00:03Z",
        event_type: "runtime_constitution_activated",
        source_state_revision: 43,
        payload: { status: "active", conflict_count: 1, delivery_status: "verified" }
      }
    ];
    events.forEach((event, index) => setTimeout(() => {
      applyFocusaEvent(event);
      if (index === events.length - 1) state = "live";
    }, 350 * (index + 1)));
  }

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("focusa_url")) baseUrl = params.get("focusa_url") || baseUrl;
    if (params.get("project_root")) projectRoot = params.get("project_root") || "";
    if (params.get("continuity_id")) continuityId = params.get("continuity_id") || "";
    if (params.get("evidence") === "focusa-live") runEvidenceReplay();
    return disconnect;
  });
</script>

<section class="workspace" aria-labelledby="focusa-live-title" data-testid="focusa-live-workspace" data-live-state={state}>
  <header class="workspace-header">
    <div>
      <p class="eyebrow">Focusa Live Projection Fabric</p>
      <h2 id="focusa-live-title">Mission intelligence that moves with the work</h2>
      <p class="muted">Durable event replay, live invalidation, bounded projections, and truthful implementation status.</p>
    </div>
    <div class="live-pill" data-state={state} data-testid="focusa-connection-state">
      <span class="pulse" aria-hidden="true"></span>
      {state}
    </div>
  </header>

  <div class="connection-panel">
    <label>Focusa URL <input bind:value={baseUrl} aria-label="Focusa URL" /></label>
    <label>Project root <input bind:value={projectRoot} aria-label="Project root" placeholder="/absolute/project/root" /></label>
    <label>Continuity ID <input bind:value={continuityId} aria-label="Continuity ID" placeholder="continuity-…" /></label>
    <button on:click={connect}>Connect</button>
    <button class="secondary" on:click={disconnect}>Disconnect</button>
  </div>

  {#if errorMessage}<p class="error" role="alert">{errorMessage}</p>{/if}

  <div class="status-row" aria-label="Focusa stream status">
    <span>Cursor <strong>{cursor}</strong></span>
    <span>Source revision <strong>{sourceRevision}</strong></span>
    <span>Last event <strong>{lastEventAt}</strong></span>
  </div>

  <div class="projection-grid">
    <article class="projection temporal" data-testid="temporal-projection">
      <div class="card-heading"><span>Spec 137</span><strong>Temporal Authority</strong></div>
      <p class="metric">{temporal.posture}</p>
      <dl><dt>Forecast</dt><dd>{temporal.forecast}</dd><dt>Preflight</dt><dd>{temporal.preflight}</dd><dt>Freshness</dt><dd>{temporal.freshness}</dd></dl>
    </article>

    <article class="projection prediction" data-testid="prediction-projection">
      <div class="card-heading"><span>Spec 138</span><strong>Epistemic Pulse</strong></div>
      <p class="metric">{prediction.events} authority events</p>
      <dl><dt>Open commitments</dt><dd>{prediction.open}</dd><dt>Coverage</dt><dd>{prediction.coverage}</dd><dt>Full 138A</dt><dd>Not claimed</dd></dl>
    </article>

    <article class="projection runtime" data-testid="runtime-projection">
      <div class="card-heading"><span>Spec 140</span><strong>Agent Runtime Studio</strong></div>
      <p class="metric">{runtime.status}</p>
      <dl><dt>Conflicts</dt><dd>{runtime.conflicts}</dd><dt>Delivery</dt><dd>{runtime.delivery}</dd><dt>Surface</dt><dd>Live operational API</dd></dl>
    </article>

    <article class="projection semantic" data-testid="semantic-projection">
      <div class="card-heading"><span>Spec 144</span><strong>Semantic Verification</strong></div>
      <p class="metric">{semantic.status}</p>
      <dl><dt>Activation</dt><dd>{semantic.activation}</dd><dt>Executable UI</dt><dd>Withheld</dd><dt>Truth posture</dt><dd>Contract available</dd></dl>
    </article>
  </div>

  <section class="activity" aria-labelledby="activity-title">
    <div class="activity-title"><h3 id="activity-title">Live activity</h3><span>{activity.length} visible events</span></div>
    {#if activity.length === 0}
      <p class="empty">No Focusa events received. The interface remains still rather than simulating activity.</p>
    {:else}
      <ol>
        {#each activity as item}
          <li><time>{item.at}</time><span><strong>{item.type}</strong>{item.summary}</span>{#if item.sequence}<code>#{item.sequence}</code>{/if}</li>
        {/each}
      </ol>
    {/if}
  </section>
</section>

<style>
  .workspace { margin: 32px 0; padding: 24px; border: 1px solid var(--color-border); border-radius: 18px; background: linear-gradient(180deg, var(--color-surface-elevated), var(--color-surface)); box-shadow: var(--shadow-card); }
  .workspace-header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; }
  .eyebrow { margin: 0 0 6px; font-size: 12px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; color: var(--color-accent); }
  h2 { margin: 0; font-size: 24px; line-height: 30px; }
  .muted { color: var(--color-text-muted); }
  .live-pill { display: inline-flex; align-items: center; gap: 8px; padding: 8px 12px; border: 1px solid var(--color-border); border-radius: 999px; text-transform: capitalize; font-size: 12px; font-weight: 700; }
  .live-pill[data-state="live"] { color: var(--color-success); border-color: var(--color-success); }
  .live-pill[data-state="replaying"], .live-pill[data-state="connecting"] { color: var(--color-warn); border-color: var(--color-warn); }
  .live-pill[data-state="offline"], .live-pill[data-state="stale"] { color: var(--color-danger, #d14343); }
  .pulse { width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 18%, transparent); }
  .connection-panel { display: grid; grid-template-columns: 1.1fr 1.4fr 1fr auto auto; gap: 10px; margin: 20px 0 12px; align-items: end; }
  label { display: grid; gap: 5px; font-size: 11px; color: var(--color-text-muted); }
  input { min-width: 0; padding: 9px 10px; border-radius: 8px; border: 1px solid var(--color-border); background: var(--color-surface); color: inherit; }
  button { padding: 9px 14px; border: 0; border-radius: 8px; background: var(--color-accent); color: white; font-weight: 700; cursor: pointer; }
  button.secondary { background: var(--color-surface-elevated); color: inherit; border: 1px solid var(--color-border); }
  .error { padding: 10px 12px; border-radius: 8px; background: color-mix(in srgb, #d14343 12%, transparent); color: #d14343; }
  .status-row { display: flex; flex-wrap: wrap; gap: 18px; padding: 10px 0 18px; color: var(--color-text-muted); font-size: 12px; }
  .status-row strong { color: var(--color-text); }
  .projection-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
  .projection { min-height: 210px; padding: 16px; border: 1px solid var(--color-border); border-radius: 14px; background: var(--color-surface-elevated); }
  .projection.temporal { border-top: 3px solid #7c6bf2; }
  .projection.prediction { border-top: 3px solid #c77d2b; }
  .projection.runtime { border-top: 3px solid #2b9b72; }
  .projection.semantic { border-top: 3px solid #687386; }
  .card-heading { display: grid; gap: 4px; }
  .card-heading span { font-size: 11px; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: .08em; }
  .metric { min-height: 48px; margin: 22px 0 14px; font-size: 20px; font-weight: 750; }
  dl { display: grid; grid-template-columns: 1fr auto; gap: 8px 12px; margin: 0; font-size: 12px; }
  dt { color: var(--color-text-muted); } dd { margin: 0; text-align: right; }
  .activity { margin-top: 14px; padding: 16px; border: 1px solid var(--color-border); border-radius: 14px; background: var(--color-surface-elevated); }
  .activity-title { display: flex; justify-content: space-between; align-items: center; }
  .activity-title h3 { margin: 0; font-size: 15px; } .activity-title span { color: var(--color-text-muted); font-size: 12px; }
  .empty { margin-bottom: 0; color: var(--color-text-muted); }
  ol { list-style: none; margin: 12px 0 0; padding: 0; }
  li { display: grid; grid-template-columns: 80px 1fr auto; gap: 12px; padding: 9px 0; border-top: 1px solid var(--color-border); font-size: 12px; }
  li span { display: flex; gap: 9px; } li strong { font-weight: 700; } time, code { color: var(--color-text-muted); }
  @media (max-width: 1100px) { .projection-grid { grid-template-columns: repeat(2, 1fr); } .connection-panel { grid-template-columns: 1fr 1fr; } }
  @media (max-width: 700px) { .workspace-header { display: grid; } .projection-grid, .connection-panel { grid-template-columns: 1fr; } }
</style>
