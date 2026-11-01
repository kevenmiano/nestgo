package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/metadata"
)

// AppOption configures an application before its dependencies are initialized.
type AppOption func(*appOptions) error

type appOptions struct {
	config []config.Option
	database []database.Option
	packages []metadata.Package
	instances []any
	logger *logging.Service
	debug bool
}

// App owns the initialized HTTP handler and its database connections.
type App struct {
	*Bootstrapped
}

// NewApp is the idiomatic application entrypoint. Construction initializes
// providers and pools; callers must Close the returned app during shutdown.
func NewApp(ctx context.Context, options ...AppOption) (*App, error) {
	opts := appOptions{}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("runtime: nil app option")
		}
		if err := option(&opts); err != nil {
			return nil, err
		}
	}
	boot, err := Bootstrap(ctx, BootstrapOptions{
		ConfigOptions: opts.config,
		DatabaseOptions: opts.database,
		Runtime: Options{Logger: opts.logger, Debug: opts.debug, Instances: opts.instances},
	}, opts.packages...)
	if err != nil {
		return nil, err
	}
	return &App{Bootstrapped: boot}, nil
}

// HTTPHandler exposes the ready-to-serve net/http handler.
func (a *App) HTTPHandler() http.Handler {
	if a == nil || a.Bootstrapped == nil {
		return nil
	}
	return a.Handler
}

// WithConfig supplies the configuration loader options.
func WithConfig(options ...config.Option) AppOption {
	return func(o *appOptions) error {
		o.config = append(o.config, options...)
		return nil
	}
}

// WithDatabase supplies named SQL connections and their factories.
func WithDatabase(options ...database.Option) AppOption {
	return func(o *appOptions) error {
		o.database = append(o.database, options...)
		return nil
	}
}

// WithPackages installs compiled NestGo metadata.
func WithPackages(packages ...metadata.Package) AppOption {
	return func(o *appOptions) error {
		o.packages = append(o.packages, packages...)
		return nil
	}
}

// WithProvider supplies an initialized external dependency to DI.
func WithProvider(instance any) AppOption {
	return func(o *appOptions) error {
		if instance == nil {
			return fmt.Errorf("runtime: nil provider")
		}
		o.instances = append(o.instances, instance)
		return nil
	}
}

// WithLogger configures bootstrap diagnostics.
func WithLogger(logger *logging.Service) AppOption {
	return func(o *appOptions) error {
		if logger == nil {
			return errors.New("runtime: nil logger")
		}
		o.logger = logger
		return nil
	}
}

// WithDebug enables verbose bootstrap diagnostics.
func WithDebug() AppOption {
	return func(o *appOptions) error {
		o.debug = true
		return nil
	}
}
