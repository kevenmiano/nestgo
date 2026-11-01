package runtime

import (
	"context"
	"errors"
	"fmt"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"net"
)

type ApplicationOptions struct {
	// Adapter selects the HTTP platform. Nil defaults to native fasthttp.
	Adapter httpadapter.Adapter
}

type Application struct {
	adapter httpadapter.Adapter
}

// NewApplication resolves modules and registers controller routes at bootstrap.
func NewApplication(options ApplicationOptions, packages ...metadata.Package) (*Application, error) {
	adapter := options.Adapter
	if adapter == nil {
		adapter = fastadapter.New()
	}
	if err := registerRoutes(adapter, packages); err != nil {
		return nil, err
	}
	return &Application{adapter: adapter}, nil
}

func ApplicationFromRegistry(options ApplicationOptions, names ...string) (*Application, error) {
	packages := make([]metadata.Package, 0, len(names))
	for _, name := range names {
		pkg, ok := metadata.Lookup(name)
		if !ok {
			return nil, fmt.Errorf("metadata package not found: %s", name)
		}
		packages = append(packages, pkg)
	}
	return NewApplication(options, packages...)
}

func (a *Application) HTTPAdapter() httpadapter.Adapter  { return a.adapter }
func (a *Application) Serve(listener net.Listener) error { return a.adapter.Serve(listener) }
func (a *Application) Listen(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	return a.Serve(listener)
}
func (a *Application) Shutdown(ctx context.Context) error {
	if a == nil || a.adapter == nil {
		return errors.New("application is not initialized")
	}
	return a.adapter.Shutdown(ctx)
}
