# HTTP platform adapters

NestGo's application-level API accepts a transport-independent `httpadapter.Adapter`.
The default adapter is `platform/fasthttp`; the standard-library adapter is
`platform/nethttp`. Custom engines can implement the same interface.

```go
app, err := runtime.ApplicationFromRegistry(runtime.ApplicationOptions{}, "shop")
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

To explicitly select the standard library adapter:

```go
app, err := runtime.ApplicationFromRegistry(
    runtime.ApplicationOptions{Adapter: nethttp.New()}, "shop",
)
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Custom adapters must implement `Register(httpadapter.Route) error`,
`Serve(net.Listener) error` and `Shutdown(context.Context) error`.
`httpadapter.Context` abstracts request parsing and response writing.
The application core performs controller/provider resolution once during
bootstrap and registers handlers independently of platform.

**Native fast path:** the fasthttp platform uses `fasthttp.RequestCtx`
without `fasthttpadaptor`; the standard-library platform uses
`http.ServeMux`. The fasthttp adapter now builds a method/segment-count index during Serve startup and only scans matching buckets for common requests. This is not a fully indexed path trie; benchmark worst-case bucket sizes and route precedence before release.

**Backward compatibility:** `runtime.New` and `runtime.FromRegistry`
continue to return `http.Handler`; these legacy APIs remain on net/http.
The application-level API uses adapters. Not all net/http-specific middleware
can operate unchanged on fasthttp; use platform-aware integration when needed.

## Open verification work

- Produce and commit `go.sum` with `go mod tidy` (network-enabled CI).
- Run `go test ./...` and `go vet ./...` on Go 1.23.
- Validate route precedence, 404/405, trailing slash, request-body limits,
  response status and headers, and shutdown parity across engines.
- Measure throughput, allocations, startup time and p95/p99 latency.
- Extend generated static invokers beyond eligible zero-argument handlers (currently supported on both runtime paths).
- Document unsupported platform capabilities such as WebSocket and streaming.
