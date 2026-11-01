package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
)

type classTestMiddleware struct{ steps *[]string }

func (m *classTestMiddleware) Use(_ httpadapter.Context, next httpadapter.Next) error {
	*m.steps = append(*m.steps, "class before")
	err := next()
	*m.steps = append(*m.steps, "class after")
	return err
}

var _ httpadapter.MiddlewareHandler = (*classTestMiddleware)(nil)
var _ httpadapter.MiddlewareHandler = httpadapter.Middleware(nil)

func TestClassMiddlewareGlobalAndRouteOrder(t *testing.T) {
	steps := []string{}
	adapter := &capturedAdapter{}
	app, err := New(
		WithAdapter(adapter),
		WithMiddlewareHandlers(&classTestMiddleware{steps: &steps}),
		WithRouteMiddlewareHandlers("GET", "/status", &classTestMiddleware{steps: &steps}),
		WithRoutes(Route{Method: "GET", Path: "/status", Handler: func(ctx Context) error {
			steps = append(steps, "handler")
			return ctx.Write([]byte("ok"))
		}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	if err := adapter.routes["GET /status"](&capturedContext{}); err != nil {
		t.Fatal(err)
	}
	want := "class before,class before,handler,class after,class after"
	if got := strings.Join(steps, ","); got != want {
		t.Fatalf("order: got %s, want %s", got, want)
	}
}

func TestClassMiddlewareTypedNilFailsAtBootstrap(t *testing.T) {
	var missing *classTestMiddleware
	for _, option := range []Option{WithMiddlewareHandlers(missing), WithRouteMiddlewareHandlers("GET", "/status", missing)} {
		_, err := New(WithAdapter(&capturedAdapter{}), option)
		if err == nil || !strings.Contains(err.Error(), "nil") {
			t.Fatalf("expected bootstrap rejection for typed nil middleware, got %v", err)
		}
	}
}
