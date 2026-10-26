package ai

import (
	"encoding/json"
	"io"
	"strings"
)

// decodeProviderResponse never publishes a partial body or retains private
// reader/parser causes. Response-size admission remains a separate policy.
func decodeProviderResponse(provider string, reader io.Reader, target any) error {
	body, err := io.ReadAll(reader)
	defer redactedString(body).wipe()
	if err != nil {
		return providerIOFailure(provider, "response read", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return providerPhaseFailure(provider, "response decoding")
	}
	return nil
}

// Required text is checked without rewriting successful providers' whitespace.
func providerResponseText(provider, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", providerPhaseFailure(provider, "response content validation")
	}
	return text, nil
}
