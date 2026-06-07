package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

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

func TestRankedToolSearchHandlesMultiTokenVisualFailureDiagnostics(t *testing.T) {
	matches := rankedToolSearch(openAITools(), "visual failure diagnostics")
	if len(matches) == 0 {
		t.Fatal("expected multi-token visual failure diagnostics search matches")
	}
	if got := matches[0]["name"]; got != "browser_diagnostics" {
		t.Fatalf("expected browser_diagnostics first, got %v", got)
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

func TestMountToolsDiscoveryServesDocsEndpoint(t *testing.T) {
	r := chi.NewRouter()
	MountToolsDiscovery(r, nil)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["schema"] != "uiai.agent_docs.v1" {
		t.Fatalf("unexpected schema: %#v", body)
	}
}

func TestAgentDocsExamplesShape(t *testing.T) {
	docs := agentDocsExamples()
	if docs["schema"] != "uiai.agent_docs.v1" {
		t.Fatalf("unexpected docs schema: %v", docs["schema"])
	}
	links, ok := docs["doc_links"].([]map[string]string)
	if !ok || len(links) == 0 {
		t.Fatalf("expected doc links, got %#v", docs["doc_links"])
	}
	foundRemoteAuth := false
	for _, link := range links {
		if link["path"] == "docs/REMOTE_AUTH_EXAMPLES.md" {
			foundRemoteAuth = true
		}
	}
	if !foundRemoteAuth {
		t.Fatalf("expected remote auth doc link in %#v", links)
	}
	examples, ok := docs["quick_examples"].([]map[string]any)
	if !ok || len(examples) < 3 {
		t.Fatalf("expected quick examples, got %#v", docs["quick_examples"])
	}
	auth := docs["auth_classification"].(map[string]string)
	if auth["endpoint"] != "/api/tools/docs" || auth["mode"] != "public" {
		t.Fatalf("unexpected auth classification: %#v", auth)
	}
	tools := docs["relevant_tools"].([]string)
	if !containsString(tools, "uiai_focusa_packet_compose") {
		t.Fatalf("expected packet composer in relevant tools: %#v", tools)
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
	focusa, ok := graph["focusa_integration"].(map[string]any)
	if !ok {
		t.Fatal("expected focusa_integration metadata")
	}
	evidenceRefs, ok := focusa["evidence_refs"].([]string)
	if !ok || !containsString(evidenceRefs, "uiai-search:<provider>:<query-hash>:<rank>") {
		t.Fatalf("expected search evidence ref in Focusa graph metadata: %#v", focusa["evidence_refs"])
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

func TestToolGraphIncludesFocusaPacketWorkflow(t *testing.T) {
	graph := toolGraph()
	focusa := graph["focusa_integration"].(map[string]any)
	if focusa["packet_schema"] != "uiai.focusa_research_diagnostics_packet.v1" {
		t.Fatalf("missing packet schema: %#v", focusa)
	}
	surfaces := focusa["packet_metadata_surfaces"].([]string)
	for _, want := range []string{"search", "browser_read", "browser_snapshot", "browser_diagnostics", "structured_errors", "screenshot", "share"} {
		if !containsString(surfaces, want) {
			t.Fatalf("missing packet surface %s in %#v", want, surfaces)
		}
	}
	workflows := graph["workflows"].([]map[string]any)
	found := false
	for _, workflow := range workflows {
		if workflow["name"] == "focusa_research_packet" {
			found = true
			if workflow["schema"] != "uiai.focusa_research_diagnostics_packet.v1" {
				t.Fatalf("workflow schema mismatch: %#v", workflow)
			}
			steps := workflow["steps"].([]string)
			for _, want := range []string{"uiai_focusa_packet_build", "focusa_evidence_capture or focusa_browser_diagnostics_intake"} {
				if !containsString(steps, want) {
					t.Fatalf("missing workflow step %s in %#v", want, steps)
				}
			}
		}
	}
	if !found {
		t.Fatal("missing focusa_research_packet workflow")
	}
	relations := toolRelations()
	if !containsString(relations["browser_read"], "uiai_focusa_packet_build") || !containsString(relations["browser_diagnostics"], "uiai_focusa_packet_build") {
		t.Fatalf("missing packet builder relations: %#v", relations)
	}
}
