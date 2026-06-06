package vision

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
)

// ═══════════════════════════════════════════════════════
// Accessibility Snapshot — inspired by agent-browser
// ═══════════════════════════════════════════════════════
//
// Generates a text-based accessibility tree with @ref IDs.
// LLMs use refs to interact: {"selector": "@e3"} for click/type/hover.
// This replaces the DOMInfo approach with a richer, more reliable model.
//
// Example output:
//   - heading "Discover the New Empire Startups" [ref=e1] [level=1]
//   - link "Join Free" [ref=e2]
//   - textbox "Search startups by name" [ref=e3]
//   - button "Search" [ref=e4]
//   - link "Browse Startups" [ref=e5]

// SnapshotOptions controls what the snapshot includes.
type SnapshotOptions struct {
	Interactive bool   `json:"interactive"` // Only interactive elements (buttons, links, inputs)
	Compact     bool   `json:"compact"`     // Remove empty structural nodes
	MaxDepth    int    `json:"max_depth"`   // 0 = unlimited
	Selector    string `json:"selector"`    // Scope to CSS selector (default: body)
}

// SnapshotRef is a stored reference for an element found in the snapshot.
type SnapshotRef struct {
	Selector string `json:"selector"` // CSS selector to find this element
	Role     string `json:"role"`
	Name     string `json:"name,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// SnapshotResult is returned by the Snapshot method.
type SnapshotResult struct {
	Tree   string                  `json:"tree"`
	Refs   map[string]SnapshotRef  `json:"refs"`
	Stats  SnapshotStats           `json:"stats"`
	Focusa *SnapshotFocusaMetadata `json:"focusa,omitempty"`
}

type SnapshotFocusaMetadata struct {
	TargetRef         string   `json:"target_ref"`
	EvidenceRef       string   `json:"evidence_ref"`
	PreferredTool     string   `json:"preferred_tool"`
	Summary           string   `json:"summary"`
	NextTools         []string `json:"next_tools"`
	FocusaScopeStatus string   `json:"focusa_scope_status"`
}

// SnapshotStats provides token-cost estimates.
type SnapshotStats struct {
	Lines       int `json:"lines"`
	Chars       int `json:"chars"`
	Tokens      int `json:"tokens"` // ~chars/4
	RefCount    int `json:"ref_count"`
	Interactive int `json:"interactive"`
}

// refStore tracks refs per snapshot call.
type refStore struct {
	mu      sync.Mutex
	counter int
	refs    map[string]SnapshotRef
}

func newRefStore() *refStore {
	return &refStore{refs: make(map[string]SnapshotRef)}
}

func (r *refStore) next() string {
	r.counter++
	return fmt.Sprintf("e%d", r.counter)
}

// interactiveRoles are ARIA roles that users can interact with.
var interactiveRoles = map[string]bool{
	"button": true, "link": true, "textbox": true, "checkbox": true,
	"radio": true, "combobox": true, "listbox": true, "menuitem": true,
	"searchbox": true, "slider": true, "spinbutton": true, "switch": true,
	"tab": true, "option": true, "treeitem": true,
}

// contentRoles provide structure/context and get refs if named.
var contentRoles = map[string]bool{
	"heading": true, "cell": true, "columnheader": true, "rowheader": true,
	"listitem": true, "article": true, "region": true, "main": true,
	"navigation": true, "img": true, "figure": true,
}

// structuralRoles are purely structural (filterable in compact mode).
var structuralRoles = map[string]bool{
	"generic": true, "group": true, "list": true, "table": true,
	"row": true, "rowgroup": true, "grid": true, "menu": true,
	"menubar": true, "toolbar": true, "tablist": true, "tree": true,
	"document": true, "application": true, "presentation": true, "none": true,
}

// Snapshot generates an accessibility tree with @ref selectors.
//
// The tree is generated via JavaScript in the browser, walking the DOM
// and computing ARIA roles, names, and generating stable CSS selectors.
// This approach works with Rod (no native a11y API needed).
func (s *Session) Snapshot(opts SnapshotOptions) (*SnapshotResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 50 // effectively unlimited
	}

	scopeSelector := opts.Selector
	if scopeSelector == "" {
		scopeSelector = "body"
	}

	// JavaScript that walks the DOM and produces an a11y-like tree.
	// Returns JSON array of {role, name, tag, depth, selector, level, interactive, visible}.
	js := fmt.Sprintf(`() => {
		const MAX_DEPTH = %d;
		const SCOPE = %q;

		function getRole(el) {
			// Explicit ARIA role takes priority
			const explicit = el.getAttribute('role');
			if (explicit) return explicit.toLowerCase();

			// Implicit roles by tag
			const tag = el.tagName.toLowerCase();
			const type = (el.getAttribute('type') || '').toLowerCase();
			const map = {
				'a': el.hasAttribute('href') ? 'link' : '',
				'button': 'button', 'input': getInputRole(type),
				'select': 'combobox', 'textarea': 'textbox',
				'h1': 'heading', 'h2': 'heading', 'h3': 'heading',
				'h4': 'heading', 'h5': 'heading', 'h6': 'heading',
				'img': 'img', 'nav': 'navigation', 'main': 'main',
				'header': 'banner', 'footer': 'contentinfo',
				'form': 'form', 'table': 'table', 'ul': 'list',
				'ol': 'list', 'li': 'listitem', 'section': 'region',
				'article': 'article', 'aside': 'complementary',
				'details': 'group', 'summary': 'button',
				'dialog': 'dialog', 'td': 'cell', 'th': 'columnheader',
				'tr': 'row', 'thead': 'rowgroup', 'tbody': 'rowgroup',
			};
			return map[tag] || '';
		}

		function getInputRole(type) {
			const m = {
				'text': 'textbox', 'email': 'textbox', 'password': 'textbox',
				'search': 'searchbox', 'tel': 'textbox', 'url': 'textbox',
				'number': 'spinbutton', 'range': 'slider',
				'checkbox': 'checkbox', 'radio': 'radio',
				'submit': 'button', 'reset': 'button', 'button': 'button',
				'file': 'button', 'image': 'button',
			};
			return m[type] || 'textbox';
		}

		function getName(el) {
			// aria-label takes priority
			const al = el.getAttribute('aria-label');
			if (al) return al.trim();

			// aria-labelledby
			const alb = el.getAttribute('aria-labelledby');
			if (alb) {
				const label = document.getElementById(alb);
				if (label) return label.textContent.trim().substring(0, 80);
			}

			const tag = el.tagName.toLowerCase();

			// img alt
			if (tag === 'img') return (el.alt || '').trim();

			// input placeholder/value
			if (tag === 'input' || tag === 'textarea') {
				// Check associated label
				if (el.id) {
					const lbl = document.querySelector('label[for="' + el.id + '"]');
					if (lbl) return lbl.textContent.trim().substring(0, 80);
				}
				return (el.placeholder || el.value || el.name || '').trim();
			}

			// For buttons/links, use text content
			if (tag === 'button' || tag === 'a' || tag === 'summary') {
				return el.textContent.trim().substring(0, 80);
			}

			// Headings
			if (/^h[1-6]$/.test(tag)) {
				return el.textContent.trim().substring(0, 120);
			}

			return '';
		}

		function getSelector(el) {
			if (el.id) return '#' + CSS.escape(el.id);
			const tag = el.tagName.toLowerCase();

			// Try class-based (unique on page)
			if (el.className && typeof el.className === 'string') {
				const cls = el.className.trim().split(/\s+/).filter(c => c && !c.startsWith('_') && c.length < 40).slice(0, 2).join('.');
				if (cls) {
					const sel = tag + '.' + cls;
					try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
				}
			}

			// Try aria-label
			const al = el.getAttribute('aria-label');
			if (al) {
				const sel = tag + '[aria-label="' + al.replace(/"/g, '\\"') + '"]';
				try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
			}

			// Try href (for links)
			if (tag === 'a' && el.href) {
				const href = el.getAttribute('href');
				if (href && href.length < 80 && !href.startsWith('javascript:')) {
					const sel = 'a[href="' + href.replace(/"/g, '\\"') + '"]';
					try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
				}
			}

			// Try name attr (for inputs)
			if (el.name) {
				const sel = tag + '[name="' + el.name.replace(/"/g, '\\"') + '"]';
				try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
			}

			// Try placeholder (for inputs)
			if (el.placeholder) {
				const sel = tag + '[placeholder="' + el.placeholder.replace(/"/g, '\\"') + '"]';
				try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
			}

			// Build path from nearest ancestor with ID
			return buildUniqueSelector(el);
		}

		function buildUniqueSelector(el) {
			const parts = [];
			let cur = el;
			while (cur && cur !== document.body && parts.length < 4) {
				const tag = cur.tagName.toLowerCase();
				if (cur.id) {
					parts.unshift('#' + CSS.escape(cur.id));
					break;
				}
				const parent = cur.parentElement;
				if (parent) {
					const siblings = Array.from(parent.children).filter(c => c.tagName === cur.tagName);
					if (siblings.length === 1) {
						parts.unshift(tag);
					} else {
						parts.unshift(tag + ':nth-of-type(' + (siblings.indexOf(cur) + 1) + ')');
					}
				} else {
					parts.unshift(tag);
				}
				cur = parent;
			}
			if (parts.length === 0) return el.tagName.toLowerCase();
			// Verify uniqueness
			const sel = parts.join(' > ');
			try { if (document.querySelectorAll(sel).length === 1) return sel; } catch(e) {}
			// Fallback: add body prefix
			return 'body > ' + parts.join(' > ');
		}

		function isVisible(el) {
			if (!el.offsetParent && el.tagName !== 'BODY' && el.tagName !== 'HTML') return false;
			const r = el.getBoundingClientRect();
			return r.width > 0 && r.height > 0;
		}

		const results = [];
		function walk(el, depth) {
			if (depth > MAX_DEPTH) return;
			if (!el || el.nodeType !== 1) return;

			const tag = el.tagName.toLowerCase();
			// Skip hidden, script, style, svg internals
			if (tag === 'script' || tag === 'style' || tag === 'noscript' || tag === 'template') return;
			if (el.hidden || el.getAttribute('aria-hidden') === 'true') return;

			const role = getRole(el);
			const name = getName(el);
			const vis = isVisible(el);

			if (role && vis) {
				const entry = { role, name: name.substring(0, 120), tag, depth, selector: getSelector(el), visible: vis };
				if (/^h[1-6]$/.test(tag)) entry.level = parseInt(tag[1]);
				results.push(entry);
			}

			for (const child of el.children) {
				walk(child, depth + 1);
			}
		}

		const root = document.querySelector(SCOPE);
		if (root) walk(root, 0);
		return JSON.stringify(results);
	}`, maxDepth, scopeSelector)

	result, err := s.page.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("snapshot eval failed: %w", err)
	}

	var elements []struct {
		Role     string `json:"role"`
		Name     string `json:"name"`
		Tag      string `json:"tag"`
		Depth    int    `json:"depth"`
		Selector string `json:"selector"`
		Level    int    `json:"level,omitempty"`
		Visible  bool   `json:"visible"`
	}

	raw := result.Value.Str()
	if err := json.Unmarshal([]byte(raw), &elements); err != nil {
		return nil, fmt.Errorf("snapshot parse failed: %w", err)
	}

	// Build the text tree and ref map
	store := newRefStore()
	var lines []string
	interactiveCount := 0

	for _, el := range elements {
		isInteractive := interactiveRoles[el.Role]
		isContent := contentRoles[el.Role]
		isStructural := structuralRoles[el.Role]

		// In interactive mode, skip non-interactive
		if opts.Interactive && !isInteractive {
			continue
		}

		// In compact mode, skip unnamed structural
		if opts.Compact && isStructural && el.Name == "" {
			continue
		}

		indent := strings.Repeat("  ", el.Depth)
		line := fmt.Sprintf("%s- %s", indent, el.Role)

		if el.Name != "" {
			line += fmt.Sprintf(" %q", el.Name)
		}

		// Assign ref to interactive or named content elements
		shouldRef := isInteractive || (isContent && el.Name != "")
		if shouldRef {
			ref := store.next()
			store.refs[ref] = SnapshotRef{
				Selector: el.Selector,
				Role:     el.Role,
				Name:     el.Name,
				Tag:      el.Tag,
			}
			line += fmt.Sprintf(" [ref=@%s]", ref)

			if isInteractive {
				interactiveCount++
			}
		}

		if el.Level > 0 {
			line += fmt.Sprintf(" [level=%d]", el.Level)
		}

		lines = append(lines, line)
	}

	tree := strings.Join(lines, "\n")
	if tree == "" {
		tree = "(empty page)"
	}

	s.SnapshotCount++
	snapshotSeq := s.SnapshotCount
	stats := SnapshotStats{
		Lines:       len(lines),
		Chars:       len(tree),
		Tokens:      len(tree) / 4,
		RefCount:    len(store.refs),
		Interactive: interactiveCount,
	}
	s.touch()

	return &SnapshotResult{
		Tree:   tree,
		Refs:   store.refs,
		Stats:  stats,
		Focusa: buildSnapshotFocusaMetadata(s.ID, snapshotSeq, s.URL, s.Title, opts.Selector, stats, s.FocusaScope),
	}, nil
}

func buildSnapshotFocusaMetadata(sessionID string, snapshotSeq int, pageURL, title, selector string, stats SnapshotStats, scope *FocusaScope) *SnapshotFocusaMetadata {
	if snapshotSeq <= 0 {
		snapshotSeq = 1
	}
	targetRef := "browser:" + focusapacket.SanitizeURL(pageURL)
	if pageURL == "" {
		targetRef = "browser:session=" + focusapacket.Truncate(sessionID, 80)
	}
	summary := fmt.Sprintf("Snapshot %d refs (%d interactive, %d lines) from %s", stats.RefCount, stats.Interactive, stats.Lines, focusapacket.Truncate(firstNonEmpty(title, pageURL, sessionID), 160))
	if selector != "" {
		summary += " selector=" + focusapacket.Truncate(selector, 80)
	}
	return &SnapshotFocusaMetadata{
		TargetRef:         targetRef,
		EvidenceRef:       fmt.Sprintf("uiai-browser:session=%s:snapshot:%d", focusapacket.Truncate(sessionID, 80), snapshotSeq),
		PreferredTool:     "focusa_evidence_capture",
		Summary:           focusapacket.Truncate(summary, focusapacket.MaxCaptureSummaryChars),
		NextTools:         []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		FocusaScopeStatus: readFocusaScopeStatus(scope),
	}
}

// StoreRefs saves snapshot refs into the session for later use by Click/Type/Hover.
func (s *Session) StoreRefs(refs map[string]SnapshotRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refs = refs
}

// ResolveRef converts "@e3" to its CSS selector from the last snapshot.
// If the input doesn't start with "@", it's returned as-is (treated as CSS selector).
func (s *Session) ResolveRef(selectorOrRef string) string {
	if !strings.HasPrefix(selectorOrRef, "@") {
		return selectorOrRef // already a CSS selector
	}
	ref := selectorOrRef[1:] // strip @

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.refs == nil {
		return selectorOrRef // no snapshot taken yet, return as-is
	}
	if r, ok := s.refs[ref]; ok {
		return r.Selector
	}
	return selectorOrRef // ref not found, return as-is (will fail at element lookup)
}
