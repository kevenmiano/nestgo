package runtime

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type staticRouteController struct{}

func (*staticRouteController) Ready() string        { return "reflect-fallback" }
func (*staticRouteController) Echo(s string) string { return s }

type capturedAdapter struct {
	routes map[string]httpadapter.Handler
}

func (a *capturedAdapter) Register(r httpadapter.Route) error {
	if a.routes == nil {
		a.routes = make(map[string]httpadapter.Handler)
	}
	a.routes[r.Method+" "+r.Path] = r.Handler
	return nil
}
func (*capturedAdapter) Serve(net.Listener) error       { return nil }
func (*capturedAdapter) Shutdown(context.Context) error { return nil }

type capturedContext struct {
	status int
	body   string
	params map[string]string
}

func (*capturedContext) Method() string           { return "GET" }
func (*capturedContext) Path() string             { return "/" }
func (c *capturedContext) Param(k string) string  { return c.params[k] }
func (*capturedContext) Query(string) string      { return "" }
func (*capturedContext) Header(string) string     { return "" }
func (*capturedContext) Body() []byte             { return nil }
func (c *capturedContext) Status(s int)           { c.status = s }
func (*capturedContext) SetHeader(string, string) {}
func (c *capturedContext) Write(p []byte) error   { c.body += string(p); return nil }

func TestAdapterStaticInvocationAndDynamicFallback(t *testing.T) {
	staticCalls := 0
	a := &capturedAdapter{}
	packages := []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*staticRouteController)(nil)),
		Prefix: "/test",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/ready", Handler: "Ready", StaticInvoke: func(v any) (any, error) {
				staticCalls++
				return "static", nil
			}},
			{Method: "GET", Path: "/echo/{id}", Handler: "Echo", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
			{Method: "GET", Path: "/broken", Handler: "Ready", StaticInvoke: func(any) (any, error) {
				return nil, errors.New("static error")
			}},
		},
	}}}}
	if err := registerRoutes(a, packages); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, want string
		status     int
		params     map[string]string
	}{
		{"/test/ready", "\"static\"\n", 200, nil},
		{"/test/echo/{id}", "\"Ada\"\n", 200, map[string]string{"id": "Ada"}},
		{"/test/broken", "Internal server error\n", 500, nil},
	} {
		ctx := &capturedContext{params: tc.params}
		if err := a.routes["GET "+tc.path](ctx); err != nil {
			t.Fatal(err)
		}
		if ctx.status != tc.status || ctx.body != tc.want {
			t.Errorf("%s: got (%d,%q), want (%d,%q)", tc.path, ctx.status, ctx.body, tc.status, tc.want)
		}
	}
	if staticCalls != 1 {
		t.Fatalf("static invoker called %d times; want 1", staticCalls)
	}
}

func BenchmarkAdapterStaticInvocation(b *testing.B) {
	a := &capturedAdapter{}
	pkgs := []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*staticRouteController)(nil)),
		Prefix: "/bench",
		Routes: []metadata.Route{{Method: "GET", Path: "/ready", Handler: "Ready", StaticInvoke: func(any) (any, error) { return "ok", nil }}},
	}}}}
	if err := registerRoutes(a, pkgs); err != nil {
		b.Fatal(err)
	}
	h := a.routes["GET /bench/ready"]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &capturedContext{}
		if err := h(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
