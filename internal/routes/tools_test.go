package routes

import "testing"

func TestRankedToolSearchPrefersExactNameOverDescription(t *testing.T) {
	matches := rankedToolSearch(openAITools(), "eval_async")
	if len(matches) == 0 {
		t.Fatal("expected eval_async search matches")
	}
	if got := matches[0]["name"]; got != "browser_eval_async" {
		t.Fatalf("expected browser_eval_async first, got %v", got)
	}
}

func TestRankedToolSearchKeepsDiagnosticsDiscoverable(t *testing.T) {
	matches := rankedToolSearch(openAITools(), "diagnostics")
	if len(matches) == 0 {
		t.Fatal("expected diagnostics search matches")
	}
	found := false
	for _, match := range matches {
		if match["name"] == "browser_diagnostics" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected browser_diagnostics in diagnostics search results")
	}
}

func TestAgentBootstrapCardOrientsLocalAndRemoteAgents(t *testing.T) {
	card := agentBootstrapCard()
	if got := card["service"]; got != "uiai-engine" {
		t.Fatalf("expected service uiai-engine, got %v", got)
	}
	discovery, ok := card["discovery"].(map[string]string)
	if !ok {
		t.Fatal("expected string discovery map")
	}
	for _, key := range []string{"agent_card", "search", "openai", "mcp", "health", "metrics"} {
		if discovery[key] == "" {
			t.Fatalf("expected discovery key %s", key)
		}
	}
	hints, ok := card["search_hints"].([]string)
	if !ok {
		t.Fatal("expected search_hints slice")
	}
	foundDiagnostics := false
	for _, hint := range hints {
		if hint == "diagnostics" {
			foundDiagnostics = true
			break
		}
	}
	if !foundDiagnostics {
		t.Fatal("expected diagnostics search hint")
	}
}

func TestUtilityToolsExposeAgentCardAndSearch(t *testing.T) {
	tools := openAITools()
	found := map[string]bool{}
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "uiai_agent_card" || name == "uiai_tool_search" {
			found[name] = true
		}
	}
	for _, name := range []string{"uiai_agent_card", "uiai_tool_search"} {
		if !found[name] {
			t.Fatalf("expected %s in OpenAI tool definitions", name)
		}
	}

	matches := rankedToolSearch(tools, "agent card")
	if len(matches) == 0 || matches[0]["name"] != "uiai_agent_card" {
		t.Fatalf("expected agent card search to prefer uiai_agent_card, got %#v", matches)
	}
}

func TestBrowserReadToolIsDiscoverableForWebSurfing(t *testing.T) {
	tools := openAITools()
	matches := rankedToolSearch(tools, "read")
	if len(matches) == 0 || matches[0]["name"] != "browser_read" {
		t.Fatalf("expected read search to prefer browser_read, got %#v", matches)
	}
	matches = rankedToolSearch(tools, "extract")
	found := false
	for _, match := range matches {
		if match["name"] == "browser_read" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected extract search to include browser_read")
	}
}

func TestToolGraphIncludesFocusaAndRelations(t *testing.T) {
	graph := toolGraph()
	if graph["schema"] != "uiai.tool_graph.v1" {
		t.Fatalf("unexpected graph schema: %v", graph["schema"])
	}
	if _, ok := graph["focusa_integration"].(map[string]any); !ok {
		t.Fatal("expected focusa_integration metadata")
	}
	relations := toolRelations()
	if !containsString(relations["browser_diagnostics"], "focusa_browser_diagnostics_intake") {
		t.Fatalf("expected browser_diagnostics to route to Focusa intake: %#v", relations["browser_diagnostics"])
	}
	tools := openAITools()
	foundGraph := false
	for _, tool := range tools {
		if tool["name"] == "uiai_tool_graph" {
			foundGraph = true
		}
		if tool["name"] == "browser_open" && !containsString(tool["related_tools"].([]string), "focusa_browser_diagnostics_intake") {
			t.Fatalf("expected browser_open related_tools to include Focusa intake: %#v", tool["related_tools"])
		}
	}
	if !foundGraph {
		t.Fatal("expected uiai_tool_graph in tool list")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
