package vision

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/proto"
	jsongson "github.com/ysmood/gson"
)

func TestStructuredConsoleCredentialRedactionBeforePreview(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_CONSOLE_CONTAINER_0123456789"
	for _, raw := range []string{
		`{"password":["` + canary + `"],"echo":"` + canary + `"}`,
		`{"echo":"` + canary + `","padding":"` + strings.Repeat("x", 600) + `","password":"` + canary + `"}`,
	} {
		sess := &Session{ID: "synthetic-console", diagnostics: newDiagnosticsRecorder()}
		sess.recordConsole(&proto.RuntimeConsoleAPICalled{Type: "log", Args: []*proto.RuntimeRemoteObject{{Type: "string", Value: jsongson.New(raw)}, {Type: "string", Value: jsongson.New(canary)}}})
		encoded, err := json.Marshal(sess.diagnostics.console)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic JSON credential escaped persisted console preview")
		}
		for _, arg := range sess.diagnostics.console[0].ArgsPreview {
			if len(arg) > 503 {
				t.Fatal("console preview budget changed")
			}
		}
	}
	sess := &Session{ID: "ordinary-console", diagnostics: newDiagnosticsRecorder()}
	sess.recordConsole(&proto.RuntimeConsoleAPICalled{Type: "log", Args: []*proto.RuntimeRemoteObject{{Type: "string", Value: jsongson.New("ordinary console message")}}})
	if sess.diagnostics.console[0].Text != "ordinary console message" {
		t.Fatal("ordinary console text changed")
	}
}
