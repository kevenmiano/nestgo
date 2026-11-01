package fasthttp

import (
	"context"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
)

func TestAdapterLazilyAllocatesRouteIndex(t *testing.T) {
	adapter := New()
	if adapter.keys != nil {
		t.Fatal("route-free HTTP adapter should not allocate route index")
	}
	handler := func(ctx httpadapter.Context) error {
		return ctx.Write([]byte("ok"))
	}
	if err := adapter.Register(httpadapter.Route{Method: "GET", Path: "invalid", Handler: handler}); err == nil {
		t.Fatal("invalid non-absolute route should fail")
	}
	if adapter.keys != nil {
		t.Fatal("invalid route allocated the index")
	}
	if err := adapter.Register(httpadapter.Route{Method: "GET", Path: "/health", Handler: handler}); err != nil {
		t.Fatal(err)
	}
	if len(adapter.keys) != 1 || !adapter.keys["GET /health"] {
		t.Fatalf("route index not populated: %#v", adapter.keys)
	}
	if err := adapter.Register(httpadapter.Route{Method: "GET", Path: "/health", Handler: handler}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate route not rejected: %v", err)
	}
	if len(adapter.routes) != 1 {
		t.Fatalf("expected exactly one route, got %d", len(adapter.routes))
	}
	if err := adapter.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
