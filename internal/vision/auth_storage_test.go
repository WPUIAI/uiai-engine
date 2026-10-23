package vision

import (
	"encoding/json"
	"testing"
)

func TestParseAuthStorageShapes(t *testing.T) {
	flat, shape := parseAuthStorage(json.RawMessage(`{"wb_token":"abc","theme":"dark"}`))
	if shape != "flat" || flat["wb_token"] != "abc" || len(flat) != 2 {
		t.Fatalf("flat: shape=%s map=%v", shape, flat)
	}
	origin, shape := parseAuthStorage(json.RawMessage(`{"https://wins.wirebot.chat":{"wb_token":"abc"},"https://x.test":{"a":"1"}}`), "https://wins.wirebot.chat/path")
	if shape != "origin-keyed" || origin["wb_token"] != "abc" || len(origin) != 1 {
		t.Fatalf("origin-keyed: shape=%s map=%v", shape, origin)
	}
	if _, shape := parseAuthStorage(json.RawMessage(`[["k","v"]]`)); shape != "unparseable" {
		t.Fatalf("array: shape=%s, want unparseable", shape)
	}
	if _, shape := parseAuthStorage(json.RawMessage(`42`)); shape != "unparseable" {
		t.Fatalf("number: shape=%s, want unparseable", shape)
	}
	if _, shape := parseAuthStorage(json.RawMessage(`null`)); shape != "empty" {
		t.Fatalf("null: shape=%s, want empty", shape)
	}
}
