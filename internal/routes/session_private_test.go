package routes

import (
	"os"
	"strings"
	"testing"
)

func TestSessionOpenDoesNotMintPublicFPVShare(t *testing.T) {
	source, err := os.ReadFile("session.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	openEnd := strings.Index(text, "// Session-scoped routes")
	if openEnd < 0 {
		t.Fatal("session-open boundary not found")
	}
	openHandler := text[:openEnd]
	if strings.Contains(openHandler, "fpvCreateShare") || strings.Contains(openHandler, `"fpv_share"`) {
		t.Fatal("session creation still mints or returns a public FPV capability")
	}
}
