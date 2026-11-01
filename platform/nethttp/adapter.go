package nethttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/kevenmiano/nestgo/httpadapter"
)

type Adapter struct {
	mu      sync.Mutex
	mux     *http.ServeMux
	server  *http.Server
	limits  httpadapter.ServerLimits
	started bool
}

var _ httpadapter.Adapter = (*Adapter)(nil)

// Option configures the net/http adapter before it starts.
type Option func(*Adapter)

// WithLimits overrides positive server limits while retaining secure defaults.
func WithLimits(limits httpadapter.ServerLimits) Option {
	return func(a *Adapter) { a.limits = a.limits.Override(limits) }
}

func New(options ...Option) *Adapter {
	a := &Adapter{mux: http.NewServeMux(), limits: httpadapter.DefaultServerLimits()}
	for _, option := range options {
		if option != nil {
			option(a)
		}
	}
	return a
}
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
		if r.ContentLength > int64(a.limits.MaxBodyBytes) {
			w.Header().Set("Connection", "close")
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(r.Body, int64(a.limits.MaxBodyBytes)+1))
		if readErr != nil {
			w.Header().Set("Connection", "close")
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body) > a.limits.MaxBodyBytes {
			w.Header().Set("Connection", "close")
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		ctx := &requestContext{w: w, r: r, body: body, status: 200}
		if e := handler(ctx); e != nil && !ctx.wrote {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
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
	server := &http.Server{
		Handler:           a.mux,
		ReadHeaderTimeout: a.limits.ReadHeaderTimeout,
		ReadTimeout:       a.limits.ReadTimeout,
		WriteTimeout:      a.limits.WriteTimeout,
		IdleTimeout:       a.limits.IdleTimeout,
		MaxHeaderBytes:    a.limits.MaxHeaderBytes,
	}
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
	body   []byte
	status int
	wrote  bool
}

func (c *requestContext) Method() string         { return c.r.Method }
func (c *requestContext) Path() string           { return c.r.URL.Path }
func (c *requestContext) Param(n string) string  { return c.r.PathValue(n) }
func (c *requestContext) Query(n string) string  { return c.r.URL.Query().Get(n) }
func (c *requestContext) Header(n string) string { return c.r.Header.Get(n) }
func (c *requestContext) Body() []byte           { return c.body }
func (c *requestContext) Status(code int)        { c.status = code }
func (c *requestContext) SetHeader(k, v string)  { c.w.Header().Set(k, v) }
func (c *requestContext) AddHeader(k, v string)  { c.w.Header().Add(k, v) }
func (c *requestContext) Write(p []byte) (intErr error) {
	if !c.wrote {
		c.w.WriteHeader(c.status)
		c.wrote = true
	}
	_, err := c.w.Write(p)
	return err
}
