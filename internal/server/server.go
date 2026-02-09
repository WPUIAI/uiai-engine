// Package server wires up the HTTP server and all route modules.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/auth"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/credits"
	"github.com/philoveracity/uiai-engine/internal/intelligence"
	"github.com/philoveracity/uiai-engine/internal/media"
	"github.com/philoveracity/uiai-engine/internal/ratelimit"
	"github.com/philoveracity/uiai-engine/internal/routes"
	"github.com/philoveracity/uiai-engine/internal/storage"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

// startTime tracks when the engine was created (for /health uptime).
var startTime = time.Now()

// Engine is the main server instance.
type Engine struct {
	cfg       *config.Config
	router    chi.Router
	server    *http.Server
	auth      *auth.Authenticator
	ai        *ai.Provider
	credits   *credits.Service
	limiter   *ratelimit.Limiter
	usage     *storage.UsageStore
	vision    *vision.Pool
	sessions  *vision.SessionManager
	mediaJobs *media.JobStore
}

// New creates a new Engine with all routes wired.
func New(cfg *config.Config) *Engine {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5)) // gzip compression — reduces transfer size for JSON + screenshot data
	r.Use(maxBodySize(10 * 1024 * 1024)) // 10MB max request body
	r.Use(requestLogger)
	r.Use(corsMiddleware(cfg))

	authenticator := auth.New(cfg)
	aiProvider := ai.NewProvider(cfg)
	creditSvc := credits.New(cfg)
	limiter := ratelimit.New(cfg)
	usage := storage.NewUsageStore(cfg.Storage.DataDir, cfg.Storage.UsageFile)

	// Vision pool (Rod/Chromium) — optional, degrades gracefully
	var visionPool *vision.Pool
	if !cfg.Server.DisableVision {
		var err error
		visionPool, err = vision.NewPoolWithConfig(vision.PoolConfig{
			MaxPages:         cfg.Server.VisionPoolSize,
			AllowPrivateURLs: cfg.Vision.AllowPrivateURLs,
		})
		if err != nil {
			log.Printf("[vision] WARNING: Pool init failed: %v (screenshot/share will return 503)", err)
		}
	}

	// Auth middleware on all /api/* except health/status
	r.Use(authenticator.Middleware)

	// Media job store + periodic cleanup
	mediaJobs := media.NewJobStore(cfg.Storage.DataDir)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			removed := mediaJobs.CleanOld(7 * 24 * time.Hour)
			if removed > 0 {
				log.Printf("[media] Cleaned %d old jobs", removed)
			}
		}
	}()

	// Session manager for persistent browser sessions (LLM tool API)
	var sessionMgr *vision.SessionManager
	if visionPool != nil {
		sessionMgr = vision.NewSessionManager(visionPool)
	}

	e := &Engine{
		cfg:       cfg,
		router:    r,
		auth:      authenticator,
		ai:        aiProvider,
		credits:   creditSvc,
		limiter:   limiter,
		usage:     usage,
		vision:    visionPool,
		sessions:  sessionMgr,
		mediaJobs: mediaJobs,
	}

	e.mountRoutes()
	return e
}

