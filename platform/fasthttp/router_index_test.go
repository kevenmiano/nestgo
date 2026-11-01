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
		method          string
		path            string
		handler, exists bool
		param           string
	}{
		{"GET", "/users/42", true, true, "42"},
		{"GET", "/users/me", true, true, ""},
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
		byMethod := make(map[string][]httpadapter.Route)
		for _, route := range routes {
			key := fmt.Sprintf("%s %d", route.Method, segmentCount(route.Path))
			byMethod[key] = append(byMethod[key], route)
		}
		target := fmt.Sprintf("/api/resource%d/42", count-1)
		for _, benchmark := range []struct {
			name string
			run  func() bool
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
			{"current_main", func() bool {
				segments := segmentCount(target)
				for _, route := range byMethod[fmt.Sprintf("GET %d", segments)] {
					if _, ok := match(route.Path, target); ok {
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

func TestIndexedRouterPrefersSpecificRoutes(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		chosen := ""
		dynamic := httpadapter.Route{
			Method: "GET", Path: "/users/{id}",
			Handler: func(httpadapter.Context) error { chosen = "dynamic"; return nil },
		}
		literal := httpadapter.Route{
			Method: "GET", Path: "/users/me",
			Handler: func(httpadapter.Context) error { chosen = "literal"; return nil },
		}
		routes := []httpadapter.Route{dynamic, literal}
		if reverse {
			routes[0], routes[1] = routes[1], routes[0]
		}
		handler, params, _ := compileRoutes(routes).lookup("GET", "/users/me")
		if handler == nil {
			t.Fatal("specific route missing")
		}
		if err := handler(nil); err != nil {
			t.Fatal(err)
		}
		if chosen != "literal" || len(params) != 0 {
			t.Fatalf("reverse=%v: handler=%q params=%v", reverse, chosen, params)
		}
	}
}

func TestRegisterRejectsAmbiguousOverlappingRoutes(t *testing.T) {
	handler := func(httpadapter.Context) error { return nil }
	for _, tc := range []struct {
		first, second string
		reject        bool
	}{
		{"/{left}/foo", "/bar/{right}", true},
		{"/users/{id}", "/users/{other}", true},
		{"/users/{id}", "/users/me", false},
		{"/users/me", "/users/{id}", false},
		{"/users/{id}", "/orders/{id}", false},
	} {
		t.Run(tc.first+" vs "+tc.second, func(t *testing.T) {
			a := New()
			if err := a.Register(httpadapter.Route{Method: "GET", Path: tc.first, Handler: handler}); err != nil {
				t.Fatal(err)
			}
			err := a.Register(httpadapter.Route{Method: "GET", Path: tc.second, Handler: handler})
			if (err != nil) != tc.reject {
				t.Fatalf("register error=%v; want rejected=%v", err, tc.reject)
			}
		})
	}
}
