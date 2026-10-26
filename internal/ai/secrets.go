package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

// redactedString owns mutable credential bytes. Cache access needs keysMu;
// transport boundaries clone under that lock and wipe their detached copy.
// This cannot erase immutable strings, HTTP headers, env/config or runtime copies.
type redactedString []byte

func (redactedString) String() string   { return "[secret]" }
func (redactedString) GoString() string { return "[secret]" }
func (redactedString) MarshalJSON() ([]byte, error) {
	return json.Marshal("[secret]")
}

// Formatter covers numeric/unsupported verbs too, which can otherwise inspect
// the underlying byte slice instead of invoking String or GoString.
func (redactedString) Format(state fmt.State, _ rune) { _, _ = io.WriteString(state, "[secret]") }
func (redactedString) MarshalText() ([]byte, error)   { return []byte("[secret]"), nil }

func (s redactedString) value() string         { return string(s) }
func (s redactedString) clone() redactedString { return append(redactedString(nil), s...) }
func (s redactedString) wipe()                 { clear(s); runtime.KeepAlive(s) }
