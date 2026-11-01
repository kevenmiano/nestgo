package smoke

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/compiler"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/runtime"
)

func TestExternalConsumerCanImportAndServe(t *testing.T) {
	_ = compiler.Parse
	_ = metadata.Lookup
	for _, tc := range []struct {
		name       string
		newAdapter func() httpadapter.Adapter
	}{
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
		{"nethttp", func() httpadapter.Adapter { return netadapter.New() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, err := runtime.New(runtime.WithAdapter(tc.newAdapter()), runtime.WithRoutes(runtime.Route{
				Method: "GET", Path: "/smoke",
				Handler: func(ctx runtime.Context) error {
					ctx.Status(200)
					return ctx.Write([]byte("external-consumer-ok"))
				},
			}))
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
			resp, err := client.Get("http://" + listener.Addr().String() + "/smoke")
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 || string(body) != "external-consumer-ok" {
				t.Fatalf("status=%d body=%q", resp.StatusCode, body)
			}
			client.CloseIdleConnections()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := app.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("external adapter did not stop")
			}
		})
	}
}
