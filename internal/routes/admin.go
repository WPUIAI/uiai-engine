package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/ratelimit"
	"github.com/WPUIAI/uiai-engine/internal/storage"
	"github.com/go-chi/chi/v5"
)

var startTime = time.Now()

// In-memory API key store (matches Bun behavior — keys managed via WP, cached here)
var apiKeyStore sync.Map // keyHash → *apiKeyEntry

type apiKeyEntry struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	KeyHash    string    `json:"key_hash"`
	KeyPrefix  string    `json:"key_prefix"` // first 8 chars for display
	Tier       string    `json:"tier"`
	SiteURL    string    `json:"site_url"`
	Scopes     []string  `json:"scopes"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsed   time.Time `json:"last_used,omitempty"`
	UsageCount int       `json:"usage_count"`
}

func MountAdminReal(r chi.Router, cfg *config.Config, usage *storage.UsageStore, limiter *ratelimit.Limiter, authenticator *auth.Authenticator) {
	// --- Dashboard ---
	r.Get("/dashboard", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		byType := map[string]int{}
		totalCost := 0.0
		for _, rec := range records {
			byType[rec.Type]++
			totalCost += rec.CostUSD
		}

		// Usage in last 24h
		cutoff24 := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		last24h := 0
		cost24h := 0.0
		for _, rec := range records {
			if rec.CreatedAt >= cutoff24 {
				last24h++
				cost24h += rec.CostUSD
			}
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		writeJSON(w, 200, map[string]any{
			"uptime":          time.Since(startTime).String(),
			"uptime_seconds":  int(time.Since(startTime).Seconds()),
			"status":          "healthy",
			"go_version":      runtime.Version(),
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,
			"memory_sys_mb":   float64(m.Sys) / 1024 / 1024,
			"total_requests":  len(records),
			"total_cost":      totalCost,
			"last_24h": map[string]any{
				"requests": last24h,
				"cost":     cost24h,
			},
			"by_type":          byType,
			"rate_limit_tiers": cfg.RateLimits.Tiers,
		})
	})

	// --- Services ---
	r.Get("/services", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"uiai-engine": map[string]any{
				"id": "uiai-engine", "name": "UIAI Engine", "running": true,
				"port": cfg.Server.Port, "health": "healthy", "uptime": time.Since(startTime).String(),
				"version": "2.0.0", "go_version": runtime.Version(),
			},
		})
	})

	// --- Resources ---
	r.Get("/resources", func(w http.ResponseWriter, req *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		writeJSON(w, 200, map[string]any{
			"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,
			"memory_sys_mb":   float64(m.Sys) / 1024 / 1024,
			"memory_total_mb": float64(m.TotalAlloc) / 1024 / 1024,
			"gc_runs":         m.NumGC,
			"goroutines":      runtime.NumGoroutine(),
			"uptime":          time.Since(startTime).String(),
			"go_version":      runtime.Version(),
			"num_cpu":         runtime.NumCPU(),
		})
	})

	// --- Usage (full) ---
	r.Get("/usage", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		writeJSON(w, 200, map[string]any{"total": len(records)})
	})

	// --- Usage aggregation (ss-api-rty) ---
	r.Get("/usage/aggregate", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		period := req.URL.Query().Get("period")
		if period == "" {
			period = "daily"
		}

		type bucket struct {
			Requests int     `json:"requests"`
			Cost     float64 `json:"cost"`
			Tokens   int     `json:"tokens"`
		}
		agg := map[string]*bucket{}

		for _, r := range records {
			var key string
			switch period {
			case "hourly":
				if len(r.CreatedAt) >= 13 {
					key = r.CreatedAt[:13] // YYYY-MM-DDTHH
				}
			case "weekly":
				if len(r.CreatedAt) >= 10 {
					t, _ := time.Parse("2006-01-02", r.CreatedAt[:10])
					year, week := t.ISOWeek()
					key = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (week-1)*7).Format("2006-01-02")
				}
			default: // daily
				if len(r.CreatedAt) >= 10 {
					key = r.CreatedAt[:10]
				}
			}
			if key == "" {
				continue
			}
			b, ok := agg[key]
			if !ok {
				b = &bucket{}
				agg[key] = b
			}
			b.Requests++
			b.Cost += r.CostUSD
			b.Tokens += r.InputTokens + r.OutputTokens
		}

		// Sort by key
		keys := make([]string, 0, len(agg))
		for k := range agg {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		type row struct {
			Period   string  `json:"period"`
			Requests int     `json:"requests"`
			Cost     float64 `json:"cost"`
			Tokens   int     `json:"tokens"`
		}
		var rows []row
		for _, k := range keys {
			b := agg[k]
			rows = append(rows, row{Period: k, Requests: b.Requests, Cost: b.Cost, Tokens: b.Tokens})
		}

		writeJSON(w, 200, map[string]any{"period": period, "data": rows, "total_periods": len(rows)})
	})

	// --- Usage by type breakdown ---
	r.Get("/usage/breakdown", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		byType := map[string]map[string]any{}
		for _, r := range records {
			t := r.Type
			if _, ok := byType[t]; !ok {
				byType[t] = map[string]any{"type": t, "requests": 0, "cost": 0.0, "tokens": 0}
			}
			byType[t]["requests"] = byType[t]["requests"].(int) + 1
			byType[t]["cost"] = byType[t]["cost"].(float64) + r.CostUSD
			byType[t]["tokens"] = byType[t]["tokens"].(int) + r.InputTokens + r.OutputTokens
		}
		types := make([]any, 0, len(byType))
		for _, v := range byType {
			types = append(types, v)
		}
		writeJSON(w, 200, map[string]any{"breakdown": types})
	})

	// --- Recent usage ---
	r.Get("/recent-usage", func(w http.ResponseWriter, req *http.Request) {
		all := usage.All()
		n := 50
		if len(all) < n {
			n = len(all)
		}
		writeJSON(w, 200, map[string]any{"records": all[len(all)-n:], "count": n})
	})

	// --- Rate limits (ss-api-jwq) ---
	r.Get("/rate-limits", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"tiers": cfg.RateLimits.Tiers,
		})
	})

	r.Get("/rate-limits/{key}", func(w http.ResponseWriter, req *http.Request) {
		key := chi.URLParam(req, "key")
		tier := req.URL.Query().Get("tier")
		if tier == "" {
			tier = "free"
		}
		hourUsed, hourLimit, dayUsed, dayLimit := limiter.Status(key, tier)
		writeJSON(w, 200, map[string]any{
			"key":    key,
			"tier":   tier,
			"hourly": map[string]int{"used": hourUsed, "limit": hourLimit},
			"daily":  map[string]int{"used": dayUsed, "limit": dayLimit},
		})
	})

	// --- API Key management (ss-api-bur) ---
	r.Post("/keys", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Name    string   `json:"name"`
			Tier    string   `json:"tier"`
			SiteURL string   `json:"site_url"`
			Scopes  []string `json:"scopes"`
			TTLDays int      `json:"ttl_days"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Name == "" {
			writeJSON(w, 400, map[string]string{"error": "name required"})
			return
		}
		if body.Tier == "" {
			body.Tier = "pro"
		}
		if len(body.Scopes) == 0 {
			body.Scopes = []string{"critique", "ui-reverse", "copilot", "screenshot"}
		}
		if body.TTLDays <= 0 {
			body.TTLDays = 365
		}

		keyBytes := make([]byte, 32)
		rand.Read(keyBytes)
		rawKey := "uiai_" + hex.EncodeToString(keyBytes)

		idBytes := make([]byte, 8)
		rand.Read(idBytes)

		entry := &apiKeyEntry{
			ID:        hex.EncodeToString(idBytes),
			Name:      body.Name,
			KeyHash:   rawKey, // In production, would hash this
			KeyPrefix: rawKey[:13],
			Tier:      body.Tier,
			SiteURL:   body.SiteURL,
			Scopes:    body.Scopes,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().AddDate(0, 0, body.TTLDays),
		}
		apiKeyStore.Store(entry.ID, entry)

		writeJSON(w, 201, map[string]any{
			"id":         entry.ID,
			"name":       entry.Name,
			"key":        rawKey, // Only shown once
			"key_prefix": entry.KeyPrefix,
			"tier":       entry.Tier,
			"scopes":     entry.Scopes,
			"expires_at": entry.ExpiresAt,
		})
	})

	r.Get("/keys", func(w http.ResponseWriter, req *http.Request) {
		var keys []any
		apiKeyStore.Range(func(_, v any) bool {
			e := v.(*apiKeyEntry)
			keys = append(keys, map[string]any{
				"id": e.ID, "name": e.Name, "key_prefix": e.KeyPrefix,
				"tier": e.Tier, "scopes": e.Scopes, "site_url": e.SiteURL,
				"created_at": e.CreatedAt, "expires_at": e.ExpiresAt,
				"last_used": e.LastUsed, "usage_count": e.UsageCount,
			})
			return true
		})
		if keys == nil {
			keys = []any{}
		}
		writeJSON(w, 200, map[string]any{"keys": keys, "count": len(keys)})
	})

	r.Delete("/keys/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if _, ok := apiKeyStore.LoadAndDelete(id); ok {
			writeJSON(w, 200, map[string]string{"message": "key revoked"})
		} else {
			writeJSON(w, 404, map[string]string{"error": "key not found"})
		}
	})

	r.Get("/tokens", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"tokens": []any{}, "count": 0})
	})

	// --- Upgrade prompts for free tier (ss-api-7ai) ---
	r.Get("/upgrade-prompts", func(w http.ResponseWriter, req *http.Request) {
		prompts := []map[string]any{
			{
				"trigger": "rate_limited",
				"message": "You've reached your free tier limit. Upgrade to Pro for 50x more AI calls.",
				"cta":     "Upgrade to Pro",
				"url":     "https://wpuiai.com/pricing/",
				"tier":    "free",
			},
			{
				"trigger": "vision_blocked",
				"message": "Cloud screenshots require Pro or higher. Upgrade to unlock AI-powered visual analysis.",
				"cta":     "Unlock Vision",
				"url":     "https://wpuiai.com/pricing/",
				"tier":    "free",
			},
			{
				"trigger": "credits_low",
				"message": "Running low on credits. Top up or upgrade for more monthly credits.",
				"cta":     "Top Up Credits",
				"url":     "https://wpuiai.com/account/credits/",
				"tier":    "starter",
			},
			{
				"trigger": "media_blocked",
				"message": "Media production (mockups, GIFs) requires Agency tier. Upgrade to create professional marketing assets.",
				"cta":     "Upgrade to Agency",
				"url":     "https://wpuiai.com/pricing/",
				"tier":    "pro",
			},
		}
		writeJSON(w, 200, map[string]any{"prompts": prompts})
	})

	// --- Config summary (non-sensitive) ---
	r.Get("/config", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"server_port":      cfg.Server.Port,
			"wp_url":           cfg.WordPress.URL,
			"vision_pool":      cfg.Vision.PoolSize,
			"vision_max_pool":  cfg.Vision.MaxPool,
			"cors_origins":     cfg.CORS.Origins,
			"rate_limit_tiers": cfg.RateLimits.Tiers,
			"default_provider": cfg.AI.DefaultProvider,
			"default_model":    cfg.AI.DefaultModel,
			"providers":        providerNames(cfg),
		})
	})
}

