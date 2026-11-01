package runtime

import (
	"context"
	"io"
	"net"
	"net/http"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
)

// BenchmarkCodeFirstInvocation isolates the direct Go handler from the HTTP
// transport, so it can be compared with BenchmarkHTTPInvocation's compiled
// static and reflective controller dispatch.
func BenchmarkCodeFirstInvocation(b *testing.B) {
	adapter := &capturedAdapter{}
	app, err := New(
		WithAdapter(adapter),
		WithRoutes(Route{
			Method: http.MethodGet,
			Path: "/bench/ready",
			Handler: func(ctx Context) error {
				ctx.Status(http.StatusOK)
				return ctx.Write([]byte("\"ok\"\n"))
			},
		}),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer app.Shutdown(context.Background())
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
}

// BenchmarkCodeFirstHTTPEngines compares the same HTTP/1.1 loopback GET and
// response bytes through direct Go routes versus generated controller routes.
// The network, router, HTTP client, JSON payload and concurrency are included.
func BenchmarkCodeFirstHTTPEngines(b *testing.B) {
	engines := []struct {
		name string
		new func() httpadapter.Adapter
	}{
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
		{"nethttp", func() httpadapter.Adapter { return netadapter.New() }},
	}
	for _, engine := range engines {
		for _, mode := range []string{"code_first", "controller"} {
			b.Run(engine.name+"/"+mode, func(b *testing.B) {
				options := []Option{WithAdapter(engine.new())}
				if mode == "code_first" {
					options = append(options, WithRoutes(Route{
						Method: http.MethodGet,
						Path: "/bench/ready",
						Handler: func(ctx Context) error {
							ctx.Status(http.StatusOK)
							ctx.SetHeader("Content-Type", "application/json")
							return ctx.Write([]byte("\"ok\"\n"))
						},
					}))
				} else {
					options = append(options, WithPackages(benchmarkMetadata()...))
				}
				app, err := New(options...)
				if err != nil {
					b.Fatal(err)
				}
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					b.Fatal(err)
				}
				done := make(chan error, 1)
				go func() { done <- app.Serve(listener) }()
				defer func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := app.Shutdown(ctx); err != nil {
						b.Error(err)
					}
					listener.Close()
					select {
					case <-done:
					case <-time.After(5*time.Second):
						b.Error("HTTP server did not stop")
					}
				}()
				client := &http.Client{
					Timeout: 5 * time.Second,
					Transport: &http.Transport{
						MaxIdleConns: 128,
						MaxIdleConnsPerHost: 128,
						MaxConnsPerHost: 128,
					},
				}
				defer client.CloseIdleConnections()
				url := "http://" + listener.Addr().String() + "/bench/ready"
				for i := 0; i < 20; i++ {
					resp, err := client.Get(url)
					if err != nil {
						b.Fatal(err)
					}
					body, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil || resp.StatusCode != http.StatusOK || string(body) != "\"ok\"\n" {
						b.Fatalf("warmup: status=%d body=%q err=%v", resp.StatusCode, body, err)
					}
				}
				latencies := make([]int64, b.N)
				var completed atomic.Int64
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						req, err := http.NewRequest(http.MethodGet, url, nil)
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
						_, copyErr := io.Copy(io.Discard, resp.Body)
						closeErr := resp.Body.Close()
						if copyErr != nil || closeErr != nil || resp.StatusCode != http.StatusOK {
							b.Errorf("status=%d copy=%v close=%v", resp.StatusCode, copyErr, closeErr)
							return
						}
						index := completed.Add(1) - 1
						if index < int64(len(latencies)) {
							latencies[index] = time.Since(start).Nanoseconds()
						}
					}
				})
				b.StopTimer()
				n := int(completed.Load())
				if n != b.N {
					b.Fatalf("completed %d requests, expected %d", n, b.N)
				}
				sort.Slice(latencies[:n], func(i, j int) bool { return latencies[i] < latencies[j] })
				b.ReportMetric(float64(latencies[(n-1)*95/100])/1e6, "p95-ms")
				b.ReportMetric(float64(latencies[(n-1)*99/100])/1e6, "p99-ms")
				b.ReportMetric(float64(n)/b.Elapsed().Seconds(), "req/s")
			})
		}
	}
}
