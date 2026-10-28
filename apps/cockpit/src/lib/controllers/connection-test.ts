import { probeEngineConnection } from "../engine-discovery";
import { discoverFocusaDaemons, focusaDaemonSummary } from "../focusa-daemon-discovery";

/** Read-only candidate test. Persisting connection or scope is the Save action's job. */
export async function testConnectionDraft(
  candidateUrl: string,
  probe = probeEngineConnection,
  discover = discoverFocusaDaemons,
): Promise<{ success: boolean; message: string }> {
  const [engine, daemons] = await Promise.allSettled([
    Promise.resolve().then(() => probe(candidateUrl)),
    Promise.resolve().then(() => discover()),
  ]);
  const success = engine.status === "fulfilled" && engine.value.status === "connected";
  const engineMessage = engine.status === "rejected"
    ? engine.reason instanceof Error ? engine.reason.message : "Engine connection test failed"
    : success ? `healthy · ${engine.value.service || "UIAI Engine"}` : "Engine unavailable";
  const daemonMessage = daemons.status === "fulfilled"
    ? focusaDaemonSummary(daemons.value) : "Focusa discovery unavailable";
  return { success, message: `${engineMessage} · ${daemonMessage} · Saved settings unchanged.` };
}
