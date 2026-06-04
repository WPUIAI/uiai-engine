package routes

import (
	"net/http"
	"sort"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

// MountToolsDiscovery serves tool definitions for LLM integration.
// GET /api/tools           → all tool definitions (OpenAI format)
// GET /api/tools/openai    → OpenAI function calling format
// GET /api/tools/mcp       → MCP tool definitions
// GET /api/tools/search?q= → search tools by name/description (low-context discovery)
// GET /api/tools/agent-card → compact bootstrap guide for local/remote agents
// GET /api/tools/graph      → tool relationship graph and workflow routes
//
// Design: tools are NEVER auto-loaded into LLM context.
// Agents discover via search, then call tools by name.
func MountToolsDiscovery(r chi.Router, _ *config.Config) {

	// Full tool list — OpenAI function calling format (default)
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"format": "openai_function_calling",
			"tools":  openAITools(),
			"count":  len(openAITools()),
			"note":   "Use GET /api/tools/search?q=screenshot for visual tools or ?q=console/?q=network/?q=error for browser_diagnostics without loading all definitions",
		})
	})

	r.Get("/openai", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, openAITools())
	})

	r.Get("/mcp", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, mcpTools())
	})

	r.Get("/agent-card", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, agentBootstrapCard())
	})

	r.Get("/graph", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, toolGraph())
	})

	// Search — the key endpoint for context-efficient discovery.
	// Returns only matching tools, not the full list.
	r.Get("/search", func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query().Get("q")
		if q == "" {
			// No query → return just names + one-line descriptions (minimal context)
			type briefTool struct {
				Name string `json:"name"`
				Desc string `json:"description"`
			}
			brief := make([]briefTool, 0)
			for _, t := range openAITools() {
				brief = append(brief, briefTool{
					Name: t["name"].(string),
					Desc: t["description"].(string),
				})
			}
			writeJSON(w, 200, map[string]any{
				"tools": brief,
				"count": len(brief),
				"hint":  "Add ?q=keyword to search. Try q=search, q=screenshot, q=console, q=network, q=error, q=exception, q=devtools, q=retry, q=flakiness, or q=failed_request. Full definitions at /api/tools/openai or /api/tools/mcp",
			})
			return
		}

		// Search by substring in name or description. Rank name matches before
		// description matches so specific searches like q=eval_async surface
		// browser_eval_async before browser_eval mentioning it in guidance.
		matches := rankedToolSearch(openAITools(), q)
		writeJSON(w, 200, map[string]any{
			"query": q,
			"tools": matches,
			"count": len(matches),
		})
	})
}

func agentBootstrapCard() map[string]any {
	return map[string]any{
		"service":  "uiai-engine",
		"purpose":  "Lightweight browser, web search, screenshot, visual QA, and diagnostics backend for local and remote agents.",
		"base_url": "http://localhost:7456",
		"discovery": map[string]string{
			"agent_card": "/api/tools/agent-card",
			"search":     "/api/tools/search?q=<keyword>",
			"openai":     "/api/tools/openai",
			"mcp":        "/api/tools/mcp",
			"graph":      "/api/tools/graph",
			"focusa":     "pass focusa_scope to browser_open; ingest diagnostics with focusa_browser_diagnostics_intake",
			"health":     "/api/health/browser",
			"metrics":    "/api/metrics/browser",
		},
		"recommended_workflows": []map[string]any{
			{
				"name":  "search_then_browse",
				"steps": []string{"browser_search", "browser_open selected result", "browser_read for page text", "browser_snapshot for actions", "browser_diagnostics on failure", "browser_close"},
			},
			{
				"name":  "persistent_browser_loop",
				"steps": []string{"browser_open", "browser_read for page text", "browser_snapshot for actions", "browser_click/browser_fill/browser_press", "browser_diagnostics on failure or visual mismatch", "browser_close"},
			},
			{
				"name":  "single_screenshot_check",
				"steps": []string{"screenshot", "frame_render when device framing is needed"},
			},
			{
				"name":  "diagnostics_first_debugging",
				"steps": []string{"reproduce with direct session actions", "browser_diagnostics", "classify console/exception/network/selector/timeout", "patch only after evidence"},
			},
		},
		"search_hints": []string{"search", "web search", "screenshot", "snapshot", "read", "extract", "click", "fill", "eval_async", "diagnostics", "console", "network", "error", "failed_request", "blank page", "visual failure"},
		"reliability_rules": []string{
			"Prefer browser_snapshot @refs over brittle CSS guessing.",
			"Keep browser_eval synchronous and short; use browser_eval_async only for bounded awaits.",
			"After any failed action, blank page, unexpected navigation, or API suspicion, read browser_diagnostics before patching.",
			"Close sessions when done to free browser pages.",
		},
	}
}

