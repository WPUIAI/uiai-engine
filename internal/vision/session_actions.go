package vision

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// ═══════════════════════════════════════════════════════
// Extended session actions — fills gaps that otherwise
// require fragile eval() workarounds.
// ═══════════════════════════════════════════════════════

// Fill replaces an input value without using Rod's Must* panic path.
// We prefer DOM assignment + input/change events so slow interactive pages
// don't crash the whole handler on click timeouts.
func (s *Session) Fill(selector, text string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := el.ScrollIntoView(); err != nil {
		return nil, fmt.Errorf("fill scroll failed: %w", err)
	}
	if err := el.Focus(); err != nil {
		return nil, fmt.Errorf("fill focus failed: %w", err)
	}
	if _, err := el.Eval(`() => {
		if (this.isContentEditable) {
			this.textContent = '';
		} else {
			this.value = '';
		}
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
		return true;
	}`); err != nil {
		return nil, fmt.Errorf("fill clear failed: %w", err)
	}
	if err := el.Input(text); err != nil {
		return nil, fmt.Errorf("fill input failed: %w", err)
	}

	time.Sleep(120 * time.Millisecond)
	return s.snap(start)
}

// Select chooses a dropdown option by value or visible text.
func (s *Session) Select(selector string, values []string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// Try selecting by value first via Rod's Select method
	err = el.Select(values, true, rod.SelectorTypeText)
	if err != nil {
		// Fallback: try by value attribute
		err = el.Select(values, true, rod.SelectorTypeCSSSector)
		if err != nil {
			return nil, fmt.Errorf("option not found: %v", values)
		}
	}

	time.Sleep(100 * time.Millisecond)
	return s.snap(start)
}

// Press sends a keyboard key (Enter, Tab, Escape, ArrowDown, etc).
func (s *Session) Press(key string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	k, ok := keyMap[key]
	if !ok {
		return nil, fmt.Errorf("unknown key: %s (supported: Enter, Tab, Escape, ArrowDown, ArrowUp, ArrowLeft, ArrowRight, Backspace, Delete, Space, Home, End, PageUp, PageDown)", key)
	}

	s.page.Keyboard.Press(k)
	time.Sleep(200 * time.Millisecond)
	s.page.Timeout(2 * time.Second).WaitDOMStable(100*time.Millisecond, 0.2)

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)

	return s.snap(start)
}

// Back navigates browser history back.
func (s *Session) Back() (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if err := s.page.NavigateBack(); err != nil {
		return nil, fmt.Errorf("back failed: %w", err)
	}
	s.page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.2)

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.NavCount++

	return s.snap(start)
}

// Forward navigates browser history forward.
func (s *Session) Forward() (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if err := s.page.NavigateForward(); err != nil {
		return nil, fmt.Errorf("forward failed: %w", err)
	}
	s.page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.2)

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.NavCount++

	return s.snap(start)
}

// TextContent returns the text content of an element.
func (s *Session) TextContent(selector string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", fmt.Errorf("session closed")
	}

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", fmt.Errorf("element not found: %s", selector)
	}

	text, err := el.Text()
	if err != nil {
		return "", err
	}

	s.touch()
	return text, nil
}

// CookieAction represents a cookie operation.
type CookieAction struct {
	Action string `json:"action"` // "get", "set", "clear"
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path,omitempty"`
}

// CookieResult is returned by cookie operations.
type CookieResult struct {
	Cookies []CookieInfo `json:"cookies"`
	Count   int          `json:"count"`
}

// CookieInfo is a simplified cookie representation.
type CookieInfo struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires,omitempty"`
	Secure   bool    `json:"secure"`
	HttpOnly bool    `json:"httpOnly"`
}

// Cookies performs get/set/clear operations on browser cookies.
func (s *Session) Cookies(action CookieAction) (*CookieResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	switch action.Action {
	case "get", "":
		cookies, err := s.page.Cookies(nil)
		if err != nil {
			return nil, err
		}
		result := &CookieResult{Count: len(cookies)}
		for _, c := range cookies {
			info := CookieInfo{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Expires:  float64(c.Expires),
				Secure:   c.Secure,
				HttpOnly: c.HTTPOnly,
			}
			// Filter by name if specified
			if action.Name != "" && c.Name != action.Name {
				continue
			}
			result.Cookies = append(result.Cookies, info)
		}
		if action.Name != "" {
			result.Count = len(result.Cookies)
		}
		s.touch()
		return result, nil

	case "set":
		if action.Name == "" || action.Value == "" {
			return nil, fmt.Errorf("name and value required for set")
		}
		domain := action.Domain
		if domain == "" {
			// Extract domain from current URL
			domain = safeEvalStr(s.page, `() => window.location.hostname`)
		}
		path := action.Path
		if path == "" {
			path = "/"
		}
		err := s.page.SetCookies([]*proto.NetworkCookieParam{{
			Name:   action.Name,
			Value:  action.Value,
			Domain: domain,
			Path:   path,
		}})
		if err != nil {
			return nil, err
		}
		s.touch()
		return &CookieResult{Count: 1, Cookies: []CookieInfo{{
			Name: action.Name, Value: action.Value, Domain: domain, Path: path,
		}}}, nil

	case "clear":
		// Get all cookies then remove them
		cookies, err := s.page.Cookies(nil)
		if err != nil {
			return nil, err
		}
		if len(cookies) > 0 {
			// Build removal list
			params := make([]*proto.NetworkCookieParam, 0, len(cookies))
			for _, c := range cookies {
				if action.Name != "" && c.Name != action.Name {
					continue
				}
				params = append(params, &proto.NetworkCookieParam{
					Name:   c.Name,
					Domain: c.Domain,
					Path:   c.Path,
				})
			}
			// Use CDP to delete
			for _, p := range params {
				proto.NetworkDeleteCookies{
					Name:   p.Name,
					Domain: p.Domain,
					Path:   p.Path,
				}.Call(s.page)
			}
		}
		s.touch()
		return &CookieResult{Count: 0}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (use get, set, clear)", action.Action)
	}
}

