package vision

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/proto"
	jsongson "github.com/ysmood/gson"
)

func TestExceptionStructuredCredentialsRedactedBeforePersistence(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_EXCEPTION_0123456789"
	for _, raw := range []string{
		`Uncaught {"password":"` + canary + `","echo":"` + canary + `"}`,
		`Error: {"password":["` + canary + `"],"echo":"` + canary + `"}\n at ordinaryFrame`,
		`Uncaught {"echo":"` + canary + `","padding":"` + strings.Repeat("x", 900) + `","password":"` + canary + `"}`,
	} {
		sess := &Session{ID: "synthetic-exception", diagnostics: newDiagnosticsRecorder()}
		sess.recordException(&proto.RuntimeExceptionThrown{ExceptionDetails: &proto.RuntimeExceptionDetails{
			Text: "Uncaught", Exception: &proto.RuntimeRemoteObject{Type: "string", Value: jsongson.New(raw)},
			StackTrace: &proto.RuntimeStackTrace{CallFrames: []*proto.RuntimeCallFrame{{FunctionName: canary, URL: "https://example.com/fixture"}}},
		}})
		encoded, err := json.Marshal(sess.diagnostics.exceptions)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic credential escaped persisted exception or stack")
		}
		event := sess.diagnostics.exceptions[0]
		if !strings.Contains(event.Text, "REDACTED_SECRET") {
			t.Fatal("exception lost redaction marker")
		}
		if len(event.Text) > 503 || len(event.StackPreview) > 1003 {
			t.Fatal("exception preview bounds changed")
		}
	}
	sess := &Session{ID: "ordinary-exception", diagnostics: newDiagnosticsRecorder()}
	sess.recordException(&proto.RuntimeExceptionThrown{ExceptionDetails: &proto.RuntimeExceptionDetails{Exception: &proto.RuntimeRemoteObject{Type: "string", Value: jsongson.New("TypeError: ordinary failure")}}})
	if sess.diagnostics.exceptions[0].Text != "TypeError: ordinary failure" {
		t.Fatal("ordinary exception text changed")
	}
}
