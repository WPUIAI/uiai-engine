package routes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

const sourceReadinessBudget = 5 * time.Second
const sourceReadinessInterval = 100 * time.Millisecond

var errNoReadableSourceContent = errors.New("no readable page content within the source read budget")

// Only initially empty read-only observations are retried. Navigation has its
// existing budget; this short budget begins after the owned session is opened.
// Nonempty text is not claimed as semantic page completeness.
func readSourcePageReady(caller context.Context, sess *vision.Session, opts vision.ReadOptions) (*vision.PageReadResult, error) {
	ctx, cancel := context.WithTimeout(caller, sourceReadinessBudget)
	defer cancel()
	tick := time.NewTicker(sourceReadinessInterval)
	defer tick.Stop()
	var last *vision.PageReadResult
	for {
		read, err := sess.ReadPageContext(ctx, opts)
		if err != nil {
			if caller.Err() != nil {
				return last, caller.Err()
			}
			if ctx.Err() != nil && last != nil {
				return last, errNoReadableSourceContent
			}
			return last, err
		}
		last = read
		if caller.Err() != nil {
			return last, caller.Err()
		}
		if strings.TrimSpace(read.Text) != "" {
			return read, nil
		}
		select {
		case <-ctx.Done():
			if caller.Err() != nil {
				return last, caller.Err()
			}
			return last, errNoReadableSourceContent
		case <-tick.C:
		}
	}
}
