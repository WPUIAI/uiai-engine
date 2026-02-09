#!/usr/bin/env node
/**
 * UIAI Browser Session — MCP Server (stdio transport)
 *
 * Thin bridge: MCP protocol ↔ Go engine HTTP API at localhost:7456.
 *
 * Context-efficient design:
 * - tools/list returns tool definitions on demand (not auto-loaded)
 * - Each tool maps 1:1 to a session HTTP endpoint
 * - Pi's mcp-adapter caches metadata, so tools/list is called once
 * - Lazy connection: server only starts when first tool is called
 *
 * Usage in ~/.pi/agent/mcp.json:
 * {
 *   "mcpServers": {
 *     "browser": {
 *       "command": "node",
 *       "args": ["/home/wpuiai/uiai-engine/mcp/browser-session-mcp.mjs"],
 *       "lifecycle": "lazy"
 *     }
 *   }
 * }
 */

import { createInterface } from "readline";

const ENGINE = process.env.UIAI_ENGINE_URL || "http://localhost:7456";

// ── MCP JSON-RPC stdio transport ──────────────────────────

const rl = createInterface({ input: process.stdin });
const send = (msg) => process.stdout.write(JSON.stringify(msg) + "\n");

rl.on("line", async (line) => {
  let req;
  try {
    req = JSON.parse(line);
  } catch {
    return;
  }

  const { id, method, params } = req;

  try {
    const result = await handleMethod(method, params || {});
    if (id !== undefined) send({ jsonrpc: "2.0", id, result });
  } catch (err) {
    if (id !== undefined) {
      send({ jsonrpc: "2.0", id, error: { code: -32000, message: err.message } });
    }
  }
});

// ── MCP Method Handlers ───────────────────────────────────

async function handleMethod(method, params) {
  switch (method) {
    case "initialize":
      return {
        protocolVersion: "2025-03-26",
        capabilities: { tools: { listChanged: false } },
        serverInfo: { name: "uiai-browser-session", version: "1.0.0" },
      };

    case "notifications/initialized":
      return undefined; // notification, no response

    case "tools/list":
      return await toolsList();

    case "tools/call":
      return await toolsCall(params.name, params.arguments || {});

    default:
      throw new Error(`Unknown method: ${method}`);
  }
}

// ── tools/list — fetched from engine, cached ──────────────

let cachedTools = null;

async function toolsList() {
  if (!cachedTools) {
    const res = await fetch(`${ENGINE}/api/tools/mcp`);
    if (!res.ok) throw new Error(`Engine tools fetch failed: ${res.status}`);
    const data = await res.json();
    cachedTools = data.tools || [];
  }
  return { tools: cachedTools };
}

// ── tools/call — routes to session HTTP endpoints ─────────

