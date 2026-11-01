package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"time"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
)

// Option configures the application before bootstrap.
type Option func(*applicationConfig)

type applicationConfig struct {
	adapter          httpadapter.Adapter
	packages         []metadata.Package
	routes           []Route
	logger           *logging.Service
	loggerConfigured bool
	debug            bool
	onBootstrap      func(BootstrapMetrics)
	configOptions    []config.Option
	databaseOptions  []database.Option
	instances        []any
	context          context.Context
}

// WithPackages supplies already-generated metadata to the application.
func WithPackages(packages ...metadata.Package) Option {
	return func(c *applicationConfig) { c.packages = append(c.packages, packages...) }
}

// Context is the HTTP-engine-independent request context.
type Context = httpadapter.Context

// Handler processes an HTTP request without compiled controller metadata.
type Handler = httpadapter.Handler

// Route declares an HTTP endpoint directly in Go.
type Route = httpadapter.Route

// WithRoutes registers direct handlers with the same namespace and lifecycle as compiled routes.
func WithRoutes(routes ...Route) Option {
	return func(c *applicationConfig) { c.routes = append(c.routes, routes...) }
}

// WithAdapter selects an HTTP engine; the default is fasthttp.
func WithAdapter(adapter httpadapter.Adapter) Option {
	return func(c *applicationConfig) { c.adapter = adapter }
}

// WithLogger enables structured framework bootstrap diagnostics.
func WithLogger(logger *logging.Service) Option {
	return func(c *applicationConfig) { c.logger = logger; c.loggerConfigured = true }
}

// WithDebug enables per-provider bootstrap diagnostics when a logger is configured.
func WithDebug(enabled bool) Option {
	return func(c *applicationConfig) { c.debug = enabled }
}

// WithConfig supplies typed configuration loader options.
func WithConfig(options ...config.Option) Option {
	return func(c *applicationConfig) { c.configOptions = append(c.configOptions, options...) }
}

// WithDatabase configures independent named SQL connection pools.
func WithDatabase(options ...database.Option) Option {
	return func(c *applicationConfig) { c.databaseOptions = append(c.databaseOptions, options...) }
}

// WithProvider injects an already initialized external service.
func WithProvider(instance any) Option {
	return func(c *applicationConfig) { c.instances = append(c.instances, instance) }
}

// WithContext supplies a cancellable bootstrap context; it does not control HTTP shutdown.
func WithContext(ctx context.Context) Option {
	return func(c *applicationConfig) { c.context = ctx }
}

// WithBootstrapMetrics registers an observer invoked after successful bootstrap.
func WithBootstrapMetrics(observer func(BootstrapMetrics)) Option {
	return func(c *applicationConfig) { c.onBootstrap = observer }
}

// Application owns one HTTP adapter and the lifecycle of its providers.
type Application struct {
	adapter     httpadapter.Adapter
	lifecycle   *lifecycleRegistry
	Config      *config.Service
	Connections *database.Connections
}

