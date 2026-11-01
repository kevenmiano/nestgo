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
`http.ServeMux`. The initial fasthttp path matcher is linear in route count;
it must be replaced with an indexed router and benchmarked before release.

**Migration status:** The canonical application lifecycle is `runtime.NewApp(ctx, With...)`. The earlier `runtime.New`, `runtime.NewWithOptions` and `runtime.FromRegistry` entrypoints were removed. The legacy adapter-specific `Application` API is still isolated pending consolidation into `NewApp` (#103). Do not introduce new dependencies on that API; it will be removed before the first stable release. HTTP engine switching must retain parity for provider factories, route handlers, and shutdown semantics.\n\n## Open verification work

- Produce and commit `go.sum` with `go mod tidy` (network-enabled CI).
- Run `go test ./...` and `go vet ./...` on Go 1.23.
- Validate route precedence, 404/405, trailing slash, request-body limits,
  response status and headers, and shutdown parity across engines.
- Measure throughput, allocations, startup time and p95/p99 latency.
- Replace runtime `reflect.Call` with generated invokers.
- Document unsupported platform capabilities such as WebSocket and streaming.
