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
  const child = spawn(process.execPath, [process.env.UIAI_TEST_MCP_SCRIPT || "mcp/browser-session-mcp.mjs"], {
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

test("MCP search preserves omission and explicit provider pinning", async (t) => {
  let captured;
  const server = createServer(async (req, res) => {
    if (req.method === "GET") { res.writeHead(200, { "content-type": "application/json" }); res.end(JSON.stringify({ tools: [] })); return; }
    const chunks = []; for await (const c of req) chunks.push(c);
    captured = JSON.parse(Buffer.concat(chunks).toString("utf8"));
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ delivery_state: "ready", artifact_url: "https://evidence.example/e/fixture", portable_url: "https://evidence.example/e/fixture" }));
  });
  server.listen(0, "127.0.0.1"); await once(server, "listening");
  const child = spawn(process.execPath, [process.env.UIAI_TEST_MCP_SCRIPT || "mcp/browser-session-mcp.mjs"], { cwd: root, env: { ...process.env, UIAI_ENGINE_URL: `http://127.0.0.1:${server.address().port}` }, stdio: ["pipe", "pipe", "pipe"] });
  t.after(() => { child.kill(); server.close(); });
  const listed = await callMcp(child, { jsonrpc: "2.0", id: 10, method: "tools/list", params: {} });
  const search = listed.result.tools.find(t => t.name === "browser_search");
  assert.ok(search); assert.equal(Object.hasOwn(search.inputSchema.properties.provider, "default"), false);
  assert.equal(search.inputSchema.properties.providers.minItems,2);
  assert.equal(search.inputSchema.properties.providers.maxItems,3);
  for (const [id, args] of [[11, { query: "fixture" }], [12, { query: "fixture", provider: "tavily" }], [13,{query:"fixture",providers:["brave","wikipedia"]}], [14,{query:"fixture",provider:"brave",providers:["brave","wikipedia"]}]]) {
    const reply = await callMcp(child, { jsonrpc: "2.0", id, method: "tools/call", params: { name: "browser_search", arguments: args } });
    assert.equal(reply.error, undefined); assert.equal(reply.result.isError, undefined);
    assert.equal(captured.provider, args.provider);
    assert.equal(Object.hasOwn(captured, "provider"), Object.hasOwn(args, "provider"));
    assert.deepEqual(captured.providers,args.providers);
    assert.equal(Object.hasOwn(captured,"providers"),Object.hasOwn(args,"providers"));
  }
});


test("MCP search errors retain safe setup next action",async(t)=>{
 for(const tc of [
  {status:503,code:"search_provider_unavailable",action:"connect_existing_credential_reference",hint:"configure search.providers.brave.api_key_env",args:{query:"fixture",provider:"brave"}},
  {status:502,code:"search_providers_failed",action:"exa: connect_existing_credential_reference",hint:"search.providers.exa.api_key_env; tavily: connect_existing_credential_reference search.providers.tavily.api_key_env",args:{query:"fixture",providers:["exa","tavily"]}},
  {status:400,code:"conflicting_search_providers",action:"choose_search_mode",hint:"use provider or providers, not both",args:{query:"fixture",provider:"brave",providers:["brave","wikipedia"]}},
  {status:403,code:"search_provider_not_approved",action:"request_provider_disclosure_approval",hint:"obtain query disclosure authorization; no automatic config or account changes",args:{query:"fixture",providers:["brave","tavily"]}},
 ]){
  const server=createServer((req,res)=>{req.resume();res.writeHead(tc.status,{"content-type":"application/json"});res.end(JSON.stringify({error:tc.code,suggested_next_action:`${tc.action}: ${tc.hint}; inspect /api/search/providers`,private_extra:"synthetic-private-extra-never-echo"}))});
  server.listen(0,"127.0.0.1");await once(server,"listening");
  const child=spawn(process.execPath,[process.env.UIAI_TEST_MCP_SCRIPT || "mcp/browser-session-mcp.mjs"],{cwd:root,env:{...process.env,UIAI_ENGINE_URL:`http://127.0.0.1:${server.address().port}`},stdio:["pipe","pipe","pipe"]});
  t.after(()=>{child.kill();server.close()});
  const reply=await callMcp(child,{jsonrpc:"2.0",id:20,method:"tools/call",params:{name:"browser_search",arguments:tc.args}});
  assert.equal(reply.error,undefined);assert.equal(reply.result.isError,true);const text=reply.result.content[0].text;assert.ok(text.includes(`Next: ${tc.action}`));assert.ok(text.includes(tc.hint));assert.doesNotMatch(text,/synthetic-private-extra-never-echo/);
 }
});

