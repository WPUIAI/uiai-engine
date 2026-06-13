package vision

import "testing"

func TestParseSelectorHelper(t *testing.T) {
	kind, value, role := parseSelectorHelper("text=Learn more")
	if kind != "text" || value != "Learn more" || role != "" {
		t.Fatalf("unexpected text= parse: kind=%q value=%q role=%q", kind, value, role)
	}
	kind, value, role = parseSelectorHelper("text/Submit")
	if kind != "text" || value != "Submit" || role != "" {
		t.Fatalf("unexpected text/ parse: kind=%q value=%q role=%q", kind, value, role)
	}
	kind, value, role = parseSelectorHelper("role=button;name=Save")
	if kind != "role" || value != "Save" || role != "button" {
		t.Fatalf("unexpected role parse: kind=%q value=%q role=%q", kind, value, role)
	}
	kind, value, role = parseSelectorHelper(".button")
	if kind != "" || value != "" || role != "" {
		t.Fatalf("css selector should not be helper: kind=%q value=%q role=%q", kind, value, role)
	}
}
