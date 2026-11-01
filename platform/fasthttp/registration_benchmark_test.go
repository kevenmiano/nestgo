package fasthttp

import (
	"fmt"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
)

func TestLiteralRegistrationPreservesWildcardSafety(t *testing.T) {
	handler := func(httpadapter.Context) error { return nil }
	for _, dynamicFirst := range []bool{false, true} {
		a := New()
		literals := []httpadapter.Route{
			{Method: "GET", Path: "/users/me", Handler: handler},
			{Method: "GET", Path: "/users/admin", Handler: handler},
			{Method: "POST", Path: "/users/me", Handler: handler},
		}
		dynamic := httpadapter.Route{Method: "GET", Path: "/users/{id}", Handler: handler}
		if dynamicFirst {
			if err := a.Register(dynamic); err != nil {
				t.Fatal(err)
			}
		}
		for _, route := range literals {
			if err := a.Register(route); err != nil {
				t.Fatalf("dynamicFirst=%v, route=%s: %v", dynamicFirst, route.Path, err)
			}
		}
		if !dynamicFirst {
			if err := a.Register(dynamic); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.Register(literals[0]); err == nil {
			t.Fatal("exact duplicate must be rejected")
		}
		if err := a.Register(httpadapter.Route{
			Method: "GET", Path: "/users/{other}", Handler: handler,
		}); err == nil {
			t.Fatal("equivalent dynamic patterns must be rejected")
		}
		if err := a.Register(httpadapter.Route{
			Method: "GET", Path: "/{section}/me", Handler: handler,
		}); err == nil {
			t.Fatal("incomparable overlapping dynamic routes must be rejected")
		}
	}
}

// BenchmarkRegisterLiteralRoutes measures the full registration cost, rather
// than only routeIndex lookup; it catches quadratic conflict-scan regressions.
func BenchmarkRegisterLiteralRoutes(b *testing.B) {
	handler := func(httpadapter.Context) error { return nil }
	for _, count := range []int{1, 10, 100, 1000} {
		routes := make([]httpadapter.Route, count)
		for i := range routes {
			routes[i] = httpadapter.Route{
				Method: "GET",
				Path: fmt.Sprintf("/resource/%d", i),
				Handler: handler,
			}
		}
		b.Run(fmt.Sprintf("routes_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				adapter := New()
				for _, route := range routes {
					if err := adapter.Register(route); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
