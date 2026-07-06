package vision

import "github.com/go-rod/rod"

// PoolSource is the route/session-facing browser-pool surface.
// Implementations: *Pool (single Chromium) and *MultiPool (multiple independent Chromiums).
type PoolSource interface {
	GetPage() (*rod.Page, error)
	ReleasePage(*rod.Page)
	ValidateNavigationURL(string) error
	IsBrowserAlive() bool
	MarkFailure()
	Reset()
	Screenshot(ScreenshotOpts) (*ScreenshotResult, error)
	Stats() map[string]any
	Close()
}
