package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// ─── IP Pool for fleet-scale captcha solving ───────────────────────────────
//
// This is a service — the IP pool must handle:
//   - Dozens of IPs ($1/mo each, add via API or config)
//   - Per-IP health tracking (success rate, flagged state, cooldown)
//   - Weighted rotation (prefer healthy IPs)
//   - Automatic cooldown on flag detection
//   - Concurrent solve limits per IP
//   - Runtime add/remove via API (no restart)
//   - Stats + observability per IP

// ─── Config ────────────────────────────────────────────────────────────────

// ProxyConfig holds IP pool settings loaded from config.yaml.
type ProxyConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	LocalIPs           []string `yaml:"local_ips" json:"local_ips"`
	Proxies            []string `yaml:"proxies" json:"proxies"`
	Strategy           string   `yaml:"strategy" json:"strategy"`                           // "weighted" | "least_conn" | "round_robin" | "random"
	MaxConcurrentPerIP int      `yaml:"max_concurrent_per_ip" json:"max_concurrent_per_ip"` // default 2
	CooldownMinutes    int      `yaml:"cooldown_minutes" json:"cooldown_minutes"`           // default 60
	HealthFile         string   `yaml:"health_file" json:"health_file"`
	HealthProbeURL     string   `yaml:"health_probe_url" json:"health_probe_url"`         // URL to probe (default: https://www.google.com/recaptcha/api.js)
	HealthProbeSeconds int      `yaml:"health_probe_seconds" json:"health_probe_seconds"` // probe interval (default: 300 = 5min)
	MaxRetries         int      `yaml:"max_retries" json:"max_retries"`                   // auto-retry on different IP (default: 2)
}

// ─── IP Pool ───────────────────────────────────────────────────────────────

// IPPool manages a fleet of outgoing IPs with health tracking.
type IPPool struct {
	mu       sync.RWMutex
	nodes    []*IPNode
	index    int // for round_robin
	config   ProxyConfig
	socksMap map[string]*socksProxy // local IP → running SOCKS5 listener
	stopCh   chan struct{}          // stops the probe loop on shutdown
}

// IPNode is a single IP endpoint with health state.
type IPNode struct {
	// Identity
	Endpoint string `json:"endpoint"` // "local:199.167.201.52" or "socks5://..."
	Label    string `json:"label"`    // human-readable

	// Health
	TotalAttempts int       `json:"total_attempts"`
	TotalSolved   int       `json:"total_solved"`
	TotalFailed   int       `json:"total_failed"`
	SuccessRate   float64   `json:"success_rate"`
	Flagged       bool      `json:"flagged"`
	FlaggedAt     time.Time `json:"flagged_at"`
	CooldownUntil time.Time `json:"cooldown_until"`
	LastUsed      time.Time `json:"last_used"`
	LastSuccess   time.Time `json:"last_success"`
	LastError     string    `json:"last_error"`

	// Active health probes
	LastProbe time.Time `json:"last_probe"`
	ProbeOK   bool      `json:"probe_ok"`

	// Concurrency
	Active int `json:"active"`
}

type socksProxy struct {
	listener net.Listener
	localIP  string
	addr     string // "127.0.0.1:PORT"
}

