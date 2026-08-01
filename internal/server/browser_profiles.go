package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/WPUIAI/uiai-engine/internal/browserprofile"
	captchaPkg "github.com/WPUIAI/uiai-engine/internal/captcha"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/routes"
	"github.com/go-rod/rod"
)

// ProfiledEngine wraps the existing engine with the shared browser-profile
// manager and closes profile-scoped runtimes during shutdown.
type ProfiledEngine struct {
	*Engine
	browserProfiles *browserprofile.Manager
}

// NewWithBrowserProfiles builds the existing engine, loads browser profiles
// from the same YAML file, mounts /api/browser/*, and connects local_ip_pool
// profiles to the existing CAPTCHA IP-pool implementation.
func NewWithBrowserProfiles(cfg *config.Config, configPath string) *ProfiledEngine {
	engine := New(cfg)
	profiled := &ProfiledEngine{Engine: engine}

	registry, err := browserprofile.LoadFile(configPath)
	if err != nil {
		log.Printf("[browser-profile] WARNING: profile config unavailable: %v", err)
		return profiled
	}
	manager, err := browserprofile.NewManagerWithLauncher(registry, profileRuntimeLauncher(engine))
	if err != nil {
		log.Printf("[browser-profile] WARNING: manager unavailable: %v", err)
		return profiled
	}
	profiled.browserProfiles = manager
	engine.router.Route("/api/browser", func(r chiRouter) {
		routes.MountBrowserProfileRoutes(r, manager)
	})
	log.Printf("[browser-profile] enabled: default=%s profiles=%v", registry.Config().DefaultProfile, registry.Names())
	return profiled
}

// chiRouter is the subset of chi.Router accepted by route mounters. The type
// alias avoids leaking router construction out of the existing Engine.
type chiRouter = interface {
	Use(middlewares ...func(httpHandler) httpHandler)
}

// Run starts the wrapped engine and closes all independently launched profile
// runtimes when the engine terminates.
func (e *ProfiledEngine) Run() error {
	if e == nil || e.Engine == nil {
		return fmt.Errorf("profiled engine is nil")
	}
	if e.browserProfiles != nil {
		defer e.browserProfiles.CloseAll()
	}
	return e.Engine.Run()
}

func profileRuntimeLauncher(engine *Engine) browserprofile.LauncherFunc {
	return func(ctx context.Context, profile browserprofile.ResolvedProfile) (browserprofile.RuntimeHandle, error) {
		switch profile.Network.Route {
		case "", "direct", "operator_route":
			return browserprofile.Launch(ctx, profile)
		case "named_proxy", "tailscale_exit":
			if profile.Network.ProxyServer == "" {
				return nil, fmt.Errorf("browser profile %q route %q requires network.proxy_server", profile.ID, profile.Network.Route)
			}
			return browserprofile.Launch(ctx, profile)
		case "local_ip_pool":
			if engine == nil || engine.captcha == nil || engine.captcha.Pool() == nil {
				return nil, fmt.Errorf("browser profile %q requires the configured CAPTCHA local IP pool", profile.ID)
			}
			return launchFromCaptchaPool(ctx, engine.captcha.Pool(), profile)
		default:
			return nil, fmt.Errorf("browser profile %q uses unsupported network route %q", profile.ID, profile.Network.Route)
		}
	}
}

func launchFromCaptchaPool(ctx context.Context, pool *captchaPkg.IPPool, profile browserprofile.ResolvedProfile) (browserprofile.RuntimeHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	endpoint, release, err := pool.Pick()
	if err != nil {
		return nil, fmt.Errorf("select local IP route: %w", err)
	}
	proxied, err := pool.LaunchWithEndpoint(endpoint)
	if err != nil {
		release()
		pool.ReportResult(endpoint, false, false, err.Error())
		return nil, fmt.Errorf("launch local IP route: %w", err)
	}
	return &captchaPoolProfileRuntime{
		profile:  profile,
		pool:     pool,
		endpoint: endpoint,
		proxied:  proxied,
		release:  release,
	}, nil
}

type captchaPoolProfileRuntime struct {
	profile  browserprofile.ResolvedProfile
	pool     *captchaPkg.IPPool
	endpoint string
	proxied  *captchaPkg.ProxiedBrowser
	release  func()
	once     sync.Once
}

func (r *captchaPoolProfileRuntime) OpenPage(ctx context.Context, targetURL string) (*rod.Page, error) {
	if r == nil || r.proxied == nil || r.proxied.RodBrowser() == nil {
		return nil, fmt.Errorf("local IP browser runtime is unavailable")
	}
	base := &browserprofile.Runtime{
		Profile: r.profile,
		Browser: r.proxied.RodBrowser(),
		PID:     r.proxied.PID(),
	}
	page, err := base.OpenPage(ctx, targetURL)
	if err != nil {
		r.pool.ReportResult(r.endpoint, false, detectRouteFailure(err), err.Error())
		return nil, err
	}
	return page, nil
}

func (r *captchaPoolProfileRuntime) RuntimePID() int {
	if r == nil || r.proxied == nil {
		return 0
	}
	return r.proxied.PID()
}

func (r *captchaPoolProfileRuntime) Close() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		if r.proxied != nil {
			r.proxied.Close()
		}
		if r.release != nil {
			r.release()
		}
	})
}

func detectRouteFailure(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "automated queries") ||
		strings.Contains(message, "unusual traffic") ||
		strings.Contains(message, "access denied") ||
		strings.Contains(message, "forbidden") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "proxy")
}
