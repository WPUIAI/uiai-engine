package jsonutil

import (
	"testing"
)

func TestRepair_ValidJSON(t *testing.T) {
	result, err := Repair(`{"score": 85, "summary": "Good design"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["score"].(float64) != 85 {
		t.Errorf("expected score 85, got %v", m["score"])
	}
}

func TestRepair_MarkdownFence(t *testing.T) {
	raw := "Here is the analysis:\n```json\n{\"score\": 72}\n```\nThat's my review."
	result, err := Repair(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["score"].(float64) != 72 {
		t.Errorf("expected score 72, got %v", m["score"])
	}
}

func TestRepair_TrailingComma(t *testing.T) {
	raw := `{"items": ["a", "b", "c",], "count": 3,}`
	result, err := Repair(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["count"].(float64) != 3 {
		t.Errorf("expected count 3, got %v", m["count"])
	}
}

func TestRepair_BoundaryExtraction(t *testing.T) {
	raw := `Sure! Here is the JSON output:
{"score": 90, "summary": "Excellent"}
I hope this helps!`
	result, err := Repair(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["score"].(float64) != 90 {
		t.Errorf("expected score 90, got %v", m["score"])
	}
}

func TestRepair_ControlChars(t *testing.T) {
	raw := "{\"text\": \"hello\x00world\x08\"}"
	result, err := Repair(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["text"] != "helloworld" {
		t.Errorf("expected 'helloworld', got %v", m["text"])
	}
}

func TestRepair_UnescapedNewlines(t *testing.T) {
	raw := "{\"text\": \"line one\nline two\"}"
	result, err := Repair(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["text"] != "line one\nline two" {
		t.Errorf("expected escaped newline in string, got %v", m["text"])
	}
}

func TestRepairObject(t *testing.T) {
	m, err := RepairObject(`{"key": "value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["key"] != "value" {
		t.Errorf("expected 'value', got %v", m["key"])
	}
}

func TestRepairObject_RejectsArray(t *testing.T) {
	_, err := RepairObject(`[1, 2, 3]`)
	if err == nil {
		t.Fatal("expected error for array input")
	}
}

func TestRepairArray(t *testing.T) {
	a, err := RepairArray(`[1, 2, 3]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a) != 3 {
		t.Errorf("expected 3 items, got %d", len(a))
	}
}

func TestRepair_ComplexCritique(t *testing.T) {
	// Simulate real LLM critique output with multiple issues
	raw := "```json\n" + `{
  "score": 78,
  "summary": "Good layout but contrast issues",
  "findings": [
    {"dimension": "contrast", "score": 5, "finding": "White text on yellow\nbackground", "suggestion": "Use dark text"},
    {"dimension": "typography", "score": 8, "finding": "Clean hierarchy", "suggestion": "None needed",},
  ],
  "priority_fixes": ["Fix contrast", "Add CTA",]
}` + "\n```"

	m, err := RepairObject(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["score"].(float64) != 78 {
		t.Errorf("expected score 78, got %v", m["score"])
	}
	findings := m["findings"].([]any)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}
