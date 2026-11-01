package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

type benchmarkBindingController struct{}

func (*benchmarkBindingController) Ready() string           { return "ok" }
func (*benchmarkBindingController) Path(id string) string   { return id }
func (*benchmarkBindingController) Search(q string) string  { return q }
func (*benchmarkBindingController) Create(v struct{ Name string }) string {
	return v.Name
}

func benchmarkMetadata() []metadata.Package {
	return []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*benchmarkBindingController)(nil)),
		Prefix: "/bench",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/ready", Handler: "Ready", StaticInvoke: func(c any) (any, error) {
				return c.(*benchmarkBindingController).Ready(), nil
			}},
			{Method: "GET", Path: "/path/{id}", Handler: "Path", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
			{Method: "GET", Path: "/search", Handler: "Search", Parameters: []metadata.Parameter{{Index: 0, Kind: "Query", Name: "q"}}},
			{Method: "POST", Path: "/create", Handler: "Create", Parameters: []metadata.Parameter{{Index: 0, Kind: "Body"}}},
		},
	}}}}
}

func benchmarkHTTPServer(b *testing.B, adapter httpadapter.Adapter) (string, func()) {
	b.Helper()
	if err := registerRoutes(adapter, benchmarkMetadata()); err != nil {
		b.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- adapter.Serve(listener) }()
	return "http://" + listener.Addr().String(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = adapter.Shutdown(ctx)
		_ = listener.Close()
		<-done
	}
}

// Each HTTP engine is measured over the same loopback HTTP/1.1 client and
// request workload. These results include routing, transport and JSON.
func BenchmarkHTTPEngines(b *testing.B) {
	engines := []struct {
		name string
		new  func() httpadapter.Adapter
	}{
		{"nethttp", func() httpadapter.Adapter { return nethttp.New() }},
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
	}
	cases := []struct {
		name, method, path, body string
	}{
		{"static", "GET", "/bench/ready", ""},
		{"path", "GET", "/bench/path/42", ""},
		{"query", "GET", "/bench/search?q=needle", ""},
		{"body", "POST", "/bench/create", `{"Name":"Ada"}`},
	}
	for _, engine := range engines {
		for _, tc := range cases {
			b.Run(engine.name+"/"+tc.name, func(b *testing.B) {
				base, stop := benchmarkHTTPServer(b, engine.new())
				defer stop()
				client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
					MaxIdleConns:        128,
					MaxIdleConnsPerHost: 128,
					MaxConnsPerHost:     128,
				}}
				defer client.CloseIdleConnections()
				url := base + tc.path
				// Warm-up is outside the measured region.
				for i := 0; i < 10; i++ {
					req, err := http.NewRequest(tc.method, url, strings.NewReader(tc.body))
					if err != nil {
						b.Fatal(err)
					}
					resp, err := client.Do(req)
					if err != nil {
						b.Fatal(err)
					}
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
				latencies := make([]int64, b.N)
				var index atomic.Int64
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						req, err := http.NewRequest(tc.method, url, strings.NewReader(tc.body))
						if err != nil {
							b.Error(err)
							return
						}
						start := time.Now()
						resp, err := client.Do(req)
						if err != nil {
							b.Error(err)
							return
						}
						_, err = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
						if err != nil || resp.StatusCode >= 400 {
							b.Errorf("status=%d err=%v", resp.StatusCode, err)
							return
						}
						i := index.Add(1) - 1
						if i < int64(len(latencies)) {
							latencies[i] = time.Since(start).Nanoseconds()
						}
					}
				})
				b.StopTimer()
				n := int(index.Load())
				if n == 0 {
					b.Fatal("no successful requests")
				}
				sort.Slice(latencies[:n], func(i, j int) bool { return latencies[i] < latencies[j] })
				b.ReportMetric(float64(latencies[(n-1)*95/100])/1e6, "p95-ms")
				b.ReportMetric(float64(latencies[(n-1)*99/100])/1e6, "p99-ms")
				b.ReportMetric(float64(n)/b.Elapsed().Seconds(), "req/s")
			})
		}
	}
}

// Bootstrap is intentionally measured without starting network listeners.
func BenchmarkAdapterBootstrap(b *testing.B) {
	for _, engine := range []struct {
		name string
		new  func() httpadapter.Adapter
	}{
		{"nethttp", func() httpadapter.Adapter { return nethttp.New() }},
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
	} {
		b.Run(engine.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := registerRoutes(engine.new(), benchmarkMetadata()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkAdapterConcurrency(b *testing.B) {
	for _, parallel := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("P%d", parallel), func(b *testing.B) {
			b.SetParallelism(parallel)
			a, err := benchmarkRoutes(true)
			if err != nil {
				b.Fatal(err)
			}
			h := a.routes["GET /bench/ready"]
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := &capturedContext{}
				for pb.Next() {
					ctx.body = ""
					if err := h(ctx); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
	}
}
