package runtime

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"

    "github.com/kevenmiano/nestgo/metadata"
    "github.com/valyala/fasthttp"
    "github.com/valyala/fasthttp/fasthttpadaptor"
)

// Engine selects the HTTP server at application bootstrap.
// Controllers and generated metadata remain independent from the engine.
type Engine string

const (
    EngineFastHTTP Engine = "fasthttp"
    EngineNetHTTP  Engine = "net/http"
)

type ApplicationOptions struct {
    Engine Engine
}

type Application struct {
    engine Engine
    handler http.Handler
    server *fasthttp.Server
    stdServer *http.Server
}

func NewApplication(options ApplicationOptions, packages ...metadata.Package) (*Application, error) {
    engine := options.Engine
    if engine == "" {
        engine = EngineFastHTTP
    }
    if engine != EngineFastHTTP && engine != EngineNetHTTP {
        return nil, fmt.Errorf("unsupported HTTP engine %q", engine)
    }
    handler, err := New(packages...)
    if err != nil {
        return nil, err
    }
    return &Application{engine: engine, handler: handler}, nil
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

func (a *Application) Engine() Engine { return a.engine }

// Handler preserves interoperability with net/http middleware and existing callers.
// For the fast engine this compatibility boundary introduces per-request conversions.
func (a *Application) Handler() http.Handler { return a.handler }

// Serve serves on an existing listener. Create a new application for each listener.
func (a *Application) Serve(listener net.Listener) error {
    if listener == nil {
        return errors.New("nil listener")
    }
    switch a.engine {
    case EngineFastHTTP:
        server := &fasthttp.Server{Handler: fasthttpadaptor.NewFastHTTPHandler(a.handler)}
        a.server = server
        return server.Serve(listener)
    case EngineNetHTTP:
        server := &http.Server{Handler: a.handler}
        a.stdServer = server
        return server.Serve(listener)
    default:
        return fmt.Errorf("unsupported HTTP engine %q", a.engine)
    }
}

// Listen starts the configured HTTP server. It blocks until the server stops.
func (a *Application) Listen(address string) error {
    listener, err := net.Listen("tcp", address)
    if err != nil { return err }
    defer listener.Close()
    return a.Serve(listener)
}

func (a *Application) Shutdown(ctx context.Context) error {
    if a.stdServer != nil {
        return a.stdServer.Shutdown(ctx)
    }
    if a.server != nil {
        // fasthttp's ShutdownWithContext supports graceful shutdown.
        return a.server.ShutdownWithContext(ctx)
    }
    return nil
}
