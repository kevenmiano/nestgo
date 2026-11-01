package fasthttp

import (
	"context"
	"errors"
	"github.com/kevenmiano/nestgo/httpadapter"
	fast "github.com/valyala/fasthttp"
	"net"
	"strings"
	"sync"
)

type Adapter struct {
	mu      sync.Mutex
	routes  []httpadapter.Route
	keys    map[string]bool
	server  *fast.Server
	started bool
}

var _ httpadapter.Adapter = (*Adapter)(nil)

func New() *Adapter { return &Adapter{keys: map[string]bool{}} }
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
	server := &fast.Server{Handler: func(c *fast.RequestCtx) {
		handler, params, foundPath := routes.lookup(string(c.Method()), string(c.Path()))
		if handler != nil {
			ctx := &requestContext{r: c, params: params, status: 200}
			if err := handler(ctx); err != nil && !ctx.wrote {
				c.Error(err.Error(), 500)
			}
			return
		}
		if foundPath {
			c.Error("Method Not Allowed", 405)
		} else {
			c.Error("Not Found", 404)
		}
	}}
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
func (c *requestContext) AddHeader(k, v string)  { c.r.Response.Header.Add(k, v) }
func (c *requestContext) Write(b []byte) error {
	if !c.wrote {
		c.r.SetStatusCode(c.status)
		c.wrote = true
	}
	_, e := c.r.Write(b)
	return e
}