test("MCP auth tools are discoverable, route exactly, fail closed, and redact secrets", async (t) => {
  const calls = [];
  const server = createServer(async (req, res) => {
    if (req.method === "GET" && req.url === "/api/tools/mcp") {
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify({ tools: [] }));
      return;
    }
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {};
    calls.push({ method: req.method, url: req.url, body });
    res.setHeader("content-type", "application/json");
    if (req.url.endsWith("/auth/resolve") && body.resume_token === "stale-private-token") {
      res.statusCode = 409;
      res.end(JSON.stringify({
        code: "auth_challenge_pending",
        state: "AWAITING_AUTH",
        error: "invalid auth challenge resume token",
        suggested_next_action: "Call browser_auth_cancel or obtain the current private token before browser_auth_resolve.",
        resume_token: "resume_server_must_not_leak",
        private_secret: "server-secret-must-not-leak",
      }));
      return;
    }
    if (req.url.endsWith("/auth/resolve")) {
      res.end(JSON.stringify({
        resolved: true,
        session: { id: "session/auth fixture", lease_state: "ACTIVE" },
        resume_token: "resume_success_must_not_leak",
        private_secret: "success-secret-must-not-leak",
      }));
      return;
    }
    if (req.url.endsWith("/auth/cancel")) {
      res.end(JSON.stringify({
        cancelled: true,
        resume_token: "resume_cancel_must_not_leak",
        credential: "cancel-secret-must-not-leak",
      }));
      return;
    }
    res.statusCode = 404;
    res.end(JSON.stringify({ error: "unexpected route" }));
  });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const child = spawn(process.execPath, [process.env.UIAI_TEST_MCP_SCRIPT || "mcp/browser-session-mcp.mjs"], {
    cwd: root,
    env: { ...process.env, UIAI_ENGINE_URL: "http://127.0.0.1:" + server.address().port },
    stdio: ["pipe", "pipe", "pipe"],
  });
  t.after(() => { child.kill(); server.close(); });

  const listed = await callMcp(child, { jsonrpc: "2.0", id: 30, method: "tools/list", params: {} });
  for (const name of ["browser_auth_resolve", "browser_auth_cancel"]) {
    assert.ok(listed.result.tools.find(tool => tool.name === name), "missing " + name);
  }
  const resolveSchema = listed.result.tools.find(tool => tool.name === "browser_auth_resolve").inputSchema;
  assert.equal(resolveSchema.properties.resume_token.writeOnly, true);
  assert.equal(resolveSchema.additionalProperties, false);

  const resolved = await callMcp(child, { jsonrpc: "2.0", id: 31, method: "tools/call", params: {
    name: "browser_auth_resolve",
    arguments: { session_id: "session/auth fixture", resume_token: "current-private-token" },
  } });
  assert.equal(resolved.result.isError, undefined);
  assert.deepEqual(calls.at(-1), {
    method: "POST",
    url: "/api/session/session%2Fauth%20fixture/auth/resolve",
    body: { resume_token: "current-private-token" },
  });
  const resolvedText = resolved.result.content[0].text;
  assert.match(resolvedText, /"resolved": true/);
  assert.doesNotMatch(resolvedText, /resume_success|success-secret|resume_token|private_secret/);

  const stale = await callMcp(child, { jsonrpc: "2.0", id: 32, method: "tools/call", params: {
    name: "browser_auth_resolve",
    arguments: { session_id: "session/auth fixture", resume_token: "stale-private-token" },
  } });
  assert.equal(stale.result.isError, true);
  assert.match(stale.result.content[0].text, /auth_challenge_pending|browser_auth_cancel/);
  assert.doesNotMatch(stale.result.content[0].text, /stale-private-token|resume_server|server-secret|resume_token|private_secret/);

  const cancelled = await callMcp(child, { jsonrpc: "2.0", id: 33, method: "tools/call", params: {
    name: "browser_auth_cancel", arguments: { session_id: "session/auth fixture" },
  } });
  assert.equal(cancelled.result.isError, undefined);
  assert.deepEqual(calls.at(-1), {
    method: "POST",
    url: "/api/session/session%2Fauth%20fixture/auth/cancel",
    body: {},
  });
  const cancelledText = cancelled.result.content[0].text;
  assert.match(cancelledText, /"cancelled": true/);
  assert.doesNotMatch(cancelledText, /resume_cancel|cancel-secret|resume_token|credential/);
});
