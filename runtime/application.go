package runtime

import (
	"context"
	"errors"
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
	adapter   httpadapter.Adapter
	lifecycle *lifecycleRegistry
}

// NewApplication resolves modules and registers controller routes at bootstrap.
func NewApplication(options ApplicationOptions, packages ...metadata.Package) (*Application, error) {
	adapter := options.Adapter
	if adapter == nil {
		adapter = fastadapter.New()
	}
	registry := &lifecycleRegistry{}
	if err := registerRoutesWithLifecycle(adapter, packages, registry); err != nil {
		return nil, err
	}
	if err := registry.initialize(context.Background()); err != nil {
		return nil, err
	}
	return &Application{adapter: adapter, lifecycle: registry}, nil
}

func ApplicationFromRegistry(options ApplicationOptions, names ...string) (*Application, error) {
	packages, err := registryImportClosure(names)
	if err != nil {
		return nil, err
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
	if a.lifecycle == nil {
		return a.adapter.Shutdown(ctx)
	}
	return a.lifecycle.shutdown(ctx, a.adapter.Shutdown)
}
