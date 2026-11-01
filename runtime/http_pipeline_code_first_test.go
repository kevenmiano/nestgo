package runtime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
)

func TestCodeFirstPipelineEngineParity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		newAdapter func() httpadapter.Adapter
	}{
		{name: "fasthttp", newAdapter: func() httpadapter.Adapter { return fastadapter.New() }},
		{name: "nethttp", newAdapter: func() httpadapter.Adapter { return netadapter.New() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trace := &pipelineTrace{}
			app, err := New(
				WithAdapter(tc.newAdapter()),
				WithRoutes(
					Route{
						Method: "GET", Path: "/hello/{id}",
						Handler: func(ctx Context) error {
							id, _, err := httpadapter.BindParam(ctx, "id", 0)
							if err != nil {
								return err
							}
							ctx.Status(202)
							return ctx.Write([]byte(id))
						},
					},
					Route{
						Method: "GET", Path: "/failed",
						Handler: func(Context) error { return errors.New("secret error") },
					},
				),
				WithMiddleware(func(ctx httpadapter.Context, next httpadapter.Next) error {
					ctx.SetHeader("X-Global-Middleware", "yes")
					return next()
				}),
				WithRouteMiddleware("GET", "/hello/{id}", func(ctx httpadapter.Context, next httpadapter.Next) error {
					ctx.SetHeader("X-Route-Middleware", "yes")
					return next()
				}),
				WithGuards(&globalTestGuard{trace: trace}),
				WithPipes(&routeTestPipe{Trace: trace}),
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
			base := "http://" + listener.Addr().String()

			resp, err := client.Get(base + "/hello/ada")
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 202 || string(data) != "ADA" ||
				resp.Header.Get("X-Global-Middleware") != "yes" ||
				resp.Header.Get("X-Route-Middleware") != "yes" {
				t.Fatalf("pipeline parity: status=%d body=%q headers=%v", resp.StatusCode, data, resp.Header)
			}
			resp, err = client.Get(base + "/failed")
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 500 || strings.Contains(string(data), "secret error") {
				t.Fatalf("unsafe server error: status=%d body=%q", resp.StatusCode, data)
			}

			stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := app.Shutdown(stopCtx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("HTTP engine did not shut down")
			}
		})
	}
}
