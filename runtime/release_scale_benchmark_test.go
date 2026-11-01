package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
)

// BenchmarkReleaseScale measures identical HTTP workloads across both engines,
// using matching counts of routes and singleton providers. Latency includes
// loopback transport and the Go client; it is not a pure route-lookup benchmark.
func BenchmarkReleaseScale(b *testing.B) {
	engines := []struct {
		name string
		newAdapter func() httpadapter.Adapter
	}{
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
		{"nethttp", func() httpadapter.Adapter { return netadapter.New() }},
	}
	for _, count := range []int{1, 10, 100, 1000} {
		pkg := metadata.Package{Providers: make([]metadata.Provider, 0, count)}
		routes := make([]Route, 0, count)
		for i := 0; i < count; i++ {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Marker",
				Type: reflect.TypeOf(0),
				Tag: reflect.StructTag(fmt.Sprintf("id:%q", fmt.Sprint(i))),
			}})
			pkg.Providers = append(pkg.Providers, metadata.Provider{Type: reflect.PointerTo(typ)})
			routes = append(routes, Route{
				Method: "GET", Path: fmt.Sprintf("/scale/%d", i),
				Handler: func(ctx Context) error {
					ctx.Status(http.StatusOK)
					return ctx.Write([]byte("ok"))
				},
			})
		}
		target := fmt.Sprintf("/scale/%d", count-1)
		for _, engine := range engines {
			b.Run(fmt.Sprintf("%s/routes_%d/providers_%d", engine.name, count, count), func(b *testing.B) {
				app, err := New(WithAdapter(engine.newAdapter()), WithPackages(pkg), WithRoutes(routes...))
				if err != nil {
					b.Fatal(err)
				}
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					b.Fatal(err)
				}
				done := make(chan error, 1)
				go func() { done <- app.Serve(listener) }()
				client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{
					MaxIdleConns: 64, MaxIdleConnsPerHost: 64, MaxConnsPerHost: 64,
				}}
				b.Cleanup(func() {
					client.CloseIdleConnections()
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := app.Shutdown(ctx); err != nil {
						b.Error(err)
					}
					_ = listener.Close()
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						b.Error("server did not stop")
					}
				})
				url := "http://" + listener.Addr().String() + target
				for i := 0; i < 10; i++ {
					resp, err := client.Get(url)
					if err != nil {
						b.Fatal(err)
					}
					_, readErr := io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					if readErr != nil || resp.StatusCode != 200 {
						b.Fatalf("warmup failed: status=%d err=%v", resp.StatusCode, readErr)
					}
				}
				latencies := make([]int64, b.N)
				var successes atomic.Int64
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						start := time.Now()
						resp, err := client.Get(url)
						if err != nil {
							b.Error(err)
							return
						}
						_, readErr := io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
						if readErr != nil || resp.StatusCode != 200 {
							b.Errorf("status=%d err=%v", resp.StatusCode, readErr)
							return
						}
						elapsed := time.Since(start).Nanoseconds()
						index := successes.Add(1) - 1
						if index < int64(len(latencies)) {
							latencies[index] = elapsed
						}
					}
				})
				b.StopTimer()
				n := int(successes.Load())
				if n == 0 {
					b.Fatal("no successful requests")
				}
				sort.Slice(latencies[:n], func(i, j int) bool {
					return latencies[i] < latencies[j]
				})
				b.ReportMetric(float64(latencies[(n-1)*95/100])/1e6, "p95-ms")
				b.ReportMetric(float64(latencies[(n-1)*99/100])/1e6, "p99-ms")
				b.ReportMetric(float64(n)/b.Elapsed().Seconds(), "req/s")
			})
		}
	}
}