// NewIPPool creates a pool from config, restoring health from disk if available.
func NewIPPool(cfg ProxyConfig) *IPPool {
	pool := &IPPool{
		config:   cfg,
		socksMap: make(map[string]*socksProxy),
		stopCh:   make(chan struct{}),
	}
	if pool.config.MaxConcurrentPerIP <= 0 {
		pool.config.MaxConcurrentPerIP = 2
	}
	if pool.config.CooldownMinutes <= 0 {
		pool.config.CooldownMinutes = 60
	}
	if pool.config.Strategy == "" {
		pool.config.Strategy = "weighted"
	}
	if pool.config.HealthProbeURL == "" {
		pool.config.HealthProbeURL = "https://www.google.com/recaptcha/api.js"
	}
	if pool.config.HealthProbeSeconds <= 0 {
		pool.config.HealthProbeSeconds = 300 // 5 min
	}
	if pool.config.MaxRetries <= 0 {
		pool.config.MaxRetries = 2
	}

	// Add local IPs
	for _, ip := range cfg.LocalIPs {
		pool.nodes = append(pool.nodes, &IPNode{
			Endpoint: "local:" + ip,
			Label:    ip,
			ProbeOK:  true, // assume healthy until proven otherwise
		})
	}
	// Add external proxies
	for _, p := range cfg.Proxies {
		pool.nodes = append(pool.nodes, &IPNode{
			Endpoint: p,
			Label:    maskEndpoint(p),
			ProbeOK:  true,
		})
	}

	// Restore health from disk
	pool.loadHealth()

	// Start active health probe goroutine
	go pool.probeLoop()

	log.Printf("[ip-pool] Initialized: %d endpoints (%d local, %d external), strategy=%s, max_concurrent=%d, cooldown=%dm, probe_interval=%ds, max_retries=%d",
		len(pool.nodes), len(cfg.LocalIPs), len(cfg.Proxies),
		pool.config.Strategy, pool.config.MaxConcurrentPerIP, pool.config.CooldownMinutes,
		pool.config.HealthProbeSeconds, pool.config.MaxRetries)

	return pool
}

// ─── Pool operations ───────────────────────────────────────────────────────

// Pick selects the best available IP and increments its active count.
// Returns the endpoint string and a release func the caller MUST defer.
func (p *IPPool) Pick() (string, func(), error) {
	return p.PickExcluding(nil)
}

// PickExcluding selects an IP, skipping any in the exclude set (for retries).
func (p *IPPool) PickExcluding(exclude map[string]bool) (string, func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	available := p.availableNodes()
	// Filter out excluded endpoints
	if len(exclude) > 0 {
		var filtered []*IPNode
		for _, n := range available {
			if !exclude[n.Endpoint] {
				filtered = append(filtered, n)
			}
		}
		available = filtered
	}

	if len(available) == 0 {
		return "", nil, fmt.Errorf("no healthy IPs available (%d total, all flagged/busy/cooling/excluded)", len(p.nodes))
	}

	var node *IPNode
	switch p.config.Strategy {
	case "random":
		node = available[secureIntn(len(available))]
	case "round_robin":
		node = available[p.index%len(available)]
		p.index++
	case "least_conn":
		node = p.pickLeastConn(available)
	default: // "weighted"
		node = p.pickWeighted(available)
	}

	node.Active++
	node.LastUsed = time.Now()
	endpoint := node.Endpoint

	release := func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		node.Active--
	}

	return endpoint, release, nil
}

// ReportResult records a solve attempt's outcome on the IP.
func (p *IPPool) ReportResult(endpoint string, solved bool, flagged bool, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	node := p.findNode(endpoint)
	if node == nil {
		return
	}

	node.TotalAttempts++
	if solved {
		node.TotalSolved++
		node.LastSuccess = time.Now()
		// Un-flag on success (IP recovered)
		node.Flagged = false
		node.CooldownUntil = time.Time{}
	} else {
		node.TotalFailed++
		node.LastError = errMsg
	}

	if flagged {
		node.Flagged = true
		node.FlaggedAt = time.Now()
		node.CooldownUntil = time.Now().Add(time.Duration(p.config.CooldownMinutes) * time.Minute)
		log.Printf("[ip-pool] IP %s FLAGGED — cooldown until %s", node.Label, node.CooldownUntil.Format("15:04:05"))
	}

	// Recalc success rate
	if node.TotalAttempts > 0 {
		node.SuccessRate = float64(node.TotalSolved) / float64(node.TotalAttempts)
	}

	// Persist health
	p.saveHealth()
}

