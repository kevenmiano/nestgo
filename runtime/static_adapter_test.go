package runtime

import (
	"context"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

type staticHookController struct{}

func (*staticHookController) Value() string { return "reflected" }

func TestAdapterUsesGeneratedStaticInvoker(t *testing.T) {
	var calls atomic.Int32
	p := metadata.Package{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*staticHookController)(nil)),
		Prefix: "/api",
		Routes: []metadata.Route{{
			Method: "GET", Path: "/static", Handler: "Value",
			StaticInvoke: func(any) (any, error) { calls.Add(1); return "generated", nil },
		}},
	}}}
	app, err := New(WithAdapter(nethttp.New()), WithPackages(p))
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- app.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/api/static")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || string(data) != "\"generated\"\n" || calls.Load() != 1 {
		t.Fatalf("status=%d data=%s generated invocations=%d", resp.StatusCode, data, calls.Load())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timeout")
	}
}
