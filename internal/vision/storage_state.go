package vision

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// StorageStateSchema identifies the opaque storage-state blob (#81):
// cookies plus web storage for one origin, exportable after a brokered
// login and importable into a fresh session so authenticated state can be
// reused without re-injecting secrets.
const StorageStateSchema = "uiai.storage_state.v1"

// maxStorageStateBytes bounds each web-storage dump so a pathological page
// cannot balloon the blob.
const maxStorageStateBytes = 256 << 10

// StorageState is the opaque blob exchanged between export and import.
type StorageState struct {
	Schema         string            `json:"schema"`
	Origin         string            `json:"origin"`
	ExportedAt     string            `json:"exported_at"`
	Cookies        []StorageCookie   `json:"cookies"`
	LocalStorage   map[string]string `json:"local_storage"`
	SessionStorage map[string]string `json:"session_storage"`
}

// StorageCookie is the minimal cookie projection carried in the blob.
type StorageCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"http_only,omitempty"`
	SameSite string `json:"same_site,omitempty"`
	Expires  int64  `json:"expires,omitempty"` // seconds since epoch; 0 = session cookie
}

const storageDumpJS = `() => {
	const dump = (store) => {
		const out = {};
		for (let i = 0; i < store.length; i++) {
			const key = store.key(i);
			try { out[key] = store.getItem(key); } catch (e) { out[key] = null; }
		}
		return out;
	};
	return JSON.stringify({localStorage: dump(window.localStorage), sessionStorage: dump(window.sessionStorage)});
}`

// ExportStorageState captures cookies and web storage for the session's
// current origin (#81): login once through a brokered fill, export the
// authenticated state, and reuse it without secrets. The blob is opaque to
// the engine and never logged.
func (s *Session) ExportStorageState() (*StorageState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}
	state := &StorageState{
		Schema:         StorageStateSchema,
		Origin:         strings.TrimSuffix(safeEvalStr(s.page, `() => window.location.origin`), "/"),
		ExportedAt:     time.Now().UTC().Format(time.RFC3339),
		LocalStorage:   map[string]string{},
		SessionStorage: map[string]string{},
	}
	cookies, err := s.page.Cookies([]string{})
	if err != nil {
		return nil, fmt.Errorf("cookie export failed: %w", err)
	}
	for _, cookie := range cookies {
		state.Cookies = append(state.Cookies, StorageCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: string(cookie.SameSite),
			Expires:  int64(cookie.Expires),
		})
	}
	dump, err := s.page.Eval(storageDumpJS)
	if err == nil {
		var parsed struct {
			LocalStorage   map[string]string `json:"localStorage"`
			SessionStorage map[string]string `json:"sessionStorage"`
		}
		if json.Unmarshal([]byte(dump.Value.Str()), &parsed) == nil {
			state.LocalStorage = parsed.LocalStorage
			state.SessionStorage = parsed.SessionStorage
		}
	}
	if encoded, err := json.Marshal(state); err != nil || len(encoded) > 2*maxStorageStateBytes {
		return nil, fmt.Errorf("storage state exceeds the bounded export size")
	}
	return state, nil
}

// ImportStorageState applies a previously exported blob into this session's
// current origin (#81): authenticated state lands without any secret
// re-entering model context.
func (s *Session) ImportStorageState(state *StorageState) error {
	if state == nil || state.Schema != StorageStateSchema {
		return fmt.Errorf("unsupported storage state (want schema %s)", StorageStateSchema)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return fmt.Errorf("session closed")
	}
	params := make([]*proto.NetworkCookieParam, 0, len(state.Cookies))
	for _, cookie := range state.Cookies {
		param := &proto.NetworkCookieParam{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			URL:      state.Origin,
		}
		if cookie.Expires > 0 {
			param.Expires = proto.TimeSinceEpoch(cookie.Expires)
		}
		if cookie.SameSite != "" {
			param.SameSite = proto.NetworkCookieSameSite(cookie.SameSite)
		}
		params = append(params, param)
	}
	if len(params) > 0 {
		if err := s.page.SetCookies(params); err != nil {
			return fmt.Errorf("cookie import failed: %w", err)
		}
	}
	if len(state.LocalStorage) > 0 || len(state.SessionStorage) > 0 {
		payload, err := json.Marshal(map[string]map[string]string{"localStorage": state.LocalStorage, "sessionStorage": state.SessionStorage})
		if err != nil {
			return fmt.Errorf("storage import encode failed: %w", err)
		}
		script := fmt.Sprintf(`() => {
			const payload = %s;
			const apply = (store, values) => { for (const [key, value] of Object.entries(values || {})) { try { if (value !== null) store.setItem(key, value); } catch (e) {} } };
			apply(window.localStorage, payload.localStorage);
			apply(window.sessionStorage, payload.sessionStorage);
			return true;
		}`, string(payload))
		if _, err := s.page.Eval(script); err != nil {
			return fmt.Errorf("web storage import failed: %w", err)
		}
	}
	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	_ = focusapacket.SanitizeURL(s.URL)
	return nil
}

var _ = rod.Page{}
