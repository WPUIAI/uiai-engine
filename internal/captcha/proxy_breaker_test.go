package captcha

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerOpensAndFailsFast(t *testing.T) {
	p := NewIPPool(ProxyConfig{Enabled: true, LocalIPs: []string{"10.255.255.1:1080", "10.255.255.2:1080"}, Strategy: "round_robin"})
	for _, n := range p.nodes {
		n.ProbeOK = false
		n.LastProbe = time.Now()
	}
	p.evaluateBreaker()
	if !p.BreakerOpen() {
		t.Fatal("breaker should be open when 0 IPs healthy")
	}
	start := time.Now()
	_, _, err := p.Pick()
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("Pick should fail fast, took %v", time.Since(start))
	}
	if !errors.Is(err, ErrEgressUnavailable) {
		t.Fatalf("want ErrEgressUnavailable, got %v", err)
	}
}

func TestCircuitBreakerClosesOnRecovery(t *testing.T) {
	p := NewIPPool(ProxyConfig{Enabled: true, LocalIPs: []string{"10.255.255.1:1080"}})
	for _, n := range p.nodes {
		n.ProbeOK = false
		n.LastProbe = time.Now()
	}
	p.evaluateBreaker()
	p.nodes[0].ProbeOK = true
	p.evaluateBreaker()
	if p.BreakerOpen() {
		t.Fatal("breaker should close when an IP recovers")
	}
	ep, release, err := p.Pick()
	if err != nil || ep == "" || release == nil {
		t.Fatalf("healthy pool should pick: ep=%q err=%v", ep, err)
	}
	release()
}

func TestHealthyIPsCount(t *testing.T) {
	p := NewIPPool(ProxyConfig{Enabled: true, LocalIPs: []string{"10.255.255.1:1080", "10.255.255.2:1080", "10.255.255.3:1080"}})
	p.nodes[0].ProbeOK = true
	p.nodes[1].ProbeOK = false
	p.nodes[2].ProbeOK = true
	if got := p.HealthyIPs(); got != 2 {
		t.Fatalf("HealthyIPs=%d want 2", got)
	}
}
