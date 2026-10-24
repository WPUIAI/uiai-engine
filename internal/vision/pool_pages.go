package vision

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const poolPageCreateTimeout = 5 * time.Second

// newPoolPage creates one independently disposable storage partition. The
// creation budget must not become a deadline on the page's eventual lifetime.
func newPoolPage(browser *rod.Browser) (*rod.Page, error) {
	if browser == nil {
		return nil, fmt.Errorf("browser unavailable for isolated page")
	}
	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		page *rod.Page
		err  error
	}
	completed := make(chan result)
	go func() {
		page, err := createPoolPage(browser.Context(ctx))
		select {
		case completed <- result{page, err}:
		case <-ctx.Done():
			if page != nil {
				_ = closePoolPage(page)
			}
		}
	}()
	timer := time.NewTimer(poolPageCreateTimeout)
	defer timer.Stop()
	select {
	case created := <-completed:
		if created.err != nil {
			cancel()
			return nil, created.err
		}
		// Page.Close cancels its session context; release that page's parent too.
		context.AfterFunc(created.page.GetContext(), cancel)
		return created.page, nil
	case <-timer.C:
		cancel()
		return nil, fmt.Errorf("isolated pool page creation timed out")
	}
}

func createPoolPage(browser *rod.Browser) (*rod.Page, error) {
	isolated, err := browser.Incognito()
	if err != nil {
		return nil, fmt.Errorf("create pool storage context: %w", err)
	}
	page, err := isolated.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		_ = isolated.Context(context.Background()).Timeout(2 * time.Second).Close()
		return nil, fmt.Errorf("create isolated pool page: %w", err)
	}
	return page, nil
}

// closePoolPage closes the target and disposes its whole storage partition,
// including popups and storage for origins no longer visible in the main tab.
// An unscoped/foreign page may be closed, but can never close the root browser.
func closePoolPage(page *rod.Page) error {
	if page == nil {
		return nil
	}
	browser := page.Browser()
	pageErr := page.Context(context.Background()).Timeout(2 * time.Second).Close()
	if browser != nil && browser.BrowserContextID != "" {
		if err := browser.Context(context.Background()).Timeout(2 * time.Second).Close(); err != nil {
			return fmt.Errorf("dispose pool storage context: %w", err)
		}
		return nil
	}
	return pageErr
}
