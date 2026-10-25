// Package httpboundary owns shared origin-confined HTTP redirect behavior.
package httpboundary

import (
	"fmt"
	"net/http"
	"strings"
)

// OriginBoundClient copies the caller's client without mutating it. Redirects
// cannot expand the request's configured origin or introduce URL credentials.
// Existing same-origin policy and the default ten-request bound are preserved.
func OriginBoundClient(base *http.Client) http.Client {
	client := *base
	policy := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) == 0 || next.URL.Scheme != via[0].URL.Scheme || !strings.EqualFold(next.URL.Host, via[0].URL.Host) || next.URL.User != nil {
			return http.ErrUseLastResponse
		}
		if policy != nil {
			return policy(next, via)
		}
		if len(via) >= 10 {
			return fmt.Errorf("origin-bound HTTP redirect limit exceeded")
		}
		return nil
	}
	return client
}
