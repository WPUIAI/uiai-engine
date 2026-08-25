package vision

import (
	"sync"
	"time"

	"github.com/go-rod/rod"
)

// WrapExternalPage creates a lightweight Session facade over a page whose
// process and lifecycle are owned by another core subsystem. It intentionally
// does not attach a second diagnostics listener or a pool-release callback.
func WrapExternalPage(page *rod.Page, url string, width, height int) *Session {
	now := time.Now()
	title := ""
	if page != nil {
		if el, err := page.Eval(`() => document.title`); err == nil {
			title = el.Value.Str()
		}
	}
	return &Session{
		ID:          generateID(),
		URL:         url,
		Title:       title,
		Width:       width,
		Height:      height,
		CreatedAt:   now,
		LastUsed:    now,
		page:        page,
		refs:        make(map[string]SnapshotRef),
		diagnostics: newDiagnosticsRecorder(),
		mu:          sync.Mutex{},
	}
}
