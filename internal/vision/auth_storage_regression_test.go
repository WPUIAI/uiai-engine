package vision

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-rod/rod"
)

func TestAuthStorageRejectsLossyAndAmbiguousShapes(t *testing.T) {
	for _, input := range []string{
		`{"key":null}`, `{"key":42}`, `{"key":true}`, `{"key":["secret"]}`,
		`{"https://target.test":{"key":42}}`,
		`{"https://target.test":{"key":null}}`,
		`{"https://target.test":{"key":"a"},"bad-origin":{"key":"b"}}`,
		`{"https://target.test/path":{"key":"a"}}`,
		`{"https://target.test":{"key":"a"},"https://target.test/":{"key":"b"}}`,
		`{"https://user:pass@target.test":{"key":"a"}}`,
	} {
		if _, shape := parseAuthStorage(json.RawMessage(input), "https://target.test"); shape != "unparseable" {
			t.Errorf("invalid shape was accepted: %s (%s)", input, shape)
		}
	}
}

func TestAuthStorageSelectsTargetOriginOnly(t *testing.T) {
	raw := json.RawMessage(`{"https://target.test/":{"key":"target"},"https://other.test":{"key":"other","foreign":"do-not-copy"}}`)
	got, shape := parseAuthStorage(raw, "https://target.test/path?x=1")
	if shape != "origin-keyed" || len(got) != 1 || got["key"] != "target" {
		t.Fatalf("wrong target origin: %#v %s", got, shape)
	}
	got, shape = parseAuthStorage(raw, "https://third.test/")
	if shape != "origin-keyed" || len(got) != 0 {
		t.Fatal("unrelated origins were merged into current page")
	}
}

func TestLoadAuthRejectsInvalidFieldsBeforeBrowserEffects(t *testing.T) {
	for _, field := range []string{"localStorage", "sessionStorage"} {
		s := &Session{page: &rod.Page{}}
		_, err := s.LoadAuth(json.RawMessage(`{"url":"https://target.test","` + field + `":{"secret":42}}`))
		if !errors.Is(err, ErrInvalidAuthState) || !strings.Contains(err.Error(), field) {
			t.Fatalf("%s: want field-specific invalid state, got %v", field, err)
		}
		if strings.Contains(err.Error(), "42") {
			t.Fatal("error exposed storage value")
		}
	}
	s := &Session{page: &rod.Page{}}
	_, err := s.LoadAuth(json.RawMessage(`{"url":"https://target.test","cookies":[null]}`))
	if !errors.Is(err, ErrInvalidAuthState) {
		t.Fatalf("null cookie was not rejected: %v", err)
	}
}
