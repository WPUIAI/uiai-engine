import { readFileSync } from "node:fs";
import { describe, expect, it, vi } from "vitest";
import { createOverviewController, initialOverviewState, type OverviewState } from "../../src/lib/controllers/overview-state";
import type { EngineHealth } from "../../src/lib/engine-client";

function setup() {
  const client = {
    health: vi.fn(async (): Promise<EngineHealth> => ({ status: "healthy", service: "uiai-engine" })),
    browserHealth: vi.fn(async () => ({ browser_state: "ready", active_pages: 0 })),
    sessions: vi.fn(async () => [{ id: "owned-fixture", url: "https://example.org/" }]),
  };
  let endpoint = "http://engine-a:7456";
  const states: OverviewState[] = [];
  const controller = createOverviewController(client, () => endpoint, (state) => states.push(state));
  return { client, controller, states, latest: () => states.at(-1)!, endpoint: (value: string) => { endpoint = value; } };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

describe("Overview live-state lifecycle", () => {
  it("starts unknown, not with a confirmed empty session list", () => {
    expect(initialOverviewState()).toMatchObject({ health: null, browserHealth: null, sessions: null, loading: true });
  });

  it("publishes a successful current snapshot", async () => {
    const t = setup();
    await t.controller.refresh();
    expect(t.latest()).toMatchObject({ endpoint: "http://engine-a:7456", health: { status: "healthy" }, browserHealth: { browser_state: "ready" }, loading: false, error: "" });
    expect(t.latest().sessions).toHaveLength(1);
  });

  it("distinguishes a confirmed empty session response from failure", async () => {
    const t = setup();
    t.client.sessions.mockResolvedValue([]);
    await t.controller.refresh();
    expect(t.latest().sessions).toEqual([]);
    expect(t.latest().error).toBe("");
  });

  it("keeps an unreachable first load unknown instead of reporting All clear", async () => {
    const t = setup();
    t.client.health.mockRejectedValue(new Error("offline"));
    t.client.browserHealth.mockRejectedValue(new Error("offline"));
    t.client.sessions.mockRejectedValue(new Error("offline"));
    await t.controller.refresh();
    expect(t.latest()).toMatchObject({ health: null, browserHealth: null, sessions: null, loading: false });
    expect(t.latest().error).toContain("Browser sessions: offline");
  });

  it("preserves independent successful reads while marking failed sessions unknown", async () => {
    const t = setup();
    t.client.sessions.mockRejectedValue(new Error("session read denied"));
    await t.controller.refresh();
    expect(t.latest().health?.status).toBe("healthy");
    expect(t.latest().browserHealth?.browser_state).toBe("ready");
    expect(t.latest().sessions).toBeNull();
    expect(t.latest().error).toBe("Browser sessions: session read denied");
  });

  it("does not throw away valid sessions because browser health fails", async () => {
    const t = setup();
    t.client.browserHealth.mockRejectedValue(new Error("browser health unavailable"));
    await t.controller.refresh();
    expect(t.latest().sessions).toHaveLength(1);
    expect(t.latest().browserHealth).toBeNull();
    expect(t.latest().error).toContain("Browser health:");
  });

  it("clears old successful data as a refresh starts and never revives it on failure", async () => {
    const t = setup();
    await t.controller.refresh();
    t.client.health.mockRejectedValue(new Error("disconnected"));
    t.client.browserHealth.mockRejectedValue(new Error("disconnected"));
    t.client.sessions.mockRejectedValue(new Error("disconnected"));
    const refresh = t.controller.refresh();
    expect(t.latest()).toMatchObject({ health: null, browserHealth: null, sessions: null, loading: true });
    await refresh;
    expect(t.latest()).toMatchObject({ health: null, browserHealth: null, sessions: null, loading: false });
  });

  it("recovers after a failed read", async () => {
    const t = setup();
    t.client.sessions.mockRejectedValueOnce(new Error("temporary failure"));
    await t.controller.refresh();
    expect(t.latest().sessions).toBeNull();
    await t.controller.refresh();
    expect(t.latest().sessions).toHaveLength(1);
    expect(t.latest().error).toBe("");
  });

  it("coalesces overlapping polls for the same endpoint", async () => {
    const t = setup();
    const health = deferred<EngineHealth>();
    t.client.health.mockImplementationOnce(() => health.promise);
    const first = t.controller.refresh();
    expect(t.controller.refresh()).toBe(first);
    await Promise.resolve();
    expect(t.client.health).toHaveBeenCalledTimes(1);
    health.resolve({ status: "healthy" });
    await first;
    await t.controller.refresh();
    expect(t.client.health).toHaveBeenCalledTimes(2);
  });

  it("ignores a late result from an endpoint superseded by a newer request", async () => {
    const t = setup();
    const oldHealth = deferred<EngineHealth>();
    t.client.health.mockImplementationOnce(() => oldHealth.promise);
    const old = t.controller.refresh();
    await Promise.resolve();
    t.endpoint("http://engine-b:7456");
    t.client.sessions.mockResolvedValue([{ id: "new-engine-session", url: "https://example.org/new" }]);
    await t.controller.refresh();
    const current = t.latest();
    expect(current.endpoint).toBe("http://engine-b:7456");
    oldHealth.resolve({ status: "unhealthy" });
    await old;
    expect(t.latest()).toBe(current);
    expect(t.latest().sessions?.[0].id).toBe("new-engine-session");
  });

  it("never labels data from a changed endpoint as current even before the next poll", async () => {
    const t = setup();
    const health = deferred<EngineHealth>();
    t.client.health.mockImplementationOnce(() => health.promise);
    const refresh = t.controller.refresh();
    await Promise.resolve();
    t.endpoint("http://engine-b:7456");
    health.resolve({ status: "healthy" });
    await refresh;
    expect(t.latest()).toMatchObject({ endpoint: "http://engine-b:7456", health: null, sessions: null, loading: false });
    expect(t.latest().error).toContain("connection changed");
  });

  it("does not publish or start more work after disposal", async () => {
    const t = setup();
    const health = deferred<EngineHealth>();
    t.client.health.mockImplementationOnce(() => health.promise);
    const refresh = t.controller.refresh();
    await Promise.resolve();
    t.controller.dispose();
    const count = t.states.length;
    health.resolve({ status: "healthy" });
    await refresh;
    await t.controller.refresh();
    expect(t.states).toHaveLength(count);
    expect(t.client.health).toHaveBeenCalledTimes(1);
  });

  it("treats a malformed session response as unknown instead of crashing the view", async () => {
    const t = setup();
    t.client.sessions.mockResolvedValue(undefined as never);
    await t.controller.refresh();
    expect(t.latest().sessions).toBeNull();
    expect(t.latest().error).toContain("invalid session list");
  });

  it("isolates synchronous adapter failures and allows retry", async () => {
    const t = setup();
    t.client.health.mockImplementationOnce(() => { throw new Error("adapter failed"); });
    await t.controller.refresh();
    expect(t.latest().health).toBeNull();
    expect(t.latest().sessions).toHaveLength(1);
    await t.controller.refresh();
    expect(t.latest().health?.status).toBe("healthy");
  });

  it("handles non-Error rejections without leaking object payloads into UI", async () => {
    const t = setup();
    t.client.sessions.mockRejectedValue({ arbitrary: "payload" });
    await t.controller.refresh();
    expect(t.latest().error).toBe("Browser sessions: The live endpoint could not be read.");
  });

  it("wires unknown, loading and confirmed-empty states into the real Overview", () => {
    const overview = readFileSync(new URL("../../src/routes/+page.svelte", import.meta.url), "utf8");
    expect(overview).toContain("createOverviewController(engineClient, engineUrl");
    expect(overview).toContain("overview.dispose()");
    expect(overview).toContain("{:else if !sessionsKnown}");
    expect(overview).toContain("No live browser sessions");
    expect(overview).toContain("Background job status is not provided here.");
    expect(overview).not.toContain("All clear");
    expect(overview).not.toContain("browserHealth?.active_pages ?? 0");
  });
});
