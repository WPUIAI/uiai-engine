package routes

import (
	"encoding/json"
	"strings"
	"testing"
)

type fakePage struct {
	texts   map[string]string
	typed   []string
	clicked []string
}

func (f *fakePage) Text(sel string) (string, error) { return f.texts[sel], nil }
func (f *fakePage) Click(sel string) error          { f.clicked = append(f.clicked, sel); return nil }
func (f *fakePage) Input(sel, v string) error       { f.typed = append(f.typed, sel+"="+v); return nil }
func (f *fakePage) Press(k string) error            { return nil }
func (f *fakePage) Navigate(u string) error         { return nil }

// C-010-01: extract returns typed data + confidence + missing fields.
func TestRunExtractConfidenceAndMissing(t *testing.T) {
	page := &fakePage{texts: map[string]string{"h1": "Hello", ".price": "$9"}}
	res := RunExtract(page, ExtractRequest{Schema: map[string]ExtractField{
		"title": {Selector: "h1"}, "price": {Selector: ".price"}, "missing_field": {Selector: ".nope"},
	}})
	if res.Data["title"] != "Hello" || res.Data["price"] != "$9" || res.Data["missing_field"] != "" {
		t.Fatalf("data wrong: %v", res.Data)
	}
	if len(res.Missing) != 1 || res.Missing[0] != "missing_field" {
		t.Fatalf("missing wrong: %v", res.Missing)
	}
	if res.Confidence < 0.66 || res.Confidence > 0.67 {
		t.Fatalf("confidence=%f want ~0.667", res.Confidence)
	}
	if res.Receipt["verb"] != "extract" {
		t.Fatal("receipt verb missing")
	}
}

// C-010-01: act redacts typed values (length only) and reports assert results.
func TestRunActRedactsValues(t *testing.T) {
	page := &fakePage{texts: map[string]string{"#status": "done"}}
	res := RunAct(page, ActRequest{
		SessionID: "s",
		Actions:   []ActStep{{Type: "type", Selector: "#pw", Value: "supersecret"}, {Type: "click", Selector: "#go"}},
		Asserts:   []AssertStep{{Selector: "#status", Contains: "done"}},
	})
	if !res.Ok {
		t.Fatalf("act should pass: %v", res)
	}
	raw, _ := jsonMarshal(res.Steps[0])
	if strings.Contains(raw, "supersecret") {
		t.Fatal("value leaked into steps")
	}
	if res.Steps[0]["value_len"] != len("supersecret") {
		t.Fatalf("value_len missing: %v", res.Steps[0])
	}
	if len(res.Asserts) != 1 || res.Asserts[0]["pass"] != true {
		t.Fatalf("asserts wrong: %v", res.Asserts)
	}
}

// C-010-03: charge enforces caps; pause blocks; resume allows.
func TestBudgetLifecycle(t *testing.T) {
	b := CreateBudget(BudgetLimits{TotalMS: 100, MaxPages: 2}, "test")
	if err := Charge(b.ID, 60, 1, 0); err != nil {
		t.Fatalf("charge1: %v", err)
	}
	if err := Charge(b.ID, 60, 0, 0); err == nil {
		t.Fatal("expected exceeded on ms cap")
	}
	SetPaused(b.ID, b.ResumeToken, true)
	if err := Charge(b.ID, 1, 0, 0); err != ErrBudgetPaused {
		t.Fatalf("want paused, got %v", err)
	}
	if _, ok := SetPaused(b.ID, "wrong-token", false); ok {
		t.Fatal("wrong token must fail")
	}
	if _, ok := SetPaused(b.ID, b.ResumeToken, false); !ok {
		t.Fatal("resume with correct token failed")
	}
	if err := Charge(b.ID, 10, 1, 0); err != nil {
		t.Fatalf("post-resume charge failed: %v", err)
	}
}

func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}
