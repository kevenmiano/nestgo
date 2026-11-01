# HTTP platform adapters

All HTTP engines use **the same** `runtime.New` factory, dependency injection, route metadata, lifecycle hooks and shutdown.

Plain Go handler registration needs no metadata or code-generation step:

```go
app, err := runtime.New(
    runtime.WithRoutes(runtime.Route{
        Method: "GET", Path: "/ping",
        Handler: func(ctx runtime.Context) error { return ctx.Write([]byte("pong")) },
    }),
)
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Generated controller metadata is optional. Native fasthttp (default):

```go
app, err := runtime.New(runtime.WithRegistry("shop"))
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Go standard-library HTTP engine:

```go
app, err := runtime.New(
    runtime.WithRegistry("shop"),
    runtime.WithAdapter(nethttp.New()),
)
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Custom engines implement `httpadapter.Adapter`: `Register(httpadapter.Route) error`, `Serve(net.Listener) error` and `Shutdown(context.Context) error`. The common `httpadapter.Context` exposes request and response operations. The application owns the adapter; always call `app.Shutdown(ctx)`.

Fasthttp uses native `fasthttp.RequestCtx` (no net/http adaptor); the standard-library engine uses `http.ServeMux`. Middleware written for net/http might require an adapter-aware integration.

## Verification priorities

Validate route precedence, 404/405, trailing slash, JSON body-size limits, headers, status codes, and graceful shutdown across engines. Benchmark startup, allocations and p95/p99 HTTP latency before production use. WebSockets and streaming are not guaranteed by the current adapter contract.

No separate legacy `http.Handler` constructor exists.
