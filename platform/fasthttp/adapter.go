package fasthttp

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/kevenmiano/nestgo/httpadapter"
	fast "github.com/valyala/fasthttp"
)

type Adapter struct {
	mu      sync.Mutex
	routes  []httpadapter.Route
	keys    map[string]bool
	server  *fast.Server
	limits  httpadapter.ServerLimits
	started bool
}

var _ httpadapter.Adapter = (*Adapter)(nil)

// Option configures the fasthttp adapter before it starts.
type Option func(*Adapter)

// WithLimits overrides positive resource limits without disabling defaults.
func WithLimits(limits httpadapter.ServerLimits) Option {
	return func(a *Adapter) { a.limits = a.limits.Override(limits) }
}

func New(options ...Option) *Adapter {
	a := &Adapter{keys: map[string]bool{}, limits: httpadapter.DefaultServerLimits()}
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
		return errors.New("cannot register after serving")
	}
	if route.Handler == nil || route.Method == "" || !strings.HasPrefix(route.Path, "/") {
		return errors.New("invalid route")
	}
	key := route.Method + " " + route.Path
	if a.keys[key] {
		return errors.New("duplicate HTTP route: " + key)
	}
	for _, existing := range a.routes {
		if existing.Method != route.Method {
			continue
		}
		overlaps, aMore, bMore := routeOverlap(existing.Path, route.Path)
		if overlaps && aMore == bMore {
			return errors.New("ambiguous HTTP routes: " + existing.Path + " and " + route.Path)
		}
	}
	a.keys[key] = true
	a.routes = append(a.routes, route)
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
	routes := compileRoutes(a.routes)
	server := &fast.Server{
		ReadTimeout:        a.limits.ReadTimeout,
		WriteTimeout:       a.limits.WriteTimeout,
		IdleTimeout:        a.limits.IdleTimeout,
		ReadBufferSize:     a.limits.MaxHeaderBytes,
		MaxRequestBodySize: a.limits.MaxBodyBytes,
		ErrorHandler: func(c *fast.RequestCtx, err error) {
			if errors.Is(err, fast.ErrBodyTooLarge) {
				c.Error("request body too large", 413)
				return
			}
			c.Error("Bad Request", 400)
		},
		Handler: func(c *fast.RequestCtx) {
			handler, params, foundPath := routes.lookup(string(c.Method()), string(c.Path()))
			if handler != nil {
				ctx := &requestContext{r: c, params: params, status: 200}
				if err := handler(ctx); err != nil && !ctx.wrote {
					c.Error("Internal server error", 500)
				}
				return
			}
			if foundPath {
				c.Error("Method Not Allowed", 405)
			} else {
				c.Error("Not Found", 404)
			}
		},
	}
	a.server = server
	a.mu.Unlock()
	return server.Serve(listener)
}
func segmentCount(path string) int {
	return strings.Count(strings.Trim(path, "/"), "/") + 1
}

func match(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return nil, true
	}
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(xs) {
		return nil, false
	}
	m := map[string]string{}
	for i, p := range ps {
		if len(p) >= 3 && p[0] == '{' && p[len(p)-1] == '}' {
			m[p[1:len(p)-1]] = xs[i]
			continue
		}
		if p != xs[i] {
			return nil, false
		}
	}
	return m, true
}
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	s := a.server
	a.mu.Unlock()
	if s == nil {
		return nil
	}
	return s.ShutdownWithContext(ctx)
}
func (a *Adapter) Instance() any { a.mu.Lock(); defer a.mu.Unlock(); return a.server }

type requestContext struct {
	r      *fast.RequestCtx
	params map[string]string
	status int
	wrote  bool
}

func (c *requestContext) Method() string         { return string(c.r.Method()) }
func (c *requestContext) Path() string           { return string(c.r.Path()) }
func (c *requestContext) Param(n string) string  { return c.params[n] }
func (c *requestContext) Query(n string) string  { return string(c.r.QueryArgs().Peek(n)) }
func (c *requestContext) Header(n string) string { return string(c.r.Request.Header.Peek(n)) }
func (c *requestContext) Body() []byte           { return c.r.PostBody() }
func (c *requestContext) Status(n int)           { c.status = n }
func (c *requestContext) SetHeader(k, v string)  { c.r.Response.Header.Set(k, v) }
func (c *requestContext) Write(b []byte) error {
	if !c.wrote {
		c.r.SetStatusCode(c.status)
		c.wrote = true
	}
	_, e := c.r.Write(b)
	return e
}
