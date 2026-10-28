import type { BrowserHealth, BrowserSession, EngineHealth } from "../engine-client";

type OverviewClient = {
  health(): Promise<EngineHealth>;
  browserHealth(): Promise<BrowserHealth>;
  sessions(): Promise<BrowserSession[]>;
};

export interface OverviewState {
  endpoint: string;
  health: EngineHealth | null;
  browserHealth: BrowserHealth | null;
  // null means unknown, whereas [] is a confirmed empty response.
  sessions: BrowserSession[] | null;
  loading: boolean;
  error: string;
}

export function initialOverviewState(endpoint = ""): OverviewState {
  return { endpoint, health: null, browserHealth: null, sessions: null, loading: true, error: "" };
}

/** Owns only this view's request lifecycle, never Engine or session authority. */
export function createOverviewController(
  client: OverviewClient,
  readEndpoint: () => string,
  publish: (state: OverviewState) => void,
) {
  let generation = 0;
  let disposed = false;
  let pending: { endpoint: string; generation: number; promise: Promise<void> } | null = null;

  function refresh(): Promise<void> {
    if (disposed) return Promise.resolve();
    const endpoint = readEndpoint();
    if (pending?.endpoint === endpoint) return pending.promise;
    const requestGeneration = ++generation;
    publish(initialOverviewState(endpoint));

    const promise = (async () => {
      const messages: string[] = [];
      const result = initialOverviewState(endpoint);
      function accept<T>(response: PromiseSettledResult<T>, label: string): T | null {
        if (response.status === "fulfilled") return response.value;
        messages.push(`${label}: ${response.reason instanceof Error ? response.reason.message : "The live endpoint could not be read."}`);
        return null;
      }
      try {
        const [health, browser, sessions] = await Promise.allSettled([
          Promise.resolve().then(() => client.health()),
          Promise.resolve().then(() => client.browserHealth()),
          Promise.resolve().then(() => client.sessions()),
        ]);
        result.health = accept(health, "Engine health");
        result.browserHealth = accept(browser, "Browser health");
        const sessionData = accept(sessions, "Browser sessions");
        if (Array.isArray(sessionData)) result.sessions = sessionData;
        else if (sessions.status === "fulfilled") messages.push("Browser sessions: The engine returned an invalid session list.");
      } catch (cause) {
        messages.push(cause instanceof Error ? cause.message : "The engine could not be reached.");
      } finally {
        if (pending?.generation === requestGeneration) pending = null;
      }
      if (disposed || requestGeneration !== generation) return;
      if (readEndpoint() !== endpoint) {
        publish({ ...initialOverviewState(readEndpoint()), loading: false, error: "The Engine connection changed. Refresh live state." });
        return;
      }
      publish({ ...result, loading: false, error: messages.join(" ") });
    })();
    pending = { endpoint, generation: requestGeneration, promise };
    return promise;
  }

  function dispose() {
    disposed = true;
    generation++;
    pending = null;
  }

  return { refresh, dispose };
}
