package runtime

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

var benchmarkHTTPResult string

func benchmarkRoutes(static bool) (*capturedAdapter, error) {
	adapter := &capturedAdapter{}
	route := metadata.Route{Method: "GET", Path: "/ready", Handler: "Ready"}
	if static {
		route.StaticInvoke = func(v any) (any, error) {
			return v.(*staticRouteController).Ready(), nil
		}
	}
	pkgs := []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*staticRouteController)(nil)),
		Prefix: "/bench",
		Routes: []metadata.Route{route},
	}}}}
	if err := registerRoutes(adapter, pkgs); err != nil {
		return nil, err
	}
	return adapter, nil
}

func BenchmarkHTTPInvocation(b *testing.B) {
	for _, tc := range []struct {
		name   string
		static bool
	}{
		{name: "static", static: true},
		{name: "reflection", static: false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			adapter, err := benchmarkRoutes(tc.static)
			if err != nil {
				b.Fatal(err)
			}
			handler := adapter.routes["GET /bench/ready"]
			ctx := &capturedContext{}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.body = ""
				if err := handler(ctx); err != nil {
					b.Fatal(err)
				}
			}
			benchmarkHTTPResult = ctx.body
		})
	}
}

func BenchmarkHTTPDirectBaseline(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /bench/ready", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("\"reflect-fallback\"\n"))
	})
	req := httptest.NewRequest(http.MethodGet, "/bench/ready", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkHTTPBootstrap(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := benchmarkRoutes(true); err != nil {
			b.Fatal(err)
		}
	}
}

// Invocation comparisons exclude the transport and router; the direct
// net/http baseline includes ServeMux and recorder overhead, so its absolute
// numbers must not be interpreted as an apples-to-apples throughput comparison.
var _ httpadapter.Adapter = (*capturedAdapter)(nil)