func rankedToolSearch(tools []map[string]any, query string) []map[string]any {
	type rankedTool struct {
		tool  map[string]any
		score int
		idx   int
	}
	ranked := make([]rankedTool, 0)
	for idx, t := range tools {
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		score := toolSearchScore(name, desc, query)
		if score > 0 {
			ranked = append(ranked, rankedTool{tool: t, score: score, idx: idx})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].idx < ranked[j].idx
	})
	matches := make([]map[string]any, 0, len(ranked))
	for _, item := range ranked {
		matches = append(matches, item.tool)
	}
	return matches
}

func toolSearchScore(name, desc, query string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0
	}
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(desc)
	score := 0
	if nameLower == q {
		score = max(score, 1000)
	}
	if strings.HasPrefix(nameLower, q) {
		score = max(score, 900)
	}
	if strings.Contains(nameLower, q) {
		score = max(score, 800)
	}
	// Token-ish aliases help users search by the suffix after browser_.
	for _, part := range strings.FieldsFunc(nameLower, func(r rune) bool { return r == '_' || r == '-' || r == '.' }) {
		if part == q {
			score = max(score, 700)
		}
	}
	if strings.Contains(descLower, q) {
		score = max(score, 100)
	}
	return score
}

// containsCI is defined in intelligence.go (shared within routes package)

func toolGraph() map[string]any {
	return map[string]any{
		"schema":    "uiai.tool_graph.v1",
		"service":   "uiai-engine",
		"principle": "Every primary tool advertises adjacent tools so agents can route from intent → action → evidence → Focusa handoff → cleanup.",
		"focusa_integration": map[string]any{
			"scope_input":            "browser_open accepts focusa_scope or flat workpoint_id/continuity_id/project_root/evidence_ref fields",
			"scope_echo":             "session info, diagnostics, and failure envelopes echo focusa_scope when present",
			"evidence_refs":          []string{"uiai-diagnostics:session=<id>:seq=<seq>", "uiai-screenshot:sha256:<prefix>", "uiai-share:<share_id>"},
			"preferred_focusa_tools": []string{"focusa_browser_diagnostics_intake", "focusa_evidence_capture", "focusa_workpoint_link_evidence", "focusa_predict_record"},
		},
		"workflows": []map[string]any{
			{"name": "search_then_browse", "steps": []string{"browser_search", "browser_open selected result", "browser_read", "browser_snapshot", "browser_diagnostics", "browser_close"}},
			{"name": "web_surfing", "steps": []string{"browser_open", "browser_read", "browser_snapshot", "browser_click/browser_fill/browser_press", "browser_diagnostics", "browser_close"}},
			{"name": "visual_debug", "steps": []string{"browser_open", "browser_screenshot", "browser_diagnostics", "browser_eval_async", "browser_close"}},
			{"name": "single_capture", "steps": []string{"screenshot", "frame_catalog", "frame_render"}},
			{"name": "focusa_evidence", "steps": []string{"browser_open with focusa_scope", "browser_diagnostics", "focusa_browser_diagnostics_intake", "browser_read", "focusa_evidence_capture/link", "screenshot/share evidence handles"}},
		},
		"related_tools": toolRelations(),
	}
}

