package routes

import (
	"net/http"

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
			"note":   "Use GET /api/tools/search?q=screenshot to find specific tools without loading all definitions",
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
				"hint":  "Add ?q=keyword to search. Full definitions at /api/tools/openai or /api/tools/mcp",
			})
			return
		}

		// Search by substring in name or description
		matches := make([]map[string]any, 0)
		for _, t := range openAITools() {
			name := t["name"].(string)
			desc := t["description"].(string)
			if containsCI(name, q) || containsCI(desc, q) {
				matches = append(matches, t)
			}
		}
		writeJSON(w, 200, map[string]any{
			"query":   q,
			"tools":   matches,
			"count":   len(matches),
		})
	})
}

// containsCI is defined in intelligence.go (shared within routes package)

// openAITools returns tool definitions in OpenAI function calling format.
func openAITools() []map[string]any {
	return []map[string]any{
		{
			"name":        "browser_open",
			"description": "Open a persistent browser session on a URL. Returns session_id and initial screenshot. The page stays alive for scrolling, clicking, and re-screenshotting.",
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
			"description": "Instant re-screenshot of current page state (~30ms). No navigation — captures what's visible now.",
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
			"description": "Click an element by CSS selector and screenshot the result. Use browser_dom to find selectors.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector to click"},
				},
				"required": []string{"session_id", "selector"},
			},
		},
		{
			"name":        "browser_hover",
			"description": "Hover over an element to trigger hover states (dropdowns, tooltips). Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector to hover"},
				},
				"required": []string{"session_id", "selector"},
			},
		},
		{
			"name":        "browser_type",
			"description": "Type text into an input field (clears existing text first). Returns screenshot.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"selector":   map[string]string{"type": "string", "description": "CSS selector of input"},
					"text":       map[string]string{"type": "string", "description": "Text to type"},
				},
				"required": []string{"session_id", "selector", "text"},
			},
		},
		{
			"name":        "browser_eval",
			"description": "Execute JavaScript on the page. Returns the result value and a screenshot. Use 'return' for output.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]string{"type": "string", "description": "Session ID"},
					"js":         map[string]string{"type": "string", "description": "JavaScript to execute (use 'return' for output)"},
				},
				"required": []string{"session_id", "js"},
			},
		},
		{
			"name":        "browser_dom",
			"description": "Get structured DOM info: headings, links, buttons, images, forms, and interactive elements with CSS selectors. No screenshot — just data. Use to discover what you can click/type.",
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
