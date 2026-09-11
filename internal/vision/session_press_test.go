package vision

import (
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/input"
)

// TestResolvePressPart covers the #225 keyboard extension: navigation keys,
// modifiers, single characters, and modifiers resolve without a browser.
func TestResolvePressPart(t *testing.T) {
	cases := []struct {
		part     string
		wantCode string
		wantMod  bool
		wantOK   bool
	}{
		{"Enter", "Enter", false, true},
		{"ArrowDown", "ArrowDown", false, true},
		{"shift", "ShiftLeft", true, true},
		{"CTRL", "ControlLeft", true, true},
		{"super", "MetaLeft", true, true},
		{"cmd", "MetaLeft", true, true},
		{"a", "KeyA", false, true},
		{"5", "Digit5", false, true},
		{"?", "Slash", false, true},
		{"", "", false, false},
		{"superkey", "", false, false},
	}
	for _, c := range cases {
		k, isMod, ok := resolvePressPart(c.part)
		if ok != c.wantOK {
			t.Fatalf("resolvePressPart(%q) ok=%v want %v", c.part, ok, c.wantOK)
		}
		if !ok {
			continue
		}
		if got := k.Info().Code; got != c.wantCode {
			t.Fatalf("resolvePressPart(%q) code=%q want %q", c.part, got, c.wantCode)
		}
		if isMod != c.wantMod {
			t.Fatalf("resolvePressPart(%q) isModifier=%v want %v", c.part, isMod, c.wantMod)
		}
	}
	// Modifiers must actually carry the CDP modifier bit (canvas remoting).
	if got := modifierKeys["super"].Modifier(); got != input.ModifierMeta {
		t.Fatalf("super modifier=%d want %d", got, input.ModifierMeta)
	}
	if got := modifierKeys["ctrl"].Modifier(); got != input.ModifierControl {
		t.Fatalf("ctrl modifier=%d want %d", got, input.ModifierControl)
	}
}

// TestPressRejectsUnknownKey documents that unknown keys are rejected and the
// error guidance lists the supported forms (feeds the 400 mapping in
// internal/routes, #226).
func TestPressRejectsUnknownKey(t *testing.T) {
	if _, _, ok := resolvePressPart("definitely-not-a-key"); ok {
		t.Fatal("expected unknown key to be rejected")
	}
	if pressSupportedKeys == "" || !strings.Contains(pressSupportedKeys, "ctrl+shift+t") {
		t.Fatalf("supported-keys guidance missing: %q", pressSupportedKeys)
	}
}
