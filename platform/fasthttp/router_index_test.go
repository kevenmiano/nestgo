package fasthttp

import (
	"fmt"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
)

func TestIndexedRouterCompatibility(t *testing.T) {
	h := func(httpadapter.Context) error { return nil }
	routes := []httpadapter.Route{
		{Method: "GET", Path: "/users/{id}", Handler: h},
		{Method: "GET", Path: "/users/me", Handler: h},
		{Method: "POST", Path: "/orders/{id}", Handler: h},
		{Method: "GET", Path: "/health", Handler: h},
	}
	index := compileRoutes(routes)
	for _, tc := range []struct {
		method         string
		path           string
		handler, exists bool
		param          string
	}{
		{"GET", "/users/42", true, true, "42"},
		{"GET", "/users/me", true, true, "me"},
		{"POST", "/orders/7", true, true, "7"},
		{"GET", "/orders/7", false, true, ""},
		{"GET", "/missing", false, false, ""},
		{"GET", "/health", true, true, ""},
	} {
		handler, params, exists := index.lookup(tc.method, tc.path)
		if (handler != nil) != tc.handler || exists != tc.exists {
			t.Errorf("%s %s: handler=%v exists=%v", tc.method, tc.path, handler != nil, exists)
		}
		if tc.param != "" && params["id"] != tc.param {
			t.Errorf("%s: id=%q want=%q", tc.path, params["id"], tc.param)
		}
	}
}

func BenchmarkRouteLookup(b *testing.B) {
	h := func(httpadapter.Context) error { return nil }
	for _, count := range []int{10, 100, 500, 1000} {
		routes := make([]httpadapter.Route, 0, count)
		for i := 0; i < count; i++ {
			routes = append(routes, httpadapter.Route{
				Method: "GET", Path: fmt.Sprintf("/api/resource%d/{id}", i), Handler: h,
			})
		}
		index := compileRoutes(routes)
		target := fmt.Sprintf("/api/resource%d/42", count-1)
		for _, benchmark := range []struct {
			name string
			run func() bool
		}{
			{"linear", func() bool {
				for _, route := range routes {
					_, ok := match(route.Path, target)
					if ok && route.Method == "GET" {
						return true
					}
				}
				return false
			}},
			{"indexed", func() bool {
				handler, _, _ := index.lookup("GET", target)
				return handler != nil
			}},
		} {
			b.Run(fmt.Sprintf("routes_%d/%s", count, benchmark.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if !benchmark.run() {
						b.Fatal("route missing")
					}
				}
			})
		}
	}
}
