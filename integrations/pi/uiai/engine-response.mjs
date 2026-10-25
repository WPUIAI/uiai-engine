// Shared JSON/session boundary for native, MCP and package adapters. Fetch owns
// Content-Encoding decoding; never gunzip a decoded body or spread raw text.
const sessionID = value => typeof value === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(value);

export async function readEngineResponse(response, { opening = false, closeSession, validate } = {}) {
  let data;
  async function cleanup() {
    if (!opening) return "not_applicable";
    const created = response.headers.get("X-UIAI-Created-Session-ID");
    const id = sessionID(created) ? created : sessionID(data?.session?.id) ? data.session.id : null;
    if (!id || !closeSession) return "unknown_session";
    try { await closeSession(id); return "closed"; }
    catch { return "failed"; } // Returned explicitly; never silently claim cleanup.
  }
  try {
    data = await response.json();
    if (!data || typeof data !== "object" || Array.isArray(data) ||
        (Object.keys(data).length && Object.keys(data).every(key => /^\d+$/.test(key)))) {
      throw new Error("invalid JSON envelope");
    }
    if (!response.ok) {
      if (opening) return { ...data, session_cleanup: await cleanup() };
      return data;
    }
    if (opening) {
      if (!sessionID(data?.session?.id)) throw new Error("missing session identity");
      const created = response.headers.get("X-UIAI-Created-Session-ID");
      if (created && created !== data.session.id) throw new Error("session identity mismatch");
    }
    validate?.(data);
    return data;
  } catch (cause) {
    const cleanupState = await cleanup();
    const evidenceFailure = cause?.code === "epwa_delivery_unavailable";
    const error = new Error((evidenceFailure ? cause.message : "UIAI invalid response: expected a typed JSON/session envelope") +
      (opening ? `; session_cleanup=${cleanupState}` : ""));
    error.code = evidenceFailure ? cause.code : "invalid_engine_response";
    error.status = response.status;
    error.cleanup_state = cleanupState;
    throw error;
  }
}