func providerNames(cfg *config.Config) []string {
	var names []string
	for name := range cfg.AI.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HandleDashboard serves a simple HTML dashboard page.
func HandleDashboard(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html><head><title>UIAI Engine Dashboard</title>
<style>body{font-family:system-ui;max-width:900px;margin:2em auto;padding:0 1em;color:#1e293b;background:#f8fafc}
h1{color:#059669}pre{background:#1e293b;color:#10b981;padding:1em;border-radius:8px;overflow-x:auto}
.card{background:white;border:1px solid #e2e8f0;border-radius:8px;padding:1em;margin:1em 0}
</style></head><body>
<h1>⚡ UIAI Engine Dashboard</h1>
<div class="card"><h3>Quick Links</h3>
<ul>
<li><a href="/api/health">/api/health</a> — Health check</li>
<li><a href="/api/admin/dashboard">/api/admin/dashboard</a> — Dashboard JSON</li>
<li><a href="/api/admin/resources">/api/admin/resources</a> — System resources</li>
<li><a href="/api/admin/usage/aggregate?period=daily">/api/admin/usage/aggregate</a> — Usage aggregation</li>
<li><a href="/api/admin/usage/breakdown">/api/admin/usage/breakdown</a> — Usage by type</li>
<li><a href="/api/admin/rate-limits">/api/admin/rate-limits</a> — Rate limit tiers</li>
<li><a href="/api/admin/config">/api/admin/config</a> — Config summary</li>
<li><a href="/api/admin/keys">/api/admin/keys</a> — API keys</li>
</ul></div>
<div class="card" id="status">Loading...</div>
<script>
fetch('/api/admin/dashboard').then(r=>r.json()).then(d=>{
  document.getElementById('status').innerHTML='<h3>Status</h3><pre>'+JSON.stringify(d,null,2)+'</pre>';
}).catch(e=>{ document.getElementById('status').textContent='Error: '+e; });
</script>
</body></html>`))
	}
}
