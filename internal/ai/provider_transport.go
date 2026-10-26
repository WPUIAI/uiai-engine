package ai

import (
	"bytes"
	"context"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/httpboundary"
)

// newProviderRequest keeps endpoint/parser details outside errors and checks
// construction before callers attach credentials to the request.
func newProviderRequest(ctx context.Context, provider, endpoint string, body []byte) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, providerRequestFailure(provider)
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

// doProviderRequest shares origin confinement and value-free transport failures
// across inference dialects without changing the caller's client or policies.
func (p *Provider) doProviderRequest(provider string, request *http.Request) (*http.Response, error) {
	client := httpboundary.OriginBoundClient(p.client)
	response, err := client.Do(request)
	if err != nil {
		return nil, providerTransportFailure(provider, err)
	}
	return response, nil
}
