package runtime

import (
	"context"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
)

func TestCodeFirstHTTPServer(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func() httpadapter.Adapter
	}{
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
		{"nethttp", func() httpadapter.Adapter { return netadapter.New() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, err := New(
				WithAdapter(tc.new()),
				WithRoutes(Route{
					Method: http.MethodGet,
					Path:   "/health",
					Handler: func(ctx Context) error {
						ctx.Status(http.StatusAccepted)
						ctx.SetHeader("X-NestGo", "code-first")
						return ctx.Write([]byte("ready"))
					},
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() { done <- app.Serve(listener) }()
			client := &http.Client{Timeout: 3 * time.Second}
			res, err := client.Get("http://" + listener.Addr().String() + "/health")
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusAccepted || string(body) != "ready" || res.Header.Get("X-NestGo") != "code-first" {
				t.Fatalf("status=%d body=%q header=%q", res.StatusCode, body, res.Header.Get("X-NestGo"))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := app.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("server did not stop")
			}
		})
	}
}

func TestCodeFirstRouteValidation(t *testing.T) {
	healthy := Route{Method: "GET", Path: "/ready", Handler: func(ctx Context) error { return ctx.Write([]byte("ok")) }}
	for _, tc := range []struct {
		name   string
		routes []Route
		want   string
	}{
		{"nil handler", []Route{{Method: "GET", Path: "/ready"}}, "invalid direct route"},
		{"invalid path", []Route{{Method: "GET", Path: "ready", Handler: healthy.Handler}}, "invalid direct route"},
		{"missing method", []Route{{Path: "/ready", Handler: healthy.Handler}}, "invalid direct route"},
		{"duplicate", []Route{healthy, healthy}, "duplicate route GET /ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, err := New(WithRoutes(tc.routes...))
			if app != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("New app=%v err=%v, want %q", app, err, tc.want)
			}
		})
	}
}

type directCollisionController struct{}

func (*directCollisionController) Ready() string { return "ok" }

func TestCodeFirstRoutesShareControllerRouteNamespace(t *testing.T) {
	pkg := metadata.Package{Controllers: []metadata.Controller{{
		Type: reflect.TypeOf((*directCollisionController)(nil)),
		Routes: []metadata.Route{{Method: "GET", Path: "/ready", Handler: "Ready"}},
	}}}
	app, err := New(
		WithPackages(pkg),
		WithRoutes(Route{Method: "GET", Path: "/ready", Handler: func(Context) error { return nil }}),
	)
	if app != nil || err == nil || !strings.Contains(err.Error(), "duplicate route GET /ready") {
		t.Fatalf("expected shared route duplicate check: app=%v err=%v", app, err)
	}
}
