package runtime

import (
	"context"
	"errors"
	"net/http"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/metadata"
)

// bootstrapOptions configures the owned resources for a single runtime.
type bootstrapOptions struct {
	ConfigOptions   []config.Option
	DatabaseOptions []database.Option
	Runtime         Options
}

// bootstrapped owns initialized services and must be closed after use.
type bootstrapped struct {
	Handler     http.Handler
	Config      *config.Service
	Connections *database.Connections
}

// bootstrap loads configuration, opens named pools, registers both as injectable
// providers, then constructs the handler. A failure closes opened resources.
func bootstrap(ctx context.Context, options bootstrapOptions, packages ...metadata.Package) (*bootstrapped, error) {
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
	handler, err := newHandler(opts, packages...)
	if err != nil {
		_ = connections.Close()
		return nil, err
	}
	return &bootstrapped{Handler: handler, Config: settings, Connections: connections}, nil
}

// Close releases owned connections; it does not shut down the HTTP listener.
func (b *bootstrapped) Close() error {
	if b == nil || b.Connections == nil {
		return nil
	}
	return b.Connections.Close()
}