// SaveAuth saves cookies + localStorage to JSON for later restoration.
func (s *Session) SaveAuth() (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	// Get cookies
	cookies, err := s.page.Cookies(nil)
	if err != nil {
		return nil, err
	}

	// Get localStorage
	ls := safeEvalStr(s.page, `() => JSON.stringify(Object.fromEntries(Object.entries(localStorage)))`)

	// Get sessionStorage
	ss := safeEvalStr(s.page, `() => JSON.stringify(Object.fromEntries(Object.entries(sessionStorage)))`)

	state := map[string]any{
		"url":             safeEvalStr(s.page, `() => window.location.href`),
		"cookies":         cookies,
		"localStorage":    json.RawMessage(ls),
		"sessionStorage":  json.RawMessage(ss),
		"savedAt":         time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	s.touch()
	return data, nil
}

// LoadAuth restores cookies + localStorage from previously saved state.
func (s *Session) LoadAuth(state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return fmt.Errorf("session closed")
	}

	var data struct {
		URL            string                 `json:"url"`
		Cookies        []*proto.NetworkCookie `json:"cookies"`
		LocalStorage   json.RawMessage        `json:"localStorage"`
		SessionStorage json.RawMessage        `json:"sessionStorage"`
	}
	if err := json.Unmarshal(state, &data); err != nil {
		return fmt.Errorf("invalid auth state: %w", err)
	}

	parseStorage := func(raw json.RawMessage) map[string]string {
		if len(raw) == 0 || string(raw) == "null" {
			return nil
		}
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err == nil {
			return m
		}
		var arr []any
		if err := json.Unmarshal(raw, &arr); err == nil {
			return map[string]string{}
		}
		return nil
	}

	if data.URL != "" {
		_ = s.page.Navigate(data.URL)
		s.page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.15)
	}

	// Restore cookies with the same metadata we captured.
	if len(data.Cookies) > 0 {
		params := make([]*proto.NetworkCookieParam, 0, len(data.Cookies))
		for _, c := range data.Cookies {
			param := &proto.NetworkCookieParam{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Secure:   c.Secure,
				HTTPOnly: c.HTTPOnly,
			}
			if c.Expires != 0 {
				param.Expires = c.Expires
			}
			if c.SameSite != "" {
				sameSite := c.SameSite
				param.SameSite = sameSite
			}
			params = append(params, param)
		}
		if err := s.page.SetCookies(params); err != nil {
			return fmt.Errorf("cookie restore failed: %w", err)
		}
	}

	// Restore localStorage
	if storage := parseStorage(data.LocalStorage); storage != nil {
		lsJSON, _ := json.Marshal(storage)
		s.page.Eval(`(items) => { for (const [k,v] of Object.entries(JSON.parse(items))) localStorage.setItem(k,v); }`, string(lsJSON))
	}

	// Restore sessionStorage
	if storage := parseStorage(data.SessionStorage); storage != nil {
		ssJSON, _ := json.Marshal(storage)
		s.page.Eval(`(items) => { for (const [k,v] of Object.entries(JSON.parse(items))) sessionStorage.setItem(k,v); }`, string(ssJSON))
	}

	s.touch()
	return nil
}

// ═══════════════════════════════════════════════════════
// Shared helpers
// ═══════════════════════════════════════════════════════

// snap takes a screenshot with standard settings. Must be called with s.mu held.
func (s *Session) snap(start time.Time) (*SnapResult, error) {
	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// keyMap maps key names to Rod key constants.
var keyMap = map[string]input.Key{
	"Enter":      input.Enter,
	"Tab":        input.Tab,
	"Escape":     input.Escape,
	"Backspace":  input.Backspace,
	"Delete":     input.Delete,
	"Space":      input.Space,
	"ArrowUp":    input.ArrowUp,
	"ArrowDown":  input.ArrowDown,
	"ArrowLeft":  input.ArrowLeft,
	"ArrowRight": input.ArrowRight,
	"Home":       input.Home,
	"End":        input.End,
	"PageUp":     input.PageUp,
	"PageDown":   input.PageDown,
}
