package routes

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

// MountToolsDiscovery serves tool definitions for LLM integration.
// GET /api/tools           → all tool definitions (OpenAI format)
// GET /api/tools/openai    → OpenAI function calling format
// GET /api/tools/mcp       → MCP tool definitions
// GET /api/tools/search?q= → search tools by name/description (low-context discovery)
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
				"hint":  "Add ?q=keyword to search. Try q=screenshot, q=console, q=network, q=error, q=exception, q=devtools, q=retry, q=flakiness, or q=failed_request. Full definitions at /api/tools/openai or /api/tools/mcp",
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

// openAITools returns tool definitions in OpenAI function calling format.
func openAITools() []map[string]any {
	return []map[string]any{
		{
			"name":        "browser_open",
			"description": "Open a persistent browser session on a URL. Retries transient page startup/navigation flakiness. Returns session_id and initial screenshot. For browser errors, console logs, JS exceptions, failed requests/failed_request, API failures, CORS, or network debugging, call browser_diagnostics next.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url":    map[string]string{"type": "string", "description": "URL to open"},
					"width":  map[string]any{"type": "integer", "description": "Viewport width", "default": 1280},
					"height": map[string]any{"type": "integer", "description": "Viewport height", "default": 800},
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
}

// mcpTools returns tool definitions in MCP format.
func mcpTools() map[string]any {
	tools := make([]map[string]any, 0)
	for _, t := range openAITools() {
		tools = append(tools, map[string]any{
			"name":        t["name"],
			"description": t["description"],
			"inputSchema": t["parameters"],
		})
	}
	return map[string]any{"tools": tools}
}