// AddIP adds an IP to the pool at runtime (no restart needed).
func (p *IPPool) AddIP(endpoint string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Dedup
	for _, n := range p.nodes {
		if n.Endpoint == endpoint {
			return fmt.Errorf("endpoint %s already in pool", endpoint)
		}
	}

	p.nodes = append(p.nodes, &IPNode{
		Endpoint: endpoint,
		Label:    maskEndpoint(endpoint),
	})
	log.Printf("[ip-pool] Added endpoint: %s (total: %d)", maskEndpoint(endpoint), len(p.nodes))
	p.saveHealth()
	return nil
}

// RemoveIP removes an IP from the pool.
func (p *IPPool) RemoveIP(endpoint string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, n := range p.nodes {
		if n.Endpoint == endpoint {
			// Close SOCKS proxy if local
			if sp, ok := p.socksMap[endpoint]; ok {
				sp.listener.Close()
				delete(p.socksMap, endpoint)
			}
			p.nodes = append(p.nodes[:i], p.nodes[i+1:]...)
			log.Printf("[ip-pool] Removed endpoint: %s (total: %d)", maskEndpoint(endpoint), len(p.nodes))
			p.saveHealth()
			return nil
		}
	}
	return fmt.Errorf("endpoint %s not found", endpoint)
}

// Status returns pool state for the API.
func (p *IPPool) Status() IPPoolStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	s := IPPoolStatus{
		TotalIPs:      len(p.nodes),
		Strategy:      p.config.Strategy,
		MaxPerIP:      p.config.MaxConcurrentPerIP,
		CooldownMin:   p.config.CooldownMinutes,
		MaxRetries:    p.config.MaxRetries,
		ProbeURL:      p.config.HealthProbeURL,
		ProbeInterval: p.config.HealthProbeSeconds,
		IPs:           make([]IPNodeStatus, 0, len(p.nodes)),
	}

	for _, n := range p.nodes {
		status := "healthy"
		if !n.ProbeOK && !n.LastProbe.IsZero() {
			status = "probe_failed"
		} else if n.Flagged && time.Now().Before(n.CooldownUntil) {
			status = "cooling"
		} else if n.Flagged {
			status = "flagged_expired"
		}
		if n.Active >= p.config.MaxConcurrentPerIP {
			status = "busy"
		}
		s.IPs = append(s.IPs, IPNodeStatus{
			Endpoint:      n.Endpoint,
			Label:         n.Label,
			Status:        status,
			Active:        n.Active,
			TotalAttempts: n.TotalAttempts,
			TotalSolved:   n.TotalSolved,
			SuccessRate:   n.SuccessRate,
			Flagged:       n.Flagged,
			CooldownUntil: n.CooldownUntil,
			LastUsed:      n.LastUsed,
			LastError:     n.LastError,
			ProbeOK:       n.ProbeOK,
			LastProbe:     n.LastProbe,
		})
	}
	return s
}

// IPPoolStatus is the API response for pool state.
type IPPoolStatus struct {
	TotalIPs      int            `json:"total_ips"`
	Strategy      string         `json:"strategy"`
	MaxPerIP      int            `json:"max_concurrent_per_ip"`
	CooldownMin   int            `json:"cooldown_minutes"`
	MaxRetries    int            `json:"max_retries"`
	ProbeURL      string         `json:"probe_url"`
	ProbeInterval int            `json:"probe_interval_seconds"`
	IPs           []IPNodeStatus `json:"ips"`
}