// New is the single application factory. All HTTP platforms share DI and hooks.
//
// Example: app, err := runtime.New(runtime.WithDatabase(...))
func New(options ...Option) (*Application, error) {
	var cfg applicationConfig
	for i, option := range options {
		if option == nil {
			return nil, fmt.Errorf("nil application option at index %d", i)
		}
		option(&cfg)
	}
	if cfg.loggerConfigured && cfg.logger == nil {
		return nil, errors.New("runtime: nil logger")
	}
	packages := cfg.packages
	if len(cfg.packages) == 0 {
		// Compiled modules self-register at import time. Their unique root and
		// transitive imports are discovered once during bootstrap.
		var err error
		packages, err = discoverCompiledPackages()
		if err != nil {
			return nil, err
		}
	}

	ctx := cfg.context
	if ctx == nil {
		ctx = context.Background()
	} else if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, instance := range cfg.instances {
		if instance == nil {
			return nil, errors.New("runtime: nil external provider")
		}
	}
	started := time.Now()
	adapter := cfg.adapter
	if adapter == nil {
		adapter = fastadapter.New()
	}
	registry := &lifecycleRegistry{}
	logger := cfg.logger
	if logger != nil {
		logger.Info(ctx, "NestFactory", "Starting NestGo application...")
		registry.onProvider = func(typ reflect.Type) {
			if cfg.debug {
				logger.Debug(ctx, "InstanceLoader", "Provider instantiated", slog.String("provider", typ.String()))
			}
		}
		registry.onController = func(typ reflect.Type, prefix string) {
			logger.Info(ctx, "RoutesResolver", "Controller registered", slog.String("controller", typ.String()), slog.String("prefix", prefix))
		}
		registry.onRoute = func(method, path string) {
			logger.Info(ctx, "RouterExplorer", "Mapped route", slog.String("method", method), slog.String("path", path))
		}
	}
	fail := func(err error) (*Application, error) {
		if logger != nil {
			logger.Error(ctx, "NestFactory", "Bootstrap failed", slog.String("error", err.Error()))
		}
		return nil, errors.Join(err, registry.shutdown(ctx, adapter.Shutdown))
	}
	settings, err := config.New(cfg.configOptions...)
	if err != nil {
		return fail(err)
	}
	module, err := database.NewModule(cfg.databaseOptions...)
	if err != nil {
		return fail(err)
	}
	connections, err := module.Open(ctx)
	if err != nil {
		return fail(err)
	}
	// Register first so the pool closes after all controller/provider hooks.
	registry.instances = append(registry.instances, &ownedSQLConnections{connections})
	instances := append([]any{settings, connections}, cfg.instances...)
	registrationStarted := time.Now()
	if err := registerRoutesWithLifecycleAndInstances(adapter, packages, registry, instances, cfg.routes...); err != nil {
		return fail(err)
	}
	registrationDuration := time.Since(registrationStarted)
	if logger != nil {
		logger.Startup(ctx, "InstanceLoader", "Provider dependencies resolved", registrationStarted)
		logger.Startup(ctx, "RouterExplorer", "Routes mapped", registrationStarted)
	}
	lifecycleStarted := time.Now()
	if err := registry.initialize(ctx); err != nil {
		return fail(err)
	}
	if logger != nil {
		logger.Startup(ctx, "NestApplication", "Application initialized", started)
	}
	if cfg.onBootstrap != nil {
		cfg.onBootstrap(BootstrapMetrics{RouteRegistration: registrationDuration, LifecycleInitialization: time.Since(lifecycleStarted), Total: time.Since(started)})
	}
	return &Application{adapter: adapter, lifecycle: registry, Config: settings, Connections: connections}, nil
}

// HTTPAdapter returns the configured transport adapter.
func (a *Application) HTTPAdapter() httpadapter.Adapter { return a.adapter }

// Serve handles HTTP requests until the listener closes or Shutdown is called.
func (a *Application) Serve(listener net.Listener) error { return a.adapter.Serve(listener) }

// Listen binds and serves on the provided address.
func (a *Application) Listen(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	return a.Serve(listener)
}

// Shutdown stops the HTTP adapter and invokes cleanup hooks once.
func (a *Application) Shutdown(ctx context.Context) error {
	if a == nil || a.adapter == nil {
		return errors.New("application is not initialized")
	}
	if a.lifecycle == nil {
		return a.adapter.Shutdown(ctx)
	}
	return a.lifecycle.shutdown(ctx, a.adapter.Shutdown)
}

// ownedSQLConnections joins SQL pool ownership to the standard application lifecycle.
type ownedSQLConnections struct{ pools *database.Connections }

func (o *ownedSQLConnections) OnModuleDestroy(context.Context) error { return o.pools.Close() }

// Close invokes the same idempotent shutdown path used by Shutdown(ctx).
func (a *Application) Close() error { return a.Shutdown(context.Background()) }