func toolRelations() map[string][]string {
	return map[string][]string{
		"uiai_agent_card":           {"uiai_tool_search", "browser_open", "browser_read", "browser_diagnostics", "focusa_browser_diagnostics_intake"},
		"uiai_tool_search":          {"uiai_agent_card", "browser_search", "browser_open", "browser_read", "browser_diagnostics"},
		"uiai_health":               {"uiai_status", "browser_open", "browser_diagnostics"},
		"uiai_status":               {"uiai_health", "uiai_agent_card", "uiai_tool_graph"},
		"critique_models":           {"critique_dimensions", "uiai_status", "uiai_tool_graph"},
		"critique_dimensions":       {"critique_models", "uiai_tool_graph"},
		"browser_search":            {"browser_open", "browser_read", "browser_diagnostics", "uiai_tool_search"},
		"browser_open":              {"browser_read", "browser_snapshot", "browser_diagnostics", "focusa_browser_diagnostics_intake", "browser_close"},
		"browser_read":              {"browser_snapshot", "browser_text", "browser_diagnostics", "browser_close"},
		"browser_snapshot":          {"browser_click", "browser_fill", "browser_hover", "browser_text", "browser_diagnostics"},
		"browser_click":             {"browser_snapshot", "browser_read", "browser_diagnostics", "browser_wait"},
		"browser_fill":              {"browser_snapshot", "browser_press", "browser_diagnostics"},
		"browser_type":              {"browser_snapshot", "browser_fill", "browser_diagnostics"},
		"browser_press":             {"browser_snapshot", "browser_read", "browser_diagnostics"},
		"browser_eval":              {"browser_eval_async", "browser_diagnostics", "browser_read"},
		"browser_eval_async":        {"browser_diagnostics", "browser_read", "browser_snapshot"},
		"browser_diagnostics":       {"focusa_browser_diagnostics_intake", "focusa_evidence_capture", "browser_read", "browser_snapshot", "browser_diagnostics_clear", "browser_close"},
		"browser_diagnostics_clear": {"browser_diagnostics"},
		"browser_screenshot":        {"browser_diagnostics", "frame_render", "browser_read"},
		"browser_scroll":            {"browser_read", "browser_snapshot", "browser_screenshot"},
		"browser_hover":             {"browser_snapshot", "browser_screenshot", "browser_diagnostics"},
		"browser_dom":               {"browser_snapshot", "browser_read"},
		"browser_navigate":          {"browser_read", "browser_snapshot", "browser_diagnostics"},
		"browser_resize":            {"browser_screenshot", "browser_snapshot", "browser_diagnostics"},
		"browser_css":               {"browser_screenshot", "browser_diagnostics"},
		"browser_wait":              {"browser_snapshot", "browser_read", "browser_diagnostics"},
		"browser_select":            {"browser_snapshot", "browser_diagnostics"},
		"browser_back":              {"browser_read", "browser_snapshot", "browser_diagnostics"},
		"browser_forward":           {"browser_read", "browser_snapshot", "browser_diagnostics"},
		"browser_text":              {"browser_read", "browser_snapshot"},
		"browser_cookies":           {"browser_diagnostics", "browser_read"},
		"browser_close":             {"browser_open"},
		"frame_catalog":             {"frame_render", "screenshot", "browser_screenshot"},
		"frame_render":              {"frame_catalog", "screenshot", "browser_screenshot"},
		"screenshot":                {"focusa_evidence_capture", "frame_render", "browser_open", "browser_diagnostics"},
	}
}

func workflowHints(name string) []string {
	if name == "uiai_health" || name == "uiai_status" {
		return []string{"Use for readiness checks before long workflows", "Pair with uiai_tool_graph for route planning", "Use browser_diagnostics for session-specific failures"}
	}
	if name == "critique_models" || name == "critique_dimensions" {
		return []string{"Read-only critique metadata", "Use before paid critique calls", "Pair with uiai_status when provider readiness is unclear"}
	}
	if name == "browser_search" {
		return []string{"Use provider-neutral search for discovery", "Open a selected result with browser_open", "Use browser_read for page text", "Use browser_diagnostics on navigation failures"}
	}
	if strings.HasPrefix(name, "browser_") {
		return []string{"Open or reuse a session", "Prefer browser_read for text and browser_snapshot for actions", "Use browser_diagnostics after failures", "If focusa_scope is present, ingest diagnostics/evidence in Focusa", "Close sessions when done"}
	}
	if strings.HasPrefix(name, "frame_") || name == "screenshot" {
		return []string{"Use one-shot screenshot for simple captures", "Use frame_catalog before frame_render", "Use sessions for multi-step browsing"}
	}
	return []string{"Use uiai_agent_card and uiai_tool_search for low-context routing"}
}

func enrichToolRelationships(tools []map[string]any) []map[string]any {
	relations := toolRelations()
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if related, ok := relations[name]; ok {
			tool["related_tools"] = related
		}
		tool["workflow_hints"] = workflowHints(name)
	}
	return tools
}

