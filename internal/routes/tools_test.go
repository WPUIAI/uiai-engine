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
