package routes

import (
	"fmt"
	"math"
	"time"
)

const ddgRequestInterval = 15 * time.Second

type searchRequestSpacingError struct{ retryAfter int }

func (*searchRequestSpacingError) Error() string {
	return "local DuckDuckGo request spacing is active; no provider request was sent"
}

// One bounded value-free slot per engine runtime, not a queue or background
// timer. Cached reads are served before admission and do not consume this slot.
func (b *SearchRuntime) admitDDGRequest() error {
	if b == nil {
		return fmt.Errorf("DuckDuckGo requires the managed search runtime")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if now.Before(b.ddgNextRequest) {
		return &searchRequestSpacingError{retryAfter: int(math.Ceil(b.ddgNextRequest.Sub(now).Seconds()))}
	}
	b.ddgNextRequest = now.Add(ddgRequestInterval)
	return nil
}
func (b *SearchRuntime) ddgRequestDelay() int {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return max(0, int(math.Ceil(b.ddgNextRequest.Sub(b.now()).Seconds())))
}
