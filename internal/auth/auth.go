package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type Identity struct {
	LicenseKey string
	LicenseID  int
	Tier       string
	APIKey     string
	ClientID   string
	UserID     string
	SiteURL    string
}

type ctxKey struct{}

func FromContext(ctx context.Context) *Identity {
	id, _ := ctx.Value(ctxKey{}).(*Identity)
	return id
}

type cachedResult struct {
	identity *Identity
	err      error
	expiry   time.Time
}

type Authenticator struct {
	cfg   *config.Config
	cache sync.Map // key → *cachedResult
}

func New(cfg *config.Config) *Authenticator {
	a := &Authenticator{cfg: cfg}
	go a.cleanLoop()
	return a
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth on health/status/public info endpoints + routes that Bun serves without auth.
		// Browser/session APIs are local tool APIs: loopback may use them without auth,
		// but non-loopback callers must authenticate to avoid accidental remote exposure.
		p := r.URL.Path
		if isLoopbackToolPath(p) {
			if id, err := a.Authenticate(r); err == nil {
				ctx := context.WithValue(r.Context(), ctxKey{}, id)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if isLoopbackRequest(r) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "authentication required for remote browser/session/search API access"})
			return
		}

		if p == "/" || p == "/health" || p == "/api/status" || p == "/dashboard" ||
			// Health / metrics
			p == "/api/health" || strings.HasPrefix(p, "/api/health/") || p == "/api/metrics/browser" ||
			// Public info
			p == "/api/critique/models" || p == "/api/critique/dimensions" ||
			p == "/api/ui-reverse/models" || p == "/api/ui-reverse/operations" ||
			p == "/api/copilot/health" || p == "/api/intelligence/health" ||
			// Extension verify + token handled inside handler (Bun returns 400/401 itself)
			p == "/api/extension/verify" || p == "/api/extension/token" ||
			// Memory, usage, workflow, training, intelligence — Bun has no global auth
			strings.HasPrefix(p, "/api/memory/") ||
			strings.HasPrefix(p, "/api/usage/") ||
			strings.HasPrefix(p, "/api/workflow/") ||
			strings.HasPrefix(p, "/api/training/") || // Has own service-token auth
			strings.HasPrefix(p, "/api/intelligence/") || // Per-handler auth
			strings.HasPrefix(p, "/api/tools") || // Tool discovery: agents search/discover tools
			p == "/api/media/jobs" || // Media job list: read-only
			strings.HasPrefix(p, "/api/media/status/") || // Media status: read-only poll
			strings.HasPrefix(p, "/api/share/") { // Share viewing is public
			// Try to extract identity if credentials present, but don't block
			if id, err := a.Authenticate(r); err == nil {
				ctx := context.WithValue(r.Context(), ctxKey{}, id)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		id, err := a.Authenticate(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		ctx := context.WithValue(r.Context(), ctxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isLoopbackToolPath(path string) bool {
	return path == "/api/screenshot" || strings.HasPrefix(path, "/api/screenshot/") || strings.HasPrefix(path, "/api/session") || path == "/api/search" || strings.HasPrefix(path, "/api/search/") || strings.HasPrefix(path, "/api/media/frame/")
}

func isLoopbackRequest(r *http.Request) bool {
	host := r.RemoteAddr
	if splitHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = splitHost
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (a *Authenticator) Authenticate(r *http.Request) (*Identity, error) {
	// Internal service auth via webhook secret (server-to-server)
	if ws := r.Header.Get("X-Webhook-Secret"); ws != "" && ws == a.cfg.WordPress.WebhookSecret {
		return &Identity{Tier: "internal", LicenseID: 0}, nil
	}
	if lk := r.Header.Get("X-License-Key"); lk != "" {
		if id, ok := validateLocalToken(lk); ok {
			return id, nil
		}
		return a.validateLicense(lk)
	}
	if ak := r.Header.Get("X-API-Key"); ak != "" {
		if id, ok := validateLocalToken(ak); ok {
			return id, nil
		}
		return a.validateAPIKey(ak)
	}
	if et := r.Header.Get("X-Extension-Token"); et != "" {
		return a.validateExtToken(et)
	}
	if ah := r.Header.Get("Authorization"); len(ah) > 7 {
		parts := strings.SplitN(ah, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			bearer := strings.TrimSpace(parts[1])
			if id, ok := validateLocalToken(bearer); ok {
				return id, nil
			}
			return a.validateLicense(bearer)
		}
	}
	return nil, fmt.Errorf("missing authentication header")
}

func validateLocalToken(token string) (*Identity, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, false
	}
	for _, candidate := range localAuthTokens() {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(candidate)) == 1 {
			return &Identity{APIKey: token, ClientID: "local-vps", Tier: "internal"}, true
		}
	}
	return nil, false
}

func localAuthTokens() []string {
	var tokens []string
	if single := os.Getenv("UIAI_LOCAL_API_TOKEN"); single != "" {
		tokens = append(tokens, single)
	}
	if multiple := os.Getenv("UIAI_LOCAL_API_TOKENS"); multiple != "" {
		tokens = append(tokens, strings.Split(multiple, ",")...)
	}
	return tokens
}

func (a *Authenticator) validateLicense(key string) (*Identity, error) {
	cacheKey := "lic:" + key
	if v, ok := a.cache.Load(cacheKey); ok {
		cr := v.(*cachedResult)
		if time.Now().Before(cr.expiry) {
			return cr.identity, cr.err
		}
		a.cache.Delete(cacheKey)
	}

	url := a.cfg.RESTURL("license/validate")
	body := fmt.Sprintf(`{"license_key":"%s"}`, key)
	resp, err := httpPost(url, body, a.cfg.WordPress.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("license validation failed: %w", err)
	}

	var data struct {
		Valid     bool   `json:"valid"`
		LicenseID int    `json:"license_id"`
		Tier      string `json:"tier"`
		Status    string `json:"status"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("invalid license response")
	}
	if !data.Valid {
		e := fmt.Errorf("invalid license: %s", data.Error)
		a.cache.Store(cacheKey, &cachedResult{nil, e, time.Now().Add(60 * time.Second)})
		return nil, e
	}

	id := &Identity{LicenseKey: key, LicenseID: data.LicenseID, Tier: data.Tier}
	ttl := time.Duration(a.cfg.WordPress.CacheTTL) * time.Second
	a.cache.Store(cacheKey, &cachedResult{id, nil, time.Now().Add(ttl)})
	return id, nil
}

func (a *Authenticator) validateAPIKey(key string) (*Identity, error) {
	cacheKey := "api:" + key
	if v, ok := a.cache.Load(cacheKey); ok {
		cr := v.(*cachedResult)
		if time.Now().Before(cr.expiry) {
			return cr.identity, cr.err
		}
		a.cache.Delete(cacheKey)
	}

	url := a.cfg.RESTURL("keys/validate")
	body := fmt.Sprintf(`{"api_key":"%s"}`, key)
	resp, err := httpPost(url, body, a.cfg.WordPress.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}

	var data struct {
		Valid     bool   `json:"valid"`
		ClientID  string `json:"client_id"`
		LicenseID int    `json:"license_id"`
		Tier      string `json:"tier"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("invalid key response")
	}
	if !data.Valid {
		e := fmt.Errorf("invalid API key")
		a.cache.Store(cacheKey, &cachedResult{nil, e, time.Now().Add(60 * time.Second)})
		return nil, e
	}

	id := &Identity{APIKey: key, ClientID: data.ClientID, LicenseID: data.LicenseID, Tier: data.Tier}
	ttl := time.Duration(a.cfg.WordPress.CacheTTL) * time.Second
	a.cache.Store(cacheKey, &cachedResult{id, nil, time.Now().Add(ttl)})
	return id, nil
}

func (a *Authenticator) validateExtToken(token string) (*Identity, error) {
	cacheKey := "ext:" + token
	if v, ok := a.cache.Load(cacheKey); ok {
		cr := v.(*cachedResult)
		if time.Now().Before(cr.expiry) {
			return cr.identity, cr.err
		}
		a.cache.Delete(cacheKey)
	}

	// Validate against the extension token endpoint on the Go engine itself.
	// Extension tokens are issued by /api/extension/token and stored in extTokens
	// (routes/extension.go). We validate by checking the in-memory store via
	// the verify endpoint path, but since we're in the same process, we
	// directly call the token store. For cross-process, this would be a JWT.
	url := fmt.Sprintf("http://127.0.0.1:%d/api/extension/verify", a.cfg.Server.Port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Extension-Token", token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("extension token validation failed: %w", err)
	}
	defer resp.Body.Close()

	var data struct {
		Valid  bool     `json:"valid"`
		UserID string   `json:"userId"`
		Scope  []string `json:"scope"`
		Error  string   `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if !data.Valid {
		e := fmt.Errorf("invalid extension token: %s", data.Error)
		a.cache.Store(cacheKey, &cachedResult{nil, e, time.Now().Add(30 * time.Second)})
		return nil, e
	}

	id := &Identity{
		Tier:   "pro", // extension tokens are always at least pro tier
		UserID: data.UserID,
	}
	a.cache.Store(cacheKey, &cachedResult{id, nil, time.Now().Add(5 * time.Minute)})
	return id, nil
}

func (a *Authenticator) cleanLoop() {
	for {
		time.Sleep(60 * time.Second)
		now := time.Now()
		a.cache.Range(func(key, value any) bool {
			if cr, ok := value.(*cachedResult); ok && now.After(cr.expiry) {
				a.cache.Delete(key)
			}
			return true
		})
	}
}

func httpPost(url, body, secret string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	req.Body = http.NoBody
	if body != "" {
		req.Body = nopCloser([]byte(body))
		req.ContentLength = int64(len(body))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf [8192]byte
	n, _ := resp.Body.Read(buf[:])
	return buf[:n], nil
}

type nopReadCloser struct {
	data []byte
	off  int
}

func nopCloser(b []byte) *nopReadCloser { return &nopReadCloser{data: b} }
func (r *nopReadCloser) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	if r.off >= len(r.data) {
		return n, io.EOF
	}
	return n, nil
}
func (r *nopReadCloser) Close() error { return nil }
