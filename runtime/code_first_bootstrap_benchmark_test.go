package runtime

import (
	"fmt"
	"net/http"
	"testing"
)

// benchmarkCodeFirstBootstrapSink ensures the application remains observable.
var benchmarkCodeFirstBootstrapSink *Application

// BenchmarkCodeFirstBootstrap measures application construction without a
// NestGo compiler, generated packages, active listener or HTTP client.
// Identical source is run against main and the candidate commit in CI.
func BenchmarkCodeFirstBootstrap(b *testing.B) {
	for _, count := range []int{0, 1, 10, 100} {
		b.Run(fmt.Sprintf("routes_%d", count), func(b *testing.B) {
			routes := make([]Route, 0, count)
			handler := func(ctx Context) error {
				return ctx.Write([]byte("ok"))
			}
			for i := 0; i < count; i++ {
				routes = append(routes, Route{
					Method:  http.MethodGet,
					Path:    fmt.Sprintf("/health/%d", i),
					Handler: handler,
				})
			}
			option := WithRoutes(routes...)
			b.ReportAllocs()
			b.ResetTimer()
			var last *Application
			for i := 0; i < b.N; i++ {
				app, err := New(option)
				if err != nil {
					b.Fatal(err)
				}
				last = app
			}
			b.StopTimer()
			benchmarkCodeFirstBootstrapSink = last
		})
	}
}
