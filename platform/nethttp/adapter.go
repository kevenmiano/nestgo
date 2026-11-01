package nethttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/kevenmiano/nestgo/httpadapter"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Adapter struct {
	mu      sync.Mutex
	mux     *http.ServeMux
	server  *http.Server
	started bool
}

var _ httpadapter.Adapter = (*Adapter)(nil)

func New() *Adapter { return &Adapter{mux: http.NewServeMux()} }
func (a *Adapter) Register(route httpadapter.Route) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return errors.New("cannot register routes after serving")
	}
	if route.Handler == nil || route.Method == "" || !strings.HasPrefix(route.Path, "/") {
		return errors.New("invalid route")
	}
	pattern := route.Method + " " + route.Path
	// Avoid ServeMux panic on duplicate or conflicting patterns.
	if err := a.register(pattern, route.Handler); err != nil {
		return err
	}
	return nil
}
func (a *Adapter) register(pattern string, handler httpadapter.Handler) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("invalid or conflicting route %s: %v", pattern, p)
		}
	}()
	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := &requestContext{w: w, r: r, status: 200}
		if e := handler(ctx); e != nil && !ctx.wrote {
			http.Error(w, e.Error(), 500)
		}
	})
	return nil
}
func (a *Adapter) Serve(listener net.Listener) error {
	if listener == nil {
		return errors.New("nil listener")
	}
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return errors.New("server already started")
	}
	a.started = true
	server := &http.Server{Handler: a.mux}
	a.server = server
	a.mu.Unlock()
	return server.Serve(listener)
}
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	server := a.server
	a.mu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}
func (a *Adapter) Instance() any { a.mu.Lock(); defer a.mu.Unlock(); return a.server }

type requestContext struct {
	w      http.ResponseWriter
	r      *http.Request
	status int
	wrote  bool
}

func (c *requestContext) Method() string         { return c.r.Method }
func (c *requestContext) Path() string           { return c.r.URL.Path }
func (c *requestContext) Param(n string) string  { return c.r.PathValue(n) }
func (c *requestContext) Query(n string) string  { return c.r.URL.Query().Get(n) }
func (c *requestContext) Header(n string) string { return c.r.Header.Get(n) }
func (c *requestContext) Body() []byte {
	b := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for len(b) <= 1<<20 {
		n, e := c.r.Body.Read(buf)
		b = append(b, buf[:n]...)
		if e != nil {
			break
		}
	}
	return b
}
func (c *requestContext) Status(code int)       { c.status = code }
func (c *requestContext) SetHeader(k, v string) { c.w.Header().Set(k, v) }
func (c *requestContext) AddHeader(k, v string) { c.w.Header().Add(k, v) }

// NewContext exposes the platform-independent response context for legacy handlers.
func NewContext(w http.ResponseWriter, r *http.Request) httpadapter.Context {
	return &requestContext{w: w, r: r, status: 200}
}
func (c *requestContext) Write(p []byte) (intErr error) {
	if !c.wrote {
		c.w.WriteHeader(c.status)
		c.wrote = true
	}
	_, err := c.w.Write(p)
	return err
}
