package routes

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFPVAuditDoesNotExposeCapability(t *testing.T) {
	const token = "synthetic-FPV-capability-canary-should-never-be-logged"
	input := map[string]any{"token": token, "nested": map[string]any{"url": "https://example.test/m/" + token}, "x": 12}
	logged := fpvRecordAuditEvent(token, "safe-session", fpvAuditLog{
		TS: time.Now().UTC(), Action: "message", Message: "link " + token, Selector: "[data-value='" + token + "']", Key: token, Error: "failed " + token, Meta: input,
	})
	data, err := json.Marshal(logged)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), token) {
		t.Fatal("audit record discloses the usable FPV capability")
	}
	if logged.Meta["token"] != "[REDACTED]" || !strings.HasPrefix(logged.Meta["share_ref"].(string), "fpv-share:sha256:") {
		t.Fatal("audit lacks redacted token and non-authorizing correlation reference")
	}
	if input["token"] != token {
		t.Fatal("audit redaction mutated caller metadata")
	}
	input["nested"].(map[string]any)["url"] = token
	events, _ := fpvAuditEventsSince(logged.Seq-1, 1)
	data, err = json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), token) {
		t.Fatal("later metadata mutation leaked into stored audit")
	}
}

func TestFPVAuditHTTPProjectionWithholdsCapability(t *testing.T) {
	const token = "synthetic-FPV-HTTP-capability-canary-not-an-actual-share"
	logged := fpvRecordAuditEvent(token, "safe-session", fpvAuditLog{TS: time.Now().UTC(), Action: "message", Message: token})
	req := httptest.NewRequest("GET", "/events?limit=1", nil)
	w := httptest.NewRecorder()
	fpvAuditEventsHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("audit HTTP status=%d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, token) {
		t.Fatal("HTTP audit projection exposes a usable capability")
	}
	if !strings.Contains(body, logged.Meta["share_ref"].(string)) {
		t.Fatal("HTTP audit lost non-authorizing correlation")
	}
}

func TestFPVAuditWithholdsUnencodableOrOversizedMetadata(t *testing.T) {
	for _, metadata := range []map[string]any{
		{"unsupported": func() {}},
		{"oversized": strings.Repeat("x", 10001)},
	} {
		logged := fpvRecordAuditEvent("synthetic-capability", "safe-session", fpvAuditLog{Meta: metadata})
		if logged.Meta["metadata_withheld"] != true {
			t.Fatal("unsafe metadata was not withheld")
		}
		if _, err := json.Marshal(logged); err != nil {
			t.Fatalf("safe audit cannot be serialized: %v", err)
		}
	}
}
