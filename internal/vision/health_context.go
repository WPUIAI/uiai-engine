package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-rod/rod/lib/proto"
)

// #147: public FPV sharing defaults to deny for health-origin sessions.
// A bounded in-page probe detects high-signal patient-portal/medical-record
// contexts; a failing probe reports an error so the share path can fail
// closed rather than publish an unverifiable session.

var healthContextMarkers = []string{
	"patient portal",
	"medical record",
	"health record",
	"mychart",
	"lab results",
	"prescription history",
	"patient id",
	"patient identifier",
}

// HealthContext reports whether the current page presents a health-record
// context. It never raises an auth challenge or blocks actions; only the
// public-sharing guard consumes it.
func (s *Session) HealthContext(caller context.Context) (bool, error) {
	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return false, fmt.Errorf("session closed")
	}
	ctx, cancel := sessionActionContext(page.GetContext(), caller, sessionInputTimeout)
	defer cancel()
	result, err := page.Context(ctx).Eval(`() => {
		const url = location.href || '';
		const text = (document.body?.innerText || '').toLowerCase();
		const markers = [];
		const add = m => { if (!markers.includes(m)) markers.push(m); };
		for (const m of ['patient portal', 'medical record', 'health record', 'mychart', 'lab results', 'prescription history', 'patient id', 'patient identifier']) {
			if (text.includes(m)) add(m);
		}
		if (/[?&/.-](mychart|patient-portal|health-portal|medical-record)([?&/.-]|$)/i.test(url)) add('health_portal_url');
		return JSON.stringify({url_hint: url.split('#')[0].split('?')[0], markers});
	}`)
	if err != nil {
		return false, err
	}
	if result == nil || result.Type != proto.RuntimeRemoteObjectTypeString {
		return false, fmt.Errorf("health probe returned no readable result")
	}
	var detected struct {
		URLHint string   `json:"url_hint"`
		Markers []string `json:"markers"`
	}
	if err := json.Unmarshal([]byte(result.Value.Str()), &detected); err != nil {
		return false, err
	}
	// Default-deny needs signal, not a single ordinary word: a text marker
	// plus a second marker (text or the URL shape) confirms the context.
	if len(detected.Markers) >= 2 {
		return true, nil
	}
	if len(detected.Markers) == 1 && detected.Markers[0] == "health_portal_url" {
		return true, nil
	}
	_ = strings.TrimSpace(detected.URLHint)
	return false, nil
}
