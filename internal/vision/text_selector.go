package vision

import (
	"fmt"
	"strings"
	"time"
)

// ResolveSelector converts @refs and simple text/role selector helpers into a CSS selector.
// Supported helpers:
//   - text=Submit or text/Submit: first visible element whose text/value/aria-label contains Submit
//   - role=button;name=Submit: first visible element matching role and accessible name/text
func (s *Session) ResolveSelector(selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	resolved := s.ResolveRef(selector)
	if resolved != selector || selector == "" {
		return resolved, nil
	}
	kind, value, role := parseSelectorHelper(selector)
	if kind == "" {
		return selector, nil
	}
	return s.resolveTextLikeSelector(kind, value, role)
}

func parseSelectorHelper(selector string) (kind, value, role string) {
	lower := strings.ToLower(selector)
	switch {
	case strings.HasPrefix(lower, "text="):
		return "text", strings.TrimSpace(selector[len("text="):]), ""
	case strings.HasPrefix(lower, "text/"):
		return "text", strings.TrimSpace(selector[len("text/"):]), ""
	case strings.HasPrefix(lower, "role="):
		parts := strings.Split(selector, ";")
		for _, part := range parts {
			k, v, ok := strings.Cut(part, "=")
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(k)) {
			case "role":
				role = strings.TrimSpace(v)
			case "name", "text":
				value = strings.TrimSpace(v)
			}
		}
		if role != "" || value != "" {
			return "role", value, role
		}
	}
	return "", "", ""
}

func (s *Session) resolveTextLikeSelector(kind, value, role string) (string, error) {
	if strings.TrimSpace(value) == "" && strings.TrimSpace(role) == "" {
		return "", fmt.Errorf("text selector requires text/name or role")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", fmt.Errorf("session closed")
	}
	id := fmt.Sprintf("uiai-text-%d", time.Now().UnixNano())
	js := `([kind, wanted, role, id]) => {
  const norm = (s) => (s || '').replace(/\s+/g, ' ').trim().toLowerCase();
  const target = norm(wanted);
  const targetRole = norm(role);
  const visible = (el) => {
    const style = window.getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    return style && style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0;
  };
  const nameOf = (el) => [el.getAttribute('aria-label'), el.getAttribute('title'), el.value, el.innerText, el.textContent].filter(Boolean).join(' ');
  const nodes = Array.from(document.querySelectorAll('button,a,input,textarea,select,label,[role],[aria-label],[title],summary,[contenteditable="true"]'));
  for (const el of nodes) {
    if (!visible(el)) continue;
    const elRole = norm(el.getAttribute('role') || el.tagName.toLowerCase());
    const hay = norm(nameOf(el));
    if (targetRole && elRole !== targetRole && !(targetRole === 'button' && el.tagName.toLowerCase() === 'button') && !(targetRole === 'link' && el.tagName.toLowerCase() === 'a')) continue;
    if (target && !hay.includes(target)) continue;
    el.setAttribute('data-uiai-text-ref', id);
    return '[data-uiai-text-ref="' + id + '"]';
  }
  return '';
}`
	result, err := s.page.Timeout(3*time.Second).Eval(js, []any{kind, value, role, id})
	if err != nil {
		return "", fmt.Errorf("resolve text selector failed: %w", err)
	}
	if result == nil || result.Value.Str() == "" {
		return "", fmt.Errorf("text selector not found: %s", value)
	}
	return result.Value.Str(), nil
}
