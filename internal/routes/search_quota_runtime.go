package routes

import (
	"context"
	"net/http"
)

type searchQuotaBindingKey struct{}
type searchQuotaBinding struct {
	provider string
	sequence uint64
}

func (b *SearchRuntime) bindQuotaObservation(ctx context.Context, provider string) context.Context {
	// Only the source-verified Brave contract is supported so far. No provider
	// identity is inferred from URLs, headers, query strings or credentials.
	if b == nil || provider != "brave" {
		return ctx
	}
	b.mu.Lock()
	if b.quotaSequences == nil {
		b.quotaSequences = map[string]uint64{}
	}
	b.quotaSequences[provider]++
	sequence := b.quotaSequences[provider]
	b.mu.Unlock()
	return context.WithValue(ctx, searchQuotaBindingKey{}, searchQuotaBinding{provider: provider, sequence: sequence})
}
func observeSearchQuota(req *http.Request, h http.Header, status int) {
	binding, ok := req.Context().Value(searchQuotaBindingKey{}).(searchQuotaBinding)
	if !ok || binding.provider != "brave" {
		return
	}
	runtime, _ := req.Context().Value(searchBreakersKey{}).(*SearchRuntime)
	if runtime == nil {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	// An earlier-started in-flight request cannot replace a newer attempt's
	// observation; request ordering is not claimed as provider billing order.
	if binding.sequence != runtime.quotaSequences[binding.provider] {
		return
	}
	observation, valid := parseBraveSearchQuota(h, status, runtime.now())
	if !valid {
		return
	}
	if runtime.quotaObservations == nil {
		runtime.quotaObservations = map[string]searchQuotaObservation{}
	}
	runtime.quotaObservations[binding.provider] = observation
}
func (b *SearchRuntime) quotaProjection(provider string) searchQuotaProjection {
	p := searchQuotaProjection{Schema: "uiai.search_quota_observation.v1", Status: "unknown", Scope: "engine_runtime_provider_slot", Authority: "advisory_only", AccountBinding: "not_verified", CurrentRemainingStatus: "unknown"}
	if b == nil {
		return p
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	observation, ok := b.quotaObservations[provider]
	if !ok {
		return p
	}
	p.Status = "observed_headers"
	p.Source = "documented_brave_response_headers"
	received := observation.receivedAt.UTC()
	p.ReceivedAt = &received
	p.ResponseStatus = observation.status
	age := b.now().Sub(observation.receivedAt)
	p.ClockUncertain = age < 0
	p.Stale = p.ClockUncertain || age > searchQuotaObservationTTL
	p.Windows = append([]searchQuotaWindow(nil), observation.windows...)
	for i := range p.Windows {
		// Historical counts stay historical. Elapsed reset delays never mint
		// credits or infer restored quota/current account identity/permission.
		p.Windows[i].ResetElapsed = !p.ClockUncertain && age.Seconds() >= float64(p.Windows[i].ResetAfterSeconds)
	}
	return p
}
