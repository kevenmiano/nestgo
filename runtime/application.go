package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
)

// Option configures the application before bootstrap.
type Option func(*applicationConfig)

type applicationConfig struct {
	adapter      httpadapter.Adapter
	packages     []metadata.Package
	routes       []Route
	registry     []string
	logger       *logging.Service
	debug        bool
	onBootstrap  func(BootstrapMetrics)
	bootstrapCtx context.Context
	contextSet   bool
	pipeline     requestPipelineConfig
}

// WithPackages supplies already-generated metadata to the application.
func WithPackages(packages ...metadata.Package) Option {
	return func(c *applicationConfig) { c.packages = append(c.packages, packages...) }
}

// Context is the HTTP-engine-independent request context.
type Context = httpadapter.Context

// Handler handles an HTTP request without generated controller metadata.
type Handler = httpadapter.Handler

// Route declares an HTTP endpoint without the NestGo compiler.
type Route = httpadapter.Route

// WithRoutes registers ordinary Go HTTP handlers on the selected adapter.
// These routes participate in the same bootstrap error handling and shutdown.
func WithRoutes(routes ...Route) Option {
	return func(c *applicationConfig) { c.routes = append(c.routes, routes...) }
}

// WithRegistry loads named registered packages and their transitive module imports.
// Import the generated Go packages so their init functions register metadata.
func WithRegistry(names ...string) Option {
	return func(c *applicationConfig) { c.registry = append(c.registry, names...) }
}

// WithAdapter selects an HTTP engine; the default is fasthttp.
func WithAdapter(adapter httpadapter.Adapter) Option {
	return func(c *applicationConfig) { c.adapter = adapter }
}

// WithContext supplies the bootstrap context for lifecycle hooks.
// Use the context passed by the calling service; it is not an automatic
// request context, and cancellation does not shut down an already started app.
func WithContext(ctx context.Context) Option {
	return func(c *applicationConfig) {
		c.bootstrapCtx = ctx
		c.contextSet = true
	}
}

// WithLogger enables structured framework bootstrap diagnostics.
func WithLogger(logger *logging.Service) Option {
	return func(c *applicationConfig) { c.logger = logger }
}

// WithDebug enables per-provider bootstrap diagnostics when a logger is configured.
func WithDebug(enabled bool) Option {
	return func(c *applicationConfig) { c.debug = enabled }
}

// WithBootstrapMetrics registers an observer invoked after successful bootstrap.
func WithBootstrapMetrics(observer func(BootstrapMetrics)) Option {
	return func(c *applicationConfig) { c.onBootstrap = observer }
}

// WithMiddleware installs global HTTP middleware in registration order.
func WithMiddleware(middlewares ...httpadapter.Middleware) Option {
	return func(c *applicationConfig) { c.pipeline.middleware = append(c.pipeline.middleware, middlewares...) }
}

// WithRouteMiddleware adds middleware to a specific registered method and path.
func WithRouteMiddleware(method, path string, middlewares ...httpadapter.Middleware) Option {
	return func(c *applicationConfig) {
		if c.pipeline.routeMiddleware == nil {
			c.pipeline.routeMiddleware = make(map[string][]httpadapter.Middleware)
		}
		key := method + " " + path
		c.pipeline.routeMiddleware[key] = append(c.pipeline.routeMiddleware[key], middlewares...)
	}
}

func WithGuards(guards ...httpadapter.Guard) Option {
	return func(c *applicationConfig) { c.pipeline.guards = append(c.pipeline.guards, guards...) }
}

func WithPipes(pipes ...httpadapter.Pipe) Option {
	return func(c *applicationConfig) { c.pipeline.pipes = append(c.pipeline.pipes, pipes...) }
}

func WithFilters(filters ...httpadapter.ExceptionFilter) Option {
	return func(c *applicationConfig) { c.pipeline.filters = append(c.pipeline.filters, filters...) }
}

// Application owns one HTTP adapter and the lifecycle of its providers.
type Application struct {
	adapter   httpadapter.Adapter
	lifecycle *lifecycleRegistry
}

// New is the single application factory. All HTTP platforms share DI and hooks.
//
// Example: app, err := runtime.New(runtime.WithRegistry("shop"))
func New(options ...Option) (*Application, error) {
	var cfg applicationConfig
	for i, option := range options {
		if option == nil {
			return nil, fmt.Errorf("nil application option at index %d", i)
		}
		option(&cfg)
	}
	if cfg.contextSet && cfg.bootstrapCtx == nil {
		return nil, errors.New("nil bootstrap context")
	}
	if len(cfg.registry) > 0 && len(cfg.packages) > 0 {
		return nil, errors.New("WithRegistry and WithPackages cannot be combined")
	}
	packages := cfg.packages
	if len(cfg.registry) > 0 {
		var err error
		packages, err = registryImportClosure(cfg.registry)
		if err != nil {
			return nil, err
		}
	}

	started := time.Now()
	adapter := cfg.adapter
	if adapter == nil {
		adapter = fastadapter.New()
	}
	registry := &lifecycleRegistry{}
	ctx := cfg.bootstrapCtx
	if ctx == nil {
		ctx = context.Background()
	}
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
	registrationStarted := time.Now()
	if err := validatePipeline(cfg.pipeline); err != nil { return fail(err) }
	if err := registerRoutesWithPipeline(adapter, packages, registry, cfg.pipeline, cfg.routes...); err != nil {
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
	return &Application{adapter: adapter, lifecycle: registry}, nil
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

// Close implements io.Closer. Use Shutdown(ctx) instead when the caller must
// enforce a shutdown deadline. Both operations share idempotent lifecycle
// cleanup so callers may safely invoke them repeatedly.
func (a *Application) Close() error {
	return a.Shutdown(context.Background())
}
