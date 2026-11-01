package runtime

import (
	"context"
	"errors"
	"net/http"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/metadata"
)

// BootstrapOptions configures the owned resources for a single runtime.
type BootstrapOptions struct {
	ConfigOptions   []config.Option
	DatabaseOptions []database.Option
	Runtime         Options
}

// Bootstrapped owns initialized services and must be closed after use.
type Bootstrapped struct {
	Handler     http.Handler
	Config      *config.Service
	Connections *database.Connections
}

// Bootstrap loads configuration, opens named pools, registers both as injectable
// providers, then constructs the handler. A failure closes opened resources.
func Bootstrap(ctx context.Context, options BootstrapOptions, packages ...metadata.Package) (*Bootstrapped, error) {
	if ctx == nil {
		return nil, errors.New("runtime: nil bootstrap context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	settings, err := config.New(options.ConfigOptions...)
	if err != nil {
		return nil, err
	}
	module, err := database.NewModule(options.DatabaseOptions...)
	if err != nil {
		return nil, err
	}
	connections, err := module.Open(ctx)
	if err != nil {
		return nil, err
	}
	opts := options.Runtime
	opts.Instances = append(append([]any{}, opts.Instances...), settings, connections)
	handler, err := NewWithOptions(opts, packages...)
	if err != nil {
		_ = connections.Close()
		return nil, err
	}
	return &Bootstrapped{Handler: handler, Config: settings, Connections: connections}, nil
}

// Close releases owned connections; it does not shut down the HTTP listener.
func (b *Bootstrapped) Close() error {
	if b == nil || b.Connections == nil {
		return nil
	}
	return b.Connections.Close()
}
