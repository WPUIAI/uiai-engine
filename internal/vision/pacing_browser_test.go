package vision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPacingBrowserCadenceSequenceAndDeadline(t *testing.T) {
	if os.Getenv("UIAI_PACING_TESTS") == "" {
		t.Skip("set UIAI_PACING_TESTS=1 for owned synthetic input fixture")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><input id="a"><input id="b"><input id="code"><button id="submit" onclick="window.fixtureDone=document.querySelector('#a').value==='alpha' && document.querySelector('#b').value==='beta' && document.querySelector('#code').value==='synthetic-code'">Submit</button><button id="counter" onclick="window.fixtureClicks=(window.fixtureClicks||0)+1">Counter</button></body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal("explicit pacing browser unavailable")
	}
	seed := int64(7)
	if err := session.SetPace(PaceConfig{Mode: "custom", MedianMS: 2, MinClickMS: 1, CPSMin: 200, CPSMax: 400, Seed: &seed}); err != nil {
		t.Fatal(err)
	}
	if _, err := session.TypeContext(context.Background(), "#a", "aé😀"); err != nil {
		t.Fatal(err)
	}
	value, err := session.page.Eval(`() => document.querySelector('#a').value==='aé😀'`)
	if err != nil || !value.Value.Bool() {
		t.Fatal("paced rune input changed content")
	}
	if session.Pacing().KeyWaits != 2 {
		t.Fatal("cadence did not operate between Unicode code points")
	}
	ops := []SequenceOperation{{Action: "fill", Selector: "#a", Text: "alpha"}, {Action: "type", Selector: "#b", Text: "beta"}, {Action: "type", Selector: "#code", Text: "synthetic-code"}, {Action: "click", Selector: "#submit"}}
	_, result, err := session.Sequence(context.Background(), ops, 4*time.Second)
	if err != nil || result.Completed != 4 || result.DurationMS >= 4000 {
		t.Fatalf("bounded synthetic sequence incomplete: completed=%d duration=%d err=%v", result.Completed, result.DurationMS, err)
	}
	value, err = session.page.Eval(`() => window.fixtureDone===true`)
	if err != nil || !value.Value.Bool() {
		t.Fatal("sequence failed external form predicate")
	}
	// A canceled request may not begin dispatch, even when an action profile exists.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, result, err = session.Sequence(ctx, []SequenceOperation{{Action: "click", Selector: "#counter"}}, time.Second)
	if err == nil || result.Attempted != 0 || result.EffectsPossible {
		t.Fatal("pre-canceled sequence attempted effects")
	}
	if err := session.SetPace(PaceConfig{Mode: "custom", MedianMS: 2000}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := session.ClickContext(ctx, "#counter"); err == nil {
		t.Fatal("deadline did not cancel pacing before click")
	}
	value, err = session.page.Eval(`() => !window.fixtureClicks`)
	if err != nil || !value.Value.Bool() {
		t.Fatal("expired pace dispatched input")
	}
	if err := session.SetPace(PaceConfig{Mode: "instant"}); err != nil {
		t.Fatal(err)
	}
	if session.Pacing() != nil {
		t.Fatal("instant reset kept profile/telemetry")
	}
	if _, err := session.Type("#a", "legacy"); err != nil {
		t.Fatal("instant input path changed")
	}
}

func TestSequenceAdmissionHasNoEffects(t *testing.T) {
	session := &Session{}
	for _, ops := range [][]SequenceOperation{nil, {{Action: "eval", Selector: "body"}}, {{Action: "click", Selector: ""}}, {{Action: "type", Selector: "@e1"}}} {
		_, result, err := session.Sequence(context.Background(), ops, time.Second)
		if err == nil || result.Attempted != 0 || result.EffectsPossible {
			t.Fatal("invalid sequence admitted effects")
		}
	}
	session.raiseAuthChallenge("credential_form", "https://example.test/login")
	_, result, err := session.Sequence(context.Background(), []SequenceOperation{{Action: "click", Selector: "button"}}, time.Second)
	if err == nil || result.Attempted != 0 {
		t.Fatal("sequence bypassed pending authentication")
	}
}