type IPNodeStatus struct {
	Endpoint      string    `json:"endpoint"`
	Label         string    `json:"label"`
	Status        string    `json:"status"` // "healthy" | "cooling" | "flagged_expired" | "busy" | "probe_failed"
	Active        int       `json:"active"`
	TotalAttempts int       `json:"total_attempts"`
	TotalSolved   int       `json:"total_solved"`
	SuccessRate   float64   `json:"success_rate"`
	Flagged       bool      `json:"flagged"`
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
	LastUsed      time.Time `json:"last_used,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	ProbeOK       bool      `json:"probe_ok"`
	LastProbe     time.Time `json:"last_probe,omitempty"`
}

// ─── Internal ──────────────────────────────────────────────────────────────

func (p *IPPool) availableNodes() []*IPNode {
	var result []*IPNode
	now := time.Now()
	for _, n := range p.nodes {
		// Skip if at max concurrency
		if n.Active >= p.config.MaxConcurrentPerIP {
			continue
		}
		// Skip if flagged and still in cooldown
		if n.Flagged && now.Before(n.CooldownUntil) {
			continue
		}
		// Skip if active probe failed (IP unreachable)
		if !n.ProbeOK && !n.LastProbe.IsZero() {
			continue
		}
		result = append(result, n)
	}
	return result
}

// pickLeastConn selects the node with the fewest active connections.
// Ties broken by higher success rate.
func (p *IPPool) pickLeastConn(nodes []*IPNode) *IPNode {
	best := nodes[0]
	for _, n := range nodes[1:] {
		if n.Active < best.Active {
			best = n
		} else if n.Active == best.Active && n.SuccessRate > best.SuccessRate {
			best = n
		}
	}
	return best
}

func (p *IPPool) pickWeighted(nodes []*IPNode) *IPNode {
	if len(nodes) == 1 {
		return nodes[0]
	}

	// Score: higher success rate + lower active count = better
	// New IPs (0 attempts) get a bonus to be tried
	type scored struct {
		node  *IPNode
		score float64
	}
	var candidates []scored
	for _, n := range nodes {
		score := 1.0
		if n.TotalAttempts > 0 {
			score = n.SuccessRate
		} else {
			score = 0.8 // new IP bonus — try it
		}
		// Penalize busy
		score -= float64(n.Active) * 0.3
		// Penalize recently flagged (but cooldown expired)
		if n.Flagged {
			score -= 0.2
		}
		if score < 0.01 {
			score = 0.01
		}
		candidates = append(candidates, scored{n, score})
	}

	// Weighted random selection
	totalScore := 0.0
	for _, c := range candidates {
		totalScore += c.score
	}
	r := secureFloat64(totalScore)
	cumulative := 0.0
	for _, c := range candidates {
		cumulative += c.score
		if r <= cumulative {
			return c.node
		}
	}
	return candidates[len(candidates)-1].node
}

func (p *IPPool) findNode(endpoint string) *IPNode {
	for _, n := range p.nodes {
		if n.Endpoint == endpoint {
			return n
		}
	}
	return nil
}

func (p *IPPool) loadHealth() {
	if p.config.HealthFile == "" {
		return
	}
	data, err := os.ReadFile(p.config.HealthFile)
	if err != nil {
		return // no file yet, normal on first run
	}
	var saved []IPNode
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("[ip-pool] Failed to load health file: %v", err)
		return
	}
	// Merge saved health into current nodes
	savedMap := make(map[string]*IPNode)
	for i := range saved {
		savedMap[saved[i].Endpoint] = &saved[i]
	}
	for _, n := range p.nodes {
		if s, ok := savedMap[n.Endpoint]; ok {
			n.TotalAttempts = s.TotalAttempts
			n.TotalSolved = s.TotalSolved
			n.TotalFailed = s.TotalFailed
			n.SuccessRate = s.SuccessRate
			n.Flagged = s.Flagged
			n.FlaggedAt = s.FlaggedAt
			n.CooldownUntil = s.CooldownUntil
			n.LastSuccess = s.LastSuccess
			n.LastError = s.LastError
			n.LastProbe = s.LastProbe
			n.ProbeOK = s.ProbeOK
		}
	}
	log.Printf("[ip-pool] Restored health for %d endpoints", len(saved))
}

func (p *IPPool) saveHealth() {
	if p.config.HealthFile == "" {
		return
	}
	data, err := json.MarshalIndent(p.nodes, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(p.config.HealthFile, data, 0600)
}

// ─── Browser launch ────────────────────────────────────────────────────────

// ProxiedBrowser is an ephemeral browser launched through an alternate IP.
type ProxiedBrowser struct {
	browser    *rod.Browser
	launcher   *launcher.Launcher
	socksClean func() // cleanup in-process SOCKS5 (local IPs only)
	label      string
}

// LaunchWithEndpoint starts Chrome bound to the given IP or proxy.
func (p *IPPool) LaunchWithEndpoint(endpoint string) (*ProxiedBrowser, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("empty endpoint")
	}

	chromePath := findChromium()
	l := launcher.New().
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-extensions").
		Set("disable-translate").
		Set("mute-audio").
		Set("disable-component-update").
		Set("disable-domain-reliability").
		Set("disable-crash-reporter").
		Set("disable-background-networking").
		Set("disable-default-apps").
		Set("disable-sync").
		Set("disable-blink-features", "AutomationControlled").
		Set("no-first-run")

	if chromePath != "" {
		l = l.Bin(chromePath)
	}

	pb := &ProxiedBrowser{label: maskEndpoint(endpoint)}

	if strings.HasPrefix(endpoint, "local:") {
		ip := strings.TrimPrefix(endpoint, "local:")
		if parsedIP := net.ParseIP(ip); parsedIP == nil {
			return nil, fmt.Errorf("invalid local IP %q", ip)
		}

		// Reuse or create SOCKS5 proxy for this local IP
		socksAddr, cleanup, err := p.getOrCreateSOCKS(ip)
		if err != nil {
			return nil, fmt.Errorf("SOCKS5 for %s: %w", ip, err)
		}
		l = l.Set("proxy-server", "socks5://"+socksAddr)
		pb.socksClean = cleanup
		pb.label = ip
	} else {
		if _, err := url.Parse(endpoint); err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		l = l.Set("proxy-server", endpoint)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	pb.browser = browser
	pb.launcher = l
	return pb, nil
}

// Close shuts down the proxied browser.
func (pb *ProxiedBrowser) Close() {
	if pb.browser != nil {
		pb.browser.Close()
	}
	if pb.launcher != nil {
		pb.launcher.Cleanup()
	}
	// Note: don't close SOCKS proxy — it's shared across browsers for same IP
	log.Printf("[ip-pool] Closed browser (%s)", pb.label)
}

// OpenPage navigates to a URL with anti-detection and returns a vision.Session.
func (pb *ProxiedBrowser) OpenPage(targetURL string, width, height int, stealth StealthConfig) (*vision.Session, error) {
	page, err := pb.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: width, Height: height, DeviceScaleFactor: 1,
	}); err != nil {
		page.Close()
		return nil, fmt.Errorf("viewport: %w", err)
	}

	// Anti-detection
	if stealth.PatchWebdriver {
		ua := ""
		if stealth.RandomUserAgent && len(stealth.UserAgents) > 0 {
			ua = stealth.UserAgents[secureIntn(len(stealth.UserAgents))]
		}
		stealthJS := `
			Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
			Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']});
			Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
			window.chrome = {runtime: {}, loadTimes: function(){}, csi: function(){}};
			delete navigator.__proto__.webdriver;
		`
		page.MustEvalOnNewDocument(stealthJS)
		if ua != "" {
			page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: ua})
		}
	}

	if err := page.Timeout(25 * time.Second).Navigate(targetURL); err != nil {
		page.Close()
		return nil, fmt.Errorf("navigate: %w", err)
	}
	page.Timeout(5*time.Second).WaitDOMStable(200*time.Millisecond, 0.15)

	return vision.WrapPage(page, targetURL, width, height), nil
}

// ─── Solver integration ────────────────────────────────────────────────────

// SolveViaProxy picks an IP, launches a browser, fills the form, solves captcha.
// On failure, automatically retries on a different IP up to MaxRetries times.
// Reports each attempt back to the pool for health tracking.
func (s *Solver) SolveViaProxy(ctx context.Context, targetURL string, width, height int,
	setup func(sess *vision.Session) error,
	solveReq SolveRequest,
) *SolveResponse {

	if s.pool == nil {
		return &SolveResponse{
			Solved: false,
			Error:  "IP pool not initialized — enable captcha.proxy in config",
			Method: "proxy",
		}
	}

	maxRetries := s.pool.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	tried := make(map[string]bool)
	var lastResult *SolveResponse

	for attempt := 1; attempt <= maxRetries; attempt++ {
		endpoint, release, err := s.pool.PickExcluding(tried)
		if err != nil {
			if lastResult != nil {
				return lastResult // return last failure if we ran out of IPs
			}
			return &SolveResponse{
				Solved: false,
				Error:  fmt.Sprintf("IP pool (attempt %d/%d): %v", attempt, maxRetries, err),
				Method: "proxy",
			}
		}
		tried[endpoint] = true

		result := s.solveOnIP(ctx, endpoint, targetURL, width, height, setup, solveReq)
		release()

		if result.Solved {
			if attempt > 1 {
				log.Printf("[ip-pool] Solved on retry %d/%d via %s", attempt, maxRetries, maskEndpoint(endpoint))
			}
			return result
		}

		lastResult = result
		log.Printf("[ip-pool] Attempt %d/%d failed on %s: %s — retrying on different IP",
			attempt, maxRetries, maskEndpoint(endpoint), result.Error)
	}

	return lastResult
}

// solveOnIP does a single solve attempt on one specific IP.
func (s *Solver) solveOnIP(ctx context.Context, endpoint, targetURL string, width, height int,
	setup func(sess *vision.Session) error,
	solveReq SolveRequest,
) *SolveResponse {

	pb, err := s.pool.LaunchWithEndpoint(endpoint)
	if err != nil {
		s.pool.ReportResult(endpoint, false, false, err.Error())
		return &SolveResponse{
			Solved: false,
			Error:  fmt.Sprintf("browser launch: %v", err),
			Method: "proxy",
		}
	}
	defer pb.Close()

	sess, err := pb.OpenPage(targetURL, width, height, s.Config.Stealth)
	if err != nil {
		flagged := strings.Contains(err.Error(), "ERR_PROXY") || strings.Contains(err.Error(), "timeout")
		s.pool.ReportResult(endpoint, false, flagged, err.Error())
		return &SolveResponse{
			Solved: false,
			Error:  fmt.Sprintf("page open: %v", err),
			Method: "proxy",
		}
	}

	if setup != nil {
		if err := setup(sess); err != nil {
			s.pool.ReportResult(endpoint, false, false, err.Error())
			return &SolveResponse{
				Solved: false,
				Error:  fmt.Sprintf("form setup: %v", err),
				Method: "proxy",
			}
		}
	}

	result := s.SolveInSession(ctx, sess, solveReq)

	// Detect IP flagging from solve result
	flagged := detectFlagged(result.Error)
	s.pool.ReportResult(endpoint, result.Solved, flagged, result.Error)

	return result
}

// detectFlagged checks error messages for IP-flagging indicators.
func detectFlagged(errMsg string) bool {
	if errMsg == "" {
		return false
	}
	lower := strings.ToLower(errMsg)
	return strings.Contains(lower, "automated queries") ||
		strings.Contains(lower, "unusual traffic") ||
		strings.Contains(lower, "blocked") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "forbidden")
}

// ─── Active health probes ──────────────────────────────────────────────────

// probeLoop periodically tests each IP can reach the target (e.g. Google reCAPTCHA).
// Runs in a goroutine, stopped via pool.stopCh.
func (p *IPPool) probeLoop() {
	// Initial probe after 10s startup grace
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-timer.C:
			p.probeAll()
			timer.Reset(time.Duration(p.config.HealthProbeSeconds) * time.Second)
		}
	}
}

// probeAll tests every IP in the pool.
func (p *IPPool) probeAll() {
	p.mu.RLock()
	nodes := make([]*IPNode, len(p.nodes))
	copy(nodes, p.nodes)
	p.mu.RUnlock()

	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(node *IPNode) {
			defer wg.Done()
			ok := p.probeOne(node.Endpoint)

			p.mu.Lock()
			node.LastProbe = time.Now()
			node.ProbeOK = ok
			if !ok {
				log.Printf("[ip-pool] Probe FAILED: %s", node.Label)
			}
			p.mu.Unlock()
		}(n)
	}
	wg.Wait()
	p.saveHealth()
}

// probeOne tests if an IP can reach the health probe URL.
func (p *IPPool) probeOne(endpoint string) bool {
	var dialer *net.Dialer

	if strings.HasPrefix(endpoint, "local:") {
		ip := strings.TrimPrefix(endpoint, "local:")
		dialer = &net.Dialer{
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(ip)},
			Timeout:   10 * time.Second,
		}
	} else {
		// External proxy — just check TCP connectivity to the proxy host
		u, err := url.Parse(endpoint)
		if err != nil {
			return false
		}
		conn, err := net.DialTimeout("tcp", u.Host, 10*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}

	// For local IPs, do an actual HTTP GET through that IP
	transport := &net.Dialer{
		LocalAddr: dialer.LocalAddr,
		Timeout:   10 * time.Second,
	}
	conn, err := transport.Dial("tcp", "www.google.com:443")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Stop shuts down the probe loop.
func (p *IPPool) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// ─── SOCKS5 management ────────────────────────────────────────────────────

func (p *IPPool) getOrCreateSOCKS(localIP string) (string, func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := "local:" + localIP
	if sp, ok := p.socksMap[key]; ok {
		return sp.addr, func() {}, nil // reuse existing, no-op cleanup
	}

	srcAddr := &net.TCPAddr{IP: net.ParseIP(localIP)}
	dialer := &net.Dialer{LocalAddr: srcAddr, Timeout: 15 * time.Second}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	addr := listener.Addr().String()
	sp := &socksProxy{listener: listener, localIP: localIP, addr: addr}
	p.socksMap[key] = sp

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleSOCKS5(conn, dialer)
		}
	}()

	log.Printf("[ip-pool] SOCKS5 started: %s → outgoing %s", addr, localIP)
	return addr, func() {}, nil
}

// handleSOCKS5 implements minimal SOCKS5 CONNECT (no auth, IPv4/domain).
func handleSOCKS5(conn net.Conn, dialer *net.Dialer) {
	defer conn.Close()

	buf := make([]byte, 258)
	n, err := conn.Read(buf)
	if err != nil || n < 3 || buf[0] != 0x05 {
		return
	}
	conn.Write([]byte{0x05, 0x00}) // no auth

	n, err = conn.Read(buf)
	if err != nil || n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var target string
	switch buf[3] {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		target = fmt.Sprintf("%s:%d", net.IPv4(buf[4], buf[5], buf[6], buf[7]), int(buf[8])<<8|int(buf[9]))
	case 0x03: // Domain
		dLen := int(buf[4])
		if n < 5+dLen+2 {
			return
		}
		target = fmt.Sprintf("%s:%d", string(buf[5:5+dLen]), int(buf[5+dLen])<<8|int(buf[5+dLen+1]))
	case 0x04: // IPv6
		if n < 22 {
			return
		}
		ip := net.IP(buf[4:20])
		target = fmt.Sprintf("[%s]:%d", ip, int(buf[20])<<8|int(buf[21]))
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	remote, err := dialer.Dial("tcp", target)
	if err != nil {
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()

	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	done := make(chan struct{}, 2)
	relay := func(dst, src net.Conn) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go relay(remote, conn)
	go relay(conn, remote)
	<-done
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func findChromium() string {
	for _, c := range []string{
		"/usr/lib64/chromium-browser/chromium-browser",
		"/usr/lib/chromium-browser/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func maskEndpoint(endpoint string) string {
	if strings.HasPrefix(endpoint, "local:") {
		return strings.TrimPrefix(endpoint, "local:")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "***"
	}
	host := u.Hostname()
	if len(host) > 8 {
		return host[:4] + "..." + host[len(host)-4:]
	}
	return host
}
