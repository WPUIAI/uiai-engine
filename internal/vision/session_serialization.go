package vision

import (
	"encoding/json"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
)

// Navigation state stays intact; only its public JSON representation is redacted.
func (s *Session) MarshalJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type plain Session
	return json.Marshal(struct {
		URL string `json:"url"`
		*plain
	}{URL: focusapacket.SanitizeURL(s.URL), plain: (*plain)(s)})
}

func (s SnapResult) MarshalJSON() ([]byte, error) {
	type plain SnapResult
	copy := plain(s)
	copy.URL = focusapacket.SanitizeURL(copy.URL)
	return json.Marshal(copy)
}
