import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer } from "node:http";
import { once } from "node:events";
import { fileURLToPath } from "node:url";
import path from "node:path";
import test from "node:test";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

function callMcp(child, request) {
  return new Promise((resolve, reject) => {
    let buffer = "";
    const timeout = setTimeout(() => {
      cleanup();
      reject(new Error("MCP response timed out"));
    }, 5000);
    const onData = (chunk) => {
      buffer += chunk;
      const newline = buffer.indexOf("\n");
      if (newline < 0) return;
      cleanup();
      try {
        resolve(JSON.parse(buffer.slice(0, newline)));
      } catch (error) {
        reject(error);
      }
    };
    const onError = (error) => {
      cleanup();
      reject(error);
    };
    const cleanup = () => {
      clearTimeout(timeout);
      child.stdout.off("data", onData);
      child.off("error", onError);
    };
    child.stdout.on("data", onData);
    child.once("error", onError);
    child.stdin.write(`${JSON.stringify(request)}\n`);
  });
}

test("MCP evidence-producing operations forward complete Focusa scope", async (t) => {
  let captured;
  const server = createServer(async (request, response) => {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    captured = {
      method: request.method,
      url: request.url,
      headers: request.headers,
      body: JSON.parse(Buffer.concat(chunks).toString("utf8")),
    };
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({
      delivery_state: "ready",
      artifact_url: "https://evidence.example/e/record",
      portable_url: "https://evidence.example/e/record",
    }));
  });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const address = server.address();
  const engine = `http://127.0.0.1:${address.port}`;
  const child = spawn(process.execPath, ["mcp/browser-session-mcp.mjs"], {
    cwd: root,
    env: { ...process.env, UIAI_ENGINE_URL: engine },
    stdio: ["pipe", "pipe", "pipe"],
  });
  t.after(() => {
    child.kill();
    server.close();
  });

  const scope = {
    project_ref: "project:uiai-engine",
    workstream_ref: "workstream:epwa",
    workset_ref: "workset:scope",
    callgraph_ref: "callgraph:scope",
    workpoint_ref: "workpoint:scope",
    work_item_ref: "work-item:scope",
    continuity_ref: "continuity:scope",
    work_items: [{ work_item_ref: "work-item:scope", title: "Forward scope" }],
  };
  const openResult = await callMcp(child, {
    jsonrpc: "2.0",
    id: 0,
    method: "tools/call",
    params: {
      name: "browser_open",
      arguments: { url: "https://example.test", focusa_scope: scope },
    },
  });
  assert.equal(openResult.error, undefined, JSON.stringify(openResult));
  assert.equal(captured.method, "POST");
  assert.equal(captured.url, "/api/session");
  assert.deepEqual(captured.body.focusa_scope, scope);
  assert.equal(captured.headers["x-uiai-workpoint-ref"], scope.workpoint_ref);
  assert.equal(captured.headers["x-uiai-continuity-ref"], scope.continuity_ref);
  assert.deepEqual(JSON.parse(captured.headers["x-uiai-work-items"]), scope.work_items);

  const result = await callMcp(child, {
    jsonrpc: "2.0",
    id: 1,
    method: "tools/call",
    params: {
      name: "browser_screenshot",
      arguments: { session_id: "session-1", format: "png", focusa_scope: scope },
    },
  });

  assert.equal(result.error, undefined, JSON.stringify(result));
  assert.equal(captured.method, "POST");
  assert.equal(captured.url, "/api/session/session-1/screenshot");
  assert.deepEqual(captured.body.focusa_scope, scope);
  assert.equal(captured.headers["x-uiai-workpoint-ref"], scope.workpoint_ref);
  assert.equal(captured.headers["x-uiai-continuity-ref"], scope.continuity_ref);
  assert.deepEqual(JSON.parse(captured.headers["x-uiai-work-items"]), scope.work_items);

  const oneShotResult = await callMcp(child, {
    jsonrpc: "2.0",
    id: 2,
    method: "tools/call",
    params: {
      name: "screenshot",
      arguments: { url: "https://example.test", format: "png", focusa_scope: scope },
    },
  });
  assert.equal(oneShotResult.error, undefined, JSON.stringify(oneShotResult));
  assert.equal(captured.method, "POST");
  assert.equal(captured.url, "/api/screenshot");
  assert.deepEqual(captured.body.focusa_scope, scope);
  assert.equal(captured.headers["x-uiai-workpoint-ref"], scope.workpoint_ref);
  assert.equal(captured.headers["x-uiai-continuity-ref"], scope.continuity_ref);
  assert.deepEqual(JSON.parse(captured.headers["x-uiai-work-items"]), scope.work_items);

  const packetResult = await callMcp(child, {
    jsonrpc: "2.0",
    id: 3,
    method: "tools/call",
    params: {
      name: "uiai_focusa_packet_compose",
      arguments: { goal: "Prove scope", responses: [], focusa_scope: scope },
    },
  });
  assert.equal(packetResult.error, undefined, JSON.stringify(packetResult));
  assert.equal(captured.method, "POST");
  assert.equal(captured.url, "/api/agent/research-packet");
  assert.deepEqual(captured.body.focusa_scope, scope);
  assert.equal(captured.headers["x-uiai-workpoint-ref"], scope.workpoint_ref);
  assert.equal(captured.headers["x-uiai-continuity-ref"], scope.continuity_ref);
  assert.deepEqual(JSON.parse(captured.headers["x-uiai-work-items"]), scope.work_items);
});