async function toolsCall(name, args) {
  let url, method, body;

  switch (name) {
    case "browser_open":
      url = `${ENGINE}/api/session`;
      method = "POST";
      body = { url: args.url, width: args.width, height: args.height };
      break;

    case "browser_screenshot":
      url = `${ENGINE}/api/session/${args.session_id}/screenshot`;
      method = "POST";
      body = { format: args.format, quality: args.quality, fullPage: args.fullPage };
      break;

    case "browser_scroll":
      url = `${ENGINE}/api/session/${args.session_id}/scroll`;
      method = "POST";
      body = { deltaX: args.deltaX, deltaY: args.deltaY, x: args.x, y: args.y };
      break;

    case "browser_click":
      url = `${ENGINE}/api/session/${args.session_id}/click`;
      method = "POST";
      body = { selector: args.selector };
      break;

    case "browser_hover":
      url = `${ENGINE}/api/session/${args.session_id}/hover`;
      method = "POST";
      body = { selector: args.selector };
      break;

    case "browser_type":
      url = `${ENGINE}/api/session/${args.session_id}/type`;
      method = "POST";
      body = { selector: args.selector, text: args.text };
      break;

    case "browser_eval":
      url = `${ENGINE}/api/session/${args.session_id}/eval`;
      method = "POST";
      body = { js: args.js };
      break;

    case "browser_snapshot":
      url = `${ENGINE}/api/session/${args.session_id}/snapshot`;
      method = "POST";
      body = { interactive: args.interactive, compact: args.compact, max_depth: args.max_depth, selector: args.selector };
      break;

    case "browser_fill":
      url = `${ENGINE}/api/session/${args.session_id}/fill`;
      method = "POST";
      body = { selector: args.selector, text: args.text };
      break;

    case "browser_select":
      url = `${ENGINE}/api/session/${args.session_id}/select`;
      method = "POST";
      body = { selector: args.selector, values: args.values };
      break;

    case "browser_press":
      url = `${ENGINE}/api/session/${args.session_id}/press`;
      method = "POST";
      body = { key: args.key };
      break;

    case "browser_back":
      url = `${ENGINE}/api/session/${args.session_id}/back`;
      method = "POST";
      break;

    case "browser_forward":
      url = `${ENGINE}/api/session/${args.session_id}/forward`;
      method = "POST";
      break;

    case "browser_text":
      url = `${ENGINE}/api/session/${args.session_id}/text`;
      method = "POST";
      body = { selector: args.selector };
      break;

    case "browser_cookies":
      url = `${ENGINE}/api/session/${args.session_id}/cookies`;
      method = "POST";
      body = { action: args.action, name: args.name, value: args.value, domain: args.domain };
      break;

    case "browser_dom":
      url = `${ENGINE}/api/session/${args.session_id}/dom`;
      method = "GET";
      break;

    case "browser_navigate":
      url = `${ENGINE}/api/session/${args.session_id}/navigate`;
      method = "POST";
      body = { url: args.url };
      break;

    case "browser_resize":
      url = `${ENGINE}/api/session/${args.session_id}/resize`;
      method = "POST";
      body = { width: args.width, height: args.height };
      break;

    case "browser_css":
      url = `${ENGINE}/api/session/${args.session_id}/css`;
      method = "POST";
      body = { css: args.css };
      break;

    case "browser_wait":
      url = `${ENGINE}/api/session/${args.session_id}/wait`;
      method = "POST";
      body = { selector: args.selector, timeout_ms: args.timeout_ms };
      break;

    case "browser_close":
      url = `${ENGINE}/api/session/${args.session_id}`;
      method = "DELETE";
      break;

    case "screenshot":
      url = `${ENGINE}/api/screenshot`;
      method = "POST";
      body = {
        url: args.url, width: args.width, height: args.height,
        format: args.format, quality: args.quality, fullPage: args.fullPage,
      };
      break;

    default:
      throw new Error(`Unknown tool: ${name}`);
  }

  const fetchOpts = { method, headers: { "Content-Type": "application/json" } };
  if (body && method !== "GET") fetchOpts.body = JSON.stringify(body);

  const res = await fetch(url, fetchOpts);
  const data = await res.json();

  if (!res.ok) {
    return {
      content: [{ type: "text", text: `Error ${res.status}: ${data.error || JSON.stringify(data)}` }],
      isError: true,
    };
  }

  // Build MCP response — include screenshot as image content if present
  const content = [];

  // Extract screenshot from response (present in most session actions)
  const b64 = data.screenshot || (data.session && data.screenshot);
  if (b64) {
    content.push({
      type: "image",
      data: b64,
      mimeType: "image/jpeg",
    });
  }

  // Build text summary (everything except the base64 blob)
  const summary = { ...data };
  delete summary.screenshot;
  if (Object.keys(summary).length > 0) {
    content.push({ type: "text", text: JSON.stringify(summary, null, 2) });
  }

  if (content.length === 0) {
    content.push({ type: "text", text: "OK" });
  }

  return { content };
}