// openAITools returns tool definitions in OpenAI function calling format.
func openAITools() []map[string]any {
	tools := []map[string]any{
		{
			"name":        "uiai_agent_card",
			"description": "Return a compact UIAI Engine agent card/bootstrap card for local/remote agents: discovery endpoints, health/metrics links, recommended browser workflows, diagnostics rules, and search hints. Prefer this before loading full tool schemas.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "uiai_tool_search",
			"description": "Search UIAI tools by keyword without loading all schemas. Useful queries: screenshot, snapshot, read, extract, click, fill, eval_async, diagnostics, console, network, error, failed_request, blank page, visual failure.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q": map[string]string{"type": "string", "description": "Keyword or phrase to search"},
				},
				"required": []string{"q"},
			},
		},

		{
			"name":        "uiai_tool_graph",
			"description": "Return UIAI tool relationship graph, workflow routes, and Focusa integration metadata. Use this when choosing adjacent tools or chaining UIAI evidence into Focusa.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "uiai_health",
			"description": "Return UIAI browser/vision health and readiness. Use before long browser workflows or diagnosing pool pressure.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "uiai_status",
			"description": "Return UIAI engine runtime status and service metadata.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "critique_models",
			"description": "List supported critique models/providers. Read-only metadata; use before paid critique calls.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "critique_dimensions",
			"description": "List UI critique scoring dimensions. Read-only metadata useful for agents explaining or preparing critique workflows.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "browser_search",
			"description": "Provider-neutral web search for browser agents. Returns result titles, snippets, and source URLs; open selected URLs with browser_open, then browser_read. Brave is the default provider but not baked into browser semantics.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]string{"type": "string", "description": "Search query"},
					"provider": map[string]any{"type": "string", "description": "Search provider id; default brave", "default": "brave"},
					"limit":    map[string]any{"type": "integer", "description": "Result limit, max 20", "default": 5},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "browser_open",
			"description": "Open a persistent browser session on a URL. Retries transient page startup/navigation flakiness. Returns session_id and initial screenshot. For browser errors, console logs, JS exceptions, failed requests/failed_request, API failures, CORS, or network debugging, call browser_diagnostics next.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url":    map[string]string{"type": "string", "description": "URL to open"},
					"width":  map[string]any{"type": "integer", "description": "Viewport width", "default": 1280},
					"height": map[string]any{"type": "integer", "description": "Viewport height", "default": 800},
					"focusa_scope": map[string]any{
						"type":        "object",
						"description": "Optional Focusa scope object echoed through diagnostics/evidence",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "browser_screenshot",
			"description": "Instant re-screenshot of current page state (~30ms). No navigation — captures what's visible now. If the page looks broken or blank, call browser_diagnostics for console/network evidence.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID from browser_open"},
					"fullPage":   map[string]any{"type": "boolean", "description": "Capture entire scrollable page", "default": false},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_scroll",
			"description": "Scroll the page and screenshot. Use deltaY (positive=down, negative=up) for relative, or y for absolute scroll position.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"deltaY":     map[string]any{"type": "integer", "description": "Pixels to scroll (positive=down)", "default": 600},
					"deltaX":     map[string]any{"type": "integer", "description": "Horizontal scroll pixels"},
					"y":          map[string]any{"type": "integer", "description": "Absolute scroll Y position"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_click",
			"description": "Click an element. Accepts CSS selector OR @ref from browser_snapshot (e.g. \"@e3\"). Uses bounded retry/backoff to reduce late-render selector flakiness. Returns screenshot. After failed clicks, unexpected UI, or navigation/API errors, call browser_diagnostics.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref (e.g. \"@e3\")"},
				},
				"required": []string{"session_id", "selector"},
			},
		},
		{
			"name":        "browser_hover",
			"description": "Hover over an element. Accepts CSS selector OR @ref from browser_snapshot. Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref"},
				},
				"required": []string{"session_id", "selector"},
			},
		},
		{
			"name":        "browser_type",
			"description": "Type text into an input. Accepts CSS selector OR @ref from browser_snapshot. Clears existing. Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref"},
					"text":       map[string]string{"type": "string", "description": "Text to type"},
				},
				"required": []string{"session_id", "selector", "text"},
			},
		},
		{
			"name":        "browser_eval",
			"description": "Execute short synchronous JavaScript on the page. Returns result + screenshot. Use 'return' for output. Avoid long async Promises here; use browser_eval_async for bounded awaits, or split into direct browser actions/clicks. For console, JS exception, or network logs, use browser_diagnostics.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"js":         map[string]string{"type": "string", "description": "Short JavaScript to execute (use 'return' for output; avoid long async Promises)"},
				},
				"required": []string{"session_id", "js"},
			},
		},
		{
			"name":        "browser_eval_async",
			"description": "Execute bounded async JavaScript and await the result with timeout_ms (max 15000). Use for small awaited DOM/network checks only; for long UI workflows prefer browser_snapshot + direct click/type/wait actions to avoid Promise collection flake.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"js":         map[string]string{"type": "string", "description": "Async-capable JavaScript body; use return/await inside"},
					"timeout_ms": map[string]any{"type": "integer", "description": "Bounded async timeout, max 15000", "default": 5000},
				},
				"required": []string{"session_id", "js"},
			},
		},
		{
			"name":        "browser_diagnostics",
			"description": "DEVTOOLS DEBUG TOOL: inspect browser console logs/errors/warnings, JS exceptions/page errors, network requests, failed requests/failed_request, HTTP 4xx/5xx, CORS/API failures, retry/flakiness clues, long async eval issues, blank page, broken page, and visual failure clues. Call during browser troubleshooting after browser_open, browser_click, browser_eval, browser_eval_async, browser_wait, or any visual failure. No screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id":  map[string]string{"type": "string", "description": "Session ID"},
					"limit":       map[string]any{"type": "integer", "description": "Max events per category", "default": 100},
					"level":       map[string]string{"type": "string", "description": "Console level filter: all, error, warning, info"},
					"failed_only": map[string]any{"type": "boolean", "description": "Return only failed network requests in network list", "default": false},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_diagnostics_clear",
			"description": "Clear diagnostic buffers for a browser session.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_snapshot",
			"description": "Get accessibility tree with @ref selectors. PREFERRED over browser_dom. Returns a text tree like: '- link \"Sign In\" [ref=e3]'. Use refs in click/type/hover: {\"selector\": \"@e3\"}. Options: interactive (only buttons/links/inputs), compact (remove empty nodes), max_depth.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id":  map[string]string{"type": "string", "description": "Session ID"},
					"interactive": map[string]string{"type": "boolean", "description": "Only show interactive elements (default: false)"},
					"compact":     map[string]string{"type": "boolean", "description": "Remove empty structural nodes (default: false)"},
					"max_depth":   map[string]string{"type": "integer", "description": "Max tree depth (default: unlimited)"},
					"selector":    map[string]string{"type": "string", "description": "Scope to CSS selector (default: body)"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_dom",
			"description": "Get structured DOM info: headings, links, buttons, forms, interactive elements. Legacy — prefer browser_snapshot for @ref support.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_navigate",
			"description": "Navigate to a new URL in the same session. Returns screenshot of new page.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"url":        map[string]string{"type": "string", "description": "URL to navigate to"},
				},
				"required": []string{"session_id", "url"},
			},
		},
		{
			"name":        "browser_resize",
			"description": "Resize viewport to test responsive design. Common: mobile 375x812, tablet 768x1024, desktop 1440x900. Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"width":      map[string]string{"type": "integer", "description": "New viewport width"},
					"height":     map[string]string{"type": "integer", "description": "New viewport height"},
				},
				"required": []string{"session_id", "width", "height"},
			},
		},
		{
			"name":        "browser_css",
			"description": "Inject CSS to test visual changes live. Replaces any previous injection. Returns screenshot showing the result.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"css":        map[string]string{"type": "string", "description": "CSS rules to inject"},
				},
				"required": []string{"session_id", "css"},
			},
		},
		{
			"name":        "browser_wait",
			"description": "Wait for a CSS selector to appear on the page (e.g., lazy-loaded content). Returns screenshot when found.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector to wait for"},
					"timeout_ms": map[string]any{"type": "integer", "description": "Max wait time in ms", "default": 5000},
				},
				"required": []string{"session_id", "selector"},
			},
		},
		{
			"name":        "browser_fill",
			"description": "Clear an input completely and type new text. More reliable than browser_type for replacing values. Accepts @ref.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref"},
					"text":       map[string]string{"type": "string", "description": "Text to fill"},
				},
				"required": []string{"session_id", "selector", "text"},
			},
		},
		{
			"name":        "browser_select",
			"description": "Choose a dropdown option by value or visible text. Accepts @ref for the select element.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref of <select> element"},
					"values":     map[string]any{"type": "array", "items": map[string]string{"type": "string"}, "description": "Option value(s) or text to select"},
				},
				"required": []string{"session_id", "selector", "values"},
			},
		},
		{
			"name":        "browser_press",
			"description": "Press a keyboard key: Enter, Tab, Escape, ArrowDown, ArrowUp, Backspace, Delete, Space, Home, End, PageUp, PageDown.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"key":        map[string]string{"type": "string", "description": "Key name (Enter, Tab, Escape, etc)"},
				},
				"required": []string{"session_id", "key"},
			},
		},
		{
			"name":        "browser_back",
			"description": "Navigate browser history back. Returns screenshot of previous page.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_forward",
			"description": "Navigate browser history forward. Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_text",
			"description": "Get text content of an element by CSS selector or @ref. Returns text only, no screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector or @ref"},
				},
				"required": []string{"session_id", "selector"},
			},
		},

		{
			"name":        "browser_read",
			"description": "Read compact page text for web surfing without a screenshot. Extracts main/article/body text, headings, optional links, and supports selector or @ref plus max_chars. Prefer this after browser_open/navigate when the agent needs page content, not pixels.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id":    map[string]string{"type": "string", "description": "Session ID"},
					"selector":      map[string]string{"type": "string", "description": "Optional CSS selector or @ref region to read"},
					"max_chars":     map[string]any{"type": "integer", "description": "Max text characters, capped at 30000", "default": 8000},
					"include_links": map[string]any{"type": "boolean", "description": "Include up to 40 visible links", "default": false},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_cookies",
			"description": "Get, set, or clear browser cookies. Actions: get (list all or by name), set (name+value), clear (all or by name).",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"action":     map[string]string{"type": "string", "description": "get, set, or clear (default: get)"},
					"name":       map[string]string{"type": "string", "description": "Cookie name (for get-by-name, set, clear-by-name)"},
					"value":      map[string]string{"type": "string", "description": "Cookie value (for set)"},
					"domain":     map[string]string{"type": "string", "description": "Cookie domain (for set, default: current page domain)"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "browser_close",
			"description": "Close a browser session and free resources. Always call when done browsing.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			"name":        "frame_catalog",
			"description": "List available device frames sourced from approved GitHub packs (id, source, safe area, output size).",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "frame_render",
			"description": "Render a screenshot into a selected device frame. Use frame_catalog first to find frameId.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"frameId":     map[string]string{"type": "string", "description": "Frame ID from frame_catalog"},
					"imageBase64": map[string]string{"type": "string", "description": "Source screenshot base64"},
					"fit":         map[string]any{"type": "string", "description": "cover or contain", "default": "cover"},
					"format":      map[string]any{"type": "string", "description": "png or jpeg", "default": "png"},
					"quality":     map[string]any{"type": "integer", "description": "JPEG quality 1-100", "default": 90},
					"scale":       map[string]any{"type": "integer", "description": "Output scale multiplier", "default": 1},
				},
				"required": []string{"frameId", "imageBase64"},
			},
		},
		{
			"name":        "screenshot",
			"description": "One-shot screenshot: navigate, capture, forget. For single checks when you don't need a persistent session.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url":      map[string]string{"type": "string", "description": "URL to screenshot"},
					"width":    map[string]any{"type": "integer", "description": "Viewport width", "default": 1280},
					"height":   map[string]any{"type": "integer", "description": "Viewport height", "default": 800},
					"format":   map[string]any{"type": "string", "description": "Image format", "default": "jpeg"},
					"quality":  map[string]any{"type": "integer", "description": "JPEG quality 1-100", "default": 80},
					"fullPage": map[string]any{"type": "boolean", "description": "Full page capture", "default": false},
				},
				"required": []string{"url"},
			},
		},
	}
	return enrichToolRelationships(tools)
}

// mcpTools returns tool definitions in MCP format.
func mcpTools() map[string]any {
	tools := make([]map[string]any, 0)
	for _, t := range openAITools() {
		tools = append(tools, map[string]any{
			"name":           t["name"],
			"description":    t["description"],
			"inputSchema":    t["parameters"],
			"related_tools":  t["related_tools"],
			"workflow_hints": t["workflow_hints"],
		})
	}
	return map[string]any{"tools": tools}
}