// mountRoutes registers all route modules — mirrors the Bun server exactly.
func (e *Engine) mountRoutes() {
	r := e.router

	// Root
	r.Get("/", e.handleRoot)

	// Health (no auth)
	r.Route("/api/health", func(r chi.Router) {
		routes.MountHealth(r, e.cfg, e.ai)
	})

	// Also respond to /health for PHP API compat + health monitor
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":      "healthy",
			"service":     "uiai-engine",
			"browserless": true,
			"dev_mode":    false,
			"uptime":      int(time.Since(startTime).Seconds()),
		})
	})

	// API status
	r.Get("/api/status", e.handleStatus)

	// -- All /api/* routes below get auth middleware in Phase A2 --

	// AI endpoints — critique is real, rest are stubs until implemented
	r.Route("/api/critique", func(r chi.Router) {
		routes.MountCritiqueReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/ui-reverse", func(r chi.Router) {
		routes.MountUIReverseReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/section-detect", func(r chi.Router) {
		routes.MountSectionDetectReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/layout-compare", func(r chi.Router) {
		routes.MountLayoutCompareReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/style-enhance", func(r chi.Router) {
		routes.MountStyleEnhanceReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/copilot", func(r chi.Router) {
		routes.MountCopilotReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})

	// Supporting endpoints
	r.Route("/api/intake", func(r chi.Router) {
		routes.MountIntakeReal(r, e.cfg, e.ai)
	})
	r.Route("/api/workflow", func(r chi.Router) {
		routes.MountWorkflowReal(r, e.cfg)
	})
	r.Route("/api/usage", func(r chi.Router) {
		routes.MountUsageReal(r, e.cfg, e.usage)
	})
	r.Route("/api/extension", func(r chi.Router) {
		routes.MountExtensionReal(r, e.cfg, e.auth)
	})
	r.Route("/api/memory", func(r chi.Router) {
		routes.MountMemoryReal(r, e.cfg)
	})
	r.Route("/api/admin", func(r chi.Router) {
		routes.MountAdminReal(r, e.cfg, e.usage, e.limiter, e.auth)
	})
	r.Route("/api/reference", func(r chi.Router) {
		routes.MountReferenceReal(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/intelligence", func(r chi.Router) {
		intel := intelligence.NewLayer(e.cfg, e.ai)
		intel.Mount(r)
	})
	r.Route("/api/training", func(r chi.Router) {
		routes.MountTrainingReal(r, e.cfg)
	})

	// Media production (mockups, GIFs, videos)
	r.Route("/api/media", func(r chi.Router) {
		routes.MountMediaReal(r, e.cfg, e.credits, e.limiter, e.usage, e.mediaJobs)
	})

	// Screenshot & Share (Rod vision pool — Phase A8)
	r.Route("/api/screenshot", func(r chi.Router) {
		routes.MountScreenshotReal(r, e.cfg, e.vision, e.usage)
		routes.MountScreenshotCompare(r, e.cfg, e.vision, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/share", func(r chi.Router) {
		routes.MountShareReal(r, e.cfg, e.vision)
	})

	// Vision Interactive (replaces PHP daemon at port 3011)
	r.Route("/vision", func(r chi.Router) {
		routes.MountVisionInteractive(r, e.cfg, e.vision)
	})

	// Browser Sessions — persistent pages for LLM tool use
	// Open → interact → screenshot → close (page stays alive between calls)
	r.Route("/api/session", func(r chi.Router) {
		routes.MountSessionRoutes(r, e.cfg, e.sessions)
	})

	// Tool Discovery — LLM agents search/discover tools without loading all definitions
	// GET /api/tools/search?q=screenshot → find specific tools (minimal context)
	// GET /api/tools/openai → OpenAI function calling format
	// GET /api/tools/mcp → MCP tool definitions
	r.Route("/api/tools", func(r chi.Router) {
		routes.MountToolsDiscovery(r, e.cfg)
	})

	// Pipeline operations (cloud-capable build steps)
	r.Route("/api/design-system", func(r chi.Router) {
		routes.MountDesignSystemRoute(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/content-map", func(r chi.Router) {
		routes.MountContentMapRoute(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/block-recipes", func(r chi.Router) {
		routes.MountBlockRecipesRoute(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})
	r.Route("/api/comparison", func(r chi.Router) {
		routes.MountComparisonRoute(r, e.cfg, e.ai, e.credits, e.limiter, e.usage)
	})

	// Data migration API
	r.Route("/api/migration", func(r chi.Router) {
		routes.MountMigrationAPI(r, e.cfg)
	})

	// SSE real-time event stream
	r.Get("/api/events", routes.MountSSEStream(e.cfg))

	// Share viewer (public, no /api prefix)
	r.Get("/v/{token}", routes.HandleShareViewer(e.cfg))

	// Dashboard HTML
	r.Get("/dashboard", routes.HandleDashboard(e.cfg))
}

// handleRoot returns service info — same shape as Bun.
func (e *Engine) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"service": "WPUIAI AI Server",
		"version": "2.0.0",
		"status":  "running",
		"runtime": "go",
		"endpoints": map[string]string{
			"critique":     "/api/critique",
			"uiReverse":    "/api/ui-reverse",
			"sectionDetect": "/api/section-detect",
			"layoutCompare": "/api/layout-compare",
			"styleEnhance":  "/api/style-enhance",
			"copilot":       "/api/copilot/chat",
			"intake":        "/api/intake",
			"workflow":      "/api/workflow",
			"usage":         "/api/usage",
			"health":        "/api/health",
			"extension":     "/api/extension/token",
			"memory":        "/api/memory/:userId",
			"admin":         "/api/admin/*",
			"intelligence":  "/api/intelligence/*",
			"dashboard":     "/dashboard",
			"screenshot":    "/api/screenshot",
			"media":         "/api/media",
			"share":         "/api/share",
		},
	})
}

// handleStatus returns service status — same shape as Bun.
func (e *Engine) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"type": "status",
		"services": map[string]any{
			"uiai-engine": map[string]any{
				"id":      "uiai-engine",
				"name":    "UIAI Engine",
				"running": true,
				"port":    e.cfg.Server.Port,
				"health":  "healthy",
			},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// Run starts the server and blocks until shutdown signal.
func (e *Engine) Run() error {
	e.server = &http.Server{
		Addr:         e.cfg.Addr(),
		Handler:      e.router,
		ReadTimeout:  e.cfg.Server.ReadTimeout,
		WriteTimeout: e.cfg.Server.WriteTimeout,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("🚀 UIAI Engine listening on %s", e.cfg.Addr())
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-stop:
		log.Printf("Received %s, shutting down...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Close browser sessions first (returns pages to pool)
		if e.sessions != nil {
			log.Printf("Closing browser sessions...")
			e.sessions.CloseAll()
		}
		// Close vision pool (kills Chrome) before server shutdown
		if e.vision != nil {
			log.Printf("Closing vision pool...")
			e.vision.Close()
		}
		return e.server.Shutdown(ctx)
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.Status(), time.Since(start).Round(time.Millisecond))
	})
}

// maxBodySize limits request body size to prevent OOM from oversized POSTs.
func maxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	origins := make(map[string]bool)
	for _, o := range cfg.CORS.Origins {
		origins[o] = true
	}
	methods := strings.Join(cfg.CORS.Methods, ", ")
	headers := strings.Join(cfg.CORS.Headers, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origins[origin] || origins["*"] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
