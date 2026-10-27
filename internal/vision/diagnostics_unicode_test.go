package vision

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-rod/rod/lib/proto"
	jsongson "github.com/ysmood/gson"
)

func TestDiagnosticBytePreviewPreservesUTF8(t *testing.T) {
	for _, tt := range []struct {
		input string
		limit int
		want  string
	}{
		{"abcdef", 3, "abc…"},
		{"abمرحبا", 3, "ab…"},
		{"A🙂B", 4, "A…"},
		{"A🙂B", 5, "A🙂…"},
		{"中英", 4, "中…"},
		{"🙂", 4, "🙂"},
		{"🙂", 1, "…"},
	} {
		got := truncateDiag(tt.input, tt.limit)
		if !utf8.ValidString(got) || got != tt.want || len(got) > tt.limit+len("…") {
			t.Fatalf("invalid or misbounded preview at byte limit%d", tt.limit)
		}
	}
}

func TestDiagnosticRecordersPersistValidUTF8Previews(t *testing.T) {
	input := "A" + strings.Repeat("🙂", 125)
	sess := &Session{ID: "unicode-preview", diagnostics: newDiagnosticsRecorder()}
	sess.recordConsole(&proto.RuntimeConsoleAPICalled{Type: "log", Args: []*proto.RuntimeRemoteObject{{Type: "string", Value: jsongson.New(input)}}})
	sess.recordException(&proto.RuntimeExceptionThrown{ExceptionDetails: &proto.RuntimeExceptionDetails{Exception: &proto.RuntimeRemoteObject{Type: "string", Value: jsongson.New(input)}}})
	console := sess.diagnostics.console[0]
	exception := sess.diagnostics.exceptions[0]
	for _, preview := range []string{console.Text, console.ArgsPreview[0], exception.Text} {
		if !utf8.ValidString(preview) || len(preview) > 503 {
			t.Fatal("recorder persisted a broken or enlarged UTF8 preview")
		}
	}
	encoded, err := json.Marshal([]any{console, exception})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "\\ufffd") || strings.Contains(string(encoded), "�") {
		t.Fatal("serialization introduced replacement characters")
	}
}
