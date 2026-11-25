package vision

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// #301 iframes: snapshots emit frame-qualified selectors of the shape
// "frame:<iframe-index>::<inner-selector>" for elements inside same-origin
// iframes (nesting stacks prefixes). Cross-origin iframe documents are
// unreachable by browser design and resolve to a typed not-found error.

const frameElementJS = `(sel) => {
	let doc = document;
	while (sel.startsWith('frame:')) {
		const boundary = sel.indexOf('::');
		if (boundary <= 6) return null;
		const idx = parseInt(sel.substring(6, boundary), 10);
		if (isNaN(idx) || idx < 0) return null;
		const ifr = doc.querySelectorAll('iframe')[idx];
		if (!ifr) return null;
		try { doc = ifr.contentDocument; } catch (e) { return null; }
		if (!doc) return null;
		sel = sel.substring(boundary + 2);
	}
	return doc.querySelector(sel);
}`

// validFrameSelector reports whether a frame-qualified selector is well
// formed: at least one frame:index:: segment with a non-empty inner selector.
func validFrameSelector(selector string) bool {
	rest := strings.TrimPrefix(selector, "frame:")
	boundary := strings.Index(rest, "::")
	if boundary <= 0 {
		return false
	}
	idx := rest[:boundary]
	if idx == "" {
		return false
	}
	for _, c := range idx {
		if c < '0' || c > '9' {
			return false
		}
	}
	return rest[boundary+2:] != ""
}

// isFrameSelector reports whether a selector is frame-qualified.
func isFrameSelector(selector string) bool {
	return strings.HasPrefix(selector, "frame:")
}

// frameCenterJS returns the page-absolute center of a frame-qualified
// element: scroll it into view inside its frame, measure its box, then sum
// every host iframe's rect along the chain.
const frameCenterJS = `(sel) => {
	const chain = [];
	let doc = document;
	while (sel.startsWith('frame:')) {
		const boundary = sel.indexOf('::');
		if (boundary <= 6) return null;
		const idx = parseInt(sel.substring(6, boundary), 10);
		if (isNaN(idx) || idx < 0) return null;
		const ifr = doc.querySelectorAll('iframe')[idx];
		if (!ifr) return null;
		chain.push(ifr);
		try { doc = ifr.contentDocument; } catch (e) { return null; }
		if (!doc) return null;
		sel = sel.substring(boundary + 2);
	}
	const el = doc.querySelector(sel);
	if (!el) return null;
	try { el.scrollIntoView({ block: 'center' }); } catch (e) {}
	const box = el.getBoundingClientRect();
	let x = box.left + box.width / 2;
	let y = box.top + box.height / 2;
	for (let i = chain.length - 1; i >= 0; i--) {
		const host = chain[i].getBoundingClientRect();
		x += host.left;
		y += host.top;
	}
	return JSON.stringify({ x: x, y: y });
}`

// frameClickLocked dispatches a native left click at the page-absolute center
// of a frame-qualified element. Caller must hold s.mu.
func (s *Session) frameClickLocked(page *rod.Page, selector string) error {
	res, err := page.Timeout(elementRetryStep).Eval(frameCenterJS, selector)
	if err != nil {
		return fmt.Errorf("frame click locate failed: %w", err)
	}
	payload := res.Value.Str()
	var point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if jsonErr := json.Unmarshal([]byte(payload), &point); jsonErr != nil || (point.X == 0 && point.Y == 0) {
		return fmt.Errorf("frame click target unavailable for %s", selector)
	}
	pressed := proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMousePressed, X: point.X, Y: point.Y,
		Button: proto.InputMouseButtonLeft, ClickCount: 1,
	}
	if err := pressed.Call(page); err != nil {
		return fmt.Errorf("frame click dispatch failed: %w", err)
	}
	released := pressed
	released.Type = proto.InputDispatchMouseEventTypeMouseReleased
	if err := released.Call(page); err != nil {
		return fmt.Errorf("frame click release failed: %w", err)
	}
	return nil
}

// retryFrameElement resolves a frame-qualified selector with the shared
// element retry budget; the lookup runs in-page and descends into the
// same-origin iframe documents.
func retryFrameElement(page *rod.Page, selector string) (*rod.Element, error) {
	if !validFrameSelector(selector) {
		return nil, fmt.Errorf("frame selector %q must match frame:<index>::<selector>", selector)
	}
	deadline := time.Now().Add(elementRetryBudget)
	var lastErr error
	for attempt := 1; ; attempt++ {
		el, ferr := page.Timeout(elementRetryStep).ElementByJS(rod.Eval(frameElementJS, selector))
		if ferr == nil && el != nil {
			return el.Context(page.GetContext()), nil
		}
		if ferr != nil {
			lastErr = ferr
		} else {
			lastErr = fmt.Errorf("frame element not found")
		}
		if time.Now().Add(elementRetryStep).After(deadline) {
			break
		}
		backoff := time.Duration(attempt) * 75 * time.Millisecond
		if backoff > 300*time.Millisecond {
			backoff = 300 * time.Millisecond
		}
		time.Sleep(backoff)
	}
	return nil, fmt.Errorf("frame element not found after %s: %s (%w)", elementRetryBudget.Round(time.Millisecond), selector, lastErr)
}
