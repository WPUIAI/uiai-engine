package ai

import "encoding/json"

// redactedString keeps credential values usable only at explicit transport
// boundaries; formatting, JSON, and Go-syntax diagnostics never reveal them.
type redactedString string

func (redactedString) String() string   { return "[secret]" }
func (redactedString) GoString() string { return "[secret]" }
func (redactedString) MarshalJSON() ([]byte, error) {
	return json.Marshal("[secret]")
}

func (s redactedString) value() string { return string(s) }
